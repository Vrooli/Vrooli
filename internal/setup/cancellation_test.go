package setup

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/privilegebroker"
	"github.com/vrooli/vrooli/internal/projectstate"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
)

func TestSetupDeclineAndCancellationNeverComplete(t *testing.T) {
	for _, tc := range []struct {
		name            string
		sudoMode        string
		cancelBroker    bool
		ensureErr       error
		wantBrokerCalls int
		wantEnsureCalls int
		wantRunError    error
	}{
		{name: "declined", sudoMode: "skip", ensureErr: hostreqkit.ErrSudoSkipped, wantBrokerCalls: 1, wantEnsureCalls: 1, wantRunError: hostreqkit.ErrSudoSkipped},
		{name: "interrupted during broker", sudoMode: "ask", cancelBroker: true, wantBrokerCalls: 1, wantRunError: context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := stubSetupDeps(t)
			root, home := t.TempDir(), t.TempDir()
			projectScenario := writeProjectFixture(t, root)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			svc.deps.currentHost = func() vrooliruntime.Host {
				return vrooliruntime.Host{SupportsSetup: true, SupportsDevelop: true}
			}
			svc.deps.loadProject = func(string) (scenario.Scenario, error) { return projectScenario, nil }
			svc.deps.resolveHostRequirements = func(string, string, hostreq.ResolveOptions) (hostreq.Resolution, error) {
				return hostreq.Resolution{}, nil
			}
			svc.deps.ensureBootstrapTools = func(_ string, opts vrooliruntime.EnsureOptions) error {
				if opts.Context != ctx {
					t.Fatal("bootstrap tools did not receive setup context")
				}
				return nil
			}
			brokerCalls, ensureCalls, completionCalls := 0, 0, 0
			svc.deps.installPrivilegeBroker = func(got context.Context, _ string, opts Options, _, _ io.Writer) (privilegebroker.SetupStatus, error) {
				brokerCalls++
				if got != ctx {
					t.Fatal("privilege broker did not receive setup context")
				}
				if tc.cancelBroker {
					if runtime.GOOS == "linux" && opts.setupPrivilegeSession == nil {
						t.Fatal("setup privilege session was not shared with broker installation")
					}
					cancel()
					return privilegebroker.SetupStatus{}, got.Err()
				}
				return privilegebroker.SetupStatus{Supported: true, Available: true}, nil
			}
			svc.deps.ensureRequirements = func(opts vrooliruntime.EnsureOptions, _ hostreq.Resolution) (vrooliruntime.Report, error) {
				ensureCalls++
				if opts.Context != ctx {
					t.Fatal("host requirements did not receive setup context")
				}
				return vrooliruntime.Report{}, tc.ensureErr
			}
			svc.deps.markComplete = func(string, string) error {
				completionCalls++
				return nil
			}

			resultPath := filepath.Join(t.TempDir(), "setup-result.json")
			err := svc.RunSetupWithOptions(root, home, Options{
				Context: ctx, Resources: "none", ResultPath: resultPath, SudoMode: tc.sudoMode,
			}, io.Discard, io.Discard)
			if !errors.Is(err, tc.wantRunError) {
				t.Fatalf("RunSetupWithOptions error = %v, want %v", err, tc.wantRunError)
			}
			if brokerCalls != tc.wantBrokerCalls || ensureCalls != tc.wantEnsureCalls || completionCalls != 0 {
				t.Fatalf("broker=%d requirements=%d completion=%d, want %d/%d/0", brokerCalls, ensureCalls, completionCalls, tc.wantBrokerCalls, tc.wantEnsureCalls)
			}
			result := testkitgo.ReadJSONFile(t, resultPath)
			if result["status"] != SetupStatusFailed {
				t.Fatalf("setup result status = %v, want failed", result["status"])
			}
			locator, err := projectstate.NewLocator(home, root)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(locator.BootstrapCompletePath()); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("completion marker stat error = %v, want no marker", err)
			}
		})
	}
}
