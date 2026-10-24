package depsapproved

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	governanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance/dependency_governance_v1connect"
)

func TestInstallFlagsReachTypedAPIWithoutApplyingByDefault(t *testing.T) {
	for _, tc := range []struct {
		name         string
		args         []string
		dev, refresh bool
	}{
		{"development", []string{"npm/coverage-fixture@1.2.3", "--scenario", "demo", "--surface", "ui", "--dev", "--exact"}, true, false},
		{"source-lock-refresh", []string{"npm/hyperframes@0.8.112", "--scenario", "hyperframes", "--surface", "host-tool", "--refresh-lock", "--exact"}, false, true},
		{"frozen-host-tool", []string{"npm/hyperframes@0.8.112", "--scenario", "hyperframes", "--surface", "host-tool", "--exact"}, false, false},
		{"default-placement", []string{"npm/coverage-fixture@1.2.3", "--scenario", "demo", "--surface", "ui", "--exact"}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			captured := make(chan *governancev1.InstallDependencyRequest, 1)
			budgets := make(chan time.Duration, 1)
			handler := connect.NewUnaryHandler(governanceconnect.DependencyGovernanceServiceInstallDependencyProcedure,
				func(ctx context.Context, req *connect.Request[governancev1.InstallDependencyRequest]) (*connect.Response[governancev1.InstallDependencyResponse], error) {
					deadline, _ := ctx.Deadline()
					budgets <- time.Until(deadline)
					captured <- req.Msg
					return connect.NewResponse(&governancev1.InstallDependencyResponse{DryRun: true, Verdict: "approved"}), nil
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
			if err := runInstall(app, tc.args); err != nil {
				t.Fatal(err)
			}
			select {
			case msg := <-captured:
				if msg.GetApply() || !msg.GetExactVersion() || msg.GetDevelopment() != tc.dev || msg.GetRefreshLock() != tc.refresh || msg.GetVersion() == "" {
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
