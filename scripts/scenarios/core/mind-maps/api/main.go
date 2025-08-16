package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    
    "github.com/gorilla/mux"
)

// MindMap represents a mind map structure
type MindMap struct {
    ID          string                 `json:"id"`
    Title       string                 `json:"title"`
    Description string                 `json:"description"`
    CampaignID  string                 `json:"campaign_id,omitempty"`
    OwnerID     string                 `json:"owner_id"`
    Metadata    map[string]interface{} `json:"metadata"`
    CreatedAt   string                 `json:"created_at"`
    UpdatedAt   string                 `json:"updated_at"`
}

// Node represents a mind map node
type Node struct {
    ID        string                 `json:"id"`
    MindMapID string                 `json:"mind_map_id"`
    Content   string                 `json:"content"`
    Type      string                 `json:"type"`
    PositionX float64                `json:"position_x"`
    PositionY float64                `json:"position_y"`
    ParentID  *string                `json:"parent_id,omitempty"`
    Metadata  map[string]interface{} `json:"metadata"`
}

// Response represents a standard API response
type Response struct {
    Status  string      `json:"status"`
    Message string      `json:"message,omitempty"`
    Data    interface{} `json:"data,omitempty"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    response := Response{
        Status:  "success",
        Message: "Mind Maps API is running",
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func getMindMapsHandler(w http.ResponseWriter, r *http.Request) {
    // Call n8n webhook to get mind maps
    response := Response{
        Status: "success",
        Data: []MindMap{
            {
                ID:          "demo-map-1",
                Title:       "Welcome to Mind Maps",
                Description: "Your first mind map",
                OwnerID:     "default-user",
                Metadata:    map[string]interface{}{},
            },
        },
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func createMindMapHandler(w http.ResponseWriter, r *http.Request) {
    var mindMap MindMap
    if err := json.NewDecoder(r.Body).Decode(&mindMap); err \!= nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Forward to n8n webhook
    response := Response{
        Status:  "success",
        Message: "Mind map created",
        Data:    mindMap,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func getMindMapHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    
    response := Response{
        Status: "success",
        Data: MindMap{
            ID:          id,
            Title:       "Sample Mind Map",
            Description: "Retrieved mind map",
            OwnerID:     "default-user",
        },
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func updateMindMapHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    
    var updates map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&updates); err \!= nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    response := Response{
        Status:  "success",
        Message: fmt.Sprintf("Mind map %s updated", id),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func deleteMindMapHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    
    response := Response{
        Status:  "success",
        Message: fmt.Sprintf("Mind map %s deleted", id),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func addNodeHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    mindMapID := vars["id"]
    
    var node Node
    if err := json.NewDecoder(r.Body).Decode(&node); err \!= nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    node.MindMapID = mindMapID
    
    response := Response{
        Status:  "success",
        Message: "Node added",
        Data:    node,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
    var searchRequest map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&searchRequest); err \!= nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    response := Response{
        Status: "success",
        Data: map[string]interface{}{
            "results": []interface{}{},
            "query":   searchRequest["query"],
        },
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func exportHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]
    format := r.URL.Query().Get("format")
    if format == "" {
        format = "json"
    }
    
    response := Response{
        Status: "success",
        Data: map[string]interface{}{
            "mind_map_id": id,
            "format":      format,
            "url":         fmt.Sprintf("/exports/%s.%s", id, format),
        },
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func enableCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

func main() {
    r := mux.NewRouter()
    
    // Health check
    r.HandleFunc("/health", healthHandler).Methods("GET")
    
    // Mind map endpoints
    r.HandleFunc("/api/mindmaps", getMindMapsHandler).Methods("GET")
    r.HandleFunc("/api/mindmaps", createMindMapHandler).Methods("POST")
    r.HandleFunc("/api/mindmaps/{id}", getMindMapHandler).Methods("GET")
    r.HandleFunc("/api/mindmaps/{id}", updateMindMapHandler).Methods("PUT")
    r.HandleFunc("/api/mindmaps/{id}", deleteMindMapHandler).Methods("DELETE")
    
    // Node endpoints
    r.HandleFunc("/api/mindmaps/{id}/nodes", addNodeHandler).Methods("POST")
    
    // Search endpoint
    r.HandleFunc("/api/mindmaps/search", searchHandler).Methods("POST")
    
    // Export endpoint
    r.HandleFunc("/api/mindmaps/{id}/export", exportHandler).Methods("GET")
    
    // Apply CORS middleware
    handler := enableCORS(r)
    
    port := os.Getenv("PORT")
    if port == "" {
        port = "8093"
    }
    
    log.Printf("Mind Maps API starting on port %s", port)
    if err := http.ListenAndServe(":"+port, handler); err \!= nil {
        log.Fatal(err)
    }
}
