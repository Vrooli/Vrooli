package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// CreateTenantHandler creates a new tenant
func (s *Server) CreateTenantHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
		Plan        string `json:"plan"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" || req.Slug == "" {
		http.Error(w, `{"error":"Name and slug are required"}`, http.StatusBadRequest)
		return
	}

	// Validate slug format
	if !isValidSlug(req.Slug) {
		http.Error(w, `{"error":"Invalid slug format. Use lowercase letters, numbers, and hyphens only"}`, http.StatusBadRequest)
		return
	}

	// Set default plan if not specified
	if req.Plan == "" {
		req.Plan = "starter"
	}

	// Set plan limits based on plan type
	maxChatbots := 3
	maxConversations := 1000
	switch req.Plan {
	case "professional":
		maxChatbots = 10
		maxConversations = 10000
	case "enterprise":
		maxChatbots = 100
		maxConversations = 100000
	}

	// Create tenant in database
	query := `
		INSERT INTO tenants (name, slug, description, plan, max_chatbots, max_conversations_per_month)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, api_key, created_at
	`

	var tenant Tenant
	tenant.Name = req.Name
	tenant.Slug = req.Slug
	tenant.Description = req.Description
	tenant.Plan = req.Plan
	tenant.MaxChatbots = maxChatbots
	tenant.MaxConversationsPerMonth = maxConversations

	err := s.db.QueryRow(query, req.Name, req.Slug, req.Description, req.Plan, maxChatbots, maxConversations).
		Scan(&tenant.ID, &tenant.APIKey, &tenant.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			http.Error(w, `{"error":"Tenant with this slug already exists"}`, http.StatusConflict)
			return
		}
		s.logger.Printf("Error creating tenant: %v", err)
		http.Error(w, `{"error":"Failed to create tenant"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant": tenant,
		"message": "Tenant created successfully",
	})
}

// GetTenantHandler retrieves tenant information
func (s *Server) GetTenantHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantID := vars["id"]

	query := `
		SELECT id, name, slug, description, plan, max_chatbots, max_conversations_per_month, 
			   is_active, created_at, updated_at
		FROM tenants
		WHERE id = $1 OR slug = $1
	`

	var tenant Tenant
	err := s.db.QueryRow(query, tenantID).Scan(
		&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Description,
		&tenant.Plan, &tenant.MaxChatbots, &tenant.MaxConversationsPerMonth,
		&tenant.IsActive, &tenant.CreatedAt, &tenant.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"Tenant not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Printf("Error fetching tenant: %v", err)
		http.Error(w, `{"error":"Failed to fetch tenant"}`, http.StatusInternalServerError)
		return
	}

	// Get usage statistics
	usageQuery := `
		SELECT 
			COUNT(DISTINCT c.id) as chatbot_count,
			COUNT(DISTINCT conv.id) FILTER (WHERE conv.started_at >= date_trunc('month', CURRENT_DATE)) as monthly_conversations
		FROM chatbots c
		LEFT JOIN conversations conv ON c.id = conv.chatbot_id
		WHERE c.tenant_id = $1
	`

	var chatbotCount, monthlyConversations int
	s.db.QueryRow(usageQuery, tenant.ID).Scan(&chatbotCount, &monthlyConversations)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant": tenant,
		"usage": map[string]interface{}{
			"chatbot_count":          chatbotCount,
			"monthly_conversations":  monthlyConversations,
		},
	})
}

