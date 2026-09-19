package portability

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"nutrition-planner/internal/planning"
	"nutrition-planner/internal/recipe"
	"nutrition-planner/internal/shopping"
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
	if out.Manifest.RecordCount != len(out.Records) || len(out.Records) != 4 {
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

func TestWorkspaceImportStagesOnlySupportedRecords(t *testing.T) {
	content, err := Workspace(WorkspaceSnapshot{ID: "w1", Recipes: []recipe.Recipe{{ID: "r1", Revision: 1, Name: "Soup"}}})
	if err != nil {
		t.Fatal(err)
	}
	staged, err := ImportWorkspace(content)
	if err != nil || staged.Format != "daily.workspace" || len(staged.Records) != 3 {
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
