package hostreqkit

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/shell"
)

var (
	LookPathFn = shell.LookPath

	ReadFileFn = os.ReadFile

	// CombinedOutputContextFn is the context-aware command seam. Callers that
	// already hold a lifecycle context should use it so cancellation propagates
	// to the child process.
	CombinedOutputContextFn = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return shell.CombinedOutput(shell.Spec{Context: ctx, Name: name, Args: args})
	}

	// CombinedOutputFn is retained for host helpers that predate context-aware
	// seams. Its bounded root context prevents a wedged host service from
	// stalling setup indefinitely; new probes should pass their caller context
	// through CombinedOutputContextFn.
	CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return CombinedOutputContextFn(ctx, name, args...)
	}

	// CombinedOutputInputFn runs a command with the supplied string on stdin
	// and returns its combined stdout/stderr. Used for tools like
	// `debconf-set-selections` that read their inputs from stdin and where
	// piping via shell would break test stubs.
	CombinedOutputInputFn = func(name, input string, args ...string) ([]byte, error) {
		return shell.CombinedOutput(shell.Spec{
			Name:  name,
			Args:  args,
			Stdin: strings.NewReader(input),
		})
	}

	RunCommandFn = func(name string, args []string, opts EnsureOptions) error {
		tail := shell.NewStderrTail(10)
		stderr := io.MultiWriter(writerOrDiscard(opts.Stderr), tail)
		err := shell.Run(shell.Spec{
			Name:   name,
			Args:   args,
			Stdout: writerOrDiscard(opts.Stdout),
			Stderr: stderr,
			Stdin:  os.Stdin,
		})
		if err != nil {
			if captured := strings.TrimSpace(tail.String()); captured != "" {
				return fmt.Errorf("%w: %s", err, captured)
			}
		}
		return err
	}

	// RunCommandInputFn is the stdin-bearing counterpart used for Vrooli-owned
	// operator flows that must drop back to the invoking user without putting a
	// secret in argv or a temporary file.
	RunCommandInputFn = func(name string, args []string, input string, opts EnsureOptions) error {
		tail := shell.NewStderrTail(10)
		stderr := io.MultiWriter(writerOrDiscard(opts.Stderr), tail)
		err := shell.Run(shell.Spec{
			Name:   name,
			Args:   args,
			Stdout: writerOrDiscard(opts.Stdout),
			Stderr: stderr,
			Stdin:  strings.NewReader(input),
		})
		if err != nil {
			if captured := strings.TrimSpace(tail.String()); captured != "" {
				return fmt.Errorf("%w: %s", err, captured)
			}
		}
		return err
	}

	// WriteTempFileFn writes content to a temporary file and returns
	// the path. The caller is responsible for removing the file.
	WriteTempFileFn = func(content string) (string, error) {
		file, err := os.CreateTemp("", "vrooli-managed-*")
		if err != nil {
			return "", err
		}
		path := file.Name()
		if _, err := file.WriteString(content); err != nil {
			file.Close()
			os.Remove(path)
			return "", err
		}
		if err := file.Close(); err != nil {
			os.Remove(path)
			return "", err
		}
		return path, nil
	}
)

func BaseStatus(requirement hostreqspec.ResolvedRequirement) ItemStatus {
	return ItemStatus{
		Name:             requirement.Name,
		Kind:             requirement.Kind,
		Required:         requirement.Required,
		MinVersion:       requirement.MinVersion,
		OperatorChoice:   requirement.OperatorChoice,
		Config:           requirement.Config,
		ConfigNonDefault: requirement.ConfigNonDefault,
		Manual:           requirement.Manual,
		SupportClass:     SupportSupported,
		ExecutionState:   ExecutionPending,
		Reasons:          append([]string(nil), requirement.Reasons...),
		Notes:            append([]string(nil), requirement.Notes...),
		Provenance:       append([]hostreqspec.Provenance(nil), requirement.Provenance...),
	}
}

