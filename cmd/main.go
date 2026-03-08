package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/prathamesh/rate-limiter/internals/cache"
	"github.com/prathamesh/rate-limiter/internals/db"
	"github.com/prathamesh/rate-limiter/internals/logger"
	"github.com/prathamesh/rate-limiter/internals/store"
)

func main() {
	// Load environment variables from .env file
	// Try current directory, then try parent directory (for cmd/ structure)
	err := godotenv.Load()
	if err != nil {
		err = godotenv.Load("../.env")
	}

	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	logger, err := logger.NewLogger()
	if err != nil {
		log.Fatal("failed to create logger: ", err)
	}
	defer logger.Sync()

	logger.Info("Logger initialized successfully")

	dbConfig := &dbConfig{
		addr:         getEnv("DB_ADDR", "postgres://user:pwd@localhost:5432/postgres?sslmode=disable"),
		maxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 10),
		maxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 5),
		maxIdleTime:  getEnv("DB_MAX_IDLE_TIME", "5m"),
	}

	redisConfig := &redisConfig{
		addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		password: getEnv("REDIS_PASSWORD", ""),
		db:       getEnvInt("REDIS_DB", 0),
	}

	db, err := db.New(dbConfig.addr, dbConfig.maxOpenConns, dbConfig.maxIdleConns, dbConfig.maxIdleTime)
	if err != nil {
		logger.Fatal("failed to connect to database: ", err)
	}
	defer db.Close()
	logger.Info("Connected to PostgreSQL successfully")

	rdb, err := cache.NewRedisClient(context.Background(), redisConfig.addr, redisConfig.password, redisConfig.db)
	if err != nil {
		logger.Fatal("failed to connect to redis: ", err)
	}
	defer rdb.Close()
	logger.Info("Connected to Redis successfully")

	storage := store.NewStorage(db, rdb)
	config := config{
		addr:        getEnv("SERVER_ADDR", ":8080"),
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

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		i, err := strconv.Atoi(value)
		if err != nil {
			return fallback
		}
		return i
	}
	return fallback
}
