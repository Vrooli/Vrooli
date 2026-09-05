package intent

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RequirementsExtractor extracts requirement claims from requirements/*/module.json.
type RequirementsExtractor interface {
	ExtractRequirementClaims(scenarioRoot string) ([]CapabilityClaim, error)
}

type FileRequirementsExtractor struct{}

type requirementModule struct {
	Requirements []requirementRecord `json:"requirements"`
}

type requirementRecord struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	PRDRef      string             `json:"prd_ref"`
	Description string             `json:"description"`
	Validation  []validationRecord `json:"validation"`
	Validations []validationRecord `json:"validations"`
}

type validationRecord struct {
	Type string `json:"type"`
	Ref  string `json:"ref"`
}

func (FileRequirementsExtractor) ExtractRequirementClaims(scenarioRoot string) ([]CapabilityClaim, error) {
	requirementsDir := filepath.Join(scenarioRoot, "requirements")
	var claims []CapabilityClaim
	err := filepath.WalkDir(requirementsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "module.json" {
			return nil
		}
		moduleClaims, err := parseRequirementModule(scenarioRoot, path)
		if err != nil {
			return err
		}
		claims = append(claims, moduleClaims...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(claims, func(i, j int) bool {
		if claims[i].Anchor == claims[j].Anchor {
			return claims[i].ID < claims[j].ID
		}
		return claims[i].Anchor < claims[j].Anchor
	})
	return claims, nil
}

func parseRequirementModule(scenarioRoot, path string) ([]CapabilityClaim, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var module requirementModule
	if err := json.Unmarshal(data, &module); err != nil {
		return nil, err
	}
	rel := relPath(scenarioRoot, path)
	claims := make([]CapabilityClaim, 0, len(module.Requirements))
	for _, req := range module.Requirements {
		id := strings.TrimSpace(req.ID)
		if id == "" {
			continue
		}
		validations := req.Validation
		if len(validations) == 0 {
			validations = req.Validations
		}
		refs := make([]Ref, 0, len(validations)+1)
		if prdRef := strings.TrimSpace(req.PRDRef); prdRef != "" {
			refs = append(refs, Ref{Raw: prdRef, Path: prdRef, Kind: RefDoc})
		}
		for _, validation := range validations {
			raw := strings.TrimSpace(validation.Ref)
			if raw == "" {
				continue
			}
			refs = append(refs, NormalizeRef(raw, validation.Type))
		}
		claims = append(claims, CapabilityClaim{
			ID:         id,
			Altitude:   Requirement,
			Text:       strings.TrimSpace(req.Title + " " + req.Description),
			Anchor:     rel,
			Refs:       refs,
			Provenance: "requirements",
		})
	}
	return claims, nil
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
