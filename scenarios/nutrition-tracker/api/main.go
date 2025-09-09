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

	_ "github.com/lib/pq"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Meal struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	MealType        string    `json:"meal_type"`
	MealDate        string    `json:"meal_date"`
	FoodDescription string    `json:"food_description"`
	Calories        float64   `json:"calories"`
	Protein         float64   `json:"protein"`
	Carbs           float64   `json:"carbs"`
	Fat             float64   `json:"fat"`
	Fiber           float64   `json:"fiber"`
	Sugar           float64   `json:"sugar"`
	Sodium          float64   `json:"sodium"`
	CreatedAt       time.Time `json:"created_at"`
}

type Food struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Brand        string  `json:"brand"`
	ServingSize  string  `json:"serving_size"`
	Calories     float64 `json:"calories"`
	Protein      float64 `json:"protein"`
	Carbs        float64 `json:"carbs"`
	Fat          float64 `json:"fat"`
	Fiber        float64 `json:"fiber"`
	Sugar        float64 `json:"sugar"`
	Sodium       float64 `json:"sodium"`
	FoodCategory string  `json:"food_category"`
}

type NutritionGoals struct {
	UserID         string  `json:"user_id"`
	DailyCalories  int     `json:"daily_calories"`
	ProteinGrams   int     `json:"protein_grams"`
	CarbsGrams     int     `json:"carbs_grams"`
	FatGrams       int     `json:"fat_grams"`
	FiberGrams     int     `json:"fiber_grams"`
	SugarLimit     int     `json:"sugar_limit"`
	SodiumLimit    int     `json:"sodium_limit"`
	WeightGoal     string  `json:"weight_goal"`
	ActivityLevel  string  `json:"activity_level"`
}

type DailySummary struct {
	Date          string  `json:"date"`
	TotalCalories float64 `json:"total_calories"`
	TotalProtein  float64 `json:"total_protein"`
	TotalCarbs    float64 `json:"total_carbs"`
	TotalFat      float64 `json:"total_fat"`
	TotalFiber    float64 `json:"total_fiber"`
	TotalSugar    float64 `json:"total_sugar"`
	TotalSodium   float64 `json:"total_sodium"`
	MealCount     int     `json:"meal_count"`
}

var db *sql.DB

func main() {
	// Initialize database connection using orchestrator-provided environment variables
	// The orchestrator provides POSTGRES_URL directly, or individual components
	var connStr string
	postgresURL := getEnv("POSTGRES_URL", "")
	if postgresURL != "" {
		// Use the full URL provided by orchestrator
		connStr = postgresURL
	} else {
		// Fallback to individual components (Vrooli standard naming)
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

		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
	}

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	
	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("🍎 Database URL configured")
	
	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
			break
		}
		
		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay) * math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		
		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter
		
		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)
		
		// Provide detailed status every few attempts
		if attempt > 0 && attempt % 3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt + 1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt * 2) * baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}
		
		time.Sleep(actualDelay)
	}
	
	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")

	// Setup routes
	router := mux.NewRouter()

	// API routes
	router.HandleFunc("/api/health", healthCheck).Methods("GET")
	router.HandleFunc("/api/meals", getMeals).Methods("GET")
	router.HandleFunc("/api/meals", createMeal).Methods("POST")
	router.HandleFunc("/api/meals/today", getTodaysMeals).Methods("GET")
	router.HandleFunc("/api/meals/{id}", getMeal).Methods("GET")
	router.HandleFunc("/api/meals/{id}", updateMeal).Methods("PUT")
	router.HandleFunc("/api/meals/{id}", deleteMeal).Methods("DELETE")
	
	router.HandleFunc("/api/foods", getFoods).Methods("GET")
	router.HandleFunc("/api/foods/search", searchFoods).Methods("GET")
	router.HandleFunc("/api/foods", createFood).Methods("POST")
	
	router.HandleFunc("/api/nutrition", getNutritionSummary).Methods("GET")
	router.HandleFunc("/api/goals", getGoals).Methods("GET")
	router.HandleFunc("/api/goals", updateGoals).Methods("POST", "PUT")
	
	router.HandleFunc("/api/suggestions", getMealSuggestions).Methods("GET")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	// Port configuration - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		log.Fatal("❌ API_PORT or PORT environment variable is required")
	}
	
	// Start server
	log.Printf("Starting Nutrition Tracker API on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "nutrition-tracker-api",
		"version": "1.0.0",
	})
}

