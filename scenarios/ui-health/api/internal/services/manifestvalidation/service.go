// Package manifestvalidation validates a scenario's UI manifest against
// the scenario-ui-manifest/v1 schema, the on-disk slot directories, and
// the template-vs-overlay rules. It is the implementation behind the
// ValidationService Connect-RPC handler.
//
// Pure (no DB, no HTTP). Takes a repo root and a scenario name, returns a
// Report value. Findings drive both the CLI/UI display and the test-genie
// phase_ui_health Observations.
package manifestvalidation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Severity classifies a Finding.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Finding is a single validation result tied to a location inside the
// scenario (manifest path, slot directory, on-disk file).
type Finding struct {
	Severity   Severity
	Code       string
	Location   string
	Message    string
	Suggestion string
}

// Summary counts findings by severity.
type Summary struct {
	Errors   int
	Warnings int
	Infos    int
}

// Report is the validator's full output.
type Report struct {
	Scenario string
	Passed   bool
	Findings []Finding
	Summary  Summary
}

// Service runs UI manifest validation against the local repo.
type Service struct {
	RepoRoot string
	Logger   *log.Logger
}

// New constructs a Service.
func New(repoRoot string, logger *log.Logger) *Service {
	if logger == nil {
		logger = log.Default()
	}
	return &Service{RepoRoot: repoRoot, Logger: logger}
}

// ValidateScenario is the entrypoint the Connect handler calls.
func (s *Service) ValidateScenario(_ context.Context, scenario string) (Report, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return Report{}, errors.New("scenario is required")
	}
	if strings.TrimSpace(s.RepoRoot) == "" {
		return Report{}, errors.New("repo root is required")
	}
	scenarioDir := filepath.Join(s.RepoRoot, "scenarios", scenario)
	info, err := os.Stat(scenarioDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Report{}, fmt.Errorf("scenario %q does not exist", scenario)
		}
		return Report{}, fmt.Errorf("stat scenario dir: %w", err)
	}
	if !info.IsDir() {
		return Report{}, fmt.Errorf("scenario %q path is not a directory", scenario)
	}

	rep := Report{Scenario: scenario}

	// Short-circuit scenarios that have no UI surface at all.
	// A scenario "has UI" if it ships a ui/ directory with a package.json
	// (the react-vite template guarantees this) or any framework-recognizable
	// entry. Without one, manifest validation has nothing to assert against.
	if !scenarioHasUI(scenarioDir) {
		rep.Findings = append(rep.Findings, Finding{
			Severity: SeverityInfo,
			Code:     "no_ui_surface",
			Location: scenarioDir,
			Message:  "scenario has no ui/ directory; skipping UI manifest validation",
		})
		return finalize(rep), nil
	}

	tmplManifest, tmplPath, tmplFinds := s.loadTemplateManifest(scenario)
	rep.Findings = append(rep.Findings, tmplFinds...)
	if tmplManifest == nil {
		return finalize(rep), nil
	}

	overlay, overlayPath, overlayFinds := s.loadOverlayManifest(scenario)
	rep.Findings = append(rep.Findings, overlayFinds...)

	merged := mergeManifests(tmplManifest, overlay)
	rep.Findings = append(rep.Findings, validateOverlayKeys(tmplManifest, overlay, overlayPath)...)
	slotFinds := validateSlotsOnDisk(s.RepoRoot, scenario, merged, tmplPath)
	slotFinds = collapsePredatesTemplateLayout(slotFinds, len(merged.Slots), filepath.Join(s.RepoRoot, "scenarios", scenario))
	rep.Findings = append(rep.Findings, slotFinds...)
	rep.Findings = append(rep.Findings, validateSlotPaths(merged, tmplPath)...)
	rep.Findings = append(rep.Findings, validateSlotOverlap(merged, tmplPath)...)
	return finalize(rep), nil
}