func UnsupportedRequirementStatus(requirement hostreqspec.ResolvedRequirement, note string) ItemStatus {
	status := BaseStatus(requirement)
	status.SupportClass = SupportUnsupported
	status.ExecutionState = ExecutionUnsupported
	if strings.TrimSpace(note) != "" {
		status.Notes = append(status.Notes, note)
	}
	return status
}

func NotApplicableRequirementStatus(requirement hostreqspec.ResolvedRequirement, note string) ItemStatus {
	status := BaseStatus(requirement)
	status.SupportClass = SupportNotApplicable
	status.ExecutionState = ExecutionNotApplicable
	if strings.TrimSpace(note) != "" {
		status.Notes = append(status.Notes, note)
	}
	return status
}

func InvalidConfigStatus(requirement hostreqspec.ResolvedRequirement, note string) ItemStatus {
	status := BaseStatus(requirement)
	status.SupportClass = SupportUnsupported
	status.ExecutionState = ExecutionUnsupported
	status.BlockingReason = BlockingInvalidParameter
	if strings.TrimSpace(note) != "" {
		status.Notes = append(status.Notes, note)
	}
	return status
}

func ResolveCommand(candidates []string) (string, bool) {
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if _, err := LookPathFn(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}

// ResolveCommandForInvokingUser is like ResolveCommand but additionally
// probes the invoking user's well-known per-user install directories
// (~/.local/bin, ~/go/bin, ~/bin). Use this in handlers that install into
// per-user dirs (go install, npm install --prefix=$HOME, etc.) — under
// `sudo vrooli setup` the running process inherits root's PATH which
// excludes those dirs, so a plain ResolveCommand false-negatives even
// when the install actually succeeded.
//
// Returns the bare candidate name (not the full path) when found; the
// caller invokes it via PATH-based exec like with ResolveCommand. The
// assumption is that if the binary is in one of the user's standard
// dirs, it will be on PATH for the user's normal shell — we just couldn't
// see it from a sudo'd context.
func ResolveCommandForInvokingUser(candidates []string) (string, bool) {
	if cmd, ok := ResolveCommand(candidates); ok {
		return cmd, true
	}
	home, err := InvokingUserHomeDir()
	if err != nil || home == "" {
		return "", false
	}
	// Standard per-user install dirs, in priority order. ~/.local/bin
	// is the canonical XDG location and where Vrooli's own symlinks live;
	// ~/go/bin is Go's default install target; ~/bin is the older
	// convention some operators still use.
	userDirs := []string{".local/bin", "go/bin", "bin"}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		for _, dir := range userDirs {
			full := filepath.Join(home, dir, candidate)
			if info, statErr := os.Stat(full); statErr == nil && !info.IsDir() {
				return candidate, true
			}
		}
	}
	return "", false
}

// AugmentUserToolPath returns currentPath with existing per-user tool
// directories prepended in deterministic priority order. Lifecycle children
// must receive the same tool search path that setup uses: setup may install
// Go/npm-generated plugins into ~/.local/bin or ~/go/bin while the Vrooli
// launcher itself lives under the repo-contract runtime home.
//
// The function is deliberately pure with respect to the process environment;
// callers provide the effective home and PATH so an overridden lifecycle
// environment cannot accidentally inherit the parent process's home.
func AugmentUserToolPath(home, currentPath, localAppData string) string {
	home = strings.TrimSpace(home)
	if home == "" {
		return currentPath
	}

	current := filepath.SplitList(currentPath)
	seen := make(map[string]struct{}, len(current))
	for _, dir := range current {
		seen[filepath.Clean(dir)] = struct{}{}
	}
	candidates := []string{
		"/opt/homebrew/bin",
		"/usr/local/go/bin",
		"/usr/local/bin",
		filepath.Join(home, "go", "bin"),
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, "bin"),
	}
	if localAppData = strings.TrimSpace(localAppData); localAppData != "" {
		candidates = append(candidates, filepath.Join(localAppData, "Microsoft", "WinGet", "Links"))
	}
	candidates = append(candidates, filepath.Join(home, "AppData", "Local", "Microsoft", "WinGet", "Links"))
	if runtimeBin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin); err == nil {
		candidates = append(candidates, runtimeBin)
	}

	prepend := make([]string, 0, len(candidates))
	for _, dir := range candidates {
		clean := filepath.Clean(dir)
		if _, ok := seen[clean]; ok {
			continue
		}
		info, err := os.Stat(clean)
		if err != nil || !info.IsDir() {
			continue
		}
		prepend = append(prepend, clean)
		seen[clean] = struct{}{}
	}
	if len(prepend) == 0 {
		return currentPath
	}
	return strings.Join(append(prepend, current...), string(os.PathListSeparator))
}

