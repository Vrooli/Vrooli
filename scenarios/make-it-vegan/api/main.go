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
)

func init() {
    n8nURL = os.Getenv("N8N_BASE_URL")
    if n8nURL == "" {
        n8nURL = "http://localhost:5678"
    }
    // Initialize the vegan database
    veganDB = InitVeganDatabase()
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

    // Use local database for ingredient checking
    isVegan, nonVeganItems, reasons := veganDB.CheckIngredients(req.Ingredients)
    
    response := map[string]interface{}{
        "isVegan":     isVegan,
        "ingredients": req.Ingredients,
        "timestamp":   time.Now().Format(time.RFC3339),
    }
    
    if !isVegan {
        response["nonVeganItems"] = nonVeganItems
        response["reasons"] = reasons
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
    
    response := map[string]interface{}{
        "request": map[string]string{
            "ingredient": req.Ingredient,
            "context":    req.Context,
        },
        "alternatives": alternatives,
        "quickSubstitute": quickSub,
        "timestamp": time.Now().Format(time.RFC3339),
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
    substitutions := make([]map[string]string, 0)
    
    // Check for common non-vegan ingredients and suggest replacements
    for nonVegan := range veganDB.NonVeganIngredients {
        if strings.Contains(recipe, nonVegan) {
            if alts := veganDB.GetAlternatives(nonVegan); len(alts) > 0 {
                substitutions = append(substitutions, map[string]string{
                    "original": nonVegan,
                    "substitute": alts[0].Name,
                    "notes": alts[0].Adjustments,
                })
                // Replace in recipe text
                recipe = strings.ReplaceAll(recipe, nonVegan, alts[0].Name)
            }
        }
    }
    
    veganVersion := map[string]interface{}{
        "name": "Veganized " + strings.Split(req.Recipe, "\n")[0],
        "substitutions": substitutions,
        "fullText": recipe,
        "cookingTips": []string{
            "Check liquid ratios when using plant milks",
            "Add nutritional yeast for cheesy flavor",
            "Use aquafaba for egg white substitutes in baking",
        },
    }
    
    if len(substitutions) == 0 {
        veganVersion["message"] = "This recipe appears to be vegan already!"
    } else {
        veganVersion["message"] = fmt.Sprintf("Made %d substitutions to veganize this recipe", len(substitutions))
    }
    
    response := map[string]interface{}{
        "originalRecipe": req.Recipe,
        "veganVersion": veganVersion,
        "difficulty": "easy",
        "estimatedTime": "similar to original",
        "nutritionNotes": "Ensure adequate B12, iron, and protein from other sources",
        "timestamp": time.Now().Format(time.RFC3339),
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func getCommonProducts(w http.ResponseWriter, r *http.Request) {
    // Return some common non-vegan ingredients for reference
    products := map[string][]string{
        "dairy": {"milk", "cheese", "butter", "yogurt", "cream", "whey", "casein", "lactose"},
        "eggs": {"eggs", "egg whites", "egg yolks", "albumin", "mayonnaise"},
        "meat": {"chicken", "beef", "pork", "fish", "lamb", "turkey", "bacon", "gelatin"},
        "other": {"honey", "beeswax", "shellac", "carmine", "vitamin D3", "omega-3 (fish-based)"},
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(products)
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
