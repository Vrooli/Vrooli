package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/gorilla/sessions"
	apisecrets "github.com/vrooli/api-core/secrets"
	"golang.org/x/crypto/bcrypt"
	adminhttp "landing-page-business-suite-api/handlers/admin"
	"landing-page-business-suite-api/internal/envx"
)

// isSecureCookiesEnabled returns whether secure cookies should be enabled.
// Defaults to true in production (LPBS_SECURE_COOKIES not set or "true").
// Can be disabled for development by setting LPBS_SECURE_COOKIES=false.
func isSecureCookiesEnabled() bool {
	val := strings.ToLower(strings.TrimSpace(envx.Get("LPBS_SECURE_COOKIES")))
	// Default to secure in production
	if val == "" {
		// Check if we're in a production environment
		env := strings.ToLower(strings.TrimSpace(envx.Get("LPBS_ENVIRONMENT")))
		return env == "production" || env == "prod"
	}
	return val != "false" && val != "0" && val != "no"
}

// generateSessionID generates a cryptographically secure session ID.
func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

const (
	defaultAdminEmail = "admin@localhost"
	seededAdminID     = 1 // Reserved ID for the seeded/default admin account
)

// secretStore resolves project-local secrets when VROOLI_ROOT explicitly points
// to one, otherwise the user-local secrets store. Environment values always win.
func secretStore() (*apisecrets.Store, error) {
	if root := strings.TrimSpace(envx.Get("VROOLI_ROOT")); root != "" {
		path := filepath.Join(root, ".vrooli", "secrets.json")
		if _, err := os.Stat(path); err == nil {
			return apisecrets.NewFileStore(path)
		}
	}
	return apisecrets.NewUserStore(apisecrets.Config{EnvLookup: envx.Get})
}

// resolveSecret resolves a secret from environment first, then an explicitly
// configured project secrets file or the shared user-local plaintext file.
func resolveSecret(key string) string {
	if value := strings.TrimSpace(envx.Get(key)); value != "" {
		return value
	}
	store, err := secretStore()
	if err != nil {
		return strings.TrimSpace(envx.Get(key))
	}
	return strings.TrimSpace(store.ResolveValue(key))
}

// findSecretsFile returns the resolved local plaintext user secrets path when
// it exists.
func findSecretsFile() string {
	store, err := secretStore()
	if err != nil {
		return ""
	}
	path := store.PlaintextPath()
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

// getAdminDefaults returns an explicitly configured admin credential. Development
// uses an ephemeral password so a committed default can never authenticate a user.
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

	if isProductionSecurityEnvironment() {
		return "", "", fmt.Errorf("ADMIN_DEFAULT_PASSWORD must be configured in production")
	}
	ephemeralPassword := make([]byte, 32)
	if _, err := rand.Read(ephemeralPassword); err != nil {
		return "", "", fmt.Errorf("generate ephemeral admin password: %w", err)
	}
	passwordHashBytes, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(ephemeralPassword)), bcrypt.DefaultCost)
	if err != nil {
		return "", "", fmt.Errorf("hash ephemeral admin password: %w", err)
	}
	logStructured("admin_password_missing", map[string]interface{}{
		"level": "warn", "message": "ADMIN_DEFAULT_PASSWORD is not configured; generated an ephemeral development credential",
	})
	return email, string(passwordHashBytes), nil
}

// sessionStore is kept for backwards compatibility with sessionAdminEmail helper.
// New code should use Server.sessionManager instead.
var sessionStore *sessions.CookieStore

// initSessionManager creates and returns a SessionManager for the server.
// It also initializes the global sessionStore for backwards compatibility.
func initSessionManager() SessionManager {
	secret := resolveSecret("SESSION_SECRET")
	if secret == "" {
		ephemeral := make([]byte, 32)
		if _, err := rand.Read(ephemeral); err != nil {
			panic(fmt.Sprintf("generate ephemeral session key: %v", err))
		}
		logStructured("session_secret_missing", map[string]interface{}{
			"level":   "warn",
			"message": "SESSION_SECRET not set; generated an ephemeral key; sessions will not survive restart",
			"action":  "Set SESSION_SECRET in environment or ~/.vrooli/secrets.json",
		})
		secret = hex.EncodeToString(ephemeral)
	}
	// Initialize global for backwards compatibility
	sessionStore = sessions.NewCookieStore([]byte(secret))
	return NewCookieSessionManager(secret)
}