func getMeals(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "demo-user-123"
	}

	query := `
		SELECT m.id, m.user_id, m.meal_type, m.meal_date, m.notes,
		       mi.custom_food_name, mi.calories, mi.protein, mi.carbs, mi.fat, 
		       mi.fiber, mi.sugar, mi.sodium, m.created_at
		FROM meals m
		LEFT JOIN meal_items mi ON m.id = mi.meal_id
		WHERE m.user_id = $1
		ORDER BY m.meal_date DESC, m.created_at DESC
		LIMIT 50
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var meals []Meal
	for rows.Next() {
		var m Meal
		var notes, foodName sql.NullString
		err := rows.Scan(&m.ID, &m.UserID, &m.MealType, &m.MealDate, &notes,
			&foodName, &m.Calories, &m.Protein, &m.Carbs, &m.Fat,
			&m.Fiber, &m.Sugar, &m.Sodium, &m.CreatedAt)
		if err != nil {
			continue
		}
		if foodName.Valid {
			m.FoodDescription = foodName.String
		}
		meals = append(meals, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meals)
}

func getTodaysMeals(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "demo-user-123"
	}

	today := time.Now().Format("2006-01-02")

	query := `
		SELECT m.id, m.user_id, m.meal_type, m.meal_date,
		       COALESCE(mi.custom_food_name, f.name) as food_description,
		       COALESCE(mi.calories, 0) as calories,
		       COALESCE(mi.protein, 0) as protein,
		       COALESCE(mi.carbs, 0) as carbs,
		       COALESCE(mi.fat, 0) as fat
		FROM meals m
		LEFT JOIN meal_items mi ON m.id = mi.meal_id
		LEFT JOIN foods f ON mi.food_id = f.id
		WHERE m.user_id = $1 AND m.meal_date = $2
		ORDER BY m.created_at
	`

	rows, err := db.Query(query, userID, today)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var meals []map[string]interface{}
	for rows.Next() {
		var id, userID, mealType, mealDate, foodDesc string
		var calories, protein, carbs, fat float64

		err := rows.Scan(&id, &userID, &mealType, &mealDate, &foodDesc,
			&calories, &protein, &carbs, &fat)
		if err != nil {
			continue
		}

		meal := map[string]interface{}{
			"id":               id,
			"user_id":          userID,
			"meal_type":        mealType,
			"meal_date":        mealDate,
			"food_description": foodDesc,
			"calories":         calories,
			"protein":          protein,
			"carbs":            carbs,
			"fat":              fat,
		}
		meals = append(meals, meal)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meals)
}

func createMeal(w http.ResponseWriter, r *http.Request) {
	var meal Meal
	if err := json.NewDecoder(r.Body).Decode(&meal); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if meal.UserID == "" {
		meal.UserID = "demo-user-123"
	}
	if meal.MealDate == "" {
		meal.MealDate = time.Now().Format("2006-01-02")
	}

	// Insert meal
	var mealID string
	err := db.QueryRow(`
		INSERT INTO meals (user_id, meal_type, meal_date, meal_time)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, meal.UserID, meal.MealType, meal.MealDate, time.Now().Format("15:04:05")).Scan(&mealID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Insert meal item
	_, err = db.Exec(`
		INSERT INTO meal_items (meal_id, custom_food_name, quantity, unit, calories, protein, carbs, fat, fiber, sugar, sodium)
		VALUES ($1, $2, 1, 'serving', $3, $4, $5, $6, $7, $8, $9)
	`, mealID, meal.FoodDescription, meal.Calories, meal.Protein, meal.Carbs, meal.Fat, meal.Fiber, meal.Sugar, meal.Sodium)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	meal.ID = mealID
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(meal)
}

func getMeal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	mealID := vars["id"]

	var meal Meal
	err := db.QueryRow(`
		SELECT m.id, m.user_id, m.meal_type, m.meal_date, m.created_at
		FROM meals m
		WHERE m.id = $1
	`, mealID).Scan(&meal.ID, &meal.UserID, &meal.MealType, &meal.MealDate, &meal.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Meal not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meal)
}

