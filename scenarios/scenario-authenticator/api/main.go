package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var (
	db         *sql.DB
	redisClient *redis.Client
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	ctx        = context.Background()
	authProcessor *AuthProcessor
)

// User represents a user account
type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Username      string    `json:"username,omitempty"`
	PasswordHash  string    `json:"-"`
	Roles         []string  `json:"roles"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
	LastLogin     *time.Time `json:"last_login,omitempty"`
}

// RegisterRequest represents registration payload
type RegisterRequest struct {
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Username string                 `json:"username,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// LoginRequest represents login payload
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Success      bool   `json:"success"`
	Token        string `json:"token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	User         *User  `json:"user,omitempty"`
	Message      string `json:"message,omitempty"`
}

// ValidationResponse represents token validation response
type ValidationResponse struct {
	Valid     bool      `json:"valid"`
	UserID    string    `json:"user_id,omitempty"`
	Email     string    `json:"email,omitempty"`
	Roles     []string  `json:"roles,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// Claims represents JWT claims
type Claims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	jwt.StandardClaims
}

func main() {
	// Load configuration - ALL values REQUIRED, no defaults
	port := os.Getenv("AUTH_API_PORT")
	if port == "" {
		port = os.Getenv("API_PORT") // Fallback to API_PORT
		if port == "" {
			log.Fatal("❌ AUTH_API_PORT or API_PORT environment variable is required")
		}
	}
	
	// Database configuration - support both POSTGRES_URL and individual components
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		
		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}
	
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("❌ REDIS_URL environment variable is required")
	}

	// Initialize database
	initDB(dbURL)
	defer db.Close()

	// Initialize Redis
	initRedis(redisURL)
	defer redisClient.Close()

	// Load JWT keys
	loadJWTKeys()

	// Initialize AuthProcessor
	authProcessor = NewAuthProcessor(db)

	// Setup routes
	router := mux.NewRouter()
	
	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")
	
	// Authentication endpoints
	router.HandleFunc("/api/v1/auth/register", registerHandler).Methods("POST")
	router.HandleFunc("/api/v1/auth/login", loginHandler).Methods("POST")
	router.HandleFunc("/api/v1/auth/validate", validateHandler).Methods("GET", "POST")
	router.HandleFunc("/api/v1/auth/refresh", refreshHandler).Methods("POST")
	router.HandleFunc("/api/v1/auth/logout", logoutHandler).Methods("POST")
	router.HandleFunc("/api/v1/auth/reset-password", resetPasswordHandler).Methods("POST")
	router.HandleFunc("/api/v1/auth/complete-reset", completeResetHandler).Methods("POST")
	
	// User management endpoints
	router.HandleFunc("/api/v1/users", getUsersHandler).Methods("GET")
	router.HandleFunc("/api/v1/users/{id}", getUserHandler).Methods("GET")
	router.HandleFunc("/api/v1/users/{id}", updateUserHandler).Methods("PUT")
	router.HandleFunc("/api/v1/users/{id}", deleteUserHandler).Methods("DELETE")
	
	// Session management
	router.HandleFunc("/api/v1/sessions", getSessionsHandler).Methods("GET")
	router.HandleFunc("/api/v1/sessions/{id}", revokeSessionHandler).Methods("DELETE")

	// CORS middleware
	router.Use(corsMiddleware)
	
	// Start server
	log.Printf("Authentication API server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func initDB(dbURL string) {
	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to open database connection:", err)
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
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
}

func initRedis(redisURL string) {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: "",
		DB:       0,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	
	log.Println("Redis connected successfully")
}

func loadJWTKeys() {
	// Try to load existing keys
	privateKeyPath := "../data/keys/private.pem"
	publicKeyPath := "../data/keys/public.pem"
	
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		// Generate new keys if they don't exist
		generateJWTKeys()
		return
	}
	
	// Load private key
	privateKeyData, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		log.Fatal("Failed to read private key:", err)
	}
	
	privateKeyBlock, _ := pem.Decode(privateKeyData)
	privateKey, err = x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		log.Fatal("Failed to parse private key:", err)
	}
	
	// Load public key
	publicKeyData, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		log.Fatal("Failed to read public key:", err)
	}
	
	publicKeyBlock, _ := pem.Decode(publicKeyData)
	pubInterface, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
	if err != nil {
		log.Fatal("Failed to parse public key:", err)
	}
	
	publicKey = pubInterface.(*rsa.PublicKey)
	log.Println("JWT keys loaded successfully")
}

