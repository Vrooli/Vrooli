package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/gorilla/mux"
    "github.com/rs/cors"
)

type Wheel struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Options     []Option  `json:"options"`
    Theme       string    `json:"theme"`
    CreatedAt   time.Time `json:"created_at"`
    TimesUsed   int       `json:"times_used"`
}

type Option struct {
    Label  string  `json:"label"`
    Color  string  `json:"color"`
    Weight float64 `json:"weight"`
}

type SpinResult struct {
    Result    string    `json:"result"`
    WheelID   string    `json:"wheel_id"`
    SessionID string    `json:"session_id"`
    Timestamp time.Time `json:"timestamp"`
}

type HealthResponse struct {
    Status  string `json:"status"`
    Service string `json:"service"`
    Version string `json:"version"`
}

var wheels []Wheel
var history []SpinResult

func init() {
    // Initialize with some default wheels
    wheels = []Wheel{
        {
            ID:          "dinner-decider",
            Name:        "Dinner Decider",
            Description: "Can't decide what to eat?",
            Options: []Option{
                {Label: "Pizza 🍕", Color: "#FF6B6B", Weight: 1},
                {Label: "Sushi 🍱", Color: "#4ECDC4", Weight: 1},
                {Label: "Tacos 🌮", Color: "#FFD93D", Weight: 1},
                {Label: "Burger 🍔", Color: "#6C5CE7", Weight: 1},
            },
            Theme:     "food",
            CreatedAt: time.Now(),
            TimesUsed: 0,
        },
    }
}

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "9851"
    }

    router := mux.NewRouter()

    // Health check
    router.HandleFunc("/health", healthHandler).Methods("GET")

    // API endpoints
    router.HandleFunc("/api/wheels", getWheelsHandler).Methods("GET")
    router.HandleFunc("/api/wheels", createWheelHandler).Methods("POST")
    router.HandleFunc("/api/wheels/{id}", getWheelHandler).Methods("GET")
    router.HandleFunc("/api/wheels/{id}", deleteWheelHandler).Methods("DELETE")
    router.HandleFunc("/api/history", getHistoryHandler).Methods("GET")
    router.HandleFunc("/api/spin", spinHandler).Methods("POST")

    // Enable CORS
    c := cors.New(cors.Options{
        AllowedOrigins:   []string{"*"},
        AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"*"},
        AllowCredentials: true,
    })

    handler := c.Handler(router)

    log.Printf("🎯 Picker Wheel API starting on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, handler))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    response := HealthResponse{
        Status:  "healthy",
        Service: "picker-wheel-api",
        Version: "1.0.0",
    }
    json.NewEncoder(w).Encode(response)
}

func getWheelsHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(wheels)
}

func createWheelHandler(w http.ResponseWriter, r *http.Request) {
    var wheel Wheel
    if err := json.NewDecoder(r.Body).Decode(&wheel); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    wheel.ID = fmt.Sprintf("wheel_%d", time.Now().Unix())
    wheel.CreatedAt = time.Now()
    wheels = append(wheels, wheel)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(wheel)
}

func getWheelHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    for _, wheel := range wheels {
        if wheel.ID == id {
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(wheel)
            return
        }
    }

    http.Error(w, "Wheel not found", http.StatusNotFound)
}

func deleteWheelHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    for i, wheel := range wheels {
        if wheel.ID == id {
            wheels = append(wheels[:i], wheels[i+1:]...)
            w.WriteHeader(http.StatusNoContent)
            return
        }
    }

    http.Error(w, "Wheel not found", http.StatusNotFound)
}

func getHistoryHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(history)
}

func spinHandler(w http.ResponseWriter, r *http.Request) {
    var spinRequest struct {
        WheelID   string   `json:"wheel_id"`
        SessionID string   `json:"session_id"`
        Options   []Option `json:"options"`
    }

    if err := json.NewDecoder(r.Body).Decode(&spinRequest); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // For simplicity, just return a mock result
    result := SpinResult{
        Result:    "Random Result",
        WheelID:   spinRequest.WheelID,
        SessionID: spinRequest.SessionID,
        Timestamp: time.Now(),
    }

    history = append(history, result)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}