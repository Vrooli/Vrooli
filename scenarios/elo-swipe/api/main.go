package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type App struct {
	DB           *sql.DB
	SmartPairing *SmartPairing
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
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Items       []ItemInput `json:"items"`
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
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "elo-swipe",
	}) {
		return // Process was re-exec'd after rebuild
	}

	app := &App{}

	// Connect to database using api-core with automatic retry
	var err error
	app.DB, err = database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	// Set connection pool settings
	app.DB.SetMaxOpenConns(25)
	app.DB.SetMaxIdleConns(5)
	app.DB.SetConnMaxLifetime(5 * time.Minute)

	log.Println("🎉 Database connection pool established successfully!")

	// Initialize SmartPairing
	app.SmartPairing = NewSmartPairing(app.DB)

	// Setup routes
	router := mux.NewRouter()

	// Root-level health endpoint for ecosystem monitoring
	router.HandleFunc("/health", app.HealthCheck).Methods("GET")

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

	// Smart Pairing routes
	api.HandleFunc("/lists/{id}/smart-pairing", app.GenerateSmartPairing).Methods("POST")
	api.HandleFunc("/lists/{id}/smart-pairing/queue", app.GetPairingQueue).Methods("GET")
	api.HandleFunc("/lists/{id}/smart-pairing/refresh", app.RefreshPairingQueue).Methods("POST")

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	log.Printf("Elo Swipe API server starting")
	if err := server.Run(server.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error {
			if app.DB != nil {
				return app.DB.Close()
			}
			return nil
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// getEnv removed to prevent hardcoded defaults

func (app *App) HealthCheck(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	dbConnected := true
	var dbLatency float64
	dbStart := time.Now()
	err := app.DB.Ping()
	if err == nil {
		dbLatency = float64(time.Since(dbStart).Milliseconds())
	} else {
		dbConnected = false
	}

	// Build health response per schema
	status := "healthy"
	if !dbConnected {
		status = "degraded"
	}

	healthResponse := map[string]interface{}{
		"status":    status,
		"service":   "elo-swipe-api",
		"timestamp": time.Now().Format(time.RFC3339),
		"readiness": dbConnected,
		"version":   "1.0.0",
		"dependencies": map[string]interface{}{
			"database": map[string]interface{}{
				"connected":  dbConnected,
				"latency_ms": dbLatency,
				"error":      nil,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthResponse)
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
	var ownerID sql.NullString
	err := app.DB.QueryRow(`
		SELECT id, name, description, owner_id, created_at, updated_at
		FROM elo_swipe.lists WHERE id = $1
	`, listID).Scan(&list.ID, &list.Name, &list.Description,
		&ownerID, &list.CreatedAt, &list.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "List not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if ownerID.Valid {
		list.OwnerID = ownerID.String
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

	// Start transaction to ensure atomicity
	tx, err := app.DB.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Get comparison details before deletion
	var winnerID, loserID string
	var winnerBefore, loserBefore float64
	err = tx.QueryRow(`
		SELECT winner_id, loser_id, winner_rating_before, loser_rating_before
		FROM elo_swipe.comparisons
		WHERE id = $1
	`, comparisonID).Scan(&winnerID, &loserID, &winnerBefore, &loserBefore)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Comparison not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve comparison", http.StatusInternalServerError)
		}
		return
	}

	// Revert winner rating
	_, err = tx.Exec(`
		UPDATE elo_swipe.items
		SET elo_rating = $1,
		    comparison_count = GREATEST(comparison_count - 1, 0),
		    wins = GREATEST(wins - 1, 0),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, winnerBefore, winnerID)
	if err != nil {
		http.Error(w, "Failed to revert winner rating", http.StatusInternalServerError)
		return
	}

	// Revert loser rating
	_, err = tx.Exec(`
		UPDATE elo_swipe.items
		SET elo_rating = $1,
		    comparison_count = GREATEST(comparison_count - 1, 0),
		    losses = GREATEST(losses - 1, 0),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, loserBefore, loserID)
	if err != nil {
		http.Error(w, "Failed to revert loser rating", http.StatusInternalServerError)
		return
	}

	// Delete the comparison record
	_, err = tx.Exec(`
		DELETE FROM elo_swipe.comparisons
		WHERE id = $1
	`, comparisonID)
	if err != nil {
		http.Error(w, "Failed to delete comparison", http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (app *App) GetRankings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	listID := vars["id"]
	format := r.URL.Query().Get("format")

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

	// Handle different formats
	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"rankings_%s.csv\"", listID))
		writer := csv.NewWriter(w)
		defer writer.Flush()

		// Write header
		writer.Write([]string{"Rank", "Item", "Elo Rating", "Confidence"})

		// Write data
		for _, item := range rankings {
			itemJSON, _ := json.Marshal(item.Item)
			writer.Write([]string{
				fmt.Sprintf("%d", item.Rank),
				string(itemJSON),
				fmt.Sprintf("%.2f", item.EloRating),
				fmt.Sprintf("%.2f", item.Confidence),
			})
		}

	case "json", "":
		response := Rankings{
			Rankings: rankings,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Unsupported format. Use ?format=json or ?format=csv", http.StatusBadRequest)
	}
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
	// totalPossible := itemCount * (itemCount - 1) / 2

	// But we don't need all comparisons for good rankings
	// Use n*log(n) as a reasonable target
	recommendedComparisons := int(float64(itemCount) * math.Log(float64(itemCount+1)) * 2)

	return Progress{
		Completed: comparisonCount,
		Total:     recommendedComparisons,
	}, nil
}

// Smart Pairing handlers
func (app *App) GenerateSmartPairing(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	listID := vars["id"]

	var req PairingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Try to get items automatically if no body provided
		items, err := app.SmartPairing.getListItems(listID)
		if err != nil {
			http.Error(w, "Failed to get list items", http.StatusInternalServerError)
			return
		}
		req = PairingRequest{
			ListID: listID,
			Items:  items,
		}
	}

	// Ensure listID matches URL parameter
	req.ListID = listID

	response, err := app.SmartPairing.GenerateSmartPairs(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (app *App) GetPairingQueue(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	listID := vars["id"]

	pairs, err := app.SmartPairing.GetQueuedPairs(r.Context(), listID, 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"list_id":     listID,
		"queue_count": len(pairs),
		"pairs":       pairs,
	}

	json.NewEncoder(w).Encode(response)
}

func (app *App) RefreshPairingQueue(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	listID := vars["id"]

	response, err := app.SmartPairing.RefreshSmartPairs(r.Context(), listID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response)
}
// Test change
