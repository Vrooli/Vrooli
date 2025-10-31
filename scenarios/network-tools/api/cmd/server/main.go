package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/lib/pq"
)

// Config holds application configuration
type Config struct {
	Port        string
	DatabaseURL string
}

// Server holds server dependencies
type Server struct {
	config      *Config
	db          *sql.DB
	router      *mux.Router
	client      *http.Client
	rateLimiter *RateLimiter
}

// RateLimiter implements a simple rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-rl.window)
	
	// Clean old requests
	var validRequests []time.Time
	for _, t := range rl.requests[key] {
		if t.After(cutoff) {
			validRequests = append(validRequests, t)
		}
	}
	
	// Check if under limit
	if len(validRequests) >= rl.limit {
		return false
	}
	
	// Add current request
	validRequests = append(validRequests, now)
	rl.requests[key] = validRequests
	
	return true
}

// Response is a generic API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// HTTPRequest represents an HTTP request
type HTTPRequest struct {
	URL      string                 `json:"url"`
	Method   string                 `json:"method"`
	Headers  map[string]string      `json:"headers,omitempty"`
	Body     interface{}            `json:"body,omitempty"`
	Options  HTTPOptions            `json:"options,omitempty"`
}

// HTTPOptions for request configuration
type HTTPOptions struct {
	TimeoutMs       int  `json:"timeout_ms,omitempty"`
	FollowRedirects bool `json:"follow_redirects"`
	VerifySSL       bool `json:"verify_ssl"`
	MaxRetries      int  `json:"max_retries,omitempty"`
}

// HTTPResponse represents an HTTP response
type HTTPResponse struct {
	StatusCode    int               `json:"status_code"`
	Headers       map[string]string `json:"headers"`
	Body          string            `json:"body"`
	ResponseTimeMs int64            `json:"response_time_ms"`
	FinalURL      string            `json:"final_url"`
	SSLInfo       interface{}       `json:"ssl_info,omitempty"`
	RedirectChain []string          `json:"redirect_chain,omitempty"`
}

// DNSRequest represents a DNS query request
type DNSRequest struct {
	Query      string     `json:"query"`
	RecordType string     `json:"record_type"`
	DNSServer  string     `json:"dns_server,omitempty"`
	Options    DNSOptions `json:"options,omitempty"`
}

// DNSOptions for DNS queries
type DNSOptions struct {
	TimeoutMs      int  `json:"timeout_ms,omitempty"`
	Recursive      bool `json:"recursive"`
	ValidateDNSSEC bool `json:"validate_dnssec"`
}

// DNSResponse represents a DNS query response
type DNSResponse struct {
	Query          string      `json:"query"`
	RecordType     string      `json:"record_type"`
	Answers        []DNSAnswer `json:"answers"`
	ResponseTimeMs int64       `json:"response_time_ms"`
	Authoritative  bool        `json:"authoritative"`
	DNSSECValid    bool        `json:"dnssec_valid"`
}

// DNSAnswer represents a DNS answer record
type DNSAnswer struct {
	Name string `json:"name"`
	Type string `json:"type"`
	TTL  int    `json:"ttl"`
	Data string `json:"data"`
}

// ConnectivityRequest for network connectivity tests
type ConnectivityRequest struct {
	Target  string              `json:"target"`
	TestType string             `json:"test_type"`
	Options ConnectivityOptions `json:"options,omitempty"`
}

// ConnectivityOptions for connectivity tests
type ConnectivityOptions struct {
	Count      int `json:"count,omitempty"`
	TimeoutMs  int `json:"timeout_ms,omitempty"`
	PacketSize int `json:"packet_size,omitempty"`
	IntervalMs int `json:"interval_ms,omitempty"`
}

// ConnectivityResponse for connectivity test results
type ConnectivityResponse struct {
	Target     string                  `json:"target"`
	TestType   string                  `json:"test_type"`
	Statistics ConnectivityStatistics `json:"statistics"`
	RouteHops  []string               `json:"route_hops,omitempty"`
}

// ConnectivityStatistics for connectivity metrics
type ConnectivityStatistics struct {
	PacketsSent        int     `json:"packets_sent"`
	PacketsReceived    int     `json:"packets_received"`
	PacketLossPercent  float64 `json:"packet_loss_percent"`
	MinRTTMs           float64 `json:"min_rtt_ms"`
	AvgRTTMs           float64 `json:"avg_rtt_ms"`
	MaxRTTMs           float64 `json:"max_rtt_ms"`
	StdDevRTTMs        float64 `json:"stddev_rtt_ms"`
}

