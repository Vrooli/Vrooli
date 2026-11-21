package intent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// This file extracts the requirements/ registry in its FULL structural
// shape (statuses, tags, criticality, children/depends_on, validation
// entries) — the model registry-structure checks and the traceability
// matrix join over. ExtractRequirementClaims (requirements.go) remains the
// claim-shaped view for the alignment checks; both read the same files, and
// both live here because intent-go is the single parser of requirements/.

// RegistryValidation is one validation entry, structurally complete.
type RegistryValidation struct {
	Type   string `json:"type"`
	Ref    string `json:"ref"`
	Status string `json:"status"`
	Phase  string `json:"phase"`
	Notes  string `json:"notes"`
}

// RegistryRequirement is one requirement record, structurally complete.
type RegistryRequirement struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	Status      string               `json:"status"`
	Criticality string               `json:"criticality"`
	PRDRef      string               `json:"prd_ref"`
	Description string               `json:"description"`
	Tags        []string             `json:"tags"`
	Children    []string             `json:"children"`
	DependsOn   []string             `json:"depends_on"`
	Validations []RegistryValidation `json:"-"`
	// Module is the scenario-relative module file path this record came from.
	Module string `json:"-"`
	// Index is the record's position within its module file (0-based),
	// used to locate records that lack an ID.
	Index int `json:"-"`
}

// HasTag reports whether the requirement carries the tag (case-insensitive).
func (r RegistryRequirement) HasTag(want string) bool {
	for _, t := range r.Tags {
		if strings.EqualFold(strings.TrimSpace(t), want) {
			return true
		}
	}
	return false
}

// RegistryModule is one parsed module.json.
type RegistryModule struct {
	// Path is scenario-relative (e.g. "requirements/01-x/module.json").
	Path         string
	Requirements []RegistryRequirement
}

// Registry is the extracted requirements/ tree.
type Registry struct {
	// Present is true when requirements/index.json exists.
	Present bool
	// IndexPath is scenario-relative ("requirements/index.json").
	IndexPath string
	// Imports as declared in index.json.
	Imports []string
	Modules []RegistryModule
	// ParseErrors records modules that exist but failed to parse — a
	// structural finding, not a load abort.
	ParseErrors []RegistryParseError
}

// RegistryParseError records one unreadable/unparseable registry file.
type RegistryParseError struct {
	Path string
	Err  string
}

// Requirements flattens every module's records in stable order.
func (r Registry) Requirements() []RegistryRequirement {
	var out []RegistryRequirement
	for _, m := range r.Modules {
		out = append(out, m.Requirements...)
	}
	return out
}

type registryIndexFile struct {
	Imports []string `json:"imports"`
}

type registryModuleFile struct {
	Requirements []registryRequirementRecord `json:"requirements"`
}

type registryRequirementRecord struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	Status      string               `json:"status"`
	Criticality string               `json:"criticality"`
	PRDRef      string               `json:"prd_ref"`
	Description string               `json:"description"`
	Tags        []string             `json:"tags"`
	Children    []string             `json:"children"`
	DependsOn   []string             `json:"depends_on"`
	Validation  []RegistryValidation `json:"validation"`
	Validations []RegistryValidation `json:"validations"`
}

// ExtractRequirementsRegistry parses requirements/ into the structural
// model. A missing index yields Present=false with no error; unparseable
// modules land in ParseErrors rather than aborting the load.
func ExtractRequirementsRegistry(scenarioRoot string) (Registry, error) {
	reg := Registry{IndexPath: "requirements/index.json"}
	indexPath := filepath.Join(scenarioRoot, "requirements", "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return reg, nil
		}
		return reg, err
	}
	reg.Present = true
	var index registryIndexFile
	if err := json.Unmarshal(data, &index); err != nil {
		reg.ParseErrors = append(reg.ParseErrors, RegistryParseError{Path: reg.IndexPath, Err: err.Error()})
		return reg, nil
	}
	reg.Imports = index.Imports

	// Walk every module.json under requirements/ (not just declared imports)
	// so orphaned module files are still visible to checks; declared-but-
	// missing imports surface as parse errors.
	seen := make(map[string]struct{})
	for _, imp := range index.Imports {
		rel := filepath.ToSlash(filepath.Join("requirements", imp))
		seen[rel] = struct{}{}
		module, err := readRegistryModule(scenarioRoot, rel)
		if err != nil {
			reg.ParseErrors = append(reg.ParseErrors, RegistryParseError{Path: rel, Err: err.Error()})
			continue
		}
		reg.Modules = append(reg.Modules, module)
	}
	_ = filepath.WalkDir(filepath.Join(scenarioRoot, "requirements"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || entry.Name() != "module.json" {
			return nil
		}
		rel := relPath(scenarioRoot, path)
		if _, ok := seen[rel]; ok {
			return nil
		}
		module, err := readRegistryModule(scenarioRoot, rel)
		if err != nil {
			reg.ParseErrors = append(reg.ParseErrors, RegistryParseError{Path: rel, Err: err.Error()})
			return nil
		}
		reg.Modules = append(reg.Modules, module)
		return nil
	})
	sort.Slice(reg.Modules, func(i, j int) bool { return reg.Modules[i].Path < reg.Modules[j].Path })
	return reg, nil
}

func readRegistryModule(scenarioRoot, rel string) (RegistryModule, error) {
	data, err := os.ReadFile(filepath.Join(scenarioRoot, filepath.FromSlash(rel)))
	if err != nil {
		return RegistryModule{}, fmt.Errorf("read module: %w", err)
	}
	var file registryModuleFile
	if err := json.Unmarshal(data, &file); err != nil {
		return RegistryModule{}, fmt.Errorf("parse module: %w", err)
	}
	module := RegistryModule{Path: rel}
	for i, rec := range file.Requirements {
		validations := rec.Validation
		if len(validations) == 0 {
			validations = rec.Validations
		}
		module.Requirements = append(module.Requirements, RegistryRequirement{
			ID:          strings.TrimSpace(rec.ID),
			Title:       strings.TrimSpace(rec.Title),
			Status:      strings.TrimSpace(rec.Status),
			Criticality: strings.TrimSpace(rec.Criticality),
			PRDRef:      strings.TrimSpace(rec.PRDRef),
			Description: rec.Description,
			Tags:        rec.Tags,
			Children:    rec.Children,
			DependsOn:   rec.DependsOn,
			Validations: validations,
			Module:      rel,
			Index:       i,
		})
	}
	return module, nil
}