func isProductionSecurityEnvironment() bool {
	environment := strings.ToLower(strings.TrimSpace(envx.Get("LPBS_ENVIRONMENT")))
	return environment == "production" || environment == "prod"
}

func validateProductionCredentials() error {
	if !isProductionSecurityEnvironment() {
		return nil
	}
	if strings.TrimSpace(resolveSecret("SESSION_SECRET")) == "" {
		return fmt.Errorf("SESSION_SECRET must be configured in production")
	}
	if strings.TrimSpace(resolveSecret("ADMIN_DEFAULT_PASSWORD")) == "" {
		return fmt.Errorf("ADMIN_DEFAULT_PASSWORD must be configured in production")
	}
	return nil
}

// LoginRequest and LoginResponse remain package aliases for remote-profile
// transport compatibility. HTTP ownership lives in handlers/admin.
type (
	LoginRequest  = adminhttp.LoginRequest
	LoginResponse = adminhttp.SessionResponse
)

func (s *Server) adminSessionDependencies() adminhttp.Dependencies {
	return adminhttp.Dependencies{
		Auth:          s.adminAuth(),
		Sessions:      s.sessionManager,
		GenerateID:    generateSessionID,
		Now:           time.Now,
		ClientIP:      getClientIP,
		SecureCookies: isSecureCookiesEnabled,
		WriteError:    writeJSONError,
		Log:           logStructured,
		LogError:      logStructuredError,
	}
}

func (s *Server) adminAuth() *AdminAuthService {
	if s.adminAuthService != nil {
		return s.adminAuthService
	}
	if s.routedDB != nil {
		return NewAdminAuthService(s.routedDB)
	}
	return NewAdminAuthService(s.db)
}

// requireAdminOrService accepts either admin session cookie OR service bearer token.
// Used for endpoints that need admin-level access from both the UI and inter-scenario calls.
func (s *Server) requireAdminOrService(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try service token first (inter-scenario calls)
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			token := strings.TrimPrefix(auth, "Bearer ")
			if s.usageService.ValidateServiceToken(token) {
				next(w, r)
				return
			}
		}
		// Fall back to admin session (browser/CLI calls)
		s.requireAdmin(next)(w, r)
	}
}

// requireAdmin is middleware to protect admin routes
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := s.sessionManager.GetSession(r, "admin_session")
		email, ok := session.Values["email"].(string)
		if !ok || email == "" {
			writeJSONError(w, http.StatusUnauthorized, "Session expired. Please log in again.", ApiErrorTypeUnauthorized)
			return
		}

		// Validate server-side session
		serverSessionID, _ := session.Values["session_id"].(string)
		if serverSessionID != "" {
			expiresAt, err := s.adminAuth().SessionExpiry(r.Context(), serverSessionID, email)

			if err == sql.ErrNoRows || (err == nil && time.Now().After(expiresAt)) {
				// Session not found or expired
				session.Options.MaxAge = -1
				if saveErr := s.sessionManager.SaveSession(r, w, session); saveErr != nil {
					logStructuredError("middleware_session_save_failed", map[string]interface{}{
						"error": saveErr.Error(),
					})
				}
				writeJSONError(w, http.StatusUnauthorized, "Session expired. Please log in again.", ApiErrorTypeUnauthorized)
				return
			} else if err != nil {
				logStructuredError("middleware_session_lookup_failed", map[string]interface{}{
					"error": err.Error(),
				})
				// Fall through on DB error (graceful degradation)
			}
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
	envEmail, _, _ := getAdminDefaults()
	configuredPassword := resolveSecret("ADMIN_DEFAULT_PASSWORD")
	isDefaultPassword := configuredPassword != "" && bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(configuredPassword)) == nil
	return AdminProfileResponse{
		Email:             email,
		IsDefaultEmail:    strings.EqualFold(email, envEmail),
		IsDefaultPassword: isDefaultPassword,
	}
}

