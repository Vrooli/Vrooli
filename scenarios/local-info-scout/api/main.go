package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"
)

type HealthResponse struct {
    Status    string    `json:"status"`
    Service   string    `json:"service"`
    Timestamp time.Time `json:"timestamp"`
}

type SearchRequest struct {
    Query       string  `json:"query"`
    Lat         float64 `json:"lat"`
    Lon         float64 `json:"lon"`
    Radius      float64 `json:"radius"`
    Category    string  `json:"category"`
    MinRating   float64 `json:"min_rating,omitempty"`
    MaxPrice    int     `json:"max_price,omitempty"`
    OpenNow     bool    `json:"open_now,omitempty"`
    Accessible  bool    `json:"accessible,omitempty"`
}

type OllamaRequest struct {
    Model  string `json:"model"`
    Prompt string `json:"prompt"`
    Stream bool   `json:"stream"`
}

type OllamaResponse struct {
    Response string `json:"response"`
}

type ParsedQuery struct {
    Category string  `json:"category"`
    Radius   float64 `json:"radius"`
    Keywords []string `json:"keywords"`
}

type Place struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Address     string   `json:"address"`
    Category    string   `json:"category"`
    Distance    float64  `json:"distance"`
    Rating      float64  `json:"rating"`
    PriceLevel  int      `json:"price_level"`
    OpenNow     bool     `json:"open_now"`
    Photos      []string `json:"photos"`
    Description string   `json:"description"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    response := HealthResponse{
        Status:    "healthy",
        Service:   "local-info-scout",
        Timestamp: time.Now(),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func parseNaturalLanguageQuery(query string) ParsedQuery {
    parsed := ParsedQuery{
        Category: "",
        Radius:   1.0, // Default 1 mile
        Keywords: []string{},
    }
    
    // Try to use Ollama for parsing if available
    ollamaHost := os.Getenv("OLLAMA_HOST")
    if ollamaHost == "" {
        ollamaHost = "http://localhost:11434"
    }
    
    prompt := fmt.Sprintf(`Parse this location search query and extract structured information.
Query: "%s"

Extract:
1. Category (restaurant, grocery, pharmacy, parks, shopping, entertainment, services, fitness, healthcare, or empty if not specified)
2. Radius in miles (default 1 if not specified)
3. Keywords (relevant search terms)

