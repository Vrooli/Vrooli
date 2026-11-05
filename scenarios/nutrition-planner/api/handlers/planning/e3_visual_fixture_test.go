package planning

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	cataloghandler "nutrition-planner/handlers/catalog"
	cookinghandler "nutrition-planner/handlers/cooking"
	nutritionhandler "nutrition-planner/handlers/nutrition"
	profilehandler "nutrition-planner/handlers/profile"
	recipehandler "nutrition-planner/handlers/recipe"
	supplementhandler "nutrition-planner/handlers/supplement"
	workspacehandler "nutrition-planner/handlers/workspace"
	internalcatalog "nutrition-planner/internal/catalog"
	"nutrition-planner/internal/cost"
	internalcooking "nutrition-planner/internal/cooking"
	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/feedback"
	"nutrition-planner/internal/inventory"
	"nutrition-planner/internal/money"
	internalnutrition "nutrition-planner/internal/nutrition"
	internal "nutrition-planner/internal/planning"
	internalprofile "nutrition-planner/internal/profile"
	"nutrition-planner/internal/recipe"
	"nutrition-planner/internal/shopping"
	internalsupplement "nutrition-planner/internal/supplement"
	"nutrition-planner/internal/workspace"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/catalog/catalog_v1connect"
	cookingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/cooking/cooking_v1connect"
	nutritionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/nutrition/nutrition_v1connect"
	planningconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/planning/planning_v1connect"
	profileconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/profile/profile_v1connect"
	recipeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/recipe/recipe_v1connect"
	supplementconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/supplement/supplement_v1connect"
	v1workspace "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/workspace"
	wspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/workspace/workspace_v1connect"
	_ "modernc.org/sqlite"
)

// TestRunE3VisualFixture runs only on explicit request. It serves the production
// UI build and serves the configured local workspace through the same anonymous
// API contract as the application, backed by an isolated RoutedDB test pool.
func TestRunE3VisualFixture(t *testing.T) {
	if os.Getenv("E3_VISUAL_FIXTURE") != "1" {
		t.Skip("set E3_VISUAL_FIXTURE=1 to serve the isolated visual fixture")
	}
	stop := make(chan os.Signal, 1)
	emptyBootstrap := os.Getenv("E3_VISUAL_FIXTURE_EMPTY_BOOTSTRAP") == "1"
	var server *httptest.Server
	var err error
	if emptyBootstrap {
		server, err = newE3EmptyBootstrapVisualFixture(t, stop)
	} else {
		server, err = newE3VisualFixture(t, stop)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	resp, err := server.Client().Get(server.URL + "/today")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("anonymous UI route status=%d", resp.StatusCode)
	}
	if emptyBootstrap {
		client := wspaceconnect.NewWorkspaceServiceClient(server.Client(), server.URL)
		listed, err := client.ListWorkspaces(context.Background(), connect.NewRequest(&v1workspace.ListWorkspacesRequest{}))
		if err != nil {
			t.Fatalf("empty fixture workspace preflight failed: %v", err)
		}
		if got := len(listed.Msg.GetWorkspaces()); got != 0 {
			t.Fatalf("empty bootstrap fixture has %d workspaces before browser load, want 0", got)
		}
		fmt.Println("E3_VISUAL_FIXTURE_MODE=empty-bootstrap; workspace list is empty before browser load")
	}
	fmt.Printf("E3_VISUAL_FIXTURE_URL=%s/today\n", server.URL)
	signal.Notify(stop, os.Interrupt)
	defer signal.Stop(stop)
	<-stop
}

