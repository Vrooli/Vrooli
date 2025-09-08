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
				// Allow connections from UI
				origin := r.Header.Get("Origin")
				return strings.Contains(origin, "localhost")
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

	// Enable CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{fmt.Sprintf("http://localhost:%s", s.config.UIPort), "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	s.router.Use(c.Handler)

	// Health check
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")

	// WebSocket endpoint
	s.router.HandleFunc("/ws", s.websocketHandler)

	// Device management
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.Use(s.authMiddleware)
	
	api.HandleFunc("/devices", s.listDevicesHandler).Methods("GET")
	api.HandleFunc("/devices", s.registerDeviceHandler).Methods("POST")
	api.HandleFunc("/devices/{id}", s.getDeviceHandler).Methods("GET")
	api.HandleFunc("/devices/{id}", s.updateDeviceHandler).Methods("PUT")
	api.HandleFunc("/devices/{id}", s.deleteDeviceHandler).Methods("DELETE")

	// Sync operations
	api.HandleFunc("/sync/clipboard", s.syncClipboardHandler).Methods("POST")
	api.HandleFunc("/sync/files", s.syncFileHandler).Methods("POST")
	api.HandleFunc("/sync/notification", s.syncNotificationHandler).Methods("POST")
	api.HandleFunc("/sync/items", s.listSyncItemsHandler).Methods("GET")
	api.HandleFunc("/sync/items/{id}", s.getSyncItemHandler).Methods("GET")

	// File operations
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
	response := map[string]interface{}{
		"status":  "healthy",
		"service": "device-sync-hub",
		"version": "1.0.0",
		"time":    time.Now(),
	}
	
	// Check database connection
	if err := s.db.Ping(); err != nil {
		response["status"] = "degraded"
		response["database"] = "disconnected"
	} else {
		response["database"] = "connected"
	}

	// Check Redis connection
	if s.redis != nil {
		if err := s.redis.Ping(context.Background()).Err(); err != nil {
			response["redis"] = "disconnected"
		} else {
			response["redis"] = "connected"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
	// File sync implementation would go here
	// This would handle file uploads and distribution to target devices
	http.Error(w, "Not implemented", http.StatusNotImplemented)
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
	
	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	
	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📆 Database URL configured")
	
	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
			break
		}
		
		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay) * math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		
		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter
		
		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)
		
		// Provide detailed status every few attempts
		if attempt > 0 && attempt % 3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt + 1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt * 2) * baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}
		
		time.Sleep(actualDelay)
	}
	
	if pingErr != nil {
		return nil, fmt.Errorf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
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