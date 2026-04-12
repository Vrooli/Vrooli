package runtime

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
)

var (
	lookPathFn       = exec.LookPath
	combinedOutputFn = func(name string, args ...string) ([]byte, error) { return exec.Command(name, args...).CombinedOutput() }
	runCommandFn     = func(name string, args []string, opts EnsureOptions) error {
		cmd := exec.Command(name, args...)
		cmd.Stdout = writerOrDiscard(opts.Stdout)
		cmd.Stderr = writerOrDiscard(opts.Stderr)
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}
)

func baseStatus(requirement hostreq.ResolvedRequirement) ItemStatus {
	return ItemStatus{
		Name:         requirement.Name,
		Kind:         requirement.Kind,
		Required:     requirement.Required,
		Manual:       requirement.Manual,
		SupportClass: SupportSupported,
		Notes:        append([]string(nil), requirement.Notes...),
		Provenance:   append([]hostreq.Provenance(nil), requirement.Provenance...),
	}
}

func unsupportedRequirementStatus(requirement hostreq.ResolvedRequirement, note string) ItemStatus {
	status := baseStatus(requirement)
	status.SupportClass = SupportUnsupported
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
