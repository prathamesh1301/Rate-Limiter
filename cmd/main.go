package main

import (
	"context"
	"log"

	"github.com/prathamesh/rate-limiter/internals/cache"
	"github.com/prathamesh/rate-limiter/internals/db"
	"github.com/prathamesh/rate-limiter/internals/logger"
	"github.com/prathamesh/rate-limiter/internals/store"
)

func main() {
	logger, err := logger.NewLogger()
	if err != nil {
		log.Fatal("failed to create logger: ", err)
	}
	defer logger.Sync()

	logger.Info("Logger initialized successfully")

	dbConfig := &dbConfig{
		addr:         "postgres://admin:adminPassword@localhost:5432/postgres?sslmode=disable",
		maxOpenConns: 10,
		maxIdleConns: 5,
		maxIdleTime:  "5m",
	}

	redisConfig := &redisConfig{
		addr:     "localhost:6379",
		password: "",
		db:       0,
	}

	db, err := db.New(dbConfig.addr, dbConfig.maxOpenConns, dbConfig.maxIdleConns, dbConfig.maxIdleTime)
	if err != nil {
		logger.Fatal("failed to connect to database: ", err)
	}
	defer db.Close()
	logger.Info("Connected to PostgreSQL successfully")

	rdb, err := cache.NewRedisClient(context.Background(), redisConfig.addr, "", 0)
	if err != nil {
		logger.Fatal("failed to connect to redis: ", err)
	}
	defer rdb.Close()
	logger.Info("Connected to Redis successfully")
	storage := store.NewStorage(db, rdb)
	config := config{
		addr:        ":8080",
		dbConfig:    dbConfig,
		redisConfig: redisConfig,
	}

	app := &Application{
		config: &config,
		store:  storage,
		logger: logger,
	}
	app.run()
}
