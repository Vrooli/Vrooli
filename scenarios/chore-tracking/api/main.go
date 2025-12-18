package main

import (
	"github.com/vrooli/api-core/preflight"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type Chore struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Points      int        `json:"points"`
	Difficulty  string     `json:"difficulty"`
	Category    string     `json:"category"`
	Frequency   string     `json:"frequency"`
	AssignedTo  *int       `json:"assigned_to,omitempty"`
	Status      string     `json:"status"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type User struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Avatar        string    `json:"avatar"`
	Level         int       `json:"level"`
	TotalPoints   int       `json:"total_points"`
	CurrentStreak int       `json:"current_streak"`
	LongestStreak int       `json:"longest_streak"`
	CreatedAt     time.Time `json:"created_at"`
}

type Achievement struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Icon        string     `json:"icon"`
	Points      int        `json:"points"`
	Criteria    string     `json:"criteria"`
	UnlockedAt  *time.Time `json:"unlocked_at,omitempty"`
}

type Reward struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Cost        int        `json:"cost"`
	Icon        string     `json:"icon"`
	Available   bool       `json:"available"`
	RedeemedAt  *time.Time `json:"redeemed_at,omitempty"`
}

var db *sql.DB
var choreProcessor *ChoreProcessor

func initDB() {
	// Database configuration - support both POSTGRES_URL and individual components
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	var err error
	db, err = sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatal("Failed to open database connection:", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📆 Database URL configured")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * rand.Float64())
		actualDelay := delay + jitter

		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt+1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")
}

// getEnv removed to prevent hardcoded defaults

// Chore handlers
func getChores(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	userID := r.URL.Query().Get("user_id")

	query := `SELECT id, title, description, points, difficulty, category, 
              frequency, assigned_to, status, due_date, completed_at, created_at 
              FROM chores WHERE 1=1`

	args := []interface{}{}
	argCount := 0

	if status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	if userID != "" {
		argCount++
		query += fmt.Sprintf(" AND assigned_to = $%d", argCount)
		args = append(args, userID)
	}

	query += " ORDER BY due_date ASC, created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	chores := []Chore{}
	for rows.Next() {
		var c Chore
		err := rows.Scan(&c.ID, &c.Title, &c.Description, &c.Points, &c.Difficulty,
			&c.Category, &c.Frequency, &c.AssignedTo, &c.Status, &c.DueDate,
			&c.CompletedAt, &c.CreatedAt)
		if err != nil {
			continue
		}
		chores = append(chores, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chores)
}

func createChore(w http.ResponseWriter, r *http.Request) {
	var chore Chore
	if err := json.NewDecoder(r.Body).Decode(&chore); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `INSERT INTO chores (title, description, points, difficulty, category, 
              frequency, assigned_to, status, due_date) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id, created_at`

	err := db.QueryRow(query, chore.Title, chore.Description, chore.Points,
		chore.Difficulty, chore.Category, chore.Frequency, chore.AssignedTo,
		"pending", chore.DueDate).Scan(&chore.ID, &chore.CreatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	chore.Status = "pending"

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(chore)
}

func completeChore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	choreID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid chore ID", http.StatusBadRequest)
		return
	}

	var userID int
	if err := json.NewDecoder(r.Body).Decode(&struct {
		UserID int `json:"user_id"`
	}{UserID: userID}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Mark chore as completed
	completedAt := time.Now()
	query := `UPDATE chores SET status = 'completed', completed_at = $1 
              WHERE id = $2 RETURNING points`

	var points int
	err = db.QueryRow(query, completedAt, choreID).Scan(&points)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update user points and streak
	updateUserQuery := `UPDATE users SET total_points = total_points + $1,
                        current_streak = current_streak + 1 WHERE id = $2`
	_, err = db.Exec(updateUserQuery, points, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"points_earned": points,
		"completed_at":  completedAt,
	})
}

// User handlers
func getUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, name, avatar, level, total_points, 
                           current_streak, longest_streak, created_at 
                           FROM users ORDER BY total_points DESC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.Name, &u.Avatar, &u.Level, &u.TotalPoints,
			&u.CurrentStreak, &u.LongestStreak, &u.CreatedAt)
		if err != nil {
			continue
		}
		users = append(users, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user User
	query := `SELECT id, name, avatar, level, total_points, current_streak, 
              longest_streak, created_at FROM users WHERE id = $1`

	err = db.QueryRow(query, userID).Scan(&user.ID, &user.Name, &user.Avatar,
		&user.Level, &user.TotalPoints, &user.CurrentStreak, &user.LongestStreak,
		&user.CreatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// Achievement handlers
func getAchievements(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")

	query := `SELECT a.id, a.name, a.description, a.icon, a.points, a.criteria,
              ua.unlocked_at FROM achievements a
              LEFT JOIN user_achievements ua ON a.id = ua.achievement_id`

	args := []interface{}{}
	if userID != "" {
		query += " AND ua.user_id = $1"
		args = append(args, userID)
	}

	query += " ORDER BY a.points ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	achievements := []Achievement{}
	for rows.Next() {
		var a Achievement
		err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.Icon, &a.Points,
			&a.Criteria, &a.UnlockedAt)
		if err != nil {
			continue
		}
		achievements = append(achievements, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(achievements)
}

// Reward handlers
func getRewards(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, name, description, cost, icon, available 
                          FROM rewards ORDER BY cost ASC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	rewards := []Reward{}
	for rows.Next() {
		var r Reward
		err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Cost, &r.Icon, &r.Available)
		if err != nil {
			continue
		}
		rewards = append(rewards, r)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rewards)
}

func redeemReward(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rewardID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid reward ID", http.StatusBadRequest)
		return
	}

	var request struct {
		UserID int `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check reward cost and user points
	var cost, userPoints int
	err = db.QueryRow("SELECT cost FROM rewards WHERE id = $1", rewardID).Scan(&cost)
	if err != nil {
		http.Error(w, "Reward not found", http.StatusNotFound)
		return
	}

	err = db.QueryRow("SELECT total_points FROM users WHERE id = $1", request.UserID).Scan(&userPoints)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if userPoints < cost {
		http.Error(w, "Insufficient points", http.StatusBadRequest)
		return
	}

	// Deduct points and record redemption
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE users SET total_points = total_points - $1 WHERE id = $2",
		cost, request.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`INSERT INTO user_rewards (user_id, reward_id, redeemed_at) 
                      VALUES ($1, $2, $3)`, request.UserID, rewardID, time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          true,
		"points_remaining": userPoints - cost,
	})
}

