package validation

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	measures "github.com/vrooli/measures-go"
	"github.com/vrooli/measures-go/manifestscan"
	repocontract "github.com/vrooli/repo-contract-go"
)

// ManifestSource reads a target scenario's raw cli/manifest.json bytes.
// Production reads the filesystem (FilesystemManifestSource); tests inject bytes.
// A scenario with no manifest yields (nil, nil) — it simply declares no measures.
type ManifestSource interface {
	Manifest(scenario string) ([]byte, error)
}

// Prober behaviorally probes one declared measure against its owning scenario's
// live endpoint. Production POSTs to the measure's serve endpoint; tests inject a
// fake. A nil Prober (or probe=false) skips the behavioral pass.
type Prober interface {
	// Probe runs the measure end-to-end. ok=false means the declaration is
	// hollow (endpoint missing / non-conforming) -> ERROR. skipped=true means the
	// target was unreachable -> INFO, never a failure.
	Probe(ctx context.Context, scenario string, decl measures.MeasureDeclaration) (ok bool, detail string, skipped bool)
}

// ScenarioLister lists the scenario ids the fleet rollup spans. Production reads
// the scenarios/ directory; tests inject a fixed list.
type ScenarioLister interface {
	Scenarios() ([]string, error)
}

// Validator grades measure adoption for one scenario and rolls the fleet up. All
// I/O is behind the seams above so the heart (Classify) stays pure and testable.
type Validator struct {
	manifests ManifestSource
	domains   DomainSource
	schema    manifestscan.SchemaSource
	prober    Prober
	scenarios ScenarioLister
	substrate SubstrateDetector
}

// Option configures a Validator.
type Option func(*Validator)

// WithProber wires the behavioral-probe seam (otherwise probing is skipped).
func WithProber(p Prober) Option { return func(v *Validator) { v.prober = p } }

// WithScenarioLister wires the fleet scenario enumerator.
func WithScenarioLister(l ScenarioLister) Option { return func(v *Validator) { v.scenarios = l } }

// WithSubstrateDetector wires the anti-under-declaration substrate cross-check
// (otherwise it is skipped).
func WithSubstrateDetector(s SubstrateDetector) Option {
	return func(v *Validator) { v.substrate = s }
}

// NewValidator constructs a Validator over its required seams.
func NewValidator(m ManifestSource, d DomainSource, s manifestscan.SchemaSource, opts ...Option) *Validator {
	v := &Validator{manifests: m, domains: d, schema: s}
	for _, o := range opts {
		o(v)
	}
	return v
}

// NewFilesystemValidator wires the production seams rooted at repoRoot: manifests
// + domains + the committed proto descriptor + a live HTTP prober + the
// scenarios/ lister.
func NewFilesystemValidator(repoRoot string) *Validator {
	return NewValidator(
		FilesystemManifestSource{RepoRoot: repoRoot},
		ProtoDomainSource{RepoRoot: repoRoot},
		manifestscan.NewDescriptorSchemaReader(repoRoot),
		WithProber(NewHTTPProber()),
		WithScenarioLister(FilesystemScenarioLister{RepoRoot: repoRoot}),
		WithSubstrateDetector(FilesystemSubstrateDetector{RepoRoot: repoRoot}),
	)
}

// ValidateScenario grades one scenario's coverage and (when probe is set and a
// prober is wired) behaviorally probes its declared measures.
func (v *Validator) ValidateScenario(ctx context.Context, scenario string, probe bool) (Report, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return Report{}, errors.New("validation: scenario is required")
	}

	collector := metricsFrom(ctx)

	load := collector.Stage("load-manifest")
	raw, err := v.manifests.Manifest(scenario)
	if err != nil {
		load.End()
		return Report{}, err
	}
	mm := &manifestscan.ManifestMeasures{}
	if len(raw) > 0 {
		mm, err = manifestscan.Parse(raw)
		if err != nil {
			load.End()
			return Report{}, err
		}
	}
	load.End()

	derive := collector.Stage("derive-domains")
	derived, err := v.domains.StatefulDomains(scenario)
	if err != nil {
		derive.End()
		return Report{}, err
	}

	mode, err := v.domains.Mode(scenario)
	if err != nil {
		derive.End()
		return Report{}, err
	}

	var detected []DetectedEntity
	if v.substrate != nil {
		detected, err = v.substrate.DetectedEntities(scenario)
		if err != nil {
			derive.End()
			return Report{}, err
		}
	}
	derive.End()

	classify := collector.Stage("classify")
	in := Inputs{
		Scenario:  scenario,
		Mode:      mode,
		Domains:   derived,
		Measures:  v.harvest(mm),
		Omitted:   normalizeOmissions(mm.Omitted),
		Overrides: normalizeOverrides(mm.Domains),
		Detected:  detected,
	}
	rep := Classify(in)
	classify.Gauge("findings", float64(len(rep.Findings)))
	classify.End()

	if probe {
		v.runProbe(ctx, scenario, mm, &rep)
	}
	return rep, nil
}

