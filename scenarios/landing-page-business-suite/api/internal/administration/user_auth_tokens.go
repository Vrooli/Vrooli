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
	"github.com/vrooli/api-core/consumeridentity"
)

// RefreshTokens validates a refresh token and returns a new token pair.
func (s *UserAuthService) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	return s.RefreshTokensFromSource(ctx, refreshToken, RefreshSourceBody)
}

// RefreshTokensFromSource applies the cookie-only grace policy. Body-token
// clients retain strict single-use replay detection.
func (s *UserAuthService) RefreshTokensFromSource(ctx context.Context, refreshToken string, source RefreshSource) (*TokenPair, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, ErrTokenInvalid
	}

	tokenHash := HashToken(refreshToken)

	// Find session by refresh token hash
	var sessionID, userID, familyID string
	var expiresAt, absoluteExpiresAt time.Time
	var revoked bool

	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, refresh_token_family_id, expires_at, absolute_expires_at, revoked
		FROM user_sessions
		WHERE refresh_token_hash = $1
	`, tokenHash).Scan(&sessionID, &userID, &familyID, &expiresAt, &absoluteExpiresAt, &revoked)

	if errors.Is(err, sql.ErrNoRows) {
		var historySession, historyUser, historyFamily string
		var retiredAt, historyExpires, historyAbsolute time.Time
		var historyRevoked bool
		historyErr := s.db.QueryRowContext(ctx, `
			SELECT h.session_id, s.user_id, h.family_id, h.retired_at, s.expires_at, s.absolute_expires_at, s.revoked
			FROM refresh_token_history h JOIN user_sessions s ON s.id = h.session_id
			WHERE h.refresh_token_hash = $1
		`, tokenHash).Scan(&historySession, &historyUser, &historyFamily, &retiredAt, &historyExpires, &historyAbsolute, &historyRevoked)
		if historyErr == nil {
			now := time.Now().UTC()
			if source == RefreshSourceCookie && !historyRevoked && now.Before(historyExpires) && now.Before(historyAbsolute) && now.Sub(retiredAt) <= s.refreshGrace {
				user, userErr := s.GetUserByID(ctx, historyUser)
				if userErr != nil {
					return nil, fmt.Errorf("get user for refresh grace: %w", userErr)
				}
				accessToken, accessExpiresAt, tokenErr := s.GenerateAccessToken(user.ID, user.Email, historySession)
				if tokenErr != nil {
					return nil, tokenErr
				}
				return &TokenPair{AccessToken: accessToken, ExpiresAt: accessExpiresAt, TokenType: "Bearer", SessionExpiresAt: historyExpires, GraceResponse: true}, nil
			}
			_, _ = s.db.ExecContext(ctx, `UPDATE user_sessions SET revoked = TRUE, revoked_at = NOW(), revoked_reason = 'reuse' WHERE refresh_token_family_id = $1`, historyFamily)
			return nil, ErrSessionRevoked
		}
		return nil, ErrTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("query session: %w", err)
	}

	if revoked {
		return nil, ErrSessionRevoked
	}

	now := time.Now().UTC()
	if now.After(absoluteExpiresAt) {
		_, _ = s.db.ExecContext(ctx, `UPDATE user_sessions SET revoked = TRUE, revoked_at = NOW(), revoked_reason = 'expired' WHERE id = $1`, sessionID)
		return nil, ErrSessionExpired
	}
	if now.After(expiresAt) {
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
	if _, err := s.db.ExecContext(ctx, `INSERT INTO refresh_token_history (refresh_token_hash, session_id, family_id) VALUES ($1, $2, $3) ON CONFLICT (refresh_token_hash) DO NOTHING`, tokenHash, sessionID, familyID); err != nil {
		return nil, fmt.Errorf("retire refresh token: %w", err)
	}

	// Update session with new refresh token and extend expiry (use UTC for consistent timezone handling)
	newExpiresAt := time.Now().UTC().Add(s.idleTTL)
	if newExpiresAt.After(absoluteExpiresAt) {
		newExpiresAt = absoluteExpiresAt
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE user_sessions
		SET refresh_token_hash = $1, expires_at = $2, last_used_at = NOW()
		WHERE id = $3 AND refresh_token_hash = $4 AND revoked = FALSE
	`, newRefreshHash, newExpiresAt, sessionID, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("update session: %w", err)
	}
	if rows, rowsErr := result.RowsAffected(); rowsErr != nil {
		return nil, fmt.Errorf("inspect rotated session: %w", rowsErr)
	} else if rows != 1 {
		// Another request won the single-use rotation race. Treat the old
		// credential as a replay and revoke the whole family, including the
		// winner's newly-issued credential.
		if source == RefreshSourceCookie {
			var retiredAt time.Time
			if graceErr := s.db.QueryRowContext(ctx, `SELECT retired_at FROM refresh_token_history WHERE refresh_token_hash = $1`, tokenHash).Scan(&retiredAt); graceErr == nil && time.Since(retiredAt) <= s.refreshGrace {
				accessToken, accessExpiresAt, tokenErr := s.GenerateAccessToken(user.ID, user.Email, sessionID)
				if tokenErr == nil {
					return &TokenPair{AccessToken: accessToken, ExpiresAt: accessExpiresAt, TokenType: "Bearer", SessionExpiresAt: newExpiresAt, GraceResponse: true}, nil
				}
			}
		}
		_, _ = s.db.ExecContext(ctx, `UPDATE user_sessions SET revoked = TRUE, revoked_at = NOW(), revoked_reason = 'reuse' WHERE refresh_token_family_id = $1`, familyID)
		return nil, ErrSessionRevoked
	}

	// Generate new access token
	accessToken, accessExpiresAt, err := s.GenerateAccessToken(user.ID, user.Email, sessionID)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     newRefreshToken,
		ExpiresAt:        accessExpiresAt,
		TokenType:        "Bearer",
		SessionExpiresAt: newExpiresAt,
	}, nil
}

// ValidateAccessToken validates a JWT access token and returns the claims.
func (s *UserAuthService) ValidateAccessToken(tokenString string) (*UserClaims, error) {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return nil, ErrTokenInvalid
	}
	claims, err := consumeridentity.NewVerifier(s.consumerKeys, s.jwtIssuer, s.consumerLeeway).Verify(tokenString)
	if err != nil {
		if errors.Is(err, consumeridentity.ErrExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	return &UserClaims{UserID: claims.UserID, Email: claims.Email, SessionID: claims.SessionID, RegisteredClaims: registeredClaims(claims)}, nil
}

func registeredClaims(claims consumeridentity.Claims) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		Issuer:    claims.Issuer,
		Subject:   claims.Subject,
		ExpiresAt: jwt.NewNumericDate(time.Unix(claims.ExpiresAt, 0)),
		IssuedAt:  jwt.NewNumericDate(time.Unix(claims.IssuedAt, 0)),
		NotBefore: jwt.NewNumericDate(time.Unix(claims.NotBefore, 0)),
	}
}

// PublicKeySet returns the currently active public keys. It is safe to serve
// this value publicly; no private signing material is included.
func (s *UserAuthService) PublicKeySet() ([]byte, error) { return s.consumerKeys.JWKS() }

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
