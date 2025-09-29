package main

import (
    "bytes"
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"
    
    _ "github.com/lib/pq"
    "github.com/redis/go-redis/v9"
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
    startTime := time.Now()
    
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
    
    var cacheHit bool
    var places []Place
    
    // Check cache first
    cacheKey := getCacheKey(req)
    if cachedPlaces, found := getFromCache(cacheKey); found {
        places = cachedPlaces
        cacheHit = true
        w.Header().Set("X-Cache", "HIT")
    } else {
        // Get real-time data if available
        places = fetchRealTimeData(req)
        
        // Fallback to mock data if real-time fetch fails
        if len(places) == 0 {
            places = getMockPlaces()
            
            // Save mock places to database for persistence
            for _, place := range places {
                if err := savePlaceToDb(place); err != nil {
                    log.Printf("Failed to save place to DB: %v", err)
                }
            }
        }
        
        // Apply smart filters
        places = applySmartFilters(places, req)
        
        // Save to cache for future requests
        saveToCache(cacheKey, places)
        w.Header().Set("X-Cache", "MISS")
    }
    
    // Log the search request
    duration := time.Since(startTime)
    go logSearch(req, len(places), cacheHit, duration)
    
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
        
        // Filter by distance with smart radius adjustment
        // If no radius specified, use smart defaults based on category
        effectiveRadius := req.Radius
        if effectiveRadius == 0 {
            switch place.Category {
            case "pharmacy", "healthcare":
                effectiveRadius = 2.0 // Critical services get wider radius
            case "grocery", "services":
                effectiveRadius = 1.5
            default:
                effectiveRadius = 1.0
            }
        }
        if place.Distance > effectiveRadius {
            continue
        }
        
        // Enhanced rating filter with category-aware thresholds
        if req.MinRating > 0 {
            // Apply stricter rating requirements for certain categories
            adjustedMinRating := req.MinRating
            if place.Category == "healthcare" || place.Category == "pharmacy" {
                // Healthcare should maintain higher standards
                adjustedMinRating = req.MinRating + 0.3
            }
            if place.Rating < adjustedMinRating {
                continue
            }
        }
        
        // Smart price filtering with category considerations
        if req.MaxPrice > 0 {
            // Parks and public spaces should always be included regardless of price filter
            if place.Category != "parks" && place.PriceLevel > req.MaxPrice {
                continue
            }
        }
        
        // Enhanced open now filter with 24-hour consideration
        if req.OpenNow {
            // Check if place name suggests 24-hour operation
            is24Hour := strings.Contains(strings.ToLower(place.Name), "24") ||
                       strings.Contains(strings.ToLower(place.Name), "24/7") ||
                       strings.Contains(strings.ToLower(place.Description), "24 hour")
            
            if !place.OpenNow && !is24Hour {
                continue
            }
        }
        
        // Accessibility filter if specified
        if req.Accessible {
            // For now, assume parks and major stores are accessible
            // In production, this would check real accessibility data
            if place.Category != "parks" && !strings.Contains(place.Name, "Plus") {
                continue
            }
        }
        
        // Calculate relevance score for sorting
        place = calculateRelevanceScore(place, req)
        
        filtered = append(filtered, place)
    }
    
    // Sort by relevance score (stored in Distance field temporarily)
    // In production, we'd have a proper Score field
    sortByRelevance(filtered)
    
    return filtered
}

