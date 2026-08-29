package hostreqkittest

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	mutationFixture = "fixture"
	mutationMutant  = "mutant"
	mutationLinux   = "linux"
)

type goodHandler struct{}

func (goodHandler) Name() string           { return mutationFixture }
func (goodHandler) Kind() hostreqspec.Kind { return hostreqspec.KindTool }

func (goodHandler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.ItemStatus{Name: mutationFixture, Kind: hostreqspec.KindTool, SupportClass: hostreqkit.SupportSupported}
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
	} else if host.OS != mutationLinux {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
	}
	return status
}

func (goodHandler) Apply(_ hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	switch status.SupportClass {
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status, nil
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		return status, nil
	}
	status.ExecutionState = hostreqkit.ExecutionWouldApply
	return status, nil
}

type mutant struct {
	rule string
}

func (m mutant) Name() string {
	if m.rule == "wrong_name" {
		return mutationMutant
	}
	return goodHandler{}.Name()
}

func (m mutant) Kind() hostreqspec.Kind {
	if m.rule == "wrong_kind" {
		return hostreqspec.KindSafeguard
	}
	return goodHandler{}.Kind()
}

func (m mutant) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	if m.rule == "manual_inspect" && requirement.Manual {
		return hostreqkit.ItemStatus{Name: "fixture", Kind: hostreqspec.KindTool, SupportClass: hostreqkit.SupportSupported}
	}
	return goodHandler{}.Inspect(host, requirement)
}

func (m mutant) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	switch m.rule {
	case "dry_run_apply":
		if opts.DryRun && status.SupportClass == hostreqkit.SupportSupported {
			status.Applied = true
			status.ExecutionState = hostreqkit.ExecutionWouldApply
			return status, nil
		}
	case "unsupported_apply":
		if status.SupportClass == hostreqkit.SupportUnsupported {
			status.Applied = true
			status.ExecutionState = hostreqkit.ExecutionApplied
			return status, nil
		}
	case "already_applied":
		if status.Applied {
			status.ExecutionState = hostreqkit.ExecutionWouldApply
			return status, nil
		}
	}
	return goodHandler{}.Apply(host, status, opts)
}

type recordingT struct {
	failed bool
	errors []string
}

func (t *recordingT) Helper() {}

func (t *recordingT) Run(_ string, f func(suiteT)) bool {
	child := &recordingT{}
	f(child)
	if child.failed {
		t.failed = true
		t.errors = append(t.errors, child.errors...)
	}
	return !child.failed
}

func (t *recordingT) Errorf(format string, args ...any) {
	t.failed = true
	t.errors = append(t.errors, fmt.Sprintf(format, args...))
}

func fixtureCase(rule string) Case {
	return Case{
		NewHandler: func() hostreqkit.Handler {
			if rule == "" {
				return goodHandler{}
			}
			return mutant{rule: rule}
		},
		Name:               "fixture",
		Kind:               hostreqspec.KindTool,
		SupportedPlatforms: []string{"linux"},
		InstallCommand:     "fixture install",
		Checks: []string{
			"name_and_kind",
			"inspect_manual_requirement",
			"inspect_unsupported_platform",
			"apply_unsupported_returns_early",
			"apply_already_applied_skips",
			"apply_dry_run",
		},
	}
}

type matrixHandler struct{ rule string }

func (matrixHandler) Name() string           { return mutationFixture }
func (matrixHandler) Kind() hostreqspec.Kind { return hostreqspec.KindTool }

func (h matrixHandler) Inspect(host hostreqkit.Host, _ hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.ItemStatus{Name: "fixture", Kind: hostreqspec.KindTool}
	if !host.SupportsSysctl {
		status.SupportClass = hostreqkit.SupportNotApplicable
		if h.rule == "inspect_no_sysctl_not_applicable" {
			status.SupportClass = hostreqkit.SupportSupported
		}
		return status
	}
	switch {
	case host.OS == "linux" && (host.PackageManager == "apt" || host.PackageManager == "apt-get"):
		status.SupportClass = hostreqkit.SupportSupported
		status.InstallSupported = true
		status.PackageName = "fixture"
		if _, err := hostreqkit.LookPathFn("fixture"); err == nil {
			status.Installed = true
			status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
			output, _ := hostreqkit.CombinedOutputFn("fixture", "version")
			status.Version = strings.TrimSpace(string(output))
		}
	case host.OS == hostreqkittestDarwin && host.PackageManager == "brew":
		status.SupportClass = hostreqkit.SupportSupported
		status.InstallSupported = true
		status.PackageName = "brew-fixture"
	case host.OS == "windows" && host.PackageManager == "winget":
		status.SupportClass = hostreqkit.SupportSupported
		status.InstallSupported = true
		status.PackageName = "winget-fixture"
	default:
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
	}
	switch h.rule {
	case "inspect_linux_apt_not_installed":
		status.InstallSupported = false
	case "inspect_linux_apt_installed":
		status.Version = mutationMutant
	case "inspect_darwin_brew":
		if host.OS == "darwin" {
			status.PackageName = "mutant"
		}
	case "inspect_windows_winget":
		if host.OS == "windows" {
			status.PackageName = "mutant"
		}
	case "inspect_unsupported_configuration":
		if host.PackageManager == "dnf" {
			status.SupportClass = hostreqkit.SupportSupported
		}
	}
	return status
}

