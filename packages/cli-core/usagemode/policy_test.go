package usagemode

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// livePolicy is a one-resource repository whose codex policy a test can edit.
type livePolicy struct {
	t    *testing.T
	path string
}

func newLivePolicy(t *testing.T, modelMap, exempt string) *livePolicy {
	t.Helper()
	useHome(t)
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	p := &livePolicy{t: t, path: filepath.Join(root, "resources", "codex", "model-policy.json")}
	if err := os.MkdirAll(filepath.Dir(p.path), 0o755); err != nil {
		t.Fatal(err)
	}
	p.write(modelMap, exempt)
	return p
}

func (p *livePolicy) write(modelMap, exempt string) {
	p.t.Helper()
	doc := `{"runner":"codex","excluded_models":["astra"],"model_aliases":{
	  "luna":{"canonical_model":"codex/luna","provider":"codex"},"terra":{"canonical_model":"codex/terra","provider":"codex"},
	  "sol":{"canonical_model":"codex/sol","provider":"codex"},"astra":{"canonical_model":"codex/astra","provider":"codex"}},
	  "usage_tiers":{"model_map":` + modelMap + `,"session_model":"luna","exempt_roles":` + exempt + `}}`
	if err := os.WriteFile(p.path, []byte(doc), 0o644); err != nil {
		p.t.Fatal(err)
	}
	// Distinct size and time, so the cache must notice the edit.
	later := time.Now().Add(time.Duration(len(doc)) * time.Second)
	_ = os.Chtimes(p.path, later, later)
}

func (p *livePolicy) tiers() Tiers {
	p.t.Helper()
	raw, err := os.ReadFile(p.path)
	if err != nil {
		p.t.Fatal(err)
	}
	tiers, err := LoadTiers(raw)
	if err != nil {
		p.t.Fatal(err)
	}
	return tiers
}

// A mode started before the policy named a model keeps working and picks the
// model up from the policy without being restarted.
func TestLegacyStoredModePicksUpPolicyAdditions(t *testing.T) {
	newLivePolicy(t, `{"sol":"luna","terra":"luna"}`, `[]`)
	stale := Mode{ID: "um-old", StartedAt: t0.Add(-time.Hour), Until: Until{Kind: UntilOff}, Runners: map[string]RunnerSettings{
		"codex": {ModelMap: map[string]string{"terra": "luna"}, SessionModel: "luna"},
	}}
	if err := Write(File{Modes: []Mode{stale}}); err != nil {
		t.Fatal(err)
	}
	m, ok := Resolve(Query{Runner: "codex", Now: t0})
	if !ok {
		t.Fatal("stale mode not resolved")
	}
	s, _ := m.SettingsFor("codex")
	if s.ModelMap["sol"] != "luna" || s.ModelMap["terra"] != "luna" || s.SessionModel != "luna" {
		t.Fatalf("settings = %+v", s)
	}
}

