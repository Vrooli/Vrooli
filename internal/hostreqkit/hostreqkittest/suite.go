// Package hostreqkittest contains the shared conformance checks for host
// requirement handlers. Handler packages provide only their constructor and
// manifest-specific case data.
package hostreqkittest

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/testenv"
)

// LoadToolManifest loads the manifest owned by a manifest-only tool package.
// Keeping this small loader here lets each package's conformance case use its
// own tool.json without duplicating JSON setup or silently copying a
// neighbour's manifest.
func LoadToolManifest(t *testing.T, path string) hostreqkit.ToolManifest {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read tool manifest %q: %v", path, err)
	}
	var manifest hostreqkit.ToolManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decode tool manifest %q: %v", path, err)
	}
	return manifest
}

// RunGenericToolSuite applies the shared contract to a manifest-only tool
// directory. The directory still supplies the name and manifest; only the
// generic runtime handler and common check selection live here.
func RunGenericToolSuite(t *testing.T, name string, newHandler func(hostreqkit.ToolManifest) hostreqkit.Handler) {
	t.Helper()
	manifest := LoadToolManifest(t, "tool.json")
	if manifest.Name != name {
		t.Fatalf("tool manifest name = %q, want %q", manifest.Name, name)
	}
	if newHandler == nil {
		t.Fatal("generic tool handler factory is nil")
	}
	RunSuite(t, Case{
		NewHandler: func() hostreqkit.Handler {
			return newHandler(manifest)
		},
		Name:               manifest.Name,
		Kind:               hostreqspec.KindTool,
		SupportedPlatforms: manifest.Platforms,
		Checks: []string{
			"name_and_kind",
			"inspect_manual_requirement",
			"apply_unsupported_returns_early",
			"apply_manual_returns_early",
			"apply_already_applied_skips",
			"apply_dry_run",
		},
	})
}

// RunSafeguardSuite keeps package-owned safeguard constructors on the same
// shared contract without repeating the Case assembly in every package.
func RunSafeguardSuite(t *testing.T, name string, newHandler func() hostreqkit.Handler, platforms []string, checks ...string) {
	t.Helper()
	RunSuite(t, Case{
		NewHandler:         newHandler,
		Name:               name,
		Kind:               hostreqspec.KindSafeguard,
		SupportedPlatforms: platforms,
		Checks:             checks,
	})
}

// APTRepoToolCase assembles the common platform matrix for tools installed
// from a vendor APT repository. Package adapters supply only manifest data and
// the seams that are genuinely package-owned.
func APTRepoToolCase(
	newHandler func() hostreqkit.Handler,
	manifest hostreqkit.ToolManifest,
	installCommand string,
	toolBinary string,
	versionOutput string,
	aptRepo *APTRepoCase,
	extraChecks ...string,
) Case {
	checks := []string{
		"name_and_kind",
		"inspect_manual_requirement",
		"apply_manual_returns_early",
		"apply_unsupported_returns_early",
		"inspect_linux_apt_not_installed",
		"inspect_linux_apt_installed",
		"inspect_darwin_brew",
		"inspect_windows_winget",
		"inspect_unsupported_configuration",
		"apply_installed_skips",
		"apply_linux_apt_dry_run",
		"apply_darwin_brew_flow",
		"apply_default_fallback_unsupported",
		"apply_linux_apt_full_flow",
		"apply_linux_key_download_failure",
	}
	checks = append(checks, extraChecks...)
	return Case{
		NewHandler:         newHandler,
		Name:               manifest.Name,
		Kind:               hostreqspec.KindTool,
		SupportedPlatforms: manifest.Platforms,
		InstallCommand:     installCommand,
		ToolBinary:         toolBinary,
		PackageNames:       manifest.Packages,
		VersionOutput:      versionOutput,
		APTRepo:            aptRepo,
		Checks:             checks,
	}
}

