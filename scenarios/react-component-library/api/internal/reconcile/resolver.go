// Package reconcile joins authored page regions to the file-granular UI inventory.
package reconcile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	ID       string
	Required bool
	Local    string
	Library  string
	TestID   string
	Fill     Fill
}

type Fill struct {
	Asset       string
	Version     string
	Placeholder string
}

type Result struct {
	Region     string     `json:"region"`
	Required   bool       `json:"required"`
	FilePath   string     `json:"filePath,omitempty"`
	JoinRule   string     `json:"joinRule,omitempty"`
	Proven     bool       `json:"proven"`
	Heuristic  bool       `json:"heuristic,omitempty"`
	Provenance Provenance `json:"provenance,omitempty"`
	Reason     string     `json:"reason,omitempty"`
	Extra      bool       `json:"extra,omitempty"`
}

type Resolver struct {
	ScenariosRoot string
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
			out = append(out, Result{Region: region.ID, Required: region.Required, Reason: "scenario has no scannable files in a declared UI slot"})
		}
		return out, nil
	}

	used := map[string]bool{}
	out := make([]Result, 0, len(regions)+len(files))
	for _, region := range regions {
		result := Result{Region: region.ID, Required: region.Required}
		if region.Fill.Placeholder != "" {
			result.Reason = "placement is a placeholder"
			out = append(out, result)
			continue
		}
		// Rule 1: a declared binding's test id must occur in one scanned file.
		if region.TestID != "" {
			for _, file := range files {
				content, readErr := os.ReadFile(filepath.Join(r.ScenariosRoot, scenario, filepath.FromSlash(file.Path)))
				if readErr == nil && containsTestID(content, region.TestID) {
					result = resolved(region, file, "binding-testid", true)
					break
				}
			}
		}
		// Rule 2: inventory provenance is backed by adoption_records.
		if result.FilePath == "" && region.Fill.Asset != "" {
			for _, file := range files {
				if assetMatches(region.Fill.Asset, file.ComponentName) && file.Provenance != ProvenanceCustom {
					result = resolved(region, file, "adoption-path", true)
					break
				}
			}
		}
		// Rule 3: slug match is explicitly heuristic.
		if result.FilePath == "" {
			slug := region.Local
			if slug == "" {
				slug = strings.TrimSuffix(region.ID, "-region")
			}
			for _, file := range files {
				if normalizeSlug(file.DisplayName) == normalizeSlug(slug) || normalizeSlug(strings.TrimSuffix(filepath.Base(file.Path), filepath.Ext(file.Path))) == normalizeSlug(slug) {
					result = resolved(region, file, "component-slug", false)
					result.Heuristic = true
					break
				}
			}
		}
		if result.FilePath == "" {
			result.Reason = "no ordered join rule resolved the region"
		} else {
			used[result.FilePath] = true
		}
		out = append(out, result)
	}
	for _, file := range files {
		if !used[file.Path] {
			out = append(out, Result{FilePath: file.Path, Provenance: file.Provenance, Reason: "scanned file resolves to no declared region", Extra: true})
		}
	}
	return out, nil
}

func resolved(region Region, file ObservedFile, rule string, proven bool) Result {
	return Result{Region: region.ID, Required: region.Required, FilePath: file.Path, JoinRule: rule, Proven: proven, Provenance: file.Provenance}
}

var (
	nonAlnum      = regexp.MustCompile(`[^a-z0-9]+`)
	camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)
)

func normalizeSlug(value string) string {
	value = camelBoundary.ReplaceAllString(value, "$1-$2")
	return strings.Trim(nonAlnum.ReplaceAllString(strings.ToLower(value), "-"), "-")
}

func assetMatches(asset, component string) bool {
	asset = strings.TrimPrefix(asset, "components.")
	asset = strings.TrimPrefix(asset, "templates.")
	parts := strings.FieldsFunc(asset, func(r rune) bool { return r == '.' || r == ':' || r == '/' || r == '#' })
	if len(parts) > 0 {
		asset = parts[len(parts)-1]
	}
	return normalizeSlug(asset) == normalizeSlug(component)
}

func containsTestID(content []byte, testID string) bool {
	needle := []byte(testID)
	return len(needle) > 0 && strings.Contains(string(content), string(needle))
}

type pageDocument struct {
	Regions []struct {
		ID        string `json:"id"`
		Required  bool   `json:"required"`
		Component struct {
			Local   string `json:"local"`
			Library struct {
				ID string `json:"id"`
			} `json:"library"`
		} `json:"component"`
	} `json:"regions"`
	Bindings struct {
		Regions map[string]struct {
			TestID string `json:"testid"`
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
		out = append(out, Region{ID: region.ID, Required: region.Required, Local: region.Component.Local, Library: region.Component.Library.ID, TestID: doc.Bindings.Regions[region.ID].TestID, Fill: fills[region.ID]})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
