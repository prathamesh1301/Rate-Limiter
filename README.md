# Rate Limiter

A token-bucket rate limiter implemented in Go with PostgreSQL as the source of truth and Redis as a cache layer.

## Overview

This project provides a robust rate-limiting service that tracks and limits requests per user. It uses the **Token Bucket algorithm** to manage request allowances, ensuring users cannot exceed their defined rate limits while allowing short bursts of traffic.

**Architecture:**
- **PostgreSQL** — source of truth for token state
- **Redis** — fast cache layer (read-first, fallback to Postgres on miss)
- On every request: Redis is checked first. On a miss, Postgres is used and Redis is repopulated.
- Redis failures are treated as best-effort — the app degrades gracefully to Postgres.

## Setup

### 1. Database
Ensure PostgreSQL is running and apply the migrations in the `migrations` folder:
```sql
CREATE TABLE rate_limits (
    user_id VARCHAR(255) PRIMARY KEY,
    tokens INT NOT NULL,
    last_refill_time TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### 2. Redis
Run Redis locally via Docker:
```bash
docker run -d --name redis -p 6379:6379 redis
```
Redis will be available at `localhost:6379`. No additional config needed.

### 3. Configuration
PostgreSQL and Redis connection strings are configured in `cmd/main.go`.

### 4. Run the server
```bash
cd cmd
go run .
```
The server will start on `http://localhost:8080`.

## How It Works

```
Request comes in
      │
      ▼
Check Redis ──── HIT ────────────────────────┐
      │                                       │
    MISS                                      │
      │                                       │
      ▼                                       ▼
Check Postgres                        Calculate new tokens
      │                               (refill + consume)
   ┌──┴──┐                                    │
FOUND   NOT FOUND                             ▼
   │       │                         Write Postgres (source of truth)
   │       ▼                                  │
   │   Create new user                        ▼
   │   (Postgres + Redis)            Update Redis (best effort)
   │                                          │
   └──────────────────────────────────────────┘
                                              │
                                              ▼
                                       Return response
                                  (X-Data-Source header set)
```

Each response includes an `X-Data-Source` header indicating where the data was served from:
- `redis` — served from cache
- `postgres` — cache miss, served from database
- `new-user` — first time seeing this user

## API Documentation

This project uses **Swagger** for API documentation. The OpenAPI/Swagger specification can be found in the `docs/` directory.

## Testing

Run the test suite with the server already running:
```bash
cd test
go run ratelimit_runner.go
```

### Test Suite Execution Results

**Target:** `http://localhost:8080/`  
**User:** `user_123`

