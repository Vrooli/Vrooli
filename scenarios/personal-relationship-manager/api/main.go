package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

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

func initDB() {
	var err error
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "personal_relationships"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("Connected to database")
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

func main() {
	initDB()
	defer db.Close()

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

	// Enable CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(r)

	port := getEnv("API_PORT", getEnv("PORT", ""))

	log.Printf("Personal Relationship Manager API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
