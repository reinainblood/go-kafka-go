// internal/store/store.go
package store

import (
	"context"
	"go-kafka-go/internal/models"
	"time"
)

// Store defines the interface for storing login events
type Store interface {
	// Save stores a login event, possibly batching it
	Save(ctx context.Context, login *models.ParsedLogin) error

	// Flush forces any batched writes to be written
	Flush(ctx context.Context) error

	// Close cleans up any resources
	Close() error

	// GetStats returns current store statistics
	GetStats() StoreStats
}

// StoreStats contains metrics about the store
type StoreStats struct {
	BatchSize     int
	QueuedWrites  int
	TotalWrites   int64
	LastFlushTime time.Time
	WriteErrors   int64
}

// Validate that PostgresStore implements Store
var _ Store = (*PostgresStore)(nil)
