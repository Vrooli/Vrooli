package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"unicode"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultAdminEmail        = "admin@localhost"
	defaultAdminPasswordHash = "$2a$10$nhmpbhFPQUZZwEH.qaYHCeiKBWDvr8z5Z7eM4v62MmNwm.0N.5xeG" // changeme123
)

var sessionStore *sessions.CookieStore

func initSessionStore() {
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		logStructured("session_secret_missing", map[string]interface{}{
			"level":   "warn",
			"message": "SESSION_SECRET not set; using placeholder for development",
			"action":  "Set SESSION_SECRET environment variable before starting the server",
		})
		secret = "dev-session-placeholder"
	}
	sessionStore = sessions.NewCookieStore([]byte(secret))
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Email         string `json:"email,omitempty"`
	Authenticated bool   `json:"authenticated"`
	ResetEnabled  bool   `json:"reset_enabled"`
}

func buildLoginResponse(email string, authenticated bool) LoginResponse {
	resp := LoginResponse{
		Authenticated: authenticated,
		ResetEnabled:  adminResetEnabled(),
	}
	if authenticated && email != "" {
		resp.Email = email
	}
	return resp
}

// handleAdminLogin authenticates admin users and creates a session
// Implements OT-P0-008 (ADMIN-AUTH)
func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Query user from database
	var passwordHash string
	err := s.db.QueryRow(
		"SELECT password_hash FROM admin_users WHERE email = $1",
		req.Email,
	).Scan(&passwordHash)

	if err == sql.ErrNoRows {
		logStructured("login_invalid_email", map[string]interface{}{
			"level": "warn",
			"email": req.Email,
		})
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	} else if err != nil {
		logStructured("login_db_error", map[string]interface{}{
			"level": "error",
			"error": err.Error(),
		})
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Verify password using bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		logStructured("login_invalid_password", map[string]interface{}{
			"level": "warn",
			"email": req.Email,
		})
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Update last login timestamp
	_, _ = s.db.Exec("UPDATE admin_users SET last_login = NOW() WHERE email = $1", req.Email)

	// Create session
	session, _ := sessionStore.Get(r, "admin_session")
	session.Values["email"] = req.Email
	session.Options.HttpOnly = true
	session.Options.Secure = false     // Set to true in production with HTTPS
	session.Options.MaxAge = 86400 * 7 // 7 days
	session.Options.Path = "/"
	session.Options.SameSite = http.SameSiteLaxMode
	if err := session.Save(r, w); err != nil {
		logStructured("session_save_error", map[string]interface{}{
			"level": "error",
			"error": err.Error(),
		})
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	logStructured("admin_login_success", map[string]interface{}{
		"level": "info",
		"email": req.Email,
	})

	// Return user data
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(buildLoginResponse(req.Email, true))
}

// handleAdminLogout destroys the admin session
func (s *Server) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "admin_session")
	email := session.Values["email"]
	session.Options.MaxAge = -1
	session.Save(r, w)

	logStructured("admin_logout", map[string]interface{}{
		"level": "info",
		"email": email,
	})

	w.WriteHeader(http.StatusNoContent)
}

// handleAdminSession checks if the current session is valid
func (s *Server) handleAdminSession(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "admin_session")
	email, ok := session.Values["email"].(string)
	if !ok || email == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(buildLoginResponse("", false))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(buildLoginResponse(email, true))
}

// requireAdmin is middleware to protect admin routes
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, "admin_session")
		email, ok := session.Values["email"].(string)
		if !ok || email == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

type AdminProfileResponse struct {
	Email             string `json:"email"`
	IsDefaultEmail    bool   `json:"is_default_email"`
	IsDefaultPassword bool   `json:"is_default_password"`
}

type AdminProfileUpdateRequest struct {
	CurrentPassword string `json:"current_password"`
	NewEmail        string `json:"new_email"`
	NewPassword     string `json:"new_password"`
}

func buildAdminProfileResponse(email, passwordHash string) AdminProfileResponse {
	return AdminProfileResponse{
		Email:             email,
		IsDefaultEmail:    strings.EqualFold(email, defaultAdminEmail),
		IsDefaultPassword: passwordHash == defaultAdminPasswordHash,
	}
}

func sessionAdminEmail(r *http.Request) (string, bool) {
	session, _ := sessionStore.Get(r, "admin_session")
	email, ok := session.Values["email"].(string)
	if !ok || strings.TrimSpace(email) == "" {
		return "", false
	}
	return email, true
}

