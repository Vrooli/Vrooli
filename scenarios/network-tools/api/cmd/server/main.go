package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Config holds application configuration
type Config struct {
	Port        string
	DatabaseURL string
}

// Server holds server dependencies
type Server struct {
	config *Config
	db     *sql.DB
	router *mux.Router
	client *http.Client
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
	config := &Config{
		Port:        getEnv("PORT", getEnv("API_PORT", "15000")),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/network_tools"),
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

	// Create HTTP client with custom settings
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
	}

	server := &Server{
		config: config,
		db:     db,
		router: mux.NewRouter(),
		client: client,
	}

	server.setupRoutes()
	return server, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Middleware
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)

	// Health check (no auth)
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
	
	// Monitoring endpoints
	api.HandleFunc("/network/targets", s.handleListTargets).Methods("GET")
	api.HandleFunc("/network/targets", s.handleCreateTarget).Methods("POST")
	api.HandleFunc("/network/alerts", s.handleListAlerts).Methods("GET")
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
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// Handler functions
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	dbStatus := "healthy"
	if err := s.db.Ping(); err != nil {
		dbStatus = "unhealthy"
	}
	
	health := map[string]interface{}{
		"status":   "healthy",
		"database": dbStatus,
		"version":  "1.0.0",
		"service":  "network-tools",
		"timestamp": time.Now().UTC(),
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

	// Store in database
	_, err = s.db.Exec(`
		INSERT INTO http_requests (url, method, status_code, response_time_ms, headers, response_headers, response_body)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		req.URL, req.Method, httpResp.StatusCode, responseTime,
		mapToJSON(req.Headers), mapToJSON(httpResp.Headers), httpResp.Body)
	
	if err != nil {
		log.Printf("Failed to store HTTP request: %v", err)
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

	// Store in database
	answersJSON, _ := json.Marshal(answers)
	_, err := s.db.Exec(`
		INSERT INTO dns_queries (query, record_type, dns_server, response_time_ms, answers, authoritative, dnssec_valid)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		req.Query, req.RecordType, req.DNSServer, responseTime, string(answersJSON), false, false)
	
	if err != nil {
		log.Printf("Failed to store DNS query: %v", err)
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
	sendSuccess(w, map[string]string{
		"message": "API test endpoint - implementation pending",
	})
}

func (s *Server) handleListTargets(w http.ResponseWriter, r *http.Request) {
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
		log.Println("Warning: Not running under Vrooli lifecycle management")
	}

	server, err := NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Run(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}