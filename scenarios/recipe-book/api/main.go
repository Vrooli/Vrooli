package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Recipe struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	Ingredients  []Ingredient  `json:"ingredients"`
	Instructions []string      `json:"instructions"`
	PrepTime     int           `json:"prep_time"`
	CookTime     int           `json:"cook_time"`
	Servings     int           `json:"servings"`
	Tags         []string      `json:"tags"`
	Cuisine      string        `json:"cuisine"`
	DietaryInfo  []string      `json:"dietary_info"`
	Nutrition    NutritionInfo `json:"nutrition"`
	PhotoURL     string        `json:"photo_url"`
	CreatedBy    string        `json:"created_by"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	Visibility   string        `json:"visibility"`
	SharedWith   []string      `json:"shared_with"`
	Source       string        `json:"source"`
	ParentID     string        `json:"parent_recipe_id,omitempty"`
}

type Ingredient struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
	Notes  string  `json:"notes,omitempty"`
}

type NutritionInfo struct {
	Calories int `json:"calories"`
	Protein  int `json:"protein"`
	Carbs    int `json:"carbs"`
	Fat      int `json:"fat"`
	Fiber    int `json:"fiber"`
	Sugar    int `json:"sugar"`
	Sodium   int `json:"sodium"`
}

type RecipeRating struct {
	ID         string    `json:"id"`
	RecipeID   string    `json:"recipe_id"`
	UserID     string    `json:"user_id"`
	Rating     int       `json:"rating"`
	Notes      string    `json:"notes"`
	CookedDate time.Time `json:"cooked_date"`
	Anonymous  bool      `json:"anonymous"`
}

type SearchRequest struct {
	Query   string `json:"query"`
	UserID  string `json:"user_id"`
	Limit   int    `json:"limit,omitempty"`
	Filters struct {
		Dietary     []string `json:"dietary,omitempty"`
		MaxTime     int      `json:"max_time,omitempty"`
		Ingredients []string `json:"ingredients,omitempty"`
	} `json:"filters,omitempty"`
}

type SearchResult struct {
	Recipe     Recipe   `json:"recipe"`
	Relevance  float64  `json:"relevance"`
	Highlights []string `json:"highlights"`
}

type GenerateRequest struct {
	Prompt               string   `json:"prompt"`
	UserID               string   `json:"user_id"`
	DietaryRestrictions  []string `json:"dietary_restrictions,omitempty"`
	AvailableIngredients []string `json:"available_ingredients,omitempty"`
	Style                string   `json:"style,omitempty"`
}

type ModifyRequest struct {
	ModificationType string `json:"modification_type"`
	UserID           string `json:"user_id"`
}

var db *sql.DB

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start recipe-book

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "3250"
	}

	initDB()
	defer db.Close()

	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// Recipe CRUD
	router.HandleFunc("/api/v1/recipes", listRecipesHandler).Methods("GET")
	router.HandleFunc("/api/v1/recipes", createRecipeHandler).Methods("POST")
	router.HandleFunc("/api/v1/recipes/{id}", getRecipeHandler).Methods("GET")
	router.HandleFunc("/api/v1/recipes/{id}", updateRecipeHandler).Methods("PUT")
	router.HandleFunc("/api/v1/recipes/{id}", deleteRecipeHandler).Methods("DELETE")

	// Recipe operations
	router.HandleFunc("/api/v1/recipes/search", searchRecipesHandler).Methods("POST")
	router.HandleFunc("/api/v1/recipes/generate", generateRecipeHandler).Methods("POST")
	router.HandleFunc("/api/v1/recipes/{id}/modify", modifyRecipeHandler).Methods("POST")
	router.HandleFunc("/api/v1/recipes/{id}/cook", markCookedHandler).Methods("POST")
	router.HandleFunc("/api/v1/recipes/{id}/rate", rateRecipeHandler).Methods("POST")
	router.HandleFunc("/api/v1/recipes/{id}/share", shareRecipeHandler).Methods("POST")

	// Shopping list
	router.HandleFunc("/api/v1/shopping-list", generateShoppingListHandler).Methods("POST")

	// User preferences
	router.HandleFunc("/api/v1/users/{id}/preferences", getUserPreferencesHandler).Methods("GET")
	router.HandleFunc("/api/v1/users/{id}/preferences", updateUserPreferencesHandler).Methods("PUT")

	// Serve UI
	staticDir := "./ui/dist"
	if _, err := os.Stat(staticDir); err != nil {
		log.Printf("[Recipe Book] UI build assets missing at %s: %v", staticDir, err)
	} else {
		router.PathPrefix("/").Handler(http.FileServer(http.Dir(staticDir)))
	}

	log.Printf("🍳 Recipe Book API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func initDB() {
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}

	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}

	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "recipe_book"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("Connected to PostgreSQL database")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "recipe-book",
	}

	// Check database connection
	if db != nil {
		if err := db.Ping(); err != nil {
			status["status"] = "unhealthy"
			status["database"] = "disconnected"
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			status["database"] = "connected"
		}
	} else {
		status["database"] = "not_configured"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func listRecipesHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	visibility := r.URL.Query().Get("visibility")
	_ = r.URL.Query()["tags"] // tags: future filter implementation
	cuisine := r.URL.Query().Get("cuisine")
	_ = r.URL.Query()["dietary"] // dietary: future filter implementation
	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")

	if limit == "" {
		limit = "20"
	}
	if offset == "" {
		offset = "0"
	}

	if db == nil {
		http.Error(w, "Database not available", http.StatusInternalServerError)
		return
	}

	// Build query based on filters
	query := `SELECT id, title, description, cuisine, prep_time, cook_time, servings,
	          visibility, created_by, created_at, updated_at
	          FROM recipes WHERE 1=1`

	var args []interface{}
	argCount := 1

	if userID != "" {
		query += fmt.Sprintf(" AND (created_by = $%d OR visibility = 'public' OR $%d = ANY(shared_with))", argCount, argCount)
		args = append(args, userID)
		argCount++
	}

	if visibility != "" {
		query += fmt.Sprintf(" AND visibility = $%d", argCount)
		args = append(args, visibility)
		argCount++
	}

	if cuisine != "" {
		query += fmt.Sprintf(" AND cuisine = $%d", argCount)
		args = append(args, cuisine)
		argCount++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to query recipes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var recipes []Recipe
	for rows.Next() {
		var r Recipe
		err := rows.Scan(&r.ID, &r.Title, &r.Description, &r.Cuisine,
			&r.PrepTime, &r.CookTime, &r.Servings, &r.Visibility,
			&r.CreatedBy, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			continue
		}
		recipes = append(recipes, r)
	}

	response := map[string]interface{}{
		"recipes":  recipes,
		"total":    len(recipes),
		"has_more": len(recipes) == 20,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func createRecipeHandler(w http.ResponseWriter, r *http.Request) {
	var recipe Recipe
	if err := json.NewDecoder(r.Body).Decode(&recipe); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	recipe.ID = uuid.New().String()
	recipe.CreatedAt = time.Now()
	recipe.UpdatedAt = time.Now()

	if recipe.Visibility == "" {
		recipe.Visibility = "private"
	}

	if recipe.Source == "" {
		recipe.Source = "original"
	}

	if db == nil {
		http.Error(w, "Database not available", http.StatusInternalServerError)
		return
	}

	// Insert into database
	ingredientsJSON, _ := json.Marshal(recipe.Ingredients)
	instructionsJSON, _ := json.Marshal(recipe.Instructions)
	tagsJSON, _ := json.Marshal(recipe.Tags)
	dietaryJSON, _ := json.Marshal(recipe.DietaryInfo)
	nutritionJSON, _ := json.Marshal(recipe.Nutrition)
	sharedWithJSON, _ := json.Marshal(recipe.SharedWith)

	_, err := db.Exec(`
		INSERT INTO recipes (id, title, description, ingredients, instructions, 
		                    prep_time, cook_time, servings, tags, cuisine, 
		                    dietary_info, nutrition, photo_url, created_by, 
		                    created_at, updated_at, visibility, shared_with, 
		                    source, parent_recipe_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)`,
		recipe.ID, recipe.Title, recipe.Description, ingredientsJSON, instructionsJSON,
		recipe.PrepTime, recipe.CookTime, recipe.Servings, tagsJSON, recipe.Cuisine,
		dietaryJSON, nutritionJSON, recipe.PhotoURL, recipe.CreatedBy,
		recipe.CreatedAt, recipe.UpdatedAt, recipe.Visibility, sharedWithJSON,
		recipe.Source, recipe.ParentID)

	if err != nil {
		log.Printf("Failed to insert recipe: %v", err)
		http.Error(w, "Failed to create recipe", http.StatusInternalServerError)
		return
	}

	// Generate embedding for semantic search
	go generateRecipeEmbedding(recipe)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(recipe)
}

func getRecipeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recipeID := vars["id"]
	userID := r.URL.Query().Get("user_id")

	if db == nil {
		http.Error(w, "Database not available", http.StatusInternalServerError)
		return
	}

	var recipe Recipe
	var ingredientsJSON, instructionsJSON, tagsJSON, dietaryJSON, nutritionJSON, sharedWithJSON []byte

	err := db.QueryRow(`
		SELECT id, title, description, ingredients, instructions, 
		       prep_time, cook_time, servings, tags, cuisine, 
		       dietary_info, nutrition, photo_url, created_by, 
		       created_at, updated_at, visibility, shared_with, 
		       source, parent_recipe_id
		FROM recipes WHERE id = $1`, recipeID).Scan(
		&recipe.ID, &recipe.Title, &recipe.Description, &ingredientsJSON, &instructionsJSON,
		&recipe.PrepTime, &recipe.CookTime, &recipe.Servings, &tagsJSON, &recipe.Cuisine,
		&dietaryJSON, &nutritionJSON, &recipe.PhotoURL, &recipe.CreatedBy,
		&recipe.CreatedAt, &recipe.UpdatedAt, &recipe.Visibility, &sharedWithJSON,
		&recipe.Source, &recipe.ParentID)

	if err == sql.ErrNoRows {
		http.Error(w, "Recipe not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to fetch recipe", http.StatusInternalServerError)
		return
	}

	// Check permissions
	if recipe.Visibility == "private" && recipe.CreatedBy != userID {
		hasAccess := false
		for _, sharedUser := range recipe.SharedWith {
			if sharedUser == userID {
				hasAccess = true
				break
			}
		}
		if !hasAccess {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
	}

	// Unmarshal JSON fields
	json.Unmarshal(ingredientsJSON, &recipe.Ingredients)
	json.Unmarshal(instructionsJSON, &recipe.Instructions)
	json.Unmarshal(tagsJSON, &recipe.Tags)
	json.Unmarshal(dietaryJSON, &recipe.DietaryInfo)
	json.Unmarshal(nutritionJSON, &recipe.Nutrition)
	json.Unmarshal(sharedWithJSON, &recipe.SharedWith)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipe)
}

func updateRecipeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recipeID := vars["id"]

	var recipe Recipe
	if err := json.NewDecoder(r.Body).Decode(&recipe); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	recipe.ID = recipeID
	recipe.UpdatedAt = time.Now()

	if db == nil {
		http.Error(w, "Database not available", http.StatusInternalServerError)
		return
	}

	// Update in database
	ingredientsJSON, _ := json.Marshal(recipe.Ingredients)
	instructionsJSON, _ := json.Marshal(recipe.Instructions)
	tagsJSON, _ := json.Marshal(recipe.Tags)
	dietaryJSON, _ := json.Marshal(recipe.DietaryInfo)
	nutritionJSON, _ := json.Marshal(recipe.Nutrition)
	sharedWithJSON, _ := json.Marshal(recipe.SharedWith)

	_, err := db.Exec(`
		UPDATE recipes SET title = $2, description = $3, ingredients = $4, 
		                  instructions = $5, prep_time = $6, cook_time = $7, 
		                  servings = $8, tags = $9, cuisine = $10, 
		                  dietary_info = $11, nutrition = $12, photo_url = $13, 
		                  updated_at = $14, visibility = $15, shared_with = $16
		WHERE id = $1`,
		recipeID, recipe.Title, recipe.Description, ingredientsJSON, instructionsJSON,
		recipe.PrepTime, recipe.CookTime, recipe.Servings, tagsJSON, recipe.Cuisine,
		dietaryJSON, nutritionJSON, recipe.PhotoURL, recipe.UpdatedAt,
		recipe.Visibility, sharedWithJSON)

	if err != nil {
		http.Error(w, "Failed to update recipe", http.StatusInternalServerError)
		return
	}

	// Update embedding
	go generateRecipeEmbedding(recipe)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipe)
}

func deleteRecipeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recipeID := vars["id"]
	userID := r.URL.Query().Get("user_id")

	if db == nil {
		http.Error(w, "Database not available", http.StatusInternalServerError)
		return
	}

	// Check ownership
	var createdBy string
	err := db.QueryRow("SELECT created_by FROM recipes WHERE id = $1", recipeID).Scan(&createdBy)
	if err == sql.ErrNoRows {
		http.Error(w, "Recipe not found", http.StatusNotFound)
		return
	}

	if createdBy != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Delete recipe
	_, err = db.Exec("DELETE FROM recipes WHERE id = $1", recipeID)
	if err != nil {
		http.Error(w, "Failed to delete recipe", http.StatusInternalServerError)
		return
	}

	// Delete from Qdrant
	go deleteRecipeEmbedding(recipeID)

	w.WriteHeader(http.StatusNoContent)
}

func searchRecipesHandler(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Limit == 0 {
		req.Limit = 10
	}

	// Call Qdrant for semantic search
	results := performSemanticSearch(req)

	response := map[string]interface{}{
		"results":              results,
		"query_interpretation": interpretQuery(req.Query),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func generateRecipeHandler(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Call n8n workflow for AI generation
	recipe := generateRecipeWithAI(req)

	response := map[string]interface{}{
		"recipe":       recipe,
		"confidence":   0.85,
		"alternatives": []string{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func modifyRecipeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recipeID := vars["id"]

	var req ModifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Fetch original recipe
	originalRecipe := fetchRecipe(recipeID)

	// Call n8n workflow for modification
	modifiedRecipe := modifyRecipeWithAI(originalRecipe, req)

	response := map[string]interface{}{
		"modified_recipe": modifiedRecipe,
		"changes_made":    describeChanges(originalRecipe, modifiedRecipe),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func markCookedHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recipeID := vars["id"]

	var rating RecipeRating
	if err := json.NewDecoder(r.Body).Decode(&rating); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	rating.ID = uuid.New().String()
	rating.RecipeID = recipeID
	rating.CookedDate = time.Now()

	if db == nil {
		http.Error(w, "Database not available", http.StatusInternalServerError)
		return
	}

	// Insert rating
	_, err := db.Exec(`
		INSERT INTO recipe_ratings (id, recipe_id, user_id, rating, notes, cooked_date, anonymous)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		rating.ID, rating.RecipeID, rating.UserID, rating.Rating,
		rating.Notes, rating.CookedDate, rating.Anonymous)

	if err != nil {
		http.Error(w, "Failed to record cooking", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "recorded"})
}

