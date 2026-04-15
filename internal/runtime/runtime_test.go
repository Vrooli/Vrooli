package runtime

import (
	"errors"
	"os"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreq"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=3 | LAST: 2026-04-11

func TestCurrentMatchesRuntimeGOOS(t *testing.T) {
	host := Current()
	if host.OS != goruntime.GOOS {
		t.Fatalf("host.OS = %q, want %q", host.OS, goruntime.GOOS)
	}
}

func TestValidateSupportMatchesCapabilityFlags(t *testing.T) {
	host := Current()
	if host.SupportsSetup {
		if err := host.ValidateSetup(); err != nil {
			t.Fatalf("ValidateSetup: %v", err)
		}
	} else if err := host.ValidateSetup(); err == nil {
		t.Fatal("ValidateSetup succeeded on unsupported host")
	}

	if host.SupportsDevelop {
		if err := host.ValidateDevelop(); err != nil {
			t.Fatalf("ValidateDevelop: %v", err)
		}
	} else if err := host.ValidateDevelop(); err == nil {
		t.Fatal("ValidateDevelop succeeded on unsupported host")
	}
}

func TestUnsupportedHostErrorsIncludePlatformNotes(t *testing.T) {
	host := Host{
		OS:              "darwin",
		SupportsSetup:   false,
		SupportsDevelop: false,
		Notes:           []string{"project-level setup/develop are native, but resource and scenario lifecycle support still assumes Linux-oriented tooling"},
	}

	setupErr := host.ValidateSetup()
	if setupErr == nil || !strings.Contains(setupErr.Error(), "not supported on darwin") {
		t.Fatalf("ValidateSetup error = %v", setupErr)
	}
	if !strings.Contains(setupErr.Error(), "Linux-oriented tooling") {
		t.Fatalf("ValidateSetup error missing note: %v", setupErr)
	}

	developErr := host.ValidateDevelop()
	if developErr == nil || !strings.Contains(developErr.Error(), "not supported on darwin") {
		t.Fatalf("ValidateDevelop error = %v", developErr)
	}
}

func TestInspectRejectsImplicitLegacyResolution(t *testing.T) {
	if _, err := Inspect("development"); err == nil || !strings.Contains(err.Error(), "explicit host requirements") {
		t.Fatalf("Inspect error = %v", err)
	}
}

func TestEnsureRejectsImplicitLegacyResolution(t *testing.T) {
	_, err := Ensure(EnsureOptions{Environment: "development"})
	if err == nil || !strings.Contains(err.Error(), "explicit host requirements") {
		t.Fatalf("Ensure error = %v", err)
	}
}

func TestEnsureRequirementsSupportsDeclaredToolAndSafeguard(t *testing.T) {
	restore := stubRuntimeLookups(t)
	defer restore()

	lookPathFn = func(name string) (string, error) {
		switch name {
		case "docker", "sh":
			return "/usr/bin/docker", nil
		case "apt-get", "sysctl", "systemctl":
			return "/usr/bin/" + name, nil
		default:
			return "", os.ErrNotExist
		}
	}

	report, err := EnsureRequirements(EnsureOptions{
		Environment: "development",
		DryRun:      true,
		AutoInstall: true,
		SudoMode:    "skip",
	}, hostreq.Resolution{
		Tools: []hostreq.ResolvedRequirement{
			{Name: "tmux", Kind: hostreq.KindTool, Required: true, Reasons: []string{"scenario tmux"}},
			{Name: "sqlite", Kind: hostreq.KindTool, Required: false, Manual: true, Reasons: []string{"manual sqlite"}},
		},
		Safeguards: []hostreq.ResolvedRequirement{
			{Name: "remote_session_protection", Kind: hostreq.KindSafeguard, Required: true, Reasons: []string{"linux guard"}},
		},
	})
	if err != nil {
		t.Fatalf("EnsureRequirements: %v", err)
	}

	tmux := findStatus(t, report.Tools, "tmux")
	if tmux.SupportClass != SupportSupported {
		t.Fatalf("tmux support class = %q", tmux.SupportClass)
	}
	if tmux.ExecutionState != ExecutionWouldInstall {
		t.Fatalf("tmux execution state = %q", tmux.ExecutionState)
	}
	if !containsNote(tmux.Notes, "dry-run: would run") {
		t.Fatalf("tmux notes = %v", tmux.Notes)
	}

	sqlite := findStatus(t, report.Tools, "sqlite")
	if sqlite.SupportClass != SupportManualOnly {
		t.Fatalf("sqlite support class = %q", sqlite.SupportClass)
	}
	if sqlite.ExecutionState != ExecutionManualActionRequired {
		t.Fatalf("sqlite execution state = %q", sqlite.ExecutionState)
	}

	safeguard := findStatus(t, report.Safeguards, "remote_session_protection")
	if safeguard.Kind != hostreq.KindSafeguard {
		t.Fatalf("safeguard kind = %q", safeguard.Kind)
	}
	if safeguard.ExecutionState != ExecutionWouldApply {
		t.Fatalf("safeguard execution state = %q", safeguard.ExecutionState)
	}
}

func TestEnsureRequirementsReportsFailedInstallWithoutPretendingSuccess(t *testing.T) {
	restore := stubRuntimeLookups(t)
	defer restore()

	lookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		if name == "apt-get" {
			return "/usr/bin/apt-get", nil
		}
		return "", os.ErrNotExist
	}
	runCommandFn = func(name string, args []string, opts EnsureOptions) error {
		return errors.New("install exploded")
	}

	report, err := EnsureRequirements(EnsureOptions{
		Environment: "development",
		AutoInstall: true,
		SudoMode:    "ask",
	}, hostreq.Resolution{
		Tools: []hostreq.ResolvedRequirement{
			{Name: "tmux", Kind: hostreq.KindTool, Required: true},
		},
	})
	if err == nil {
		t.Fatal("expected missing required tmux error")
	}

	tmux := findStatus(t, report.Tools, "tmux")
	if tmux.Installed {
		t.Fatalf("tmux should remain uninstalled: %+v", tmux)
	}
	if tmux.ExecutionState != ExecutionFailed {
		t.Fatalf("tmux execution state = %q", tmux.ExecutionState)
	}
	if !containsNote(tmux.Notes, "install exploded") {
		t.Fatalf("tmux notes = %v", tmux.Notes)
	}
}

