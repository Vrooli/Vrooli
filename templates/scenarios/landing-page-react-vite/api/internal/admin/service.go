// Package admin is the admin-operator application layer: bcrypt credential
// verification against admin_users and stateless signed-cookie sessions. The
// old flat API used gorilla/sessions; this rewrite replaces that with a
// hand-rolled HMAC-SHA256 signed token (no gorilla/sessions dependency) plus a
// Connect interceptor that gates admin-only procedures on a valid session
// cookie. The AdminAuth Connect handler in handlers/admin adapts this Service.
package admin

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/crypto/bcrypt"
)

// CookieName is the admin session cookie the Login RPC sets and the interceptor
// (and Session/Logout RPCs) read.
const CookieName = "admin_session"

// sessionMaxAge is the admin session lifetime (7 days), matching the old store.
const sessionMaxAge = 7 * 24 * time.Hour

// ErrInvalidCredentials is returned by Login for an unknown email or bad password.
var ErrInvalidCredentials = errors.New("invalid credentials")

// Service verifies admin credentials and mints/validates signed session tokens.
type Service struct {
	db     *sql.DB
	secret []byte
}

// NewService constructs the admin Service, reading the cookie signing secret
// from SESSION_SECRET (a development placeholder is used when unset).
func NewService(db *sql.DB) *Service {
	secret := strings.TrimSpace(os.Getenv("SESSION_SECRET"))
	if secret == "" {
		secret = "dev-session-placeholder"
	}
	return &Service{db: db, secret: []byte(secret)}
}

// Login verifies the email/password against admin_users and, on success, stamps
// last_login. It returns ErrInvalidCredentials for an unknown email or a
// password mismatch (indistinguishable to the caller, by design).
func (s *Service) Login(ctx context.Context, email, password string) error {
	var passwordHash string
	err := s.db.QueryRowContext(ctx, `SELECT password_hash FROM admin_users WHERE email = $1`, email).Scan(&passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidCredentials
	}
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE admin_users SET last_login = NOW() WHERE email = $1`, email)
	return nil
}

// EncodeToken returns a signed session token binding the admin email.
func (s *Service) EncodeToken(email string) string {
	payload := base64.RawURLEncoding.EncodeToString([]byte(email))
	return payload + "." + s.sign(payload)
}

// DecodeToken validates a signed token and returns the bound email.
func (s *Service) DecodeToken(token string) (string, bool) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", false
	}
	if !hmac.Equal([]byte(parts[1]), []byte(s.sign(parts[0]))) {
		return "", false
	}
	email, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || len(email) == 0 {
		return "", false
	}
	return string(email), true
}

func (s *Service) sign(payload string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// SessionCookie builds the Set-Cookie value that establishes a session.
func (s *Service) SessionCookie(email string) *http.Cookie {
	return &http.Cookie{
		Name:     CookieName,
		Value:    s.EncodeToken(email),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionMaxAge.Seconds()),
	}
}

// ClearCookie builds the Set-Cookie value that ends a session.
func (s *Service) ClearCookie() *http.Cookie {
	return &http.Cookie{Name: CookieName, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1}
}

// EmailFromHeader extracts and validates the admin email from a request's
// Cookie header, returning ("", false) when no valid session cookie is present.
func (s *Service) EmailFromHeader(header http.Header) (string, bool) {
	r := &http.Request{Header: header}
	cookie, err := r.Cookie(CookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return s.DecodeToken(cookie.Value)
}

// Interceptor returns a Connect interceptor that rejects any request lacking a
// valid admin session cookie with CodeUnauthenticated. Apply it to admin-only
// service handlers via connect.WithInterceptors.
func (s *Service) Interceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if _, ok := s.EmailFromHeader(req.Header()); !ok {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("admin session required"))
			}
			return next(ctx, req)
		}
	}
}

// ResetEnabled reports whether destructive demo-reset is permitted (env-gated).
func ResetEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("ENABLE_ADMIN_RESET")), "true")
}
