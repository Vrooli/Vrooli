package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandleClaudeCompletion(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		mockResponse   *http.Response
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "valid completion request",
			requestBody: `{
				"model": "claude-3-sonnet",
				"messages": [
					{"role": "user", "content": "Hello"}
				],
				"max_tokens": 100
			}`,
			mockResponse: &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(strings.NewReader(`{
					"content": [{"text": "Hello! How can I help you?"}],
					"usage": {"input_tokens": 10, "output_tokens": 8}
				}`)),
				Header: make(http.Header),
			},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				json.Unmarshal(body, &resp)
				if resp["content"] == nil {
					t.Error("Expected content in response")
				}
			},
		},
		{
			name:           "invalid JSON",
			requestBody:    `{invalid json}`,
			expectedStatus: 400,
		},
		{
			name: "missing required fields",
			requestBody: `{
				"messages": []
			}`,
			expectedStatus: 400,
		},
		{
			name: "API error response",
			requestBody: `{
				"model": "claude-3-sonnet",
				"messages": [{"role": "user", "content": "Test"}],
				"max_tokens": 100
			}`,
			mockResponse: &http.Response{
				StatusCode: 429,
				Body:       io.NopCloser(strings.NewReader(`{"error": {"message": "Rate limited"}}`)),
				Header:     make(http.Header),
			},
			expectedStatus: 429,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalClient := httpClient
			if tt.mockResponse != nil {
				httpClient = &mockHTTPTransport{
					response: tt.mockResponse,
				}
			}
			defer func() { httpClient = originalClient }()
			
			req := httptest.NewRequest("POST", "/api/v1/claude/completion", 
				strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-API-Key", "test-key")
			
			w := httptest.NewRecorder()
			handleClaudeCompletion(w, req)
			
			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)
			
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
			
			if tt.checkResponse != nil {
				tt.checkResponse(t, body)
			}
		})
	}
}

func TestHandleClaudeStream(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		mockEvents     []string
		expectedStatus int
		checkEvents    func(t *testing.T, events []string)
	}{
		{
			name: "successful streaming",
			requestBody: `{
				"model": "claude-3-sonnet",
				"messages": [{"role": "user", "content": "Hello"}],
				"stream": true,
				"max_tokens": 100
			}`,
			mockEvents: []string{
				`event: message_start\ndata: {"type": "message_start"}\n\n`,
				`event: content_block_delta\ndata: {"delta": {"text": "Hello"}}\n\n`,
				`event: content_block_delta\ndata: {"delta": {"text": " there!"}}\n\n`,
				`event: message_stop\ndata: {"type": "message_stop"}\n\n`,
			},
			expectedStatus: 200,
			checkEvents: func(t *testing.T, events []string) {
				if len(events) != 4 {
					t.Errorf("Expected 4 events, got %d", len(events))
				}
			},
		},
		{
			name:           "invalid stream request",
			requestBody:    `{"invalid": true}`,
			expectedStatus: 400,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockEvents != nil {
				originalClient := httpClient
				httpClient = &mockStreamTransport{
					events: tt.mockEvents,
				}
				defer func() { httpClient = originalClient }()
			}
			
			req := httptest.NewRequest("POST", "/api/v1/claude/stream",
				strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-API-Key", "test-key")
			
			w := httptest.NewRecorder()
			handleClaudeStream(w, req)
			
			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
			
			if tt.checkEvents != nil {
				body, _ := io.ReadAll(resp.Body)
				events := strings.Split(string(body), "\n\n")
				tt.checkEvents(t, events)
			}
		})
	}
}

func TestHandleStandardsFix(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		setupMocks     func()
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful fix generation",
			requestBody: `{
				"scenario": "test-scenario",
				"violations": [
					{
						"rule": "api-versioning",
						"file": "main.go",
						"line": 10,
						"message": "Missing API version"
					}
				]
			}`,
			setupMocks: func() {
			},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				json.Unmarshal(body, &resp)
				if resp["fixes"] == nil {
					t.Error("Expected fixes in response")
				}
			},
		},
		{
			name: "invalid scenario",
			requestBody: `{
				"scenario": "",
				"violations": []
			}`,
			expectedStatus: 400,
		},
		{
			name: "no violations provided",
			requestBody: `{
				"scenario": "test-scenario",
				"violations": []
			}`,
			expectedStatus: 400,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}
			
			req := httptest.NewRequest("POST", "/api/v1/standards/fix",
				strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			handleStandardsFix(w, req)
			
			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)
			
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", 
					tt.expectedStatus, resp.StatusCode, string(body))
			}
			
			if tt.checkResponse != nil && resp.StatusCode == 200 {
				tt.checkResponse(t, body)
			}
		})
	}
}