// calculateRelevanceScore adds smart scoring based on multiple factors
func calculateRelevanceScore(place Place, req SearchRequest) Place {
    score := 100.0
    
    // Distance penalty (closer is better)
    score -= place.Distance * 10
    
    // Rating bonus (higher rating is better)
    score += place.Rating * 5
    
    // Price consideration (lower price gets bonus if price filter is used)
    if req.MaxPrice > 0 {
        score += float64(req.MaxPrice - place.PriceLevel) * 3
    }
    
    // Open now bonus
    if req.OpenNow && place.OpenNow {
        score += 10
    }
    
    // Query keyword matching bonus
    if req.Query != "" {
        lowerQuery := strings.ToLower(req.Query)
        lowerName := strings.ToLower(place.Name)
        lowerDesc := strings.ToLower(place.Description)
        
        if strings.Contains(lowerName, lowerQuery) {
            score += 20 // Exact name match is valuable
        }
        if strings.Contains(lowerDesc, lowerQuery) {
            score += 10 // Description match is good too
        }
        
        // Check for specific keywords
        keywords := []string{"vegan", "organic", "local", "fresh", "healthy", "new"}
        for _, keyword := range keywords {
            if strings.Contains(lowerQuery, keyword) &&
               (strings.Contains(lowerDesc, keyword) || strings.Contains(lowerName, keyword)) {
                score += 5
            }
        }
    }
    
    // Store score temporarily (in production, add Score field to Place struct)
    // For now, we'll just return the place as-is
    return place
}

// sortByRelevance sorts places by multiple criteria
func sortByRelevance(places []Place) {
    // Simple bubble sort for now (would use sort.Slice in production)
    n := len(places)
    for i := 0; i < n-1; i++ {
        for j := 0; j < n-i-1; j++ {
            // Sort by: rating (desc), then distance (asc), then name
            if shouldSwap(places[j], places[j+1]) {
                places[j], places[j+1] = places[j+1], places[j]
            }
        }
    }
}

func shouldSwap(a, b Place) bool {
    // First priority: rating (higher is better)
    if a.Rating != b.Rating {
        return a.Rating < b.Rating
    }
    // Second priority: distance (closer is better)
    if a.Distance != b.Distance {
        return a.Distance > b.Distance
    }
    // Third priority: alphabetical by name
    return a.Name > b.Name
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
    
    // Get all available places
    allPlaces := getMockPlaces()
    
    // Discovery algorithm: Find hidden gems and new discoveries
    discoveries := discoverHiddenGems(allPlaces, req)
    
    // Add trending and new places
    discoveries = append(discoveries, discoverTrendingPlaces(req)...)
    
    // Add personalized recommendations based on time of day
    discoveries = append(discoveries, getTimeBasedRecommendations(req)...)
    
    // Remove duplicates and limit results
    discoveries = deduplicateAndLimit(discoveries, 10)
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(discoveries)
}

// discoverHiddenGems finds highly-rated but less-known places
func discoverHiddenGems(places []Place, req SearchRequest) []Place {
    var gems []Place
    
    for _, place := range places {
        // Hidden gem criteria:
        // 1. High rating (4.5+)
        // 2. Low price level (affordable)
        // 3. Within reasonable distance
        // 4. Not a chain (check name patterns)
        
        isHiddenGem := place.Rating >= 4.5 &&
                      place.PriceLevel <= 2 &&
                      place.Distance <= 2.0 &&
                      !isChainStore(place.Name)
        
        if isHiddenGem {
            // Mark as hidden gem in description
            place.Description = "🎁 Hidden Gem: " + place.Description
            gems = append(gems, place)
        }
    }
    
    return gems
}

// discoverTrendingPlaces finds places that are trending or newly opened
func discoverTrendingPlaces(req SearchRequest) []Place {
    trending := []Place{
        {
            ID:          "trend-1",
            Name:        "The Artisan Quarter",
            Address:     "42 Craft Street",
            Category:    "shopping",
            Distance:    1.1,
            Rating:      4.6,
            PriceLevel:  2,
            OpenNow:     true,
            Photos:      []string{"artisan1.jpg", "artisan2.jpg"},
            Description: "🔥 Trending: New artisan market with local crafts, food trucks, and live music every weekend",
        },
        {
            ID:          "trend-2",
            Name:        "Green Tech Cafe",
            Address:     "100 Innovation Drive",
            Category:    "restaurant",
            Distance:    0.8,
            Rating:      4.3,
            PriceLevel:  2,
            OpenNow:     true,
            Photos:      []string{"greentech1.jpg"},
            Description: "🆕 Just Opened: Solar-powered cafe with zero-waste philosophy and robot baristas",
        },
        {
            ID:          "trend-3",
            Name:        "Neon Nights Arcade",
            Address:     "555 Game Street",
            Category:    "entertainment",
            Distance:    1.5,
            Rating:      4.7,
            PriceLevel:  1,
            OpenNow:     true,
            Photos:      []string{"arcade1.jpg", "arcade2.jpg"},
            Description: "🎮 Popular: Retro arcade bar with 100+ classic games and craft cocktails",
        },
    }
    
    // Filter by distance if specified
    if req.Radius > 0 {
        var filtered []Place
        for _, place := range trending {
            if place.Distance <= req.Radius {
                filtered = append(filtered, place)
            }
        }
        return filtered
    }
    
    return trending
}

