package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

var (
	n8nURL  string
	veganDB *VeganDatabase
	cache   *CacheClient
)

func init() {
	n8nURL = os.Getenv("N8N_BASE_URL")
	if n8nURL == "" {
		n8nURL = "http://localhost:5678"
	}
	// Initialize the vegan database
	veganDB = InitVeganDatabase()
	// Initialize cache client
	cache = NewCacheClient()
}

type CheckRequest struct {
	Ingredients string `json:"ingredients"`
}

type SubstituteRequest struct {
	Ingredient string `json:"ingredient"`
	Context    string `json:"context"`
}

type VeganizeRequest struct {
	Recipe string `json:"recipe"`
}

func checkIngredients(w http.ResponseWriter, r *http.Request) {
	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check cache first
	isVegan, nonVeganItems, reasons, cached := cache.GetCachedIngredientCheck(req.Ingredients)
	fromCache := cached

	// If not cached, check with database
	if !cached {
		isVegan, nonVeganItems, reasons = veganDB.CheckIngredients(req.Ingredients)
		// Cache the result
		cache.CacheIngredientCheck(req.Ingredients, isVegan, nonVeganItems, reasons)
	}

	response := map[string]interface{}{
		"isVegan":     isVegan,
		"ingredients": req.Ingredients,
		"timestamp":   time.Now().Format(time.RFC3339),
		"cached":      fromCache,
	}

	if !isVegan {
		response["nonVeganItems"] = nonVeganItems
		response["nonVeganIngredients"] = nonVeganItems // UI compatibility
		response["reasons"] = reasons

		// Create explanations map for UI
		explanations := make(map[string]string)
		for i, item := range nonVeganItems {
			if i < len(reasons) {
				explanations[item] = reasons[i]
			}
		}
		response["explanations"] = explanations

		response["analysis"] = fmt.Sprintf("Found %d non-vegan ingredient(s): %v", len(nonVeganItems), nonVeganItems)

		// Add suggestions for alternatives
		suggestions := make(map[string]interface{})
		for _, item := range nonVeganItems {
			if alts := veganDB.GetAlternatives(item); len(alts) > 0 {
				suggestions[item] = alts[0].Name // Suggest the top alternative
			}
		}
		if len(suggestions) > 0 {
			response["suggestions"] = suggestions
		}
	} else {
		response["analysis"] = "All ingredients appear to be vegan!"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func findSubstitute(w http.ResponseWriter, r *http.Request) {
	var req SubstituteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Context == "" {
		req.Context = "general cooking"
	}

	// Use local database for finding alternatives
	alternatives := veganDB.GetAlternatives(req.Ingredient)
	quickSub := veganDB.GetQuickSubstitute(req.Ingredient)

	// Format alternatives for UI compatibility
	formattedAlternatives := make([]map[string]interface{}, len(alternatives))
	for i, alt := range alternatives {
		formattedAlternatives[i] = map[string]interface{}{
			"name":        alt.Name,
			"description": fmt.Sprintf("Rating: %.1f/5 - %s", alt.Rating, alt.Notes),
			"tips":        alt.Adjustments,
		}
	}

	response := map[string]interface{}{
		// UI-compatible fields
		"ingredient":   req.Ingredient,
		"context":      req.Context,
		"alternatives": formattedAlternatives,

		// Additional fields for backward compatibility
		"request": map[string]string{
			"ingredient": req.Ingredient,
			"context":    req.Context,
		},
		"quickSubstitute": quickSub,
		"timestamp":       time.Now().Format(time.RFC3339),
	}

	if len(alternatives) == 0 {
		response["message"] = fmt.Sprintf("No specific alternatives found for '%s', but most animal products have plant-based substitutes available in health food stores.", req.Ingredient)
		response["quickTip"] = "Try searching for 'vegan ' + the ingredient name at your local store."
	} else {
		// Filter alternatives by context if applicable
		var contextTip string
		switch req.Context {
		case "baking":
			contextTip = "For baking, binding and moisture are key considerations."
		case "cooking":
			contextTip = "For cooking, flavor and texture matching are important."
		case "spreading":
			contextTip = "For spreading, consistency at room temperature matters."
		default:
			contextTip = fmt.Sprintf("Best alternatives for %s.", req.Context)
		}
		response["quickTip"] = contextTip
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func veganizeRecipe(w http.ResponseWriter, r *http.Request) {
	var req VeganizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse recipe for ingredients and create vegan version
	recipe := strings.ToLower(req.Recipe)
	veganRecipe := req.Recipe
	substitutionsList := make([]map[string]string, 0)
	substitutionsMap := make(map[string]string)

	// Check for common non-vegan ingredients and suggest replacements
	for nonVegan := range veganDB.NonVeganIngredients {
		if strings.Contains(recipe, nonVegan) {
			if alts := veganDB.GetAlternatives(nonVegan); len(alts) > 0 {
				substitute := alts[0].Name
				substitutionsList = append(substitutionsList, map[string]string{
					"original":   nonVegan,
					"substitute": substitute,
					"notes":      alts[0].Adjustments,
				})
				substitutionsMap[nonVegan] = substitute
				// Replace in recipe text (case-insensitive)
				veganRecipe = strings.ReplaceAll(veganRecipe, nonVegan, substitute)
				veganRecipe = strings.ReplaceAll(veganRecipe, strings.Title(nonVegan), strings.Title(substitute))
			}
		}
	}

	cookingTips := []string{
		"Check liquid ratios when using plant milks",
		"Add nutritional yeast for cheesy flavor",
		"Use aquafaba for egg white substitutes in baking",
	}

	response := map[string]interface{}{
		// UI-compatible fields
		"veganRecipe":   veganRecipe,
		"substitutions": substitutionsMap,
		"cookingTips":   cookingTips,

		// Additional fields for backward compatibility
		"originalRecipe": req.Recipe,
		"veganVersion": map[string]interface{}{
			"name":          "Veganized Recipe",
			"substitutions": substitutionsList,
			"fullText":      veganRecipe,
		},
		"nutritionHighlights": "This vegan version maintains protein through plant sources and provides fiber, vitamins, and minerals.",
		"timestamp":           time.Now().Format(time.RFC3339),
	}

	if len(substitutionsList) == 0 {
		response["message"] = "This recipe appears to be vegan already!"
	} else {
		response["message"] = fmt.Sprintf("Made %d substitutions to veganize this recipe", len(substitutionsList))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getCommonProducts(w http.ResponseWriter, r *http.Request) {
	// Return some common non-vegan ingredients for reference
	products := map[string][]string{
		"dairy": {"milk", "cheese", "butter", "yogurt", "cream", "whey", "casein", "lactose"},
		"eggs":  {"eggs", "egg whites", "egg yolks", "albumin", "mayonnaise"},
		"meat":  {"chicken", "beef", "pork", "fish", "lamb", "turkey", "bacon", "gelatin"},
		"other": {"honey", "beeswax", "shellac", "carmine", "vitamin D3", "omega-3 (fish-based)"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func getNutrition(w http.ResponseWriter, r *http.Request) {
	insights := veganDB.GetNutritionalInsights()

	response := map[string]interface{}{
		"nutritionalInfo": insights,
		"timestamp":       time.Now().Format(time.RFC3339),
		"message":         "Key nutritional information for a healthy vegan diet",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start make-it-vegan

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}
	router := mux.NewRouter()

	// API routes
	router.HandleFunc("/api/check", checkIngredients).Methods("POST")
	router.HandleFunc("/api/substitute", findSubstitute).Methods("POST")
	router.HandleFunc("/api/veganize", veganizeRecipe).Methods("POST")
	router.HandleFunc("/api/products", getCommonProducts).Methods("GET")
	router.HandleFunc("/api/nutrition", getNutrition).Methods("GET")
	router.HandleFunc("/health", healthCheck).Methods("GET")

	// Enable CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	port := getEnv("API_PORT", getEnv("PORT", ""))

	log.Printf("Make It Vegan API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
