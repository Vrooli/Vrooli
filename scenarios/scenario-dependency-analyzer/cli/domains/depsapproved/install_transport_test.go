package depsapproved

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	governanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance/dependency_governance_v1connect"
)

func TestInstallFlagsReachTypedAPIWithApplyModes(t *testing.T) {
	for _, tc := range []struct {
		name                string
		args                []string
		apply, dev, refresh bool
	}{
		{"development", []string{"npm/coverage-fixture@1.2.3", "--scenario", "demo", "--surface", "ui", "--dev", "--exact"}, false, true, false},
		{"source-lock-refresh", []string{"npm/hyperframes@0.8.112", "--scenario", "hyperframes", "--surface", "host-tool", "--refresh-lock", "--exact"}, false, false, true},
		{"frozen-host-tool", []string{"npm/hyperframes@0.8.112", "--scenario", "hyperframes", "--surface", "host-tool", "--exact"}, false, false, false},
		{"default-placement", []string{"npm/coverage-fixture@1.2.3", "--scenario", "demo", "--surface", "ui", "--exact"}, false, false, false},
		{"explicit-apply", []string{"npm/coverage-fixture@1.2.3", "--scenario", "demo", "--surface", "ui", "--exact", "--apply"}, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			captured := make(chan *governancev1.InstallDependencyRequest, 1)
			budgets := make(chan time.Duration, 1)
			handler := connect.NewUnaryHandler(governanceconnect.DependencyGovernanceServiceInstallDependencyProcedure,
				func(ctx context.Context, req *connect.Request[governancev1.InstallDependencyRequest]) (*connect.Response[governancev1.InstallDependencyResponse], error) {
					deadline, _ := ctx.Deadline()
					budgets <- time.Until(deadline)
					captured <- req.Msg
					return connect.NewResponse(&governancev1.InstallDependencyResponse{DryRun: !req.Msg.GetApply(), Installed: req.Msg.GetApply(), Verdict: "approved"}), nil
				})
			server := httptest.NewServer(handler)
			defer server.Close()
			t.Setenv("SDA_INSTALL_TEST_CONFIG_DIR", t.TempDir())
			app, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
				Name: "sda-install-fixture", DefaultAPIBase: server.URL,
				ConfigDirEnvVars: []string{"SDA_INSTALL_TEST_CONFIG_DIR"}, AllowAnonymous: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			ctx, err := cliapp.NewTestRunContextFromArgs(installArgumentSchema(), tc.args, app, io.Discard, io.Discard)
			if err != nil {
				t.Fatalf("parse install arguments: %v", err)
			}
			if err := runInstall(app, ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case msg := <-captured:
				if msg.GetApply() != tc.apply || !msg.GetExactVersion() || msg.GetDevelopment() != tc.dev || msg.GetRefreshLock() != tc.refresh || msg.GetVersion() == "" {
					t.Fatalf("flags lost at RPC boundary: %v", msg)
				}
				budget := <-budgets
				want := 45 * time.Second
				if msg.GetSurface() == "host-tool" {
					want = 3*time.Minute + 15*time.Second
				}
				if budget <= want-time.Second || budget > want {
					t.Fatalf("RPC budget = %v, want near %v", budget, want)
				}
			default:
				t.Fatal("CLI did not reach InstallDependency")
			}
		})
	}
}

func TestInstallHelpShowsSchemaArgumentsWithoutCallingAPI(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		http.NotFound(w, r)
	}))
	defer server.Close()
	t.Setenv("SDA_INSTALL_TEST_CONFIG_DIR", t.TempDir())
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name: "scenario-dependency-analyzer", DefaultAPIBase: server.URL,
		ConfigDirEnvVars: []string{"SDA_INSTALL_TEST_CONFIG_DIR"}, AllowAnonymous: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	core.SetCommandsWithSubgroups(nil, []cliapp.SubcommandGroup{RegisterInstall(core)})

	var stdout, stderr strings.Builder
	if err := core.CLI.RunWithWriters([]string{"deps", "install", "--help"}, &stdout, &stderr); err != nil {
		t.Fatalf("help: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"scenario-dependency-analyzer deps install <ecosystem>/<package>[@version] --scenario <name> --surface <surface> [options]",
		"<dependency>",
		"--scenario",
		"--surface",
		"--version",
		"--apply",
		"--exact",
		"--dev",
		"--refresh-lock",
		"--json",
		"default is a dry-run preview",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("help missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "[dependency...") {
		t.Fatalf("help presents dependency as optional/repeated:\n%s", got)
	}
	if requests != 0 {
		t.Fatalf("help made %d API request(s), want none", requests)
	}
}

func TestInstallArgumentSchemaUsesCanonicalParser(t *testing.T) {
	ctx, err := cliapp.NewTestRunContextFromArgs(installArgumentSchema(), []string{
		"--scenario=demo", "npm/pkg@1.2.3", "--surface", "ui", "--apply", "--json",
	}, nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("parse install arguments: %v", err)
	}
	if got := ctx.Positional("dependency"); got != "npm/pkg@1.2.3" {
		t.Fatalf("dependency positional = %q", got)
	}
	if ctx.Flag("scenario") != "demo" || ctx.Flag("surface") != "ui" || !ctx.BoolFlag("apply") || !ctx.JSON() {
		t.Fatalf("parsed install args lost values: scenario=%q surface=%q apply=%t json=%t", ctx.Flag("scenario"), ctx.Flag("surface"), ctx.BoolFlag("apply"), ctx.JSON())
	}
}

func TestInstallArgumentSchemaRequiresExactlyOneDependencyBeforeAPI(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		http.NotFound(w, r)
	}))
	defer server.Close()
	t.Setenv("SDA_INSTALL_TEST_CONFIG_DIR", t.TempDir())
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name: "scenario-dependency-analyzer", DefaultAPIBase: server.URL,
		ConfigDirEnvVars: []string{"SDA_INSTALL_TEST_CONFIG_DIR"}, AllowAnonymous: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "missing", args: []string{"--scenario", "demo", "--surface", "ui"}},
		{name: "extra", args: []string{"npm/one", "npm/two", "--scenario", "demo", "--surface", "ui"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := cliapp.NewTestRunContextFromArgs(installArgumentSchema(), tc.args, core, io.Discard, io.Discard); err == nil {
				t.Fatal("expected schema cardinality error")
			}
		})
	}
	if requests != 0 {
		t.Fatalf("invalid positional arguments made %d API request(s), want none", requests)
	}
}
