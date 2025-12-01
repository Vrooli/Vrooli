package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
)

type OrientationSummary struct {
	HeroStats             HeroStats                `json:"hero_stats"`
	Journeys              []JourneyCard            `json:"journeys"`
	TierReadiness         []TierReadiness          `json:"tier_readiness"`
	ResourceInsights      []ResourceInsight        `json:"resource_insights"`
	VulnerabilityInsights []VulnerabilityHighlight `json:"vulnerability_insights"`
	UpdatedAt             time.Time                `json:"updated_at"`
}

type HeroStats struct {
	VaultConfigured int     `json:"vault_configured"`
	VaultTotal      int     `json:"vault_total"`
	MissingSecrets  int     `json:"missing_secrets"`
	RiskScore       int     `json:"risk_score"`
	OverallScore    int     `json:"overall_score"`
	LastScan        string  `json:"last_scan"`
	ReadinessLabel  string  `json:"readiness_label"`
	Confidence      float64 `json:"confidence"`
}

type JourneyCard struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	CtaLabel    string   `json:"cta_label"`
	CtaAction   string   `json:"cta_action"`
	Primers     []string `json:"primers"`
	Badge       string   `json:"badge"`
}

type TierReadiness struct {
	Tier                 string   `json:"tier"`
	Label                string   `json:"label"`
	Strategized          int      `json:"strategized"`
	Total                int      `json:"total"`
	ReadyPercent         int      `json:"ready_percent"`
	BlockingSecretSample []string `json:"blocking_secret_sample"`
}

type ResourceInsight struct {
	ResourceName   string                  `json:"resource_name"`
	TotalSecrets   int                     `json:"total_secrets"`
	ValidSecrets   int                     `json:"valid_secrets"`
	MissingSecrets int                     `json:"missing_secrets"`
	InvalidSecrets int                     `json:"invalid_secrets"`
	LastValidation *time.Time              `json:"last_validation"`
	Secrets        []ResourceSecretInsight `json:"secrets"`
}

type ResourceSecretInsight struct {
	SecretKey      string            `json:"secret_key"`
	SecretType     string            `json:"secret_type"`
	Classification string            `json:"classification"`
	Required       bool              `json:"required"`
	TierStrategies map[string]string `json:"tier_strategies"`
}

type VulnerabilityHighlight struct {
	Severity string `json:"severity"`
	Count    int    `json:"count"`
	Message  string `json:"message"`
}

type OrientationHandlers struct {
	builder *OrientationBuilder
}

func NewOrientationHandlers(builder *OrientationBuilder) *OrientationHandlers {
	return &OrientationHandlers{builder: builder}
}

func (s *APIServer) orientationSummaryHandler(w http.ResponseWriter, r *http.Request) {
	s.handlers.orientation.Summary(w, r)
}

func (h *OrientationHandlers) Summary(w http.ResponseWriter, r *http.Request) {
	if h.builder == nil {
		http.Error(w, "orientation summary unavailable: database not initialized", http.StatusServiceUnavailable)
		return
	}

	summary, err := h.builder.Build(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to build orientation summary: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summary)
}

type OrientationBuilder struct {
	db                    *sql.DB
	logger                *Logger
	vaultStatusFn         func(string) (*VaultSecretsStatus, error)
	vulnerabilityScanFunc func(string, string, string) (*SecurityScanResult, error)
}

func NewOrientationBuilder(db *sql.DB, logger *Logger) *OrientationBuilder {
	return &OrientationBuilder{
		db:                    db,
		logger:                logger,
		vaultStatusFn:         getVaultSecretsStatus,
		vulnerabilityScanFunc: scanComponentsForVulnerabilities,
	}
}

func (b *OrientationBuilder) Build(ctx context.Context) (*OrientationSummary, error) {
	if b.db == nil {
		return nil, fmt.Errorf("database not initialized for orientation summary")
	}

	vaultStatus, err := b.vaultStatusFn("")
	if err != nil {
		b.logger.Info("orientation vault status fallback: %v", err)
		vaultStatus = &VaultSecretsStatus{}
	}

	securityResult, err := b.vulnerabilityScanFunc("", "", "")
	if err != nil {
		b.logger.Info("orientation scan fallback: %v", err)
		securityResult = &SecurityScanResult{
			ScanID:          uuid.New().String(),
			Vulnerabilities: []SecurityVulnerability{},
			RiskScore:       0,
		}
	}

	tierReadiness, err := b.fetchTierReadiness(ctx)
	if err != nil {
		return nil, err
	}

	resourceInsights, err := b.fetchResourceInsights(ctx)
	if err != nil {
		return nil, err
	}

	return &OrientationSummary{
		HeroStats:             b.buildHeroStats(vaultStatus, securityResult, tierReadiness),
		Journeys:              b.buildJourneyCards(vaultStatus, securityResult, tierReadiness),
		TierReadiness:         tierReadiness,
		ResourceInsights:      resourceInsights,
		VulnerabilityInsights: b.buildVulnerabilityHighlights(securityResult),
		UpdatedAt:             time.Now(),
	}, nil
}

