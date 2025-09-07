package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Scenario represents a kid-friendly scenario
type Scenario struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Color       string `json:"color"`
	Category    string `json:"category"`
	Port        int    `json:"port"`
	AgeRange    string `json:"ageRange"`
}

// ServiceConfig represents the service.json structure
type ServiceConfig struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    []string `json:"category"`
	Metadata    struct {
		Tags           []string `json:"tags"`
		TargetAudience struct {
			AgeRange string `json:"ageRange"`
		} `json:"targetAudience"`
	} `json:"metadata"`
	Deployment struct {
		Port int `json:"port"`
	} `json:"deployment"`
}

// Global list of kid-friendly scenarios
var kidScenarios []Scenario

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3500"
	}

	// Initialize by scanning for kid-friendly scenarios
	scanScenarios()

	// Set up routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/v1/kids/scenarios", scenariosHandler)
	http.HandleFunc("/api/v1/kids/launch", launchHandler)
	
	// Serve static files
	fs := http.FileServer(http.Dir("../ui/dist"))
	http.Handle("/", fs)

	log.Printf("Kids Dashboard API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func scenariosHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	ageRange := r.URL.Query().Get("ageRange")
	category := r.URL.Query().Get("category")
	
	filtered := filterScenarios(kidScenarios, ageRange, category)
	
	response := map[string]interface{}{
		"scenarios": filtered,
		"count":     len(filtered),
	}
	
	json.NewEncoder(w).Encode(response)
}

func launchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		ScenarioID string `json:"scenarioId"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	// Find the scenario
	var scenario *Scenario
	for _, s := range kidScenarios {
		if s.ID == request.ScenarioID {
			scenario = &s
			break
		}
	}
	
	if scenario == nil {
		http.Error(w, "Scenario not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"url":       fmt.Sprintf("http://localhost:%d", scenario.Port),
		"sessionId": generateSessionID(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(response)
}

func scanScenarios() {
	scenariosPath := "../../../scenarios"
	
	// Define known kid-friendly scenarios with their styling
	knownKidScenarios := map[string]Scenario{
		"retro-game-launcher": {
			Title:       "Retro Games",
			Description: "Play classic arcade games!",
			Icon:        "🕹️",
			Color:       "bg-gradient-to-br from-purple-500 to-pink-500",
			Category:    "games",
			Port:        3301,
			AgeRange:    "5-12",
		},
		"picker-wheel": {
			Title:       "Picker Wheel",
			Description: "Spin the wheel for fun choices!",
			Icon:        "🎯",
			Color:       "bg-gradient-to-br from-yellow-400 to-orange-500",
			Category:    "games",
			Port:        3302,
			AgeRange:    "5-12",
		},
		"word-games": {
			Title:       "Word Games",
			Description: "Fun puzzles with letters and words!",
			Icon:        "📝",
			Color:       "bg-gradient-to-br from-green-400 to-blue-500",
			Category:    "learn",
			Port:        3303,
			AgeRange:    "9-12",
		},
		"study-buddy": {
			Title:       "Study Buddy",
			Description: "A friendly helper for homework!",
			Icon:        "📚",
			Color:       "bg-gradient-to-br from-teal-400 to-cyan-500",
			Category:    "learn",
			Port:        3304,
			AgeRange:    "9-12",
		},
	}
	
	// Walk through scenarios directory
	err := filepath.Walk(scenariosPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Look for service.json files
		if filepath.Base(path) == "service.json" && strings.Contains(path, ".vrooli") {
			// Read and parse the service.json
			data, err := ioutil.ReadFile(path)
			if err != nil {
				log.Printf("Error reading %s: %v", path, err)
				return nil
			}
			
			var config ServiceConfig
			if err := json.Unmarshal(data, &config); err != nil {
				log.Printf("Error parsing %s: %v", path, err)
				return nil
			}
			
			// Check if it's kid-friendly
			if isKidFriendly(config) {
				scenario := Scenario{
					ID:   config.Name,
					Name: config.Name,
				}
				
				// Use known metadata if available
				if known, ok := knownKidScenarios[config.Name]; ok {
					scenario.Title = known.Title
					scenario.Description = known.Description
					scenario.Icon = known.Icon
					scenario.Color = known.Color
					scenario.Category = known.Category
					scenario.Port = known.Port
					scenario.AgeRange = known.AgeRange
				} else {
					// Use defaults from service.json
					scenario.Title = config.Name
					scenario.Description = config.Description
					scenario.Port = config.Deployment.Port
					if config.Metadata.TargetAudience.AgeRange != "" {
						scenario.AgeRange = config.Metadata.TargetAudience.AgeRange
					} else {
						scenario.AgeRange = "5-12"
					}
				}
				
				kidScenarios = append(kidScenarios, scenario)
				log.Printf("Found kid-friendly scenario: %s", config.Name)
			}
		}
		
		return nil
	})
	
	if err != nil {
		log.Printf("Error scanning scenarios: %v", err)
	}
	
	log.Printf("Found %d kid-friendly scenarios", len(kidScenarios))
}

func isKidFriendly(config ServiceConfig) bool {
	// Check for explicit kid-friendly category
	for _, cat := range config.Category {
		if cat == "kid-friendly" || cat == "kids" || cat == "children" || cat == "family" {
			return true
		}
	}
	
	// Check metadata tags
	for _, tag := range config.Metadata.Tags {
		if tag == "kid-friendly" || tag == "kids" || tag == "children" {
			return true
		}
	}
	
	// Check blacklisted categories
	blacklist := []string{"system", "development", "admin", "financial", "debug", "infrastructure"}
	for _, cat := range config.Category {
		for _, blocked := range blacklist {
			if cat == blocked {
				return false
			}
		}
	}
	
	// Known kid-friendly scenarios
	knownKidFriendly := []string{"retro-game-launcher", "picker-wheel", "word-games", "study-buddy"}
	for _, known := range knownKidFriendly {
		if config.Name == known {
			return true
		}
	}
	
	return false
}

func filterScenarios(scenarios []Scenario, ageRange, category string) []Scenario {
	var filtered []Scenario
	
	for _, s := range scenarios {
		// Filter by age range
		if ageRange != "" && s.AgeRange != ageRange && s.AgeRange != "5-12" {
			continue
		}
		
		// Filter by category
		if category != "" && s.Category != category {
			continue
		}
		
		filtered = append(filtered, s)
	}
	
	return filtered
}

func generateSessionID() string {
	// Simple session ID generation
	return fmt.Sprintf("session-%d", os.Getpid())
}