// Health check
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"service":   "chore-tracking-api",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// generateScheduleHandler generates a weekly schedule for a user
func generateScheduleHandler(w http.ResponseWriter, r *http.Request) {
	var req ScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	assignments, err := choreProcessor.GenerateWeeklySchedule(ctx, req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate schedule: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"assignments": assignments,
	})
}

// calculatePointsHandler calculates points for a completed chore
func calculatePointsHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID  int `json:"user_id"`
		ChoreID int `json:"chore_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	points, err := choreProcessor.CalculatePoints(ctx, req.UserID, req.ChoreID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to calculate points: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"points_awarded": points,
	})
}

// processAchievementsHandler processes achievements for a user
func processAchievementsHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID int `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	achievements, err := choreProcessor.ProcessAchievements(ctx, req.UserID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process achievements: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          true,
		"new_achievements": achievements,
	})
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "chore-tracking",
	}) {
		return // Process was re-exec'd after rebuild
	}

	initDB()
	defer db.Close()

	// Initialize chore processor
	choreProcessor = NewChoreProcessor(db)

	router := mux.NewRouter()

	// API routes
	router.HandleFunc("/health", healthCheck).Methods("GET")

	// Chore routes
	router.HandleFunc("/api/chores", getChores).Methods("GET")
	router.HandleFunc("/api/chores", createChore).Methods("POST")
	router.HandleFunc("/api/chores/{id}/complete", completeChore).Methods("POST")

	// User routes
	router.HandleFunc("/api/users", getUsers).Methods("GET")
	router.HandleFunc("/api/users/{id}", getUser).Methods("GET")

	// Achievement routes
	router.HandleFunc("/api/achievements", getAchievements).Methods("GET")

	// Reward routes
	router.HandleFunc("/api/rewards", getRewards).Methods("GET")
	router.HandleFunc("/api/rewards/{id}/redeem", redeemReward).Methods("POST")

	// Processor routes (replacing n8n workflows)
	router.HandleFunc("/api/schedule/generate", generateScheduleHandler).Methods("POST")
	router.HandleFunc("/api/points/calculate", calculatePointsHandler).Methods("POST")
	router.HandleFunc("/api/achievements/process", processAchievementsHandler).Methods("POST")

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	log.Printf("🎮 ChoreQuest API Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
