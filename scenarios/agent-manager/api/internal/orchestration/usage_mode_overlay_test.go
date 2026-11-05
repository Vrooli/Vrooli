package orchestration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/rolepolicy"

	"github.com/vrooli/cli-core/usagemode"
)

type modeRoleResolver struct{ model string }

func (r modeRoleResolver) Resolve(_ context.Context, runnerType domain.RunnerType, role string) (rolepolicy.ResolvedRole, error) {
	return rolepolicy.ResolvedRole{Runner: runnerType, Role: role, Model: r.model, CanonicalModel: "openai/" + r.model,
		Billing: domain.BillingSnapshot{Mode: domain.BillingModeSubscription, Provider: "openai"}}, nil
}

type modeFixture struct {
	o      *Orchestrator
	probed []string
}

// newModeFixture builds an orchestrator whose code.default role resolves to
// gpt-6.1-terra on codex, with the usage mode file in a private VROOLI_HOME.
func newModeFixture(t *testing.T, probe func(string) error) *modeFixture {
	t.Helper()
	t.Setenv("VROOLI_HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "role-policy.json")
	const catalog = `{"schemaVersion":1,"metadata":{"catalogId":"usage-mode-test","updatedAt":"2026-10-30"},"defaultRole":"code.default","roles":{"code.default":{"description":"test","intent":"test","candidates":[{"runner":"codex","resourceRole":"code.default"}]},"code.review":{"description":"test","intent":"test","candidates":[{"runner":"codex","resourceRole":"code.review"}]}}}`
	if err := os.WriteFile(path, []byte(catalog), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := rolepolicy.NewState(path, rolepolicy.Requirement{Required: true})
	if err != nil {
		t.Fatal(err)
	}
	f := &modeFixture{}
	mock := runner.NewMockRunner(domain.RunnerTypeCodex)
	mock.ProbeFunc = func(_ context.Context, model string) error {
		f.probed = append(f.probed, model)
		if probe != nil {
			return probe(model)
		}
		return nil
	}
	registry := runner.NewRegistry()
	if err := registry.Register(mock); err != nil {
		t.Fatal(err)
	}
	f.o = &Orchestrator{rolePolicy: state, roleResolver: modeRoleResolver{model: "gpt-6.1-terra"}, runners: registry, usageModes: true}
	return f
}

func writeMode(t *testing.T, mutate func(*usagemode.Mode)) usagemode.Mode {
	t.Helper()
	now := time.Now()
	expires := now.Add(2 * time.Hour)
	mode := usagemode.Mode{
		ID: "um-test-aaaaaa", StartedAt: now.Add(-time.Minute), Source: "agent-manager",
		Until: usagemode.Until{Kind: usagemode.UntilTime}, ExpiresAt: &expires,
		Runners: map[string]usagemode.RunnerSettings{"codex": {ModelMap: map[string]string{"gpt-6.1-terra": "gpt-6-luna"}, SessionModel: "gpt-6-luna"}},
	}
	if mutate != nil {
		mutate(&mode)
	}
	if err := usagemode.Write(usagemode.File{Modes: []usagemode.Mode{mode}}); err != nil {
		t.Fatal(err)
	}
	return mode
}

func (f *modeFixture) create(t *testing.T, role string, req CreateRunRequest, profile *domain.AgentProfile) *domain.RunConfig {
	t.Helper()
	ctx := context.Background()
	cfg := domain.DefaultRunConfig()
	cfg.RoleRef = role
	if err := f.o.resolveExecutionPolicy(ctx, cfg); err != nil {
		t.Fatalf("resolve policy: %v", err)
	}
	req.usageModeEligible = true
	f.o.applyUsageMode(ctx, cfg, profile, req)
	if err := f.o.applyModelOverride(ctx, cfg, req.Model); err != nil {
		t.Fatalf("model override: %v", err)
	}
	return cfg
}

func requireLowered(t *testing.T, cfg *domain.RunConfig, mode usagemode.Mode) {
	t.Helper()
	selected := cfg.PolicySnapshot.SelectedCandidate
	if cfg.Model != "gpt-6-luna" || selected.Model != "gpt-6-luna" || cfg.PolicySnapshot.Candidates[cfg.PolicySnapshot.SelectedIndex].Model != "gpt-6-luna" {
		t.Fatalf("model = %q selected = %q, want gpt-6-luna everywhere", cfg.Model, selected.Model)
	}
	if selected.CanonicalModel != "" {
		t.Fatalf("canonical %q still names the replaced model", selected.CanonicalModel)
	}
	if !strings.Contains(cfg.PolicySnapshot.SelectionReason, "usage_mode:"+mode.ID) {
		t.Fatalf("selection reason %q must record usage_mode:%s", cfg.PolicySnapshot.SelectionReason, mode.ID)
	}
	if selected.Billing.Mode != domain.BillingModeSubscription || selected.Billing.Provider != "openai" {
		t.Fatalf("billing basis changed: %+v", selected.Billing)
	}
}

func TestUsageModeLowersAMappedModel(t *testing.T) {
	f := newModeFixture(t, nil)
	mode := writeMode(t, nil)
	requireLowered(t, f.create(t, "code.default", CreateRunRequest{}, nil), mode)
}

func TestUsageModeLeavesUnmappedModelsAlone(t *testing.T) {
	f := newModeFixture(t, nil)
	f.o.roleResolver = modeRoleResolver{model: "gpt-6-luna"}
	writeMode(t, nil)
	cfg := f.create(t, "code.default", CreateRunRequest{}, nil)
	if cfg.Model != "gpt-6-luna" || strings.Contains(cfg.PolicySnapshot.SelectionReason, "usage_mode") {
		t.Fatalf("unmapped model changed: model=%q reason=%q", cfg.Model, cfg.PolicySnapshot.SelectionReason)
	}
}

func TestUsageModeSkipsExemptRoles(t *testing.T) {
	f := newModeFixture(t, nil)
	writeMode(t, nil)
	cfg := f.create(t, "code.review", CreateRunRequest{}, nil)
	if cfg.Model != "gpt-6.1-terra" || strings.Contains(cfg.PolicySnapshot.SelectionReason, "usage_mode") {
		t.Fatalf("exempt role lowered: model=%q reason=%q", cfg.Model, cfg.PolicySnapshot.SelectionReason)
	}
	// A mode scoped to the role lists it explicitly.
	mode := writeMode(t, func(m *usagemode.Mode) { m.Scope.Role = "code.review" })
	requireLowered(t, f.create(t, "code.review", CreateRunRequest{}, nil), mode)
}

func TestExplicitModelStillWins(t *testing.T) {
	f := newModeFixture(t, nil)
	writeMode(t, nil)
	explicit := "gpt-6-sol"
	cfg := f.create(t, "code.default", CreateRunRequest{Model: &explicit}, nil)
	if cfg.Model != "gpt-6-sol" || strings.Contains(cfg.PolicySnapshot.SelectionReason, "usage_mode") {
		t.Fatalf("explicit model lost to the mode: model=%q reason=%q", cfg.Model, cfg.PolicySnapshot.SelectionReason)
	}
}

func TestUsageModeKeepsOriginalCandidateWhenProbeFails(t *testing.T) {
	f := newModeFixture(t, func(model string) error {
		if model == "gpt-6-luna" {
			return context.DeadlineExceeded
		}
		return nil
	})
	writeMode(t, nil)
	cfg := f.create(t, "code.default", CreateRunRequest{}, nil)
	if cfg.Model != "gpt-6.1-terra" || cfg.PolicySnapshot.SelectedCandidate.Model != "gpt-6.1-terra" {
		t.Fatalf("a rejected target replaced the candidate: %q", cfg.Model)
	}
	if !strings.Contains(cfg.PolicySnapshot.SelectionReason, "usage_mode_skipped:probe_failed") || strings.Contains(cfg.PolicySnapshot.SelectionReason, ";usage_mode:") {
		t.Fatalf("selection reason = %q", cfg.PolicySnapshot.SelectionReason)
	}
}

func TestUsageModeNeverLowersMeteredCandidatesOrExcludedTargets(t *testing.T) {
	f := newModeFixture(t, nil)
	writeMode(t, nil)
	f.o.roleResolver = modeRoleResolver{model: "gpt-6.1-terra"}
	cfg := domain.DefaultRunConfig()
	cfg.RoleRef = "code.default"
	if err := f.o.resolveExecutionPolicy(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	snapshot := cfg.PolicySnapshot
	snapshot.Candidates[0].Billing.Mode, snapshot.Candidates[0].Billing.Basis = domain.BillingModeMetered, domain.ChargeBasisMetered
	snapshot.SelectedCandidate = snapshot.Candidates[0]
	f.o.applyUsageMode(context.Background(), cfg, nil, CreateRunRequest{usageModeEligible: true})
	if cfg.Model != "gpt-6.1-terra" || !strings.Contains(snapshot.SelectionReason, "usage_mode_skipped:metered_billing") {
		t.Fatalf("metered candidate lowered: %q %q", cfg.Model, snapshot.SelectionReason)
	}

	snapshot.SelectionReason = ""
	snapshot.Candidates[0].Billing = domain.BillingSnapshot{Mode: domain.BillingModeSubscription}
	snapshot.Candidates[0].ExcludedModels = []string{"gpt-6-luna"}
	snapshot.SelectedCandidate = snapshot.Candidates[0]
	f.o.applyUsageMode(context.Background(), cfg, nil, CreateRunRequest{usageModeEligible: true})
	if cfg.Model != "gpt-6.1-terra" || !strings.Contains(snapshot.SelectionReason, "usage_mode_skipped:target_excluded") {
		t.Fatalf("excluded target chosen: %q %q", cfg.Model, snapshot.SelectionReason)
	}
}

func TestNoExpiredOrDisabledModeChangesAnything(t *testing.T) {
	f := newModeFixture(t, nil)
	if cfg := f.create(t, "code.default", CreateRunRequest{}, nil); cfg.Model != "gpt-6.1-terra" {
		t.Fatalf("no mode: model = %q", cfg.Model)
	}
	writeMode(t, func(m *usagemode.Mode) {
		past := time.Now().Add(-time.Minute)
		m.StartedAt, m.ExpiresAt = past.Add(-time.Hour), &past
	})
	if cfg := f.create(t, "code.default", CreateRunRequest{}, nil); cfg.Model != "gpt-6.1-terra" || strings.Contains(cfg.PolicySnapshot.SelectionReason, "usage_mode") {
		t.Fatalf("expired mode applied: %q %q", cfg.Model, cfg.PolicySnapshot.SelectionReason)
	}
	mode := writeMode(t, nil)
	f.o.usageModes = false
	if cfg := f.create(t, "code.default", CreateRunRequest{}, nil); cfg.Model != "gpt-6.1-terra" {
		t.Fatalf("an orchestrator without usage modes read the file: %q", cfg.Model)
	}
	f.o.usageModes = true
	requireLowered(t, f.create(t, "code.default", CreateRunRequest{}, nil), mode)
}

func TestUsageModeScopeByRoleAndProfile(t *testing.T) {
	f := newModeFixture(t, nil)
	profile := &domain.AgentProfile{ProfileKey: "delivery"}
	for name, tc := range map[string]struct {
		scope   usagemode.Scope
		role    string
		profile *domain.AgentProfile
		lowered bool
	}{
		"profile match":    {usagemode.Scope{Profile: "delivery"}, "code.default", profile, true},
		"profile mismatch": {usagemode.Scope{Profile: "delivery"}, "code.default", &domain.AgentProfile{ProfileKey: "other"}, false},
		"profile absent":   {usagemode.Scope{Profile: "delivery"}, "code.default", nil, false},
		"role match":       {usagemode.Scope{Role: "code.default"}, "code.default", nil, true},
		"role mismatch":    {usagemode.Scope{Role: "code.smart"}, "code.default", nil, false},
		"runner mismatch":  {usagemode.Scope{Runner: "claude"}, "code.default", nil, false},
		"runner match":     {usagemode.Scope{Runner: "codex"}, "code.default", nil, true},
	} {
		mode := writeMode(t, func(m *usagemode.Mode) { m.Scope = tc.scope })
		cfg := f.create(t, tc.role, CreateRunRequest{}, tc.profile)
		if got := cfg.Model == "gpt-6-luna"; got != tc.lowered {
			t.Errorf("%s: lowered = %v, want %v (model %q)", name, got, tc.lowered, cfg.Model)
			continue
		}
		if tc.lowered {
			requireLowered(t, cfg, mode)
		}
	}
}

func TestUsageModeLowersEffortOnlyWhenLowerAndNotExplicit(t *testing.T) {
	f := newModeFixture(t, nil)
	f.o.roleResolver = modeRoleResolver{model: "gpt-6-luna"}
	mode := writeMode(t, func(m *usagemode.Mode) {
		s := m.Runners["codex"]
		s.Effort = "low"
		m.Runners["codex"] = s
	})
	ctx := context.Background()
	for name, tc := range map[string]struct {
		current  domain.Effort
		explicit bool
		want     domain.Effort
	}{
		"higher is lowered": {domain.EffortHigh, false, domain.EffortLow},
		"unset is set":      {"", false, domain.EffortLow},
		"explicit wins":     {domain.EffortHigh, true, domain.EffortHigh},
		"lower is kept":     {domain.EffortLow, false, domain.EffortLow},
	} {
		cfg := domain.DefaultRunConfig()
		cfg.RoleRef, cfg.Effort = "code.default", tc.current
		if err := f.o.resolveExecutionPolicy(ctx, cfg); err != nil {
			t.Fatal(err)
		}
		req := CreateRunRequest{usageModeEligible: true}
		if tc.explicit {
			effort := tc.current
			req.Effort = &effort
		}
		f.o.applyUsageMode(ctx, cfg, nil, req)
		if cfg.Effort != tc.want {
			t.Errorf("%s: effort = %q, want %q", name, cfg.Effort, tc.want)
		}
		_ = mode
	}
}

func TestResumeAndRecoveryAreNotEligibleForUsageModes(t *testing.T) {
	f := newModeFixture(t, nil)
	writeMode(t, nil)
	cfg := domain.DefaultRunConfig()
	cfg.RoleRef = "code.default"
	if err := f.o.resolveExecutionPolicy(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	// resolveRunConfig(CreateRunRequest{AgentProfileID}) in resume/explain paths never sets usageModeEligible.
	f.o.applyUsageMode(context.Background(), cfg, nil, CreateRunRequest{})
	if cfg.Model != "gpt-6.1-terra" || strings.Contains(cfg.PolicySnapshot.SelectionReason, "usage_mode") {
		t.Fatalf("a request that is not a new run was lowered: %q %q", cfg.Model, cfg.PolicySnapshot.SelectionReason)
	}
}

func TestResolveRunConfigAppliesTheOverlayBeforeAnExplicitModel(t *testing.T) {
	f := newModeFixture(t, nil)
	mode := writeMode(t, nil)
	role := "code.default"
	ctx := context.Background()

	cfg, _, err := f.o.resolveRunConfig(ctx, CreateRunRequest{RoleRef: &role, usageModeEligible: true})
	if err != nil {
		t.Fatal(err)
	}
	requireLowered(t, cfg, mode)

	explicit := "gpt-6-sol"
	cfg, _, err = f.o.resolveRunConfig(ctx, CreateRunRequest{RoleRef: &role, Model: &explicit, usageModeEligible: true})
	if err != nil || cfg.Model != "gpt-6-sol" || cfg.PolicySnapshot.SelectedCandidate.Model != "gpt-6-sol" {
		t.Fatalf("explicit model through resolveRunConfig: model=%v err=%v", cfg, err)
	}

	cfg, _, err = f.o.resolveRunConfig(ctx, CreateRunRequest{RoleRef: &role})
	if err != nil || cfg.Model != "gpt-6.1-terra" {
		t.Fatalf("a request not marked as a new run was lowered: %v err=%v", cfg, err)
	}
}
