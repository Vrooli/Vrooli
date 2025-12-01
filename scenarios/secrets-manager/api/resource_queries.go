package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func fetchResourceDetail(ctx context.Context, resourceName string) (*ResourceDetail, error) {
	resourceName = strings.TrimSpace(resourceName)
	if resourceName == "" {
		return nil, fmt.Errorf("resource name is required")
	}
	if db == nil {
		return &ResourceDetail{ResourceName: resourceName, Secrets: []ResourceSecretDetail{}}, nil
	}
	detail := &ResourceDetail{ResourceName: resourceName, Secrets: []ResourceSecretDetail{}}
	var (
		totalSecrets   sql.NullInt64
		missingSecrets sql.NullInt64
		validSecrets   sql.NullInt64
		lastValidation sql.NullTime
	)
	if err := db.QueryRowContext(ctx, `
		SELECT total_secrets, missing_required_secrets, valid_secrets, last_validation
		FROM secret_health_summary WHERE resource_name = $1
	`, resourceName).Scan(&totalSecrets, &missingSecrets, &validSecrets, &lastValidation); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if totalSecrets.Valid {
		detail.TotalSecrets = int(totalSecrets.Int64)
	}
	if missingSecrets.Valid {
		detail.MissingSecrets = int(missingSecrets.Int64)
	}
	if validSecrets.Valid {
		detail.ValidSecrets = int(validSecrets.Int64)
	}
	if lastValidation.Valid {
		detail.LastValidation = &lastValidation.Time
	}
	secretRows, err := db.QueryContext(ctx, `
		SELECT rs.id, rs.secret_key, rs.secret_type, COALESCE(rs.description,''),
		       COALESCE(rs.classification,'service'), rs.required,
		       COALESCE(rs.owner_team,''), COALESCE(rs.owner_contact,''),
		       COALESCE(tiers.tier_map, '{}'::jsonb),
		       COALESCE(v.validation_status,'unknown') as validation_status,
		       v.validation_timestamp
		FROM resource_secrets rs
		LEFT JOIN (
			SELECT DISTINCT ON (resource_secret_id)
				resource_secret_id, validation_status, validation_timestamp
			FROM secret_validations
			ORDER BY resource_secret_id, validation_timestamp DESC
		) v ON v.resource_secret_id = rs.id
		LEFT JOIN (
			SELECT resource_secret_id, jsonb_object_agg(tier, handling_strategy) AS tier_map
			FROM secret_deployment_strategies
			GROUP BY resource_secret_id
		) tiers ON tiers.resource_secret_id = rs.id
		WHERE rs.resource_name = $1
		ORDER BY rs.secret_key
	`, resourceName)
	if err != nil {
		return nil, err
	}
	defer secretRows.Close()
	for secretRows.Next() {
		var (
			id             string
			secretKey      string
			secretType     string
			description    string
			classification string
			required       bool
			ownerTeam      string
			ownerContact   string
			tierJSON       []byte
			validation     string
			validationTime sql.NullTime
		)
		if err := secretRows.Scan(&id, &secretKey, &secretType, &description, &classification, &required, &ownerTeam, &ownerContact, &tierJSON, &validation, &validationTime); err != nil {
			return nil, err
		}
		secret := ResourceSecretDetail{
			ID:              id,
			SecretKey:       secretKey,
			SecretType:      secretType,
			Description:     description,
			Classification:  classification,
			Required:        required,
			OwnerTeam:       ownerTeam,
			OwnerContact:    ownerContact,
			TierStrategies:  decodeStringMap(tierJSON),
			ValidationState: validation,
		}
		if validationTime.Valid {
			secret.LastValidated = &validationTime.Time
		}
		detail.Secrets = append(detail.Secrets, secret)
	}
	if err := secretRows.Err(); err != nil {
		return nil, err
	}
	vulnRows, err := db.QueryContext(ctx, `
		SELECT id, fingerprint, file_path, line_number, severity, vulnerability_type,
		       title, description, recommendation, code_snippet, can_auto_fix,
		       status, first_observed_at, last_observed_at
		FROM security_vulnerabilities
		WHERE component_type = 'resource' AND component_name = $1 AND status <> 'resolved'
		ORDER BY severity DESC, last_observed_at DESC
	`, resourceName)
	if err == nil {
		defer vulnRows.Close()
		for vulnRows.Next() {
			var (
				id             string
				fingerprint    string
				filePath       string
				lineNumber     sql.NullInt64
				severity       string
				typeName       string
				title          string
				description    string
				recommendation string
				codeSnippet    string
				canAutoFix     bool
				status         string
				firstSeen      time.Time
				lastSeen       time.Time
			)
			if err := vulnRows.Scan(&id, &fingerprint, &filePath, &lineNumber, &severity, &typeName, &title, &description, &recommendation, &codeSnippet, &canAutoFix, &status, &firstSeen, &lastSeen); err != nil {
				return nil, err
			}
			vuln := SecurityVulnerability{
				ID:             id,
				ComponentType:  "resource",
				ComponentName:  resourceName,
				FilePath:       filePath,
				Severity:       severity,
				Type:           typeName,
				Title:          title,
				Description:    description,
				Code:           codeSnippet,
				Recommendation: recommendation,
				CanAutoFix:     canAutoFix,
				DiscoveredAt:   firstSeen,
				Status:         status,
				Fingerprint:    fingerprint,
				LastObservedAt: lastSeen,
			}
			if lineNumber.Valid {
				vuln.LineNumber = int(lineNumber.Int64)
			}
			detail.OpenVulnerabilities = append(detail.OpenVulnerabilities, vuln)
		}
		if err := vulnRows.Err(); err != nil {
			return nil, err
		}
	}
	return detail, nil
}