// NewServer creates a new server instance
func NewServer() (*Server, error) {
	// Build database URL from components or use provided URL
	// Priority: DATABASE_URL > POSTGRES_URL > component-based construction
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("POSTGRES_URL")
	}
	if dbURL == "" {
		// Build from individual components with Vrooli resource defaults
		dbHost := getEnv("POSTGRES_HOST", getEnv("DB_HOST", "localhost"))
		dbPort := getEnv("POSTGRES_PORT", getEnv("DB_PORT", "5433"))
		dbUser := getEnv("POSTGRES_USER", getEnv("DB_USER", "vrooli"))
		dbPass := getEnv("POSTGRES_PASSWORD", getEnv("DB_PASSWORD", ""))
		dbName := getEnv("POSTGRES_DB", getEnv("DB_NAME", "vrooli"))
		dbSSL := getEnv("POSTGRES_SSLMODE", getEnv("DB_SSL_MODE", "disable"))

		if dbPass == "" {
			return nil, fmt.Errorf("database password not configured - set POSTGRES_PASSWORD or DB_PASSWORD environment variable")
		}
		
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			dbUser, dbPass, dbHost, dbPort, dbName, dbSSL)
	}
	
	config := &Config{
		Port:        getEnv("PORT", getEnv("API_PORT", "15000")),
		DatabaseURL: dbURL,
	}

	// Connect to database
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	// Initialize database schema
	if err := InitializeDatabase(db); err != nil {
		log.Printf("Warning: Failed to initialize database schema: %v", err)
		// Continue anyway - schema might already exist
	}

	// Create HTTP client with custom settings
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
	}

	// Create rate limiter with configurable limits
	rateLimit := 100
	if rl := os.Getenv("RATE_LIMIT_REQUESTS"); rl != "" {
		if parsed, err := strconv.Atoi(rl); err == nil {
			rateLimit = parsed
		}
	}
	
	rateWindow := time.Minute
	if rw := os.Getenv("RATE_LIMIT_WINDOW"); rw != "" {
		if parsed, err := time.ParseDuration(rw); err == nil {
			rateWindow = parsed
		}
	}
	
	rateLimiter := NewRateLimiter(rateLimit, rateWindow)
	log.Printf("Rate limiting: %d requests per %v", rateLimit, rateWindow)
	
	server := &Server{
		config:      config,
		db:          db,
		router:      mux.NewRouter(),
		client:      client,
		rateLimiter: rateLimiter,
	}

	server.setupRoutes()
	return server, nil
}

// rateLimitMiddleware implements rate limiting per IP
func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for health checks
		if strings.HasSuffix(r.URL.Path, "/health") {
			next.ServeHTTP(w, r)
			return
		}
		
		// Get client IP
		clientIP := r.RemoteAddr
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			clientIP = strings.Split(xff, ",")[0]
		} else if xri := r.Header.Get("X-Real-IP"); xri != "" {
			clientIP = xri
		}
		
		// Check rate limit
		if !s.rateLimiter.Allow(clientIP) {
			sendError(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Middleware
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)
	s.router.Use(s.rateLimitMiddleware)
	s.router.Use(authMiddleware)

	// Health check (no auth required due to authMiddleware logic)
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/health", s.handleHealth).Methods("GET", "OPTIONS")

	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Network operations
	api.HandleFunc("/network/http", s.handleHTTPRequest).Methods("POST", "OPTIONS")
	api.HandleFunc("/network/dns", s.handleDNSQuery).Methods("POST", "OPTIONS")
	api.HandleFunc("/network/test/connectivity", s.handleConnectivityTest).Methods("POST", "OPTIONS")
	api.HandleFunc("/network/scan", s.handleNetworkScan).Methods("POST", "OPTIONS")
	api.HandleFunc("/network/api/test", s.handleAPITest).Methods("POST", "OPTIONS")
	api.HandleFunc("/network/ssl/validate", s.handleSSLValidation).Methods("POST", "OPTIONS")
	
	// Monitoring endpoints
	api.HandleFunc("/network/targets", s.handleListTargets).Methods("GET")
	api.HandleFunc("/network/targets", s.handleCreateTarget).Methods("POST")
	api.HandleFunc("/network/alerts", s.handleListAlerts).Methods("GET")

	// API Management endpoints
	api.HandleFunc("/api/definitions", s.handleListAPIDefinitions).Methods("GET", "OPTIONS")
	api.HandleFunc("/api/definitions", s.handleCreateAPIDefinition).Methods("POST", "OPTIONS")
	api.HandleFunc("/api/definitions/{id}", s.handleGetAPIDefinition).Methods("GET", "OPTIONS")
	api.HandleFunc("/api/definitions/{id}", s.handleUpdateAPIDefinition).Methods("PUT", "OPTIONS")
	api.HandleFunc("/api/definitions/{id}", s.handleDeleteAPIDefinition).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/api/discover", s.handleDiscoverAPIEndpoints).Methods("POST", "OPTIONS")
}

