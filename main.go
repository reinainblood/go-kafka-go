// main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-kafka-go/internal/consumer"
	"go-kafka-go/internal/store"
)

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC)
	log.Printf("Starting login event consumer...")

	// Get database configuration from environment
	dbUser := getEnvOrDefault("DB_USER", "loginapp")
	dbPass := getEnvOrDefault("DB_PASSWORD", "securepassword")
	dbName := getEnvOrDefault("DB_NAME", "logindb")
	dbHost := getEnvOrDefault("DB_HOST", "localhost")
	dbPort := getEnvOrDefault("DB_PORT", "5432")

	// Construct database connection string using key=value format
	dbConnString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName,
	)

	// Initialize our store with retries
	ctx := context.Background()
	var pgStore *store.PostgresStore
	var err error

	// Try to connect to postgres with retries
	for retries := 0; retries < 5; retries++ {
		if retries > 0 {
			log.Printf("Retrying database connection in 5 seconds... (attempt %d/5)", retries+1)
			time.Sleep(5 * time.Second)
		}

		pgStore, err = store.NewPostgresStore(
			ctx,
			dbConnString,
			1000, // batch size
		)
		if err == nil {
			break
		}
		log.Printf("Database connection attempt failed: %v", err)
	}

	if err != nil {
		log.Fatalf("Failed to initialize store after retries: %v", err)
	}
	defer pgStore.Close()
	log.Printf("Connected to Postgres successfully")

	// Get Kafka configuration from environment
	kafkaBrokers := getEnvOrDefault("KAFKA_BOOTSTRAP_SERVERS", "localhost:29092")
	kafkaTopic := getEnvOrDefault("KAFKA_TOPIC", "user-login")
	kafkaGroupID := getEnvOrDefault("KAFKA_GROUP_ID", "login-processor-group")

	// Set up consumer config
	consumerCfg := &consumer.Config{
		Brokers: []string{kafkaBrokers},
		Topic:   kafkaTopic,
		GroupID: kafkaGroupID,
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

	err = kafkaConsumer.Stop(shutdownCtx)
	if err != nil {
		log.Println("Error shutting down Kafka Consumer")
		return
	}
	log.Println("Shutdown complete")
}