func updateMeal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	mealID := vars["id"]

	var meal Meal
	if err := json.NewDecoder(r.Body).Decode(&meal); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := db.Exec(`
		UPDATE meals 
		SET meal_type = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, meal.MealType, mealID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func deleteMeal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	mealID := vars["id"]

	_, err := db.Exec("DELETE FROM meals WHERE id = $1", mealID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getFoods(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, brand, serving_size, calories, protein, carbs, fat, food_category
		FROM foods
		ORDER BY name
		LIMIT 100
	`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var foods []Food
	for rows.Next() {
		var f Food
		var brand sql.NullString
		err := rows.Scan(&f.ID, &f.Name, &brand, &f.ServingSize,
			&f.Calories, &f.Protein, &f.Carbs, &f.Fat, &f.FoodCategory)
		if err != nil {
			continue
		}
		if brand.Valid {
			f.Brand = brand.String
		}
		foods = append(foods, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(foods)
}

func searchFoods(w http.ResponseWriter, r *http.Request) {
	searchQuery := r.URL.Query().Get("q")
	if searchQuery == "" {
		http.Error(w, "Search query required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, name, brand, serving_size, calories, protein, carbs, fat, food_category
		FROM foods
		WHERE LOWER(name) LIKE LOWER($1) OR LOWER(brand) LIKE LOWER($1)
		ORDER BY name
		LIMIT 20
	`

	searchPattern := "%" + searchQuery + "%"
	rows, err := db.Query(query, searchPattern)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var foods []Food
	for rows.Next() {
		var f Food
		var brand sql.NullString
		err := rows.Scan(&f.ID, &f.Name, &brand, &f.ServingSize,
			&f.Calories, &f.Protein, &f.Carbs, &f.Fat, &f.FoodCategory)
		if err != nil {
			continue
		}
		if brand.Valid {
			f.Brand = brand.String
		}
		foods = append(foods, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(foods)
}

func createFood(w http.ResponseWriter, r *http.Request) {
	var food Food
	if err := json.NewDecoder(r.Body).Decode(&food); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var foodID string
	err := db.QueryRow(`
		INSERT INTO foods (name, brand, serving_size, serving_unit, calories, protein, carbs, fat, fiber, sugar, sodium, food_category, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'user_created')
		RETURNING id
	`, food.Name, food.Brand, food.ServingSize, "serving", food.Calories, food.Protein, food.Carbs, food.Fat, food.Fiber, food.Sugar, food.Sodium, food.FoodCategory).Scan(&foodID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	food.ID = foodID
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(food)
}

func getNutritionSummary(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	date := r.URL.Query().Get("date")
	
	if userID == "" {
		userID = "demo-user-123"
	}
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	var summary DailySummary
	err := db.QueryRow(`
		SELECT 
			COALESCE(SUM(mi.calories), 0) as total_calories,
			COALESCE(SUM(mi.protein), 0) as total_protein,
			COALESCE(SUM(mi.carbs), 0) as total_carbs,
			COALESCE(SUM(mi.fat), 0) as total_fat,
			COALESCE(SUM(mi.fiber), 0) as total_fiber,
			COALESCE(SUM(mi.sugar), 0) as total_sugar,
			COALESCE(SUM(mi.sodium), 0) as total_sodium,
			COUNT(DISTINCT m.id) as meal_count
		FROM meals m
		LEFT JOIN meal_items mi ON m.id = mi.meal_id
		WHERE m.user_id = $1 AND m.meal_date = $2
	`, userID, date).Scan(&summary.TotalCalories, &summary.TotalProtein, &summary.TotalCarbs,
		&summary.TotalFat, &summary.TotalFiber, &summary.TotalSugar, &summary.TotalSodium, &summary.MealCount)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	summary.Date = date
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func getGoals(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "demo-user-123"
	}

	var goals NutritionGoals
	err := db.QueryRow(`
		SELECT user_id, daily_calories, protein_grams, carbs_grams, fat_grams, 
		       fiber_grams, sugar_limit, sodium_limit, weight_goal, activity_level
		FROM user_goals
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&goals.UserID, &goals.DailyCalories, &goals.ProteinGrams,
		&goals.CarbsGrams, &goals.FatGrams, &goals.FiberGrams, &goals.SugarLimit,
		&goals.SodiumLimit, &goals.WeightGoal, &goals.ActivityLevel)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return default goals
			goals = NutritionGoals{
				UserID:        userID,
				DailyCalories: 2000,
				ProteinGrams:  75,
				CarbsGrams:    225,
				FatGrams:      65,
				FiberGrams:    25,
				SugarLimit:    50,
				SodiumLimit:   2300,
				WeightGoal:    "maintain",
				ActivityLevel: "moderately_active",
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(goals)
}

func updateGoals(w http.ResponseWriter, r *http.Request) {
	var goals NutritionGoals
	if err := json.NewDecoder(r.Body).Decode(&goals); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if goals.UserID == "" {
		goals.UserID = "demo-user-123"
	}

	// Upsert goals
	_, err := db.Exec(`
		INSERT INTO user_goals (user_id, daily_calories, protein_grams, carbs_grams, fat_grams, 
		                        fiber_grams, sugar_limit, sodium_limit, weight_goal, activity_level)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id) DO UPDATE
		SET daily_calories = $2, protein_grams = $3, carbs_grams = $4, fat_grams = $5,
		    fiber_grams = $6, sugar_limit = $7, sodium_limit = $8, weight_goal = $9, activity_level = $10,
		    updated_at = CURRENT_TIMESTAMP
	`, goals.UserID, goals.DailyCalories, goals.ProteinGrams, goals.CarbsGrams,
		goals.FatGrams, goals.FiberGrams, goals.SugarLimit, goals.SodiumLimit,
		goals.WeightGoal, goals.ActivityLevel)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(goals)
}

func getMealSuggestions(w http.ResponseWriter, r *http.Request) {
	// This would normally call the n8n meal-suggester workflow
	// For now, return mock suggestions
	suggestions := []map[string]interface{}{
		{
			"meal_name":           "Grilled Chicken Salad",
			"calories":            380,
			"protein":             35,
			"carbs":               20,
			"fat":                 18,
			"recommendation_reason": "High protein to meet your daily goal",
		},
		{
			"meal_name":           "Quinoa Buddha Bowl",
			"calories":            420,
			"protein":             18,
			"carbs":               55,
			"fat":                 15,
			"recommendation_reason": "Balanced macros with complex carbs",
		},
		{
			"meal_name":           "Greek Yogurt Parfait",
			"calories":            280,
			"protein":             20,
			"carbs":               35,
			"fat":                 8,
			"recommendation_reason": "Perfect post-workout snack",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"suggestions": suggestions,
	})
}

