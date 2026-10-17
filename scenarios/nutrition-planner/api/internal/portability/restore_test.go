package portability

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/inventory"
	"nutrition-planner/internal/nutrition"
	"nutrition-planner/internal/planning"
	"nutrition-planner/internal/profile"
	"nutrition-planner/internal/recipe"
	"nutrition-planner/internal/shopping"
	"nutrition-planner/internal/supplement"
	"nutrition-planner/internal/workspace"
)

func restoreDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	for _, schema := range []string{workspace.Schema(), recipe.Schema(), planning.Schema(), nutrition.Schema(), profile.Schema(), Schema(), shopping.Schema(), inventory.Schema(), supplement.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO workspaces(id,name,owner_subject,revision,created_at,updated_at) VALUES('w1','Daily','actor',3,'now','now')`); err != nil {
		t.Fatal(err)
	}
	return db
}

func assertScheduleRevisionEqual(t *testing.T, got, want supplement.Schedule) {
	t.Helper()
	if got.ID != want.ID || got.Revision != want.Revision || got.WorkspaceID != want.WorkspaceID || got.ProductRevisionID != want.ProductRevisionID || got.Dose.String() != want.Dose.String() || got.DoseUnit != want.DoseUnit || !reflect.DeepEqual(got.Weekdays, want.Weekdays) || got.StartDate != want.StartDate || got.EndDate != want.EndDate || got.Paused != want.Paused || got.Confirmed != want.Confirmed || !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("schedule revision mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestSupplementScheduleRevisionRoundTripAndCheckpointRecovery(t *testing.T) {
	db := restoreDB(t)
	ctx := context.Background()
	clock := schedule.System()
	repo := supplement.NewSQLiteRepository(db, clock)
	seed := func(id, product string, dose string) supplement.Schedule {
		d, err := decimalx.Parse(dose)
		if err != nil {
			t.Fatal(err)
		}
		item, err := repo.Create(ctx, supplement.Schedule{ID: id, WorkspaceID: "w1", ProductRevisionID: product, Dose: d, DoseUnit: "capsule", Weekdays: []int{1, 3, 5}, StartDate: "2026-01-01", EndDate: "2026-12-31", Confirmed: true, CreatedAt: time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)})
		if err != nil {
			t.Fatal(err)
		}
		return item
	}
	prior := seed("schedule-prior", "old-product-revision", "1.25")
	priorNextDose, _ := decimalx.Parse("1.75")
	priorNext, err := repo.Update(ctx, supplement.UpdateInput{WorkspaceID: "w1", ID: prior.ID, ExpectedRevision: 1, Dose: priorNextDose, DoseUnit: "tablet", Weekdays: []int{2, 4}, StartDate: "2026-02-01", EndDate: "2026-10-31", Paused: true, Confirmed: false})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workspaces(id,name,owner_subject,revision,created_at,updated_at) VALUES('w2','Other','actor',1,'now','now')`); err != nil {
		t.Fatal(err)
	}
	foreignDose, _ := decimalx.Parse("1")
	if _, err := repo.Create(ctx, supplement.Schedule{ID: "schedule-shared", WorkspaceID: "w2", ProductRevisionID: "foreign-product", Dose: foreignDose, DoseUnit: "capsule", Weekdays: []int{1}, StartDate: "2026-01-01", Confirmed: true}); err != nil {
		t.Fatal(err)
	}
	incomingOne := supplement.Schedule{ID: "schedule-shared", Revision: 1, WorkspaceID: "source", ProductRevisionID: "catalog-product-revision:unresolved-A", Dose: decimalx.KnownInt(2), DoseUnit: "ml", Weekdays: []int{0, 6}, StartDate: "2026-03-01", EndDate: "2026-09-01", Confirmed: true, CreatedAt: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)}
	incomingTwo := incomingOne
	incomingTwo.Revision, incomingTwo.ProductRevisionID = 2, "catalog-product-revision:unresolved-B"
	incomingTwo.Dose = decimalx.KnownInt(3)
	incomingTwo.DoseUnit, incomingTwo.Weekdays = "softgel", []int{2, 5}
	incomingTwo.StartDate, incomingTwo.EndDate = "2026-04-01", "2026-11-30"
	incomingTwo.Paused, incomingTwo.Confirmed = true, false
	incomingTwo.CreatedAt = time.Date(2026, 4, 2, 11, 12, 13, 987654321, time.UTC)
	content, err := Workspace(WorkspaceSnapshot{ID: "source", SupplementSchedules: []supplement.Schedule{incomingOne, incomingTwo}})
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := ImportWorkspace(content)
	if err != nil {
		t.Fatal(err)
	}
	restore := NewSQLiteRestorer(db, clock)
	result, err := restore.ApplyWorkspace(ctx, "w1", 3, "supplement-import", envelope)
	if err != nil {
		t.Fatal(err)
	}
	if result.CheckpointID == "" {
		t.Fatal("restore omitted checkpoint")
	}
	current, err := repo.List(ctx, "w1")
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != 1 || current[0].ID == incomingOne.ID || current[0].Revision != 2 {
		t.Fatalf("current schedule after collision remap: %#v", current)
	}
	importedID := current[0].ID
	gotOne, err := repo.Get(ctx, importedID, "w1", 1)
	if err != nil {
		t.Fatal(err)
	}
	gotTwo, err := repo.Get(ctx, importedID, "w1", 2)
	if err != nil {
		t.Fatal(err)
	}
	wantOne, wantTwo := incomingOne, incomingTwo
	wantOne.ID, wantTwo.ID = importedID, importedID
	wantOne.WorkspaceID, wantTwo.WorkspaceID = "w1", "w1"
	assertScheduleRevisionEqual(t, gotOne, wantOne)
	assertScheduleRevisionEqual(t, gotTwo, wantTwo)
	all := []supplement.Schedule{gotOne, gotTwo}
	reexport, err := Workspace(WorkspaceSnapshot{ID: "w1", SupplementSchedules: all})
	if err != nil {
		t.Fatal(err)
	}
	restaged, err := ImportWorkspace(reexport)
	if err != nil {
		t.Fatal(err)
	}
	var seenProducts []string
	for _, record := range restaged.Records {
		if record.Kind == "supplement_schedule_revision" {
			var item supplementScheduleRevisionRecord
			if err := json.Unmarshal(record.Data, &item); err != nil {
				t.Fatal(err)
			}
			seenProducts = append(seenProducts, item.ProductRevisionID)
		}
	}
	if !reflect.DeepEqual(seenProducts, []string{incomingOne.ProductRevisionID, incomingTwo.ProductRevisionID}) {
		t.Fatalf("re-export source refs=%v", seenProducts)
	}
	recoveredResult, err := restore.RecoverRestoreCheckpoint(ctx, "w1", result.CheckpointID, 4, "supplement-recovery")
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := repo.List(ctx, "w1")
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered) != 1 || recovered[0].ID != prior.ID || recovered[0].Revision != priorNext.Revision {
		t.Fatalf("recovered current schedule=%#v", recovered)
	}
	recoveredOne, err := repo.Get(ctx, prior.ID, "w1", 1)
	if err != nil {
		t.Fatal(err)
	}
	recoveredTwo, err := repo.Get(ctx, prior.ID, "w1", 2)
	if err != nil {
		t.Fatal(err)
	}
	assertScheduleRevisionEqual(t, recoveredOne, prior)
	assertScheduleRevisionEqual(t, recoveredTwo, priorNext)
	legacyBytes, err := Workspace(WorkspaceSnapshot{ID: "legacy"})
	if err != nil {
		t.Fatal(err)
	}
	var legacy Export
	if err := json.Unmarshal(legacyBytes, &legacy); err != nil {
		t.Fatal(err)
	}
	legacy.Manifest.RecordKinds = removeStrings(legacy.Manifest.RecordKinds, "supplement_schedule", "supplement_schedule_revision")
	legacy.Manifest.Omissions = append(legacy.Manifest.Omissions, "supplements")
	legacyBytes, err = json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err = ImportWorkspace(legacyBytes)
	if err != nil {
		t.Fatal(err)
	}
	legacyResult, err := restore.ApplyWorkspace(ctx, "w1", recoveredResult.WorkspaceRevision, "legacy-omission", legacy)
	if err != nil {
		t.Fatal(err)
	}
	preserved, err := repo.List(ctx, "w1")
	if err != nil || len(preserved) != 1 || preserved[0].Revision != 2 {
		t.Fatalf("legacy omission changed schedules: %#v err=%v", preserved, err)
	}
	emptyBytes, err := Workspace(WorkspaceSnapshot{ID: "empty"})
	if err != nil {
		t.Fatal(err)
	}
	empty, err := ImportWorkspace(emptyBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(empty.Manifest.RecordKinds, "supplement_schedule") {
		t.Fatal("supported empty export did not declare schedules")
	}
	if _, err := restore.ApplyWorkspace(ctx, "w1", legacyResult.WorkspaceRevision, "declared-empty", empty); err != nil {
		t.Fatal(err)
	}
	cleared, err := repo.List(ctx, "w1")
	if err != nil || len(cleared) != 0 {
		t.Fatalf("declared-empty schedule family was not cleared: %#v err=%v", cleared, err)
	}
}

