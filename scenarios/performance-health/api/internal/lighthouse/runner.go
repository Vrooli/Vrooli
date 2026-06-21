package lighthouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"

	"performance-health/internal/scenarioroot"
)

const uiPortKey = "UI_PORT"

// CLIRunner is the production Lighthouse Runner. It owns its OWN Chrome via the
// Lighthouse CLI (NOT via BAS): it resolves the scenario's served UI URL, loads
// per-page thresholds from .vrooli/lighthouse.json, runs Lighthouse once per
// configured page, and reports category scores plus any threshold violations.
//
// It is Tier 0 (works on any URL) and SILENTLY SKIPS (never errors) when:
//   - Lighthouse config is disabled or has no pages,
//   - the Lighthouse CLI is absent (and npx cannot supply it),
//   - the scenario serves no resolvable UI URL.
type CLIRunner struct {
	RepoRoot string

	// Resolve maps (scenario, path) → scenario root; nil uses the repo contract.
	Resolve func(scenario, path string) (string, error)

	// ResolveUIURL maps a scenario slug → its served UI base URL; nil uses the
	// discovery resolver against UI_PORT. An error means "no UI" (clean skip).
	ResolveUIURL func(ctx context.Context, scenario string) (string, error)

	// LookLighthouse returns the lighthouse invocation ("lighthouse" path or
	// "npx") or an error when neither is available; nil uses the default lookup.
	LookLighthouse func(ctx context.Context) (string, error)

	// RunAudit runs one Lighthouse audit for url and returns 0..1 category
	// scores keyed by category id; nil uses the real CLI.
	RunAudit func(ctx context.Context, lighthouse, url string, categories []string) (map[string]float64, error)
}

var _ Runner = (*CLIRunner)(nil)

// Run scores a scenario's pages with Lighthouse, skipping cleanly when impossible.
func (r *CLIRunner) Run(ctx context.Context, scenario, path string) (Result, error) {
	root, err := r.resolveRoot(scenario, path)
	if err != nil {
		return Result{Scenario: scenario, Outcome: OutcomeSkipped, Reason: "could not resolve scenario root: " + err.Error()}, nil
	}

	cfg, err := loadConfig(root)
	if err != nil {
		return Result{Scenario: scenario, Outcome: OutcomeSkipped, Reason: "invalid lighthouse config: " + err.Error()}, nil
	}
	if !cfg.Enabled || len(cfg.Pages) == 0 {
		return Result{Scenario: scenario, Outcome: OutcomeSkipped, Reason: "lighthouse is disabled or has no pages configured"}, nil
	}

	baseURL, err := r.resolveUIURL(ctx, scenario)
	if err != nil || strings.TrimSpace(baseURL) == "" {
		return Result{Scenario: scenario, Outcome: OutcomeSkipped, Reason: "no resolvable UI URL for scenario"}, nil
	}

	lh, err := r.lookLighthouse(ctx)
	if err != nil {
		return Result{Scenario: scenario, Outcome: OutcomeSkipped, Reason: "lighthouse CLI not available"}, nil
	}

	categories := cfg.categories()
	var pages []PageScore
	for _, page := range cfg.Pages {
		url := joinURL(baseURL, page.Path)
		scores, aerr := r.runAudit(ctx, lh, url, categories)
		if aerr != nil {
			return Result{Scenario: scenario, Outcome: OutcomeFailed, Pages: pages, Reason: fmt.Sprintf("lighthouse audit of %s failed: %v", url, aerr)}, nil
		}
		pages = append(pages, scorePage(url, scores, page.Thresholds))
	}

	return Result{Scenario: scenario, Outcome: OutcomeScored, Pages: pages}, nil
}

// scorePage maps raw category scores onto a PageScore and records a violation
// for each category whose score falls below its configured error threshold.
func scorePage(url string, scores map[string]float64, thresholds map[string]threshold) PageScore {
	ps := PageScore{
		URL:           url,
		Performance:   scores["performance"],
		Accessibility: scores["accessibility"],
		BestPractices: scores["best-practices"],
		SEO:           scores["seo"],
	}
	for category, th := range thresholds {
		score, ok := scores[category]
		if !ok {
			continue
		}
		switch {
		case th.Error > 0 && score < th.Error:
			ps.Violations = append(ps.Violations, fmt.Sprintf("%s %.2f < error %.2f", category, score, th.Error))
		case th.Warn > 0 && score < th.Warn:
			ps.Violations = append(ps.Violations, fmt.Sprintf("%s %.2f < warn %.2f", category, score, th.Warn))
		}
	}
	return ps
}

func (r *CLIRunner) resolveRoot(scenario, path string) (string, error) {
	if r.Resolve != nil {
		return r.Resolve(scenario, path)
	}
	return scenarioroot.Resolve(r.RepoRoot, scenario, path)
}

