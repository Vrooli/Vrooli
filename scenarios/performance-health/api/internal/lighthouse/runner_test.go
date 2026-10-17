package lighthouse

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
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
		RunAudit: func(_ context.Context, _, _ string, _ AuditOptions) (map[string]float64, error) {
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
		RunAudit: func(_ context.Context, _, _ string, _ AuditOptions) (map[string]float64, error) {
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

func TestCLIRunnerUsesDeclaredDeviceAndPageOverride(t *testing.T) {
	root := writeLighthouseConfig(t, `{
 "enabled":true,
 "pages":[
 {"id":"desktop","path":"/","viewport":"desktop","thresholds":{"accessibility":{"error":0.9,"warn":0.95}}},
 {"id":"phone","path":"/phone","viewport":"mobile","thresholds":{"accessibility":{"error":0.9,"warn":0.95}}}],
 "global_options":{"lighthouse":{"settings":{"formFactor":"desktop","throttlingMethod":"devtools","onlyCategories":["performance","accessibility"],"screenEmulation":{"width":1440,"height":900,"mobile":false},"throttling":{"cpuSlowdownMultiplier":2}}},"chrome_flags":["--headless","--no-sandbox"],"timeout_ms":90000}}`)
	var options []AuditOptions
	r := &CLIRunner{
		Resolve:        func(_, _ string) (string, error) { return root, nil },
		ResolveUIURL:   func(context.Context, string) (string, error) { return "http://localhost:3000", nil },
		LookLighthouse: func(context.Context) (string, error) { return "lighthouse", nil },
		RunAudit: func(_ context.Context, _, _ string, o AuditOptions) (map[string]float64, error) {
			options = append(options, o)
			return map[string]float64{"performance": 1, "accessibility": 1}, nil
		},
	}
	result, err := r.Run(context.Background(), "demo", "")
	if err != nil || result.Outcome != OutcomeScored || len(options) != 2 {
		t.Fatalf("run: %#v %v", result, err)
	}
	desktop, phone := options[0], options[1]
	if desktop.FormFactor != "desktop" || desktop.ScreenEmulation["width"] != float64(1440) || desktop.Throttling["cpuSlowdownMultiplier"] != float64(2) {
		t.Fatalf("lost desktop settings: %#v", desktop)
	}
	if phone.FormFactor != "mobile" || phone.ScreenEmulation != nil || phone.Throttling != nil {
		t.Fatalf("phone inherited desktop emulation: %#v", phone)
	}
	if desktop.Timeout != 90*time.Second || desktop.ThrottlingMethod != "devtools" || !reflect.DeepEqual(desktop.Categories, []string{"performance", "accessibility"}) || !reflect.DeepEqual(desktop.ChromeFlags, []string{"--headless", "--no-sandbox"}) {
		t.Fatalf("lost runner settings: %#v", desktop)
	}
}

func TestDefaultRunAuditPassesDesktopSettingsToCLI(t *testing.T) {
	dir := t.TempDir()
	executable := filepath.Join(dir, "fake-lighthouse")
	capture := filepath.Join(dir, "args")
	t.Setenv("LH_TEST_ARGS", capture)
	script := `#!/bin/sh
printf '%s\n' "$@" > "$LH_TEST_ARGS"
for arg in "$@"; do
 case "$arg" in --output-path=*) output="${arg#--output-path=}";; esac
done
printf '%s' '{"configSettings":{"formFactor":"desktop"},"categories":{"performance":{"score":0.95}}}' > "$output"
`
	if err := os.WriteFile(executable, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	scores, err := defaultRunAudit(context.Background(), executable, "http://localhost:3000", AuditOptions{
		Categories: []string{"performance"}, FormFactor: "desktop", ThrottlingMethod: "simulate",
		ScreenEmulation: map[string]any{"width": float64(1440), "mobile": false},
		Throttling:      map[string]any{"cpuSlowdownMultiplier": float64(2)},
		ChromeFlags:     []string{"--headless", "--no-sandbox"}, Timeout: time.Second,
	})
	if err != nil || scores["performance"] != 0.95 {
		t.Fatalf("audit: %v %v", scores, err)
	}
	raw, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Split(strings.TrimSpace(string(raw)), "\n")
	for _, want := range []string{"--preset=desktop", "--throttling-method=simulate", "--screenEmulation.width=1440", "--screenEmulation.mobile=false", "--throttling.cpuSlowdownMultiplier=2", "--chrome-flags=--headless --no-sandbox", "--only-categories=performance"} {
		found := false
		for _, arg := range args {
			if arg == want {
				found = true
			}
		}
		if !found {
			t.Errorf("missing %q in %#v", want, args)
		}
	}
	// A successful process must not become scored evidence for the wrong device.
	script = strings.Replace(script, `"formFactor":"desktop"`, `"formFactor":"mobile"`, 1)
	if err := os.WriteFile(executable, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = defaultRunAudit(context.Background(), executable, "http://localhost:3000", AuditOptions{FormFactor: "desktop", Timeout: time.Second})
	if err == nil || !strings.Contains(err.Error(), `instead of declared "desktop"`) {
		t.Fatalf("accepted a mobile report as desktop: %v", err)
	}

}

func TestCLIRunnerRejectsUnknownDevice(t *testing.T) {
	root := writeLighthouseConfig(t, strings.Replace(enabledConfig, `"path":"/"`, `"path":"/","viewport":"tablet"`, 1))
	r := &CLIRunner{Resolve: func(_, _ string) (string, error) { return root, nil }}
	result, _ := r.Run(context.Background(), "demo", "")
	if result.Outcome != OutcomeConfigurationInvalid {
		t.Fatalf("accepted unknown device: %#v", result)
	}
}
