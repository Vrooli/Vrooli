// Package reconcile joins authored page regions to the file-granular UI inventory.
package reconcile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"react-component-library/internal/gates"
	"regexp"
	"sort"
	"strings"
)

type Provenance string

const (
	ProvenanceCustom            Provenance = "CUSTOM"
	ProvenanceAdoptedUnmodified Provenance = "ADOPTED_UNMODIFIED"
	ProvenanceAdoptedModified   Provenance = "ADOPTED_MODIFIED"
	ProvenanceUnknown           Provenance = "UNKNOWN"
)

type ObservedFile struct {
	CatalogID     string
	Path          string
	DisplayName   string
	Provenance    Provenance
	ComponentName string
	Library       string
	Version       string
}

// Scanner is deliberately file-granular. Resolver calls it exactly once.
type Scanner interface {
	ScanScenario(context.Context, string) ([]ObservedFile, error)
}

type Region struct {
	ID             string
	Required       bool
	Local          string
	Library        string
	LibraryVersion string
	TestID         string
	Selector       string
	Fill           Fill
}

type Fill struct {
	Asset       string
	Version     string
	Placeholder string
}

type Result struct {
	LibraryAsset    string     `json:"libraryAsset,omitempty"`
	LibraryVersion  string     `json:"libraryVersion,omitempty"`
	LocalComponent  string     `json:"localComponent,omitempty"`
	SelectedAsset   string     `json:"selectedAsset,omitempty"`
	SelectedVersion string     `json:"selectedVersion,omitempty"`
	ObservedAsset   string     `json:"observedAsset,omitempty"`
	ObservedVersion string     `json:"observedVersion,omitempty"`
	ReasonCode      string     `json:"reasonCode,omitempty"`
	EvidenceQuality string     `json:"evidenceQuality,omitempty"`
	Candidates      []string   `json:"candidates,omitempty"`
	Region          string     `json:"region"`
	Required        bool       `json:"required"`
	FilePath        string     `json:"filePath,omitempty"`
	JoinRule        string     `json:"joinRule,omitempty"`
	Proven          bool       `json:"proven"`
	Heuristic       bool       `json:"heuristic,omitempty"`
	Provenance      Provenance `json:"provenance,omitempty"`
	Reason          string     `json:"reason,omitempty"`
	Extra           bool       `json:"extra,omitempty"`
}

type Resolver struct {
	ScenariosRoot string
	ToolingRoot   string
	Facts         func(context.Context, string, ...string) (map[string]gates.SourceFacts, error)
	Scanner       Scanner
}

