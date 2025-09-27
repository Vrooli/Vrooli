package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type LoadTestStats struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalDuration   time.Duration
	MinDuration     time.Duration
	MaxDuration     time.Duration
	AvgDuration     time.Duration
	mu              sync.Mutex
	durations       []time.Duration
}

func (s *LoadTestStats) AddRequest(duration time.Duration, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	atomic.AddInt64(&s.TotalRequests, 1)
	if success {
		atomic.AddInt64(&s.SuccessRequests, 1)
	} else {
		atomic.AddInt64(&s.FailedRequests, 1)
	}
	
	s.durations = append(s.durations, duration)
	
	if s.MinDuration == 0 || duration < s.MinDuration {
		s.MinDuration = duration
	}
	if duration > s.MaxDuration {
		s.MaxDuration = duration
	}
}

func (s *LoadTestStats) Calculate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if len(s.durations) > 0 {
		var total time.Duration
		for _, d := range s.durations {
			total += d
		}
		s.AvgDuration = total / time.Duration(len(s.durations))
	}
}

func main() {
	// Get API URL from environment or use default
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		// Try to get port dynamically
		apiURL = "http://localhost:17143"
	}
	
	// Configuration
	concurrentUsers := 50   // Run 50 concurrent connections at a time
	totalMessages := 500    // Total messages to send
	testDuration := 60 * time.Second
	
	fmt.Println("=========================================")
	fmt.Println("AI Chatbot Manager Load Test")
	fmt.Println("=========================================")
	fmt.Printf("API URL: %s\n", apiURL)
	fmt.Printf("Concurrent Users: %d\n", concurrentUsers)
	fmt.Printf("Total Messages: %d\n", totalMessages)
	fmt.Printf("Test Duration: %v\n\n", testDuration)
	
	// Step 1: Create test chatbot
	fmt.Println("[SETUP] Creating test chatbot for load testing...")
	chatbotID := createTestChatbot(apiURL)
	if chatbotID == "" {
		fmt.Println("[FAIL] Failed to create test chatbot")
		os.Exit(1)
	}
	fmt.Printf("[SUCCESS] Created chatbot: %s\n\n", chatbotID)
	
	// Step 2: Warm up
	fmt.Println("[WARMUP] Sending initial requests to warm up the system...")
	for i := 0; i < 10; i++ {
		sendChatMessage(apiURL, chatbotID, fmt.Sprintf("warmup-%d", i), "Warmup message")
	}
	fmt.Println("[SUCCESS] Warmup complete\n")
	
	// Step 3: Load test
	fmt.Println("[LOAD TEST] Starting load test...")
	stats := &LoadTestStats{}
	
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrentUsers)
	
	startTime := time.Now()
	deadline := startTime.Add(testDuration)
	
	// Message sender
	messageCount := 0
	for time.Now().Before(deadline) && messageCount < totalMessages {
		semaphore <- struct{}{} // Acquire slot
		wg.Add(1)
		
		sessionID := fmt.Sprintf("load-test-%d", messageCount)
		messageText := fmt.Sprintf("Load test message %d at %v", messageCount, time.Now())
		messageCount++
		
		go func(sid, msg string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release slot
			
			start := time.Now()
			success := sendChatMessage(apiURL, chatbotID, sid, msg)
			duration := time.Since(start)
			
			stats.AddRequest(duration, success)
			
			if atomic.LoadInt64(&stats.TotalRequests)%100 == 0 {
				fmt.Printf("  Progress: %d messages sent (%.1f%% success rate)\n", 
					atomic.LoadInt64(&stats.TotalRequests),
					float64(atomic.LoadInt64(&stats.SuccessRequests))*100/float64(atomic.LoadInt64(&stats.TotalRequests)))
			}
		}(sessionID, messageText)
		
		// Small delay between spawning goroutines to avoid overwhelming
		time.Sleep(10 * time.Millisecond)
	}
	
	// Wait for all requests to complete or timeout
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()
	
	select {
	case <-done:
		fmt.Println("\n[SUCCESS] All requests completed")
	case <-time.After(testDuration + 10*time.Second):
		fmt.Println("\n[TIMEOUT] Test duration exceeded")
	}
	
	stats.TotalDuration = time.Since(startTime)
	stats.Calculate()
	
	// Step 4: Display results
	fmt.Println("\n=========================================")
	fmt.Println("LOAD TEST RESULTS")
	fmt.Println("=========================================")
	fmt.Printf("Total Requests:    %d\n", stats.TotalRequests)
	fmt.Printf("Successful:        %d (%.1f%%)\n", stats.SuccessRequests, 
		float64(stats.SuccessRequests)*100/float64(stats.TotalRequests))
	fmt.Printf("Failed:            %d (%.1f%%)\n", stats.FailedRequests,
		float64(stats.FailedRequests)*100/float64(stats.TotalRequests))
	fmt.Printf("Test Duration:     %v\n", stats.TotalDuration)
	fmt.Printf("Requests/Second:   %.2f\n", float64(stats.TotalRequests)/stats.TotalDuration.Seconds())
	fmt.Printf("Min Response Time: %v\n", stats.MinDuration)
	fmt.Printf("Max Response Time: %v\n", stats.MaxDuration)
	fmt.Printf("Avg Response Time: %v\n", stats.AvgDuration)
	
	// Performance assessment
	fmt.Println("\n=========================================")
	fmt.Println("PERFORMANCE ASSESSMENT")
	fmt.Println("=========================================")
	
	successRate := float64(stats.SuccessRequests) * 100 / float64(stats.TotalRequests)
	avgResponseMs := stats.AvgDuration.Milliseconds()
	
	if successRate >= 99 && avgResponseMs < 500 {
		fmt.Println("✅ EXCELLENT: System handles load very well")
		fmt.Printf("   - %.1f%% success rate (target: >99%%)\n", successRate)
		fmt.Printf("   - %dms avg response (target: <500ms)\n", avgResponseMs)
	} else if successRate >= 95 && avgResponseMs < 1000 {
		fmt.Println("✓ GOOD: System handles load acceptably")
		fmt.Printf("   - %.1f%% success rate (target: >95%%)\n", successRate)
		fmt.Printf("   - %dms avg response (target: <1000ms)\n", avgResponseMs)
	} else {
		fmt.Println("⚠️ NEEDS IMPROVEMENT: System struggles under load")
		fmt.Printf("   - %.1f%% success rate\n", successRate)
		fmt.Printf("   - %dms avg response time\n", avgResponseMs)
	}
	
	// Clean up
	fmt.Printf("\n[CLEANUP] Deleting test chatbot...\n")
	deleteTestChatbot(apiURL, chatbotID)
	
	// Exit with appropriate code
	if successRate >= 95 {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func createTestChatbot(apiURL string) string {
	reqBody := map[string]interface{}{
		"name":           "Load Test Bot",
		"description":    "Bot for load testing",
		"personality":    "You are a helpful assistant. Keep responses brief.",
		"knowledge_base": "This is a load test bot.",
		"model_config": map[string]interface{}{
			"model":       "llama3.2",
			"temperature": 0.7,
			"max_tokens":  50, // Keep responses short for load testing
		},
	}
	
	jsonData, _ := json.Marshal(reqBody)
	resp, err := http.Post(apiURL+"/api/v1/chatbots", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}
	
	// Check for nested chatbot object
	if chatbot, ok := result["chatbot"].(map[string]interface{}); ok {
		if id, ok := chatbot["id"].(string); ok {
			return id
		}
	}
	// Fallback to direct id field
	if id, ok := result["id"].(string); ok {
		return id
	}
	return ""
}

func sendChatMessage(apiURL, chatbotID, sessionID, message string) bool {
	reqBody := map[string]interface{}{
		"message":    message,
		"session_id": sessionID,
		"context":    map[string]interface{}{"load_test": true},
	}
	
	jsonData, _ := json.Marshal(reqBody)
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(
		fmt.Sprintf("%s/api/v1/chat/%s", apiURL, chatbotID),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	// Drain response body to reuse connection
	io.Copy(io.Discard, resp.Body)
	
	return resp.StatusCode == http.StatusOK
}

func deleteTestChatbot(apiURL, chatbotID string) {
	client := &http.Client{}
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/chatbots/%s", apiURL, chatbotID), nil)
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		fmt.Printf("[INFO] Deleted test chatbot: %s\n", chatbotID)
	}
}