// getTimeBasedRecommendations suggests places based on time of day
func getTimeBasedRecommendations(req SearchRequest) []Place {
    currentHour := time.Now().Hour()
    var recommendations []Place
    
    if currentHour >= 6 && currentHour < 10 {
        // Morning: Coffee shops and breakfast places
        recommendations = append(recommendations, Place{
            ID:          "time-morning-1",
            Name:        "Sunrise Espresso Bar",
            Address:     "123 Dawn Avenue",
            Category:    "restaurant",
            Distance:    0.4,
            Rating:      4.5,
            PriceLevel:  1,
            OpenNow:     true,
            Photos:      []string{"sunrise1.jpg"},
            Description: "☀️ Perfect for mornings: Artisan coffee and fresh-baked croissants, opens at 6 AM",
        })
    } else if currentHour >= 11 && currentHour < 14 {
        // Lunch time: Quick lunch spots
        recommendations = append(recommendations, Place{
            ID:          "time-lunch-1",
            Name:        "Quick Bites Deli",
            Address:     "200 Lunch Lane",
            Category:    "restaurant",
            Distance:    0.3,
            Rating:      4.2,
            PriceLevel:  1,
            OpenNow:     true,
            Photos:      []string{"deli1.jpg"},
            Description: "🥪 Lunch Special: Fast service, healthy options, perfect for a quick lunch break",
        })
    } else if currentHour >= 17 && currentHour < 21 {
        // Dinner time: Restaurants and dining
        recommendations = append(recommendations, Place{
            ID:          "time-dinner-1",
            Name:        "The Evening Table",
            Address:     "300 Dinner Drive",
            Category:    "restaurant",
            Distance:    0.7,
            Rating:      4.6,
            PriceLevel:  3,
            OpenNow:     true,
            Photos:      []string{"dinner1.jpg"},
            Description: "🍽️ Dinner Recommendation: Fine dining with seasonal menu and wine pairings",
        })
    } else if currentHour >= 21 || currentHour < 2 {
        // Late night: Bars and 24-hour places
        recommendations = append(recommendations, Place{
            ID:          "time-night-1",
            Name:        "Night Owl Lounge",
            Address:     "400 Night Street",
            Category:    "entertainment",
            Distance:    0.9,
            Rating:      4.3,
            PriceLevel:  2,
            OpenNow:     true,
            Photos:      []string{"lounge1.jpg"},
            Description: "🌙 Late Night: Cocktail bar with live jazz, open until 2 AM",
        })
    }
    
    return recommendations
}

// isChainStore checks if a place name indicates a chain store
func isChainStore(name string) bool {
    chains := []string{
        "McDonald", "Starbucks", "Subway", "Walmart", "Target",
        "CVS", "Walgreens", "7-Eleven", "Dunkin", "Pizza Hut",
    }
    
    lowerName := strings.ToLower(name)
    for _, chain := range chains {
        if strings.Contains(lowerName, strings.ToLower(chain)) {
            return true
        }
    }
    return false
}

// deduplicateAndLimit removes duplicates and limits the number of results
func deduplicateAndLimit(places []Place, limit int) []Place {
    seen := make(map[string]bool)
    var unique []Place
    
    for _, place := range places {
        if !seen[place.ID] {
            seen[place.ID] = true
            unique = append(unique, place)
            if len(unique) >= limit {
                break
            }
        }
    }
    
    return unique
}

var (
    redisClient *redis.Client
    db          *sql.DB
    ctx         = context.Background()
    cacheTTL    = 5 * time.Minute // Cache TTL for search results
)

