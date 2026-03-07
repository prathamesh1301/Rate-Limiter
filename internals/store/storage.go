package store

import (
	"database/sql"

	"github.com/redis/go-redis/v9"
)

type Storage struct {
	RateLimiter *RateLimiter
}

func NewStorage(db *sql.DB,rdb *redis.Client) *Storage {
	return &Storage{
		RateLimiter: &RateLimiter{db: db,rdb: rdb},
	}
}