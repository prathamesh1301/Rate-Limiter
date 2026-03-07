package store

import "database/sql"

type Storage struct {
	RateLimiter *RateLimiter
}

func NewStorage(db *sql.DB) *Storage {
	return &Storage{
		RateLimiter: &RateLimiter{db: db},
	}
}