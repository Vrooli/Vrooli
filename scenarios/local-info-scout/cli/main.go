package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type SearchRequest struct {
	Query    string  `json:"query"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Radius   float64 `json:"radius"`
	Category string  `json:"category"`
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

func main() {
	// Command line flags
	var (
		query    = flag.String("query", "", "Search query")
		lat      = flag.Float64("lat", 40.7128, "Latitude")
		lon      = flag.Float64("lon", -74.0060, "Longitude")
		radius   = flag.Float64("radius", 5.0, "Search radius in miles")
		category = flag.String("category", "", "Category filter")
		apiURL   = flag.String("api", "", "API URL (default: auto-detect from environment)")
		listCats = flag.Bool("categories", false, "List available categories")
		help     = flag.Bool("help", false, "Show help")
	)
	
	flag.Parse()
	
	if *help || (flag.NArg() == 0 && *query == "" && !*listCats) {
		fmt.Println("Local Info Scout CLI - Discover local places")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  local-info-scout [options] <search term>")
		fmt.Println("  local-info-scout --query=\"search term\" [options]")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --query       Search query text")
		fmt.Println("  --lat         Latitude (default: 40.7128)")
		fmt.Println("  --lon         Longitude (default: -74.0060)")
		fmt.Println("  --radius      Search radius in miles (default: 5.0)")
		fmt.Println("  --category    Filter by category")
		fmt.Println("  --categories  List available categories")
		fmt.Println("  --api         API URL (default: auto-detect)")
		fmt.Println("  --help        Show this help message")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  local-info-scout \"vegan restaurants\"")
		fmt.Println("  local-info-scout --query=\"coffee shops\" --radius=2")
		fmt.Println("  local-info-scout --categories")
		os.Exit(0)
	}
	
	// Determine API URL
	if *apiURL == "" {
		port := os.Getenv("API_PORT")
		if port == "" {
			// Try to detect from running scenario
			port = "18782" // fallback to known port
		}
		*apiURL = fmt.Sprintf("http://localhost:%s", port)
	}
	
	// List categories if requested
	if *listCats {
		listCategories(*apiURL)
		return
	}
	
	// Get query from args if not provided via flag
	searchQuery := *query
	if searchQuery == "" && flag.NArg() > 0 {
		searchQuery = strings.Join(flag.Args(), " ")
	}
	
	if searchQuery == "" {
		fmt.Println("Error: Please provide a search query")
		os.Exit(1)
	}
	
	// Perform search
	search(*apiURL, searchQuery, *lat, *lon, *radius, *category)
}

func listCategories(apiURL string) {
	resp, err := http.Get(apiURL + "/api/categories")
	if err != nil {
		fmt.Printf("Error fetching categories: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}
	
	var categories []string
	if err := json.Unmarshal(body, &categories); err != nil {
		fmt.Printf("Error parsing categories: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("Available categories:")
	for _, cat := range categories {
		fmt.Printf("  • %s\n", cat)
	}
}

func search(apiURL, query string, lat, lon, radius float64, category string) {
	req := SearchRequest{
		Query:    query,
		Lat:      lat,
		Lon:      lon,
		Radius:   radius,
		Category: category,
	}
	
	jsonData, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}
	
	resp, err := http.Post(apiURL+"/api/search", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error performing search: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}
	
	var places []Place
	if err := json.Unmarshal(body, &places); err != nil {
		fmt.Printf("Error parsing results: %v\n", err)
		os.Exit(1)
	}
	
	// Display results
	fmt.Printf("\n🔍 Search results for \"%s\" near (%.4f, %.4f):\n\n", query, lat, lon)
	
	if len(places) == 0 {
		fmt.Println("No places found matching your criteria.")
		return
	}
	
	for i, place := range places {
		fmt.Printf("%d. %s", i+1, place.Name)
		if place.Rating > 0 {
			fmt.Printf(" ⭐ %.1f", place.Rating)
		}
		if place.OpenNow {
			fmt.Printf(" 🟢 Open")
		} else {
			fmt.Printf(" 🔴 Closed")
		}
		fmt.Println()
		
		fmt.Printf("   📍 %s (%.1f miles)\n", place.Address, place.Distance)
		fmt.Printf("   📂 Category: %s", place.Category)
		if place.PriceLevel > 0 {
			fmt.Printf(" | Price: %s", strings.Repeat("$", place.PriceLevel))
		}
		fmt.Println()
		
		if place.Description != "" {
			fmt.Printf("   📝 %s\n", place.Description)
		}
		fmt.Println()
	}
}