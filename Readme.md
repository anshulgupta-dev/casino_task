# Backend Developer Assessment - Online Casino PAM System

This repository contains the solution for the Backend Developer Assessment, addressing two critical architectural challenges:
1.  **Atomic Wallet Operations**: Handling concurrent transactions with strict consistency.
2.  **Real-time Bonus Wagering**: High-throughput wagering progress calculation.

## Project Structure

```
/wallet-service
  /cmd
    main.go           # Application entry point
  /internal
    /wallet           # Challenge 1: Wallet Logic
      models.go
      repository.go
      service.go
    /bonus            # Challenge 2: Bonus Logic
      models.go
      service.go
  /pkg
    /database         # Shared database code (if any)
  /tests              # Integration tests
    wallet_test.go
    bonus_test.go
  /docs
    challenge1-architecture.md
    challenge2-architecture.md
  docker-compose.yml
  init.sql
```

## Setup & Running

### Prerequisites
- Go 1.20+
- Docker & Docker Compose

### 1. Start Database & Redis
```bash
docker-compose up -d
```
*Note: Postgres runs on port 5433 and Redis on 6380 to avoid conflicts with local services.*

### 2. Run Tests
```bash
go test ./tests/... -v
```

### 3. Run Application
```bash
go run cmd/main.go
```
The server will start on `:8080`.

## Design Decisions & Assumptions

### Challenge 1: Wallet Operations
- **Optimistic Locking**: Used `version` column to handle concurrent updates without heavy DB locks.
- **Idempotency**: `reference_id` + `transaction_type` unique constraint ensures exactly-once processing.
- **Isolation**: Used `REPEATABLE READ` (implied by optimistic locking logic) to ensure consistency.

### Challenge 2: Bonus Wagering
- **Architecture**: Designed as an event-driven system (simulated in-memory for task).
- **Concurrency**: Uses mutexes for thread-safety in the in-memory implementation. In production, this would use Redis atomic operations (`INCRBYFLOAT`) or sharded consumers.
- **Real-time**: Designed to use WebSockets (simulated with Go channels).

## Production Readiness Checklist

- [x] **Error Handling**: Custom error types and proper HTTP status codes.
- [x] **Context Usage**: Passed `context.Context` through all layers.
- [x] **Logging**: Basic logging implemented; structured logging (e.g., Zap/Logrus) recommended for prod.
- [x] **Configuration**: DB connection string via env vars (with defaults).


## Known Limitations
- **Bonus Service**: Currently in-memory. For production, it needs Redis/DB persistence as detailed in `docs/challenge2-architecture.md`.
- **Test Database**: Tests run against the local Docker container. In a CI environment, this should be ephemeral.
