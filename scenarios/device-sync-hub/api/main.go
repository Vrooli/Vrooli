package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/rs/cors"
)

// Configuration
type Config struct {
	Port               string
	UIPort             string
	AuthServiceURL     string
	StoragePath        string
	MaxFileSize        int64
	DefaultExpiryHours int
	ThumbnailSize      int
}

// User represents an authenticated user
type User struct {
	ID    string   `json:"id"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

// Device represents a connected device
type Device struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Platform     string    `json:"platform"`
	LastSeen     time.Time `json:"last_seen"`
	Capabilities []string  `json:"capabilities"`
	IsOnline     bool      `json:"is_online"`
}

// SyncItem represents an item to sync between devices
type SyncItem struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	Type        string                 `json:"type"` // clipboard, file, notification, etc.
	Content     map[string]interface{} `json:"content"`
	SourceDevice string                `json:"source_device"`
	TargetDevices []string             `json:"target_devices"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
	Status      string                 `json:"status"`
}

// Server holds our application state
type Server struct {
	config      *Config
	db          *sql.DB
	redis       *redis.Client
	router      *mux.Router
	upgrader    websocket.Upgrader
	connections map[string]*WebSocketConnection
	mu          sync.RWMutex
	broadcast   chan BroadcastMessage
}

// WebSocketConnection represents a WebSocket client
type WebSocketConnection struct {
	UserID   string
	DeviceID string
	Conn     *websocket.Conn
	Send     chan []byte
}

// BroadcastMessage represents a message to broadcast
type BroadcastMessage struct {
	UserID    string
	DeviceIDs []string
	Message   []byte
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use one of these commands:
   vrooli scenario start device-sync-hub
   cd scenarios/device-sync-hub && make run

💡 The lifecycle system provides:
   - Automatic port allocation from configured ranges
   - Environment variable management
   - Resource dependency orchestration
   - Health check monitoring
   - Process lifecycle management

Direct execution is not supported for security and consistency.
`)
		os.Exit(1)
	}

	// Load configuration - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}
	uiPort := os.Getenv("UI_PORT")
	if uiPort == "" {
		log.Fatal("❌ UI_PORT environment variable is required")
	}
	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		log.Fatal("❌ AUTH_SERVICE_URL environment variable is required")
	}
	storagePath := os.Getenv("STORAGE_PATH")
	if storagePath == "" {
		log.Fatal("❌ STORAGE_PATH environment variable is required")
	}
	maxFileSize := int64(10485760) // 10MB default
	if mfs := os.Getenv("MAX_FILE_SIZE"); mfs != "" {
		var parsed int64
		if _, err := fmt.Sscanf(mfs, "%d", &parsed); err == nil {
			maxFileSize = parsed
		}
	}
	defaultExpiryHours := 24
	if deh := os.Getenv("DEFAULT_EXPIRY_HOURS"); deh != "" {
		var parsed int
		if _, err := fmt.Sscanf(deh, "%d", &parsed); err == nil {
			defaultExpiryHours = parsed
		}
	}
	thumbnailSize := 200
	if ts := os.Getenv("THUMBNAIL_SIZE"); ts != "" {
		var parsed int
		if _, err := fmt.Sscanf(ts, "%d", &parsed); err == nil {
			thumbnailSize = parsed
		}
	}

	config := &Config{
		Port:               port,
		UIPort:             uiPort,
		AuthServiceURL:     authServiceURL,
		StoragePath:        storagePath,
		MaxFileSize:        maxFileSize,
		DefaultExpiryHours: defaultExpiryHours,
		ThumbnailSize:      thumbnailSize,
	}

	// Connect to database
	db, err := connectDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Connect to Redis
	redisClient := connectRedis()

	// Create server
	server := &Server{
		config:      config,
		db:          db,
		redis:       redisClient,
		connections: make(map[string]*WebSocketConnection),
		broadcast:   make(chan BroadcastMessage),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from UI port and any localhost/127.0.0.1 origin for development
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true // Allow connections without origin (like from CLI tools)
				}
				
				// Allow localhost, 127.0.0.1, and the UI port
				return strings.Contains(origin, "localhost") || 
					   strings.Contains(origin, "127.0.0.1") ||
					   strings.Contains(origin, config.UIPort)
			},
		},
	}

	// Ensure storage directory exists
	if err := os.MkdirAll(config.StoragePath, 0755); err != nil {
		log.Fatal("Failed to create storage directory:", err)
	}
	if err := os.MkdirAll(filepath.Join(config.StoragePath, "thumbnails"), 0755); err != nil {
		log.Fatal("Failed to create thumbnails directory:", err)
	}

	// Start broadcast handler
	go server.handleBroadcast()

	// Start cleanup routine
	go server.startCleanupRoutine()

	// Setup routes
	server.setupRoutes()

	// Start server
	log.Printf("🔄 Device Sync Hub starting on port %s", config.Port)
	log.Printf("🌐 UI available at http://localhost:%s", config.UIPort)
	
	if err := http.ListenAndServe(":"+config.Port, server.router); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func (s *Server) setupRoutes() {
	s.router = mux.NewRouter()

	// Enable CORS with dynamic origins
	c := cors.New(cors.Options{
		AllowOriginFunc: func(origin string) bool {
			// Allow UI port and development origins
			if origin == "" {
				return true // Allow requests without origin (like from CLI tools)
			}
			
			// Allow localhost, 127.0.0.1, and the configured UI port
			return strings.Contains(origin, "localhost") || 
				   strings.Contains(origin, "127.0.0.1") ||
				   strings.Contains(origin, s.config.UIPort)
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	s.router.Use(c.Handler)

	// Health check
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")

	// WebSocket endpoint
	s.router.HandleFunc("/ws", s.websocketHandler)
	s.router.HandleFunc("/api/v1/sync/websocket", s.websocketHandler)

	// Device management
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.Use(s.authMiddleware)
	
	api.HandleFunc("/devices", s.listDevicesHandler).Methods("GET")
	api.HandleFunc("/devices", s.registerDeviceHandler).Methods("POST")
	api.HandleFunc("/devices/{id}", s.getDeviceHandler).Methods("GET")
	api.HandleFunc("/devices/{id}", s.updateDeviceHandler).Methods("PUT")
	api.HandleFunc("/devices/{id}", s.deleteDeviceHandler).Methods("DELETE")

	// Sync operations
	api.HandleFunc("/sync/upload", s.syncUploadHandler).Methods("POST")
	api.HandleFunc("/sync/clipboard", s.syncClipboardHandler).Methods("POST")
	api.HandleFunc("/sync/files", s.syncFileHandler).Methods("POST")
	api.HandleFunc("/sync/notification", s.syncNotificationHandler).Methods("POST")
	api.HandleFunc("/sync/items", s.listSyncItemsHandler).Methods("GET")
	api.HandleFunc("/sync/items/{id}", s.getSyncItemHandler).Methods("GET")
	api.HandleFunc("/sync/items/{id}/download", s.downloadSyncItemHandler).Methods("GET")
	api.HandleFunc("/sync/items/{id}", s.deleteSyncItemHandler).Methods("DELETE")

	// Legacy file operations (for backwards compatibility)
	api.HandleFunc("/files/upload", s.uploadFileHandler).Methods("POST")
	api.HandleFunc("/files/{id}/download", s.downloadFileHandler).Methods("GET")
	api.HandleFunc("/files/{id}/thumbnail", s.getThumbnailHandler).Methods("GET")
}

// Authentication middleware
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "No authentication token provided", http.StatusUnauthorized)
			return
		}

		// Remove "Bearer " prefix if present
		token = strings.TrimPrefix(token, "Bearer ")

		// Validate token with auth service
		user, err := s.validateToken(token)
		if err != nil {
			http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
			return
		}

		// Store user in context
		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) validateToken(token string) (*User, error) {
	// Check cache first
	if s.redis != nil {
		cached, err := s.redis.Get(context.Background(), "auth:"+token).Result()
		if err == nil {
			var user User
			if err := json.Unmarshal([]byte(cached), &user); err == nil {
				return &user, nil
			}
		}
	}

	// Validate with auth service
	req, err := http.NewRequest("GET", s.config.AuthServiceURL+"/api/v1/auth/validate", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authentication failed")
	}

	var authResp struct {
		Valid  bool   `json:"valid"`
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Roles  []string `json:"roles"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	if !authResp.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	user := &User{
		ID:    authResp.UserID,
		Email: authResp.Email,
		Roles: authResp.Roles,
	}

	// Cache the result
	if s.redis != nil {
		data, _ := json.Marshal(user)
		s.redis.Set(context.Background(), "auth:"+token, data, 5*time.Minute)
	}

	return user, nil
}

