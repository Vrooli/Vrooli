package lifecycle

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

func hybridHub() scenario.Scenario {
	return scenario.Scenario{Slug: "notification-hub", Manifest: scenario.ServiceManifest{Authentication: &scenario.AuthenticationProfile{
		Profile:        "hybrid",
		DefaultMode:    "shared_provider",
		SupportedModes: []string{"personal_local", "local_multi_user", "remote_vrooli", "shared_provider"},
		TeamDomain:     "${VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN}",
		Audience:       "${VROOLI_CLOUDFLARE_ACCESS_AUDIENCE}",
		Owner:          "tunnel-manager",
	}}}
}

func managedBindingRunner() *Runner {
	return &Runner{AuthBindings: &fakeAuthenticationBindingResolver{binding: scenario.AuthenticationBinding{
		TeamDomain: "https://team.example.test", Audience: "hub-audience",
	}}}
}

// The 2026-11-14 incident: an agent shell exporting personal_local restarted a
// shared_provider scenario, and the restarted processes treated every loopback
// caller as the owner.
func TestApplyManifestAuthenticationIgnoresCallerAuthMode(t *testing.T) {
	t.Setenv("VROOLI_AUTH_MODE", "personal_local")
	t.Setenv("VROOLI_AUTH_PROVIDERS", "")
	env := map[string]string{}
	if err := managedBindingRunner().applyManifestAuthentication(context.Background(), hybridHub(), env); err != nil {
		t.Fatalf("applyManifestAuthentication: %v", err)
	}
	if env["VROOLI_AUTH_MODE"] != "shared_provider" || env["VROOLI_AUTH_MODE_SOURCE"] != AuthModeSourceManifest {
		t.Fatalf("mode = %q source = %q, want the manifest default shared_provider", env["VROOLI_AUTH_MODE"], env["VROOLI_AUTH_MODE_SOURCE"])
	}
	if env["VROOLI_AUTH_PROVIDERS"] != "cloudflare_access,scenario_authenticator" {
		t.Fatalf("providers = %q, want the shared-provider chain", env["VROOLI_AUTH_PROVIDERS"])
	}
	stepEnv := lifecycleStepEnvFrom(os.Environ(), phasesDevelop, env)
	for _, entry := range stepEnv {
		if entry == "VROOLI_AUTH_MODE=personal_local" {
			t.Fatal("the caller's personal_local reached the scenario step environment")
		}
	}
}

// Placeholders resolve from the lifecycle environment and the owning control
// plane, never from a caller's exported Cloudflare audience.
func TestApplyManifestAuthenticationDoesNotResolvePlaceholdersFromCaller(t *testing.T) {
	t.Setenv("VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN", "https://attacker.example.test")
	t.Setenv("VROOLI_CLOUDFLARE_ACCESS_AUDIENCE", "attacker-audience")
	err := (&Runner{}).applyManifestAuthentication(context.Background(), hybridHub(), map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "unresolved placeholder VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN") {
		t.Fatalf("error = %v, want unresolved placeholders when only the caller supplies them", err)
	}
}

func TestResolveAuthModeIsTheDeclaredDefault(t *testing.T) {
	t.Setenv("VROOLI_AUTH_MODE", "personal_local")
	profile := &scenario.AuthenticationProfile{Profile: "hybrid", DefaultMode: "shared_provider"}
	if got := ResolveAuthMode(profile); got != (AuthModeResolution{Mode: "shared_provider", Source: AuthModeSourceManifest}) {
		t.Fatalf("resolution = %#v, want the manifest default regardless of the caller", got)
	}
	if got := ResolveAuthMode(nil); got != (AuthModeResolution{}) {
		t.Fatalf("resolution without a profile = %#v", got)
	}
}

