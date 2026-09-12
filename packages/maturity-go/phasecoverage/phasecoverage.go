// Package phasecoverage projects Test Genie provider descriptors into
// phase-to-dimension lookup tables. It intentionally contains no embedded phase
// catalog; provider descriptors are the source of truth.
package phasecoverage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/maturity-go/dimensions"
)

const DescriptorRelPath = ".vrooli/test-genie.json"

type descriptor struct {
	Phase                string   `json:"phase"`
	OrderHint            int      `json:"orderHint"`
	Dimensions           []string `json:"dimensions"`
	FreshnessRequirement string   `json:"freshnessRequirement"`
}

type Coverage struct {
	byPhase     map[string][]dimensions.Dimension
	byDimension map[dimensions.Dimension][]string
	freshness   []string
}

func Load(repoRoot string) (Coverage, error) {
	if strings.TrimSpace(repoRoot) == "" {
		repoRoot = "."
	}
	matches, err := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", filepath.FromSlash(DescriptorRelPath)))
	if err != nil {
		return Coverage{}, err
	}
	c := Coverage{
		byPhase:     map[string][]dimensions.Dimension{},
		byDimension: map[dimensions.Dimension][]string{},
	}
	descriptors := make([]descriptor, 0, len(matches))
	for _, path := range matches {
		raw, err := os.ReadFile(path)
		if err != nil {
			return Coverage{}, err
		}
		var d descriptor
		if err := json.Unmarshal(raw, &d); err != nil {
			return Coverage{}, err
		}
		descriptors = append(descriptors, d)
	}
	sort.SliceStable(descriptors, func(i, j int) bool {
		if descriptors[i].OrderHint != descriptors[j].OrderHint {
			return descriptors[i].OrderHint < descriptors[j].OrderHint
		}
		return descriptors[i].Phase < descriptors[j].Phase
	})
	for _, d := range descriptors {
		phase := strings.TrimSpace(d.Phase)
		if phase == "" {
			continue
		}
		seen := map[dimensions.Dimension]struct{}{}
		for _, rawDim := range d.Dimensions {
			dim := dimensions.Dimension(strings.TrimSpace(rawDim))
			if !dimensions.IsValid(dim) {
				continue
			}
			if _, dup := seen[dim]; dup {
				continue
			}
			seen[dim] = struct{}{}
			c.byPhase[phase] = append(c.byPhase[phase], dim)
			c.byDimension[dim] = append(c.byDimension[dim], phase)
		}
		switch strings.TrimSpace(d.FreshnessRequirement) {
		case "always", "when_applicable":
			c.freshness = append(c.freshness, phase)
		}
	}
	for dim := range c.byDimension {
		sort.Strings(c.byDimension[dim])
	}
	return c, nil
}

func (c Coverage) FreshnessRequiredPhases() []string {
	return append([]string(nil), c.freshness...)
}

func (c Coverage) DimensionsForPhase(phase string) []dimensions.Dimension {
	out := append([]dimensions.Dimension(nil), c.byPhase[strings.TrimSpace(phase)]...)
	return out
}

func (c Coverage) FirstDimensionForPhase(phase string) (dimensions.Dimension, bool) {
	dims := c.DimensionsForPhase(phase)
	if len(dims) == 0 {
		return "", false
	}
	return dims[0], true
}

func (c Coverage) PhasesForDimensions(dims ...dimensions.Dimension) []string {
	seen := map[string]struct{}{}
	for _, dim := range dims {
		for _, phase := range c.byDimension[dim] {
			seen[phase] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for phase := range seen {
		out = append(out, phase)
	}
	sort.Strings(out)
	return out
}