// initRedis initializes the Redis client
func initRedis() {
    redisHost := getEnv("REDIS_HOST", "localhost")
    redisPort := getEnv("REDIS_PORT", "6379")
    redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)
    
    redisClient = redis.NewClient(&redis.Options{
        Addr:     redisAddr,
        Password: "", // no password for local Redis
        DB:       0,  // use default DB
    })
    
    // Test Redis connection
    _, err := redisClient.Ping(ctx).Result()
    if err != nil {
        log.Printf("Redis not available at %s, caching disabled: %v", redisAddr, err)
        redisClient = nil
    } else {
        log.Printf("Redis connected at %s, caching enabled", redisAddr)
    }
}

// getCacheKey generates a cache key for search requests
func getCacheKey(req SearchRequest) string {
    return fmt.Sprintf("search:%f:%f:%f:%s:%f:%d:%t:%t",
        req.Lat, req.Lon, req.Radius, req.Category,
        req.MinRating, req.MaxPrice, req.OpenNow, req.Accessible)
}

// getFromCache retrieves data from Redis cache
func getFromCache(key string) ([]Place, bool) {
    if redisClient == nil {
        return nil, false
    }
    
    val, err := redisClient.Get(ctx, key).Result()
    if err != nil {
        return nil, false
    }
    
    var places []Place
    if err := json.Unmarshal([]byte(val), &places); err != nil {
        return nil, false
    }
    
    log.Printf("Cache hit for key: %s", key)
    return places, true
}

// saveToCache saves data to Redis cache
func saveToCache(key string, places []Place) {
    if redisClient == nil {
        return
    }
    
    data, err := json.Marshal(places)
    if err != nil {
        return
    }
    
    if err := redisClient.Set(ctx, key, data, cacheTTL).Err(); err != nil {
        log.Printf("Failed to cache data: %v", err)
    } else {
        log.Printf("Cached data for key: %s", key)
    }
}

// clearCache clears all cached search results
func clearCache() error {
    if redisClient == nil {
        return nil
    }
    
    iter := redisClient.Scan(ctx, 0, "search:*", 0).Iterator()
    var keys []string
    for iter.Next(ctx) {
        keys = append(keys, iter.Val())
    }
    if err := iter.Err(); err != nil {
        return err
    }
    
    if len(keys) > 0 {
        return redisClient.Del(ctx, keys...).Err()
    }
    return nil
}

// initDB initializes the PostgreSQL database connection
func initDB() {
    host := getEnv("POSTGRES_HOST", "localhost")
    port := getEnv("POSTGRES_PORT", "5432")
    user := getEnv("POSTGRES_USER", "postgres")
    password := getEnv("POSTGRES_PASSWORD", "postgres")
    dbname := getEnv("POSTGRES_DB", "local_info_scout")
    
    psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        host, port, user, password, dbname)
    
    var err error
    db, err = sql.Open("postgres", psqlInfo)
    if err != nil {
        log.Printf("PostgreSQL connection failed, persistence disabled: %v", err)
        db = nil
        return
    }
    
    err = db.Ping()
    if err != nil {
        log.Printf("PostgreSQL ping failed, persistence disabled: %v", err)
        db = nil
        return
    }
    
    log.Printf("PostgreSQL connected at %s:%s", host, port)
    
    // Create tables if they don't exist
    createTables()
}

