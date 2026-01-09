// Package main provides database query functions for scenario secret strategy overrides.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
)

// ScenarioOverrideStore handles database operations for scenario overrides.
type ScenarioOverrideStore struct {
	db     *sql.DB
	logger *Logger
}

// NewScenarioOverrideStore creates a new store for scenario override operations.
func NewScenarioOverrideStore(db *sql.DB, logger *Logger) *ScenarioOverrideStore {
	return &ScenarioOverrideStore{db: db, logger: logger}
}

// FetchOverridesByScenario retrieves all overrides for a scenario across all tiers.
func (s *ScenarioOverrideStore) FetchOverridesByScenario(ctx context.Context, scenario string) ([]ScenarioSecretOverride, error) {
	query := `
		SELECT
			sso.id,
			sso.scenario_name,
			sso.resource_secret_id,
			rs.resource_name,
			rs.secret_key,
			sso.tier,
			sso.handling_strategy,
			sso.fallback_strategy,
			sso.requires_user_input,
			sso.prompt_label,
			sso.prompt_description,
			sso.generator_template,
			sso.bundle_hints,
			sso.override_reason,
			sso.created_at,
			sso.updated_at
		FROM scenario_secret_strategy_overrides sso
		JOIN resource_secrets rs ON rs.id = sso.resource_secret_id
		WHERE sso.scenario_name = $1
		ORDER BY rs.resource_name, rs.secret_key, sso.tier
	`

	rows, err := s.db.QueryContext(ctx, query, scenario)
	if err != nil {
		return nil, fmt.Errorf("query overrides: %w", err)
	}
	defer rows.Close()

	return s.scanOverrideRows(rows)
}

// FetchOverridesByScenarioTier retrieves all overrides for a scenario and tier.
func (s *ScenarioOverrideStore) FetchOverridesByScenarioTier(ctx context.Context, scenario, tier string) ([]ScenarioSecretOverride, error) {
	query := `
		SELECT
			sso.id,
			sso.scenario_name,
			sso.resource_secret_id,
			rs.resource_name,
			rs.secret_key,
			sso.tier,
			sso.handling_strategy,
			sso.fallback_strategy,
			sso.requires_user_input,
			sso.prompt_label,
			sso.prompt_description,
			sso.generator_template,
			sso.bundle_hints,
			sso.override_reason,
			sso.created_at,
			sso.updated_at
		FROM scenario_secret_strategy_overrides sso
		JOIN resource_secrets rs ON rs.id = sso.resource_secret_id
		WHERE sso.scenario_name = $1 AND sso.tier = $2
		ORDER BY rs.resource_name, rs.secret_key
	`

	rows, err := s.db.QueryContext(ctx, query, scenario, tier)
	if err != nil {
		return nil, fmt.Errorf("query overrides: %w", err)
	}
	defer rows.Close()

	return s.scanOverrideRows(rows)
}

// FetchOverride retrieves a specific override by scenario, tier, resource, and secret.
func (s *ScenarioOverrideStore) FetchOverride(ctx context.Context, scenario, tier, resource, secret string) (*ScenarioSecretOverride, error) {
	query := `
		SELECT
			sso.id,
			sso.scenario_name,
			sso.resource_secret_id,
			rs.resource_name,
			rs.secret_key,
			sso.tier,
			sso.handling_strategy,
			sso.fallback_strategy,
			sso.requires_user_input,
			sso.prompt_label,
			sso.prompt_description,
			sso.generator_template,
			sso.bundle_hints,
			sso.override_reason,
			sso.created_at,
			sso.updated_at
		FROM scenario_secret_strategy_overrides sso
		JOIN resource_secrets rs ON rs.id = sso.resource_secret_id
		WHERE sso.scenario_name = $1
		  AND sso.tier = $2
		  AND rs.resource_name = $3
		  AND rs.secret_key = $4
	`

	row := s.db.QueryRowContext(ctx, query, scenario, tier, resource, secret)
	override, err := s.scanOverrideRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fetch override: %w", err)
	}
	return override, nil
}