func CommandAvailable(name string) bool {
	_, err := LookPathFn(name)
	return err == nil
}

func DetectFirstAvailable(candidates []string) string {
	for _, candidate := range candidates {
		if CommandAvailable(candidate) {
			return candidate
		}
	}
	return ""
}

func ReadVersion(command string, args []string) string {
	version, _ := ReadVersionErr(command, args)
	return version
}

// ReadVersionErr reads a tool's version line and reports why the probe failed
// when it did. Callers that pin an exact release need this distinction: a probe
// that ran and printed a different release is a genuine version mismatch, while
// a probe that could not execute at all says nothing about the installed
// version. Collapsing the second case into an empty string makes a broken host
// environment look like a wrong or missing tool, and sends the operator to an
// install command that cannot fix it. The returned error carries the command's
// combined output, which is where tools explain themselves (for example the Go
// toolchain's "cannot determine current directory" when the working directory
// is unreadable).
func ReadVersionErr(command string, args []string) (string, error) {
	if command == "" || len(args) == 0 {
		return "", nil
	}
	output, err := CombinedOutputFn(command, args...)
	if err != nil {
		detail := FirstLine(strings.TrimSpace(string(output)))
		if detail == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, detail)
	}
	return FirstLine(strings.TrimSpace(string(output))), nil
}

// VersionMatches reports whether a version-command line contains expected as a
// complete version token. It accepts the conventional leading "v" and "go"
// prefixes while deliberately rejecting partial matches (for example 1.25.1
// does not satisfy 1.25.12). Manifests use this for exact release pins.
func VersionMatches(observed, expected string) bool {
	expected = normalizeVersionToken(expected)
	if expected == "" {
		return true
	}
	for _, token := range strings.FieldsFunc(observed, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && r != '.' && r != '+' && r != '-'
	}) {
		if normalizeVersionToken(token) == expected {
			return true
		}
	}
	return false
}

// CompareVersions compares the first numeric dotted version in each value.
// It returns -1, 0, or 1 and treats missing trailing segments as zero.
func CompareVersions(left, right string) int {
	l := numericVersion(left)
	r := numericVersion(right)
	limit := len(l)
	if len(r) > limit {
		limit = len(r)
	}
	for index := 0; index < limit; index++ {
		lv, rv := 0, 0
		if index < len(l) {
			lv = l[index]
		}
		if index < len(r) {
			rv = r[index]
		}
		if lv < rv {
			return -1
		}
		if lv > rv {
			return 1
		}
	}
	return 0
}

func numericVersion(value string) []int {
	start := -1
	for index, char := range value {
		if char >= '0' && char <= '9' {
			start = index
			break
		}
	}
	if start < 0 {
		return nil
	}
	var numbers []int
	for _, token := range strings.Split(value[start:], ".") {
		digits := strings.Builder{}
		for _, char := range token {
			if char < '0' || char > '9' {
				break
			}
			digits.WriteRune(char)
		}
		if digits.Len() == 0 {
			break
		}
		number, _ := strconv.Atoi(digits.String())
		numbers = append(numbers, number)
	}
	return numbers
}

