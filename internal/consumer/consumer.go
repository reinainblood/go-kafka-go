// internal/consumer/consumer.go
package consumer

import (
	"context"
	"time"
)

// Consumer defines the interface for consuming login events
type Consumer interface {
	// Start begins consuming messages. It blocks until the context is cancelled
	Start(ctx context.Context) error

	// Stop gracefully shuts down the consumer
	Stop(ctx context.Context) error

	// GetStats returns current consumer statistics
	GetStats() ConsumerStats
}

// ConsumerStats contains metrics about the consumer
type ConsumerStats struct {
	MessagesProcessed int64
	ProcessingErrors  int64
	LastMessageTime   time.Time
	BatchesProcessed  int64
	AvgProcessingTime time.Duration
}

// Config holds the configuration for a consumer
type Config struct {
	Brokers       []string
	Topic         string
	GroupID       string
	BatchSize     int
	FlushInterval time.Duration
}

// NewConfig creates a new Config with default values
func NewConfig() *Config {
	return &Config{
		BatchSize:     1000,
		FlushInterval: 5 * time.Second,
	}
}

// Validate that KafkaConsumer implements Consumer
var _ Consumer = (*KafkaConsumer)(nil)