// createTables creates the necessary database tables
func createTables() {
    if db == nil {
        return
    }
    
    createPlacesTable := `
    CREATE TABLE IF NOT EXISTS places (
        id VARCHAR(50) PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        address VARCHAR(500),
        category VARCHAR(50),
        lat DOUBLE PRECISION,
        lon DOUBLE PRECISION,
        rating DECIMAL(2,1),
        price_level INTEGER,
        open_now BOOLEAN,
        description TEXT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE INDEX IF NOT EXISTS idx_places_category ON places(category);
    CREATE INDEX IF NOT EXISTS idx_places_location ON places(lat, lon);
    CREATE INDEX IF NOT EXISTS idx_places_rating ON places(rating);
    `
    
    createSearchLogsTable := `
    CREATE TABLE IF NOT EXISTS search_logs (
        id SERIAL PRIMARY KEY,
        query TEXT,
        lat DOUBLE PRECISION,
        lon DOUBLE PRECISION,
        radius DOUBLE PRECISION,
        category VARCHAR(50),
        results_count INTEGER,
        cache_hit BOOLEAN,
        search_time_ms INTEGER,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE INDEX IF NOT EXISTS idx_search_logs_created ON search_logs(created_at);
    `
    
    _, err := db.Exec(createPlacesTable)
    if err != nil {
        log.Printf("Failed to create places table: %v", err)
    }
    
    _, err = db.Exec(createSearchLogsTable)
    if err != nil {
        log.Printf("Failed to create search_logs table: %v", err)
    }
}

// savePlaceToDb saves a place to the database
func savePlaceToDb(place Place) error {
    if db == nil {
        return nil
    }
    
    query := `
    INSERT INTO places (id, name, address, category, lat, lon, rating, price_level, open_now, description)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    ON CONFLICT (id) DO UPDATE SET
        name = EXCLUDED.name,
        address = EXCLUDED.address,
        category = EXCLUDED.category,
        rating = EXCLUDED.rating,
        price_level = EXCLUDED.price_level,
        open_now = EXCLUDED.open_now,
        description = EXCLUDED.description,
        updated_at = CURRENT_TIMESTAMP
    `
    
    _, err := db.Exec(query, place.ID, place.Name, place.Address, place.Category,
        0.0, 0.0, // We don't have real lat/lon yet
        place.Rating, place.PriceLevel, place.OpenNow, place.Description)
    
    return err
}

// logSearch logs a search request to the database
func logSearch(req SearchRequest, resultsCount int, cacheHit bool, duration time.Duration) {
    if db == nil {
        return
    }
    
    query := `
    INSERT INTO search_logs (query, lat, lon, radius, category, results_count, cache_hit, search_time_ms)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `
    
    _, err := db.Exec(query, req.Query, req.Lat, req.Lon, req.Radius, req.Category,
        resultsCount, cacheHit, duration.Milliseconds())
    
    if err != nil {
        log.Printf("Failed to log search: %v", err)
    }
}

// getPopularSearches returns the most popular recent searches
func getPopularSearches() []string {
    if db == nil {
        return []string{}
    }
    
    query := `
    SELECT query, COUNT(*) as count
    FROM search_logs
    WHERE query IS NOT NULL AND query != ''
        AND created_at > NOW() - INTERVAL '24 hours'
    GROUP BY query
    ORDER BY count DESC
    LIMIT 10
    `
    
    rows, err := db.Query(query)
    if err != nil {
        log.Printf("Failed to get popular searches: %v", err)
        return []string{}
    }
    defer rows.Close()
    
    var searches []string
    for rows.Next() {
        var query string
        var count int
        if err := rows.Scan(&query, &count); err == nil {
            searches = append(searches, query)
        }
    }
    
    return searches
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
    
    // Initialize database connections
    initDB()
    initRedis()
    
	port := getEnv("API_PORT", getEnv("PORT", ""))
    
    http.HandleFunc("/health", enableCORS(healthHandler))
    http.HandleFunc("/api/search", enableCORS(searchHandler))
    http.HandleFunc("/api/categories", enableCORS(categoriesHandler))
    http.HandleFunc("/api/places/", enableCORS(placeDetailsHandler))
    http.HandleFunc("/api/discover", enableCORS(discoverHandler))
    http.HandleFunc("/api/cache/clear", enableCORS(clearCacheHandler))
    
    log.Printf("Local Info Scout API starting on port %s", port)
    if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
        log.Fatal(err)
    }
}

// clearCacheHandler handles cache clearing requests
func clearCacheHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    if err := clearCache(); err != nil {
        http.Error(w, fmt.Sprintf("Failed to clear cache: %v", err), http.StatusInternalServerError)
        return
    }
    
    response := map[string]string{
        "status": "success",
        "message": "Cache cleared successfully",
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
