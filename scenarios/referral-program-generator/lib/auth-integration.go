package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// ScenarioAuthenticator integration for user management
type AuthService struct {
	baseURL string
	apiKey  string
	db      *sql.DB
}

type AuthUser struct {
	ID       string `json:"id" db:"id"`
	Username string `json:"username" db:"username"`
	Email    string `json:"email" db:"email"`
	Role     string `json:"role" db:"role"`
}

type ReferralProfile struct {
	ID             string  `json:"id" db:"id"`
	UserID         string  `json:"user_id" db:"user_id"`
	ReferralCode   string  `json:"referral_code" db:"referral_code"`
	TotalEarnings  float64 `json:"total_earnings" db:"total_earnings"`
	ActiveLinks    int     `json:"active_links" db:"active_links"`
	TotalClicks    int     `json:"total_clicks" db:"total_clicks"`
	TotalConversions int   `json:"total_conversions" db:"total_conversions"`
	CreatedAt      string  `json:"created_at" db:"created_at"`
	UpdatedAt      string  `json:"updated_at" db:"updated_at"`
}

// Initialize auth service
func NewAuthService(db *sql.DB) *AuthService {
	return &AuthService{
		baseURL: getEnvWithDefault("SCENARIO_AUTHENTICATOR_URL", "http://localhost:8090"),
		apiKey:  os.Getenv("SCENARIO_AUTHENTICATOR_API_KEY"),
		db:      db,
	}
}

// Get user by ID from scenario-authenticator
func (as *AuthService) GetUser(userID string) (*AuthUser, error) {
	// First try to get from local cache/database
	var user AuthUser
	query := `SELECT id, username, email, role FROM users WHERE id = $1`
	err := as.db.QueryRow(query, userID).Scan(&user.ID, &user.Username, &user.Email, &user.Role)
	
	if err == nil {
		return &user, nil
	}
	
	// If not found locally, fetch from scenario-authenticator
	resp, err := http.Get(fmt.Sprintf("%s/api/users/%s", as.baseURL, userID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user from authenticator: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user not found in authenticator: %s", resp.Status)
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}
	
	// Cache user locally
	_, err = as.db.Exec(`
		INSERT INTO users (id, username, email, role, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET 
			username = EXCLUDED.username,
			email = EXCLUDED.email,
			role = EXCLUDED.role,
			updated_at = NOW()`,
		user.ID, user.Username, user.Email, user.Role)
	
	if err != nil {
		log.Printf("Failed to cache user locally: %v", err)
	}
	
	return &user, nil
}

// Create or get referral profile for user
func (as *AuthService) GetOrCreateReferralProfile(userID string) (*ReferralProfile, error) {
	// Check if profile exists
	var profile ReferralProfile
	query := `
		SELECT id, user_id, referral_code, total_earnings, active_links, 
		       total_clicks, total_conversions, created_at, updated_at
		FROM referral_profiles 
		WHERE user_id = $1`
	
	err := as.db.QueryRow(query, userID).Scan(
		&profile.ID, &profile.UserID, &profile.ReferralCode,
		&profile.TotalEarnings, &profile.ActiveLinks,
		&profile.TotalClicks, &profile.TotalConversions,
		&profile.CreatedAt, &profile.UpdatedAt)
	
	if err == nil {
		return &profile, nil
	}
	
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("database error: %w", err)
	}
	
	// Create new profile
	profile = ReferralProfile{
		ID:               uuid.New().String(),
		UserID:           userID,
		ReferralCode:     generateReferralCode(),
		TotalEarnings:    0.0,
		ActiveLinks:      0,
		TotalClicks:      0,
		TotalConversions: 0,
	}
	
	insertQuery := `
		INSERT INTO referral_profiles 
		(id, user_id, referral_code, total_earnings, active_links, total_clicks, total_conversions, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING created_at, updated_at`
	
	err = as.db.QueryRow(insertQuery,
		profile.ID, profile.UserID, profile.ReferralCode,
		profile.TotalEarnings, profile.ActiveLinks,
		profile.TotalClicks, profile.TotalConversions).Scan(&profile.CreatedAt, &profile.UpdatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create referral profile: %w", err)
	}
	
	return &profile, nil
}

// Update referral profile statistics
func (as *AuthService) UpdateReferralStats(userID string) error {
	query := `
		UPDATE referral_profiles SET
			total_earnings = (
				SELECT COALESCE(SUM(c.amount), 0)
				FROM referral_links rl
				JOIN commissions c ON rl.id = c.link_id
				WHERE rl.referrer_id = $1 AND c.status IN ('approved', 'paid')
			),
			active_links = (
				SELECT COUNT(*)
				FROM referral_links
				WHERE referrer_id = $1 AND is_active = true
			),
			total_clicks = (
				SELECT COALESCE(SUM(clicks), 0)
				FROM referral_links
				WHERE referrer_id = $1
			),
			total_conversions = (
				SELECT COALESCE(SUM(conversions), 0)
				FROM referral_links
				WHERE referrer_id = $1
			),
			updated_at = NOW()
		WHERE user_id = $1`
	
	_, err := as.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to update referral stats: %w", err)
	}
	
	return nil
}