func removeStrings(values []string, removals ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		remove := false
		for _, candidate := range removals {
			if value == candidate {
				remove = true
				break
			}
		}
		if !remove {
			out = append(out, value)
		}
	}
	return out
}

func TestWorkspaceProfileRoundTripAuthorizedRemapAndCheckpointRecovery(t *testing.T) {
	db := restoreDB(t)
	ctx := context.Background()
	repo := profile.NewSQLiteRepository(db, schedule.System())
	before, err := repo.Apply(ctx, profile.ApplyInput{WorkspaceID: "w1", Preset: "vegetarian", ExcludedGroups: []string{"nightshades"}, Allergies: []string{"peanut"}, Appliances: []string{"oven"}, CostWeight: .2, EffortWeight: .3, VarietyWeight: .5})
	if err != nil {
		t.Fatal(err)
	}
	before, err = repo.SaveDraft(ctx, "w1", `{"servings":4,"note":"keep"}`)
	if err != nil {
		t.Fatal(err)
	}
	source := profile.Profile{WorkspaceID: "source-workspace", Revision: 7, Preset: "vegan", PresetVersion: profile.CurrentPresetVersion, ActiveRules: profile.ExpandPreset("vegan"), ExcludedGroups: []string{"citrus"}, Allergies: []string{"sesame"}, Appliances: []string{"stovetop", "blender"}, CostWeight: .25, EffortWeight: .25, VarietyWeight: .5, DraftJSON: `{"servings":2,"custom":true}`}
	content, err := Workspace(WorkspaceSnapshot{ID: source.WorkspaceID, Profile: &source})
	if err != nil {
		t.Fatal(err)
	}
	staged, err := ImportWorkspace(content)
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewSQLiteRestorer(db, schedule.System()).ApplyWorkspace(ctx, "w1", 3, "profile-import", staged)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, "w1")
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkspaceID != "w1" || got.Revision != source.Revision || got.Preset != source.Preset || got.PresetVersion != source.PresetVersion || !reflect.DeepEqual(got.ActiveRules, source.ActiveRules) || !reflect.DeepEqual(got.ExcludedGroups, source.ExcludedGroups) || !reflect.DeepEqual(got.Allergies, source.Allergies) || !reflect.DeepEqual(got.Appliances, source.Appliances) || got.CostWeight != source.CostWeight || got.EffortWeight != source.EffortWeight || got.VarietyWeight != source.VarietyWeight || got.DraftJSON != source.DraftJSON {
		t.Fatalf("profile import mismatch: %#v", got)
	}
	reexport, err := Workspace(WorkspaceSnapshot{ID: "w1", Profile: &got})
	if err != nil {
		t.Fatal(err)
	}
	restaged, err := ImportWorkspace(reexport)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip profile.Profile
	for _, record := range restaged.Records {
		if record.Kind == "profile" {
			if err := json.Unmarshal(record.Data, &roundTrip); err != nil {
				t.Fatal(err)
			}
		}
	}
	if roundTrip.WorkspaceID != "w1" || roundTrip.DraftJSON != source.DraftJSON || !reflect.DeepEqual(roundTrip.ActiveRules, source.ActiveRules) {
		t.Fatalf("profile re-export mismatch: %#v", roundTrip)
	}
	if _, err := NewSQLiteRestorer(db, schedule.System()).RecoverRestoreCheckpoint(ctx, "w1", result.CheckpointID, result.WorkspaceRevision, "profile-recover"); err != nil {
		t.Fatal(err)
	}
	recovered, err := repo.Get(ctx, "w1")
	if err != nil {
		t.Fatal(err)
	}
	before.WorkspaceID = "w1"
	if !reflect.DeepEqual(recovered, before) {
		t.Fatalf("checkpoint profile differs\nwant=%#v\ngot=%#v", before, recovered)
	}
}

func TestWorkspaceProfileLegacyOmissionPreservesAndDeclaredEmptyClears(t *testing.T) {
	for _, legacy := range []bool{true, false} {
		t.Run(map[bool]string{true: "legacy", false: "declared-empty"}[legacy], func(t *testing.T) {
			db := restoreDB(t)
			ctx := context.Background()
			repo := profile.NewSQLiteRepository(db, schedule.System())
			before, err := repo.Apply(ctx, profile.ApplyInput{WorkspaceID: "w1", Preset: "pescatarian", Allergies: []string{"egg"}, CostWeight: .34, EffortWeight: .33, VarietyWeight: .33})
			if err != nil {
				t.Fatal(err)
			}
			content, err := Workspace(WorkspaceSnapshot{ID: "source"})
			if err != nil {
				t.Fatal(err)
			}
			var envelope Export
			if err := json.Unmarshal(content, &envelope); err != nil {
				t.Fatal(err)
			}
			if legacy {
				envelope.Manifest.RecordKinds = removeString(envelope.Manifest.RecordKinds, "profile")
				envelope.Manifest.Omissions = append(envelope.Manifest.Omissions, "profile")
			}
			envelope.Manifest.RecordCount = len(envelope.Records)
			staged, err := ImportWorkspace(mustJSON(t, envelope))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := NewSQLiteRestorer(db, schedule.System()).ApplyWorkspace(ctx, "w1", 3, "profile-empty", staged); err != nil {
				t.Fatal(err)
			}
			got, err := repo.Get(ctx, "w1")
			if legacy {
				if err != nil || !reflect.DeepEqual(got, before) {
					t.Fatalf("legacy profile changed: %#v err=%v", got, err)
				}
			} else if !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("declared empty profile was not cleared: %#v err=%v", got, err)
			}
		})
	}
}

