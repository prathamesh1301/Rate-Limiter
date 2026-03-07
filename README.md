# Rate Limiter

A token-bucket rate limiter implemented in Go with a PostgreSQL database.

## Overview

This project provides a robust rate-limiting service that can track and limit requests per user. It uses the Token Bucket algorithm to manage request allowances, ensuring that users cannot exceed their defined rate limits while allowing for short bursts of traffic.

## Setup

1. **Database:** Ensure you have PostgreSQL running. Make sure you apply the migrations inside the `migrations` folder to create the required `rate_limits` table.
   ```sql
   CREATE TABLE rate_limits (
       user_id VARCHAR(255) PRIMARY KEY,
       tokens INT NOT NULL,
       last_refill_time TIMESTAMP NOT NULL DEFAULT NOW()
   );
   ```
2. **Configuration:** The application expects a PostgreSQL connection string (configured in `cmd/main.go`).
3. **Run the server:**
   ```bash
   cd cmd
   go run .
   ```
   The server will start on `http://localhost:8080`.

## Testing

You can use the provided test suite to verify the rate limiter's behavior under different conditions.

### Test Suite Execution Results

**Target:** `http://localhost:8080/`
**User:** `user_123`

```text
========== TEST 1: BURST (15 requests instantly) ==========
  Request  1 → ✅ ALLOWED
  Request  2 → ✅ ALLOWED
  Request  3 → ✅ ALLOWED
  Request  4 → ✅ ALLOWED
  Request  5 → ✅ ALLOWED
  Request  6 → ✅ ALLOWED
  Request  7 → ✅ ALLOWED
  Request  8 → ✅ ALLOWED
  Request  9 → ✅ ALLOWED
  Request 10 → ✅ ALLOWED
  Request 11 → ❌ BLOCKED  (retry after 1s)
  Request 12 → ❌ BLOCKED  (retry after 1s)
  Request 13 → ❌ BLOCKED  (retry after 1s)
  Request 14 → ❌ BLOCKED  (retry after 1s)
  Request 15 → ❌ BLOCKED  (retry after 1s)

  Summary: 10 allowed, 5 blocked
  Expected: 10 allowed, 5 blocked

  Waiting 15s to refill bucket before next test...

========== TEST 2: REFILL (exhaust → wait 3s → retry) ==========
  Exhausting tokens...
  ✅ Confirmed exhausted
  Waiting 3 seconds for refill...
  Request 1 → ✅ ALLOWED
  Request 2 → ✅ ALLOWED
  Request 3 → ✅ ALLOWED
  Request 4 → ❌ BLOCKED
  Request 5 → ❌ BLOCKED

  Got 3 tokens after 3s wait (expected ~3)

  Waiting 15s to refill bucket before next test...

========== TEST 3: CONCURRENT (10 goroutines at once) ==========
  Goroutine 10 → ✅ ALLOWED
  Goroutine  9 → ✅ ALLOWED
  Goroutine  4 → ✅ ALLOWED
  Goroutine  8 → ✅ ALLOWED
  Goroutine  7 → ✅ ALLOWED
  Goroutine  2 → ✅ ALLOWED
  Goroutine  6 → ✅ ALLOWED
  Goroutine  3 → ✅ ALLOWED
  Goroutine  5 → ✅ ALLOWED
  Goroutine  1 → ✅ ALLOWED

  Summary: 10 allowed, 0 blocked

  Waiting 15s to refill bucket before next test...

========== TEST 4: MULTIPLE USERS (independent buckets) ==========
  alice      → ✅ ALLOWED
  bob        → ✅ ALLOWED
  charlie    → ✅ ALLOWED

  Waiting 15s to refill bucket before next test...

========== TEST 5: SLOW DRIP (1 req/sec for 5 secs) ==========
  Request 1 → ✅ ALLOWED
  Request 2 → ✅ ALLOWED
  Request 3 → ✅ ALLOWED
  Request 4 → ✅ ALLOWED
  Request 5 → ✅ ALLOWED

✅ All tests done
```
