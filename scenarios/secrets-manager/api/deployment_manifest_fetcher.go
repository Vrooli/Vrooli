// Package main provides data fetching abstractions for deployment manifest generation.
//
// This file contains interfaces and implementations for:
//   - SecretStore: Database abstraction for fetching and persisting deployment secrets
//   - AnalyzerClient: HTTP client abstraction for the scenario-dependency-analyzer service
//
// Using interfaces enables dependency injection and testability via mocks.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
)

// -----------------------------------------------------------------------------
// Interfaces
// -----------------------------------------------------------------------------

// SecretStore abstracts database access for deployment secrets.
type SecretStore interface {
	// FetchSecrets retrieves secrets for the given resources and tier.
	FetchSecrets(ctx context.Context, tier string, resources []string, includeOptional bool) ([]DeploymentSecretEntry, error)

	// PersistManifest stores manifest telemetry for audit and debugging.
	PersistManifest(ctx context.Context, scenario, tier string, manifest *DeploymentManifest) error
}

// AnalyzerClient abstracts calls to the scenario-dependency-analyzer service.
type AnalyzerClient interface {
	// FetchDeploymentReport retrieves the analyzer report for a scenario.
	// Returns nil (without error) if the analyzer is unavailable.
	FetchDeploymentReport(ctx context.Context, scenario string) (*analyzerDeploymentReport, error)
}

// -----------------------------------------------------------------------------
// PostgresSecretStore Implementation
// -----------------------------------------------------------------------------

// PostgresSecretStore implements SecretStore using PostgreSQL.
type PostgresSecretStore struct {
	db     *sql.DB
	logger *Logger
}

// NewPostgresSecretStore creates a production SecretStore backed by PostgreSQL.
func NewPostgresSecretStore(db *sql.DB, logger *Logger) *PostgresSecretStore {
	return &PostgresSecretStore{db: db, logger: logger}
}

// FetchSecrets retrieves deployment secrets from the database for the specified
// tier and resources. It joins resource_secrets with secret_deployment_strategies
// to include tier-specific handling information.
func (s *PostgresSecretStore) FetchSecrets(ctx context.Context, tier string, resources []string, includeOptional bool) ([]DeploymentSecretEntry, error) {
	query := `
		SELECT
			rs.id,
			rs.resource_name,
			rs.secret_key,
			rs.secret_type,
			COALESCE(rs.validation_pattern, '') as validation_pattern,
			rs.required,
			COALESCE(rs.description, '') as description,
			COALESCE(rs.classification, 'service') as classification,
			COALESCE(rs.owner_team, '') as owner_team,
			COALESCE(rs.owner_contact, '') as owner_contact,
			COALESCE(sds.handling_strategy, '') as handling_strategy,
			COALESCE(sds.fallback_strategy, '') as fallback_strategy,
			COALESCE(sds.requires_user_input, false) as requires_user_input,
			COALESCE(sds.prompt_label, '') as prompt_label,
			COALESCE(sds.prompt_description, '') as prompt_description,
			sds.generator_template,
			sds.bundle_hints,
			COALESCE(tiers.tier_map, '{}'::jsonb) as tier_map
		FROM resource_secrets rs
		LEFT JOIN secret_deployment_strategies sds
			ON sds.resource_secret_id = rs.id AND sds.tier = $1
		LEFT JOIN (
			SELECT resource_secret_id, jsonb_object_agg(tier, handling_strategy) AS tier_map
			FROM secret_deployment_strategies
			GROUP BY resource_secret_id
		) tiers ON tiers.resource_secret_id = rs.id
	`

	args := []interface{}{tier}
	var filters []string
	argPos := 2

	if len(resources) > 0 {
		filters = append(filters, fmt.Sprintf("rs.resource_name = ANY($%d)", argPos))
		args = append(args, pq.Array(resources))
		argPos++
	}
	if !includeOptional {
		filters = append(filters, "rs.required = TRUE")
	}
	if len(filters) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(filters, " AND "))
	}
	query += " ORDER BY rs.resource_name, rs.secret_key"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query secrets: %w", err)
	}
	defer rows.Close()

	var entries []DeploymentSecretEntry
	for rows.Next() {
		entry, err := s.scanSecretRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan secret row: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate secrets: %w", err)
	}

	return entries, nil
}

