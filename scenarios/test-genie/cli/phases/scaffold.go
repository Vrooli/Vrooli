package phases

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/maturity-go/assessment"

	"test-genie/internal/providerconformance"
)

// runScaffold implements `test-genie phases scaffold <phase>`, emitting the
// Phase Capability Contract structure — the remediation-doc skeleton and a
// maturity-spec stub — for a new or non-conformant phase. It produces STRUCTURE
// ONLY: every prose slot is a TODO placeholder, never fabricated content, so an
// owner fills the real North Star, gates, and finding meanings. The output
// passes provider-conformance's structural checks by construction.
func runScaffold(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("phases scaffold", flag.ContinueOnError)
	fs.SetOutput(w)
	provider := fs.String("provider", "", "Provider scenario id (defaults to the phase name)")
	docsOut := fs.String("docs-out", "", "Write the remediation-doc skeleton to this path instead of stdout")
	maturityOut := fs.String("maturity-out", "", "Write the maturity-spec stub to this path instead of stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 || strings.TrimSpace(rest[0]) == "" {
		return errors.New("exactly one <phase> is required")
	}
	phase := strings.TrimSpace(rest[0])
	prov := strings.TrimSpace(*provider)
	if prov == "" {
		prov = phase
	}

	doc := scaffoldDoc(phase)
	maturity, err := scaffoldMaturity(prov, phase)
	if err != nil {
		return err
	}

	wroteFile := false
	if path := strings.TrimSpace(*docsOut); path != "" {
		if err := writeScaffoldFile(path, doc); err != nil {
			return err
		}
		fmt.Fprintf(w, "wrote remediation-doc skeleton: %s\n", path)
		wroteFile = true
	}
	if path := strings.TrimSpace(*maturityOut); path != "" {
		if err := writeScaffoldFile(path, maturity); err != nil {
			return err
		}
		fmt.Fprintf(w, "wrote maturity-spec stub: %s\n", path)
		wroteFile = true
	}
	if wroteFile {
		fmt.Fprintln(w, "\nFill every TODO with real content, then wire maturity into .vrooli/test-genie.json and point docs.path at the doc.")
		return nil
	}

	fmt.Fprintf(w, "# Remediation-doc skeleton (write to the provider's docs, referenced by docs.path)\n\n%s\n", doc)
	fmt.Fprintf(w, "# Maturity-spec stub (merge into the provider's .vrooli/test-genie.json \"maturity\" block)\n\n%s\n", maturity)
	return nil
}

// scaffoldDoc builds the remediation doc with the required H2 skeleton headings
// (the SSOT is providerconformance.RequiredDocHeadings) plus TODO prompts.
func scaffoldDoc(phase string) string {
	var b strings.Builder
	title := strings.ToUpper(phase[:1]) + phase[1:]
	fmt.Fprintf(&b, "# %s — Phase Capability Contract\n\n", title)
	b.WriteString("> Scaffolded structure. Replace every TODO with real content; do not ship placeholder prose.\n")
	prompts := map[string]string{
		"North Star":                "TODO: state what maximum maturity looks like for this capability — the aspiration an agent aims at, not the absence of findings.",
		"The rungs and their gates": "TODO: describe the L0–L4 ladder — each rung's entry/exit criteria and the single next unlock.",
		"What each finding means":   "TODO: list each finding code, the rung it caps the capability at, its severity, and whether it fails the phase.",
		"The canonical fix":         "TODO: for each class of finding, give the specific remediation an agent should apply.",
		"How to verify":             "TODO: give the exact command(s) that confirm a rung was climbed.",
	}
	for _, heading := range providerconformance.RequiredDocHeadings {
		fmt.Fprintf(&b, "\n## %s\n\n%s\n", heading, prompts[heading])
	}
	return b.String()
}

// scaffoldMaturity builds a maturity-spec stub with an L0–L4 ladder, one
// capability, and placeholder gates. Placeholders are non-empty so the stub
// passes provider-conformance's structural checks while clearly prompting for
// real content.
func scaffoldMaturity(provider, phase string) (string, error) {
	levels := []assessment.Level{
		scaffoldLevel("L0", "Blocked", "Unavailable", false),
		scaffoldLevel("L1", "Foundation", "Foundation", false),
		scaffoldLevel("L2", "Ready", "Ready", false),
		scaffoldLevel("L3", "Hardened", "Ready", false),
		scaffoldLevel("L4", "Complete", "Complete", true),
	}
	spec := assessment.Spec{
		Provider: provider,
		Phase:    phase,
		Version:  "1.0.0",
		Levels:   levels,
		Capabilities: []assessment.CapabilitySpec{
			{
				ID:     phase + "_capability",
				Label:  "TODO: capability label",
				Levels: scaffoldLevels(),
			},
		},
		Findings: map[string]assessment.FindingMapping{},
	}
	raw, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func scaffoldLevels() []assessment.Level {
	return []assessment.Level{
		scaffoldLevel("L0", "Blocked", "Unavailable", false),
		scaffoldLevel("L1", "Foundation", "Foundation", false),
		scaffoldLevel("L2", "Ready", "Ready", false),
		scaffoldLevel("L3", "Hardened", "Ready", false),
		scaffoldLevel("L4", "Complete", "Complete", true),
	}
}

func scaffoldLevel(id, name, statusLabel string, top bool) assessment.Level {
	level := assessment.Level{
		ID:            id,
		Name:          name,
		StatusLabel:   statusLabel,
		EntryCriteria: []string{"TODO: what must be true to enter " + id},
		ExitCriteria:  []string{"TODO: what must be true to leave " + id},
	}
	if top {
		level.CapabilitySummary = "TODO: North Star — what maximum maturity looks like."
	} else {
		level.NextUnlock = "TODO: the single highest-unlock next move toward the next rung."
	}
	return level
}

func writeScaffoldFile(path, content string) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
