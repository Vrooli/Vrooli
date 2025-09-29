package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
	
	"email-triage/models"
)

// RealtimeProcessor handles real-time email processing and sync
type RealtimeProcessor struct {
	db                   *sql.DB
	emailService         *EmailService
	ruleService          *RuleService
	searchService        *SearchService
	prioritizationService *PrioritizationService
	triageActionService  *TriageActionService
	
	syncInterval         time.Duration
	stopChan            chan struct{}
	wg                  sync.WaitGroup
	isRunning           bool
	mu                  sync.Mutex
}

// NewRealtimeProcessor creates a new RealtimeProcessor instance
func NewRealtimeProcessor(
	db *sql.DB,
	emailService *EmailService,
	ruleService *RuleService,
	searchService *SearchService,
) *RealtimeProcessor {
	return &RealtimeProcessor{
		db:                   db,
		emailService:         emailService,
		ruleService:          ruleService,
		searchService:        searchService,
		prioritizationService: NewPrioritizationService(),
		triageActionService:  NewTriageActionService(db, emailService),
		syncInterval:         5 * time.Minute, // Default sync every 5 minutes
		stopChan:            make(chan struct{}),
	}
}

// Start begins the real-time processing loop
func (rp *RealtimeProcessor) Start() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	
	if rp.isRunning {
		return fmt.Errorf("processor already running")
	}
	
	rp.isRunning = true
	rp.wg.Add(1)
	
	go rp.processingLoop()
	
	log.Println("🚀 Real-time email processor started")
	return nil
}

// Stop gracefully stops the real-time processor
func (rp *RealtimeProcessor) Stop() {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	
	if !rp.isRunning {
		return
	}
	
	close(rp.stopChan)
	rp.wg.Wait()
	rp.isRunning = false
	
	log.Println("🛑 Real-time email processor stopped")
}

// processingLoop is the main processing loop
func (rp *RealtimeProcessor) processingLoop() {
	defer rp.wg.Done()
	
	// Initial sync on startup
	rp.syncAllAccounts()
	
	ticker := time.NewTicker(rp.syncInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			rp.syncAllAccounts()
		case <-rp.stopChan:
			return
		}
	}
}

// syncAllAccounts syncs emails for all active accounts
func (rp *RealtimeProcessor) syncAllAccounts() {
	log.Println("📧 Starting email sync for all accounts...")
	
	// Get all active accounts
	accounts, err := rp.getActiveAccounts()
	if err != nil {
		log.Printf("Error getting active accounts: %v", err)
		return
	}
	
	// Process each account
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Limit concurrent syncs to 5
	
	for _, account := range accounts {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore
		
		go func(acc *models.EmailAccount) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore
			
			rp.syncAccount(acc)
		}(account)
	}
	
	wg.Wait()
	log.Printf("✅ Email sync completed for %d accounts", len(accounts))
}

// syncAccount syncs emails for a single account
func (rp *RealtimeProcessor) syncAccount(account *models.EmailAccount) {
	log.Printf("📬 Syncing account: %s", account.EmailAddress)
	
	// Determine sync starting point
	syncSince := time.Now().Add(-24 * time.Hour) // Default: sync last 24 hours
	if account.LastSync != nil {
		syncSince = *account.LastSync
	}
	
	// Fetch new emails
	newEmails, err := rp.emailService.FetchNewEmails(account, syncSince)
	if err != nil {
		log.Printf("Error fetching emails for %s: %v", account.EmailAddress, err)
		return
	}
	
	if len(newEmails) == 0 {
		log.Printf("No new emails for %s", account.EmailAddress)
		rp.updateLastSync(account.ID)
		return
	}
	
	log.Printf("📨 Processing %d new emails for %s", len(newEmails), account.EmailAddress)
	
	// Process each email
	for _, email := range newEmails {
		rp.processEmail(account, email)
	}
	
	// Update last sync time
	rp.updateLastSync(account.ID)
}

// processEmail processes a single email through the triage pipeline
func (rp *RealtimeProcessor) processEmail(account *models.EmailAccount, email *models.ProcessedEmail) {
	// Step 1: Calculate priority score
	email.PriorityScore = rp.prioritizationService.CalculatePriority(email)
	
	// Step 2: Save email to database
	emailID, err := rp.saveEmail(email)
	if err != nil {
		log.Printf("Error saving email: %v", err)
		return
	}
	email.ID = emailID
	
	// Step 3: Generate and store vector embedding for semantic search
	rp.generateAndStoreEmbedding(email)
	
	// Step 4: Apply triage rules
	rp.applyTriageRules(account.UserID, email)
	
	// Step 5: Send notifications for high-priority emails
	if email.PriorityScore > 0.8 {
		rp.sendHighPriorityNotification(account.UserID, email)
	}
}

