// Package inventory implements the InventoryService Connect-RPC contract
// defined by ui-health for the React framework. ui-health calls this
// service over RPC to scan a scenario's UI tree and obtain canonical
// ComponentProvenance + WidgetDeclaration + SurfaceRecord values keyed by
// file path.
//
// SQLite (the existing adoptions store) is authoritative for provenance;
// the on-disk JSDoc block is the heal-from signal when the DB row is
// missing or the file has drifted.
package inventory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"connectrpc.com/connect"

	"react-component-library/internal/adoptions"
	"react-component-library/internal/uimanifest"

	provenancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/contracts/provenance"
	widgetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/contracts/widget"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/inventory"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/search"
)

// Deps wires the seams the inventory handler needs.
type Deps struct {
	Logger        *log.Logger
	Adoptions     AdoptionsReader
	ManifestLoad  uimanifest.Loader
	ScenariosRoot string
}

// AdoptionsReader is the slice of the adoptions service used by inventory.
type AdoptionsReader interface {
	List(ctx context.Context, q adoptions.ListQuery) ([]adoptions.Adoption, error)
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ScanScenario(ctx context.Context, req *connect.Request[inventoryv1.ScanScenarioRequest]) (*connect.Response[inventoryv1.ScanScenarioResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	if h.deps.ManifestLoad == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("uimanifest loader not configured"))
	}
	mf, err := h.deps.ManifestLoad.Load(scenario)
	if err != nil {
		var notFound uimanifest.ErrScenarioNotFound
		if errors.As(err, &notFound) {
			return connect.NewResponse(&inventoryv1.ScanScenarioResponse{Scenario: scenario}), nil
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Index adoption rows by file path (scenarios-relative) so we can
	// produce provenance per .tsx file in O(1).
	adoptionsByPath := map[string]adoptions.Adoption{}
	if h.deps.Adoptions != nil {
		rows, err := h.deps.Adoptions.List(ctx, adoptions.ListQuery{Scenario: scenario, Limit: 1024})
		if err != nil {
			h.deps.Logger.Printf("[rcl/inventory] list adoptions for %s: %v", scenario, err)
		}
		for _, r := range rows {
			adoptionsByPath[normalizeRel(r.AdoptedPath)] = r
		}
	}

	scenarioRoot := filepath.Join(h.deps.ScenariosRoot, scenario)
	resp := &inventoryv1.ScanScenarioResponse{Scenario: scenario}

	// Walk each slot's dir for .tsx files.
	slotNames := make([]string, 0, len(mf.Slots))
	for k := range mf.Slots {
		slotNames = append(slotNames, k)
	}
	sort.Strings(slotNames)

	seen := map[string]struct{}{}
	for _, name := range slotNames {
		slot := mf.Slots[name]
		dir := filepath.Join(scenarioRoot, slot.Dir)
		entries, err := walkTSX(dir)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				h.deps.Logger.Printf("[rcl/inventory] walk %s: %v", dir, err)
			}
			continue
		}
		for _, abs := range entries {
			rel, err := filepath.Rel(scenarioRoot, abs)
			if err != nil {
				continue
			}
			rel = filepath.ToSlash(rel)
			if _, dup := seen[rel]; dup {
				continue
			}
			seen[rel] = struct{}{}

			content, readErr := os.ReadFile(abs)
			if readErr != nil {
				h.deps.Logger.Printf("[rcl/inventory] read %s: %v", abs, readErr)
				continue
			}

			provenance := buildProvenance(rel, content, adoptionsByPath)
			resp.Provenance = append(resp.Provenance, provenance)
			if w := parseWidget(rel, content); w != nil {
				resp.Widgets = append(resp.Widgets, w)
			}
			resp.Surfaces = append(resp.Surfaces, &inventoryv1.SurfaceRecord{
				Scenario:    scenario,
				Slot:        name,
				Kind:        slotKind(name),
				DisplayName: componentNameFromPath(rel),
				Description: firstJSDocLine(content),
				FilePath:    rel,
			})
		}
	}
	return connect.NewResponse(resp), nil
}

// isTestFile reports whether a basename is a test/story/spec/test-utility
// file that should not appear in the production-surface inventory.
func isTestFile(base string) bool {
	lower := strings.ToLower(base)
	for _, suf := range []string{".test.tsx", ".test.jsx", ".spec.tsx", ".spec.jsx", ".stories.tsx", ".stories.jsx"} {
		if strings.HasSuffix(lower, suf) {
			return true
		}
	}
	return false
}