func (b *OrientationBuilder) fetchTierReadiness(ctx context.Context) ([]TierReadiness, error) {
	var totalRequired int
	if err := b.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM resource_secrets WHERE required = TRUE`).Scan(&totalRequired); err != nil {
		return nil, err
	}

	readiness := make([]TierReadiness, 0, len(deploymentTierCatalog))

	for _, tier := range deploymentTierCatalog {
		var strategized int
		if err := b.db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM resource_secrets rs
			WHERE rs.required = TRUE AND EXISTS (
				SELECT 1 FROM secret_deployment_strategies sds
				WHERE sds.resource_secret_id = rs.id AND sds.tier = $1
			)
		`, tier.Name).Scan(&strategized); err != nil {
			return nil, err
		}

		missingRows, err := b.db.QueryContext(ctx, `
			SELECT rs.resource_name, rs.secret_key
			FROM resource_secrets rs
			WHERE rs.required = TRUE AND NOT EXISTS (
				SELECT 1 FROM secret_deployment_strategies sds
				WHERE sds.resource_secret_id = rs.id AND sds.tier = $1
			)
			ORDER BY rs.resource_name, rs.secret_key
			LIMIT 5
		`, tier.Name)
		if err != nil {
			return nil, err
		}
		var blockers []string
		for missingRows.Next() {
			var rName, sKey string
			if err := missingRows.Scan(&rName, &sKey); err != nil {
				missingRows.Close()
				return nil, err
			}
			blockers = append(blockers, fmt.Sprintf("%s:%s", rName, sKey))
		}
		missingRows.Close()

		readyPercent := 0
		if totalRequired == 0 {
			readyPercent = 100
		} else {
			readyPercent = int(math.Round((float64(strategized) / float64(totalRequired)) * 100))
		}

		readiness = append(readiness, TierReadiness{
			Tier:                 tier.Name,
			Label:                tier.Label,
			Strategized:          strategized,
			Total:                totalRequired,
			ReadyPercent:         readyPercent,
			BlockingSecretSample: blockers,
		})
	}

	return readiness, nil
}

