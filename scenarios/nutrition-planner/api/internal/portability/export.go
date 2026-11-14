package portability

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/nutrients"
	"nutrition-planner/internal/nutrition"
	"nutrition-planner/internal/profile"
	"nutrition-planner/internal/recipe"
	"nutrition-planner/internal/shopping"
	"nutrition-planner/internal/supplement"
	"nutrition-planner/internal/units"
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
	ID, Name            string
	Revision            int64
	Recipes             []recipe.Recipe
	RecipeHistory       []recipe.Recipe
	PlanRevision        int64
	PlanJSON            string
	Shopping            shopping.PersistedState
	NutritionTargets    []nutrition.Target
	Profile             *profile.Profile
	IntakeEvents        []nutrition.IntakeEvent
	SupplementSchedules []supplement.Schedule
}

type intakeEventRecord struct {
	WorkspaceID    string `json:"workspaceId"`
	EventID        string `json:"eventId"`
	Date           string `json:"date"`
	RecipeID       string `json:"recipeId,omitempty"`
	RecipeRevision int64  `json:"recipeRevision,omitempty"`
	NutrientID     string `json:"nutrientId"`
	Amount         string `json:"amount"`
	Unit           string `json:"unit"`
	Reason         string `json:"reason,omitempty"`
	CorrectionOf   string `json:"correctionOf,omitempty"`
	RecordedAt     string `json:"recordedAt"`
}

type profileRecord struct {
	WorkspaceID    string   `json:"workspaceId"`
	Revision       int64    `json:"revision"`
	Preset         string   `json:"preset"`
	PresetVersion  int64    `json:"presetVersion"`
	ActiveRules    []string `json:"activeRules"`
	ExcludedGroups []string `json:"excludedGroups"`
	Allergies      []string `json:"allergies"`
	Appliances     []string `json:"appliances"`
	CostWeight     float64  `json:"costWeight"`
	EffortWeight   float64  `json:"effortWeight"`
	VarietyWeight  float64  `json:"varietyWeight"`
	DraftJSON      string   `json:"draftJson"`
}

type supplementScheduleRecord struct {
	WorkspaceID     string `json:"workspaceId"`
	CurrentRevision int64  `json:"currentRevision"`
}

type supplementScheduleRevisionRecord struct {
	WorkspaceID       string `json:"workspaceId"`
	ProductRevisionID string `json:"productRevisionId"`
	Dose              string `json:"dose"`
	DoseUnit          string `json:"doseUnit"`
	Weekdays          []int  `json:"weekdays"`
	StartDate         string `json:"startDate"`
	EndDate           string `json:"endDate,omitempty"`
	Paused            bool   `json:"paused"`
	Confirmed         bool   `json:"confirmed"`
	CreatedAt         string `json:"createdAt"`
}

func portableSupplementSchedule(value supplement.Schedule) supplementScheduleRevisionRecord {
	return supplementScheduleRevisionRecord{WorkspaceID: value.WorkspaceID, ProductRevisionID: value.ProductRevisionID, Dose: value.Dose.String(), DoseUnit: value.DoseUnit, Weekdays: value.Weekdays, StartDate: value.StartDate, EndDate: value.EndDate, Paused: value.Paused, Confirmed: value.Confirmed, CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339Nano)}
}

func portableProfile(value profile.Profile) profileRecord {
	return profileRecord{WorkspaceID: value.WorkspaceID, Revision: value.Revision, Preset: value.Preset, PresetVersion: value.PresetVersion, ActiveRules: value.ActiveRules, ExcludedGroups: value.ExcludedGroups, Allergies: value.Allergies, Appliances: value.Appliances, CostWeight: value.CostWeight, EffortWeight: value.EffortWeight, VarietyWeight: value.VarietyWeight, DraftJSON: value.DraftJSON}
}

func (value profileRecord) domain() profile.Profile {
	return profile.Profile{WorkspaceID: value.WorkspaceID, Revision: value.Revision, Preset: value.Preset, PresetVersion: value.PresetVersion, ActiveRules: value.ActiveRules, ExcludedGroups: value.ExcludedGroups, Allergies: value.Allergies, Appliances: value.Appliances, CostWeight: value.CostWeight, EffortWeight: value.EffortWeight, VarietyWeight: value.VarietyWeight, DraftJSON: value.DraftJSON}
}

