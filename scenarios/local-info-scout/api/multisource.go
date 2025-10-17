package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DataSource represents a source of place data
type DataSource interface {
	Fetch(req SearchRequest) ([]Place, error)
	Name() string
	Priority() int // Higher number = higher priority
}

// SearXNGSource fetches data from SearXNG search engine
type SearXNGSource struct {
	baseURL string
}

func NewSearXNGSource() *SearXNGSource {
	host := getEnv("SEARXNG_HOST", "http://localhost:8280")
	return &SearXNGSource{baseURL: host}
}

func (s *SearXNGSource) Name() string  { return "SearXNG" }
func (s *SearXNGSource) Priority() int { return 2 }

func (s *SearXNGSource) Fetch(req SearchRequest) ([]Place, error) {
	// Build search query
	searchQuery := req.Query
	if req.Category != "" {
		searchQuery = fmt.Sprintf("%s %s", req.Category, searchQuery)
	}
	if req.Lat != 0 && req.Lon != 0 {
		searchQuery = fmt.Sprintf("%s near lat:%f lon:%f", searchQuery, req.Lat, req.Lon)
	}

	// URL encode the query
	encodedQuery := url.QueryEscape(searchQuery)
	searchURL := fmt.Sprintf("%s/search?q=%s&format=json&categories=map", s.baseURL, encodedQuery)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("SearXNG request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("SearXNG returned status %d", resp.StatusCode)
	}

	// Parse SearXNG response (simplified - real implementation would parse actual results)
	// For now, return empty as SearXNG integration needs proper result parsing
	return []Place{}, nil
}

// LocalDBSource fetches data from local PostgreSQL database
type LocalDBSource struct{}

func NewLocalDBSource() *LocalDBSource {
	return &LocalDBSource{}
}

func (s *LocalDBSource) Name() string  { return "LocalDB" }
func (s *LocalDBSource) Priority() int { return 3 } // Highest priority - our own data

