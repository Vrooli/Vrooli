package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// CredentialValidator validates credentials for a specific resource type.
// Implement this interface to add support for new database/service types.
type CredentialValidator interface {
	// ResourceName returns the name of the resource (e.g., "postgres", "redis")
	ResourceName() string

	// CheckID returns the preflight check ID (e.g., "postgres_credentials")
	CheckID() string

	// Title returns human-readable title for the check
	Title() string

	// Validate checks credentials and returns a PreflightCheck result
	Validate(ctx context.Context, cfg SSHConfig, sshRunner SSHRunner,
		manifest CloudManifest, secrets map[string]interface{}) PreflightCheck
}

// credentialValidators is the registry of all credential validators.
// Add new validators here to enable credential checking for additional resource types.
var credentialValidators = []CredentialValidator{
	&PostgresCredentialValidator{},
	// Add new validators here:
	// &RedisCredentialValidator{},
	// &MySQLCredentialValidator{},
}

// RunCredentialValidation runs all applicable credential validators.
// Skips validators for resources not in the manifest's dependencies.
func RunCredentialValidation(
	ctx context.Context,
	cfg SSHConfig,
	sshRunner SSHRunner,
	manifest CloudManifest,
	workdir string,
) []PreflightCheck {
	var checks []PreflightCheck

	// Build set of required resources from manifest
	requiredResources := make(map[string]bool)
	for _, res := range manifest.Dependencies.Resources {
		requiredResources[res] = true
	}

	// If no resources require credential validation, skip entirely
	hasValidatable := false
	for _, validator := range credentialValidators {
		if requiredResources[validator.ResourceName()] {
			hasValidatable = true
			break
		}
	}
	if !hasValidatable {
		return checks // No credential checks needed
	}

	// Read secrets.json once for all validators
	secrets, secretsErr := readSecretsForValidation(ctx, cfg, sshRunner, workdir)
	if secretsErr != nil {
		// If we can't read secrets, warn but don't fail - secrets may be created during deploy
		checks = append(checks, PreflightCheck{
			ID:      "secrets_read",
			Title:   "Secrets file",
			Status:  PreflightWarn,
			Details: "secrets.json not found or unreadable - will be created during deployment",
		})
		return checks
	}

	// Run each validator for required resources
	for _, validator := range credentialValidators {
		if !requiredResources[validator.ResourceName()] {
			continue // Skip validators for resources not in manifest
		}

		check := validator.Validate(ctx, cfg, sshRunner, manifest, secrets)
		checks = append(checks, check)
	}

	return checks
}

func readSecretsForValidation(ctx context.Context, cfg SSHConfig, sshRunner SSHRunner, workdir string) (map[string]interface{}, error) {
	secretsPath := safeRemoteJoin(workdir, ".vrooli", "secrets.json")
	readCmd := fmt.Sprintf("cat %s 2>/dev/null", shellQuoteSingle(secretsPath))
	result, err := sshRunner.Run(ctx, cfg, readCmd)
	if err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("secrets.json not found")
	}

	var secrets map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &secrets); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return secrets, nil
}

// ============================================
// PostgreSQL Credential Validator
// ============================================

// PostgresCredentialValidator validates PostgreSQL credentials by testing
// the connection with the password from secrets.json.
type PostgresCredentialValidator struct{}

func (v *PostgresCredentialValidator) ResourceName() string { return "postgres" }
func (v *PostgresCredentialValidator) CheckID() string      { return "postgres_credentials" }
func (v *PostgresCredentialValidator) Title() string        { return "PostgreSQL credentials" }

func (v *PostgresCredentialValidator) Validate(
	ctx context.Context,
	cfg SSHConfig,
	sshRunner SSHRunner,
	manifest CloudManifest,
	secrets map[string]interface{},
) PreflightCheck {
	password, ok := secrets["POSTGRES_PASSWORD"].(string)
	if !ok || password == "" {
		return PreflightCheck{
			ID:      v.CheckID(),
			Title:   v.Title(),
			Status:  PreflightWarn,
			Details: "POSTGRES_PASSWORD not found in secrets.json",
			Hint:    "Password will be generated during deployment",
		}
	}

	// Build database name from scenario (matches ports.sh convention)
	scenarioID := manifest.Scenario.ID
	dbName := "vrooli_" + strings.ReplaceAll(scenarioID, "-", "_")

	// Test connection using psql with the password from secrets.json
	// Using port 5433 which is the standard Vrooli postgres port mapping
	testCmd := fmt.Sprintf(
		`PGPASSWORD=%s psql -h localhost -p 5433 -U vrooli -d %s -c "SELECT 1" 2>&1 || true`,
		shellQuoteSingle(password),
		shellQuoteSingle(dbName),
	)
	testResult, _ := sshRunner.Run(ctx, cfg, testCmd)
	output := testResult.Stdout

	// Check for password mismatch - this is a FAIL (blocks deployment)
	if strings.Contains(output, "password authentication failed") {
		return PreflightCheck{
			ID:      v.CheckID(),
			Title:   v.Title(),
			Status:  PreflightFail,
			Details: "PostgreSQL password in secrets.json doesn't match database",
			Hint:    "Delete the stale PostgreSQL container (docker rm -f vrooli-postgres-main) or update secrets.json to match",
			Data:    map[string]string{"error": "password_mismatch"},
		}
	}

	// Check for successful connection (SELECT 1 returns "1" in a row)
	if strings.Contains(output, "1") && !strings.Contains(output, "error") && !strings.Contains(output, "FATAL") {
		return PreflightCheck{
			ID:      v.CheckID(),
			Title:   v.Title(),
			Status:  PreflightPass,
			Details: "PostgreSQL credentials verified",
			Data:    map[string]string{"database": dbName},
		}
	}

	// Database might not exist yet - that's OK for first deployment
	if strings.Contains(output, "does not exist") {
		return PreflightCheck{
			ID:      v.CheckID(),
			Title:   v.Title(),
			Status:  PreflightPass,
			Details: fmt.Sprintf("Database %s will be created during deployment", dbName),
		}
	}

	// PostgreSQL might not be running - warn but don't fail
	if strings.Contains(output, "could not connect") || strings.Contains(output, "Connection refused") {
		return PreflightCheck{
			ID:      v.CheckID(),
			Title:   v.Title(),
			Status:  PreflightWarn,
			Details: "PostgreSQL not running - will be started during deployment",
		}
	}

	// psql command not found - warn (will be installed during bootstrap)
	if strings.Contains(output, "command not found") || strings.Contains(output, "not found") {
		return PreflightCheck{
			ID:      v.CheckID(),
			Title:   v.Title(),
			Status:  PreflightWarn,
			Details: "psql not installed - cannot verify credentials",
			Hint:    "Credential validation will be skipped; deployment will attempt connection",
		}
	}

	// Unknown state - warn but don't block
	return PreflightCheck{
		ID:      v.CheckID(),
		Title:   v.Title(),
		Status:  PreflightWarn,
		Details: "Could not verify database connection",
		Hint:    "Connection will be attempted during deployment",
		Data:    map[string]string{"output": truncateString(output, 200)},
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