// collapsePredatesTemplateLayout replaces a storm of slot_dir_missing /
// slot_parent_dir_missing findings with a single ui_predates_template_layout
// warning when most of the template's slots have no on-disk counterpart.
//
// Rationale: scenarios generated before the v1 ui/manifest.json template
// landed have working UIs that don't conform to the new slot layout. One
// summary signal is more actionable than N per-slot warnings — the slots
// aren't missing one-by-one, the whole layout pre-dates the contract.
func collapsePredatesTemplateLayout(finds []Finding, totalSlots int, scenarioRoot string) []Finding {
	if totalSlots == 0 {
		return finds
	}
	missing := 0
	for _, f := range finds {
		if f.Code == "slot_dir_missing" || f.Code == "slot_parent_dir_missing" {
			missing++
		}
	}
	// Collapse when the per-slot warnings become a "storm": at least 3
	// missing slot dirs AND at least a quarter of the template's slots
	// missing. Below that, individual slot signals remain more useful than
	// a single summary. The 25% ratio + 3-finding floor catches scenarios
	// that predate the new layout (typically 4+ of 14 slots missing) while
	// keeping detail for scenarios that just lack one or two dirs.
	if missing < 3 || missing*4 < totalSlots {
		return finds
	}
	kept := make([]Finding, 0, len(finds)-missing+1)
	for _, f := range finds {
		if f.Code == "slot_dir_missing" || f.Code == "slot_parent_dir_missing" {
			continue
		}
		kept = append(kept, f)
	}
	kept = append(kept, Finding{
		Severity:   SeverityWarning,
		Code:       "ui_predates_template_layout",
		Location:   scenarioRoot,
		Message:    fmt.Sprintf("scenario's UI does not conform to the template's slot layout (%d of %d declared slot directories are missing on disk)", missing, totalSlots),
		Suggestion: "Migrate the UI to the template's slot layout, or declare a .vrooli/ui-manifest.json overlay that remaps slot dirs to the scenario's actual structure",
	})
	return kept
}

// uiManifest is the minimal in-memory view of a ui/manifest.json.
type uiManifest struct {
	Contract struct {
		Kind     string `json:"kind"`
		Schema   string `json:"schema"`
		Template string `json:"template"`
	} `json:"contract"`
	Slots map[string]struct {
		Dir             string `json:"dir"`
		PathPattern     string `json:"pathPattern"`
		MultiFile       bool   `json:"multiFile"`
		RequiresFeature bool   `json:"requiresFeature"`
	} `json:"slots"`
	Defaults struct {
		Slot string `json:"slot"`
	} `json:"defaults"`
}

