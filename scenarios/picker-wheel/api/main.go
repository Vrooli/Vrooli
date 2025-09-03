package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "math/rand"
    "net/http"
    "os"
    "time"

    "github.com/gorilla/mux"
    "github.com/rs/cors"
    _ "github.com/lib/pq"
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

var db *sql.DB
var wheels []Wheel
var history []SpinResult

func initDB() {
    var err error
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        log.Println("DATABASE_URL not set, using in-memory fallback")
        return
    }
    
    db, err = sql.Open("postgres", dbURL)
    if err != nil {
        log.Printf("Failed to connect to database: %v", err)
        return
    }
    
    if err = db.Ping(); err != nil {
        log.Printf("Failed to ping database: %v", err)
        db = nil
        return
    }
    
    log.Println("Successfully connected to PostgreSQL database")
}

func main() {
    // Initialize database connection
    initDB()
    defer func() {
        if db != nil {
            db.Close()
        }
    }()
    
    // Initialize in-memory data as fallback
    wheels = getDefaultWheels()
    history = []SpinResult{}
    
	port := getEnv("API_PORT", getEnv("PORT", ""))

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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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
    
    if db == nil {
        // Fallback to default wheels if no database
        json.NewEncoder(w).Encode(getDefaultWheels())
        return
    }
    
    rows, err := db.Query(`
        SELECT id, name, description, options, theme, times_used, created_at 
        FROM wheels 
        WHERE is_public = true 
        ORDER BY times_used DESC, created_at DESC
        LIMIT 20
    `)
    if err != nil {
        log.Printf("Error querying wheels: %v", err)
        json.NewEncoder(w).Encode(getDefaultWheels())
        return
    }
    defer rows.Close()
    
    var wheels []Wheel
    for rows.Next() {
        var wheel Wheel
        var optionsJSON []byte
        err := rows.Scan(&wheel.ID, &wheel.Name, &wheel.Description, 
                        &optionsJSON, &wheel.Theme, &wheel.TimesUsed, &wheel.CreatedAt)
        if err != nil {
            log.Printf("Error scanning wheel: %v", err)
            continue
        }
        
        if err := json.Unmarshal(optionsJSON, &wheel.Options); err != nil {
            log.Printf("Error unmarshalling options: %v", err)
            continue
        }
        
        wheels = append(wheels, wheel)
    }
    
    if len(wheels) == 0 {
        wheels = getDefaultWheels()
    }
    
    json.NewEncoder(w).Encode(wheels)
}

func createWheelHandler(w http.ResponseWriter, r *http.Request) {
    var wheel Wheel
    if err := json.NewDecoder(r.Body).Decode(&wheel); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    wheel.CreatedAt = time.Now()
    
    if db == nil {
        // Fallback to in-memory if no database
        wheel.ID = fmt.Sprintf("wheel_%d", time.Now().Unix())
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(wheel)
        return
    }
    
    optionsJSON, err := json.Marshal(wheel.Options)
    if err != nil {
        http.Error(w, "Failed to marshal options", http.StatusInternalServerError)
        return
    }
    
    var id string
    err = db.QueryRow(`
        INSERT INTO wheels (name, description, options, theme, is_public) 
        VALUES ($1, $2, $3, $4, $5) 
        RETURNING id
    `, wheel.Name, wheel.Description, optionsJSON, wheel.Theme, false).Scan(&id)
    
    if err != nil {
        log.Printf("Error creating wheel: %v", err)
        http.Error(w, "Failed to create wheel", http.StatusInternalServerError)
        return
    }
    
    wheel.ID = id
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

func getDefaultWheels() []Wheel {
    return []Wheel{
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
        {
            ID:          "yes-or-no",
            Name:        "Yes or No",
            Description: "Simple decision maker",
            Options: []Option{
                {Label: "YES! ✅", Color: "#4CAF50", Weight: 1},
                {Label: "NO ❌", Color: "#F44336", Weight: 1},
            },
            Theme:     "minimal",
            CreatedAt: time.Now(),
            TimesUsed: 0,
        },
    }
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

    // Select random option based on weights
    var selectedOption string
    if len(spinRequest.Options) > 0 {
        // Calculate total weight
        var totalWeight float64
        for _, option := range spinRequest.Options {
            totalWeight += option.Weight
        }
        
        // Generate random number and select option
        randValue := rand.Float64() * totalWeight
        var currentWeight float64
        
        for _, option := range spinRequest.Options {
            currentWeight += option.Weight
            if randValue <= currentWeight {
                selectedOption = option.Label
                break
            }
        }
    } else {
        selectedOption = "No options provided"
    }

    result := SpinResult{
        Result:    selectedOption,
        WheelID:   spinRequest.WheelID,
        SessionID: spinRequest.SessionID,
        Timestamp: time.Now(),
    }

    history = append(history, result)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
