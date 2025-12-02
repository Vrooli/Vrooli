package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
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
	json.NewEncoder(w).Encode(LoginResponse{
		Email:         req.Email,
		Authenticated: true,
	})
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
		json.NewEncoder(w).Encode(LoginResponse{
			Authenticated: false,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Email:         email,
		Authenticated: true,
	})
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