func rateRecipeHandler(w http.ResponseWriter, r *http.Request) {
	markCookedHandler(w, r) // Same functionality
}

func shareRecipeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recipeID := vars["id"]

	var shareReq struct {
		UserIDs []string `json:"user_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&shareReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if db == nil {
		http.Error(w, "Database not available", http.StatusInternalServerError)
		return
	}

	sharedWithJSON, _ := json.Marshal(shareReq.UserIDs)

	_, err := db.Exec("UPDATE recipes SET shared_with = $2 WHERE id = $1",
		recipeID, sharedWithJSON)

	if err != nil {
		http.Error(w, "Failed to share recipe", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "shared"})
}

func generateShoppingListHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RecipeIDs []string `json:"recipe_ids"`
		UserID    string   `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Aggregate ingredients from recipes
	shoppingList := aggregateIngredients(req.RecipeIDs)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"shopping_list":         shoppingList,
		"organized_by_category": organizeByCategory(shoppingList),
	})
}

func getUserPreferencesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	preferences := fetchUserPreferences(userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(preferences)
}

func updateUserPreferencesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	var preferences map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&preferences); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updateUserPreferences(userID, preferences)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(preferences)
}

// Helper functions (stubs for now)
func generateRecipeEmbedding(recipe Recipe) {
	// TODO: Call embedding-generator workflow
	log.Printf("Generating embedding for recipe %s", recipe.ID)
}