Respond in JSON format like:
{"category": "restaurant", "radius": 2.0, "keywords": ["vegan", "organic"]}`, query)
    
    reqBody, _ := json.Marshal(OllamaRequest{
        Model:  "llama3.2:latest",
        Prompt: prompt,
        Stream: false,
    })
    
    resp, err := http.Post(ollamaHost+"/api/generate", "application/json", bytes.NewBuffer(reqBody))
    if err == nil && resp.StatusCode == 200 {
        defer resp.Body.Close()
        body, _ := io.ReadAll(resp.Body)
        var ollamaResp OllamaResponse
        if json.Unmarshal(body, &ollamaResp) == nil {
            // Try to parse the JSON from Ollama's response
            json.Unmarshal([]byte(ollamaResp.Response), &parsed)
        }
    }
    
    // Fallback parsing if Ollama is not available or fails
    if len(parsed.Keywords) == 0 {
        lowerQuery := strings.ToLower(query)
        
        // Parse radius
        radiusPatterns := []string{" mile", " mi", " km", " kilometer"}
        for _, pattern := range radiusPatterns {
            if idx := strings.Index(lowerQuery, pattern); idx > 0 {
                // Try to extract number before the pattern
                parts := strings.Fields(lowerQuery[:idx])
                if len(parts) > 0 {
                    if radius, err := strconv.ParseFloat(parts[len(parts)-1], 64); err == nil {
                        parsed.Radius = radius
                    }
                }
            }
        }
        
        // Parse category
        categories := []string{"restaurant", "grocery", "pharmacy", "park", "shopping", "entertainment", "service", "fitness", "healthcare"}
        for _, cat := range categories {
            if strings.Contains(lowerQuery, cat) {
                parsed.Category = cat
                if cat == "park" {
                    parsed.Category = "parks"
                } else if cat == "service" {
                    parsed.Category = "services"
                }
                break
            }
        }
        
        // Extract keywords
        keywords := []string{"vegan", "organic", "healthy", "fast", "cheap", "luxury", "local", "new", "24 hour", "open late"}
        for _, keyword := range keywords {
            if strings.Contains(lowerQuery, keyword) {
                parsed.Keywords = append(parsed.Keywords, keyword)
            }
        }
    }
    
    return parsed
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    var req SearchRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Parse natural language query if provided
    if req.Query != "" {
        parsed := parseNaturalLanguageQuery(req.Query)
        if req.Category == "" {
            req.Category = parsed.Category
        }
        if req.Radius == 0 {
            req.Radius = parsed.Radius
        }
    }
    
    // Get real-time data if available
    places := fetchRealTimeData(req)
    
    // Fallback to mock data if real-time fetch fails
    if len(places) == 0 {
        places = getMockPlaces()
    }
    
    // Apply smart filters
    places = applySmartFilters(places, req)
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(places)
}

func getMockPlaces() []Place {
    return []Place{
        {
            ID:          "1",
            Name:        "Green Garden Cafe",
            Address:     "123 Main St",
            Category:    "restaurant",
            Distance:    0.5,
            Rating:      4.5,
            PriceLevel:  2,
            OpenNow:     true,
            Description: "Cozy vegan restaurant with organic options",
        },
        {
            ID:          "2",
            Name:        "Nature's Bounty Market",
            Address:     "456 Oak Ave",
            Category:    "grocery",
            Distance:    0.8,
            Rating:      4.2,
            PriceLevel:  2,
            OpenNow:     true,
            Description: "Local organic grocery store",
        },
        {
            ID:          "3",
            Name:        "24/7 Pharmacy Plus",
            Address:     "789 Health Ave",
            Category:    "pharmacy",
            Distance:    1.2,
            Rating:      3.8,
            PriceLevel:  3,
            OpenNow:     true,
            Description: "24-hour pharmacy with medical supplies",
        },
        {
            ID:          "4",
            Name:        "Sunset Park",
            Address:     "101 Park Drive",
            Category:    "parks",
            Distance:    0.3,
            Rating:      4.7,
            PriceLevel:  0,
            OpenNow:     true,
            Description: "Beautiful public park with walking trails",
        },
    }
}

func applySmartFilters(places []Place, req SearchRequest) []Place {
    var filtered []Place
    
    for _, place := range places {
        // Filter by category
        if req.Category != "" && place.Category != req.Category {
            continue
        }
        
        // Filter by distance
        if req.Radius > 0 && place.Distance > req.Radius {
            continue
        }
        
        // Filter by rating
        if req.MinRating > 0 && place.Rating < req.MinRating {
            continue
        }
        
        // Filter by price
        if req.MaxPrice > 0 && place.PriceLevel > req.MaxPrice {
            continue
        }
        
        // Filter by open now
        if req.OpenNow && !place.OpenNow {
            continue
        }
        
        filtered = append(filtered, place)
    }
    
    return filtered
}

func fetchRealTimeData(req SearchRequest) []Place {
    // Try to fetch from SearXNG if available
    searxHost := os.Getenv("SEARXNG_HOST")
    if searxHost == "" {
        searxHost = "http://localhost:8280"
    }
    
    // Build search query
    searchQuery := fmt.Sprintf("%s near lat:%f lon:%f", req.Query, req.Lat, req.Lon)
    if req.Category != "" {
        searchQuery = fmt.Sprintf("%s %s", req.Category, searchQuery)
    }
    
    // Try SearXNG search
    searchURL := fmt.Sprintf("%s/search?q=%s&format=json&categories=map", searxHost, searchQuery)
    resp, err := http.Get(searchURL)
    if err != nil || resp.StatusCode != 200 {
        return []Place{} // Return empty, will fallback to mock data
    }
    defer resp.Body.Close()
    
    // For now, return empty as SearXNG integration would need more complex parsing
    // In production, this would parse the search results and convert to Place objects
    return []Place{}
}

func categoriesHandler(w http.ResponseWriter, r *http.Request) {
    categories := []string{
        "restaurants",
        "grocery",
        "pharmacy",
        "parks",
        "shopping",
        "entertainment",
        "services",
        "fitness",
        "healthcare",
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(categories)
}

func placeDetailsHandler(w http.ResponseWriter, r *http.Request) {
    // Extract place ID from URL path
    path := r.URL.Path
    id := path[len("/api/places/"):]
    
    if id == "" {
        http.Error(w, "Place ID required", http.StatusBadRequest)
        return
    }
    
    // Mock place details for now
    place := Place{
        ID:          id,
        Name:        "Green Garden Cafe",
        Address:     "123 Main St",
        Category:    "restaurant",
        Distance:    0.5,
        Rating:      4.5,
        PriceLevel:  2,
        OpenNow:     true,
        Photos:      []string{"photo1.jpg", "photo2.jpg"},
        Description: "Cozy vegan restaurant with organic options. Full menu available with daily specials.",
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(place)
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next(w, r)
    }
}

func discoverHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    var req SearchRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Discovery mode returns hidden gems and new openings
    discoveries := []Place{
        {
            ID:          "disc-1",
            Name:        "Hidden Gem Cafe",
            Address:     "789 Secret Lane",
            Category:    "restaurant",
            Distance:    1.2,
            Rating:      4.8,
            PriceLevel:  1,
            OpenNow:     true,
            Photos:      []string{"gem1.jpg"},
            Description: "A hidden gem known only to locals - amazing coffee and pastries",
        },
        {
            ID:          "disc-2",
            Name:        "New Opening: Fresh Market",
            Address:     "321 New Street",
            Category:    "grocery",
            Distance:    0.9,
            Rating:      0, // New, no ratings yet
            PriceLevel:  2,
            OpenNow:     true,
            Photos:      []string{"new1.jpg"},
            Description: "Just opened this week! Featuring local produce and artisan goods",
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(discoveries)
}

func main() {
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start local-info-scout

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        os.Exit(1)
    }
	port := getEnv("API_PORT", getEnv("PORT", ""))
    
    http.HandleFunc("/health", enableCORS(healthHandler))
    http.HandleFunc("/api/search", enableCORS(searchHandler))
    http.HandleFunc("/api/categories", enableCORS(categoriesHandler))
    http.HandleFunc("/api/places/", enableCORS(placeDetailsHandler))
    http.HandleFunc("/api/discover", enableCORS(discoverHandler))
    
    log.Printf("Local Info Scout API starting on port %s", port)
    if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
        log.Fatal(err)
    }
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
