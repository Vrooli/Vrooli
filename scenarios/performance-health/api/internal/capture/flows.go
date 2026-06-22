package capture

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// flowSlugRE constrains a --workflow slug to a single safe path segment so a
// slug can never traverse out of the scenario's bas/flows directory.
var flowSlugRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// FileFlowResolver resolves a perf-flow slug to the bytes of
// <RepoRoot>/scenarios/<scenario>/bas/flows/<slug>.json. It is the production
// FlowResolver; tests drive a fake.
type FileFlowResolver struct {
	RepoRoot string
}

var _ FlowResolver = (*FileFlowResolver)(nil)

// Resolve reads the scenario's bas/flows/<slug>.json. A missing file, an
// invalid slug, or an empty repo root is a typed error (surfaced as a FAILED
// audit, never a silent skip).
func (r *FileFlowResolver) Resolve(scenario, slug string) ([]byte, error) {
	scenario = strings.TrimSpace(scenario)
	slug = strings.TrimSpace(slug)
	if r == nil || strings.TrimSpace(r.RepoRoot) == "" {
		return nil, fmt.Errorf("flow resolver has no repo root; cannot resolve workflow %q", slug)
	}
	if scenario == "" {
		return nil, fmt.Errorf("scenario is required to resolve workflow %q", slug)
	}
	if !flowSlugRE.MatchString(slug) {
		return nil, fmt.Errorf("workflow slug %q must match %s (a single bas/flows file stem)", slug, flowSlugRE)
	}
	path := filepath.Join(r.RepoRoot, "scenarios", scenario, "bas", "flows", slug+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read perf flow %q for scenario %q: %w", slug, scenario, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, fmt.Errorf("perf flow %q for scenario %q is empty", slug, scenario)
	}
	if err := ValidatePerfFlow(data); err != nil {
		return nil, fmt.Errorf("perf flow %q for scenario %q: %w", slug, scenario, err)
	}
	return data, nil
}

// ValidatePerfFlow enforces the perf-flow authoring convention at the point of
// use: a capture target must be assertion-free (an ASSERT belongs in
// bas/cases/** and the functional Playbooks suite, never in a perf trace that
// must not pass/fail on behavior). A perf capture only drives an interaction.
func ValidatePerfFlow(raw []byte) error {
	if bytes.Contains(raw, []byte("ACTION_TYPE_ASSERT")) {
		return errors.New("contains an ASSERT node — perf-capture flows must be assertion-free (move assertions to bas/cases/**)")
	}
	return nil
}
