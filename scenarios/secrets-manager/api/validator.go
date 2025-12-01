package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SecretValidator owns secret validation so handlers and jobs share a single pipeline
// instead of duplicating logic across the codebase.
type SecretValidator struct {
	db     *sql.DB
	logger Logger
}

// NewSecretValidator creates a new secret validator
func NewSecretValidator(db *sql.DB) *SecretValidator {
	return &SecretValidator{
		db:     db,
		logger: *NewLogger("validator"),
	}
}

// ValidationMethod represents different validation methods
type ValidationMethod string

const (
	ValidationMethodEnv   ValidationMethod = "env"
	ValidationMethodVault ValidationMethod = "vault"
	ValidationMethodFile  ValidationMethod = "file"
)

// ValidateSecrets validates secrets for a specific resource or all resources
func (v *SecretValidator) ValidateSecrets(resource string) (*ValidationResponse, error) {
	validationID := uuid.New().String()
	_ = time.Now() // startTime for future use

	v.logger.Info(fmt.Sprintf("Starting secret validation (ID: %s, Resource: %s)", validationID, resource))

	// Get secrets to validate
	secrets, err := v.getSecretsForValidation(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to get secrets for validation: %w", err)
	}

	var validSecrets []SecretValidation
	var missingSecrets []SecretValidation
	var invalidSecrets []SecretValidation

	// Validate each secret
	for _, secret := range secrets {
		validation := v.validateSecret(secret)

		// Store validation result in database
		if err := v.storeValidationResult(validation); err != nil {
			v.logger.Error(fmt.Sprintf("Failed to store validation for %s", secret.SecretKey), err)
		}

		// Categorize validation result
		switch validation.ValidationStatus {
		case "valid":
			validSecrets = append(validSecrets, validation)
		case "missing":
			missingSecrets = append(missingSecrets, validation)
		case "invalid":
			invalidSecrets = append(invalidSecrets, validation)
		}
	}

	// Generate health summary
	healthSummary, err := v.generateHealthSummary(resource)
	if err != nil {
		v.logger.Error("Failed to generate health summary", err)
		healthSummary = []SecretHealthSummary{}
	}

	response := &ValidationResponse{
		ValidationID:   validationID,
		TotalSecrets:   len(secrets),
		ValidSecrets:   len(validSecrets),
		MissingSecrets: missingSecrets,
		InvalidSecrets: invalidSecrets,
		HealthSummary:  healthSummary,
	}

	v.logger.Info(fmt.Sprintf("Validation completed: %d/%d secrets valid", len(validSecrets), len(secrets)))

	return response, nil
}

// getSecretsForValidation retrieves secrets that need validation
func (v *SecretValidator) getSecretsForValidation(resource string) ([]ResourceSecret, error) {
	if v.db == nil {
		// If no database, return empty list
		return []ResourceSecret{}, nil
	}

	var query string
	var args []interface{}

	if resource != "" {
		query = `
			SELECT id, resource_name, secret_key, secret_type, required,
			       description, validation_pattern, documentation_url, default_value,
			       created_at, updated_at
			FROM resource_secrets
			WHERE resource_name = $1
			ORDER BY resource_name, secret_key
		`
		args = []interface{}{resource}
	} else {
		query = `
			SELECT id, resource_name, secret_key, secret_type, required,
			       description, validation_pattern, documentation_url, default_value,
			       created_at, updated_at
			FROM resource_secrets
			ORDER BY resource_name, secret_key
		`
		args = []interface{}{}
	}

	rows, err := v.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []ResourceSecret
	for rows.Next() {
		var secret ResourceSecret

		err := rows.Scan(
			&secret.ID, &secret.ResourceName, &secret.SecretKey, &secret.SecretType,
			&secret.Required, &secret.Description, &secret.ValidationPattern,
			&secret.DocumentationURL, &secret.DefaultValue,
			&secret.CreatedAt, &secret.UpdatedAt,
		)
		if err != nil {
			v.logger.Error("Failed to scan secret row", err)
			continue
		}

		secrets = append(secrets, secret)
	}

	return secrets, nil
}