func generateJWTKeys() {
	// Generate new RSA key pair
	var err error
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal("Failed to generate private key:", err)
	}
	
	publicKey = &privateKey.PublicKey
	
	// Create keys directory
	os.MkdirAll("../data/keys", 0755)
	
	// Save private key
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	
	privateFile, err := os.Create("../data/keys/private.pem")
	if err != nil {
		log.Fatal("Failed to create private key file:", err)
	}
	defer privateFile.Close()
	
	if err := pem.Encode(privateFile, privateKeyPEM); err != nil {
		log.Fatal("Failed to write private key:", err)
	}
	
	// Save public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		log.Fatal("Failed to marshal public key:", err)
	}
	
	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	
	publicFile, err := os.Create("../data/keys/public.pem")
	if err != nil {
		log.Fatal("Failed to create public key file:", err)
	}
	defer publicFile.Close()
	
	if err := pem.Encode(publicFile, publicKeyPEM); err != nil {
		log.Fatal("Failed to write public key:", err)
	}
	
	log.Println("JWT keys generated successfully")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	dbHealthy := db.Ping() == nil
	
	// Check Redis connection
	redisHealthy := redisClient.Ping(ctx).Err() == nil
	
	status := "healthy"
	if !dbHealthy || !redisHealthy {
		status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	response := map[string]interface{}{
		"status":   status,
		"database": dbHealthy,
		"redis":    redisHealthy,
		"version":  "1.0.0",
	}
	
	json.NewEncoder(w).Encode(response)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate input
	if req.Email == "" || req.Password == "" {
		sendError(w, "Email and password are required", http.StatusBadRequest)
		return
	}
	
	if len(req.Password) < 8 {
		sendError(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}
	
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		sendError(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}
	
	// Create user
	userID := uuid.New().String()
	
	// Default roles
	roles := []string{"user"}
	rolesJSON, _ := json.Marshal(roles)
	
	// Default metadata
	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}
	metadataJSON, _ := json.Marshal(req.Metadata)
	
	// Insert user into database
	query := `
		INSERT INTO users (id, email, username, password_hash, roles, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at`
	
	var createdAt time.Time
	err = db.QueryRow(query, userID, req.Email, req.Username, string(hashedPassword), 
		rolesJSON, metadataJSON, time.Now()).Scan(&createdAt)
	
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			sendError(w, "Email already exists", http.StatusConflict)
			return
		}
		sendError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}
	
	// Create user object
	user := &User{
		ID:            userID,
		Email:         req.Email,
		Username:      req.Username,
		Roles:         roles,
		EmailVerified: false,
		CreatedAt:     createdAt,
	}
	
	// Generate tokens
	token, err := generateToken(user)
	if err != nil {
		sendError(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	
	refreshToken := generateRefreshToken()
	
	// Store refresh token in Redis
	storeRefreshToken(userID, refreshToken)
	
	// Log registration event
	logAuthEvent(userID, "user.registered", getClientIP(r), r.UserAgent(), true, nil)
	
	// Send response
	response := AuthResponse{
		Success:      true,
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Get user from database
	var user User
	var rolesJSON string
	query := `
		SELECT id, email, username, password_hash, roles, email_verified, created_at, last_login
		FROM users
		WHERE email = $1 AND deleted_at IS NULL`
	
	err := db.QueryRow(query, req.Email).Scan(
		&user.ID, &user.Email, &user.Username, &user.PasswordHash,
		&rolesJSON, &user.EmailVerified, &user.CreatedAt, &user.LastLogin)
	
	if err == sql.ErrNoRows {
		sendError(w, "Invalid email or password", http.StatusUnauthorized)
		logAuthEvent("", "user.login.failed", getClientIP(r), r.UserAgent(), false, 
			map[string]interface{}{"email": req.Email})
		return
	} else if err != nil {
		sendError(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	// Unmarshal roles
	json.Unmarshal([]byte(rolesJSON), &user.Roles)
	
	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		sendError(w, "Invalid email or password", http.StatusUnauthorized)
		
		// Update failed login attempts
		db.Exec("UPDATE users SET failed_login_attempts = failed_login_attempts + 1 WHERE id = $1", user.ID)
		logAuthEvent(user.ID, "user.login.failed", getClientIP(r), r.UserAgent(), false, nil)
		return
	}
	
	// Update last login
	now := time.Now()
	db.Exec("UPDATE users SET last_login = $1, login_count = login_count + 1, failed_login_attempts = 0 WHERE id = $2", 
		now, user.ID)
	user.LastLogin = &now
	
	// Generate tokens
	token, err := generateToken(&user)
	if err != nil {
		sendError(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	
	refreshToken := generateRefreshToken()
	
	// Store session in Redis
	storeSession(user.ID, token, refreshToken, r)
	
	// Log login event
	logAuthEvent(user.ID, "user.logged_in", getClientIP(r), r.UserAgent(), true, nil)
	
	// Send response
	response := AuthResponse{
		Success:      true,
		Token:        token,
		RefreshToken: refreshToken,
		User:         &user,
	}
	
	json.NewEncoder(w).Encode(response)
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	// Use AuthProcessor for validation (supports both GET and POST)
	response := authProcessor.ValidateAuthToken(ctx, r)
	
	// Set the appropriate status code
	if response.HTTPStatus != 0 {
		w.WriteHeader(response.HTTPStatus)
	}
	
	json.NewEncoder(w).Encode(response)
}

func refreshHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate refresh token and get user ID
	userID := validateRefreshToken(req.RefreshToken)
	if userID == "" {
		sendError(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}
	
	// Get user from database
	var user User
	var rolesJSON string
	query := `SELECT id, email, username, roles FROM users WHERE id = $1 AND deleted_at IS NULL`
	
	err := db.QueryRow(query, userID).Scan(&user.ID, &user.Email, &user.Username, &rolesJSON)
	if err != nil {
		sendError(w, "User not found", http.StatusNotFound)
		return
	}
	
	json.Unmarshal([]byte(rolesJSON), &user.Roles)
	
	// Generate new tokens
	token, err := generateToken(&user)
	if err != nil {
		sendError(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	
	newRefreshToken := generateRefreshToken()
	
	// Revoke old refresh token and store new one
	revokeRefreshToken(req.RefreshToken)
	storeRefreshToken(userID, newRefreshToken)
	
	// Send response
	response := AuthResponse{
		Success:      true,
		Token:        token,
		RefreshToken: newRefreshToken,
		User:         &user,
	}
	
	json.NewEncoder(w).Encode(response)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	// Get token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		sendError(w, "No token provided", http.StatusBadRequest)
		return
	}
	
	tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
	
	// Parse token to get user ID
	token, _ := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})
	
	if token != nil && token.Valid {
		claims, _ := token.Claims.(*Claims)
		
		// Blacklist the token
		blacklistToken(tokenString, claims.ExpiresAt)
		
		// Clear user sessions from Redis
		clearUserSessions(claims.UserID)
		
		// Log logout event
		logAuthEvent(claims.UserID, "user.logged_out", getClientIP(r), r.UserAgent(), true, nil)
	}
	
	response := map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}

func resetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req PasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Process password reset
	response := authProcessor.ProcessPasswordReset(ctx, req)
	
	// Set appropriate status code
	if !response.Success {
		w.WriteHeader(http.StatusBadRequest)
	}
	
	json.NewEncoder(w).Encode(response)
}

func completeResetHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Complete password reset
	err := authProcessor.CompletePasswordReset(ctx, req.Token, req.NewPassword)
	if err != nil {
		sendError(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	response := map[string]interface{}{
		"success": true,
		"message": "Password reset successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}

func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	// This would implement user listing with pagination
	// For now, return empty list
	response := map[string]interface{}{
		"users": []User{},
		"total": 0,
	}
	
	json.NewEncoder(w).Encode(response)
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]
	
	var user User
	var rolesJSON string
	query := `
		SELECT id, email, username, roles, email_verified, created_at, last_login
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`
	
	err := db.QueryRow(query, userID).Scan(
		&user.ID, &user.Email, &user.Username, &rolesJSON,
		&user.EmailVerified, &user.CreatedAt, &user.LastLogin)
	
	if err == sql.ErrNoRows {
		sendError(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		sendError(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	json.Unmarshal([]byte(rolesJSON), &user.Roles)
	json.NewEncoder(w).Encode(user)
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	// Placeholder for user update
	sendError(w, "Not implemented", http.StatusNotImplemented)
}

func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	// Placeholder for user soft delete
	sendError(w, "Not implemented", http.StatusNotImplemented)
}

func getSessionsHandler(w http.ResponseWriter, r *http.Request) {
	// Placeholder for session listing
	response := map[string]interface{}{
		"sessions": []interface{}{},
		"total":    0,
	}
	
	json.NewEncoder(w).Encode(response)
}

func revokeSessionHandler(w http.ResponseWriter, r *http.Request) {
	// Placeholder for session revocation
	response := map[string]interface{}{
		"success": true,
		"message": "Session revoked",
	}
	
	json.NewEncoder(w).Encode(response)
}

// Helper functions

func generateToken(user *User) (string, error) {
	expirationTime := time.Now().Add(1 * time.Hour)
	
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Roles:  user.Roles,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "scenario-authenticator",
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

func generateRefreshToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func storeRefreshToken(userID, refreshToken string) {
	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	redisClient.Set(ctx, key, userID, 7*24*time.Hour)
	
	// Add to user's token set
	userKey := fmt.Sprintf("user_tokens:%s", userID)
	redisClient.SAdd(ctx, userKey, refreshToken)
	redisClient.Expire(ctx, userKey, 7*24*time.Hour)
}

func validateRefreshToken(refreshToken string) string {
	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	userID, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return ""
	}
	return userID
}

func revokeRefreshToken(refreshToken string) {
	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	redisClient.Del(ctx, key)
}

func storeSession(userID, token, refreshToken string, r *http.Request) {
	session := map[string]interface{}{
		"user_id":       userID,
		"token":         token,
		"refresh_token": refreshToken,
		"ip_address":    getClientIP(r),
		"user_agent":    r.UserAgent(),
		"created_at":    time.Now().Unix(),
	}
	
	sessionJSON, _ := json.Marshal(session)
	sessionKey := fmt.Sprintf("session:%s:%s", userID, token[:20])
	
	redisClient.Set(ctx, sessionKey, sessionJSON, 24*time.Hour)
	
	// Add to user's session set
	userSessionKey := fmt.Sprintf("user_sessions:%s", userID)
	redisClient.SAdd(ctx, userSessionKey, sessionKey)
	redisClient.Expire(ctx, userSessionKey, 24*time.Hour)
	
	// Update active sessions counter
	redisClient.Incr(ctx, "stats:active_sessions")
}

func clearUserSessions(userID string) {
	userSessionKey := fmt.Sprintf("user_sessions:%s", userID)
	sessions, _ := redisClient.SMembers(ctx, userSessionKey).Result()
	
	for _, session := range sessions {
		redisClient.Del(ctx, session)
	}
	
	redisClient.Del(ctx, userSessionKey)
	
	// Update active sessions counter
	redisClient.Decr(ctx, "stats:active_sessions")
}

func blacklistToken(token string, expiresAt int64) {
	ttl := time.Until(time.Unix(expiresAt, 0))
	if ttl > 0 {
		key := fmt.Sprintf("blacklist:%s", hashToken(token))
		redisClient.Set(ctx, key, "1", ttl)
	}
}

func isTokenBlacklisted(token string) bool {
	key := fmt.Sprintf("blacklist:%s", hashToken(token))
	exists, _ := redisClient.Exists(ctx, key).Result()
	return exists > 0
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func logAuthEvent(userID, action, ipAddress, userAgent string, success bool, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	
	metadataJSON, _ := json.Marshal(metadata)
	
	query := `
		INSERT INTO audit_logs (user_id, action, ip_address, user_agent, success, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}
	
	db.Exec(query, userIDPtr, action, ipAddress, userAgent, success, metadataJSON, time.Now())
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func sendError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	response := map[string]interface{}{
		"success": false,
		"error":   message,
	}
	json.NewEncoder(w).Encode(response)
}

func sendValidationResponse(w http.ResponseWriter, valid bool, claims *Claims) {
	response := ValidationResponse{
		Valid: valid,
	}
	
	if valid && claims != nil {
		response.UserID = claims.UserID
		response.Email = claims.Email
		response.Roles = claims.Roles
		response.ExpiresAt = time.Unix(claims.ExpiresAt, 0)
	}
	
	json.NewEncoder(w).Encode(response)
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}
	
	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	
	// Fall back to RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0]
}

// getEnv removed to prevent hardcoded defaults