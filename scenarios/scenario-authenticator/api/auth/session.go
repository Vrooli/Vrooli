package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"scenario-authenticator/db"
	"time"
)

// Session represents a user session
type Session struct {
	SessionID    string    `json:"session_id"`
	UserID       string    `json:"user_id"`
	Token        string    `json:"jwt_token"`
	RefreshToken string    `json:"refresh_token"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// StoreSession stores a new session in Redis
func StoreSession(userID, token, refreshToken string, r *http.Request) error {
	session := Session{
		SessionID:    GenerateRefreshToken(), // Use same function for session ID
		UserID:       userID,
		Token:        token,
		RefreshToken: refreshToken,
		IPAddress:    GetClientIP(r),
		UserAgent:    r.Header.Get("User-Agent"),
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		CreatedAt:    time.Now(),
	}
	
	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	
	// Store in Redis with expiration
	err = db.RedisClient.Set(db.Ctx, 
		fmt.Sprintf("session:%s", session.SessionID), 
		sessionData, 
		7*24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}
	
	// Also store user->sessions mapping
	err = db.RedisClient.SAdd(db.Ctx, 
		fmt.Sprintf("user_sessions:%s", userID), 
		session.SessionID).Err()
	if err != nil {
		log.Printf("Warning: Failed to add session to user set: %v", err)
	}
	
	return nil
}

// StoreRefreshToken stores a refresh token in Redis
func StoreRefreshToken(userID, refreshToken string) {
	tokenHash := HashToken(refreshToken)
	err := db.RedisClient.Set(db.Ctx, 
		fmt.Sprintf("refresh:%s", tokenHash), 
		userID, 
		7*24*time.Hour).Err()
	if err != nil {
		log.Printf("Failed to store refresh token: %v", err)
	}
}

// ValidateRefreshToken validates a refresh token and returns the user ID
func ValidateRefreshToken(refreshToken string) string {
	tokenHash := HashToken(refreshToken)
	userID, err := db.RedisClient.Get(db.Ctx, fmt.Sprintf("refresh:%s", tokenHash)).Result()
	if err != nil {
		return ""
	}
	return userID
}

// RevokeRefreshToken revokes a refresh token
func RevokeRefreshToken(refreshToken string) {
	tokenHash := HashToken(refreshToken)
	db.RedisClient.Del(db.Ctx, fmt.Sprintf("refresh:%s", tokenHash))
}

// ClearUserSessions clears all sessions for a user
func ClearUserSessions(userID string) {
	// Get all session IDs for the user
	sessionIDs, err := db.RedisClient.SMembers(db.Ctx, fmt.Sprintf("user_sessions:%s", userID)).Result()
	if err != nil {
		log.Printf("Failed to get user sessions: %v", err)
		return
	}
	
	// Delete each session
	for _, sessionID := range sessionIDs {
		db.RedisClient.Del(db.Ctx, fmt.Sprintf("session:%s", sessionID))
	}
	
	// Delete the user sessions set
	db.RedisClient.Del(db.Ctx, fmt.Sprintf("user_sessions:%s", userID))
}

// BlacklistToken adds a token to the blacklist
func BlacklistToken(token string, expiresAt int64) {
	tokenHash := HashToken(token)
	duration := time.Unix(expiresAt, 0).Sub(time.Now())
	if duration > 0 {
		db.RedisClient.Set(db.Ctx, fmt.Sprintf("blacklist:%s", tokenHash), true, duration)
	}
}

// IsTokenBlacklisted checks if a token is blacklisted
func IsTokenBlacklisted(token string) bool {
	tokenHash := HashToken(token)
	exists, _ := db.RedisClient.Exists(db.Ctx, fmt.Sprintf("blacklist:%s", tokenHash)).Result()
	return exists > 0
}

// GetClientIP extracts the client IP address from the request
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}
	
	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	
	// Fall back to RemoteAddr
	return r.RemoteAddr
}