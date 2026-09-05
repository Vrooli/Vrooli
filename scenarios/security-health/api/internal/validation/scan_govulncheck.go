package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"time"
)

// govulncheckScanner runs golang.org/x/vuln/cmd/govulncheck, which reports only
// vulnerabilities that are actually reachable from the scenario's code — a
// reached Go vuln is high-confidence, so it normalizes to ERROR. The binary is
// gated behind the install-approval rule; when absent the Service records it as
// a skipped (degraded) scanner, never a failure.
type govulncheckScanner struct {
	cmd Commander
}

func newGovulncheckScanner(cmd Commander) Scanner { return &govulncheckScanner{cmd: cmd} }

func (g *govulncheckScanner) Name() string             { return "govulncheck" }
func (g *govulncheckScanner) Binary() string           { return "govulncheck" }
func (g *govulncheckScanner) Applies(s Substrate) bool { return s.Go }
func (g *govulncheckScanner) EvidencePlan(ctx context.Context, scenarioDir string, sub Substrate, now time.Time) (ScannerEvidencePlan, error) {
	return scannerEvidencePlan(ctx, g.cmd, g.Name(), g.Binary(), scenarioDir, sub, now)
}

// govulncheck -json emits a stream of newline-delimited JSON message objects,
// each carrying exactly one of these fields. We collect OSV metadata by id and
// the set of reached (called) vuln ids, then join them.
type govulncheckMessage struct {
	OSV     *govulncheckOSV     `json:"osv,omitempty"`
	Finding *govulncheckFinding `json:"finding,omitempty"`
}

type govulncheckOSV struct {
	ID       string `json:"id"`
	Summary  string `json:"summary"`
	Details  string `json:"details"`
	Affected []struct {
		Package struct {
			Name string `json:"name"`
		} `json:"package"`
		Ranges []struct {
			Events []struct {
				Fixed string `json:"fixed"`
			} `json:"events"`
		} `json:"ranges"`
	} `json:"affected"`
}

type govulncheckFinding struct {
	OSV   string `json:"osv"`
	Trace []struct {
		Module   string `json:"module"`
		Package  string `json:"package"`
		Function string `json:"function"`
		Position *struct {
			Filename string `json:"filename"`
			Line     int    `json:"line"`
		} `json:"position"`
	} `json:"trace"`
}

func (g *govulncheckScanner) Scan(ctx context.Context, scenarioDir string, sub Substrate) ([]Finding, error) {
	dirs := sub.GoModDirs
	if len(dirs) == 0 {
		dirs = []string{"."}
	}
	var findings []Finding
	var lastErr error
	parsedAny := false
	for _, rel := range dirs {
		modDir := filepath.Join(scenarioDir, rel)
		args := append([]string{"-json"}, goPackagePatterns(sub, rel)...)
		stdout, stderr, _, _, err := runCommandWithEnv(ctx, g.cmd, modDir, scannerEnvironment(), "govulncheck", args...)
		if err != nil {
			lastErr = fmt.Errorf("govulncheck failed to run in %s: %w", rel, err)
			continue
		}
		if len(stdout) == 0 {
			lastErr = fmt.Errorf("govulncheck produced no output in %s: %s", rel, truncate(stderr, 200))
			continue
		}
		osvByID := map[string]govulncheckOSV{}
		reached := map[string]govulncheckFinding{}
		dec := json.NewDecoder(bytes.NewReader(stdout))
		for {
			var msg govulncheckMessage
			if err := dec.Decode(&msg); err != nil {
				if err == io.EOF {
					break
				}
				lastErr = fmt.Errorf("parse govulncheck stream in %s: %w", rel, err)
				break
			}
			switch {
			case msg.OSV != nil:
				osvByID[msg.OSV.ID] = *msg.OSV
			case msg.Finding != nil && len(msg.Finding.Trace) > 0 && msg.Finding.Trace[0].Function != "":
				// A trace with a function frame means the vuln is reachable.
				reached[msg.Finding.OSV] = *msg.Finding
			}
		}
		parsedAny = true
		ids := make([]string, 0, len(reached))
		for id := range reached {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			f := reached[id]
			osv := osvByID[id]
			top := f.Trace[0]
			loc := relPath(scenarioDir, rel)
			if top.Position != nil {
				loc = fmt.Sprintf("%s:%d", relPath(scenarioDir, top.Position.Filename), top.Position.Line)
			}
			// A reachable third-party-module vuln is scenario-actionable (bump
			// the dep) → ERROR. A reachable *stdlib* vuln is only fixable by
			// upgrading the Go toolchain itself, which is a host concern, not a
			// scenario concern — gating every Go scenario's R1 on it would make
			// R1 permanently unreachable. So stdlib vulns are WARNING (advisory).
			severity := SeverityError
			remediation := fmt.Sprintf("Upgrade the affected module to a patched version (%s) or remove the reachable call path. See https://pkg.go.dev/vuln/%s", govulncheckFixHint(osv), id)
			if top.Module == "stdlib" {
				severity = SeverityWarning
				remediation = fmt.Sprintf("This is a Go standard-library vulnerability — upgrade the Go toolchain to a release that includes the fix (%s). Not fixable from scenario code. See https://pkg.go.dev/vuln/%s", govulncheckFixHint(osv), id)
			}
			findings = append(findings, Finding{
				RuleID:       "govulncheck." + id,
				Severity:     severity,
				Title:        fmt.Sprintf("%s: %s", id, nonEmpty(osv.Summary, "reachable Go vulnerability")),
				Description:  nonEmpty(osv.Summary, osv.Details),
				Remediation:  remediation,
				FilePath:     loc,
				Scanner:      g.Name(),
				Class:        FindingVulnerability,
				Owner:        "scenario-dependency-analyzer",
				FixClass:     FixAssisted,
				PolicyImpact: "go-reachability",
			})
			if top.Module == "stdlib" {
				findings[len(findings)-1].Owner = "toolchain-owner"
				findings[len(findings)-1].FixClass = FixManual
				findings[len(findings)-1].PolicyImpact = "toolchain-upgrade"
			}
		}
	}
	if !parsedAny && lastErr != nil {
		return nil, lastErr
	}
	return findings, nil
}

// govulncheckFixHint extracts the first "fixed" version across the OSV's
// affected ranges, or a generic phrase when none is present.
func govulncheckFixHint(osv govulncheckOSV) string {
	for _, a := range osv.Affected {
		for _, r := range a.Ranges {
			for _, e := range r.Events {
				if e.Fixed != "" {
					return ">= " + e.Fixed
				}
			}
		}
	}
	return "the latest patched release"
}