// validateSecret validates a single secret using multiple methods
func (v *SecretValidator) validateSecret(secret ResourceSecret) SecretValidation {
	validation := SecretValidation{
		ID:                  uuid.New().String(),
		ResourceSecretID:    secret.ID,
		ValidationTimestamp: time.Now(),
	}

	// Try different validation methods in order of preference
	methods := []ValidationMethod{
		ValidationMethodEnv,   // Check environment variables first
		ValidationMethodVault, // Then check vault
		ValidationMethodFile,  // Finally check files
	}

	var lastError string
	isValid := false

	for _, method := range methods {
		switch method {
		case ValidationMethodEnv:
			if valid, err := v.validateEnvironmentVariable(secret); valid {
				validation.ValidationStatus = "valid"
				validation.ValidationMethod = string(ValidationMethodEnv)
				isValid = true
				break
			} else if err != nil {
				lastError = err.Error()
			}

		case ValidationMethodVault:
			if valid, err := v.validateVaultSecret(secret); valid {
				validation.ValidationStatus = "valid"
				validation.ValidationMethod = string(ValidationMethodVault)
				isValid = true
				break
			} else if err != nil {
				lastError = err.Error()
			}

		case ValidationMethodFile:
			if valid, err := v.validateFileSecret(secret); valid {
				validation.ValidationStatus = "valid"
				validation.ValidationMethod = string(ValidationMethodFile)
				isValid = true
				break
			} else if err != nil {
				lastError = err.Error()
			}
		}

		if isValid {
			break
		}
	}

	// Set validation status and error message
	if !isValid {
		if secret.Required {
			validation.ValidationStatus = "missing"
		} else {
			validation.ValidationStatus = "optional_missing"
		}

		if lastError != "" {
			validation.ErrorMessage = &lastError
		}
	}

	// Add validation details
	details := fmt.Sprintf("Validated using methods: %v", methods)
	validation.ValidationDetails = &details

	return validation
}

// validateEnvironmentVariable checks if the secret is available as an environment variable
func (v *SecretValidator) validateEnvironmentVariable(secret ResourceSecret) (bool, error) {
	value := os.Getenv(secret.SecretKey)
	if value == "" {
		return false, fmt.Errorf("environment variable %s is not set", secret.SecretKey)
	}

	// Additional validation based on secret type
	if err := v.validateSecretValue(value, secret); err != nil {
		return false, fmt.Errorf("environment variable %s value is invalid: %w", secret.SecretKey, err)
	}

	return true, nil
}

// validateVaultSecret checks if the secret is available in vault
func (v *SecretValidator) validateVaultSecret(secret ResourceSecret) (bool, error) {
	// This is a placeholder implementation
	// In a real implementation, you would integrate with HashiCorp Vault
	// or another secret management system

	// For now, we'll simulate vault validation by checking for vault-specific env vars
	vaultToken := os.Getenv("VAULT_TOKEN")
	vaultAddr := os.Getenv("VAULT_ADDR")

	if vaultToken == "" || vaultAddr == "" {
		return false, fmt.Errorf("vault not configured (VAULT_TOKEN or VAULT_ADDR missing)")
	}

	// Simulate vault lookup - in reality you would make API calls to vault
	// Return false for now since this is just a placeholder
	return false, fmt.Errorf("vault integration not implemented")
}

// validateFileSecret checks if the secret is available in configuration files
func (v *SecretValidator) validateFileSecret(secret ResourceSecret) (bool, error) {
	// Check common configuration file locations for the secret
	configPaths := []string{
		"/etc/vrooli/secrets.conf",
		os.ExpandEnv("$HOME/.vrooli/secrets"),
		".env",
		".env.local",
	}

	for _, path := range configPaths {
		if v.checkSecretInFile(path, secret.SecretKey) {
			return true, nil
		}
	}

	return false, fmt.Errorf("secret %s not found in any configuration file", secret.SecretKey)
}

// checkSecretInFile checks if a secret key exists in a file
func (v *SecretValidator) checkSecretInFile(filePath, secretKey string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read file content and look for the secret key
	content := make([]byte, 10240) // 10KB limit
	n, _ := file.Read(content)
	fileContent := string(content[:n])

	// Look for key=value patterns
	lines := strings.Split(fileContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, secretKey+"=") {
			return true
		}
	}

	return false
}

// validateSecretValue validates the format/content of a secret value
func (v *SecretValidator) validateSecretValue(value string, secret ResourceSecret) error {
	if value == "" {
		return fmt.Errorf("secret value is empty")
	}

	// Apply custom validation pattern if available
	if secret.ValidationPattern != nil && *secret.ValidationPattern != "" {
		// This would typically use regexp to validate the pattern
		// For now, just check basic constraints
		if len(value) < 3 {
			return fmt.Errorf("secret value too short (minimum 3 characters)")
		}
	}

	// Type-specific validation
	switch secret.SecretType {
	case "password":
		if len(value) < 8 {
			return fmt.Errorf("password too short (minimum 8 characters)")
		}
	case "api_key":
		if len(value) < 16 {
			return fmt.Errorf("API key too short (minimum 16 characters)")
		}
	case "token":
		if len(value) < 10 {
			return fmt.Errorf("token too short (minimum 10 characters)")
		}
	}

	return nil
}

