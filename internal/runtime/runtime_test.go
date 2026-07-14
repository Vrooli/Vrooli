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
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=3 | LAST: 2026-04-11

// Constants mirroring the remote-session-protection safeguard handler for
// integration-level assertions. The canonical values live in
// internal/safeguards/remote-session-protection/handler.go.
const (
	remoteSessionSysctlPath    = "/etc/sysctl.d/99-vrooli-remote-session-protection.conf"
	remoteSessionSystemdDir    = "/etc/systemd/system/user@.service.d"
	remoteSessionSystemdPath   = "/etc/systemd/system/user@.service.d/90-vrooli-remote-session-protection.conf"
	remoteSessionLogindDir     = "/etc/systemd/logind.conf.d"
	remoteSessionLogindPath    = "/etc/systemd/logind.conf.d/90-vrooli-remote-session-protection.conf"
	remoteSessionSysctlContent = "vm.swappiness = 10\nvm.oom_kill_allocating_task = 0\n"
	remoteSessionUnitContent   = "[Service]\nOOMScoreAdjust=-900\n"
	remoteSessionLogindContent = "[Login]\nKillUserProcesses=no\n"
)

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

func TestEnsureRequirementsSupportsDeclaredToolAndSafeguard(t *testing.T) {
	restore := stubRuntimeLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		switch name {
		case "docker", "sh":
			return "/usr/bin/docker", nil
		case "apt-get", "sysctl", "systemctl":
			return "/usr/bin/" + name, nil
		default:
			return "", os.ErrNotExist
		}
	}
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		return nil, os.ErrNotExist
	}

	report, err := EnsureRequirements(EnsureOptions{
		Environment: "development",
		DryRun:      true,
		AutoInstall: true,
		SudoMode:    "skip",
	}, hostreq.Resolution{
		Tools: []hostreq.ResolvedRequirement{
			{Name: "tmux", Kind: hostreq.KindTool, Required: true, Reasons: []string{"scenario tmux"}},
			{Name: "bats", Kind: hostreq.KindTool, Required: false, Manual: true, Reasons: []string{"manual bats"}},
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

	bats := findStatus(t, report.Tools, "bats")
	if bats.SupportClass != SupportManualOnly {
		t.Fatalf("bats support class = %q", bats.SupportClass)
	}
	if bats.ExecutionState != ExecutionManualActionRequired {
		t.Fatalf("bats execution state = %q", bats.ExecutionState)
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

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		if name == "apt-get" {
			return "/usr/bin/apt-get", nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts EnsureOptions) error {
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

	hostreqkit.LookPathFn = func(name string) (string, error) {
		switch name {
		case "git", "curl":
			return "/usr/bin/" + name, nil
		default:
			return "", os.ErrNotExist
		}
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
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

	hostreqkit.LookPathFn = func(name string) (string, error) {
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

	// Prevent desktop detection from running real systemctl
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return nil, errors.New("stubbed: no desktop")
	}

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
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

	// Prevent desktop detection from running real systemctl
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return nil, errors.New("stubbed: no desktop")
	}

	hostreqkit.LookPathFn = func(name string) (string, error) {
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
	hostreqkit.RunCommandFn = func(name string, args []string, opts EnsureOptions) error {
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

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "", os.ErrNotExist
	}

	command, args, err := hostreqkit.InstallCommand(Host{OS: "linux", PackageManager: "apt-get"}, "jq", "ask")
	if err != nil {
		t.Fatalf("linux InstallCommand: %v", err)
	}
	if command != "sudo" || strings.Join(args, " ") != "apt-get install -y jq" {
		t.Fatalf("linux install command = %s %v", command, args)
	}

	command, args, err = hostreqkit.InstallCommand(Host{OS: "darwin", PackageManager: "brew"}, "jq", "ask")
	if err != nil {
		t.Fatalf("darwin InstallCommand: %v", err)
	}
	if command != "brew" || strings.Join(args, " ") != "install jq" {
		t.Fatalf("darwin install command = %s %v", command, args)
	}
}

func TestGenericToolSudoSkippedRemainsSupportedAndNeedsSudo(t *testing.T) {
	restore := stubRuntimeLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		switch name {
		case "sudo":
			return "/usr/bin/sudo", nil
		default:
			return "", os.ErrNotExist
		}
	}

	h := newGenericToolHandler(hostreqkit.ToolManifest{
		Name:           "java",
		Commands:       []string{"java"},
		DefaultPackage: "openjdk-17-jre",
	})
	req := hostreq.ResolvedRequirement{Name: "java", Kind: hostreq.KindTool, Required: true}
	status := h.Inspect(Host{OS: "linux", PackageManager: "apt-get"}, req)
	out, err := h.Apply(Host{OS: "linux", PackageManager: "apt-get"}, status, EnsureOptions{SudoMode: "skip"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.SupportClass != SupportSupported {
		t.Fatalf("SupportClass = %q, want supported", out.SupportClass)
	}
	if out.ExecutionState != ExecutionFailed {
		t.Fatalf("ExecutionState = %q, want failed", out.ExecutionState)
	}
	if out.BlockingReason != hostreqkit.BlockingNeedsSudo {
		t.Fatalf("BlockingReason = %q, want needs_sudo", out.BlockingReason)
	}
	if !containsNote(out.Notes, "sudo skipped") {
		t.Fatalf("notes should mention sudo skipped, got %v", out.Notes)
	}
}

func TestRegistryContainsUniqueToolAndSafeguardHandlers(t *testing.T) {
	reg, err := ensureRegistry()
	if err != nil {
		t.Fatalf("ensureRegistry: %v", err)
	}
	toolNames := reg.names(hostreq.KindTool)
	expectedTools := []string{
		"Xvfb", "ast-grep", "bats", "buf", "cloudflared", "curl",
		"docker", "ffmpeg", "git", "go", "helm", "iopaint", "java", "jq", "kdump-tools",
		"llama-cpp", "lychee",
		"mcelog", "node", "openbox", "pnpm", "protoc", "protoc-gen-connect-go",
		"protoc-gen-es", "protoc-gen-go",
		"python", "quint", "rasdaemon", "realesrgan-ncnn-vulkan", "rembg", "sd", "sd-gpu",
		"stripe", "tmux", "uv", "vault", "websockify",
		"x11vnc", "xdotool", "yq",
	}
	if len(toolNames) != len(expectedTools) {
		t.Fatalf("tool count = %d, want %d; got %v", len(toolNames), len(expectedTools), toolNames)
	}
	if !sort.StringsAreSorted(toolNames) {
		t.Fatalf("tool names not sorted: %v", toolNames)
	}
	for _, name := range expectedTools {
		if !contains(toolNames, name) {
			t.Fatalf("tool %q not found in registry: %v", name, toolNames)
		}
	}

	safeguardNames := reg.names(hostreq.KindSafeguard)
	expectedSafeguards := []string{
		"clock", "cloudflared_recovery_privileges", "crashkernel_reserve", "dns_resolution", "docker_host_firewall",
		"edac_modules", "host_hardening", "kernel_config", "nat_protection", "netconsole",
		"ollama_resource_controls",
		"pstore_native", "pstore_observability", "pstore_ramoops", "remote_session_protection", "tcp_tuning",
		"vrooli_launcher", "workspace_sandbox_userns",
	}
	if len(safeguardNames) != len(expectedSafeguards) {
		t.Fatalf("safeguard count = %d, want %d; got %v", len(safeguardNames), len(expectedSafeguards), safeguardNames)
	}
	if !sort.StringsAreSorted(safeguardNames) {
		t.Fatalf("safeguard names not sorted: %v", safeguardNames)
	}
	for _, name := range expectedSafeguards {
		if !contains(safeguardNames, name) {
			t.Fatalf("safeguard %q not found in registry: %v", name, safeguardNames)
		}
	}
}

func TestDetectFirstAvailableUsesSharedLookup(t *testing.T) {
	restore := stubRuntimeLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "apk" {
			return "/sbin/apk", nil
		}
		return "", os.ErrNotExist
	}

	if got := hostreqkit.DetectFirstAvailable([]string{"apt-get", "apk", "brew"}); got != "apk" {
		t.Fatalf("detectFirstAvailable = %q", got)
	}
}

func TestNewRegistryRejectsDuplicateHandlers(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected duplicate handler panic")
		}
	}()

	dup := newGenericToolHandler(hostreqkit.ToolManifest{Name: "dup"})
	newRegistry(dup, dup)
}

func stubRuntimeLookups(t *testing.T) func() {
	t.Helper()
	originalLookPathFn := hostreqkit.LookPathFn
	originalReadFileFn := hostreqkit.ReadFileFn
	originalCombinedOutputFn := hostreqkit.CombinedOutputFn
	originalRunCommandFn := hostreqkit.RunCommandFn
	originalWriteTempFileFn := hostreqkit.WriteTempFileFn
	return func() {
		hostreqkit.LookPathFn = originalLookPathFn
		hostreqkit.ReadFileFn = originalReadFileFn
		hostreqkit.CombinedOutputFn = originalCombinedOutputFn
		hostreqkit.RunCommandFn = originalRunCommandFn
		hostreqkit.WriteTempFileFn = originalWriteTempFileFn
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

func TestRequirementSatisfiedTreatsAlreadyPresentAsSatisfied(t *testing.T) {
	// Reproduces the mcelog supersede bug: a tool returns
	// ExecutionAlreadyPresent (e.g. superseded by rasdaemon) without setting
	// Installed=true. Before the fix, summarizeReport would still flag it as
	// missing.
	status := ItemStatus{
		Name:           "mcelog",
		Kind:           hostreq.KindTool,
		Required:       true,
		Installed:      false,
		ExecutionState: hostreqkit.ExecutionAlreadyPresent,
	}
	if !requirementSatisfied(status) {
		t.Fatalf("ExecutionAlreadyPresent should satisfy the requirement even if Installed=false")
	}
}

func TestMarkOptionalSkippedTagsBlockingReason(t *testing.T) {
	in := ItemStatus{Name: "x", Required: false, ExecutionState: hostreqkit.ExecutionPending}
	out := markOptionalSkipped(in)
	if out.BlockingReason != hostreqkit.BlockingOptionalSkipped {
		t.Fatalf("BlockingReason = %q", out.BlockingReason)
	}
}

func TestAnnotateBlockingReasonDetectsSudoSentinel(t *testing.T) {
	// Handlers fold privileged-command errors into Notes via fmt.Sprintf.
	// annotateBlockingReason should pick up the typed sentinel substring.
	status := ItemStatus{
		ExecutionState: hostreqkit.ExecutionFailed,
		Notes:          []string{"install kdump-tools failed: sudo skipped: automatic install skipped because --sudo-mode=skip"},
	}
	out := annotateBlockingReason(status)
	if out.BlockingReason != hostreqkit.BlockingNeedsSudo {
		t.Fatalf("BlockingReason = %q, want needs_sudo", out.BlockingReason)
	}
}

func TestAnnotateBlockingReasonPreservesExplicitValue(t *testing.T) {
	status := ItemStatus{
		ExecutionState: hostreqkit.ExecutionFailed,
		BlockingReason: hostreqkit.BlockingNeedsEnv,
		Notes:          []string{"sudo skipped"},
	}
	out := annotateBlockingReason(status)
	if out.BlockingReason != hostreqkit.BlockingNeedsEnv {
		t.Fatalf("explicit BlockingReason should be preserved, got %q", out.BlockingReason)
	}
}

func TestAnnotateBlockingReasonDerivesFromExecutionState(t *testing.T) {
	if got := annotateBlockingReason(ItemStatus{ExecutionState: hostreqkit.ExecutionRebootRequired}).BlockingReason; got != hostreqkit.BlockingNeedsReboot {
		t.Fatalf("reboot_required -> %q", got)
	}
	if got := annotateBlockingReason(ItemStatus{ExecutionState: hostreqkit.ExecutionManualActionRequired}).BlockingReason; got != hostreqkit.BlockingManual {
		t.Fatalf("manual_action_required -> %q", got)
	}
}
