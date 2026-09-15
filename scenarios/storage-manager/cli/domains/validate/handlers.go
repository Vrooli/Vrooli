package validate

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	httpClient httpDoer
	baseURL    string
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 60*time.Second)
	return &handlers{httpClient: httpClient, baseURL: baseURL}
}

func (h *handlers) validateScenario(ctx cliapp.RunContext) error {
	return h.validateOwner(ctx, "scenario", ctx.Positional("name"))
}

type validationFinding struct {
	Code     string `json:"code"`
	Severity int    `json:"severity"`
	Message  string `json:"message"`
}

type validationReport struct {
	Scenario  string              `json:"scenario"`
	OwnerKind string              `json:"owner_kind"`
	OwnerID   string              `json:"owner_id"`
	Platform  string              `json:"platform"`
	Status    string              `json:"status"`
	Findings  []validationFinding `json:"findings"`
}

func (h *handlers) validateOwner(ctx cliapp.RunContext, kind, id string) error {
	report, raw, err := h.getReport(kind, id, ctx.Flag("platform"))
	if err != nil {
		return err
	}
	if ctx.JSON() {
		_, err = ctx.Stdout().Write(raw)
		return err
	}
	results := make([]string, 0, len(report.Findings))
	for _, finding := range report.Findings {
		results = append(results, fmt.Sprintf("[%d] %s: %s", finding.Severity, finding.Code, finding.Message))
	}
	if err := ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("Validated %s/%s on %s — status=%s findings=%d", report.OwnerKind, report.OwnerID, report.Platform, report.Status, len(results))}, ResultsHeading: "Findings", Results: results}); err != nil {
		return err
	}
	if report.Status == "failed" {
		return fmt.Errorf("%s %q failed storage validation", kind, id)
	}
	return nil
}

func (h *handlers) getReport(kind, id, platform string) (validationReport, []byte, error) {
	endpoint := fmt.Sprintf("%s/api/v1/validation/validate/%s/%s", h.baseURL, url.PathEscape(kind), url.PathEscape(id))
	if strings.TrimSpace(platform) != "" {
		endpoint += "?platform=" + url.QueryEscape(platform)
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return validationReport{}, nil, err
	}
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return validationReport{}, nil, cliapp.WrapAPIError(fmt.Sprintf("validate %s %q", kind, id), err, nil)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return validationReport{}, nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return validationReport{}, nil, fmt.Errorf("validate %s: %s", kind, strings.TrimSpace(string(raw)))
	}
	var report validationReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return validationReport{}, nil, fmt.Errorf("decode validation report: %w", err)
	}
	return report, raw, nil
}

func (h *handlers) validateFleet(ctx cliapp.RunContext) error {
	endpoint := h.baseURL + "/api/v1/validation/validate/fleet"
	query := url.Values{}
	if platform := strings.TrimSpace(ctx.Flag("platform")); platform != "" {
		query.Set("platform", platform)
	}
	for _, kind := range ctx.FlagValues("kind") {
		if strings.TrimSpace(kind) != "" {
			query.Add("kind", kind)
		}
	}
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return cliapp.WrapAPIError("validate fleet", err, nil)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("validate fleet: %s", strings.TrimSpace(string(raw)))
	}
	var report struct {
		Reports    []validationReport `json:"reports"`
		ErrorCount int                `json:"error_count"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		return fmt.Errorf("decode fleet validation report: %w", err)
	}
	if ctx.JSON() {
		if _, err = ctx.Stdout().Write(raw); err != nil {
			return err
		}
	} else {
		results := []string{fmt.Sprintf("Validated %d owner(s); errors=%d", len(report.Reports), report.ErrorCount)}
		if err := ctx.RenderList(cliapp.ListReport{Summary: results, ResultsHeading: "Fleet validation", Results: results}); err != nil {
			return err
		}
	}
	if report.ErrorCount > 0 {
		return fmt.Errorf("fleet validation found %d error or blocker finding(s)", report.ErrorCount)
	}
	return nil
}

func (h *handlers) proveIsolation(ctx cliapp.RunContext) error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	scenario := ctx.Positional("name")
	apiDir := filepath.Join(root, "scenarios", scenario, "api")
	required := []struct {
		name   string
		marker string
	}{
		{"database.Open", "database.Open"},
		{"database.EnsureSchemas", "database.EnsureSchemas"},
		{"apihttp.TestModeMiddleware", "apihttp.TestModeMiddleware"},
		{"devrouting.RegisterWithFileRoots", "devrouting.RegisterWithFileRoots"},
		{"filerouting.New", "filerouting.New"},
		{"RoutedRoots.Pick", "RoutedRoots.Pick"},
	}
	var source strings.Builder
	if walkErr := filepath.WalkDir(apiDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		source.Write(data)
		return nil
	}); walkErr != nil {
		return fmt.Errorf("prove isolation: scan %s: %w", scenario, walkErr)
	}
	missing := make([]string, 0)
	for _, check := range required {
		if !strings.Contains(source.String(), check.marker) {
			missing = append(missing, check.name)
		}
	}
	results := []string{fmt.Sprintf("Scenario %s isolation proof: %t", scenario, len(missing) == 0)}
	if len(missing) > 0 {
		results = append(results, "Missing seams: "+strings.Join(missing, ", "))
	}
	if err := ctx.RenderList(cliapp.ListReport{Summary: []string{"API-free static isolation proof"}, ResultsHeading: "Isolation", Results: results}); err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("scenario %q isolation is not proven", scenario)
	}
	return nil
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, ".vrooli", "repo-contract.json")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no repository root found above %s", dir)
		}
		dir = parent
	}
}
