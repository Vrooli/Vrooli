package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type App struct {
	DB *sql.DB
}

type List struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	OwnerID     string          `json:"owner_id,omitempty"`
	ItemCount   int             `json:"item_count"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type Item struct {
	ID              string          `json:"id"`
	ListID          string          `json:"list_id"`
	Content         json.RawMessage `json:"content"`
	EloRating       float64         `json:"elo_rating"`
	ComparisonCount int             `json:"comparison_count"`
	Wins            int             `json:"wins"`
	Losses          int             `json:"losses"`
	Confidence      float64         `json:"confidence"`
}

type Comparison struct {
	ID                 string    `json:"id"`
	ListID             string    `json:"list_id"`
	WinnerID           string    `json:"winner_id"`
	LoserID            string    `json:"loser_id"`
	WinnerRatingBefore float64   `json:"winner_rating_before"`
	LoserRatingBefore  float64   `json:"loser_rating_before"`
	WinnerRatingAfter  float64   `json:"winner_rating_after"`
	LoserRatingAfter   float64   `json:"loser_rating_after"`
	Timestamp          time.Time `json:"timestamp"`
}

type NextComparison struct {
	ItemA    *Item    `json:"item_a"`
	ItemB    *Item    `json:"item_b"`
	Progress Progress `json:"progress"`
}

type Progress struct {
	Completed int `json:"completed"`
	Total     int `json:"total"`
}

type CreateListRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Items       []ItemInput     `json:"items"`
}

type ItemInput struct {
	Content json.RawMessage `json:"content"`
}

type CreateComparisonRequest struct {
	ListID   string `json:"list_id"`
	WinnerID string `json:"winner_id"`
	LoserID  string `json:"loser_id"`
}

type Rankings struct {
	Rankings []RankedItem `json:"rankings"`
}

type RankedItem struct {
	Rank       int             `json:"rank"`
	Item       json.RawMessage `json:"item"`
	EloRating  float64         `json:"elo_rating"`
	Confidence float64         `json:"confidence"`
}

func main() {
	app := &App{}
	
	// Database connection
	dbHost := getEnv("POSTGRES_HOST", "")
	if dbHost == "" {
		log.Fatal("POSTGRES_HOST environment variable is required")
	}
	dbPort := getEnv("POSTGRES_PORT", "")
	if dbPort == "" {
		log.Fatal("POSTGRES_PORT environment variable is required")
	}
	dbUser := getEnv("POSTGRES_USER", "")
	if dbUser == "" {
		log.Fatal("POSTGRES_USER environment variable is required")
	}
	dbPassword := getEnv("POSTGRES_PASSWORD", "")
	if dbPassword == "" {
		log.Fatal("POSTGRES_PASSWORD environment variable is required")
	}
	dbName := getEnv("POSTGRES_DB", "")
	if dbName == "" {
		log.Fatal("POSTGRES_DB environment variable is required")
	}
	
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	
	var err error
	app.DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer app.DB.Close()
	
	// Test connection
	if err := app.DB.Ping(); err != nil {
		log.Fatal("Database ping failed:", err)
	}
	
	// Setup routes
	router := mux.NewRouter()
	
	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", app.HealthCheck).Methods("GET")
	api.HandleFunc("/lists", app.GetLists).Methods("GET")
	api.HandleFunc("/lists", app.CreateList).Methods("POST")
	api.HandleFunc("/lists/{id}", app.GetList).Methods("GET")
	api.HandleFunc("/lists/{id}/next-comparison", app.GetNextComparison).Methods("GET")
	api.HandleFunc("/lists/{id}/rankings", app.GetRankings).Methods("GET")
	api.HandleFunc("/comparisons", app.CreateComparison).Methods("POST")
	api.HandleFunc("/comparisons/{id}", app.DeleteComparison).Methods("DELETE")
	
	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	
	handler := c.Handler(router)
	
	// Start server
	port := getEnv("API_PORT", "30400")
	log.Printf("Elo Swipe API server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (app *App) HealthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (app *App) GetLists(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT l.id, l.name, l.description, l.owner_id, 
		       COUNT(i.id) as item_count, l.created_at, l.updated_at
		FROM elo_swipe.lists l
		LEFT JOIN elo_swipe.items i ON l.id = i.list_id
		GROUP BY l.id
		ORDER BY l.updated_at DESC
	`
	
	rows, err := app.DB.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	lists := []List{}
	for rows.Next() {
		var list List
		var ownerID sql.NullString
		err := rows.Scan(&list.ID, &list.Name, &list.Description, &ownerID,
			&list.ItemCount, &list.CreatedAt, &list.UpdatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ownerID.Valid {
			list.OwnerID = ownerID.String
		}
		lists = append(lists, list)
	}
	
	json.NewEncoder(w).Encode(lists)
}

