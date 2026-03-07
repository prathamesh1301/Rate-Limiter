package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	baseURL = "http://localhost:8080/" // change to your route
	userID  = "user_123"
)

type ResultDetailNew struct {
	Allowed      bool `json:"allowed"`
	RefreshAfter int  `json:"refresh_after"`
}

type Response struct {
	Result ResultDetailNew
	Source string // "redis", "postgres", "new-user"
}

// ---- Test 1: Burst — fire 15 requests instantly, expect first 10 to pass ----
func testBurst() {
	fmt.Println("\n========== TEST 1: BURST (15 requests instantly) ==========")

	allowed := 0
	denied := 0

	for i := 1; i <= 15; i++ {
		resp, err := sendRequest(userID)
		if err != nil {
			fmt.Printf("  Request %2d → ERROR: %v\n", i, err)
			continue
		}

		source := formatSource(resp.Source)

		if resp.Result.Allowed {
			allowed++
			fmt.Printf("  Request %2d → ✅ ALLOWED   [source: %s]\n", i, source)
		} else {
			denied++
			fmt.Printf("  Request %2d → ❌ BLOCKED   [source: %s] (retry after %ds)\n", i, source, resp.Result.RefreshAfter)
		}
	}

	fmt.Printf("\n  Summary: %d allowed, %d blocked\n", allowed, denied)
	fmt.Printf("  Expected: 10 allowed, 5 blocked\n")
}

// ---- Test 2: Refill — exhaust tokens, wait, then try again ----
func testRefill() {
	fmt.Println("\n========== TEST 2: REFILL (exhaust → wait 3s → retry) ==========")

	fmt.Println("  Exhausting tokens...")
	for i := 1; i <= 10; i++ {
		sendRequest(userID)
	}

	result, _ := sendRequest(userID)
	if !result.Result.Allowed {
		fmt.Println("  ✅ Confirmed exhausted")
	}

	waitSecs := 3
	fmt.Printf("  Waiting %d seconds for refill...\n", waitSecs)
	time.Sleep(time.Duration(waitSecs) * time.Second)

	allowed := 0
	for i := 1; i <= 5; i++ {
		resp, err := sendRequest(userID)
		if err != nil {
			fmt.Printf("  Request %d → ERROR: %v\n", i, err)
			continue
		}
		source := formatSource(resp.Source)
		if resp.Result.Allowed {
			allowed++
			fmt.Printf("  Request %d → ✅ ALLOWED   [source: %s]\n", i, source)
		} else {
			fmt.Printf("  Request %d → ❌ BLOCKED   [source: %s]\n", i, source)
		}
	}

	fmt.Printf("\n  Got %d tokens after %ds wait (expected ~%d)\n", allowed, waitSecs, waitSecs)
}

