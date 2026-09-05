package administration

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"time"
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

type SubscriptionInfo struct {
	Status   string `json:"status"`
	PlanTier string `json:"plan_tier"`
}

type CreditInfo struct {
	Balance int64 `json:"balance"`
	Bonus   int64 `json:"bonus"`
}

type UserSessionResponse struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	IPAddress  *string   `json:"ip_address,omitempty"`
	UserAgent  *string   `json:"user_agent,omitempty"`
	Revoked    bool      `json:"revoked"`
}

type UsersListResponse struct {
	Users      []UserAccountResponse `json:"users"`
	Total      int                   `json:"total"`
	Page       int                   `json:"page"`
	PerPage    int                   `json:"per_page"`
	TotalPages int                   `json:"total_pages"`
}

// UserManagementStore is the context-aware persistence contract for admin
// user-management operations.
//
// seam: UserManagementStore keeps admin user management independent of a
// concrete database pool and preserves request-scoped test isolation.
type UserManagementStore interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// UserManagementService owns user-account and user-session administration.
type UserManagementService struct{ db UserManagementStore }

func NewUserManagementService(db UserManagementStore) *UserManagementService {
	return &UserManagementService{db: db}
}

func (s *UserManagementService) List(ctx context.Context, search string, page, perPage int) (UsersListResponse, error) {
	search = strings.TrimSpace(search)
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var total int
	var rows *sql.Rows
	var err error
	if search == "" {
		if err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
			return UsersListResponse{}, err
		}
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, email, email_verified, stripe_customer_id, created_at, last_login_at
			FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`, perPage, offset)
	} else {
		pattern := "%" + strings.ToLower(search) + "%"
		if err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE LOWER(email) LIKE $1`, pattern).Scan(&total); err != nil {
			return UsersListResponse{}, err
		}
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, email, email_verified, stripe_customer_id, created_at, last_login_at
			FROM users WHERE LOWER(email) LIKE $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, pattern, perPage, offset)
	}
	if err != nil {
		return UsersListResponse{}, err
	}
	defer rows.Close()

	users := make([]UserAccountResponse, 0)
	for rows.Next() {
		user, err := scanUserAccount(rows)
		if err != nil {
			continue
		}
		s.enrich(ctx, &user)
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return UsersListResponse{}, err
	}
	return UsersListResponse{Users: users, Total: total, Page: page, PerPage: perPage, TotalPages: (total + perPage - 1) / perPage}, nil
}

func (s *UserManagementService) Get(ctx context.Context, userID string) (*UserAccountResponse, error) {
	var user UserAccountResponse
	var stripeCustomerID sql.NullString
	var lastLoginAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, email_verified, stripe_customer_id, created_at, last_login_at
		FROM users WHERE id = $1`, userID).Scan(&user.ID, &user.Email, &user.EmailVerified, &stripeCustomerID, &user.CreatedAt, &lastLoginAt)
	if err != nil {
		return nil, err
	}
	if stripeCustomerID.Valid {
		user.StripeCustomerID = &stripeCustomerID.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	s.enrich(ctx, &user)
	return &user, nil
}

func (s *UserManagementService) ListSessions(ctx context.Context, userID string) ([]UserSessionResponse, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, created_at, last_used_at, expires_at, ip_address::text, user_agent, revoked
		FROM user_sessions WHERE user_id = $1 ORDER BY last_used_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sessions := make([]UserSessionResponse, 0)
	for rows.Next() {
		var session UserSessionResponse
		var ipAddress, userAgent sql.NullString
		if err := rows.Scan(&session.ID, &session.CreatedAt, &session.LastUsedAt, &session.ExpiresAt, &ipAddress, &userAgent, &session.Revoked); err != nil {
			return nil, err
		}
		if ipAddress.Valid {
			session.IPAddress = &ipAddress.String
		}
		if userAgent.Valid {
			session.UserAgent = &userAgent.String
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (s *UserManagementService) RevokeSession(ctx context.Context, userID, sessionID string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE user_sessions SET revoked = TRUE WHERE id = $1 AND user_id = $2`, sessionID, userID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

func (s *UserManagementService) RevokeAllSessions(ctx context.Context, userID string) (int64, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE user_sessions SET revoked = TRUE WHERE user_id = $1 AND revoked = FALSE`, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *UserManagementService) enrich(ctx context.Context, user *UserAccountResponse) {
	var status, planTier sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT status, plan_tier FROM subscriptions
		WHERE customer_email = $1 AND status IN ('active', 'trialing')
		ORDER BY created_at DESC LIMIT 1`, strings.ToLower(user.Email)).Scan(&status, &planTier)
	if err == nil && status.Valid {
		user.Subscription = &SubscriptionInfo{Status: status.String, PlanTier: planTier.String}
	}

	var balance, bonus int64
	err = s.db.QueryRowContext(ctx, `SELECT balance_credits, bonus_credits FROM credit_wallets WHERE customer_email = $1`, strings.ToLower(user.Email)).Scan(&balance, &bonus)
	if err == nil {
		user.Credits = &CreditInfo{Balance: balance, Bonus: bonus}
	}

	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user_sessions WHERE user_id = $1 AND revoked = FALSE AND expires_at > NOW()`, user.ID).Scan(&user.SessionCount); err != nil {
		log.Printf("user session count lookup failed for user %s: %v", user.ID, err)
		user.SessionCount = 0
	}
}

type userAccountScanner interface{ Scan(...any) error }

func scanUserAccount(scanner userAccountScanner) (UserAccountResponse, error) {
	var user UserAccountResponse
	var stripeCustomerID sql.NullString
	var lastLoginAt sql.NullTime
	err := scanner.Scan(&user.ID, &user.Email, &user.EmailVerified, &stripeCustomerID, &user.CreatedAt, &lastLoginAt)
	if err != nil {
		return UserAccountResponse{}, err
	}
	if stripeCustomerID.Valid {
		user.StripeCustomerID = &stripeCustomerID.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	return user, nil
}