func (s *Server) sessionAdminEmail(r *http.Request) (string, bool) {
	session, _ := s.sessionManager.GetSession(r, "admin_session")
	email, ok := session.Values["email"].(string)
	if !ok || strings.TrimSpace(email) == "" {
		return "", false
	}
	return email, true
}

// handleAdminProfile returns the authenticated admin's profile
func (s *Server) handleAdminProfile(w http.ResponseWriter, r *http.Request) {
	email, ok := s.sessionAdminEmail(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "Session expired. Please log in again.", ApiErrorTypeUnauthorized)
		return
	}

	profile, err := s.adminAuth().Profile(r.Context(), email)
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
	if err := json.NewEncoder(w).Encode(buildAdminProfileResponse(profile.Email, profile.PasswordHash)); err != nil {
		logStructuredError("admin_profile_encode_failed", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// handleAdminProfileUpdate updates the admin email and/or password
func (s *Server) handleAdminProfileUpdate(w http.ResponseWriter, r *http.Request) {
	currentEmail, ok := s.sessionAdminEmail(r)
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

	profile, err := s.adminAuth().Profile(r.Context(), currentEmail)
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

	if err := bcrypt.CompareHashAndPassword([]byte(profile.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Invalid credentials", ApiErrorTypeUnauthorized)
		return
	}

	targetEmail := profile.Email
	if req.NewEmail != "" && !strings.EqualFold(req.NewEmail, profile.Email) {
		// Use centralized email validation (RFC 5322 compliant)
		if _, err := ValidateEmail(req.NewEmail); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid email address", ApiErrorTypeValidation)
			return
		}
		exists, err := s.adminAuth().EmailInUse(r.Context(), req.NewEmail, profile.ID)
		if err != nil {
			logStructuredError("admin_email_validation_failed", map[string]interface{}{
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to validate email. Please try again.", ApiErrorTypeServerError)
			return
		}
		if exists {
			writeJSONError(w, http.StatusConflict, "Email already in use", ApiErrorTypeValidation)
			return
		}
		targetEmail = req.NewEmail
	}

	targetPasswordHash := profile.PasswordHash
	if req.NewPassword != "" {
		if err := validateAdminPasswordUpdate(req.NewPassword, profile.PasswordHash); err != nil {
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

	if targetEmail == profile.Email && targetPasswordHash == profile.PasswordHash {
		writeJSONError(w, http.StatusBadRequest, "No changes detected", ApiErrorTypeValidation)
		return
	}

	if err := s.adminAuth().UpdateProfile(r.Context(), profile.ID, targetEmail, targetPasswordHash); err != nil {
		logStructuredError("admin_profile_update_failed", map[string]interface{}{
			"error": err.Error(),
		})
		writeJSONError(w, http.StatusInternalServerError, "Failed to update profile. Please try again.", ApiErrorTypeServerError)
		return
	}

	session, _ := s.sessionManager.GetSession(r, "admin_session")
	currentSessionID, _ := session.Values["session_id"].(string)

	// If password was changed, invalidate all other sessions for security
	if targetPasswordHash != profile.PasswordHash {
		affected, err := s.adminAuth().RevokeOtherSessions(r.Context(), currentEmail, currentSessionID)
		if err != nil {
			logStructuredError("admin_sessions_invalidation_failed", map[string]interface{}{
				"error": err.Error(),
				"email": currentEmail,
			})
		} else if affected > 0 {
			logStructured("admin_sessions_invalidated_on_password_change", map[string]interface{}{
				"level":            "info",
				"email":            currentEmail,
				"sessions_revoked": affected,
				"security":         true,
			})
		}
	}

	// Update email in current session if changed
	session.Values["email"] = targetEmail
	if err := s.sessionManager.SaveSession(r, w, session); err != nil {
		logStructuredError("session_save_after_profile_update_failed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	logStructured("admin_profile_updated", map[string]interface{}{
		"level":          "info",
		"changed_email":  targetEmail != profile.Email,
		"changed_secret": targetPasswordHash != profile.PasswordHash,
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