// ---- Test 3: Concurrent — fire requests from multiple goroutines simultaneously ----
func testConcurrent() {
	fmt.Println("\n========== TEST 3: CONCURRENT (10 goroutines at once) ==========")

	var wg sync.WaitGroup
	var mu sync.Mutex

	allowed := 0
	denied := 0
	redisHits := 0
	postgresHits := 0
	goroutines := 10

	for i := 1; i <= goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			resp, err := sendRequest(userID)
			if err != nil {
				fmt.Printf("  Goroutine %2d → ERROR: %v\n", id, err)
				return
			}

			mu.Lock()
			defer mu.Unlock()

			source := formatSource(resp.Source)

			if resp.Source == "redis" {
				redisHits++
			} else {
				postgresHits++
			}

			if resp.Result.Allowed {
				allowed++
				fmt.Printf("  Goroutine %2d → ✅ ALLOWED   [source: %s]\n", id, source)
			} else {
				denied++
				fmt.Printf("  Goroutine %2d → ❌ BLOCKED   [source: %s]\n", id, source)
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("\n  Summary: %d allowed, %d blocked\n", allowed, denied)
	fmt.Printf("  Cache hits: %d redis, %d postgres\n", redisHits, postgresHits)
}

// ---- Test 4: Different users — each user should have independent buckets ----
func testMultipleUsers() {
	fmt.Println("\n========== TEST 4: MULTIPLE USERS (independent buckets) ==========")

	users := []string{"alice", "bob", "charlie"}

	for _, user := range users {
		resp, err := sendRequest(user)
		if err != nil {
			fmt.Printf("  %-10s → ERROR: %v\n", user, err)
			continue
		}
		source := formatSource(resp.Source)
		if resp.Result.Allowed {
			fmt.Printf("  %-10s → ✅ ALLOWED   [source: %s]\n", user, source)
		} else {
			fmt.Printf("  %-10s → ❌ BLOCKED   [source: %s]\n", user, source)
		}
	}
}

// ---- Test 5: Slow drip — 1 request every second, should always pass ----
func testSlowDrip() {
	fmt.Println("\n========== TEST 5: SLOW DRIP (1 req/sec for 5 secs) ==========")

	for i := 1; i <= 5; i++ {
		resp, err := sendRequest(userID)
		if err != nil {
			fmt.Printf("  Request %d → ERROR: %v\n", i, err)
		} else if resp.Result.Allowed {
			fmt.Printf("  Request %d → ✅ ALLOWED   [source: %s]\n", i, formatSource(resp.Source))
		} else {
			fmt.Printf("  Request %d → ❌ BLOCKED   [source: %s] (unexpected!)\n", i, formatSource(resp.Source))
		}
		time.Sleep(1 * time.Second)
	}
}

// ---- Test 6: Cache vs DB — first request hits DB, second hits cache ----
func testCacheVsDB() {
	fmt.Println("\n========== TEST 6: CACHE VS DB (new user flow) ==========")

	// Use a fresh user so we can observe new-user → redis flow
	freshUser := fmt.Sprintf("fresh_user_%d", time.Now().Unix())

	resp1, err := sendRequest(freshUser)
	if err != nil {
		fmt.Printf("  Request 1 → ERROR: %v\n", err)
		return
	}
	fmt.Printf("  Request 1 → [source: %s] (expected: new-user)\n", formatSource(resp1.Source))

	resp2, err := sendRequest(freshUser)
	if err != nil {
		fmt.Printf("  Request 2 → ERROR: %v\n", err)
		return
	}
	fmt.Printf("  Request 2 → [source: %s] (expected: redis)\n", formatSource(resp2.Source))

	resp3, err := sendRequest(freshUser)
	if err != nil {
		fmt.Printf("  Request 3 → ERROR: %v\n", err)
		return
	}
	fmt.Printf("  Request 3 → [source: %s] (expected: redis)\n", formatSource(resp3.Source))
}

// ---- Helper ----
func sendRequest(userID string) (Response, error) {
	url := fmt.Sprintf("%s?user_id=%s", baseURL, userID)

	resp, err := http.Get(url)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	var result ResultDetailNew
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Response{}, err
	}

	// Read source header set by the server
	source := resp.Header.Get("X-Data-Source")
	if source == "" {
		source = "unknown"
	}

	return Response{Result: result, Source: source}, nil
}

// formatSource makes the source label colourful and consistent width
func formatSource(source string) string {
	switch source {
	case "redis":
		return "🔴 redis   "
	case "postgres":
		return "🐘 postgres"
	case "new-user":
		return "🆕 new-user"
	default:
		return "❓ unknown "
	}
}

func main() {
	fmt.Println("🚀 Rate Limiter Test Suite")
	fmt.Println("   Target:", baseURL)
	fmt.Println("   User:  ", userID)

	// Test 6 first — uses a fresh user so source tracking is clean
	testCacheVsDB()

	testBurst()

	fmt.Println("\n  Waiting 15s to refill bucket before next test...")
	time.Sleep(15 * time.Second)

	testRefill()

	fmt.Println("\n  Waiting 15s to refill bucket before next test...")
	time.Sleep(15 * time.Second)

	testConcurrent()

	fmt.Println("\n  Waiting 15s to refill bucket before next test...")
	time.Sleep(15 * time.Second)

	testMultipleUsers()

	fmt.Println("\n  Waiting 15s to refill bucket before next test...")
	time.Sleep(15 * time.Second)

	testSlowDrip()

	fmt.Println("\n✅ All tests done")
}