// saveEmail saves a processed email to the database
func (rp *RealtimeProcessor) saveEmail(email *models.ProcessedEmail) (string, error) {
	query := `
		INSERT INTO processed_emails (
			id, account_id, message_id, subject, sender_email,
			recipient_emails, body_preview, full_body, priority_score,
			processed_at, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		) ON CONFLICT (message_id) DO UPDATE SET
			priority_score = EXCLUDED.priority_score,
			processed_at = EXCLUDED.processed_at
		RETURNING id`
	
	// Generate ID if not set
	if email.ID == "" {
		email.ID = generateUUID()
	}
	
	// Initialize empty metadata if nil
	if email.Metadata == nil {
		email.Metadata = json.RawMessage(`{}`)
	}
	
	var emailID string
	err := rp.db.QueryRow(query,
		email.ID,
		email.AccountID,
		email.MessageID,
		email.Subject,
		email.SenderEmail,
		email.RecipientEmails,
		email.BodyPreview,
		email.FullBody,
		email.PriorityScore,
		email.ProcessedAt,
		email.Metadata,
	).Scan(&emailID)
	
	if err != nil {
		return "", fmt.Errorf("failed to save email: %w", err)
	}
	
	return emailID, nil
}

// generateAndStoreEmbedding creates vector embedding for semantic search
func (rp *RealtimeProcessor) generateAndStoreEmbedding(email *models.ProcessedEmail) {
	// Create a text representation for embedding
	textForEmbedding := fmt.Sprintf("%s %s %s", email.Subject, email.SenderEmail, email.BodyPreview)
	
	// TODO: Use actual embedding model (e.g., sentence-transformers)
	// For now, create a dummy embedding
	embedding := make([]float32, 384) // all-MiniLM-L6-v2 dimensions
	for i := range embedding {
		// Generate pseudo-random values based on content
		embedding[i] = float32(len(textForEmbedding)%100) / 100.0
	}
	
	// Store in Qdrant
	vectorID := generateUUID()
	payload := map[string]interface{}{
		"email_id":       email.ID,
		"subject":        email.Subject,
		"sender":         email.SenderEmail,
		"timestamp":      email.ProcessedAt.Unix(),
		"priority_score": email.PriorityScore,
	}
	
	err := rp.searchService.StoreEmailVector(vectorID, embedding, payload)
	if err != nil {
		log.Printf("Error storing email vector: %v", err)
		return
	}
	
	// Update email with vector ID
	query := `UPDATE processed_emails SET vector_id = $2 WHERE id = $1`
	_, err = rp.db.Exec(query, email.ID, vectorID)
	if err != nil {
		log.Printf("Error updating email with vector ID: %v", err)
	}
}

// applyTriageRules applies user's triage rules to an email
func (rp *RealtimeProcessor) applyTriageRules(userID string, email *models.ProcessedEmail) {
	// Get user's active rules
	rules, err := rp.getRulesForUser(userID)
	if err != nil {
		log.Printf("Error getting rules for user %s: %v", userID, err)
		return
	}
	
	// Check each rule
	for _, rule := range rules {
		if rp.ruleMatches(rule, email) {
			log.Printf("📋 Rule '%s' matched for email '%s'", rule.Name, email.Subject)
			
			// Parse and execute actions
			var actions []models.RuleAction
			if err := json.Unmarshal(rule.Actions, &actions); err != nil {
				log.Printf("Error parsing rule actions: %v", err)
				continue
			}
			
			results, err := rp.triageActionService.ExecuteActions(userID, email.ID, actions)
			if err != nil {
				log.Printf("Error executing actions: %v", err)
				continue
			}
			
			// Update rule match count
			rp.incrementRuleMatchCount(rule.ID)
			
			// Log results
			for _, result := range results {
				if result.Status == "success" {
					log.Printf("✅ Action %s succeeded: %s", result.ActionType, result.Message)
				} else {
					log.Printf("❌ Action %s failed: %s", result.ActionType, result.Message)
				}
			}
		}
	}
}

