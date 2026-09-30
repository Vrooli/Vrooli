// Package manifestscan is the join between a scenario's raw cli/manifest.json and
// the measures contract library (github.com/vrooli/measures-go). It extracts each
// command's curated `measure` block (which cliapp.ParseManifest deliberately
// drops), assembles it against the proto-derived param schema, and grades the
// adoption tier. It is the single canonical scanner: cli-health's discovery
// indexer + static validator and the measures-health validator + central index
// all consume it (it was promoted out of two byte-compatible scenario-local
// copies so a tier-grading change lands once, not twice).
//
// Param types/enum membership/numeric bounds are NEVER read from the manifest —
// they come from the bound proto request message via a SchemaSource. The
// manifest only contributes curated prose and the two things proto cannot
// express: a presentation default and a dynamic-enum values_source.
package manifestscan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	measures "github.com/vrooli/measures-go"
	"github.com/vrooli/vrooli/packages/proto/descriptorimage"
)

// ManifestMeasures is the parsed measure surface of one manifest: the commands
// that declare a measure block, plus the scenario-level measures{} metadata.
type ManifestMeasures struct {
	Commands []CommandMeasure
	Omitted  []Omission
	Domains  []DomainOverride
}

// CommandMeasure is one command that declares a measure block, joined with the
// binding + governance the measure layer needs to assemble a declaration.
type CommandMeasure struct {
	Group      string
	Command    string
	Domain     string // measure.domain override, else the group name
	Binding    measures.Binding
	Governance measures.Governance
	Measure    measures.ManifestMeasure
}

// MeasureName is the conventional "<domain>.<command>" measure identifier
// (e.g. "backlog.completed").
func (c CommandMeasure) MeasureName() string {
	return c.Domain + "." + c.Command
}

// Omission is a measures.omitted[] waiver entry: a stateful domain intentionally
// left without a measure, with a reason.
type Omission struct {
	Domain string `json:"domain"`
	Reason string `json:"reason"`
}

// DomainOverride is a measures.domains[] statefulness-classification override —
// the escape hatch for misclassified domains (consumed by measures-health).
type DomainOverride struct {
	Domain   string `json:"domain"`
	Stateful bool   `json:"stateful"`
	Reason   string `json:"reason,omitempty"`
}

// rawManifest is the slice of cli/manifest.json manifestscan needs. Fields
// validated elsewhere (flags, positionals, descriptions) are omitted.
type rawManifest struct {
	Name     string           `json:"name"`
	Groups   []rawGroup       `json:"groups"`
	Measures *rawMeasuresMeta `json:"measures"`
}

type rawGroup struct {
	Name     string       `json:"name"`
	Commands []rawCommand `json:"commands"`
}

type rawCommand struct {
	Name       string              `json:"name"`
	Binding    measures.Binding    `json:"binding"`
	Governance measures.Governance `json:"governance"`
	Measure    *rawMeasureBlock    `json:"measure"`
}

// rawMeasureBlock mirrors the schema's Measure object. It carries `domain`
// (which measures.ManifestMeasure does not model) alongside the curated prose;
// Parse projects the rest onto measures.ManifestMeasure.
type rawMeasureBlock struct {
	Domain    string                            `json:"domain"`
	Intent    string                            `json:"intent"`
	Questions []string                          `json:"questions"`
	Params    map[string]measures.ManifestParam `json:"params"`
	Result    measures.Result                   `json:"result"`
}

type rawMeasuresMeta struct {
	Omitted []Omission       `json:"omitted"`
	Domains []DomainOverride `json:"domains"`
}

// Parse extracts the measure surface from raw cli/manifest.json bytes. It is
// tolerant of manifests with no measures (returns an empty *ManifestMeasures)
// and never inspects fields outside the measure surface.
func Parse(raw []byte) (*ManifestMeasures, error) {
	var m rawManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("manifestscan: parse manifest: %w", err)
	}
	out := &ManifestMeasures{}
	for _, g := range m.Groups {
		group := strings.TrimSpace(g.Name)
		for _, c := range g.Commands {
			if c.Measure == nil {
				continue
			}
			domain := strings.TrimSpace(c.Measure.Domain)
			if domain == "" {
				domain = group
			}
			out.Commands = append(out.Commands, CommandMeasure{
				Group:      group,
				Command:    strings.TrimSpace(c.Name),
				Domain:     domain,
				Binding:    c.Binding,
				Governance: c.Governance,
				Measure: measures.ManifestMeasure{
					Intent:    c.Measure.Intent,
					Questions: c.Measure.Questions,
					Params:    c.Measure.Params,
					Result:    c.Measure.Result,
				},
			})
		}
	}
	if m.Measures != nil {
		out.Omitted = m.Measures.Omitted
		out.Domains = m.Measures.Domains
	}
	return out, nil
}

// SchemaSource resolves the proto-derived param schema for a binding. The
// measures-go descriptor reader satisfies it; tests stub it. A nil SchemaSource
// means "no proto schema available" — Assemble then validates the measure on
// manifest-only data (no params, so drift cannot be detected).
type SchemaSource interface {
	RequestParams(service, method string) ([]measures.ParamSchema, error)
}