func (h matrixHandler) Apply(_ hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	switch {
	case status.Installed:
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		if h.rule == "apply_installed_skips" {
			status.ExecutionState = hostreqkit.ExecutionWouldInstall
		}
	case status.SupportClass == hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		if h.rule == "apply_not_applicable_returns_early" {
			status.ExecutionState = hostreqkit.ExecutionWouldInstall
		}
	case opts.DryRun:
		status.ExecutionState = hostreqkit.ExecutionWouldInstall
		if h.rule == "apply_linux_apt_dry_run" {
			status.ExecutionState = hostreqkit.ExecutionInstalled
		}
	}
	return status, nil
}

func matrixFixtureCase(rule string) Case {
	properties := map[string]ManifestProperty{
		"required": {Default: "fixture"},
	}
	if rule == "defaults_match_manifest" {
		properties["required"] = ManifestProperty{Default: "mutant"}
	}
	c := Case{
		NewHandler:      func() hostreqkit.Handler { return matrixHandler{rule: rule} },
		Name:            "fixture",
		Kind:            hostreqspec.KindTool,
		ToolBinary:      "fixture",
		PackageNames:    map[string]string{"brew": "brew-fixture", "winget": "winget-fixture"},
		VersionOutput:   "fixture version 1.0\n",
		ManifestVersion: "v1.0.0",
		DefaultVersion:  "v1.0.0",
		ManifestDefaults: &ManifestDefaultsCase{
			Load:     func() (map[string]ManifestProperty, error) { return properties, nil },
			Required: []string{"required"},
			Expected: map[string]any{"required": "fixture"},
		},
		Checks: []string{
			"inspect_linux_apt_not_installed",
			"inspect_linux_apt_installed",
			"inspect_darwin_brew",
			"inspect_windows_winget",
			"inspect_unsupported_configuration",
			"apply_installed_skips",
			"apply_not_applicable_returns_early",
			"apply_linux_apt_dry_run",
			"inspect_no_sysctl_not_applicable",
			"pinned_version_matches_manifest",
			"defaults_match_manifest",
		},
	}
	if rule == "pinned_version_matches_manifest" {
		c.DefaultVersion = "v2.0.0"
	}
	return c
}

type aptRepoMutationState struct {
	keyDownload func() ([]byte, error)
}

type aptRepoMutationHandler struct {
	rule  string
	state *aptRepoMutationState
}

func (aptRepoMutationHandler) Name() string           { return "fixture" }
func (aptRepoMutationHandler) Kind() hostreqspec.Kind { return hostreqspec.KindTool }
func (aptRepoMutationHandler) Inspect(hostreqkit.Host, hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	return hostreqkit.ItemStatus{Name: "fixture", Kind: hostreqspec.KindTool, SupportClass: hostreqkit.SupportSupported}
}

func (h aptRepoMutationHandler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, _ hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	switch {
	case host.OS == "darwin":
		_ = hostreqkit.RunCommandFn("brew", []string{"install", "fixture"}, hostreqkit.EnsureOptions{})
		status.ExecutionState = hostreqkit.ExecutionInstalled
		if h.rule == "apply_darwin_brew_flow" {
			status.ExecutionState = hostreqkit.ExecutionWouldInstall
		}
	case host.PackageManager == "dnf":
		status.SupportClass = hostreqkit.SupportUnsupported
		if h.rule == "apply_default_fallback_unsupported" {
			status.SupportClass = hostreqkit.SupportSupported
		}
	case host.OS == "linux":
		if h.state.keyDownload != nil {
			if _, err := h.state.keyDownload(); err != nil {
				status.ExecutionState = hostreqkit.ExecutionFailed
				if h.rule == "apply_linux_key_download_failure" {
					status.ExecutionState = hostreqkit.ExecutionInstalled
				}
				return status, nil
			}
		}
		_ = hostreqkit.RunCommandFn("apt-get", []string{"update"}, hostreqkit.EnsureOptions{})
		_ = hostreqkit.RunCommandFn("apt-get", []string{"install", "fixture"}, hostreqkit.EnsureOptions{})
		status.ExecutionState = hostreqkit.ExecutionInstalled
		if h.rule == "apply_linux_apt_full_flow" {
			status.ExecutionState = hostreqkit.ExecutionWouldInstall
		}
	}
	return status, nil
}