func TestWorkspaceIntakeRoundTripRemapsRecipeAndRecoversCheckpoint(t *testing.T) {
	db := restoreDB(t)
	ctx := context.Background()
	clock := intakeTestClock{time.Date(2026, 10, 7, 9, 30, 0, 123456789, time.UTC)}
	if _, err := db.Exec(`INSERT INTO workspaces(id,name,owner_subject,revision,created_at,updated_at) VALUES('w2','Other','actor',0,'now','now')`); err != nil {
		t.Fatal(err)
	}
	service := recipe.NewService(recipe.NewSQLiteRepository(db, schedule.System()))
	foreign, err := service.Create(ctx, recipe.CreateInput{WorkspaceID: "w2", Name: "Existing", IdempotencyKey: "foreign"})
	if err != nil {
		t.Fatal(err)
	}
	intakes := nutrition.NewSQLiteIntakeRepository(db, clock)
	old := nutrition.IntakeEvent{ID: "old", Date: "2026-10-06", NutrientID: "protein", Amount: mustDecimal(t, "3.5"), Unit: "g", Reason: "before restore"}
	if err := intakes.Append(ctx, "w1", old); err != nil {
		t.Fatal(err)
	}
	oldCorrection := nutrition.IntakeEvent{ID: "old-correction", Date: "2026-10-06", NutrientID: "protein", Amount: mustDecimal(t, "3.25"), Unit: "g", Reason: "corrected before restore", CorrectionOf: "old"}
	if err := intakes.Append(ctx, "w1", oldCorrection); err != nil {
		t.Fatal(err)
	}
	incomingEvents := []nutrition.IntakeEvent{{ID: "source-event", Date: "2026-10-07", RecipeID: foreign.ID, RecipeRevision: 1, NutrientID: "protein", Amount: mustDecimal(t, "18.75"), Unit: "g", Reason: "source meal"}, {ID: "source-correction", Date: "2026-10-07", NutrientID: "protein", Amount: mustDecimal(t, "17.25"), Unit: "g", Reason: "weighed again", CorrectionOf: "source-event"}}
	for i := range incomingEvents {
		incomingEvents[i].ID = []string{"source-event", "source-correction"}[i]
		if err := intakes.Append(ctx, "source", incomingEvents[i]); err != nil {
			t.Fatal(err)
		}
	}
	sourceEvents, err := intakes.List(ctx, "source")
	if err != nil {
		t.Fatal(err)
	}
	content, err := Workspace(WorkspaceSnapshot{ID: "source", Recipes: []recipe.Recipe{{ID: foreign.ID, Revision: 1, Name: "Imported", Status: "draft"}}, IntakeEvents: sourceEvents})
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := ImportWorkspace(content)
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewSQLiteRestorer(db, schedule.System()).ApplyWorkspace(ctx, "w1", 3, "intake-apply", envelope)
	if err != nil {
		t.Fatal(err)
	}
	got, err := intakes.List(ctx, "w1")
	if err != nil || len(got) != 2 {
		t.Fatalf("events=%+v err=%v", got, err)
	}
	var mapped string
	if err := db.QueryRow(`SELECT id FROM recipes WHERE workspace_id='w1'`).Scan(&mapped); err != nil {
		t.Fatal(err)
	}
	byID := map[string]nutrition.IntakeEvent{}
	for _, event := range got {
		byID[event.ID] = event
	}
	if byID["source-event"].RecipeID != mapped || byID["source-event"].RecipeRevision != 1 || byID["source-event"].Amount.String() != "18.75" || byID["source-correction"].CorrectionOf != "source-event" {
		t.Fatalf("restored=%+v mapped=%s", got, mapped)
	}
	reexport, err := Workspace(WorkspaceSnapshot{ID: "w1", IntakeEvents: got})
	if err != nil {
		t.Fatal(err)
	}
	restaged, err := ImportWorkspace(reexport)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip := map[string]intakeEventRecord{}
	for _, record := range restaged.Records {
		if record.Kind == "intake_event" {
			var item intakeEventRecord
			if err := json.Unmarshal(record.Data, &item); err != nil {
				t.Fatal(err)
			}
			roundTrip[item.EventID] = item
		}
	}
	for _, event := range got {
		item, ok := roundTrip[event.ID]
		if !ok || item.WorkspaceID != "w1" || item.EventID != event.ID || item.Date != event.Date || item.RecipeID != event.RecipeID || item.RecipeRevision != event.RecipeRevision || item.NutrientID != event.NutrientID || item.Amount != event.Amount.String() || item.Unit != event.Unit || item.Reason != event.Reason || item.CorrectionOf != event.CorrectionOf || item.RecordedAt != event.RecordedAt().Format(time.RFC3339Nano) {
			t.Fatalf("round-trip mismatch for %s: %+v event=%+v", event.ID, item, event)
		}
	}
	if _, err := NewSQLiteRestorer(db, schedule.System()).RecoverRestoreCheckpoint(ctx, "w1", result.CheckpointID, result.WorkspaceRevision, "intake-recover"); err != nil {
		t.Fatal(err)
	}
	recovered, err := intakes.List(ctx, "w1")
	recoveredByID := map[string]nutrition.IntakeEvent{}
	for _, event := range recovered {
		recoveredByID[event.ID] = event
	}
	if err != nil || len(recovered) != 2 || recoveredByID["old"].Amount.String() != "3.5" || recoveredByID["old"].Reason != "before restore" || recoveredByID["old-correction"].CorrectionOf != "old" || recoveredByID["old-correction"].Amount.String() != "3.25" {
		t.Fatalf("checkpoint events=%+v err=%v", recovered, err)
	}
}