func (app *App) CreateList(w http.ResponseWriter, r *http.Request) {
	var req CreateListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Start transaction
	tx, err := app.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	
	// Create list
	listID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO elo_swipe.lists (id, name, description)
		VALUES ($1, $2, $3)
	`, listID, req.Name, req.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Add items
	for _, item := range req.Items {
		itemID := uuid.New().String()
		_, err = tx.Exec(`
			INSERT INTO elo_swipe.items (id, list_id, content)
			VALUES ($1, $2, $3)
		`, itemID, listID, item.Content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	
	// Commit transaction
	if err = tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Return response
	response := map[string]interface{}{
		"list_id":        listID,
		"item_count":     len(req.Items),
		"comparison_url": fmt.Sprintf("/swipe?list=%s", listID),
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (app *App) GetList(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	listID := vars["id"]
	
	var list List
	err := app.DB.QueryRow(`
		SELECT id, name, description, owner_id, created_at, updated_at
		FROM elo_swipe.lists WHERE id = $1
	`, listID).Scan(&list.ID, &list.Name, &list.Description, 
		&list.OwnerID, &list.CreatedAt, &list.UpdatedAt)
	
	if err == sql.ErrNoRows {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(w).Encode(list)
}

func (app *App) GetNextComparison(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	listID := vars["id"]
	
	// Try to get next comparison from the database function
	var itemAID, itemBID string
	var priority float64
	
	err := app.DB.QueryRow(`
		SELECT item_a_id, item_b_id, priority
		FROM elo_swipe.get_next_comparison($1)
	`, listID).Scan(&itemAID, &itemBID, &priority)
	
	if err == sql.ErrNoRows {
		http.Error(w, "No more comparisons needed", http.StatusNoContent)
		return
	}
	if err != nil {
		// Fallback: get two items with lowest comparison count
		query := `
			SELECT id, content, elo_rating, comparison_count
			FROM elo_swipe.items
			WHERE list_id = $1
			ORDER BY comparison_count ASC, RANDOM()
			LIMIT 2
		`
		rows, err := app.DB.Query(query, listID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		
		items := []*Item{}
		for rows.Next() {
			var item Item
			err := rows.Scan(&item.ID, &item.Content, &item.EloRating, &item.ComparisonCount)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			items = append(items, &item)
		}
		
		if len(items) < 2 {
			http.Error(w, "Not enough items to compare", http.StatusNoContent)
			return
		}
		
		itemAID = items[0].ID
		itemBID = items[1].ID
	}
	
	// Get full item details
	itemA, err := app.getItem(itemAID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	itemB, err := app.getItem(itemBID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Get progress
	progress, err := app.getProgress(listID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := NextComparison{
		ItemA:    itemA,
		ItemB:    itemB,
		Progress: progress,
	}
	
	json.NewEncoder(w).Encode(response)
}

func (app *App) CreateComparison(w http.ResponseWriter, r *http.Request) {
	var req CreateComparisonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Get current ratings
	winnerRating, loserRating, err := app.getRatings(req.WinnerID, req.LoserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Calculate new ratings using Elo formula
	kFactor := 32.0
	expectedWinner := 1.0 / (1.0 + math.Pow(10, (loserRating-winnerRating)/400.0))
	expectedLoser := 1.0 - expectedWinner
	
	newWinnerRating := winnerRating + kFactor*(1.0-expectedWinner)
	newLoserRating := loserRating + kFactor*(0.0-expectedLoser)
	
	// Start transaction
	tx, err := app.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	
	// Record comparison
	comparisonID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO elo_swipe.comparisons 
		(id, list_id, winner_id, loser_id, winner_rating_before, loser_rating_before,
		 winner_rating_after, loser_rating_after, k_factor)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, comparisonID, req.ListID, req.WinnerID, req.LoserID,
		winnerRating, loserRating, newWinnerRating, newLoserRating, kFactor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Update winner
	_, err = tx.Exec(`
		UPDATE elo_swipe.items
		SET elo_rating = $1,
		    comparison_count = comparison_count + 1,
		    wins = wins + 1,
		    confidence_score = LEAST(1.0, (comparison_count + 1) * 0.1)
		WHERE id = $2
	`, newWinnerRating, req.WinnerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Update loser
	_, err = tx.Exec(`
		UPDATE elo_swipe.items
		SET elo_rating = $1,
		    comparison_count = comparison_count + 1,
		    losses = losses + 1,
		    confidence_score = LEAST(1.0, (comparison_count + 1) * 0.1)
		WHERE id = $2
	`, newLoserRating, req.LoserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Commit transaction
	if err = tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Return comparison details
	comparison := Comparison{
		ID:                 comparisonID,
		ListID:             req.ListID,
		WinnerID:           req.WinnerID,
		LoserID:            req.LoserID,
		WinnerRatingBefore: winnerRating,
		LoserRatingBefore:  loserRating,
		WinnerRatingAfter:  newWinnerRating,
		LoserRatingAfter:   newLoserRating,
		Timestamp:          time.Now(),
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comparison)
}

func (app *App) DeleteComparison(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	comparisonID := vars["id"]
	
	// TODO: Implement undo logic (revert ratings)
	// For now, just mark as deleted or remove
	
	w.WriteHeader(http.StatusNoContent)
}

func (app *App) GetRankings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	listID := vars["id"]
	
	query := `
		SELECT content, elo_rating, confidence_score
		FROM elo_swipe.items
		WHERE list_id = $1
		ORDER BY elo_rating DESC
	`
	
	rows, err := app.DB.Query(query, listID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	rankings := []RankedItem{}
	rank := 1
	for rows.Next() {
		var item RankedItem
		err := rows.Scan(&item.Item, &item.EloRating, &item.Confidence)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		item.Rank = rank
		rankings = append(rankings, item)
		rank++
	}
	
	response := Rankings{
		Rankings: rankings,
	}
	
	json.NewEncoder(w).Encode(response)
}

// Helper functions
func (app *App) getItem(itemID string) (*Item, error) {
	var item Item
	err := app.DB.QueryRow(`
		SELECT id, list_id, content, elo_rating, comparison_count, wins, losses, confidence_score
		FROM elo_swipe.items
		WHERE id = $1
	`, itemID).Scan(&item.ID, &item.ListID, &item.Content, &item.EloRating,
		&item.ComparisonCount, &item.Wins, &item.Losses, &item.Confidence)
	
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (app *App) getRatings(id1, id2 string) (float64, float64, error) {
	var rating1, rating2 float64
	
	err := app.DB.QueryRow(`
		SELECT elo_rating FROM elo_swipe.items WHERE id = $1
	`, id1).Scan(&rating1)
	if err != nil {
		return 0, 0, err
	}
	
	err = app.DB.QueryRow(`
		SELECT elo_rating FROM elo_swipe.items WHERE id = $1
	`, id2).Scan(&rating2)
	if err != nil {
		return 0, 0, err
	}
	
	return rating1, rating2, nil
}

func (app *App) getProgress(listID string) (Progress, error) {
	var itemCount, comparisonCount int
	
	err := app.DB.QueryRow(`
		SELECT COUNT(*) FROM elo_swipe.items WHERE list_id = $1
	`, listID).Scan(&itemCount)
	if err != nil {
		return Progress{}, err
	}
	
	err = app.DB.QueryRow(`
		SELECT COUNT(*) FROM elo_swipe.comparisons WHERE list_id = $1
	`, listID).Scan(&comparisonCount)
	if err != nil {
		return Progress{}, err
	}
	
	// Total possible comparisons is n*(n-1)/2
	totalPossible := itemCount * (itemCount - 1) / 2
	
	// But we don't need all comparisons for good rankings
	// Use n*log(n) as a reasonable target
	recommendedComparisons := int(float64(itemCount) * math.Log(float64(itemCount+1)) * 2)
	
	return Progress{
		Completed: comparisonCount,
		Total:     recommendedComparisons,
	}, nil
}