func aptRepoFixtureCase(rule string) Case {
	state := &aptRepoMutationState{}
	return Case{
		NewHandler:    func() hostreqkit.Handler { return aptRepoMutationHandler{rule: rule, state: state} },
		Name:          "fixture",
		Kind:          hostreqspec.KindTool,
		ToolBinary:    "fixture",
		VersionOutput: "fixture version 1.0\n",
		APTRepo: &APTRepoCase{
			Setup: func() func() {
				originalLookPath := hostreqkit.LookPathFn
				originalOutput := hostreqkit.CombinedOutputFn
				originalRun := hostreqkit.RunCommandFn
				return func() {
					hostreqkit.LookPathFn = originalLookPath
					hostreqkit.CombinedOutputFn = originalOutput
					hostreqkit.RunCommandFn = originalRun
					state.keyDownload = nil
				}
			},
			SetKeyDownload: func(download func() ([]byte, error)) { state.keyDownload = download },
			MinCommands:    2,
		},
		Checks: []string{
			"apply_darwin_brew_flow",
			"apply_default_fallback_unsupported",
			"apply_linux_apt_full_flow",
			"apply_linux_key_download_failure",
		},
	}
}

func TestSuiteAcceptsGoodHandler(t *testing.T) {
	RunSuite(t, fixtureCase(""))
}

func TestMutationSuiteRejectsEachMutant(t *testing.T) {
	for _, rule := range []string{"wrong_name", "wrong_kind", "dry_run_apply", "unsupported_apply", "manual_inspect", "already_applied"} {
		t.Run(rule, func(t *testing.T) {
			recorder := &recordingT{}
			runSuite(recorder, fixtureCase(rule))
			if !recorder.failed {
				t.Fatalf("RunSuite accepted mutant %q", rule)
			}
		})
	}
}

func TestMatrixSuiteAcceptsGoodHandler(t *testing.T) {
	originalLookPath := hostreqkit.LookPathFn
	originalOutput := hostreqkit.CombinedOutputFn
	defer func() {
		hostreqkit.LookPathFn = originalLookPath
		hostreqkit.CombinedOutputFn = originalOutput
	}()
	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	recorder := &recordingT{}
	runSuite(recorder, matrixFixtureCase(""))
	if recorder.failed {
		t.Fatalf("RunSuite rejected valid matrix handler: %v", recorder.errors)
	}
}

func TestMutationMatrixSuiteRejectsEachMutant(t *testing.T) {
	for _, rule := range []string{
		"inspect_linux_apt_not_installed",
		"inspect_linux_apt_installed",
		"inspect_darwin_brew",
		"inspect_windows_winget",
		"inspect_unsupported_configuration",
		"apply_installed_skips",
		"apply_not_applicable_returns_early",
		"apply_linux_apt_dry_run",
		"inspect_no_sysctl_not_applicable",
		"pinned_version_matches_manifest",
		"defaults_match_manifest",
	} {
		t.Run(rule, func(t *testing.T) {
			recorder := &recordingT{}
			runSuite(recorder, matrixFixtureCase(rule))
			if !recorder.failed {
				t.Fatalf("RunSuite accepted matrix mutant %q", rule)
			}
		})
	}
}

func TestAPTRepoSuiteAcceptsGoodHandler(t *testing.T) {
	recorder := &recordingT{}
	runSuite(recorder, aptRepoFixtureCase(""))
	if recorder.failed {
		t.Fatal("RunSuite rejected valid APT repository handler")
	}
}

func TestMutationAPTRepoSuiteRejectsEachMutant(t *testing.T) {
	for _, rule := range []string{
		"apply_darwin_brew_flow",
		"apply_default_fallback_unsupported",
		"apply_linux_apt_full_flow",
		"apply_linux_key_download_failure",
	} {
		t.Run(rule, func(t *testing.T) {
			recorder := &recordingT{}
			runSuite(recorder, aptRepoFixtureCase(rule))
			if !recorder.failed {
				t.Fatalf("RunSuite accepted APT repository mutant %q", rule)
			}
		})
	}
}

func TestRunSuiteExecutesPackageChecks(t *testing.T) {
	invoked := false
	c := fixtureCase("")
	c.PackageChecks = []PackageCheck{{
		Name: "package_specific_mutant_guard",
		Run:  func(*testing.T) { invoked = true },
	}}
	RunSuite(t, c)
	if !invoked {
		t.Fatal("RunSuite dropped a package-specific check")
	}
}
