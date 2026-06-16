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
// `/api/v1/scan/tidiness` contract (kept minimal — only the fields this
// producer consumes). The canonical request/response live in tidiness-manager;
// this is the consumer view.
type tidinessScanRequest struct {
	ScenarioName string `json:"scenario_name"`
}

type tidinessScanResponse struct {
	Findings   []tidinessFinding `json:"findings"`
	Violations []tidinessFinding `json:"violations"`
}

type tidinessFinding struct {
	RuleID                 string `json:"rule_id"`
	Category               string `json:"category"`
	Severity               string `json:"severity"`
	Title                  string `json:"title"`
	Description            string `json:"description"`
	WhyItMatters           string `json:"why_it_matters"`
	RecommendedRemediation string `json:"recommended_remediation"`
	Remediation            string `json:"remediation,omitempty"`
	FilePath               string `json:"file_path,omitempty"`
	Symbol                 string `json:"symbol,omitempty"`
	LineNumber             int    `json:"line_number,omitempty"`
}

// runTidinessPhase delegates file/function quality checks to the
// tidiness-manager scenario and maps maintainability findings into
// `tidiness`-source findings. If tidiness-manager is not running it follows
// the runnability-gate pattern — a clear SKIP, never a suite failure — because
// tidiness is an optional external dependency.
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

	findingsFromManager, err := fetchTidinessFindings(ctx, baseURL, scenario)
	if err != nil {
		shared.LogWarn(cleanLog, "tidiness-manager scan failed; skipping tidiness phase: %v", err)
		return RunReport{
			Observations: []Observation{
				NewSectionObservation("🧹", "Tidiness"),
				NewSkipObservation(fmt.Sprintf("tidiness-manager scan failed — phase skipped: %v", err)),
			},
		}
	}

	findings := tidinessArchFindings(scenario, findingsFromManager)
	obs := []Observation{NewSectionObservation("🧹", "Tidiness")}
	if len(findingsFromManager) == 0 {
		obs = append(obs, NewSuccessObservation("No tidiness violations detected"))
		return RunReport{Observations: obs, Findings: findings}
	}
	obs = append(obs, NewInfoObservation(fmt.Sprintf("Tidiness findings: %d", len(findingsFromManager))))
	for _, v := range findingsFromManager {
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

func fetchTidinessFindings(ctx context.Context, baseURL, scenario string) ([]tidinessFinding, error) {
	payload, _ := json.Marshal(tidinessScanRequest{ScenarioName: scenario})
	endpoint := fmt.Sprintf("%s/api/v1/scan/tidiness", strings.TrimRight(baseURL, "/"))
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
	if len(parsed.Findings) > 0 {
		return parsed.Findings, nil
	}
	return parsed.Violations, nil
}

// tidinessArchFindings maps tidiness-manager findings into the shared
// ArchitectureFinding contract (source=TIDINESS). code = rule id, severity is
// normalized by newFinding, location = file path.
func tidinessArchFindings(scenario string, findings []tidinessFinding) []*architecturev1.ArchitectureFinding {
	out := make([]*architecturev1.ArchitectureFinding, 0, len(findings))
	for _, v := range findings {
		title := strings.TrimSpace(v.Title)
		if title == "" {
			title = strings.TrimSpace(v.Description)
		}
		if title == "" {
			title = v.RuleID
		}
		code := strings.TrimSpace(v.RuleID)
		if code == "" {
			code = strings.TrimSpace(v.Category)
		}
		remediation := strings.TrimSpace(v.RecommendedRemediation)
		if remediation == "" {
			remediation = strings.TrimSpace(v.Remediation)
		}
		if why := strings.TrimSpace(v.WhyItMatters); why != "" && remediation != "" {
			remediation = why + "\n\n" + remediation
		}
		location := strings.TrimSpace(v.FilePath)
		if location != "" && v.LineNumber > 0 {
			location = fmt.Sprintf("%s:%d", location, v.LineNumber)
		}
		out = append(out, newFinding(
			scenario,
			architecturev1.FindingSource_FINDING_SOURCE_TIDINESS,
			code, v.Severity, title, remediation,
			nonEmptyLocations(location), nil,
		))
	}
	return out
}