// Middleware functions
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		
		// Get allowed origins from environment or use defaults
		allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
		var allowedOrigins []string
		
		if allowedOriginsStr != "" {
			allowedOrigins = strings.Split(allowedOriginsStr, ",")
			for i, origin := range allowedOrigins {
				allowedOrigins[i] = strings.TrimSpace(origin)
			}
		} else {
			// Default allowed origins
			uiPort := getEnv("UI_PORT", "35000")
			allowedOrigins = []string{
				fmt.Sprintf("http://localhost:%s", uiPort),
				"http://localhost:3000",
			}
			
			// Add dynamic UI port range in development
			if os.Getenv("VROOLI_ENV") == "development" {
				for i := 35000; i <= 35005; i++ {
					allowedOrigins = append(allowedOrigins, fmt.Sprintf("http://localhost:%d", i))
				}
			}
		}
		
		// Check if origin is allowed
		originAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				originAllowed = true
				break
			}
		}
		
		// In development, be more permissive
		if os.Getenv("VROOLI_ENV") == "development" && origin != "" && strings.HasPrefix(origin, "http://localhost:") {
			originAllowed = true
		}
		
		if originAllowed && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		// Note: Requests without Origin header (e.g., CLI, curl) will not get CORS headers
		// This is correct behavior - CORS is only needed for browser-based requests
		
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		w.Header().Set("Access-Control-Max-Age", "3600")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health endpoints
		if strings.HasSuffix(r.URL.Path, "/health") {
			next.ServeHTTP(w, r)
			return
		}
		
		// Skip auth for OPTIONS requests
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}
		
		// Get API key enforcement mode
		authMode := getEnv("AUTH_MODE", "development")
		if os.Getenv("VROOLI_ENV") == "production" {
			authMode = "production"
		}
		
		// Check for API key in header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			// Check Authorization header as fallback
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				apiKey = strings.TrimPrefix(auth, "Bearer ")
			}
		}
		
		// Validate based on auth mode
		switch authMode {
		case "production", "strict":
			// Production mode: require valid API key
			expectedKey := os.Getenv("NETWORK_TOOLS_API_KEY")
			if expectedKey == "" {
				// API key required but not configured
				log.Printf("Error: No API key configured in production mode. Configure authentication via environment variables.")
				sendError(w, "Service misconfigured - API key not set", http.StatusInternalServerError)
				return
			}
			
			if apiKey == "" {
				sendError(w, "API key required", http.StatusUnauthorized)
				return
			}
			
			if apiKey != expectedKey {
				// Log failed attempts for security monitoring
				log.Printf("Invalid API key attempt from %s", r.RemoteAddr)
				sendError(w, "Invalid API key", http.StatusUnauthorized)
				return
			}
			
		case "optional":
			// Optional mode: validate if provided, but don't require
			expectedKey := os.Getenv("NETWORK_TOOLS_API_KEY")
			if expectedKey != "" && apiKey != "" && apiKey != expectedKey {
				sendError(w, "Invalid API key", http.StatusUnauthorized)
				return
			}
			
		case "development", "disabled":
			// Development mode: no authentication required
			// This allows easy testing and CLI access
			
		default:
			// Unknown mode, default to development behavior
			log.Printf("Unknown AUTH_MODE: %s, defaulting to development", authMode)
		}
		
		next.ServeHTTP(w, r)
	})
}

