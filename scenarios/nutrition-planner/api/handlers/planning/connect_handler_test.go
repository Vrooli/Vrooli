package planning

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/storage"
	nutritionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/nutrition"
	nutritionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/nutrition/nutrition_v1connect"
	planningv1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/planning"
	planningconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/planning/planning_v1connect"
	supplementv1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/supplement"
	supplementconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/supplement/supplement_v1connect"
	_ "modernc.org/sqlite"
	nutritionhandler "nutrition-planner/handlers/nutrition"
	supplementhandler "nutrition-planner/handlers/supplement"
	"nutrition-planner/internal/cost"
	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/feedback"
	"nutrition-planner/internal/inventory"
	"nutrition-planner/internal/money"
	internalnutrition "nutrition-planner/internal/nutrition"
	internal "nutrition-planner/internal/planning"
	"nutrition-planner/internal/recipe"
	"nutrition-planner/internal/shopping"
	internalsupplement "nutrition-planner/internal/supplement"
	"nutrition-planner/internal/workspace"
)

func TestGetPlanReadsPersistedDateRangeForWorkspaceOwnerOnly(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, schema := range []string{workspace.Schema(), recipe.Schema(), internal.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	clock := schedule.System()
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	ownerWorkspace, err := workspaces.Create(context.Background(), workspace.CreateInput{Name: "Personal", OwnerSubject: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	repository := internal.NewSQLiteRepository(db, clock)
	draft := internal.Draft{RunID: "read-model-test", Occurrences: []internal.Occurrence{
		{Date: "2026-10-03", SlotName: "dinner", RecipeID: "r1", RecipeRevision: 3, RecipeName: "Soup"},
		{Date: "2026-10-04", SlotName: "dinner", RecipeID: "r2", RecipeRevision: 1, RecipeName: "Rice"},
	}}
	raw, _ := json.Marshal(draft)
	if _, err := repository.Apply(context.Background(), ownerWorkspace.ID, 0, string(raw)); err != nil {
		t.Fatal(err)
	}
	handler := NewConnectHandler(workspaces, recipe.NewService(recipe.NewSQLiteRepository(db, clock)), nil, repository, nil, nil, nil)
	request := connect.NewRequest(&planningv1.GetPlanRequest{WorkspaceId: ownerWorkspace.ID, FromDate: "2026-10-03", ToDate: "2026-10-03"})
	if _, err := handler.GetPlan(context.Background(), request); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unauthenticated read error=%v", err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true})
	response, err := handler.GetPlan(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	var got internal.Draft
	if err := json.Unmarshal([]byte(response.Msg.DraftJson), &got); err != nil {
		t.Fatal(err)
	}
	if !response.Msg.HasPlan || response.Msg.CurrentRevision != 1 || len(got.Occurrences) != 1 || got.Occurrences[0].Date != "2026-10-03" || got.Occurrences[0].RecipeRevision != 3 {
		t.Fatalf("response=%+v draft=%+v", response.Msg, got)
	}
	other := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "other", Verified: true})
	if _, err := handler.GetPlan(other, request); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("cross-actor read error=%v", err)
	}
}

func TestPreviewSwapChangesOnlyRequestedPersistedSlotAndRejectsStaleRevision(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, schema := range []string{workspace.Schema(), recipe.Schema(), internal.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	clock := schedule.System()
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	ownerWorkspace, err := workspaces.Create(ctx, workspace.CreateInput{Name: "Personal", OwnerSubject: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	recipes := recipe.NewService(recipe.NewSQLiteRepository(db, clock))
	original, err := recipes.Create(ctx, recipe.CreateInput{WorkspaceID: ownerWorkspace.ID, Name: "Soup"})
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := recipes.Create(ctx, recipe.CreateInput{WorkspaceID: ownerWorkspace.ID, Name: "Rice"})
	if err != nil {
		t.Fatal(err)
	}
	repository := internal.NewSQLiteRepository(db, clock)
	draft := internal.Draft{RunID: "slot-swap", Occurrences: []internal.Occurrence{
		{Date: "2026-10-03", SlotName: "breakfast", RecipeID: original.ID, RecipeRevision: original.Revision, RecipeName: original.Name},
		{Date: "2026-10-03", SlotName: "dinner", RecipeID: original.ID, RecipeRevision: original.Revision, RecipeName: original.Name, Locked: true},
	}}
	raw, _ := json.Marshal(draft)
	if revision, err := repository.Apply(ctx, ownerWorkspace.ID, 0, string(raw)); err != nil || revision != 1 {
		t.Fatalf("seed revision=%d err=%v", revision, err)
	}
	handler := NewConnectHandler(workspaces, recipes, nil, repository, nil, nil, nil)
	authorized := identity.WithPrincipal(ctx, identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true})
	response, err := handler.PreviewSwap(authorized, connect.NewRequest(&planningv1.PreviewSwapRequest{WorkspaceId: ownerWorkspace.ID, ExpectedRevision: 1, Date: "2026-10-03", SlotName: "breakfast", ReplacementRecipeId: replacement.ID}))
	if err != nil {
		t.Fatal(err)
	}
	var preview internal.SwapPreview
	if err := json.Unmarshal([]byte(response.Msg.PreviewJson), &preview); err != nil {
		t.Fatal(err)
	}
	if len(preview.Changes) != 1 || preview.Changes[0].SlotName != "breakfast" || preview.Draft.Occurrences[0].RecipeID != replacement.ID || preview.Draft.Occurrences[1].RecipeID != original.ID || !preview.Draft.Occurrences[1].Locked {
		t.Fatalf("preview changed a sibling slot or lost its lock: %+v", preview)
	}
	_, err = handler.PreviewSwap(authorized, connect.NewRequest(&planningv1.PreviewSwapRequest{WorkspaceId: ownerWorkspace.ID, ExpectedRevision: 0, Date: "2026-10-03", SlotName: "breakfast", ReplacementRecipeId: replacement.ID}))
	if connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale preview error=%v, want aborted", err)
	}
}

func TestPreviewSwapRejectsChangedCapturedRecipeInputs(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, schema := range []string{workspace.Schema(), recipe.Schema(), internal.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	clock := schedule.System()
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	ownerWorkspace, err := workspaces.Create(ctx, workspace.CreateInput{Name: "Personal", OwnerSubject: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	recipes := recipe.NewService(recipe.NewSQLiteRepository(db, clock))
	original, err := recipes.Create(ctx, recipe.CreateInput{WorkspaceID: ownerWorkspace.ID, Name: "Original soup"})
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := recipes.Create(ctx, recipe.CreateInput{WorkspaceID: ownerWorkspace.ID, Name: "Rice"})
	if err != nil {
		t.Fatal(err)
	}
	repository := internal.NewSQLiteRepository(db, clock)
	draft := internal.Draft{RunID: "captured-recipe-input", InputReferences: []string{fmt.Sprintf("recipe:%s:%d", original.ID, original.Revision)}, Occurrences: []internal.Occurrence{{Date: "2026-10-03", SlotName: "dinner", RecipeID: original.ID, RecipeRevision: original.Revision, RecipeName: original.Name}}}
	raw, _ := json.Marshal(draft)
	if _, err := repository.Apply(ctx, ownerWorkspace.ID, 0, string(raw)); err != nil {
		t.Fatal(err)
	}
	if _, err := recipes.Update(ctx, recipe.UpdateInput{WorkspaceID: ownerWorkspace.ID, ID: original.ID, ExpectedRevision: original.Revision, Name: "Original soup revised"}); err != nil {
		t.Fatal(err)
	}
	handler := NewConnectHandler(workspaces, recipes, nil, repository, nil, nil, nil)
	authorized := identity.WithPrincipal(ctx, identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true})
	_, err = handler.PreviewSwap(authorized, connect.NewRequest(&planningv1.PreviewSwapRequest{WorkspaceId: ownerWorkspace.ID, ExpectedRevision: 1, Date: "2026-10-03", SlotName: "dinner", ReplacementRecipeId: replacement.ID}))
	if connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("preview must reject its changed captured recipe input, got %v", err)
	}
}

func TestPreviewSwapPinsReplacementRecipeThroughApply(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, schema := range []string{workspace.Schema(), recipe.Schema(), internal.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	clock := schedule.System()
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	ownerWorkspace, err := workspaces.Create(ctx, workspace.CreateInput{Name: "Personal", OwnerSubject: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	recipes := recipe.NewService(recipe.NewSQLiteRepository(db, clock))
	original, err := recipes.Create(ctx, recipe.CreateInput{WorkspaceID: ownerWorkspace.ID, Name: "Original soup"})
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := recipes.Create(ctx, recipe.CreateInput{WorkspaceID: ownerWorkspace.ID, Name: "Rice"})
	if err != nil {
		t.Fatal(err)
	}
	repository := internal.NewSQLiteRepository(db, clock)
	draft := internal.Draft{RunID: "new-replacement-input", Occurrences: []internal.Occurrence{{Date: "2026-10-03", SlotName: "dinner", RecipeID: original.ID, RecipeRevision: original.Revision, RecipeName: original.Name}}}
	raw, _ := json.Marshal(draft)
	if _, err := repository.Apply(ctx, ownerWorkspace.ID, 0, string(raw)); err != nil {
		t.Fatal(err)
	}
	handler := NewConnectHandler(workspaces, recipes, nil, repository, nil, nil, nil)
	authorized := identity.WithPrincipal(ctx, identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true})
	response, err := handler.PreviewSwap(authorized, connect.NewRequest(&planningv1.PreviewSwapRequest{WorkspaceId: ownerWorkspace.ID, ExpectedRevision: 1, Date: "2026-10-03", SlotName: "dinner", ReplacementRecipeId: replacement.ID}))
	if err != nil {
		t.Fatal(err)
	}
	var preview struct {
		Draft internal.Draft `json:"draft"`
	}
	if err := json.Unmarshal([]byte(response.Msg.PreviewJson), &preview); err != nil {
		t.Fatal(err)
	}
	expectedReference := fmt.Sprintf("recipe:%s:%d", replacement.ID, replacement.Revision)
	found := false
	for _, ref := range preview.Draft.InputReferences {
		if ref == expectedReference {
			found = true
		}
	}
	if !found {
		t.Fatalf("replacement recipe revision not pinned in swap draft: %+v", preview.Draft.InputReferences)
	}
	if _, err := recipes.Update(ctx, recipe.UpdateInput{WorkspaceID: ownerWorkspace.ID, ID: replacement.ID, ExpectedRevision: replacement.Revision, Name: "Rice revised"}); err != nil {
		t.Fatal(err)
	}
	previewJSON, _ := json.Marshal(preview.Draft)
	_, err = handler.ApplyPlan(authorized, connect.NewRequest(&planningv1.ApplyPlanRequest{WorkspaceId: ownerWorkspace.ID, ExpectedRevision: 1, DraftJson: string(previewJSON)}))
	if connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("apply must reject a replacement recipe changed after preview, got %v", err)
	}
	currentRevision, currentRaw, err := repository.Get(ctx, ownerWorkspace.ID)
	var current internal.Draft
	if err == nil {
		err = json.Unmarshal([]byte(currentRaw), &current)
	}
	if err != nil || currentRevision != 1 || current.Occurrences[0].RecipeID != original.ID {
		t.Fatalf("stale replacement preview changed the persisted plan: revision=%d raw=%s err=%v", currentRevision, currentRaw, err)
	}
}

func TestPreviewSwapReportsPreparedBatchAvailabilityWithoutConsumingPortions(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, schema := range []string{workspace.Schema(), recipe.Schema(), internal.Schema(), inventory.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	clock := schedule.System()
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	owner, err := workspaces.Create(ctx, workspace.CreateInput{Name: "Personal", OwnerSubject: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	recipes := recipe.NewService(recipe.NewSQLiteRepository(db, clock))
	original, err := recipes.Create(ctx, recipe.CreateInput{WorkspaceID: owner.ID, Name: "Beans", CanonicalYield: "2", ServingUnit: "servings", Ingredients: []recipe.Ingredient{{ID: "beans", Name: "Beans", Amount: "200", Unit: "g"}}})
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := recipes.Create(ctx, recipe.CreateInput{WorkspaceID: owner.ID, Name: "Rice"})
	if err != nil {
		t.Fatal(err)
	}
	plans := internal.NewSQLiteRepository(db, clock)
	draft := internal.Draft{Occurrences: []internal.Occurrence{{Date: "2026-10-03", SlotName: "dinner", RecipeID: original.ID, RecipeRevision: original.Revision, RecipeName: original.Name}}}
	raw, _ := json.Marshal(draft)
	if _, err := plans.Apply(ctx, owner.ID, 0, string(raw)); err != nil {
		t.Fatal(err)
	}
	inventoryRepository := inventory.NewSQLiteRepository(db).(inventory.BatchRepository)
	yield := decimalx.KnownInt(5)
	if _, err := inventoryRepository.PrepareBatch(ctx, owner.ID, "prepare-beans", "beans-batch", original.ID, original.Revision, yield, "servings", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := inventoryRepository.PrepareBatch(ctx, owner.ID, "prepare-older-beans", "older-beans-batch", original.ID, original.Revision+1, decimalx.KnownInt(2), "servings", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := inventoryRepository.ConsumeBatchPortion(ctx, owner.ID, "eat-beans", "beans-batch", decimalx.KnownInt(1), "servings", original.ID, false); err != nil {
		t.Fatal(err)
	}
	handler := NewConnectHandler(workspaces, recipes, nil, plans, nil, nil, nil, shopping.Evidence{Inventory: inventoryRepository})
	ctx = identity.WithPrincipal(ctx, identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true})
	response, err := handler.PreviewSwap(ctx, connect.NewRequest(&planningv1.PreviewSwapRequest{WorkspaceId: owner.ID, ExpectedRevision: 1, Date: "2026-10-03", SlotName: "dinner", ReplacementRecipeId: replacement.ID}))
	if err != nil {
		t.Fatal(err)
	}
	var preview struct {
		PreparedBatchImpacts []internal.PreparedBatchImpact `json:"preparedBatchImpacts"`
	}
	if err := json.Unmarshal([]byte(response.Msg.GetPreviewJson()), &preview); err != nil {
		t.Fatal(err)
	}
	if len(preview.PreparedBatchImpacts) != 1 || preview.PreparedBatchImpacts[0].BatchID != "beans-batch" || preview.PreparedBatchImpacts[0].RecipeID != original.ID || preview.PreparedBatchImpacts[0].RecipeRevision != original.Revision || preview.PreparedBatchImpacts[0].Available != "4" || preview.PreparedBatchImpacts[0].Unit != "servings" {
		t.Fatalf("swap review must report actual available batch portions: %+v", preview.PreparedBatchImpacts)
	}
	batches, err := inventoryRepository.ListBatches(ctx, owner.ID)
	if err != nil || len(batches) != 2 || batches[0].Available.String() != "4" {
		t.Fatalf("preview must leave batch availability unchanged: %+v err=%v", batches, err)
	}
	var envelope struct {
		Draft json.RawMessage `json:"draft"`
	}
	if err := json.Unmarshal([]byte(response.Msg.GetPreviewJson()), &envelope); err != nil {
		t.Fatal(err)
	}
	if _, err := inventoryRepository.ConsumeBatchPortion(ctx, owner.ID, "eat-beans-after-review", "beans-batch", decimalx.KnownInt(1), "servings", original.ID, false); err != nil {
		t.Fatal(err)
	}
	_, err = handler.ApplyPlan(ctx, connect.NewRequest(&planningv1.ApplyPlanRequest{WorkspaceId: owner.ID, ExpectedRevision: 1, DraftJson: string(envelope.Draft)}))
	if connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("changed prepared availability must invalidate the captured batch impact: %v", err)
	}
	currentRevision, currentRaw, err := plans.Get(ctx, owner.ID)
	var current internal.Draft
	if err == nil {
		err = json.Unmarshal([]byte(currentRaw), &current)
	}
	if err != nil || currentRevision != 1 || current.Occurrences[0].RecipeID != original.ID {
		t.Fatalf("stale batch review must not apply a plan mutation: revision=%d plan=%+v err=%v", currentRevision, current, err)
	}
}

func TestApplyPlanRequiresLockedSlotsAndHonorsExplicitUnlock(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, schema := range []string{workspace.Schema(), recipe.Schema(), internal.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	clock := schedule.System()
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	ownerWorkspace, err := workspaces.Create(ctx, workspace.CreateInput{Name: "Personal", OwnerSubject: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	recipes := recipe.NewService(recipe.NewSQLiteRepository(db, clock))
	soup, err := recipes.Create(ctx, recipe.CreateInput{WorkspaceID: ownerWorkspace.ID, Name: "Soup"})
	if err != nil {
		t.Fatal(err)
	}
	rice, err := recipes.Create(ctx, recipe.CreateInput{WorkspaceID: ownerWorkspace.ID, Name: "Rice"})
	if err != nil {
		t.Fatal(err)
	}
	plans := internal.NewSQLiteRepository(db, clock)
	current := internal.Draft{RunID: "locked-plan", Occurrences: []internal.Occurrence{
		{Date: "2026-10-03", SlotName: "breakfast", RecipeID: soup.ID, RecipeRevision: soup.Revision, RecipeName: soup.Name},
		{Date: "2026-10-03", SlotName: "dinner", RecipeID: soup.ID, RecipeRevision: soup.Revision, RecipeName: soup.Name, Locked: true},
	}}
	raw, _ := json.Marshal(current)
	if revision, err := plans.Apply(ctx, ownerWorkspace.ID, 0, string(raw)); err != nil || revision != 1 {
		t.Fatalf("seed revision=%d err=%v", revision, err)
	}
	handler := NewConnectHandler(workspaces, recipes, nil, plans, nil, nil, nil)
	owner := identity.WithPrincipal(ctx, identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true})
	partial, _ := json.Marshal(internal.Draft{RunID: "partial", Occurrences: current.Occurrences[:1]})
	_, err = handler.ApplyPlan(owner, connect.NewRequest(&planningv1.ApplyPlanRequest{WorkspaceId: ownerWorkspace.ID, ExpectedRevision: 1, DraftJson: string(partial)}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("omitted locked slot error=%v", err)
	}
	updated := internal.Draft{RunID: "unlock", Occurrences: []internal.Occurrence{
		current.Occurrences[0],
		{Date: "2026-10-03", SlotName: "dinner", RecipeID: rice.ID, RecipeRevision: rice.Revision, RecipeName: rice.Name, Reason: "explicitly unlocked", Locked: false},
	}}
	updatedRaw, _ := json.Marshal(updated)
	if _, err := handler.ApplyPlan(owner, connect.NewRequest(&planningv1.ApplyPlanRequest{WorkspaceId: ownerWorkspace.ID, ExpectedRevision: 1, DraftJson: string(updatedRaw)})); err != nil {
		t.Fatalf("explicit unlock and replacement failed: %v", err)
	}
	_, savedRaw, err := plans.Get(ctx, ownerWorkspace.ID)
	if err != nil {
		t.Fatal(err)
	}
	var saved internal.Draft
	if err := json.Unmarshal([]byte(savedRaw), &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Occurrences) != 2 || saved.Occurrences[1].RecipeID != rice.ID || saved.Occurrences[1].Locked {
		t.Fatalf("explicitly updated plan=%+v", saved.Occurrences)
	}
}

func TestNoochAuthenticatedRouteVerifiesManifestAudienceAndIssuer(t *testing.T) {
	db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, TestDriver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "nooch-primary.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	db.SetTestPoolInitializer(func(_ context.Context, pool *sql.DB) error {
		for _, schema := range []string{workspace.Schema(), recipe.Schema(), internal.Schema(), shopping.Schema(), feedback.Schema(), internalsupplement.Schema(), internalnutrition.Schema(), inventory.Schema(), cost.Schema()} {
			if _, err := pool.Exec(schema); err != nil {
				return err
			}
		}
		return nil
	})
	if err := db.InstallTestPool(context.Background(), filepath.Join(t.TempDir(), "nooch-auth-fixture.db"), "nooch-auth-fixture", time.Minute); err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := database.WithTestMode(context.Background())
	clock := schedule.System()
	now := time.Now().UTC()
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	ownerWorkspace, err := workspaces.Create(ctx, workspace.CreateInput{Name: "Personal", OwnerSubject: "signed-in-owner"})
	if err != nil {
		t.Fatal(err)
	}
	recipes := recipe.NewService(recipe.NewSQLiteRepository(db, clock))
	original, err := recipes.Create(ctx, recipe.CreateInput{WorkspaceID: ownerWorkspace.ID, Name: "Beans", CanonicalYield: "2", ServingUnit: "servings", Ingredients: []recipe.Ingredient{{ID: "beans", Name: "Beans", Amount: "200", Unit: "g"}}, Methods: []recipe.Method{{ID: "cook", Steps: []recipe.MethodStep{{ID: "step", Instruction: "Cook beans", Inputs: []string{"beans"}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := recipes.Create(ctx, recipe.CreateInput{WorkspaceID: ownerWorkspace.ID, Name: "Rice", CanonicalYield: "2", ServingUnit: "servings", Ingredients: []recipe.Ingredient{{ID: "rice", Name: "Rice", Amount: "300", Unit: "g"}}, Methods: []recipe.Method{{ID: "cook", Steps: []recipe.MethodStep{{ID: "step", Instruction: "Cook rice", Inputs: []string{"rice"}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	anchor, err := recipes.Create(ctx, recipe.CreateInput{WorkspaceID: ownerWorkspace.ID, Name: "Oats", CanonicalYield: "2", ServingUnit: "servings", Ingredients: []recipe.Ingredient{{ID: "oats", Name: "Oats", Amount: "100", Unit: "g"}}, Methods: []recipe.Method{{ID: "cook", Steps: []recipe.MethodStep{{ID: "step", Instruction: "Cook oats", Inputs: []string{"oats"}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	repository := internal.NewSQLiteRepository(db, clock)
	shoppingRepository := shopping.NewSQLiteRepository(db, clock)
	feedbackRepository := feedback.NewSQLiteRepository(db, clock)
	inventoryRepository := inventory.NewSQLiteRepository(db)
	costRepository := cost.NewSQLiteRepository(db)
	riceStock, _ := decimalx.Parse("50")
	if err := inventoryRepository.Append(ctx, ownerWorkspace.ID, inventory.Event{ID: "rice-stock-50g", Kind: inventory.Purchase, ItemID: "rice", Amount: riceStock, Unit: "g"}); err != nil {
		t.Fatal(err)
	}
	packageAmount, _ := decimalx.Parse("200")
	packagePrice, _ := money.New(400, "USD", 2)
	if _, err := costRepository.Create(ctx, cost.Observation{ID: "rice-package-current", WorkspaceID: ownerWorkspace.ID, ItemID: "rice", PackageLabel: "200 g bag", PackageAmount: packageAmount, PackageUnit: "g", Price: packagePrice, ObservedAt: now.Add(-time.Hour), ValidThrough: now.Add(24 * time.Hour), Available: true, Source: "isolated connected fixture"}); err != nil {
		t.Fatal(err)
	}
	draft := internal.Draft{RunID: "auth-routed-read", Occurrences: []internal.Occurrence{
		{Date: "2026-10-03", SlotName: "dinner", Mode: "flexible", Quantity: "1", RecipeID: original.ID, RecipeRevision: original.Revision, RecipeName: original.Name},
		{Date: "2026-10-03", SlotName: "breakfast", Mode: "fixed", Quantity: "1", RecipeID: anchor.ID, RecipeRevision: anchor.Revision, RecipeName: anchor.Name, Locked: true},
		{Date: "2026-10-03", SlotName: "snack", Mode: "open", Quantity: "1", Reason: "owner left this slot open"},
		{Date: "2026-10-05", SlotName: "dinner", Mode: "flexible", Quantity: "1", RecipeID: replacement.ID, RecipeRevision: replacement.Revision, RecipeName: replacement.Name},
	}}
	raw, _ := json.Marshal(draft)
	if _, err := repository.Apply(ctx, ownerWorkspace.ID, 0, string(raw)); err != nil {
		t.Fatal(err)
	}
	servicePath, serviceHandler := planningconnect.NewPlanningServiceHandler(NewConnectHandler(workspaces, recipes, nil, repository, shoppingRepository, feedbackRepository, nil, shopping.Evidence{Inventory: inventoryRepository, Costs: costRepository, Now: func() time.Time { return now }}))
	supplementPath, supplementServiceHandler := supplementconnect.NewSupplementServiceHandler(supplementhandler.NewConnectHandler(internalsupplement.NewSQLiteRepository(db, clock), workspaces, nil))
	nutritionPath, nutritionServiceHandler := nutritionconnect.NewNutritionServiceHandler(nutritionhandler.NewConnectHandler(internalnutrition.NewSQLiteTargetRepository(db, clock), internalnutrition.NewSQLiteIntakeRepository(db, clock), workspaces, nil))
	routes := mux.NewRouter()
	routes.Handle(servicePath, serviceHandler)
	routes.PathPrefix(servicePath).Handler(serviceHandler)
	routes.Handle(supplementPath, supplementServiceHandler)
	routes.PathPrefix(supplementPath).Handler(supplementServiceHandler)
	routes.Handle(nutritionPath, nutritionServiceHandler)
	routes.PathPrefix(nutritionPath).Handler(nutritionServiceHandler)
	primaryData := filepath.Join(t.TempDir(), "primary-data")
	testData := filepath.Join(t.TempDir(), "test-data")
	if err := os.MkdirAll(primaryData, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(testData, 0o700); err != nil {
		t.Fatal(err)
	}
	fileRoots := filerouting.New(storage.Paths{DataDir: primaryData})
	if err := fileRoots.InstallTestRoots(storage.Paths{DataDir: testData}, "nooch-auth-fixture", time.Minute); err != nil {
		t.Fatal(err)
	}
	routes.HandleFunc("/__fixture/nooch-data", func(w http.ResponseWriter, r *http.Request) {
		principal, err := authn.RequireHuman(r.Context())
		if err != nil {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		root, err := fileRoots.Pick(r.Context(), storage.ClassData)
		if err != nil {
			http.Error(w, "file root unavailable", http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(filepath.Join(root, "authenticated-fixture.txt"), []byte(principal.Subject), 0o600); err != nil {
			http.Error(w, "write failed", http.StatusInternalServerError)
			return
		}
		fileRoots.RecordWrite(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	issuer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"keys":[{"kid":"nooch-fixture","kty":"RSA","alg":"RS256","n":%q,"e":%q}]}`,
			base64.RawURLEncoding.EncodeToString(key.N.Bytes()), base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}))
	}))
	defer issuer.Close()
	provider := authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{
		Issuer: "scenario-authenticator", Audience: "scenario-authenticator:nutrition-planner", JWKSURL: issuer.URL,
		CookieName: "vrooli_access_token",
	})
	authenticated := authn.Middleware(authn.Config{Providers: []authn.Provider{provider}})(routes)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authenticated.ServeHTTP(w, r.WithContext(database.WithTestMode(r.Context())))
	}))
	defer server.Close()
	valid := signNoochFixtureJWT(t, key, "scenario-authenticator", "scenario-authenticator:nutrition-planner", "signed-in-owner", now.Add(time.Hour))
	read := func(token string) (*connect.Response[planningv1.GetPlanResponse], error) {
		client := planningconnect.NewPlanningServiceClient(server.Client(), server.URL)
		req := connect.NewRequest(&planningv1.GetPlanRequest{WorkspaceId: ownerWorkspace.ID, FromDate: "2026-10-03", ToDate: "2026-10-03"})
		req.Header().Set("Cookie", "vrooli_access_token="+token)
		return client.GetPlan(ctx, req)
	}
	got, err := read(valid)
	if err != nil || !got.Msg.GetHasPlan() || got.Msg.GetCurrentRevision() != 1 {
		t.Fatalf("verified owner route: response=%v err=%v", got, err)
	}
	client := planningconnect.NewPlanningServiceClient(server.Client(), server.URL)
	supplementClient := supplementconnect.NewSupplementServiceClient(server.Client(), server.URL)
	nutritionClient := nutritionconnect.NewNutritionServiceClient(server.Client(), server.URL)
	scheduleCreate := connect.NewRequest(&supplementv1.CreateScheduleRequest{WorkspaceId: ownerWorkspace.ID, ProductRevisionId: "vitamin-d@revision-2", Dose: "1000", DoseUnit: "IU", Weekdays: []int32{1, 3, 5}, StartDate: "2026-10-01", EndDate: "2026-12-31", Confirmed: true})
	scheduleCreate.Header().Set("Cookie", "vrooli_access_token="+valid)
	createdSchedule, err := supplementClient.CreateSchedule(ctx, scheduleCreate)
	if err != nil {
		t.Fatalf("authenticated fixed schedule creation: %v", err)
	}
	targetRequest := connect.NewRequest(&nutritionv1.CreateTargetRequest{WorkspaceId: ownerWorkspace.ID, NutrientId: "energy_kcal", Lower: "0", Upper: "2500", Period: "local_day", Scope: "expected_day", Enforcement: "informational", Provenance: "test fixture with explicit known and unknown contributions", EffectiveFrom: now.Format(time.RFC3339Nano), Active: true})
	targetRequest.Header().Set("Cookie", "vrooli_access_token="+valid)
	target, err := nutritionClient.CreateTarget(ctx, targetRequest)
	if err != nil {
		t.Fatalf("authenticated nutrition target creation: %v", err)
	}
	intakeRequest := connect.NewRequest(&nutritionv1.RecordIntakeRequest{WorkspaceId: ownerWorkspace.ID, Id: "intake-dinner-energy", Date: "2026-10-03", RecipeId: original.ID, RecipeRevision: original.Revision, NutrientId: "energy_kcal", Amount: "220", Unit: "kcal", Reason: "recorded dinner"})
	intakeRequest.Header().Set("Cookie", "vrooli_access_token="+valid)
	if recorded, err := nutritionClient.RecordIntake(ctx, intakeRequest); err != nil || recorded.Msg.GetEvent().GetAmount() != "220" {
		t.Fatalf("authenticated intake record: response=%v err=%v", recorded, err)
	}
	intakesRequest := connect.NewRequest(&nutritionv1.ListIntakesRequest{WorkspaceId: ownerWorkspace.ID})
	intakesRequest.Header().Set("Cookie", "vrooli_access_token="+valid)
	intakeLog, err := nutritionClient.ListIntakes(ctx, intakesRequest)
	if err != nil || len(intakeLog.Msg.GetEvents()) != 1 || intakeLog.Msg.GetEvents()[0].GetId() != "intake-dinner-energy" {
		t.Fatalf("persisted intake log: response=%v err=%v", intakeLog, err)
	}
	evaluate := func(scope string, values []*nutritionv1.Intake) *nutritionv1.EvaluateScopeResponse {
		t.Helper()
		request := connect.NewRequest(&nutritionv1.EvaluateScopeRequest{WorkspaceId: ownerWorkspace.ID, TargetId: target.Msg.GetTarget().GetId(), TargetRevision: target.Msg.GetTarget().GetRevision(), NutrientId: "energy_kcal", Scope: scope, Intakes: values})
		request.Header().Set("Cookie", "vrooli_access_token="+valid)
		response, evalErr := nutritionClient.EvaluateScope(ctx, request)
		if evalErr != nil {
			t.Fatalf("authenticated %s nutrition evaluation: %v", scope, evalErr)
		}
		return response.Msg
	}
	dayFacts := []*nutritionv1.Intake{{Date: "2026-10-03", NutrientId: "energy_kcal", Planned: "300"}, {Date: "2026-10-03", NutrientId: "energy_kcal", Planned: "500", Actual: "220", Recorded: true}, {Date: "2026-10-03", NutrientId: "energy_kcal", Past: false}}
	dayEvaluation := evaluate("expected_day", dayFacts)
	if dayEvaluation.GetKnown() != "520" || dayEvaluation.GetComplete() || len(dayEvaluation.GetUnresolved()) != 1 {
		t.Fatalf("expected-day arithmetic must replace recorded dinner plan once and preserve unknown snack: %+v", dayEvaluation)
	}
	mealEvaluation := evaluate("selected_meal", dayFacts[1:2])
	if mealEvaluation.GetKnown() != "500" || !mealEvaluation.GetComplete() {
		t.Fatalf("selected meal planned scope=%+v", mealEvaluation)
	}
	recordedEvaluation := evaluate("recorded_so_far", dayFacts[1:2])
	if recordedEvaluation.GetKnown() != "220" || !recordedEvaluation.GetComplete() {
		t.Fatalf("recorded intake scope=%+v", recordedEvaluation)
	}
	generatedRequest := connect.NewRequest(&planningv1.GeneratePlanRequest{WorkspaceId: ownerWorkspace.ID, Dates: []string{"2026-10-03"}, MealSlots: []*planningv1.MealSlot{
		{Date: "2026-10-03", SlotName: "breakfast", Mode: "fixed", Quantity: "1", LockedRecipeId: anchor.ID},
		{Date: "2026-10-03", SlotName: "dinner", Mode: "flexible", Quantity: "1"},
		{Date: "2026-10-03", SlotName: "snack", Mode: "open", Quantity: "1"},
	}})
	generatedRequest.Header().Set("Cookie", "vrooli_access_token="+valid)
	generated, err := client.GeneratePlan(ctx, generatedRequest)
	if err != nil {
		t.Fatalf("authenticated plan draft: %v", err)
	}
	var generatedDraft internal.Draft
	if err := json.Unmarshal([]byte(generated.Msg.GetDraftJson()), &generatedDraft); err != nil {
		t.Fatal(err)
	}
	if len(generatedDraft.Occurrences) != 3 || !generatedDraft.Occurrences[0].Locked || generatedDraft.Occurrences[0].RecipeID != anchor.ID || generatedDraft.Occurrences[2].Mode != "open" || generatedDraft.Occurrences[2].RecipeID != "" {
		t.Fatalf("generated draft lost lock/open slot identity: %+v", generatedDraft.Occurrences)
	}
	if stillSaved, err := read(valid); err != nil || stillSaved.Msg.GetCurrentRevision() != 1 {
		t.Fatalf("draft generation must leave the saved revision unchanged: response=%v err=%v", stillSaved, err)
	}
	feedbackRequest := connect.NewRequest(&planningv1.RecordFeedbackRequest{WorkspaceId: ownerWorkspace.ID, ExpectedRevision: 1, Date: "2026-10-03", RecipeId: original.ID, Portion: "one serving", Minutes: 28})
	feedbackRequest.Header().Set("Cookie", "vrooli_access_token="+valid)
	if recorded, err := client.RecordFeedback(ctx, feedbackRequest); err != nil || recorded.Msg.GetRecipeId() != original.ID {
		t.Fatalf("authenticated feedback record: response=%v err=%v", recorded, err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		request := connect.NewRequest(&planningv1.SetShoppingCheckedRequest{WorkspaceId: ownerWorkspace.ID, LineKey: "ingredient:beans", Checked: true})
		request.Header().Set("Cookie", "vrooli_access_token="+valid)
		if response, err := client.SetShoppingChecked(ctx, request); err != nil || !response.Msg.GetChecked() {
			t.Fatalf("authenticated checklist retry %d: response=%v err=%v", attempt+1, response, err)
		}
	}
	checked, err := shoppingRepository.Checked(ctx, ownerWorkspace.ID)
	if err != nil || !checked["ingredient:beans"] {
		t.Fatalf("repeated checklist mutation must converge to one checked fact: checked=%v err=%v", checked, err)
	}
	// A checked row is only a picked-up checklist fact. It must not append
	// stock, purchase, preparation, batch-portion, or meal-intake evidence.
	shoppingEventsAfterCheck, err := inventoryRepository.List(ctx, ownerWorkspace.ID)
	if err != nil || len(shoppingEventsAfterCheck) != 1 || shoppingEventsAfterCheck[0].ID != "rice-stock-50g" || shoppingEventsAfterCheck[0].Kind != inventory.Purchase {
		t.Fatalf("checking a shopping row must leave the routed inventory ledger unchanged: events=%+v err=%v", shoppingEventsAfterCheck, err)
	}
	haveRequest := connect.NewRequest(&planningv1.SetShoppingHaveThisRequest{WorkspaceId: ownerWorkspace.ID, LineKey: "ingredient:beans", HaveThis: true})
	haveRequest.Header().Set("Cookie", "vrooli_access_token="+valid)
	if have, err := client.SetShoppingHaveThis(ctx, haveRequest); err != nil || !have.Msg.GetHaveThis() {
		t.Fatalf("qualitative Have-this assertion: response=%v err=%v", have, err)
	}
	actual, _ := decimalx.Parse("25")
	for attempt := 0; attempt < 2; attempt++ {
		confirmRequest := connect.NewRequest(&planningv1.ConfirmShoppingPurchasesRequest{WorkspaceId: ownerWorkspace.ID, ReviewId: "trip-review-1", Lines: []*planningv1.ShoppingPurchaseActual{{LineKey: "ingredient:rice", ItemId: "rice", Amount: "25", Unit: "g", Price: "0.75"}, {LineKey: "ingredient:beans", ItemId: "beans", Unit: "g", Omitted: true}}})
		confirmRequest.Header().Set("Cookie", "vrooli_access_token="+valid)
		if confirmed, err := client.ConfirmShoppingPurchases(ctx, confirmRequest); err != nil || !confirmed.Msg.GetConfirmed() {
			t.Fatalf("actual purchase retry %d: response=%v err=%v", attempt+1, confirmed, err)
		}
	}
	if events, err := inventoryRepository.List(ctx, ownerWorkspace.ID); err != nil || len(events) != 2 || events[1].Amount.String() != actual.String() || events[1].Kind != inventory.Purchase {
		t.Fatalf("confirmed partial purchase must append once: events=%+v err=%v", events, err)
	}
	checkedPreviewRequest := connect.NewRequest(&planningv1.GetShoppingPreviewRequest{WorkspaceId: ownerWorkspace.ID, ExpectedRevision: 1})
	checkedPreviewRequest.Header().Set("Cookie", "vrooli_access_token="+valid)
	checkedPreview, err := client.GetShoppingPreview(ctx, checkedPreviewRequest)
	if err != nil {
		t.Fatalf("authenticated shopping reload after picked-up check: %v", err)
	}
	foundChecked := false
	for _, line := range checkedPreview.Msg.GetLines() {
		if line.GetKey() == "ingredient:beans" {
			foundChecked = line.GetChecked() && line.GetHaveThis() && line.GetNeed() == "100 g" && line.GetStock() == "unknown"
		}
		if line.GetKey() == "ingredient:rice" && (line.GetStock() != "75 g" || line.GetMissing() != "75 g") {
			t.Fatalf("actual partial purchase must reduce remaining need: line=%+v", line)
		}
	}
	if !foundChecked {
		t.Fatalf("picked-up check must persist without inventing stock or reducing remaining need: %+v", checkedPreview.Msg.GetLines())
	}
	swapRequest := connect.NewRequest(&planningv1.PreviewSwapRequest{WorkspaceId: ownerWorkspace.ID, ExpectedRevision: 1, Date: "2026-10-03", SlotName: "dinner", ReplacementRecipeId: replacement.ID})
	swapRequest.Header().Set("Cookie", "vrooli_access_token="+valid)
	swap, err := client.PreviewSwap(ctx, swapRequest)
	if err != nil {
		t.Fatalf("authenticated connected swap preview: %v", err)
	}
	var swapPreview struct {
		ShoppingChanges []shopping.Change
	}
	if err := json.Unmarshal([]byte(swap.Msg.GetPreviewJson()), &swapPreview); err != nil {
		t.Fatal(err)
	}
	if len(swapPreview.ShoppingChanges) != 2 || swapPreview.ShoppingChanges[0].Key != "ingredient:beans" || swapPreview.ShoppingChanges[0].Before == nil || swapPreview.ShoppingChanges[1].Key != "ingredient:rice" || swapPreview.ShoppingChanges[1].Before == nil || swapPreview.ShoppingChanges[1].After == nil {
		t.Fatalf("connected swap shopping impact=%+v", swapPreview.ShoppingChanges)
	}
	riceImpact := swapPreview.ShoppingChanges[1].After
	if swapPreview.ShoppingChanges[1].Before.Need != "150 g" || riceImpact.Need != "300 g" || riceImpact.Stock != "75 g" || riceImpact.Missing != "225 g" || riceImpact.PackageCount != "2" || riceImpact.Price != "4.00 USD / 200 g" || riceImpact.PortionCost != "6 USD" || riceImpact.CheckoutTotal != "8 USD" || riceImpact.ActualSpend != "unknown" {
		t.Fatalf("authenticated swap must expose package-aware stock and price impacts without claiming spend: %+v", riceImpact)
	}
	encodedRiceImpact, err := json.Marshal(riceImpact)
	if err != nil {
		t.Fatal(err)
	}
	var riceImpactJSON map[string]any
	if err := json.Unmarshal(encodedRiceImpact, &riceImpactJSON); err != nil {
		t.Fatal(err)
	}
	if riceImpactJSON["label"] != "rice" || riceImpactJSON["need"] != "300 g" || riceImpactJSON["stock"] != "75 g" || riceImpactJSON["missing"] != "225 g" || riceImpactJSON["packageCount"] != "2" || riceImpactJSON["price"] != "4.00 USD / 200 g" || riceImpactJSON["portionCost"] != "6 USD" || riceImpactJSON["checkoutTotal"] != "8 USD" || riceImpactJSON["actualSpend"] != "unknown" {
		t.Fatalf("authenticated shopping impact JSON must use the UI's camelCase fields: %s", encodedRiceImpact)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(swap.Msg.GetPreviewJson()), &envelope); err != nil {
		t.Fatal(err)
	}
	applyRequest := connect.NewRequest(&planningv1.ApplyPlanRequest{WorkspaceId: ownerWorkspace.ID, ExpectedRevision: 1, DraftJson: string(envelope["draft"])})
	applyRequest.Header().Set("Cookie", "vrooli_access_token="+valid)
	stockIncrease, _ := decimalx.Parse("10")
	if err := inventoryRepository.Append(ctx, ownerWorkspace.ID, inventory.Event{ID: "rice-stock-added-after-review", Kind: inventory.Correction, ItemID: "rice", Amount: stockIncrease, Unit: "g"}); err != nil {
		t.Fatal(err)
	}
	newPackagePrice, _ := money.New(300, "USD", 2)
	if _, err := costRepository.Create(ctx, cost.Observation{ID: "rice-package-better-after-review", WorkspaceID: ownerWorkspace.ID, ItemID: "rice", PackageLabel: "200 g bag", PackageAmount: packageAmount, PackageUnit: "g", Price: newPackagePrice, ObservedAt: now, ValidThrough: now.Add(24 * time.Hour), Available: true, Source: "isolated updated connected fixture"}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ApplyPlan(ctx, applyRequest); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("changed stock and price must invalidate the captured shopping review: %v", err)
	}
	unchangedRevision, unchangedRaw, err := repository.Get(ctx, ownerWorkspace.ID)
	var unchanged internal.Draft
	if err == nil {
		err = json.Unmarshal([]byte(unchangedRaw), &unchanged)
	}
	unchangedDinner := ""
	for _, occurrence := range unchanged.Occurrences {
		if occurrence.Date == "2026-10-03" && occurrence.SlotName == "dinner" {
			unchangedDinner = occurrence.RecipeID
		}
	}
	if err != nil || unchangedRevision != 1 || unchangedDinner != original.ID {
		t.Fatalf("stale shopping review must not apply a plan mutation: revision=%d plan=%+v err=%v", unchangedRevision, unchanged, err)
	}
	swapRequest = connect.NewRequest(&planningv1.PreviewSwapRequest{WorkspaceId: ownerWorkspace.ID, ExpectedRevision: 1, Date: "2026-10-03", SlotName: "dinner", ReplacementRecipeId: replacement.ID})
	swapRequest.Header().Set("Cookie", "vrooli_access_token="+valid)
	swap, err = client.PreviewSwap(ctx, swapRequest)
	if err != nil {
		t.Fatalf("refresh current shopping preview: %v", err)
	}
	if err := json.Unmarshal([]byte(swap.Msg.GetPreviewJson()), &swapPreview); err != nil {
		t.Fatal(err)
	}
	riceImpact = swapPreview.ShoppingChanges[1].After
	if riceImpact.Stock != "85 g" || riceImpact.Missing != "215 g" || riceImpact.Price != "3.00 USD / 200 g" || riceImpact.PortionCost != "4.5 USD" || riceImpact.CheckoutTotal != "6 USD" {
		t.Fatalf("refreshed shopping impact did not use current stock and price: %+v", riceImpact)
	}
	if err := json.Unmarshal([]byte(swap.Msg.GetPreviewJson()), &envelope); err != nil {
		t.Fatal(err)
	}
	applyRequest = connect.NewRequest(&planningv1.ApplyPlanRequest{WorkspaceId: ownerWorkspace.ID, ExpectedRevision: 1, DraftJson: string(envelope["draft"])})
	applyRequest.Header().Set("Cookie", "vrooli_access_token="+valid)
	applied, err := client.ApplyPlan(ctx, applyRequest)
	if err != nil || applied.Msg.GetRevision() != 2 {
		t.Fatalf("authenticated apply of reviewed swap: response=%v err=%v", applied, err)
	}
	schedulesRequest := connect.NewRequest(&supplementv1.ListSchedulesRequest{WorkspaceId: ownerWorkspace.ID})
	schedulesRequest.Header().Set("Cookie", "vrooli_access_token="+valid)
	schedules, err := supplementClient.ListSchedules(ctx, schedulesRequest)
	if err != nil || len(schedules.Msg.GetSchedules()) != 1 || schedules.Msg.GetSchedules()[0].GetDose() != createdSchedule.Msg.GetSchedule().GetDose() || schedules.Msg.GetSchedules()[0].GetRevision() != createdSchedule.Msg.GetSchedule().GetRevision() {
		t.Fatalf("plan change must not adjust fixed supplement schedule: schedules=%v err=%v", schedules, err)
	}
	staleApply := connect.NewRequest(&planningv1.ApplyPlanRequest{WorkspaceId: ownerWorkspace.ID, ExpectedRevision: 1, DraftJson: string(raw)})
	staleApply.Header().Set("Cookie", "vrooli_access_token="+valid)
	if _, err := client.ApplyPlan(ctx, staleApply); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("authenticated stale plan write error=%v, want aborted", err)
	}
	reloadRequest := connect.NewRequest(&planningv1.GetPlanRequest{WorkspaceId: ownerWorkspace.ID, FromDate: "2026-10-03", ToDate: "2026-10-03"})
	reloadRequest.Header().Set("Cookie", "vrooli_access_token="+valid)
	reloaded, err := client.GetPlan(ctx, reloadRequest)
	if err != nil {
		t.Fatal(err)
	}
	var persisted internal.Draft
	if err := json.Unmarshal([]byte(reloaded.Msg.GetDraftJson()), &persisted); err != nil {
		t.Fatal(err)
	}
	if reloaded.Msg.GetCurrentRevision() != 2 || len(persisted.Occurrences) != 3 {
		t.Fatalf("reloaded persisted swap=%+v revision=%d", persisted, reloaded.Msg.GetCurrentRevision())
	}
	history, err := feedbackRepository.Get(ctx, ownerWorkspace.ID, "2026-10-03")
	if err != nil || history.RecipeID != original.ID || history.Portion != "one serving" || history.Minutes != 28 {
		t.Fatalf("plan apply must preserve recorded meal history: history=%+v err=%v", history, err)
	}
	bySlot := map[string]internal.Occurrence{}
	for _, occurrence := range persisted.Occurrences {
		bySlot[occurrence.SlotName] = occurrence
	}
	if bySlot["dinner"].RecipeID != replacement.ID || !bySlot["breakfast"].Locked || bySlot["breakfast"].RecipeID != anchor.ID || bySlot["snack"].Mode != "open" || bySlot["snack"].RecipeID != "" {
		t.Fatalf("swap lost locked anchor or open slot: %+v", persisted.Occurrences)
	}
	weekRequest := connect.NewRequest(&planningv1.GetPlanRequest{WorkspaceId: ownerWorkspace.ID, FromDate: "2026-10-03", ToDate: "2026-10-09"})
	weekRequest.Header().Set("Cookie", "vrooli_access_token="+valid)
	weekPlan, err := client.GetPlan(ctx, weekRequest)
	if err != nil {
		t.Fatal(err)
	}
	var weekDraft internal.Draft
	if err := json.Unmarshal([]byte(weekPlan.Msg.GetDraftJson()), &weekDraft); err != nil {
		t.Fatal(err)
	}
	if len(weekDraft.Occurrences) != 4 || weekPlan.Msg.GetCurrentRevision() != reloaded.Msg.GetCurrentRevision() || !reflect.DeepEqual(weekDraft.Occurrences[:len(persisted.Occurrences)], persisted.Occurrences) || weekDraft.Occurrences[3].Date != "2026-10-05" || weekDraft.Occurrences[3].RecipeID != replacement.ID {
		t.Fatalf("Today and Week range reads diverged: today=%+v week=%+v", persisted.Occurrences, weekDraft.Occurrences)
	}
	shoppingRequest := connect.NewRequest(&planningv1.GetShoppingPreviewRequest{WorkspaceId: ownerWorkspace.ID, ExpectedRevision: 2})
	shoppingRequest.Header().Set("Cookie", "vrooli_access_token="+valid)
	shoppingPreview, err := client.GetShoppingPreview(ctx, shoppingRequest)
	if err != nil {
		t.Fatal(err)
	}
	if len(shoppingPreview.Msg.GetLines()) != 2 || shoppingPreview.Msg.GetLines()[0].GetKey() != "ingredient:oats" || shoppingPreview.Msg.GetLines()[0].GetStock() != "unknown" || shoppingPreview.Msg.GetLines()[0].GetPrice() != "unknown" || shoppingPreview.Msg.GetLines()[1].GetKey() != "ingredient:rice" || shoppingPreview.Msg.GetLines()[1].GetNeed() != "300 g" || shoppingPreview.Msg.GetLines()[1].GetStock() != "85 g" || shoppingPreview.Msg.GetLines()[1].GetMissing() != "215 g" || shoppingPreview.Msg.GetLines()[1].GetPackageCount() != "2" || shoppingPreview.Msg.GetLines()[1].GetPortionCost() != "4.5 USD" || shoppingPreview.Msg.GetLines()[1].GetCheckoutTotal() != "6 USD" || shoppingPreview.Msg.GetLines()[1].GetActualSpend() != "unknown" {
		t.Fatalf("persisted shopping preview=%+v", shoppingPreview.Msg.GetLines())
	}
	stockAfterPlan, err := inventoryRepository.List(ctx, ownerWorkspace.ID)
	stockTotal := decimalx.KnownInt(0)
	for _, event := range stockAfterPlan {
		stockTotal, _ = decimalx.Add(stockTotal, event.Amount)
	}
	if err != nil || len(stockAfterPlan) != 3 || stockTotal.String() != "85" {
		t.Fatalf("planned future and current portions must not reserve or consume stock: events=%+v err=%v", stockAfterPlan, err)
	}
	wrongAudience := signNoochFixtureJWT(t, key, "scenario-authenticator", "scenario-authenticator:other", "signed-in-owner", now.Add(time.Hour))
	if _, err := read(wrongAudience); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("wrong audience error=%v", err)
	}
	wrongIssuer := signNoochFixtureJWT(t, key, "untrusted-issuer", "scenario-authenticator:nutrition-planner", "signed-in-owner", now.Add(time.Hour))
	if _, err := read(wrongIssuer); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("wrong issuer error=%v", err)
	}
	foreignOwner := signNoochFixtureJWT(t, key, "scenario-authenticator", "scenario-authenticator:nutrition-planner", "different-owner", now.Add(time.Hour))
	if _, err := read(foreignOwner); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("foreign owner error=%v", err)
	}
	writeReq, _ := http.NewRequest(http.MethodPost, server.URL+"/__fixture/nooch-data", nil)
	writeReq.AddCookie(&http.Cookie{Name: "vrooli_access_token", Value: valid})
	writeResp, err := server.Client().Do(writeReq)
	if err != nil {
		t.Fatal(err)
	}
	writeResp.Body.Close()
	if writeResp.StatusCode != http.StatusNoContent {
		t.Fatalf("authenticated routed file write status=%d", writeResp.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(primaryData, "authenticated-fixture.txt")); !os.IsNotExist(err) {
		t.Fatalf("authenticated test write leaked into primary file root: err=%v", err)
	}
	contents, err := os.ReadFile(filepath.Join(testData, "authenticated-fixture.txt"))
	if err != nil || string(contents) != "signed-in-owner" {
		t.Fatalf("test-root file contents=%q err=%v", contents, err)
	}
	fileStats := fileRoots.LeaseStats()
	if fileStats.TestRootWrites != 1 || fileStats.PrimaryWritesDuringTestMode != 0 {
		t.Fatalf("routed file stats=%+v", fileStats)
	}
	stats := db.LeaseStats()
	if stats.TestPoolRequests == 0 || stats.PrimaryDuringTestModeRequests != 0 {
		t.Fatalf("routed storage stats=%+v; authenticated route should use only the isolated test SQL pool", stats)
	}
	var primaryPlanTable int
	if err := db.Primary().QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='plan_snapshots'`).Scan(&primaryPlanTable); err != nil {
		t.Fatal(err)
	}
	if primaryPlanTable != 0 {
		t.Fatalf("test schema leaked into primary database: found %d plan tables", primaryPlanTable)
	}
}

func signNoochFixtureJWT(t *testing.T, key *rsa.PrivateKey, issuer, audience, subject string, expiry time.Time) string {
	t.Helper()
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "nooch-fixture", "typ": "JWT"})
	claims, _ := json.Marshal(map[string]any{"sub": subject, "iss": issuer, "aud": audience, "exp": expiry.Unix()})
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}
