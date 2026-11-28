package portability

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/nutrition"
	"nutrition-planner/internal/planning"
	"nutrition-planner/internal/profile"
	"nutrition-planner/internal/recipe"
	"nutrition-planner/internal/shopping"
	"nutrition-planner/internal/supplement"
)

func TestNativeRecipeExportIsNamespacedAndHonest(t *testing.T) {
	b, err := Recipes([]recipe.Recipe{{
		ID: "r1", Revision: 1, Name: "Draft", Status: "draft",
		CanonicalYield: "4", ServingUnit: "servings", Ingredients: []recipe.Ingredient{{ID: "i1", Name: "Rice", Amount: "200", Unit: "g", Preparation: "rinsed"}},
		Groups: []string{"vegan"}, Methods: []recipe.Method{{ID: "m1", Name: "Cook", Steps: []recipe.MethodStep{{ID: "s1", Instruction: "stir"}}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	var out Export
	if err = json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Format != "daily.recipes" || out.SchemaVersion != 2 || out.Scope.Kind != "recipe_collection" || len(out.Recipes) != 1 || len(out.Records) != 2 || out.Manifest.RecordCount != 2 {
		t.Fatalf("%s", b)
	}
	if len(out.Recipes[0].Methods) != 1 || len(out.Recipes[0].Groups) != 1 {
		t.Fatalf("recipe graph was not exported: %s", b)
	}
	if out.Recipes[0].CanonicalYield != "4" || out.Recipes[0].ServingUnit != "servings" || len(out.Recipes[0].Ingredients) != 1 || out.Recipes[0].Ingredients[0].Preparation != "rinsed" {
		t.Fatalf("recipe yield and ingredients were not exported: %#v", out.Recipes[0])
	}
	for _, omission := range out.Manifest.Omissions {
		if omission == "methods" {
			t.Fatal("export omits a supported recipe field")
		}
	}
}

func TestWorkspaceSupplementScheduleRevisionHistoryValidatesAndExports(t *testing.T) {
	dose, _ := decimalx.Parse("2.375")
	base := supplement.Schedule{ID: "sched-1", Revision: 1, WorkspaceID: "w1", ProductRevisionID: "catalog-rev:opaque/one", Dose: dose, DoseUnit: "capsule", Weekdays: []int{1, 4}, StartDate: "2026-01-02", EndDate: "2026-03-04", Paused: false, Confirmed: true, CreatedAt: time.Date(2026, 1, 1, 12, 30, 0, 0, time.UTC)}
	current := base
	current.Revision, current.ProductRevisionID, current.DoseUnit = 2, "unresolved-catalog-revision", "tablet"
	current.Dose, _ = decimalx.Parse("3.125")
	current.Weekdays = []int{0, 6}
	current.StartDate, current.EndDate, current.Paused, current.Confirmed = "2026-02-02", "2026-04-04", true, false
	current.CreatedAt = time.Date(2026, 2, 1, 9, 8, 7, 123456789, time.UTC)
	content, err := Workspace(WorkspaceSnapshot{ID: "w1", SupplementSchedules: []supplement.Schedule{base, current}})
	if err != nil {
		t.Fatal(err)
	}
	var out Export
	if err := json.Unmarshal(content, &out); err != nil {
		t.Fatal(err)
	}
	if containsString(out.Manifest.Omissions, "supplements") || !containsString(out.Manifest.RecordKinds, "supplement_schedule") || !containsString(out.Manifest.RecordKinds, "supplement_schedule_revision") {
		t.Fatalf("manifest=%#v", out.Manifest)
	}
	staged, err := ImportWorkspace(content)
	if err != nil {
		t.Fatal(err)
	}
	var got []supplement.Schedule
	for _, record := range staged.Records {
		if record.Kind != "supplement_schedule_revision" {
			continue
		}
		var data supplementScheduleRevisionRecord
		if err := json.Unmarshal(record.Data, &data); err != nil {
			t.Fatal(err)
		}
		d, err := decimalx.Parse(data.Dose)
		if err != nil {
			t.Fatal(err)
		}
		at, err := time.Parse(time.RFC3339Nano, data.CreatedAt)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, supplement.Schedule{ID: record.ID, Revision: record.Revision, WorkspaceID: data.WorkspaceID, ProductRevisionID: data.ProductRevisionID, Dose: d, DoseUnit: data.DoseUnit, Weekdays: data.Weekdays, StartDate: data.StartDate, EndDate: data.EndDate, Paused: data.Paused, Confirmed: data.Confirmed, CreatedAt: at})
	}
	if !reflect.DeepEqual(got, []supplement.Schedule{base, current}) {
		t.Fatalf("schedule history mismatch\n got: %#v\nwant: %#v", got, []supplement.Schedule{base, current})
	}
	var pointerCount int
	for _, record := range staged.Records {
		if record.Kind == "supplement_schedule" {
			pointerCount++
			if record.Revision != 2 {
				t.Fatalf("current revision=%d", record.Revision)
			}
		}
	}
	if pointerCount != 1 {
		t.Fatalf("current pointers=%d", pointerCount)
	}
	missing := out
	missing.Records = append([]Record(nil), out.Records...)
	for i, record := range missing.Records {
		if record.Kind == "supplement_schedule_revision" && record.Revision == 1 {
			missing.Records = append(missing.Records[:i], missing.Records[i+1:]...)
			break
		}
	}
	missing.Manifest.RecordCount = len(missing.Records)
	bad, _ := json.Marshal(missing)
	if _, err := ImportWorkspace(bad); err == nil {
		t.Fatal("missing schedule history revision was accepted")
	}
	mutate := func(t *testing.T, change func(*Export)) {
		t.Helper()
		candidate := out
		candidate.Records = append([]Record(nil), out.Records...)
		change(&candidate)
		candidate.Manifest.RecordCount = len(candidate.Records)
		encoded, err := json.Marshal(candidate)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ImportWorkspace(encoded); err == nil {
			t.Fatal("invalid supplement schedule manifest was staged")
		}
	}
	for _, tc := range []struct {
		name   string
		change func(*Export)
	}{
		{"duplicate current identity", func(e *Export) {
			for _, r := range e.Records {
				if r.Kind == "supplement_schedule" {
					e.Records = append(e.Records, r)
					return
				}
			}
		}},
		{"duplicate revision", func(e *Export) {
			for _, r := range e.Records {
				if r.Kind == "supplement_schedule_revision" {
					e.Records = append(e.Records, r)
					return
				}
			}
		}},
		{"missing current pointer", func(e *Export) {
			for i, r := range e.Records {
				if r.Kind == "supplement_schedule" {
					e.Records = append(e.Records[:i], e.Records[i+1:]...)
					return
				}
			}
		}},
		{"out of range current pointer", func(e *Export) {
			for i, r := range e.Records {
				if r.Kind == "supplement_schedule" {
					var p supplementScheduleRecord
					_ = json.Unmarshal(r.Data, &p)
					p.CurrentRevision = 3
					r.Revision = 3
					r.Data, _ = json.Marshal(p)
					e.Records[i] = r
					return
				}
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) { mutate(t, tc.change) })
	}
	for _, tc := range []struct {
		name   string
		change func(*supplementScheduleRevisionRecord)
	}{
		{"workspace ownership", func(v *supplementScheduleRevisionRecord) { v.WorkspaceID = "other" }},
		{"missing product reference", func(v *supplementScheduleRevisionRecord) { v.ProductRevisionID = "" }},
		{"negative dose", func(v *supplementScheduleRevisionRecord) { v.Dose = "-1" }},
		{"empty unit", func(v *supplementScheduleRevisionRecord) { v.DoseUnit = "" }},
		{"invalid weekdays", func(v *supplementScheduleRevisionRecord) { v.Weekdays = []int{8} }},
		{"invalid dates", func(v *supplementScheduleRevisionRecord) { v.StartDate = "2026/03/01" }},
		{"invalid timestamp", func(v *supplementScheduleRevisionRecord) { v.CreatedAt = "yesterday" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mutate(t, func(e *Export) {
				for i, r := range e.Records {
					if r.Kind == "supplement_schedule_revision" && r.Revision == 1 {
						var v supplementScheduleRevisionRecord
						_ = json.Unmarshal(r.Data, &v)
						tc.change(&v)
						r.Data, _ = json.Marshal(v)
						e.Records[i] = r
						return
					}
				}
			})
		})
	}
}

func TestWorkspaceProfileValidationRejectsMalformedIdentityRevisionAndPreferences(t *testing.T) {
	base := profile.Profile{WorkspaceID: "w1", Revision: 2, Preset: "vegan", PresetVersion: profile.CurrentPresetVersion, ActiveRules: profile.ExpandPreset("vegan"), CostWeight: .2, EffortWeight: .3, VarietyWeight: .5, DraftJSON: `{"draft":true}`}
	for _, mutate := range []struct {
		name   string
		change func(*profile.Profile)
	}{
		{"workspace identity", func(p *profile.Profile) { p.WorkspaceID = "other" }},
		{"revision", func(p *profile.Profile) { p.Revision = -1 }},
		{"preset", func(p *profile.Profile) { p.Preset = "unknown" }},
		{"weights", func(p *profile.Profile) { p.CostWeight = 1.5 }},
		{"weight total", func(p *profile.Profile) { p.VarietyWeight = .1 }},
		{"draft", func(p *profile.Profile) { p.DraftJSON = `{` }},
	} {
		t.Run(mutate.name, func(t *testing.T) {
			item := base
			mutate.change(&item)
			data, err := json.Marshal(item)
			if err != nil {
				t.Fatal(err)
			}
			content, err := Workspace(WorkspaceSnapshot{ID: "w1"})
			if err != nil {
				t.Fatal(err)
			}
			var envelope Export
			if err := json.Unmarshal(content, &envelope); err != nil {
				t.Fatal(err)
			}
			envelope.Records = append(envelope.Records, Record{Kind: "profile", ID: "w1", Revision: item.Revision, Data: data})
			envelope.Manifest.RecordCount = len(envelope.Records)
			if _, err := ImportWorkspace(mustJSON(t, envelope)); err == nil {
				t.Fatal("invalid profile was staged")
			}
		})
	}
	malformed := []byte(`{"format":"daily.workspace","schemaVersion":2,"scope":{"kind":"workspace_backup"},"manifest":{"recordKinds":["workspace","profile"],"recordCount":2,"attachmentsIncluded":false},"records":[{"kind":"workspace","id":"w1","revision":0,"data":{}},{"kind":"profile","id":"w1","revision":1,"data":{"broken"}]}`)
	if _, err := ImportWorkspace(malformed); err == nil {
		t.Fatal("malformed profile JSON was staged")
	}
}

func TestWorkspaceTargetRevisionValidationRejectsInvalidBoundsAndDuplicates(t *testing.T) {
	start, _ := time.Parse(time.RFC3339, "2026-10-01T00:00:00Z")
	target := nutrition.Target{ID: "target-1", Revision: 1, WorkspaceID: "w1", NutrientID: "protein", Lower: decimalx.KnownInt(70), Upper: decimalx.KnownInt(100), Period: "local_day", Scope: "planned_day", Enforcement: "required", Provenance: "user_assertion:test", EffectiveFrom: start, Active: true}
	content, err := Workspace(WorkspaceSnapshot{ID: "w1", NutritionTargets: []nutrition.Target{target}})
	if err != nil {
		t.Fatal(err)
	}
	var envelope Export
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatal(err)
	}
	invalid := target
	invalid.Lower = decimalx.KnownInt(101)
	invalidContent, err := Workspace(WorkspaceSnapshot{ID: "w1", NutritionTargets: []nutrition.Target{invalid}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ImportWorkspace(invalidContent); err == nil {
		t.Fatal("invalid bounds passed staging")
	}
	var duplicate Record
	for _, record := range envelope.Records {
		if record.Kind == "nutrition_target" {
			duplicate = record
		}
	}
	envelope.Records = append(envelope.Records, duplicate)
	envelope.Manifest.RecordCount = len(envelope.Records)
	duplicateContent, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ImportWorkspace(duplicateContent); err == nil {
		t.Fatal("duplicate target revision passed staging")
	}
}

func TestNativeRecordEnvelopeStagesRevisionRecords(t *testing.T) {
	content, err := Recipes([]recipe.Recipe{{ID: "r1", Revision: 1, Name: "Soup"}})
	if err != nil {
		t.Fatal(err)
	}
	var envelope Export
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.Recipes = nil
	content, err = json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	staged, err := ImportRecipes(content)
	if err != nil || len(staged.Recipes) != 1 || staged.Recipes[0].Name != "Soup" {
		t.Fatalf("staged=%#v err=%v", staged, err)
	}
}

func TestWorkspaceExportNamesSupportedDomainsAndOmissions(t *testing.T) {
	content, err := Workspace(WorkspaceSnapshot{
		ID: "w1", Name: "Household", Revision: 3,
		Recipes:      []recipe.Recipe{{ID: "r1", Revision: 2, Name: "Soup", Status: "draft"}},
		PlanRevision: 4, PlanJSON: `{"occurrences":[]}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	var out Export
	if err := json.Unmarshal(content, &out); err != nil {
		t.Fatal(err)
	}
	if out.Format != "daily.workspace" || out.SchemaVersion != 2 || out.Scope.Kind != "workspace_backup" {
		t.Fatalf("unexpected envelope: %s", content)
	}
	if out.Manifest.RecordCount != len(out.Records) || len(out.Records) != 5 {
		t.Fatalf("record count=%d records=%d", out.Manifest.RecordCount, len(out.Records))
	}
	for _, record := range out.Records {
		if strings.Contains(string(record.Data), "owner") || strings.Contains(string(record.Data), "subject") {
			t.Fatalf("private authority data leaked into record %s: %s", record.Kind, record.Data)
		}
	}
	omissions := strings.Join(out.Manifest.Omissions, ",")
	if !strings.Contains(omissions, "inventory") || !strings.Contains(omissions, "authentication") {
		t.Fatalf("unsupported domains were not declared: %v", out.Manifest.Omissions)
	}
}

func TestWorkspaceIntakeEventsPreserveExactFieldsAndValidateCorrectionLinks(t *testing.T) {
	stamp := time.Date(2026, 10, 7, 11, 2, 3, 456789123, time.FixedZone("source", -4*60*60))
	db := restoreDB(t)
	repo := nutrition.NewSQLiteIntakeRepository(db, intakeTestClock{stamp})
	for _, item := range []nutrition.IntakeEvent{
		{ID: "eat-1", Date: "2026-10-07", RecipeID: "historical-recipe", RecipeRevision: 7, NutrientID: "protein", Amount: mustDecimal(t, "12.3400"), Unit: "g", Reason: "recorded portion"},
		{ID: "correct-1", Date: "2026-10-07", NutrientID: "protein", Amount: mustDecimal(t, "11.25"), Unit: "g", Reason: "corrected weighing", CorrectionOf: "eat-1"},
	} {
		if err := repo.Append(context.Background(), "w1", item); err != nil {
			t.Fatal(err)
		}
	}
	items, err := repo.List(context.Background(), "w1")
	if err != nil {
		t.Fatal(err)
	}
	content, err := Workspace(WorkspaceSnapshot{ID: "w1", IntakeEvents: items})
	if err != nil {
		t.Fatal(err)
	}
	staged, err := ImportWorkspace(content)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(staged.Manifest.RecordKinds, "intake_event") || containsString(staged.Manifest.Omissions, "intake_events") {
		t.Fatalf("manifest=%+v", staged.Manifest)
	}
	var got []intakeEventRecord
	for _, record := range staged.Records {
		if record.Kind == "intake_event" {
			var item intakeEventRecord
			if err := json.Unmarshal(record.Data, &item); err != nil {
				t.Fatal(err)
			}
			got = append(got, item)
		}
	}
	if len(got) != 2 || got[1].Amount != "12.34" || got[1].RecipeID != "historical-recipe" || got[1].RecipeRevision != 7 || got[1].RecordedAt != "2026-10-07T15:02:03.456789123Z" || got[0].CorrectionOf != "eat-1" {
		t.Fatalf("events=%+v", got)
	}
	for name, change := range map[string]func(*Export){
		"cycle": func(e *Export) {
			for i := range e.Records {
				if e.Records[i].Kind == "intake_event" {
					var item intakeEventRecord
					_ = json.Unmarshal(e.Records[i].Data, &item)
					if item.EventID == "eat-1" {
						item.CorrectionOf = "correct-1"
						e.Records[i].Data, _ = json.Marshal(item)
					}
				}
			}
		},
		"bad-unit": func(e *Export) {
			for i := range e.Records {
				if e.Records[i].Kind == "intake_event" {
					var item intakeEventRecord
					_ = json.Unmarshal(e.Records[i].Data, &item)
					item.Unit = "mystery"
					e.Records[i].Data, _ = json.Marshal(item)
					break
				}
			}
		},
		"bad-nutrient": func(e *Export) {
			for i := range e.Records {
				if e.Records[i].Kind == "intake_event" {
					var item intakeEventRecord
					_ = json.Unmarshal(e.Records[i].Data, &item)
					item.NutrientID = "invented-nutrient"
					e.Records[i].Data, _ = json.Marshal(item)
					break
				}
			}
		},
		"bad-date":            func(e *Export) { mutateIntakeRecord(e, func(item *intakeEventRecord) { item.Date = "2026-02-31" }) },
		"zero-amount":         func(e *Export) { mutateIntakeRecord(e, func(item *intakeEventRecord) { item.Amount = "0" }) },
		"bad-source-revision": func(e *Export) { mutateIntakeRecord(e, func(item *intakeEventRecord) { item.RecipeRevision = -1 }) },
		"bad-workspace":       func(e *Export) { mutateIntakeRecord(e, func(item *intakeEventRecord) { item.WorkspaceID = "foreign" }) },
		"bad-timestamp": func(e *Export) {
			mutateIntakeRecord(e, func(item *intakeEventRecord) { item.RecordedAt = "yesterday" })
		},
		"unknown-correction-target": func(e *Export) {
			mutateIntakeRecord(e, func(item *intakeEventRecord) {
				if item.EventID == "correct-1" {
					item.CorrectionOf = "missing-event"
				}
			})
		},
		"mismatched-record-id": func(e *Export) {
			for i := range e.Records {
				if e.Records[i].Kind == "intake_event" {
					e.Records[i].ID = "other-id"
					break
				}
			}
		},
		"malformed-payload": func(e *Export) {
			for i := range e.Records {
				if e.Records[i].Kind == "intake_event" {
					e.Records[i].Data = json.RawMessage(`"not-an-event-object"`)
					break
				}
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			var e Export
			_ = json.Unmarshal(content, &e)
			change(&e)
			e.Manifest.RecordCount = len(e.Records)
			if _, err := ImportWorkspace(mustJSON(t, e)); err == nil {
				t.Fatal("invalid intake event staged")
			}
		})
	}
	t.Run("duplicate-event-id", func(t *testing.T) {
		var e Export
		_ = json.Unmarshal(content, &e)
		for _, record := range e.Records {
			if record.Kind == "intake_event" {
				e.Records = append(e.Records, record)
				break
			}
		}
		e.Manifest.RecordCount = len(e.Records)
		if _, err := ImportWorkspace(mustJSON(t, e)); err == nil {
			t.Fatal("duplicate intake event ID passed staging")
		}
	})
}

func TestWorkspaceIntakeReferenceMustMatchImportedRecipeRevision(t *testing.T) {
	amount := mustDecimal(t, "1.5")
	content, err := Workspace(WorkspaceSnapshot{ID: "w1", Recipes: []recipe.Recipe{{ID: "included", Revision: 2, Name: "Source", Status: "draft"}}, IntakeEvents: []nutrition.IntakeEvent{{ID: "meal", WorkspaceID: "w1", Date: "2026-10-07", RecipeID: "included", RecipeRevision: 1, NutrientID: "protein", Amount: amount, Unit: "g"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ImportWorkspace(content); err == nil {
		t.Fatal("intake event pointing to missing imported recipe revision passed staging")
	}
	legacy, err := Workspace(WorkspaceSnapshot{ID: "w1", IntakeEvents: []nutrition.IntakeEvent{{ID: "historical", WorkspaceID: "w1", Date: "2026-10-07", RecipeID: "not-in-export", RecipeRevision: 8, NutrientID: "protein", Amount: amount, Unit: "g"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ImportWorkspace(legacy); err != nil {
		t.Fatalf("opaque historical recipe reference rejected: %v", err)
	}
}

func mutateIntakeRecord(envelope *Export, mutate func(*intakeEventRecord)) {
	for i := range envelope.Records {
		if envelope.Records[i].Kind != "intake_event" {
			continue
		}
		var item intakeEventRecord
		_ = json.Unmarshal(envelope.Records[i].Data, &item)
		mutate(&item)
		envelope.Records[i].Data, _ = json.Marshal(item)
		return
	}
}

type intakeTestClock struct{ value time.Time }

func (c intakeTestClock) Now() time.Time { return c.value }

func TestWorkspaceImportStagesOnlySupportedRecords(t *testing.T) {
	content, err := Workspace(WorkspaceSnapshot{ID: "w1", Recipes: []recipe.Recipe{{ID: "r1", Revision: 1, Name: "Soup"}}})
	if err != nil {
		t.Fatal(err)
	}
	staged, err := ImportWorkspace(content)
	if err != nil || staged.Format != "daily.workspace" || len(staged.Records) != 4 {
		t.Fatalf("staged=%#v err=%v", staged, err)
	}
	var envelope Export
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.Manifest.RecordCount++
	bad, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ImportWorkspace(bad); err == nil {
		t.Fatal("expected manifest count rejection")
	}
}

func TestImportRecipesStagesAndRejectsDuplicateRevisions(t *testing.T) {
	content, err := Recipes([]recipe.Recipe{{ID: "r1", Revision: 1, Name: "Soup"}})
	if err != nil {
		t.Fatal(err)
	}
	staged, err := ImportRecipes(content)
	if err != nil || len(staged.Recipes) != 1 {
		t.Fatalf("staged=%#v err=%v", staged, err)
	}
	duplicate := append([]byte(nil), content...)
	duplicate = bytes.Replace(duplicate, []byte(`"recipes":[{`), []byte(`"recipes":[{"id":"r1","revision":1,"name":"Soup"},{`), 1)
	if _, err := ImportRecipes(duplicate); err == nil {
		t.Fatal("expected duplicate revision rejection")
	}
	var envelope Export
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.Manifest.RecordCount++
	bad, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ImportRecipes(bad); err == nil {
		t.Fatal("expected recipe manifest count rejection")
	}
}

func TestImportRecipesEnforcesDocumentedEntityLimits(t *testing.T) {
	item := recipe.Recipe{ID: "r1", Revision: 1, Name: "Soup", Notes: strings.Repeat("n", 10_001)}
	content, err := Recipes([]recipe.Recipe{item})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ImportRecipes(content); err == nil || !strings.Contains(err.Error(), "10,000") {
		t.Fatalf("expected note-size rejection, got %v", err)
	}

	item.Notes = ""
	item.Methods = []recipe.Method{{Name: "Cook", Steps: make([]recipe.MethodStep, 31)}}
	content, err = Recipes([]recipe.Recipe{item})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ImportRecipes(content); err == nil || !strings.Contains(err.Error(), "30 preparation") {
		t.Fatalf("expected step-count rejection, got %v", err)
	}
}

func TestGroceriesCSVQuotesAndNeutralizesFormulaCells(t *testing.T) {
	content, err := GroceriesCSV([]GroceryRow{{Key: "ingredient:rice", Label: "Rice, long grain", Need: "unknown", Price: "=IMPORTXML(\"x\")"}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, `"Rice, long grain"`) || !strings.Contains(text, `'=IMPORTXML`) {
		t.Fatalf("unsafe or unquoted CSV: %q", text)
	}
}

func TestRecipePDFPreservesUnicodeAndLongInstructions(t *testing.T) {
	content, err := RecipePDF(recipe.Recipe{
		Name:    "Café crème — 早餐",
		Status:  "draft",
		Notes:   "A long note that remains useful when printed for a shared kitchen.",
		Methods: []recipe.Method{{Name: "Préparation", Steps: []recipe.MethodStep{{Instruction: "Stir gently; 再煮五分钟; serve warm."}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(content, []byte("%PDF-")) || !bytes.Contains(content, []byte("/Type /Page")) {
		t.Fatalf("not a valid PDF artifact: %q", content[:min(32, len(content))])
	}
}

func TestWeeklyPDFKeepsOpenSlotsAndUnknownValues(t *testing.T) {
	content, err := WeeklyPDF(planning.Draft{
		ObjectiveVersion: "normalized-loss-v1",
		Occurrences:      []planning.Occurrence{{Date: "2026-09-18", SlotName: "dinner", Mode: "social", Reason: "open social slot"}},
		Unresolved:       []planning.Unresolved{{Date: "2026-09-19", Message: "No eligible recipe"}},
	}, []shopping.Line{{Label: "rice", Need: "unknown", Stock: "unknown", Price: "unknown"}})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(content, []byte("%PDF-")) {
		t.Fatalf("not a valid PDF artifact")
	}
}

func TestPDFSupportsLetterAndRejectsUnknownPageSize(t *testing.T) {
	content, err := RecipePDFWithPageSize(recipe.Recipe{Name: "Letter recipe"}, "LETTER")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, []byte("/MediaBox [0 0 612.00 792.00]")) {
		t.Fatalf("expected US Letter page, got %q", content)
	}
	if _, err := WeeklyPDFWithPageSize(planning.Draft{}, nil, "tabloid"); err == nil {
		t.Fatal("expected unsupported page size error")
	}
}

func TestImportLegacyMinimalRecipePreservesEvidenceLimits(t *testing.T) {
	content := []byte(`{"id":"meal_legacy_example","name":"My usual breakfast","calories":null,"protein":null,"cost":null,"minutes":null,"items":[],"steps":[],"groups":[],"allergens":[],"reviewed":false,"sample":true,"notes":"old notes"}`)
	result, err := ImportLegacy(content)
	if err != nil {
		t.Fatal(err)
	}
	if result.Provenance != "legacy_import" || len(result.Recipes) != 1 || result.Recipes[0].SourceType != "legacy_import" {
		t.Fatalf("result=%#v", result)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected explicit evidence warning")
	}
}

func TestImportLegacyWorkspaceDoesNotInventDatesOrConsumption(t *testing.T) {
	content := []byte(`{"schemaVersion":1,"profile":{"diet":"vegan"},"recipes":[],"plan":["r1",null,null,null,null,null,null],"completed":[0],"checked":["rice"]}`)
	result, err := ImportLegacy(content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.WeekPlan) != 7 || result.WeekPlan[0] != "r1" || result.WeekPlan[1] != "" {
		t.Fatalf("plan=%#v", result.WeekPlan)
	}
	if len(result.Completed) != 1 || result.Completed[0] != 0 || len(result.Checked) != 1 {
		t.Fatalf("history was not preserved as legacy metadata: %#v", result)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