// Handler implementations

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	overallStatus := "healthy"
	
	// Schema-compliant health response
	healthResponse := map[string]interface{}{
		"status":    overallStatus,
		"service":   "device-sync-hub-api",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": true, // Service is ready to accept requests
		"version":   "1.0.0",
		"dependencies": map[string]interface{}{},
		"metrics": map[string]interface{}{
			"uptime_seconds": time.Since(startTime).Seconds(),
		},
	}
	
	dependencies := healthResponse["dependencies"].(map[string]interface{})
	
	// 1. Check PostgreSQL database (critical for device/sync data)
	dbHealth := s.checkDatabase()
	dependencies["database"] = dbHealth
	if dbHealth["connected"] == false {
		overallStatus = "unhealthy" // Database is critical
	}
	
	// 2. Check Redis cache (important for auth caching and performance)
	redisHealth := s.checkRedis()
	dependencies["redis"] = redisHealth
	if redisHealth["connected"] == false {
		if overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}
	
	// 3. Check authentication service (critical for user validation)
	authHealth := s.checkAuthService()
	dependencies["auth_service"] = authHealth
	if authHealth["connected"] == false {
		overallStatus = "unhealthy" // Auth is critical
	}
	
	// 4. Check file storage system (important for file sync)
	storageHealth := s.checkStorageSystem()
	dependencies["storage_system"] = storageHealth
	if storageHealth["connected"] == false {
		if overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}
	
	// 5. Check WebSocket system (important for real-time sync)
	wsHealth := s.checkWebSocketSystem()
	dependencies["websocket_system"] = wsHealth
	if wsHealth["connected"] == false {
		if overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}
	
	// Update overall status
	healthResponse["status"] = overallStatus
	
	// Add current WebSocket connection metrics
	s.mu.RLock()
	wsConnections := len(s.connections)
	s.mu.RUnlock()
	
	systemMetrics := healthResponse["metrics"].(map[string]interface{})
	systemMetrics["websocket_connections"] = wsConnections
	systemMetrics["max_file_size_mb"] = float64(s.config.MaxFileSize) / 1024 / 1024
	systemMetrics["default_expiry_hours"] = s.config.DefaultExpiryHours
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthResponse)
}

var startTime = time.Now()

