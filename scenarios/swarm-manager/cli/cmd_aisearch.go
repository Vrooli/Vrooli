// AI search CLI commands — thin wrappers over /api/v1/search/ai endpoints.
//
// Human output follows the cli-steer contracts:
//   - status            → Operational (Status → Triage → Next Steps)
//   - search-ai         → Data Retrieval (Summary → Results → Retrieval Hints)
//   - reconcile         → Mutation (Result → What Changed → Next Command)
//   - reconcile-status  → Operational
//   - reconcile-cancel  → Mutation
//
// The --json flag is supported on every command for scripting.
//
// Greenfield rename (per the aisearch reconciler refactor): "reindex" → "reconcile".
// The new verb reflects diff-and-converge semantics, not rebuild-from-scratch.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// AISearchStatus mirrors aisearch.AvailabilityStatus from the API.
type AISearchStatus struct {
	Available          bool   `json:"available"`
	Ollama             bool   `json:"ollama"`
	Qdrant             bool   `json:"qdrant"`
	IndexedBacklog     int    `json:"indexedBacklog"`
	IndexedInitiatives int    `json:"indexedInitiatives"`
	OnDiskBacklog      int    `json:"onDiskBacklog"`
	OnDiskInitiatives  int    `json:"onDiskInitiatives"`
	Message            string `json:"message,omitempty"`
}

// AISearchReconcileStatus mirrors aisearch.ReconcileStatus.
type AISearchReconcileStatus struct {
	Running    bool                 `json:"running"`
	StartedAt  string               `json:"startedAt,omitempty"`
	FinishedAt string               `json:"finishedAt,omitempty"`
	LastPlan   *AISearchDriftReport `json:"lastPlan,omitempty"`
	LastResult *AISearchApplyResult `json:"lastResult,omitempty"`
	LastError  string               `json:"lastError,omitempty"`
	Canceled   bool                 `json:"canceled,omitempty"`
}

// AISearchDriftReport mirrors aisearch.DriftReport (the count-only projection
// that crosses the wire — ItemRef arrays are server-internal).
type AISearchDriftReport struct {
	PlannedAt           string   `json:"plannedAt"`
	ToDeleteBacklog     []string `json:"toDeleteBacklog,omitempty"`
	ToDeleteInitiative  []string `json:"toDeleteInitiative,omitempty"`
	UnchangedBacklog    int      `json:"unchangedBacklog"`
	UnchangedInitiative int      `json:"unchangedInitiative"`
	LegacyBacklog       int      `json:"legacyBacklog"`
	LegacyInitiative    int      `json:"legacyInitiative"`
}

// AISearchApplyResult mirrors aisearch.ApplyResult.
type AISearchApplyResult struct {
	StartedAt          string                   `json:"startedAt"`
	FinishedAt         string                   `json:"finishedAt"`
	UpsertedBacklog    int                      `json:"upsertedBacklog"`
	UpsertedInitiative int                      `json:"upsertedInitiative"`
	DeletedBacklog     int                      `json:"deletedBacklog"`
	DeletedInitiative  int                      `json:"deletedInitiative"`
	Errors             []AISearchReconcileError `json:"errors,omitempty"`
}

// AISearchReconcileError mirrors aisearch.ReconcileError.
type AISearchReconcileError struct {
	Kind    string `json:"kind"`
	PointID string `json:"pointId,omitempty"`
	Name    string `json:"name,omitempty"`
	Op      string `json:"op"`
	Err     string `json:"err"`
}

// aiSearchDryRunResponse is the {"dry_run": true, "plan": …} body returned
// when the reconcile endpoint is invoked with X-Dry-Run.
type aiSearchDryRunResponse struct {
	DryRun bool                 `json:"dry_run"`
	Plan   *AISearchDriftReport `json:"plan"`
}

// AISearchResult / Response mirror aisearch.AISearchResponse.
type AISearchResult struct {
	Entity       string                 `json:"entity"`
	ID           string                 `json:"id"`
	Score        float64                `json:"score"`
	ScorePercent int                    `json:"scorePercent"`
	Payload      map[string]interface{} `json:"payload"`
}

type AISearchResponse struct {
	Results   []AISearchResult `json:"results"`
	Total     int              `json:"total"`
	Query     string           `json:"query"`
	Entity    string           `json:"entity"`
	Fallback  string           `json:"fallback"`
	LatencyMs int64            `json:"latencyMs"`
}

// --- Status ---