func normalizeVersionToken(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "go")
	value = strings.TrimPrefix(value, "v")
	return value
}

func FirstLine(value string) string {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	return strings.TrimSpace(lines[0])
}

func writerOrDiscard(w io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return io.Discard
}

func RunInstallCommand(command string, args []string, opts EnsureOptions) error {
	if request, ok := inferredPackageInstall(command, args); ok && RecordPackageInstallFn != nil {
		return RunInstallCommandWithProvenance(command, args, opts, request)
	}
	return RunCommandFn(command, args, opts)
}

func inferredPackageInstall(command string, args []string) (InstallProvenanceRequest, bool) {
	manager := strings.ToLower(strings.TrimSpace(command))
	if manager == "sudo" && len(args) > 0 {
		manager = strings.ToLower(strings.TrimSpace(args[0]))
		args = args[1:]
	}
	if manager == "env" {
		for i, arg := range args {
			candidate := strings.ToLower(strings.TrimSpace(arg))
			if candidate == "apt-get" || candidate == "apt" || candidate == "dnf" || candidate == "yum" || candidate == "brew" || candidate == "winget" || candidate == "pacman" || candidate == "apk" {
				manager = candidate
				args = args[i+1:]
				break
			}
		}
	}
	if manager == "homebrew" {
		manager = "brew"
	}
	if manager != "apt" && manager != "apt-get" && manager != "dnf" && manager != "yum" && manager != "brew" && manager != "winget" && manager != "pacman" && manager != "apk" && manager != "choco" && manager != "scoop" {
		return InstallProvenanceRequest{}, false
	}
	actionIndex := -1
	for i, arg := range args {
		if strings.EqualFold(strings.TrimSpace(arg), "install") || strings.EqualFold(strings.TrimSpace(arg), "add") || strings.EqualFold(strings.TrimSpace(arg), "-S") {
			actionIndex = i
			break
		}
	}
	if actionIndex < 0 {
		return InstallProvenanceRequest{}, false
	}
	for _, arg := range args[actionIndex+1:] {
		arg = strings.TrimSpace(arg)
		if arg == "" || strings.HasPrefix(arg, "-") {
			continue
		}
		return InstallProvenanceRequest{PackageManager: manager, PackageName: arg}, true
	}
	return InstallProvenanceRequest{}, false
}

// InstallProvenanceRequest identifies the package install being executed.
// Version probing is best-effort and conservative: a failed probe records
// unknown rather than authorizing ownership.
type InstallProvenanceRequest struct {
	PackageManager string
	PackageName    string
	VersionCommand string
	VersionArgs    []string
	Shared         bool
}

type PackageObservation string

const (
	PackagePresent PackageObservation = "present"
	PackageAbsent  PackageObservation = "absent"
	PackageUnknown PackageObservation = "unknown"
)

type PackageAction string

const (
	PackageInstalled PackageAction = "installed"
	PackageAdopted   PackageAction = "adopted"
)

type PackageInstallRecord struct {
	Home           string
	PackageManager string
	PackageName    string
	ObservedBefore PackageObservation
	Action         PackageAction
	VersionBefore  string
	VersionAfter   string
	OwningNode     string
	Shared         bool
}

// RecordPackageInstallFn is registered by the control-plane ledger package.
// Keeping the callback at this seam avoids a package cycle with config while
// ensuring every package-manager command uses the same recorder.
var RecordPackageInstallFn func(PackageInstallRecord) error

// ProbePackageStateFn is injectable because package-manager state belongs to
// the host seam, not to unit tests for individual tool handlers.
var ProbePackageStateFn = probePackageState

