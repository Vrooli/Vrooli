package main

import (
	"github.com/vrooli/api-core/preflight"
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

	"github.com/lib/pq"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Contact struct {
	ID               int      `json:"id"`
	Name             string   `json:"name"`
	Nickname         string   `json:"nickname"`
	RelationshipType string   `json:"relationship_type"`
	Birthday         string   `json:"birthday"`
	Anniversary      string   `json:"anniversary"`
	Email            string   `json:"email"`
	Phone            string   `json:"phone"`
	Address          string   `json:"address"`
	Notes            string   `json:"notes"`
	Interests        []string `json:"interests"`
	Tags             []string `json:"tags"`
	FavoriteColor    string   `json:"favorite_color"`
	ClothingSize     string   `json:"clothing_size"`
	PhotoURL         string   `json:"photo_url"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type Interaction struct {
	ID              int    `json:"id"`
	ContactID       int    `json:"contact_id"`
	InteractionType string `json:"interaction_type"`
	InteractionDate string `json:"interaction_date"`
	Description     string `json:"description"`
	Sentiment       string `json:"sentiment"`
	CreatedAt       string `json:"created_at"`
}

type Gift struct {
	ID           int     `json:"id"`
	ContactID    int     `json:"contact_id"`
	GiftName     string  `json:"gift_name"`
	Description  string  `json:"description"`
	Price        float64 `json:"price"`
	Occasion     string  `json:"occasion"`
	GivenDate    string  `json:"given_date"`
	Status       string  `json:"status"`
	Rating       int     `json:"rating"`
	Notes        string  `json:"notes"`
	PurchaseLink string  `json:"purchase_link"`
	CreatedAt    string  `json:"created_at"`
}

type Reminder struct {
	ID           int    `json:"id"`
	ContactID    int    `json:"contact_id"`
	ReminderType string `json:"reminder_type"`
	ReminderDate string `json:"reminder_date"`
	ReminderTime string `json:"reminder_time"`
	Message      string `json:"message"`
	Sent         bool   `json:"sent"`
	CreatedAt    string `json:"created_at"`
}

var db *sql.DB
var relationshipProcessor *RelationshipProcessor

func initDB() {
	// ALL database configuration MUST come from environment - no defaults
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	// Validate required environment variables
	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		log.Fatal("❌ Missing required database configuration. Please set: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to open database connection:", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	
	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📊 Connecting to: %s:%s/%s as user %s", dbHost, dbPort, dbName, dbUser)
	
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
		
		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * rand.Float64())
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
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func getContactsHandler(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, name, nickname, relationship_type, birthday, email, phone, notes, tags 
			  FROM contacts ORDER BY name`
	
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var c Contact
		var tags sql.NullString
		err := rows.Scan(&c.ID, &c.Name, &c.Nickname, &c.RelationshipType, 
						&c.Birthday, &c.Email, &c.Phone, &c.Notes, &tags)
		if err != nil {
			continue
		}
		contacts = append(contacts, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contacts)
}

func getContactHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var c Contact
	query := `SELECT id, name, nickname, relationship_type, birthday, anniversary, 
			  email, phone, address, notes, interests, tags, favorite_color, 
			  clothing_size, photo_url, created_at, updated_at 
			  FROM contacts WHERE id = $1`
	
	err := db.QueryRow(query, id).Scan(
		&c.ID, &c.Name, &c.Nickname, &c.RelationshipType, &c.Birthday, &c.Anniversary,
		&c.Email, &c.Phone, &c.Address, &c.Notes, pq.Array(&c.Interests), 
		pq.Array(&c.Tags), &c.FavoriteColor, &c.ClothingSize, &c.PhotoURL, 
		&c.CreatedAt, &c.UpdatedAt)
	
	if err != nil {
		http.Error(w, "Contact not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

func createContactHandler(w http.ResponseWriter, r *http.Request) {
	var c Contact
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `INSERT INTO contacts (name, nickname, relationship_type, birthday, email, phone, notes, interests, tags) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	
	err := db.QueryRow(query, c.Name, c.Nickname, c.RelationshipType, c.Birthday, 
					   c.Email, c.Phone, c.Notes, pq.Array(c.Interests), pq.Array(c.Tags)).Scan(&c.ID)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

func updateContactHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var c Contact
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `UPDATE contacts SET name=$1, nickname=$2, relationship_type=$3, birthday=$4, 
			  email=$5, phone=$6, notes=$7, interests=$8, tags=$9, updated_at=CURRENT_TIMESTAMP 
			  WHERE id=$10`
	
	_, err := db.Exec(query, c.Name, c.Nickname, c.RelationshipType, c.Birthday,
					  c.Email, c.Phone, c.Notes, pq.Array(c.Interests), pq.Array(c.Tags), id)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	c.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

func deleteContactHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	_, err := db.Exec("DELETE FROM contacts WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getInteractionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	contactID, _ := strconv.Atoi(vars["contactId"])

	query := `SELECT id, contact_id, interaction_type, interaction_date, description, sentiment, created_at 
			  FROM interactions WHERE contact_id = $1 ORDER BY interaction_date DESC`
	
	rows, err := db.Query(query, contactID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var interactions []Interaction
	for rows.Next() {
		var i Interaction
		err := rows.Scan(&i.ID, &i.ContactID, &i.InteractionType, &i.InteractionDate,
						&i.Description, &i.Sentiment, &i.CreatedAt)
		if err != nil {
			continue
		}
		interactions = append(interactions, i)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(interactions)
}

func createInteractionHandler(w http.ResponseWriter, r *http.Request) {
	var i Interaction
	if err := json.NewDecoder(r.Body).Decode(&i); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `INSERT INTO interactions (contact_id, interaction_type, interaction_date, description, sentiment) 
			  VALUES ($1, $2, $3, $4, $5) RETURNING id`
	
	err := db.QueryRow(query, i.ContactID, i.InteractionType, i.InteractionDate,
					   i.Description, i.Sentiment).Scan(&i.ID)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(i)
}

func getGiftsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	contactID, _ := strconv.Atoi(vars["contactId"])

	query := `SELECT id, contact_id, gift_name, description, price, occasion, given_date, 
			  status, rating, notes, purchase_link, created_at 
			  FROM gifts WHERE contact_id = $1 ORDER BY created_at DESC`
	
	rows, err := db.Query(query, contactID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var gifts []Gift
	for rows.Next() {
		var g Gift
		err := rows.Scan(&g.ID, &g.ContactID, &g.GiftName, &g.Description, &g.Price,
						&g.Occasion, &g.GivenDate, &g.Status, &g.Rating, &g.Notes,
						&g.PurchaseLink, &g.CreatedAt)
		if err != nil {
			continue
		}
		gifts = append(gifts, g)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gifts)
}

func createGiftHandler(w http.ResponseWriter, r *http.Request) {
	var g Gift
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `INSERT INTO gifts (contact_id, gift_name, description, price, occasion, status, notes) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	
	err := db.QueryRow(query, g.ContactID, g.GiftName, g.Description, g.Price,
					   g.Occasion, g.Status, g.Notes).Scan(&g.ID)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(g)
}

func getRemindersHandler(w http.ResponseWriter, r *http.Request) {
	query := `SELECT r.id, r.contact_id, r.reminder_type, r.reminder_date, r.reminder_time, 
			  r.message, r.sent, r.created_at, c.name 
			  FROM reminders r 
			  JOIN contacts c ON r.contact_id = c.id 
			  WHERE r.sent = false AND r.reminder_date >= CURRENT_DATE 
			  ORDER BY r.reminder_date, r.reminder_time`
	
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var reminders []map[string]interface{}
	for rows.Next() {
		var r Reminder
		var contactName string
		err := rows.Scan(&r.ID, &r.ContactID, &r.ReminderType, &r.ReminderDate,
						&r.ReminderTime, &r.Message, &r.Sent, &r.CreatedAt, &contactName)
		if err != nil {
			continue
		}
		reminder := map[string]interface{}{
			"id":            r.ID,
			"contact_id":    r.ContactID,
			"contact_name":  contactName,
			"reminder_type": r.ReminderType,
			"reminder_date": r.ReminderDate,
			"reminder_time": r.ReminderTime,
			"message":       r.Message,
			"sent":          r.Sent,
			"created_at":    r.CreatedAt,
		}
		reminders = append(reminders, reminder)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reminders)
}

func getBirthdaysHandler(w http.ResponseWriter, r *http.Request) {
	daysAhead := 7 // default
	if d := r.URL.Query().Get("days_ahead"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			daysAhead = parsed
		}
	}

	ctx := r.Context()
	reminders, err := relationshipProcessor.GetUpcomingBirthdays(ctx, daysAhead)
	if err != nil {
		http.Error(w, "Failed to get birthday reminders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reminders)
}

func enrichContactHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	contactID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid contact ID", http.StatusBadRequest)
		return
	}

	// Get contact name from database
	var name string
	err = db.QueryRow("SELECT name FROM contacts WHERE id = $1", contactID).Scan(&name)
	if err != nil {
		http.Error(w, "Contact not found", http.StatusNotFound)
		return
	}

	ctx := r.Context()
	enrichment, err := relationshipProcessor.EnrichContact(ctx, contactID, name)
	if err != nil {
		http.Error(w, "Failed to enrich contact", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(enrichment)
}

func suggestGiftsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	contactID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid contact ID", http.StatusBadRequest)
		return
	}

	var req GiftSuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.ContactID = contactID

	// Set defaults if not provided
	if req.Budget == "" {
		req.Budget = "50-100"
	}
	if req.Occasion == "" {
		req.Occasion = "birthday"
	}

	ctx := r.Context()
	suggestions, err := relationshipProcessor.SuggestGifts(ctx, req)
	if err != nil {
		http.Error(w, "Failed to generate gift suggestions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

func getInsightsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	contactID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid contact ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	insights, err := relationshipProcessor.AnalyzeRelationships(ctx, contactID)
	if err != nil {
		http.Error(w, "Failed to generate relationship insights", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(insights)
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "personal-relationship-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	initDB()
	defer db.Close()

	// Initialize RelationshipProcessor
	relationshipProcessor = NewRelationshipProcessor(db)

	r := mux.NewRouter()

	// Health check
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// Contact routes
	r.HandleFunc("/api/contacts", getContactsHandler).Methods("GET")
	r.HandleFunc("/api/contacts", createContactHandler).Methods("POST")
	r.HandleFunc("/api/contacts/{id}", getContactHandler).Methods("GET")
	r.HandleFunc("/api/contacts/{id}", updateContactHandler).Methods("PUT")
	r.HandleFunc("/api/contacts/{id}", deleteContactHandler).Methods("DELETE")

	// Interaction routes
	r.HandleFunc("/api/contacts/{contactId}/interactions", getInteractionsHandler).Methods("GET")
	r.HandleFunc("/api/interactions", createInteractionHandler).Methods("POST")

	// Gift routes
	r.HandleFunc("/api/contacts/{contactId}/gifts", getGiftsHandler).Methods("GET")
	r.HandleFunc("/api/gifts", createGiftHandler).Methods("POST")

	// Reminder routes
	r.HandleFunc("/api/reminders", getRemindersHandler).Methods("GET")

	// Relationship processor routes
	r.HandleFunc("/api/birthdays", getBirthdaysHandler).Methods("GET")
	r.HandleFunc("/api/contacts/{id}/enrich", enrichContactHandler).Methods("POST")
	r.HandleFunc("/api/contacts/{id}/gifts/suggest", suggestGiftsHandler).Methods("POST")
	r.HandleFunc("/api/contacts/{id}/insights", getInsightsHandler).Methods("GET")

	// Enable CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(r)

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	log.Printf("Personal Relationship Manager API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
