// Package dimensions is the single source of truth for the controller's
// improvement-dimension vocabulary. Test Genie findings (by FindingSource) and
// skill target_dimensions declarations map into this vocabulary; the
// controller's selection policy matches the two.
//
// The vocabulary and its mapping tables live in dimensions.json (embedded
// below) so the data is machine-readable and shared with the prose in
// docs/concepts/DIMENSIONS.md. Edit the JSON, never these accessors.
package dimensions

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

//go:embed dimensions.json
var rawCatalog []byte

// Dimension is a canonical improvement axis (e.g. "standards", "tests").
type Dimension string

type dimensionEntry struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type catalog struct {
	Version    int              `json:"version"`
	Dimensions []dimensionEntry `json:"dimensions"`
	// SourceMap keys are architecture.v1.FindingSource proto enum NAMES.
	SourceMap map[string]string `json:"testgenie_source_map"`
}

var (
	loaded      catalog
	byID        map[Dimension]string // dimension -> description
	ordered     []Dimension          // declaration order
	bySourceDim map[string]Dimension // proto enum name -> dimension
)

func init() {
	if err := json.Unmarshal(rawCatalog, &loaded); err != nil {
		panic(fmt.Sprintf("dimensions: invalid embedded dimensions.json: %v", err))
	}

	byID = make(map[Dimension]string, len(loaded.Dimensions))
	ordered = make([]Dimension, 0, len(loaded.Dimensions))
	for _, d := range loaded.Dimensions {
		dim := Dimension(d.ID)
		if dim == "" {
			panic("dimensions: empty dimension id in dimensions.json")
		}
		if _, dup := byID[dim]; dup {
			panic(fmt.Sprintf("dimensions: duplicate dimension id %q", d.ID))
		}
		byID[dim] = d.Description
		ordered = append(ordered, dim)
	}

	bySourceDim = make(map[string]Dimension, len(loaded.SourceMap))
	for src, dimID := range loaded.SourceMap {
		dim := Dimension(dimID)
		if _, ok := byID[dim]; !ok {
			panic(fmt.Sprintf("dimensions: source %q maps to unknown dimension %q", src, dimID))
		}
		bySourceDim[src] = dim
	}
}

// All returns the canonical dimensions in declaration order.
func All() []Dimension {
	out := make([]Dimension, len(ordered))
	copy(out, ordered)
	return out
}

// IsValid reports whether d is a member of the canonical vocabulary.
func IsValid(d Dimension) bool {
	_, ok := byID[d]
	return ok
}

// Describe returns the human-readable description for a dimension, or "".
func Describe(d Dimension) string {
	return byID[d]
}

// ForSource resolves a test-genie FindingSource to its dimension. The bool is
// false for FINDING_SOURCE_UNSPECIFIED or any source absent from the SSOT map.
func ForSource(src architecturev1.FindingSource) (Dimension, bool) {
	name, ok := architecturev1.FindingSource_name[int32(src)]
	if !ok {
		return "", false
	}
	dim, ok := bySourceDim[name]
	return dim, ok
}

// MappedSources returns the proto enum names that carry a dimension mapping,
// sorted for stability. Used by the anti-drift guard.
func MappedSources() []string {
	out := make([]string, 0, len(bySourceDim))
	for name := range bySourceDim {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