func TestInspectRequirementsMarksUnknownHandlerUnsupported(t *testing.T) {
	report, err := InspectRequirements("development", hostreq.Resolution{
		Tools: []hostreq.ResolvedRequirement{
			{Name: "missing-tool", Kind: hostreq.KindTool, Required: true},
		},
	})
	if err != nil {
		t.Fatalf("InspectRequirements: %v", err)
	}
	status := findStatus(t, report.Tools, "missing-tool")
	if status.SupportClass != SupportUnsupported {
		t.Fatalf("support class = %q", status.SupportClass)
	}
	if status.ExecutionState != ExecutionUnsupported {
		t.Fatalf("execution state = %q", status.ExecutionState)
	}
}

func TestInspectRequirementsIncludesNewCoreHandlers(t *testing.T) {
	restore := stubRuntimeLookups(t)
	defer restore()

	lookPathFn = func(name string) (string, error) {
		switch name {
		case "git", "curl":
			return "/usr/bin/" + name, nil
		default:
			return "", os.ErrNotExist
		}
	}
	combinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte(name + " version\n"), nil
	}

	report, err := InspectRequirements("development", hostreq.Resolution{
		Tools: []hostreq.ResolvedRequirement{
			{Name: "git", Kind: hostreq.KindTool, Required: true},
			{Name: "curl", Kind: hostreq.KindTool, Required: true},
			{Name: "jq", Kind: hostreq.KindTool, Required: true},
			{Name: "ffmpeg", Kind: hostreq.KindTool, Required: false},
		},
	})
	if err != nil {
		t.Fatalf("InspectRequirements: %v", err)
	}

	git := findStatus(t, report.Tools, "git")
	if !git.Installed || git.SupportClass != SupportSupported {
		t.Fatalf("git status = %+v", git)
	}
	if git.ExecutionState != ExecutionAlreadyPresent {
		t.Fatalf("git execution state = %q", git.ExecutionState)
	}
	jq := findStatus(t, report.Tools, "jq")
	if jq.SupportClass != SupportSupported || jq.PackageName != "jq" {
		t.Fatalf("jq status = %+v", jq)
	}
	if jq.ExecutionState != ExecutionPending {
		t.Fatalf("jq execution state = %q", jq.ExecutionState)
	}
	if !contains(report.MissingRequired, "jq") {
		t.Fatalf("missing required = %v", report.MissingRequired)
	}
	ffmpeg := findStatus(t, report.Tools, "ffmpeg")
	if ffmpeg.SupportClass != SupportSupported || ffmpeg.PackageName != "ffmpeg" {
		t.Fatalf("ffmpeg status = %+v", ffmpeg)
	}
}