func (r Resolver) Resolve(ctx context.Context, scenario, page string) ([]Result, error) {
	if r.Scanner == nil {
		return nil, fmt.Errorf("reconcile %s/%s: scanner is required", scenario, page)
	}
	regions, err := loadRegions(r.ScenariosRoot, scenario, page)
	if err != nil {
		return nil, err
	}
	files, err := r.Scanner.ScanScenario(ctx, scenario)
	if err != nil {
		return nil, fmt.Errorf("scan scenario %s: %w", scenario, err)
	}
	if len(files) == 0 {
		out := make([]Result, 0, len(regions))
		for _, region := range regions {
			out = append(out, Result{Region: region.ID, Required: region.Required, LibraryAsset: region.Library, LibraryVersion: region.LibraryVersion, LocalComponent: region.Local, Reason: "scenario has no scannable files in a declared UI slot"})
		}
		return out, nil
	}

	reader := r.Facts
	if reader == nil {
		reader = gates.ReadSourceFacts
	}
	toolingRoot := r.ToolingRoot
	if toolingRoot == "" {
		toolingRoot = filepath.Dir(r.ScenariosRoot)
	}
	roots := make([]string, 0, len(files))
	for _, file := range files {
		if filepath.IsAbs(file.Path) || !filepath.IsLocal(filepath.FromSlash(file.Path)) {
			return nil, fmt.Errorf("inventory path escapes scenario: %q", file.Path)
		}
		roots = append(roots, filepath.Join(r.ScenariosRoot, scenario, filepath.FromSlash(file.Path)))
	}
	facts, factsErr := reader(ctx, toolingRoot, roots...)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]Result, 0, len(regions))
	for _, region := range regions {
		result := Result{Region: region.ID, Required: region.Required, LibraryAsset: region.Library, LibraryVersion: region.LibraryVersion, LocalComponent: region.Local, SelectedAsset: region.Fill.Asset, SelectedVersion: region.Fill.Version}
		if region.Fill.Placeholder != "" {
			result.Reason = "placement is a placeholder"
			result.ReasonCode = "placeholder"
			out = append(out, result)
			continue
		}
		var candidates []ObservedFile
		rule, proven := "", false
		if region.TestID != "" || region.Selector != "" {
			attribute, value, parseErr := regionAttribute(region)
			if parseErr != nil {
				result.ReasonCode = "unsupported_binding"
				result.Reason = parseErr.Error()
				out = append(out, result)
				continue
			}
			if factsErr != nil {
				result.ReasonCode = "source_unavailable"
				result.Reason = factsErr.Error()
				out = append(out, result)
				continue
			}
			rule, proven = "binding-testid", true
			if region.Selector != "" {
				rule = "binding-selector"
			}
			for _, file := range files {
				fact := facts[filepath.Clean(filepath.Join(r.ScenariosRoot, scenario, filepath.FromSlash(file.Path)))]
				if hasLiteralAttribute(fact, attribute, value) {
					candidates = append(candidates, file)
				}
			}
		}
		if len(candidates) == 0 && region.Fill.Asset != "" {
			rule, proven = "adoption-path", false
			for _, file := range files {
				if file.CatalogID == region.Fill.Asset && (file.Provenance == ProvenanceAdoptedUnmodified || file.Provenance == ProvenanceAdoptedModified) {
					candidates = append(candidates, file)
				}
			}
		}
		if len(candidates) == 0 {
			rule, proven = "component-slug", false
			slug := region.Local
			if slug == "" {
				slug = strings.TrimSuffix(region.ID, "-region")
			}
			for _, file := range files {
				if normalizeSlug(file.DisplayName) == normalizeSlug(slug) || normalizeSlug(strings.TrimSuffix(filepath.Base(file.Path), filepath.Ext(file.Path))) == normalizeSlug(slug) {
					candidates = append(candidates, file)
				}
			}
		}
		switch len(candidates) {
		case 0:
			result.ReasonCode = "binding_unresolved"
			result.Reason = "no source evidence resolved the region"
		case 1:
			result = resolved(region, candidates[0], rule, proven)
			result.Heuristic = !proven
			if proven {
				result.EvidenceQuality = "structured"
			} else {
				result.EvidenceQuality = "heuristic"
			}
		default:
			result.ReasonCode = "ambiguous_binding"
			result.Reason = "multiple files satisfy the region binding"
			result.EvidenceQuality = "ambiguous"
			for _, file := range candidates {
				result.Candidates = append(result.Candidates, file.Path)
			}
			sort.Strings(result.Candidates)
		}
		out = append(out, result)
	}
	// Scenario-wide inventory does not prove that unused files are reachable
	// from this page. Extra findings require page reachability evidence.

	return out, nil
}

func resolved(region Region, file ObservedFile, rule string, proven bool) Result {
	return Result{Region: region.ID, Required: region.Required, LibraryAsset: region.Library, LibraryVersion: region.LibraryVersion, LocalComponent: region.Local, FilePath: file.Path, JoinRule: rule, Proven: proven, Provenance: file.Provenance, SelectedAsset: region.Fill.Asset, SelectedVersion: region.Fill.Version, ObservedAsset: file.CatalogID, ObservedVersion: file.Version}
}

var (
	nonAlnum      = regexp.MustCompile(`[^a-z0-9]+`)
	camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)
)

