package lighthouse

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeLighthouseConfig(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lighthouse.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "ui"), 0o755); err != nil {
		t.Fatalf("mkdir ui: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "ui", "package.json"), []byte(`{"name":"demo-ui"}`), 0o644); err != nil {
		t.Fatalf("write ui package: %v", err)
	}
	return root
}

const enabledConfig = `{
  "enabled": true,
  "pages": [{"id":"home","path":"/","thresholds":{"performance":{"error":0.75,"warn":0.85},"accessibility":{"error":0.90,"warn":0.95}}}]
}`

// [REQ:PH-LH-001] A page meeting its threshold SCORES with no violations.
func TestCLIRunnerScoresPass(t *testing.T) {
	root := writeLighthouseConfig(t, enabledConfig)
	r := &CLIRunner{
		Resolve:        func(_, _ string) (string, error) { return root, nil },
		ResolveUIURL:   func(context.Context, string) (string, error) { return "http://localhost:3000", nil },
		LookLighthouse: func(context.Context) (string, error) { return "lighthouse", nil },
		RunAudit: func(_ context.Context, _, _ string, _ []string) (map[string]float64, error) {
			return map[string]float64{"performance": 0.92}, nil
		},
	}
	res, err := r.Run(context.Background(), "demo", "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Outcome != OutcomeScored || len(res.Pages) != 1 {
		t.Fatalf("expected one scored page, got %#v", res)
	}
	if len(res.Pages[0].Violations) != 0 {
		t.Fatalf("expected no violations, got %v", res.Pages[0].Violations)
	}
	if res.Pages[0].Performance != 0.92 {
		t.Fatalf("score not mapped: %#v", res.Pages[0])
	}
}

// [REQ:PH-LH-001] A page BELOW its error threshold records a violation.
func TestCLIRunnerScoresThresholdViolation(t *testing.T) {
	root := writeLighthouseConfig(t, enabledConfig)
	r := &CLIRunner{
		Resolve:        func(_, _ string) (string, error) { return root, nil },
		ResolveUIURL:   func(context.Context, string) (string, error) { return "http://localhost:3000", nil },
		LookLighthouse: func(context.Context) (string, error) { return "lighthouse", nil },
		RunAudit: func(_ context.Context, _, _ string, _ []string) (map[string]float64, error) {
			return map[string]float64{"performance": 0.50}, nil
		},
	}
	res, _ := r.Run(context.Background(), "demo", "")
	if res.Outcome != OutcomeScored || len(res.Pages[0].Violations) != 1 {
		t.Fatalf("expected one violation, got %#v", res.Pages)
	}
}

// [REQ:PH-LH-002] The Lighthouse CLI being absent → clean SKIP.
func TestCLIRunnerSkipsWhenCLIAbsent(t *testing.T) {
	root := writeLighthouseConfig(t, enabledConfig)
	r := &CLIRunner{
		Resolve:        func(_, _ string) (string, error) { return root, nil },
		ResolveUIURL:   func(context.Context, string) (string, error) { return "http://localhost:3000", nil },
		LookLighthouse: func(context.Context) (string, error) { return "", errors.New("not installed") },
	}
	res, _ := r.Run(context.Background(), "demo", "")
	if res.Outcome != OutcomeUnavailable {
		t.Fatalf("expected UNAVAILABLE, got %#v", res)
	}
}

// [REQ:PH-LH-002] No resolvable UI URL → clean SKIP.
func TestCLIRunnerSkipsWhenNoUIURL(t *testing.T) {
	root := writeLighthouseConfig(t, enabledConfig)
	r := &CLIRunner{
		Resolve:      func(_, _ string) (string, error) { return root, nil },
		ResolveUIURL: func(context.Context, string) (string, error) { return "", errors.New("no ui") },
	}
	res, _ := r.Run(context.Background(), "demo", "")
	if res.Outcome != OutcomeUnavailable {
		t.Fatalf("expected UNAVAILABLE, got %#v", res)
	}
}

// [REQ:PH-LH-002] A disabled config (or no config file) → clean SKIP before any
// CLI/URL resolution.
func TestCLIRunnerRejectsDisabledConfigForBrowserUI(t *testing.T) {
	root := writeLighthouseConfig(t, `{"enabled": false}`)
	r := &CLIRunner{Resolve: func(_, _ string) (string, error) { return root, nil }}
	res, _ := r.Run(context.Background(), "demo", "")
	if res.Outcome != OutcomeConfigurationInvalid {
		t.Fatalf("expected configuration failure, got %#v", res)
	}
}

func TestCLIRunnerRejectsMissingAccessibilityThreshold(t *testing.T) {
	root := writeLighthouseConfig(t, `{"enabled":true,"pages":[{"id":"home","path":"/","thresholds":{"performance":{"error":0.75,"warn":0.85}}}]}`)
	r := &CLIRunner{Resolve: func(_, _ string) (string, error) { return root, nil }}
	res, _ := r.Run(context.Background(), "demo", "")
	if res.Outcome != OutcomeConfigurationInvalid {
		t.Fatalf("expected configuration failure, got %#v", res)
	}
}

func TestParseScores(t *testing.T) {
	scores, err := parseScores([]byte(`{"categories":{"performance":{"score":0.9},"seo":{"score":null}}}`))
	if err != nil {
		t.Fatalf("parseScores: %v", err)
	}
	if scores["performance"] != 0.9 {
		t.Fatalf("expected 0.9, got %v", scores)
	}
	if _, ok := scores["seo"]; ok {
		t.Fatalf("null score should be omitted, got %v", scores)
	}
}

func TestJoinURL(t *testing.T) {
	cases := map[string]string{
		"":      "http://h:1/",
		"/":     "http://h:1/",
		"/x":    "http://h:1/x",
		"about": "http://h:1/about",
	}
	for path, want := range cases {
		if got := joinURL("http://h:1/", path); got != want {
			t.Fatalf("joinURL(%q) = %q, want %q", path, got, want)
		}
	}
}
