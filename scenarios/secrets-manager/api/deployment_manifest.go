package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lib/pq"
)

type analyzerDeploymentReport struct {
	Scenario      string                       `json:"scenario"`
	ReportVersion int                          `json:"report_version"`
	GeneratedAt   time.Time                    `json:"generated_at"`
	Dependencies  []analyzerDependency         `json:"dependencies"`
	Aggregates    map[string]analyzerAggregate `json:"aggregates"`
}

type analyzerDependency struct {
	Name         string                         `json:"name"`
	Type         string                         `json:"type"`
	ResourceType string                         `json:"resource_type"`
	Requirements analyzerRequirement            `json:"requirements"`
	TierSupport  map[string]analyzerTierSupport `json:"tier_support"`
	Alternatives []string                       `json:"alternatives"`
	Source       string                         `json:"source"`
}

type analyzerRequirement struct {
	RAMMB      int    `json:"ram_mb"`
	DiskMB     int    `json:"disk_mb"`
	CPUCores   int    `json:"cpu_cores"`
	Network    string `json:"network"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
}

type analyzerTierSupport struct {
	Supported    bool     `json:"supported"`
	FitnessScore float64  `json:"fitness_score"`
	Notes        string   `json:"notes"`
	Reason       string   `json:"reason"`
	Alternatives []string `json:"alternatives"`
}

type analyzerAggregate struct {
	FitnessScore          float64                   `json:"fitness_score"`
	DependencyCount       int                       `json:"dependency_count"`
	BlockingDependencies  []string                  `json:"blocking_dependencies"`
	EstimatedRequirements analyzerRequirementTotals `json:"estimated_requirements"`
}

type analyzerRequirementTotals struct {
	RAMMB    int `json:"ram_mb"`
	DiskMB   int `json:"disk_mb"`
	CPUCores int `json:"cpu_cores"`
}

func generateDeploymentManifest(ctx context.Context, req DeploymentManifestRequest) (*DeploymentManifest, error) {
	scenario := strings.TrimSpace(req.Scenario)
	tier := strings.TrimSpace(strings.ToLower(req.Tier))
	if scenario == "" || tier == "" {
		return nil, fmt.Errorf("scenario and tier are required")
	}

	resources := dedupeStrings(req.Resources)
	scenarioResources := resolveScenarioResources(scenario)
	effectiveResources := resources
	if len(scenarioResources) > 0 {
		if len(effectiveResources) == 0 {
			effectiveResources = scenarioResources
		} else {
			if intersected := intersectStrings(effectiveResources, scenarioResources); len(intersected) > 0 {
				effectiveResources = intersected
			} else {
				effectiveResources = scenarioResources
			}
		}
	}
	analyzerReport := ensureAnalyzerDeploymentReport(ctx, scenario)

	if db == nil {
		return buildFallbackManifest(scenario, tier, effectiveResources), nil
	}

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
	filters := []string{}
	argPos := 2
	if len(effectiveResources) > 0 {
		filters = append(filters, fmt.Sprintf("rs.resource_name = ANY($%d)", argPos))
		args = append(args, pq.Array(effectiveResources))
		argPos++
	}
	if !req.IncludeOptional {
		filters = append(filters, "rs.required = TRUE")
	}
	if len(filters) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(filters, " AND "))
	}
	query = query + " ORDER BY rs.resource_name, rs.secret_key"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []DeploymentSecretEntry{}
	resourceSet := map[string]struct{}{}

	for rows.Next() {
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

		if err := rows.Scan(&secretID, &resourceName, &secretKey, &secretType, &validationPattern, &required, &description, &classification,
			&ownerTeam, &ownerContact, &handlingStrategy, &fallbackStrategy, &requiresUser,
			&promptLabel, &promptDesc, &generatorJSON, &bundleJSON, &tierMapJSON); err != nil {
			return nil, err
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

		if entry.HandlingStrategy == "" {
			entry.HandlingStrategy = "unspecified"
		}
		if entry.FallbackStrategy == "" {
			entry.FallbackStrategy = ""
		}
		if promptLabel != "" || promptDesc != "" {
			entry.Prompt = &PromptMetadata{Label: promptLabel, Description: promptDesc}
		}

		entries = append(entries, entry)
		resourceSet[resourceName] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no secrets discovered for manifest request")
	}

	resourcesList := make([]string, 0, len(resourceSet))
	for resource := range resourceSet {
		resourcesList = append(resourcesList, resource)
	}
	if analyzerReport != nil {
		resourcesList = mergeResourceLists(resourcesList, analyzerResourceNames(analyzerReport))
	}
	sort.Strings(resourcesList)

	classificationTotals := map[string]int{}
	classificationReady := map[string]int{}
	strategyBreakdown := map[string]int{}
	blockingSecrets := []string{}

	for _, entry := range entries {
		classificationTotals[entry.Classification]++
		if entry.HandlingStrategy != "unspecified" {
			classificationReady[entry.Classification]++
			strategyBreakdown[entry.HandlingStrategy]++
		} else {
			blockingSecrets = append(blockingSecrets, fmt.Sprintf("%s:%s", entry.ResourceName, entry.SecretKey))
		}
	}

	if len(blockingSecrets) > 10 {
		blockingSecrets = blockingSecrets[:10]
	}

	summary := DeploymentSummary{
		TotalSecrets:          len(entries),
		StrategizedSecrets:    len(entries) - len(blockingSecrets),
		RequiresAction:        len(blockingSecrets),
		BlockingSecrets:       blockingSecrets,
		ClassificationWeights: classificationTotals,
		StrategyBreakdown:     strategyBreakdown,
		ScopeReadiness:        map[string]string{},
	}

	for class, total := range classificationTotals {
		ready := classificationReady[class]
		summary.ScopeReadiness[class] = fmt.Sprintf("%d/%d", ready, total)
	}

	manifest := &DeploymentManifest{
		Scenario:    scenario,
		Tier:        tier,
		GeneratedAt: time.Now(),
		Resources:   resourcesList,
		Secrets:     entries,
		Summary:     summary,
	}

	if analyzerReport != nil {
		manifest.Dependencies = convertAnalyzerDependencies(analyzerReport)
		manifest.TierAggregates = convertAnalyzerAggregates(analyzerReport)
		reportTime := analyzerReport.GeneratedAt
		manifest.AnalyzerGeneratedAt = &reportTime
	}

	manifest.BundleSecrets = deriveBundleSecretPlans(entries)

	if payload, err := json.Marshal(manifest); err == nil {
		if _, insertErr := db.ExecContext(ctx, `INSERT INTO deployment_manifests (scenario_name, tier, manifest) VALUES ($1, $2, $3)`, scenario, tier, payload); insertErr != nil {
			logger.Info("failed to persist deployment manifest telemetry: %v", insertErr)
		}
	} else {
		logger.Info("failed to marshal deployment manifest for telemetry: %v", err)
	}

	return manifest, nil
}

func buildFallbackManifest(scenario, tier string, resources []string) *DeploymentManifest {
	defaultResources := resources
	if len(defaultResources) == 0 {
		defaultResources = []string{"core-platform"}
	}
	entries := make([]DeploymentSecretEntry, 0, len(defaultResources))
	for idx, resource := range defaultResources {
		entries = append(entries, DeploymentSecretEntry{
			ResourceName:      resource,
			SecretKey:         fmt.Sprintf("PLACEHOLDER_SECRET_%d", idx+1),
			SecretType:        "token",
			Required:          true,
			Classification:    "service",
			Description:       "Fallback manifest entry (no database connection)",
			HandlingStrategy:  "prompt",
			RequiresUserInput: true,
			Prompt: &PromptMetadata{
				Label:       "Provide secret",
				Description: "Enter the secure value manually",
			},
			TierStrategies: map[string]string{
				tier: "prompt",
			},
		})
	}
	summary := DeploymentSummary{
		TotalSecrets:          len(entries),
		StrategizedSecrets:    len(entries),
		RequiresAction:        0,
		BlockingSecrets:       []string{},
		ClassificationWeights: map[string]int{"service": len(entries)},
		StrategyBreakdown:     map[string]int{"prompt": len(entries)},
		ScopeReadiness:        map[string]string{"service": fmt.Sprintf("%d/%d", len(entries), len(entries))},
	}
	manifest := &DeploymentManifest{
		Scenario:    scenario,
		Tier:        tier,
		GeneratedAt: time.Now(),
		Resources:   defaultResources,
		Secrets:     entries,
		Summary:     summary,
	}
	manifest.BundleSecrets = deriveBundleSecretPlans(entries)
	return manifest
}

func deriveBundleSecretPlans(entries []DeploymentSecretEntry) []BundleSecretPlan {
	plans := make([]BundleSecretPlan, 0, len(entries))
	for _, entry := range entries {
		if plan := bundleSecretFromEntry(entry); plan != nil {
			plans = append(plans, *plan)
		}
	}
	return plans
}

func bundleSecretFromEntry(entry DeploymentSecretEntry) *BundleSecretPlan {
	class := deriveSecretClass(entry)
	// Do not emit infrastructure secrets into bundle plans
	if class == "infrastructure" {
		return nil
	}

	plan := &BundleSecretPlan{
		ID:          deriveSecretID(entry),
		Class:       class,
		Required:    entry.Required,
		Description: entry.Description,
		Format:      entry.ValidationPattern,
		Target:      deriveBundleTarget(entry),
	}

	if class == "user_prompt" {
		plan.Prompt = derivePrompt(entry)
	}

	if class == "per_install_generated" {
		if entry.GeneratorTemplate != nil && len(entry.GeneratorTemplate) > 0 {
			plan.Generator = entry.GeneratorTemplate
		} else {
			plan.Generator = map[string]interface{}{
				"type":    "random",
				"length":  32,
				"charset": "alnum",
			}
		}
	}

	return plan
}

func deriveSecretClass(entry DeploymentSecretEntry) string {
	if strings.EqualFold(entry.Classification, "infrastructure") {
		return "infrastructure"
	}

	switch strings.ToLower(entry.HandlingStrategy) {
	case "prompt":
		return "user_prompt"
	case "generate":
		return "per_install_generated"
	case "delegate":
		return "remote_fetch"
	case "strip":
		return "infrastructure"
	}

	if entry.RequiresUserInput {
		return "user_prompt"
	}

	return "per_install_generated"
}

func deriveSecretID(entry DeploymentSecretEntry) string {
	if strings.TrimSpace(entry.ID) != "" {
		return entry.ID
	}
	key := strings.TrimSpace(entry.SecretKey)
	resource := strings.TrimSpace(entry.ResourceName)
	if key == "" && resource == "" {
		return "secret"
	}
	if resource == "" {
		return strings.ToLower(key)
	}
	if key == "" {
		return strings.ToLower(resource)
	}
	return strings.ToLower(fmt.Sprintf("%s_%s", resource, key))
}

func deriveBundleTarget(entry DeploymentSecretEntry) BundleSecretTarget {
	targetType := "env"
	targetName := strings.ToUpper(entry.SecretKey)

	if entry.BundleHints != nil {
		if v, ok := entry.BundleHints["target_type"].(string); ok && strings.TrimSpace(v) != "" {
			targetType = strings.TrimSpace(v)
		}
		if v, ok := entry.BundleHints["target_name"].(string); ok && strings.TrimSpace(v) != "" {
			targetName = strings.TrimSpace(v)
		} else if v, ok := entry.BundleHints["file_path"].(string); ok && strings.TrimSpace(v) != "" {
			targetType = "file"
			targetName = strings.TrimSpace(v)
		}
	}

	return BundleSecretTarget{
		Type: targetType,
		Name: targetName,
	}
}

func derivePrompt(entry DeploymentSecretEntry) *PromptMetadata {
	if entry.Prompt != nil && (strings.TrimSpace(entry.Prompt.Label) != "" || strings.TrimSpace(entry.Prompt.Description) != "") {
		return entry.Prompt
	}

	label := strings.TrimSpace(entry.SecretKey)
	if label == "" {
		label = "Provide secret"
	}
	description := entry.Description
	if strings.TrimSpace(description) == "" {
		description = "Enter the value for this bundled secret."
	}
	return &PromptMetadata{
		Label:       label,
		Description: description,
	}
}

func loadAnalyzerDeploymentReport(scenario string) (*analyzerDeploymentReport, error) {
	if scenario == "" {
		return nil, fmt.Errorf("scenario required")
	}
	root := os.Getenv("VROOLI_ROOT")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		root = filepath.Join(home, "Vrooli")
	}
	reportPath := filepath.Join(root, "scenarios", scenario, ".vrooli", "deployment", "deployment-report.json")
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

func persistAnalyzerDeploymentReport(scenario string, report *analyzerDeploymentReport) error {
	if scenario == "" || report == nil {
		return fmt.Errorf("scenario and report required")
	}
	root := os.Getenv("VROOLI_ROOT")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		root = filepath.Join(home, "Vrooli")
	}
	reportDir := filepath.Join(root, "scenarios", scenario, ".vrooli", "deployment")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(reportDir, "deployment-report.json")
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, encoded, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func ensureAnalyzerDeploymentReport(ctx context.Context, scenario string) *analyzerDeploymentReport {
	if scenario == "" {
		return nil
	}
	report, err := loadAnalyzerDeploymentReport(scenario)
	if err == nil && report != nil {
		// Check if report is stale (> 24 hours old)
		age := time.Since(report.GeneratedAt)
		if age > 24*time.Hour {
			logger.Info("analyzer report for %s is %v old, attempting refresh", scenario, age.Round(time.Hour))
			// Try to refresh but don't fail if unavailable
			if freshReport, fetchErr := fetchAnalyzerReportViaService(ctx, scenario); fetchErr == nil && freshReport != nil {
				if persistErr := persistAnalyzerDeploymentReport(scenario, freshReport); persistErr == nil {
					return freshReport
				}
			}
			// Return stale report if refresh fails
			logger.Info("using stale analyzer report for %s (refresh failed)", scenario)
		}
		return report
	}
	remoteReport, fetchErr := fetchAnalyzerReportViaService(ctx, scenario)
	if fetchErr != nil {
		logger.Info("deployment analyzer fallback failed for %s: %v", scenario, fetchErr)
		return nil
	}
	if remoteReport == nil {
		return nil
	}
	if persistErr := persistAnalyzerDeploymentReport(scenario, remoteReport); persistErr != nil {
		logger.Info("failed to persist analyzer report for %s: %v", scenario, persistErr)
	}
	return remoteReport
}

func fetchAnalyzerReportViaService(ctx context.Context, scenario string) (*analyzerDeploymentReport, error) {
	port, err := discoverAnalyzerPort(ctx)
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

func discoverAnalyzerPort(ctx context.Context) (string, error) {
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

func analyzerResourceNames(report *analyzerDeploymentReport) []string {
	if report == nil {
		return nil
	}
	names := []string{}
	for _, dep := range report.Dependencies {
		if dep.Type == "resource" && dep.Name != "" {
			names = append(names, dep.Name)
		}
	}
	return names
}

func mergeResourceLists(base []string, extras []string) []string {
	set := map[string]struct{}{}
	for _, item := range base {
		if item == "" {
			continue
		}
		set[item] = struct{}{}
	}
	for _, item := range extras {
		if item == "" {
			continue
		}
		set[item] = struct{}{}
	}
	merged := make([]string, 0, len(set))
	for key := range set {
		merged = append(merged, key)
	}
	return merged
}

func convertAnalyzerDependencies(report *analyzerDeploymentReport) []DependencyInsight {
	if report == nil {
		return nil
	}
	insights := make([]DependencyInsight, 0, len(report.Dependencies))
	for _, dep := range report.Dependencies {
		insight := DependencyInsight{
			Name:         dep.Name,
			Kind:         dep.Type,
			ResourceType: dep.ResourceType,
			Source:       dep.Source,
			Alternatives: dep.Alternatives,
		}
		if dep.Requirements.RAMMB != 0 || dep.Requirements.DiskMB != 0 || dep.Requirements.CPUCores != 0 || dep.Requirements.Network != "" {
			insight.Requirements = &DependencyRequirementSummary{
				RAMMB:      dep.Requirements.RAMMB,
				DiskMB:     dep.Requirements.DiskMB,
				CPUCores:   dep.Requirements.CPUCores,
				Network:    dep.Requirements.Network,
				Source:     dep.Requirements.Source,
				Confidence: dep.Requirements.Confidence,
			}
		}
		if len(dep.TierSupport) > 0 {
			insight.TierSupport = map[string]DependencyTierSupportView{}
			for tier, support := range dep.TierSupport {
				insight.TierSupport[tier] = DependencyTierSupportView{
					Supported:    support.Supported,
					FitnessScore: support.FitnessScore,
					Notes:        support.Notes,
					Reason:       support.Reason,
					Alternatives: support.Alternatives,
				}
			}
		}
		insights = append(insights, insight)
	}
	return insights
}

func convertAnalyzerAggregates(report *analyzerDeploymentReport) map[string]TierAggregateView {
	if report == nil || len(report.Aggregates) == 0 {
		return nil
	}
	result := make(map[string]TierAggregateView, len(report.Aggregates))
	for tier, aggregate := range report.Aggregates {
		view := TierAggregateView{
			FitnessScore:         aggregate.FitnessScore,
			DependencyCount:      aggregate.DependencyCount,
			BlockingDependencies: aggregate.BlockingDependencies,
		}
		if aggregate.EstimatedRequirements.RAMMB != 0 || aggregate.EstimatedRequirements.DiskMB != 0 || aggregate.EstimatedRequirements.CPUCores != 0 {
			view.EstimatedRequirements = &DependencyRequirementSummary{
				RAMMB:    aggregate.EstimatedRequirements.RAMMB,
				DiskMB:   aggregate.EstimatedRequirements.DiskMB,
				CPUCores: aggregate.EstimatedRequirements.CPUCores,
			}
		}
		result[tier] = view
	}
	return result
}

func resolveScenarioResources(scenario string) []string {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil
	}
	root := os.Getenv("VROOLI_ROOT")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		root = filepath.Join(home, "Vrooli")
	}
	servicePath := filepath.Join(root, "scenarios", scenario, ".vrooli", "service.json")
	data, err := os.ReadFile(servicePath)
	if err != nil {
		return nil
	}
	var doc struct {
		Service struct {
			Dependencies struct {
				Resources map[string]json.RawMessage `json:"resources"`
			} `json:"dependencies"`
		} `json:"service"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil
	}
	if len(doc.Service.Dependencies.Resources) == 0 {
		return nil
	}
	resources := make([]string, 0, len(doc.Service.Dependencies.Resources))
	for name := range doc.Service.Dependencies.Resources {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		resources = append(resources, name)
	}
	return dedupeStrings(resources)
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func intersectStrings(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	set := map[string]struct{}{}
	for _, value := range b {
		set[value] = struct{}{}
	}
	result := []string{}
	seen := map[string]struct{}{}
	for _, candidate := range a {
		if _, ok := set[candidate]; ok {
			if _, dup := seen[candidate]; dup {
				continue
			}
			seen[candidate] = struct{}{}
			result = append(result, candidate)
		}
	}
	if len(result) == 0 {
		return nil
	}
	sort.Strings(result)
	return result
}

func decodeJSONMap(payload []byte) map[string]interface{} {
	if len(payload) == 0 || string(payload) == "null" {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil
	}
	return result
}

func decodeStringMap(payload []byte) map[string]string {
	if len(payload) == 0 || string(payload) == "null" {
		return map[string]string{}
	}
	var result map[string]string
	if err := json.Unmarshal(payload, &result); err != nil {
		return map[string]string{}
	}
	return result
}
