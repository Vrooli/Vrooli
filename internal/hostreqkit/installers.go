package hostreqkit

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// AptRepoInstaller describes a tool installed from an apt repository. The
// same installer also handles the package-manager fallbacks declared by the
// tool manifest; only the repository-specific values belong to the caller.
type AptRepoInstaller struct {
	Manifest             ToolManifest
	AptPackage           string
	KeyringPath          string
	SourcePath           string
	KeyURL               string
	SourceLine           func(Host) string
	DownloadKey          func() ([]byte, error)
	BinaryKey            bool
	AptDryRunNotes       []string
	BrewDryRunNote       string
	WingetDryRunNote     string
	UnsupportedApplyNote string
}

// Inspect returns the common tool status for an apt-repository-backed tool.
func (i AptRepoInstaller) Inspect(host Host, requirement hostreqspec.ResolvedRequirement) ItemStatus {
	status := BaseStatus(requirement)
	status.Command, status.Installed = ResolveCommand(i.Manifest.Commands)
	status.SupportClass = SupportSupported
	status.Notes = append(status.Notes, i.Manifest.InstallHint)
	if requirement.Manual {
		status.SupportClass = SupportManualOnly
		status.ExecutionState = ExecutionManualActionRequired
		return status
	}

	switch {
	case host.OS == "linux" && (host.PackageManager == "apt" || host.PackageManager == "apt-get"):
		status.PackageName = i.AptPackage
		status.InstallSupported = true
	case host.OS == "darwin" && host.PackageManager == "brew":
		status.PackageName = i.Manifest.Packages["brew"]
		status.InstallSupported = true
	case host.OS == "windows" && host.PackageManager == "winget":
		status.PackageName = i.Manifest.Packages["winget"]
		status.InstallSupported = status.PackageName != ""
	default:
		status.SupportClass = SupportUnsupported
		status.ExecutionState = ExecutionUnsupported
	}

	if status.Installed {
		status.ExecutionState = ExecutionAlreadyPresent
		status.Version = ReadVersion(status.Command, i.Manifest.VersionArgs)
	}
	return status
}

// Apply performs the shared installation flow. Handler-specific notes remain
// at the call site because they are part of each tool's operator contract.
func (i AptRepoInstaller) Apply(host Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error) {
	if status.Installed {
		status.ExecutionState = ExecutionAlreadyPresent
		return status, nil
	}
	if status.SupportClass != SupportSupported {
		return status, nil
	}

	switch {
	case host.OS == "linux" && (host.PackageManager == "apt" || host.PackageManager == "apt-get"):
		if err := i.installLinux(host, opts); err != nil {
			return status, err
		}
	case host.OS == "darwin" && host.PackageManager == "brew":
		packageName := status.PackageName
		if packageName == "" {
			packageName = i.Manifest.Packages["brew"]
		}
		command, args, err := InstallCommand(host, packageName, opts.SudoMode)
		if err != nil {
			return status, err
		}
		if err := RunInstallCommand(command, args, opts); err != nil {
			return status, err
		}
	case host.OS == "windows" && host.PackageManager == "winget":
		packageName := status.PackageName
		if packageName == "" {
			packageName = i.Manifest.Packages["winget"]
		}
		command, args, err := InstallCommand(host, packageName, opts.SudoMode)
		if err != nil {
			return status, err
		}
		if err := RunInstallCommand(command, args, opts); err != nil {
			return status, err
		}
	default:
		status.SupportClass = SupportUnsupported
		status.ExecutionState = ExecutionUnsupported
		return status, nil
	}

	status.Command, status.Installed = ResolveCommand(i.Manifest.Commands)
	if status.Installed {
		status.ExecutionState = ExecutionInstalled
		status.Version = ReadVersion(status.Command, i.Manifest.VersionArgs)
		return status, nil
	}
	status.ExecutionState = ExecutionFailed
	return status, fmt.Errorf("install commands completed but %s is still not available on PATH", i.Manifest.Name)
}