// checkDatabase tests PostgreSQL connectivity and basic operations
func (s *Server) checkDatabase() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}
	
	// Test database ping
	dbStart := time.Now()
	err := s.db.Ping()
	dbLatency := time.Since(dbStart)
	
	if err != nil {
		errorCode := "DATABASE_PING_FAILED"
		category := "network"
		
		if strings.Contains(err.Error(), "password authentication failed") {
			errorCode = "DATABASE_AUTH_FAILED"
			category = "authentication"
		} else if strings.Contains(err.Error(), "connection refused") {
			errorCode = "DATABASE_CONNECTION_REFUSED"
		}
		
		result["error"] = map[string]interface{}{
			"code":      errorCode,
			"message":   fmt.Sprintf("Database ping failed: %v", err),
			"category":  category,
			"retryable": true,
		}
		return result
	}
	
	// Test basic query - check if devices table exists and is accessible
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM devices WHERE user_id = $1", "health-check").Scan(&count)
	if err != nil && !strings.Contains(err.Error(), "no rows") {
		result["error"] = map[string]interface{}{
			"code":      "DATABASE_QUERY_FAILED",
			"message":   fmt.Sprintf("Database query failed: %v", err),
			"category":  "resource",
			"retryable": true,
		}
		return result
	}
	
	result["connected"] = true
	result["latency_ms"] = float64(dbLatency.Nanoseconds()) / 1e6
	
	// Get connection pool stats
	stats := s.db.Stats()
	result["pool_stats"] = map[string]interface{}{
		"open_connections": stats.OpenConnections,
		"in_use":          stats.InUse,
		"idle":            stats.Idle,
	}
	
	// Count total devices and sync items for metrics
	var totalDevices, totalSyncItems int
	s.db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&totalDevices)
	s.db.QueryRow("SELECT COUNT(*) FROM sync_items WHERE expires_at > NOW()").Scan(&totalSyncItems)
	
	result["total_devices"] = totalDevices
	result["active_sync_items"] = totalSyncItems
	
	return result
}

// checkRedis tests Redis connectivity and operations
func (s *Server) checkRedis() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}
	
	if s.redis == nil {
		result["error"] = map[string]interface{}{
			"code":      "REDIS_NOT_CONFIGURED",
			"message":   "Redis client not initialized",
			"category":  "configuration",
			"retryable": false,
		}
		return result
	}
	
	// Test Redis ping
	redisStart := time.Now()
	err := s.redis.Ping(context.Background()).Err()
	redisLatency := time.Since(redisStart)
	
	if err != nil {
		errorCode := "REDIS_PING_FAILED"
		category := "network"
		
		if strings.Contains(err.Error(), "connection refused") {
			errorCode = "REDIS_CONNECTION_REFUSED"
		} else if strings.Contains(err.Error(), "NOAUTH") {
			errorCode = "REDIS_AUTH_REQUIRED"
			category = "authentication"
		}
		
		result["error"] = map[string]interface{}{
			"code":      errorCode,
			"message":   fmt.Sprintf("Redis ping failed: %v", err),
			"category":  category,
			"retryable": true,
		}
		return result
	}
	
	// Test set/get operation
	testKey := "health_check_test"
	testValue := time.Now().Format(time.RFC3339)
	
	err = s.redis.Set(context.Background(), testKey, testValue, 10*time.Second).Err()
	if err != nil {
		result["error"] = map[string]interface{}{
			"code":      "REDIS_WRITE_FAILED",
			"message":   fmt.Sprintf("Redis write test failed: %v", err),
			"category":  "resource",
			"retryable": true,
		}
		return result
	}
	
	// Clean up test key
	s.redis.Del(context.Background(), testKey)
	
	result["connected"] = true
	result["latency_ms"] = float64(redisLatency.Nanoseconds()) / 1e6
	
	return result
}

// checkAuthService tests authentication service connectivity
func (s *Server) checkAuthService() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}
	
	if s.config.AuthServiceURL == "" {
		result["error"] = map[string]interface{}{
			"code":      "AUTH_SERVICE_URL_MISSING",
			"message":   "Authentication service URL not configured",
			"category":  "configuration",
			"retryable": false,
		}
		return result
	}
	
	// Test auth service health endpoint
	authStart := time.Now()
	client := &http.Client{Timeout: 5 * time.Second}
	
	// Try to hit the auth service health endpoint
	authHealthURL := s.config.AuthServiceURL + "/health"
	resp, err := client.Get(authHealthURL)
	authLatency := time.Since(authStart)
	
	if err != nil {
		errorCode := "AUTH_SERVICE_CONNECTION_FAILED"
		category := "network"
		
		if strings.Contains(err.Error(), "connection refused") {
			errorCode = "AUTH_SERVICE_CONNECTION_REFUSED"
		} else if strings.Contains(err.Error(), "timeout") {
			errorCode = "AUTH_SERVICE_TIMEOUT"
		}
		
		result["error"] = map[string]interface{}{
			"code":      errorCode,
			"message":   fmt.Sprintf("Cannot connect to auth service: %v", err),
			"category":  category,
			"retryable": true,
		}
		return result
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		result["error"] = map[string]interface{}{
			"code":      fmt.Sprintf("AUTH_SERVICE_HTTP_%d", resp.StatusCode),
			"message":   fmt.Sprintf("Auth service returned status %d", resp.StatusCode),
			"category":  "network",
			"retryable": resp.StatusCode >= 500,
		}
		return result
	}
	
	result["connected"] = true
	result["latency_ms"] = float64(authLatency.Nanoseconds()) / 1e6
	result["auth_service_url"] = s.config.AuthServiceURL
	
	return result
}

