# Challenge 2: Real-time Bonus Wagering - Architecture & Analysis

## 1. Synchronous vs. Asynchronous Processing

### Decision: Asynchronous Processing (Event-Driven)
Given the requirement of handling **1 million bets/second** (10k players * 100 bets/sec), synchronous processing is not feasible. Blocking the game round completion to calculate wagering progress would introduce unacceptable latency and potential points of failure.

- **Architecture**:
  1.  **Game Service** publishes `BetEvent` to a high-throughput message queue (e.g., Kafka or NATS JetStream).
  2.  **Wagering Service** consumes these events asynchronously.
  3.  **Wagering Service** calculates progress and updates the database.
  4.  **Notification Service** pushes updates to the player via WebSockets.

### Trade-offs
- **Pros**: High throughput, decoupling, failure isolation (game continues even if wagering is down).
- **Cons**: Eventual consistency (progress bar might lag by ms), complexity in handling out-of-order events (though less critical for cumulative sums).

## 2. Data Flow

1.  **Bet Placement**: Game Service deducts balance (Challenge 1) -> Emits `BetEvent`.
2.  **Ingestion**: Wagering Service consumes `BetEvent`.
3.  **Rule Lookup**: Fetch `GameContribution` (Cached in Redis).
    - Slots: 100%, Blackjack: 10%.
4.  **Calculation**: `ContributionAmount = BetAmount * ContributionPercentage`.
5.  **State Update**:
    - Increment `WageringCompleted` in Redis (for real-time speed).
    - Persist to PostgreSQL (batch updates or write-behind) for durability.
6.  **Completion Check**: If `WageringCompleted >= WageringRequired`:
    - Trigger `BonusConversionEvent`.
    - Credit Real Money Wallet, Debit/Close Bonus Wallet.
7.  **Notification**: Publish `WageringUpdate` to Redis Pub/Sub -> WebSocket Gateway -> Player Client.

## 3. Scale Challenges (1M bets/sec)

### Database Hot-spotting
- **Problem**: Updating `player_bonuses` row 100 times/sec per player is heavy for Postgres.
- **Solution**:
  - **Redis as Primary Counter**: Maintain active wagering progress in Redis.
  - **Write-Behind/Batching**: Flush updates to Postgres every X seconds or Y events.
  - **Sharding**: Shard Redis and Postgres by `PlayerID`.

### Real-time Updates
- **Redis Pub/Sub**: Efficient for broadcasting updates to WebSocket nodes.
- **WebSocket Gateway**: Stateless service that holds connections.

## 4. Edge Cases

- **Bonus Expiry**: Check `ExpiresAt` before processing. If expired, ignore or mark as forfeited.
- **Race Conditions**:
  - *Bonus converts while new bet comes in*: Use atomic CAS (Compare-And-Swap) in Redis or Versioning in DB.
  - *Network Partition*: Message queue ensures at-least-once delivery. Idempotency key (`BetID`) prevents double counting.

## 5. Monitoring & Observability

- **Metrics**:
  - `wagering_lag_ms`: Time from BetEvent timestamp to processing.
  - `events_processed_sec`: Throughput.
  - `conversion_rate`: Bonuses completed vs. forfeited.
- **Drift Detection**:
  - Periodic reconciliation job: Sum `BetEvents` from Data Warehouse/Logs and compare with `player_bonuses` table.

## 6. Implementation Plan (MVP)
For this assessment, we will implement a simplified version:
- **Channel-based Async**: Use Go channels to simulate the message queue.
- **Redis Caching**: Cache game rules and active bonus progress.
- **Postgres Persistence**: Update DB on every Nth event or completion to show understanding of batching, or simple direct updates if load allows (for the test).
