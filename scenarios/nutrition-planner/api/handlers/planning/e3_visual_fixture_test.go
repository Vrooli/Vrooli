package planning

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/catalog/catalog_v1connect"
	nutritionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/nutrition/nutrition_v1connect"
	planningconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/planning/planning_v1connect"
	recipeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/recipe/recipe_v1connect"
	supplementconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/supplement/supplement_v1connect"
	wspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/workspace/workspace_v1connect"
	_ "modernc.org/sqlite"
	cataloghandler "nutrition-planner/handlers/catalog"
	nutritionhandler "nutrition-planner/handlers/nutrition"
	recipehandler "nutrition-planner/handlers/recipe"
	supplementhandler "nutrition-planner/handlers/supplement"
	workspacehandler "nutrition-planner/handlers/workspace"
	internalcatalog "nutrition-planner/internal/catalog"
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

const e3VisualSubject = "e3-isolated-visual-fixture"

// TestRunE3VisualFixture runs only on explicit request. It serves the production
// UI build while every API request uses Nooch's actual Authenticator verifier
// and an isolated RoutedDB test pool. The temporary signer is fixture-only.
func TestRunE3VisualFixture(t *testing.T) {
	if os.Getenv("E3_VISUAL_FIXTURE") != "1" {
		t.Skip("set E3_VISUAL_FIXTURE=1 to serve the isolated visual fixture")
	}
	stop := make(chan os.Signal, 1)
	server, err := newE3VisualFixture(t, stop)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	fmt.Printf("E3_VISUAL_FIXTURE_URL=%s/__fixture/login?next=/today\n", server.URL)
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
		for _, schema := range []string{workspace.Schema(), recipe.Schema(), internal.Schema(), shopping.Schema(), feedback.Schema(), internalnutrition.Schema(), internalsupplement.Schema(), internalcatalog.Schema(), inventory.Schema(), cost.Schema()} {
			if _, err := pool.ExecContext(ctx, schema); err != nil {
				return err
			}
		}
		return nil
	})
	if err := db.InstallTestPool(ctx, filepath.Join(temp, "fixture.db"), "nutrition-planner-e3-visual-fixture", time.Hour); err != nil {
		return nil, err
	}
	dataCtx := database.WithTestMode(ctx)
	workspaces := workspace.NewService(workspace.NewSQLiteRepository(db, clock))
	owner, err := workspaces.Create(dataCtx, workspace.CreateInput{Name: "Nooch visual review", OwnerSubject: e3VisualSubject})
	if err != nil {
		return nil, err
	}
	recipes := recipe.NewService(recipe.NewSQLiteRepository(db, clock))
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

	api := mux.NewRouter()
	workspacePath, workspaceService := wspaceconnect.NewWorkspaceServiceHandler(workspacehandler.NewConnectHandler(workspaces, nil))
	recipePath, recipeService := recipeconnect.NewRecipeServiceHandler(recipehandler.NewConnectHandler(recipes, workspaces, nil))
	planningPath, planningService := planningconnect.NewPlanningServiceHandler(NewConnectHandler(workspaces, recipes, nil, planRepo, shoppingRepo, feedbackRepo, nil, shopping.Evidence{Inventory: inventoryRepo, Costs: costRepo, Now: time.Now}))
	nutritionPath, nutritionService := nutritionconnect.NewNutritionServiceHandler(nutritionhandler.NewConnectHandler(internalnutrition.NewSQLiteTargetRepository(db, clock), intakeRepo, workspaces, nil))
	supplementPath, supplementService := supplementconnect.NewSupplementServiceHandler(supplementhandler.NewConnectHandler(supplementRepo, workspaces, nil))
	catalogPath, catalogService := catalogconnect.NewCatalogServiceHandler(cataloghandler.NewConnectHandler(catalogRepo, workspaces, nil))
	for _, route := range []struct {
		path    string
		handler http.Handler
	}{{workspacePath, workspaceService}, {recipePath, recipeService}, {planningPath, planningService}, {nutritionPath, nutritionService}, {supplementPath, supplementService}, {catalogPath, catalogService}} {
		api.Handle(route.path, route.handler)
		api.PathPrefix(route.path).Handler(route.handler)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	baseURL := "http://" + listener.Addr().String()
	issuer := "scenario-authenticator"
	audience := "scenario-authenticator:nutrition-planner"
	token := signNoochFixtureJWT(t, key, issuer, audience, e3VisualSubject, time.Now().Add(12*time.Hour))
	provider := authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{Issuer: issuer, Audience: audience, JWKSURL: baseURL + "/__fixture/jwks", CookieName: "vrooli_access_token"})
	apiWithAuth := authn.Middleware(authn.Config{Providers: []authn.Provider{provider}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.ServeHTTP(w, r.WithContext(database.WithTestMode(r.Context())))
	}))
	root := http.NewServeMux()
	root.HandleFunc("/__fixture/jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"keys":[{"kid":"nooch-fixture","kty":"RSA","alg":"RS256","n":%q,"e":%q}]}`,
			base64.RawURLEncoding.EncodeToString(key.N.Bytes()), base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}))
	})
	root.HandleFunc("/__fixture/login", func(w http.ResponseWriter, r *http.Request) {
		next := r.URL.Query().Get("next")
		if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
			next = "/today"
		}
		http.SetCookie(w, &http.Cookie{Name: "vrooli_access_token", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Expires: time.Now().Add(12 * time.Hour)})
		http.Redirect(w, r, next, http.StatusSeeOther)
	})
	root.HandleFunc("/__fixture/shutdown", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		stop <- os.Interrupt
	})
	root.HandleFunc("/__fixture/bump", func(w http.ResponseWriter, r *http.Request) {
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
	root.Handle("/api/v1/", http.StripPrefix("/api/v1", apiWithAuth))
	for _, route := range []struct {
		path string
	}{
		{workspacePath}, {recipePath}, {planningPath}, {nutritionPath}, {supplementPath}, {catalogPath},
	} {
		root.Handle(route.path, apiWithAuth)
	}
	_, source, _, _ := runtime.Caller(0)
	uiRoot := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", "..", "ui", "dist"))
	if info, err := os.Stat(filepath.Join(uiRoot, "index.html")); err != nil || info.IsDir() {
		_ = listener.Close()
		return nil, fmt.Errorf("production UI build not found at %s; build the UI before launching the fixture", uiRoot)
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
	server := httptest.NewUnstartedServer(root)
	server.Listener = listener
	server.Config.ReadHeaderTimeout = 5 * time.Second
	server.Start()
	t.Cleanup(server.Close)
	return server, nil
}