func TestDescribeAuthenticationReportsSourceAndMismatch(t *testing.T) {
	environs := map[int]string{
		101: "PATH=/bin\x00VROOLI_AUTH_MODE=shared_provider\x00VROOLI_AUTH_MODE_SOURCE=manifest-default\x00VROOLI_AUTH_PROVIDERS=cloudflare_access,scenario_authenticator\x00",
		102: "PATH=/bin\x00VROOLI_AUTH_MODE=personal_local\x00",
	}
	previous := readProcessEnviron
	readProcessEnviron = func(pid int) ([]byte, error) {
		if data, ok := environs[pid]; ok {
			return []byte(data), nil
		}
		return nil, fmt.Errorf("no such process")
	}
	t.Cleanup(func() { readProcessEnviron = previous })

	status := DescribeAuthentication(hybridHub(), []process.Record{
		{Step: "start-api", PID: 101}, {Step: "start-ui", PID: 102}, {Step: "gone", PID: 103},
	})
	if status == nil || status.Configured != (AuthModeResolution{Mode: "shared_provider", Source: AuthModeSourceManifest}) {
		t.Fatalf("configured = %#v", status)
	}
	if len(status.Processes) != 3 || status.Processes[0].Source != AuthModeSourceManifest || status.Processes[1].Mode != "personal_local" || status.Processes[1].Source != "" || status.Processes[2].Error == "" {
		t.Fatalf("processes = %#v", status.Processes)
	}
	if !status.Mismatch {
		t.Fatal("a process carrying personal_local under a shared_provider configuration must be reported as a mismatch")
	}
	if DescribeAuthentication(scenario.Scenario{Slug: "plain"}, nil) != nil {
		t.Fatal("a scenario without an authentication profile has no authentication status")
	}
}

func personalLocalProfile(mutate func(*scenario.AuthenticationProfile)) *scenario.AuthenticationProfile {
	profile := &scenario.AuthenticationProfile{
		Profile: "hybrid", DefaultMode: AuthModePersonalLocal,
		SupportedModes: []string{AuthModePersonalLocal, "shared_provider"},
		TeamDomain:     "${VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN}", Audience: "${VROOLI_CLOUDFLARE_ACCESS_AUDIENCE}", Owner: "tunnel-manager",
	}
	if mutate != nil {
		mutate(profile)
	}
	return profile
}

// personal_local is refused where the manifest declares that callers are
// verified remotely, and otherwise accepted with its risk named.
func TestCheckPersonalLocal(t *testing.T) {
	cases := []struct {
		name     string
		profile  *scenario.AuthenticationProfile
		mode     string
		refusal  string
		wantRisk bool
	}{
		{name: "tunnel-owned hybrid profile is accepted with its risk", profile: personalLocalProfile(nil), mode: AuthModePersonalLocal, wantRisk: true},
		{name: "cloudflare_access profile names Access as its only verifier", profile: personalLocalProfile(func(p *scenario.AuthenticationProfile) { p.Profile = "cloudflare_access" }), mode: AuthModePersonalLocal, refusal: "cloudflare_access"},
		{name: "declared public hostname serves remote callers", profile: personalLocalProfile(func(p *scenario.AuthenticationProfile) { p.Hostname = "hub.example.test" }), mode: AuthModePersonalLocal, refusal: `public hostname "hub.example.test"`},
		{name: "shared_provider carries no personal_local risk", profile: personalLocalProfile(func(p *scenario.AuthenticationProfile) { p.Hostname = "hub.example.test" }), mode: "shared_provider"},
		{name: "no profile", mode: AuthModePersonalLocal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			risk, err := CheckPersonalLocal(tc.profile, tc.mode)
			if tc.refusal != "" {
				if err == nil || !strings.Contains(err.Error(), tc.refusal) || !strings.Contains(err.Error(), PersonalLocalRisk) {
					t.Fatalf("err = %v, want a refusal naming %q and the risk", err, tc.refusal)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected refusal: %v", err)
			}
			if (risk != "") != tc.wantRisk {
				t.Fatalf("risk = %q, want risk %v", risk, tc.wantRisk)
			}
		})
	}
}

