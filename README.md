# go-kafka-go

# Go Kafka Go

A Go service that consumes login events from Kafka and stores them in PostgreSQL with efficient batching.

## Prerequisites
- Docker and Docker Compose
- Go 1.19 or higher

## Setup and Running

1. First, start the infrastructure (Kafka, Zookeeper, Postgres, and the data generator):
```bash
docker-compose up -d
```

2. Verify the services are running:
```bash
docker-compose ps
```
You should see the following services running:
- zookeeper
- kafka
- postgres
- my-python-producer (this generates sample login data)

3. Build the Go service:
```bash
go build -o go-kafka-go
```

4. Run the service:
```bash
./go-kafka-go
```

## Monitoring

The service provides several ways to monitor its operation:

1. Console logs will show:
    - Connection status to Kafka and Postgres
    - Message processing stats every 10 seconds
    - Batch writing stats every 5 seconds

2. To check data in Postgres:
```bash
docker exec -it go-kafka-go_postgres_1 psql -U loginapp -d logindb
```

Then in psql:
```sql
-- Check total number of processed logins
SELECT COUNT(*) FROM logins;

-- View sample data
SELECT * FROM logins LIMIT 5;
```

## Shutting Down

1. Stop the Go service with Ctrl+C (it will gracefully shut down)

2. Stop the infrastructure:
```bash
docker-compose down
```

Add -v flag to also remove volumes:
```bash
docker-compose down -v
```

## Troubleshooting

If you see connection errors:
1. Ensure all containers are running: `docker-compose ps`
2. Check container logs: `docker-compose logs [service_name]`
3. Verify Postgres is accessible: `docker exec -it go-kafka-go_postgres_1 psql -U loginapp -d logindb`
4. Verify Kafka is accessible: `docker exec -it go-kafka-go_kafka_1 kafka-topics --list --bootstrap-server localhost:9092`