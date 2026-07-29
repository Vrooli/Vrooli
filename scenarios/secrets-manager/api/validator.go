package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/database"
)

// SecretValidator owns secret validation so handlers and jobs share a single pipeline
// instead of duplicating logic across the codebase.
type SecretValidator struct {
	db     *database.RoutedDB
	logger Logger
}

// NewSecretValidator creates a new secret validator
func NewSecretValidator(db *database.RoutedDB) *SecretValidator {
	return &SecretValidator{
		db:     db,
		logger: *NewLogger("validator"),
	}
}

// ValidationMethod identifies the authority that supplied metadata-only status.
type ValidationMethod string

const (
	ValidationMethodAuthority ValidationMethod = "credential_authority"
)

// -----------------------------------------------------------------------------
// Validation Status Decisions
// -----------------------------------------------------------------------------
//
// Validation status reflects whether a secret was found and is accessible.
// The status differs based on whether the secret is required:
//   - Required secrets that aren't found → "missing" (blocks functionality)
//   - Optional secrets that aren't found → "optional_missing" (informational)
//   - Found and valid secrets → "valid"
//   - Found but malformed secrets → "invalid"

// determineValidationFailureStatus returns the appropriate status when
// a secret cannot be found or validated.
//
// Decision logic:
//   - Required secrets get "missing" status (this is a blocking issue)
//   - Optional secrets get "optional_missing" status (informational only)
func determineValidationFailureStatus(isRequired bool) string {
	if isRequired {
		return "missing"
	}
	return "optional_missing"
}

// ValidateSecrets validates secrets for a specific resource or all resources
func (v *SecretValidator) ValidateSecrets(resource string) (*ValidationResponse, error) {
	return v.ValidateSecretsContext(context.Background(), resource)
}

// ValidateSecretsContext validates secrets using the caller's context. HTTP
// handlers pass their request context so apihttp test-mode routing reaches the
// per-run database instead of the primary database.
func (v *SecretValidator) ValidateSecretsContext(ctx context.Context, resource string) (*ValidationResponse, error) {
	validationID := uuid.New().String()
	_ = time.Now() // startTime for future use

	v.logger.Info(fmt.Sprintf("Starting secret validation (ID: %s, Resource: %s)", validationID, resource))

	// Get secrets to validate
	secrets, err := v.getSecretsForValidationContext(ctx, resource)
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
		if err := v.storeValidationResultContext(ctx, validation); err != nil {
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
	healthSummary, err := v.generateHealthSummaryContext(ctx, resource)
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
	return v.getSecretsForValidationContext(context.Background(), resource)
}

func (v *SecretValidator) getSecretsForValidationContext(ctx context.Context, resource string) ([]ResourceSecret, error) {
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

	rows, err := v.db.QueryContext(ctx, query, args...)
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

// validateSecret asks the canonical credential authority for status metadata.
// It never reads a value from the process environment, Vault, or a file.
func (v *SecretValidator) validateSecret(secret ResourceSecret) SecretValidation {
	validation := SecretValidation{
		ID:                  uuid.New().String(),
		ResourceSecretID:    secret.ID,
		ValidationTimestamp: time.Now(),
	}

	configured, err := credentialConfigured(context.Background(), secret.ResourceName, secret.SecretKey)
	validation.ValidationMethod = string(ValidationMethodAuthority)
	if configured {
		validation.ValidationStatus = "valid"
	} else {
		validation.ValidationStatus = determineValidationFailureStatus(secret.Required)
		if err != nil {
			detail := err.Error()
			validation.ErrorMessage = &detail
		}
	}
	details := "Validated through canonical credential authority status metadata."
	validation.ValidationDetails = &details

	return validation
}

// storeValidationResult stores a validation result in the database
func (v *SecretValidator) storeValidationResult(validation SecretValidation) error {
	return v.storeValidationResultContext(context.Background(), validation)
}

func (v *SecretValidator) storeValidationResultContext(ctx context.Context, validation SecretValidation) error {
	if v.db == nil {
		return nil // Skip if no database connection
	}

	query := `
		INSERT INTO secret_validations (
			id, resource_secret_id, validation_status, validation_method,
			validation_timestamp, error_message, validation_details
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := v.db.ExecContext(ctx, query,
		validation.ID, validation.ResourceSecretID, validation.ValidationStatus,
		validation.ValidationMethod, validation.ValidationTimestamp,
		validation.ErrorMessage, validation.ValidationDetails,
	)

	return err
}

// generateHealthSummary generates a health summary for resources
func (v *SecretValidator) generateHealthSummary(resourceFilter string) ([]SecretHealthSummary, error) {
	return v.generateHealthSummaryContext(context.Background(), resourceFilter)
}

func (v *SecretValidator) generateHealthSummaryContext(ctx context.Context, resourceFilter string) ([]SecretHealthSummary, error) {
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

	rows, err := v.db.QueryContext(ctx, query, args...)
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

	rows, err := v.db.QueryContext(context.Background(), query, args...)
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
