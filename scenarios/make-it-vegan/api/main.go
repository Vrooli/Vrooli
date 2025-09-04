package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/gorilla/mux"
    "github.com/rs/cors"
)

var n8nURL string

func init() {
    n8nURL = os.Getenv("N8N_BASE_URL")
    if n8nURL == "" {
        n8nURL = "http://localhost:5678"
    }
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

    // Call n8n webhook with retry logic and better error handling
    webhookURL := fmt.Sprintf("%s/webhook/make-it-vegan/check", n8nURL)
    payload, _ := json.Marshal(map[string]string{
        "ingredients": req.Ingredients,
    })

    var resp *http.Response
    var err error
    maxRetries := 3
    client := &http.Client{Timeout: 30 * time.Second}
    
    for i := 0; i < maxRetries; i++ {
        resp, err = client.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
        if err == nil && resp.StatusCode == http.StatusOK {
            break
        }
        if resp != nil {
            resp.Body.Close()
        }
        if i < maxRetries-1 {
            log.Printf("Retry %d: n8n webhook call failed, retrying...", i+1)
            time.Sleep(time.Duration(i+1) * time.Second)
        }
    }
    
    if err != nil || (resp != nil && resp.StatusCode != http.StatusOK) {
        log.Printf("Error calling n8n webhook: %v", err)
        // Provide a fallback response
        fallbackResponse := map[string]interface{}{
            "isVegan": false,
            "analysis": "Service temporarily unavailable. Please try again later.",
            "error": "Unable to process request at this time",
            "timestamp": time.Now().Format(time.RFC3339),
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(fallbackResponse)
        return
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    w.Header().Set("Content-Type", "application/json")
    w.Write(body)
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

    webhookURL := fmt.Sprintf("%s/webhook/make-it-vegan/substitute", n8nURL)
    payload, _ := json.Marshal(map[string]string{
        "ingredient": req.Ingredient,
        "context":    req.Context,
    })

    var resp *http.Response
    var err error
    maxRetries := 3
    client := &http.Client{Timeout: 30 * time.Second}
    
    for i := 0; i < maxRetries; i++ {
        resp, err = client.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
        if err == nil && resp.StatusCode == http.StatusOK {
            break
        }
        if resp != nil {
            resp.Body.Close()
        }
        if i < maxRetries-1 {
            log.Printf("Retry %d: n8n webhook call failed, retrying...", i+1)
            time.Sleep(time.Duration(i+1) * time.Second)
        }
    }
    
    if err != nil || (resp != nil && resp.StatusCode != http.StatusOK) {
        log.Printf("Error calling n8n webhook: %v", err)
        // Provide a fallback response with common alternatives
        fallbackResponse := map[string]interface{}{
            "request": map[string]string{
                "ingredient": req.Ingredient,
                "context": req.Context,
            },
            "alternatives": []map[string]interface{}{
                {
                    "name": "Service temporarily unavailable",
                    "reason": "Please try again later",
                    "adjustments": "N/A",
                    "availability": "N/A",
                    "rating": 0,
                },
            },
            "quickTip": "Service is temporarily unavailable. Please try again.",
            "timestamp": time.Now().Format(time.RFC3339),
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(fallbackResponse)
        return
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    w.Header().Set("Content-Type", "application/json")
    w.Write(body)
}

func veganizeRecipe(w http.ResponseWriter, r *http.Request) {
    var req VeganizeRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    webhookURL := fmt.Sprintf("%s/webhook/make-it-vegan/veganize", n8nURL)
    payload, _ := json.Marshal(map[string]string{
        "recipe": req.Recipe,
    })

    var resp *http.Response
    var err error
    maxRetries := 3
    client := &http.Client{Timeout: 30 * time.Second}
    
    for i := 0; i < maxRetries; i++ {
        resp, err = client.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
        if err == nil && resp.StatusCode == http.StatusOK {
            break
        }
        if resp != nil {
            resp.Body.Close()
        }
        if i < maxRetries-1 {
            log.Printf("Retry %d: n8n webhook call failed, retrying...", i+1)
            time.Sleep(time.Duration(i+1) * time.Second)
        }
    }
    
    if err != nil || (resp != nil && resp.StatusCode != http.StatusOK) {
        log.Printf("Error calling n8n webhook: %v", err)
        // Provide a fallback response
        fallbackResponse := map[string]interface{}{
            "originalRecipe": req.Recipe,
            "veganVersion": map[string]interface{}{
                "name": "Veganized Recipe",
                "substitutions": []map[string]string{},
                "instructions": []string{"Service temporarily unavailable"},
                "cookingTips": []string{"Please try again later"},
                "nutritionNotes": "Unable to process at this time",
            },
            "difficulty": "unknown",
            "estimatedTime": "N/A",
            "timestamp": time.Now().Format(time.RFC3339),
            "error": "Service temporarily unavailable",
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(fallbackResponse)
        return
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    w.Header().Set("Content-Type", "application/json")
    w.Write(body)
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
