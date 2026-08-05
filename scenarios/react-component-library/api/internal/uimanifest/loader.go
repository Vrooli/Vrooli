// Package uimanifest loads a target scenario's UI manifest by chasing
// `scenarios/<scenario>/.vrooli/service.json` -> generation.template.id ->
// `templates/scenarios/<id>/ui/manifest.json`.
//
// Pure: takes a repo root, returns parsed manifests. No DB, no HTTP, no proto.
// The adoption resolver in internal/adoptions consumes this.
package uimanifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Manifest is the parsed scenario UI manifest. Mirrors
// scenario-ui-manifest/v1 (.vrooli/schemas/scenario-ui-manifest.schema.json).
type Manifest struct {
	SchemaVersion            string          `json:"schemaVersion"`
	Contract                 Contract        `json:"contract"`
	Slots                    map[string]Slot `json:"slots"`
	Defaults                 Defaults        `json:"defaults"`
	Provides                 []string        `json:"provides"`
	Targets                  []string        `json:"targets"`
	Idiom                    string          `json:"idiom"`
	Gates                    []string        `json:"gates"`
	TemplateStructureVersion string          `json:"templateStructureVersion"`
	SlotCoverage             string          `json:"slotCoverage"`
}

// Contract block (mirrors the schema's Contract subobject).
type Contract struct {
	Kind          string `json:"kind"`
	Schema        string `json:"schema"`
	Template      string `json:"template"`
	ExtensionRule string `json:"extensionRule"`
}

// Slot is a single slot definition.
type Slot struct {
	Dir             string  `json:"dir"`
	Description     string  `json:"description"`
	PathPattern     string  `json:"pathPattern"`
	MultiFile       bool    `json:"multiFile"`
	RequiresFeature bool    `json:"requiresFeature"`
	Barrel          *string `json:"barrel"`
}

// Defaults block.
type Defaults struct {
	Slot        string `json:"slot"`
	FallbackDir string `json:"fallbackDir"`
}

// Loader resolves a scenario name to its template's UI manifest. LoadTemplate
// reads a template's manifest directly, skipping scenario resolution — used by
// callers that want scenario-agnostic placement against a named template.
type Loader interface {
	Load(scenario string) (Manifest, error)
	LoadTemplate(templateID string) (Manifest, error)
}

// FSLoader reads from the local filesystem rooted at RepoRoot.
type FSLoader struct {
	RepoRoot string
}

// NewFSLoader constructs a Loader rooted at the given repo path.
func NewFSLoader(repoRoot string) *FSLoader {
	return &FSLoader{RepoRoot: repoRoot}
}

// ErrScenarioNotFound is returned when the scenario directory or its
// service.json is missing.
type ErrScenarioNotFound struct{ Scenario string }

func (e ErrScenarioNotFound) Error() string {
	return fmt.Sprintf("scenario %q: service.json not found", e.Scenario)
}

// ErrTemplateNotDeclared is returned when service.json exists but does not
// declare a generation.template.id.
type ErrTemplateNotDeclared struct{ Scenario string }

func (e ErrTemplateNotDeclared) Error() string {
	return fmt.Sprintf("scenario %q: generation.template.id not declared", e.Scenario)
}

// ErrTemplateManifestMissing is returned when the template directory has no
// ui/manifest.json.
type ErrTemplateManifestMissing struct{ Template string }

func (e ErrTemplateManifestMissing) Error() string {
	return fmt.Sprintf("template %q: ui/manifest.json not found", e.Template)
}

// ErrInvalidManifest wraps JSON / shape errors.
type ErrInvalidManifest struct {
	Path   string
	Reason string
}

func (e ErrInvalidManifest) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Reason)
}

// serviceJSON is the subset of .vrooli/service.json we care about.
type serviceJSON struct {
	Generation struct {
		Template struct {
			ID string `json:"id"`
		} `json:"template"`
	} `json:"generation"`
}