// ApplyWithNotes preserves the small amount of operator-facing variation in
// the APT handlers while sharing their status and error transitions.
func (i AptRepoInstaller) ApplyWithNotes(host Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error) {
	if status, done := ApplyStatus(status); done {
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = ExecutionWouldInstall
		switch {
		case host.OS == "linux" && (host.PackageManager == "apt" || host.PackageManager == "apt-get"):
			status.Notes = append(status.Notes, i.AptDryRunNotes...)
		case host.OS == "darwin" && host.PackageManager == "brew":
			status.Notes = append(status.Notes, i.BrewDryRunNote+" "+i.Manifest.Packages["brew"])
		case host.OS == "windows" && host.PackageManager == "winget":
			status.Notes = append(status.Notes, i.WingetDryRunNote+" "+i.Manifest.Packages["winget"])
		default:
			status.SupportClass = SupportUnsupported
			status.ExecutionState = ExecutionUnsupported
			status.Notes = append(status.Notes, i.UnsupportedApplyNote)
		}
		return status, nil
	}
	result, err := i.Apply(host, status, opts)
	if err != nil {
		result.ExecutionState = ExecutionFailed
		result.Notes = append(result.Notes, err.Error())
	}
	if result.SupportClass == SupportUnsupported && i.UnsupportedApplyNote != "" {
		result.Notes = append(result.Notes, i.UnsupportedApplyNote)
	}
	return result, nil
}

func (i AptRepoInstaller) installLinux(host Host, opts EnsureOptions) error {
	keyData, err := i.downloadKey()
	if err != nil {
		return err
	}
	keyFile, err := os.CreateTemp("", "vrooli-apt-key-*")
	if err != nil {
		return fmt.Errorf("create apt key temp file: %w", err)
	}
	keyTempPath := keyFile.Name()
	if _, err := keyFile.Write(keyData); err != nil {
		_ = keyFile.Close()
		return fmt.Errorf("write apt key temp file: %w", err)
	}
	if err := keyFile.Close(); err != nil {
		return fmt.Errorf("close apt key temp file: %w", err)
	}
	defer os.Remove(keyTempPath)

	installKeyPath := keyTempPath
	if !i.BinaryKey {
		converted, err := os.CreateTemp("", "vrooli-apt-keyring-*")
		if err != nil {
			return fmt.Errorf("create apt keyring temp file: %w", err)
		}
		installKeyPath = converted.Name()
		if err := converted.Close(); err != nil {
			return fmt.Errorf("close apt keyring temp file: %w", err)
		}
		defer os.Remove(installKeyPath)
		if !CommandAvailable("gpg") {
			command, args, err := InstallCommand(host, "gpg", opts.SudoMode)
			if err != nil {
				return err
			}
			if err := RunInstallCommand(command, args, opts); err != nil {
				return err
			}
		}
		if err := RunCommandFn("gpg", []string{"--dearmor", "--yes", "--output", installKeyPath, keyTempPath}, opts); err != nil {
			return err
		}
	}

	sourceFile, err := os.CreateTemp("", "vrooli-apt-source-*.list")
	if err != nil {
		return fmt.Errorf("create apt source temp file: %w", err)
	}
	sourceTempPath := sourceFile.Name()
	line := ""
	if i.SourceLine != nil {
		line = i.SourceLine(host)
	}
	if _, err := sourceFile.WriteString(line + "\n"); err != nil {
		_ = sourceFile.Close()
		return fmt.Errorf("write apt source temp file: %w", err)
	}
	if err := sourceFile.Close(); err != nil {
		return fmt.Errorf("close apt source temp file: %w", err)
	}
	defer os.Remove(sourceTempPath)

	for _, dir := range []string{filepath.Dir(i.KeyringPath), filepath.Dir(i.SourcePath)} {
		if err := RunPrivilegedCommand(opts.SudoMode, "mkdir", []string{"-p", dir}, opts); err != nil {
			return err
		}
	}
	if err := RunPrivilegedCommand(opts.SudoMode, "install", []string{"-m", "0644", installKeyPath, i.KeyringPath}, opts); err != nil {
		return err
	}
	if err := RunPrivilegedCommand(opts.SudoMode, "install", []string{"-m", "0644", sourceTempPath, i.SourcePath}, opts); err != nil {
		return err
	}
	if err := RunPrivilegedCommand(opts.SudoMode, "apt-get", []string{"update", "-qq"}, opts); err != nil {
		return err
	}
	command, args, err := InstallCommand(host, i.AptPackage, opts.SudoMode)
	if err != nil {
		return err
	}
	return RunInstallCommand(command, args, opts)
}

