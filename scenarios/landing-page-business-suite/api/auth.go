package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultAdminEmail        = "admin@localhost"
	defaultAdminPasswordHash = "$2a$10$nhmpbhFPQUZZwEH.qaYHCeiKBWDvr8z5Z7eM4v62MmNwm.0N.5xeG" // changeme123
	seededAdminID            = 1                                                              // Reserved ID for the seeded/default admin account
)

// resolveSecret implements the same 3-layer secret resolution as the shell secrets::resolve().
// Priority: 1) Environment variable, 2) .vrooli/secrets.json, 3) empty string
func resolveSecret(key string) string {
	// Layer 1: Environment variable
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	// Layer 2: .vrooli/secrets.json (used by scenario-to-cloud deployments)
	secretsFile := findSecretsFile()
	if secretsFile != "" {
		if value := readSecretFromJSON(secretsFile, key); value != "" {
			return value
		}
	}

	// Layer 3: Not found
	return ""
}

// findSecretsFile locates .vrooli/secrets.json by walking up from cwd or using VROOLI_ROOT
func findSecretsFile() string {
	// Try VROOLI_ROOT first (production deployments set this)
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		candidate := filepath.Join(root, ".vrooli", "secrets.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Walk up from current working directory to find .vrooli/secrets.json
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir := cwd
	for {
		candidate := filepath.Join(dir, ".vrooli", "secrets.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root
		}
		dir = parent
	}

	return ""
}

// readSecretFromJSON reads a single key from a JSON file
func readSecretFromJSON(filePath, key string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	var secrets map[string]interface{}
	if err := json.Unmarshal(data, &secrets); err != nil {
		return ""
	}

	if value, ok := secrets[key]; ok {
		if str, ok := value.(string); ok {
			return strings.TrimSpace(str)
		}
	}

	return ""
}

// getAdminDefaults returns admin email and password hash.
// Resolution order: environment variables -> .vrooli/secrets.json -> hardcoded defaults.
// This enables customization via scenario-to-cloud's Secrets Tab.
func getAdminDefaults() (email string, passwordHash string, err error) {
	email = resolveSecret("ADMIN_DEFAULT_EMAIL")
	if email == "" {
		email = defaultAdminEmail
	}

	// Check for plaintext password override (will be hashed)
	plaintextPassword := resolveSecret("ADMIN_DEFAULT_PASSWORD")
	if plaintextPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), bcrypt.DefaultCost)
		if err != nil {
			return "", "", fmt.Errorf("hash admin password: %w", err)
		}
		return email, string(hash), nil
	}

	// Fall back to default hash
	return email, defaultAdminPasswordHash, nil
}

var sessionStore *sessions.CookieStore