// Workspace emits a truthful R0 workspace backup. It includes only domains
// currently supported by the exporter and names every omitted R1/operational
// domain rather than implying a complete restore.
func Workspace(snapshot WorkspaceSnapshot) ([]byte, error) {
	if snapshot.ID == "" {
		return nil, errors.New("workspace export requires an id")
	}
	out := Export{Format: "daily.workspace", SchemaVersion: 2, ExportID: "workspace-" + snapshot.ID, Scope: Scope{Kind: "workspace_backup"}, Manifest: Manifest{RecordKinds: []string{"workspace", "recipe", "recipe_revision", "plan", "shopping_state", "nutrition_target", "profile", "intake_event", "supplement_schedule", "supplement_schedule_revision"}, AttachmentsIncluded: false, Omissions: []string{"catalog", "inventory", "prices", "feedback", "jobs", "provider_credentials", "authentication", "entitlements"}}}
	workspaceData, _ := json.Marshal(map[string]any{"name": snapshot.Name, "revision": snapshot.Revision})
	out.Records = append(out.Records, Record{Kind: "workspace", ID: snapshot.ID, Revision: snapshot.Revision, Data: workspaceData})
	content, err := RecipesWithHistory(snapshot.Recipes, snapshot.RecipeHistory)
	if err != nil {
		return nil, err
	}
	var nested Export
	if err := json.Unmarshal(content, &nested); err != nil {
		return nil, err
	}
	out.Records = append(out.Records, nested.Records...)
	if snapshot.PlanJSON != "" {
		data, _ := json.Marshal(map[string]any{"revision": snapshot.PlanRevision, "planJson": snapshot.PlanJSON})
		out.Records = append(out.Records, Record{Kind: "plan", ID: snapshot.ID, Revision: snapshot.PlanRevision, Data: data})
	}
	if snapshot.Shopping.Checks == nil {
		snapshot.Shopping.Checks = map[string]bool{}
	}
	if snapshot.Shopping.HaveThis == nil {
		snapshot.Shopping.HaveThis = map[string]bool{}
	}
	if snapshot.Shopping.Reviews == nil {
		snapshot.Shopping.Reviews = []shopping.PersistedReview{}
	}
	shoppingData, err := json.Marshal(snapshot.Shopping)
	if err != nil {
		return nil, err
	}
	out.Records = append(out.Records, Record{Kind: "shopping_state", ID: snapshot.ID, Data: shoppingData})
	for _, target := range snapshot.NutritionTargets {
		data, err := json.Marshal(target)
		if err != nil {
			return nil, err
		}
		out.Records = append(out.Records, Record{Kind: "nutrition_target", ID: target.ID, Revision: target.Revision, Data: data})
	}
	if snapshot.Profile != nil {
		p := *snapshot.Profile
		if p.WorkspaceID != snapshot.ID {
			return nil, errors.New("profile export workspace does not match owning workspace")
		}
		data, err := json.Marshal(portableProfile(p))
		if err != nil {
			return nil, err
		}
		out.Records = append(out.Records, Record{Kind: "profile", ID: snapshot.ID, Revision: p.Revision, Data: data})
	}
	for _, event := range snapshot.IntakeEvents {
		if event.WorkspaceID != snapshot.ID {
			return nil, errors.New("intake event workspace does not match owning workspace")
		}
		record := intakeEventRecord{WorkspaceID: event.WorkspaceID, EventID: event.ID, Date: event.Date, RecipeID: event.RecipeID, RecipeRevision: event.RecipeRevision, NutrientID: event.NutrientID, Amount: event.Amount.String(), Unit: event.Unit, Reason: event.Reason, CorrectionOf: event.CorrectionOf, RecordedAt: event.RecordedAt().Format(time.RFC3339Nano)}
		data, err := json.Marshal(record)
		if err != nil {
			return nil, err
		}
		out.Records = append(out.Records, Record{Kind: "intake_event", ID: event.ID, Data: data})
	}
	currentSchedules := map[string]supplement.Schedule{}
	for _, schedule := range snapshot.SupplementSchedules {
		if schedule.WorkspaceID != snapshot.ID {
			return nil, errors.New("supplement schedule export workspace does not match owning workspace")
		}
		if schedule.ID == "" || schedule.Revision < 1 {
			return nil, errors.New("supplement schedule export requires stable identity and positive revision")
		}
		if prior, ok := currentSchedules[schedule.ID]; !ok || schedule.Revision > prior.Revision {
			currentSchedules[schedule.ID] = schedule
		}
		data, err := json.Marshal(portableSupplementSchedule(schedule))
		if err != nil {
			return nil, err
		}
		out.Records = append(out.Records, Record{Kind: "supplement_schedule_revision", ID: schedule.ID, Revision: schedule.Revision, Data: data})
	}
	ids := make([]string, 0, len(currentSchedules))
	for id := range currentSchedules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		current := currentSchedules[id]
		data, err := json.Marshal(supplementScheduleRecord{WorkspaceID: current.WorkspaceID, CurrentRevision: current.Revision})
		if err != nil {
			return nil, err
		}
		out.Records = append(out.Records, Record{Kind: "supplement_schedule", ID: id, Revision: current.Revision, Data: data})
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
	declaredKinds := make(map[string]bool, len(envelope.Manifest.RecordKinds))
	for _, kind := range envelope.Manifest.RecordKinds {
		declaredKinds[kind] = true
	}
	if declaredKinds["supplement_schedule"] != declaredKinds["supplement_schedule_revision"] {
		return Export{}, errors.New("workspace manifest must declare both supplement schedule record kinds")
	}
	if declaredKinds["supplement_schedule"] && containsString(envelope.Manifest.Omissions, "supplements") {
		return Export{}, errors.New("workspace manifest both declares and omits supplement schedules")
	}
	if declaredKinds["nutrition_target"] && containsString(envelope.Manifest.Omissions, "nutrition_targets") {
		return Export{}, errors.New("workspace manifest both declares and omits nutrition targets")
	}
	allowed := map[string]bool{"workspace": true, "recipe": true, "recipe_revision": true, "plan": true, "shopping_state": true, "nutrition_target": true, "profile": true, "intake_event": true, "supplement_schedule": true, "supplement_schedule_revision": true}
	workspaceCount := 0
	workspaceID := ""
	shoppingCount := 0
	profileCount := 0
	intakeIDs := map[string]bool{}
	intakeEvents := map[string]intakeEventRecord{}
	targetRevisions := map[string]bool{}
	recipeCount := 0
	ingredientUses := 0
	seenRevisions := map[string]bool{}
	current := map[string]int64{}
	scheduleCurrent := map[string]int64{}
	scheduleRevisions := map[string]map[int64]supplement.Schedule{}
	for _, record := range envelope.Records {
		if !declaredKinds[record.Kind] {
			return Export{}, fmt.Errorf("workspace record kind %q is not declared by manifest", record.Kind)
		}
		if !allowed[record.Kind] {
			return Export{}, fmt.Errorf("workspace export contains unsupported record kind %q", record.Kind)
		}
		if record.ID == "" || len(record.Data) == 0 {
			return Export{}, fmt.Errorf("workspace export contains an invalid %s record", record.Kind)
		}
		if record.Kind == "workspace" {
			workspaceCount++
			workspaceID = record.ID
		}
		if record.Kind == "shopping_state" {
			shoppingCount++
			var state shopping.PersistedState
			if err := json.Unmarshal(record.Data, &state); err != nil {
				return Export{}, fmt.Errorf("invalid shopping state: %w", err)
			}
			if err := validateShoppingState(state); err != nil {
				return Export{}, err
			}
		}
		if record.Kind == "profile" {
			profileCount++
			var data profileRecord
			if err := json.Unmarshal(record.Data, &data); err != nil {
				return Export{}, fmt.Errorf("invalid profile: %w", err)
			}
			item := data.domain()
			if record.ID != workspaceID || item.WorkspaceID != workspaceID || item.Revision < 0 || item.Revision != record.Revision || item.PresetVersion != profile.CurrentPresetVersion {
				return Export{}, errors.New("profile identity or revision is invalid")
			}
			if !validProfilePreset(item.Preset) || !validProfileWeight(item.CostWeight) || !validProfileWeight(item.EffortWeight) || !validProfileWeight(item.VarietyWeight) {
				return Export{}, errors.New("profile preferences are invalid")
			}
			total := item.CostWeight + item.EffortWeight + item.VarietyWeight
			if total < .99 || total > 1.01 {
				return Export{}, errors.New("profile weights must sum to one")
			}
			if len(item.DraftJSON) > 256*1024 || (item.DraftJSON != "" && !json.Valid([]byte(item.DraftJSON))) {
				return Export{}, errors.New("profile draft is invalid JSON or exceeds its size limit")
			}
			for _, values := range [][]string{item.ActiveRules, item.ExcludedGroups, item.Allergies, item.Appliances} {
				for _, value := range values {
					if value == "" || len(value) > 256 {
						return Export{}, errors.New("profile contains an invalid preference value")
					}
				}
			}
		}
		if record.Kind == "intake_event" {
			var item intakeEventRecord
			if err := json.Unmarshal(record.Data, &item); err != nil {
				return Export{}, fmt.Errorf("record %s: invalid intake event: %w", record.ID, err)
			}
			if item.EventID != record.ID || item.WorkspaceID != workspaceID || item.EventID == "" || intakeIDs[item.EventID] || record.Revision != 0 {
				return Export{}, fmt.Errorf("intake event identity does not match %q", record.ID)
			}
			if _, err := time.Parse("2006-01-02", item.Date); err != nil {
				return Export{}, fmt.Errorf("intake event %q has invalid date", item.EventID)
			}
			if item.RecipeID == "" && item.RecipeRevision != 0 || item.RecipeID != "" && item.RecipeRevision < 1 {
				return Export{}, fmt.Errorf("intake event %q has invalid recipe revision reference", item.EventID)
			}
			if !validIntakeNutrient(item.NutrientID) || !validIntakeUnit(item.Unit) {
				return Export{}, fmt.Errorf("intake event %q has invalid nutrient or unit", item.EventID)
			}
			amount, err := decimalx.Parse(item.Amount)
			cmp, cmpErr := decimalx.Compare(amount, decimalx.KnownInt(0))
			if err != nil || cmpErr != nil || amount.IsUnknown() || cmp <= 0 {
				return Export{}, fmt.Errorf("intake event %q has invalid amount", item.EventID)
			}
			if item.RecordedAt == "" {
				return Export{}, fmt.Errorf("intake event %q has invalid recorded timestamp", item.EventID)
			}
			if _, err := time.Parse(time.RFC3339Nano, item.RecordedAt); err != nil {
				return Export{}, fmt.Errorf("intake event %q has invalid recorded timestamp", item.EventID)
			}
			intakeIDs[item.EventID] = true
			intakeEvents[item.EventID] = item
		}
		if record.Kind == "nutrition_target" {
			var item nutrition.Target
			if err := json.Unmarshal(record.Data, &item); err != nil {
				return Export{}, fmt.Errorf("record %s: invalid nutrition target: %w", record.ID, err)
			}
			if item.ID != record.ID || item.Revision != record.Revision || item.Revision < 1 {
				return Export{}, fmt.Errorf("nutrition target record identity does not match %q", record.ID)
			}
			if item.WorkspaceID != workspaceID {
				return Export{}, fmt.Errorf("nutrition target %q belongs to a different workspace", record.ID)
			}
			if item.Provenance == "" {
				return Export{}, fmt.Errorf("nutrition target %q is missing provenance", record.ID)
			}
			key := fmt.Sprintf("%s:%d", item.ID, item.Revision)
			if targetRevisions[key] {
				return Export{}, fmt.Errorf("duplicate nutrition target revision %s", key)
			}
			targetRevisions[key] = true
			if item.WorkspaceID == "" || item.Lower.IsUnknown() && item.Upper.IsUnknown() || !item.Lower.IsUnknown() && !item.Upper.IsUnknown() && func() bool { c, _ := decimalx.Compare(item.Lower, item.Upper); return c > 0 }() || item.EffectiveFrom.IsZero() || item.EffectiveTo != nil && !item.EffectiveTo.After(item.EffectiveFrom) {
				return Export{}, fmt.Errorf("nutrition target %s has invalid bounds or effective window", key)
			}
			if err := nutrition.ValidateTarget(item); err != nil {
				return Export{}, fmt.Errorf("nutrition target %s: %w", key, err)
			}
		}
		if record.Kind == "supplement_schedule" {
			var item supplementScheduleRecord
			if err := json.Unmarshal(record.Data, &item); err != nil {
				return Export{}, fmt.Errorf("invalid supplement schedule %q: %w", record.ID, err)
			}
			if item.WorkspaceID != workspaceID || item.CurrentRevision < 1 || record.Revision != item.CurrentRevision || scheduleCurrent[record.ID] != 0 {
				return Export{}, fmt.Errorf("supplement schedule identity or current revision is invalid for %q", record.ID)
			}
			scheduleCurrent[record.ID] = item.CurrentRevision
		}
		if record.Kind == "supplement_schedule_revision" {
			var item supplementScheduleRevisionRecord
			if err := json.Unmarshal(record.Data, &item); err != nil {
				return Export{}, fmt.Errorf("invalid supplement schedule revision %q: %w", record.ID, err)
			}
			if item.WorkspaceID != workspaceID || record.Revision < 1 || strings.TrimSpace(record.ID) == "" || strings.TrimSpace(item.ProductRevisionID) == "" {
				return Export{}, fmt.Errorf("supplement schedule revision identity or source reference is invalid for %q", record.ID)
			}
			dose, err := decimalx.Parse(item.Dose)
			createdAt, timeErr := time.Parse(time.RFC3339Nano, item.CreatedAt)
			cmp, cmpErr := decimalx.Compare(dose, decimalx.KnownInt(0))
			if err != nil || cmpErr != nil || dose.IsUnknown() || cmp <= 0 || timeErr != nil || createdAt.IsZero() {
				return Export{}, fmt.Errorf("supplement schedule revision %s:%d has invalid dose or timestamp", record.ID, record.Revision)
			}
			schedule := supplement.Schedule{ID: record.ID, Revision: record.Revision, WorkspaceID: item.WorkspaceID, ProductRevisionID: item.ProductRevisionID, Dose: dose, DoseUnit: item.DoseUnit, Weekdays: item.Weekdays, StartDate: item.StartDate, EndDate: item.EndDate, Paused: item.Paused, Confirmed: item.Confirmed, CreatedAt: createdAt}
			if err := supplement.Validate(schedule); err != nil {
				return Export{}, fmt.Errorf("supplement schedule revision %s:%d: %w", record.ID, record.Revision, err)
			}
			if scheduleRevisions[record.ID] == nil {
				scheduleRevisions[record.ID] = map[int64]supplement.Schedule{}
			}
			if _, exists := scheduleRevisions[record.ID][record.Revision]; exists {
				return Export{}, fmt.Errorf("duplicate supplement schedule revision %s:%d", record.ID, record.Revision)
			}
			scheduleRevisions[record.ID][record.Revision] = schedule
		}
		if record.Kind == "recipe_revision" {
			var item Recipe
			if err := json.Unmarshal(record.Data, &item); err != nil {
				return Export{}, fmt.Errorf("record %s: %w", record.ID, err)
			}
			if err := validatePortableRecipe(item); err != nil {
				return Export{}, fmt.Errorf("record %s: %w", record.ID, err)
			}
			key := fmt.Sprintf("%s:%d", item.ID, item.Revision)
			if seenRevisions[key] {
				return Export{}, fmt.Errorf("duplicate recipe revision %s", key)
			}
			seenRevisions[key] = true
			recipeCount++
			ingredientUses += len(item.Ingredients)
		}
		if record.Kind == "recipe" {
			var identity struct {
				CurrentRevision int64 `json:"currentRevision"`
			}
			if err := json.Unmarshal(record.Data, &identity); err != nil || identity.CurrentRevision < 1 {
				return Export{}, fmt.Errorf("invalid current recipe identity %q", record.ID)
			}
			if current[record.ID] != 0 {
				return Export{}, fmt.Errorf("duplicate current recipe identity %q", record.ID)
			}
			current[record.ID] = identity.CurrentRevision
		}
	}
	if shoppingCount > 1 {
		return Export{}, errors.New("workspace export contains duplicate shopping state")
	}
	if len(targetRevisions) > 500 {
		return Export{}, errors.New("workspace export exceeds nutrition target revision limit")
	}
	for id, revisions := range scheduleRevisions {
		currentRevision, exists := scheduleCurrent[id]
		if !exists || currentRevision < 1 || int64(len(revisions)) != currentRevision {
			return Export{}, fmt.Errorf("supplement schedule %q has incomplete revision history", id)
		}
		for revision := int64(1); revision <= currentRevision; revision++ {
			if _, ok := revisions[revision]; !ok {
				return Export{}, fmt.Errorf("supplement schedule %q is missing revision %d", id, revision)
			}
		}
	}
	for id := range scheduleCurrent {
		if _, ok := scheduleRevisions[id]; !ok {
			return Export{}, fmt.Errorf("supplement schedule %q has no revisions", id)
		}
	}
	if len(current) > 0 {
		for id, revision := range current {
			if !seenRevisions[fmt.Sprintf("%s:%d", id, revision)] {
				return Export{}, fmt.Errorf("recipe %q current revision %d is missing", id, revision)
			}
		}
	}
	for id, item := range intakeEvents {
		if item.RecipeID == "" || current[item.RecipeID] == 0 {
			continue
		} // Historical references stay opaque when their recipe is not in this export.
		if !seenRevisions[fmt.Sprintf("%s:%d", item.RecipeID, item.RecipeRevision)] {
			return Export{}, fmt.Errorf("intake event %q references missing imported recipe revision %s:%d", id, item.RecipeID, item.RecipeRevision)
		}
	}
	if workspaceCount != 1 {
		return Export{}, errors.New("workspace export must contain exactly one workspace record")
	}
	if profileCount > 1 {
		return Export{}, errors.New("workspace export contains multiple profiles")
	}
	if len(intakeEvents) > 10000 {
		return Export{}, errors.New("workspace export exceeds intake event limit")
	}
	if declaredKinds["intake_event"] && containsString(envelope.Manifest.Omissions, "intake_events") {
		return Export{}, errors.New("workspace manifest both declares and omits intake events")
	}
	for id, item := range intakeEvents {
		if item.CorrectionOf == "" {
			continue
		}
		if item.CorrectionOf == id {
			return Export{}, fmt.Errorf("intake event %q cannot correct itself", id)
		}
		seen := map[string]bool{id: true}
		for target := item.CorrectionOf; target != ""; {
			if seen[target] {
				return Export{}, fmt.Errorf("intake event correction cycle includes %q", target)
			}
			seen[target] = true
			prior, ok := intakeEvents[target]
			if !ok {
				return Export{}, fmt.Errorf("intake event %q corrects unknown event %q", id, target)
			}
			target = prior.CorrectionOf
		}
	}
	if declaredKinds["profile"] && containsString(envelope.Manifest.Omissions, "profile") {
		return Export{}, errors.New("workspace manifest both declares and omits profile")
	}
	if recipeCount > 200 || ingredientUses > 80 {
		return Export{}, errors.New("workspace export exceeds recipe or ingredient-use limit")
	}
	return envelope, nil
}