func (i AptRepoInstaller) downloadKey() ([]byte, error) {
	if i.DownloadKey != nil {
		return i.DownloadKey()
	}
	if strings.TrimSpace(i.KeyURL) == "" {
		return nil, fmt.Errorf("apt repository signing key URL is empty")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Get(i.KeyURL)
	if err != nil {
		return nil, fmt.Errorf("download apt repository signing key: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("download apt repository signing key: unexpected HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read apt repository signing key: %w", err)
	}
	return data, nil
}

// InstallKind identifies the per-user package command used by
// GoInstallInstaller. The npm mode is included because two long-standing
// protoc plugin handlers use the same per-user install, link, and provenance
// workflow with npm rather than go.
type InstallKind string

const (
	InstallKindGo  InstallKind = "go"
	InstallKindNPM InstallKind = "npm"
)

// GoInstallInstaller centralizes per-user binary installation. Despite its
// historical name, it also supports npm-backed command-line plugins through
// InstallKindNPM; this keeps the shared contract honest to the existing tool
// manifests while preserving the public type named by the plan.
type GoInstallInstaller struct {
	Manifest        ToolManifest
	ModulePath      string
	Version         string
	BinaryName      string
	Kind            InstallKind
	CacheDir        func() (string, error)
	RecordArtifacts func(home, source, prefix, link string) error
}

func (i GoInstallInstaller) packageRef() string { return i.ModulePath + "@" + i.Version }

// VersionOrDefault resolves an optional manifest version for installer
// constructors without repeating the same trim-and-fallback logic.
func VersionOrDefault(declared, fallback string) string {
	if value := strings.TrimSpace(declared); value != "" {
		return value
	}
	return fallback
}

func (i GoInstallInstaller) Inspect(host Host, requirement hostreqspec.ResolvedRequirement) ItemStatus {
	status := BaseStatus(requirement)
	status.Command, status.Installed = ResolveCommandForInvokingUser(i.Manifest.Commands)
	status.SupportClass = SupportSupported
	status.Notes = append(status.Notes, i.Manifest.InstallHint)
	if requirement.Manual {
		status.SupportClass = SupportManualOnly
		status.ExecutionState = ExecutionManualActionRequired
		return status
	}
	prerequisite := "go"
	if i.Kind == InstallKindNPM {
		prerequisite = "npm"
	}
	if !CommandAvailable(prerequisite) {
		status.SupportClass = SupportUnsupported
		status.ExecutionState = ExecutionUnsupported
		status.Notes = append(status.Notes, prerequisite+" not on PATH; install the required toolchain first")
		return status
	}
	status.InstallSupported = true
	status.PackageName = i.packageRef()
	if status.Installed {
		status.ExecutionState = ExecutionAlreadyPresent
		status.Version = ReadVersion(status.Command, i.Manifest.VersionArgs)
	}
	return status
}

func (i GoInstallInstaller) Apply(status ItemStatus, opts EnsureOptions) (ItemStatus, error) {
	if status.Installed {
		status.ExecutionState = ExecutionAlreadyPresent
		return status, nil
	}
	if status.SupportClass != SupportSupported {
		return status, nil
	}

	prefix, source, err := i.installPaths()
	if err != nil {
		return status, err
	}
	if opts.DryRun {
		status.ExecutionState = ExecutionWouldInstall
		if i.Kind == InstallKindNPM {
			status.Notes = append(status.Notes, fmt.Sprintf("dry-run: npm install --prefix %s %s", prefix, i.packageRef()))
		} else {
			status.Notes = append(status.Notes, "dry-run: go install "+i.packageRef())
		}
		return status, nil
	}

	if i.Kind == InstallKindNPM {
		if err := RunAsInvokingUser("mkdir", []string{"-p", prefix}, opts); err != nil {
			return status, fmt.Errorf("create %s: %w", prefix, err)
		}
		args := []string{"install", "--prefix", prefix, "--no-audit", "--no-fund", "--silent", i.packageRef()}
		if err := RunAsInvokingUser("npm", args, opts); err != nil {
			return status, err
		}
	} else if err := RunAsInvokingUser("go", []string{"install", i.packageRef()}, opts); err != nil {
		return status, err
	}

	home, err := InvokingUserHomeDir()
	if err != nil {
		return status, fmt.Errorf("resolve home dir: %w", err)
	}
	link := filepath.Join(home, ".local", "bin", i.BinaryName)
	if err := EnsureUserToolLink(source, link, opts); err != nil {
		status.Notes = append(status.Notes, "post-install symlink: "+err.Error())
	}
	if i.RecordArtifacts != nil {
		if err := i.RecordArtifacts(home, source, prefix, link); err != nil {
			return status, fmt.Errorf("record install inventory: %w", err)
		}
	}

	status.Command, status.Installed = ResolveCommandForInvokingUser(i.Manifest.Commands)
	if !status.Installed {
		status.ExecutionState = ExecutionFailed
		return status, fmt.Errorf("install %s succeeded but %s is not on PATH; ensure the per-user install directory is on PATH or rerun setup", i.packageRef(), i.BinaryName)
	}
	status.ExecutionState = ExecutionInstalled
	status.Version = ReadVersion(status.Command, i.Manifest.VersionArgs)
	return status, nil
}

func (i GoInstallInstaller) installPaths() (prefix, source string, err error) {
	if i.Kind == InstallKindNPM {
		if i.CacheDir == nil {
			return "", "", fmt.Errorf("npm install cache directory is undefined")
		}
		prefix, err = i.CacheDir()
		if err != nil {
			return "", "", err
		}
		source = filepath.Join(prefix, "node_modules", ".bin", i.BinaryName)
		return prefix, source, nil
	}
	if gobin := strings.TrimSpace(os.Getenv("GOBIN")); gobin != "" {
		prefix = gobin
	} else if gopath := strings.TrimSpace(os.Getenv("GOPATH")); gopath != "" {
		prefix = filepath.Join(gopath, "bin")
	} else {
		home, homeErr := InvokingUserHomeDir()
		if homeErr != nil {
			return "", "", homeErr
		}
		prefix = filepath.Join(home, "go", "bin")
	}
	return prefix, filepath.Join(prefix, i.BinaryName), nil
}

// EnsureUserToolLink creates the standard ~/.local/bin link used by
// per-user tool installs.
func EnsureUserToolLink(source, link string, opts EnsureOptions) error {
	if _, err := os.Stat(source); err != nil {
		return fmt.Errorf("post-install: %s missing at %s: %w", filepath.Base(source), source, err)
	}
	if err := RunAsInvokingUser("mkdir", []string{"-p", filepath.Dir(link)}, opts); err != nil {
		return fmt.Errorf("ensure %s: %w", filepath.Dir(link), err)
	}
	if err := RunAsInvokingUser("ln", []string{"-sfn", source, link}, opts); err != nil {
		return fmt.Errorf("create symlink %s -> %s: %w", link, source, err)
	}
	return nil
}

// SysctlParameter is one managed sysctl key. Minimum=true means values above
// the declared value are accepted; otherwise the value must match exactly.
type SysctlParameter struct {
	Name        string
	Value       int
	Minimum     bool
	ReadFailure int
}

// SysctlApplier centralizes inspection and application of managed sysctl
// files. Note strings are configurable so each safeguard retains its own
// operator-facing language.
type SysctlApplier struct {
	ConfigPath        string
	Parameters        []SysctlParameter
	UnsupportedNote   string
	NotApplicableNote string
	ManualNote        string
	PendingNote       string
	AppliedNote       string
	DryRunNote        string
}

func (a SysctlApplier) Inspect(host Host, requirement hostreqspec.ResolvedRequirement) ItemStatus {
	status := BaseStatus(requirement)
	status.SupportClass = SupportSupported
	if requirement.Manual {
		status.SupportClass = SupportManualOnly
		status.ExecutionState = ExecutionManualActionRequired
		if a.ManualNote != "" {
			status.Notes = append(status.Notes, a.ManualNote)
		}
		return status
	}
	if host.OS != "linux" {
		status.SupportClass = SupportUnsupported
		status.ExecutionState = ExecutionUnsupported
		status.Notes = append(status.Notes, a.UnsupportedNote)
		return status
	}
	if !host.SupportsSysctl {
		status.SupportClass = SupportNotApplicable
		status.ExecutionState = ExecutionNotApplicable
		status.Notes = append(status.Notes, a.NotApplicableNote)
		return status
	}

	pending := a.pending()
	if !FileContentMatches(a.ConfigPath, a.ConfigContent()) && len(pending) == 0 {
		pending = append(pending, a.ConfigPath+" needs update")
	}
	if len(pending) == 0 {
		status.Applied = true
		status.ExecutionState = ExecutionAlreadyPresent
		status.Notes = append(status.Notes, a.AppliedNote)
		return status
	}
	status.Notes = append(status.Notes, a.PendingNote, "pending: "+strings.Join(pending, ", "))
	return status
}

func (a SysctlApplier) Apply(status ItemStatus, opts EnsureOptions) (ItemStatus, error) {
	switch status.SupportClass {
	case SupportUnsupported:
		status.ExecutionState = ExecutionUnsupported
		return status, nil
	case SupportNotApplicable:
		status.ExecutionState = ExecutionNotApplicable
		return status, nil
	case SupportManualOnly:
		status.ExecutionState = ExecutionManualActionRequired
		if a.ManualNote != "" {
			status.Notes = append(status.Notes, a.ManualNote)
		}
		return status, nil
	}
	if status.Applied {
		status.ExecutionState = ExecutionAlreadyPresent
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = ExecutionWouldApply
		status.Notes = append(status.Notes, a.DryRunNote)
		return status, nil
	}
	if err := EnsureManagedDir(filepath.Dir(a.ConfigPath), opts.SudoMode, opts); err != nil {
		return status, err
	}
	if err := InstallManagedContent(a.ConfigPath, a.ConfigContent(), opts.SudoMode, opts); err != nil {
		return status, err
	}
	if err := RunPrivilegedCommand(opts.SudoMode, "sysctl", []string{"--system"}, opts); err != nil {
		return status, err
	}
	status.Applied = true
	status.ExecutionState = ExecutionApplied
	return status, nil
}

func (a SysctlApplier) pending() []string {
	pending := make([]string, 0, len(a.Parameters))
	for _, parameter := range a.Parameters {
		path := "/proc/sys/" + strings.ReplaceAll(parameter.Name, ".", "/")
		current := parameter.ReadFailure
		if data, err := ReadFileFn(path); err == nil {
			if _, scanErr := fmt.Sscan(strings.TrimSpace(string(data)), &current); scanErr != nil {
				current = parameter.ReadFailure
			}
		}
		matches := current == parameter.Value
		if parameter.Minimum {
			matches = current >= parameter.Value
		}
		if !matches {
			if parameter.Minimum {
				pending = append(pending, fmt.Sprintf("%s=%d (current: %d, minimum: %d)", parameter.Name, parameter.Value, current, parameter.Value))
			} else {
				pending = append(pending, fmt.Sprintf("%s=%d (current: %d)", parameter.Name, parameter.Value, current))
			}
		}
	}
	return pending
}

func (a SysctlApplier) ConfigContent() string {
	var b strings.Builder
	b.WriteString("# Managed by Vrooli -- do not edit manually\n")
	for _, parameter := range a.Parameters {
		fmt.Fprintf(&b, "%s = %d\n", parameter.Name, parameter.Value)
	}
	return b.String()
}

// InstallUserFile writes a user-owned managed file through the invoking-user
// command seam. It is shared by native watchdog schedulers.
func InstallUserFile(path, content string, opts EnsureOptions) error {
	if err := RunAsInvokingUser("mkdir", []string{"-p", filepath.Dir(path)}, opts); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	tmp, err := WriteTempFileFn(content)
	if err != nil {
		return fmt.Errorf("prepare file: %w", err)
	}
	defer os.Remove(tmp)
	if err := os.Chmod(tmp, 0o644); err != nil {
		return fmt.Errorf("make file readable: %w", err)
	}
	if err := RunAsInvokingUser("install", []string{"-m", "0644", tmp, path}, opts); err != nil {
		return fmt.Errorf("install file: %w", err)
	}
	return nil
}
