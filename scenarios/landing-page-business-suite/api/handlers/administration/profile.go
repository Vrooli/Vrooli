package administration

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
	"landing-page-business-suite-api/internal/administration"
)

type (
	ProfileResponse struct {
		Email             string `json:"email"`
		IsDefaultEmail    bool   `json:"is_default_email"`
		IsDefaultPassword bool   `json:"is_default_password"`
	}
)

type ProfileService interface {
	Profile(context.Context, string) (administration.AdminProfile, error)
	EmailInUse(context.Context, string, int64) (bool, error)
	UpdateProfile(context.Context, int64, string, string) error
	RevokeOtherSessions(context.Context, string, string) (int64, error)
}

type SessionRotator interface {
	RotateSession(context.Context, string, string, string) (string, error)
}

type ProfileDependencies struct {
	Auth            ProfileService
	Sessions        SessionManager
	DefaultEmail    func() string
	DefaultPassword func() string
	ValidateEmail   func(string) error
	Log             func(string, map[string]any)
	LogError        func(string, map[string]any)
	SecurityEvents  SecurityEvents
	Notifier        SecurityNotifier
}

func profileSessionEmail(deps ProfileDependencies, r *http.Request) (string, bool) {
	session, _ := deps.Sessions.GetSession(r, sessionName)
	if session == nil {
		return "", false
	}
	email, ok := session.Values["email"].(string)
	if !ok || strings.TrimSpace(email) == "" {
		return "", false
	}
	id, _ := session.Values["session_id"].(string)
	if reader, ok := deps.Auth.(interface {
		SessionState(context.Context, string, string) (administration.AdminSessionState, error)
	}); ok {
		if strings.TrimSpace(id) == "" {
			return "", false
		}
		state, err := reader.SessionState(r.Context(), id, email)
		if err != nil || errors.Is(err, sql.ErrNoRows) || time.Now().After(state.ExpiresAt) || time.Now().After(state.LastActivity.Add(AdminSessionIdleTTL())) {
			return "", false
		}
	}
	return email, true
}

func profileResponse(deps ProfileDependencies, email, passwordHash string) ProfileResponse {
	configured := deps.DefaultPassword()
	return ProfileResponse{Email: email, IsDefaultEmail: strings.EqualFold(email, deps.DefaultEmail()), IsDefaultPassword: configured != "" && bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(configured)) == nil}
}

func ValidateProfilePassword(candidate, currentHash, defaultPassword string) error {
	if len(candidate) < 12 {
		return fmt.Errorf("Password must be at least 12 characters")
	}
	letter, digit := false, false
	for _, c := range candidate {
		letter = letter || unicode.IsLetter(c)
		digit = digit || unicode.IsDigit(c)
	}
	if !letter || !digit {
		return fmt.Errorf("Password must include letters and numbers")
	}
	if bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(candidate)) == nil {
		return fmt.Errorf("New password must be different from the current password")
	}
	if defaultPassword != "" && subtle.ConstantTimeCompare([]byte(defaultPassword), []byte(candidate)) == 1 {
		return fmt.Errorf("New password cannot use the default credential")
	}
	return nil
}