// Case describes the stable contract shared by a host requirement handler.
type Case struct {
	NewHandler         func() hostreqkit.Handler
	Name               string
	Kind               hostreqspec.Kind
	SupportedPlatforms []string
	InstallCommand     string
	DryRunNotes        []string
	ToolBinary         string
	PackageNames       map[string]string
	VersionOutput      string
	ManifestVersion    string
	DefaultVersion     string
	APTRepo            *APTRepoCase
	ManifestDefaults   *ManifestDefaultsCase
	PackageChecks      []PackageCheck
	// Checks limits a package adapter to the stamped rules it actually owns.
	// An empty list runs the complete shared suite.
	Checks []string
}

// PackageCheck preserves a package-specific behavior that is not shared with
// enough handlers to belong in the common matrix. RunSuite still owns its
// registration and stable subtest identity.
type PackageCheck struct {
	Name string
	Run  func(*testing.T)
}

// ManifestDefaultsCase declares the shared invariant between a safeguard's
// JSON schema and the defaults used by its handler.
type ManifestDefaultsCase struct {
	Load      func() (map[string]ManifestProperty, error)
	Required  []string
	Forbidden []string
	Expected  map[string]any
}

// ManifestProperty is the portion of a config property needed by the shared
// manifest-default check.
type ManifestProperty struct {
	Default any `json:"default"`
}

// APTRepoCase carries the two package-owned seams needed by repository-backed
// APT installers. The behavior and assertions remain in the shared suite.
type APTRepoCase struct {
	Setup          func() func()
	SetKeyDownload func(func() ([]byte, error))
	MinCommands    int
	SudoMode       string
}

type suiteT interface {
	Helper()
	Run(string, func(suiteT)) bool
	Errorf(string, ...any)
}

type realT struct{ *testing.T }

func (t realT) Run(name string, f func(suiteT)) bool {
	return t.T.Run(name, func(child *testing.T) { f(realT{child}) })
}

// RunSuite runs the shared handler contract with stable subtest names.
func RunSuite(t *testing.T, c Case) {
	t.Helper()
	runSuite(realT{t}, c)
	for _, check := range c.PackageChecks {
		check := check
		if strings.TrimSpace(check.Name) == "" || check.Run == nil {
			t.Errorf("package check requires a name and runner")
			continue
		}
		t.Run(check.Name, check.Run)
	}
}