func initSessionStore() {
	secret := resolveSecret("SESSION_SECRET")
	if secret == "" {
		logStructured("session_secret_missing", map[string]interface{}{
			"level":   "warn",
			"message": "SESSION_SECRET not set; using placeholder for development",
			"action":  "Set SESSION_SECRET in environment or .vrooli/secrets.json",
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
		ResetEnabled:  true, // Always enabled - UI handles confirmation
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
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
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
		writeJSONError(w, http.StatusUnauthorized, "Invalid credentials", ApiErrorTypeUnauthorized)
		return
	} else if err != nil {
		logStructuredError("login_db_error", map[string]interface{}{
			"error": err.Error(),
		})
		writeJSONError(w, http.StatusInternalServerError, "Unable to verify credentials. Please try again.", ApiErrorTypeServerError)
		return
	}

	// Verify password using bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		logStructured("login_invalid_password", map[string]interface{}{
			"level": "warn",
			"email": req.Email,
		})
		writeJSONError(w, http.StatusUnauthorized, "Invalid credentials", ApiErrorTypeUnauthorized)
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
		logStructuredError("session_save_error", map[string]interface{}{
			"error": err.Error(),
		})
		writeJSONError(w, http.StatusInternalServerError, "Failed to create session. Please try again.", ApiErrorTypeServerError)
		return
	}

	logStructured("admin_login_success", map[string]interface{}{
		"level": "info",
		"email": req.Email,
	})

	// Return user data
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(buildLoginResponse(req.Email, true)); err != nil {
		logStructuredError("login_response_encode_failed", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// handleAdminLogout destroys the admin session
func (s *Server) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "admin_session")
	email := session.Values["email"]
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		logStructuredError("admin_session_save_failed", map[string]interface{}{
			"error": err.Error(),
		})
	}

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
		if err := json.NewEncoder(w).Encode(buildLoginResponse("", false)); err != nil {
			http.Error(w, "Failed to encode session response", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(buildLoginResponse(email, true)); err != nil {
		http.Error(w, "Failed to encode session response", http.StatusInternalServerError)
	}
}

// requireAdmin is middleware to protect admin routes
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionStore.Get(r, "admin_session")
		email, ok := session.Values["email"].(string)
		if !ok || email == "" {
			writeJSONError(w, http.StatusUnauthorized, "Session expired. Please log in again.", ApiErrorTypeUnauthorized)
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
	envEmail, envHash, _ := getAdminDefaults()
	return AdminProfileResponse{
		Email:             email,
		IsDefaultEmail:    strings.EqualFold(email, envEmail),
		IsDefaultPassword: passwordHash == envHash,
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
		writeJSONError(w, http.StatusUnauthorized, "Session expired. Please log in again.", ApiErrorTypeUnauthorized)
		return
	}

	var storedEmail, passwordHash string
	err := s.db.QueryRow(`SELECT email, password_hash FROM admin_users WHERE email = $1`, email).Scan(&storedEmail, &passwordHash)
	if err == sql.ErrNoRows {
		writeJSONError(w, http.StatusUnauthorized, "Session expired. Please log in again.", ApiErrorTypeUnauthorized)
		return
	} else if err != nil {
		logStructuredError("admin_profile_lookup_failed", map[string]interface{}{
			"error": err.Error(),
		})
		writeJSONError(w, http.StatusInternalServerError, "Failed to load profile. Please try again.", ApiErrorTypeServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(buildAdminProfileResponse(storedEmail, passwordHash)); err != nil {
		logStructuredError("admin_profile_encode_failed", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// handleAdminProfileUpdate updates the admin email and/or password
func (s *Server) handleAdminProfileUpdate(w http.ResponseWriter, r *http.Request) {
	currentEmail, ok := sessionAdminEmail(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "Session expired. Please log in again.", ApiErrorTypeUnauthorized)
		return
	}

	var req AdminProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
		return
	}

	req.CurrentPassword = strings.TrimSpace(req.CurrentPassword)
	req.NewEmail = strings.TrimSpace(req.NewEmail)
	req.NewPassword = strings.TrimSpace(req.NewPassword)

	if req.CurrentPassword == "" {
		writeJSONError(w, http.StatusBadRequest, "Current password is required", ApiErrorTypeValidation)
		return
	}
	if req.NewEmail == "" && req.NewPassword == "" {
		writeJSONError(w, http.StatusBadRequest, "Provide a new email or password to update", ApiErrorTypeValidation)
		return
	}

	var adminID int64
	var storedEmail, passwordHash string
	err := s.db.QueryRow(`SELECT id, email, password_hash FROM admin_users WHERE email = $1`, currentEmail).
		Scan(&adminID, &storedEmail, &passwordHash)
	if err == sql.ErrNoRows {
		writeJSONError(w, http.StatusUnauthorized, "Session expired. Please log in again.", ApiErrorTypeUnauthorized)
		return
	} else if err != nil {
		logStructuredError("admin_profile_load_failed", map[string]interface{}{
			"error": err.Error(),
		})
		writeJSONError(w, http.StatusInternalServerError, "Failed to load admin profile. Please try again.", ApiErrorTypeServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.CurrentPassword)); err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Invalid credentials", ApiErrorTypeUnauthorized)
		return
	}

	targetEmail := storedEmail
	if req.NewEmail != "" && !strings.EqualFold(req.NewEmail, storedEmail) {
		if !strings.Contains(req.NewEmail, "@") || len(req.NewEmail) < 5 {
			writeJSONError(w, http.StatusBadRequest, "Invalid email address", ApiErrorTypeValidation)
			return
		}
		var exists int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM admin_users WHERE LOWER(email) = LOWER($1) AND id <> $2`, req.NewEmail, adminID).Scan(&exists); err != nil {
			logStructuredError("admin_email_validation_failed", map[string]interface{}{
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to validate email. Please try again.", ApiErrorTypeServerError)
			return
		}
		if exists > 0 {
			writeJSONError(w, http.StatusConflict, "Email already in use", ApiErrorTypeValidation)
			return
		}
		targetEmail = req.NewEmail
	}

	targetPasswordHash := passwordHash
	if req.NewPassword != "" {
		if err := validateAdminPasswordUpdate(req.NewPassword, passwordHash); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			logStructuredError("admin_password_hash_failed", map[string]interface{}{
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to process password. Please try again.", ApiErrorTypeServerError)
			return
		}
		targetPasswordHash = string(hashed)
	}

	if targetEmail == storedEmail && targetPasswordHash == passwordHash {
		writeJSONError(w, http.StatusBadRequest, "No changes detected", ApiErrorTypeValidation)
		return
	}

	if _, err := s.db.Exec(`UPDATE admin_users SET email = $1, password_hash = $2 WHERE id = $3`, targetEmail, targetPasswordHash, adminID); err != nil {
		logStructuredError("admin_profile_update_failed", map[string]interface{}{
			"error": err.Error(),
		})
		writeJSONError(w, http.StatusInternalServerError, "Failed to update profile. Please try again.", ApiErrorTypeServerError)
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
	if err := json.NewEncoder(w).Encode(buildAdminProfileResponse(targetEmail, targetPasswordHash)); err != nil {
		logStructuredError("admin_profile_update_encode_failed", map[string]interface{}{
			"error": err.Error(),
		})
	}
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
	_, envHash, _ := getAdminDefaults()
	if bcrypt.CompareHashAndPassword([]byte(envHash), []byte(candidate)) == nil {
		return fmt.Errorf("New password cannot use the default credential")
	}
	return nil
}
