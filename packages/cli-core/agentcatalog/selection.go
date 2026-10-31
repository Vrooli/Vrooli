package agentcatalog

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnknownCodingRole     = errors.New("unknown coding role")
	ErrUnsupportedModel      = errors.New("model is not declared for this role")
	ErrUnsupportedSource     = errors.New("source is not declared for this role")
	ErrSourceSelectionNeeded = errors.New("source selection is required for this model")
	ErrModelSelectionNeeded  = errors.New("model selection is required for this source")
	ErrInvalidRoleModel      = errors.New("role model declaration is invalid")
)

// RoleSelectionIntent is a runner-neutral request to choose a resource-declared
// role model. Empty Model and Source preserve the role's resource-owned default.
type RoleSelectionIntent struct {
	Role   string
	Model  string
	Source string
}

// RoleModelChoice is one concrete model/source choice declared by a resource.
// Source is intentionally opaque: callers must not derive it from the model
// name, canonical alias, or billing mode.
type RoleModelChoice struct {
	Model          string  `json:"model"`
	CanonicalModel string  `json:"canonical_model,omitempty"`
	Provider       string  `json:"provider,omitempty"`
	Source         string  `json:"source,omitempty"`
	Effort         *string `json:"effort,omitempty"`
}

// ResolvedRoleSelection is the exact resource-owned answer for one intent.
// SourceStatus distinguishes a declared source from an ambiguous or unknown
// source without inventing a provider choice.
type ResolvedRoleSelection struct {
	Runner         string  `json:"runner"`
	Role           string  `json:"role"`
	Model          string  `json:"model"`
	CanonicalModel string  `json:"canonical_model,omitempty"`
	Provider       string  `json:"provider,omitempty"`
	Source         string  `json:"source,omitempty"`
	SourceStatus   string  `json:"source_status"`
	Effort         *string `json:"effort,omitempty"`
	IsDefault      bool    `json:"is_default"`
}

// RoleModelChoices returns only the choices declared for one role, excluding
// resource-wide and role-specific denied models. A legacy role with no Models
// list exposes only its declared primary model, with source evidence unknown.
func RoleModelChoices(catalog CodingRoleCatalog, roleName string) ([]RoleModelChoice, error) {
	roleName = strings.TrimSpace(roleName)
	role, ok := catalog.Roles[roleName]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownCodingRole, roleName)
	}
	choices := make([]RoleModelChoice, 0, len(role.Models))
	if len(role.Models) == 0 {
		if modelAllowed(catalog, roleName, role.Model, role.CanonicalModel) {
			choice := choiceFor(catalog, role, role.Model, "", nil)
			choices = append(choices, choice)
		}
		return choices, nil
	}
	for _, declared := range role.Models {
		if strings.TrimSpace(declared.Model) == "" || strings.TrimSpace(declared.Model) != declared.Model || strings.TrimSpace(declared.Source) != declared.Source {
			return nil, fmt.Errorf("%w: role %q contains an empty or untrimmed model/source", ErrInvalidRoleModel, roleName)
		}
		if !modelAllowed(catalog, roleName, declared.Model, canonicalModel(catalog, role, declared.Model)) {
			continue
		}
		choices = append(choices, choiceFor(catalog, role, declared.Model, declared.Source, declared.Effort))
	}
	return choices, nil
}