// checkStorageSystem tests file storage capabilities
func (s *Server) checkStorageSystem() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}
	
	// Check if storage directory exists and is accessible
	if _, err := os.Stat(s.config.StoragePath); err != nil {
		result["error"] = map[string]interface{}{
			"code":      "STORAGE_DIR_ACCESS_FAILED",
			"message":   fmt.Sprintf("Cannot access storage directory: %v", err),
			"category":  "resource",
			"retryable": false,
		}
		return result
	}
	
	// Check thumbnails directory
	thumbnailsPath := filepath.Join(s.config.StoragePath, "thumbnails")
	if _, err := os.Stat(thumbnailsPath); err != nil {
		result["error"] = map[string]interface{}{
			"code":      "THUMBNAILS_DIR_ACCESS_FAILED",
			"message":   fmt.Sprintf("Cannot access thumbnails directory: %v", err),
			"category":  "resource",
			"retryable": false,
		}
		return result
	}
	
	// Test write access with a small test file
	testFile := filepath.Join(s.config.StoragePath, ".health_check_test")
	testContent := fmt.Sprintf("Health check test at %s", time.Now().Format(time.RFC3339))
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		result["error"] = map[string]interface{}{
			"code":      "STORAGE_WRITE_TEST_FAILED",
			"message":   fmt.Sprintf("Cannot write test file: %v", err),
			"category":  "resource",
			"retryable": true,
		}
		return result
	}
	
	// Test read access
	_, err = os.ReadFile(testFile)
	if err != nil {
		result["error"] = map[string]interface{}{
			"code":      "STORAGE_READ_TEST_FAILED",
			"message":   fmt.Sprintf("Cannot read test file: %v", err),
			"category":  "resource",
			"retryable": true,
		}
		return result
	}
	
	// Clean up test file
	os.Remove(testFile)
	
	result["connected"] = true
	result["storage_path"] = s.config.StoragePath
	result["max_file_size_bytes"] = s.config.MaxFileSize
	
	// Get storage directory size and file count
	var totalSize int64
	var fileCount int
	err = filepath.Walk(s.config.StoragePath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})
	
	if err == nil {
		result["total_files"] = fileCount
		result["total_size_mb"] = float64(totalSize) / 1024 / 1024
	}
	
	return result
}

// checkWebSocketSystem tests WebSocket functionality
func (s *Server) checkWebSocketSystem() map[string]interface{} {
	result := map[string]interface{}{
		"connected": false,
		"error":     nil,
	}
	
	// Check if broadcast channel is working
	if s.broadcast == nil {
		result["error"] = map[string]interface{}{
			"code":      "WEBSOCKET_BROADCAST_CHANNEL_NIL",
			"message":   "WebSocket broadcast channel not initialized",
			"category":  "internal",
			"retryable": false,
		}
		return result
	}
	
	// Test broadcast channel by sending a test message (non-blocking)
	testMessage := BroadcastMessage{
		UserID:    "health-check",
		DeviceIDs: []string{},
		Message:   []byte(`{"type":"health_check","data":"test"}`),
	}
	
	// Use a select with default to avoid blocking if broadcast channel is full
	select {
	case s.broadcast <- testMessage:
		// Successfully sent test message
	default:
		result["error"] = map[string]interface{}{
			"code":      "WEBSOCKET_BROADCAST_CHANNEL_FULL",
			"message":   "WebSocket broadcast channel is full or blocked",
			"category":  "resource",
			"retryable": true,
		}
		return result
	}
	
	// Get current WebSocket connection metrics
	s.mu.RLock()
	connectionCount := len(s.connections)
	
	// Count connections by status (we can't easily test individual connections without disruption)
	connectionsByDevice := make(map[string]int)
	for deviceID := range s.connections {
		connectionsByDevice[deviceID] = 1
	}
	s.mu.RUnlock()
	
	result["connected"] = true
	result["active_connections"] = connectionCount
	result["unique_devices"] = len(connectionsByDevice)
	
	return result
}

func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	// Validate token from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "No authentication token", http.StatusUnauthorized)
		return
	}

	user, err := s.validateToken(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Upgrade connection
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Get device ID from query
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		deviceID = uuid.New().String()
	}

	// Create connection wrapper
	wsConn := &WebSocketConnection{
		UserID:   user.ID,
		DeviceID: deviceID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
	}

	// Register connection
	s.mu.Lock()
	s.connections[deviceID] = wsConn
	s.mu.Unlock()

	// Update device status
	s.updateDeviceStatus(deviceID, user.ID, true)

	// Start goroutines for reading and writing
	go wsConn.writePump()
	go wsConn.readPump(s)

	log.Printf("WebSocket connected: user=%s, device=%s", user.ID, deviceID)
}