// CreateABTestHandler creates a new A/B test for a chatbot
func (s *Server) CreateABTestHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatbotID := vars["chatbot_id"]

	var req struct {
		Name         string                 `json:"name"`
		Description  string                 `json:"description"`
		VariantA     map[string]interface{} `json:"variant_a"`
		VariantB     map[string]interface{} `json:"variant_b"`
		TrafficSplit float64                `json:"traffic_split"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" || req.VariantA == nil || req.VariantB == nil {
		http.Error(w, `{"error":"Name, variant_a, and variant_b are required"}`, http.StatusBadRequest)
		return
	}

	// Default traffic split to 50/50
	if req.TrafficSplit == 0 {
		req.TrafficSplit = 0.5
	}

	// Get tenant ID from chatbot
	var tenantID string
	err := s.db.QueryRow("SELECT tenant_id FROM chatbots WHERE id = $1", chatbotID).Scan(&tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error":"Chatbot not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"Failed to verify chatbot"}`, http.StatusInternalServerError)
		}
		return
	}

	// Create A/B test
	variantAJSON, _ := json.Marshal(req.VariantA)
	variantBJSON, _ := json.Marshal(req.VariantB)

	query := `
		INSERT INTO ab_tests (chatbot_id, tenant_id, name, description, variant_a, variant_b, traffic_split, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'draft')
		RETURNING id, created_at
	`

	var testID string
	var createdAt string
	err = s.db.QueryRow(query, chatbotID, tenantID, req.Name, req.Description, variantAJSON, variantBJSON, req.TrafficSplit).
		Scan(&testID, &createdAt)
	
	if err != nil {
		s.logger.Printf("Error creating A/B test: %v", err)
		http.Error(w, `{"error":"Failed to create A/B test"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ab_test": map[string]interface{}{
			"id":            testID,
			"chatbot_id":    chatbotID,
			"tenant_id":     tenantID,
			"name":          req.Name,
			"description":   req.Description,
			"variant_a":     req.VariantA,
			"variant_b":     req.VariantB,
			"traffic_split": req.TrafficSplit,
			"status":        "draft",
			"created_at":    createdAt,
		},
		"message": "A/B test created successfully",
	})
}

// StartABTestHandler starts an A/B test
func (s *Server) StartABTestHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["test_id"]

	query := `
		UPDATE ab_tests 
		SET status = 'running', started_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'draft'
		RETURNING chatbot_id
	`

	var chatbotID string
	err := s.db.QueryRow(query, testID).Scan(&chatbotID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error":"A/B test not found or already running"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"Failed to start A/B test"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "A/B test started successfully",
		"test_id": testID,
	})
}

// GetABTestResultsHandler retrieves results for an A/B test
func (s *Server) GetABTestResultsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["test_id"]

	// Get test details
	testQuery := `
		SELECT name, variant_a, variant_b, traffic_split, status, started_at, ended_at
		FROM ab_tests
		WHERE id = $1
	`

	var test ABTest
	var variantAJSON, variantBJSON []byte
	err := s.db.QueryRow(testQuery, testID).Scan(
		&test.Name, &variantAJSON, &variantBJSON, 
		&test.TrafficSplit, &test.Status, &test.StartedAt, &test.EndedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error":"A/B test not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"Failed to fetch A/B test"}`, http.StatusInternalServerError)
		}
		return
	}

	json.Unmarshal(variantAJSON, &test.VariantA)
	json.Unmarshal(variantBJSON, &test.VariantB)

	// Get results
	resultsQuery := `
		SELECT 
			variant,
			COUNT(*) as total_conversations,
			COUNT(*) FILTER (WHERE conversion = true) as conversions,
			AVG(engagement_score) as avg_engagement
		FROM ab_test_results
		WHERE ab_test_id = $1
		GROUP BY variant
	`

	rows, err := s.db.Query(resultsQuery, testID)
	if err != nil {
		s.logger.Printf("Error fetching A/B test results: %v", err)
		http.Error(w, `{"error":"Failed to fetch results"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	results := make(map[string]interface{})
	for rows.Next() {
		var variant string
		var total, conversions int
		var avgEngagement sql.NullFloat64

		rows.Scan(&variant, &total, &conversions, &avgEngagement)
		
		conversionRate := float64(0)
		if total > 0 {
			conversionRate = float64(conversions) / float64(total) * 100
		}

		results[fmt.Sprintf("variant_%s", variant)] = map[string]interface{}{
			"total_conversations": total,
			"conversions":        conversions,
			"conversion_rate":    conversionRate,
			"avg_engagement":     avgEngagement.Float64,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"test":    test,
		"results": results,
	})
}

// CreateCRMIntegrationHandler creates a new CRM integration
func (s *Server) CreateCRMIntegrationHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID  string                 `json:"tenant_id"`
		ChatbotID *string                `json:"chatbot_id"`
		Type      string                 `json:"type"`
		Config    map[string]interface{} `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.TenantID == "" || req.Type == "" || req.Config == nil {
		http.Error(w, `{"error":"tenant_id, type, and config are required"}`, http.StatusBadRequest)
		return
	}

	// Validate CRM type
	validTypes := []string{"salesforce", "hubspot", "pipedrive", "webhook"}
	isValid := false
	for _, t := range validTypes {
		if req.Type == t {
			isValid = true
			break
		}
	}
	if !isValid {
		http.Error(w, `{"error":"Invalid CRM type. Must be one of: salesforce, hubspot, pipedrive, webhook"}`, http.StatusBadRequest)
		return
	}

	configJSON, _ := json.Marshal(req.Config)

	query := `
		INSERT INTO crm_integrations (tenant_id, chatbot_id, type, config)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	var integrationID, createdAt string
	err := s.db.QueryRow(query, req.TenantID, req.ChatbotID, req.Type, configJSON).
		Scan(&integrationID, &createdAt)
	
	if err != nil {
		s.logger.Printf("Error creating CRM integration: %v", err)
		http.Error(w, `{"error":"Failed to create CRM integration"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"integration": map[string]interface{}{
			"id":         integrationID,
			"tenant_id":  req.TenantID,
			"chatbot_id": req.ChatbotID,
			"type":       req.Type,
			"config":     req.Config,
			"created_at": createdAt,
		},
		"message": "CRM integration created successfully",
	})
}