func TestWorkspaceIntakeLegacyOmissionPreservesAndDeclaredEmptyClears(t *testing.T) {
	for _, legacy := range []bool{true, false} {
		t.Run(map[bool]string{true: "legacy", false: "declared-empty"}[legacy], func(t *testing.T) {
			db := restoreDB(t)
			ctx := context.Background()
			repo := nutrition.NewSQLiteIntakeRepository(db, intakeTestClock{time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)})
			amount := mustDecimal(t, "4")
			if err := repo.Append(ctx, "w1", nutrition.IntakeEvent{ID: "keep", Date: "2026-10-07", NutrientID: "protein", Amount: amount, Unit: "g"}); err != nil {
				t.Fatal(err)
			}
			content, err := Workspace(WorkspaceSnapshot{ID: "source"})
			if err != nil {
				t.Fatal(err)
			}
			var envelope Export
			if err := json.Unmarshal(content, &envelope); err != nil {
				t.Fatal(err)
			}
			if legacy {
				envelope.Manifest.RecordKinds = removeString(envelope.Manifest.RecordKinds, "intake_event")
				envelope.Manifest.Omissions = append(envelope.Manifest.Omissions, "intake_events")
			}
			staged, err := ImportWorkspace(mustJSON(t, envelope))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := NewSQLiteRestorer(db, schedule.System()).ApplyWorkspace(ctx, "w1", 3, "intake-empty", staged); err != nil {
				t.Fatal(err)
			}
			events, err := repo.List(ctx, "w1")
			if err != nil {
				t.Fatal(err)
			}
			if legacy && (len(events) != 1 || events[0].ID != "keep") {
				t.Fatalf("legacy family changed: %+v", events)
			}
			if !legacy && len(events) != 0 {
				t.Fatalf("declared empty family remains: %+v", events)
			}
		})
	}
}

func removeString(values []string, target string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}