// Assemble joins a command's measure block with its proto-derived param schema
// into a validated MeasureDeclaration, surfacing the same drift/validation
// errors measures.Assemble enforces (a manifest param naming a field absent from
// the proto request → error; a malformed result/effect → error).
func (c CommandMeasure) Assemble(src SchemaSource) (measures.MeasureDeclaration, error) {
	var protoParams []measures.ParamSchema
	if src != nil {
		ps, err := src.RequestParams(c.Binding.Service, c.Binding.Method)
		if err != nil {
			return measures.MeasureDeclaration{}, fmt.Errorf("resolve proto params for %s.%s: %w", c.Binding.Service, c.Binding.Method, err)
		}
		protoParams = ps
	}
	return measures.Assemble(c.MeasureName(), c.Domain, c.Binding, c.Measure, c.Governance, protoParams)
}

// Tier grades a measure's parameter-extraction maturity. measures-health surfaces
// it; cli-health records it.
type Tier string

const (
	// TierFull: every param resolves deterministically (canonical time_window)
	// or against a bounded value space (enum/numeric bounds) — no best-effort
	// guesses. A measure with no params is full (nothing to extract).
	TierFull Tier = "full"
	// TierPartial: some params are canonical/constrained, at least one is bare.
	TierPartial Tier = "partial"
	// TierFallback: no param is canonical/constrained (all best-effort).
	TierFallback Tier = "fallback"
)

// GradeTier grades the assembled declaration per the tier definitions above.
func GradeTier(decl measures.MeasureDeclaration) Tier {
	if len(decl.Params) == 0 {
		return TierFull
	}
	var strong, total int
	for _, name := range decl.ParamNames() {
		p := decl.Params[name]
		total++
		if p.IsCanonical() || p.IsConstrained() {
			strong++
		}
	}
	switch {
	case strong == total:
		return TierFull
	case strong == 0:
		return TierFallback
	default:
		return TierPartial
	}
}

// KnownManifestParamType reports whether a manifest param `type` annotation is a
// legal canonical-convention upgrade. The proto descriptor is authoritative for
// real field kinds; the manifest may only annotate the two canonical
// conventions (time_window, enum) or leave the type empty.
func KnownManifestParamType(t string) bool {
	switch strings.TrimSpace(t) {
	case "", measures.ParamTypeTimeWindow, measures.ParamTypeEnum:
		return true
	default:
		return false
	}
}

// ManifestParamTypes returns the manifest-annotated `type` for each param that
// carries one, keyed by param name. Used by the static validator to flag unknown
// annotations independently of proto resolution.
func (c CommandMeasure) ManifestParamTypes() map[string]string {
	out := make(map[string]string, len(c.Measure.Params))
	for name, p := range c.Measure.Params {
		if t := strings.TrimSpace(p.Type); t != "" {
			out[name] = t
		}
	}
	return out
}

// DefaultDescriptorPath returns the committed proto descriptor image path under
// repoRoot. MEASURES_DESCRIPTOR_PATH overrides it (useful when running from an
// unusual CWD or against a relocated image).
func DefaultDescriptorPath(repoRoot string) string {
	if p := strings.TrimSpace(os.Getenv("MEASURES_DESCRIPTOR_PATH")); p != "" {
		return p
	}
	return filepath.Join(repoRoot, "packages", "proto", "gen", "descriptor", "image.binpb")
}

// DescriptorSchemaReader is a SchemaSource backed by the committed descriptor
// image. It loads the image lazily on first use and caches the reader (and any
// load error) so a missing descriptor degrades to a per-call error rather than
// crashing construction or indexing.
type DescriptorSchemaReader struct {
	path    string
	source  *descriptorimage.Source
	initErr error
	mu      sync.Mutex
	loaded  uint64
	reader  *measures.SchemaReader
}

// NewDescriptorSchemaReader returns a reader resolving the descriptor image
// relative to repoRoot (honoring MEASURES_DESCRIPTOR_PATH).
func NewDescriptorSchemaReader(repoRoot string) *DescriptorSchemaReader {
	path := DefaultDescriptorPath(repoRoot)
	source, err := descriptorimage.New(descriptorimage.Config{DescriptorPath: path})
	return &DescriptorSchemaReader{path: path, source: source, initErr: err}
}

func (d *DescriptorSchemaReader) load() (*measures.SchemaReader, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.initErr != nil {
		return nil, d.initErr
	}
	snapshot, err := d.source.Snapshot()
	if err != nil {
		return nil, err
	}
	if d.loaded == snapshot.Generation && d.reader != nil {
		return d.reader, nil
	}
	reader, err := measures.NewSchemaReaderFromBytes(snapshot.DescriptorBytes())
	if err != nil {
		return nil, err
	}
	d.loaded = snapshot.Generation
	d.reader = reader
	return reader, nil
}

// RequestParams resolves the request-message param schema for (service, method),
// loading the descriptor image on first call.
func (d *DescriptorSchemaReader) RequestParams(service, method string) ([]measures.ParamSchema, error) {
	r, err := d.load()
	if err != nil {
		return nil, fmt.Errorf("manifestscan: descriptor image unavailable (%s): %w", d.path, err)
	}
	return r.RequestParams(service, method)
}