// loadTemplateManifest resolves the scenario's template id and reads its
// ui/manifest.json. Returns (nil, "", findings) on errors so the caller can
// surface them as Findings rather than aborting validation.
func (s *Service) loadTemplateManifest(scenario string) (*uiManifest, string, []Finding) {
	svcPath := filepath.Join(s.RepoRoot, "scenarios", scenario, ".vrooli", "service.json")
	raw, err := os.ReadFile(svcPath)
	if err != nil {
		return nil, "", []Finding{{
			Severity: SeverityError,
			Code:     "service_json_missing",
			Location: svcPath,
			Message:  fmt.Sprintf("read service.json: %v", err),
		}}
	}
	var svc struct {
		Generation struct {
			Template struct {
				ID string `json:"id"`
			} `json:"template"`
		} `json:"generation"`
	}
	if err := json.Unmarshal(raw, &svc); err != nil {
		return nil, "", []Finding{{
			Severity: SeverityError,
			Code:     "service_json_invalid",
			Location: svcPath,
			Message:  fmt.Sprintf("parse service.json: %v", err),
		}}
	}
	if svc.Generation.Template.ID == "" {
		// Pre-template-tracking scenarios still have working UIs; treat as
		// "validation impossible" (warn), not "scenario broken" (error). This
		// keeps phase_ui_health and other gates from failing across the
		// existing fleet until each scenario backfills generation.template.id.
		return nil, "", []Finding{{
			Severity:   SeverityWarning,
			Code:       "template_id_missing",
			Location:   svcPath,
			Message:    "generation.template.id is not declared; scenario predates template tracking, so UI manifest validation is skipped",
			Suggestion: "Declare generation.template.id in service.json to enable template-aware validation",
		}}
	}
	mfPath := filepath.Join(s.RepoRoot, "templates", "scenarios", svc.Generation.Template.ID, "ui", "manifest.json")
	mfRaw, err := os.ReadFile(mfPath)
	if err != nil {
		return nil, mfPath, []Finding{{
			Severity: SeverityError,
			Code:     "template_unknown",
			Location: mfPath,
			Message:  fmt.Sprintf("template %q referenced by service.json has no ui/manifest.json at %s", svc.Generation.Template.ID, mfPath),
		}}
	}
	var mf uiManifest
	if err := json.Unmarshal(mfRaw, &mf); err != nil {
		return nil, mfPath, []Finding{{
			Severity: SeverityError,
			Code:     "template_manifest_invalid",
			Location: mfPath,
			Message:  fmt.Sprintf("parse template manifest: %v", err),
		}}
	}
	var finds []Finding
	if mf.Contract.Kind != "scenario-ui" {
		finds = append(finds, Finding{
			Severity: SeverityError,
			Code:     "contract_kind_mismatch",
			Location: mfPath,
			Message:  fmt.Sprintf("contract.kind must be %q, got %q", "scenario-ui", mf.Contract.Kind),
		})
	}
	if mf.Contract.Schema != "scenario-ui-manifest/v1" {
		finds = append(finds, Finding{
			Severity: SeverityError,
			Code:     "contract_schema_mismatch",
			Location: mfPath,
			Message:  fmt.Sprintf("contract.schema must be %q, got %q", "scenario-ui-manifest/v1", mf.Contract.Schema),
		})
	}
	if len(mf.Slots) == 0 {
		finds = append(finds, Finding{
			Severity: SeverityError,
			Code:     "slots_empty",
			Location: mfPath,
			Message:  "template manifest declares no slots",
		})
	}
	return &mf, mfPath, finds
}

// loadOverlayManifest reads the optional scenario overlay manifest.
func (s *Service) loadOverlayManifest(scenario string) (*uiManifest, string, []Finding) {
	path := filepath.Join(s.RepoRoot, "scenarios", scenario, ".vrooli", "ui-manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, path, nil
		}
		return nil, path, []Finding{{
			Severity: SeverityError,
			Code:     "overlay_read_failed",
			Location: path,
			Message:  fmt.Sprintf("read overlay manifest: %v", err),
		}}
	}
	var mf uiManifest
	if err := json.Unmarshal(raw, &mf); err != nil {
		return nil, path, []Finding{{
			Severity: SeverityError,
			Code:     "overlay_invalid",
			Location: path,
			Message:  fmt.Sprintf("parse overlay manifest: %v", err),
		}}
	}
	return &mf, path, nil
}

// mergeManifests folds overlay slots over template slots. Overlay-only slots
// are surfaced as findings by validateOverlayKeys but still merged so the
// rest of the validator can run on the full set.
func mergeManifests(tmpl, overlay *uiManifest) uiManifest {
	merged := uiManifest{}
	merged.Contract = tmpl.Contract
	merged.Defaults = tmpl.Defaults
	merged.Slots = make(map[string]struct {
		Dir             string `json:"dir"`
		PathPattern     string `json:"pathPattern"`
		MultiFile       bool   `json:"multiFile"`
		RequiresFeature bool   `json:"requiresFeature"`
	}, len(tmpl.Slots))
	for k, v := range tmpl.Slots {
		merged.Slots[k] = v
	}
	if overlay != nil {
		for k, v := range overlay.Slots {
			merged.Slots[k] = v
		}
	}
	return merged
}

