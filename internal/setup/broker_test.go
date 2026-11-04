package setup

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/privilegebroker"
	vruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
)

func TestSetupBrokerTouchesOnlyBrokerAndNeverMarksProjectComplete(t *testing.T) {
	svc := stubSetupDeps(t)
	calls := 0
	svc.deps.currentHost = func() vruntime.Host { return vruntime.Host{OS: "linux", SupportsSetup: true} }
	svc.deps.loadProject = func(string) (scenario.Scenario, error) { return scenario.Scenario{}, nil }
	svc.deps.osExecutable = func() (string, error) { return "/reviewed/vrooli", nil }
	svc.deps.inspectPrivilegeBroker = func() privilegebroker.SetupStatus { return privilegebroker.SetupStatus{Supported: true} }
	svc.deps.installPrivilegeBroker = func(_ context.Context, executable string, opts Options, _, _ io.Writer) (privilegebroker.SetupStatus, error) {
		calls++
		if executable != "/reviewed/vrooli" || opts.SudoMode != "ask" {
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
	if err := svc.RunSetupWithOptions(root, home, Options{Subcommand: "broker", DryRun: true}, out, io.Discard); err != nil || calls != 0 {
		t.Fatalf("dry run err=%v calls=%d", err, calls)
	}
	if err := svc.RunSetupWithOptions(root, home, Options{Subcommand: "broker", SudoMode: "ask"}, out, io.Discard); err != nil || calls != 1 {
		t.Fatalf("apply err=%v calls=%d", err, calls)
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
	svc.deps.inspectPrivilegeBroker = func() privilegebroker.SetupStatus { return privilegebroker.SetupStatus{Supported: true} }
	svc.deps.installPrivilegeBroker = func(context.Context, string, Options, io.Writer, io.Writer) (privilegebroker.SetupStatus, error) {
		return privilegebroker.SetupStatus{Supported: true, Reason: "not elevated"}, nil
	}
	if err := svc.RunSetupWithOptions(t.TempDir(), t.TempDir(), Options{Subcommand: "broker"}, io.Discard, io.Discard); err == nil {
		t.Fatal("missing broker reported success")
	}
}
