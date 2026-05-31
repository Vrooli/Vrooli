package findings

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// AuditRequest scopes one test-genie audit. When Phases is non-empty the audit
// is restricted to those phases (targeted re-audit); otherwise Preset selects
// the phase set. Preset defaults to "comprehensive" when both are empty.
type AuditRequest struct {
	Scenario string
	Preset   string
	Phases   []string
}

// AuditRunner runs a test-genie audit and returns the parsed result.
//
// seam: AuditRunner is the controller's boundary to test-genie. Production wires
// TestGenieRunner (shells out to the test-genie CLI); tests wire FakeRunner.
type AuditRunner interface {
	Audit(ctx context.Context, req AuditRequest) (*Audit, error)
}

// TestGenieRunner invokes the test-genie CLI's `execute --json` and parses the
// result. test-genie exits non-zero on a FAIL verdict but still emits the JSON
// report, so a non-zero exit is not treated as fatal when stdout parses.
type TestGenieRunner struct {
	// ProjectRoot is the repo root; the command runs with this as its working
	// directory so test-genie resolves the scenario by name.
	ProjectRoot string
	// Binary is the test-genie executable; defaults to "test-genie" (or
	// $ECOSYSTEM_MANAGER_TESTGENIE_BIN) resolved on PATH.
	Binary string
	// Timeout bounds a single audit. Zero uses defaultAuditTimeout.
	Timeout time.Duration
}

const defaultAuditTimeout = 20 * time.Minute

var _ AuditRunner = (*TestGenieRunner)(nil)

func (r *TestGenieRunner) binary() string {
	if r.Binary != "" {
		return r.Binary
	}
	if env := strings.TrimSpace(os.Getenv("ECOSYSTEM_MANAGER_TESTGENIE_BIN")); env != "" {
		return env
	}
	return "test-genie"
}

// Args builds the argv for a request. Exposed for testability.
func (r *TestGenieRunner) Args(req AuditRequest) []string {
	args := []string{"execute", req.Scenario, "--json"}
	switch {
	case len(req.Phases) > 0:
		args = append(args, "--phases", strings.Join(req.Phases, ","))
	case req.Preset != "":
		args = append(args, "--preset", req.Preset)
	default:
		args = append(args, "--preset", "comprehensive")
	}
	return args
}

func (r *TestGenieRunner) Audit(ctx context.Context, req AuditRequest) (*Audit, error) {
	if strings.TrimSpace(req.Scenario) == "" {
		return nil, fmt.Errorf("audit: scenario is required")
	}

	timeout := r.Timeout
	if timeout <= 0 {
		timeout = defaultAuditTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.binary(), r.Args(req)...)
	if r.ProjectRoot != "" {
		cmd.Dir = r.ProjectRoot
	}
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	// Parse whatever JSON test-genie produced first: a FAIL verdict is a
	// non-zero exit but a fully valid report.
	if out := strings.TrimSpace(stdout.String()); out != "" {
		if audit, perr := ParseAudit([]byte(out)); perr == nil {
			return audit, nil
		}
	}

	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("audit %q timed out after %s", req.Scenario, timeout)
	}
	if runErr != nil {
		return nil, fmt.Errorf("test-genie execute failed: %w (stderr: %s)", runErr, truncate(stderr.String(), 500))
	}
	return nil, fmt.Errorf("test-genie produced no parseable JSON for %q", req.Scenario)
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// FakeRunner is the test double for AuditRunner. Audits map scenario name to a
// canned Audit; an optional Err short-circuits.
type FakeRunner struct {
	Audits map[string]*Audit
	Err    error
	Calls  []AuditRequest
}

var _ AuditRunner = (*FakeRunner)(nil)

func (f *FakeRunner) Audit(_ context.Context, req AuditRequest) (*Audit, error) {
	f.Calls = append(f.Calls, req)
	if f.Err != nil {
		return nil, f.Err
	}
	if a, ok := f.Audits[req.Scenario]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("FakeRunner: no audit configured for %q", req.Scenario)
}

// LoadAuditFile parses an audit JSON file from disk (used to seed FakeRunner
// from a fixture).
func LoadAuditFile(path string) (*Audit, error) {
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	return ParseAudit(raw)
}