// walkTSX returns every .tsx file under dir, recursively. Non-existent dir
// returns os.ErrNotExist; callers may treat that as "skip slot".
func walkTSX(dir string) ([]string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}
	var out []string
	err = filepath.Walk(dir, func(path string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if fi.IsDir() {
			name := fi.Name()
			// Skip dotfiles, node_modules, and conventional test/test-utility
			// dirs (__tests__, test-utils, __mocks__). Surfaces under these
			// dirs are infrastructure, not user-facing UI.
			if strings.HasPrefix(name, ".") || name == "node_modules" ||
				name == "__tests__" || name == "test-utils" || name == "__mocks__" {
				return filepath.SkipDir
			}
			return nil
		}
		if !(strings.HasSuffix(path, ".tsx") || strings.HasSuffix(path, ".jsx")) {
			return nil
		}
		// Exclude test files from the inventory: ui-health is a discovery
		// index of *production* surfaces, and test-only files (renderWith-
		// Providers, *.test.tsx, *.spec.tsx, .stories.tsx) crowd the search
		// results without representing reusable UI.
		base := fi.Name()
		if isTestFile(base) {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

var (
	tagSourceRE   = regexp.MustCompile(`@vrooliComponentSource\s+(\S+)`)
	tagVersionRE  = regexp.MustCompile(`@vrooliComponentVersion\s+(\S+)`)
	tagAdoptionRE = regexp.MustCompile(`@vrooliComponentAdoption\s+(\S+)`)
	tagAppliedRE  = regexp.MustCompile(`@vrooliComponentAppliedAt\s+(\S+)`)
	tagSourceShaRE = regexp.MustCompile(`@vrooliComponentSourceSha256\s+(\S+)`)
	tagDriftHashRE = regexp.MustCompile(`@vrooliComponentDriftHash\s+(\S+)`)
	tagComponentNameRE = regexp.MustCompile(`@vrooliComponentName\s+(\S+)`)
)

// buildProvenance combines the SQLite row (authoritative) with the on-disk
// JSDoc block (heal-from) to produce a canonical ComponentProvenance.
//
// Decision logic:
//   - DB row present:
//       compute drift_hash from on-disk content.
//       if drift_hash == source_sha256 from row → ADOPTED_UNMODIFIED.
//       else → ADOPTED_MODIFIED.
//   - DB row absent but JSDoc block present:
//       UNKNOWN (DB drift — heal-from signal exists but no record).
//   - Neither:
//       CUSTOM.
func buildProvenance(filePath string, content []byte, byPath map[string]adoptions.Adoption) *provenancev1.ComponentProvenance {
	drift := sha256Hex(content)
	if row, ok := byPath[filePath]; ok {
		p := &provenancev1.ComponentProvenance{
			Library:        row.LibraryID,
			LibraryVersion: row.AdoptedVersion,
			ComponentName:  matchOrEmpty(tagComponentNameRE, content),
			AdoptionId:     row.ID,
			AppliedAt:      row.AppliedAt.UTC().Format("2006-01-02T15:04:05Z"),
			SourceSha256:   row.SourceSHA256,
			DriftHash:      drift,
			FilePath:       filePath,
		}
		if drift == row.SourceSHA256 && row.SourceSHA256 != "" {
			p.Provenance = provenancev1.Provenance_PROVENANCE_ADOPTED_UNMODIFIED
		} else {
			p.Provenance = provenancev1.Provenance_PROVENANCE_ADOPTED_MODIFIED
		}
		return p
	}

	source := matchOrEmpty(tagSourceRE, content)
	if source != "" {
		return &provenancev1.ComponentProvenance{
			Provenance:     provenancev1.Provenance_PROVENANCE_UNKNOWN,
			Library:        source,
			LibraryVersion: matchOrEmpty(tagVersionRE, content),
			ComponentName:  matchOrEmpty(tagComponentNameRE, content),
			AdoptionId:     matchOrEmpty(tagAdoptionRE, content),
			AppliedAt:      matchOrEmpty(tagAppliedRE, content),
			SourceSha256:   matchOrEmpty(tagSourceShaRE, content),
			DriftHash:      driftOrTag(drift, matchOrEmpty(tagDriftHashRE, content)),
			FilePath:       filePath,
		}
	}

	return &provenancev1.ComponentProvenance{
		Provenance: provenancev1.Provenance_PROVENANCE_CUSTOM,
		DriftHash:  drift,
		FilePath:   filePath,
	}
}

func driftOrTag(computed, tagged string) string {
	if computed != "" {
		return computed
	}
	return tagged
}

// parseWidget looks for a JSDoc @vrooliWidget block in the file. Shape is
// minimal in v1: the tag's argument is the widget_id; props_schema is
// optional and provided via an adjacent @vrooliWidgetProps line containing
// a single-line JSON literal.
var (
	tagWidgetIDRE     = regexp.MustCompile(`@vrooliWidget\s+(\S+)`)
	tagWidgetPropsRE  = regexp.MustCompile(`@vrooliWidgetProps\s+(\{.*?\})`)
	tagWidgetSlotRE   = regexp.MustCompile(`@vrooliWidgetSlot\s+(\S+)`)
	tagWidgetScopeRE  = regexp.MustCompile(`@vrooliWidgetScope\s+(\S+)`)
	tagWidgetDescRE   = regexp.MustCompile(`@vrooliWidgetDescription\s+(.+)`)
	tagWidgetExportRE = regexp.MustCompile(`@vrooliWidgetExport\s+(\S+)`)
)

func parseWidget(filePath string, content []byte) *widgetv1.WidgetDeclaration {
	id := matchOrEmpty(tagWidgetIDRE, content)
	if id == "" {
		return nil
	}
	component := matchOrEmpty(tagWidgetExportRE, content)
	if component == "" {
		component = componentNameFromPath(filePath)
	}
	slot := widgetv1.WidgetSlot_WIDGET_SLOT_UNSPECIFIED
	switch strings.ToLower(matchOrEmpty(tagWidgetSlotRE, content)) {
	case "inline":
		slot = widgetv1.WidgetSlot_WIDGET_SLOT_INLINE
	case "sidebar":
		slot = widgetv1.WidgetSlot_WIDGET_SLOT_SIDEBAR
	case "full":
		slot = widgetv1.WidgetSlot_WIDGET_SLOT_FULL
	}
	scope := widgetv1.WidgetScope_WIDGET_SCOPE_UNSPECIFIED
	switch strings.ToLower(matchOrEmpty(tagWidgetScopeRE, content)) {
	case "scenario":
		scope = widgetv1.WidgetScope_WIDGET_SCOPE_SCENARIO
	case "global":
		scope = widgetv1.WidgetScope_WIDGET_SCOPE_GLOBAL
	}
	return &widgetv1.WidgetDeclaration{
		WidgetId:        id,
		ComponentName:   component,
		PropsSchemaJson: matchOrEmpty(tagWidgetPropsRE, content),
		Slot:            slot,
		Scope:           scope,
		Description:     strings.TrimSpace(matchOrEmpty(tagWidgetDescRE, content)),
		FilePath:        filePath,
	}
}

// slotKind maps a slot name to the SurfaceKind enum used by the search
// index. The mapping is heuristic — when in doubt, OTHER.
func slotKind(slotName string) searchv1.SurfaceKind {
	switch slotName {
	case "ui-primitive", "shared-component", "feature-component":
		return searchv1.SurfaceKind_SURFACE_KIND_COMPONENT
	case "page":
		return searchv1.SurfaceKind_SURFACE_KIND_PAGE
	case "feature":
		return searchv1.SurfaceKind_SURFACE_KIND_FEATURE
	case "hook":
		return searchv1.SurfaceKind_SURFACE_KIND_HOOK
	case "layout-shell", "layout-nav":
		return searchv1.SurfaceKind_SURFACE_KIND_LAYOUT
	default:
		return searchv1.SurfaceKind_SURFACE_KIND_OTHER
	}
}

// componentNameFromPath derives a PascalCase component name from the file
// path's basename (without extension).
func componentNameFromPath(rel string) string {
	base := filepath.Base(rel)
	if i := strings.LastIndexByte(base, '.'); i > 0 {
		base = base[:i]
	}
	return base
}

func firstJSDocLine(content []byte) string {
	s := string(content)
	idx := strings.Index(s, "/**")
	if idx < 0 {
		return ""
	}
	end := strings.Index(s[idx:], "*/")
	if end < 0 {
		return ""
	}
	block := s[idx : idx+end]
	for _, raw := range strings.Split(block, "\n") {
		line := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "*"))
		line = strings.TrimSpace(strings.TrimPrefix(line, "/**"))
		if line == "" || strings.HasPrefix(line, "@") {
			continue
		}
		return line
	}
	return ""
}

func matchOrEmpty(re *regexp.Regexp, content []byte) string {
	m := re.FindSubmatch(content)
	if len(m) < 2 {
		return ""
	}
	return string(m[1])
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// normalizeRel converts a stored adopted_path to forward-slash form.
func normalizeRel(p string) string {
	p = strings.TrimPrefix(p, "./")
	return filepath.ToSlash(strings.TrimSpace(p))
}

// silence unused import linter when no fmt usage triggers above.
var _ = fmt.Sprint