// scanSecretRow extracts a DeploymentSecretEntry from a database row.
func (s *PostgresSecretStore) scanSecretRow(rows *sql.Rows) (DeploymentSecretEntry, error) {
	var (
		secretID          string
		resourceName      string
		secretKey         string
		secretType        string
		validationPattern string
		required          bool
		description       string
		classification    string
		ownerTeam         string
		ownerContact      string
		handlingStrategy  string
		fallbackStrategy  string
		requiresUser      bool
		promptLabel       string
		promptDesc        string
		generatorJSON     []byte
		bundleJSON        []byte
		tierMapJSON       []byte
	)

	if err := rows.Scan(
		&secretID, &resourceName, &secretKey, &secretType,
		&validationPattern, &required, &description, &classification,
		&ownerTeam, &ownerContact, &handlingStrategy, &fallbackStrategy,
		&requiresUser, &promptLabel, &promptDesc,
		&generatorJSON, &bundleJSON, &tierMapJSON,
	); err != nil {
		return DeploymentSecretEntry{}, err
	}

	entry := DeploymentSecretEntry{
		ID:                secretID,
		ResourceName:      resourceName,
		SecretKey:         secretKey,
		SecretType:        secretType,
		Required:          required,
		Classification:    classification,
		Description:       description,
		ValidationPattern: validationPattern,
		OwnerTeam:         ownerTeam,
		OwnerContact:      ownerContact,
		HandlingStrategy:  handlingStrategy,
		FallbackStrategy:  fallbackStrategy,
		RequiresUserInput: requiresUser,
		GeneratorTemplate: decodeJSONMap(generatorJSON),
		BundleHints:       decodeJSONMap(bundleJSON),
		TierStrategies:    decodeStringMap(tierMapJSON),
	}

	// Apply defaults for missing strategies
	if entry.HandlingStrategy == "" {
		entry.HandlingStrategy = "unspecified"
	}

	// Set prompt metadata if provided
	if promptLabel != "" || promptDesc != "" {
		entry.Prompt = &PromptMetadata{Label: promptLabel, Description: promptDesc}
	}

	return entry, nil
}