// ruleMatches checks if a rule matches an email
func (rp *RealtimeProcessor) ruleMatches(rule *models.TriageRule, email *models.ProcessedEmail) bool {
	// Parse conditions
	var conditions []models.RuleCondition
	if err := json.Unmarshal(rule.Conditions, &conditions); err != nil {
		log.Printf("Error parsing rule conditions: %v", err)
		return false
	}
	
	// Check all conditions (AND logic)
	for _, condition := range conditions {
		if !rp.conditionMatches(condition, email) {
			return false
		}
	}
	
	return true
}

// conditionMatches checks if a single condition matches an email
func (rp *RealtimeProcessor) conditionMatches(condition models.RuleCondition, email *models.ProcessedEmail) bool {
	// Get field value from email
	fieldValue := rp.getEmailFieldValue(condition.Field, email)
	conditionValue := fmt.Sprintf("%v", condition.Value)
	
	switch condition.Operator {
	case "contains":
		return contains(fieldValue, conditionValue)
	case "equals":
		return fieldValue == conditionValue
	case "starts_with":
		return startsWith(fieldValue, conditionValue)
	case "ends_with":
		return endsWith(fieldValue, conditionValue)
	case "matches": // regex match
		// TODO: Implement regex matching
		return false
	default:
		return false
	}
}

// getEmailFieldValue extracts a field value from an email
func (rp *RealtimeProcessor) getEmailFieldValue(field string, email *models.ProcessedEmail) string {
	switch field {
	case "sender":
		return email.SenderEmail
	case "subject":
		return email.Subject
	case "body":
		return email.FullBody
	case "preview":
		return email.BodyPreview
	default:
		return ""
	}
}

// sendHighPriorityNotification sends notification for high-priority emails
func (rp *RealtimeProcessor) sendHighPriorityNotification(userID string, email *models.ProcessedEmail) {
	// TODO: Integrate with notification-hub when available
	log.Printf("🔔 High priority email from %s: %s (score: %.2f)",
		email.SenderEmail, email.Subject, email.PriorityScore)
}

// Helper functions

func (rp *RealtimeProcessor) getActiveAccounts() ([]*models.EmailAccount, error) {
	query := `
		SELECT id, user_id, email_address, imap_settings, smtp_settings, last_sync
		FROM email_accounts
		WHERE sync_enabled = true`
	
	rows, err := rp.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var accounts []*models.EmailAccount
	for rows.Next() {
		var account models.EmailAccount
		err := rows.Scan(
			&account.ID,
			&account.UserID,
			&account.EmailAddress,
			&account.IMAPSettings,
			&account.SMTPSettings,
			&account.LastSync,
		)
		if err != nil {
			log.Printf("Error scanning account: %v", err)
			continue
		}
		accounts = append(accounts, &account)
	}
	
	return accounts, nil
}

func (rp *RealtimeProcessor) updateLastSync(accountID string) {
	query := `UPDATE email_accounts SET last_sync = $2 WHERE id = $1`
	_, err := rp.db.Exec(query, accountID, time.Now())
	if err != nil {
		log.Printf("Error updating last sync for account %s: %v", accountID, err)
	}
}

func (rp *RealtimeProcessor) getRulesForUser(userID string) ([]*models.TriageRule, error) {
	query := `
		SELECT id, user_id, name, description, conditions, actions, priority, enabled, created_by_ai, ai_confidence
		FROM triage_rules
		WHERE user_id = $1 AND enabled = true
		ORDER BY priority DESC`
	
	rows, err := rp.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var rules []*models.TriageRule
	for rows.Next() {
		var rule models.TriageRule
		err := rows.Scan(
			&rule.ID,
			&rule.UserID,
			&rule.Name,
			&rule.Description,
			&rule.Conditions,
			&rule.Actions,
			&rule.Priority,
			&rule.Enabled,
			&rule.CreatedByAI,
			&rule.AIConfidence,
		)
		if err != nil {
			log.Printf("Error scanning rule: %v", err)
			continue
		}
		rules = append(rules, &rule)
	}
	
	return rules, nil
}

func (rp *RealtimeProcessor) incrementRuleMatchCount(ruleID string) {
	query := `UPDATE triage_rules SET match_count = match_count + 1 WHERE id = $1`
	_, err := rp.db.Exec(query, ruleID)
	if err != nil {
		log.Printf("Error incrementing rule match count: %v", err)
	}
}

// String helper functions
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		len(s) >= len(substr) && 
		findSubstring(s, substr) != -1
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}