// internal/store/postgres.go
package store

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"go-kafka-go/internal/models"
	"sync"
	"sync/atomic"
	"time"
)

type PostgresStore struct {
	pool          *pgxpool.Pool
	batch         []*models.ParsedLogin
	batchSize     int
	mu            sync.Mutex
	flushCh       chan struct{}
	stats         StoreStats
	totalWrites   int64 // atomic
	writeErrors   int64 // atomic
	lastFlushTime atomic.Value
}

func NewPostgresStore(ctx context.Context, connString string, batchSize int) (*PostgresStore, error) {
	pool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		return nil, err
	}

	s := &PostgresStore{
		pool:      pool,
		batch:     make([]*models.ParsedLogin, 0, batchSize),
		batchSize: batchSize,
		flushCh:   make(chan struct{}, 1),
	}

	s.lastFlushTime.Store(time.Now())

	// Start background batch processor
	go s.processBatches(ctx)

	return s, nil
}

func (s *PostgresStore) Save(ctx context.Context, login *models.ParsedLogin) error {
	s.mu.Lock()
	s.batch = append(s.batch, login)
	currentBatchSize := len(s.batch)
	s.mu.Unlock()

	// Update stats
	s.stats.QueuedWrites = currentBatchSize

	if currentBatchSize >= s.batchSize {
		select {
		case s.flushCh <- struct{}{}:
		default:
			// Channel is full, skip signal as a flush is already pending
		}
	}

	return nil
}

func (s *PostgresStore) processBatches(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.Flush(ctx) // Final flush
			return
		case <-s.flushCh:
			s.Flush(ctx)
		case <-ticker.C:
			s.Flush(ctx)
		}
	}
}

// Flush implements the Store interface
func (s *PostgresStore) Flush(ctx context.Context) error {
	s.mu.Lock()
	if len(s.batch) == 0 {
		s.mu.Unlock()
		return nil
	}

	batch := s.batch
	s.batch = make([]*models.ParsedLogin, 0, s.batchSize)
	s.mu.Unlock()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		atomic.AddInt64(&s.writeErrors, 1)
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"logins"},
		[]string{
			"user_id", "device_id", "app_version", "device_type",
			"ip", "locale", "original_timestamp", "parsed_timestamp",
			"processed_at",
		},
		pgx.CopyFromSlice(len(batch), func(i int) ([]interface{}, error) {
			login := batch[i]
			return []interface{}{
				login.UserID,
				login.DeviceID,
				login.AppVersion,
				login.DeviceType,
				login.IP,
				login.Locale,
				login.Timestamp,
				login.ParsedTime,
				login.ProcessedAt,
			}, nil
		}),
	)

	if err != nil {
		atomic.AddInt64(&s.writeErrors, 1)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		atomic.AddInt64(&s.writeErrors, 1)
		return err
	}

	atomic.AddInt64(&s.totalWrites, int64(len(batch)))
	s.lastFlushTime.Store(time.Now())

	return nil
}

func (s *PostgresStore) GetStats() StoreStats {
	s.mu.Lock()
	queuedWrites := len(s.batch)
	s.mu.Unlock()

	return StoreStats{
		BatchSize:     s.batchSize,
		QueuedWrites:  queuedWrites,
		TotalWrites:   atomic.LoadInt64(&s.totalWrites),
		LastFlushTime: s.lastFlushTime.Load().(time.Time),
		WriteErrors:   atomic.LoadInt64(&s.writeErrors),
	}
}

func (s *PostgresStore) Close() error {
	s.pool.Close()
	return nil
}