// storeValidationResult stores a validation result in the database
func (v *SecretValidator) storeValidationResult(validation SecretValidation) error {
	if v.db == nil {
		return nil // Skip if no database connection
	}

	query := `
		INSERT INTO secret_validations (
			id, resource_secret_id, validation_status, validation_method,
			validation_timestamp, error_message, validation_details
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := v.db.Exec(query,
		validation.ID, validation.ResourceSecretID, validation.ValidationStatus,
		validation.ValidationMethod, validation.ValidationTimestamp,
		validation.ErrorMessage, validation.ValidationDetails,
	)

	return err
}

// generateHealthSummary generates a health summary for resources
func (v *SecretValidator) generateHealthSummary(resourceFilter string) ([]SecretHealthSummary, error) {
	if v.db == nil {
		return []SecretHealthSummary{}, nil
	}

	var query string
	var args []interface{}

	if resourceFilter != "" {
		query = `
			SELECT 
				rs.resource_name,
				COUNT(*) as total_secrets,
				SUM(CASE WHEN rs.required THEN 1 ELSE 0 END) as required_secrets,
				SUM(CASE WHEN sv.validation_status = 'valid' THEN 1 ELSE 0 END) as valid_secrets,
				SUM(CASE WHEN rs.required AND (sv.validation_status != 'valid' OR sv.validation_status IS NULL) THEN 1 ELSE 0 END) as missing_required_secrets,
				SUM(CASE WHEN sv.validation_status IN ('invalid', 'missing') THEN 1 ELSE 0 END) as invalid_secrets,
				MAX(sv.validation_timestamp) as last_validation
			FROM resource_secrets rs
			LEFT JOIN secret_validations sv ON rs.id = sv.resource_secret_id
			WHERE rs.resource_name = $1
			GROUP BY rs.resource_name
			ORDER BY rs.resource_name
		`
		args = []interface{}{resourceFilter}
	} else {
		query = `
			SELECT 
				rs.resource_name,
				COUNT(*) as total_secrets,
				SUM(CASE WHEN rs.required THEN 1 ELSE 0 END) as required_secrets,
				SUM(CASE WHEN sv.validation_status = 'valid' THEN 1 ELSE 0 END) as valid_secrets,
				SUM(CASE WHEN rs.required AND (sv.validation_status != 'valid' OR sv.validation_status IS NULL) THEN 1 ELSE 0 END) as missing_required_secrets,
				SUM(CASE WHEN sv.validation_status IN ('invalid', 'missing') THEN 1 ELSE 0 END) as invalid_secrets,
				MAX(sv.validation_timestamp) as last_validation
			FROM resource_secrets rs
			LEFT JOIN secret_validations sv ON rs.id = sv.resource_secret_id
			GROUP BY rs.resource_name
			ORDER BY rs.resource_name
		`
		args = []interface{}{}
	}

	rows, err := v.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []SecretHealthSummary
	for rows.Next() {
		var summary SecretHealthSummary
		var lastValidation sql.NullTime

		err := rows.Scan(
			&summary.ResourceName, &summary.TotalSecrets, &summary.RequiredSecrets,
			&summary.ValidSecrets, &summary.MissingRequiredSecrets, &summary.InvalidSecrets,
			&lastValidation,
		)
		if err != nil {
			v.logger.Error("Failed to scan health summary row", err)
			continue
		}

		if lastValidation.Valid {
			summary.LastValidation = &lastValidation.Time
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// GetValidationHistory returns recent validation history
func (v *SecretValidator) GetValidationHistory(resourceName string, limit int) ([]SecretValidation, error) {
	if v.db == nil {
		return []SecretValidation{}, nil
	}

	var query string
	var args []interface{}

	if resourceName != "" {
		query = `
			SELECT sv.id, sv.resource_secret_id, sv.validation_status, 
			       sv.validation_method, sv.validation_timestamp, sv.error_message,
			       sv.validation_details, rs.resource_name, rs.secret_key
			FROM secret_validations sv
			JOIN resource_secrets rs ON sv.resource_secret_id = rs.id
			WHERE rs.resource_name = $1
			ORDER BY sv.validation_timestamp DESC
			LIMIT $2
		`
		args = []interface{}{resourceName, limit}
	} else {
		query = `
			SELECT sv.id, sv.resource_secret_id, sv.validation_status, 
			       sv.validation_method, sv.validation_timestamp, sv.error_message,
			       sv.validation_details, rs.resource_name, rs.secret_key
			FROM secret_validations sv
			JOIN resource_secrets rs ON sv.resource_secret_id = rs.id
			ORDER BY sv.validation_timestamp DESC
			LIMIT $1
		`
		args = []interface{}{limit}
	}

	rows, err := v.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var validations []SecretValidation
	for rows.Next() {
		var validation SecretValidation
		var resourceName, secretKey string

		err := rows.Scan(
			&validation.ID, &validation.ResourceSecretID, &validation.ValidationStatus,
			&validation.ValidationMethod, &validation.ValidationTimestamp,
			&validation.ErrorMessage, &validation.ValidationDetails,
			&resourceName, &secretKey,
		)
		if err != nil {
			v.logger.Error("Failed to scan validation row", err)
			continue
		}

		validations = append(validations, validation)
	}

	return validations, nil
}