// The interactive default decides what the mode lowers: with sol configured a
// new session resolves to luna while a mode is active, and is left alone
// without one.
func TestLaunchLowersTheConfiguredInteractiveDefaultOnlyWhileAModeIsActive(t *testing.T) {
	p := newLivePolicy(t, `{"sol":"luna"}`, `[]`)
	config := filepath.Join(os.Getenv("CODEX_HOME"), "config.toml")
	if err := os.WriteFile(config, []byte("model = \"sol\"\nmodel_reasoning_effort = \"high\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"-C", "x"}
	if plan := LaunchPlan("codex", args, nil, t0); plan.Applied || !reflect.DeepEqual(plan.Args, args) {
		t.Fatalf("no mode must leave the launch alone: %+v", plan)
	}
	intent, err := NewMode(p.tiers(), NewModeOptions{Now: t0.Add(-time.Minute), Scope: Scope{Runner: "codex"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(File{Modes: []Mode{intent}}); err != nil {
		t.Fatal(err)
	}
	plan := LaunchPlan("codex", args, nil, t0)
	if want := []string{"-c", "model=luna", "-C", "x"}; !reflect.DeepEqual(plan.Args, want) {
		t.Fatalf("args = %v, want %v", plan.Args, want)
	}
}

// A new mode stores intent only, so a policy edit reaches it at once.
func TestNewModeStoresIntentAndFollowsPolicyEdits(t *testing.T) {
	p := newLivePolicy(t, `{"terra":"luna"}`, `["code.judgment"]`)
	m, err := NewMode(p.tiers(), NewModeOptions{Now: t0, Scope: Scope{Runner: "codex"}, Effort: "low"})
	if err != nil {
		t.Fatal(err)
	}
	if stored := m.Runners["codex"]; stored.ModelMap != nil || stored.SessionModel != "" || stored.Effort != "low" || len(m.ExemptRoles) != 0 {
		t.Fatalf("a mode must store intent only: %+v exempt=%v", stored, m.ExemptRoles)
	}
	s, _ := m.SettingsFor("codex")
	if s.ModelMap["terra"] != "luna" || s.ModelMap["sol"] != "" || s.Effort != "low" {
		t.Fatalf("before the edit: %+v", s)
	}
	if m.RoleExempt("code.review") != true || m.RoleExempt("code.judgment") != true || m.RoleExempt("code.default") {
		t.Fatalf("exempt roles = %v", m.EffectiveExemptRoles())
	}

	p.write(`{"terra":"luna","sol":"luna"}`, `[]`)
	s, _ = m.SettingsFor("codex")
	if s.ModelMap["sol"] != "luna" {
		t.Fatalf("policy edit did not reach the active mode: %+v", s)
	}
	if m.RoleExempt("code.judgment") {
		t.Fatal("an exemption the policy dropped is still applied")
	}
}

func TestOverrideModelIsStoredAndAppliedLive(t *testing.T) {
	p := newLivePolicy(t, `{"sol":"terra","terra":"luna"}`, `[]`)
	m, err := NewMode(p.tiers(), NewModeOptions{Now: t0, Scope: Scope{Runner: "codex"}, OverrideModel: "luna"})
	if err != nil {
		t.Fatal(err)
	}
	s, _ := m.SettingsFor("codex")
	if m.Runners["codex"].OverrideModel != "luna" || s.ModelMap["sol"] != "luna" || s.SessionModel != "luna" {
		t.Fatalf("override: stored=%+v live=%+v", m.Runners["codex"], s)
	}
	if _, err := NewMode(p.tiers(), NewModeOptions{Now: t0, OverrideModel: "sol"}); err == nil {
		t.Fatal("an override that raises was accepted")
	}
}

// Without a readable policy a mode still applies what it stored.
func TestModeFailsOpenToStoredSettingsWithoutAPolicy(t *testing.T) {
	useHome(t)
	m := Mode{ID: "um-x", Runners: map[string]RunnerSettings{"codex": {ModelMap: map[string]string{"a": "b"}, SessionModel: "b"}}}
	s, ok := m.SettingsFor("codex")
	if !ok || s.ModelMap["a"] != "b" || s.SessionModel != "b" {
		t.Fatalf("stored settings lost: %+v", s)
	}
}

// brokenPolicies are the ways a model-policy.json goes wrong. A launch under an
// active mode must survive every one: it proceeds on what the mode stored (or
// the harness default), never errors, and names the problem once in Warning.
var brokenPolicies = map[string]func(t *testing.T, path string){
	"invalid json":    func(t *testing.T, path string) { writeFile(t, path, `{"usage_tiers":`) },
	"empty":           func(t *testing.T, path string) { writeFile(t, path, ``) },
	"wrong schema":    func(t *testing.T, path string) { writeFile(t, path, `{"roles":{}}`) },
	"not an object":   func(t *testing.T, path string) { writeFile(t, path, `[1,2]`) },
	"huge":            func(t *testing.T, path string) { writeFile(t, path, strings.Repeat(" ", int(MaxFileBytes)+1)) },
	"unreadable file": func(t *testing.T, path string) { _ = os.Remove(path); _ = os.Mkdir(path, 0o755) },
	"missing":         func(t *testing.T, path string) { _ = os.Remove(path) },
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLaunchUnderAModeSurvivesABrokenPolicy(t *testing.T) {
	for name, breakPolicy := range brokenPolicies {
		t.Run(name, func(t *testing.T) {
			p := newLivePolicy(t, `{"sol":"luna"}`, `[]`)
			intent, err := NewMode(p.tiers(), NewModeOptions{Now: t0.Add(-time.Minute), Scope: Scope{Runner: "codex"}})
			if err != nil {
				t.Fatal(err)
			}
			if err := Write(File{Modes: []Mode{intent}}); err != nil {
				t.Fatal(err)
			}
			breakPolicy(t, p.path)
			args := []string{"-C", "x"}
			plan := LaunchPlan("codex", args, nil, t0)
			if plan.Warning == "" {
				t.Fatal("a broken policy was not reported")
			}
			if strings.Contains(strings.Join(plan.Args, " "), "model=sol") {
				t.Fatalf("a broken policy raised the launch: %v", plan.Args)
			}
			if got := PolicyProblem("codex"); got != plan.Warning {
				t.Fatalf("PolicyProblem = %q, plan warning = %q", got, plan.Warning)
			}
		})
	}
}

func TestLaunchWithAHealthyPolicyHasNoWarning(t *testing.T) {
	p := newLivePolicy(t, `{"sol":"luna"}`, `[]`)
	intent, err := NewMode(p.tiers(), NewModeOptions{Now: t0.Add(-time.Minute), Scope: Scope{Runner: "codex"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(File{Modes: []Mode{intent}}); err != nil {
		t.Fatal(err)
	}
	if plan := LaunchPlan("codex", nil, nil, t0); plan.Warning != "" {
		t.Fatalf("warning on a healthy policy: %q", plan.Warning)
	}
}
