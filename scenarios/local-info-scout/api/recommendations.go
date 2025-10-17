package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// RecommendationEngine provides personalized recommendations based on user history
type RecommendationEngine struct {
	db *sql.DB
}

func NewRecommendationEngine(database *sql.DB) *RecommendationEngine {
	return &RecommendationEngine{db: database}
}

// UserProfile represents a user's preferences and history
type UserProfile struct {
	UserID             string
	FavoriteCategories []string
	HiddenCategories   []string
	RecentSearches     []SearchHistory
	SavedPlaces        []Place
	DefaultRadius      float64
}

type SearchHistory struct {
	Query        string
	Category     string
	Lat          float64
	Lon          float64
	Radius       float64
	ResultsCount int
	Timestamp    time.Time
}

// GetUserProfile retrieves a user's complete profile
func (re *RecommendationEngine) GetUserProfile(userID string) (*UserProfile, error) {
	if re.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	profile := &UserProfile{
		UserID:        userID,
		DefaultRadius: 2.0, // Default
	}

	// Get user preferences
	var favCategories, hiddenCategories sql.NullString
	err := re.db.QueryRow(`
		SELECT default_radius, favorite_categories, hidden_categories
		FROM lis_user_preferences
		WHERE user_id = $1
	`, userID).Scan(&profile.DefaultRadius, &favCategories, &hiddenCategories)

	if err != nil && err != sql.ErrNoRows {
		recLogger.Error("Error fetching user preferences", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
	}

	// Parse category arrays (PostgreSQL text arrays come as {cat1,cat2})
	if favCategories.Valid {
		profile.FavoriteCategories = parsePostgresArray(favCategories.String)
	}
	if hiddenCategories.Valid {
		profile.HiddenCategories = parsePostgresArray(hiddenCategories.String)
	}

	// Get recent search history (last 30 searches)
	profile.RecentSearches = re.getRecentSearches(userID, 30)

	// Get saved places (last 50)
	profile.SavedPlaces = re.getSavedPlaces(userID, 50)

	return profile, nil
}

// parsePostgresArray converts PostgreSQL array format {a,b,c} to Go slice
func parsePostgresArray(s string) []string {
	s = strings.Trim(s, "{}")
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}

// getRecentSearches retrieves user's recent search history
func (re *RecommendationEngine) getRecentSearches(userID string, limit int) []SearchHistory {
	if re.db == nil {
		return []SearchHistory{}
	}

	rows, err := re.db.Query(`
		SELECT query, category, lat, lon, radius, results_count, created_at
		FROM lis_search_history
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)

	if err != nil {
		recLogger.Error("Error fetching search history", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		return []SearchHistory{}
	}
	defer rows.Close()

	var history []SearchHistory
	for rows.Next() {
		var h SearchHistory
		var category sql.NullString
		var lat, lon, radius sql.NullFloat64

		err := rows.Scan(&h.Query, &category, &lat, &lon, &radius, &h.ResultsCount, &h.Timestamp)
		if err != nil {
			recLogger.Error("Error scanning search history", map[string]interface{}{
				"user_id": userID,
				"error":   err.Error(),
			})
			continue
		}

		if category.Valid {
			h.Category = category.String
		}
		if lat.Valid {
			h.Lat = lat.Float64
		}
		if lon.Valid {
			h.Lon = lon.Float64
		}
		if radius.Valid {
			h.Radius = radius.Float64
		}

		history = append(history, h)
	}

	return history
}

// getSavedPlaces retrieves user's saved places
func (re *RecommendationEngine) getSavedPlaces(userID string, limit int) []Place {
	if re.db == nil {
		return []Place{}
	}

	rows, err := re.db.Query(`
		SELECT place_id, name, address, category, rating
		FROM lis_saved_places
		WHERE user_id = $1
		ORDER BY saved_at DESC
		LIMIT $2
	`, userID, limit)

	if err != nil {
		recLogger.Error("Error fetching saved places", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		return []Place{}
	}
	defer rows.Close()

	var places []Place
	for rows.Next() {
		var p Place
		var address, category sql.NullString
		var rating sql.NullFloat64

		err := rows.Scan(&p.ID, &p.Name, &address, &category, &rating)
		if err != nil {
			recLogger.Error("Error scanning saved place", map[string]interface{}{
				"user_id": userID,
				"error":   err.Error(),
			})
			continue
		}

		if address.Valid {
			p.Address = address.String
		}
		if category.Valid {
			p.Category = category.String
		}
		if rating.Valid {
			p.Rating = rating.Float64
		}

		places = append(places, p)
	}

	return places
}

// GenerateRecommendations creates personalized recommendations for a user
func (re *RecommendationEngine) GenerateRecommendations(userID string, req SearchRequest) []Place {
	profile, err := re.GetUserProfile(userID)
	if err != nil {
		recLogger.Error("Error getting user profile", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
		return []Place{}
	}

	// Use user's default radius if not specified
	if req.Radius == 0 {
		req.Radius = profile.DefaultRadius
	}

	// Start with base search results
	var allPlaces []Place
	if aggregator != nil {
		allPlaces = aggregator.Aggregate(req)
	}

	// If no results, use mock data
	if len(allPlaces) == 0 {
		allPlaces = getMockPlaces()
	}

	// Score and rank places based on user preferences
	scoredPlaces := re.scorePlaces(allPlaces, profile, req)

	// Filter out hidden categories
	filteredPlaces := re.filterHiddenCategories(scoredPlaces, profile.HiddenCategories)

	// Apply smart filters
	finalPlaces := applySmartFilters(filteredPlaces, req)

	// Limit results
	if len(finalPlaces) > 20 {
		finalPlaces = finalPlaces[:20]
	}

	return finalPlaces
}

// scorePlaces assigns relevance scores to places based on user profile
func (re *RecommendationEngine) scorePlaces(places []Place, profile *UserProfile, req SearchRequest) []Place {
	for i := range places {
		score := 100.0 // Base score

		// Boost favorite categories
		for _, favCat := range profile.FavoriteCategories {
			if places[i].Category == favCat {
				score += 30.0
				break
			}
		}

		// Boost categories from recent searches
		categoryFrequency := re.calculateCategoryFrequency(profile.RecentSearches)
		if freq, exists := categoryFrequency[places[i].Category]; exists {
			score += float64(freq) * 5.0 // 5 points per occurrence
		}

		// Boost places similar to saved places
		for _, saved := range profile.SavedPlaces {
			if places[i].Category == saved.Category {
				score += 10.0
				break
			}
		}

		// Boost by rating
		score += places[i].Rating * 10.0

		// Penalize by distance
		score -= places[i].Distance * 5.0

		// Store score in a temporary field (in production, add Score field to Place)
		// For now, we adjust the rating to reflect the score
		places[i].Rating = score / 100.0 // Normalize back to rating scale
	}

	// Sort by adjusted rating (which now includes personalization)
	sortByRelevance(places)

	return places
}

// calculateCategoryFrequency counts how often each category appears in searches
func (re *RecommendationEngine) calculateCategoryFrequency(searches []SearchHistory) map[string]int {
	freq := make(map[string]int)
	for _, search := range searches {
		if search.Category != "" {
			freq[search.Category]++
		}
	}
	return freq
}

// filterHiddenCategories removes places from hidden categories
func (re *RecommendationEngine) filterHiddenCategories(places []Place, hidden []string) []Place {
	if len(hidden) == 0 {
		return places
	}

	hiddenMap := make(map[string]bool)
	for _, cat := range hidden {
		hiddenMap[cat] = true
	}

	var filtered []Place
	for _, place := range places {
		if !hiddenMap[place.Category] {
			filtered = append(filtered, place)
		}
	}

	return filtered
}

// SavePlace saves a place to user's favorites
func (re *RecommendationEngine) SavePlace(userID string, place Place) error {
	if re.db == nil {
		return fmt.Errorf("database not connected")
	}

	_, err := re.db.Exec(`
		INSERT INTO lis_saved_places (user_id, place_id, name, address, category, rating)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, place_id) DO NOTHING
	`, userID, place.ID, place.Name, place.Address, place.Category, place.Rating)

	return err
}

// UpdateUserPreferences updates user preferences
func (re *RecommendationEngine) UpdateUserPreferences(userID string, favoriteCategories, hiddenCategories []string, defaultRadius float64) error {
	if re.db == nil {
		return fmt.Errorf("database not connected")
	}

	// Convert slices to PostgreSQL array format
	favCatStr := "{" + strings.Join(favoriteCategories, ",") + "}"
	hiddenCatStr := "{" + strings.Join(hiddenCategories, ",") + "}"

	_, err := re.db.Exec(`
		INSERT INTO lis_user_preferences (user_id, favorite_categories, hidden_categories, default_radius)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
			favorite_categories = $2,
			hidden_categories = $3,
			default_radius = $4,
			updated_at = CURRENT_TIMESTAMP
	`, userID, favCatStr, hiddenCatStr, defaultRadius)

	return err
}

// GetPopularPlaces returns popular places based on how often they're saved
func (re *RecommendationEngine) GetPopularPlaces(category string, limit int) []Place {
	if re.db == nil {
		return []Place{}
	}

	query := `
		SELECT place_id, name, category, COUNT(*) as save_count, AVG(rating) as avg_rating
		FROM lis_saved_places
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if category != "" {
		query += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, category)
		argIndex++
	}

	query += fmt.Sprintf(" GROUP BY place_id, name, category ORDER BY save_count DESC, avg_rating DESC LIMIT $%d", argIndex)
	args = append(args, limit)

	rows, err := re.db.Query(query, args...)
	if err != nil {
		recLogger.Error("Error fetching popular places", map[string]interface{}{
			"category": category,
			"error":    err.Error(),
		})
		return []Place{}
	}
	defer rows.Close()

	var places []Place
	for rows.Next() {
		var place Place
		var saveCount int
		err := rows.Scan(&place.ID, &place.Name, &place.Category, &saveCount, &place.Rating)
		if err != nil {
			recLogger.Error("Error scanning popular place", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}
		place.Description = fmt.Sprintf("Saved by %d users", saveCount)
		places = append(places, place)
	}

	return places
}

// GetTrendingSearches returns trending search queries
func (re *RecommendationEngine) GetTrendingSearches(limit int) []string {
	if re.db == nil {
		return []string{}
	}

	rows, err := re.db.Query(`
		SELECT query, COUNT(*) as search_count
		FROM lis_search_history
		WHERE created_at > NOW() - INTERVAL '7 days'
		GROUP BY query
		ORDER BY search_count DESC
		LIMIT $1
	`, limit)

	if err != nil {
		recLogger.Error("Error fetching trending searches", map[string]interface{}{
			"error": err.Error(),
		})
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
