package store

import (
	"database/sql"
	"time"
)

type RateLimiter struct {
	db *sql.DB
}

type RateLimitDetails struct {
	UserId string
	Tokens int
	LastRefillTime time.Time
}

func(rateLimiter *RateLimiter) GetUserLimitDetails(userId string) (RateLimitDetails, error) {
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