func newE3VisualFixture(t *testing.T, stop chan<- os.Signal) (*httptest.Server, error) {
	t.Helper()
	ctx := context.Background()
	temp := t.TempDir()
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, TestDriver: database.DriverSQLite, DSN: filepath.Join(temp, "primary.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = db.Close() })
	clock := schedule.System()
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		for _, schema := range []string{workspace.Schema(), recipe.Schema(), internal.Schema(), shopping.Schema(), feedback.Schema(), internalprofile.Schema(), internalnutrition.Schema(), internalsupplement.Schema(), internalcatalog.Schema(), inventory.Schema(), cost.Schema(), internalcooking.Schema()} {
			if _, err := pool.ExecContext(ctx, schema); err != nil {
				return err
			}
		}
		return nil
	})
	if err := db.InstallTestPool(ctx, filepath.Join(temp, "fixture.db"), "nutrition-planner-e3-visual-fixture", time.Hour); err != nil {
		return nil, err
	}
	// The profile repository inspects its columns at construction using a
	// background context, before request contexts select the isolated test pool.
	// Keep that probe inside the disposable fixture's primary database too.
	if _, err := db.ExecContext(ctx, internalprofile.Schema()); err != nil {
		return nil, fmt.Errorf("initialize fixture profile schema: %w", err)
	}
	dataCtx := database.WithTestMode(ctx)
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	owner, err := workspaces.Create(dataCtx, workspace.CreateInput{Name: "Nooch visual review", OwnerSubject: workspace.LocalOwnerSubject})
	if err != nil {
		return nil, err
	}
	recipes := recipe.NewService(recipe.NewSQLiteRepository(db, clock))
	profiles := internalprofile.NewService(internalprofile.NewSQLiteRepository(db, clock))
	makeRecipe := func(name, ingredientID string, amount string) (recipe.Recipe, error) {
		return recipes.Create(dataCtx, recipe.CreateInput{WorkspaceID: owner.ID, Name: name, Notes: "A clear, practical meal for the selected Nooch visual fixture.", CanonicalYield: "2", ServingUnit: "servings", Ingredients: []recipe.Ingredient{{ID: ingredientID, Name: ingredientID, Amount: amount, Unit: "g"}}, Methods: []recipe.Method{{ID: "main", Name: "Stovetop", Steps: []recipe.MethodStep{{ID: "prepare", Instruction: "Prepare the ingredients", Inputs: []string{ingredientID}}, {ID: "cook", Instruction: "Cook until ready", Inputs: []string{ingredientID}}}}}})
	}
	oats, err := makeRecipe("Cinnamon pear oats", "oats", "160")
	if err != nil {
		return nil, err
	}
	tofu, err := makeRecipe("Crispy tofu bowls", "tofu", "400")
	if err != nil {
		return nil, err
	}
	beans, err := makeRecipe("Smoky bean chili", "beans", "360")
	if err != nil {
		return nil, err
	}
	salmon, err := makeRecipe("Lemon salmon plate", "salmon", "340")
	if err != nil {
		return nil, err
	}

	localNow := time.Now().In(time.Local)
	today := localNow.Format("2006-01-02")
	dates := make([]string, 7)
	occurrences := make([]internal.Occurrence, 0, 22)
	for day := 0; day < len(dates); day++ {
		date := localNow.AddDate(0, 0, day).Format("2006-01-02")
		dates[day] = date
		if day == 0 {
			occurrences = append(occurrences,
				internal.Occurrence{Date: date, SlotName: "breakfast", Mode: "fixed", Quantity: "1", RecipeID: oats.ID, RecipeRevision: oats.Revision, RecipeName: oats.Name, Reason: "Pinned morning anchor", Locked: true},
				internal.Occurrence{Date: date, SlotName: "lunch", Mode: "flexible", Quantity: "1", RecipeID: tofu.ID, RecipeRevision: tofu.Revision, RecipeName: tofu.Name, Reason: "Fits the saved routine"},
				internal.Occurrence{Date: date, SlotName: "dinner", Mode: "flexible", Quantity: "1", RecipeID: beans.ID, RecipeRevision: beans.Revision, RecipeName: beans.Name, Reason: "Balanced weeknight option"},
				internal.Occurrence{Date: date, SlotName: "snack", Mode: "open", Quantity: "1", Reason: "Left open by the user"})
			continue
		}
		selected := []recipe.Recipe{tofu, beans, salmon}[(day-1)%3]
		occurrences = append(occurrences, internal.Occurrence{Date: date, SlotName: "dinner", Mode: "flexible", Quantity: "1", RecipeID: selected.ID, RecipeRevision: selected.Revision, RecipeName: selected.Name, Reason: "Saved week plan"})
	}
	planRepo := internal.NewSQLiteRepository(db, clock)
	draft := internal.Draft{RunID: "e3-visual-fixture", Occurrences: occurrences, InputReferences: []string{"isolated fixture recipes", "isolated fixed schedule", "isolated inventory and current package observation"}}
	raw, err := json.Marshal(draft)
	if err != nil {
		return nil, err
	}
	if _, err := planRepo.Apply(dataCtx, owner.ID, 0, string(raw)); err != nil {
		return nil, err
	}
	feedbackRepo := feedback.NewSQLiteRepository(db, clock)
	shoppingRepo := shopping.NewSQLiteRepository(db, clock)
	inventoryRepo := inventory.NewSQLiteRepository(db)
	costRepo := cost.NewSQLiteRepository(db)
	batchRepo := inventoryRepo.(inventory.BatchRepository)
	if _, err := batchRepo.PrepareBatch(dataCtx, owner.ID, "fixture-beans-preparation", "fixture-beans-batch", beans.ID, beans.Revision, decimalx.KnownInt(4), "servings", nil); err != nil {
		return nil, err
	}
	if _, err := batchRepo.ConsumeBatchPortion(dataCtx, owner.ID, "fixture-beans-served", "fixture-beans-batch", decimalx.KnownInt(2), "servings", beans.ID, false); err != nil {
		return nil, err
	}
	stockAmount, _ := decimalx.Parse("200")
	if err := inventoryRepo.Append(dataCtx, owner.ID, inventory.Event{ID: "fixture-tofu-stock", Kind: inventory.Purchase, ItemID: "tofu", Amount: stockAmount, Unit: "g"}); err != nil {
		return nil, err
	}
	packageAmount, _ := decimalx.Parse("400")
	packagePrice, _ := money.New(599, "USD", 2)
	if _, err := costRepo.Create(dataCtx, cost.Observation{ID: "fixture-tofu-package", WorkspaceID: owner.ID, ItemID: "tofu", PackageLabel: "400 g tofu pack", PackageAmount: packageAmount, PackageUnit: "g", Price: packagePrice, Retailer: "Fixture market", ObservedAt: time.Now().Add(-time.Hour), ValidThrough: time.Now().Add(24 * time.Hour), Available: true, Source: "isolated visual fixture"}); err != nil {
		return nil, err
	}
	catalogRepo := internalcatalog.NewSQLiteRepository(db, clock)
	catalogBasis := decimalx.KnownInt(1000)
	vitaminAmount, _ := decimalx.Parse("25")
	vitamin, err := catalogRepo.Create(dataCtx, internalcatalog.Revision{ID: "fixture-vitamin-d", Revision: 2, WorkspaceID: owner.ID, ConceptID: "supplement:fixture-vitamin-d", Name: "Vitamin D", ProductName: "Vitamin D3", ServingQuantity: catalogBasis, ServingUnit: "IU", Nutrients: []internalcatalog.NutrientValue{{NutrientID: "vitamin_d", Amount: vitaminAmount, Unit: "mcg", Basis: catalogBasis, BasisUnit: "IU", Evidence: internalcatalog.EvidenceLabel, SourceRef: "isolated fixture label"}}, SourceType: "label", SourceRef: "isolated fixture label"})
	if err != nil {
		return nil, err
	}
	supplementRepo := internalsupplement.NewSQLiteRepository(db, clock)
	weekday := int(localNow.Weekday())
	if _, err := supplementRepo.Create(dataCtx, internalsupplement.Schedule{WorkspaceID: owner.ID, ProductRevisionID: fmt.Sprintf("catalog:%s:%d", vitamin.ID, vitamin.Revision), Dose: catalogBasis, DoseUnit: "IU", Weekdays: []int{weekday}, StartDate: today, EndDate: localNow.AddDate(0, 0, 60).Format("2006-01-02"), Confirmed: true}); err != nil {
		return nil, err
	}
	intakeRepo := internalnutrition.NewSQLiteIntakeRepository(db, clock)
	intakeAmount := decimalx.KnownInt(420)
	if err := intakeRepo.Append(dataCtx, owner.ID, internalnutrition.IntakeEvent{ID: "fixture-recorded-energy", WorkspaceID: owner.ID, Date: today, RecipeID: tofu.ID, RecipeRevision: tofu.Revision, NutrientID: "energy_kcal", Amount: intakeAmount, Unit: "kcal", Reason: "Recorded in isolated fixture"}); err != nil {
		return nil, err
	}

	workspacePath, workspaceService := wspaceconnect.NewWorkspaceServiceHandler(workspacehandler.NewConnectHandler(workspaces, nil))
	recipePath, recipeService := recipeconnect.NewRecipeServiceHandler(recipehandler.NewConnectHandler(recipes, workspaces, nil))
	profilePath, profileService := profileconnect.NewProfileServiceHandler(profilehandler.NewConnectHandler(profiles, workspaces, recipes, nil))
	planningPath, planningService := planningconnect.NewPlanningServiceHandler(NewConnectHandler(workspaces, recipes, profiles, planRepo, shoppingRepo, feedbackRepo, nil, shopping.Evidence{Inventory: inventoryRepo, Costs: costRepo, Now: time.Now}))
	nutritionPath, nutritionService := nutritionconnect.NewNutritionServiceHandler(nutritionhandler.NewConnectHandler(internalnutrition.NewSQLiteTargetRepository(db, clock), intakeRepo, workspaces, nil))
	supplementPath, supplementService := supplementconnect.NewSupplementServiceHandler(supplementhandler.NewConnectHandler(supplementRepo, workspaces, nil))
	catalogPath, catalogService := catalogconnect.NewCatalogServiceHandler(cataloghandler.NewConnectHandler(catalogRepo, workspaces, nil))
	cookingPath, cookingService := cookingconnect.NewCookingServiceHandler(cookinghandler.NewConnectHandler(internalcooking.NewSQLiteRepository(db), recipes, workspaces, nil))
	routes := []e3VisualRoute{{workspacePath, workspaceService}, {recipePath, recipeService}, {profilePath, profileService}, {planningPath, planningService}, {nutritionPath, nutritionService}, {supplementPath, supplementService}, {catalogPath, catalogService}, {cookingPath, cookingService}}
	bump := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		revision, raw, err := planRepo.Get(dataCtx, owner.ID)
		if err != nil || revision == 0 {
			http.Error(w, "fixture plan unavailable", http.StatusConflict)
			return
		}
		if _, err := planRepo.Apply(dataCtx, owner.ID, revision, raw); err != nil {
			http.Error(w, "fixture plan revision bump failed", http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return newE3VisualFixtureServer(t, stop, routes, bump)
}

// newE3EmptyBootstrapVisualFixture serves the production Today UI over a fresh,
// empty RoutedDB test pool. The browser's anonymous workspace bootstrap is the
// only workspace write in this mode.
func newE3EmptyBootstrapVisualFixture(t *testing.T, stop chan<- os.Signal) (*httptest.Server, error) {
	t.Helper()
	ctx := context.Background()
	temp := t.TempDir()
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, TestDriver: database.DriverSQLite, DSN: filepath.Join(temp, "primary.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = db.Close() })
	clock := schedule.System()
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		for _, schema := range []string{workspace.Schema(), recipe.Schema(), internal.Schema(), shopping.Schema(), feedback.Schema(), internalprofile.Schema(), internalnutrition.Schema(), internalsupplement.Schema(), internalcatalog.Schema(), inventory.Schema(), cost.Schema()} {
			if _, err := pool.ExecContext(ctx, schema); err != nil {
				return err
			}
		}
		return nil
	})
	if err := db.InstallTestPool(ctx, filepath.Join(temp, "fixture.db"), "nutrition-planner-e3-visual-fixture", time.Hour); err != nil {
		return nil, err
	}
	// See the matching note in newE3VisualFixture: repository construction
	// probes the primary disposable DB before request-level test routing begins.
	if _, err := db.ExecContext(ctx, internalprofile.Schema()); err != nil {
		return nil, fmt.Errorf("initialize fixture profile schema: %w", err)
	}
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	recipes := recipe.NewService(recipe.NewSQLiteRepository(db, clock))
	profiles := internalprofile.NewService(internalprofile.NewSQLiteRepository(db, clock))
	planRepo := internal.NewSQLiteRepository(db, clock)
	feedbackRepo := feedback.NewSQLiteRepository(db, clock)
	shoppingRepo := shopping.NewSQLiteRepository(db, clock)
	inventoryRepo := inventory.NewSQLiteRepository(db)
	costRepo := cost.NewSQLiteRepository(db)
	catalogRepo := internalcatalog.NewSQLiteRepository(db, clock)
	supplementRepo := internalsupplement.NewSQLiteRepository(db, clock)
	intakeRepo := internalnutrition.NewSQLiteIntakeRepository(db, clock)

	workspacePath, workspaceService := wspaceconnect.NewWorkspaceServiceHandler(workspacehandler.NewConnectHandler(workspaces, nil))
	recipePath, recipeService := recipeconnect.NewRecipeServiceHandler(recipehandler.NewConnectHandler(recipes, workspaces, nil))
	profilePath, profileService := profileconnect.NewProfileServiceHandler(profilehandler.NewConnectHandler(profiles, workspaces, recipes, nil))
	planningPath, planningService := planningconnect.NewPlanningServiceHandler(NewConnectHandler(workspaces, recipes, profiles, planRepo, shoppingRepo, feedbackRepo, nil, shopping.Evidence{Inventory: inventoryRepo, Costs: costRepo, Now: time.Now}))
	nutritionPath, nutritionService := nutritionconnect.NewNutritionServiceHandler(nutritionhandler.NewConnectHandler(internalnutrition.NewSQLiteTargetRepository(db, clock), intakeRepo, workspaces, nil))
	supplementPath, supplementService := supplementconnect.NewSupplementServiceHandler(supplementhandler.NewConnectHandler(supplementRepo, workspaces, nil))
	catalogPath, catalogService := catalogconnect.NewCatalogServiceHandler(cataloghandler.NewConnectHandler(catalogRepo, workspaces, nil))
	routes := []e3VisualRoute{{workspacePath, workspaceService}, {recipePath, recipeService}, {profilePath, profileService}, {planningPath, planningService}, {nutritionPath, nutritionService}, {supplementPath, supplementService}, {catalogPath, catalogService}}
	return newE3VisualFixtureServer(t, stop, routes, nil)
}

