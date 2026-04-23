package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// UserAccountResponse represents a user account with enriched data.
type UserAccountResponse struct {
	ID               string            `json:"id"`
	Email            string            `json:"email"`
	EmailVerified    bool              `json:"email_verified"`
	StripeCustomerID *string           `json:"stripe_customer_id,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	LastLoginAt      *time.Time        `json:"last_login_at,omitempty"`
	Subscription     *SubscriptionInfo `json:"subscription,omitempty"`
	Credits          *CreditInfo       `json:"credits,omitempty"`
	SessionCount     int               `json:"session_count"`
}

// SubscriptionInfo contains subscription details for a user.
type SubscriptionInfo struct {
	Status   string `json:"status"`
	PlanTier string `json:"plan_tier"`
}

// CreditInfo contains credit balance details for a user.
type CreditInfo struct {
	Balance int64 `json:"balance"`
	Bonus   int64 `json:"bonus"`
}

// UserSessionResponse represents a user session.
type UserSessionResponse struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	IPAddress  *string   `json:"ip_address,omitempty"`
	UserAgent  *string   `json:"user_agent,omitempty"`
	Revoked    bool      `json:"revoked"`
}

// UsersListResponse is the paginated response for user listing.
type UsersListResponse struct {
	Users      []UserAccountResponse `json:"users"`
	Total      int                   `json:"total"`
	Page       int                   `json:"page"`
	PerPage    int                   `json:"per_page"`
	TotalPages int                   `json:"total_pages"`
}

// handleAdminListUsers returns a paginated list of users with optional search.
// GET /api/v1/admin/users?search=email&page=1&per_page=20
func handleAdminListUsers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse query parameters
		search := strings.TrimSpace(r.URL.Query().Get("search"))
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

		if page < 1 {
			page = 1
		}
		if perPage < 1 || perPage > 100 {
			perPage = 20
		}
		offset := (page - 1) * perPage

		// Build query with optional search
		var countQuery, selectQuery string
		var args []interface{}

		if search != "" {
			searchPattern := "%" + strings.ToLower(search) + "%"
			countQuery = `SELECT COUNT(*) FROM users WHERE LOWER(email) LIKE $1`
			selectQuery = `
				SELECT id, email, email_verified, stripe_customer_id, created_at, last_login_at
				FROM users
				WHERE LOWER(email) LIKE $1
				ORDER BY created_at DESC
				LIMIT $2 OFFSET $3
			`
			args = []interface{}{searchPattern, perPage, offset}
		} else {
			countQuery = `SELECT COUNT(*) FROM users`
			selectQuery = `
				SELECT id, email, email_verified, stripe_customer_id, created_at, last_login_at
				FROM users
				ORDER BY created_at DESC
				LIMIT $1 OFFSET $2
			`
			args = []interface{}{perPage, offset}
		}

		// Get total count
		var total int
		countArgs := args[:0]
		if search != "" {
			countArgs = []interface{}{"%" + strings.ToLower(search) + "%"}
		}
		if err := db.QueryRowContext(r.Context(), countQuery, countArgs...).Scan(&total); err != nil {
			http.Error(w, "Failed to count users", http.StatusInternalServerError)
			return
		}

		// Get users
		rows, err := db.QueryContext(r.Context(), selectQuery, args...)
		if err != nil {
			http.Error(w, "Failed to query users", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var users []UserAccountResponse
		for rows.Next() {
			var user UserAccountResponse
			var stripeCustomerID sql.NullString
			var lastLoginAt sql.NullTime

			if err := rows.Scan(
				&user.ID,
				&user.Email,
				&user.EmailVerified,
				&stripeCustomerID,
				&user.CreatedAt,
				&lastLoginAt,
			); err != nil {
				continue
			}

			if stripeCustomerID.Valid {
				user.StripeCustomerID = &stripeCustomerID.String
			}
			if lastLoginAt.Valid {
				user.LastLoginAt = &lastLoginAt.Time
			}

			// Get subscription info
			user.Subscription = getUserSubscription(r.Context(), db, user.Email)

			// Get credit info
			user.Credits = getUserCredits(r.Context(), db, user.Email)

			// Get session count
			user.SessionCount = getUserSessionCount(r.Context(), db, user.ID)

			users = append(users, user)
		}

		if users == nil {
			users = []UserAccountResponse{}
		}

		totalPages := (total + perPage - 1) / perPage

		response := UsersListResponse{
			Users:      users,
			Total:      total,
			Page:       page,
			PerPage:    perPage,
			TotalPages: totalPages,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

// handleAdminGetUser returns details for a specific user.
// GET /api/v1/admin/users/:id
func handleAdminGetUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userID := vars["id"]

		if userID == "" {
			http.Error(w, "User ID required", http.StatusBadRequest)
			return
		}

		var user UserAccountResponse
		var stripeCustomerID sql.NullString
		var lastLoginAt sql.NullTime

		err := db.QueryRowContext(r.Context(), `
			SELECT id, email, email_verified, stripe_customer_id, created_at, last_login_at
			FROM users
			WHERE id = $1
		`, userID).Scan(
			&user.ID,
			&user.Email,
			&user.EmailVerified,
			&stripeCustomerID,
			&user.CreatedAt,
			&lastLoginAt,
		)

		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Failed to get user", http.StatusInternalServerError)
			return
		}

		if stripeCustomerID.Valid {
			user.StripeCustomerID = &stripeCustomerID.String
		}
		if lastLoginAt.Valid {
			user.LastLoginAt = &lastLoginAt.Time
		}

		user.Subscription = getUserSubscription(r.Context(), db, user.Email)
		user.Credits = getUserCredits(r.Context(), db, user.Email)
		user.SessionCount = getUserSessionCount(r.Context(), db, user.ID)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(user); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

// handleAdminGetUserSessions returns active sessions for a user.
// GET /api/v1/admin/users/:id/sessions
func handleAdminGetUserSessions(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userID := vars["id"]

		if userID == "" {
			http.Error(w, "User ID required", http.StatusBadRequest)
			return
		}

		rows, err := db.QueryContext(r.Context(), `
			SELECT id, created_at, last_used_at, expires_at, ip_address::text, user_agent, revoked
			FROM user_sessions
			WHERE user_id = $1
			ORDER BY last_used_at DESC
		`, userID)
		if err != nil {
			http.Error(w, "Failed to query sessions", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var sessions []UserSessionResponse
		for rows.Next() {
			var session UserSessionResponse
			var ipAddress, userAgent sql.NullString

			if err := rows.Scan(
				&session.ID,
				&session.CreatedAt,
				&session.LastUsedAt,
				&session.ExpiresAt,
				&ipAddress,
				&userAgent,
				&session.Revoked,
			); err != nil {
				continue
			}

			if ipAddress.Valid {
				session.IPAddress = &ipAddress.String
			}
			if userAgent.Valid {
				session.UserAgent = &userAgent.String
			}

			sessions = append(sessions, session)
		}

		if sessions == nil {
			sessions = []UserSessionResponse{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(sessions); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

// handleAdminRevokeUserSession revokes a specific user session.
// DELETE /api/v1/admin/users/:id/sessions/:sid
func handleAdminRevokeUserSession(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userID := vars["id"]
		sessionID := vars["sid"]

		if userID == "" || sessionID == "" {
			http.Error(w, "User ID and session ID required", http.StatusBadRequest)
			return
		}

		result, err := db.ExecContext(r.Context(), `
			UPDATE user_sessions
			SET revoked = TRUE
			WHERE id = $1 AND user_id = $2
		`, sessionID, userID)
		if err != nil {
			http.Error(w, "Failed to revoke session", http.StatusInternalServerError)
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}

		logStructured("admin_revoke_session", map[string]interface{}{
			"level":      "info",
			"user_id":    userID,
			"session_id": sessionID,
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Session revoked",
		}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

// handleAdminRevokeAllUserSessions revokes all sessions for a user.
// POST /api/v1/admin/users/:id/sessions/revoke-all
func handleAdminRevokeAllUserSessions(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userID := vars["id"]

		if userID == "" {
			http.Error(w, "User ID required", http.StatusBadRequest)
			return
		}

		result, err := db.ExecContext(r.Context(), `
			UPDATE user_sessions
			SET revoked = TRUE
			WHERE user_id = $1 AND revoked = FALSE
		`, userID)
		if err != nil {
			http.Error(w, "Failed to revoke sessions", http.StatusInternalServerError)
			return
		}

		rowsAffected, _ := result.RowsAffected()

		logStructured("admin_revoke_all_sessions", map[string]interface{}{
			"level":            "info",
			"user_id":          userID,
			"sessions_revoked": rowsAffected,
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success":          true,
			"message":          "All sessions revoked",
			"sessions_revoked": rowsAffected,
		}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

// Helper functions

func getUserSubscription(ctx context.Context, db *sql.DB, email string) *SubscriptionInfo {
	var status, planTier sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT status, plan_tier
		FROM subscriptions
		WHERE customer_email = $1 AND status IN ('active', 'trialing')
		ORDER BY created_at DESC
		LIMIT 1
	`, strings.ToLower(email)).Scan(&status, &planTier)

	if err != nil || !status.Valid {
		return nil
	}

	return &SubscriptionInfo{
		Status:   status.String,
		PlanTier: planTier.String,
	}
}

func getUserCredits(ctx context.Context, db *sql.DB, email string) *CreditInfo {
	var balance, bonus int64
	err := db.QueryRowContext(ctx, `
		SELECT balance_credits, bonus_credits
		FROM credit_wallets
		WHERE customer_email = $1
	`, strings.ToLower(email)).Scan(&balance, &bonus)
	if err != nil {
		return nil
	}

	return &CreditInfo{
		Balance: balance,
		Bonus:   bonus,
	}
}

func getUserSessionCount(ctx context.Context, db *sql.DB, userID string) int {
	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM user_sessions
		WHERE user_id = $1 AND revoked = FALSE AND expires_at > NOW()
	`, userID).Scan(&count); err != nil {
		return 0
	}
	return count
}