func TestInspectRequirementsIncludesStripeHandler(t *testing.T) {
	restore := stubRuntimeLookups(t)
	defer restore()

	lookPathFn = func(name string) (string, error) {
		if name == "apt-get" {
			return "/usr/bin/apt-get", nil
		}
		return "", os.ErrNotExist
	}

	report, err := InspectRequirements("development", hostreq.Resolution{
		Tools: []hostreq.ResolvedRequirement{
			{Name: "stripe", Kind: hostreq.KindTool, Required: false},
		},
	})
	if err != nil {
		t.Fatalf("InspectRequirements: %v", err)
	}

	stripe := findStatus(t, report.Tools, "stripe")
	if stripe.SupportClass != SupportSupported {
		t.Fatalf("stripe support class = %q", stripe.SupportClass)
	}
	if stripe.PackageName != "stripe" {
		t.Fatalf("stripe package name = %q", stripe.PackageName)
	}
	if stripe.ExecutionState != ExecutionPending {
		t.Fatalf("stripe execution state = %q", stripe.ExecutionState)
	}
}

func TestRemoteSessionProtectionClassifiesUnsupportedAndNotApplicable(t *testing.T) {
	restore := stubRuntimeLookups(t)
	defer restore()

	readFileFn = func(path string) ([]byte, error) {
		switch path {
		case remoteSessionSysctlPath:
			return []byte(remoteSessionSysctlContent), nil
		case remoteSessionSystemdPath:
			return []byte(remoteSessionUnitContent), nil
		case remoteSessionLogindPath:
			return []byte(remoteSessionLogindContent), nil
		default:
			return nil, os.ErrNotExist
		}
	}

	applied := inspectRequirement(Host{
		OS:              "linux",
		SupportsSysctl:  true,
		SupportsSystemd: true,
	}, hostreq.ResolvedRequirement{
		Name: "remote_session_protection",
		Kind: hostreq.KindSafeguard,
	})
	if !applied.Applied {
		t.Fatalf("expected applied safeguard, got %+v", applied)
	}
	if applied.ExecutionState != ExecutionAlreadyPresent {
		t.Fatalf("applied execution state = %q", applied.ExecutionState)
	}

	unsupported := inspectRequirement(Host{
		OS: "darwin",
	}, hostreq.ResolvedRequirement{
		Name: "remote_session_protection",
		Kind: hostreq.KindSafeguard,
	})
	if unsupported.SupportClass != SupportUnsupported {
		t.Fatalf("unsupported class = %q", unsupported.SupportClass)
	}
	if unsupported.ExecutionState != ExecutionUnsupported {
		t.Fatalf("unsupported execution state = %q", unsupported.ExecutionState)
	}

	notApplicable := inspectRequirement(Host{
		OS:              "linux",
		SupportsSysctl:  false,
		SupportsSystemd: false,
	}, hostreq.ResolvedRequirement{
		Name: "remote_session_protection",
		Kind: hostreq.KindSafeguard,
	})
	if notApplicable.SupportClass != SupportNotApplicable {
		t.Fatalf("notApplicable class = %q", notApplicable.SupportClass)
	}
	if notApplicable.ExecutionState != ExecutionNotApplicable {
		t.Fatalf("notApplicable execution state = %q", notApplicable.ExecutionState)
	}
}