func nullString(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullBytes(value []byte) interface{} {
	if len(value) == 0 {
		return nil
	}
	return value
}

func fetchSingleSecretDetail(ctx context.Context, resourceName, secretKey string) (*ResourceSecretDetail, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	row := db.QueryRowContext(ctx, `
		SELECT rs.id, rs.secret_key, rs.secret_type, COALESCE(rs.description,''),
		       COALESCE(rs.classification,'service'), rs.required,
		       COALESCE(rs.owner_team,''), COALESCE(rs.owner_contact,''),
		       COALESCE(tiers.tier_map, '{}'::jsonb),
		       COALESCE(v.validation_status,'unknown'), v.validation_timestamp
		FROM resource_secrets rs
		LEFT JOIN (
			SELECT DISTINCT ON (resource_secret_id)
				resource_secret_id, validation_status, validation_timestamp
			FROM secret_validations
			ORDER BY resource_secret_id, validation_timestamp DESC
		) v ON v.resource_secret_id = rs.id
		LEFT JOIN (
			SELECT resource_secret_id, jsonb_object_agg(tier, handling_strategy) AS tier_map
			FROM secret_deployment_strategies
			GROUP BY resource_secret_id
		) tiers ON tiers.resource_secret_id = rs.id
		WHERE rs.resource_name = $1 AND rs.secret_key = $2
	`, resourceName, secretKey)
	var (
		id             string
		secretType     string
		description    string
		classification string
		required       bool
		ownerTeam      string
		ownerContact   string
		tierJSON       []byte
		validation     string
		validationTime sql.NullTime
	)
	if err := row.Scan(&id, &secretKey, &secretType, &description, &classification, &required, &ownerTeam, &ownerContact, &tierJSON, &validation, &validationTime); err != nil {
		return nil, err
	}
	secret := &ResourceSecretDetail{
		ID:              id,
		SecretKey:       secretKey,
		SecretType:      secretType,
		Description:     description,
		Classification:  classification,
		Required:        required,
		OwnerTeam:       ownerTeam,
		OwnerContact:    ownerContact,
		TierStrategies:  decodeStringMap(tierJSON),
		ValidationState: validation,
	}
	if validationTime.Valid {
		secret.LastValidated = &validationTime.Time
	}
	return secret, nil
}

func getResourceSecretID(ctx context.Context, resourceName, secretKey string) (string, error) {
	if db == nil {
		return "", fmt.Errorf("database not initialized")
	}
	var id string
	if err := db.QueryRowContext(ctx, `SELECT id FROM resource_secrets WHERE resource_name = $1 AND secret_key = $2`, resourceName, secretKey).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}
