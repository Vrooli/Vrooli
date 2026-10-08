package portability

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/schedule"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/portability"
	_ "modernc.org/sqlite"
	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/nutrition"
	"nutrition-planner/internal/planning"
	internalPortability "nutrition-planner/internal/portability"
	internalProfile "nutrition-planner/internal/profile"
	"nutrition-planner/internal/recipe"
	"nutrition-planner/internal/shopping"
	"nutrition-planner/internal/supplement"
	"nutrition-planner/internal/workspace"
)

type checkpointWorkspaceService struct{}

func (checkpointWorkspaceService) Create(context.Context, workspace.CreateInput) (workspace.Workspace, error) {
	panic("unused")
}
func (checkpointWorkspaceService) List(context.Context, string) ([]workspace.Workspace, error) {
	panic("unused")
}
func (checkpointWorkspaceService) Get(_ context.Context, id, owner string) (workspace.Workspace, error) {
	if id != "w1" {
		return workspace.Workspace{}, workspace.ErrNotFound{ID: id}
	}
	if owner != "owner" {
		return workspace.Workspace{}, workspace.ErrForbidden{ID: id}
	}
	return workspace.Workspace{ID: id, Name: "Daily", OwnerSubject: owner, Revision: 2}, nil
}

func checkpointHandler(t *testing.T) *connectHandler {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, schema := range []string{workspace.Schema(), recipe.Schema(), planning.Schema(), nutrition.Schema(), internalProfile.Schema(), shopping.Schema(), supplement.Schema(), internalPortability.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO workspaces(id,name,owner_subject,revision,created_at,updated_at) VALUES('w1','Daily','owner',2,'now','now')`); err != nil {
		t.Fatal(err)
	}
	clock := schedule.System()
	return NewConnectHandlerWithSchedules(recipe.NewService(recipe.NewSQLiteRepository(db, clock)), checkpointWorkspaceService{}, planning.NewSQLiteRepository(db, clock), nil, nutrition.NewSQLiteTargetRepository(db, clock), internalProfile.NewService(internalProfile.NewSQLiteRepository(db, clock)), nutrition.NewSQLiteIntakeRepository(db), supplement.NewSQLiteRepository(db, clock), internalPortability.NewSQLiteRestorer(db, clock), log.Default())
}

func TestWorkspaceExportIncludesIntakeEventProvenance(t *testing.T) {
	h := checkpointHandler(t)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "owner", Verified: true})
	if _, err := h.profiles.Apply(ctx, internalProfile.ApplyInput{WorkspaceID: "w1", Preset: "everything", CostWeight: .34, EffortWeight: .33, VarietyWeight: .33}); err != nil {
		t.Fatal(err)
	}
	amount, _ := decimalx.Parse("22.125")
	if err := h.intakes.Append(ctx, "w1", nutrition.IntakeEvent{ID: "meal-1", Date: "2026-10-07", RecipeID: "recipe-old", RecipeRevision: 4, NutrientID: "protein", Amount: amount, Unit: "g", Reason: "actual portion"}); err != nil {
		t.Fatal(err)
	}
	response, err := h.ExportWorkspace(ctx, connect.NewRequest(&v1.ExportWorkspaceRequest{WorkspaceId: "w1"}))
	if err != nil {
		t.Fatal(err)
	}
	staged, err := internalPortability.ImportWorkspace([]byte(response.Msg.ContentJson))
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range staged.Records {
		if record.Kind == "intake_event" {
			var got map[string]any
			if err := json.Unmarshal(record.Data, &got); err != nil {
				t.Fatal(err)
			}
			if got["eventId"] != "meal-1" || got["recipeId"] != "recipe-old" || got["recipeRevision"] != float64(4) || got["amount"] != "22.125" || got["reason"] != "actual portion" {
				t.Fatalf("event=%s", record.Data)
			}
			return
		}
	}
	t.Fatal("workspace export omitted intake event")
}

func TestWorkspaceExportIncludesSupplementScheduleHistoryAndOpaqueProductRevisions(t *testing.T) {
	h := checkpointHandler(t)
	h.profiles = nil
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "owner", Verified: true})
	if _, err := h.plans.Apply(ctx, "w1", 0, `{"occurrences":[{"date":"2026-10-07","slotName":"dinner","mode":"fixed","quantity":"1"}]}`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := h.plans.Get(ctx, "w1"); err != nil {
		t.Fatalf("seed plan: %v", err)
	}
	dose, _ := decimalx.Parse("1.125")
	created, err := h.supplements.Create(ctx, supplement.Schedule{ID: "supplement-schedule-1", WorkspaceID: "w1", ProductRevisionID: "catalog-revision-not-loaded", Dose: dose, DoseUnit: "capsule", Weekdays: []int{1, 5}, StartDate: "2026-01-01", EndDate: "2026-12-31", Confirmed: true, CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	updatedDose, _ := decimalx.Parse("2.875")
	if _, err := h.supplements.Update(ctx, supplement.UpdateInput{WorkspaceID: "w1", ID: created.ID, ExpectedRevision: 1, Dose: updatedDose, DoseUnit: "tablet", Weekdays: []int{0, 6}, StartDate: "2026-03-01", EndDate: "2026-10-31", Paused: true, Confirmed: false}); err != nil {
		t.Fatal(err)
	}
	response, err := h.ExportWorkspace(ctx, connect.NewRequest(&v1.ExportWorkspaceRequest{WorkspaceId: "w1"}))
	if err != nil {
		t.Fatal(err)
	}
	staged, err := internalPortability.ImportWorkspace([]byte(response.Msg.ContentJson))
	if err != nil {
		t.Fatal(err)
	}
	var revisions, pointers int
	for _, record := range staged.Records {
		if record.Kind == "supplement_schedule" {
			pointers++
			if record.Revision != 2 {
				t.Fatalf("current revision=%d", record.Revision)
			}
		}
		if record.Kind == "supplement_schedule_revision" {
			revisions++
			var item struct {
				WorkspaceID       string `json:"workspaceId"`
				ProductRevisionID string `json:"productRevisionId"`
			}
			if err := json.Unmarshal(record.Data, &item); err != nil {
				t.Fatal(err)
			}
			if item.ProductRevisionID == "" || item.WorkspaceID != "w1" {
				t.Fatalf("portable schedule=%#v", item)
			}
		}
	}
	if revisions != 2 || pointers != 1 || strings.Contains(strings.Join(response.Msg.Omissions, ","), "supplements") {
		t.Fatalf("revisions=%d pointers=%d omissions=%v", revisions, pointers, response.Msg.Omissions)
	}
}

func TestWorkspaceIntakeApplyRequiresDestinationAuthority(t *testing.T) {
	h := checkpointHandler(t)
	owner := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "owner", Verified: true})
	outsider := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "intruder", Verified: true})
	amount, _ := decimalx.Parse("5.75")
	if err := h.intakes.Append(owner, "w1", nutrition.IntakeEvent{ID: "destination", Date: "2026-10-06", NutrientID: "protein", Amount: amount, Unit: "g"}); err != nil {
		t.Fatal(err)
	}
	sourceAmount, _ := decimalx.Parse("9.125")
	content, err := internalPortability.Workspace(internalPortability.WorkspaceSnapshot{ID: "source", IntakeEvents: []nutrition.IntakeEvent{{ID: "incoming", WorkspaceID: "source", Date: "2026-10-07", NutrientID: "protein", Amount: sourceAmount, Unit: "g", Reason: "source record"}}})
	if err != nil {
		t.Fatal(err)
	}
	request := func(key string) *connect.Request[v1.ApplyWorkspaceImportRequest] {
		return connect.NewRequest(&v1.ApplyWorkspaceImportRequest{WorkspaceId: "w1", ContentJson: string(content), ExpectedWorkspaceRevision: 2, IdempotencyKey: key})
	}
	if _, err := h.ApplyWorkspaceImport(outsider, request("denied-intake")); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("unauthorized apply err=%v", err)
	}
	prior, err := h.intakes.List(owner, "w1")
	if err != nil || len(prior) != 1 || prior[0].ID != "destination" {
		t.Fatalf("unauthorized apply mutated intake: %+v err=%v", prior, err)
	}
	if _, err := h.ApplyWorkspaceImport(owner, request("authorized-intake")); err != nil {
		t.Fatal(err)
	}
	got, err := h.intakes.List(owner, "w1")
	if err != nil || len(got) != 1 || got[0].ID != "incoming" || got[0].Amount.String() != "9.125" {
		t.Fatalf("authorized apply intake=%+v err=%v", got, err)
	}
}

func TestWorkspaceExportIncludesOwnedProfilePreferences(t *testing.T) {
	h := checkpointHandler(t)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "owner", Verified: true})
	want, err := h.profiles.Apply(ctx, internalProfile.ApplyInput{WorkspaceID: "w1", Preset: "vegetarian", ExcludedGroups: []string{"nightshades"}, Allergies: []string{"peanut"}, Appliances: []string{"oven"}, CostWeight: .2, EffortWeight: .3, VarietyWeight: .5})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.profiles.SaveDraft(ctx, "w1", `{"servings":3}`); err != nil {
		t.Fatal(err)
	}
	response, err := h.ExportWorkspace(ctx, connect.NewRequest(&v1.ExportWorkspaceRequest{WorkspaceId: "w1"}))
	if err != nil {
		t.Fatal(err)
	}
	staged, err := internalPortability.ImportWorkspace([]byte(response.Msg.ContentJson))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(staged.Manifest.Omissions, ","), "profile") {
		t.Fatalf("profile is still omitted: %v", staged.Manifest.Omissions)
	}
	var got internalProfile.Profile
	for _, record := range staged.Records {
		if record.Kind == "profile" {
			if err := json.Unmarshal(record.Data, &got); err != nil {
				t.Fatal(err)
			}
		}
	}
	want, err = h.profiles.Get(ctx, "w1")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("profile export differs\nwant=%#v\ngot=%#v", want, got)
	}
}

func TestWorkspaceProfileApplyRequiresDestinationAuthorityAndRemapsIdentity(t *testing.T) {
	h := checkpointHandler(t)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "owner", Verified: true})
	prior, err := h.profiles.Apply(ctx, internalProfile.ApplyInput{WorkspaceID: "w1", Preset: "vegetarian", Allergies: []string{"peanut"}, CostWeight: .2, EffortWeight: .3, VarietyWeight: .5})
	if err != nil {
		t.Fatal(err)
	}
	source := internalProfile.Profile{WorkspaceID: "source", Revision: 8, Preset: "vegan", PresetVersion: internalProfile.CurrentPresetVersion, ActiveRules: internalProfile.ExpandPreset("vegan"), ExcludedGroups: []string{"nightshades"}, Allergies: []string{"sesame"}, Appliances: []string{"oven"}, CostWeight: .25, EffortWeight: .25, VarietyWeight: .5, DraftJSON: `{"servings":2}`}
	content, err := internalPortability.Workspace(internalPortability.WorkspaceSnapshot{ID: "source", Profile: &source})
	if err != nil {
		t.Fatal(err)
	}
	unauthorized := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "intruder", Verified: true})
	_, err = h.ApplyWorkspaceImport(unauthorized, connect.NewRequest(&v1.ApplyWorkspaceImportRequest{WorkspaceId: "w1", ContentJson: string(content), ExpectedWorkspaceRevision: 2, IdempotencyKey: "denied-profile"}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("unauthorized apply error=%v", err)
	}
	unchanged, err := h.profiles.Get(ctx, "w1")
	if err != nil || !reflect.DeepEqual(unchanged, prior) {
		t.Fatalf("unauthorized apply changed profile: %#v err=%v", unchanged, err)
	}
	if _, err := h.ApplyWorkspaceImport(ctx, connect.NewRequest(&v1.ApplyWorkspaceImportRequest{WorkspaceId: "w1", ContentJson: string(content), ExpectedWorkspaceRevision: 2, IdempotencyKey: "authorized-profile"})); err != nil {
		t.Fatal(err)
	}
	got, err := h.profiles.Get(ctx, "w1")
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkspaceID != "w1" || got.Revision != source.Revision || got.Preset != source.Preset || got.DraftJSON != source.DraftJSON || !reflect.DeepEqual(got.Allergies, source.Allergies) {
		t.Fatalf("authorized apply did not remap profile: %#v", got)
	}
}

func TestWorkspaceTargetPreviewValidatesWithoutMutation(t *testing.T) {
	h := checkpointHandler(t)
	start, _ := time.Parse(time.RFC3339, "2026-10-01T00:00:00Z")
	target := nutrition.Target{ID: "t1", Revision: 2, WorkspaceID: "source", NutrientID: "protein", Lower: decimalx.KnownInt(70), Upper: decimalx.KnownInt(100), Period: "local_day", Scope: "planned_day", Enforcement: "required", Provenance: "user_assertion:test", EffectiveFrom: start, Active: true}
	content, err := internalPortability.Workspace(internalPortability.WorkspaceSnapshot{ID: "source", NutritionTargets: []nutrition.Target{target}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "owner", Verified: true})
	before, err := h.targets.List(ctx, "w1")
	if err != nil {
		t.Fatal(err)
	}
	response, err := h.PreviewWorkspaceImport(ctx, connect.NewRequest(&v1.PreviewWorkspaceImportRequest{WorkspaceId: "w1", ContentJson: string(content)}))
	if err != nil || !response.Msg.Valid || strings.Contains(strings.Join(response.Msg.Omissions, ","), "nutrition_targets") || !strings.Contains(strings.Join(response.Msg.RecordKinds, ","), "nutrition_target") {
		t.Fatalf("preview=%#v err=%v", response, err)
	}
	after, err := h.targets.List(ctx, "w1")
	if err != nil || len(after) != len(before) {
		t.Fatalf("preview mutated target set: before=%#v after=%#v err=%v", before, after, err)
	}
	bad := strings.Replace(string(content), `"kind":"nutrition_target"`, `"kind":"unsupported_target"`, 1)
	invalid, err := h.PreviewWorkspaceImport(ctx, connect.NewRequest(&v1.PreviewWorkspaceImportRequest{WorkspaceId: "w1", ContentJson: bad}))
	if err != nil || invalid.Msg.Valid {
		t.Fatalf("invalid preview=%#v err=%v", invalid, err)
	}
}

func TestCheckpointEndpointsRequireAuthorizedWorkspace(t *testing.T) {
	h := checkpointHandler(t)
	get := func(ctx context.Context, id string) error {
		_, err := h.GetRestoreCheckpoint(ctx, connect.NewRequest(&v1.GetRestoreCheckpointRequest{WorkspaceId: "w1", CheckpointId: id}))
		return err
	}
	if err := get(context.Background(), "cp1"); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("anonymous checkpoint read code=%v err=%v", connect.CodeOf(err), err)
	}
	other := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "other", Verified: true})
	if err := get(other, "cp1"); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("foreign workspace checkpoint read code=%v err=%v", connect.CodeOf(err), err)
	}
	owner := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "owner", Verified: true})
	if err := get(owner, "cp1"); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing checkpoint code=%v err=%v", connect.CodeOf(err), err)
	}
	_, err := h.RecoverRestoreCheckpoint(owner, connect.NewRequest(&v1.RecoverRestoreCheckpointRequest{WorkspaceId: "w1", CheckpointId: "cp1", ExpectedWorkspaceRevision: 2, IdempotencyKey: "key"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing checkpoint recovery code=%v err=%v", connect.CodeOf(err), err)
	}
	_, err = h.RecoverRestoreCheckpoint(context.Background(), connect.NewRequest(&v1.RecoverRestoreCheckpointRequest{WorkspaceId: "w1", CheckpointId: "cp1", ExpectedWorkspaceRevision: 2, IdempotencyKey: "key"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("anonymous recovery code=%v err=%v", connect.CodeOf(err), err)
	}
}

func TestRecipeAndWorkspaceExportsReadEveryStoredRevision(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, schema := range []string{workspace.Schema(), recipe.Schema(), planning.Schema(), nutrition.Schema(), internalPortability.Schema(), shopping.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	clock := schedule.System()
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	owned, err := workspaces.Create(context.Background(), workspace.CreateInput{Name: "Daily", OwnerSubject: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	recipes := recipe.NewService(recipe.NewSQLiteRepository(db, clock))
	created, err := recipes.Create(context.Background(), recipe.CreateInput{WorkspaceID: owned.ID, Name: "First", OriginalText: "original text", SourceType: "web", SourceURL: "https://example.test/source", Ingredients: []recipe.Ingredient{{ID: "rice", Name: "Rice", Amount: "½", Unit: "cup"}}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = recipes.Update(context.Background(), recipe.UpdateInput{WorkspaceID: owned.ID, ID: created.ID, ExpectedRevision: 1, Name: "Second", OriginalText: "new original text", SourceType: "web", SourceURL: "https://example.test/source", Ingredients: []recipe.Ingredient{{ID: "rice", Name: "Rice", Amount: "¾", Unit: "cup"}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		id                                                           string
		rev                                                          int
		lower, upper, period, scope, enforce, provenance, start, end string
		active                                                       int
	}{
		{"target", 1, "60.25", "90", "local_day", "planned_day", "preferred", "clinician:case-1", "2026-01-01T00:00:00Z", "2026-09-01T00:00:00Z", 0},
		{"retired", 1, "55.5", "85", "rolling_days", "recorded_so_far", "informational", "user_assertion:retired", "2026-02-01T00:00:00Z", "2026-08-01T00:00:00Z", 0},
		{"target", 2, "70.125", "unknown", "calendar_week", "selected_week", "required", "user_assertion:revision-2", "2026-10-01T00:00:00Z", "", 1},
	} {
		var end any
		if row.end != "" {
			end = row.end
		}
		if _, err := db.Exec(`INSERT INTO nutrition_targets(id,revision,workspace_id,nutrient_id,lower_bound,upper_bound,period,scope,enforcement,provenance,effective_from,effective_to,active) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, row.id, row.rev, owned.ID, "protein", row.lower, row.upper, row.period, row.scope, row.enforce, row.provenance, row.start, end, row.active); err != nil {
			t.Fatal(err)
		}
	}
	targets := nutrition.NewSQLiteTargetRepository(db, clock)
	h := NewConnectHandlerWithDomains(recipes, workspaces, planning.NewSQLiteRepository(db, clock), nil, targets, internalPortability.NewSQLiteRestorer(db, clock), log.Default())
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "owner", Verified: true})
	check := func(content string, format string) {
		t.Helper()
		var exported internalPortability.Export
		if err := json.Unmarshal([]byte(content), &exported); err != nil {
			t.Fatal(err)
		}
		if exported.Format != format || exported.Manifest.RecordCount != len(exported.Records) {
			t.Fatalf("format=%q manifest=%d records=%d", exported.Format, exported.Manifest.RecordCount, len(exported.Records))
		}
		count := 0
		latest := int64(0)
		for _, record := range exported.Records {
			if record.Kind == "recipe_revision" {
				count++
				if record.Revision > latest {
					latest = record.Revision
				}
			}
		}
		if count != 2 || latest != 2 {
			t.Fatalf("%s revisions=%d latest=%d", format, count, latest)
		}
	}
	recipeExport, err := h.ExportRecipes(ctx, connect.NewRequest(&v1.ExportRecipesRequest{WorkspaceId: owned.ID}))
	if err != nil {
		t.Fatal(err)
	}
	check(recipeExport.Msg.ContentJson, "daily.recipes")
	workspaceExport, err := h.ExportWorkspace(ctx, connect.NewRequest(&v1.ExportWorkspaceRequest{WorkspaceId: owned.ID}))
	if err != nil {
		t.Fatal(err)
	}
	check(workspaceExport.Msg.ContentJson, "daily.workspace")
	var staged internalPortability.Export
	if err := json.Unmarshal([]byte(workspaceExport.Msg.ContentJson), &staged); err != nil {
		t.Fatal(err)
	}
	var targetCount int
	for _, record := range staged.Records {
		if record.Kind == "nutrition_target" {
			targetCount++
		}
	}
	if targetCount != 3 || strings.Contains(strings.Join(workspaceExport.Msg.Omissions, ","), "nutrition_targets") {
		t.Fatalf("target export count=%d omissions=%v", targetCount, workspaceExport.Msg.Omissions)
	}
}

