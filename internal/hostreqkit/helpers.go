package hostreqkit

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	return RunCommandFn(command, args, opts)
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
