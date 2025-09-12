package handlers

import (
	"encoding/json"
	"net/http"
	"scenario-authenticator/auth"
	"scenario-authenticator/db"
	"scenario-authenticator/utils"

	"github.com/go-chi/chi/v5"
)

// GetSessionsHandler lists active sessions
func GetSessionsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	
	// Get sessions from Redis
	if userID != "" {
		// Get sessions for specific user
		sessionIDs, err := db.RedisClient.SMembers(db.Ctx, "user_sessions:"+userID).Result()
		if err != nil {
			utils.SendError(w, "Failed to fetch sessions", http.StatusInternalServerError)
			return
		}
		
		sessions := []auth.Session{}
		for _, sessionID := range sessionIDs {
			sessionData, err := db.RedisClient.Get(db.Ctx, "session:"+sessionID).Result()
			if err != nil {
				continue
			}
			
			var session auth.Session
			if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
				continue
			}
			
			sessions = append(sessions, session)
		}
		
		utils.SendJSON(w, map[string]interface{}{
			"sessions": sessions,
			"total":    len(sessions),
		}, http.StatusOK)
	} else {
		// Get all sessions (admin function)
		// This would need pagination in production
		utils.SendJSON(w, map[string]interface{}{
			"sessions": []auth.Session{},
			"total":    0,
			"message":  "Listing all sessions requires user_id parameter",
		}, http.StatusOK)
	}
}

// RevokeSessionHandler revokes a specific session
func RevokeSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	
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