func validIntakeNutrient(value string) bool {
	_, ok := nutrients.Lookup(value)
	return ok
}

func validIntakeUnit(value string) bool {
	if value == "kcal" {
		return true
	}
	_, ok := units.Lookup(value)
	return ok
}

func validProfilePreset(value string) bool {
	switch value {
	case "everything", "vegan", "vegetarian", "pescatarian", "plant-forward", "my-own-way":
		return true
	default:
		return false
	}
}

func validProfileWeight(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func validateShoppingState(state shopping.PersistedState) error {
	if len(state.Checks)+len(state.HaveThis) > 10000 || len(state.Reviews) > 1000 {
		return errors.New("workspace shopping state exceeds record limits")
	}
	for key := range state.Checks {
		if key == "" {
			return errors.New("shopping check requires a line key")
		}
	}
	for key := range state.HaveThis {
		if key == "" {
			return errors.New("Have-this choice requires a line key")
		}
	}
	seenReviews := map[string]bool{}
	for _, review := range state.Reviews {
		if review.ID == "" || review.PayloadHash == "" || review.CreatedAt == "" || seenReviews[review.ID] {
			return errors.New("purchase review requires unique id, payload hash, and created time")
		}
		seenReviews[review.ID] = true
		if len(review.Lines) == 0 {
			return fmt.Errorf("purchase review %q has no lines", review.ID)
		}
		seenLines := map[string]bool{}
		for _, line := range review.Lines {
			if line.Key == "" || line.ItemID == "" || line.Unit == "" || seenLines[line.Key] {
				return fmt.Errorf("purchase review %q contains invalid lines", review.ID)
			}
			seenLines[line.Key] = true
			if !line.Omitted {
				amount, err := decimalx.Parse(line.Amount)
				comparison, compareErr := decimalx.Compare(amount, decimalx.KnownInt(0))
				if err != nil || compareErr != nil || amount.IsUnknown() || comparison <= 0 {
					return fmt.Errorf("purchase review %q line %q has invalid amount", review.ID, line.Key)
				}
			}
		}
	}
	return nil
}

type Recipe struct {
	ID                 string              `json:"id"`
	Revision           int64               `json:"revision"`
	Name               string              `json:"name"`
	Notes              string              `json:"notes,omitempty"`
	SourceURL          string              `json:"sourceUrl,omitempty"`
	SourceType         string              `json:"sourceType,omitempty"`
	OriginalText       string              `json:"originalText,omitempty"`
	Status             string              `json:"status"`
	CanonicalYield     string              `json:"canonicalYield,omitempty"`
	ServingUnit        string              `json:"servingUnit,omitempty"`
	Ingredients        []recipe.Ingredient `json:"ingredients,omitempty"`
	Groups             []string            `json:"groups,omitempty"`
	RequiredAppliances []string            `json:"requiredAppliances,omitempty"`
	AllergenEvidence   map[string]string   `json:"allergenEvidence,omitempty"`
	Methods            []recipe.Method     `json:"methods,omitempty"`
}

func Recipes(items []recipe.Recipe) ([]byte, error) {
	return RecipesWithHistory(items, nil)
}

// RecipesWithHistory writes the current identity plus every supported immutable revision.
func RecipesWithHistory(items, history []recipe.Recipe) ([]byte, error) {
	out := Export{Format: "daily.recipes", SchemaVersion: 2, Manifest: Manifest{
		RecordKinds: []string{"recipe", "recipe_revision"}, AttachmentsIncluded: false,
		Omissions: []string{"nutrition", "cost"},
	}, Scope: Scope{Kind: "recipe_collection"}, Recipes: make([]Recipe, 0, len(items)), Records: make([]Record, 0, len(items)*2)}
	byID := make(map[string][]recipe.Recipe, len(items))
	current := make(map[string]recipe.Recipe, len(items))
	for _, item := range items {
		current[item.ID] = item
		byID[item.ID] = append(byID[item.ID], item)
	}
	for _, item := range history {
		if _, ok := current[item.ID]; ok {
			byID[item.ID] = append(byID[item.ID], item)
		}
	}
	for _, r := range items {
		portable := Recipe{ID: r.ID, Revision: r.Revision, Name: r.Name, Notes: r.Notes, SourceURL: r.SourceURL, SourceType: r.SourceType, OriginalText: r.OriginalText, Status: r.Status, CanonicalYield: r.CanonicalYield, ServingUnit: r.ServingUnit, Ingredients: r.Ingredients, Groups: r.Groups, RequiredAppliances: r.RequiredAppliances, AllergenEvidence: r.AllergenEvidence, Methods: r.Methods}
		out.Recipes = append(out.Recipes, portable)
		out.Scope.RecipeIDs = append(out.Scope.RecipeIDs, r.ID)
		identity, _ := json.Marshal(map[string]any{"currentRevision": r.Revision, "status": r.Status})
		out.Records = append(out.Records, Record{Kind: "recipe", ID: r.ID, Revision: r.Revision, Data: identity})
		for _, version := range byID[r.ID] {
			portableVersion := Recipe{ID: version.ID, Revision: version.Revision, Name: version.Name, Notes: version.Notes, SourceURL: version.SourceURL, SourceType: version.SourceType, OriginalText: version.OriginalText, Status: version.Status, CanonicalYield: version.CanonicalYield, ServingUnit: version.ServingUnit, Ingredients: version.Ingredients, Groups: version.Groups, RequiredAppliances: version.RequiredAppliances, AllergenEvidence: version.AllergenEvidence, Methods: version.Methods}
			revision, _ := json.Marshal(portableVersion)
			out.Records = append(out.Records, Record{Kind: "recipe_revision", ID: r.ID, Revision: version.Revision, Data: revision})
		}
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
	if envelope.Manifest.RecordCount != len(envelope.Records) {
		return Export{}, errors.New("recipe export record count does not match manifest")
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
	ingredientUses := 0
	for _, item := range envelope.Recipes {
		if err := validatePortableRecipe(item); err != nil {
			return Export{}, err
		}
		key := fmt.Sprintf("%s:%d", item.ID, item.Revision)
		if seen[key] {
			return Export{}, fmt.Errorf("duplicate recipe revision %s", key)
		}
		seen[key] = true
		ingredientUses += len(item.Ingredients)
	}
	seenRecords := map[string]bool{}
	for _, record := range envelope.Records {
		if record.Kind == "recipe_revision" {
			var item Recipe
			if err := json.Unmarshal(record.Data, &item); err != nil {
				return Export{}, fmt.Errorf("record %s: %w", record.ID, err)
			}
			if err := validatePortableRecipe(item); err != nil {
				return Export{}, err
			}
			key := fmt.Sprintf("%s:%d", item.ID, item.Revision)
			if seenRecords[key] {
				return Export{}, fmt.Errorf("duplicate recipe revision %s", key)
			}
			seenRecords[key] = true
		}
	}
	if ingredientUses > 80 {
		return Export{}, errors.New("recipe export exceeds 80 ingredient-use limit")
	}
	return envelope, nil
}

func validatePortableRecipe(item Recipe) error {
	if item.ID == "" || item.Revision < 1 || item.Name == "" {
		return errors.New("recipe export contains an invalid recipe")
	}
	if len(item.Notes) > 10_000 {
		return fmt.Errorf("recipe %q notes exceed 10,000 characters", item.ID)
	}
	steps := 0
	for _, method := range item.Methods {
		steps += len(method.Steps)
	}
	if steps > 30 {
		return fmt.Errorf("recipe %q exceeds 30 preparation steps", item.ID)
	}
	return nil
}

func portableRecipeHistory(envelope Export) (map[string][]Recipe, map[string]int64, error) {
	history := map[string][]Recipe{}
	current := map[string]int64{}
	for _, record := range envelope.Records {
		switch record.Kind {
		case "recipe":
			var identity struct {
				CurrentRevision int64 `json:"currentRevision"`
			}
			if err := json.Unmarshal(record.Data, &identity); err != nil || identity.CurrentRevision < 1 {
				return nil, nil, fmt.Errorf("invalid current recipe identity %q", record.ID)
			}
			if current[record.ID] != 0 {
				return nil, nil, fmt.Errorf("duplicate current recipe identity %q", record.ID)
			}
			current[record.ID] = identity.CurrentRevision
		case "recipe_revision":
			var item Recipe
			if err := json.Unmarshal(record.Data, &item); err != nil {
				return nil, nil, err
			}
			if err := validatePortableRecipe(item); err != nil {
				return nil, nil, err
			}
			if item.ID != record.ID || (record.Revision > 0 && item.Revision != record.Revision) {
				return nil, nil, fmt.Errorf("recipe revision identity mismatch for %q", record.ID)
			}
			history[item.ID] = append(history[item.ID], item)
		}
	}
	for _, item := range envelope.Recipes {
		if current[item.ID] == 0 {
			current[item.ID] = item.Revision
		}
		if len(history[item.ID]) == 0 {
			history[item.ID] = append(history[item.ID], item)
		}
	}
	for id, revisions := range history {
		if current[id] == 0 {
			current[id] = revisions[len(revisions)-1].Revision
		}
	}
	return history, current, nil
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
