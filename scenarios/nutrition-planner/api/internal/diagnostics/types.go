// Package diagnostics provides honest, scoped operational and data-health
// reporting. It reports actionable findings rather than a single global score.
package diagnostics

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type SQLExecutor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type ProviderStatus struct {
	Name, State, Reason string `json:"name,state,reason"`
}
type Finding struct {
	Code, Severity  string
	Count           int    `json:"count"`
	Message, Action string `json:"message,action"`
}
type Report struct {
	WorkspaceID     string                `json:"workspaceId"`
	GeneratedAt     time.Time             `json:"generatedAt"`
	SchemaVersion   int64                 `json:"schemaVersion"`
	DatabaseOK      bool                  `json:"databaseOk"`
	Providers       []ProviderStatus      `json:"providers"`
	ImageGeneration ImageGenerationStatus `json:"imageGeneration"`
	Findings        []Finding             `json:"findings"`
}

// ImageGenerationStatus keeps readiness, user permission, and dispatch policy
// separate. A ready Image Tools model is not a cost quote or permission to run.
type ImageGenerationStatus struct {
	Capability      string `json:"capability"`
	Reason          string `json:"reason"`
	Permission      string `json:"permission"`
	DispatchAllowed bool   `json:"dispatchAllowed"`
	QuoteAvailable  bool   `json:"quoteAvailable"`
}

func ImageGeneration(capability, reason string) ImageGenerationStatus {
	if capability != "available" && capability != "unavailable" {
		capability = "unknown"
	}
	if reason == "" {
		reason = "A reliable pre-dispatch cost upper bound is unavailable."
	}
	return ImageGenerationStatus{Capability: capability, Reason: reason, Permission: "off", DispatchAllowed: false, QuoteAvailable: false}
}

// ImageGenerationFromCandidates classifies only successful Image Tools reads.
// An absent service or failed read cannot prove that capability is unavailable.
func ImageGenerationFromCandidates(serviceFound, readSucceeded bool, readyStates []string) ImageGenerationStatus {
	if !serviceFound {
		return ImageGeneration("unknown", "Image Tools service is unavailable or not discoverable.")
	}
	if !readSucceeded {
		return ImageGeneration("unknown", "Image Tools capability could not be read.")
	}
	if len(readyStates) == 0 {
		return ImageGeneration("unavailable", "Image Tools reports no text-to-image model candidates; no pre-dispatch cost quote is available.")
	}

	unknownState := false
	for _, state := range readyStates {
		if state == "ready" {
			return ImageGeneration("available", "Image Tools reports a ready local text-to-image model; no pre-dispatch cost quote is available.")
		}
		if !knownNonReadyState(state) {
			unknownState = true
		}
	}
	if unknownState {
		return ImageGeneration("unknown", "Image Tools returned an unrecognized readiness state; no pre-dispatch cost quote is available.")
	}
	return ImageGeneration("unavailable", "Image Tools reports no ready local text-to-image model; no pre-dispatch cost quote is available.")
}

func knownNonReadyState(state string) bool {
	switch state {
	case "needs_model_install", "needs_backend", "needs_backend_manual", "needs_both", "insufficient", "unsupported", "disabled", "derived_pipeline_unproven", "env_not_provisioned", "smoke_failed":
		return true
	default:
		return false
	}
}