// PersistManifest stores the generated manifest as telemetry for audit purposes.
func (s *PostgresSecretStore) PersistManifest(ctx context.Context, scenario, tier string, manifest *DeploymentManifest) error {
	payload, err := json.Marshal(manifest)
	if err != nil {
		if s.logger != nil {
			s.logger.Info("failed to marshal deployment manifest for telemetry: %v", err)
		}
		return err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO deployment_manifests (scenario_name, tier, manifest) VALUES ($1, $2, $3)`,
		scenario, tier, payload,
	)
	if err != nil {
		if s.logger != nil {
			s.logger.Info("failed to persist deployment manifest telemetry: %v", err)
		}
		return err
	}
	return nil
}

// -----------------------------------------------------------------------------
// HTTPAnalyzerClient Implementation
// -----------------------------------------------------------------------------

// HTTPAnalyzerClient implements AnalyzerClient using HTTP calls to the analyzer service.
type HTTPAnalyzerClient struct {
	logger      *Logger
	reportCache map[string]*analyzerReportCacheEntry
	cacheMu     sync.RWMutex
}

// analyzerReportCacheEntry caches analyzer reports with freshness tracking.
type analyzerReportCacheEntry struct {
	report    *analyzerDeploymentReport
	fetchedAt time.Time
}

// reportStalenessThreshold defines when cached reports should be refreshed.
const reportStalenessThreshold = 24 * time.Hour

// NewHTTPAnalyzerClient creates a production AnalyzerClient using HTTP.
func NewHTTPAnalyzerClient(logger *Logger) *HTTPAnalyzerClient {
	return &HTTPAnalyzerClient{
		logger:      logger,
		reportCache: make(map[string]*analyzerReportCacheEntry),
	}
}

// FetchDeploymentReport retrieves the analyzer report for a scenario.
// It first checks for a cached/persisted report, then falls back to HTTP.
// Stale reports (older than 24 hours) trigger a background refresh attempt.
func (c *HTTPAnalyzerClient) FetchDeploymentReport(ctx context.Context, scenario string) (*analyzerDeploymentReport, error) {
	if scenario == "" {
		return nil, nil
	}

	// Try to load from persisted file first
	report, err := c.loadPersistedReport(scenario)
	if err == nil && report != nil {
		age := time.Since(report.GeneratedAt)
		if age <= reportStalenessThreshold {
			return report, nil
		}

		// Report is stale, attempt refresh
		if c.logger != nil {
			c.logger.Info("analyzer report for %s is %v old, attempting refresh", scenario, age.Round(time.Hour))
		}

		freshReport, fetchErr := c.fetchFromService(ctx, scenario)
		if fetchErr == nil && freshReport != nil {
			if persistErr := c.persistReport(scenario, freshReport); persistErr == nil {
				return freshReport, nil
			}
		}

		// Return stale report if refresh fails
		if c.logger != nil {
			c.logger.Info("using stale analyzer report for %s (refresh failed)", scenario)
		}
		return report, nil
	}

	// No persisted report, fetch from service
	remoteReport, fetchErr := c.fetchFromService(ctx, scenario)
	if fetchErr != nil {
		if c.logger != nil {
			c.logger.Info("deployment analyzer fallback failed for %s: %v", scenario, fetchErr)
		}
		return nil, nil
	}

	if remoteReport == nil {
		return nil, nil
	}

	// Persist for future use
	if persistErr := c.persistReport(scenario, remoteReport); persistErr != nil && c.logger != nil {
		c.logger.Info("failed to persist analyzer report for %s: %v", scenario, persistErr)
	}

	return remoteReport, nil
}

// loadPersistedReport reads a previously saved analyzer report from disk.
func (c *HTTPAnalyzerClient) loadPersistedReport(scenario string) (*analyzerDeploymentReport, error) {
	reportPath := c.reportPath(scenario)
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, err
	}

	var report analyzerDeploymentReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

// persistReport saves an analyzer report to disk for caching.
func (c *HTTPAnalyzerClient) persistReport(scenario string, report *analyzerDeploymentReport) error {
	if scenario == "" || report == nil {
		return fmt.Errorf("scenario and report required")
	}

	reportDir := filepath.Dir(c.reportPath(scenario))
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return err
	}

	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	path := c.reportPath(scenario)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, encoded, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// reportPath returns the filesystem path for a scenario's analyzer report.
func (c *HTTPAnalyzerClient) reportPath(scenario string) string {
	root := os.Getenv("VROOLI_ROOT")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			root = "/tmp"
		} else {
			root = filepath.Join(home, "Vrooli")
		}
	}
	return filepath.Join(root, "scenarios", scenario, ".vrooli", "deployment", "deployment-report.json")
}

// fetchFromService makes an HTTP request to the analyzer service.
func (c *HTTPAnalyzerClient) fetchFromService(ctx context.Context, scenario string) (*analyzerDeploymentReport, error) {
	port, err := c.discoverAnalyzerPort(ctx)
	if err != nil {
		return nil, err
	}

	requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	url := fmt.Sprintf("http://localhost:%s/api/v1/scenarios/%s/deployment", port, scenario)
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("analyzer responded with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var report analyzerDeploymentReport
	if err := json.Unmarshal(body, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

// discoverAnalyzerPort finds the port where scenario-dependency-analyzer is running.
func (c *HTTPAnalyzerClient) discoverAnalyzerPort(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", "scenario-dependency-analyzer", "API_PORT")
	cmd.Env = os.Environ()
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to discover analyzer port: %w", err)
	}

	port := strings.TrimSpace(string(output))
	if port == "" {
		return "", fmt.Errorf("analyzer API port not available")
	}
	return port, nil
}