func (s *Server) listDevicesHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	query := `
		SELECT id, name, type, platform, last_seen, capabilities, 
		       CASE WHEN last_seen > NOW() - INTERVAL '1 minute' THEN true ELSE false END as is_online
		FROM devices 
		WHERE user_id = $1 
		ORDER BY last_seen DESC
	`

	rows, err := s.db.Query(query, user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch devices", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	devices := []Device{}
	for rows.Next() {
		var d Device
		var capabilities string
		err := rows.Scan(&d.ID, &d.Name, &d.Type, &d.Platform, &d.LastSeen, &capabilities, &d.IsOnline)
		if err != nil {
			continue
		}
		d.UserID = user.ID
		d.Capabilities = strings.Split(capabilities, ",")
		devices = append(devices, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

func (s *Server) registerDeviceHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	var device Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	device.ID = uuid.New().String()
	device.UserID = user.ID
	device.LastSeen = time.Now()

	capabilities := strings.Join(device.Capabilities, ",")
	query := `
		INSERT INTO devices (id, user_id, name, type, platform, capabilities, last_seen)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			name = $3, type = $4, platform = $5, capabilities = $6, last_seen = $7
	`

	_, err := s.db.Exec(query, device.ID, device.UserID, device.Name, 
		device.Type, device.Platform, capabilities, device.LastSeen)
	if err != nil {
		http.Error(w, "Failed to register device", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(device)
}

func (s *Server) getDeviceHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	vars := mux.Vars(r)
	deviceID := vars["id"]

	var device Device
	var capabilities string
	query := `
		SELECT id, name, type, platform, last_seen, capabilities,
		       CASE WHEN last_seen > NOW() - INTERVAL '1 minute' THEN true ELSE false END as is_online
		FROM devices WHERE id = $1 AND user_id = $2
	`
	err := s.db.QueryRow(query, deviceID, user.ID).Scan(
		&device.ID, &device.Name, &device.Type, &device.Platform,
		&device.LastSeen, &capabilities, &device.IsOnline,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to fetch device", http.StatusInternalServerError)
		return
	}

	device.UserID = user.ID
	device.Capabilities = strings.Split(capabilities, ",")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(device)
}

func (s *Server) updateDeviceHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	vars := mux.Vars(r)
	deviceID := vars["id"]

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Build update query dynamically
	setClause := []string{"last_seen = NOW()"}
	args := []interface{}{deviceID, user.ID}
	argIndex := 3

	if name, ok := updates["name"].(string); ok {
		setClause = append(setClause, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, name)
		argIndex++
	}

	query := fmt.Sprintf("UPDATE devices SET %s WHERE id = $1 AND user_id = $2",
		strings.Join(setClause, ", "))

	result, err := s.db.Exec(query, args...)
	if err != nil {
		http.Error(w, "Failed to update device", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteDeviceHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	vars := mux.Vars(r)
	deviceID := vars["id"]

	query := "DELETE FROM devices WHERE id = $1 AND user_id = $2"
	result, err := s.db.Exec(query, deviceID, user.ID)
	if err != nil {
		http.Error(w, "Failed to delete device", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	// Disconnect WebSocket if connected
	s.mu.Lock()
	if conn, ok := s.connections[deviceID]; ok {
		conn.Conn.Close()
		delete(s.connections, deviceID)
	}
	s.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) syncClipboardHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	var req struct {
		Content       string   `json:"content"`
		SourceDevice  string   `json:"source_device"`
		TargetDevices []string `json:"target_devices"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Create sync item
	item := SyncItem{
		ID:            uuid.New().String(),
		UserID:        user.ID,
		Type:          "clipboard",
		Content:       map[string]interface{}{"text": req.Content},
		SourceDevice:  req.SourceDevice,
		TargetDevices: req.TargetDevices,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(time.Duration(s.config.DefaultExpiryHours) * time.Hour),
		Status:        "pending",
	}

	// Store in database
	contentJSON, _ := json.Marshal(item.Content)
	targetsJSON, _ := json.Marshal(item.TargetDevices)
	
	query := `
		INSERT INTO sync_items (id, user_id, type, content, source_device, target_devices, expires_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := s.db.Exec(query, item.ID, item.UserID, item.Type, contentJSON,
		item.SourceDevice, targetsJSON, item.ExpiresAt, item.Status)
	if err != nil {
		http.Error(w, "Failed to create sync item", http.StatusInternalServerError)
		return
	}

	// Broadcast to target devices
	message, _ := json.Marshal(map[string]interface{}{
		"type": "clipboard_sync",
		"data": item,
	})
	s.broadcast <- BroadcastMessage{
		UserID:    user.ID,
		DeviceIDs: req.TargetDevices,
		Message:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func (s *Server) syncFileHandler(w http.ResponseWriter, r *http.Request) {
	// This endpoint is deprecated in favor of the unified /sync/upload endpoint
	// Redirect to the new endpoint for backwards compatibility
	http.Redirect(w, r, "/api/v1/sync/upload", http.StatusPermanentRedirect)
}

func (s *Server) syncNotificationHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	var req struct {
		Title         string   `json:"title"`
		Body          string   `json:"body"`
		Icon          string   `json:"icon"`
		SourceDevice  string   `json:"source_device"`
		TargetDevices []string `json:"target_devices"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Create notification sync item
	item := SyncItem{
		ID:       uuid.New().String(),
		UserID:   user.ID,
		Type:     "notification",
		Content: map[string]interface{}{
			"title": req.Title,
			"body":  req.Body,
			"icon":  req.Icon,
		},
		SourceDevice:  req.SourceDevice,
		TargetDevices: req.TargetDevices,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		Status:        "pending",
	}

	// Store and broadcast
	contentJSON, _ := json.Marshal(item.Content)
	targetsJSON, _ := json.Marshal(item.TargetDevices)
	
	query := `
		INSERT INTO sync_items (id, user_id, type, content, source_device, target_devices, expires_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := s.db.Exec(query, item.ID, item.UserID, item.Type, contentJSON,
		item.SourceDevice, targetsJSON, item.ExpiresAt, item.Status)
	if err != nil {
		http.Error(w, "Failed to create notification", http.StatusInternalServerError)
		return
	}

	// Broadcast to devices
	message, _ := json.Marshal(map[string]interface{}{
		"type": "notification_sync",
		"data": item,
	})
	s.broadcast <- BroadcastMessage{
		UserID:    user.ID,
		DeviceIDs: req.TargetDevices,
		Message:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func (s *Server) listSyncItemsHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	query := `
		SELECT id, type, content, source_device, target_devices, created_at, expires_at, status
		FROM sync_items
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := s.db.Query(query, user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch sync items", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []SyncItem{}
	for rows.Next() {
		var item SyncItem
		var contentJSON, targetsJSON []byte
		err := rows.Scan(&item.ID, &item.Type, &contentJSON, &item.SourceDevice,
			&targetsJSON, &item.CreatedAt, &item.ExpiresAt, &item.Status)
		if err != nil {
			continue
		}
		item.UserID = user.ID
		json.Unmarshal(contentJSON, &item.Content)
		json.Unmarshal(targetsJSON, &item.TargetDevices)
		items = append(items, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (s *Server) getSyncItemHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	vars := mux.Vars(r)
	itemID := vars["id"]

	var item SyncItem
	var contentJSON, targetsJSON []byte
	query := `
		SELECT id, type, content, source_device, target_devices, created_at, expires_at, status
		FROM sync_items
		WHERE id = $1 AND user_id = $2
	`
	err := s.db.QueryRow(query, itemID, user.ID).Scan(
		&item.ID, &item.Type, &contentJSON, &item.SourceDevice,
		&targetsJSON, &item.CreatedAt, &item.ExpiresAt, &item.Status,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Sync item not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to fetch sync item", http.StatusInternalServerError)
		return
	}

	item.UserID = user.ID
	json.Unmarshal(contentJSON, &item.Content)
	json.Unmarshal(targetsJSON, &item.TargetDevices)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// Unified sync upload handler - handles both files and text content
func (s *Server) syncUploadHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	
	// Check content type to determine if it's multipart (file) or JSON (text)
	contentType := r.Header.Get("Content-Type")
	
	var item SyncItem
	var filePath string
	
	if strings.Contains(contentType, "multipart/form-data") {
		// Handle file upload
		err := r.ParseMultipartForm(s.config.MaxFileSize)
		if err != nil {
			http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to get file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Create unique filename and save file
		fileID := uuid.New().String()
		ext := filepath.Ext(header.Filename)
		filename := fileID + ext
		filePath = filepath.Join(s.config.StoragePath, filename)

		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			os.Remove(filePath)
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		// Get expiry hours from form data
		expiryHours := s.config.DefaultExpiryHours
		if expiryStr := r.FormValue("expires_in"); expiryStr != "" {
			if parsed, err := strconv.Atoi(expiryStr); err == nil && parsed > 0 {
				expiryHours = parsed
			}
		}

		// Create sync item for file
		item = SyncItem{
			ID:       uuid.New().String(),
			UserID:   user.ID,
			Type:     "file",
			Content: map[string]interface{}{
				"filename":      header.Filename,
				"original_name": header.Filename,
				"file_size":     header.Size,
				"mime_type":     header.Header.Get("Content-Type"),
				"storage_path":  filePath,
				"file_id":       fileID,
			},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Duration(expiryHours) * time.Hour),
			Status:    "active",
		}
	} else {
		// Handle JSON text upload
		var req struct {
			Text        string `json:"text"`
			ContentType string `json:"content_type"`
			ExpiresIn   int    `json:"expires_in"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Text == "" {
			http.Error(w, "Text content required", http.StatusBadRequest)
			return
		}

		contentTypeValue := req.ContentType
		if contentTypeValue == "" {
			contentTypeValue = "text"
		}

		expiryHours := req.ExpiresIn
		if expiryHours <= 0 {
			expiryHours = s.config.DefaultExpiryHours
		}

		// Create sync item for text
		item = SyncItem{
			ID:       uuid.New().String(),
			UserID:   user.ID,
			Type:     contentTypeValue,
			Content:  map[string]interface{}{"text": req.Text},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Duration(expiryHours) * time.Hour),
			Status:    "active",
		}
	}

	// Store sync item in database
	contentJSON, _ := json.Marshal(item.Content)
	
	query := `
		INSERT INTO sync_items (id, user_id, type, content, source_device, target_devices, expires_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := s.db.Exec(query, item.ID, item.UserID, item.Type, contentJSON,
		item.SourceDevice, "[]", item.ExpiresAt, item.Status)
	if err != nil {
		if filePath != "" {
			os.Remove(filePath) // Clean up file if database insert fails
		}
		http.Error(w, "Failed to store sync item", http.StatusInternalServerError)
		return
	}

	// Broadcast to connected devices (real-time sync)
	message, _ := json.Marshal(map[string]interface{}{
		"type": "item_added",
		"item": item,
	})
	s.broadcast <- BroadcastMessage{
		UserID:    user.ID,
		DeviceIDs: []string{}, // Broadcast to all user's devices
		Message:   message,
	}

	// Return success response
	response := map[string]interface{}{
		"success":    true,
		"item_id":    item.ID,
		"expires_at": item.ExpiresAt.Format(time.RFC3339),
	}

	if item.Type == "file" {
		response["filename"] = item.Content["filename"]
		response["file_size"] = item.Content["file_size"]
		response["thumbnail_url"] = fmt.Sprintf("/api/v1/sync/items/%s/thumbnail", item.ID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Download sync item handler
func (s *Server) downloadSyncItemHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	vars := mux.Vars(r)
	itemID := vars["id"]

	// Get sync item from database
	var item SyncItem
	var contentJSON []byte
	query := `
		SELECT id, type, content, expires_at, status
		FROM sync_items
		WHERE id = $1 AND user_id = $2 AND expires_at > NOW()
	`
	err := s.db.QueryRow(query, itemID, user.ID).Scan(
		&item.ID, &item.Type, &contentJSON, &item.ExpiresAt, &item.Status)
	
	if err == sql.ErrNoRows {
		http.Error(w, "Item not found or expired", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to fetch item", http.StatusInternalServerError)
		return
	}

	json.Unmarshal(contentJSON, &item.Content)

	if item.Type == "file" {
		// Handle file download
		storagePath, ok := item.Content["storage_path"].(string)
		if !ok {
			http.Error(w, "Invalid file data", http.StatusInternalServerError)
			return
		}

		originalName, _ := item.Content["original_name"].(string)
		mimeType, _ := item.Content["mime_type"].(string)

		// Serve file
		w.Header().Set("Content-Type", mimeType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", originalName))
		http.ServeFile(w, r, storagePath)
	} else {
		// Handle text download
		text, ok := item.Content["text"].(string)
		if !ok {
			http.Error(w, "Invalid text data", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.txt\"", item.ID))
		w.Write([]byte(text))
	}
}

// Delete sync item handler
func (s *Server) deleteSyncItemHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	vars := mux.Vars(r)
	itemID := vars["id"]

	// Get item before deletion to clean up files
	var item SyncItem
	var contentJSON []byte
	query := `
		SELECT id, type, content FROM sync_items
		WHERE id = $1 AND user_id = $2
	`
	err := s.db.QueryRow(query, itemID, user.ID).Scan(&item.ID, &item.Type, &contentJSON)
	if err == sql.ErrNoRows {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to fetch item", http.StatusInternalServerError)
		return
	}

	json.Unmarshal(contentJSON, &item.Content)

	// Delete from database
	deleteQuery := "DELETE FROM sync_items WHERE id = $1 AND user_id = $2"
	result, err := s.db.Exec(deleteQuery, itemID, user.ID)
	if err != nil {
		http.Error(w, "Failed to delete item", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	// Clean up file if it's a file type
	if item.Type == "file" {
		if storagePath, ok := item.Content["storage_path"].(string); ok {
			os.Remove(storagePath)
		}
	}

	// Broadcast deletion to connected devices
	message, _ := json.Marshal(map[string]interface{}{
		"type": "item_deleted",
		"item_id": itemID,
	})
	s.broadcast <- BroadcastMessage{
		UserID:    user.ID,
		DeviceIDs: []string{}, // Broadcast to all user's devices
		Message:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"deleted_at": time.Now().Format(time.RFC3339),
	})
}

func (s *Server) uploadFileHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	// Parse multipart form
	err := r.ParseMultipartForm(s.config.MaxFileSize)
	if err != nil {
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create unique filename
	fileID := uuid.New().String()
	ext := filepath.Ext(header.Filename)
	filename := fileID + ext
	filepath := filepath.Join(s.config.StoragePath, filename)

	// Save file
	dst, err := os.Create(filepath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Store file metadata
	query := `
		INSERT INTO files (id, user_id, filename, original_name, size, mime_type, path, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	expiresAt := time.Now().Add(time.Duration(s.config.DefaultExpiryHours) * time.Hour)
	_, err = s.db.Exec(query, fileID, user.ID, filename, header.Filename,
		header.Size, header.Header.Get("Content-Type"), filepath, expiresAt)
	if err != nil {
		os.Remove(filepath)
		http.Error(w, "Failed to store file metadata", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":            fileID,
		"filename":      header.Filename,
		"size":          header.Size,
		"download_url":  fmt.Sprintf("/api/v1/files/%s/download", fileID),
		"thumbnail_url": fmt.Sprintf("/api/v1/files/%s/thumbnail", fileID),
		"expires_at":    expiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) downloadFileHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	vars := mux.Vars(r)
	fileID := vars["id"]

	var filepath, originalName, mimeType string
	query := "SELECT path, original_name, mime_type FROM files WHERE id = $1 AND user_id = $2"
	err := s.db.QueryRow(query, fileID, user.ID).Scan(&filepath, &originalName, &mimeType)
	
	if err == sql.ErrNoRows {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to fetch file", http.StatusInternalServerError)
		return
	}

	// Serve file
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", originalName))
	http.ServeFile(w, r, filepath)
}

func (s *Server) getThumbnailHandler(w http.ResponseWriter, r *http.Request) {
	// Thumbnail generation would be implemented here
	// For now, return a placeholder
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// WebSocket methods

func (conn *WebSocketConnection) readPump(s *Server) {
	defer func() {
		s.mu.Lock()
		delete(s.connections, conn.DeviceID)
		s.mu.Unlock()
		conn.Conn.Close()
		s.updateDeviceStatus(conn.DeviceID, conn.UserID, false)
		log.Printf("WebSocket disconnected: user=%s, device=%s", conn.UserID, conn.DeviceID)
	}()

	conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.Conn.SetPongHandler(func(string) error {
		conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Process message
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// Handle different message types
		switch msg["type"] {
		case "ping":
			conn.Send <- []byte(`{"type":"pong"}`)
		case "sync_request":
			// Handle sync request
			s.handleSyncRequest(conn, msg)
		}
	}
}

func (conn *WebSocketConnection) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		conn.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-conn.Send:
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			conn.Conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *Server) handleBroadcast() {
	for msg := range s.broadcast {
		s.mu.RLock()
		for deviceID, conn := range s.connections {
			// Check if this connection should receive the message
			if conn.UserID == msg.UserID {
				if len(msg.DeviceIDs) == 0 || contains(msg.DeviceIDs, deviceID) {
					select {
					case conn.Send <- msg.Message:
					default:
						// Channel full, close connection
						close(conn.Send)
						delete(s.connections, deviceID)
					}
				}
			}
		}
		s.mu.RUnlock()
	}
}

func (s *Server) handleSyncRequest(conn *WebSocketConnection, msg map[string]interface{}) {
	// Handle sync request from device
	// This would process the request and broadcast to appropriate devices
}

func (s *Server) updateDeviceStatus(deviceID, userID string, isOnline bool) {
	query := "UPDATE devices SET last_seen = NOW() WHERE id = $1 AND user_id = $2"
	s.db.Exec(query, deviceID, userID)
}

func (s *Server) startCleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		// Clean up expired sync items
		query := "DELETE FROM sync_items WHERE expires_at < NOW()"
		s.db.Exec(query)

		// Clean up expired files
		var expiredFiles []string
		rows, err := s.db.Query("SELECT path FROM files WHERE expires_at < NOW()")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var path string
				if rows.Scan(&path) == nil {
					expiredFiles = append(expiredFiles, path)
				}
			}
		}

		// Delete expired files from filesystem
		for _, path := range expiredFiles {
			os.Remove(path)
		}

		// Delete expired file records
		s.db.Exec("DELETE FROM files WHERE expires_at < NOW()")
		
		log.Printf("Cleanup completed: removed expired items")
	}
}

// Helper functions

func connectDB() (*sql.DB, error) {
	// Database configuration - support both POSTGRES_URL and individual components
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		
		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			return nil, fmt.Errorf("❌ Missing database configuration. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}
	
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to open database connection: %v", err)
	}
	
	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	
	// Implement exponential backoff for database connection with enhanced monitoring
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	
	log.Println("🔄 Attempting PostgreSQL connection with exponential backoff...")
	log.Printf("📊 Connection strategy: max %d attempts, delays from %v to %v", maxRetries, baseDelay, maxDelay)
	
	var pingErr error
	startTime := time.Now()
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		attemptStart := time.Now()
		pingErr = db.Ping()
		attemptDuration := time.Since(attemptStart)
		
		if pingErr == nil {
			totalDuration := time.Since(startTime)
			log.Printf("✅ Database connected successfully!")
			log.Printf("   📊 Connection established on attempt %d/%d", attempt + 1, maxRetries)
			log.Printf("   ⏱️  Total connection time: %v", totalDuration)
			log.Printf("   🔗 Connection pool: %d max open, %d max idle", 25, 5)
			break
		}
		
		// Calculate exponential backoff delay with capped growth
		delay := time.Duration(math.Min(
			float64(baseDelay) * math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		
		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.3  // 30% jitter range
		jitter := time.Duration(jitterRange * (0.5 + float64(attempt) / float64(maxRetries*2)))
		actualDelay := delay + jitter
		
		// Determine error category for better diagnostics
		errorCategory := "unknown"
		if strings.Contains(pingErr.Error(), "connection refused") {
			errorCategory = "connection_refused"
		} else if strings.Contains(pingErr.Error(), "password") {
			errorCategory = "authentication"
		} else if strings.Contains(pingErr.Error(), "timeout") {
			errorCategory = "timeout"
		} else if strings.Contains(pingErr.Error(), "no such host") {
			errorCategory = "dns_resolution"
		}
		
		log.Printf("⚠️  Connection attempt %d/%d failed", attempt + 1, maxRetries)
		log.Printf("   🔍 Error category: %s", errorCategory)
		log.Printf("   📝 Error details: %v", pingErr)
		log.Printf("   ⏱️  Attempt duration: %v", attemptDuration)
		log.Printf("   ⏳ Waiting %v before retry (base: %v, jitter: %v)", actualDelay, delay, jitter)
		
		// Provide detailed status every few attempts
		if attempt > 0 && attempt % 3 == 0 {
			log.Printf("📈 Connection retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt + 1, maxRetries)
			log.Printf("   - Total elapsed time: %v", time.Since(startTime))
			log.Printf("   - Success rate: 0%% (will retry)")
			
			// Suggest troubleshooting based on error category
			switch errorCategory {
			case "connection_refused":
				log.Printf("   💡 Hint: Check if PostgreSQL is running and listening on the correct port")
			case "authentication":
				log.Printf("   💡 Hint: Verify database credentials in environment variables")
			case "timeout":
				log.Printf("   💡 Hint: Check network connectivity and firewall rules")
			case "dns_resolution":
				log.Printf("   💡 Hint: Verify database hostname is correct")
			}
		}
		
		time.Sleep(actualDelay)
	}
	
	if pingErr != nil {
		totalDuration := time.Since(startTime)
		return nil, fmt.Errorf("❌ Database connection failed after %d attempts over %v: %v", maxRetries, totalDuration, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
	log.Println("✨ Ready to handle sync operations")
	return db, nil
}

func connectRedis() *redis.Client {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("Failed to parse Redis URL: %v", err)
		return nil
	}

	client := redis.NewClient(opt)
	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		return nil
	}

	return client
}


func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}