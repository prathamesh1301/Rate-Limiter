package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	db  *sql.DB
	rdb *redis.Client
}

type RateLimitDetails struct {
	UserId         string
	Tokens         int
	LastRefillTime time.Time
}

func (rateLimiter *RateLimiter) GetUserLimitDetails(userId string) (RateLimitDetails, error) {
	query := "SELECT tokens, last_refill_time FROM rate_limits WHERE user_id = $1"
	row := rateLimiter.db.QueryRow(query, userId)
	var rateLimitDetails RateLimitDetails
	err := row.Scan(&rateLimitDetails.Tokens, &rateLimitDetails.LastRefillTime)
	if err != nil {
		return RateLimitDetails{}, err
	}
	return rateLimitDetails, nil
}

func (rateLimiter *RateLimiter) CreateUserLimitDetails(userId string, tokens int, lastRefillTime time.Time) error {
	query := "INSERT INTO rate_limits (user_id, tokens, last_refill_time) VALUES ($1, $2, $3)"
	_, err := rateLimiter.db.Exec(query, userId, tokens, lastRefillTime)
	if err != nil {
		return err
	}
	return nil
}

func (rateLimiter *RateLimiter) UpdateUserLimitDetails(userId string, tokens int, lastRefillTime time.Time) error {
	query := "UPDATE rate_limits SET tokens = $1, last_refill_time = $2 WHERE user_id = $3"
	_, err := rateLimiter.db.Exec(query, tokens, lastRefillTime, userId)
	if err != nil {
		return err
	}
	return nil
}

func (rateLimiter *RateLimiter) GetUserLimitDetailsFromCache(userId string) (RateLimitDetails, error) {
	ctx := context.Background()
	val, err := rateLimiter.rdb.Get(ctx, "rate_limit:"+userId).Result()
	if err != nil {
		return RateLimitDetails{}, err // Caller should handle redis.Nil to know if it's a cache miss
	}

	var details RateLimitDetails
	err = json.Unmarshal([]byte(val), &details)
	if err != nil {
		return RateLimitDetails{}, err
	}

	details.UserId = userId
	return details, nil
}

func (rateLimiter *RateLimiter) SetUserLimitDetailsInCache(userId string, tokens int, lastRefillTime time.Time) error {
	ctx := context.Background()
	details := RateLimitDetails{
		UserId:         userId,
		Tokens:         tokens,
		LastRefillTime: lastRefillTime,
	}
	data, err := json.Marshal(details)
	if err != nil {
		return err
	}
	// Setting with a 1-hour expiration; adjust if your rate-limit duration is different
	err = rateLimiter.rdb.Set(ctx, "rate_limit:"+userId, data, time.Hour).Err()
	return err
}
