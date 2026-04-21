// AI search CLI commands — thin wrappers over /api/v1/search/ai endpoints.
//
// Human output follows the cli-steer contracts:
//   - status       → Operational (Status → Triage → Next Steps)
//   - search-ai    → Data Retrieval (Summary → Results → Retrieval Hints)
//   - reindex      → Mutation (Result → What Changed → Next Command)
//
// The --json flag is supported on every command for scripting.
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

// AISearchReindexStatus mirrors aisearch.ReindexStatus.
type AISearchReindexStatus struct {
	Running    bool   `json:"running"`
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
	Indexed    int    `json:"indexed"`
	Skipped    int    `json:"skipped"`
	Errors     int    `json:"errors"`
	Total      int    `json:"total"`
	Message    string `json:"message,omitempty"`
	Canceled   bool   `json:"canceled,omitempty"`
	Error      string `json:"error,omitempty"`
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
		report.NextSteps = append(report.NextSteps, "swarm-manager ai-search reindex")
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

// --- Reindex ---

func (a *App) cmdAISearchReindex(args []string) error {
	fs := flag.NewFlagSet("ai-search reindex", flag.ContinueOnError)
	wait := fs.Bool("wait", false, "Poll until reindex finishes")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := a.core.Request("POST", "/search/ai/reindex", nil, nil)
	if err != nil {
		return err
	}
	if a.globalDry {
		// Server honors X-Dry-Run and returns {dry_run, status}.
		if *jsonOut {
			cliutil.PrintJSON(body)
			return nil
		}
		report := cliapp.MutationReport{
			Result:  []string{"Dry-run: reindex not started"},
			Changes: []string{"No vectors written"},
		}
		return cliapp.RenderMutationReport(os.Stdout, report)
	}

	st, err := decodeReindexStatus(body)
	if err != nil {
		return err
	}

	if *wait {
		st, err = a.pollReindexUntilDone()
		if err != nil {
			return err
		}
	}

	if *jsonOut {
		return encodeStdoutJSON(st)
	}
	return renderReindexMutation(st)
}

func (a *App) cmdAISearchReindexStatus(args []string) error {
	fs := flag.NewFlagSet("ai-search reindex-status", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.core.Get("/search/ai/reindex/status", nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	st, err := decodeResponse[AISearchReindexStatus](body)
	if err != nil {
		return err
	}
	return renderReindexOperational(st)
}

func (a *App) cmdAISearchReindexCancel(args []string) error {
	fs := flag.NewFlagSet("ai-search reindex-cancel", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.core.Request("POST", "/search/ai/reindex/cancel", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	st, err := decodeResponse[AISearchReindexStatus](body)
	if err != nil {
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{
		Result:  []string{"Cancel requested"},
		Changes: []string{fmt.Sprintf("Running: %t, Indexed so far: %d", st.Running, st.Indexed)},
	})
}

func (a *App) pollingReindexOnce() (AISearchReindexStatus, error) {
	body, err := a.core.Get("/search/ai/reindex/status", nil)
	if err != nil {
		return AISearchReindexStatus{}, err
	}
	return decodeResponse[AISearchReindexStatus](body)
}

func (a *App) pollReindexUntilDone() (AISearchReindexStatus, error) {
	const (
		pollInterval = 500 * time.Millisecond
		maxWait      = 15 * time.Minute
	)
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		st, err := a.pollingReindexOnce()
		if err != nil {
			return AISearchReindexStatus{}, err
		}
		if !st.Running {
			return st, nil
		}
		time.Sleep(pollInterval)
	}
	return AISearchReindexStatus{}, fmt.Errorf("timed out waiting for reindex after %s", maxWait)
}

func decodeReindexStatus(body []byte) (AISearchReindexStatus, error) {
	return decodeResponse[AISearchReindexStatus](body)
}

func renderReindexMutation(st AISearchReindexStatus) error {
	report := cliapp.MutationReport{
		Result: []string{func() string {
			if st.Running {
				return "Reindex started"
			}
			if st.Canceled {
				return "Reindex canceled"
			}
			return "Reindex finished"
		}()},
		Changes: []string{
			fmt.Sprintf("Indexed: %d", st.Indexed),
			fmt.Sprintf("Skipped: %d", st.Skipped),
			fmt.Sprintf("Errors: %d", st.Errors),
			fmt.Sprintf("Total: %d", st.Total),
		},
		NextCommand: []string{"swarm-manager ai-search reindex-status"},
	}
	if st.Error != "" {
		report.Changes = append(report.Changes, "Last error: "+st.Error)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderReindexOperational(st AISearchReindexStatus) error {
	report := cliapp.OperationalReport{}
	if st.Running {
		report.Status = []string{
			"Reindex in progress",
			fmt.Sprintf("Started: %s", st.StartedAt),
		}
	} else {
		report.Status = []string{"No reindex in progress"}
	}
	report.Triage = []cliapp.TriageGroup{{
		Heading: "Progress",
		Items: []string{
			fmt.Sprintf("Indexed: %d / %d", st.Indexed, st.Total),
			fmt.Sprintf("Skipped: %d", st.Skipped),
			fmt.Sprintf("Errors: %d", st.Errors),
		},
	}}
	if st.Running {
		report.NextSteps = append(report.NextSteps, "swarm-manager ai-search reindex-cancel")
	} else if !st.Canceled && st.Error == "" && st.FinishedAt != "" {
		report.NextSteps = append(report.NextSteps, "swarm-manager ai-search status")
	} else {
		report.NextSteps = append(report.NextSteps, "swarm-manager ai-search reindex")
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
		entity := fs.String("entity", "", "backlog | initiative | both (default both)")
		limit := fs.Int("limit", 20, "Max results (1-100)")
		threshold := fs.Float64("threshold", 0, "Min cosine similarity (0-1); 0 uses server default")
		kindCSV := fs.String("kind", "", "Comma-separated backlog kinds to include")
		statusCSV := fs.String("status", "", "Comma-separated statuses to include")
		initiative := fs.String("initiative", "", "Restrict to a single initiative")
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