func (r *CLIRunner) resolveUIURL(ctx context.Context, scenario string) (string, error) {
	if r.ResolveUIURL != nil {
		return r.ResolveUIURL(ctx, scenario)
	}
	if strings.TrimSpace(scenario) == "" {
		return "", errors.New("scenario is required to resolve a UI URL")
	}
	return discovery.NewResolver(discovery.ResolverConfig{}).ResolveScenarioURL(ctx, scenario, uiPortKey)
}

func (r *CLIRunner) lookLighthouse(ctx context.Context) (string, error) {
	if r.LookLighthouse != nil {
		return r.LookLighthouse(ctx)
	}
	return defaultLookLighthouse(ctx)
}

func (r *CLIRunner) runAudit(ctx context.Context, lh, url string, categories []string) (map[string]float64, error) {
	if r.RunAudit != nil {
		return r.RunAudit(ctx, lh, url, categories)
	}
	return defaultRunAudit(ctx, lh, url, categories)
}

// --- config (migrated shape from test-genie's .vrooli/lighthouse.json) ---

type config struct {
	Enabled       bool         `json:"enabled"`
	Pages         []pageConfig `json:"pages"`
	GlobalOptions *struct {
		Lighthouse *struct {
			Settings *struct {
				OnlyCategories []string `json:"onlyCategories"`
			} `json:"settings"`
		} `json:"lighthouse"`
	} `json:"global_options"`
}

type pageConfig struct {
	ID         string               `json:"id"`
	Path       string               `json:"path"`
	Thresholds map[string]threshold `json:"thresholds"`
}

type threshold struct {
	Error float64 `json:"error"`
	Warn  float64 `json:"warn"`
}

func (c config) categories() []string {
	if c.GlobalOptions != nil && c.GlobalOptions.Lighthouse != nil &&
		c.GlobalOptions.Lighthouse.Settings != nil &&
		len(c.GlobalOptions.Lighthouse.Settings.OnlyCategories) > 0 {
		return c.GlobalOptions.Lighthouse.Settings.OnlyCategories
	}
	return []string{"performance", "accessibility", "best-practices", "seo"}
}

func loadConfig(root string) (config, error) {
	raw, err := os.ReadFile(filepath.Join(root, ".vrooli", "lighthouse.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config{Enabled: false}, nil
		}
		return config{}, err
	}
	var cfg config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}

// --- default CLI invocation ---

// defaultLookLighthouse locates the lighthouse CLI, falling back to npx, mirroring
// test-genie's resolution. Returns the invocation token or an error.
func defaultLookLighthouse(ctx context.Context) (string, error) {
	if path, err := exec.LookPath("lighthouse"); err == nil {
		return path, nil
	}
	if _, err := exec.LookPath("npx"); err == nil {
		probe, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if exec.CommandContext(probe, "npx", "--no", "lighthouse", "--version").Run() == nil {
			return "npx", nil
		}
		// npx exists but lighthouse is not pre-installed: do NOT auto-install in
		// an autonomous run — treat as absent so the runner skips cleanly.
		return "", errors.New("lighthouse not installed (npx present but no cached lighthouse)")
	}
	return "", errors.New("neither 'lighthouse' nor 'npx' found in PATH")
}

// defaultRunAudit shells the Lighthouse CLI once and returns 0..1 category scores.
func defaultRunAudit(ctx context.Context, lh, url string, categories []string) (map[string]float64, error) {
	tmp, err := os.MkdirTemp("", "perf-health-lh-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	out := filepath.Join(tmp, "report")

	args := []string{
		url,
		"--output=json",
		"--output-path=" + out,
		"--chrome-flags=--headless --no-sandbox --disable-gpu",
		"--only-categories=" + strings.Join(categories, ","),
	}

	runCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	var cmd *exec.Cmd
	if lh == "npx" {
		cmd = exec.CommandContext(runCtx, "npx", append([]string{"lighthouse"}, args...)...)
	} else {
		cmd = exec.CommandContext(runCtx, lh, args...)
	}
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		return nil, fmt.Errorf("read lighthouse output: %w", err)
	}
	return parseScores(raw)
}

// parseScores extracts 0..1 category scores from a Lighthouse JSON report.
func parseScores(raw []byte) (map[string]float64, error) {
	var lhr struct {
		Categories map[string]struct {
			Score *float64 `json:"score"`
		} `json:"categories"`
	}
	if err := json.Unmarshal(raw, &lhr); err != nil {
		return nil, fmt.Errorf("parse lighthouse output: %w", err)
	}
	scores := make(map[string]float64, len(lhr.Categories))
	for id, c := range lhr.Categories {
		if c.Score != nil {
			scores[id] = *c.Score
		}
	}
	return scores, nil
}

func joinURL(base, path string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return base + "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}
