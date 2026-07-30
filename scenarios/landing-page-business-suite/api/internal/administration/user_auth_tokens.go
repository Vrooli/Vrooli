package administration

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// RefreshTokens validates a refresh token and returns a new token pair.
func (s *UserAuthService) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, ErrTokenInvalid
	}

	tokenHash := HashToken(refreshToken)

	// Find session by refresh token hash
	var sessionID, userID string
	var expiresAt time.Time
	var revoked bool

	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, expires_at, revoked
		FROM user_sessions
		WHERE refresh_token_hash = $1
	`, tokenHash).Scan(&sessionID, &userID, &expiresAt, &revoked)

	if err == sql.ErrNoRows {
		return nil, ErrTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("query session: %w", err)
	}

	if revoked {
		return nil, ErrSessionRevoked
	}

	if time.Now().After(expiresAt) {
		return nil, ErrTokenExpired
	}

	// Get user
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	// Generate new refresh token
	newRefreshBytes := make([]byte, 32)
	if _, err := rand.Read(newRefreshBytes); err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	newRefreshToken := hex.EncodeToString(newRefreshBytes)
	newRefreshHash := HashToken(newRefreshToken)

	// Update session with new refresh token and extend expiry (use UTC for consistent timezone handling)
	newExpiresAt := time.Now().UTC().Add(s.refreshTTL)
	_, err = s.db.ExecContext(ctx, `
		UPDATE user_sessions
		SET refresh_token_hash = $1, expires_at = $2, last_used_at = NOW()
		WHERE id = $3
	`, newRefreshHash, newExpiresAt, sessionID)
	if err != nil {
		return nil, fmt.Errorf("update session: %w", err)
	}

	// Generate new access token
	accessToken, accessExpiresAt, err := s.GenerateAccessToken(user.ID, user.Email, sessionID)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    accessExpiresAt,
		TokenType:    "Bearer",
	}, nil
}

// ValidateAccessToken validates a JWT access token and returns the claims.
func (s *UserAuthService) ValidateAccessToken(tokenString string) (*UserClaims, error) {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return nil, ErrTokenInvalid
	}

	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// Logout revokes the session associated with the given session ID.
func (s *UserAuthService) Logout(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE user_sessions
		SET revoked = TRUE
		WHERE id = $1
	`, sessionID)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	s.log("user_logout", map[string]interface{}{
		"level":      "info",
		"session_id": sessionID,
	})

	return nil
}

// LogoutAllSessions revokes all sessions for a user except the current one.
func (s *UserAuthService) LogoutAllSessions(ctx context.Context, userID, exceptSessionID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE user_sessions
		SET revoked = TRUE
		WHERE user_id = $1 AND id != $2
	`, userID, exceptSessionID)
	if err != nil {
		return fmt.Errorf("revoke all sessions: %w", err)
	}

	return nil
}