```text
🚀 Rate Limiter Test Suite
   Target: http://localhost:8080/
   User:   user_123

========== TEST 6: CACHE VS DB (new user flow) ==========
  Request 1 → [source: 🆕 new-user] (expected: new-user)
  Request 2 → [source: 🔴 redis   ] (expected: redis)
  Request 3 → [source: 🔴 redis   ] (expected: redis)

========== TEST 1: BURST (15 requests instantly) ==========
  Request  1 → ✅ ALLOWED   [source: 🐘 postgres]
  Request  2 → ✅ ALLOWED   [source: 🔴 redis   ]
  Request  3 → ✅ ALLOWED   [source: 🔴 redis   ]
  Request  4 → ✅ ALLOWED   [source: 🔴 redis   ]
  Request  5 → ✅ ALLOWED   [source: 🔴 redis   ]
  Request  6 → ✅ ALLOWED   [source: 🔴 redis   ]
  Request  7 → ✅ ALLOWED   [source: 🔴 redis   ]
  Request  8 → ✅ ALLOWED   [source: 🔴 redis   ]
  Request  9 → ✅ ALLOWED   [source: 🔴 redis   ]
  Request 10 → ✅ ALLOWED   [source: 🔴 redis   ]
  Request 11 → ❌ BLOCKED   [source: 🔴 redis   ] (retry after 1s)
  Request 12 → ❌ BLOCKED   [source: 🔴 redis   ] (retry after 1s)
  Request 13 → ❌ BLOCKED   [source: 🔴 redis   ] (retry after 1s)
  Request 14 → ❌ BLOCKED   [source: 🔴 redis   ] (retry after 1s)
  Request 15 → ❌ BLOCKED   [source: 🔴 redis   ] (retry after 1s)

  Summary: 10 allowed, 5 blocked
  Expected: 10 allowed, 5 blocked

  Waiting 15s to refill bucket before next test...

========== TEST 2: REFILL (exhaust → wait 3s → retry) ==========
  Exhausting tokens...
  ✅ Confirmed exhausted
  Waiting 3 seconds for refill...
  Request 1 → ✅ ALLOWED   [source: 🔴 redis   ]
  Request 2 → ✅ ALLOWED   [source: 🔴 redis   ]
  Request 3 → ✅ ALLOWED   [source: 🔴 redis   ]
  Request 4 → ❌ BLOCKED   [source: 🔴 redis   ]
  Request 5 → ❌ BLOCKED   [source: 🔴 redis   ]

  Got 3 tokens after 3s wait (expected ~3)

  Waiting 15s to refill bucket before next test...

========== TEST 3: CONCURRENT (10 goroutines at once) ==========
  Goroutine  1 → ✅ ALLOWED   [source: 🔴 redis   ]
  Goroutine  8 → ✅ ALLOWED   [source: 🔴 redis   ]
  Goroutine  3 → ✅ ALLOWED   [source: 🔴 redis   ]
  Goroutine  5 → ✅ ALLOWED   [source: 🔴 redis   ]
  Goroutine 10 → ✅ ALLOWED   [source: 🔴 redis   ]
  Goroutine  9 → ✅ ALLOWED   [source: 🔴 redis   ]
  Goroutine  6 → ✅ ALLOWED   [source: 🔴 redis   ]
  Goroutine  2 → ✅ ALLOWED   [source: 🔴 redis   ]
  Goroutine  7 → ✅ ALLOWED   [source: 🔴 redis   ]
  Goroutine  4 → ✅ ALLOWED   [source: 🔴 redis   ]

  Summary: 10 allowed, 0 blocked
  Cache hits: 10 redis, 0 postgres

  Waiting 15s to refill bucket before next test...

========== TEST 4: MULTIPLE USERS (independent buckets) ==========
  alice      → ✅ ALLOWED   [source: 🐘 postgres]
  bob        → ✅ ALLOWED   [source: 🐘 postgres]
  charlie    → ✅ ALLOWED   [source: 🐘 postgres]

  Waiting 15s to refill bucket before next test...

========== TEST 5: SLOW DRIP (1 req/sec for 5 secs) ==========
  Request 1 → ✅ ALLOWED   [source: 🔴 redis   ]
  Request 2 → ✅ ALLOWED   [source: 🔴 redis   ]
  Request 3 → ✅ ALLOWED   [source: 🔴 redis   ]
  Request 4 → ✅ ALLOWED   [source: 🔴 redis   ]
  Request 5 → ✅ ALLOWED   [source: 🔴 redis   ]

✅ All tests done
```

### What each test covers

| Test | What it verifies |
|---|---|
| **Cache vs DB** | New user hits Postgres, subsequent requests served from Redis |
| **Burst** | 15 instant requests — first 10 pass, last 5 blocked |
| **Refill** | Tokens refill correctly after waiting |
| **Concurrent** | 10 simultaneous goroutines — no race conditions |
| **Multiple users** | Each user has an independent bucket |
| **Slow drip** | 1 req/sec never gets blocked |

## Logging

This project uses **Uber Zap** for high-performance, structured logging. Logs are output in JSON format (production mode), making them easy to parse with log aggregators (like ELK, Datadog, or Grafana Loki).

The logs capture critical application events, including database/cache connectivity and real-time rate-limiting decisions.

### Example Logs

**Startup & Initialization:**
```json
{"level":"info","ts":1709905200.123,"msg":"Logger initialized successfully"}
{"level":"info","ts":1709905200.456,"msg":"Connected to PostgreSQL successfully"}
{"level":"info","ts":1709905200.789,"msg":"Connected to Redis successfully"}
{"level":"info","ts":1709905201.012,"msg":"Starting server on :8080"}
```

**Rate Limiting Events:**
```json
{"level":"info","ts":1709905205.123,"msg":"processing rate limit check","user_id":"user_123"}
{"level":"info","ts":1709905205.456,"msg":"rate limit allowed","user_id":"user_123","remaining_tokens":9,"source":"new-user -> postgres"}
{"level":"info","ts":1709905210.789,"msg":"rate limit exceeded","user_id":"user_123","source":"redis"}
```

**Error Handling:**
```json
{"level":"error","ts":1709905215.123,"msg":"failed to update rate limit in DB","user_id":"user_123","error":"connection pool exhausted"}
```