func TestWorkspaceNutritionTargetRevisionsRoundTripAndCheckpointRecovery(t *testing.T) {
	db := restoreDB(t)
	ctx := context.Background()
	seed := func(id string, rev int64, workspaceID, lower, upper, period, scope, enforcement, provenance, start string, end any, active int) {
		if _, err := db.Exec(`INSERT INTO nutrition_targets(id,revision,workspace_id,nutrient_id,lower_bound,upper_bound,period,scope,enforcement,provenance,effective_from,effective_to,active) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, id, rev, workspaceID, "protein", lower, upper, period, scope, enforcement, provenance, start, end, active); err != nil {
			t.Fatal(err)
		}
	}
	priorEnd := "2026-09-30T00:00:00Z"
	seed("prior", 1, "w1", "70.25", "100", "local_day", "planned_day", "preferred", "dietitian:case-7", "2026-09-01T00:00:00Z", priorEnd, 0)
	seed("expired", 1, "w2", "60", "90.5", "rolling_days", "recorded_so_far", "informational", "imported:legacy-4", "2026-01-01T00:00:00Z", priorEnd, 1)
	seed("inactive", 1, "w2", "55.5", "85", "local_day", "selected_meal", "preferred", "user_assertion:retired", "2026-02-01T00:00:00Z", "2026-08-01T00:00:00Z", 0)
	seed("current", 2, "w2", "80.125", "unknown", "calendar_week", "selected_week", "required", "user_assertion:change-2", "2026-10-01T00:00:00Z", nil, 1)
	read := func(workspaceID string) []nutrition.Target {
		t.Helper()
		targets, err := nutrition.NewSQLiteTargetRepository(db, schedule.System()).List(ctx, workspaceID)
		if err != nil {
			t.Fatal(err)
		}
		return targets
	}
	destinationBefore := read("w1")
	source := read("w2")
	content, err := Workspace(WorkspaceSnapshot{ID: "w2", NutritionTargets: source})
	if err != nil {
		t.Fatal(err)
	}
	staged, err := ImportWorkspace(content)
	if err != nil {
		t.Fatal(err)
	}
	if got := read("w1"); !reflect.DeepEqual(got, destinationBefore) {
		t.Fatalf("staging mutated targets: got=%#v want=%#v", got, destinationBefore)
	}
	service := NewSQLiteRestorer(db, schedule.System())
	result, err := service.ApplyWorkspace(ctx, "w1", 3, "target-roundtrip", staged)
	if err != nil {
		t.Fatal(err)
	}
	want := append([]nutrition.Target(nil), source...)
	for i := range want {
		want[i].WorkspaceID = "w1"
		want[i].ID = ""
	}
	got := read("w1")
	for i := range got {
		got[i].ID = ""
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("restored targets differ\nwant=%#v\ngot=%#v", want, got)
	}
	reexport, err := Workspace(WorkspaceSnapshot{ID: "w1", NutritionTargets: read("w1")})
	if err != nil {
		t.Fatal(err)
	}
	var envelope Export
	if err := json.Unmarshal(reexport, &envelope); err != nil {
		t.Fatal(err)
	}
	if contains(envelope.Manifest.Omissions, "nutrition_targets") || !contains(envelope.Manifest.RecordKinds, "nutrition_target") {
		t.Fatalf("manifest omits supported target domain: %#v", envelope.Manifest)
	}
	var reexported []nutrition.Target
	for _, record := range envelope.Records {
		if record.Kind != "nutrition_target" {
			continue
		}
		var target nutrition.Target
		if err := json.Unmarshal(record.Data, &target); err != nil {
			t.Fatal(err)
		}
		reexported = append(reexported, target)
	}
	if !reflect.DeepEqual(reexported, read("w1")) {
		t.Fatalf("re-export lost target semantics: %#v", reexported)
	}
	checkpoint, err := service.GetRestoreCheckpoint(ctx, "w1", result.CheckpointID)
	if err != nil || contains(checkpoint.Omissions, "nutrition_targets") {
		t.Fatalf("checkpoint=%#v err=%v", checkpoint, err)
	}
	if _, err := service.RecoverRestoreCheckpoint(ctx, "w1", result.CheckpointID, 3, "stale-target-recovery"); err == nil {
		t.Fatal("expected stale revision fence")
	}
	recovered, err := service.RecoverRestoreCheckpoint(ctx, "w1", result.CheckpointID, 4, "target-recovery")
	if err != nil || recovered.WorkspaceRevision != 5 {
		t.Fatalf("recovery=%#v err=%v", recovered, err)
	}
	if got := read("w1"); !reflect.DeepEqual(got, destinationBefore) {
		t.Fatalf("checkpoint did not restore prior target set\nwant=%#v\ngot=%#v", destinationBefore, got)
	}
}

func TestLegacyWorkspaceRestorePreservesDestinationNutritionTargets(t *testing.T) {
	db := restoreDB(t)
	ctx := context.Background()
	seedTarget(t, db, "kept", "w1")
	before := readTargets(t, db, "w1")
	content, err := Workspace(WorkspaceSnapshot{ID: "legacy"})
	if err != nil {
		t.Fatal(err)
	}
	var envelope Export
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.Manifest.RecordKinds = []string{"workspace", "recipe", "recipe_revision", "plan", "shopping_state"}
	envelope.Manifest.Omissions = append(envelope.Manifest.Omissions, "nutrition_targets")
	envelope.Manifest.RecordCount = len(envelope.Records)
	staged, err := ImportWorkspace(mustJSON(t, envelope))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewSQLiteRestorer(db, schedule.System()).ApplyWorkspace(ctx, "w1", 3, "legacy-restore", staged); err != nil {
		t.Fatal(err)
	}
	if got := readTargets(t, db, "w1"); !reflect.DeepEqual(got, before) {
		t.Fatalf("legacy omission changed targets: got=%#v want=%#v", got, before)
	}
}

func TestSupportedEmptyNutritionTargetsReplaceDestination(t *testing.T) {
	db := restoreDB(t)
	ctx := context.Background()
	seedTarget(t, db, "removed", "w1")
	content, err := Workspace(WorkspaceSnapshot{ID: "empty"})
	if err != nil {
		t.Fatal(err)
	}
	staged, err := ImportWorkspace(content)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(staged.Manifest.RecordKinds, "nutrition_target") {
		t.Fatal("empty supported export omitted nutrition_target declaration")
	}
	if _, err := NewSQLiteRestorer(db, schedule.System()).ApplyWorkspace(ctx, "w1", 3, "empty-restore", staged); err != nil {
		t.Fatal(err)
	}
	if got := readTargets(t, db, "w1"); len(got) != 0 {
		t.Fatalf("supported empty export retained targets: %#v", got)
	}
}

func TestInvalidNutritionTargetStagingRejectsWithoutMutation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*nutrition.Target)
	}{
		{"invalid effective window", func(target *nutrition.Target) { end := target.EffectiveFrom; target.EffectiveTo = &end }},
		{"workspace mismatch", func(target *nutrition.Target) { target.WorkspaceID = "another-workspace" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := restoreDB(t)
			seedTarget(t, db, "kept", "w1")
			before := readTargets(t, db, "w1")
			target := nutrition.Target{ID: "incoming", Revision: 1, WorkspaceID: "source", NutrientID: "protein", Lower: mustDecimal(t, "50"), Upper: mustDecimal(t, "100"), Period: "local_day", Scope: "planned_day", Enforcement: "preferred", Provenance: "user_assertion:test", EffectiveFrom: mustTime(t, "2026-01-01T00:00:00Z"), Active: true}
			tc.mutate(&target)
			content, err := Workspace(WorkspaceSnapshot{ID: "source", NutritionTargets: []nutrition.Target{target}})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ImportWorkspace(content); err == nil {
				t.Fatal("invalid target staging unexpectedly succeeded")
			}
			if got := readTargets(t, db, "w1"); !reflect.DeepEqual(got, before) {
				t.Fatalf("rejected staging mutated destination: got=%#v want=%#v", got, before)
			}
		})
	}
}

func seedTarget(t *testing.T, db *sql.DB, id, workspaceID string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO nutrition_targets(id,revision,workspace_id,nutrient_id,lower_bound,upper_bound,period,scope,enforcement,provenance,effective_from,effective_to,active) VALUES(?,1,?,'protein','60','100','local_day','planned_day','preferred','user_assertion:test','2026-01-01T00:00:00Z',NULL,1)`, id, workspaceID)
	if err != nil {
		t.Fatal(err)
	}
}

func readTargets(t *testing.T, db *sql.DB, workspaceID string) []nutrition.Target {
	t.Helper()
	got, err := nutrition.NewSQLiteTargetRepository(db, schedule.System()).List(context.Background(), workspaceID)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mustDecimal(t *testing.T, value string) decimalx.Decimal {
	t.Helper()
	got, err := decimalx.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	got, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestApplyWorkspaceReplacesSupportedDomainsWithCheckpoint(t *testing.T) {
	db := restoreDB(t)
	ctx := context.Background()
	content, err := Workspace(WorkspaceSnapshot{
		ID: "w-export", Name: "Imported", Revision: 9,
		Recipes:      []recipe.Recipe{{ID: "r1", Revision: 2, Name: "Soup", Status: "draft", CanonicalYield: "4", ServingUnit: "servings", Ingredients: []recipe.Ingredient{{ID: "i1", Name: "Rice", Amount: "200", Unit: "g"}}}},
		PlanRevision: 4, PlanJSON: `{"occurrences":[{"recipeId":"r1"}]}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	var envelope Export
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatal(err)
	}
	result, err := NewSQLiteRestorer(db, schedule.System()).ApplyWorkspace(ctx, "w1", 3, "restore-1", envelope)
	if err != nil || result.WorkspaceRevision != 4 || result.RecipesApplied != 1 || result.CheckpointID == "" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM recipes WHERE workspace_id='w1'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("recipe count=%d err=%v", count, err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, plan, planErr := planning.ReadPlanTx(ctx, tx, "w1")
	_ = tx.Rollback()
	if planErr != nil || plan == "" {
		t.Fatalf("plan=%q err=%v", plan, planErr)
	}
	var yield, unit, ingredients string
	if err := db.QueryRow(`SELECT canonical_yield,serving_unit,ingredients_json FROM recipe_revisions WHERE recipe_id='r1'`).Scan(&yield, &unit, &ingredients); err != nil || yield != "4" || unit != "servings" || !strings.Contains(ingredients, "Rice") {
		t.Fatalf("restored recipe fields yield=%q unit=%q ingredients=%q err=%v", yield, unit, ingredients, err)
	}
	var checkpoints int
	if err := db.QueryRow(`SELECT COUNT(*) FROM portability_restore_checkpoints WHERE id=?`, result.CheckpointID).Scan(&checkpoints); err != nil || checkpoints != 1 {
		t.Fatalf("checkpoint count=%d err=%v", checkpoints, err)
	}
	replay, err := NewSQLiteRestorer(db, schedule.System()).ApplyWorkspace(ctx, "w1", 3, "restore-1", envelope)
	if err != nil || replay.CheckpointID != result.CheckpointID {
		t.Fatalf("idempotent replay=%#v err=%v", replay, err)
	}
}

func TestWorkspaceShoppingStateRoundTripAndCheckpointRecovery(t *testing.T) {
	db := restoreDB(t)
	ctx := context.Background()
	// The destination has shopping state that must be recoverable after apply.
	for _, row := range []struct {
		key     string
		checked int
	}{{"dest:checked", 1}, {"dest:unchecked", 0}} {
		if _, err := db.Exec(`INSERT INTO shopping_checks(workspace_id,line_key,checked,updated_at) VALUES('w1',?,?,?)`, row.key, row.checked, "dest-time"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO shopping_have_this(workspace_id,line_key,asserted,updated_at) VALUES('w1','dest:have',1,'dest-time')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO shopping_purchase_reviews(workspace_id,review_id,payload_hash,created_at) VALUES('w1','dest-trip','dest-hash','dest-review-time')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO shopping_purchase_review_lines(workspace_id,review_id,line_key,item_id,amount,unit,price,omitted) VALUES('w1','dest-trip','dest:line','rice','2','kg','4.50',0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO inventory_events(workspace_id,event_id,kind,item_id,amount,unit,created_at) VALUES('w1','existing-stock','purchase','rice','7','kg','stock-time')`); err != nil {
		t.Fatal(err)
	}
	restorer := NewSQLiteRestorer(db, schedule.System())
	destBefore := readShoppingState(t, db, "w1")
	var inventoryBefore int
	if err := db.QueryRow(`SELECT count(*) FROM inventory_events WHERE workspace_id='w1'`).Scan(&inventoryBefore); err != nil {
		t.Fatal(err)
	}
	source := shopping.PersistedState{Checks: map[string]bool{"ingredient:rice": true, "ingredient:beans": false}, HaveThis: map[string]bool{"ingredient:rice": true, "ingredient:beans": false}, Reviews: []shopping.PersistedReview{
		{ID: "trip-1", PayloadHash: "hash-one", CreatedAt: "2026-10-01T10:00:00Z", Lines: []shopping.PersistedLine{{Key: "ingredient:rice", ItemID: "rice", Amount: "2.5", Unit: "kg", Price: "5.00"}, {Key: "ingredient:unknown", ItemID: "beans", Unit: "can", Price: "", Omitted: true}}},
		{ID: "trip-2", PayloadHash: "hash-two", CreatedAt: "2026-10-02T11:00:00Z", Lines: []shopping.PersistedLine{{Key: "ingredient:rice", ItemID: "rice", Amount: "3", Unit: "kg", Price: "6.00"}}},
	}}
	content, err := Workspace(WorkspaceSnapshot{ID: "source", Shopping: source})
	if err != nil {
		t.Fatal(err)
	}
	staged, err := ImportWorkspace(content)
	if err != nil {
		t.Fatal(err)
	}
	// Staging is pure: destination shopping and inventory are still unchanged.
	unchanged := readShoppingState(t, db, "w1")
	if !reflect.DeepEqual(unchanged, destBefore) {
		t.Fatalf("staging mutated destination: got=%#v want=%#v", unchanged, destBefore)
	}
	if _, err := db.Exec(`INSERT INTO workspaces(id,name,owner_subject,revision,created_at,updated_at) VALUES('w2','Source','actor',0,'now','now')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO shopping_checks(workspace_id,line_key,checked,updated_at) VALUES('w2','ingredient:rice',1,'source-time'),('w2','ingredient:beans',0,'source-time')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO shopping_have_this(workspace_id,line_key,asserted,updated_at) VALUES('w2','ingredient:rice',1,'source-time'),('w2','ingredient:beans',0,'source-time')`); err != nil {
		t.Fatal(err)
	}
	for _, review := range source.Reviews {
		if _, err := db.Exec(`INSERT INTO shopping_purchase_reviews(workspace_id,review_id,payload_hash,created_at) VALUES('w2',?,?,?)`, review.ID, review.PayloadHash, review.CreatedAt); err != nil {
			t.Fatal(err)
		}
		for _, line := range review.Lines {
			omitted := 0
			if line.Omitted {
				omitted = 1
			}
			if _, err := db.Exec(`INSERT INTO shopping_purchase_review_lines(workspace_id,review_id,line_key,item_id,amount,unit,price,omitted) VALUES('w2',?,?,?,?,?,?,?)`, review.ID, line.Key, line.ItemID, line.Amount, line.Unit, line.Price, omitted); err != nil {
				t.Fatal(err)
			}
		}
	}
	// Export from the source repository and verify all review headers and lines are read.
	exportedState, err := shopping.NewSQLiteRepository(db, schedule.System()).(shopping.PortableRepository).PersistedState(ctx, "w2")
	if err != nil || !reflect.DeepEqual(exportedState, source) {
		t.Fatalf("source state=%#v err=%v", exportedState, err)
	}
	content, err = Workspace(WorkspaceSnapshot{ID: "w2", Shopping: exportedState})
	if err != nil {
		t.Fatal(err)
	}
	staged, err = ImportWorkspace(content)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workspaces(id,name,owner_subject,revision,created_at,updated_at) VALUES('w3','Empty','actor',0,'now','now')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO inventory_events(workspace_id,event_id,kind,item_id,amount,unit,created_at) VALUES('w3','existing-empty-stock','purchase','rice','9','kg','stock-time')`); err != nil {
		t.Fatal(err)
	}
	var emptyInventoryBefore int
	if err := db.QueryRow(`SELECT count(*) FROM inventory_events WHERE workspace_id='w3'`).Scan(&emptyInventoryBefore); err != nil {
		t.Fatal(err)
	}
	applied, err := restorer.ApplyWorkspace(ctx, "w1", 3, "shopping-import", staged)
	if err != nil {
		t.Fatal(err)
	}
	got := readShoppingState(t, db, "w1")
	if !reflect.DeepEqual(got, source) {
		t.Fatalf("applied state differs\nwant=%#v\ngot=%#v", source, got)
	}
	emptyApplied, err := restorer.ApplyWorkspace(ctx, "w3", 0, "empty-shopping-import", staged)
	if err != nil {
		t.Fatal(err)
	}
	emptyGot := readShoppingState(t, db, "w3")
	if !reflect.DeepEqual(emptyGot, source) {
		t.Fatalf("empty workspace apply differs\nwant=%#v\ngot=%#v", source, emptyGot)
	}
	// A native export/import from the empty destination remains semantically identical.
	reexport, err := Workspace(WorkspaceSnapshot{ID: "w3", Shopping: emptyGot})
	if err != nil {
		t.Fatal(err)
	}
	restaged, err := ImportWorkspace(reexport)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip shopping.PersistedState
	for _, record := range restaged.Records {
		if record.Kind == "shopping_state" {
			if err := json.Unmarshal(record.Data, &roundTrip); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !reflect.DeepEqual(roundTrip, source) {
		t.Fatalf("re-export differs\nwant=%#v\ngot=%#v", source, roundTrip)
	}
	checkpoint, err := restorer.RecoverRestoreCheckpoint(ctx, "w1", applied.CheckpointID, 4, "shopping-recovery")
	if err != nil || checkpoint.WorkspaceRevision != 5 {
		t.Fatalf("recovery=%#v err=%v", checkpoint, err)
	}
	recovered := readShoppingState(t, db, "w1")
	if !reflect.DeepEqual(recovered, destBefore) {
		t.Fatalf("checkpoint shopping recovery differs\nwant=%#v\ngot=%#v", destBefore, recovered)
	}
	var inventoryAfter int
	if err := db.QueryRow(`SELECT count(*) FROM inventory_events WHERE workspace_id='w1'`).Scan(&inventoryAfter); err != nil || inventoryAfter != inventoryBefore {
		t.Fatalf("inventory changed: before=%d after=%d err=%v", inventoryBefore, inventoryAfter, err)
	}
	emptyRecovered, err := restorer.RecoverRestoreCheckpoint(ctx, "w3", emptyApplied.CheckpointID, 1, "empty-shopping-recovery")
	if err != nil || emptyRecovered.WorkspaceRevision != 2 {
		t.Fatalf("empty workspace recovery=%#v err=%v", emptyRecovered, err)
	}
	if emptyState := readShoppingState(t, db, "w3"); !reflect.DeepEqual(emptyState, shopping.PersistedState{Checks: map[string]bool{}, HaveThis: map[string]bool{}, Reviews: []shopping.PersistedReview{}}) {
		t.Fatalf("empty checkpoint did not recover: %#v", emptyState)
	}
	var emptyInventoryAfter int
	if err := db.QueryRow(`SELECT count(*) FROM inventory_events WHERE workspace_id='w3'`).Scan(&emptyInventoryAfter); err != nil || emptyInventoryAfter != emptyInventoryBefore {
		t.Fatalf("empty workspace inventory changed: before=%d after=%d err=%v", emptyInventoryBefore, emptyInventoryAfter, err)
	}
}

func readShoppingState(t *testing.T, db *sql.DB, workspaceID string) shopping.PersistedState {
	t.Helper()
	state, err := shopping.NewSQLiteRepository(db, schedule.System()).(shopping.PortableRepository).PersistedState(context.Background(), workspaceID)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func TestImmutableRecipeHistorySurvivesWorkspaceAndRecipeRoundTrips(t *testing.T) {
	db := restoreDB(t)
	for _, id := range []string{"w2", "w3"} {
		if _, err := db.Exec(`INSERT INTO workspaces(id,name,owner_subject,revision,created_at,updated_at) VALUES(?,?,?,0,'now','now')`, id, id, "actor"); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	prior := recipe.Recipe{ID: "r-history", Revision: 1, Name: "First", OriginalText: "verbatim ½ cup source", SourceType: "web", SourceURL: "https://example.test/source", Status: "draft", CanonicalYield: "2", ServingUnit: "bowls", Ingredients: []recipe.Ingredient{{ID: "rice", Name: "Rice", Amount: "½", Unit: "cup", Preparation: "rinsed"}}, Methods: []recipe.Method{{ID: "cook", Name: "Cook", Steps: []recipe.MethodStep{{ID: "simmer", Instruction: "Simmer for 12 minutes."}}}}}
	current := prior
	current.Revision = 2
	current.Name = "Second"
	current.OriginalText = "revised verbatim text"
	current.Ingredients = []recipe.Ingredient{{ID: "rice", Name: "Rice", Amount: "¾", Unit: "cup", Preparation: "rinsed"}}
	workspaceJSON, err := Workspace(WorkspaceSnapshot{ID: "source-workspace", Recipes: []recipe.Recipe{current}, RecipeHistory: []recipe.Recipe{prior}})
	if err != nil {
		t.Fatal(err)
	}
	workspaceEnvelope, err := ImportWorkspace(workspaceJSON)
	if err != nil {
		t.Fatal(err)
	}
	service := NewSQLiteRestorer(db, schedule.System())
	if _, err := service.ApplyWorkspace(ctx, "w2", 0, "workspace-import", workspaceEnvelope); err != nil {
		t.Fatal(err)
	}
	assertHistory := func(workspaceID, recipeID string) {
		t.Helper()
		var count int
		var currentRevision int64
		if err := db.QueryRow(`SELECT COUNT(*) FROM recipe_revisions WHERE workspace_id=? AND recipe_id=?`, workspaceID, recipeID).Scan(&count); err != nil || count != 2 {
			t.Fatalf("revision count=%d err=%v", count, err)
		}
		if err := db.QueryRow(`SELECT current_revision FROM recipes WHERE workspace_id=? AND id=?`, workspaceID, recipeID).Scan(&currentRevision); err != nil || currentRevision != 2 {
			t.Fatalf("current revision=%d err=%v", currentRevision, err)
		}
		var text, name, amount, method string
		if err := db.QueryRow(`SELECT original_text,name,ingredients_json,methods_json FROM recipe_revisions WHERE workspace_id=? AND recipe_id=? AND revision=1`, workspaceID, recipeID).Scan(&text, &name, &amount, &method); err != nil {
			t.Fatal(err)
		}
		if text != prior.OriginalText || name != prior.Name || !strings.Contains(amount, `"Amount":"½"`) || !strings.Contains(method, "Simmer for 12 minutes.") {
			t.Fatalf("revision 1 changed: text=%q name=%q ingredients=%s method=%s", text, name, amount, method)
		}
	}
	assertHistory("w2", "r-history")
	canonicalVersions := func(envelope Export) map[int64]Recipe {
		t.Helper()
		out := map[int64]Recipe{}
		for _, record := range envelope.Records {
			if record.Kind != "recipe_revision" {
				continue
			}
			var item Recipe
			if err := json.Unmarshal(record.Data, &item); err != nil {
				t.Fatal(err)
			}
			item.ID = "canonical-recipe"
			out[item.Revision] = item
		}
		return out
	}
	workspaceExpected, _, err := portableRecipeHistory(workspaceEnvelope)
	if err != nil {
		t.Fatal(err)
	}
	want := map[int64]Recipe{}
	for _, item := range workspaceExpected["r-history"] {
		item.ID = "canonical-recipe"
		want[item.Revision] = item
	}
	readExport := func(workspaceID, recipeID string) Export {
		t.Helper()
		repository := recipe.NewSQLiteRepository(db, schedule.System())
		current, err := repository.Get(ctx, recipeID, workspaceID)
		if err != nil {
			t.Fatal(err)
		}
		var history []recipe.Recipe
		for revision := int64(1); revision < current.Revision; revision++ {
			item, err := repository.GetRevision(ctx, recipeID, workspaceID, revision)
			if err != nil {
				t.Fatal(err)
			}
			history = append(history, item)
		}
		content, err := RecipesWithHistory([]recipe.Recipe{current}, history)
		if err != nil {
			t.Fatal(err)
		}
		var result Export
		if err := json.Unmarshal(content, &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	readWorkspaceExport := func(workspaceID, recipeID string) Export {
		t.Helper()
		current, err := recipe.NewSQLiteRepository(db, schedule.System()).Get(ctx, recipeID, workspaceID)
		if err != nil {
			t.Fatal(err)
		}
		var history []recipe.Recipe
		for revision := int64(1); revision < current.Revision; revision++ {
			item, err := recipe.NewSQLiteRepository(db, schedule.System()).GetRevision(ctx, recipeID, workspaceID, revision)
			if err != nil {
				t.Fatal(err)
			}
			history = append(history, item)
		}
		content, err := Workspace(WorkspaceSnapshot{ID: workspaceID, Recipes: []recipe.Recipe{current}, RecipeHistory: history})
		if err != nil {
			t.Fatal(err)
		}
		var result Export
		if err := json.Unmarshal(content, &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	if got := canonicalVersions(readWorkspaceExport("w2", "r-history")); !reflect.DeepEqual(got, want) {
		t.Fatalf("workspace round-trip differs\nwant=%#v\ngot=%#v", want, got)
	}
	imported, err := service.ApplyRecipes(ctx, "w3", 0, "recipes-import", "copy_incoming", mustRecipes(t, current, prior))
	if err != nil {
		t.Fatal(err)
	}
	importedID := "r-history"
	if len(imported.RemappedIDs) > 0 {
		_, importedID, _ = strings.Cut(imported.RemappedIDs[0], "=")
	}
	assertHistory("w3", importedID)
	if got := canonicalVersions(readExport("w3", importedID)); !reflect.DeepEqual(got, want) {
		t.Fatalf("recipe collection round-trip differs\nwant=%#v\ngot=%#v", want, got)
	}
}

func mustRecipes(t *testing.T, current, prior recipe.Recipe) Export {
	t.Helper()
	content, err := RecipesWithHistory([]recipe.Recipe{current}, []recipe.Recipe{prior})
	if err != nil {
		t.Fatal(err)
	}
	staged, err := ImportRecipes(content)
	if err != nil {
		t.Fatal(err)
	}
	return staged
}

func TestCheckpointReviewAndExplicitRecoveryAreRevisionFencedAndIdempotent(t *testing.T) {
	db := restoreDB(t)
	ctx := context.Background()
	service := NewSQLiteRestorer(db, schedule.System())
	seed, _ := Workspace(WorkspaceSnapshot{ID: "w1", Name: "Daily", Revision: 3, Recipes: []recipe.Recipe{{ID: "before", Revision: 1, Name: "Before", Status: "draft"}}})
	var original Export
	if err := json.Unmarshal(seed, &original); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ApplyWorkspace(ctx, "w1", 3, "seed", original); err != nil {
		t.Fatal(err)
	}
	emptyBytes, _ := Workspace(WorkspaceSnapshot{ID: "w1", Name: "Daily", Revision: 4})
	var empty Export
	if err := json.Unmarshal(emptyBytes, &empty); err != nil {
		t.Fatal(err)
	}
	applied, err := service.ApplyWorkspace(ctx, "w1", 4, "replace", empty)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err := service.GetRestoreCheckpoint(ctx, "w1", applied.CheckpointID)
	if err != nil || checkpoint.RecipeCount != 1 || checkpoint.RestoreRevision != 4 || checkpoint.PlanIncluded || len(checkpoint.Omissions) == 0 {
		t.Fatalf("checkpoint=%#v err=%v", checkpoint, err)
	}
	if _, err := service.RecoverRestoreCheckpoint(ctx, "w1", applied.CheckpointID, 3, "stale"); err == nil {
		t.Fatal("expected stale revision rejection")
	}
	result, err := service.RecoverRestoreCheckpoint(ctx, "w1", applied.CheckpointID, 5, "recover")
	if err != nil || result.RecipesApplied != 1 || result.WorkspaceRevision != 6 {
		t.Fatalf("recovery=%#v err=%v", result, err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM recipes WHERE workspace_id='w1' AND id='before'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("recovered recipe count=%d err=%v", count, err)
	}
	replay, err := service.RecoverRestoreCheckpoint(ctx, "w1", applied.CheckpointID, 999, "recover")
	if err != nil || replay.WorkspaceRevision != result.WorkspaceRevision || replay.CheckpointID != result.CheckpointID {
		t.Fatalf("replay=%#v err=%v", replay, err)
	}
	second, err := service.ApplyWorkspace(ctx, "w1", result.WorkspaceRevision, "replace-again", empty)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RecoverRestoreCheckpoint(ctx, "w1", second.CheckpointID, result.WorkspaceRevision+1, "recover"); err == nil || !strings.Contains(err.Error(), "idempotency key was reused") {
		t.Fatalf("expected changed checkpoint payload conflict, got %v", err)
	}
	if _, err := service.GetRestoreCheckpoint(ctx, "other-workspace", applied.CheckpointID); err == nil {
		t.Fatal("foreign workspace disclosed checkpoint")
	}
}

func TestApplyWorkspaceRollsBackWhenImportedPlanIsInvalid(t *testing.T) {
	db := restoreDB(t)
	ctx := context.Background()
	content, err := Workspace(WorkspaceSnapshot{ID: "w-export", Recipes: []recipe.Recipe{{ID: "r1", Revision: 1, Name: "Soup", Status: "draft"}}})
	if err != nil {
		t.Fatal(err)
	}
	var envelope Export
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatal(err)
	}
	badPlan, _ := json.Marshal(map[string]any{"revision": 1, "planJson": "not-json"})
	envelope.Records = append(envelope.Records, Record{Kind: "plan", ID: "w-export", Revision: 1, Data: badPlan})
	envelope.Manifest.RecordCount = len(envelope.Records)
	if _, err := NewSQLiteRestorer(db, schedule.System()).ApplyWorkspace(ctx, "w1", 3, "restore-bad", envelope); err == nil {
		t.Fatal("expected invalid plan rejection")
	}
	var revision int64
	if err := db.QueryRow(`SELECT revision FROM workspaces WHERE id='w1'`).Scan(&revision); err != nil || revision != 3 {
		t.Fatalf("workspace revision=%d err=%v", revision, err)
	}
	var checkpoints int
	if err := db.QueryRow(`SELECT COUNT(*) FROM portability_restore_checkpoints`).Scan(&checkpoints); err != nil || checkpoints != 0 {
		t.Fatalf("rollback left checkpoint count=%d err=%v", checkpoints, err)
	}
}

func TestRecipeImportStagesConflictsAndAppliesIdempotently(t *testing.T) {
	db := restoreDB(t)
	ctx := context.Background()
	content, err := Recipes([]recipe.Recipe{{ID: "r1", Revision: 1, Name: "Soup", Status: "draft"}})
	if err != nil {
		t.Fatal(err)
	}
	var envelope Export
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatal(err)
	}
	r := NewSQLiteRestorer(db, schedule.System())
	preview, err := r.PreviewRecipes(ctx, "w1", envelope)
	if err != nil || preview.RecipeCount != 1 || preview.ConflictCount != 0 {
		t.Fatalf("preview=%#v err=%v", preview, err)
	}
	first, err := r.ApplyRecipes(ctx, "w1", 3, "recipes-1", "copy_incoming", envelope)
	if err != nil || first.WorkspaceRevision != 4 || first.RecipesApplied != 1 {
		t.Fatalf("first=%#v err=%v", first, err)
	}
	replay, err := r.ApplyRecipes(ctx, "w1", 3, "recipes-1", "copy_incoming", envelope)
	if err != nil || replay.WorkspaceRevision != first.WorkspaceRevision || replay.RecipesApplied != first.RecipesApplied {
		t.Fatalf("replay=%#v err=%v", replay, err)
	}

	changed, _ := Recipes([]recipe.Recipe{{ID: "r1", Revision: 1, Name: "Soup revised", Status: "draft"}})
	var changedEnvelope Export
	if err := json.Unmarshal(changed, &changedEnvelope); err != nil {
		t.Fatal(err)
	}
	preview, err = r.PreviewRecipes(ctx, "w1", changedEnvelope)
	if err != nil || preview.ConflictCount != 1 {
		t.Fatalf("changed preview=%#v err=%v", preview, err)
	}
	second, err := r.ApplyRecipes(ctx, "w1", 4, "recipes-2", "copy_incoming", changedEnvelope)
	if err != nil || second.RecipesApplied != 1 || len(second.RemappedIDs) != 1 {
		t.Fatalf("second=%#v err=%v", second, err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM recipes WHERE workspace_id='w1'`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("recipe count=%d err=%v", count, err)
	}
}