func normalizeSlug(value string) string {
	value = camelBoundary.ReplaceAllString(value, "$1-$2")
	return strings.Trim(nonAlnum.ReplaceAllString(strings.ToLower(value), "-"), "-")
}

func hasLiteralTestID(fact gates.SourceFacts, testID string) bool {
	return testID != "" && hasLiteralAttribute(fact, "data-testid", testID)
}

// Only a single exact attribute selector can be proved from file-local JSX
// facts. Combinators, pseudo-classes, CSS escapes and substring matches need a
// richer oracle and must remain explicit unknowns.
var regionAttributeSelector = regexp.MustCompile(`^\[\s*([A-Za-z_][A-Za-z0-9_-]*)\s*=\s*(?:"([^"\\]*)"|'([^'\\]*)'|([A-Za-z0-9_-]+))\s*\]$`)

func regionAttribute(region Region) (string, string, error) {
	if region.Selector == "" {
		return "data-testid", region.TestID, nil
	}
	if region.TestID != "" {
		return "", "", fmt.Errorf("region %q declares both testid and selector bindings", region.ID)
	}
	match := regionAttributeSelector.FindStringSubmatch(strings.TrimSpace(region.Selector))
	if match == nil {
		return "", "", fmt.Errorf("selector %q is unsupported: expected one exact attribute selector such as [data-experience-surface=\"results\"]", region.Selector)
	}
	value := match[2] + match[3] + match[4]
	return match[1], value, nil
}

func hasLiteralAttribute(fact gates.SourceFacts, attribute, expected string) bool {
	for _, binding := range fact.DOMBindings {
		if binding.Attribute == attribute && binding.Value == expected && len(binding.Via) > 1 {
			return true
		}
	}
	for _, element := range fact.Elements {
		// A custom component's testId prop does not prove that it is forwarded to
		// a DOM hook. Only intrinsic elements establish this binding directly.
		if element.Tag == "" || element.Tag[0] < 'a' || element.Tag[0] > 'z' {
			continue
		}
		for _, value := range element.Attributes[attribute] {
			value = strings.TrimSpace(value)
			if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
				value = strings.TrimSpace(value[1 : len(value)-1])
			}
			if value == "\""+expected+"\"" || value == "'"+expected+"'" {
				return true
			}
		}
	}
	return false
}

type pageDocument struct {
	Regions []struct {
		ID        string `json:"id"`
		Required  *bool  `json:"required"`
		Component struct {
			Local   string `json:"local"`
			Library struct {
				Component string `json:"component"`
				Version   string `json:"version"`
			} `json:"library"`
		} `json:"component"`
	} `json:"regions"`
	Bindings struct {
		Regions map[string]struct {
			TestID   string `json:"testid"`
			Selector string `json:"selector"`
		} `json:"regions"`
	} `json:"bindings"`
	Sketch struct {
		Placements []struct {
			Region string `json:"region"`
			Fills  Fill   `json:"fills"`
		} `json:"placements"`
	} `json:"sketch"`
}

func loadRegions(root, scenario, page string) ([]Region, error) {
	path := filepath.Join(root, scenario, "experience", "pages", page+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read page document %s: %w", path, err)
	}
	var doc pageDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("decode page document %s: %w", path, err)
	}
	fills := map[string]Fill{}
	for _, placement := range doc.Sketch.Placements {
		fills[placement.Region] = placement.Fills
	}
	out := make([]Region, 0, len(doc.Regions))
	for _, region := range doc.Regions {
		required := true // Matches the experience schema; false must be explicit.
		if region.Required != nil {
			required = *region.Required
		}
		out = append(out, Region{ID: region.ID, Required: required, Local: region.Component.Local, Library: region.Component.Library.Component, LibraryVersion: region.Component.Library.Version, TestID: doc.Bindings.Regions[region.ID].TestID, Selector: doc.Bindings.Regions[region.ID].Selector, Fill: fills[region.ID]})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
