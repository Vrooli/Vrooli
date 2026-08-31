package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"golang.org/x/crypto/bcrypt"
	adminhttp "landing-page-business-suite-api/handlers/administration"
	"landing-page-business-suite-api/internal/administration"
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

func resolveAuthorityCredential(key string) (string, error) {
	return administration.ResolveAuthorityCredential(key)
}

// resolveSecret is retained for optional credentials and compatibility edges.
// Security-critical generated/operator credentials use resolveAuthorityCredential
// directly so provider failures cannot become an empty-string fallback.
func resolveSecret(key string) string {
	return administration.ResolveSecret(key, logStructured)
}

func resolveGeneratedSecret(key string, mint func() (string, error)) (string, error) {
	return administration.ResolveGeneratedSecret(key, mint)
}

func resolveConfig(key string) string { return administration.ResolveConfig(key) }

// getAdminDefaults returns an explicitly configured admin credential. Development
// uses an ephemeral password so a committed default can never authenticate a user.
func getAdminDefaults() (email string, passwordHash string, err error) {
	email = resolveConfig("ADMIN_DEFAULT_EMAIL")
	if email == "" {
		email = defaultAdminEmail
	}

	// Check for plaintext password override (will be hashed)
	plaintextPassword, credentialErr := resolveAuthorityCredential("ADMIN_DEFAULT_PASSWORD")
	if credentialErr != nil && !errors.Is(credentialErr, credentialauthority.ErrUnconfigured) {
		return "", "", fmt.Errorf("resolve admin password: %w", credentialErr)
	}
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

// initSessionManager creates and returns a SessionManager for the server.
func initSessionManager() SessionManager {
	secret, err := resolveGeneratedSecret("SESSION_SECRET", func() (string, error) {
		return credentialauthority.RandomBase64(32)
	})
	if err != nil {
		panic(fmt.Sprintf("resolve session secret: %v", err))
	}
	if secret == "" {
		if envx.Get("LPBS_TEST_CREDENTIAL_FALLBACK") != "1" {
			panic("credential authority returned an empty session-secret")
		}
		ephemeral := make([]byte, 32)
		if _, err := rand.Read(ephemeral); err != nil {
			panic(fmt.Sprintf("generate ephemeral session key: %v", err))
		}
		logStructured("session_secret_missing", map[string]interface{}{
			"level":   "warn",
			"message": "SESSION_SECRET not set; generated an ephemeral key; sessions will not survive restart",
			"action":  "provision SESSION_SECRET with `vrooli credentials provision`",
		})
		secret = hex.EncodeToString(ephemeral)
	}
	previous := ""
	if envx.Get("LPBS_TEST_CREDENTIAL_FALLBACK") == "1" {
		previous = strings.TrimSpace(envx.Get("SESSION_SECRET_PREVIOUS"))
	} else {
		resolved, resolveErr := resolveAuthorityCredential("SESSION_SECRET_PREVIOUS")
		if resolveErr != nil && !errors.Is(resolveErr, credentialauthority.ErrUnconfigured) {
			panic(fmt.Sprintf("resolve previous session secret: %v", resolveErr))
		}
		previous = resolved
	}
	return NewCookieSessionManager(secret, previous)
}

// RotateSessionSecret overlaps verification: the old secret remains readable
// for the session lifetime while only the new active secret signs cookies.
func RotateSessionSecret() error {
	authority, err := credentialauthority.Default()
	if err != nil {
		return fmt.Errorf("initialize credential authority: %w", err)
	}
	identity := credentialauthority.Identity("vrooli/landing-page-business-suite")
	active, err := authority.Require(identity, "session-secret")
	if err != nil {
		return fmt.Errorf("resolve active session secret: %w", err)
	}
	if err := authority.Put(identity, "session-secret-previous", active); err != nil {
		return fmt.Errorf("retain previous session secret: %w", err)
	}
	newSecret, err := credentialauthority.RandomBase64(32)
	if err != nil {
		return err
	}
	if err := authority.Put(identity, "session-secret", newSecret); err != nil {
		return fmt.Errorf("store rotated session secret: %w", err)
	}
	return nil
}

func isProductionSecurityEnvironment() bool {
	environment := strings.ToLower(strings.TrimSpace(envx.Get("LPBS_ENVIRONMENT")))
	return environment == "production" || environment == "prod"
}

func validateProductionCredentials() error {
	if !isProductionSecurityEnvironment() {
		return nil
	}
	if _, err := resolveAuthorityCredential("SESSION_SECRET"); err != nil {
		return fmt.Errorf("SESSION_SECRET must be configured in production")
	}
	adminPassword, err := resolveAuthorityCredential("ADMIN_DEFAULT_PASSWORD")
	if err != nil {
		return fmt.Errorf("ADMIN_DEFAULT_PASSWORD must be configured in production: %w", err)
	}
	if strings.TrimSpace(adminPassword) == "" {
		return fmt.Errorf("ADMIN_DEFAULT_PASSWORD must be configured in production")
	}
	if len([]rune(adminPassword)) < 12 {
		return fmt.Errorf("ADMIN_DEFAULT_PASSWORD must be at least 12 characters in production")
	}
	magicLinkBaseURL := resolveConfig("AUTH_MAGIC_LINK_BASE_URL")
	if magicLinkBaseURL == "" {
		return fmt.Errorf("AUTH_MAGIC_LINK_BASE_URL must be configured in production")
	}
	parsedMagicLinkBaseURL, err := url.Parse(magicLinkBaseURL)
	if err != nil || parsedMagicLinkBaseURL.Scheme != "https" || parsedMagicLinkBaseURL.Host == "" {
		return fmt.Errorf("AUTH_MAGIC_LINK_BASE_URL must be an absolute https URL in production")
	}
	return nil
}

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

func (s *Server) adminProfileDependencies() adminhttp.ProfileDependencies {
	return adminhttp.ProfileDependencies{
		Auth: s.adminAuth(), Sessions: s.sessionManager,
		DefaultEmail:    func() string { email, _, _ := getAdminDefaults(); return email },
		DefaultPassword: func() string { return resolveSecret("ADMIN_DEFAULT_PASSWORD") },
		ValidateEmail:   func(email string) error { _, err := ValidateEmail(email); return err },
		Log:             logStructured, LogError: logStructuredError,
	}
}

func (s *Server) adminAuth() *administration.AdminAuthService {
	if s.adminAuthService != nil {
		return s.adminAuthService
	}
	if s.routedDB != nil {
		return administration.NewAdminAuthService(s.routedDB)
	}
	return administration.NewAdminAuthService(s.db)
}

// requireAdminOrService is retained as a route-composition seam for legacy
// admin handlers, but no longer accepts a shared service bearer token.
func (s *Server) requireAdminOrService(next http.HandlerFunc) http.HandlerFunc {
	return s.requireAdmin(next)
}

// requireAdmin is middleware to protect admin routes
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := s.sessionManager.GetSession(r, "admin_session")
		if err != nil || session == nil {
			writeJSONError(w, http.StatusUnauthorized, "Session expired. Please log in again.", ApiErrorTypeUnauthorized)
			return
		}
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

func (s *Server) sessionAdminEmail(r *http.Request) (string, bool) {
	session, err := s.sessionManager.GetSession(r, "admin_session")
	if err != nil || session == nil {
		return "", false
	}
	email, ok := session.Values["email"].(string)
	if !ok || strings.TrimSpace(email) == "" {
		return "", false
	}
	return email, true
}
