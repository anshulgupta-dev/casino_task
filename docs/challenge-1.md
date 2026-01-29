# Challenge 1: Atomic Wallet Operations - Architecture & Analysis

## 1. Concurrency Strategy

### Race Condition Prevention
To prevent race conditions where multiple transactions attempt to modify the same wallet balance simultaneously, we will use **Optimistic Locking** via a `version` column in the `wallets` table.

- **Why Optimistic Locking?**
  - **Performance**: Avoids long-held database locks (Pessimistic Locking) which can reduce throughput.
  - **Scalability**: Better suited for high-concurrency environments where conflicts are possible but not always guaranteed.
  - **Deadlock Prevention**: Reduces the risk of deadlocks compared to pessimistic locking.

### Implementation Details
- Each wallet record has a `version` integer.
- When updating a balance, we check if the version matches the one we read.
- SQL: `UPDATE wallets SET balance = $1, version = version + 1 WHERE wallet_id = $2 AND version = $3`
- If `RowsAffected` is 0, it means the record was modified by another transaction. We will retry the operation (read-modify-write loop).

### Distributed Locking (Redis)
While optimistic locking handles the database integrity, we can use Redis distributed locks (Redlock or simple SETNX with expiry) as a first line of defense to serialize requests for the same `wallet_id` at the application level. This reduces database contention.
- **Key**: `lock:wallet:{wallet_id}`
- **TTL**: Short duration (e.g., 200ms) to prevent indefinite locking if a service crashes.

## 2. Database Schema Design

### Tables

#### `wallets`
| Column | Type | Description |
| :--- | :--- | :--- |
| `wallet_id` | UUID (PK) | Unique identifier |
| `player_id` | UUID | Player owner |
| `wallet_type` | VARCHAR | 'main', 'bonus' |
| `currency` | VARCHAR | 'USD', 'EUR' |
| `balance` | NUMERIC(20, 2) | Current balance |
| `version` | INTEGER | Optimistic lock version |
| `updated_at` | TIMESTAMP | Last update time |

*Constraints*: `balance >= 0` (Database level enforcement)

#### `transactions`
| Column | Type | Description |
| :--- | :--- | :--- |
| `transaction_id` | UUID (PK) | Unique identifier |
| `wallet_id` | UUID (FK) | Link to wallet |
| `reference_id` | VARCHAR | External ref (idempotency key) |
| `transaction_type` | VARCHAR | 'deposit', 'withdrawal', 'bet', 'win' |
| `amount` | NUMERIC(20, 2) | Transaction amount |
| `balance_before` | NUMERIC(20, 2) | Snapshot before tx |
| `balance_after` | NUMERIC(20, 2) | Snapshot after tx |
| `status` | VARCHAR | 'pending', 'completed', 'failed' |

### Indexes
- `idx_wallets_player`: Fast lookup by player.
- `idx_transactions_wallet`: History retrieval.
- `idx_transactions_ref`: Idempotency checks.

## 3. Transaction Flow

### Sequence for Debit (Bet)
1.  **Request**: `POST /transaction` (Debit $50)
2.  **Idempotency Check**: Check `transactions` table for `reference_id`. If exists, return saved result.
3.  **Read Wallet**: `SELECT * FROM wallets WHERE wallet_id = ?`
4.  **Validation**: Check if `balance >= amount`.
5.  **Begin DB Transaction**:
    - `INSERT INTO transactions (status='pending', ...)`
    - `UPDATE wallets SET balance = balance - amount, version = version + 1 WHERE wallet_id = ? AND version = ?`
    - If Update fails (version mismatch): Rollback and Retry from Step 3.
    - If Update succeeds:
        - `UPDATE transactions SET status='completed', balance_after=...`
        - `COMMIT`
6.  **Response**: Return new balance.

### Error Handling
- **Insufficient Funds**: Return 402 Payment Required.
- **Version Mismatch**: Retry (up to N times).
- **DB Error**: Return 500 Internal Server Error.

## 4. Edge Cases

- **Negative Balance**: Prevented by `CHECK (balance >= 0)` constraint and application logic.
- **Timeout**: Context cancellation. If DB tx is committed, it's done. If not, it rolls back.
- **Idempotency**: `reference_id` + `transaction_type` unique constraint ensures we don't process the same external event twice.

## 5. Performance Considerations

- **Throughput**: Target 100+ TPS/player. Optimistic locking allows high concurrency.
- **Connection Pooling**: Configure `pgx` or `sql.DB` max open connections.
- **Redis Caching**: Cache balance for read-heavy operations (e.g., showing balance in UI), invalidate on write.
