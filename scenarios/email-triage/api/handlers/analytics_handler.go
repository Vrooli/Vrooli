package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
	
	"email-triage/models"
)

// AnalyticsHandler handles analytics and dashboard endpoints
type AnalyticsHandler struct {
	db *sql.DB
}

// NewAnalyticsHandler creates a new AnalyticsHandler instance
func NewAnalyticsHandler(db *sql.DB) *AnalyticsHandler {
	return &AnalyticsHandler{
		db: db,
	}
}

// GetDashboard handles GET /api/v1/analytics/dashboard
func (ah *AnalyticsHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	// Build dashboard statistics
	stats, err := ah.buildDashboardStats(userID)
	if err != nil {
		http.Error(w, `{"error":"failed to generate dashboard stats"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetUsageStats handles GET /api/v1/analytics/usage
func (ah *AnalyticsHandler) GetUsageStats(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	usage, err := ah.buildUsageStats(userID)
	if err != nil {
		http.Error(w, `{"error":"failed to generate usage stats"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usage)
}

// GetRulePerformance handles GET /api/v1/analytics/rules-performance
func (ah *AnalyticsHandler) GetRulePerformance(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, `{"error":"user authentication required"}`, http.StatusUnauthorized)
		return
	}

	performance, err := ah.buildRulePerformanceStats(userID)
	if err != nil {
		http.Error(w, `{"error":"failed to generate rule performance stats"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"rules": performance,
		"count": len(performance),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper methods for building analytics data

func (ah *AnalyticsHandler) buildDashboardStats(userID string) (*models.DashboardStats, error) {
	stats := &models.DashboardStats{}

	// Count processed emails
	emailCount, err := ah.getEmailCount(userID)
	if err != nil {
		return nil, err
	}
	stats.EmailsProcessed = emailCount

	// Count active rules
	ruleCount, err := ah.getRuleCount(userID)
	if err != nil {
		return nil, err
	}
	stats.RulesActive = ruleCount

	// Count automated actions (placeholder)
	stats.ActionsAutomated = emailCount / 2 // Rough estimate

	// Average processing time (placeholder)
	stats.AverageProcessTime = 150.5 // milliseconds

	// Get top performing rules
	topRules, err := ah.buildRulePerformanceStats(userID)
	if err != nil {
		return nil, err
	}
	// Take top 5
	if len(topRules) > 5 {
		stats.TopRules = topRules[:5]
	} else {
		stats.TopRules = topRules
	}

	// Get recent activity
	activity, err := ah.buildRecentActivity(userID)
	if err != nil {
		return nil, err
	}
	stats.RecentActivity = activity

	// Get usage statistics
	usage, err := ah.buildUsageStats(userID)
	if err != nil {
		return nil, err
	}
	stats.UsageStats = *usage

	return stats, nil
}

func (ah *AnalyticsHandler) getEmailCount(userID string) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM processed_emails pe
		JOIN email_accounts ea ON pe.account_id = ea.id
		WHERE ea.user_id = $1`
		
	var count int
	err := ah.db.QueryRow(query, userID).Scan(&count)
	return count, err
}

func (ah *AnalyticsHandler) getRuleCount(userID string) (int, error) {
	query := `SELECT COUNT(*) FROM triage_rules WHERE user_id = $1 AND enabled = true`
	
	var count int
	err := ah.db.QueryRow(query, userID).Scan(&count)
	return count, err
}

func (ah *AnalyticsHandler) buildUsageStats(userID string) (*models.UsageStatistics, error) {
	usage := &models.UsageStatistics{
		PlanType: "free", // Default plan
	}

	// Get user's plan type
	planQuery := `SELECT COALESCE(plan_type, 'free') FROM users WHERE id = $1`
	ah.db.QueryRow(planQuery, userID).Scan(&usage.PlanType)

	// Set limits based on plan type
	switch usage.PlanType {
	case "pro":
		usage.EmailAccountsLimit = 5
		usage.MonthlyEmailsLimit = 10000
		usage.RulesLimit = 50
	case "business":
		usage.EmailAccountsLimit = 20
		usage.MonthlyEmailsLimit = 100000
		usage.RulesLimit = 200
	default: // free
		usage.EmailAccountsLimit = 1
		usage.MonthlyEmailsLimit = 1000
		usage.RulesLimit = 5
	}

	// Count current usage
	accountQuery := `SELECT COUNT(*) FROM email_accounts WHERE user_id = $1`
	ah.db.QueryRow(accountQuery, userID).Scan(&usage.EmailAccountsUsed)

	ruleQuery := `SELECT COUNT(*) FROM triage_rules WHERE user_id = $1`
	ah.db.QueryRow(ruleQuery, userID).Scan(&usage.RulesUsed)

	// Count emails processed this month
	monthStart := time.Now().AddDate(0, 0, -time.Now().Day()+1)
	emailQuery := `
		SELECT COUNT(*) 
		FROM processed_emails pe
		JOIN email_accounts ea ON pe.account_id = ea.id
		WHERE ea.user_id = $1 AND pe.processed_at >= $2`
	ah.db.QueryRow(emailQuery, userID, monthStart).Scan(&usage.MonthlyEmailsProcessed)

	return usage, nil
}

func (ah *AnalyticsHandler) buildRulePerformanceStats(userID string) ([]models.RulePerformance, error) {
	query := `
		SELECT id, name, match_count, created_at
		FROM triage_rules 
		WHERE user_id = $1 
		ORDER BY match_count DESC, created_at DESC
		LIMIT 10`

	rows, err := ah.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var performance []models.RulePerformance

	for rows.Next() {
		var ruleID, ruleName string
		var matchCount int
		var createdAt time.Time

		err := rows.Scan(&ruleID, &ruleName, &matchCount, &createdAt)
		if err != nil {
			continue
		}

		perf := models.RulePerformance{
			RuleID:      ruleID,
			RuleName:    ruleName,
			MatchCount:  matchCount,
			SuccessRate: 0.95, // Placeholder - would calculate from actual data
			AverageTime: 45.2, // Placeholder - would calculate from actual data
		}

		performance = append(performance, perf)
	}

	return performance, nil
}

func (ah *AnalyticsHandler) buildRecentActivity(userID string) ([]models.ActivityLog, error) {
	// Build recent activity from various sources
	var activity []models.ActivityLog

	// Recent rule creations
	ruleQuery := `
		SELECT name, created_at 
		FROM triage_rules 
		WHERE user_id = $1 
		ORDER BY created_at DESC 
		LIMIT 5`

	rows, err := ah.db.Query(ruleQuery, userID)
	if err != nil {
		return activity, nil // Return empty rather than error
	}
	defer rows.Close()

	for rows.Next() {
		var ruleName string
		var createdAt time.Time

		if err := rows.Scan(&ruleName, &createdAt); err != nil {
			continue
		}

		activity = append(activity, models.ActivityLog{
			Timestamp:   createdAt,
			Type:        "rule_created",
			Description: "Created new triage rule: " + ruleName,
			Details: map[string]interface{}{
				"rule_name": ruleName,
				"type":      "triage_rule",
			},
		})
	}

	// Recent email account additions
	accountQuery := `
		SELECT email_address, created_at 
		FROM email_accounts 
		WHERE user_id = $1 
		ORDER BY created_at DESC 
		LIMIT 3`

	rows2, err := ah.db.Query(accountQuery, userID)
	if err != nil {
		return activity, nil // Return what we have
	}
	defer rows2.Close()

	for rows2.Next() {
		var emailAddress string
		var createdAt time.Time

		if err := rows2.Scan(&emailAddress, &createdAt); err != nil {
			continue
		}

		activity = append(activity, models.ActivityLog{
			Timestamp:   createdAt,
			Type:        "account_added",
			Description: "Connected email account: " + emailAddress,
			Details: map[string]interface{}{
				"email_address": emailAddress,
				"type":          "email_account",
			},
		})
	}

	// Sort by timestamp (most recent first)
	// Simple bubble sort for small arrays
	for i := 0; i < len(activity); i++ {
		for j := i + 1; j < len(activity); j++ {
			if activity[i].Timestamp.Before(activity[j].Timestamp) {
				activity[i], activity[j] = activity[j], activity[i]
			}
		}
	}

	// Return most recent 10 activities
	if len(activity) > 10 {
		activity = activity[:10]
	}

	return activity, nil
}