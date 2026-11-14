package portability

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"nutrition-planner/internal/recipe"
)

// LegacyImport is the deliberately named adapter for the pre-Daily prototype.
// It preserves the old facts as reviewable content and never upgrades them to
// current nutrition, allergy, purchase, or consumption evidence.
type LegacyImport struct {
	Recipes    []Recipe
	Warnings   []string
	WeekPlan   []string
	Completed  []int
	Checked    []string
	Profile    LegacyProfile
	Provenance string
}

type LegacyProfile struct {
	Diet, CustomName string
	ExcludedGroups   []string
	Allergens        []string
	Avoid            []string
	Appliances       []string
	SetupComplete    bool
}

type legacyRecipe struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Notes      string          `json:"notes"`
	Source     string          `json:"source"`
	Reviewed   bool            `json:"reviewed"`
	Sample     bool            `json:"sample"`
	Servings   json.RawMessage `json:"servings"`
	Items      []legacyItem    `json:"items"`
	Steps      []legacyStep    `json:"steps"`
	Groups     []string        `json:"groups"`
	Allergens  []string        `json:"allergens"`
	Appliances []string        `json:"appliances"`
	Calories   json.RawMessage `json:"calories"`
	Protein    json.RawMessage `json:"protein"`
	Cost       json.RawMessage `json:"cost"`
	Minutes    json.RawMessage `json:"minutes"`
}

type legacyItem struct {
	ID, Name, Unit, StockKey string
	Quantity                 json.RawMessage `json:"quantity"`
}
type legacyStep struct {
	ID, Title, Detail string
	Minutes           json.RawMessage `json:"minutes"`
	Inputs            []string        `json:"inputs"`
	After             []string        `json:"after"`
}
type legacyWorkspace struct {
	SchemaVersion int            `json:"schemaVersion"`
	Profile       LegacyProfile  `json:"profile"`
	Recipes       []legacyRecipe `json:"recipes"`
	Plan          []*string      `json:"plan"`
	Completed     []int          `json:"completed"`
	Checked       []string       `json:"checked"`
}

// ImportLegacy accepts exactly the prototype's documented single/array/recipes
// and schemaVersion-1 workspace shapes. It rejects unidentified versioned data.
func ImportLegacy(data []byte) (LegacyImport, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return LegacyImport{}, errors.New("legacy import is empty")
	}
	var root map[string]json.RawMessage
	if data[0] == '{' {
		if err := json.Unmarshal(data, &root); err != nil {
			return LegacyImport{}, fmt.Errorf("legacy import: %w", err)
		}
	}
	result := LegacyImport{Provenance: "legacy_import"}
	if raw, ok := root["schemaVersion"]; ok {
		var version int
		if err := json.Unmarshal(raw, &version); err != nil || version != 1 {
			return LegacyImport{}, errors.New("unsupported legacy schema version")
		}
		var workspace legacyWorkspace
		if err := json.Unmarshal(data, &workspace); err != nil {
			return LegacyImport{}, fmt.Errorf("legacy workspace: %w", err)
		}
		result.Profile, result.WeekPlan, result.Completed, result.Checked = workspace.Profile, flattenPlan(workspace.Plan), workspace.Completed, workspace.Checked
		result.Recipes = make([]Recipe, 0, len(workspace.Recipes))
		for _, item := range workspace.Recipes {
			converted, warnings, err := convertLegacyRecipe(item)
			if err != nil {
				return LegacyImport{}, err
			}
			result.Recipes = append(result.Recipes, converted)
			result.Warnings = append(result.Warnings, warnings...)
		}
		return result, nil
	}
	var rawRecipes []json.RawMessage
	if raw, ok := root["recipes"]; ok {
		if err := json.Unmarshal(raw, &rawRecipes); err != nil {
			return LegacyImport{}, errors.New("legacy recipes must be an array")
		}
	} else if len(data) > 0 && data[0] == '[' {
		if err := json.Unmarshal(data, &rawRecipes); err != nil {
			return LegacyImport{}, fmt.Errorf("legacy recipes: %w", err)
		}
	} else {
		rawRecipes = []json.RawMessage{data}
	}
	for _, raw := range rawRecipes {
		var item legacyRecipe
		if err := json.Unmarshal(raw, &item); err != nil {
			return LegacyImport{}, fmt.Errorf("legacy recipe: %w", err)
		}
		converted, warnings, err := convertLegacyRecipe(item)
		if err != nil {
			return LegacyImport{}, err
		}
		result.Recipes = append(result.Recipes, converted)
		result.Warnings = append(result.Warnings, warnings...)
	}
	return result, nil
}

func convertLegacyRecipe(item legacyRecipe) (Recipe, []string, error) {
	if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Name) == "" {
		return Recipe{}, nil, errors.New("legacy recipe requires id and name")
	}
	available := make(map[string]bool, len(item.Items))
	for _, ingredient := range item.Items {
		if ingredient.ID != "" {
			available[ingredient.ID] = true
		}
	}
	method := recipe.Method{ID: "legacy-method", Name: "Legacy steps"}
	warnings := []string{"legacy nutrition, cost, effort, allergen, and reviewed values remain assertions requiring review"}
	if item.Sample {
		warnings = append(warnings, "sample recipe remains illustrative")
	}
	for index, step := range item.Steps {
		id := step.ID
		if id == "" {
			id = fmt.Sprintf("legacy-step-%d", index+1)
		}
		method.Steps = append(method.Steps, recipe.MethodStep{ID: id, Instruction: strings.TrimSpace(step.Title + " " + step.Detail), DependsOn: append([]string(nil), step.After...), Inputs: append([]string(nil), step.Inputs...)})
	}
	if err := recipe.ValidateMethod(method, available); err != nil {
		return Recipe{}, warnings, fmt.Errorf("legacy recipe %q method: %w", item.ID, err)
	}
	if len(item.Steps) == 0 {
		method.Steps = nil
	}
	allergenEvidence := make(map[string]string, len(item.Allergens))
	for _, allergen := range item.Allergens {
		allergenEvidence[allergen] = "legacy_assertion"
	}
	if item.Reviewed {
		warnings = append(warnings, "reviewed is preserved as a legacy assertion, not modern clearance")
	}
	return Recipe{ID: item.ID, Revision: 1, Name: item.Name, Notes: item.Notes, SourceURL: item.Source, SourceType: "legacy_import", OriginalText: item.Notes, Status: "draft", Groups: item.Groups, RequiredAppliances: item.Appliances, AllergenEvidence: allergenEvidence, Methods: []recipe.Method{method}}, warnings, nil
}

func flattenPlan(plan []*string) []string {
	out := make([]string, len(plan))
	for i, value := range plan {
		if value != nil {
			out[i] = *value
		}
	}
	return out
}

func decimalText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return "unknown"
	}
	if number, err := strconv.ParseFloat(string(raw), 64); err == nil {
		return strconv.FormatFloat(number, 'f', -1, 64)
	}
	return "unknown"
}
