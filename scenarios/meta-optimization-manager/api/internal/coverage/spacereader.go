package coverage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"

	"github.com/vrooli/api-core/spacedoc"
)

// SpaceReader reads a projection's denominator (the curated intended space) from
// its owner. It is the read side of the cross-scenario contract; the numerator
// is NEVER read here. Production reads the owner's `space --projection <p>
// --json` verb; tests fake it.
type SpaceReader interface {
	Read(ctx context.Context, p Projection) (*spacedoc.SpaceDefinition, error)
}

// execSpaceReader reads the denominator by invoking the owner's `space` verb.
// If the owner CLI is not reachable (not installed / errors), it falls back to
// parsing the owner's space doc directly with the SAME parser the verb uses
// (api-core/spacedoc) — guaranteed-consistent output — so the scoreboard works
// in a dev checkout whether or not owner CLIs are installed. Either way the
// result is schema-shaped spacedoc.SpaceDefinition.
type execSpaceReader struct {
	run CommandRunner
}

// NewSpaceReader returns the production SpaceReader.
func NewSpaceReader() SpaceReader { return &execSpaceReader{run: execRunner} }

// NewSpaceReaderWithRunner returns a SpaceReader using the given runner (tests).
func NewSpaceReaderWithRunner(run CommandRunner) SpaceReader {
	return &execSpaceReader{run: run}
}

func (r *execSpaceReader) Read(ctx context.Context, p Projection) (*spacedoc.SpaceDefinition, error) {
	owner := OwnerFor(p)
	if owner == "" {
		return nil, fmt.Errorf("coverage: no owner for projection %q", p)
	}
	// Preferred path: the owner's space verb (true decoupling).
	if r.run != nil {
		if out, err := r.run(ctx, owner, "space", "--projection", string(p), "--json"); err == nil {
			def, perr := decodeSpaceDefinition(out, p)
			if perr == nil {
				augmentGeneratedValidateCapabilities(def)
				return def, nil
			}
		}
	}
	// Fallback: parse the owner's doc directly with the shared parser.
	return readSpaceDocFromFile(p, owner)
}

// decodeSpaceDefinition unmarshals the verb's JSON and validates the projection.
func decodeSpaceDefinition(out []byte, p Projection) (*spacedoc.SpaceDefinition, error) {
	var def spacedoc.SpaceDefinition
	if err := json.Unmarshal(out, &def); err != nil {
		return nil, fmt.Errorf("coverage: decode space JSON for %q: %w", p, err)
	}
	if def.Projection != p {
		return nil, fmt.Errorf("coverage: space JSON projection %q != requested %q", def.Projection, p)
	}
	if len(def.Cells) == 0 {
		return nil, fmt.Errorf("coverage: space JSON for %q has no cells", p)
	}
	return &def, nil
}

// readSpaceDocFromFile resolves and parses scenarios/<owner>/docs/spaces/<p>-space.md.
func readSpaceDocFromFile(p Projection, owner string) (*spacedoc.SpaceDefinition, error) {
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return nil, fmt.Errorf("coverage: locate repo root: %w", err)
	}
	scenarioRoot, err := repocontract.ResolveScenarioPath(root, owner)
	if err != nil {
		return nil, fmt.Errorf("coverage: resolve %s: %w", owner, err)
	}
	docPath := filepath.Join(scenarioRoot, "docs", "spaces", string(p)+"-space.md")
	md, err := os.ReadFile(docPath)
	if err != nil {
		return nil, fmt.Errorf("coverage: read %s: %w", docPath, err)
	}
	def, err := spacedoc.Parse(p, md)
	if err != nil {
		return nil, fmt.Errorf("coverage: parse %s: %w", docPath, err)
	}
	rel, rerr := filepath.Rel(root, docPath)
	if rerr == nil {
		def.Source = rel
	}
	augmentGeneratedValidateCapabilities(def)
	return def, nil
}

// augmentGeneratedValidateCapabilities adds the descriptor-derived tier to
// Validate's authored concern denominator. The descriptor is the source of
// truth for capability declarations; the space document remains the stable
// authored tier. Keeping this expansion in the denominator reader means a
// provider can add a capability without a meta-optimization-manager source
// edit, while the moving denominator is still visible in the emitted space.
func augmentGeneratedValidateCapabilities(def *spacedoc.SpaceDefinition) {
	if def == nil || def.Projection != spacedoc.ProjectionValidate {
		return
	}
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return
	}
	type capability struct {
		ID string `json:"id"`
	}
	type maturity struct {
		Capabilities []capability `json:"capabilities"`
	}
	type descriptor struct {
		Scenario string   `json:"scenario"`
		Maturity maturity `json:"maturity"`
	}
	type generated struct {
		Cell spacedoc.Cell
		Key  string
	}
	var generatedCells []generated
	paths, _ := filepath.Glob(filepath.Join(root, "scenarios", "*", ".vrooli", "test-genie.json"))
	for _, path := range paths {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var d descriptor
		if json.Unmarshal(raw, &d) != nil {
			continue
		}
		scenario := strings.TrimSpace(d.Scenario)
		if scenario == "" {
			scenario = filepath.Base(filepath.Dir(filepath.Dir(path)))
		}
		for _, cap := range d.Maturity.Capabilities {
			id := strings.TrimSpace(cap.ID)
			if id == "" {
				continue
			}
			key := strings.ToLower(scenario + "/" + id)
			generatedCells = append(generatedCells, generated{
				Key: key,
				Cell: spacedoc.Cell{
					ID:       "cap/" + scenario + "/" + id,
					Group:    "Generated capability tier",
					Question: id,
					Owner:    scenario,
					Status:   spacedoc.StatusMissing,
					Notes:    []string{"generated from " + filepath.ToSlash(filepath.Join("scenarios", scenario, ".vrooli", "test-genie.json")) + ":maturity.capabilities"},
				},
			})
		}
	}
	sort.Slice(generatedCells, func(i, j int) bool { return generatedCells[i].Key < generatedCells[j].Key })
	seen := make(map[string]struct{}, len(def.Cells)+len(generatedCells))
	for _, c := range def.Cells {
		seen[strings.ToLower(c.ID)] = struct{}{}
	}
	for _, item := range generatedCells {
		if _, exists := seen[strings.ToLower(item.Cell.ID)]; exists {
			continue
		}
		seen[strings.ToLower(item.Cell.ID)] = struct{}{}
		def.Cells = append(def.Cells, item.Cell)
	}
}