func (a *App) cmdAISearchStatus(args []string) error {
	fs := flag.NewFlagSet("ai-search status", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.core.Get("/search/ai/status", nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	st, err := decodeResponse[AISearchStatus](body)
	if err != nil {
		return err
	}
	return renderAISearchStatus(st)
}

func renderAISearchStatus(st AISearchStatus) error {
	report := cliapp.OperationalReport{}

	if st.Available {
		report.Status = []string{"AI search available"}
	} else {
		report.Status = []string{"AI search unavailable"}
	}
	report.Status = append(report.Status,
		fmt.Sprintf("Ollama reachable: %s", boolLabel(st.Ollama)),
		fmt.Sprintf("Qdrant reachable: %s", boolLabel(st.Qdrant)),
	)
	if st.Message != "" {
		report.Status = append(report.Status, st.Message)
	}

	indexLines := []string{
		fmt.Sprintf("Backlog: %d indexed / %d on disk", st.IndexedBacklog, st.OnDiskBacklog),
		fmt.Sprintf("Initiatives: %d indexed / %d on disk", st.IndexedInitiatives, st.OnDiskInitiatives),
	}
	report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Index coverage", Items: indexLines})

	backlogDrift := st.IndexedBacklog != st.OnDiskBacklog
	initDrift := st.IndexedInitiatives != st.OnDiskInitiatives
	if backlogDrift || initDrift {
		report.NextSteps = append(report.NextSteps, "swarm-manager ai-search reconcile")
	}
	if !st.Ollama {
		report.NextSteps = append(report.NextSteps, "ollama pull nomic-embed-text")
	}
	if !st.Qdrant {
		report.NextSteps = append(report.NextSteps, "vrooli scenario restart qdrant")
	}
	if len(report.NextSteps) == 0 {
		report.NextSteps = []string{"swarm-manager backlog search-ai <query>"}
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

// --- Reconcile ---

func (a *App) cmdAISearchReconcile(args []string) error {
	fs := flag.NewFlagSet("ai-search reconcile", flag.ContinueOnError)
	wait := fs.Bool("wait", false, "Poll until reconcile finishes")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.core.Request("POST", "/search/ai/reconcile", nil, nil)
	if err != nil {
		return err
	}

	if a.globalDry {
		// Server returns {"dry_run": true, "plan": <DriftReport>}
		if *jsonOut {
			cliutil.PrintJSON(body)
			return nil
		}
		dry, err := decodeResponse[aiSearchDryRunResponse](body)
		if err != nil {
			return err
		}
		return renderReconcileDryRun(dry)
	}

	st, err := decodeReconcileStatus(body)
	if err != nil {
		return err
	}

	if *wait {
		st, err = a.pollReconcileUntilDone()
		if err != nil {
			return err
		}
	}

	if *jsonOut {
		return encodeStdoutJSON(st)
	}
	return renderReconcileMutation(st)
}

func (a *App) cmdAISearchReconcileStatus(args []string) error {
	fs := flag.NewFlagSet("ai-search reconcile-status", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.core.Get("/search/ai/reconcile/status", nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	st, err := decodeResponse[AISearchReconcileStatus](body)
	if err != nil {
		return err
	}
	return renderReconcileOperational(st)
}

func (a *App) cmdAISearchReconcileCancel(args []string) error {
	fs := flag.NewFlagSet("ai-search reconcile-cancel", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.core.Request("POST", "/search/ai/reconcile/cancel", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	st, err := decodeResponse[AISearchReconcileStatus](body)
	if err != nil {
		return err
	}
	changes := []string{fmt.Sprintf("Running: %t", st.Running)}
	if st.LastResult != nil {
		changes = append(changes,
			fmt.Sprintf("Upserted so far: %d backlog, %d initiative",
				st.LastResult.UpsertedBacklog, st.LastResult.UpsertedInitiative))
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:  []string{"Cancel requested"},
		Changes: changes,
	})
}

func (a *App) pollingReconcileOnce() (AISearchReconcileStatus, error) {
	body, err := a.core.Get("/search/ai/reconcile/status", nil)
	if err != nil {
		return AISearchReconcileStatus{}, err
	}
	return decodeResponse[AISearchReconcileStatus](body)
}

func (a *App) pollReconcileUntilDone() (AISearchReconcileStatus, error) {
	const (
		pollInterval = 500 * time.Millisecond
		maxWait      = 15 * time.Minute
	)
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		st, err := a.pollingReconcileOnce()
		if err != nil {
			return AISearchReconcileStatus{}, err
		}
		if !st.Running {
			return st, nil
		}
		time.Sleep(pollInterval)
	}
	return AISearchReconcileStatus{}, fmt.Errorf("timed out waiting for reconcile after %s", maxWait)
}

func decodeReconcileStatus(body []byte) (AISearchReconcileStatus, error) {
	return decodeResponse[AISearchReconcileStatus](body)
}

func renderReconcileDryRun(dry aiSearchDryRunResponse) error {
	plan := dry.Plan
	report := cliapp.MutationReport{
		Result:  []string{"Dry-run: reconcile preview (no mutations)"},
		Changes: []string{},
	}
	if plan == nil {
		report.Changes = []string{"No plan returned"}
		return cliapp.RenderMutationReport(os.Stdout, report)
	}
	upsertCount := plan.LegacyBacklog + plan.LegacyInitiative + len(plan.ToDeleteBacklog) + len(plan.ToDeleteInitiative)
	_ = upsertCount
	report.Changes = []string{
		fmt.Sprintf("To delete (backlog): %d", len(plan.ToDeleteBacklog)),
		fmt.Sprintf("To delete (initiative): %d", len(plan.ToDeleteInitiative)),
		fmt.Sprintf("Unchanged (backlog): %d", plan.UnchangedBacklog),
		fmt.Sprintf("Unchanged (initiative): %d", plan.UnchangedInitiative),
		fmt.Sprintf("Legacy hash drain (backlog): %d", plan.LegacyBacklog),
		fmt.Sprintf("Legacy hash drain (initiative): %d", plan.LegacyInitiative),
	}
	report.NextCommand = []string{"swarm-manager ai-search reconcile"}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderReconcileMutation(st AISearchReconcileStatus) error {
	resultLine := "Reconcile started"
	if !st.Running {
		if st.Canceled {
			resultLine = "Reconcile canceled"
		} else {
			resultLine = "Reconcile finished"
		}
	}
	changes := []string{}
	if st.LastResult != nil {
		changes = append(changes,
			fmt.Sprintf("Upserted (backlog): %d", st.LastResult.UpsertedBacklog),
			fmt.Sprintf("Upserted (initiative): %d", st.LastResult.UpsertedInitiative),
			fmt.Sprintf("Deleted (backlog): %d", st.LastResult.DeletedBacklog),
			fmt.Sprintf("Deleted (initiative): %d", st.LastResult.DeletedInitiative),
			fmt.Sprintf("Errors: %d", len(st.LastResult.Errors)),
		)
	}
	if st.LastPlan != nil {
		changes = append(changes,
			fmt.Sprintf("Unchanged (backlog): %d", st.LastPlan.UnchangedBacklog),
			fmt.Sprintf("Unchanged (initiative): %d", st.LastPlan.UnchangedInitiative),
		)
	}
	if st.LastError != "" {
		changes = append(changes, "Last error: "+st.LastError)
	}
	report := cliapp.MutationReport{
		Result:      []string{resultLine},
		Changes:     changes,
		NextCommand: []string{"swarm-manager ai-search reconcile-status"},
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderReconcileOperational(st AISearchReconcileStatus) error {
	report := cliapp.OperationalReport{}
	if st.Running {
		report.Status = []string{
			"Reconcile in progress",
			fmt.Sprintf("Started: %s", st.StartedAt),
		}
	} else {
		report.Status = []string{"No reconcile in progress"}
		if st.FinishedAt != "" {
			report.Status = append(report.Status, "Last finished: "+st.FinishedAt)
		}
	}
	if st.LastResult != nil {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Last result",
			Items: []string{
				fmt.Sprintf("Upserted: %d backlog, %d initiative",
					st.LastResult.UpsertedBacklog, st.LastResult.UpsertedInitiative),
				fmt.Sprintf("Deleted: %d backlog, %d initiative",
					st.LastResult.DeletedBacklog, st.LastResult.DeletedInitiative),
				fmt.Sprintf("Errors: %d", len(st.LastResult.Errors)),
			},
		})
	}
	if st.LastPlan != nil {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Last plan",
			Items: []string{
				fmt.Sprintf("Unchanged: %d backlog, %d initiative",
					st.LastPlan.UnchangedBacklog, st.LastPlan.UnchangedInitiative),
				fmt.Sprintf("Legacy drain: %d backlog, %d initiative",
					st.LastPlan.LegacyBacklog, st.LastPlan.LegacyInitiative),
			},
		})
	}
	switch {
	case st.Running:
		report.NextSteps = append(report.NextSteps, "swarm-manager ai-search reconcile-cancel")
	case st.LastError != "":
		report.NextSteps = append(report.NextSteps, "swarm-manager ai-search reconcile")
	default:
		report.NextSteps = append(report.NextSteps, "swarm-manager ai-search status")
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

// --- Search ---

// cmdAISearchSearch is the generic search command used by both the top-level
// `ai-search query` and the domain-shortcut `backlog search-ai` /
// `initiatives search-ai` commands. entityOverride, when non-empty, is applied
// to every request regardless of the --entity flag.
func (a *App) cmdAISearchSearch(entityOverride string) support.CommandFunc {
	return func(args []string) error {
		fs := flag.NewFlagSet("search-ai", flag.ContinueOnError)
		entity := fs.String("entity", "", "backlog | initiative | record | both (default both)")
		limit := fs.Int("limit", 20, "Max results (1-100)")
		threshold := fs.Float64("threshold", 0, "Min cosine similarity (0-1); 0 uses server default")
		kindCSV := fs.String("kind", "", "Comma-separated backlog kinds to include")
		statusCSV := fs.String("status", "", "Comma-separated statuses to include")
		initiative := fs.String("initiative", "", "Restrict to a single initiative")
		targetScenario := fs.String("target-scenario", "", "Restrict to backlog items targeting a specific scenario")
		includeArchived := fs.Bool("include-archived", false, "Include archived items")
		jsonOut := cliutil.JSONFlag(fs)
		if err := cliutil.ParseInterspersed(fs, args); err != nil {
			return err
		}
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: search-ai <query> [--entity backlog|initiative|both] [--limit N] [--kind KIND,...] [--status STATUS,...] [--initiative NAME] [--include-archived] [--json]")
		}
		query := strings.TrimSpace(strings.Join(fs.Args(), " "))
		if query == "" {
			return fmt.Errorf("query is required")
		}

		payload := map[string]any{
			"query": query,
			"limit": *limit,
		}
		if entityOverride != "" {
			payload["entity"] = entityOverride
		} else if strings.TrimSpace(*entity) != "" {
			payload["entity"] = strings.TrimSpace(*entity)
		}
		if *threshold > 0 {
			payload["threshold"] = *threshold
		}
		filters := map[string]any{}
		if *includeArchived {
			filters["include_archived"] = true
		}
		if s := strings.TrimSpace(*statusCSV); s != "" {
			filters["status"] = cliutil.ParseCSV(s)
		}
		if s := strings.TrimSpace(*kindCSV); s != "" {
			filters["kind"] = cliutil.ParseCSV(s)
		}
		if s := strings.TrimSpace(*initiative); s != "" {
			filters["initiative"] = s
		}
		if s := strings.TrimSpace(*targetScenario); s != "" {
			filters["target_scenario"] = s
		}
		if len(filters) > 0 {
			payload["filters"] = filters
		}

		body, err := a.core.Request("POST", "/search/ai", nil, mustJSON(payload))
		if err != nil {
			return err
		}
		if *jsonOut {
			cliutil.PrintJSON(body)
			return nil
		}
		resp, err := decodeResponse[AISearchResponse](body)
		if err != nil {
			return err
		}
		return renderAISearchResults(resp, entityOverride)
	}
}

func renderAISearchResults(resp AISearchResponse, entityOverride string) error {
	report := cliapp.ListReport{}

	entity := resp.Entity
	if entity == "" {
		entity = "both"
	}
	if entityOverride != "" {
		entity = entityOverride
	}
	summary := []string{
		fmt.Sprintf("Query: %s", resp.Query),
		fmt.Sprintf("Entity: %s", entity),
		fmt.Sprintf("Results: %d", resp.Total),
		fmt.Sprintf("Latency: %dms", resp.LatencyMs),
	}
	if resp.Fallback != "" && resp.Fallback != "none" {
		summary = append(summary, fmt.Sprintf("Fallback: %s (AI search unavailable)", resp.Fallback))
	}
	report.Summary = summary

	if resp.Total == 0 {
		report.Results = []string{"(no matches)"}
	} else {
		for _, r := range resp.Results {
			line := fmt.Sprintf("[%d%%] %s %s", r.ScorePercent, r.Entity, r.ID)
			if title, _ := r.Payload["title"].(string); title != "" {
				line += " — " + title
			}
			if status, _ := r.Payload["status"].(string); status != "" {
				line += fmt.Sprintf(" (status=%s)", status)
			}
			report.Results = append(report.Results, line)
		}
	}
	report.ResultsHeading = "Matches"
	report.RetrievalHints = []string{"Add --json for raw scored payload"}
	if resp.Fallback == "unavailable" {
		report.RetrievalHints = append(report.RetrievalHints, "swarm-manager ai-search status")
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func boolLabel(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func mustJSON(v any) []byte {
	out, _ := json.Marshal(v)
	return out
}

func encodeStdoutJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