// Validate auth token (middleware function)
func (as *AuthService) ValidateToken(token string) (*AuthUser, error) {
	// For now, implement a simple validation
	// In production, this would verify JWT tokens or session tokens with scenario-authenticator
	
	if token == "" {
		return nil, fmt.Errorf("no token provided")
	}
	
	// Mock implementation - in real scenario, validate with authenticator service
	// This would make an HTTP request to scenario-authenticator to validate the token
	
	// For demo purposes, return a mock user
	user := &AuthUser{
		ID:       "demo-user-123",
		Username: "demouser",
		Email:    "demo@example.com",
		Role:     "user",
	}
	
	return user, nil
}

// Generate unique referral code
func generateReferralCode() string {
	// Generate a short, human-readable referral code
	return fmt.Sprintf("REF%s", uuid.New().String()[:8])
}

// Middleware for protecting routes
func (as *AuthService) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			// Try cookie
			if cookie, err := r.Cookie("auth_token"); err == nil {
				token = cookie.Value
			}
		}
		
		user, err := as.ValidateToken(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		
		// Add user to request context (if using context)
		// For simplicity, we'll add it as a header
		r.Header.Set("X-User-ID", user.ID)
		r.Header.Set("X-User-Email", user.Email)
		
		next.ServeHTTP(w, r)
	}
}

// HTTP handlers for referral profile management
func (as *AuthService) GetProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusBadRequest)
		return
	}
	
	profile, err := as.GetOrCreateReferralProfile(userID)
	if err != nil {
		log.Printf("Error getting referral profile: %v", err)
		http.Error(w, "Failed to get referral profile", http.StatusInternalServerError)
		return
	}
	
	// Update stats before returning
	if err := as.UpdateReferralStats(userID); err != nil {
		log.Printf("Error updating referral stats: %v", err)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (as *AuthService) GetUserStatsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusBadRequest)
		return
	}
	
	// Get user's referral links with stats
	query := `
		SELECT 
			rl.id, rl.tracking_code, rl.clicks, rl.conversions, 
			rl.total_commission, rl.created_at, rl.is_active,
			rp.scenario_name, rp.commission_rate
		FROM referral_links rl
		JOIN referral_programs rp ON rl.program_id = rp.id
		WHERE rl.referrer_id = $1
		ORDER BY rl.created_at DESC`
	
	rows, err := as.db.Query(query, userID)
	if err != nil {
		log.Printf("Error fetching user stats: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var links []map[string]interface{}
	var totalClicks, totalConversions int
	var totalEarnings float64
	
	for rows.Next() {
		var link struct {
			ID              string  `json:"id"`
			TrackingCode    string  `json:"tracking_code"`
			Clicks          int     `json:"clicks"`
			Conversions     int     `json:"conversions"`
			TotalCommission float64 `json:"total_commission"`
			CreatedAt       string  `json:"created_at"`
			IsActive        bool    `json:"is_active"`
			ScenarioName    string  `json:"scenario_name"`
			CommissionRate  float64 `json:"commission_rate"`
		}
		
		err := rows.Scan(
			&link.ID, &link.TrackingCode, &link.Clicks, &link.Conversions,
			&link.TotalCommission, &link.CreatedAt, &link.IsActive,
			&link.ScenarioName, &link.CommissionRate)
		
		if err != nil {
			log.Printf("Error scanning link: %v", err)
			continue
		}
		
		totalClicks += link.Clicks
		totalConversions += link.Conversions
		totalEarnings += link.TotalCommission
		
		links = append(links, map[string]interface{}{
			"id":               link.ID,
			"tracking_code":    link.TrackingCode,
			"clicks":           link.Clicks,
			"conversions":      link.Conversions,
			"total_commission": link.TotalCommission,
			"created_at":       link.CreatedAt,
			"is_active":        link.IsActive,
			"scenario_name":    link.ScenarioName,
			"commission_rate":  link.CommissionRate,
		})
	}
	
	response := map[string]interface{}{
		"links":              links,
		"totalClicks":        totalClicks,
		"totalConversions":   totalConversions,
		"totalEarnings":      totalEarnings,
		"conversionRate":     0.0,
	}
	
	if totalClicks > 0 {
		response["conversionRate"] = float64(totalConversions) / float64(totalClicks)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Setup auth routes
func SetupAuthRoutes(router *mux.Router, as *AuthService) {
	// Protected routes
	api := router.PathPrefix("/api/user").Subrouter()
	api.Use(as.AuthMiddleware)
	
	api.HandleFunc("/profile", as.GetProfileHandler).Methods("GET")
	api.HandleFunc("/{user_id}/stats", as.GetUserStatsHandler).Methods("GET")
}

// Database migration for user-related tables
const authSchemaMigration = `
-- Create users table (cache from scenario-authenticator)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'user',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create referral profiles table
CREATE TABLE IF NOT EXISTS referral_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    referral_code VARCHAR(32) UNIQUE NOT NULL,
    total_earnings DECIMAL(12,2) DEFAULT 0.00,
    active_links INTEGER DEFAULT 0,
    total_clicks INTEGER DEFAULT 0,
    total_conversions INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_referral_profiles_user_id ON referral_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_referral_profiles_code ON referral_profiles(referral_code);

-- Update referral_links to reference users table
ALTER TABLE referral_links 
ADD CONSTRAINT fk_referral_links_referrer 
FOREIGN KEY (referrer_id) REFERENCES users(id) ON DELETE CASCADE;
`