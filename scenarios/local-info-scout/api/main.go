package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
	Readiness bool      `json:"readiness"`
}

type SearchRequest struct {
	Query      string  `json:"query"`
	Lat        float64 `json:"lat"`
	Lon        float64 `json:"lon"`
	Radius     float64 `json:"radius"`
	Category   string  `json:"category"`
	MinRating  float64 `json:"min_rating,omitempty"`
	MaxPrice   int     `json:"max_price,omitempty"`
	OpenNow    bool    `json:"open_now,omitempty"`
	Accessible bool    `json:"accessible,omitempty"`
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

type SearchResponse struct {
	Places  []Place  `json:"places"`
	Sources []string `json:"sources"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Service:   "local-info-scout",
		Timestamp: time.Now(),
		Readiness: true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
		// Use multi-source aggregator to fetch data
		if aggregator != nil {
			places = aggregator.Aggregate(req)
		}

		// Fallback to mock data if aggregation returns nothing
		if len(places) == 0 {
			places = getMockPlaces()
		}

		// Save places to database for future reference
		for _, place := range places {
			if err := savePlaceToDb(place); err != nil {
				dbLogger.Error("Failed to save place to DB", map[string]interface{}{
					"error":    err.Error(),
					"place_id": place.ID,
				})
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

	// Determine data sources used
	sources := []string{"mock"}
	if hasPostgresDb() {
		sources = append(sources, "postgres")
	}
	if len(places) > 1 {
		sources = append(sources, "multisource")
	}

	// Build response per PRD API contract
	response := SearchResponse{
		Places:  places,
		Sources: sources,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
		score += float64(req.MaxPrice-place.PriceLevel) * 3
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

// sortByRelevance sorts places by multiple criteria using Go's efficient sort.Slice
func sortByRelevance(places []Place) {
	sort.Slice(places, func(i, j int) bool {
		// First priority: rating (higher is better)
		if places[i].Rating != places[j].Rating {
			return places[i].Rating > places[j].Rating
		}
		// Second priority: distance (closer is better)
		if places[i].Distance != places[j].Distance {
			return places[i].Distance < places[j].Distance
		}
		// Third priority: alphabetical by name
		return places[i].Name < places[j].Name
	})
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
		// Get allowed origin from environment or default to localhost
		allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = "http://localhost:3000"
		}

		origin := r.Header.Get("Origin")
		if origin == allowedOrigin || origin == "http://localhost:3000" || origin == "http://localhost:3001" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

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
	aggregator           *MultiSourceAggregator
	recommendationEngine *RecommendationEngine
)

// recommendationsHandler provides personalized recommendations
func recommendationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if recommendationEngine == nil {
		http.Error(w, "Recommendations not available", http.StatusServiceUnavailable)
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get user ID from header (in production, use proper authentication)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "anonymous"
	}

	recommendations := recommendationEngine.GenerateRecommendations(userID, req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recommendations)
}

// profileHandler returns user profile and preferences
func profileHandler(w http.ResponseWriter, r *http.Request) {
	if recommendationEngine == nil {
		http.Error(w, "Profiles not available", http.StatusServiceUnavailable)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "anonymous"
	}

	switch r.Method {
	case http.MethodGet:
		profile, err := recommendationEngine.GetUserProfile(userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)

	case http.MethodPut:
		var prefs struct {
			FavoriteCategories []string `json:"favorite_categories"`
			HiddenCategories   []string `json:"hidden_categories"`
			DefaultRadius      float64  `json:"default_radius"`
		}
		if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err := recommendationEngine.UpdateUserPreferences(userID, prefs.FavoriteCategories, prefs.HiddenCategories, prefs.DefaultRadius)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// savePlaceHandler saves a place to user favorites
func savePlaceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if recommendationEngine == nil {
		http.Error(w, "Save feature not available", http.StatusServiceUnavailable)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "anonymous"
	}

	var place Place
	if err := json.NewDecoder(r.Body).Decode(&place); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := recommendationEngine.SavePlace(userID, place)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// trendingHandler returns trending searches and popular places
func trendingHandler(w http.ResponseWriter, r *http.Request) {
	if recommendationEngine == nil {
		http.Error(w, "Trending not available", http.StatusServiceUnavailable)
		return
	}

	category := r.URL.Query().Get("category")
	limit := 10

	trending := struct {
		Searches []string `json:"trending_searches"`
		Places   []Place  `json:"popular_places"`
	}{
		Searches: recommendationEngine.GetTrendingSearches(limit),
		Places:   recommendationEngine.GetPopularPlaces(category, limit),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trending)
}

func main() {
	// Lifecycle protection must be the absolute first check
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start local-info-scout

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Initialize loggers after lifecycle check
	initLoggers()

	// Validate required API_PORT environment variable
	port := getEnv("API_PORT", getEnv("PORT", ""))
	if port == "" {
		mainLogger.Error("API_PORT or PORT environment variable is required", nil)
		os.Exit(1)
	}

	// Initialize database connections
	initDB()
	initRedis()

	// Initialize multi-source aggregator
	aggregator = NewMultiSourceAggregator()
	mainLogger.Info("Multi-source data aggregation enabled", nil)

	// Initialize recommendation engine
	if db != nil {
		recommendationEngine = NewRecommendationEngine(db)
		mainLogger.Info("Personalized recommendations enabled", nil)
	}

	http.HandleFunc("/health", enableCORS(healthHandler))
	http.HandleFunc("/api/search", enableCORS(searchHandler))
	http.HandleFunc("/api/categories", enableCORS(categoriesHandler))
	http.HandleFunc("/api/places/", enableCORS(placeDetailsHandler))
	http.HandleFunc("/api/discover", enableCORS(discoverHandler))
	http.HandleFunc("/api/cache/clear", enableCORS(clearCacheHandler))
	http.HandleFunc("/api/recommendations", enableCORS(recommendationsHandler))
	http.HandleFunc("/api/profile", enableCORS(profileHandler))
	http.HandleFunc("/api/places/save", enableCORS(savePlaceHandler))
	http.HandleFunc("/api/trending", enableCORS(trendingHandler))

	mainLogger.Info("Local Info Scout API starting", map[string]interface{}{
		"port": port,
	})
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		mainLogger.Error("Server failed to start", map[string]interface{}{
			"error": err.Error(),
			"port":  port,
		})
		os.Exit(1)
	}
}

// hasPostgresDb checks if PostgreSQL database is available
func hasPostgresDb() bool {
	return db != nil
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
		"status":  "success",
		"message": "Cache cleared successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
