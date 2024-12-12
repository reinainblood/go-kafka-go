// main.go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-kafka-go/internal/consumer"
	"go-kafka-go/internal/store"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC)
	log.Printf("Starting login event consumer...")

	// Initialize our store
	ctx := context.Background()
	pgStore, err := store.NewPostgresStore(
		ctx,
		"postgres://loginapp:securepassword@localhost:5432/logindb",
		1000, // batch size
	)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer pgStore.Close()
	log.Printf("Connected to Postgres successfully")

	// Set up consumer config
	consumerCfg := &consumer.Config{
		Brokers: []string{"localhost:29092"},
		Topic:   "user-login",
		GroupID: "login-processor-group",
	}

	// Initialize Kafka consumer
	kafkaConsumer, err := consumer.NewKafkaConsumer(consumerCfg, pgStore)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer kafkaConsumer.Stop(ctx)
	log.Printf("Connected to Kafka successfully")

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start consumer in background
	go func() {
		log.Printf("Starting to consume messages...")
		if err := kafkaConsumer.Start(ctx); err != nil {
			log.Printf("Consumer error: %v", err)
			cancel()
		}
	}()

	// Print store stats periodically
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats := pgStore.GetStats()
				log.Printf("Store stats - Queued: %d, Total Writes: %d, Errors: %d",
					stats.QueuedWrites, stats.TotalWrites, stats.WriteErrors)
			}
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Received shutdown signal, starting graceful shutdown...")

	// Give it a moment to finish processing
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	kafkaConsumer.Stop(shutdownCtx)
	log.Println("Shutdown complete")
}
