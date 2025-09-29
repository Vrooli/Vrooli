package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"scenario-authenticator/auth"
	"scenario-authenticator/db"
	"scenario-authenticator/models"
	"scenario-authenticator/utils"

	"github.com/go-chi/chi/v5"
)

type sessionView struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func toSessionView(session auth.Session) sessionView {
	return sessionView{
		SessionID: session.SessionID,
		UserID:    session.UserID,
		IPAddress: session.IPAddress,
		UserAgent: session.UserAgent,
		CreatedAt: session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
	}
}

func hasRole(roles []string, target string) bool {
	for _, role := range roles {
		if strings.EqualFold(role, target) {
			return true
		}
	}
	return false
}

// GetSessionsHandler lists active sessions
func GetSessionsHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("claims").(*models.Claims)
	if !ok {
		utils.SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 200
	if limitParam := strings.TrimSpace(r.URL.Query().Get("limit")); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 {
			if parsed > 1000 {
				parsed = 1000
			}
			limit = parsed
		}
	}

	requestedUserID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	scope := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("scope")))
	isAdmin := hasRole(claims.Roles, "admin")

	if requestedUserID != "" && requestedUserID != claims.UserID && !isAdmin {
		utils.SendError(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	if requestedUserID == "" && (!isAdmin || scope != "all") {
		requestedUserID = claims.UserID
	}

	if requestedUserID != "" {
		sessionIDs, err := db.RedisClient.SMembers(db.Ctx, "user_sessions:"+requestedUserID).Result()
		if err != nil {
			log.Printf("Failed to fetch sessions for user %s: %v", requestedUserID, err)
			utils.SendError(w, "Failed to fetch sessions", http.StatusInternalServerError)
			return
		}

		sanitized := make([]sessionView, 0, len(sessionIDs))
		for _, sessionID := range sessionIDs {
			sessionData, err := db.RedisClient.Get(db.Ctx, "session:"+sessionID).Result()
			if err != nil {
				continue
			}

			var session auth.Session
			if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
				continue
			}

			sanitized = append(sanitized, toSessionView(session))
		}

		sort.Slice(sanitized, func(i, j int) bool {
			return sanitized[i].CreatedAt.After(sanitized[j].CreatedAt)
		})

		if len(sanitized) > limit {
			sanitized = sanitized[:limit]
		}

		utils.SendJSON(w, map[string]interface{}{
			"sessions": sanitized,
			"total":    len(sessionIDs),
		}, http.StatusOK)
		return
	}

	if !isAdmin {
		utils.SendError(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	sanitized := make([]sessionView, 0, limit)
	totalCount := 0
	cursor := uint64(0)

	for {
		keys, nextCursor, err := db.RedisClient.Scan(db.Ctx, cursor, "session:*", 200).Result()
		if err != nil {
			log.Printf("Failed to scan sessions: %v", err)
			utils.SendError(w, "Failed to fetch sessions", http.StatusInternalServerError)
			return
		}

		totalCount += len(keys)

		for _, key := range keys {
			if len(sanitized) >= limit {
				continue
			}

			sessionData, err := db.RedisClient.Get(db.Ctx, key).Result()
			if err != nil {
				continue
			}

			var session auth.Session
			if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
				continue
			}

			sanitized = append(sanitized, toSessionView(session))
		}

		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}

	sort.Slice(sanitized, func(i, j int) bool {
		return sanitized[i].CreatedAt.After(sanitized[j].CreatedAt)
	})

	response := map[string]interface{}{
		"sessions": sanitized,
		"total":    totalCount,
	}

	if totalCount > len(sanitized) {
		response["has_more"] = true
	}

	utils.SendJSON(w, response, http.StatusOK)
}

// RevokeSessionHandler revokes a specific session
func RevokeSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	claims, ok := r.Context().Value("claims").(*models.Claims)
	if !ok {
		utils.SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get session to find associated tokens
	sessionData, err := db.RedisClient.Get(db.Ctx, "session:"+sessionID).Result()
	if err != nil {
		utils.SendError(w, "Session not found", http.StatusNotFound)
		return
	}

	var session auth.Session
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		utils.SendError(w, "Invalid session data", http.StatusInternalServerError)
		return
	}

	isAdmin := hasRole(claims.Roles, "admin")
	if session.UserID != claims.UserID && !isAdmin {
		utils.SendError(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	// Blacklist the token
	auth.BlacklistToken(session.Token, session.ExpiresAt.Unix())

	// Revoke refresh token
	auth.RevokeRefreshToken(session.RefreshToken)

	// Delete session
	db.RedisClient.Del(db.Ctx, "session:"+sessionID)

	// Remove from user's session set
	db.RedisClient.SRem(db.Ctx, "user_sessions:"+session.UserID, sessionID)

	utils.SendJSON(w, map[string]interface{}{
		"success": true,
		"message": "Session revoked successfully",
	}, http.StatusOK)
}
