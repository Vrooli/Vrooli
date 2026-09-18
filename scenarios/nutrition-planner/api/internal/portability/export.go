package portability

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"nutrition-planner/internal/recipe"
)

type Export struct {
	Format        string       `json:"format"`
	SchemaVersion int          `json:"schemaVersion"`
	ExportID      string       `json:"exportId"`
	CreatedAt     string       `json:"createdAt"`
	Scope         Scope        `json:"scope"`
	Manifest      Manifest     `json:"manifest"`
	Recipes       []Recipe     `json:"recipes"`
	Records       []Record     `json:"records,omitempty"`
	Attachments   []Attachment `json:"attachments,omitempty"`
}
type Manifest struct {
	RecordKinds         []string `json:"recordKinds"`
	RecordCount         int      `json:"recordCount"`
	AttachmentsIncluded bool     `json:"attachmentsIncluded"`
	Omissions           []string `json:"omissions"`
}
type Scope struct {
	Kind      string   `json:"kind"`
	RecipeIDs []string `json:"recipeIds,omitempty"`
}
type Record struct {
	Kind     string          `json:"kind"`
	ID       string          `json:"id"`
	Revision int64           `json:"revision"`
	Data     json.RawMessage `json:"data"`
}
type Attachment struct {
	ID         string `json:"id"`
	MediaType  string `json:"mediaType"`
	ByteLength int64  `json:"byteLength"`
	SHA256     string `json:"sha256"`
	Omission   string `json:"omission,omitempty"`
}

type WorkspaceSnapshot struct {
	ID, Name     string
	Revision     int64
	Recipes      []recipe.Recipe
	PlanRevision int64
	PlanJSON     string
}

// Workspace emits a truthful R0 workspace backup. It includes only domains
// currently supported by the exporter and names every omitted R1/operational
// domain rather than implying a complete restore.
func Workspace(snapshot WorkspaceSnapshot) ([]byte, error) {
	if snapshot.ID == "" {
		return nil, errors.New("workspace export requires an id")
	}
	out := Export{Format: "daily.workspace", SchemaVersion: 2, ExportID: "workspace-" + snapshot.ID, Scope: Scope{Kind: "workspace_backup"}, Manifest: Manifest{RecordKinds: []string{"workspace", "recipe", "recipe_revision", "recipe_method", "recipe_step", "plan"}, AttachmentsIncluded: true, Omissions: []string{"profile", "catalog", "nutrition_targets", "supplements", "inventory", "prices", "feedback", "jobs", "provider_credentials", "authentication", "entitlements"}}}
	workspaceData, _ := json.Marshal(map[string]any{"name": snapshot.Name, "revision": snapshot.Revision})
	out.Records = append(out.Records, Record{Kind: "workspace", ID: snapshot.ID, Revision: snapshot.Revision, Data: workspaceData})
	for _, item := range snapshot.Recipes {
		content, err := Recipes([]recipe.Recipe{item})
		if err != nil {
			return nil, err
		}
		var nested Export
		if err := json.Unmarshal(content, &nested); err != nil {
			return nil, err
		}
		out.Records = append(out.Records, nested.Records...)
	}
	if snapshot.PlanJSON != "" {
		data, _ := json.Marshal(map[string]any{"revision": snapshot.PlanRevision, "planJson": snapshot.PlanJSON})
		out.Records = append(out.Records, Record{Kind: "plan", ID: snapshot.ID, Revision: snapshot.PlanRevision, Data: data})
	}
	out.Manifest.RecordCount = len(out.Records)
	return json.Marshal(out)
}

// ImportWorkspace validates a workspace backup before any repository mutation.
// The returned envelope is still staged; applying it belongs to a transaction
// that can enforce the caller's workspace authority and revision barrier.
func ImportWorkspace(data []byte) (Export, error) {
	if len(data) > 5*1024*1024 {
		return Export{}, errors.New("workspace export exceeds 5 MB limit")
	}
	var envelope Export
	if err := json.Unmarshal(data, &envelope); err != nil {
		return Export{}, fmt.Errorf("invalid workspace export: %w", err)
	}
	if envelope.Format != "daily.workspace" || envelope.SchemaVersion != 2 {
		return Export{}, errors.New("unsupported workspace export format or schema version")
	}
	if envelope.Manifest.RecordCount != len(envelope.Records) {
		return Export{}, errors.New("workspace export record count does not match manifest")
	}
	allowed := map[string]bool{"workspace": true, "recipe": true, "recipe_revision": true, "plan": true}
	workspaceCount := 0
	for _, record := range envelope.Records {
		if !allowed[record.Kind] {
			return Export{}, fmt.Errorf("workspace export contains unsupported record kind %q", record.Kind)
		}
		if record.ID == "" || len(record.Data) == 0 {
			return Export{}, fmt.Errorf("workspace export contains an invalid %s record", record.Kind)
		}
		if record.Kind == "workspace" {
			workspaceCount++
		}
	}
	if workspaceCount != 1 {
		return Export{}, errors.New("workspace export must contain exactly one workspace record")
	}
	return envelope, nil
}

