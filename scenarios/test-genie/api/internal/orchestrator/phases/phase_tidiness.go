package phases

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"

	"github.com/vrooli/api-core/discovery"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// tidinessClient is the seam tests override to stand in for the
// tidiness-manager HTTP call. Production wiring resolves the running
// tidiness-manager via discovery; tests inject a fake.
var tidinessClient = &http.Client{Timeout: 2 * time.Minute}

// resolveTidinessBaseURL is a package-level seam (mirrors the standards phase's
// resolveScenarioAuditorBaseURL) so tests can override discovery.
var resolveTidinessBaseURL = func(ctx context.Context) (string, error) {
	return discovery.ResolveScenarioURLDefault(ctx, "tidiness-manager")
}

// tidinessScanRequest / tidinessScanResponse mirror the tidiness-manager
// `/api/v1/scan/type-safety` contract (kept minimal — only the fields this
// producer consumes). The canonical request/response live in tidiness-manager;
// this is the consumer view.
type tidinessScanRequest struct {
	ScenarioName    string `json:"scenario_name"`
	IncludePatterns bool   `json:"include_patterns"`
}

type tidinessScanResponse struct {
	Violations []tidinessViolation `json:"violations"`
}

type tidinessViolation struct {
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Remediation string `json:"remediation,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
}

// runTidinessPhase delegates file/function quality checks to the
// tidiness-manager scenario (the same provider scenario-auditor uses) and maps
// its violations into `tidiness`-source findings. If tidiness-manager is not
// running it follows the runnability-gate pattern — a clear SKIP, never a
// suite failure — because tidiness is an optional external dependency.
func runTidinessPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if report := CheckContext(ctx); report != nil {
		return *report
	}

	cleanLog := wrapLogSansANSI(logWriter)
	scenario := strings.TrimSpace(env.ScenarioName)
	if scenario == "" {
		return RunReport{Observations: []Observation{
			NewSectionObservation("🧹", "Tidiness"),
			NewSkipObservation("no scenario name available"),
		}}
	}

	shared.LogStep(cleanLog, "running tidiness scan via tidiness-manager API")
	baseURL, err := resolveTidinessBaseURL(ctx)
	if err != nil {
		shared.LogWarn(cleanLog, "tidiness-manager unavailable; skipping tidiness phase: %v", err)
		return RunReport{
			Observations: []Observation{
				NewSectionObservation("🧹", "Tidiness"),
				NewSkipObservation("tidiness-manager not reachable — phase skipped (start tidiness-manager to enable)"),
			},
		}
	}

	violations, err := fetchTidinessViolations(ctx, baseURL, scenario)
	if err != nil {
		shared.LogWarn(cleanLog, "tidiness-manager scan failed; skipping tidiness phase: %v", err)
		return RunReport{
			Observations: []Observation{
				NewSectionObservation("🧹", "Tidiness"),
				NewSkipObservation(fmt.Sprintf("tidiness-manager scan failed — phase skipped: %v", err)),
			},
		}
	}

	findings := tidinessArchFindings(scenario, violations)
	obs := []Observation{NewSectionObservation("🧹", "Tidiness")}
	if len(violations) == 0 {
		obs = append(obs, NewSuccessObservation("No tidiness violations detected"))
		return RunReport{Observations: obs, Findings: findings}
	}
	obs = append(obs, NewInfoObservation(fmt.Sprintf("Tidiness violations: %d", len(violations))))
	for _, v := range violations {
		title := strings.TrimSpace(v.Title)
		if title == "" {
			title = v.RuleID
		}
		loc := strings.TrimSpace(v.FilePath)
		msg := fmt.Sprintf("[%s] %s", strings.ToUpper(strings.TrimSpace(v.Severity)), title)
		if loc != "" {
			msg += " -> " + loc
		}
		obs = append(obs, NewWarningObservation(msg))
	}
	return RunReport{Observations: obs, Findings: findings}
}

func fetchTidinessViolations(ctx context.Context, baseURL, scenario string) ([]tidinessViolation, error) {
	payload, _ := json.Marshal(tidinessScanRequest{ScenarioName: scenario, IncludePatterns: true})
	endpoint := fmt.Sprintf("%s/api/v1/scan/type-safety", strings.TrimRight(baseURL, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := tidinessClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tidiness-manager responded with %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var parsed tidinessScanResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return parsed.Violations, nil
}

// tidinessArchFindings maps tidiness-manager violations into the shared
// ArchitectureFinding contract (source=TIDINESS). code = rule id, severity is
// normalized by newFinding, location = file path.
func tidinessArchFindings(scenario string, violations []tidinessViolation) []*architecturev1.ArchitectureFinding {
	out := make([]*architecturev1.ArchitectureFinding, 0, len(violations))
	for _, v := range violations {
		title := strings.TrimSpace(v.Title)
		if title == "" {
			title = strings.TrimSpace(v.Description)
		}
		if title == "" {
			title = v.RuleID
		}
		out = append(out, newFinding(
			scenario,
			architecturev1.FindingSource_FINDING_SOURCE_TIDINESS,
			v.RuleID, v.Severity, title, strings.TrimSpace(v.Remediation),
			nonEmptyLocations(strings.TrimSpace(v.FilePath)), nil,
		))
	}
	return out
}
