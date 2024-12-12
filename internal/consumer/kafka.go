// internal/consumer/kafka.go
package consumer

import (
	"context"
	"encoding/json"
	"github.com/IBM/sarama"
	"go-kafka-go/internal/models"
	"go-kafka-go/internal/store"
	"log"
	"sync/atomic"
	"time"
)

type KafkaConsumer struct {
	client            sarama.ConsumerGroup
	store             store.Store
	ready             chan bool
	messagesProcessed int64 // atomic
	processingErrors  int64 // atomic
	batchesProcessed  int64 // atomic
	lastMessageTime   atomic.Value
	processingTime    atomic.Value // stores time.Duration
	cfg               *Config
}

func NewKafkaConsumer(cfg *Config, store store.Store) (*KafkaConsumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	client, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, config)
	if err != nil {
		return nil, err
	}

	c := &KafkaConsumer{
		client: client,
		store:  store,
		ready:  make(chan bool),
		cfg:    cfg,
	}

	c.lastMessageTime.Store(time.Now())
	c.processingTime.Store(time.Duration(0))

	return c, nil
}

func (c *KafkaConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		start := time.Now()

		if err := c.handleMessage(session.Context(), message); err != nil {
			atomic.AddInt64(&c.processingErrors, 1)
			log.Printf("Error processing message: %v", err)
			continue
		}

		// Update metrics
		atomic.AddInt64(&c.messagesProcessed, 1)
		c.lastMessageTime.Store(time.Now())

		// Update average processing time
		duration := time.Since(start)
		oldAvg := c.processingTime.Load().(time.Duration)
		newAvg := (oldAvg + duration) / 2
		c.processingTime.Store(newAvg)

		session.MarkMessage(message, "")
	}

	return nil
}

func (c *KafkaConsumer) handleMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	var loginEvent models.LoginEvent
	if err := json.Unmarshal(message.Value, &loginEvent); err != nil {
		return err
	}

	parsedLogin, err := models.NewParsedLogin(loginEvent)
	if err != nil {
		return err
	}

	return c.store.Save(ctx, parsedLogin)
}

func (c *KafkaConsumer) Start(ctx context.Context) error {
	// Start metrics reporting
	go c.reportMetrics(ctx)

	topics := []string{c.cfg.Topic}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := c.client.Consume(ctx, topics, c)
		if err != nil {
			if err == context.Canceled {
				return nil
			}
			log.Printf("Error from consumer: %v", err)
			atomic.AddInt64(&c.processingErrors, 1)
			continue
		}

		c.ready = make(chan bool)
	}
}

func (c *KafkaConsumer) Setup(sarama.ConsumerGroupSession) error {
	close(c.ready)
	return nil
}

func (c *KafkaConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (c *KafkaConsumer) Stop(ctx context.Context) error {
	return c.client.Close()
}

func (c *KafkaConsumer) GetStats() ConsumerStats {
	return ConsumerStats{
		MessagesProcessed: atomic.LoadInt64(&c.messagesProcessed),
		ProcessingErrors:  atomic.LoadInt64(&c.processingErrors),
		LastMessageTime:   c.lastMessageTime.Load().(time.Time),
		BatchesProcessed:  atomic.LoadInt64(&c.batchesProcessed),
		AvgProcessingTime: c.processingTime.Load().(time.Duration),
	}
}

func (c *KafkaConsumer) reportMetrics(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := c.GetStats()
			log.Printf("Consumer Metrics - Processed: %d, Errors: %d, Avg Processing Time: %v",
				stats.MessagesProcessed, stats.ProcessingErrors, stats.AvgProcessingTime)
		}
	}
}
