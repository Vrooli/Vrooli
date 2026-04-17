package hostreqkit

import (
	"fmt"
	"io"
	"os"
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

	RunCommandFn = func(name string, args []string, opts EnsureOptions) error {
		return shell.Run(shell.Spec{
			Name:   name,
			Args:   args,
			Stdout: writerOrDiscard(opts.Stdout),
			Stderr: writerOrDiscard(opts.Stderr),
			Stdin:  os.Stdin,
		})
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
	if len(lines) == 0 {
		return ""
	}
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
	file, err := os.CreateTemp("", "vrooli-managed-*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", path, err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if _, err := file.WriteString(content); err != nil {
		file.Close()
		return fmt.Errorf("write temp file for %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temp file for %s: %w", path, err)
	}
	return RunPrivilegedCommand(sudoMode, "install", []string{"-m", "0644", tempPath, path}, opts)
}
