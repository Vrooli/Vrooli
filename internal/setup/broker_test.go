package setup

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/privilegebroker"
	vruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/setupprivilege"
)

func TestSetupBrokerTouchesOnlyBrokerAndNeverMarksProjectComplete(t *testing.T) {
	svc := stubSetupDeps(t)
	calls := 0
	var owned privilegebroker.SetupPrivilegeSession
	svc.deps.currentHost = func() vruntime.Host { return vruntime.Host{OS: "linux", SupportsSetup: true} }
	svc.deps.loadProject = func(string) (scenario.Scenario, error) { return scenario.Scenario{}, nil }
	inspections := 0
	svc.deps.inspectPrivilegeBroker = func() privilegebroker.SetupStatus {
		inspections++
		return privilegebroker.SetupStatus{}
	}
	svc.deps.osExecutable = nil
	svc.deps.elevationFacts = nil
	svc.deps.installPrivilegeBroker = func(_ context.Context, executable string, opts Options, _, _ io.Writer) (privilegebroker.SetupStatus, error) {
		calls++
		owned = opts.setupPrivilegeSession
		if executable != "/reviewed/vrooli" || opts.SudoMode != "ask" || opts.setupPrivilegeSession == nil {
			t.Fatal("lost fixed setup owner arguments")
		}
		return privilegebroker.SetupStatus{Supported: true, Available: true}, nil
	}
	svc.deps.resolveHostRequirements = func(string, string, hostreq.ResolveOptions) (hostreq.Resolution, error) {
		t.Fatal("broker-only setup resolved unrelated requirements")
		return hostreq.Resolution{}, nil
	}
	root, home := t.TempDir(), t.TempDir()
	out := &strings.Builder{}
	if err := svc.RunSetupWithOptions(root, home, Options{Subcommand: "broker", DryRun: true}, out, io.Discard); err != nil || calls != 0 || inspections != 1 {
		t.Fatalf("dry run err=%v calls=%d", err, calls)
	}
	svc.deps.osExecutable = func() (string, error) { return "/reviewed/vrooli", nil }
	svc.deps.elevationFacts = func() hostreqkit.ElevationFacts { return hostreqkit.ElevationFacts{Platform: "linux"} }
	if err := svc.RunSetupWithOptions(root, home, Options{Subcommand: "broker", SudoMode: "ask"}, out, io.Discard); err != nil || calls != 1 || inspections != 1 {
		t.Fatalf("apply err=%v calls=%d", err, calls)
	}
	t.Setenv("PATH", t.TempDir())
	if _, err := owned.InstallPrivilegeBroker(context.Background(), root); err == nil || !strings.Contains(err.Error(), "session is closed") {
		t.Fatalf("focused setup did not close its worker: %v", err)
	}
	if strings.Contains(out.String(), "SETUP") || strings.Contains(out.String(), "Configuration: complete") {
		t.Fatalf("broker update claimed project completion: %s", out)
	}
}

func TestSetupBrokerUnavailableIsNotSuccess(t *testing.T) {
	svc := stubSetupDeps(t)
	svc.deps.currentHost = func() vruntime.Host { return vruntime.Host{OS: "linux", SupportsSetup: true} }
	svc.deps.loadProject = func(string) (scenario.Scenario, error) { return scenario.Scenario{}, nil }
	svc.deps.osExecutable = func() (string, error) { return "/reviewed/vrooli", nil }
	for _, available := range []bool{false, true} {
		svc.deps.installPrivilegeBroker = func(context.Context, string, Options, io.Writer, io.Writer) (privilegebroker.SetupStatus, error) {
			return privilegebroker.SetupStatus{Supported: true, Available: available, UpgradeRequired: available}, nil
		}
		err := svc.RunSetupWithOptions(t.TempDir(), t.TempDir(), Options{Subcommand: "broker"}, io.Discard, io.Discard)
		if err == nil || !strings.Contains(err.Error(), privilegebroker.SetupBrokerRecoveryCommand) {
			t.Fatalf("available=%v recovery error=%v", available, err)
		}
	}
}

func TestSetupBrokerSessionOwnership(t *testing.T) {
	svc := stubSetupDeps(t)
	svc.deps.currentHost = func() vruntime.Host { return vruntime.Host{SupportsSetup: true} }
	svc.deps.loadProject = func(string) (scenario.Scenario, error) { return scenario.Scenario{}, nil }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	injected := setupprivilege.NewSession(ctx, "", io.Discard, io.Discard)
	defer injected.Close()
	for _, tc := range []struct {
		facts   hostreqkit.ElevationFacts
		mode    string
		session privilegebroker.SetupPrivilegeSession
	}{
		{hostreqkit.ElevationFacts{Platform: "linux"}, "ask", injected},
		{hostreqkit.ElevationFacts{Platform: "linux", Elevated: true}, "ask", nil},
		{hostreqkit.ElevationFacts{Platform: "darwin"}, "ask", nil},
		{hostreqkit.ElevationFacts{Platform: "windows"}, "ask", nil},
		{hostreqkit.ElevationFacts{Platform: "linux"}, "skip", nil},
	} {
		svc.deps.elevationFacts = func() hostreqkit.ElevationFacts { return tc.facts }
		svc.deps.installPrivilegeBroker = func(got context.Context, _ string, opts Options, _, _ io.Writer) (privilegebroker.SetupStatus, error) {
			if got != ctx || opts.setupPrivilegeSession != tc.session {
				t.Fatalf("unexpected session or context for %+v", tc)
			}
			return privilegebroker.SetupStatus{Available: true}, nil
		}
		opts := Options{Context: ctx, Subcommand: "broker", SudoMode: tc.mode, setupPrivilegeSession: tc.session}
		if err := svc.RunSetupWithOptions(t.TempDir(), t.TempDir(), opts, io.Discard, io.Discard); err != nil {
			t.Fatal(err)
		}
	}
	cancel()
	if err := svc.RunSetupWithOptions("", "", Options{Context: ctx, Subcommand: "broker"}, io.Discard, io.Discard); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled broker setup: %v", err)
	}
}