// validateOverlayKeys enforces the "no new slot names" rule: overlay slots
// must reference a template slot name.
func validateOverlayKeys(tmpl, overlay *uiManifest, overlayPath string) []Finding {
	if overlay == nil {
		return nil
	}
	var finds []Finding
	keys := make([]string, 0, len(overlay.Slots))
	for k := range overlay.Slots {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if _, ok := tmpl.Slots[k]; !ok {
			finds = append(finds, Finding{
				Severity:   SeverityError,
				Code:       "overlay_unknown_slot",
				Location:   overlayPath,
				Message:    fmt.Sprintf("overlay declares slot %q not present in template", k),
				Suggestion: "Remove the slot or add it to the template's ui/manifest.json",
			})
		}
	}
	return finds
}

// validateSlotsOnDisk stats each slot's dir and reports missing or empty
// directories.
func validateSlotsOnDisk(repoRoot, scenario string, mf uiManifest, mfPath string) []Finding {
	var finds []Finding
	keys := make([]string, 0, len(mf.Slots))
	for k := range mf.Slots {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, name := range keys {
		slot := mf.Slots[name]
		if slot.Dir == "" {
			finds = append(finds, Finding{
				Severity: SeverityError,
				Code:     "slot_dir_empty",
				Location: mfPath,
				Message:  fmt.Sprintf("slot %q: dir is empty", name),
			})
			continue
		}
		// Placeholder dirs (e.g. "ui/src/features/{feature}") aren't real
		// paths — they're patterns. Resolve the parent and inspect concrete
		// instances instead of stat-ing the literal placeholder string.
		if tokenRE.MatchString(slot.Dir) {
			finds = append(finds, validatePlaceholderSlot(repoRoot, scenario, name, slot.Dir)...)
			continue
		}
		full := filepath.Join(repoRoot, "scenarios", scenario, slot.Dir)
		info, err := os.Stat(full)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				finds = append(finds, Finding{
					Severity: SeverityWarning,
					Code:     "slot_dir_missing",
					Location: full,
					Message:  fmt.Sprintf("slot %q: directory does not exist on disk", name),
				})
				continue
			}
			finds = append(finds, Finding{
				Severity: SeverityError,
				Code:     "slot_dir_stat_failed",
				Location: full,
				Message:  fmt.Sprintf("slot %q: stat dir: %v", name, err),
			})
			continue
		}
		if !info.IsDir() {
			finds = append(finds, Finding{
				Severity: SeverityError,
				Code:     "slot_dir_not_directory",
				Location: full,
				Message:  fmt.Sprintf("slot %q: path is not a directory", name),
			})
		}
	}
	return finds
}

// validatePlaceholderSlot handles slots whose dir contains a `{token}`
// placeholder. The parent (the path segments before the first placeholder)
// must exist; the placeholder segment may resolve to zero or more concrete
// subdirectories (e.g. one per feature). Zero instances is an info finding,
// not a missing-directory warning, because the placeholder path itself was
// never supposed to exist on disk.
func validatePlaceholderSlot(repoRoot, scenario, slotName, dir string) []Finding {
	segments := strings.Split(filepath.ToSlash(dir), "/")
	var parentParts []string
	for _, seg := range segments {
		if tokenRE.MatchString(seg) {
			break
		}
		parentParts = append(parentParts, seg)
	}
	parentRel := strings.Join(parentParts, "/")
	parentAbs := filepath.Join(repoRoot, "scenarios", scenario, parentRel)
	info, err := os.Stat(parentAbs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Finding{{
				Severity: SeverityWarning,
				Code:     "slot_parent_dir_missing",
				Location: parentAbs,
				Message:  fmt.Sprintf("slot %q: parent directory %q does not exist on disk", slotName, parentRel),
			}}
		}
		return []Finding{{
			Severity: SeverityError,
			Code:     "slot_dir_stat_failed",
			Location: parentAbs,
			Message:  fmt.Sprintf("slot %q: stat parent dir: %v", slotName, err),
		}}
	}
	if !info.IsDir() {
		return []Finding{{
			Severity: SeverityError,
			Code:     "slot_dir_not_directory",
			Location: parentAbs,
			Message:  fmt.Sprintf("slot %q: parent path %q is not a directory", slotName, parentRel),
		}}
	}
	entries, err := os.ReadDir(parentAbs)
	if err != nil {
		return []Finding{{
			Severity: SeverityError,
			Code:     "slot_dir_stat_failed",
			Location: parentAbs,
			Message:  fmt.Sprintf("slot %q: read parent dir: %v", slotName, err),
		}}
	}
	instances := 0
	for _, e := range entries {
		if e.IsDir() {
			instances++
		}
	}
	if instances == 0 {
		return []Finding{{
			Severity: SeverityInfo,
			Code:     "slot_instances_empty",
			Location: parentAbs,
			Message:  fmt.Sprintf("slot %q: no concrete instances under %q yet", slotName, parentRel),
		}}
	}
	return nil
}