func RunInstallCommandWithProvenance(command string, args []string, opts EnsureOptions, request InstallProvenanceRequest) error {
	if strings.TrimSpace(request.PackageManager) == "" || strings.TrimSpace(request.PackageName) == "" {
		return RunCommandFn(command, args, opts)
	}
	observed, versionBefore, probeErr := ProbePackageStateFn(request.PackageManager, request.PackageName)
	if probeErr != nil {
		// A probe error is never evidence of absence. Preserve the install
		// result, but make the resulting ledger entry unattributable so an
		// uninstall cannot turn an operational failure into deletion authority.
		observed = PackageUnknown
		versionBefore = ""
	}
	if err := RunCommandFn(command, args, opts); err != nil {
		return err
	}
	versionAfter := ""
	if strings.TrimSpace(request.VersionCommand) != "" {
		versionAfter, _ = ReadVersionErr(request.VersionCommand, request.VersionArgs)
	}
	action := PackageInstalled
	if observed == PackagePresent {
		action = PackageAdopted
	}
	home, err := InvokingUserHomeDir()
	if err != nil {
		return fmt.Errorf("install succeeded but provenance home is unavailable: %w", err)
	}
	node, _ := os.Hostname()
	if RecordPackageInstallFn == nil {
		return fmt.Errorf("install succeeded but package provenance recorder is unavailable")
	}
	if err := RecordPackageInstallFn(PackageInstallRecord{
		Home: home, PackageManager: request.PackageManager, PackageName: request.PackageName,
		ObservedBefore: observed, Action: action, VersionBefore: versionBefore,
		VersionAfter: versionAfter, OwningNode: node, Shared: request.Shared,
	}); err != nil {
		return fmt.Errorf("install succeeded but package provenance could not be recorded: %w", err)
	}
	return nil
}

func RunPackageInstallCommand(command string, args []string, opts EnsureOptions, manager, packageName, versionCommand string, versionArgs []string) error {
	return RunInstallCommandWithProvenance(command, args, opts, InstallProvenanceRequest{
		PackageManager: manager,
		PackageName:    packageName,
		VersionCommand: versionCommand,
		VersionArgs:    versionArgs,
	})
}

func probePackageState(manager, packageName string) (PackageObservation, string, error) {
	manager = strings.ToLower(strings.TrimSpace(manager))
	packageName = strings.TrimSpace(packageName)
	if packageName == "" {
		return PackageUnknown, "", fmt.Errorf("package name is empty")
	}
	var command string
	var args []string
	switch manager {
	case "brew", "homebrew":
		command, args = "brew", []string{"list", "--versions", packageName}
	case "apt", "apt-get":
		command, args = "dpkg-query", []string{"-W", "-f=${Version}", packageName}
	case "dnf", "yum":
		command, args = manager, []string{"list", "installed", packageName}
	case "winget":
		command, args = "winget", []string{"list", "--id", packageName, "--exact"}
	default:
		return PackageUnknown, "", fmt.Errorf("unsupported package manager %q", manager)
	}
	if _, err := LookPathFn(command); err != nil {
		return PackageUnknown, "", err
	}
	output, err := CombinedOutputFn(command, args...)
	if err != nil {
		// Package managers use a non-zero exit for a normal no-match result,
		// but a blank or unrelated failure is not evidence of absence.
		lower := strings.ToLower(string(output))
		if strings.Contains(lower, "no such keg") || strings.Contains(lower, "not installed") || strings.Contains(lower, "no path found matching") || strings.Contains(lower, "no packages found") || strings.Contains(lower, "no matching packages") || strings.Contains(lower, "no installed package") {
			return PackageAbsent, "", nil
		}
		return PackageUnknown, "", err
	}
	version := FirstLine(string(output))
	if strings.TrimSpace(version) == "" {
		return PackageAbsent, "", nil
	}
	return PackagePresent, version, nil
}

func RunPrivilegedCommand(sudoMode, command string, args []string, opts EnsureOptions) error {
	command, args, err := WithSudo(sudoMode, command, args)
	if err != nil {
		return err
	}
	return RunCommandFn(command, args, opts)
}