// Handler functions
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	dbConnected := false
	var dbLatency *float64
	var dbError map[string]interface{}

	if s.db != nil {
		start := time.Now()
		if err := s.db.Ping(); err == nil {
			dbConnected = true
			latency := float64(time.Since(start).Milliseconds())
			dbLatency = &latency
		} else {
			dbError = map[string]interface{}{
				"code":      "CONNECTION_FAILED",
				"message":   "Failed to connect to database",
				"category":  "resource",
				"retryable": true,
			}
		}
	} else {
		dbError = map[string]interface{}{
			"code":      "NOT_CONFIGURED",
			"message":   "Database connection not configured",
			"category":  "configuration",
			"retryable": false,
		}
	}

	// Determine overall status and readiness
	status := "healthy"
	readiness := true

	if !dbConnected {
		status = "degraded"
		readiness = false
	}

	health := map[string]interface{}{
		"status":    status,
		"service":   "network-tools",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": readiness,
		"version":   "1.0.0",
		"dependencies": map[string]interface{}{
			"database": map[string]interface{}{
				"connected":  dbConnected,
				"latency_ms": dbLatency,
				"error":      dbError,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (s *Server) handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	var req HTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.URL == "" {
		sendError(w, "URL is required", http.StatusBadRequest)
		return
	}
	
	// Validate URL format
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		sendError(w, fmt.Sprintf("Invalid URL format: %v", err), http.StatusBadRequest)
		return
	}
	
	// Ensure URL has a scheme
	if parsedURL.Scheme == "" {
		sendError(w, "URL must include a scheme (http:// or https://)", http.StatusBadRequest)
		return
	}
	
	// Ensure URL has a host
	if parsedURL.Host == "" {
		sendError(w, "URL must include a host", http.StatusBadRequest)
		return
	}
	
	if req.Method == "" {
		req.Method = "GET"
	}

	// Create HTTP request
	startTime := time.Now()
	
	httpReq, err := http.NewRequest(req.Method, req.URL, nil)
	if err != nil {
		sendError(w, fmt.Sprintf("Failed to create request: %v", err), http.StatusBadRequest)
		return
	}

	// Set headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Configure client
	client := &http.Client{
		Timeout: time.Duration(req.Options.TimeoutMs) * time.Millisecond,
	}
	if req.Options.TimeoutMs == 0 {
		client.Timeout = 30 * time.Second
	}

	// Execute request
	resp, err := client.Do(httpReq)
	if err != nil {
		sendError(w, fmt.Sprintf("Request failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sendError(w, fmt.Sprintf("Failed to read response: %v", err), http.StatusInternalServerError)
		return
	}

	// Build response
	responseTime := time.Since(startTime).Milliseconds()
	
	httpResp := HTTPResponse{
		StatusCode:     resp.StatusCode,
		Headers:        make(map[string]string),
		Body:          string(body),
		ResponseTimeMs: responseTime,
		FinalURL:      resp.Request.URL.String(),
	}

	// Copy headers
	for k := range resp.Header {
		httpResp.Headers[k] = resp.Header.Get(k)
	}

	// Store in database (skip if no database connection, for testing)
	if s.db != nil {
		_, err = s.db.Exec(`
			INSERT INTO http_requests (url, method, status_code, response_time_ms, headers, response_headers, response_body)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			req.URL, req.Method, httpResp.StatusCode, responseTime,
			mapToJSON(req.Headers), mapToJSON(httpResp.Headers), httpResp.Body)

		if err != nil {
			log.Printf("Failed to store HTTP request: %v", err)
		}
	}

	sendSuccess(w, httpResp)
}

func (s *Server) handleDNSQuery(w http.ResponseWriter, r *http.Request) {
	var req DNSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Query == "" {
		sendError(w, "Query is required", http.StatusBadRequest)
		return
	}
	if req.RecordType == "" {
		req.RecordType = "A"
	}

	startTime := time.Now()
	
	// Perform DNS lookup (simplified)
	var answers []DNSAnswer
	
	switch req.RecordType {
	case "A":
		ips, err := net.LookupIP(req.Query)
		if err != nil {
			sendError(w, fmt.Sprintf("DNS lookup failed: %v", err), http.StatusInternalServerError)
			return
		}
		for _, ip := range ips {
			if ipv4 := ip.To4(); ipv4 != nil {
				answers = append(answers, DNSAnswer{
					Name: req.Query,
					Type: "A",
					TTL:  300,
					Data: ipv4.String(),
				})
			}
		}
	case "CNAME":
		cname, err := net.LookupCNAME(req.Query)
		if err != nil {
			sendError(w, fmt.Sprintf("DNS lookup failed: %v", err), http.StatusInternalServerError)
			return
		}
		answers = append(answers, DNSAnswer{
			Name: req.Query,
			Type: "CNAME",
			TTL:  300,
			Data: cname,
		})
	case "MX":
		mxRecords, err := net.LookupMX(req.Query)
		if err != nil {
			sendError(w, fmt.Sprintf("DNS lookup failed: %v", err), http.StatusInternalServerError)
			return
		}
		for _, mx := range mxRecords {
			answers = append(answers, DNSAnswer{
				Name: req.Query,
				Type: "MX",
				TTL:  300,
				Data: fmt.Sprintf("%d %s", mx.Pref, mx.Host),
			})
		}
	case "TXT":
		txtRecords, err := net.LookupTXT(req.Query)
		if err != nil {
			sendError(w, fmt.Sprintf("DNS lookup failed: %v", err), http.StatusInternalServerError)
			return
		}
		for _, txt := range txtRecords {
			answers = append(answers, DNSAnswer{
				Name: req.Query,
				Type: "TXT",
				TTL:  300,
				Data: txt,
			})
		}
	default:
		sendError(w, "Unsupported record type", http.StatusBadRequest)
		return
	}

	responseTime := time.Since(startTime).Milliseconds()
	
	dnsResp := DNSResponse{
		Query:          req.Query,
		RecordType:     req.RecordType,
		Answers:        answers,
		ResponseTimeMs: responseTime,
		Authoritative:  false,
		DNSSECValid:    false,
	}

	// Store in database (skip if no database connection, for testing)
	if s.db != nil {
		answersJSON, _ := json.Marshal(answers)
		_, err := s.db.Exec(`
			INSERT INTO dns_queries (query, record_type, dns_server, response_time_ms, answers, authoritative, dnssec_valid)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			req.Query, req.RecordType, req.DNSServer, responseTime, string(answersJSON), false, false)

		if err != nil {
			log.Printf("Failed to store DNS query: %v", err)
		}
	}

	sendSuccess(w, dnsResp)
}

func (s *Server) handleConnectivityTest(w http.ResponseWriter, r *http.Request) {
	var req ConnectivityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Target == "" {
		sendError(w, "Target is required", http.StatusBadRequest)
		return
	}
	if req.TestType == "" {
		req.TestType = "ping"
	}

	// Validate test_type is valid
	validTestTypes := map[string]bool{
		"ping":       true,
		"traceroute": true,
		"mtr":        true,
		"bandwidth":  true,
		"latency":    true,
		"tcp":        true,
	}
	if !validTestTypes[req.TestType] {
		sendError(w, fmt.Sprintf("Invalid test_type '%s'. Valid types: ping, traceroute, mtr, bandwidth, latency, tcp", req.TestType), http.StatusBadRequest)
		return
	}

	// Simple connectivity test (TCP connection)
	startTime := time.Now()
	
	// Try to resolve the target
	ips, err := net.LookupIP(req.Target)
	if err != nil {
		// If lookup fails, try as IP
		ip := net.ParseIP(req.Target)
		if ip == nil {
			sendError(w, fmt.Sprintf("Failed to resolve target: %v", err), http.StatusBadRequest)
			return
		}
		ips = []net.IP{ip}
	}

	// Simple TCP connectivity test to common ports
	testPorts := []int{80, 443, 22, 3389}
	successCount := 0
	
	for _, port := range testPorts {
		address := fmt.Sprintf("%s:%d", ips[0].String(), port)
		conn, err := net.DialTimeout("tcp", address, 2*time.Second)
		if err == nil {
			successCount++
			conn.Close()
		}
	}

	responseTime := time.Since(startTime).Milliseconds()
	
	// Calculate simple statistics
	stats := ConnectivityStatistics{
		PacketsSent:       len(testPorts),
		PacketsReceived:   successCount,
		PacketLossPercent: float64(len(testPorts)-successCount) / float64(len(testPorts)) * 100,
		AvgRTTMs:         float64(responseTime) / float64(len(testPorts)),
		MinRTTMs:         float64(responseTime) / float64(len(testPorts)),
		MaxRTTMs:         float64(responseTime) / float64(len(testPorts)),
	}

	connResp := ConnectivityResponse{
		Target:     req.Target,
		TestType:   req.TestType,
		Statistics: stats,
	}

	sendSuccess(w, connResp)
}

func (s *Server) handleNetworkScan(w http.ResponseWriter, r *http.Request) {
	// Simplified port scan implementation
	var req struct {
		Target   string   `json:"target"`
		ScanType string   `json:"scan_type"`
		Ports    []int    `json:"ports"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Target == "" {
		sendError(w, "Target is required", http.StatusBadRequest)
		return
	}

	// Default ports if not specified
	if len(req.Ports) == 0 {
		req.Ports = []int{21, 22, 23, 25, 80, 443, 3306, 5432, 8080, 8443}
	}

	results := make([]map[string]interface{}, 0)
	
	for _, port := range req.Ports {
		address := fmt.Sprintf("%s:%d", req.Target, port)
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		
		state := "closed"
		if err == nil {
			state = "open"
			conn.Close()
		} else if strings.Contains(err.Error(), "refused") {
			state = "closed"
		} else {
			state = "filtered"
		}
		
		results = append(results, map[string]interface{}{
			"port":     port,
			"protocol": "tcp",
			"state":    state,
			"service":  getServiceName(port),
		})
	}

	sendSuccess(w, map[string]interface{}{
		"target":  req.Target,
		"results": results,
	})
}

func (s *Server) handleAPITest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIDefinitionID string `json:"api_definition_id,omitempty"`
		BaseURL        string `json:"base_url,omitempty"`
		SpecURL        string `json:"spec_url,omitempty"`
		TestSuite      []struct {
			Endpoint  string `json:"endpoint"`
			Method    string `json:"method"`
			TestCases []struct {
				Name           string      `json:"name"`
				Input          interface{} `json:"input,omitempty"`
				ExpectedStatus int         `json:"expected_status"`
				ExpectedSchema interface{} `json:"expected_schema,omitempty"`
				Assertions     []string    `json:"assertions,omitempty"`
			} `json:"test_cases"`
		} `json:"test_suite"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate input
	baseURL := req.BaseURL
	if baseURL == "" && req.APIDefinitionID != "" {
		// Look up API definition from database (only if database is configured)
		if s.db == nil {
			sendError(w, "Database not configured - cannot look up API definition", http.StatusBadRequest)
			return
		}
		var url string
		err := s.db.QueryRow("SELECT base_url FROM api_definitions WHERE id = $1", req.APIDefinitionID).Scan(&url)
		if err != nil {
			sendError(w, "API definition not found", http.StatusNotFound)
			return
		}
		baseURL = url
	}
	
	if baseURL == "" {
		sendError(w, "Base URL or API definition ID required", http.StatusBadRequest)
		return
	}
	
	// Execute test suite
	results := make([]map[string]interface{}, 0)
	
	for _, test := range req.TestSuite {
		endpoint := test.Endpoint
		testResults := map[string]interface{}{
			"endpoint":       endpoint,
			"total_tests":    len(test.TestCases),
			"passed_tests":   0,
			"failed_tests":   0,
			"execution_time_ms": 0,
			"failures":       make([]map[string]interface{}, 0),
		}
		
		for _, testCase := range test.TestCases {
			startTime := time.Now()
			
			// Prepare request
			url := baseURL + endpoint
			method := test.Method
			if method == "" {
				method = "GET"
			}
			
			// Create request with input data
			var bodyReader io.Reader
			if testCase.Input != nil {
				bodyData, _ := json.Marshal(testCase.Input)
				bodyReader = strings.NewReader(string(bodyData))
			}
			
			req, err := http.NewRequest(method, url, bodyReader)
			if err != nil {
				testResults["failed_tests"] = testResults["failed_tests"].(int) + 1
				failures := testResults["failures"].([]map[string]interface{})
				testResults["failures"] = append(failures, map[string]interface{}{
					"test_name": testCase.Name,
					"error":     fmt.Sprintf("Failed to create request: %v", err),
				})
				continue
			}
			
			if bodyReader != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			
			// Execute request
			resp, err := s.client.Do(req)
			if err != nil {
				testResults["failed_tests"] = testResults["failed_tests"].(int) + 1
				failures := testResults["failures"].([]map[string]interface{})
				testResults["failures"] = append(failures, map[string]interface{}{
					"test_name": testCase.Name,
					"error":     fmt.Sprintf("Request failed: %v", err),
				})
				continue
			}
			defer resp.Body.Close()
			
			// Check status code
			if testCase.ExpectedStatus != 0 && resp.StatusCode != testCase.ExpectedStatus {
				testResults["failed_tests"] = testResults["failed_tests"].(int) + 1
				failures := testResults["failures"].([]map[string]interface{})
				testResults["failures"] = append(failures, map[string]interface{}{
					"test_name": testCase.Name,
					"error":     fmt.Sprintf("Expected status %d, got %d", testCase.ExpectedStatus, resp.StatusCode),
				})
			} else {
				testResults["passed_tests"] = testResults["passed_tests"].(int) + 1
			}
			
			execTime := time.Since(startTime).Milliseconds()
			testResults["execution_time_ms"] = testResults["execution_time_ms"].(int) + int(execTime)
		}
		
		results = append(results, testResults)
	}
	
	// Calculate overall success rate
	totalTests := 0
	passedTests := 0
	for _, result := range results {
		totalTests += result["total_tests"].(int)
		passedTests += result["passed_tests"].(int)
	}
	
	successRate := 0.0
	if totalTests > 0 {
		successRate = float64(passedTests) / float64(totalTests) * 100
	}
	
	sendSuccess(w, map[string]interface{}{
		"test_results":        results,
		"overall_success_rate": successRate,
	})
}

func (s *Server) handleSSLValidation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL     string `json:"url"`
		Options struct {
			CheckExpiry      bool `json:"check_expiry"`
			CheckChain       bool `json:"check_chain"`
			CheckHostname    bool `json:"check_hostname"`
			TimeoutMs        int  `json:"timeout_ms"`
		} `json:"options"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate URL
	if req.URL == "" {
		sendError(w, "URL is required", http.StatusBadRequest)
		return
	}
	
	// Parse URL to extract hostname
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		sendError(w, fmt.Sprintf("Invalid URL: %v", err), http.StatusBadRequest)
		return
	}
	
	// Default to HTTPS if no scheme
	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "https"
	}
	
	// Get port (default to 443 for HTTPS)
	port := parsedURL.Port()
	if port == "" {
		if parsedURL.Scheme == "https" {
			port = "443"
		} else {
			sendError(w, "HTTPS URL required for SSL validation", http.StatusBadRequest)
			return
		}
	}
	
	// Build address with port if missing
	address := parsedURL.Host
	if !strings.Contains(address, ":") {
		address = fmt.Sprintf("%s:%s", parsedURL.Hostname(), port)
	}
	
	// Set timeout
	timeout := time.Duration(req.Options.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	
	// Connect to server
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", address, &tls.Config{
		InsecureSkipVerify: true, // We'll validate manually
	})
	if err != nil {
		sendError(w, fmt.Sprintf("Failed to connect: %v", err), http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	
	// Get certificate chain
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		sendError(w, "No certificates found", http.StatusInternalServerError)
		return
	}
	
	// Primary certificate
	cert := certs[0]
	
	// Build response
	response := map[string]interface{}{
		"url":        req.URL,
		"hostname":   parsedURL.Hostname(),
		"port":       port,
		"valid":      true,
		"issues":     []string{},
		"warnings":   []string{},
		"certificate": map[string]interface{}{
			"subject":     cert.Subject.String(),
			"issuer":      cert.Issuer.String(),
			"serial":      cert.SerialNumber.String(),
			"not_before":  cert.NotBefore,
			"not_after":   cert.NotAfter,
			"dns_names":   cert.DNSNames,
			"ip_addresses": cert.IPAddresses,
			"key_usage":   cert.KeyUsage,
			"version":     cert.Version,
			"signature_algorithm": cert.SignatureAlgorithm.String(),
		},
		"chain_length": len(certs),
		"cipher_suite": tls.CipherSuiteName(conn.ConnectionState().CipherSuite),
		"tls_version":  tlsVersionString(conn.ConnectionState().Version),
	}
	
	issues := []string{}
	warnings := []string{}
	
	// Check expiry
	if req.Options.CheckExpiry {
		now := time.Now()
		if now.After(cert.NotAfter) {
			issues = append(issues, fmt.Sprintf("Certificate expired on %s", cert.NotAfter))
			response["valid"] = false
		} else if now.Before(cert.NotBefore) {
			issues = append(issues, fmt.Sprintf("Certificate not yet valid (starts %s)", cert.NotBefore))
			response["valid"] = false
		} else {
			daysUntilExpiry := cert.NotAfter.Sub(now).Hours() / 24
			if daysUntilExpiry < 30 {
				warnings = append(warnings, fmt.Sprintf("Certificate expires in %.0f days", daysUntilExpiry))
			}
			response["days_remaining"] = int(daysUntilExpiry)
		}
	}
	
	// Check hostname
	if req.Options.CheckHostname {
		if err := cert.VerifyHostname(parsedURL.Hostname()); err != nil {
			issues = append(issues, fmt.Sprintf("Hostname verification failed: %v", err))
			response["valid"] = false
		}
	}
	
	// Check chain
	if req.Options.CheckChain && len(certs) > 1 {
		roots := x509.NewCertPool()
		intermediates := x509.NewCertPool()
		
		// Add system roots
		if systemRoots, err := x509.SystemCertPool(); err == nil {
			roots = systemRoots
		}
		
		// Add intermediates
		for i := 1; i < len(certs); i++ {
			intermediates.AddCert(certs[i])
		}
		
		opts := x509.VerifyOptions{
			Intermediates: intermediates,
			Roots:         roots,
		}
		
		if _, err := cert.Verify(opts); err != nil {
			warnings = append(warnings, fmt.Sprintf("Chain validation warning: %v", err))
		}
	}
	
	response["issues"] = issues
	response["warnings"] = warnings
	
	// Store in database (skip if no database connection, for testing)
	if s.db != nil {
		validationResult, _ := json.Marshal(response)
		_, err = s.db.Exec(`
			INSERT INTO scan_results (target_id, scan_type, status, results, severity_score)
			VALUES (
				(SELECT id FROM network_targets WHERE address = $1 LIMIT 1),
				'ssl_check',
				$2,
				$3,
			$4
		)`,
			parsedURL.Host,
			func() string { if response["valid"].(bool) { return "success" } else { return "failed" } }(),
			string(validationResult),
			func() float64 { if response["valid"].(bool) { return 0.0 } else { return 8.0 } }(),
		)

		if err != nil {
			log.Printf("Failed to store SSL validation result: %v", err)
		}
	}
	
	sendSuccess(w, response)
}

// Helper function to get TLS version string
func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}

func (s *Server) handleListTargets(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		sendError(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	rows, err := s.db.Query(`
		SELECT id, name, target_type, address, port, protocol, is_active, created_at
		FROM network_targets
		WHERE is_active = true
		ORDER BY created_at DESC
		LIMIT 100`)

	if err != nil {
		sendError(w, fmt.Sprintf("Failed to query targets: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	targets := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, name, targetType, address string
		var port sql.NullInt64
		var protocol sql.NullString
		var isActive bool
		var createdAt time.Time
		
		err := rows.Scan(&id, &name, &targetType, &address, &port, &protocol, &isActive, &createdAt)
		if err != nil {
			continue
		}
		
		target := map[string]interface{}{
			"id":          id,
			"name":        name,
			"target_type": targetType,
			"address":     address,
			"is_active":   isActive,
			"created_at":  createdAt,
		}
		
		if port.Valid {
			target["port"] = port.Int64
		}
		if protocol.Valid {
			target["protocol"] = protocol.String
		}
		
		targets = append(targets, target)
	}

	sendSuccess(w, targets)
}

func (s *Server) handleCreateTarget(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		sendError(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	var target struct {
		Name       string   `json:"name"`
		TargetType string   `json:"target_type"`
		Address    string   `json:"address"`
		Port       int      `json:"port,omitempty"`
		Protocol   string   `json:"protocol,omitempty"`
		Tags       []string `json:"tags,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()

	_, err := s.db.Exec(`
		INSERT INTO network_targets (id, name, target_type, address, port, protocol, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, target.Name, target.TargetType, target.Address, target.Port, target.Protocol, pq.Array(target.Tags))

	if err != nil {
		sendError(w, fmt.Sprintf("Failed to create target: %v", err), http.StatusInternalServerError)
		return
	}

	sendSuccess(w, map[string]string{"id": id})
}

func (s *Server) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		sendError(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	rows, err := s.db.Query(`
		SELECT a.id, a.alert_type, a.severity, a.title, a.message, a.created_at, nt.name
		FROM alerts a
		JOIN network_targets nt ON a.target_id = nt.id
		WHERE a.is_active = true
		ORDER BY a.severity DESC, a.created_at DESC
		LIMIT 50`)

	if err != nil {
		sendError(w, fmt.Sprintf("Failed to query alerts: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	alerts := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, alertType, severity, title, message, targetName string
		var createdAt time.Time
		
		err := rows.Scan(&id, &alertType, &severity, &title, &message, &createdAt, &targetName)
		if err != nil {
			continue
		}
		
		alerts = append(alerts, map[string]interface{}{
			"id":          id,
			"alert_type":  alertType,
			"severity":    severity,
			"title":       title,
			"message":     message,
			"target_name": targetName,
			"created_at":  createdAt,
		})
	}

	sendSuccess(w, alerts)
}

// Helper functions
func sendSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    data,
	})
}

func sendError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Error:   message,
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func mapToJSON(m map[string]string) string {
	b, _ := json.Marshal(m)
	return string(b)
}

func getServiceName(port int) string {
	services := map[int]string{
		21:   "ftp",
		22:   "ssh",
		23:   "telnet",
		25:   "smtp",
		80:   "http",
		443:  "https",
		3306: "mysql",
		5432: "postgresql",
		8080: "http-proxy",
		8443: "https-alt",
	}
	if service, ok := services[port]; ok {
		return service
	}
	return "unknown"
}

// Run starts the server
func (s *Server) Run() error {
	srv := &http.Server{
		Addr:         ":" + s.config.Port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Network Tools API server starting on port %s", s.config.Port)
	return srv.ListenAndServe()
}

func main() {
	// Check if running under lifecycle management
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		log.Println("ERROR: Network Tools must be started via Vrooli lifecycle system")
		log.Println("Please use: vrooli scenario start network-tools")
		log.Println("Or from scenario directory: make start")
		log.Println("Direct execution bypasses critical infrastructure (process management, port allocation, health monitoring)")
		os.Exit(1)
	}

	server, err := NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Run(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}