func (s *LocalDBSource) Fetch(req SearchRequest) ([]Place, error) {
	if db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT place_id, name, address, category, lat, lon, rating
		FROM lis_saved_places
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Add category filter
	if req.Category != "" {
		query += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, req.Category)
		argIndex++
	}

	// Add location-based filtering if coordinates provided
	if req.Lat != 0 && req.Lon != 0 && req.Radius > 0 {
		// Simple box filter (real implementation would use PostGIS)
		latDelta := req.Radius / 69.0 // Rough miles to degrees
		lonDelta := req.Radius / 54.6
		query += fmt.Sprintf(" AND lat BETWEEN $%d AND $%d AND lon BETWEEN $%d AND $%d",
			argIndex, argIndex+1, argIndex+2, argIndex+3)
		args = append(args, req.Lat-latDelta, req.Lat+latDelta, req.Lon-lonDelta, req.Lon+lonDelta)
	}

	query += " LIMIT 50"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	var places []Place
	for rows.Next() {
		var place Place
		var lat, lon *float64
		err := rows.Scan(&place.ID, &place.Name, &place.Address, &place.Category, &lat, &lon, &place.Rating)
		if err != nil {
			dbLogger.Error("Error scanning place", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		// Calculate distance if coordinates available
		if lat != nil && lon != nil && req.Lat != 0 && req.Lon != 0 {
			place.Distance = calculateDistance(req.Lat, req.Lon, *lat, *lon)
		}

		places = append(places, place)
	}

	return places, nil
}

// MockDataSource provides fallback mock data
type MockDataSource struct{}

func NewMockDataSource() *MockDataSource {
	return &MockDataSource{}
}

func (s *MockDataSource) Name() string  { return "MockData" }
func (s *MockDataSource) Priority() int { return 1 } // Lowest priority - fallback only

func (s *MockDataSource) Fetch(req SearchRequest) ([]Place, error) {
	places := getMockPlaces()

	// Filter by category if specified
	if req.Category != "" {
		var filtered []Place
		for _, p := range places {
			if p.Category == req.Category {
				filtered = append(filtered, p)
			}
		}
		places = filtered
	}

	return places, nil
}

// OpenStreetMapSource fetches data from OpenStreetMap Nominatim
type OpenStreetMapSource struct {
	baseURL string
}

func NewOpenStreetMapSource() *OpenStreetMapSource {
	return &OpenStreetMapSource{
		baseURL: "https://nominatim.openstreetmap.org",
	}
}

func (s *OpenStreetMapSource) Name() string  { return "OpenStreetMap" }
func (s *OpenStreetMapSource) Priority() int { return 4 } // High priority - real data

type NominatimResult struct {
	PlaceID     int64  `json:"place_id"`
	DisplayName string `json:"display_name"`
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	Type        string `json:"type"`
	Class       string `json:"class"`
	Address     struct {
		Road     string `json:"road"`
		City     string `json:"city"`
		State    string `json:"state"`
		Postcode string `json:"postcode"`
	} `json:"address"`
}

func (s *OpenStreetMapSource) Fetch(req SearchRequest) ([]Place, error) {
	if req.Lat == 0 || req.Lon == 0 {
		return nil, fmt.Errorf("coordinates required for OSM search")
	}

	// Map our categories to OSM amenity types
	amenityTypes := s.mapCategoryToAmenity(req.Category)
	if len(amenityTypes) == 0 {
		return nil, fmt.Errorf("unsupported category: %s", req.Category)
	}

	var allPlaces []Place
	for _, amenity := range amenityTypes {
		// Search around the provided coordinates
		searchURL := fmt.Sprintf(
			"%s/search?format=json&lat=%f&lon=%f&amenity=%s&limit=20",
			s.baseURL, req.Lat, req.Lon, amenity,
		)

		client := &http.Client{
			Timeout: 10 * time.Second,
			// Set User-Agent as required by Nominatim usage policy
			Transport: &userAgentTransport{http.DefaultTransport},
		}

		resp, err := client.Get(searchURL)
		if err != nil {
			mainLogger.Warn("OSM request failed", map[string]interface{}{
				"amenity": amenity,
				"error":   err.Error(),
			})
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			mainLogger.Error("Failed to read OSM response", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		if resp.StatusCode != 200 {
			mainLogger.Warn("OSM returned non-200 status", map[string]interface{}{
				"status": resp.StatusCode,
				"body":   string(body),
			})
			continue
		}

		var results []NominatimResult
		if err := json.Unmarshal(body, &results); err != nil {
			mainLogger.Error("Failed to parse OSM response", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		// Convert to our Place format
		for _, result := range results {
			place := s.convertToPlace(result, req)
			if place.ID != "" {
				allPlaces = append(allPlaces, place)
			}
		}

		// Respect Nominatim rate limits (1 request per second)
		time.Sleep(1 * time.Second)
	}

	return allPlaces, nil
}

// userAgentTransport adds User-Agent header required by Nominatim
type userAgentTransport struct {
	base http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", "LocalInfoScout/1.0 (Vrooli)")
	return t.base.RoundTrip(req)
}

func (s *OpenStreetMapSource) mapCategoryToAmenity(category string) []string {
	mapping := map[string][]string{
		"restaurant":    {"restaurant", "cafe", "fast_food"},
		"restaurants":   {"restaurant", "cafe", "fast_food"},
		"grocery":       {"supermarket", "grocery"},
		"pharmacy":      {"pharmacy"},
		"parks":         {"park"},
		"shopping":      {"shop", "mall"},
		"entertainment": {"cinema", "theatre", "nightclub"},
		"services":      {"bank", "post_office"},
		"fitness":       {"gym", "sports_centre"},
		"healthcare":    {"hospital", "clinic", "doctors"},
	}

	if amenities, ok := mapping[category]; ok {
		return amenities
	}
	return []string{}
}

func (s *OpenStreetMapSource) convertToPlace(result NominatimResult, req SearchRequest) Place {
	// Parse coordinates
	var lat, lon float64
	fmt.Sscanf(result.Lat, "%f", &lat)
	fmt.Sscanf(result.Lon, "%f", &lon)

	// Calculate distance from search center
	distance := calculateDistance(req.Lat, req.Lon, lat, lon)

	// Build address
	address := result.DisplayName
	if result.Address.Road != "" {
		address = fmt.Sprintf("%s, %s", result.Address.Road, result.Address.City)
	}

	return Place{
		ID:          fmt.Sprintf("osm-%d", result.PlaceID),
		Name:        extractPlaceName(result.DisplayName),
		Address:     address,
		Category:    req.Category,
		Distance:    distance,
		Rating:      0.0, // OSM doesn't provide ratings
		PriceLevel:  0,
		OpenNow:     true, // OSM doesn't provide real-time hours
		Description: fmt.Sprintf("From OpenStreetMap: %s", result.Type),
	}
}

func extractPlaceName(displayName string) string {
	// Take first part before comma as place name
	parts := strings.Split(displayName, ",")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return displayName
}

// MultiSourceAggregator aggregates data from multiple sources
type MultiSourceAggregator struct {
	sources []DataSource
	timeout time.Duration
}

func NewMultiSourceAggregator() *MultiSourceAggregator {
	agg := &MultiSourceAggregator{
		timeout: 15 * time.Second,
		sources: []DataSource{},
	}

	// Add all available sources
	agg.sources = append(agg.sources, NewLocalDBSource())
	agg.sources = append(agg.sources, NewOpenStreetMapSource())
	agg.sources = append(agg.sources, NewSearXNGSource())
	agg.sources = append(agg.sources, NewMockDataSource()) // Fallback

	return agg
}

// AggregateResult holds results from a single source
type AggregateResult struct {
	Source   string
	Places   []Place
	Error    error
	Duration time.Duration
}

// Aggregate fetches data from all sources concurrently
func (a *MultiSourceAggregator) Aggregate(req SearchRequest) []Place {
	results := make(chan AggregateResult, len(a.sources))
	var wg sync.WaitGroup

	// Fetch from all sources concurrently
	for _, source := range a.sources {
		wg.Add(1)
		go func(src DataSource) {
			defer wg.Done()

			start := time.Now()
			places, err := src.Fetch(req)
			duration := time.Since(start)

			results <- AggregateResult{
				Source:   src.Name(),
				Places:   places,
				Error:    err,
				Duration: duration,
			}

			if err != nil {
				mainLogger.Warn("Data source failed", map[string]interface{}{
					"source":   src.Name(),
					"error":    err.Error(),
					"duration": duration.String(),
				})
			} else {
				mainLogger.Debug("Data source completed", map[string]interface{}{
					"source":   src.Name(),
					"places":   len(places),
					"duration": duration.String(),
				})
			}
		}(source)
	}

	// Close results channel when all sources complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect all results
	var allPlaces []Place
	sourceMap := make(map[string][]Place)

	for result := range results {
		if result.Error == nil && len(result.Places) > 0 {
			sourceMap[result.Source] = result.Places
			allPlaces = append(allPlaces, result.Places...)
		}
	}

	// Deduplicate and merge results
	merged := a.deduplicateAndMerge(allPlaces)

	mainLogger.Info("Aggregation complete", map[string]interface{}{
		"total_places":  len(allPlaces),
		"sources":       len(sourceMap),
		"unique_places": len(merged),
	})

	return merged
}

// deduplicateAndMerge removes duplicates and merges similar places
func (a *MultiSourceAggregator) deduplicateAndMerge(places []Place) []Place {
	if len(places) == 0 {
		return places
	}

	// Group similar places by name and location
	groups := make(map[string][]Place)
	for _, place := range places {
		// Create a key based on normalized name and approximate location
		key := a.createPlaceKey(place)
		groups[key] = append(groups[key], place)
	}

	// Merge each group into a single place
	var merged []Place
	for _, group := range groups {
		if len(group) == 1 {
			merged = append(merged, group[0])
		} else {
			merged = append(merged, a.mergePlaces(group))
		}
	}

	return merged
}

func (a *MultiSourceAggregator) createPlaceKey(place Place) string {
	// Normalize name: lowercase, remove special chars
	name := strings.ToLower(place.Name)
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return ' '
	}, name)
	name = strings.Join(strings.Fields(name), "")

	// Round distance to nearest 0.1 miles for grouping
	distKey := fmt.Sprintf("%.1f", place.Distance)

	return fmt.Sprintf("%s-%s-%s", name, place.Category, distKey)
}

func (a *MultiSourceAggregator) mergePlaces(places []Place) Place {
	// Take the first place as base
	merged := places[0]

	// Merge information from other places
	for i := 1; i < len(places); i++ {
		place := places[i]

		// Use better rating if available
		if place.Rating > merged.Rating {
			merged.Rating = place.Rating
		}

		// Use more detailed description
		if len(place.Description) > len(merged.Description) {
			merged.Description = place.Description
		}

		// Merge photos
		if len(place.Photos) > 0 {
			merged.Photos = append(merged.Photos, place.Photos...)
		}

		// Use more precise address
		if len(place.Address) > len(merged.Address) {
			merged.Address = place.Address
		}

		// Average the distance
		merged.Distance = (merged.Distance + place.Distance) / 2
	}

	// Deduplicate photos
	photoSet := make(map[string]bool)
	var uniquePhotos []string
	for _, photo := range merged.Photos {
		if !photoSet[photo] {
			photoSet[photo] = true
			uniquePhotos = append(uniquePhotos, photo)
		}
	}
	merged.Photos = uniquePhotos

	return merged
}

// calculateDistance computes distance in miles between two lat/lon points
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Simple Euclidean approximation (good enough for short distances)
	// In production, use Haversine formula for accuracy
	latDiff := (lat2 - lat1) * 69.0 // 1 degree latitude ≈ 69 miles
	lonDiff := (lon2 - lon1) * 54.6 // 1 degree longitude ≈ 54.6 miles (at mid-latitudes)
	distance := ((latDiff * latDiff) + (lonDiff * lonDiff))

	// Simple square root approximation
	if distance > 0 {
		// Newton's method for square root (2 iterations sufficient for small distances)
		sqrt := distance / 2
		for i := 0; i < 2; i++ {
			sqrt = (sqrt + distance/sqrt) / 2
		}
		return sqrt
	}
	return 0
}
