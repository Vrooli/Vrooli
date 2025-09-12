package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"scenario-authenticator/auth"
	"scenario-authenticator/db"
	"scenario-authenticator/models"
	"scenario-authenticator/utils"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// RegisterHandler handles user registration
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		return
	}
	
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate input
	if req.Email == "" || req.Password == "" {
		utils.SendError(w, "Email and password are required", http.StatusBadRequest)
		return
	}
	
	if len(req.Password) < 8 {
		utils.SendError(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}
	
	// Check if user already exists
	var existingID string
	err := db.DB.QueryRow("SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL", req.Email).Scan(&existingID)
	if err == nil {
		utils.SendError(w, "Email already registered", http.StatusConflict)
		return
	} else if err != sql.ErrNoRows {
		log.Printf("Database error: %v", err)
		utils.SendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		utils.SendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	// Create user
	user := &models.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Roles:        []string{"user"},
		CreatedAt:    time.Now(),
	}
	
	// Insert into database
	_, err = db.DB.Exec(`
		INSERT INTO users (id, email, username, password_hash, roles, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		user.ID, user.Email, user.Username, user.PasswordHash, 
		`["user"]`, `{}`, user.CreatedAt, user.CreatedAt,
	)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		utils.SendError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}
	
	// Generate tokens
	token, err := auth.GenerateToken(user)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		utils.SendError(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	
	refreshToken := auth.GenerateRefreshToken()
	auth.StoreRefreshToken(user.ID, refreshToken)
	
	// Store session
	if err := auth.StoreSession(user.ID, token, refreshToken, r); err != nil {
		log.Printf("Failed to store session: %v", err)
	}
	
	// Log auth event
	logAuthEvent(user.ID, "user.registered", auth.GetClientIP(r), r.Header.Get("User-Agent"), true, nil)
	
	// Send response
	response := models.AuthResponse{
		Success:      true,
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
	}
	
	utils.SendJSON(w, response, http.StatusCreated)
}

// LoginHandler handles user login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		return
	}
	
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate input
	if req.Email == "" || req.Password == "" {
		utils.SendError(w, "Email and password are required", http.StatusBadRequest)
		return
	}
	
	// Get user from database
	var user models.User
	var passwordHash string
	var rolesJSON string
	err := db.DB.QueryRow(`
		SELECT id, email, username, password_hash, roles, email_verified, created_at, last_login
		FROM users 
		WHERE email = $1 AND deleted_at IS NULL`,
		req.Email,
	).Scan(&user.ID, &user.Email, &user.Username, &passwordHash, &rolesJSON, 
		&user.EmailVerified, &user.CreatedAt, &user.LastLogin)
	
	if err != nil {
		if err == sql.ErrNoRows {
			logAuthEvent("", "user.login.failed", auth.GetClientIP(r), r.Header.Get("User-Agent"), false, 
				map[string]interface{}{"email": req.Email, "reason": "user_not_found"})
			utils.SendError(w, "Invalid credentials", http.StatusUnauthorized)
		} else {
			log.Printf("Database error: %v", err)
			utils.SendError(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}
	
	// Parse roles
	if err := json.Unmarshal([]byte(rolesJSON), &user.Roles); err != nil {
		user.Roles = []string{"user"}
	}
	
	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		logAuthEvent(user.ID, "user.login.failed", auth.GetClientIP(r), r.Header.Get("User-Agent"), false, 
			map[string]interface{}{"reason": "invalid_password"})
		utils.SendError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	
	// Update last login
	now := time.Now()
	user.LastLogin = &now
	_, err = db.DB.Exec("UPDATE users SET last_login = $1, login_count = login_count + 1 WHERE id = $2", now, user.ID)
	if err != nil {
		log.Printf("Failed to update last login: %v", err)
	}
	
	// Generate tokens
	token, err := auth.GenerateToken(&user)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		utils.SendError(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	
	refreshToken := auth.GenerateRefreshToken()
	auth.StoreRefreshToken(user.ID, refreshToken)
	
	// Store session
	if err := auth.StoreSession(user.ID, token, refreshToken, r); err != nil {
		log.Printf("Failed to store session: %v", err)
	}
	
	// Log auth event
	logAuthEvent(user.ID, "user.logged_in", auth.GetClientIP(r), r.Header.Get("User-Agent"), true, nil)
	
	// Send response
	response := models.AuthResponse{
		Success:      true,
		Token:        token,
		RefreshToken: refreshToken,
		User:         &user,
	}
	
	utils.SendJSON(w, response, http.StatusOK)
}

// ValidateHandler handles token validation
func ValidateHandler(w http.ResponseWriter, r *http.Request) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		utils.SendValidationResponse(w, false, nil)
		return
	}
	
	// Parse Bearer token
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		utils.SendValidationResponse(w, false, nil)
		return
	}
	
	token := parts[1]
	
	// Check if token is blacklisted
	if auth.IsTokenBlacklisted(token) {
		utils.SendValidationResponse(w, false, nil)
		return
	}
	
	// Validate token
	claims, err := auth.ValidateToken(token)
	if err != nil {
		utils.SendValidationResponse(w, false, nil)
		return
	}
	
	// Log validation event (only sample to avoid flooding)
	if time.Now().Unix() % 100 == 0 { // Log 1% of validations
		logAuthEvent(claims.UserID, "token.validated", "", "", true, nil)
	}
	
	utils.SendValidationResponse(w, true, claims)
}

// RefreshHandler handles token refresh
func RefreshHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if req.RefreshToken == "" {
		utils.SendError(w, "Refresh token is required", http.StatusBadRequest)
		return
	}
	
	// Validate refresh token
	userID := auth.ValidateRefreshToken(req.RefreshToken)
	if userID == "" {
		utils.SendError(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}
	
	// Get user from database
	var user models.User
	var rolesJSON string
	err := db.DB.QueryRow(`
		SELECT id, email, username, roles, email_verified
		FROM users 
		WHERE id = $1 AND deleted_at IS NULL`,
		userID,
	).Scan(&user.ID, &user.Email, &user.Username, &rolesJSON, &user.EmailVerified)
	
	if err != nil {
		utils.SendError(w, "User not found", http.StatusNotFound)
		return
	}
	
	// Parse roles
	if err := json.Unmarshal([]byte(rolesJSON), &user.Roles); err != nil {
		user.Roles = []string{"user"}
	}
	
	// Generate new tokens
	token, err := auth.GenerateToken(&user)
	if err != nil {
		utils.SendError(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	
	// Revoke old refresh token and create new one
	auth.RevokeRefreshToken(req.RefreshToken)
	newRefreshToken := auth.GenerateRefreshToken()
	auth.StoreRefreshToken(user.ID, newRefreshToken)
	
	// Store new session
	if err := auth.StoreSession(user.ID, token, newRefreshToken, r); err != nil {
		log.Printf("Failed to store session: %v", err)
	}
	
	// Send response
	response := models.AuthResponse{
		Success:      true,
		Token:        token,
		RefreshToken: newRefreshToken,
		User:         &user,
	}
	
	utils.SendJSON(w, response, http.StatusOK)
}

// LogoutHandler handles user logout
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		utils.SendError(w, "No authorization header", http.StatusBadRequest)
		return
	}
	
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		utils.SendError(w, "Invalid authorization format", http.StatusBadRequest)
		return
	}
	
	token := parts[1]
	
	// Validate token to get claims
	claims, err := auth.ValidateToken(token)
	if err != nil {
		utils.SendError(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	
	// Blacklist the token
	auth.BlacklistToken(token, claims.ExpiresAt)
	
	// Clear user sessions
	auth.ClearUserSessions(claims.UserID)
	
	// Log auth event
	logAuthEvent(claims.UserID, "user.logged_out", auth.GetClientIP(r), r.Header.Get("User-Agent"), true, nil)
	
	utils.SendJSON(w, map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	}, http.StatusOK)
}

// ResetPasswordHandler initiates password reset
func ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Always return success to prevent email enumeration
	utils.SendJSON(w, map[string]interface{}{
		"success": true,
		"message": "If the email exists, a reset link has been sent",
	}, http.StatusOK)
	
	// Check if user exists
	var userID string
	err := db.DB.QueryRow("SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL", req.Email).Scan(&userID)
	if err != nil {
		return // Silently fail to prevent enumeration
	}
	
	// Generate reset token
	resetToken := auth.GenerateRefreshToken()
	tokenHash := auth.HashToken(resetToken)
	expires := time.Now().Add(1 * time.Hour)
	
	// Store reset token in database
	_, err = db.DB.Exec(`
		UPDATE users 
		SET password_reset_token = $1, password_reset_expires = $2 
		WHERE id = $3`,
		tokenHash, expires, userID,
	)
	if err != nil {
		log.Printf("Failed to store reset token: %v", err)
		return
	}
	
	// TODO: Send email with reset link
	// For now, log the reset token
	log.Printf("Password reset token for user %s: %s", userID, resetToken)
	
	// Log auth event
	logAuthEvent(userID, "password.reset.requested", auth.GetClientIP(r), r.Header.Get("User-Agent"), true, nil)
}

// CompleteResetHandler completes password reset
func CompleteResetHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if len(req.NewPassword) < 8 {
		utils.SendError(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}
	
	// Hash the token to compare with stored hash
	tokenHash := auth.HashToken(req.Token)
	
	// Find user with this reset token
	var userID string
	var expires time.Time
	err := db.DB.QueryRow(`
		SELECT id, password_reset_expires 
		FROM users 
		WHERE password_reset_token = $1 AND deleted_at IS NULL`,
		tokenHash,
	).Scan(&userID, &expires)
	
	if err != nil {
		utils.SendError(w, "Invalid or expired reset token", http.StatusBadRequest)
		return
	}
	
	// Check if token is expired
	if time.Now().After(expires) {
		utils.SendError(w, "Reset token has expired", http.StatusBadRequest)
		return
	}
	
	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.SendError(w, "Failed to process password", http.StatusInternalServerError)
		return
	}
	
	// Update password and clear reset token
	_, err = db.DB.Exec(`
		UPDATE users 
		SET password_hash = $1, password_reset_token = NULL, password_reset_expires = NULL 
		WHERE id = $2`,
		string(hashedPassword), userID,
	)
	if err != nil {
		utils.SendError(w, "Failed to update password", http.StatusInternalServerError)
		return
	}
	
	// Clear all user sessions
	auth.ClearUserSessions(userID)
	
	// Log auth event
	logAuthEvent(userID, "password.reset.completed", auth.GetClientIP(r), r.Header.Get("User-Agent"), true, nil)
	
	utils.SendJSON(w, map[string]interface{}{
		"success": true,
		"message": "Password reset successfully",
	}, http.StatusOK)
}

// logAuthEvent logs authentication events to the audit log
func logAuthEvent(userID, action, ipAddress, userAgent string, success bool, metadata map[string]interface{}) {
	metadataJSON, _ := json.Marshal(metadata)
	
	_, err := db.DB.Exec(`
		INSERT INTO audit_logs (user_id, action, ip_address, user_agent, success, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		userID, action, ipAddress, userAgent, success, string(metadataJSON), time.Now(),
	)
	if err != nil {
		log.Printf("Failed to log auth event: %v", err)
	}
}