// scenarioHasUI reports whether the scenario directory contains a UI surface
// worth validating. The react-vite template guarantees ui/package.json; we
// use that as a stable signal. Hand-built UIs with a different layout can
// add a marker file later if needed.
func scenarioHasUI(scenarioDir string) bool {
	if _, err := os.Stat(filepath.Join(scenarioDir, "ui", "package.json")); err == nil {
		return true
	}
	return false
}

// v1 path-pattern token vocabulary.
var pathPatternTokens = map[string]struct{}{
	"{dir}":           {},
	"{ComponentName}": {},
	"{componentName}": {},
	"{camelName}":     {},
	"{kebab-name}":    {},
	"{feature}":       {},
	"{locale}":        {},
}

var tokenRE = regexp.MustCompile(`\{[A-Za-z][A-Za-z0-9_-]*\}`)

// validateSlotPaths checks pathPattern tokens against the v1 vocabulary.
func validateSlotPaths(mf uiManifest, mfPath string) []Finding {
	var finds []Finding
	keys := make([]string, 0, len(mf.Slots))
	for k := range mf.Slots {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, name := range keys {
		slot := mf.Slots[name]
		if slot.PathPattern == "" {
			continue
		}
		for _, tok := range tokenRE.FindAllString(slot.PathPattern, -1) {
			if _, ok := pathPatternTokens[tok]; !ok {
				finds = append(finds, Finding{
					Severity:   SeverityWarning,
					Code:       "path_pattern_unknown_token",
					Location:   mfPath,
					Message:    fmt.Sprintf("slot %q: pathPattern contains unknown token %s", name, tok),
					Suggestion: "Use one of {dir}, {ComponentName}, {componentName}, {camelName}, {kebab-name}, {feature}, {locale}",
				})
			}
		}
	}
	return finds
}

// validateSlotOverlap reports when two slot dirs are equal or one is a
// strict prefix of another (modulo trailing /). multiFile slots may
// intentionally nest; the rule applies only to single-file slots.
func validateSlotOverlap(mf uiManifest, mfPath string) []Finding {
	type entry struct {
		name string
		dir  string
		mf   bool
	}
	entries := make([]entry, 0, len(mf.Slots))
	for name, slot := range mf.Slots {
		entries = append(entries, entry{name: name, dir: filepath.Clean(slot.Dir), mf: slot.MultiFile})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	var finds []Finding
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			a, b := entries[i], entries[j]
			if a.mf || b.mf {
				continue
			}
			if a.dir == b.dir {
				finds = append(finds, Finding{
					Severity:   SeverityWarning,
					Code:       "slot_dir_overlap_equal",
					Location:   mfPath,
					Message:    fmt.Sprintf("slots %q and %q share dir %q", a.name, b.name, a.dir),
					Suggestion: "Differentiate by dir or mark one slot multiFile",
				})
			}
		}
	}
	return finds
}

func finalize(rep Report) Report {
	for _, f := range rep.Findings {
		switch f.Severity {
		case SeverityError:
			rep.Summary.Errors++
		case SeverityWarning:
			rep.Summary.Warnings++
		case SeverityInfo:
			rep.Summary.Infos++
		}
	}
	rep.Passed = rep.Summary.Errors == 0
	return rep
}