// handleAdminProfile returns the authenticated admin's profile
func (s *Server) handleAdminProfile(w http.ResponseWriter, r *http.Request) {
	email, ok := sessionAdminEmail(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var storedEmail, passwordHash string
	err := s.db.QueryRow(`SELECT email, password_hash FROM admin_users WHERE email = $1`, email).Scan(&storedEmail, &passwordHash)
	if err == sql.ErrNoRows {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	} else if err != nil {
		logStructuredError("admin_profile_lookup_failed", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Failed to load profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(buildAdminProfileResponse(storedEmail, passwordHash))
}

// handleAdminProfileUpdate updates the admin email and/or password
func (s *Server) handleAdminProfileUpdate(w http.ResponseWriter, r *http.Request) {
	currentEmail, ok := sessionAdminEmail(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req AdminProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.CurrentPassword = strings.TrimSpace(req.CurrentPassword)
	req.NewEmail = strings.TrimSpace(req.NewEmail)
	req.NewPassword = strings.TrimSpace(req.NewPassword)

	if req.CurrentPassword == "" {
		http.Error(w, "Current password is required", http.StatusBadRequest)
		return
	}
	if req.NewEmail == "" && req.NewPassword == "" {
		http.Error(w, "Provide a new email or password to update", http.StatusBadRequest)
		return
	}

	var adminID int64
	var storedEmail, passwordHash string
	err := s.db.QueryRow(`SELECT id, email, password_hash FROM admin_users WHERE email = $1`, currentEmail).
		Scan(&adminID, &storedEmail, &passwordHash)
	if err == sql.ErrNoRows {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	} else if err != nil {
		logStructuredError("admin_profile_load_failed", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Failed to load admin profile", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.CurrentPassword)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	targetEmail := storedEmail
	if req.NewEmail != "" && !strings.EqualFold(req.NewEmail, storedEmail) {
		if !strings.Contains(req.NewEmail, "@") || len(req.NewEmail) < 5 {
			http.Error(w, "Invalid email address", http.StatusBadRequest)
			return
		}
		var exists int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM admin_users WHERE LOWER(email) = LOWER($1) AND id <> $2`, req.NewEmail, adminID).Scan(&exists); err != nil {
			http.Error(w, "Failed to validate email", http.StatusInternalServerError)
			return
		}
		if exists > 0 {
			http.Error(w, "Email already in use", http.StatusConflict)
			return
		}
		targetEmail = req.NewEmail
	}

	targetPasswordHash := passwordHash
	if req.NewPassword != "" {
		if err := validateAdminPasswordUpdate(req.NewPassword, passwordHash); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Failed to process password", http.StatusInternalServerError)
			return
		}
		targetPasswordHash = string(hashed)
	}

	if targetEmail == storedEmail && targetPasswordHash == passwordHash {
		http.Error(w, "No changes detected", http.StatusBadRequest)
		return
	}

	if _, err := s.db.Exec(`UPDATE admin_users SET email = $1, password_hash = $2 WHERE id = $3`, targetEmail, targetPasswordHash, adminID); err != nil {
		logStructuredError("admin_profile_update_failed", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	session, _ := sessionStore.Get(r, "admin_session")
	session.Values["email"] = targetEmail
	_ = session.Save(r, w)

	logStructured("admin_profile_updated", map[string]interface{}{
		"level":          "info",
		"changed_email":  targetEmail != storedEmail,
		"changed_secret": targetPasswordHash != passwordHash,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(buildAdminProfileResponse(targetEmail, targetPasswordHash))
}

func validateAdminPasswordUpdate(candidate, currentHash string) error {
	if len(candidate) < 12 {
		return fmt.Errorf("Password must be at least 12 characters")
	}
	hasLetter := false
	hasDigit := false
	for _, c := range candidate {
		if unicode.IsLetter(c) {
			hasLetter = true
		}
		if unicode.IsDigit(c) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return fmt.Errorf("Password must include letters and numbers")
	}
	if bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(candidate)) == nil {
		return fmt.Errorf("New password must be different from the current password")
	}
	if bcrypt.CompareHashAndPassword([]byte(defaultAdminPasswordHash), []byte(candidate)) == nil {
		return fmt.Errorf("New password cannot use the default credential")
	}
	return nil
}