//nolint:gocyclo // the requirement test harness reports setup, observation, remediation, and verification phases.
func runSuite(t suiteT, c Case) {
	t.Helper()
	if c.NewHandler == nil {
		t.Errorf("NewHandler is nil")
		return
	}
	if strings.TrimSpace(c.Name) == "" {
		t.Errorf("Name is empty")
	}

	t.Run("name_and_kind", func(t suiteT) {
		h := c.NewHandler()
		if h.Name() != c.Name {
			t.Errorf("Name() = %q, want %q", h.Name(), c.Name)
		}
		if h.Kind() != c.Kind {
			t.Errorf("Kind() = %q, want %q", h.Kind(), c.Kind)
		}
	})

	if enabled(c, "inspect_manual_requirement") {
		t.Run("inspect_manual_requirement", func(t suiteT) {
			h := c.NewHandler()
			status := h.Inspect(LinuxHost(), manualRequirement(c))
			if status.SupportClass != hostreqkit.SupportManualOnly {
				t.Errorf("SupportClass = %q, want %q", status.SupportClass, hostreqkit.SupportManualOnly)
			}
			if status.ExecutionState != hostreqkit.ExecutionManualActionRequired {
				t.Errorf("ExecutionState = %q, want %q", status.ExecutionState, hostreqkit.ExecutionManualActionRequired)
			}
		})
	}

	unsupported := unsupportedOS(c.SupportedPlatforms)
	if unsupported != "" && enabled(c, "inspect_unsupported_platform") {
		t.Run("inspect_unsupported_platform", func(t suiteT) {
			h := c.NewHandler()
			status := h.Inspect(hostreqkit.Host{OS: unsupported}, BaseRequirement(c))
			if status.SupportClass != hostreqkit.SupportUnsupported {
				t.Errorf("SupportClass = %q, want %q", status.SupportClass, hostreqkit.SupportUnsupported)
			}
		})
	}

	if enabled(c, "inspect_linux_apt_not_installed") {
		t.Run("inspect_linux_apt_not_installed", func(t suiteT) {
			original := hostreqkit.LookPathFn
			hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
			status := c.NewHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt-get", SupportsSysctl: true}, BaseRequirement(c))
			hostreqkit.LookPathFn = original
			if status.SupportClass != hostreqkit.SupportSupported {
				t.Errorf("SupportClass = %q, want %q", status.SupportClass, hostreqkit.SupportSupported)
			}
			if !status.InstallSupported {
				t.Errorf("InstallSupported = false, want true")
			}
			if status.PackageName != c.Name {
				t.Errorf("PackageName = %q, want %q", status.PackageName, c.Name)
			}
			if status.Installed {
				t.Errorf("Installed = true, want false")
			}
		})
	}

	if enabled(c, "inspect_linux_apt_installed") {
		t.Run("inspect_linux_apt_installed", func(t suiteT) {
			originalLookPath := hostreqkit.LookPathFn
			originalOutput := hostreqkit.CombinedOutputFn
			binary := firstNonEmpty(c.ToolBinary, c.Name)
			hostreqkit.LookPathFn = func(name string) (string, error) {
				if name == binary {
					return "/usr/bin/" + binary, nil
				}
				return "", os.ErrNotExist
			}
			hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) { return []byte(c.VersionOutput), nil }
			status := c.NewHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt", SupportsSysctl: true}, BaseRequirement(c))
			hostreqkit.LookPathFn = originalLookPath
			hostreqkit.CombinedOutputFn = originalOutput
			if !status.Installed {
				t.Errorf("Installed = false, want true")
			}
			if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
				t.Errorf("ExecutionState = %q, want %q", status.ExecutionState, hostreqkit.ExecutionAlreadyPresent)
			}
			if want := strings.TrimSpace(c.VersionOutput); status.Version != want {
				t.Errorf("Version = %q, want %q", status.Version, want)
			}
		})
	}

	if enabled(c, "inspect_darwin_brew") {
		runPackageInspection(t, c, "inspect_darwin_brew", hostreqkit.Host{OS: "darwin", PackageManager: "brew"}, "brew")
	}
	if enabled(c, "inspect_windows_winget") {
		runPackageInspection(t, c, "inspect_windows_winget", hostreqkit.Host{OS: "windows", PackageManager: "winget"}, "winget")
	}
	if enabled(c, "inspect_unsupported_configuration") {
		t.Run("inspect_unsupported_configuration", func(t suiteT) {
			original := hostreqkit.LookPathFn
			hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
			status := c.NewHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "dnf", SupportsSysctl: true}, BaseRequirement(c))
			hostreqkit.LookPathFn = original
			if status.SupportClass != hostreqkit.SupportUnsupported {
				t.Errorf("SupportClass = %q, want %q", status.SupportClass, hostreqkit.SupportUnsupported)
			}
		})
	}
	if enabled(c, "inspect_no_sysctl_not_applicable") {
		t.Run("inspect_no_sysctl_not_applicable", func(t suiteT) {
			host := LinuxHost()
			host.SupportsSysctl = false
			status := c.NewHandler().Inspect(host, BaseRequirement(c))
			if status.SupportClass != hostreqkit.SupportNotApplicable {
				t.Errorf("SupportClass = %q, want %q", status.SupportClass, hostreqkit.SupportNotApplicable)
			}
		})
	}
	if enabled(c, "pinned_version_matches_manifest") {
		t.Run("pinned_version_matches_manifest", func(t suiteT) {
			if c.ManifestVersion != c.DefaultVersion {
				t.Errorf("manifest version %q, default version %q", c.ManifestVersion, c.DefaultVersion)
			}
		})
	}
	if enabled(c, "defaults_match_manifest") {
		t.Run("defaults_match_manifest", func(t suiteT) {
			if c.ManifestDefaults == nil || c.ManifestDefaults.Load == nil {
				t.Errorf("ManifestDefaults configuration is incomplete")
				return
			}
			properties, err := c.ManifestDefaults.Load()
			if err != nil {
				t.Errorf("load manifest defaults: %v", err)
				return
			}
			for _, name := range c.ManifestDefaults.Required {
				if _, ok := properties[name]; !ok {
					t.Errorf("manifest declares no %q property", name)
				}
			}
			for _, name := range c.ManifestDefaults.Forbidden {
				if _, ok := properties[name]; ok {
					t.Errorf("obsolete manifest property remains: %s", name)
				}
			}
			for name, want := range c.ManifestDefaults.Expected {
				property, ok := properties[name]
				if !ok {
					t.Errorf("manifest declares no %q property", name)
					continue
				}
				if !reflect.DeepEqual(property.Default, want) {
					t.Errorf("%s: manifest default %v, want %v", name, property.Default, want)
				}
			}
		})
	}

	if enabled(c, "apply_unsupported_returns_early") {
		t.Run("apply_unsupported_returns_early", func(t suiteT) {
			h := c.NewHandler()
			status, err := h.Apply(hostreqkit.Host{OS: unsupportedOr("darwin", unsupported)}, hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportUnsupported}, hostreqkit.EnsureOptions{})
			if err != nil {
				t.Errorf("Apply() error = %v", err)
			}
			if status.ExecutionState != hostreqkit.ExecutionUnsupported {
				t.Errorf("ExecutionState = %q, want %q", status.ExecutionState, hostreqkit.ExecutionUnsupported)
			}
		})
	}

	if enabled(c, "apply_manual_returns_early") {
		t.Run("apply_manual_returns_early", func(t suiteT) {
			h := c.NewHandler()
			status, err := h.Apply(LinuxHost(), hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportManualOnly}, hostreqkit.EnsureOptions{})
			if err != nil {
				t.Errorf("Apply() error = %v", err)
			}
			if status.ExecutionState != hostreqkit.ExecutionManualActionRequired {
				t.Errorf("ExecutionState = %q, want %q", status.ExecutionState, hostreqkit.ExecutionManualActionRequired)
			}
		})
	}

	if enabled(c, "apply_already_applied_skips") {
		t.Run("apply_already_applied_skips", func(t suiteT) {
			h := c.NewHandler()
			status, err := h.Apply(LinuxHost(), hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported, Installed: true, Applied: true, ExecutionState: hostreqkit.ExecutionAlreadyPresent}, hostreqkit.EnsureOptions{})
			if err != nil {
				t.Errorf("Apply() error = %v", err)
			}
			if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
				t.Errorf("ExecutionState = %q, want %q", status.ExecutionState, hostreqkit.ExecutionAlreadyPresent)
			}
		})
	}

	if enabled(c, "apply_installed_skips") {
		t.Run("apply_installed_skips", func(t suiteT) {
			status, err := c.NewHandler().Apply(LinuxHost(), hostreqkit.ItemStatus{Installed: true}, hostreqkit.EnsureOptions{})
			if err != nil {
				t.Errorf("Apply() error = %v", err)
			}
			if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
				t.Errorf("ExecutionState = %q, want %q", status.ExecutionState, hostreqkit.ExecutionAlreadyPresent)
			}
		})
	}

	if enabled(c, "apply_not_applicable_returns_early") {
		t.Run("apply_not_applicable_returns_early", func(t suiteT) {
			status, err := c.NewHandler().Apply(LinuxHost(), hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportNotApplicable}, hostreqkit.EnsureOptions{})
			if err != nil {
				t.Errorf("Apply() error = %v", err)
			}
			if status.ExecutionState != hostreqkit.ExecutionNotApplicable {
				t.Errorf("ExecutionState = %q, want %q", status.ExecutionState, hostreqkit.ExecutionNotApplicable)
			}
		})
	}

	if enabled(c, "apply_linux_apt_dry_run") {
		t.Run("apply_linux_apt_dry_run", func(t suiteT) {
			status, err := c.NewHandler().Apply(hostreqkit.Host{OS: "linux", PackageManager: "apt-get"}, hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}, hostreqkit.EnsureOptions{DryRun: true})
			if err != nil {
				t.Errorf("Apply() error = %v", err)
			}
			if status.ExecutionState != hostreqkit.ExecutionWouldInstall {
				t.Errorf("ExecutionState = %q, want %q", status.ExecutionState, hostreqkit.ExecutionWouldInstall)
			}
		})
	}

	if enabled(c, "apply_darwin_brew_flow") {
		runAPTRepoCheck(t, c, "apply_darwin_brew_flow", func(t suiteT, cluster *APTRepoCase) {
			var commands []string
			binary := firstNonEmpty(c.ToolBinary, c.Name)
			hostreqkit.LookPathFn = func(name string) (string, error) {
				if name == binary && len(commands) > 0 {
					return "/usr/local/bin/" + binary, nil
				}
				return "", os.ErrNotExist
			}
			hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
				commands = append(commands, name+" "+strings.Join(args, " "))
				return nil
			}
			hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) { return []byte(c.VersionOutput), nil }
			status, err := c.NewHandler().Apply(hostreqkit.Host{OS: "darwin", PackageManager: "brew"}, hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}, hostreqkit.EnsureOptions{AutoInstall: true})
			if err != nil {
				t.Errorf("Apply() error = %v", err)
			}
			if status.ExecutionState != hostreqkit.ExecutionInstalled {
				t.Errorf("ExecutionState = %q, want %q", status.ExecutionState, hostreqkit.ExecutionInstalled)
			}
			if len(commands) != 1 || !strings.Contains(commands[0], "brew install") {
				t.Errorf("brew commands = %v, want one install", commands)
			}
		})
	}

	if enabled(c, "apply_default_fallback_unsupported") {
		runAPTRepoCheck(t, c, "apply_default_fallback_unsupported", func(t suiteT, _ *APTRepoCase) {
			status, err := c.NewHandler().Apply(hostreqkit.Host{OS: "linux", PackageManager: "dnf"}, hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}, hostreqkit.EnsureOptions{AutoInstall: true})
			if err != nil {
				t.Errorf("Apply() error = %v", err)
			}
			if status.SupportClass != hostreqkit.SupportUnsupported {
				t.Errorf("SupportClass = %q, want %q", status.SupportClass, hostreqkit.SupportUnsupported)
			}
		})
	}

	if enabled(c, "apply_linux_apt_full_flow") {
		runAPTRepoCheck(t, c, "apply_linux_apt_full_flow", func(t suiteT, cluster *APTRepoCase) {
			var commands []string
			binary := firstNonEmpty(c.ToolBinary, c.Name)
			hostreqkit.LookPathFn = func(name string) (string, error) {
				if name == "gpg" || name == "sudo" {
					return "/usr/bin/" + name, nil
				}
				if name == binary && len(commands) > 0 {
					return "/usr/bin/" + binary, nil
				}
				return "", os.ErrNotExist
			}
			hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
				commands = append(commands, name+" "+strings.Join(args, " "))
				return nil
			}
			hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) { return []byte(c.VersionOutput), nil }
			cluster.SetKeyDownload(func() ([]byte, error) { return []byte("fixture-gpg-key"), nil })
			sudoMode := firstNonEmpty(cluster.SudoMode, "skip")
			status, err := c.NewHandler().Apply(hostreqkit.Host{OS: "linux", PackageManager: "apt-get"}, hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}, hostreqkit.EnsureOptions{AutoInstall: true, SudoMode: sudoMode})
			if err != nil {
				t.Errorf("Apply() error = %v", err)
			}
			if status.ExecutionState != hostreqkit.ExecutionInstalled {
				t.Errorf("ExecutionState = %q, want %q; notes=%v", status.ExecutionState, hostreqkit.ExecutionInstalled, status.Notes)
			}
			if len(commands) < cluster.MinCommands {
				t.Errorf("commands = %v, want at least %d", commands, cluster.MinCommands)
			}
		})
	}

	if enabled(c, "apply_linux_key_download_failure") {
		runAPTRepoCheck(t, c, "apply_linux_key_download_failure", func(t suiteT, cluster *APTRepoCase) {
			hostreqkit.LookPathFn = func(name string) (string, error) {
				if name == "gpg" {
					return "/usr/bin/gpg", nil
				}
				return "", os.ErrNotExist
			}
			cluster.SetKeyDownload(func() ([]byte, error) { return nil, os.ErrPermission })
			status, err := c.NewHandler().Apply(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}, hostreqkit.EnsureOptions{AutoInstall: true, SudoMode: "skip"})
			if err != nil {
				t.Errorf("Apply() error = %v", err)
			}
			if status.ExecutionState != hostreqkit.ExecutionFailed {
				t.Errorf("ExecutionState = %q, want %q", status.ExecutionState, hostreqkit.ExecutionFailed)
			}
		})
	}

	if enabled(c, "apply_dry_run") {
		t.Run("apply_dry_run", func(t suiteT) {
			h := c.NewHandler()
			status, err := h.Apply(LinuxHost(), hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported, Installed: true, ExecutionState: hostreqkit.ExecutionAlreadyPresent}, hostreqkit.EnsureOptions{DryRun: true})
			if err != nil {
				t.Errorf("dry-run Apply() error = %v", err)
				return
			}
			if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent && status.ExecutionState != hostreqkit.ExecutionWouldApply && status.ExecutionState != hostreqkit.ExecutionWouldInstall {
				t.Errorf("dry-run ExecutionState = %q, want already_present, would_apply, or would_install", status.ExecutionState)
			}
			if status.Applied {
				t.Errorf("dry-run marked the requirement Applied")
			}
		})
	}
}

