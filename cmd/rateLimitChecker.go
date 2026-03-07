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

	details, err := app.store.RateLimiter.GetUserLimitDetails(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// User has no record yet — create one with full tokens
			now := time.Now().UTC()
			err = app.store.RateLimiter.CreateUserLimitDetails(userId, maxTokens, now)
			if err != nil {
				http.Error(w, "Failed to create rate limit details", http.StatusInternalServerError)
				return
			}
			details.Tokens = maxTokens
			details.LastRefillTime = now
		} else {
			http.Error(w, "Failed to get rate limit details", http.StatusInternalServerError)
			return
		}
	}

	rateLimitCkDetail := RateLimitCKDetail{
		Tokens:         details.Tokens,
		LastRefillTime: details.LastRefillTime,
	}

	now := time.Now().UTC()
	elapsedTime := now.Sub(rateLimitCkDetail.LastRefillTime.UTC()).Seconds()

	// Step 1: Refill tokens based on elapsed time
	newTokens := rateLimitCkDetail.Tokens + int(elapsedTime)*refillRate
	if newTokens > maxTokens {
		newTokens = maxTokens
	}

	// Step 2: Check if a token is available and consume it
	var result ResultDetail

	if newTokens > 0 {
		// Allow the request, consume one token
		newTokens--
		result = ResultDetail{
			Allowed:      true,
			RefreshAfter: 0,
		}
	} else {
		// Deny the request, tell them when the next token arrives
		refreshAfter := int(math.Ceil(1.0 / float64(refillRate)))
		result = ResultDetail{
			Allowed:      false,
			RefreshAfter: refreshAfter,
		}
	}

	// Step 3: Update the DB with the new token count and refill time
	err = app.store.RateLimiter.UpdateUserLimitDetails(userId, newTokens, now.UTC())
	if err != nil {
		http.Error(w, "Failed to update rate limit details", http.StatusInternalServerError)
		return
	}

	// Step 4: Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if !result.Allowed {
		w.WriteHeader(http.StatusTooManyRequests)
	}
	json.NewEncoder(w).Encode(result)
}