type Recipe struct {
	ID                 string            `json:"id"`
	Revision           int64             `json:"revision"`
	Name               string            `json:"name"`
	Notes              string            `json:"notes,omitempty"`
	SourceURL          string            `json:"sourceUrl,omitempty"`
	SourceType         string            `json:"sourceType,omitempty"`
	OriginalText       string            `json:"originalText,omitempty"`
	Status             string            `json:"status"`
	Groups             []string          `json:"groups,omitempty"`
	RequiredAppliances []string          `json:"requiredAppliances,omitempty"`
	AllergenEvidence   map[string]string `json:"allergenEvidence,omitempty"`
	Methods            []recipe.Method   `json:"methods,omitempty"`
}

func Recipes(items []recipe.Recipe) ([]byte, error) {
	out := Export{Format: "daily.recipes", SchemaVersion: 2, Manifest: Manifest{
		RecordKinds: []string{"recipe", "recipe_revision", "recipe_method", "recipe_step"}, AttachmentsIncluded: true,
		Omissions: []string{"nutrition", "cost", "ingredients"},
	}, Scope: Scope{Kind: "recipe_collection"}, Recipes: make([]Recipe, 0, len(items)), Records: make([]Record, 0, len(items)*2)}
	for _, r := range items {
		portable := Recipe{ID: r.ID, Revision: r.Revision, Name: r.Name, Notes: r.Notes, SourceURL: r.SourceURL, SourceType: r.SourceType, OriginalText: r.OriginalText, Status: r.Status, Groups: r.Groups, RequiredAppliances: r.RequiredAppliances, AllergenEvidence: r.AllergenEvidence, Methods: r.Methods}
		out.Recipes = append(out.Recipes, portable)
		out.Scope.RecipeIDs = append(out.Scope.RecipeIDs, r.ID)
		identity, _ := json.Marshal(map[string]any{"currentRevision": r.Revision, "status": r.Status})
		revision, _ := json.Marshal(portable)
		out.Records = append(out.Records, Record{Kind: "recipe", ID: r.ID, Revision: r.Revision, Data: identity}, Record{Kind: "recipe_revision", ID: r.ID, Revision: r.Revision, Data: revision})
	}
	out.Manifest.RecordCount = len(out.Records)
	out.ExportID = "recipe-collection-" + fmt.Sprint(len(items))
	return json.Marshal(out)
}

// ImportRecipes validates the native recipe envelope before any caller applies
// it. It deliberately returns a staged value and performs no persistence.
func ImportRecipes(data []byte) (Export, error) {
	if len(data) > 2*1024*1024 {
		return Export{}, errors.New("recipe export exceeds 2 MB limit")
	}
	var envelope Export
	if err := json.Unmarshal(data, &envelope); err != nil {
		return Export{}, fmt.Errorf("invalid recipe export: %w", err)
	}
	if envelope.Format != "daily.recipes" || envelope.SchemaVersion != 2 {
		return Export{}, errors.New("unsupported recipe export format or schema version")
	}
	if len(envelope.Recipes) == 0 && len(envelope.Records) > 0 {
		for _, record := range envelope.Records {
			if record.Kind != "recipe_revision" {
				continue
			}
			var item Recipe
			if err := json.Unmarshal(record.Data, &item); err != nil {
				return Export{}, fmt.Errorf("record %s: %w", record.ID, err)
			}
			envelope.Recipes = append(envelope.Recipes, item)
		}
	}
	if len(envelope.Recipes) > 200 {
		return Export{}, errors.New("recipe export exceeds 200 recipe limit")
	}
	seen := map[string]bool{}
	for _, item := range envelope.Recipes {
		if item.ID == "" || item.Revision < 1 || item.Name == "" {
			return Export{}, errors.New("recipe export contains an invalid recipe")
		}
		key := fmt.Sprintf("%s:%d", item.ID, item.Revision)
		if seen[key] {
			return Export{}, fmt.Errorf("duplicate recipe revision %s", key)
		}
		seen[key] = true
	}
	return envelope, nil
}

type GroceryRow struct {
	Key, Label, Need, Stock, Missing, PackageCount, Price string
	Checked                                               bool
}

// GroceriesCSV emits RFC 4180 CSV and neutralizes spreadsheet formulas. Unknown
// values remain the literal word "unknown" rather than becoming zero/blank.
func GroceriesCSV(rows []GroceryRow) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"key", "label", "need", "stock", "missing", "package_count", "price", "checked"}); err != nil {
		return nil, err
	}
	for _, row := range rows {
		values := []string{safeCell(row.Key), safeCell(row.Label), safeCell(row.Need), safeCell(row.Stock), safeCell(row.Missing), safeCell(row.PackageCount), safeCell(row.Price), fmt.Sprintf("%t", row.Checked)}
		if err := writer.Write(values); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func safeCell(value string) string {
	if len(value) > 0 && (value[0] == '=' || value[0] == '+' || value[0] == '-' || value[0] == '@') {
		return "'" + value
	}
	return value
}