// harvest assembles each command's measure block against its proto schema and
// grades its tier. Assembly errors are carried (not dropped) so Classify can
// raise them as hard findings.
func (v *Validator) harvest(mm *manifestscan.ManifestMeasures) []HarvestedMeasure {
	out := make([]HarvestedMeasure, 0, len(mm.Commands))
	for _, cm := range mm.Commands {
		hm := HarvestedMeasure{
			Name:          cm.MeasureName(),
			Domain:        normalizeDomain(cm.Domain),
			Intent:        cm.Measure.Intent,
			Effect:        string(cm.Governance.Effect),
			QuestionCount: len(cm.Measure.Questions),
		}
		decl, err := cm.Assemble(v.schema)
		if err != nil {
			hm.AssembleErr = err.Error()
			out = append(out, hm)
			continue
		}
		hm.Decl = decl
		hm.Tier = manifestscan.GradeTier(decl)
		hm.TierNote = tierNote(decl)
		out = append(out, hm)
	}
	return out
}

// runProbe behaviorally probes each successfully-assembled measure and folds the
// results into the report (probe fields on the matching MeasureSummary + a hard
// finding on a hollow declaration + a skipped note when the target is down).
func (v *Validator) runProbe(ctx context.Context, scenario string, mm *manifestscan.ManifestMeasures, rep *Report) {
	if v.prober == nil {
		return
	}
	// Index the assembled declarations by measure name for the probe.
	decls := map[string]measures.MeasureDeclaration{}
	for _, cm := range mm.Commands {
		if decl, err := cm.Assemble(v.schema); err == nil {
			decls[cm.MeasureName()] = decl
		}
	}

	anySkipped := false
	for di := range rep.Domains {
		dc := &rep.Domains[di]
		for mi := range dc.Measures {
			ms := &dc.Measures[mi]
			decl, ok := decls[ms.Name]
			if !ok {
				continue
			}
			okProbe, detail, skipped := v.prober.Probe(ctx, scenario, decl)
			if skipped {
				anySkipped = true
				ms.ProbeDetail = detail
				continue
			}
			ms.ProbePassed = okProbe
			ms.ProbeDetail = detail
			if !okProbe {
				rep.Findings = append(rep.Findings, Finding{
					RuleID:      "measures.hollow-declaration",
					Severity:    SeverityError,
					Title:       "Hollow measure declaration: " + ms.Name,
					Description: "The measure endpoint did not answer conforming to the declared result shape: " + detail,
					Remediation: "Implement the bound RPC so it returns the declared result.value_field, or remove the measure block until the endpoint exists.",
					FilePath:    manifestPath,
					Scanner:     "probe",
				})
			}
		}
	}
	if anySkipped {
		rep.SkippedScanners = append(rep.SkippedScanners, "probe (scenario not reachable)")
	}
	// Recompute the verdict in case the probe added ERROR findings.
	errs, _, _ := rep.Summary()
	rep.Passed = errs == 0
}