type e3VisualRoute struct {
	path    string
	handler http.Handler
}

func newE3VisualFixtureServer(t *testing.T, stop chan<- os.Signal, routes []e3VisualRoute, bump http.HandlerFunc) (*httptest.Server, error) {
	t.Helper()
	api := mux.NewRouter()
	for _, route := range routes {
		api.Handle(route.path, route.handler)
		api.PathPrefix(route.path).Handler(route.handler)
	}
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.ServeHTTP(w, r.WithContext(database.WithTestMode(r.Context())))
	})
	root := http.NewServeMux()
	root.HandleFunc("/__fixture/shutdown", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		stop <- os.Interrupt
	})
	if bump != nil {
		root.HandleFunc("/__fixture/bump", bump)
	}
	root.Handle("/api/v1/", http.StripPrefix("/api/v1", apiHandler))
	for _, route := range routes {
		root.Handle(route.path, apiHandler)
	}
	_, source, _, _ := runtime.Caller(0)
	uiRoot := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", "..", "ui", "dist"))
	if info, err := os.Stat(filepath.Join(uiRoot, "index.html")); err != nil || info.IsDir() {
		return nil, fmt.Errorf("production UI build not found at %s; build the UI before launching the visual fixture", uiRoot)
	}
	files := http.FileServer(http.Dir(uiRoot))
	root.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean("/" + r.URL.Path)
		if strings.HasPrefix(clean, "/assets/") || clean == "/favicon.svg" {
			files.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(uiRoot, "index.html"))
	})
	server := httptest.NewServer(root)
	t.Cleanup(server.Close)
	return server, nil
}