func deleteRecipeEmbedding(recipeID string) {
	// TODO: Delete from Qdrant
	log.Printf("Deleting embedding for recipe %s", recipeID)
}

func performSemanticSearch(req SearchRequest) []SearchResult {
	// TODO: Query Qdrant
	return []SearchResult{}
}

func interpretQuery(query string) string {
	return fmt.Sprintf("Searching for: %s", query)
}

func generateRecipeWithAI(req GenerateRequest) Recipe {
	// TODO: Call n8n workflow
	return Recipe{
		ID:          uuid.New().String(),
		Title:       "AI Generated Recipe",
		Description: req.Prompt,
		Source:      "ai_generated",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func fetchRecipe(recipeID string) Recipe {
	// TODO: Fetch from database
	return Recipe{ID: recipeID}
}

func modifyRecipeWithAI(original Recipe, req ModifyRequest) Recipe {
	// TODO: Call n8n workflow
	modified := original
	modified.ID = uuid.New().String()
	modified.ParentID = original.ID
	modified.Source = "modified"
	return modified
}

func describeChanges(original, modified Recipe) []string {
	return []string{"Made " + modified.Source}
}

func aggregateIngredients(recipeIDs []string) []Ingredient {
	// TODO: Aggregate from database
	return []Ingredient{}
}

func organizeByCategory(ingredients []Ingredient) map[string][]Ingredient {
	// TODO: Organize ingredients
	return map[string][]Ingredient{}
}

func fetchUserPreferences(userID string) map[string]interface{} {
	// TODO: Fetch from database or contact-book
	return map[string]interface{}{
		"dietary_restrictions": []string{},
		"favorite_cuisines":    []string{},
	}
}

func updateUserPreferences(userID string, preferences map[string]interface{}) {
	// TODO: Update in database
	log.Printf("Updating preferences for user %s", userID)
}
// Test change for rebuild detection
// Test change