// ListFleetCoverage statically grades every requested scenario (or all
// discovered scenarios when the request is empty) and returns one rollup each.
func (v *Validator) ListFleetCoverage(ctx context.Context, scenarios []string) ([]FleetEntry, error) {
	if len(scenarios) == 0 {
		if v.scenarios == nil {
			return nil, errors.New("validation: no scenario lister wired")
		}
		all, err := v.scenarios.Scenarios()
		if err != nil {
			return nil, err
		}
		scenarios = all
	}
	sort.Strings(scenarios)
	out := make([]FleetEntry, 0, len(scenarios))
	for _, s := range scenarios {
		rep, err := v.ValidateScenario(ctx, s, false)
		if err != nil {
			// A scenario we cannot read is reported as an empty rollup, not a fatal
			// error — the fleet view degrades per-row like search-hub providers.
			out = append(out, FleetEntry{Scenario: s})
			continue
		}
		out = append(out, rollup(rep))
	}
	return out, nil
}

// FleetEntry is one scenario's static coverage rollup.
type FleetEntry struct {
	Scenario     string
	Passed       bool
	Expected     int
	Covered      int
	Waived       int
	Uncovered    int
	WorstTier    manifestscan.Tier
	MeasureCount int
}

func rollup(rep Report) FleetEntry {
	e := FleetEntry{Scenario: rep.Scenario, Passed: rep.Passed}
	rank := map[manifestscan.Tier]int{manifestscan.TierFull: 0, manifestscan.TierPartial: 1, manifestscan.TierFallback: 2}
	worst := manifestscan.Tier("")
	for _, d := range rep.Domains {
		switch d.Status {
		case StatusCovered:
			e.Covered++
			e.Expected++
			e.MeasureCount += d.MeasureCount
			if worst == "" || rank[d.Tier] > rank[worst] {
				worst = d.Tier
			}
		case StatusWaived:
			e.Waived++
			e.Expected++
		case StatusUncovered:
			e.Uncovered++
			e.Expected++
		}
	}
	e.WorstTier = worst
	return e
}

// tierNote describes why an assembled measure is not full tier (the bare params),
// for the partial-tier advisory. Empty when every param is canonical/constrained.
func tierNote(decl measures.MeasureDeclaration) string {
	var bare []string
	for _, name := range decl.ParamNames() {
		p := decl.Params[name]
		if !p.IsCanonical() && !p.IsConstrained() {
			bare = append(bare, name)
		}
	}
	if len(bare) == 0 {
		return ""
	}
	return "best-effort params: " + strings.Join(bare, ", ")
}

func normalizeOmissions(in []manifestscan.Omission) []manifestscan.Omission {
	out := make([]manifestscan.Omission, len(in))
	for i, o := range in {
		out[i] = manifestscan.Omission{Domain: normalizeDomain(o.Domain), Reason: o.Reason}
	}
	return out
}

func normalizeOverrides(in []manifestscan.DomainOverride) []manifestscan.DomainOverride {
	out := make([]manifestscan.DomainOverride, len(in))
	for i, o := range in {
		out[i] = manifestscan.DomainOverride{Domain: normalizeDomain(o.Domain), Stateful: o.Stateful, Reason: o.Reason}
	}
	return out
}

// -----------------------------------------------------------------------------
// Production filesystem seams
// -----------------------------------------------------------------------------

// FilesystemManifestSource reads the target scenario's CLI manifest under
// RepoRoot.
type FilesystemManifestSource struct {
	RepoRoot string
}

// Manifest returns the raw manifest bytes, or (nil, nil) when the scenario has no
// CLI manifest (it simply declares no measures).
func (f FilesystemManifestSource) Manifest(scenario string) ([]byte, error) {
	if strings.TrimSpace(scenario) == "control-plane" {
		// The control plane has no scenario CLI manifest. It is a valid,
		// intentionally empty measures target rather than an unresolved scenario.
		return nil, nil
	}
	path, err := repocontract.ScenarioCLIManifestPath(f.RepoRoot, scenario)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return b, nil
}

// FilesystemScenarioLister lists the directory names under scenarios/.
type FilesystemScenarioLister struct {
	RepoRoot string
}

// Scenarios returns every directory name under scenarios/ that carries a
// .vrooli/service.json (the scenario marker), sorted.
func (f FilesystemScenarioLister) Scenarios() ([]string, error) {
	root := filepath.Join(f.RepoRoot, "scenarios")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, e.Name(), ".vrooli", "service.json")); err != nil {
			continue
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out, nil
}