// SyncCRMLeadHandler manually triggers CRM sync for a conversation
func (s *Server) SyncCRMLeadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["conversation_id"]

	// Get conversation and chatbot details
	query := `
		SELECT c.chatbot_id, c.lead_data, ch.tenant_id
		FROM conversations c
		JOIN chatbots ch ON c.chatbot_id = ch.id
		WHERE c.id = $1 AND c.lead_captured = true
	`

	var chatbotID, tenantID string
	var leadDataJSON []byte
	err := s.db.QueryRow(query, conversationID).Scan(&chatbotID, &leadDataJSON, &tenantID)
	
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error":"Conversation not found or no lead data"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"Failed to fetch conversation"}`, http.StatusInternalServerError)
		}
		return
	}

	var leadData map[string]interface{}
	json.Unmarshal(leadDataJSON, &leadData)

	// Get active CRM integration for this tenant/chatbot
	integrationQuery := `
		SELECT id, type, config
		FROM crm_integrations
		WHERE tenant_id = $1 AND (chatbot_id = $2 OR chatbot_id IS NULL) 
		AND sync_enabled = true
		ORDER BY chatbot_id DESC NULLS LAST
		LIMIT 1
	`

	var integrationID, integrationType string
	var configJSON []byte
	err = s.db.QueryRow(integrationQuery, tenantID, chatbotID).Scan(&integrationID, &integrationType, &configJSON)
	
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error":"No active CRM integration found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"Failed to fetch CRM integration"}`, http.StatusInternalServerError)
		}
		return
	}

	var config map[string]interface{}
	json.Unmarshal(configJSON, &config)

	// Perform the sync (simplified for demo - in production this would call actual CRM APIs)
	syncResult := performCRMSync(integrationType, config, leadData)

	// Log the sync attempt
	logQuery := `
		INSERT INTO crm_sync_log (integration_id, conversation_id, action, status, request_data, response_data)
		VALUES ($1, $2, 'create_lead', $3, $4, $5)
	`

	requestData, _ := json.Marshal(leadData)
	responseData, _ := json.Marshal(syncResult)
	status := "success"
	if !syncResult["success"].(bool) {
		status = "failed"
	}

	s.db.Exec(logQuery, integrationID, conversationID, status, requestData, responseData)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(syncResult)
}

// Helper function to validate slug format
func isValidSlug(slug string) bool {
	for _, char := range slug {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-') {
			return false
		}
	}
	return len(slug) > 0
}

// Simplified CRM sync function (in production, this would integrate with actual CRM APIs)
func performCRMSync(crmType string, config map[string]interface{}, leadData map[string]interface{}) map[string]interface{} {
	// This is a simplified implementation
	// In production, this would make actual API calls to the respective CRM systems
	
	switch crmType {
	case "webhook":
		// Send to webhook endpoint
		webhookURL, _ := config["webhook_url"].(string)
		if webhookURL != "" {
			// Would make HTTP POST to webhook URL with lead data
			return map[string]interface{}{
				"success": true,
				"message": "Lead data sent to webhook",
				"webhook_url": webhookURL,
			}
		}
	case "salesforce":
		// Would use Salesforce API
		return map[string]interface{}{
			"success": true,
			"message": "Lead created in Salesforce",
			"lead_id": "SF-" + generateID(),
		}
	case "hubspot":
		// Would use HubSpot API
		return map[string]interface{}{
			"success": true,
			"message": "Contact created in HubSpot",
			"contact_id": "HS-" + generateID(),
		}
	case "pipedrive":
		// Would use Pipedrive API
		return map[string]interface{}{
			"success": true,
			"message": "Person created in Pipedrive",
			"person_id": "PD-" + generateID(),
		}
	}

	return map[string]interface{}{
		"success": false,
		"error": "CRM sync not implemented for type: " + crmType,
	}
}

// Simple ID generator for demo purposes
func generateID() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}