func TestHandleVulnerabilityFix(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful vulnerability fix",
			requestBody: `{
				"scenario": "test-scenario",
				"vulnerabilities": [
					{
						"type": "SQL Injection",
						"severity": "HIGH",
						"file": "db.go",
						"line": 25,
						"description": "Unsanitized user input in SQL query"
					}
				]
			}`,
			expectedStatus: 200,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				json.Unmarshal(body, &resp)
				if resp["fixes"] == nil {
					t.Error("Expected fixes in response")
				}
			},
		},
		{
			name: "multiple vulnerabilities",
			requestBody: `{
				"scenario": "test-scenario",
				"vulnerabilities": [
					{
						"type": "XSS",
						"severity": "MEDIUM",
						"file": "handler.go",
						"line": 50
					},
					{
						"type": "Path Traversal",
						"severity": "HIGH",
						"file": "files.go",
						"line": 100
					}
				]
			}`,
			expectedStatus: 200,
		},
		{
			name:           "malformed request",
			requestBody:    `{"invalid": }`,
			expectedStatus: 400,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/vulnerabilities/fix",
				strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			handleVulnerabilityFix(w, req)
			
			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)
			
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
			
			if tt.checkResponse != nil && resp.StatusCode == 200 {
				tt.checkResponse(t, body)
			}
		})
	}
}

func TestRateLimiting(t *testing.T) {
	originalLimiter := rateLimiter
	rateLimiter = NewRateLimiter(2, time.Second)
	defer func() { rateLimiter = originalLimiter }()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rateLimiter.Allow(r.RemoteAddr) {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		w := httptest.NewRecorder()
		
		handler(w, req)
		
		expectedStatus := http.StatusOK
		if i >= 2 {
			expectedStatus = http.StatusTooManyRequests
		}
		
		if w.Code != expectedStatus {
			t.Errorf("Request %d: expected status %d, got %d", i, expectedStatus, w.Code)
		}
	}
	
	time.Sleep(time.Second)
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()
	handler(w, req)
	
	if w.Code != http.StatusOK {
		t.Error("Expected request to succeed after rate limit window")
	}
}

func TestAPIKeyValidation(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		expectedStatus int
	}{
		{
			name:           "valid API key",
			apiKey:         "valid-key-123",
			expectedStatus: 200,
		},
		{
			name:           "missing API key",
			apiKey:         "",
			expectedStatus: 401,
		},
		{
			name:           "invalid API key",
			apiKey:         "invalid-key",
			expectedStatus: 401,
		},
	}
	
	handler := requireAPIKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestCORSHeaders(t *testing.T) {
	handler := setCORSHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	headers := w.Header()
	
	if headers.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected CORS origin header")
	}
	
	if headers.Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected CORS methods header")
	}
	
	if headers.Get("Access-Control-Allow-Headers") == "" {
		t.Error("Expected CORS headers header")
	}
}

func TestRequestLogging(t *testing.T) {
	var logBuffer bytes.Buffer
	originalLogger := logger
	logger = &TestLogger{buffer: &logBuffer}
	defer func() { logger = originalLogger }()
	
	handler := logRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	
	req := httptest.NewRequest("GET", "/test/endpoint?param=value", nil)
	req.Header.Set("User-Agent", "test-agent")
	
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	logOutput := logBuffer.String()
	
	if !strings.Contains(logOutput, "GET") {
		t.Error("Expected method in log")
	}
	
	if !strings.Contains(logOutput, "/test/endpoint") {
		t.Error("Expected path in log")
	}
	
	if !strings.Contains(logOutput, "200") {
		t.Error("Expected status code in log")
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.HandlerFunc
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "panic recovery",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("test panic")
			}),
			expectedStatus: 500,
			expectedBody:   "Internal Server Error",
		},
		{
			name: "custom error",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Custom error message", http.StatusBadRequest)
			}),
			expectedStatus: 400,
			expectedBody:   "Custom error message",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := recoverPanic(tt.handler)
			
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			
			body := strings.TrimSpace(w.Body.String())
			if !strings.Contains(body, tt.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tt.expectedBody, body)
			}
		})
	}
}

type mockHTTPTransport struct {
	response *http.Response
}

func (m *mockHTTPTransport) Do(req *http.Request) (*http.Response, error) {
	return m.response, nil
}

type mockStreamTransport struct {
	events []string
}

func (m *mockStreamTransport) Do(req *http.Request) (*http.Response, error) {
	body := strings.Join(m.events, "")
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
	}, nil
}

type TestLogger struct {
	buffer *bytes.Buffer
}

func (l *TestLogger) Printf(format string, v ...interface{}) {
	l.buffer.WriteString(strings.TrimSpace(format) + "\n")
}

func requireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" || apiKey != "valid-key-123" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func setCORSHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(wrapped, r)
		
		if logger != nil {
			logger.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
		}
	})
}

func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

type RateLimiter struct {
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (r *RateLimiter) Allow(key string) bool {
	now := time.Now()
	
	if times, exists := r.requests[key]; exists {
		var validTimes []time.Time
		for _, t := range times {
			if now.Sub(t) < r.window {
				validTimes = append(validTimes, t)
			}
		}
		
		if len(validTimes) >= r.limit {
			return false
		}
		
		r.requests[key] = append(validTimes, now)
	} else {
		r.requests[key] = []time.Time{now}
	}
	
	return true
}

var (
	httpClient   HTTPClient
	rateLimiter  *RateLimiter
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func init() {
	httpClient = &http.Client{Timeout: 30 * time.Second}
	rateLimiter = NewRateLimiter(100, time.Minute)
}