// ResolveRoleSelection resolves an exact role/model/source combination. An
// explicit unknown choice fails closed; a missing source remains visibly
// unknown or ambiguous rather than being inferred from another field.
func ResolveRoleSelection(catalog CodingRoleCatalog, intent RoleSelectionIntent) (ResolvedRoleSelection, error) {
	roleName := strings.TrimSpace(intent.Role)
	role, ok := catalog.Roles[roleName]
	if !ok {
		return ResolvedRoleSelection{}, fmt.Errorf("%w: %q", ErrUnknownCodingRole, roleName)
	}
	explicitModel := strings.TrimSpace(intent.Model)
	explicitSource := strings.TrimSpace(intent.Source)
	choices, err := RoleModelChoices(catalog, roleName)
	if err != nil {
		return ResolvedRoleSelection{}, err
	}
	model := explicitModel
	if model == "" && explicitSource == "" {
		model = role.Model
	}
	matches := make([]RoleModelChoice, 0, len(choices))
	for _, choice := range choices {
		if (model == "" || choice.Model == model) && (explicitSource == "" || choice.Source == explicitSource) {
			matches = append(matches, choice)
		}
	}
	if len(matches) == 0 {
		if explicitSource != "" {
			return ResolvedRoleSelection{}, fmt.Errorf("%w: role %q model %q source %q", ErrUnsupportedSource, roleName, model, explicitSource)
		}
		return ResolvedRoleSelection{}, fmt.Errorf("%w: role %q model %q", ErrUnsupportedModel, roleName, model)
	}
	if model == "" {
		models := distinctModels(matches)
		if len(models) > 1 {
			return ResolvedRoleSelection{}, fmt.Errorf("%w: role %q source %q", ErrModelSelectionNeeded, roleName, explicitSource)
		}
		model = models[0]
	}
	if explicitSource == "" {
		sources := distinctSources(matches)
		if len(sources) > 1 && explicitModel != "" {
			return ResolvedRoleSelection{}, fmt.Errorf("%w: role %q model %q", ErrSourceSelectionNeeded, roleName, model)
		}
		if len(sources) == 1 && sources[0] != "" {
			for _, choice := range matches {
				if choice.Source == sources[0] {
					matches = []RoleModelChoice{choice}
					break
				}
			}
		}
	}

	choice := matches[0]
	sourceStatus := "unknown"
	if explicitSource == "" && len(distinctSources(matches)) > 1 {
		choice.Source = ""
		choice.Provider = ""
		sourceStatus = "ambiguous"
	} else if choice.Source != "" {
		sourceStatus = "declared"
	}
	return ResolvedRoleSelection{
		Runner:         catalog.Runner,
		Role:           roleName,
		Model:          choice.Model,
		CanonicalModel: choice.CanonicalModel,
		Provider:       choice.Provider,
		Source:         choice.Source,
		SourceStatus:   sourceStatus,
		Effort:         copyString(choice.Effort),
		IsDefault:      explicitModel == "" && explicitSource == "",
	}, nil
}

func choiceFor(catalog CodingRoleCatalog, role CodingRole, model, source string, effort *string) RoleModelChoice {
	canonical := canonicalModel(catalog, role, model)
	choice := RoleModelChoice{Model: model, CanonicalModel: canonical, Source: source, Effort: copyString(effort)}
	if alias, ok := catalog.ModelAliases[model]; ok {
		choice.Provider = alias.Provider
	}
	return choice
}

func canonicalModel(catalog CodingRoleCatalog, role CodingRole, model string) string {
	if alias, ok := catalog.ModelAliases[model]; ok && strings.TrimSpace(alias.CanonicalModel) != "" {
		return alias.CanonicalModel
	}
	if model == role.Model {
		return role.CanonicalModel
	}
	return ""
}

func modelAllowed(catalog CodingRoleCatalog, role, model, canonical string) bool {
	keys := map[string]struct{}{strings.ToLower(model): {}}
	if strings.TrimSpace(canonical) != "" {
		keys[strings.ToLower(canonical)] = struct{}{}
	}
	for _, excluded := range catalog.ExcludedModels {
		if _, denied := keys[strings.ToLower(excluded)]; denied {
			return false
		}
	}
	for restricted, roles := range catalog.RestrictedModels {
		if _, applies := keys[strings.ToLower(restricted)]; !applies {
			continue
		}
		allowed := false
		for _, allowedRole := range roles {
			if allowedRole == role {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}

func distinctSources(choices []RoleModelChoice) []string {
	seen := make(map[string]struct{}, len(choices))
	sources := make([]string, 0, len(choices))
	for _, choice := range choices {
		if _, exists := seen[choice.Source]; exists {
			continue
		}
		seen[choice.Source] = struct{}{}
		sources = append(sources, choice.Source)
	}
	return sources
}

func distinctModels(choices []RoleModelChoice) []string {
	seen := make(map[string]struct{}, len(choices))
	models := make([]string, 0, len(choices))
	for _, choice := range choices {
		if _, exists := seen[choice.Model]; exists {
			continue
		}
		seen[choice.Model] = struct{}{}
		models = append(models, choice.Model)
	}
	return models
}

func copyString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