// RunPrivilegedCommandWithOutput is the context-free command seam used by
// callers that already own their executor and need to preserve stdout. The
// elevation decision stays in hostreqkit, so scenario code never constructs
// a sudo invocation itself.
func RunPrivilegedCommandWithOutput(sudoMode, command string, args []string, run func(string, ...string) ([]byte, error)) ([]byte, error) {
	command, args, err := WithSudo(sudoMode, command, args)
	if err != nil {
		return nil, err
	}
	if command == "sudo" {
		args = append([]string{"-n"}, args...)
	}
	return run(command, args...)
}

// RunPrivilegedCommandWithStdin is the stdin-fed sibling of
// RunPrivilegedCommand. Use it for tools like debconf-set-selections that read
// their inputs from stdin: WithSudo wraps the command, then the input is piped
// in. Returns the typed sudo error (ErrSudoSkipped / ErrSudoUnavailable) when
// the operator's --sudo-mode forbids privilege escalation.
func RunPrivilegedCommandWithStdin(sudoMode, command, input string, args []string) ([]byte, error) {
	command, args, err := WithSudo(sudoMode, command, args)
	if err != nil {
		return nil, err
	}
	return CombinedOutputInputFn(command, input, args...)
}

func FileContentMatches(path, want string) bool {
	content, err := ReadFileFn(path)
	if err != nil {
		return false
	}
	return string(content) == want
}

func EnsureManagedDir(path, sudoMode string, opts EnsureOptions) error {
	if opts.DryRun {
		return nil
	}
	return RunPrivilegedCommand(sudoMode, "mkdir", []string{"-p", path}, opts)
}

func InstallManagedContent(path, content, sudoMode string, opts EnsureOptions) error {
	if opts.DryRun {
		return nil
	}
	tempPath, err := WriteTempFileFn(content)
	if err != nil {
		return fmt.Errorf("prepare managed content for %s: %w", path, err)
	}
	defer os.Remove(tempPath)
	return RunPrivilegedCommand(sudoMode, "install", []string{"-m", "0644", tempPath, path}, opts)
}

// InstallManagedExecutable is the executable-bit sibling of
// InstallManagedContent. The only difference is the install mode (0755 vs
// 0644) — they're separate functions, not a single mode-parameterised one,
// so the call site is explicit about whether the file being written is data
// or an executable. Reading
// `InstallManagedExecutable("/usr/local/bin/vrooli", shim, ...)` makes the
// intent unmistakable; a stray mode int does not.
//
// Use this for shell shims, hook scripts, or any other file the operator (or
// other tools) will exec. Use InstallManagedContent for config / data files.
func InstallManagedExecutable(path, content, sudoMode string, opts EnsureOptions) error {
	if opts.DryRun {
		return nil
	}
	tempPath, err := WriteTempFileFn(content)
	if err != nil {
		return fmt.Errorf("prepare managed executable for %s: %w", path, err)
	}
	defer os.Remove(tempPath)
	return RunPrivilegedCommand(sudoMode, "install", []string{"-m", "0755", tempPath, path}, opts)
}

// RunVerificationCheck runs the post-action verification defined in a manifest
// and returns whether it passed plus a human-readable detail string on failure.
func RunVerificationCheck(vc *VerificationCheck) (bool, string) {
	if vc == nil {
		return true, ""
	}
	for _, path := range vc.Files {
		if _, err := ReadFileFn(path); err != nil {
			return false, fmt.Sprintf("verification: required file missing: %s", path)
		}
	}
	if vc.Command != "" {
		output, err := CombinedOutputFn(vc.Command, vc.Args...)
		if err != nil {
			detail := FirstLine(strings.TrimSpace(string(output)))
			if detail == "" {
				detail = err.Error()
			}
			return false, fmt.Sprintf("verification failed: %s: %s", vc.Command, detail)
		}
	}
	return true, ""
}