// Load resolves the scenario's template and returns the parsed manifest.
func (l *FSLoader) Load(scenario string) (Manifest, error) {
	if l == nil || l.RepoRoot == "" {
		return Manifest{}, errors.New("uimanifest: loader has no repo root")
	}
	svcPath, err := repocontract.ScenarioServiceManifestPath(l.RepoRoot, scenario)
	if err != nil {
		return Manifest{}, err
	}
	raw, err := os.ReadFile(svcPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Manifest{}, ErrScenarioNotFound{Scenario: scenario}
		}
		return Manifest{}, fmt.Errorf("read %s: %w", svcPath, err)
	}
	var svc serviceJSON
	if err := json.Unmarshal(raw, &svc); err != nil {
		return Manifest{}, ErrInvalidManifest{Path: svcPath, Reason: err.Error()}
	}
	templateID := svc.Generation.Template.ID
	if templateID == "" {
		return Manifest{}, ErrTemplateNotDeclared{Scenario: scenario}
	}
	mf, err := l.LoadTemplate(templateID)
	if err != nil {
		return Manifest{}, err
	}
	overlayPath := filepath.Join(l.RepoRoot, "scenarios", scenario, ".vrooli", "ui-manifest.json")
	data, readErr := os.ReadFile(overlayPath)
	if readErr == nil {
		var overlay Manifest
		if err := json.Unmarshal(data, &overlay); err != nil {
			return Manifest{}, ErrInvalidManifest{Path: overlayPath, Reason: err.Error()}
		}
		for name := range overlay.Slots {
			if _, ok := mf.Slots[name]; !ok {
				return Manifest{}, ErrInvalidManifest{Path: overlayPath, Reason: fmt.Sprintf("overlay slot %q is not declared by the template", name)}
			}
		}
		mf = merge(mf, overlay)
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return Manifest{}, fmt.Errorf("read %s: %w", overlayPath, readErr)
	}
	return mf, nil
}

// LoadTemplate reads the template-owned UI manifest directly. Exported so
// callers that already know the template ID (or want to validate a template's
// own manifest) can skip the scenario-resolution step.
func (l *FSLoader) LoadTemplate(templateID string) (Manifest, error) {
	mfPath := filepath.Join(l.RepoRoot, "templates", "scenarios", templateID, "ui", "manifest.json")
	raw, err := os.ReadFile(mfPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Manifest{}, ErrTemplateManifestMissing{Template: templateID}
		}
		return Manifest{}, fmt.Errorf("read %s: %w", mfPath, err)
	}
	var mf Manifest
	if err := json.Unmarshal(raw, &mf); err != nil {
		return Manifest{}, ErrInvalidManifest{Path: mfPath, Reason: err.Error()}
	}
	if err := validate(mf, mfPath); err != nil {
		return Manifest{}, err
	}
	return mf, nil
}

func validate(mf Manifest, path string) error {
	if mf.Contract.Kind != "scenario-ui" {
		return ErrInvalidManifest{Path: path, Reason: fmt.Sprintf("contract.kind must be %q, got %q", "scenario-ui", mf.Contract.Kind)}
	}
	if mf.Contract.Schema != "scenario-ui-manifest/v1" && mf.Contract.Schema != "scenario-ui-manifest/v2" {
		return ErrInvalidManifest{Path: path, Reason: "contract.schema must be scenario-ui-manifest/v1 or scenario-ui-manifest/v2"}
	}
	if len(mf.Slots) == 0 {
		return ErrInvalidManifest{Path: path, Reason: "slots: at least one slot is required"}
	}
	for name, slot := range mf.Slots {
		if slot.Dir == "" {
			return ErrInvalidManifest{Path: path, Reason: fmt.Sprintf("slot %q: dir is required", name)}
		}
	}
	return nil
}

func merge(base, overlay Manifest) Manifest {
	merged := base
	merged.Slots = make(map[string]Slot, len(base.Slots)+len(overlay.Slots))
	for key, value := range base.Slots {
		merged.Slots[key] = value
	}
	for key, value := range overlay.Slots {
		merged.Slots[key] = value
	}
	if overlay.Defaults.Slot != "" {
		merged.Defaults.Slot = overlay.Defaults.Slot
	}
	if overlay.Defaults.FallbackDir != "" {
		merged.Defaults.FallbackDir = overlay.Defaults.FallbackDir
	}
	if len(overlay.Provides) > 0 {
		merged.Provides = overlay.Provides
	}
	if len(overlay.Targets) > 0 {
		merged.Targets = overlay.Targets
	}
	if overlay.Idiom != "" {
		merged.Idiom = overlay.Idiom
	}
	if len(overlay.Gates) > 0 {
		merged.Gates = overlay.Gates
	}
	if overlay.TemplateStructureVersion != "" {
		merged.TemplateStructureVersion = overlay.TemplateStructureVersion
	}
	if overlay.SlotCoverage != "" {
		merged.SlotCoverage = overlay.SlotCoverage
	}
	return merged
}

// LookupSlot returns the named slot or false. Helper kept here so the
// resolver doesn't reach into the map directly.
func (m Manifest) LookupSlot(name string) (Slot, bool) {
	s, ok := m.Slots[name]
	return s, ok
}
