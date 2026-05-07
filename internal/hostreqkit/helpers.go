package hostreqkit

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/shell"
)

var (
	LookPathFn = shell.LookPath

	ReadFileFn = os.ReadFile

	CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return shell.CombinedOutput(shell.Spec{Name: name, Args: args})
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
		Name:           requirement.Name,
		Kind:           requirement.Kind,
		Required:       requirement.Required,
		Manual:         requirement.Manual,
		SupportClass:   SupportSupported,
		ExecutionState: ExecutionPending,
		Reasons:        append([]string(nil), requirement.Reasons...),
		Notes:          append([]string(nil), requirement.Notes...),
		Provenance:     append([]hostreqspec.Provenance(nil), requirement.Provenance...),
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
	if command == "" || len(args) == 0 {
		return ""
	}
	output, err := CombinedOutputFn(command, args...)
	if err != nil {
		return ""
	}
	return FirstLine(strings.TrimSpace(string(output)))
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
