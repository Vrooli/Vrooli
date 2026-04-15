package runtime

import (
	"io"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/shell"
)

var (
	lookPathFn       = shell.LookPath
	readFileFn       = os.ReadFile
	combinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return shell.CombinedOutput(shell.Spec{Name: name, Args: args})
	}
	runCommandFn = func(name string, args []string, opts EnsureOptions) error {
		return shell.Run(shell.Spec{
			Name:   name,
			Args:   args,
			Stdout: writerOrDiscard(opts.Stdout),
			Stderr: writerOrDiscard(opts.Stderr),
			Stdin:  os.Stdin,
		})
	}
)

func baseStatus(requirement hostreq.ResolvedRequirement) ItemStatus {
	return ItemStatus{
		Name:           requirement.Name,
		Kind:           requirement.Kind,
		Required:       requirement.Required,
		Manual:         requirement.Manual,
		SupportClass:   SupportSupported,
		ExecutionState: ExecutionPending,
		Reasons:        append([]string(nil), requirement.Reasons...),
		Notes:          append([]string(nil), requirement.Notes...),
		Provenance:     append([]hostreq.Provenance(nil), requirement.Provenance...),
	}
}

func unsupportedRequirementStatus(requirement hostreq.ResolvedRequirement, note string) ItemStatus {
	status := baseStatus(requirement)
	status.SupportClass = SupportUnsupported
	status.ExecutionState = ExecutionUnsupported
	if strings.TrimSpace(note) != "" {
		status.Notes = append(status.Notes, note)
	}
	return status
}

func resolveCommand(candidates []string) (string, bool) {
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if _, err := lookPathFn(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}

func commandAvailable(name string) bool {
	_, err := lookPathFn(name)
	return err == nil
}

func readVersion(command string, args []string) string {
	if command == "" || len(args) == 0 {
		return ""
	}
	output, err := combinedOutputFn(command, args...)
	if err != nil {
		return ""
	}
	return firstLine(strings.TrimSpace(string(output)))
}

func firstLine(value string) string {
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

func runInstallCommand(command string, args []string, opts EnsureOptions) error {
	return runCommandFn(command, args, opts)
}

func runPrivilegedCommand(sudoMode, command string, args []string, opts EnsureOptions) error {
	command, args, err := withSudo(sudoMode, command, args)
	if err != nil {
		return err
	}
	return runCommandFn(command, args, opts)
}