// UpsertOverride creates or updates a scenario override.
func (s *ScenarioOverrideStore) UpsertOverride(ctx context.Context, scenario, tier, resource, secret string, req SetOverrideRequest) (*ScenarioSecretOverride, error) {
	// First, find the resource_secret_id
	var resourceSecretID string
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM resource_secrets WHERE resource_name = $1 AND secret_key = $2`,
		resource, secret,
	).Scan(&resourceSecretID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("secret %s/%s not found", resource, secret)
	}
	if err != nil {
		return nil, fmt.Errorf("lookup secret: %w", err)
	}

	// Convert maps to JSON
	var generatorJSON, bundleJSON []byte
	if req.GeneratorTemplate != nil {
		generatorJSON, _ = json.Marshal(req.GeneratorTemplate)
	}
	if req.BundleHints != nil {
		bundleJSON, _ = json.Marshal(req.BundleHints)
	}

	// Upsert the override
	query := `
		INSERT INTO scenario_secret_strategy_overrides (
			scenario_name, resource_secret_id, tier,
			handling_strategy, fallback_strategy, requires_user_input,
			prompt_label, prompt_description, generator_template, bundle_hints,
			override_reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (scenario_name, resource_secret_id, tier) DO UPDATE SET
			handling_strategy = EXCLUDED.handling_strategy,
			fallback_strategy = EXCLUDED.fallback_strategy,
			requires_user_input = EXCLUDED.requires_user_input,
			prompt_label = EXCLUDED.prompt_label,
			prompt_description = EXCLUDED.prompt_description,
			generator_template = EXCLUDED.generator_template,
			bundle_hints = EXCLUDED.bundle_hints,
			override_reason = EXCLUDED.override_reason,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id
	`

	var id string
	err = s.db.QueryRowContext(ctx, query,
		scenario, resourceSecretID, tier,
		req.HandlingStrategy, req.FallbackStrategy, req.RequiresUserInput,
		req.PromptLabel, req.PromptDescription, generatorJSON, bundleJSON,
		req.OverrideReason,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("upsert override: %w", err)
	}

	// Fetch and return the complete override
	return s.FetchOverride(ctx, scenario, tier, resource, secret)
}

// DeleteOverride removes a scenario override.
func (s *ScenarioOverrideStore) DeleteOverride(ctx context.Context, scenario, tier, resource, secret string) error {
	query := `
		DELETE FROM scenario_secret_strategy_overrides sso
		USING resource_secrets rs
		WHERE sso.resource_secret_id = rs.id
		  AND sso.scenario_name = $1
		  AND sso.tier = $2
		  AND rs.resource_name = $3
		  AND rs.secret_key = $4
	`

	result, err := s.db.ExecContext(ctx, query, scenario, tier, resource, secret)
	if err != nil {
		return fmt.Errorf("delete override: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil // Not an error - just means it didn't exist
	}
	return nil
}

// FetchEffectiveStrategies retrieves the effective strategies for a scenario and tier,
// merging resource defaults with any scenario overrides.
func (s *ScenarioOverrideStore) FetchEffectiveStrategies(ctx context.Context, scenario, tier string, resources []string) ([]EffectiveSecretStrategy, error) {
	query := `
		SELECT
			rs.resource_name,
			rs.secret_key,
			$2 as tier,
			COALESCE(sso.handling_strategy, sds.handling_strategy, 'unspecified') as handling_strategy,
			COALESCE(sso.fallback_strategy, sds.fallback_strategy, '') as fallback_strategy,
			COALESCE(sso.requires_user_input, sds.requires_user_input, false) as requires_user_input,
			COALESCE(sso.prompt_label, sds.prompt_label, '') as prompt_label,
			COALESCE(sso.prompt_description, sds.prompt_description, '') as prompt_description,
			(sso.id IS NOT NULL) as is_overridden,
			sso.override_reason
		FROM resource_secrets rs
		LEFT JOIN secret_deployment_strategies sds
			ON sds.resource_secret_id = rs.id AND sds.tier = $2
		LEFT JOIN scenario_secret_strategy_overrides sso
			ON sso.resource_secret_id = rs.id
			AND sso.tier = $2
			AND sso.scenario_name = $1
	`

	args := []interface{}{scenario, tier}
	if len(resources) > 0 {
		query += " WHERE rs.resource_name = ANY($3)"
		args = append(args, pq.Array(resources))
	}
	query += " ORDER BY rs.resource_name, rs.secret_key"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query effective strategies: %w", err)
	}
	defer rows.Close()

	var strategies []EffectiveSecretStrategy
	for rows.Next() {
		var strategy EffectiveSecretStrategy
		var overrideReason sql.NullString

		if err := rows.Scan(
			&strategy.ResourceName,
			&strategy.SecretKey,
			&strategy.Tier,
			&strategy.HandlingStrategy,
			&strategy.FallbackStrategy,
			&strategy.RequiresUserInput,
			&strategy.PromptLabel,
			&strategy.PromptDescription,
			&strategy.IsOverridden,
			&overrideReason,
		); err != nil {
			return nil, fmt.Errorf("scan effective strategy: %w", err)
		}

		if overrideReason.Valid {
			strategy.OverrideReason = overrideReason.String
		}

		// Determine which fields are overridden (if any)
		if strategy.IsOverridden {
			strategy.OverriddenFields = s.determineOverriddenFields(ctx, scenario, tier, strategy.ResourceName, strategy.SecretKey)
		}

		strategies = append(strategies, strategy)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate strategies: %w", err)
	}

	return strategies, nil
}

// CopyOverridesFromTier copies all overrides from one tier to another for a scenario.
func (s *ScenarioOverrideStore) CopyOverridesFromTier(ctx context.Context, scenario, sourceTier, targetTier string, overwrite bool) (int, error) {
	var query string
	if overwrite {
		query = `
			INSERT INTO scenario_secret_strategy_overrides (
				scenario_name, resource_secret_id, tier,
				handling_strategy, fallback_strategy, requires_user_input,
				prompt_label, prompt_description, generator_template, bundle_hints,
				override_reason
			)
			SELECT
				scenario_name, resource_secret_id, $3,
				handling_strategy, fallback_strategy, requires_user_input,
				prompt_label, prompt_description, generator_template, bundle_hints,
				override_reason
			FROM scenario_secret_strategy_overrides
			WHERE scenario_name = $1 AND tier = $2
			ON CONFLICT (scenario_name, resource_secret_id, tier) DO UPDATE SET
				handling_strategy = EXCLUDED.handling_strategy,
				fallback_strategy = EXCLUDED.fallback_strategy,
				requires_user_input = EXCLUDED.requires_user_input,
				prompt_label = EXCLUDED.prompt_label,
				prompt_description = EXCLUDED.prompt_description,
				generator_template = EXCLUDED.generator_template,
				bundle_hints = EXCLUDED.bundle_hints,
				override_reason = EXCLUDED.override_reason,
				updated_at = CURRENT_TIMESTAMP
		`
	} else {
		query = `
			INSERT INTO scenario_secret_strategy_overrides (
				scenario_name, resource_secret_id, tier,
				handling_strategy, fallback_strategy, requires_user_input,
				prompt_label, prompt_description, generator_template, bundle_hints,
				override_reason
			)
			SELECT
				scenario_name, resource_secret_id, $3,
				handling_strategy, fallback_strategy, requires_user_input,
				prompt_label, prompt_description, generator_template, bundle_hints,
				override_reason
			FROM scenario_secret_strategy_overrides
			WHERE scenario_name = $1 AND tier = $2
			ON CONFLICT (scenario_name, resource_secret_id, tier) DO NOTHING
		`
	}

	result, err := s.db.ExecContext(ctx, query, scenario, sourceTier, targetTier)
	if err != nil {
		return 0, fmt.Errorf("copy overrides from tier: %w", err)
	}

	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// CopyOverridesFromScenario copies overrides from another scenario for a specific tier.
func (s *ScenarioOverrideStore) CopyOverridesFromScenario(ctx context.Context, sourceScenario, targetScenario, tier string, overwrite bool) (int, error) {
	var query string
	if overwrite {
		query = `
			INSERT INTO scenario_secret_strategy_overrides (
				scenario_name, resource_secret_id, tier,
				handling_strategy, fallback_strategy, requires_user_input,
				prompt_label, prompt_description, generator_template, bundle_hints,
				override_reason
			)
			SELECT
				$2, resource_secret_id, tier,
				handling_strategy, fallback_strategy, requires_user_input,
				prompt_label, prompt_description, generator_template, bundle_hints,
				override_reason
			FROM scenario_secret_strategy_overrides
			WHERE scenario_name = $1 AND tier = $3
			ON CONFLICT (scenario_name, resource_secret_id, tier) DO UPDATE SET
				handling_strategy = EXCLUDED.handling_strategy,
				fallback_strategy = EXCLUDED.fallback_strategy,
				requires_user_input = EXCLUDED.requires_user_input,
				prompt_label = EXCLUDED.prompt_label,
				prompt_description = EXCLUDED.prompt_description,
				generator_template = EXCLUDED.generator_template,
				bundle_hints = EXCLUDED.bundle_hints,
				override_reason = EXCLUDED.override_reason,
				updated_at = CURRENT_TIMESTAMP
		`
	} else {
		query = `
			INSERT INTO scenario_secret_strategy_overrides (
				scenario_name, resource_secret_id, tier,
				handling_strategy, fallback_strategy, requires_user_input,
				prompt_label, prompt_description, generator_template, bundle_hints,
				override_reason
			)
			SELECT
				$2, resource_secret_id, tier,
				handling_strategy, fallback_strategy, requires_user_input,
				prompt_label, prompt_description, generator_template, bundle_hints,
				override_reason
			FROM scenario_secret_strategy_overrides
			WHERE scenario_name = $1 AND tier = $3
			ON CONFLICT (scenario_name, resource_secret_id, tier) DO NOTHING
		`
	}

	result, err := s.db.ExecContext(ctx, query, sourceScenario, targetScenario, tier)
	if err != nil {
		return 0, fmt.Errorf("copy overrides from scenario: %w", err)
	}

	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// FetchAllOverrides retrieves all overrides in the database (for admin purposes).
func (s *ScenarioOverrideStore) FetchAllOverrides(ctx context.Context) ([]ScenarioSecretOverride, error) {
	query := `
		SELECT
			sso.id,
			sso.scenario_name,
			sso.resource_secret_id,
			rs.resource_name,
			rs.secret_key,
			sso.tier,
			sso.handling_strategy,
			sso.fallback_strategy,
			sso.requires_user_input,
			sso.prompt_label,
			sso.prompt_description,
			sso.generator_template,
			sso.bundle_hints,
			sso.override_reason,
			sso.created_at,
			sso.updated_at
		FROM scenario_secret_strategy_overrides sso
		JOIN resource_secrets rs ON rs.id = sso.resource_secret_id
		ORDER BY sso.scenario_name, rs.resource_name, rs.secret_key, sso.tier
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all overrides: %w", err)
	}
	defer rows.Close()

	return s.scanOverrideRows(rows)
}