func runAPTRepoCheck(t suiteT, c Case, name string, check func(suiteT, *APTRepoCase)) {
	t.Run(name, func(t suiteT) {
		if c.APTRepo == nil || c.APTRepo.Setup == nil || c.APTRepo.SetKeyDownload == nil {
			t.Errorf("APTRepo configuration is incomplete")
			return
		}
		restore := c.APTRepo.Setup()
		defer restore()
		check(t, c.APTRepo)
	})
}

func runPackageInspection(t suiteT, c Case, check string, host hostreqkit.Host, packageKey string) {
	t.Run(check, func(t suiteT) {
		original := hostreqkit.LookPathFn
		hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
		host.SupportsSysctl = true
		status := c.NewHandler().Inspect(host, BaseRequirement(c))
		hostreqkit.LookPathFn = original
		if status.SupportClass != hostreqkit.SupportSupported {
			t.Errorf("SupportClass = %q, want %q", status.SupportClass, hostreqkit.SupportSupported)
		}
		if !status.InstallSupported {
			t.Errorf("InstallSupported = false, want true")
		}
		if want := c.PackageNames[packageKey]; status.PackageName != want {
			t.Errorf("PackageName = %q, want %q", status.PackageName, want)
		}
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// LoadManifestProperties decodes the shared config.properties shape from a
// safeguard manifest while tolerating unrelated and future fields.
func LoadManifestProperties(path string) (map[string]ManifestProperty, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest struct {
		Config struct {
			Properties map[string]ManifestProperty `json:"properties"`
		} `json:"config"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return manifest.Config.Properties, nil
}

func enabled(c Case, check string) bool {
	if len(c.Checks) == 0 {
		return true
	}
	for _, candidate := range c.Checks {
		if candidate == check {
			return true
		}
	}
	return false
}

// LinuxHost is the neutral host fixture used by the shared suite.
func LinuxHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", PackageManager: "apt-get", SupportsSetup: true, SupportsDevelop: true, SupportsSysctl: true, SupportsSystemd: true}
}

// BaseRequirement returns a required, non-manual requirement for a case.
func BaseRequirement(c Case) hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: c.Name, Kind: c.Kind, Required: true}
}

// LinuxReq is the Linux alias retained for package-specific unique tests.
func LinuxReq(c Case) hostreqspec.ResolvedRequirement { return BaseRequirement(c) }

// StubAll is a cleanup hook for package-specific tests. Global seams are
// owned by the package under test, so the shared suite does not mutate them.
func StubAll(t *testing.T) func() {
	t.Helper()
	return func() {}
}

// StubLookups is the lookup-only counterpart of StubAll.
func StubLookups(t *testing.T) func() { return StubAll(t) }

// StubInvokingUser isolates tests that exercise installation into a user's
// bin directory from the developer's real home and PATH.
func StubInvokingUser(t *testing.T) (string, func()) {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	origReadFile := hostreqkit.ReadFileFn
	origRoot := hostreqkit.RunningAsRootFn

	tmp := testenv.RuntimeHome(t)
	hostreqkit.RunningAsRootFn = func() bool { return false }
	testenv.AsCurrentUser(t, "alice")
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == "/etc/passwd" {
			return []byte("alice:x:1000:1000:Alice:" + tmp + ":/bin/sh\n"), nil
		}
		return nil, os.ErrNotExist
	}
	return tmp, func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.RunningAsRootFn = origRoot
	}
}

// AssertInstalled checks the common installed-and-versioned observation made
// by tool-specific tests that also verify a handler's version probe.
func AssertInstalled(t *testing.T, status hostreqkit.ItemStatus, version string) {
	t.Helper()
	if !status.Installed {
		t.Fatal("should be installed")
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
	if status.Version != version {
		t.Fatalf("Version = %q", status.Version)
	}
}

// AssertExecutionState checks a handler result without repeating the same
// failure text in each package-specific test.
func AssertExecutionState(t *testing.T, status hostreqkit.ItemStatus, want hostreqkit.ExecutionState) {
	t.Helper()
	if status.ExecutionState != want {
		t.Fatalf("ExecutionState = %q, want %q", status.ExecutionState, want)
	}
}

// AssertInstallSupported checks the common package-discovery assertion used
// by handlers whose unique test verifies a pinned package name.
func AssertInstallSupported(t *testing.T, status hostreqkit.ItemStatus, packageNeedle string) {
	t.Helper()
	if !status.InstallSupported {
		t.Fatal("InstallSupported should be true with the prerequisite available")
	}
	if !strings.Contains(status.PackageName, packageNeedle) {
		t.Fatalf("PackageName = %q; want %q", status.PackageName, packageNeedle)
	}
}

// RunInstallSupportedProbe keeps the prerequisite setup and common discovery
// assertion together while leaving the package-specific lookup seam injectable.
func RunInstallSupportedProbe(t *testing.T, configure func(), inspect func() hostreqkit.ItemStatus, packageNeedle string) {
	t.Helper()
	configure()
	AssertInstallSupported(t, inspect(), packageNeedle)
}

// AssertSupportClass checks the result of a package-specific fallback probe.
func AssertSupportClass(t *testing.T, status hostreqkit.ItemStatus, want hostreqkit.SupportClass) {
	t.Helper()
	if status.SupportClass != want {
		t.Fatalf("SupportClass = %q, want %q", status.SupportClass, want)
	}
}

// RunFallbackUnsupported keeps the common unsupported-package-manager probe
// shared between handlers with otherwise distinct install implementations.
func RunFallbackUnsupported(t *testing.T, apply func() (hostreqkit.ItemStatus, error)) {
	t.Helper()
	result, err := apply()
	if err != nil {
		t.Fatal(err)
	}
	AssertSupportClass(t, result, hostreqkit.SupportUnsupported)
}

// AssertDryRunNote checks the common dry-run state and note evidence while
// retaining the package-specific assertion about the requested package.
func AssertDryRunNote(t *testing.T, status hostreqkit.ItemStatus, packageNeedle string) {
	t.Helper()
	if status.ExecutionState != hostreqkit.ExecutionWouldInstall {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
	for _, note := range status.Notes {
		if strings.Contains(note, "npm install") && strings.Contains(note, packageNeedle) {
			return
		}
	}
	t.Fatalf("dry-run note should mention npm install and %q, got %v", packageNeedle, status.Notes)
}

// RunApplyWithCommandEvidence runs an install flow and checks its state and
// recorded command without repeating that harness in each tool package.
func RunApplyWithCommandEvidence(t *testing.T, configure func(), apply func() (hostreqkit.ItemStatus, error), commands func() []string, fragment string) {
	t.Helper()
	configure()
	result, err := apply()
	if err != nil {
		t.Fatal(err)
	}
	AssertExecutionState(t, result, hostreqkit.ExecutionInstalled)
	AssertSingleCommandContaining(t, commands(), fragment)
}

// RunBrewInstallFlow owns the common command recorder for package-specific
// Homebrew tests. The lookup callback retains each handler's distinct binary
// and version behavior.
func RunBrewInstallFlow(t *testing.T, lookPath func(string, int) (string, error), version string, apply func() (hostreqkit.ItemStatus, error), fragment string) {
	t.Helper()
	var commands []string
	hostreqkit.LookPathFn = func(name string) (string, error) { return lookPath(name, len(commands)) }
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		commands = append(commands, name+" "+strings.Join(args, " "))
		return nil
	}
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return []byte(version + "\n"), nil
	}
	result, err := apply()
	if err != nil {
		t.Fatal(err)
	}
	AssertExecutionState(t, result, hostreqkit.ExecutionInstalled)
	AssertSingleCommandContaining(t, commands, fragment)
}

// AssertSingleCommandContaining checks the command evidence retained by
// package-specific install-flow tests.
func AssertSingleCommandContaining(t *testing.T, commands []string, fragment string) {
	t.Helper()
	if len(commands) != 1 || !strings.Contains(commands[0], fragment) {
		t.Fatalf("expected %s command, got %v", fragment, commands)
	}
}

// RunApplyShortCircuitCases exercises the shared terminal support-state
// contract while allowing each package to provide its own handler and command
// recorder.
func RunApplyShortCircuitCases(t *testing.T, apply func(*testing.T, hostreqkit.SupportClass) (hostreqkit.ItemStatus, int, error)) {
	t.Helper()
	cases := []struct {
		name string
		sc   hostreqkit.SupportClass
		want hostreqkit.ExecutionState
	}{
		{"unsupported", hostreqkit.SupportUnsupported, hostreqkit.ExecutionUnsupported},
		{"not_applicable", hostreqkit.SupportNotApplicable, hostreqkit.ExecutionNotApplicable},
		{"manual_only", hostreqkit.SupportManualOnly, hostreqkit.ExecutionManualActionRequired},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			out, commandCount, err := apply(t, c.sc)
			if err != nil {
				t.Fatalf("Apply: %v", err)
			}
			AssertExecutionState(t, out, c.want)
			if commandCount != 0 {
				t.Errorf("commands ran: %d", commandCount)
			}
		})
	}
}

func manualRequirement(c Case) hostreqspec.ResolvedRequirement {
	requirement := BaseRequirement(c)
	requirement.Manual = true
	return requirement
}

func unsupportedOS(platforms []string) string {
	for _, candidate := range []string{"darwin", "windows", "linux"} {
		if !containsPlatform(platforms, candidate) {
			return candidate
		}
	}
	return ""
}

func unsupportedOr(fallback, value string) string {
	if value == "" {
		return fallback
	}
	return value
}

func containsPlatform(platforms []string, candidate string) bool {
	for _, platform := range platforms {
		platform = strings.ToLower(strings.TrimSpace(platform))
		if platform == candidate || candidate == "darwin" && platform == "macos" {
			return true
		}
	}
	return false
}