func TestApplyManifestAuthenticationRefusesPersonalLocalForPublicHostname(t *testing.T) {
	item := scenario.Scenario{Slug: "exposed", Manifest: scenario.ServiceManifest{Authentication: personalLocalProfile(func(p *scenario.AuthenticationProfile) { p.Hostname = "exposed.example.test" })}}
	env := map[string]string{}
	err := managedBindingRunner().applyManifestAuthentication(context.Background(), item, env)
	if err == nil || !strings.Contains(err.Error(), "scenario exposed") || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("err = %v, want the start refused", err)
	}
	if _, ok := env["VROOLI_AUTH_MODE"]; ok {
		t.Fatalf("a refused start must not bind an authentication environment: %v", env)
	}
}

func TestApplyManifestAuthenticationWarnsWhenStartingPersonalLocal(t *testing.T) {
	var logs bytes.Buffer
	runner := managedBindingRunner()
	runner.Logger = slog.New(slog.NewTextHandler(&logs, nil))
	item := scenario.Scenario{Slug: "console", Manifest: scenario.ServiceManifest{Authentication: personalLocalProfile(nil)}}
	env := map[string]string{}
	if err := runner.applyManifestAuthentication(context.Background(), item, env); err != nil {
		t.Fatalf("applyManifestAuthentication: %v", err)
	}
	if env["VROOLI_AUTH_MODE"] != AuthModePersonalLocal {
		t.Fatalf("mode = %q", env["VROOLI_AUTH_MODE"])
	}
	out := logs.String()
	if !strings.Contains(out, "level=WARN") || !strings.Contains(out, "personal_local") || !strings.Contains(out, "local agents") || !strings.Contains(out, "scenario=console") {
		t.Fatalf("log = %q, want a WARN naming the scenario and the loopback-owner risk", out)
	}
	logs.Reset()
	if err := runner.applyManifestAuthentication(context.Background(), hybridHub(), map[string]string{}); err != nil {
		t.Fatalf("applyManifestAuthentication: %v", err)
	}
	if strings.Contains(logs.String(), "level=WARN") {
		t.Fatalf("a shared_provider start must not warn: %q", logs.String())
	}
}

func TestDescribeAuthenticationReportsPersonalLocalRiskAndRefusal(t *testing.T) {
	risky := DescribeAuthentication(scenario.Scenario{Manifest: scenario.ServiceManifest{Authentication: personalLocalProfile(nil)}}, nil)
	if risky.Risk != PersonalLocalRisk || risky.Refused != "" {
		t.Fatalf("status = %#v, want the personal_local risk", risky)
	}
	refused := DescribeAuthentication(scenario.Scenario{Manifest: scenario.ServiceManifest{Authentication: personalLocalProfile(func(p *scenario.AuthenticationProfile) { p.Profile = "cloudflare_access" })}}, nil)
	if refused.Refused == "" || refused.Risk != "" {
		t.Fatalf("status = %#v, want the refusal", refused)
	}
	if shared := DescribeAuthentication(hybridHub(), nil); shared.Risk != "" || shared.Refused != "" {
		t.Fatalf("status = %#v, want neither for shared_provider", shared)
	}
}

// No scenario in the repository declares a personal_local configuration the
// start gate refuses, so the gate cannot stop an existing scenario.
func TestRepositoryManifestsPassThePersonalLocalGate(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	items, err := scenario.Discover(root, scenario.SandboxEnv{})
	if err != nil {
		t.Fatalf("discover scenarios: %v", err)
	}
	checked := 0
	for _, item := range items {
		if item.Manifest.Authentication == nil {
			continue
		}
		checked++
		if _, err := CheckPersonalLocal(item.Manifest.Authentication, ResolveAuthMode(item.Manifest.Authentication).Mode); err != nil {
			t.Errorf("scenario %s: %v", item.Slug, err)
		}
	}
	if checked == 0 {
		t.Fatal("no scenario declares an authentication profile; the repository scan found nothing to check")
	}
}
