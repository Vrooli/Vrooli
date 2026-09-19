package golden

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// errNoRegenerator is surfaced when Service.Regenerate is called against
// a service constructed without a RegeneratorRunner. Public callers
// should never hit this in production; tests that exercise only the
// CRUD surface intentionally pass a nil runner.
var errNoRegenerator = errors.New("regenerator runner not configured")

// SubprocessRunner is the production RegeneratorRunner. It shells out to
// `vrooli scenario generate <template> --id <slug> --dest <path> --force`
// and parses the template version from the generator's output.
//
// The struct is exported (and its fields plumbable) so callers / tests
// can swap the binary or working directory without rewriting the
// service.
type SubprocessRunner struct {
	// Binary is the vrooli CLI path. Defaults to "vrooli" when empty.
	Binary string

	// WorkDir is the working directory the subprocess runs in. Defaults
	// to the current working directory of the API process when empty.
	WorkDir string
}

// NewSubprocessRunner returns a RegeneratorRunner that shells out to
// the real `vrooli` CLI. Tests use a hand-rolled fake instead.
func NewSubprocessRunner(binary, workDir string) *SubprocessRunner {
	return &SubprocessRunner{Binary: binary, WorkDir: workDir}
}

var _ RegeneratorRunner = (*SubprocessRunner)(nil)

// Regenerate invokes the vrooli CLI. The current implementation does
// not parse output for the produced template version — the caller's
// recorded TemplateVersion is propagated unchanged. Future work: parse
// the generator's emitted "template: <id> (<version>)" line.
func (r *SubprocessRunner) Regenerate(ctx context.Context, in RegenerateRunnerInput) (RegenerateRunnerOutput, error) {
	bin := strings.TrimSpace(r.Binary)
	if bin == "" {
		bin = "vrooli"
	}
	args := []string{
		"scenario", "generate", in.TemplateID,
		"--id", in.Slug,
		"--dest", in.Path,
		"--force",
		"--run-hooks",
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	if d := strings.TrimSpace(r.WorkDir); d != "" {
		cmd.Dir = d
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return RegenerateRunnerOutput{}, fmt.Errorf("%s %s: %w (output: %s)",
			bin, strings.Join(args, " "), err, truncateOutput(string(out), 1024))
	}
	version := parseTemplateVersion(string(out))
	if version == "" {
		// Fall back to the requested version. The generator owns version
		// selection in principle but the parser is best-effort today;
		// the service layer accepts an empty string here as "leave
		// pinned version unchanged".
		version = ""
	}
	_ = filepath.Clean(in.Path)
	return RegenerateRunnerOutput{TemplateVersion: version}, nil
}

// parseTemplateVersion best-effort extracts the produced template
// version from a line like `template: react-vite (1.0.1)`. Returns ""
// when no such line is present.
func parseTemplateVersion(out string) string {
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		// Match the "template: <id> (<version>)" line emitted by
		// vrooli scenario generate.
		if !strings.HasPrefix(trimmed, "template:") {
			continue
		}
		open := strings.LastIndex(trimmed, "(")
		close := strings.LastIndex(trimmed, ")")
		if open < 0 || close <= open {
			continue
		}
		return strings.TrimSpace(trimmed[open+1 : close])
	}
	return ""
}

func truncateOutput(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "…"
}
