package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"time"
)

const (
	maxTokens  = 10 // maximum tokens a user can have
	refillRate = 1  // tokens added per second
)

type RateLimitCKDetail struct {
	Tokens         int
	LastRefillTime time.Time
}

type ResultDetail struct {
	Allowed      bool `json:"allowed"`
	RefreshAfter int  `json:"refresh_after"` // seconds until next token
}

func (app *Application) rateLimitChecker(w http.ResponseWriter, r *http.Request) {
	userId := r.URL.Query().Get("user_id")
	if userId == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	var rateLimitDetail RateLimitCKDetail
	isNewUser := false 

	// ---- GET: Redis first, fallback to Postgres ----
	cacheDetail, err := app.store.RateLimiter.GetUserLimitDetailsFromCache(userId)
	if err != nil {
		details, err := app.store.RateLimiter.GetUserLimitDetails(userId)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "Failed to get rate limit details", http.StatusInternalServerError)
				return
			}
			// New user — create in Postgres + cache, skip the update at the end
			now := time.Now().UTC()
			if err = app.store.RateLimiter.CreateUserLimitDetails(userId, maxTokens, now); err != nil {
				http.Error(w, "Failed to create rate limit details", http.StatusInternalServerError)
				return
			}
			w.Header().Set("X-Data-Source", "new-user -> postgres")
			app.store.RateLimiter.SetUserLimitDetailsInCache(userId, maxTokens, now)
			rateLimitDetail = RateLimitCKDetail{Tokens: maxTokens, LastRefillTime: now}
			isNewUser = true 
		} else {
			w.Header().Set("X-Data-Source", "postgres")
			rateLimitDetail = RateLimitCKDetail{Tokens: details.Tokens, LastRefillTime: details.LastRefillTime}
		}
	} else {
		w.Header().Set("X-Data-Source", "redis")
		rateLimitDetail = RateLimitCKDetail{Tokens: cacheDetail.Tokens, LastRefillTime: cacheDetail.LastRefillTime}
	}

	// ---- CALCULATE: refill + consume ----
	now := time.Now().UTC()

	elapsedTime := now.Sub(rateLimitDetail.LastRefillTime.UTC()).Seconds()
	if elapsedTime < 0 {
		elapsedTime = 0
	}

	newTokens := min(rateLimitDetail.Tokens+int(elapsedTime)*refillRate, maxTokens)

	var result ResultDetail
	if newTokens > 0 {
		newTokens--
		result = ResultDetail{Allowed: true}
	} else {
		result = ResultDetail{
			Allowed:      false,
			RefreshAfter: int(math.Ceil(1.0 / float64(refillRate))),
		}
	}

	// ---- WRITE: skip if new user, already written above ----
	if !isNewUser {
		if err = app.store.RateLimiter.UpdateUserLimitDetails(userId, newTokens, now); err != nil {
			http.Error(w, "Failed to update rate limit details", http.StatusInternalServerError)
			return
		}
		app.store.RateLimiter.SetUserLimitDetailsInCache(userId, newTokens, now)
	}

	// ---- RESPOND ----
	w.Header().Set("Content-Type", "application/json")
	if !result.Allowed {
		w.WriteHeader(http.StatusTooManyRequests)
	}
	json.NewEncoder(w).Encode(result)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}