func (b *OrientationBuilder) fetchResourceInsights(ctx context.Context) ([]ResourceInsight, error) {
	rows, err := b.db.QueryContext(ctx, `
		SELECT resource_name, total_secrets, required_secrets, valid_secrets, missing_required_secrets, invalid_secrets, last_validation
		FROM secret_health_summary
		ORDER BY resource_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := []SecretHealthSummary{}
	for rows.Next() {
		var summary SecretHealthSummary
		if err := rows.Scan(&summary.ResourceName, &summary.TotalSecrets, &summary.RequiredSecrets,
			&summary.ValidSecrets, &summary.MissingRequiredSecrets, &summary.InvalidSecrets, &summary.LastValidation); err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	secretRows, err := b.db.QueryContext(ctx, `
		SELECT rs.resource_name,
		       rs.secret_key,
		       rs.secret_type,
		       COALESCE(rs.classification, 'service') as classification,
		       rs.required,
		       COALESCE(tiers.tier_map, '{}'::jsonb) as tier_map
		FROM resource_secrets rs
		LEFT JOIN (
			SELECT resource_secret_id, jsonb_object_agg(tier, handling_strategy) AS tier_map
			FROM secret_deployment_strategies
			GROUP BY resource_secret_id
		) tiers ON tiers.resource_secret_id = rs.id
		ORDER BY rs.resource_name, rs.secret_key
	`)
	if err != nil {
		return nil, err
	}
	defer secretRows.Close()

	resourceSecretMap := map[string][]ResourceSecretInsight{}
	for secretRows.Next() {
		var (
			resourceName   string
			secretKey      string
			secretType     string
			classification string
			required       bool
			tierMapJSON    []byte
		)
		if err := secretRows.Scan(&resourceName, &secretKey, &secretType, &classification, &required, &tierMapJSON); err != nil {
			return nil, err
		}
		insight := ResourceSecretInsight{
			SecretKey:      secretKey,
			SecretType:     secretType,
			Classification: classification,
			Required:       required,
			TierStrategies: decodeStringMap(tierMapJSON),
		}
		resourceSecretMap[resourceName] = append(resourceSecretMap[resourceName], insight)
	}
	if err := secretRows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].MissingRequiredSecrets == summaries[j].MissingRequiredSecrets {
			return summaries[i].ResourceName < summaries[j].ResourceName
		}
		return summaries[i].MissingRequiredSecrets > summaries[j].MissingRequiredSecrets
	})

	limit := 5
	if len(summaries) < limit {
		limit = len(summaries)
	}
	insights := make([]ResourceInsight, 0, limit)
	for idx := 0; idx < limit; idx++ {
		summary := summaries[idx]
		secrets := resourceSecretMap[summary.ResourceName]
		if len(secrets) > 6 {
			secrets = secrets[:6]
		}
		insights = append(insights, ResourceInsight{
			ResourceName:   summary.ResourceName,
			TotalSecrets:   summary.TotalSecrets,
			ValidSecrets:   summary.ValidSecrets,
			MissingSecrets: summary.MissingRequiredSecrets,
			InvalidSecrets: summary.InvalidSecrets,
			LastValidation: summary.LastValidation,
			Secrets:        secrets,
		})
	}

	return insights, nil
}

func (b *OrientationBuilder) buildHeroStats(vaultStatus *VaultSecretsStatus, security *SecurityScanResult, readiness []TierReadiness) HeroStats {
	hero := HeroStats{}
	if vaultStatus != nil {
		hero.VaultConfigured = vaultStatus.ConfiguredResources
		hero.VaultTotal = vaultStatus.TotalResources
		hero.MissingSecrets = len(vaultStatus.MissingSecrets)
		hero.LastScan = vaultStatus.LastUpdated.Format(time.RFC3339)
	}
	if security != nil {
		hero.RiskScore = security.RiskScore
		if hero.LastScan == "" {
			hero.LastScan = time.Now().Format(time.RFC3339)
		}
	}
	vaultHealth := 0
	if hero.VaultTotal > 0 {
		vaultHealth = (hero.VaultConfigured * 100) / hero.VaultTotal
	}
	securityScore := 100 - hero.RiskScore
	if securityScore < 0 {
		securityScore = 0
	}
	hero.OverallScore = (vaultHealth + securityScore) / 2
	bestReadiness := 0
	for _, tier := range readiness {
		if tier.ReadyPercent > bestReadiness {
			bestReadiness = tier.ReadyPercent
		}
	}
	hero.Confidence = math.Round((float64(bestReadiness)/100.0)*100) / 100
	switch {
	case hero.OverallScore >= 80:
		hero.ReadinessLabel = "Operational"
	case hero.OverallScore >= 50:
		hero.ReadinessLabel = "Stabilize"
	default:
		hero.ReadinessLabel = "Blocked"
	}
	return hero
}

func (b *OrientationBuilder) buildJourneyCards(vaultStatus *VaultSecretsStatus, security *SecurityScanResult, readiness []TierReadiness) []JourneyCard {
	missing := 0
	if vaultStatus != nil {
		missing = len(vaultStatus.MissingSecrets)
	}
	risk := 0
	if security != nil {
		risk = security.RiskScore
	}
	deploymentCoverage := 0
	for _, tier := range readiness {
		if tier.Tier == "tier-2-desktop" {
			deploymentCoverage = tier.ReadyPercent
			break
		}
	}
	journeys := []JourneyCard{
		{
			ID:          "orientation",
			Title:       "Orientation",
			Description: "Get familiar with your security posture and available workflows.",
			Status:      "steady",
			CtaLabel:    "Start Tour",
			CtaAction:   "open-orientation-flow",
			Primers:     []string{"Getting started"},
			Badge:       "Start",
		},
		{
			ID:          "configure-secrets",
			Title:       "Configure Secrets",
			Description: "Audit vault coverage and close the gap per resource with guided provisioning.",
			Status:      map[bool]string{true: "attention", false: "steady"}[missing > 0],
			CtaLabel:    "Start Coverage Flow",
			CtaAction:   "open-configure-flow",
			Primers:     []string{fmt.Sprintf("%d resources missing secrets", missing)},
			Badge:       "P0",
		},
		{
			ID:          "fix-vulnerabilities",
			Title:       "Address Vulnerabilities",
			Description: "Triage findings, review recommended fixes, and trigger agents for remediation.",
			Status:      map[bool]string{true: "attention", false: "steady"}[risk > 40],
			CtaLabel:    "Launch Security Flow",
			CtaAction:   "open-security-flow",
			Primers:     []string{fmt.Sprintf("Risk score %d", risk)},
			Badge:       "Security",
		},
		{
			ID:          "prep-deployment",
			Title:       "Prep Deployment",
			Description: "Select target tiers, review tier strategies, and emit bundle manifests.",
			Status:      map[bool]string{true: "attention", false: "steady"}[deploymentCoverage < 60],
			CtaLabel:    "Open Deployment Wizard",
			CtaAction:   "open-deployment-flow",
			Primers:     []string{fmt.Sprintf("Tier 2 coverage %d%%", deploymentCoverage)},
			Badge:       "Deploy",
		},
	}
	return journeys
}

func (b *OrientationBuilder) buildVulnerabilityHighlights(results *SecurityScanResult) []VulnerabilityHighlight {
	if results == nil {
		return []VulnerabilityHighlight{}
	}
	counts := map[string]int{}
	for _, vuln := range results.Vulnerabilities {
		counts[vuln.Severity]++
	}
	levels := []string{"critical", "high", "medium", "low"}
	highlights := []VulnerabilityHighlight{}
	for _, level := range levels {
		if counts[level] == 0 {
			continue
		}
		message := fmt.Sprintf("%d %s issues", counts[level], level)
		highlights = append(highlights, VulnerabilityHighlight{Severity: level, Count: counts[level], Message: message})
	}
	return highlights
}
