package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
	
	"email-triage/models"
)

// TriageActionService handles execution of email triage actions
type TriageActionService struct {
	db           *sql.DB
	emailService *EmailService
}

// NewTriageActionService creates a new TriageActionService instance
func NewTriageActionService(db *sql.DB, emailService *EmailService) *TriageActionService {
	return &TriageActionService{
		db:           db,
		emailService: emailService,
	}
}

// ExecuteActions executes a list of triage actions on an email
func (tas *TriageActionService) ExecuteActions(userID string, emailID string, actions []models.RuleAction) ([]ActionResult, error) {
	// First verify the user owns this email
	var accountID string
	query := `
		SELECT pe.account_id
		FROM processed_emails pe
		JOIN email_accounts ea ON pe.account_id = ea.id
		WHERE pe.id = $1 AND ea.user_id = $2`
	
	err := tas.db.QueryRow(query, emailID, userID).Scan(&accountID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("email not found or access denied")
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	
	// Get email details
	email, err := tas.getEmailDetails(emailID)
	if err != nil {
		return nil, err
	}
	
	// Get account details for actions that need it
	account, err := tas.getAccountDetails(accountID)
	if err != nil {
		return nil, err
	}
	
	// Execute each action
	results := make([]ActionResult, 0, len(actions))
	for _, action := range actions {
		result := tas.executeAction(email, account, action)
		results = append(results, result)
		
		// Log the action
		tas.logAction(emailID, action, result)
	}
	
	// Update the email's actions_taken field
	tas.updateEmailActionsTaken(emailID, results)
	
	return results, nil
}

// ActionResult represents the result of executing a triage action
type ActionResult struct {
	ActionType string                 `json:"action_type"`
	Status     string                 `json:"status"` // success, failed, partial
	Message    string                 `json:"message,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// executeAction executes a single triage action
func (tas *TriageActionService) executeAction(email *models.ProcessedEmail, account *models.EmailAccount, action models.RuleAction) ActionResult {
	result := ActionResult{
		ActionType: action.Type,
		Timestamp:  time.Now(),
		Details:    make(map[string]interface{}),
	}
	
	switch action.Type {
	case "forward":
		return tas.executeForwardAction(email, account, action.Parameters)
	
	case "archive":
		return tas.executeArchiveAction(email, action.Parameters)
	
	case "mark_important":
		return tas.executeMarkImportantAction(email, action.Parameters)
	
	case "auto_reply":
		return tas.executeAutoReplyAction(email, account, action.Parameters)
	
	case "label":
		return tas.executeLabelAction(email, action.Parameters)
	
	case "move_to_folder":
		return tas.executeMoveToFolderAction(email, action.Parameters)
	
	case "delete":
		return tas.executeDeleteAction(email, action.Parameters)
	
	case "mark_read":
		return tas.executeMarkReadAction(email, action.Parameters)
	
	default:
		result.Status = "failed"
		result.Message = fmt.Sprintf("unknown action type: %s", action.Type)
		return result
	}
}

// executeForwardAction forwards an email to specified recipients
func (tas *TriageActionService) executeForwardAction(email *models.ProcessedEmail, account *models.EmailAccount, params map[string]interface{}) ActionResult {
	result := ActionResult{
		ActionType: "forward",
		Timestamp:  time.Now(),
		Details:    make(map[string]interface{}),
	}
	
	// Extract forward recipients
	recipients, ok := params["recipients"].([]interface{})
	if !ok {
		// Try string slice
		if recipientList, ok := params["recipients"].([]string); ok {
			recipientsInterface := make([]interface{}, len(recipientList))
			for i, r := range recipientList {
				recipientsInterface[i] = r
			}
			recipients = recipientsInterface
		} else {
			result.Status = "failed"
			result.Message = "recipients parameter is required for forward action"
			return result
		}
	}
	
	// Convert to string slice
	forwardTo := make([]string, 0, len(recipients))
	for _, r := range recipients {
		if email, ok := r.(string); ok {
			forwardTo = append(forwardTo, email)
		}
	}
	
	if len(forwardTo) == 0 {
		result.Status = "failed"
		result.Message = "no valid recipients specified"
		return result
	}
	
	// Execute forward
	err := tas.emailService.ForwardEmail(account, email, forwardTo)
	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("forward failed: %v", err)
	} else {
		result.Status = "success"
		result.Message = fmt.Sprintf("forwarded to %d recipients", len(forwardTo))
		result.Details["recipients"] = forwardTo
	}
	
	return result
}

// executeArchiveAction archives an email
func (tas *TriageActionService) executeArchiveAction(email *models.ProcessedEmail, params map[string]interface{}) ActionResult {
	result := ActionResult{
		ActionType: "archive",
		Timestamp:  time.Now(),
		Details:    make(map[string]interface{}),
	}
	
	// Update email status to archived
	query := `UPDATE processed_emails SET metadata = jsonb_set(
		COALESCE(metadata, '{}'), 
		'{archived}', 
		'true'
	) WHERE id = $1`
	
	_, err := tas.db.Exec(query, email.ID)
	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("archive failed: %v", err)
	} else {
		result.Status = "success"
		result.Message = "email archived"
		result.Details["archived_at"] = time.Now()
	}
	
	return result
}

// executeMarkImportantAction marks an email as important
func (tas *TriageActionService) executeMarkImportantAction(email *models.ProcessedEmail, params map[string]interface{}) ActionResult {
	result := ActionResult{
		ActionType: "mark_important",
		Timestamp:  time.Now(),
		Details:    make(map[string]interface{}),
	}
	
	// Update priority score to high value
	query := `UPDATE processed_emails SET 
		priority_score = GREATEST(priority_score, 0.9),
		metadata = jsonb_set(
			COALESCE(metadata, '{}'), 
			'{important}', 
			'true'
		) WHERE id = $1`
	
	_, err := tas.db.Exec(query, email.ID)
	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("mark important failed: %v", err)
	} else {
		result.Status = "success"
		result.Message = "email marked as important"
		result.Details["new_priority"] = 0.9
	}
	
	return result
}

// executeAutoReplyAction sends an automatic reply to the sender
func (tas *TriageActionService) executeAutoReplyAction(email *models.ProcessedEmail, account *models.EmailAccount, params map[string]interface{}) ActionResult {
	result := ActionResult{
		ActionType: "auto_reply",
		Timestamp:  time.Now(),
		Details:    make(map[string]interface{}),
	}
	
	// Get reply template
	template, ok := params["template"].(string)
	if !ok {
		// Use default template
		template = "Thank you for your email. We have received your message and will respond as soon as possible."
	}
	
	// Customize template with email details
	replyBody := fmt.Sprintf(`%s

---
Original Message:
From: %s
Subject: %s
Date: %s`, template, email.SenderEmail, email.Subject, email.ProcessedAt.Format(time.RFC3339))
	
	replySubject := fmt.Sprintf("Re: %s", email.Subject)
	
	// Send auto-reply
	err := tas.emailService.SendEmail(account, []string{email.SenderEmail}, replySubject, replyBody)
	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("auto-reply failed: %v", err)
	} else {
		result.Status = "success"
		result.Message = "auto-reply sent"
		result.Details["recipient"] = email.SenderEmail
		result.Details["template_used"] = template
	}
	
	return result
}

// executeLabelAction adds a label to an email
func (tas *TriageActionService) executeLabelAction(email *models.ProcessedEmail, params map[string]interface{}) ActionResult {
	result := ActionResult{
		ActionType: "label",
		Timestamp:  time.Now(),
		Details:    make(map[string]interface{}),
	}
	
	label, ok := params["label"].(string)
	if !ok {
		result.Status = "failed"
		result.Message = "label parameter is required"
		return result
	}
	
	// Update email metadata to include label
	query := `UPDATE processed_emails SET metadata = jsonb_set(
		COALESCE(metadata, '{}'), 
		'{labels}', 
		COALESCE(metadata->'labels', '[]'::jsonb) || to_jsonb($2::text)
	) WHERE id = $1`
	
	_, err := tas.db.Exec(query, email.ID, label)
	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("label failed: %v", err)
	} else {
		result.Status = "success"
		result.Message = fmt.Sprintf("label '%s' added", label)
		result.Details["label"] = label
	}
	
	return result
}

// executeMoveToFolderAction moves email to a specific folder
func (tas *TriageActionService) executeMoveToFolderAction(email *models.ProcessedEmail, params map[string]interface{}) ActionResult {
	result := ActionResult{
		ActionType: "move_to_folder",
		Timestamp:  time.Now(),
		Details:    make(map[string]interface{}),
	}
	
	folder, ok := params["folder"].(string)
	if !ok {
		result.Status = "failed"
		result.Message = "folder parameter is required"
		return result
	}
	
	// Update email metadata to include folder
	query := `UPDATE processed_emails SET metadata = jsonb_set(
		COALESCE(metadata, '{}'), 
		'{folder}', 
		to_jsonb($2::text)
	) WHERE id = $1`
	
	_, err := tas.db.Exec(query, email.ID, folder)
	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("move to folder failed: %v", err)
	} else {
		result.Status = "success"
		result.Message = fmt.Sprintf("moved to folder '%s'", folder)
		result.Details["folder"] = folder
	}
	
	return result
}

// executeDeleteAction marks email as deleted (soft delete)
func (tas *TriageActionService) executeDeleteAction(email *models.ProcessedEmail, params map[string]interface{}) ActionResult {
	result := ActionResult{
		ActionType: "delete",
		Timestamp:  time.Now(),
		Details:    make(map[string]interface{}),
	}
	
	// Soft delete by updating metadata
	query := `UPDATE processed_emails SET metadata = jsonb_set(
		COALESCE(metadata, '{}'), 
		'{deleted}', 
		'true'
	) WHERE id = $1`
	
	_, err := tas.db.Exec(query, email.ID)
	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("delete failed: %v", err)
	} else {
		result.Status = "success"
		result.Message = "email deleted"
		result.Details["deleted_at"] = time.Now()
	}
	
	return result
}

// executeMarkReadAction marks email as read
func (tas *TriageActionService) executeMarkReadAction(email *models.ProcessedEmail, params map[string]interface{}) ActionResult {
	result := ActionResult{
		ActionType: "mark_read",
		Timestamp:  time.Now(),
		Details:    make(map[string]interface{}),
	}
	
	// Update email metadata
	query := `UPDATE processed_emails SET metadata = jsonb_set(
		COALESCE(metadata, '{}'), 
		'{read}', 
		'true'
	) WHERE id = $1`
	
	_, err := tas.db.Exec(query, email.ID)
	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("mark read failed: %v", err)
	} else {
		result.Status = "success"
		result.Message = "email marked as read"
		result.Details["read_at"] = time.Now()
	}
	
	return result
}

// Helper functions

func (tas *TriageActionService) getEmailDetails(emailID string) (*models.ProcessedEmail, error) {
	query := `
		SELECT id, account_id, message_id, subject, sender_email, 
			   recipient_emails, body_preview, full_body, priority_score,
			   processed_at, actions_taken, metadata
		FROM processed_emails
		WHERE id = $1`
	
	var email models.ProcessedEmail
	err := tas.db.QueryRow(query, emailID).Scan(
		&email.ID, &email.AccountID, &email.MessageID, &email.Subject,
		&email.SenderEmail, &email.RecipientEmails, &email.BodyPreview,
		&email.FullBody, &email.PriorityScore, &email.ProcessedAt,
		&email.ActionsTaken, &email.Metadata,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get email details: %w", err)
	}
	
	return &email, nil
}

func (tas *TriageActionService) getAccountDetails(accountID string) (*models.EmailAccount, error) {
	query := `
		SELECT id, user_id, email_address, imap_settings, smtp_settings
		FROM email_accounts
		WHERE id = $1`
	
	var account models.EmailAccount
	err := tas.db.QueryRow(query, accountID).Scan(
		&account.ID, &account.UserID, &account.EmailAddress,
		&account.IMAPSettings, &account.SMTPSettings,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get account details: %w", err)
	}
	
	return &account, nil
}

func (tas *TriageActionService) logAction(emailID string, action models.RuleAction, result ActionResult) {
	// Log to console for debugging
	log.Printf("Action executed on email %s: type=%s, status=%s, message=%s",
		emailID, action.Type, result.Status, result.Message)
	
	// TODO: Could also log to audit table
}

func (tas *TriageActionService) updateEmailActionsTaken(emailID string, results []ActionResult) {
	// Convert results to JSON
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		log.Printf("Failed to marshal action results: %v", err)
		return
	}
	
	// Update the email's actions_taken field
	query := `UPDATE processed_emails SET actions_taken = $2 WHERE id = $1`
	_, err = tas.db.Exec(query, emailID, resultsJSON)
	if err != nil {
		log.Printf("Failed to update actions_taken: %v", err)
	}
}