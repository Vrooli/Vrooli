package main

import (
	"fmt"
	"net/http"
	"os"
)

// Test-only helpers moved from production files by the private-function debt lens.
// getPopularSearches returns the most popular recent searches
func getPopularSearches() []string {
	if db == nil {
		return []string{}
	}

	query := `
    SELECT query, COUNT(*) as count
    FROM lis_search_logs
    WHERE query IS NOT NULL AND query != ''
        AND created_at > NOW() - INTERVAL '24 hours'
    GROUP BY query
    ORDER BY count DESC
    LIMIT 10
    `

	rows, err := db.Query(query)
	if err != nil {
		dbLogger.Error("Failed to get popular searches", map[string]interface{}{
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
