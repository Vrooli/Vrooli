package setup

import (
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/privilegebroker"
	"github.com/vrooli/vrooli/internal/projectstate"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/scenario"
)

// statusReport is the canned report returned by inspectRequirements for both
// the status and explain subcommand tests. It mirrors the shape of a real
// inspection so the printer + lookup paths are exercised end-to-end.
func statusReport() vrooliruntime.Report {
	prov := []hostreqspec.Provenance{{Kind: "root", Name: "vrooli", Source: ".vrooli/service.json"}}
	return vrooliruntime.Report{
		Environment: "development",
		Host:        vrooliruntime.Host{OS: "linux", PackageManager: "apt-get"},
		Tools: []vrooliruntime.ToolStatus{
			{
				Name: "mcelog", Kind: hostreq.KindTool, Required: true,
				ExecutionState: vrooliruntime.ExecutionAlreadyPresent,
				Reasons:        []string{"Capture MCEs as a fallback for rasdaemon"},
				Notes:          []string{"superseded by rasdaemon (no mcelog package available on this distribution)"},
				Provenance:     prov,
			},
			{
				Name: "rasdaemon", Kind: hostreq.KindTool, Required: true,
				ExecutionState: vrooliruntime.ExecutionAlreadyPresent,
				Provenance:     prov,
			},
		},
	}
}

func TestRunSetupStatusPrintsGroupedAndDoesNotMutate(t *testing.T) {
	svc := stubSetupDeps(t)
	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	svc.deps.currentHost = func() vrooliruntime.Host {
		return vrooliruntime.Host{OS: "linux", PackageManager: "apt-get", SupportsSetup: true, SupportsDevelop: true}
	}
	svc.deps.loadProject = func(string) (scenario.Scenario, error) { return projectScenario, nil }
	svc.deps.resolveHostRequirements = func(string, string, hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	inspectCalls := 0
	svc.deps.inspectRequirements = func(string, hostreq.Resolution) (vrooliruntime.Report, error) {
		inspectCalls++
		return statusReport(), nil
	}
	svc.deps.ensureRequirements = func(vrooliruntime.EnsureOptions, hostreq.Resolution) (vrooliruntime.Report, error) {
		t.Fatal("ensureRequirements must not run during `setup status`")
		return vrooliruntime.Report{}, nil
	}
	svc.deps.inspectPrivilegeBroker = func() privilegebroker.SetupStatus {
		return privilegebroker.SetupStatus{Supported: true, Reason: "setup was not elevated", Recovery: "Re-run sudo vrooli setup"}
	}

	stdout := &strings.Builder{}
	if err := svc.RunSetupWithOptions(root, home, Options{Subcommand: "status"}, stdout, io.Discard); err != nil {
		t.Fatalf("RunSetupWithOptions(status): %v", err)
	}
	if inspectCalls != 1 {
		t.Fatalf("inspectRequirements calls = %d, want 1", inspectCalls)
	}

	out := stdout.String()
	for _, expected := range []string{
		"Host requirements status",
		"Already present (2): mcelog, rasdaemon",
		"Run 'vrooli setup explain <name>'",
		"Privilege broker: unavailable — setup was not elevated",
		"Configuration: pending (.configuration-complete absent)",
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("status output missing %q:\n%s", expected, out)
		}
	}

	if err := writeSetupCompleteMarker(t, home, root); err != nil {
		t.Fatalf("write bootstrap marker: %v", err)
	}
	if err := projectstate.MarkConfigurationComplete(home, root, "status-selection"); err != nil {
		t.Fatalf("mark configuration complete: %v", err)
	}
	stdout.Reset()
	if err := svc.RunSetupWithOptions(root, home, Options{Subcommand: "status"}, stdout, io.Discard); err != nil {
		t.Fatalf("configured RunSetupWithOptions(status): %v", err)
	}
	configured := stdout.String()
	if !strings.Contains(configured, "Configuration: complete") || !strings.Contains(configured, "selection_digest=status-selection") {
		t.Fatalf("configured status output = %q", configured)
	}
}

func TestRunSetupExplainPrintsVerboseBlock(t *testing.T) {
	svc := stubSetupDeps(t)
	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	svc.deps.currentHost = func() vrooliruntime.Host {
		return vrooliruntime.Host{OS: "linux", PackageManager: "apt-get", SupportsSetup: true, SupportsDevelop: true}
	}
	svc.deps.loadProject = func(string) (scenario.Scenario, error) { return projectScenario, nil }
	svc.deps.resolveHostRequirements = func(string, string, hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	svc.deps.inspectRequirements = func(string, hostreq.Resolution) (vrooliruntime.Report, error) {
		return statusReport(), nil
	}

	stdout := &strings.Builder{}
	if err := svc.RunSetupWithOptions(root, home, Options{Subcommand: "explain", ExplainName: "mcelog"}, stdout, io.Discard); err != nil {
		t.Fatalf("RunSetupWithOptions(explain): %v", err)
	}
	out := stdout.String()
	for _, expected := range []string{
		"mcelog [required]",
		"reasons: Capture MCEs as a fallback for rasdaemon",
		"superseded by rasdaemon",
		"declared by root:vrooli",
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("explain output missing %q:\n%s", expected, out)
		}
	}
}

func TestRunSetupExplainErrorsOnUnknownName(t *testing.T) {
	svc := stubSetupDeps(t)
	root := t.TempDir()
	home := t.TempDir()
	projectScenario := writeProjectFixture(t, root)

	svc.deps.currentHost = func() vrooliruntime.Host {
		return vrooliruntime.Host{OS: "linux", PackageManager: "apt-get", SupportsSetup: true, SupportsDevelop: true}
	}
	svc.deps.loadProject = func(string) (scenario.Scenario, error) { return projectScenario, nil }
	svc.deps.resolveHostRequirements = func(string, string, hostreq.ResolveOptions) (hostreq.Resolution, error) {
		return hostreq.Resolution{}, nil
	}
	svc.deps.inspectRequirements = func(string, hostreq.Resolution) (vrooliruntime.Report, error) {
		return statusReport(), nil
	}

	err := svc.RunSetupWithOptions(root, home, Options{Subcommand: "explain", ExplainName: "nope"}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "no host requirement") {
		t.Fatalf("expected unknown-name error, got %v", err)
	}
}
