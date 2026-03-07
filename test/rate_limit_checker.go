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

// ---- Test 1: Burst — fire 15 requests instantly, expect first 10 to pass ----
func testBurst() {
	fmt.Println("\n========== TEST 1: BURST (15 requests instantly) ==========")

	allowed := 0
	denied := 0

	for i := 1; i <= 15; i++ {
		result, err := sendRequest(userID)
		if err != nil {
			fmt.Printf("  Request %2d → ERROR: %v\n", i, err)
			continue
		}

		if result.Allowed {
			allowed++
			fmt.Printf("  Request %2d → ✅ ALLOWED\n", i)
		} else {
			denied++
			fmt.Printf("  Request %2d → ❌ BLOCKED  (retry after %ds)\n", i, result.RefreshAfter)
		}
	}

	fmt.Printf("\n  Summary: %d allowed, %d blocked\n", allowed, denied)
	fmt.Printf("  Expected: 10 allowed, 5 blocked\n")
}

// ---- Test 2: Refill — exhaust tokens, wait, then try again ----
func testRefill() {
	fmt.Println("\n========== TEST 2: REFILL (exhaust → wait 3s → retry) ==========")

	// Exhaust all tokens
	fmt.Println("  Exhausting tokens...")
	for i := 1; i <= 10; i++ {
		sendRequest(userID)
	}

	// Confirm exhausted
	result, _ := sendRequest(userID)
	if !result.Allowed {
		fmt.Println("  ✅ Confirmed exhausted")
	}

	// Wait for refill
	waitSecs := 3
	fmt.Printf("  Waiting %d seconds for refill...\n", waitSecs)
	time.Sleep(time.Duration(waitSecs) * time.Second)

	// Try again — should get waitSecs * refillRate tokens back
	allowed := 0
	for i := 1; i <= 5; i++ {
		result, err := sendRequest(userID)
		if err != nil {
			fmt.Printf("  Request %d → ERROR: %v\n", i, err)
			continue
		}
		if result.Allowed {
			allowed++
			fmt.Printf("  Request %d → ✅ ALLOWED\n", i)
		} else {
			fmt.Printf("  Request %d → ❌ BLOCKED\n", i)
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
	goroutines := 10

	for i := 1; i <= goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			result, err := sendRequest(userID)
			if err != nil {
				fmt.Printf("  Goroutine %2d → ERROR: %v\n", id, err)
				return
			}

			mu.Lock()
			defer mu.Unlock()

			if result.Allowed {
				allowed++
				fmt.Printf("  Goroutine %2d → ✅ ALLOWED\n", id)
			} else {
				denied++
				fmt.Printf("  Goroutine %2d → ❌ BLOCKED\n", id)
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("\n  Summary: %d allowed, %d blocked\n", allowed, denied)
}

// ---- Test 4: Different users — each user should have independent buckets ----
func testMultipleUsers() {
	fmt.Println("\n========== TEST 4: MULTIPLE USERS (independent buckets) ==========")

	users := []string{"alice", "bob", "charlie"}

	for _, user := range users {
		result, err := sendRequest(user)
		if err != nil {
			fmt.Printf("  %-10s → ERROR: %v\n", user, err)
			continue
		}
		if result.Allowed {
			fmt.Printf("  %-10s → ✅ ALLOWED\n", user)
		} else {
			fmt.Printf("  %-10s → ❌ BLOCKED\n", user)
		}
	}
}

// ---- Test 5: Slow drip — 1 request every second, should always pass ----
func testSlowDrip() {
	fmt.Println("\n========== TEST 5: SLOW DRIP (1 req/sec for 5 secs) ==========")

	for i := 1; i <= 5; i++ {
		result, err := sendRequest(userID)
		if err != nil {
			fmt.Printf("  Request %d → ERROR: %v\n", i, err)
		} else if result.Allowed {
			fmt.Printf("  Request %d → ✅ ALLOWED\n", i)
		} else {
			fmt.Printf("  Request %d → ❌ BLOCKED (unexpected!)\n", i)
		}
		time.Sleep(1 * time.Second)
	}
}

// ---- Helper ----
func sendRequest(userID string) (ResultDetailNew, error) {
	url := fmt.Sprintf("%s?user_id=%s", baseURL, userID)

	resp, err := http.Get(url)
	if err != nil {
		return ResultDetailNew{}, err
	}
	defer resp.Body.Close()

	var result ResultDetailNew
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ResultDetailNew{}, err
	}

	return result, nil
}

func main() {
	fmt.Println("🚀 Rate Limiter Test Suite")
	fmt.Println("   Target:", baseURL)
	fmt.Println("   User:  ", userID)

	testBurst()
	
	// Reset between tests by waiting a bit
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