// DeleteOverridesByID deletes overrides by their IDs.
func (s *ScenarioOverrideStore) DeleteOverridesByID(ctx context.Context, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	query := `DELETE FROM scenario_secret_strategy_overrides WHERE id = ANY($1)`
	result, err := s.db.ExecContext(ctx, query, pq.Array(ids))
	if err != nil {
		return 0, fmt.Errorf("delete overrides by id: %w", err)
	}

	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// Helper functions

func (s *ScenarioOverrideStore) scanOverrideRows(rows *sql.Rows) ([]ScenarioSecretOverride, error) {
	var overrides []ScenarioSecretOverride
	for rows.Next() {
		override, err := s.scanOverrideFromRows(rows)
		if err != nil {
			return nil, err
		}
		overrides = append(overrides, *override)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate overrides: %w", err)
	}
	return overrides, nil
}

func (s *ScenarioOverrideStore) scanOverrideFromRows(rows *sql.Rows) (*ScenarioSecretOverride, error) {
	var override ScenarioSecretOverride
	var handlingStrategy, fallbackStrategy, promptLabel, promptDescription, overrideReason sql.NullString
	var requiresUserInput sql.NullBool
	var generatorJSON, bundleJSON []byte

	if err := rows.Scan(
		&override.ID,
		&override.ScenarioName,
		&override.ResourceSecretID,
		&override.ResourceName,
		&override.SecretKey,
		&override.Tier,
		&handlingStrategy,
		&fallbackStrategy,
		&requiresUserInput,
		&promptLabel,
		&promptDescription,
		&generatorJSON,
		&bundleJSON,
		&overrideReason,
		&override.CreatedAt,
		&override.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan override: %w", err)
	}

	if handlingStrategy.Valid {
		override.HandlingStrategy = &handlingStrategy.String
	}
	if fallbackStrategy.Valid {
		override.FallbackStrategy = &fallbackStrategy.String
	}
	if requiresUserInput.Valid {
		override.RequiresUserInput = &requiresUserInput.Bool
	}
	if promptLabel.Valid {
		override.PromptLabel = &promptLabel.String
	}
	if promptDescription.Valid {
		override.PromptDescription = &promptDescription.String
	}
	if overrideReason.Valid {
		override.OverrideReason = &overrideReason.String
	}
	override.GeneratorTemplate = generatorJSON
	override.BundleHints = bundleJSON

	return &override, nil
}

func (s *ScenarioOverrideStore) scanOverrideRow(row *sql.Row) (*ScenarioSecretOverride, error) {
	var override ScenarioSecretOverride
	var handlingStrategy, fallbackStrategy, promptLabel, promptDescription, overrideReason sql.NullString
	var requiresUserInput sql.NullBool
	var generatorJSON, bundleJSON []byte

	if err := row.Scan(
		&override.ID,
		&override.ScenarioName,
		&override.ResourceSecretID,
		&override.ResourceName,
		&override.SecretKey,
		&override.Tier,
		&handlingStrategy,
		&fallbackStrategy,
		&requiresUserInput,
		&promptLabel,
		&promptDescription,
		&generatorJSON,
		&bundleJSON,
		&overrideReason,
		&override.CreatedAt,
		&override.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if handlingStrategy.Valid {
		override.HandlingStrategy = &handlingStrategy.String
	}
	if fallbackStrategy.Valid {
		override.FallbackStrategy = &fallbackStrategy.String
	}
	if requiresUserInput.Valid {
		override.RequiresUserInput = &requiresUserInput.Bool
	}
	if promptLabel.Valid {
		override.PromptLabel = &promptLabel.String
	}
	if promptDescription.Valid {
		override.PromptDescription = &promptDescription.String
	}
	if overrideReason.Valid {
		override.OverrideReason = &overrideReason.String
	}
	override.GeneratorTemplate = generatorJSON
	override.BundleHints = bundleJSON

	return &override, nil
}

// determineOverriddenFields compares override values with resource defaults to find which fields are overridden.
func (s *ScenarioOverrideStore) determineOverriddenFields(ctx context.Context, scenario, tier, resource, secret string) []string {
	query := `
		SELECT
			sso.handling_strategy IS NOT NULL AND sso.handling_strategy IS DISTINCT FROM sds.handling_strategy as handling_overridden,
			sso.fallback_strategy IS NOT NULL AND sso.fallback_strategy IS DISTINCT FROM sds.fallback_strategy as fallback_overridden,
			sso.requires_user_input IS NOT NULL AND sso.requires_user_input IS DISTINCT FROM sds.requires_user_input as requires_overridden,
			sso.prompt_label IS NOT NULL AND sso.prompt_label IS DISTINCT FROM sds.prompt_label as label_overridden,
			sso.prompt_description IS NOT NULL AND sso.prompt_description IS DISTINCT FROM sds.prompt_description as desc_overridden
		FROM scenario_secret_strategy_overrides sso
		JOIN resource_secrets rs ON rs.id = sso.resource_secret_id
		LEFT JOIN secret_deployment_strategies sds ON sds.resource_secret_id = rs.id AND sds.tier = sso.tier
		WHERE sso.scenario_name = $1 AND sso.tier = $2 AND rs.resource_name = $3 AND rs.secret_key = $4
	`

	var handlingOverridden, fallbackOverridden, requiresOverridden, labelOverridden, descOverridden bool
	err := s.db.QueryRowContext(ctx, query, scenario, tier, resource, secret).Scan(
		&handlingOverridden, &fallbackOverridden, &requiresOverridden, &labelOverridden, &descOverridden,
	)
	if err != nil {
		return nil
	}

	var fields []string
	if handlingOverridden {
		fields = append(fields, "handling_strategy")
	}
	if fallbackOverridden {
		fields = append(fields, "fallback_strategy")
	}
	if requiresOverridden {
		fields = append(fields, "requires_user_input")
	}
	if labelOverridden {
		fields = append(fields, "prompt_label")
	}
	if descOverridden {
		fields = append(fields, "prompt_description")
	}
	return fields
}

// ValidateScenarioDependency checks if a scenario depends on a resource.
// Returns an error message if validation fails, empty string if valid.
func (s *ScenarioOverrideStore) ValidateScenarioDependency(ctx context.Context, scenario, resource string) string {
	// Check if scenario exists by using the scenario CLI
	scenarios, err := fetchScenarioList(ctx)
	if err != nil {
		// Can't fetch scenarios - allow the override (fail open for validation)
		if s.logger != nil {
			s.logger.Info("could not fetch scenarios for validation: %v", err)
		}
		return ""
	}

	scenarioExists := false
	for _, sc := range scenarios {
		if sc.Name == scenario {
			scenarioExists = true
			break
		}
	}
	if !scenarioExists {
		return fmt.Sprintf("scenario '%s' not found", scenario)
	}

	// Load scenario's service.json to check dependencies using a temporary resolver
	resolver := NewResourceResolver(nil, s.logger)
	resources := resolver.resolveScenarioResources(scenario)

	// If no resources defined in service.json, allow it (can't validate)
	if len(resources) == 0 {
		return ""
	}

	// Check if resource is in dependencies
	for _, dep := range resources {
		if dep == resource {
			return ""
		}
	}

	return fmt.Sprintf("scenario '%s' does not depend on resource '%s'", scenario, resource)
}
