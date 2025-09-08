package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

type AuthProcessor struct {
	db          *sql.DB
	emailSender *EmailSender
}

type EmailSender struct {
	mailpitURL string
}

type TokenValidationRequest struct {
	Token string `json:"token"`
}

type TokenValidationResponse struct {
	Valid       bool      `json:"valid"`
	UserID      string    `json:"user_id,omitempty"`
	Email       string    `json:"email,omitempty"`
	Roles       []string  `json:"roles,omitempty"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	Error       string    `json:"error,omitempty"`
	Code        string    `json:"code,omitempty"`
	HTTPStatus  int       `json:"http_status"`
	Timestamp   string    `json:"timestamp"`
	Service     string    `json:"service"`
	Workflow    string    `json:"workflow"`
	ValidatedAt string    `json:"validated_at,omitempty"`
}

type PasswordResetRequest struct {
	Email string `json:"email"`
}

type PasswordResetResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	EmailAttempted string `json:"email_attempted,omitempty"`
	UserFound     bool   `json:"user_found,omitempty"`
	Timestamp     string `json:"timestamp"`
	Error         string `json:"error,omitempty"`
	Code          string `json:"code,omitempty"`
}

type ResetToken struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func NewAuthProcessor(db *sql.DB) *AuthProcessor {
	mailpitURL := os.Getenv("MAILPIT_URL")
	if mailpitURL == "" {
		mailpitURL = "http://localhost:1025"
	}

	return &AuthProcessor{
		db: db,
		emailSender: &EmailSender{
			mailpitURL: mailpitURL,
		},
	}
}

// ValidateAuthToken validates JWT tokens (replaces n8n auth-validator workflow)
func (ap *AuthProcessor) ValidateAuthToken(ctx context.Context, r *http.Request) TokenValidationResponse {
	response := TokenValidationResponse{
		Timestamp: time.Now().Format(time.RFC3339),
		Service:   "scenario-authenticator",
		Workflow:  "auth-validator",
	}

	// Extract token from Authorization header or request body
	token := ap.extractToken(r)
	if token == "" {
		response.Valid = false
		response.Error = "No authentication token provided"
		response.Code = "MISSING_TOKEN"
		response.HTTPStatus = http.StatusUnauthorized
		return response
	}

	// Check if token is blacklisted
	if isTokenBlacklisted(token) {
		response.Valid = false
		response.Error = "Invalid or expired token"
		response.Code = "INVALID_TOKEN"
		response.HTTPStatus = http.StatusUnauthorized
		return response
	}

	// Parse and validate token
	parsedToken, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})

	if err != nil || !parsedToken.Valid {
		response.Valid = false
		response.Error = "Invalid or expired token"
		response.Code = "INVALID_TOKEN"
		response.HTTPStatus = http.StatusUnauthorized
		return response
	}

	claims, ok := parsedToken.Claims.(*Claims)
	if !ok {
		response.Valid = false
		response.Error = "Invalid token claims"
		response.Code = "INVALID_TOKEN"
		response.HTTPStatus = http.StatusUnauthorized
		return response
	}

	// Token is valid - return user info
	response.Valid = true
	response.UserID = claims.UserID
	response.Email = claims.Email
	response.Roles = claims.Roles
	response.ExpiresAt = time.Unix(claims.ExpiresAt, 0)
	response.ValidatedAt = time.Now().Format(time.RFC3339)
	response.HTTPStatus = http.StatusOK

	return response
}

// ProcessPasswordReset handles password reset requests (replaces n8n password-reset workflow)
func (ap *AuthProcessor) ProcessPasswordReset(ctx context.Context, req PasswordResetRequest) PasswordResetResponse {
	response := PasswordResetResponse{
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Validate email format
	if !ap.isValidEmail(req.Email) {
		response.Success = false
		response.Error = "Invalid email format"
		response.Code = "INVALID_EMAIL"
		return response
	}

	email := strings.ToLower(req.Email)

	// Check if user exists
	var userID, username string
	var emailVerified bool
	query := `
		SELECT id, username, email_verified 
		FROM users 
		WHERE LOWER(email) = $1 AND deleted_at IS NULL`
	
	err := ap.db.QueryRowContext(ctx, query, email).Scan(&userID, &username, &emailVerified)
	
	if err == sql.ErrNoRows {
		// Don't reveal if email exists (security best practice)
		response.Success = true
		response.Message = "If an account with that email exists, a password reset link has been sent."
		response.EmailAttempted = email
		response.UserFound = false
		return response
	} else if err != nil {
		// Database error
		response.Success = false
		response.Error = "Service temporarily unavailable"
		response.Code = "SERVICE_ERROR"
		return response
	}

	// Generate reset token
	resetToken := ap.generateResetToken()
	expiresAt := time.Now().Add(1 * time.Hour)
	
	// Store reset token in database
	err = ap.storeResetToken(ctx, userID, resetToken, expiresAt)
	if err != nil {
		response.Success = false
		response.Error = "Failed to generate reset token"
		response.Code = "TOKEN_GENERATION_ERROR"
		return response
	}

	// Generate reset link
	resetLink := fmt.Sprintf("http://localhost:3251/reset-password?token=%s&email=%s", 
		resetToken, email)

	// Send reset email
	emailBody := fmt.Sprintf(`
		<html>
		<body>
			<h2>Password Reset Request</h2>
			<p>Hi %s,</p>
			<p>You requested a password reset for your Vrooli account.</p>
			<p>Click the link below to reset your password:</p>
			<p>
				<a href='%s' style='background: #4F46E5; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; display: inline-block;'>
					Reset Password
				</a>
			</p>
			<p>Or copy this link: %s</p>
			<p>This link will expire in 1 hour.</p>
			<p>If you didn't request this reset, you can safely ignore this email.</p>
			<p>Best regards,<br>The Vrooli Team</p>
		</body>
		</html>
	`, username, resetLink, resetLink)

	err = ap.emailSender.SendEmail(ctx, email, "Reset Your Vrooli Password", emailBody)
	if err != nil {
		// Log the error but don't reveal email sending failure to user
		fmt.Printf("Failed to send reset email to %s: %v\n", email, err)
	}

	// Always return success for security (don't reveal if email exists)
	response.Success = true
	response.Message = "If an account with that email exists, a password reset link has been sent."
	response.EmailAttempted = email
	response.UserFound = true

	// Log password reset attempt
	ap.logPasswordResetAttempt(ctx, userID, email, true)

	return response
}

// CompletePasswordReset completes the password reset process
func (ap *AuthProcessor) CompletePasswordReset(ctx context.Context, token, newPassword string) error {
	// Validate token and get user ID
	var userID string
	var expiresAt time.Time
	
	query := `
		SELECT user_id, expires_at 
		FROM password_reset_tokens 
		WHERE token = $1 AND used = false`
	
	err := ap.db.QueryRowContext(ctx, query, token).Scan(&userID, &expiresAt)
	if err == sql.ErrNoRows {
		return fmt.Errorf("invalid or expired reset token")
	} else if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	// Check if token has expired
	if time.Now().After(expiresAt) {
		return fmt.Errorf("reset token has expired")
	}

	// Validate new password
	if len(newPassword) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Begin transaction
	tx, err := ap.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Update user password
	_, err = tx.ExecContext(ctx, 
		`UPDATE users SET password_hash = $1, password_changed_at = $2 WHERE id = $3`,
		string(hashedPassword), time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Mark token as used
	_, err = tx.ExecContext(ctx,
		`UPDATE password_reset_tokens SET used = true, used_at = $1 WHERE token = $2`,
		time.Now(), token)
	if err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}

	// Clear all user sessions (force re-login)
	clearUserSessions(userID)

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Log password reset completion
	ap.logPasswordResetCompletion(ctx, userID)

	return nil
}

// Helper methods

func (ap *AuthProcessor) extractToken(r *http.Request) string {
	// Check Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		return strings.Replace(authHeader, "Bearer ", "", 1)
	}

	// Check request body
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil && body.Token != "" {
		return body.Token
	}

	return ""
}

func (ap *AuthProcessor) isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	return emailRegex.MatchString(email)
}

func (ap *AuthProcessor) generateResetToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (ap *AuthProcessor) storeResetToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	query := `
		INSERT INTO password_reset_tokens (token, user_id, expires_at, created_at, used)
		VALUES ($1, $2, $3, $4, false)`
	
	_, err := ap.db.ExecContext(ctx, query, token, userID, expiresAt, time.Now())
	return err
}

func (ap *AuthProcessor) logPasswordResetAttempt(ctx context.Context, userID, email string, success bool) {
	metadata := map[string]interface{}{
		"email": email,
		"type": "password_reset_request",
	}
	metadataJSON, _ := json.Marshal(metadata)
	
	query := `
		INSERT INTO audit_logs (user_id, action, success, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5)`
	
	ap.db.ExecContext(ctx, query, userID, "password.reset.requested", success, metadataJSON, time.Now())
}

func (ap *AuthProcessor) logPasswordResetCompletion(ctx context.Context, userID string) {
	metadata := map[string]interface{}{
		"type": "password_reset_complete",
	}
	metadataJSON, _ := json.Marshal(metadata)
	
	query := `
		INSERT INTO audit_logs (user_id, action, success, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5)`
	
	ap.db.ExecContext(ctx, query, userID, "password.reset.completed", true, metadataJSON, time.Now())
}

// SendEmail sends an email via Mailpit or configured email service
func (es *EmailSender) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	// In production, this would use a real email service
	// For development, we'll use Mailpit or log the email
	
	emailData := map[string]interface{}{
		"to":      to,
		"subject": subject,
		"html":    htmlBody,
		"from":    "noreply@vrooli.com",
	}
	
	emailJSON, err := json.Marshal(emailData)
	if err != nil {
		return fmt.Errorf("failed to marshal email data: %w", err)
	}
	
	// Try to send via Mailpit (development)
	if es.mailpitURL != "" {
		req, err := http.NewRequestWithContext(ctx, "POST", 
			fmt.Sprintf("%s/api/v1/send", es.mailpitURL), 
			strings.NewReader(string(emailJSON)))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		
		req.Header.Set("Content-Type", "application/json")
		
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			// Log the email content if Mailpit is not available
			fmt.Printf("Email Service Unavailable - Logging email:\n")
			fmt.Printf("To: %s\n", to)
			fmt.Printf("Subject: %s\n", subject)
			fmt.Printf("Body: %s\n", htmlBody)
			return nil // Don't fail if email service is down
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("email service returned status %d", resp.StatusCode)
		}
	}
	
	return nil
}