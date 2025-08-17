package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "strings"

    "github.com/gorilla/mux"
    "github.com/rs/cors"
)

var n8nURL string

func init() {
    n8nURL = os.Getenv("N8N_URL")
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

    // Call n8n webhook
    webhookURL := fmt.Sprintf("%s/webhook/make-it-vegan/check", n8nURL)
    payload, _ := json.Marshal(map[string]string{
        "ingredients": req.Ingredients,
    })

    resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
    if err != nil {
        http.Error(w, "Failed to check ingredients", http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
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

    resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
    if err != nil {
        http.Error(w, "Failed to find alternatives", http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
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

    resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
    if err != nil {
        http.Error(w, "Failed to veganize recipe", http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
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
    
    port := os.Getenv("PORT")
    if port == "" {
        port = os.Getenv("SERVICE_PORT")
        if port == "" {
            port = "8080"
        }
    }

    log.Printf("Make It Vegan API starting on port %s", port)
    if err := http.ListenAndServe(":"+port, handler); err != nil {
        log.Fatal(err)
    }
}