func TestWorkspaceExportIncludesAllPersistedShoppingRows(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	for _, schema := range []string{workspace.Schema(), recipe.Schema(), planning.Schema(), internalPortability.Schema(), shopping.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	clock := schedule.System()
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	owned, err := workspaces.Create(context.Background(), workspace.CreateInput{Name: "Daily", OwnerSubject: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		key   string
		value int
	}{{"ingredient:rice", 1}, {"ingredient:beans", 0}} {
		if _, err := db.Exec(`INSERT INTO shopping_checks(workspace_id,line_key,checked,updated_at) VALUES(?,?,?,?)`, owned.ID, row.key, row.value, "now"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO shopping_have_this(workspace_id,line_key,asserted,updated_at) VALUES(?,?,?,?)`, owned.ID, row.key, row.value, "now"); err != nil {
			t.Fatal(err)
		}
	}
	for _, review := range []struct{ id, hash, time string }{{"trip-a", "hash-a", "2026-10-01"}, {"trip-b", "hash-b", "2026-10-02"}} {
		if _, err := db.Exec(`INSERT INTO shopping_purchase_reviews(workspace_id,review_id,payload_hash,created_at) VALUES(?,?,?,?)`, owned.ID, review.id, review.hash, review.time); err != nil {
			t.Fatal(err)
		}
		omitted := 0
		amount := "2"
		if review.id == "trip-b" {
			omitted = 1
			amount = ""
		}
		if _, err := db.Exec(`INSERT INTO shopping_purchase_review_lines(workspace_id,review_id,line_key,item_id,amount,unit,price,omitted) VALUES(?,?,?,?,?,?,?,?)`, owned.ID, review.id, "ingredient:rice", "rice", amount, "kg", "4.00", omitted); err != nil {
			t.Fatal(err)
		}
	}
	recipes := recipe.NewService(recipe.NewSQLiteRepository(db, clock))
	h := NewConnectHandlerWithRestorer(recipes, workspaces, planning.NewSQLiteRepository(db, clock), shopping.NewSQLiteRepository(db, clock), internalPortability.NewSQLiteRestorer(db, clock), log.Default())
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Subject: "owner", Verified: true})
	response, err := h.ExportWorkspace(ctx, connect.NewRequest(&v1.ExportWorkspaceRequest{WorkspaceId: owned.ID}))
	if err != nil {
		t.Fatal(err)
	}
	staged, err := internalPortability.ImportWorkspace([]byte(response.Msg.ContentJson))
	if err != nil {
		t.Fatal(err)
	}
	var got shopping.PersistedState
	for _, record := range staged.Records {
		if record.Kind == "shopping_state" {
			if err := json.Unmarshal(record.Data, &got); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(got.Checks) != 2 || !got.Checks["ingredient:rice"] || got.Checks["ingredient:beans"] || len(got.HaveThis) != 2 || !got.HaveThis["ingredient:rice"] || got.HaveThis["ingredient:beans"] || len(got.Reviews) != 2 || len(got.Reviews[1].Lines) != 1 {
		t.Fatalf("shopping export lost persisted state: %#v", got)
	}
	if !strings.Contains(strings.Join(staged.Manifest.Omissions, ","), "inventory") {
		t.Fatalf("inventory omission missing: %v", staged.Manifest.Omissions)
	}
}

func TestPopulatedWorkspaceTransferReExportAndCheckpointRecovery(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	for _, schema := range []string{workspace.Schema(), recipe.Schema(), planning.Schema(), nutrition.Schema(), internalProfile.Schema(), shopping.Schema(), supplement.Schema(), internalPortability.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	clock := schedule.System()
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	sourceWorkspace, err := workspaces.Create(context.Background(), workspace.CreateInput{Name: "Source", OwnerSubject: "source-owner"})
	if err != nil {
		t.Fatal(err)
	}
	destinationWorkspace, err := workspaces.Create(context.Background(), workspace.CreateInput{Name: "Destination", OwnerSubject: "destination-owner"})
	if err != nil {
		t.Fatal(err)
	}
	recipes := recipe.NewService(recipe.NewSQLiteRepository(db, clock))
	plans := planning.NewSQLiteRepository(db, clock)
	targets := nutrition.NewSQLiteTargetRepository(db, clock)
	profiles := internalProfile.NewService(internalProfile.NewSQLiteRepository(db, clock))
	intakes := nutrition.NewSQLiteIntakeRepository(db)
	supplements := supplement.NewSQLiteRepository(db, clock)
	restore := internalPortability.NewSQLiteRestorer(db, clock)
	h := NewConnectHandlerWithSchedules(recipes, workspaces, plans, shopping.NewSQLiteRepository(db, clock), targets, profiles, intakes, supplements, restore, log.Default())
	owner := func(subject string) context.Context {
		return identity.WithPrincipal(context.Background(), identity.Principal{Subject: subject, Verified: true})
	}
	seed := func(id, subject, prefix string) {
		t.Helper()
		ctx := owner(subject)
		created, err := recipes.Create(ctx, recipe.CreateInput{WorkspaceID: id, Name: prefix + " first", OriginalText: "source text", SourceType: "manual", Ingredients: []recipe.Ingredient{{ID: "rice", Name: "Rice", Amount: "½", Unit: "cup"}}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := recipes.Update(ctx, recipe.UpdateInput{WorkspaceID: id, ID: created.ID, ExpectedRevision: 1, Name: prefix + " current", OriginalText: "revised source text", SourceType: "manual", Ingredients: []recipe.Ingredient{{ID: "rice", Name: "Rice", Amount: "¾", Unit: "cup"}}}); err != nil {
			t.Fatal(err)
		}
		if _, err := plans.Apply(ctx, id, 0, `{"occurrences":[{"date":"2026-10-07","slotName":"dinner","mode":"fixed","recipeId":"`+created.ID+`"}]}`); err != nil {
			t.Fatal(err)
		}
		if _, err := profiles.Apply(ctx, internalProfile.ApplyInput{WorkspaceID: id, Preset: "vegan", Allergies: []string{prefix + "-allergy"}, CostWeight: .2, EffortWeight: .3, VarietyWeight: .5}); err != nil {
			t.Fatal(err)
		}
		from, _ := time.Parse(time.RFC3339, "2026-10-01T00:00:00Z")
		if _, err := targets.Create(ctx, nutrition.Target{ID: prefix + "-target", WorkspaceID: id, NutrientID: "protein", Lower: decimalx.KnownInt(60), Upper: decimalx.KnownInt(90), Period: "local_day", Scope: "planned_day", Enforcement: "required", Provenance: "user_assertion:" + prefix, EffectiveFrom: from, Active: true}); err != nil {
			t.Fatal(err)
		}
		amount, _ := decimalx.Parse("12.5")
		if err := intakes.Append(ctx, id, nutrition.IntakeEvent{ID: prefix + "-meal", Date: "2026-10-07", RecipeID: created.ID, RecipeRevision: 2, NutrientID: "protein", Amount: amount, Unit: "g", Reason: "measured"}); err != nil {
			t.Fatal(err)
		}
		correction, _ := decimalx.Parse("13.25")
		if err := intakes.Append(ctx, id, nutrition.IntakeEvent{ID: prefix + "-correction", Date: "2026-10-07", RecipeID: created.ID, RecipeRevision: 2, NutrientID: "protein", Amount: correction, Unit: "g", Reason: "corrected amount", CorrectionOf: prefix + "-meal"}); err != nil {
			t.Fatal(err)
		}
		dose, _ := decimalx.Parse("1.25")
		scheduleItem, err := supplements.Create(ctx, supplement.Schedule{ID: prefix + "-supplement", WorkspaceID: id, ProductRevisionID: "catalog-revision-stays-opaque", Dose: dose, DoseUnit: "capsule", Weekdays: []int{1, 5}, StartDate: "2026-01-01", EndDate: "2026-12-31", Confirmed: true})
		if err != nil {
			t.Fatal(err)
		}
		updated, _ := decimalx.Parse("2.5")
		if _, err := supplements.Update(ctx, supplement.UpdateInput{WorkspaceID: id, ID: scheduleItem.ID, ExpectedRevision: 1, Dose: updated, DoseUnit: "tablet", Weekdays: []int{2, 6}, StartDate: "2026-02-01", EndDate: "2026-11-30", Paused: true, Confirmed: false}); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO shopping_checks(workspace_id,line_key,checked,updated_at) VALUES(?,?,1,'seed-time')`, id, prefix+":rice"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO shopping_have_this(workspace_id,line_key,asserted,updated_at) VALUES(?,?,1,'seed-time')`, id, prefix+":beans"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO shopping_purchase_reviews(workspace_id,review_id,payload_hash,created_at) VALUES(?,?,?,'review-time')`, id, prefix+"-trip", prefix+"-hash"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO shopping_purchase_review_lines(workspace_id,review_id,line_key,item_id,amount,unit,price,omitted) VALUES(?,?,?,'rice','2','kg','4.00',0)`, id, prefix+"-trip", prefix+":rice"); err != nil {
			t.Fatal(err)
		}
	}
	seed(sourceWorkspace.ID, "source-owner", "source")
	seed(destinationWorkspace.ID, "destination-owner", "destination")
	export := func(id, subject string) internalPortability.Export {
		t.Helper()
		response, err := h.ExportWorkspace(owner(subject), connect.NewRequest(&v1.ExportWorkspaceRequest{WorkspaceId: id}))
		if err != nil {
			t.Fatal(err)
		}
		staged, err := internalPortability.ImportWorkspace([]byte(response.Msg.ContentJson))
		if err != nil {
			t.Fatal(err)
		}
		return staged
	}
	source := export(sourceWorkspace.ID, "source-owner")
	destinationBefore := export(destinationWorkspace.ID, "destination-owner")
	preview, err := h.PreviewWorkspaceImport(owner("destination-owner"), connect.NewRequest(&v1.PreviewWorkspaceImportRequest{WorkspaceId: destinationWorkspace.ID, ContentJson: mustJSON(t, source)}))
	if err != nil || !preview.Msg.Valid || int(preview.Msg.RecordCount) != len(source.Records) {
		t.Fatalf("preview=%#v err=%v", preview, err)
	}
	if !reflect.DeepEqual(preview.Msg.Omissions, source.Manifest.Omissions) {
		t.Fatalf("staged preview misrepresented omissions: got=%v want=%v", preview.Msg.Omissions, source.Manifest.Omissions)
	}
	if got, want := portableRecords(export(destinationWorkspace.ID, "destination-owner"), destinationWorkspace.ID, destinationWorkspace.ID, "", ""), portableRecords(destinationBefore, destinationWorkspace.ID, destinationWorkspace.ID, "", ""); !reflect.DeepEqual(got, want) {
		t.Fatal("staging the import mutated the destination")
	}
	firstApply, err := h.ApplyWorkspaceImport(owner("destination-owner"), connect.NewRequest(&v1.ApplyWorkspaceImportRequest{WorkspaceId: destinationWorkspace.ID, ContentJson: mustJSON(t, source), ExpectedWorkspaceRevision: 1, IdempotencyKey: "populated-transfer"}))
	if err != nil {
		t.Fatal(err)
	}
	transferred := export(destinationWorkspace.ID, "destination-owner")
	if len(transferred.Records) != len(source.Records) || !reflect.DeepEqual(transferred.Manifest.RecordKinds, source.Manifest.RecordKinds) {
		t.Fatalf("source kinds=%v destination kinds=%v", source.Manifest.RecordKinds, transferred.Manifest.RecordKinds)
	}
	var destRecipe string
	for _, record := range transferred.Records {
		if record.Kind == "recipe_revision" && record.Revision == 2 {
			var item recipe.Recipe
			if err := json.Unmarshal(record.Data, &item); err != nil {
				t.Fatal(err)
			}
			if item.Name == "source current" {
				destRecipe = record.ID
			}
		}
	}
	var sourceRecipe string
	for _, record := range source.Records {
		if record.Kind == "recipe_revision" && record.Revision == 2 {
			var item recipe.Recipe
			_ = json.Unmarshal(record.Data, &item)
			if item.Name == "source current" {
				sourceRecipe = record.ID
			}
		}
	}
	if destRecipe == "" || sourceRecipe == "" || destRecipe == sourceRecipe {
		t.Fatalf("recipe reference was not remapped: source=%q destination=%q", sourceRecipe, destRecipe)
	}
	var linked int
	for _, record := range transferred.Records {
		if record.Kind == "intake_event" {
			var item struct {
				RecipeID     string `json:"recipeId"`
				CorrectionOf string `json:"correctionOf"`
			}
			if err := json.Unmarshal(record.Data, &item); err != nil {
				t.Fatal(err)
			}
			if item.RecipeID == destRecipe {
				linked++
			}
			if item.CorrectionOf == "source-meal" && item.RecipeID == destRecipe {
				linked++
			}
		}
	}
	if linked != 3 {
		t.Fatalf("source recipe/intake correction references were not preserved: linked=%d", linked)
	}
	if got, want := portableRecords(transferred, sourceWorkspace.ID, destinationWorkspace.ID, sourceRecipe, destRecipe), portableRecords(source, sourceWorkspace.ID, destinationWorkspace.ID, sourceRecipe, destRecipe); !reflect.DeepEqual(got, want) {
		t.Fatalf("re-export differs from portable source state\nwant=%s\ngot=%s", mustJSON(t, want), mustJSON(t, got))
	}
	if firstApply.Msg.CheckpointId == "" {
		t.Fatal("apply did not retain a destination checkpoint")
	}
	if _, err := h.RecoverRestoreCheckpoint(owner("destination-owner"), connect.NewRequest(&v1.RecoverRestoreCheckpointRequest{WorkspaceId: destinationWorkspace.ID, CheckpointId: firstApply.Msg.CheckpointId, ExpectedWorkspaceRevision: 2, IdempotencyKey: "populated-recovery"})); err != nil {
		t.Fatal(err)
	}
	recovered := export(destinationWorkspace.ID, "destination-owner")
	if len(recovered.Records) != len(destinationBefore.Records) {
		t.Fatalf("recovery records=%d before=%d", len(recovered.Records), len(destinationBefore.Records))
	}
	if got, want := portableRecords(recovered, destinationWorkspace.ID, destinationWorkspace.ID, "", ""), portableRecords(destinationBefore, destinationWorkspace.ID, destinationWorkspace.ID, "", ""); !reflect.DeepEqual(got, want) {
		t.Fatalf("checkpoint recovery lost prior portable state\nwant=%s\ngot=%s", mustJSON(t, want), mustJSON(t, got))
	}
	if _, err := h.ExportWorkspace(owner("source-owner"), connect.NewRequest(&v1.ExportWorkspaceRequest{WorkspaceId: destinationWorkspace.ID})); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("source identity gained destination authority: %v", err)
	}
	if got, want := portableRecords(export(sourceWorkspace.ID, "source-owner"), sourceWorkspace.ID, sourceWorkspace.ID, "", ""), portableRecords(source, sourceWorkspace.ID, sourceWorkspace.ID, "", ""); !reflect.DeepEqual(got, want) {
		t.Fatal("destination transfer or recovery changed the isolated source workspace")
	}
}

func portableRecords(envelope internalPortability.Export, sourceWorkspace, destinationWorkspace, sourceRecipe, destinationRecipe string) []internalPortability.Record {
	var records []internalPortability.Record
	for _, record := range envelope.Records {
		if record.Kind == "workspace" {
			continue
		} // Owner, name, and workspace revision belong to the destination.
		record.ID = strings.ReplaceAll(record.ID, sourceWorkspace, destinationWorkspace)
		if record.Kind == "recipe" || record.Kind == "recipe_revision" {
			record.ID = strings.ReplaceAll(record.ID, sourceRecipe, destinationRecipe)
		}
		if record.Kind == "nutrition_target" || record.Kind == "supplement_schedule" || record.Kind == "supplement_schedule_revision" {
			record.ID = "portable-domain-id"
		}
		data := strings.ReplaceAll(string(record.Data), sourceWorkspace, destinationWorkspace)
		data = strings.ReplaceAll(data, sourceRecipe, destinationRecipe)
		if record.Kind == "nutrition_target" {
			var target map[string]any
			if err := json.Unmarshal([]byte(data), &target); err == nil {
				target["ID"] = "portable-domain-id"
				raw, _ := json.Marshal(target)
				data = string(raw)
			}
		}
		if record.Kind == "plan" {
			record.Revision = 0
			var plan map[string]any
			if err := json.Unmarshal([]byte(data), &plan); err == nil {
				plan["revision"] = float64(0)
				raw, _ := json.Marshal(plan)
				data = string(raw)
			}
		}
		record.Data = json.RawMessage(data)
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Kind != records[j].Kind {
			return records[i].Kind < records[j].Kind
		}
		if records[i].ID != records[j].ID {
			return records[i].ID < records[j].ID
		}
		return records[i].Revision < records[j].Revision
	})
	return records
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