func TestRemoteSessionProtectionApplyRunsManagedScript(t *testing.T) {
	restore := stubRuntimeLookups(t)
	defer restore()

	lookPathFn = func(name string) (string, error) {
		switch name {
		case "sudo", "mkdir", "install", "sysctl", "systemctl":
			return "/usr/bin/" + name, nil
		default:
			return "", os.ErrNotExist
		}
	}

	type commandCall struct {
		name string
		args []string
	}
	calls := []commandCall{}
	runCommandFn = func(name string, args []string, opts EnsureOptions) error {
		calls = append(calls, commandCall{name: name, args: append([]string(nil), args...)})
		return nil
	}

	status, err := applyRequirement(Host{
		OS:              "linux",
		SupportsSysctl:  true,
		SupportsSystemd: true,
	}, ItemStatus{
		Name:         "remote_session_protection",
		Kind:         hostreq.KindSafeguard,
		Required:     true,
		SupportClass: SupportSupported,
	}, EnsureOptions{
		SudoMode: "ask",
	})
	if err != nil {
		t.Fatalf("applyRequirement: %v", err)
	}
	if !status.Applied || status.ExecutionState != ExecutionApplied {
		t.Fatalf("status = %+v", status)
	}
	if len(calls) != 8 {
		t.Fatalf("command call count = %d, want 8 (%+v)", len(calls), calls)
	}
	for _, call := range calls {
		if call.name != "sudo" {
			t.Fatalf("command = %q, want sudo", call.name)
		}
	}
	got := make([]string, 0, len(calls))
	for _, call := range calls {
		got = append(got, strings.Join(call.args, " "))
	}
	expected := []string{
		"mkdir -p " + filepath.Dir(remoteSessionSysctlPath),
		"install -m 0644",
		"sysctl -p " + remoteSessionSysctlPath,
		"mkdir -p " + remoteSessionSystemdDir,
		"mkdir -p " + remoteSessionLogindDir,
		"install -m 0644",
		"install -m 0644",
		"systemctl daemon-reload",
	}
	for _, needle := range expected {
		matched := false
		for _, call := range got {
			if strings.Contains(call, needle) {
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("expected command containing %q, got %v", needle, got)
		}
	}
	if !containsJoined(got, remoteSessionSystemdPath) || !containsJoined(got, remoteSessionLogindPath) {
		t.Fatalf("install commands missing managed paths: %v", got)
	}
}

func TestInstallCommandMappings(t *testing.T) {
	restore := stubRuntimeLookups(t)
	defer restore()

	lookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "", os.ErrNotExist
	}

	command, args, err := installCommand(Host{OS: "linux", PackageManager: "apt-get"}, "jq", "ask")
	if err != nil {
		t.Fatalf("linux installCommand: %v", err)
	}
	if command != "sudo" || strings.Join(args, " ") != "apt-get install -y jq" {
		t.Fatalf("linux install command = %s %v", command, args)
	}

	command, args, err = installCommand(Host{OS: "darwin", PackageManager: "brew"}, "jq", "ask")
	if err != nil {
		t.Fatalf("darwin installCommand: %v", err)
	}
	if command != "brew" || strings.Join(args, " ") != "install jq" {
		t.Fatalf("darwin install command = %s %v", command, args)
	}
}

func TestRegistryContainsUniqueToolAndSafeguardHandlers(t *testing.T) {
	toolNames := runtimeRegistry.names(hostreq.KindTool)
	if len(toolNames) == 0 {
		t.Fatal("expected tool handlers")
	}
	if !sort.StringsAreSorted(toolNames) {
		t.Fatalf("tool names not sorted: %v", toolNames)
	}
	if !contains(toolNames, "docker") ||
		!contains(toolNames, "ffmpeg") ||
		!contains(toolNames, "stripe") ||
		!contains(toolNames, "tmux") ||
		!contains(toolNames, "bats") ||
		!contains(toolNames, "yq") ||
		!contains(toolNames, "Xvfb") ||
		!contains(toolNames, "x11vnc") ||
		!contains(toolNames, "xdotool") ||
		!contains(toolNames, "websockify") ||
		!contains(toolNames, "openbox") {
		t.Fatalf("tool names missing expected entries: %v", toolNames)
	}

	safeguardNames := runtimeRegistry.names(hostreq.KindSafeguard)
	if got := strings.Join(safeguardNames, ","); got != "remote_session_protection" {
		t.Fatalf("safeguard names = %q", got)
	}
}

func TestDetectFirstAvailableUsesSharedLookup(t *testing.T) {
	restore := stubRuntimeLookups(t)
	defer restore()

	lookPathFn = func(name string) (string, error) {
		if name == "apk" {
			return "/sbin/apk", nil
		}
		return "", os.ErrNotExist
	}

	if got := detectFirstAvailable([]string{"apt-get", "apk", "brew"}); got != "apk" {
		t.Fatalf("detectFirstAvailable = %q", got)
	}
}

func TestNewRegistryRejectsDuplicateHandlers(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected duplicate handler panic")
		}
	}()

	newRegistry(newDockerTool(), newDockerTool())
}

func stubRuntimeLookups(t *testing.T) func() {
	t.Helper()
	originalLookPathFn := lookPathFn
	originalReadFileFn := readFileFn
	originalCombinedOutputFn := combinedOutputFn
	originalRunCommandFn := runCommandFn
	return func() {
		lookPathFn = originalLookPathFn
		readFileFn = originalReadFileFn
		combinedOutputFn = originalCombinedOutputFn
		runCommandFn = originalRunCommandFn
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsNote(values []string, target string) bool {
	for _, value := range values {
		if strings.Contains(value, target) {
			return true
		}
	}
	return false
}

func containsJoined(values []string, target string) bool {
	for _, value := range values {
		if strings.Contains(value, target) {
			return true
		}
	}
	return false
}

func findStatus(t *testing.T, items []ItemStatus, name string) ItemStatus {
	t.Helper()
	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("status %q not found", name)
	return ItemStatus{}
}
