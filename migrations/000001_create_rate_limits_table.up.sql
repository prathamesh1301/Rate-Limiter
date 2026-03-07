CREATE TABLE rate_limits (
    user_id VARCHAR(255) PRIMARY KEY,
    tokens INT NOT NULL,
    last_refill_time TIMESTAMP NOT NULL DEFAULT NOW()
);