func Build(ctx context.Context, db SQLExecutor, workspaceID string, now time.Time, providers []ProviderStatus) (Report, error) {
	report := Report{WorkspaceID: workspaceID, GeneratedAt: now.UTC(), DatabaseOK: true, Providers: providers, ImageGeneration: ImageGeneration("unknown", "Image Tools capability has not been checked."), Findings: []Finding{}}
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&report.SchemaVersion); err != nil {
		return Report{}, err
	}
	failed, err := count(ctx, db, `SELECT COUNT(*) FROM jobs WHERE workspace_id=? AND state=?`, workspaceID, "failed")
	if err != nil {
		return Report{}, err
	}
	if failed > 0 {
		report.Findings = append(report.Findings, Finding{Code: "failed_jobs", Severity: "warning", Count: failed, Message: "Some optional work failed and needs review.", Action: "Review failed jobs before retrying."})
	}
	cutoff := now.UTC().Add(-30 * 24 * time.Hour).Format(time.RFC3339Nano)
	oldPrices, err := count(ctx, db, `SELECT COUNT(*) FROM price_observations WHERE workspace_id=? AND observed_at<?`, workspaceID, cutoff)
	if err != nil {
		return Report{}, err
	}
	if oldPrices > 0 {
		report.Findings = append(report.Findings, Finding{Code: "old_prices", Severity: "info", Count: oldPrices, Message: "Some price observations are older than 30 days.", Action: "Refresh or confirm prices before relying on checkout estimates."})
	}
	unknownTargets, err := count(ctx, db, `SELECT COUNT(*) FROM nutrition_targets WHERE workspace_id=? AND active=1 AND lower_bound='' AND upper_bound=''`, workspaceID)
	if err != nil {
		return Report{}, err
	}
	if unknownTargets > 0 {
		report.Findings = append(report.Findings, Finding{Code: "unknown_targets", Severity: "warning", Count: unknownTargets, Message: "Some active nutrient targets have no known bound.", Action: "Review target values before treating coverage as complete."})
	}
	unresolvedRecipes, err := unresolvedRecipeCount(ctx, db, workspaceID)
	if err != nil {
		return Report{}, err
	}
	if unresolvedRecipes > 0 {
		report.Findings = append(report.Findings, Finding{Code: "unresolved_ingredients", Severity: "warning", Count: unresolvedRecipes, Message: "Some recipes contain unresolved or unknown ingredient data.", Action: "Review ingredient mappings before using exact nutrition or cost totals."})
	}
	unresolvedPlan, err := unresolvedPlanCount(ctx, db, workspaceID)
	if err != nil {
		return Report{}, err
	}
	if unresolvedPlan > 0 {
		report.Findings = append(report.Findings, Finding{Code: "incomplete_coverage", Severity: "warning", Count: unresolvedPlan, Message: "The current plan contains unresolved slots or evaluations.", Action: "Open the current plan and resolve the highest-priority gap."})
	}
	return report, nil
}

func count(ctx context.Context, db SQLExecutor, query string, args ...any) (int, error) {
	var value int
	err := db.QueryRowContext(ctx, query, args...).Scan(&value)
	return value, err
}

func unresolvedRecipeCount(ctx context.Context, db SQLExecutor, workspaceID string) (int, error) {
	rows, err := db.QueryContext(ctx, `SELECT groups_json FROM recipe_revisions WHERE workspace_id=?`, workspaceID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return 0, err
		}
		var value any
		if json.Unmarshal([]byte(raw), &value) == nil && containsUnknown(value) {
			count++
		}
	}
	return count, rows.Err()
}

func unresolvedPlanCount(ctx context.Context, db SQLExecutor, workspaceID string) (int, error) {
	var raw string
	err := db.QueryRowContext(ctx, `SELECT unresolved_json FROM plan_state WHERE workspace_id=?`, workspaceID).Scan(&raw)
	if err != nil {
		// No current plan is a normal empty-workspace state, not a diagnostic failure.
		return 0, nil
	}
	var unresolved []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &unresolved); err != nil {
		return 0, err
	}
	return len(unresolved), nil
}

func containsUnknown(value any) bool {
	switch v := value.(type) {
	case string:
		return v == "unknown" || v == "unresolved" || v == "needs_information"
	case []any:
		for _, item := range v {
			if containsUnknown(item) {
				return true
			}
		}
	case map[string]any:
		for key, item := range v {
			if key == "unresolved" || key == "unknown" || containsUnknown(item) {
				return true
			}
		}
	}
	return false
}
