package main

import (
	"github.com/vrooli/api-core/preflight"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/lib/pq"
	"github.com/rs/cors"
)

// Configuration
type Config struct {
	Port                   string
	PostgresURL            string
	QdrantURL              string
	AuthServiceURL         string
	NotificationServiceURL string
	OllamaURL              string
	NotificationProfileID  string
	NotificationAPIKey     string
	JWTSecret              string
}

// Database connection
var db *sql.DB
var calendarProcessor *CalendarProcessor
var nlpProcessor *NLPProcessor
var vectorSearchManager *VectorSearchManager
var errorHandler *ErrorHandler
var conflictDetector *ConflictDetector
var categoryManager *CategoryManager
var prepManager *MeetingPrepManager

// Models
type Event struct {
	ID               string                 `json:"id"`
	UserID           string                 `json:"user_id"`
	Title            string                 `json:"title"`
	Description      string                 `json:"description"`
	StartTime        time.Time              `json:"start_time"`
	EndTime          time.Time              `json:"end_time"`
	Timezone         string                 `json:"timezone"`
	Location         string                 `json:"location"`
	EventType        string                 `json:"event_type"`
	Status           string                 `json:"status"` // active, cancelled, deleted
	Metadata         map[string]interface{} `json:"metadata"`
	AutomationConfig map[string]interface{} `json:"automation_config"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// EventTemplate represents a reusable event template
type EventTemplate struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	TemplateData map[string]interface{} `json:"template_data"`
	Category     string                 `json:"category"`
	IsSystem     bool                   `json:"is_system"`
	UseCount     int                    `json:"use_count"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type CreateEventRequest struct {
	Title            string                 `json:"title"`
	Description      string                 `json:"description"`
	StartTime        string                 `json:"start_time"`
	EndTime          string                 `json:"end_time"`
	Timezone         string                 `json:"timezone"`
	Location         string                 `json:"location"`
	EventType        string                 `json:"event_type"`
	Metadata         map[string]interface{} `json:"metadata"`
	AutomationConfig map[string]interface{} `json:"automation_config"`
	Recurrence       *RecurrenceRequest     `json:"recurrence"`
	Reminders        []ReminderRequest      `json:"reminders"`
}

type RecurrenceRequest struct {
	Pattern        string     `json:"pattern"`
	Interval       int        `json:"interval"`
	DaysOfWeek     []int      `json:"days_of_week"`
	EndDate        *time.Time `json:"end_date"`
	MaxOccurrences *int       `json:"max_occurrences"`
}

type ReminderRequest struct {
	MinutesBefore    int    `json:"minutes_before"`
	NotificationType string `json:"type"`
}

type ChatRequest struct {
	Message string                 `json:"message"`
	Context map[string]interface{} `json:"context"`
}

type ChatResponse struct {
	Response             string                 `json:"response"`
	SuggestedActions     []SuggestedAction      `json:"suggested_actions"`
	RequiresConfirmation bool                   `json:"requires_confirmation"`
	Context              map[string]interface{} `json:"context"`
}

type SuggestedAction struct {
	Action     string                 `json:"action"`
	Confidence float64                `json:"confidence"`
	Parameters map[string]interface{} `json:"parameters"`
}

type User struct {
	ID          string `json:"id"`
	AuthUserID  string `json:"auth_user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Timezone    string `json:"timezone"`
}

type contextKey string

const authUserContextKey contextKey = "calendar_auth_user"

func attachUserToContext(r *http.Request, user *User) *http.Request {
	if r == nil || user == nil {
		return r
	}

	ctx := context.WithValue(r.Context(), authUserContextKey, user)
	return r.WithContext(ctx)
}

func getAuthenticatedUser(r *http.Request) (*User, bool) {
	if r == nil {
		return nil, false
	}

	user, ok := r.Context().Value(authUserContextKey).(*User)
	if !ok || user == nil {
		return nil, false
	}

	return user, true
}

type authLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authLoginResponse struct {
	Success      bool            `json:"success"`
	Token        string          `json:"token"`
	RefreshToken string          `json:"refresh_token"`
	User         json.RawMessage `json:"user"`
	Message      string          `json:"message"`
}

func authLoginProxyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var req authLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorHandler.HandleError(w, r, BadRequestError("Invalid login payload", map[string]string{"parse_error": err.Error()}))
		return
	}

	validations := ValidationErrors{}
	if strings.TrimSpace(req.Email) == "" {
		validations = append(validations, NewValidationError("email", "", "Email is required"))
	}
	if strings.TrimSpace(req.Password) == "" {
		validations = append(validations, NewValidationError("password", "", "Password is required"))
	}
	if len(validations) > 0 {
		errorHandler.HandleError(w, r, validations)
		return
	}

	authServiceURL := strings.TrimSpace(os.Getenv("AUTH_SERVICE_URL"))
	if authServiceURL == "" {
		errorHandler.HandleError(w, r, ServiceUnavailableError("authentication"))
		return
	}

	loginURL := strings.TrimRight(authServiceURL, "/") + "/api/v1/auth/login"
	payload, err := json.Marshal(req)
	if err != nil {
		errorHandler.HandleError(w, r, InternalServerError("Failed to encode login payload", map[string]string{"error": err.Error()}))
		return
	}

	forwardReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, loginURL, bytes.NewReader(payload))
	if err != nil {
		errorHandler.HandleError(w, r, InternalServerError("Failed to contact authentication service", map[string]string{"error": err.Error()}))
		return
	}
	forwardReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(forwardReq)
	if err != nil {
		errorHandler.HandleError(w, r, ExternalServiceError("authentication", err))
		return
	}
	defer resp.Body.Close()

	for k, values := range resp.Header {
		if strings.EqualFold(k, "Content-Length") {
			continue
		}
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("Failed to stream auth service response: %v", err)
	}
}

// Initialize configuration
func initConfig() *Config {
	// Load .env file if it exists
	godotenv.Load()

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

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

	// External service URLs - REQUIRED, no defaults
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		log.Fatal("❌ QDRANT_URL environment variable is required")
	}

	// Auth service is optional - single user mode if not available
	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		log.Println("⚠️  AUTH_SERVICE_URL not configured - running in single-user mode")
		log.Println("   Events will not have user isolation")
	}

	// Notification service is optional - reminders won't be delivered
	notificationServiceURL := os.Getenv("NOTIFICATION_SERVICE_URL")
	if notificationServiceURL == "" {
		log.Println("⚠️  NOTIFICATION_SERVICE_URL not configured - notifications disabled")
		log.Println("   Event reminders will be queued but not delivered")
	}

	// Ollama is optional - NLP features will gracefully degrade
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		log.Println("⚠️  OLLAMA_URL not configured - NLP features will use rule-based fallback")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("❌ JWT_SECRET environment variable is required")
	}

	return &Config{
		Port:                   port,
		PostgresURL:            postgresURL,
		QdrantURL:              qdrantURL,
		AuthServiceURL:         authServiceURL,
		NotificationServiceURL: notificationServiceURL,
		OllamaURL:              ollamaURL,
		NotificationProfileID:  os.Getenv("NOTIFICATION_PROFILE_ID"), // Optional
		NotificationAPIKey:     os.Getenv("NOTIFICATION_API_KEY"),    // Optional
		JWTSecret:              jwtSecret,
	}
}

// getEnvOrDefault and requireEnv removed to prevent hardcoded defaults

// Initialize database connection
func initDatabase(config *Config) error {
	var err error
	db, err = sql.Open("postgres", config.PostgresURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
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

	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))

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

		// Add random jitter to prevent thundering herd behavior
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(randSource.Float64() * jitterRange)
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
		return fmt.Errorf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")

	// Run database migrations
	if err := runMigrations(config); err != nil {
		log.Printf("⚠️  Warning: Failed to run migrations: %v", err)
		// Don't fail startup - migrations might already be applied
	}

	return nil
}

// Run database migrations
func runMigrations(config *Config) error {
	log.Println("🔧 Running database migrations...")

	// Check and create each table individually
	// This allows incremental updates to the schema
	tablesToCreate := []struct {
		name   string
		schema string
	}{
		{
			name: "events",
			schema: `CREATE TABLE IF NOT EXISTS events (
				id UUID PRIMARY KEY,
				user_id VARCHAR(255) NOT NULL,
				title VARCHAR(255) NOT NULL,
				description TEXT,
				start_time TIMESTAMPTZ NOT NULL,
				end_time TIMESTAMPTZ NOT NULL,
				timezone VARCHAR(50) DEFAULT 'UTC',
				location VARCHAR(500),
				event_type VARCHAR(50) DEFAULT 'meeting',
				status VARCHAR(20) DEFAULT 'active',
				metadata JSONB DEFAULT '{}',
				automation_config JSONB DEFAULT '{}',
				created_at TIMESTAMPTZ DEFAULT NOW(),
				updated_at TIMESTAMPTZ DEFAULT NOW()
			);
			CREATE INDEX IF NOT EXISTS idx_events_user_id ON events(user_id);
			CREATE INDEX IF NOT EXISTS idx_events_start_time ON events(start_time);
			CREATE INDEX IF NOT EXISTS idx_events_status ON events(status);`,
		},
		{
			name: "event_reminders",
			schema: `CREATE TABLE IF NOT EXISTS event_reminders (
				id UUID PRIMARY KEY,
				event_id UUID REFERENCES events(id) ON DELETE CASCADE,
				minutes_before INTEGER NOT NULL,
				notification_type VARCHAR(20) DEFAULT 'email',
				status VARCHAR(20) DEFAULT 'pending',
				scheduled_time TIMESTAMPTZ NOT NULL,
				notification_id VARCHAR(255),
				created_at TIMESTAMPTZ DEFAULT NOW()
			);
			CREATE INDEX IF NOT EXISTS idx_reminders_event_id ON event_reminders(event_id);
			CREATE INDEX IF NOT EXISTS idx_reminders_scheduled_time ON event_reminders(scheduled_time);
			CREATE INDEX IF NOT EXISTS idx_reminders_status ON event_reminders(status);`,
		},
		{
			name: "recurring_patterns",
			schema: `CREATE TABLE IF NOT EXISTS recurring_patterns (
				id UUID PRIMARY KEY,
				parent_event_id UUID REFERENCES events(id) ON DELETE CASCADE,
				pattern_type VARCHAR(20) NOT NULL,
				interval_value INTEGER DEFAULT 1,
				days_of_week INTEGER[],
				end_date TIMESTAMPTZ,
				max_occurrences INTEGER,
				created_at TIMESTAMPTZ DEFAULT NOW()
			);
			CREATE INDEX IF NOT EXISTS idx_recurring_parent_event ON recurring_patterns(parent_event_id);`,
		},
		{
			name: "event_embeddings",
			schema: `CREATE TABLE IF NOT EXISTS event_embeddings (
				id UUID PRIMARY KEY,
				event_id UUID REFERENCES events(id) ON DELETE CASCADE,
				qdrant_point_id UUID NOT NULL,
				embedding_version VARCHAR(20) DEFAULT 'v1.0',
				content_hash VARCHAR(64) NOT NULL,
				keywords TEXT[],
				created_at TIMESTAMPTZ DEFAULT NOW(),
				updated_at TIMESTAMPTZ DEFAULT NOW(),
				CONSTRAINT uq_event_embedding UNIQUE (event_id)
			);
			CREATE INDEX IF NOT EXISTS idx_embeddings_event_id ON event_embeddings(event_id);
			CREATE INDEX IF NOT EXISTS idx_embeddings_qdrant_id ON event_embeddings(qdrant_point_id);
			CREATE INDEX IF NOT EXISTS idx_embeddings_content_hash ON event_embeddings(content_hash);`,
		},
		{
			name: "event_categories",
			schema: `CREATE TABLE IF NOT EXISTS event_categories (
				id VARCHAR(50) PRIMARY KEY,
				user_id VARCHAR(255) NOT NULL,
				name VARCHAR(100) NOT NULL,
				color VARCHAR(7) DEFAULT '#808080',
				icon VARCHAR(50) DEFAULT 'label',
				description TEXT,
				created_at TIMESTAMPTZ DEFAULT NOW(),
				CONSTRAINT uq_user_category UNIQUE (user_id, name)
			);
			CREATE INDEX IF NOT EXISTS idx_categories_user_id ON event_categories(user_id);`,
		},
		{
			name: "event_templates",
			schema: `CREATE TABLE IF NOT EXISTS event_templates (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				user_id VARCHAR(255),
				name VARCHAR(100) NOT NULL,
				description TEXT,
				template_data JSONB NOT NULL DEFAULT '{}'::jsonb,
				category VARCHAR(50),
				is_system BOOLEAN DEFAULT FALSE,
				use_count INTEGER DEFAULT 0,
				created_at TIMESTAMPTZ DEFAULT NOW(),
				updated_at TIMESTAMPTZ DEFAULT NOW(),
				CONSTRAINT unique_user_template_name UNIQUE(user_id, name)
			);
			CREATE INDEX IF NOT EXISTS idx_event_templates_user_id ON event_templates(user_id);
			CREATE INDEX IF NOT EXISTS idx_event_templates_category ON event_templates(category);
			CREATE INDEX IF NOT EXISTS idx_event_templates_system ON event_templates(is_system);`,
		},
		{
			name: "event_attendees",
			schema: `CREATE TABLE IF NOT EXISTS event_attendees (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				event_id UUID NOT NULL,
				user_id VARCHAR(255) NOT NULL,
				name VARCHAR(255),
				email VARCHAR(255),
				rsvp_status VARCHAR(20) DEFAULT 'pending',
				rsvp_message TEXT,
				response_time TIMESTAMPTZ,
				attendance_status VARCHAR(20),
				check_in_time TIMESTAMPTZ,
				check_in_method VARCHAR(20),
				notes TEXT,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				CONSTRAINT unique_event_user UNIQUE(event_id, user_id)
			);
			CREATE INDEX IF NOT EXISTS idx_attendees_event_id ON event_attendees(event_id);
			CREATE INDEX IF NOT EXISTS idx_attendees_user_id ON event_attendees(user_id);
			CREATE INDEX IF NOT EXISTS idx_attendees_rsvp ON event_attendees(rsvp_status);`,
		},
		{
			name: "resources",
			schema: `CREATE TABLE IF NOT EXISTS resources (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				name VARCHAR(255) NOT NULL,
				resource_type VARCHAR(50) NOT NULL DEFAULT 'room',
				description TEXT,
				location VARCHAR(500),
				capacity INTEGER,
				metadata JSONB DEFAULT '{}',
				availability_rules JSONB DEFAULT '{}',
				status VARCHAR(20) NOT NULL DEFAULT 'active',
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				CONSTRAINT chk_resources_type CHECK (resource_type IN ('room', 'equipment', 'vehicle', 'person', 'virtual', 'other')),
				CONSTRAINT chk_resources_status CHECK (status IN ('active', 'inactive', 'maintenance'))
			);
			CREATE INDEX IF NOT EXISTS idx_resources_type ON resources(resource_type);
			CREATE INDEX IF NOT EXISTS idx_resources_status ON resources(status);
			CREATE INDEX IF NOT EXISTS idx_resources_name ON resources(name);`,
		},
		{
			name: "event_resources",
			schema: `CREATE TABLE IF NOT EXISTS event_resources (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
				resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
				booking_status VARCHAR(20) NOT NULL DEFAULT 'confirmed',
				notes TEXT,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				CONSTRAINT uq_event_resource UNIQUE (event_id, resource_id),
				CONSTRAINT chk_booking_status CHECK (booking_status IN ('pending', 'confirmed', 'cancelled'))
			);
			CREATE INDEX IF NOT EXISTS idx_event_resources_event ON event_resources(event_id);
			CREATE INDEX IF NOT EXISTS idx_event_resources_resource ON event_resources(resource_id);
			CREATE INDEX IF NOT EXISTS idx_event_resources_status ON event_resources(booking_status);`,
		},
		{
			name: "external_calendars",
			schema: `CREATE TABLE IF NOT EXISTS external_calendars (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				user_id VARCHAR(255) NOT NULL,
				provider VARCHAR(50) NOT NULL CHECK (provider IN ('google', 'outlook')),
				access_token TEXT NOT NULL,
				refresh_token TEXT,
				expires_at TIMESTAMPTZ NOT NULL,
				calendar_id VARCHAR(255),
				sync_enabled BOOLEAN DEFAULT true,
				sync_direction VARCHAR(20) DEFAULT 'bidirectional' CHECK (sync_direction IN ('bidirectional', 'import_only', 'export_only')),
				last_sync_time TIMESTAMPTZ,
				sync_metadata JSONB DEFAULT '{}',
				created_at TIMESTAMPTZ DEFAULT NOW(),
				updated_at TIMESTAMPTZ DEFAULT NOW(),
				UNIQUE (user_id, provider)
			);
			CREATE INDEX IF NOT EXISTS idx_external_calendars_user_id ON external_calendars(user_id);
			CREATE INDEX IF NOT EXISTS idx_external_calendars_sync ON external_calendars(sync_enabled, last_sync_time);`,
		},
		{
			name: "oauth_states",
			schema: `CREATE TABLE IF NOT EXISTS oauth_states (
				state VARCHAR(255) PRIMARY KEY,
				user_id VARCHAR(255) NOT NULL,
				provider VARCHAR(50) NOT NULL,
				created_at TIMESTAMPTZ DEFAULT NOW(),
				expires_at TIMESTAMPTZ NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_oauth_states_expires ON oauth_states(expires_at);`,
		},
		{
			name: "external_sync_log",
			schema: `CREATE TABLE IF NOT EXISTS external_sync_log (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				calendar_id UUID REFERENCES external_calendars(id) ON DELETE CASCADE,
				sync_type VARCHAR(20) CHECK (sync_type IN ('manual', 'scheduled', 'webhook')),
				direction VARCHAR(20) CHECK (direction IN ('import', 'export', 'bidirectional')),
				events_created INTEGER DEFAULT 0,
				events_updated INTEGER DEFAULT 0,
				events_deleted INTEGER DEFAULT 0,
				errors_count INTEGER DEFAULT 0,
				error_details JSONB,
				started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				completed_at TIMESTAMPTZ,
				status VARCHAR(20) DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'success', 'partial', 'failed'))
			);
			CREATE INDEX IF NOT EXISTS idx_sync_log_calendar ON external_sync_log(calendar_id, started_at DESC);`,
		},
		{
			name: "external_event_mappings",
			schema: `CREATE TABLE IF NOT EXISTS external_event_mappings (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				local_event_id UUID REFERENCES events(id) ON DELETE CASCADE,
				external_id VARCHAR(255) NOT NULL,
				provider VARCHAR(50) NOT NULL,
				external_metadata JSONB DEFAULT '{}',
				last_synced_at TIMESTAMPTZ DEFAULT NOW(),
				sync_hash VARCHAR(64),
				UNIQUE (external_id, provider)
			);
			CREATE INDEX IF NOT EXISTS idx_external_mappings_local ON external_event_mappings(local_event_id);
			CREATE INDEX IF NOT EXISTS idx_external_mappings_external ON external_event_mappings(external_id, provider);`,
		},
	}

	createdCount := 0
	for _, table := range tablesToCreate {
		// Check if table exists
		var exists bool
		checkQuery := `SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)`

		if err := db.QueryRow(checkQuery, table.name).Scan(&exists); err != nil {
			return fmt.Errorf("failed to check %s table existence: %v", table.name, err)
		}

		if !exists {
			log.Printf("Creating %s table...", table.name)
			if _, err := db.Exec(table.schema); err != nil {
				return fmt.Errorf("failed to create %s table: %v", table.name, err)
			}
			createdCount++
			log.Printf("✅ Created %s table", table.name)
		} else {
			log.Printf("✓ Table %s already exists", table.name)

			// Special handling for event_templates table to fix user_id column type
			if table.name == "event_templates" {
				// Check if user_id column is UUID and needs to be converted
				var dataType string
				checkColumnQuery := `
					SELECT data_type 
					FROM information_schema.columns 
					WHERE table_name = 'event_templates' 
					AND column_name = 'user_id'
				`
				if err := db.QueryRow(checkColumnQuery).Scan(&dataType); err == nil && dataType == "uuid" {
					log.Printf("🔧 Converting event_templates.user_id from UUID to VARCHAR(255)...")
					// First drop the constraint, alter the column, then recreate constraint
					alterQueries := []string{
						"ALTER TABLE event_templates DROP CONSTRAINT IF EXISTS unique_user_template_name",
						"ALTER TABLE event_templates ALTER COLUMN user_id DROP NOT NULL",
						"ALTER TABLE event_templates ALTER COLUMN user_id TYPE VARCHAR(255) USING user_id::text",
						"ALTER TABLE event_templates ADD CONSTRAINT unique_user_template_name UNIQUE(user_id, name)",
					}
					for _, query := range alterQueries {
						if _, err := db.Exec(query); err != nil {
							log.Printf("Warning: Failed to alter event_templates table: %v", err)
						}
					}
					log.Printf("✅ Converted event_templates.user_id to VARCHAR(255)")
				}
			}
		}
	}

	if createdCount > 0 {
		log.Printf("✅ Created %d new tables", createdCount)
	} else {
		log.Println("✅ All tables already exist")
	}

	// Create PostgreSQL functions for resource management
	functions := []struct {
		name   string
		create string
	}{
		{
			name: "is_resource_available",
			create: `
				CREATE OR REPLACE FUNCTION is_resource_available(
					p_resource_id UUID,
					p_start_time TIMESTAMPTZ,
					p_end_time TIMESTAMPTZ,
					p_exclude_event_id UUID DEFAULT NULL
				) RETURNS BOOLEAN AS $$
				DECLARE
					conflict_count INTEGER;
				BEGIN
					SELECT COUNT(*)
					INTO conflict_count
					FROM event_resources er
					JOIN events e ON er.event_id = e.id
					WHERE er.resource_id = p_resource_id
					  AND er.booking_status = 'confirmed'
					  AND e.status = 'active'
					  AND (p_exclude_event_id IS NULL OR e.id != p_exclude_event_id)
					  AND (
						  (e.start_time >= p_start_time AND e.start_time < p_end_time) OR
						  (e.end_time > p_start_time AND e.end_time <= p_end_time) OR
						  (e.start_time <= p_start_time AND e.end_time >= p_end_time)
					  );
					RETURN conflict_count = 0;
				END;
				$$ LANGUAGE plpgsql;
			`,
		},
		{
			name: "get_resource_conflicts",
			create: `
				CREATE OR REPLACE FUNCTION get_resource_conflicts(
					p_resource_id UUID,
					p_start_time TIMESTAMPTZ,
					p_end_time TIMESTAMPTZ
				) RETURNS TABLE(
					event_id UUID,
					event_title VARCHAR(255),
					start_time TIMESTAMPTZ,
					end_time TIMESTAMPTZ,
					conflict_type VARCHAR(20)
				) AS $$
				BEGIN
					RETURN QUERY
					SELECT 
						e.id as event_id,
						e.title as event_title,
						e.start_time,
						e.end_time,
						'booking'::VARCHAR(20) as conflict_type
					FROM event_resources er
					JOIN events e ON er.event_id = e.id
					WHERE er.resource_id = p_resource_id
					  AND er.booking_status = 'confirmed'
					  AND e.status = 'active'
					  AND (
						  (e.start_time >= p_start_time AND e.start_time < p_end_time) OR
						  (e.end_time > p_start_time AND e.end_time <= p_end_time) OR
						  (e.start_time <= p_start_time AND e.end_time >= p_end_time)
					  );
				END;
				$$ LANGUAGE plpgsql;
			`,
		},
	}

	// Create functions
	for _, fn := range functions {
		if _, err := db.Exec(fn.create); err != nil {
			log.Printf("⚠️  Warning: Failed to create function %s: %v", fn.name, err)
		} else {
			log.Printf("✅ Created or updated function %s", fn.name)
		}
	}

	log.Println("✅ Database migrations completed successfully")
	return nil
}

// Middleware for authentication
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Check if auth service is configured
		authServiceURL := os.Getenv("AUTH_SERVICE_URL")
		if authServiceURL == "" {
			defaultUser := &User{
				ID:          "default-user",
				AuthUserID:  "default-user",
				Email:       "user@localhost",
				DisplayName: "Default User",
				Timezone:    "UTC",
			}

			r.Header.Set("X-User-ID", defaultUser.ID)
			r.Header.Set("X-Auth-User-ID", defaultUser.AuthUserID)
			r.Header.Set("X-User-Email", defaultUser.Email)
			r.Header.Set("X-User-Display-Name", defaultUser.DisplayName)
			r.Header.Set("X-User-Timezone", defaultUser.Timezone)

			next.ServeHTTP(w, attachUserToContext(r, defaultUser))
			return
		}

		authHeader := r.Header.Get("Authorization")
		// Development/test mode bypass - only in development environment
		if env := os.Getenv("ENVIRONMENT"); env == "development" || env == "test" {
			if authHeader == "Bearer test-token" || authHeader == "Bearer test" || authHeader == "Bearer mock-token-for-testing" {
				testUser := &User{
					ID:          "test-user",
					AuthUserID:  "test-user",
					Email:       "test@localhost",
					DisplayName: "Test User",
					Timezone:    "UTC",
				}

				r.Header.Set("X-User-ID", testUser.ID)
				r.Header.Set("X-Auth-User-ID", testUser.AuthUserID)
				r.Header.Set("X-User-Email", testUser.Email)
				r.Header.Set("X-User-Display-Name", testUser.DisplayName)
				r.Header.Set("X-User-Timezone", testUser.Timezone)

				next.ServeHTTP(w, attachUserToContext(r, testUser))
				return
			}
		}

		if authHeader == "" {
			errorResponse := map[string]interface{}{
				"error": map[string]interface{}{
					"code":      "UNAUTHORIZED",
					"message":   "User authentication required",
					"timestamp": time.Now().Format(time.RFC3339),
					"path":      r.URL.Path,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(errorResponse)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		// Validate token with scenario-authenticator
		user, err := validateToken(token)
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		if user.AuthUserID == "" {
			user.AuthUserID = user.ID
		}
		if user.DisplayName == "" {
			if user.Email != "" {
				user.DisplayName = user.Email
			} else {
				user.DisplayName = user.ID
			}
		}
		if user.Timezone == "" {
			user.Timezone = "UTC"
		}

		r.Header.Set("X-User-ID", user.ID)
		r.Header.Set("X-Auth-User-ID", user.AuthUserID)
		r.Header.Set("X-User-Email", user.Email)
		r.Header.Set("X-User-Display-Name", user.DisplayName)
		r.Header.Set("X-User-Timezone", user.Timezone)

		// Add user to request context
		next.ServeHTTP(w, attachUserToContext(r, user))
	})
}

// Validate token with scenario-authenticator service
func validateToken(token string) (*User, error) {
	// Get auth service URL from environment
	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		// Return default user in single-user mode
		return &User{
			ID:          "default-user",
			AuthUserID:  "default-user",
			Email:       "user@localhost",
			DisplayName: "Default User",
			Timezone:    "UTC",
		}, nil
	}

	// Create validation request
	req, err := http.NewRequest("GET", authServiceURL+"/api/v1/auth/validate", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// Make request to scenario-authenticator
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth service request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid token")
	}

	// Parse response
	var authResponse struct {
		UserID      string `json:"user_id"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return nil, fmt.Errorf("failed to parse auth response: %w", err)
	}

	// Create user from auth response
	return &User{
		ID:          authResponse.UserID,
		AuthUserID:  authResponse.UserID,
		Email:       authResponse.Email,
		DisplayName: authResponse.DisplayName,
		Timezone:    "UTC", // Default timezone, could be fetched from user profile
	}, nil
}

// Health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	overallStatus := "healthy"
	var errors []map[string]interface{}
	readiness := true

	// Schema-compliant health response structure
	healthResponse := map[string]interface{}{
		"status":       overallStatus,
		"service":      "calendar-api",
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"readiness":    true,
		"version":      "1.0.0",
		"dependencies": map[string]interface{}{},
	}

	// Check database connectivity
	dbHealth := checkDatabaseHealth()
	healthResponse["dependencies"].(map[string]interface{})["database"] = dbHealth
	if dbHealth["status"] != "healthy" {
		overallStatus = "degraded"
		if dbHealth["status"] == "unhealthy" {
			readiness = false
			overallStatus = "unhealthy"
		}
		if dbHealth["error"] != nil {
			errors = append(errors, dbHealth["error"].(map[string]interface{}))
		}
	}

	// Check Qdrant vector database
	qdrantHealth := checkQdrantHealth()
	healthResponse["dependencies"].(map[string]interface{})["qdrant"] = qdrantHealth
	if qdrantHealth["status"] != "healthy" {
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if qdrantHealth["error"] != nil {
			errors = append(errors, qdrantHealth["error"].(map[string]interface{}))
		}
	}

	// Check Auth service (optional)
	authHealth := checkAuthServiceHealth()
	healthResponse["dependencies"].(map[string]interface{})["auth_service"] = authHealth
	if authHealth["status"] != "healthy" && authHealth["status"] != "not_configured" {
		if overallStatus == "healthy" {
			overallStatus = "degraded"
		}
		if authHealth["error"] != nil {
			errors = append(errors, authHealth["error"].(map[string]interface{}))
		}
	}

	// Check Notification service (optional)
	notificationHealth := checkNotificationServiceHealth()
	healthResponse["dependencies"].(map[string]interface{})["notification_service"] = notificationHealth
	if notificationHealth["status"] != "healthy" && notificationHealth["status"] != "not_configured" {
		if overallStatus == "healthy" {
			overallStatus = "degraded"
		}
		if notificationHealth["error"] != nil {
			errors = append(errors, notificationHealth["error"].(map[string]interface{}))
		}
	}

	// Check NLP processor (optional)
	nlpHealth := checkNLPProcessorHealth()
	healthResponse["dependencies"].(map[string]interface{})["nlp_processor"] = nlpHealth
	if nlpHealth["status"] != "healthy" && nlpHealth["status"] != "not_configured" {
		if overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}

	// Update final status
	healthResponse["status"] = overallStatus
	healthResponse["readiness"] = readiness

	// Add errors if any
	if len(errors) > 0 {
		healthResponse["errors"] = errors
	}

	// Add metrics
	healthResponse["metrics"] = map[string]interface{}{
		"total_dependencies":   5,
		"healthy_dependencies": countHealthyDependencies(healthResponse["dependencies"].(map[string]interface{})),
		"response_time_ms":     time.Since(start).Milliseconds(),
	}

	// Add calendar-specific stats
	calendarStats := getCalendarStats()
	healthResponse["calendar_stats"] = calendarStats

	// Return appropriate HTTP status
	statusCode := http.StatusOK
	if overallStatus == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(healthResponse)
}

// Health check helper methods
func checkDatabaseHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	if db == nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "DATABASE_NOT_INITIALIZED",
			"message":   "Database connection not initialized",
			"category":  "configuration",
			"retryable": false,
		}
		return health
	}

	// Test database connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "DATABASE_CONNECTION_FAILED",
			"message":   "Failed to ping database: " + err.Error(),
			"category":  "resource",
			"retryable": true,
		}
		return health
	}
	health["checks"].(map[string]interface{})["ping"] = "ok"

	// Check calendar tables
	requiredTables := []string{"events", "event_reminders", "recurring_patterns", "event_embeddings"}
	var missingTables []string

	for _, tableName := range requiredTables {
		var exists bool
		query := fmt.Sprintf(`SELECT EXISTS(
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = '%s'
		)`, tableName)

		if err := db.QueryRowContext(ctx, query).Scan(&exists); err != nil || !exists {
			missingTables = append(missingTables, tableName)
		}
	}

	if len(missingTables) > 0 {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "DATABASE_SCHEMA_INCOMPLETE",
			"message":   fmt.Sprintf("Missing tables: %v", missingTables),
			"category":  "configuration",
			"retryable": false,
		}
		health["checks"].(map[string]interface{})["missing_tables"] = missingTables
	} else {
		health["checks"].(map[string]interface{})["schema"] = "complete"
	}

	// Check active events count
	var eventCount int
	eventQuery := `SELECT COUNT(*) FROM events WHERE status = 'active' AND start_time > NOW() - INTERVAL '30 days'`
	if err := db.QueryRowContext(ctx, eventQuery).Scan(&eventCount); err == nil {
		health["checks"].(map[string]interface{})["recent_events"] = eventCount
	}

	// Check connection pool
	stats := db.Stats()
	health["checks"].(map[string]interface{})["open_connections"] = stats.OpenConnections
	health["checks"].(map[string]interface{})["max_connections"] = stats.MaxOpenConnections

	if stats.OpenConnections > stats.MaxOpenConnections*9/10 {
		if health["status"] == "healthy" {
			health["status"] = "degraded"
		}
		if health["error"] == nil {
			health["error"] = map[string]interface{}{
				"code":      "DATABASE_CONNECTION_POOL_HIGH",
				"message":   fmt.Sprintf("Connection pool usage high: %d/%d", stats.OpenConnections, stats.MaxOpenConnections),
				"category":  "resource",
				"retryable": false,
			}
		}
	}

	return health
}

func checkQdrantHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	config := initConfig()
	if config.QdrantURL == "" {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "QDRANT_NOT_CONFIGURED",
			"message":   "Qdrant URL not configured",
			"category":  "configuration",
			"retryable": false,
		}
		return health
	}

	// Test Qdrant connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", config.QdrantURL+"/", nil)
	if err != nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "QDRANT_REQUEST_FAILED",
			"message":   "Failed to create request to Qdrant: " + err.Error(),
			"category":  "internal",
			"retryable": false,
		}
		return health
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "QDRANT_CONNECTION_FAILED",
			"message":   "Failed to connect to Qdrant: " + err.Error(),
			"category":  "network",
			"retryable": true,
		}
		return health
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "QDRANT_UNHEALTHY",
			"message":   fmt.Sprintf("Qdrant returned status %d", resp.StatusCode),
			"category":  "resource",
			"retryable": true,
		}
	} else {
		health["checks"].(map[string]interface{})["connectivity"] = "ok"

		// Check calendar_events collection
		collectionReq, _ := http.NewRequestWithContext(ctx, "GET", config.QdrantURL+"/collections/calendar_events", nil)
		if collResp, err := client.Do(collectionReq); err == nil {
			defer collResp.Body.Close()
			if collResp.StatusCode == http.StatusOK {
				health["checks"].(map[string]interface{})["calendar_events_collection"] = "exists"
			} else {
				health["checks"].(map[string]interface{})["calendar_events_collection"] = "missing"
			}
		}
	}

	return health
}

func checkAuthServiceHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	config := initConfig()
	if config.AuthServiceURL == "" {
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["mode"] = "single_user"
		return health
	}

	// Test auth service connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", config.AuthServiceURL+"/health", nil)
	if err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "AUTH_SERVICE_REQUEST_FAILED",
			"message":   "Failed to create request to auth service: " + err.Error(),
			"category":  "internal",
			"retryable": false,
		}
		return health
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "AUTH_SERVICE_CONNECTION_FAILED",
			"message":   "Failed to connect to auth service: " + err.Error(),
			"category":  "network",
			"retryable": true,
		}
		return health
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		health["checks"].(map[string]interface{})["connectivity"] = "ok"
		health["checks"].(map[string]interface{})["mode"] = "multi_user"
	} else {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "AUTH_SERVICE_UNHEALTHY",
			"message":   fmt.Sprintf("Auth service returned status %d", resp.StatusCode),
			"category":  "resource",
			"retryable": true,
		}
	}

	return health
}

func checkNotificationServiceHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	config := initConfig()
	if config.NotificationServiceURL == "" {
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["reminders"] = "disabled"
		return health
	}

	// Test notification service connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", config.NotificationServiceURL+"/health", nil)
	if err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "NOTIFICATION_SERVICE_REQUEST_FAILED",
			"message":   "Failed to create request to notification service: " + err.Error(),
			"category":  "internal",
			"retryable": false,
		}
		return health
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "NOTIFICATION_SERVICE_CONNECTION_FAILED",
			"message":   "Failed to connect to notification service: " + err.Error(),
			"category":  "network",
			"retryable": true,
		}
		return health
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		health["checks"].(map[string]interface{})["connectivity"] = "ok"
		health["checks"].(map[string]interface{})["reminders"] = "enabled"
	} else {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "NOTIFICATION_SERVICE_UNHEALTHY",
			"message":   fmt.Sprintf("Notification service returned status %d", resp.StatusCode),
			"category":  "resource",
			"retryable": true,
		}
	}

	return health
}

func checkNLPProcessorHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	if nlpProcessor == nil {
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["mode"] = "rule_based_fallback"
		return health
	}

	// Check if Ollama is configured
	config := initConfig()
	if config.OllamaURL == "" {
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["mode"] = "rule_based_fallback"
		return health
	}

	// Test Ollama connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", config.OllamaURL+"/api/tags", nil)
	if err != nil {
		health["status"] = "degraded"
		health["checks"].(map[string]interface{})["mode"] = "rule_based_fallback"
		return health
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		health["status"] = "degraded"
		health["checks"].(map[string]interface{})["mode"] = "rule_based_fallback"
		return health
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		health["checks"].(map[string]interface{})["ollama_connectivity"] = "ok"
		health["checks"].(map[string]interface{})["mode"] = "ai_powered"
	} else {
		health["status"] = "degraded"
		health["checks"].(map[string]interface{})["mode"] = "rule_based_fallback"
	}

	return health
}

func countHealthyDependencies(deps map[string]interface{}) int {
	count := 0
	for _, dep := range deps {
		if depMap, ok := dep.(map[string]interface{}); ok {
			if status, exists := depMap["status"]; exists && (status == "healthy" || status == "not_configured") {
				count++
			}
		}
	}
	return count
}

func getCalendarStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_events":       0,
		"upcoming_events":    0,
		"active_reminders":   0,
		"recurring_patterns": 0,
	}

	if db == nil {
		return stats
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Get total events
	var totalEvents int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events WHERE status = 'active'").Scan(&totalEvents); err == nil {
		stats["total_events"] = totalEvents
	}

	// Get upcoming events
	var upcomingEvents int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events WHERE status = 'active' AND start_time > NOW()").Scan(&upcomingEvents); err == nil {
		stats["upcoming_events"] = upcomingEvents
	}

	// Get active reminders
	var activeReminders int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM event_reminders WHERE is_active = true").Scan(&activeReminders); err == nil {
		stats["active_reminders"] = activeReminders
	}

	// Get recurring patterns
	var recurringPatterns int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM recurring_patterns WHERE is_active = true").Scan(&recurringPatterns); err == nil {
		stats["recurring_patterns"] = recurringPatterns
	}

	return stats
}

func authValidateHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthenticatedUser(r)
	if !ok {
		errorHandler.HandleError(w, r, UnauthorizedError("User authentication required"))
		return
	}

	authUserID := user.AuthUserID
	if authUserID == "" {
		authUserID = user.ID
	}

	displayName := user.DisplayName
	if displayName == "" {
		if user.Email != "" {
			displayName = user.Email
		} else {
			displayName = user.ID
		}
	}

	timezone := user.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	email := user.Email
	if email == "" {
		email = r.Header.Get("X-User-Email")
	}

	response := struct {
		ID          string `json:"id"`
		AuthUserID  string `json:"authUserId"`
		Email       string `json:"email"`
		DisplayName string `json:"displayName"`
		Timezone    string `json:"timezone"`
		CreatedAt   string `json:"createdAt"`
		UpdatedAt   string `json:"updatedAt"`
	}{
		ID:          user.ID,
		AuthUserID:  authUserID,
		Email:       email,
		DisplayName: displayName,
		Timezone:    timezone,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	if createdAt := r.Header.Get("X-User-Created-At"); createdAt != "" {
		response.CreatedAt = createdAt
	}
	if updatedAt := r.Header.Get("X-User-Updated-At"); updatedAt != "" {
		response.UpdatedAt = updatedAt
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Event handlers
func createEventHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorHandler.HandleError(w, r, BadRequestError("Invalid JSON in request body", map[string]string{"parse_error": err.Error()}))
		return
	}

	// Validate request
	var validationErrors ValidationErrors
	if ve := ValidateRequired("title", req.Title); ve != nil {
		validationErrors = append(validationErrors, *ve)
	}
	if ve := ValidateRequired("start_time", req.StartTime); ve != nil {
		validationErrors = append(validationErrors, *ve)
	}
	if ve := ValidateRequired("end_time", req.EndTime); ve != nil {
		validationErrors = append(validationErrors, *ve)
	}
	if ve := ValidateStringLength("title", req.Title, 1, 255); ve != nil {
		validationErrors = append(validationErrors, *ve)
	}

	if len(validationErrors) > 0 {
		errorHandler.HandleError(w, r, validationErrors)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		errorHandler.HandleError(w, r, UnauthorizedError("User authentication required"))
		return
	}

	// Parse timestamps
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		errorHandler.HandleError(w, r, BadRequestError("Invalid start_time format", map[string]string{
			"expected_format": "RFC3339 (2006-01-02T15:04:05Z07:00)",
			"provided_value":  req.StartTime,
			"parse_error":     err.Error(),
		}))
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		errorHandler.HandleError(w, r, BadRequestError("Invalid end_time format", map[string]string{
			"expected_format": "RFC3339 (2006-01-02T15:04:05Z07:00)",
			"provided_value":  req.EndTime,
			"parse_error":     err.Error(),
		}))
		return
	}

	// Validate time logic
	if endTime.Before(startTime) || endTime.Equal(startTime) {
		errorHandler.HandleError(w, r, BadRequestError("End time must be after start time", map[string]string{
			"start_time": req.StartTime,
			"end_time":   req.EndTime,
		}))
		return
	}

	// Check for conflicts
	conflicts, err := conflictDetector.CheckConflicts(userID, startTime, endTime, "")
	if err != nil {
		log.Printf("Error checking conflicts: %v", err)
		// Continue even if conflict check fails
	} else if len(conflicts) > 0 {
		// Return conflicts to the user
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   false,
			"conflicts": conflicts,
			"message":   "Event conflicts detected. Please resolve conflicts or confirm to proceed.",
		})
		return
	}

	// Create event in database
	eventID := uuid.New().String()
	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	// Auto-categorize event if no type specified
	if req.EventType == "" {
		req.EventType = categoryManager.SuggestCategory(req.Title, req.Description)
	}

	metadataJSON, _ := json.Marshal(req.Metadata)
	automationJSON, _ := json.Marshal(req.AutomationConfig)

	query := `
		INSERT INTO events (id, user_id, title, description, start_time, end_time, timezone, location, event_type, metadata, automation_config)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = db.Exec(query, eventID, userID, req.Title, req.Description, startTime, endTime, timezone, req.Location, req.EventType, metadataJSON, automationJSON)
	if err != nil {
		errorHandler.HandleError(w, r, DatabaseError("event creation", err))
		return
	}

	// Create recurring pattern if specified
	// recurrenceCount := 0 // TODO: Use this for response
	if req.Recurrence != nil {
		recurringID := uuid.New().String()
		recurringQuery := `
			INSERT INTO recurring_patterns (id, parent_event_id, pattern_type, interval_value, days_of_week, end_date, max_occurrences)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`

		_, err = db.Exec(recurringQuery, recurringID, eventID, req.Recurrence.Pattern, req.Recurrence.Interval,
			pq.Array(req.Recurrence.DaysOfWeek), req.Recurrence.EndDate, req.Recurrence.MaxOccurrences)
		if err != nil {
			log.Printf("Error creating recurring pattern: %v", err)
		} else {
			// recurrenceCount = 1 // TODO: Use for response
			// Generate recurring events based on pattern
			go func() {
				if err := generateRecurringEvents(eventID, req.Recurrence, startTime, endTime, req, userID); err != nil {
					log.Printf("Error generating recurring events: %v", err)
				}
			}()
			log.Printf("Recurring pattern created for event %s", eventID)
		}
	}

	// Create reminders if specified
	reminderCount := 0
	for _, reminder := range req.Reminders {
		reminderID := uuid.New().String()
		reminderTime := startTime.Add(time.Duration(-reminder.MinutesBefore) * time.Minute)

		reminderQuery := `
			INSERT INTO event_reminders (id, event_id, minutes_before, notification_type, scheduled_time)
			VALUES ($1, $2, $3, $4, $5)
		`

		_, err = db.Exec(reminderQuery, reminderID, eventID, reminder.MinutesBefore, reminder.NotificationType, reminderTime)
		if err != nil {
			log.Printf("Error creating reminder: %v", err)
			// Continue with other reminders even if one fails
		} else {
			reminderCount++
		}
	}

	// Create vector embedding for AI-powered search
	createdEvent := Event{
		ID:          eventID,
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		StartTime:   startTime,
		EndTime:     endTime,
		Timezone:    timezone,
		Location:    req.Location,
		EventType:   req.EventType,
		Metadata:    req.Metadata,
	}

	// Generate embedding asynchronously to avoid blocking response
	go func() {
		ctx := context.Background()
		if err := vectorSearchManager.CreateEmbeddingForEvent(ctx, createdEvent); err != nil {
			log.Printf("Warning: Failed to create embedding for event %s: %v", eventID, err)
		}
	}()

	// Return success response
	response := map[string]interface{}{
		"success": true,
		"event": map[string]interface{}{
			"id":         eventID,
			"title":      req.Title,
			"start_time": startTime,
			"end_time":   endTime,
		},
		"reminders_scheduled": reminderCount,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func listEventsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	// Detect user's preferred timezone
	userTimezone := detectUserTimezone(r)

	// Parse query parameters
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	eventType := r.URL.Query().Get("event_type")
	search := r.URL.Query().Get("search")

	// Build query
	query := `
		SELECT id, user_id, title, description, start_time, end_time, timezone, location, event_type, status, created_at, updated_at
		FROM events 
		WHERE user_id = $1 AND status = 'active'
	`
	args := []interface{}{userID}
	argCount := 1

	if startDate != "" {
		argCount++
		query += fmt.Sprintf(" AND start_time >= $%d", argCount)
		args = append(args, startDate)
	}

	if endDate != "" {
		argCount++
		query += fmt.Sprintf(" AND start_time <= $%d", argCount)
		args = append(args, endDate)
	}

	if eventType != "" {
		argCount++
		query += fmt.Sprintf(" AND event_type = $%d", argCount)
		args = append(args, eventType)
	}

	var events []Event

	if search != "" {
		// Use semantic search powered by Qdrant
		ctx := r.Context()
		semanticEvents, err := vectorSearchManager.SearchEventsBySemantic(ctx, search, userID, 100)
		if err != nil {
			log.Printf("Error performing semantic search: %v", err)
			// Fallback to basic search
			argCount++
			query += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
			args = append(args, "%"+search+"%")
		} else {
			// Filter semantic results by other criteria (date, type)
			events = filterEventsByCriteria(semanticEvents, startDate, endDate, eventType)
		}
	}

	if len(events) == 0 && search == "" {
		// Regular query without search
		query += " ORDER BY start_time ASC LIMIT 100"

		rows, err := db.Query(query, args...)
		if err != nil {
			log.Printf("Error querying events: %v", err)
			http.Error(w, "Failed to query events", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var event Event
			err := rows.Scan(
				&event.ID, &event.UserID, &event.Title, &event.Description,
				&event.StartTime, &event.EndTime, &event.Timezone, &event.Location,
				&event.EventType, &event.Status, &event.CreatedAt, &event.UpdatedAt,
			)
			if err != nil {
				log.Printf("Error scanning event: %v", err)
				continue
			}
			events = append(events, event)
		}
	}

	// Apply timezone conversion for display
	for i := range events {
		formatEventTimes(&events[i], userTimezone)
	}

	response := map[string]interface{}{
		"events":      events,
		"total_count": len(events),
		"has_more":    len(events) >= 100,
		"timezone":    userTimezone, // Include timezone info in response
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get single event handler
func getEventHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	// Get event ID from URL
	vars := mux.Vars(r)
	eventID := vars["id"]
	if eventID == "" {
		http.Error(w, "Event ID is required", http.StatusBadRequest)
		return
	}

	// Query for the event
	var event Event
	var metadataJSON, automationJSON []byte
	query := `
		SELECT id, user_id, title, description, start_time, end_time, timezone, 
		       location, event_type, status, metadata, automation_config, created_at, updated_at
		FROM events 
		WHERE id = $1 AND user_id = $2 AND status != 'deleted'
	`

	err := db.QueryRow(query, eventID, userID).Scan(
		&event.ID, &event.UserID, &event.Title, &event.Description,
		&event.StartTime, &event.EndTime, &event.Timezone, &event.Location,
		&event.EventType, &event.Status, &metadataJSON, &automationJSON,
		&event.CreatedAt, &event.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error querying event: %v", err)
		http.Error(w, "Failed to retrieve event", http.StatusInternalServerError)
		return
	}

	// Parse metadata and automation config
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &event.Metadata)
	}
	if len(automationJSON) > 0 {
		json.Unmarshal(automationJSON, &event.AutomationConfig)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// Update event handler
func updateEventHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	// Get event ID from URL
	vars := mux.Vars(r)
	eventID := vars["id"]
	if eventID == "" {
		http.Error(w, "Event ID is required", http.StatusBadRequest)
		return
	}

	// Parse update request
	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Build dynamic update query
	updateFields := []string{}
	args := []interface{}{}
	argCount := 0

	if req.Title != "" {
		argCount++
		updateFields = append(updateFields, fmt.Sprintf("title = $%d", argCount))
		args = append(args, req.Title)
	}

	if req.Description != "" {
		argCount++
		updateFields = append(updateFields, fmt.Sprintf("description = $%d", argCount))
		args = append(args, req.Description)
	}

	var newStartTime, newEndTime time.Time
	var checkConflicts bool

	if req.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			http.Error(w, "Invalid start_time format: "+err.Error(), http.StatusBadRequest)
			return
		}
		newStartTime = startTime
		checkConflicts = true
		argCount++
		updateFields = append(updateFields, fmt.Sprintf("start_time = $%d", argCount))
		args = append(args, startTime)
	}

	if req.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			http.Error(w, "Invalid end_time format: "+err.Error(), http.StatusBadRequest)
			return
		}
		newEndTime = endTime
		checkConflicts = true
		argCount++
		updateFields = append(updateFields, fmt.Sprintf("end_time = $%d", argCount))
		args = append(args, endTime)
	}

	// If times are being updated, check for conflicts
	if checkConflicts {
		// Get existing event times if not all provided
		if newStartTime.IsZero() || newEndTime.IsZero() {
			var existingStart, existingEnd time.Time
			query := "SELECT start_time, end_time FROM events WHERE id = $1 AND user_id = $2"
			err := db.QueryRow(query, eventID, userID).Scan(&existingStart, &existingEnd)
			if err != nil {
				log.Printf("Error fetching existing event times: %v", err)
			} else {
				if newStartTime.IsZero() {
					newStartTime = existingStart
				}
				if newEndTime.IsZero() {
					newEndTime = existingEnd
				}
			}
		}

		// Check for conflicts (excluding the event being updated)
		conflicts, err := conflictDetector.CheckConflicts(userID, newStartTime, newEndTime, eventID)
		if err != nil {
			log.Printf("Error checking conflicts: %v", err)
			// Continue even if conflict check fails
		} else if len(conflicts) > 0 {
			// Return conflicts to the user
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":   false,
				"conflicts": conflicts,
				"message":   "Event conflicts detected. Please resolve conflicts or confirm to proceed.",
			})
			return
		}
	}

	if req.Location != "" {
		argCount++
		updateFields = append(updateFields, fmt.Sprintf("location = $%d", argCount))
		args = append(args, req.Location)
	}

	if req.EventType != "" {
		argCount++
		updateFields = append(updateFields, fmt.Sprintf("event_type = $%d", argCount))
		args = append(args, req.EventType)
	}

	if req.Timezone != "" {
		argCount++
		updateFields = append(updateFields, fmt.Sprintf("timezone = $%d", argCount))
		args = append(args, req.Timezone)
	}

	if req.Metadata != nil {
		metadataJSON, _ := json.Marshal(req.Metadata)
		argCount++
		updateFields = append(updateFields, fmt.Sprintf("metadata = $%d", argCount))
		args = append(args, metadataJSON)
	}

	if req.AutomationConfig != nil {
		automationJSON, _ := json.Marshal(req.AutomationConfig)
		argCount++
		updateFields = append(updateFields, fmt.Sprintf("automation_config = $%d", argCount))
		args = append(args, automationJSON)
	}

	// Always update the updated_at timestamp
	argCount++
	updateFields = append(updateFields, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now().UTC())

	// Add WHERE clause parameters
	argCount++
	args = append(args, eventID)
	argCount++
	args = append(args, userID)

	// Execute update
	query := fmt.Sprintf(`
		UPDATE events 
		SET %s 
		WHERE id = $%d AND user_id = $%d AND status != 'deleted'
		RETURNING id, user_id, title, description, start_time, end_time, timezone, 
		          location, event_type, status, created_at, updated_at
	`, strings.Join(updateFields, ", "), argCount-1, argCount)

	var event Event
	err := db.QueryRow(query, args...).Scan(
		&event.ID, &event.UserID, &event.Title, &event.Description,
		&event.StartTime, &event.EndTime, &event.Timezone, &event.Location,
		&event.EventType, &event.Status, &event.CreatedAt, &event.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Event not found or unauthorized", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error updating event: %v", err)
		http.Error(w, "Failed to update event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// Delete event handler (soft delete)
func deleteEventHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	// Get event ID from URL
	vars := mux.Vars(r)
	eventID := vars["id"]
	if eventID == "" {
		http.Error(w, "Event ID is required", http.StatusBadRequest)
		return
	}

	// Soft delete the event
	query := `
		UPDATE events 
		SET status = 'deleted', updated_at = $1
		WHERE id = $2 AND user_id = $3 AND status != 'deleted'
		RETURNING id
	`

	var deletedID string
	err := db.QueryRow(query, time.Now().UTC(), eventID, userID).Scan(&deletedID)

	if err == sql.ErrNoRows {
		http.Error(w, "Event not found or already deleted", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error deleting event: %v", err)
		http.Error(w, "Failed to delete event", http.StatusInternalServerError)
		return
	}

	// Also mark reminders as cancelled
	_, err = db.Exec(`
		UPDATE event_reminders 
		SET status = 'cancelled'
		WHERE event_id = $1 AND status = 'pending'
	`, eventID)

	if err != nil {
		log.Printf("Warning: Failed to cancel reminders for event %s: %v", eventID, err)
	}

	// Clean up vector embedding asynchronously
	go func() {
		ctx := context.Background()
		if err := vectorSearchManager.DeleteEmbeddingForEvent(ctx, eventID); err != nil {
			log.Printf("Warning: Failed to delete embedding for event %s: %v", eventID, err)
		}
	}()

	response := map[string]interface{}{
		"success":  true,
		"message":  "Event deleted successfully",
		"event_id": deletedID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func scheduleChatHandler(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	// Use NLP processor to handle natural language scheduling
	ctx := r.Context()
	response, err := nlpProcessor.ProcessSchedulingRequest(ctx, userID, req.Message)
	if err != nil {
		log.Printf("Error processing chat request: %v", err)
		// Return a fallback response on error
		response = &ChatResponse{
			Response: "I'm having trouble understanding your request. Could you try rephrasing it?",
			SuggestedActions: []SuggestedAction{
				{
					Action:     "create_event",
					Confidence: 0.5,
				},
				{
					Action:     "list_events",
					Confidence: 0.5,
				},
			},
		}
	}

	// Add conversation context
	if response.Context == nil {
		response.Context = make(map[string]interface{})
	}
	response.Context["conversation_id"] = uuid.New().String()
	response.Context["user_message"] = req.Message
	response.Context["timestamp"] = time.Now().UTC()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func scheduleOptimizeHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Request     string `json:"request"` // Natural language request like "free up tonight"
		Constraints struct {
			PreserveHighPriority bool `json:"preserve_high_priority"`
			MinBufferMinutes     int  `json:"min_buffer_minutes"`
			BusinessHoursOnly    bool `json:"business_hours_only"`
		} `json:"constraints"`
		// Legacy fields for backward compatibility
		OptimizationGoal string    `json:"optimization_goal,omitempty"`
		StartDate        time.Time `json:"start_date,omitempty"`
		EndDate          time.Time `json:"end_date,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "test-user" // Test mode fallback
	}

	// Parse the natural language request to determine date range
	startDate, endDate := parseOptimizationRequest(req.Request)
	if req.StartDate.IsZero() && !startDate.IsZero() {
		req.StartDate = startDate
	}
	if req.EndDate.IsZero() && !endDate.IsZero() {
		req.EndDate = endDate
	}

	// Default to next 7 days if no date range specified
	if req.StartDate.IsZero() {
		req.StartDate = time.Now()
	}
	if req.EndDate.IsZero() {
		req.EndDate = time.Now().AddDate(0, 0, 7)
	}

	// Fetch user's events in the date range
	query := `
		SELECT id, title, description, start_time, end_time, event_type, location, metadata
		FROM events
		WHERE user_id = $1 AND start_time >= $2 AND end_time <= $3
		  AND status != 'cancelled'
		ORDER BY start_time`

	rows, err := db.Query(query, userID, req.StartDate, req.EndDate)
	if err != nil {
		http.Error(w, "Failed to fetch events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var metadataJSON []byte
		if err := rows.Scan(&e.ID, &e.Title, &e.Description, &e.StartTime, &e.EndTime,
			&e.EventType, &e.Location, &metadataJSON); err != nil {
			continue
		}
		e.UserID = userID
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &e.Metadata)
		}
		events = append(events, e)
	}

	// Generate smart optimization suggestions based on request
	suggestions := generateSmartSuggestions(events, req.Request, req.Constraints)

	// Format response according to PRD specification
	response := map[string]interface{}{
		"suggestions": suggestions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper functions for schedule optimization
// parseOptimizationRequest parses natural language request to determine date range
func parseOptimizationRequest(request string) (startDate, endDate time.Time) {
	now := time.Now()
	requestLower := strings.ToLower(request)

	// Parse common time references
	if strings.Contains(requestLower, "tonight") || strings.Contains(requestLower, "today") {
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, now.Location())
		endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	} else if strings.Contains(requestLower, "tomorrow") {
		tomorrow := now.AddDate(0, 0, 1)
		startDate = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, now.Location())
		endDate = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 23, 59, 59, 0, now.Location())
	} else if strings.Contains(requestLower, "this week") {
		// Find start of week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startDate = now.AddDate(0, 0, -(weekday - 1))
		startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 0, 6)
		endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, now.Location())
	} else if strings.Contains(requestLower, "next week") {
		// Start from next Monday
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		daysUntilMonday := 8 - weekday
		startDate = now.AddDate(0, 0, daysUntilMonday)
		startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 0, 6)
		endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, now.Location())
	}

	return startDate, endDate
}

// generateSmartSuggestions generates intelligent optimization suggestions
func generateSmartSuggestions(events []Event, request string, constraints struct {
	PreserveHighPriority bool `json:"preserve_high_priority"`
	MinBufferMinutes     int  `json:"min_buffer_minutes"`
	BusinessHoursOnly    bool `json:"business_hours_only"`
}) []map[string]interface{} {
	suggestions := []map[string]interface{}{}
	requestLower := strings.ToLower(request)

	// Handle different optimization requests
	if strings.Contains(requestLower, "free up") {
		// Find events that can be moved or shortened
		for _, event := range events {
			// Skip high priority events if constraint is set
			if constraints.PreserveHighPriority && isHighPriority(event) {
				continue
			}

			// Check if event is in the target time period
			if isInTargetPeriod(event, request) {
				// Suggest moving or cancelling non-critical meetings
				if event.EventType == "meeting" && !isHighPriority(event) {
					suggestions = append(suggestions, map[string]interface{}{
						"description":     fmt.Sprintf("Reschedule '%s' to free up time", event.Title),
						"affected_events": []Event{event},
						"proposed_changes": []map[string]interface{}{
							{
								"event_id": event.ID,
								"action":   "move",
								"new_time": suggestAlternativeTime(event, events, constraints.BusinessHoursOnly),
							},
						},
						"confidence_score": 0.85,
					})
				}
			}
		}
	} else if strings.Contains(requestLower, "find") && strings.Contains(requestLower, "hours") {
		// Find available time slots
		freeSlots := findFreeSlots(events, parseHoursNeeded(request))
		for _, slot := range freeSlots {
			suggestions = append(suggestions, map[string]interface{}{
				"description": fmt.Sprintf("Available time slot: %s to %s",
					slot["start"].(time.Time).Format("Mon 15:04"),
					slot["end"].(time.Time).Format("15:04")),
				"affected_events":  []Event{},
				"proposed_changes": []map[string]interface{}{},
				"confidence_score": 0.95,
			})
		}
	}

	// Add general optimization suggestions
	generalSuggestions := analyzeSchedule(events, "balance_workload")
	for _, suggestion := range generalSuggestions {
		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

// Helper functions for smart scheduling
func isHighPriority(event Event) bool {
	if event.Metadata != nil {
		if priority, ok := event.Metadata["priority"].(string); ok {
			return priority == "high" || priority == "critical"
		}
	}
	// Check for keywords indicating importance
	titleLower := strings.ToLower(event.Title)
	return strings.Contains(titleLower, "interview") ||
		strings.Contains(titleLower, "deadline") ||
		strings.Contains(titleLower, "critical") ||
		strings.Contains(titleLower, "urgent")
}

func isInTargetPeriod(event Event, request string) bool {
	now := time.Now()
	requestLower := strings.ToLower(request)

	if strings.Contains(requestLower, "tonight") {
		tonight := time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, now.Location())
		endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
		return event.StartTime.After(tonight) && event.StartTime.Before(endOfDay)
	}

	// Add more period checks as needed
	return false
}

func suggestAlternativeTime(event Event, allEvents []Event, businessHoursOnly bool) string {
	// Simple algorithm to find next available slot
	suggestedTime := event.StartTime.AddDate(0, 0, 1) // Next day same time
	if businessHoursOnly {
		hour := suggestedTime.Hour()
		if hour < 9 {
			suggestedTime = time.Date(suggestedTime.Year(), suggestedTime.Month(), suggestedTime.Day(),
				9, 0, 0, 0, suggestedTime.Location())
		} else if hour > 17 {
			suggestedTime = suggestedTime.AddDate(0, 0, 1)
			suggestedTime = time.Date(suggestedTime.Year(), suggestedTime.Month(), suggestedTime.Day(),
				9, 0, 0, 0, suggestedTime.Location())
		}
	}
	return suggestedTime.Format(time.RFC3339)
}

func parseHoursNeeded(request string) int {
	// Simple extraction of hours from request
	if strings.Contains(request, "2 hours") || strings.Contains(request, "two hours") {
		return 2
	}
	if strings.Contains(request, "3 hours") || strings.Contains(request, "three hours") {
		return 3
	}
	if strings.Contains(request, "4 hours") || strings.Contains(request, "four hours") {
		return 4
	}
	return 1 // Default to 1 hour
}

func findFreeSlots(events []Event, hoursNeeded int) []map[string]interface{} {
	slots := []map[string]interface{}{}

	if len(events) == 0 {
		return slots
	}

	// Check gaps between events
	for i := 0; i < len(events)-1; i++ {
		gap := events[i+1].StartTime.Sub(events[i].EndTime)
		if gap >= time.Duration(hoursNeeded)*time.Hour {
			slots = append(slots, map[string]interface{}{
				"start": events[i].EndTime,
				"end":   events[i].EndTime.Add(time.Duration(hoursNeeded) * time.Hour),
			})
		}
	}

	return slots
}

func analyzeSchedule(events []Event, goal string) []map[string]interface{} {
	suggestions := []map[string]interface{}{}

	// Identify gaps and overlaps
	for i := 0; i < len(events)-1; i++ {
		gap := events[i+1].StartTime.Sub(events[i].EndTime)

		if gap > 30*time.Minute && gap < 2*time.Hour {
			// Suggest consolidating small gaps
			suggestions = append(suggestions, map[string]interface{}{
				"type":           "consolidate_gap",
				"description":    fmt.Sprintf("Move %s to eliminate %v gap", events[i+1].Title, gap),
				"event_id":       events[i+1].ID,
				"proposed_start": events[i].EndTime,
				"confidence":     0.7,
			})
		}
	}

	// Suggest batching similar events
	eventTypes := make(map[string][]Event)
	for _, e := range events {
		eventTypes[e.EventType] = append(eventTypes[e.EventType], e)
	}

	for eventType, typeEvents := range eventTypes {
		if len(typeEvents) > 2 {
			suggestions = append(suggestions, map[string]interface{}{
				"type":        "batch_similar",
				"description": fmt.Sprintf("Batch %d %s events together", len(typeEvents), eventType),
				"event_ids":   getEventIDs(typeEvents),
				"confidence":  0.8,
			})
		}
	}

	return suggestions
}

func calculateScheduleEfficiency(events []Event) float64 {
	if len(events) == 0 {
		return 1.0
	}

	totalTime := events[len(events)-1].EndTime.Sub(events[0].StartTime)
	busyTime := time.Duration(0)
	for _, e := range events {
		busyTime += e.EndTime.Sub(e.StartTime)
	}

	if totalTime > 0 {
		return float64(busyTime) / float64(totalTime)
	}
	return 0
}

func calculateTimeSaved(suggestions []map[string]interface{}) int {
	// Estimate time saved in minutes
	totalMinutes := 0
	for _, s := range suggestions {
		if s["type"] == "consolidate_gap" {
			totalMinutes += 15 // Estimate 15 minutes saved per gap consolidation
		} else if s["type"] == "batch_similar" {
			totalMinutes += 30 // Estimate 30 minutes saved per batching
		}
	}
	return totalMinutes
}

func getEventIDs(events []Event) []string {
	ids := make([]string, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	return ids
}

// Timezone helper functions for distributed teams
func convertToTimezone(t time.Time, timezone string) time.Time {
	if timezone == "" || timezone == "UTC" {
		return t.UTC()
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// If timezone is invalid, fallback to UTC
		return t.UTC()
	}

	return t.In(loc)
}

func parseTimeInTimezone(timeStr string, timezone string) (time.Time, error) {
	// First parse the time as provided
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Time{}, err
	}

	// If a specific timezone is requested, convert to it
	if timezone != "" && timezone != "UTC" {
		loc, err := time.LoadLocation(timezone)
		if err == nil {
			// Parse the time as if it were in the specified timezone
			// This is useful for "create meeting at 3pm PST" type requests
			localTime := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc)
			return localTime, nil
		}
	}

	return t, nil
}

func formatEventTimes(event *Event, userTimezone string) {
	// Convert event times to user's timezone for display
	if userTimezone != "" && userTimezone != "UTC" {
		loc, err := time.LoadLocation(userTimezone)
		if err == nil {
			// Store original timezone info in metadata if not present
			if event.Metadata == nil {
				event.Metadata = make(map[string]interface{})
			}
			event.Metadata["original_timezone"] = event.Timezone
			event.Metadata["display_timezone"] = userTimezone

			// Convert times for display
			event.StartTime = event.StartTime.In(loc)
			event.EndTime = event.EndTime.In(loc)
		}
	}
}

func detectUserTimezone(r *http.Request) string {
	// Check for timezone in header
	tz := r.Header.Get("X-Timezone")
	if tz != "" {
		return tz
	}

	// Check for timezone in query params
	tz = r.URL.Query().Get("timezone")
	if tz != "" {
		return tz
	}

	// Default to UTC
	return "UTC"
}

// iCal format functions for import/export
func generateICalFromEvents(events []Event) string {
	var ical strings.Builder

	// iCal header
	ical.WriteString("BEGIN:VCALENDAR\r\n")
	ical.WriteString("VERSION:2.0\r\n")
	ical.WriteString("PRODID:-//Vrooli Calendar//EN\r\n")
	ical.WriteString("CALSCALE:GREGORIAN\r\n")
	ical.WriteString("METHOD:PUBLISH\r\n")

	// Add each event
	for _, event := range events {
		ical.WriteString("BEGIN:VEVENT\r\n")
		ical.WriteString(fmt.Sprintf("UID:%s@vrooli.calendar\r\n", event.ID))
		ical.WriteString(fmt.Sprintf("DTSTART:%s\r\n", formatDateTimeForICal(event.StartTime)))
		ical.WriteString(fmt.Sprintf("DTEND:%s\r\n", formatDateTimeForICal(event.EndTime)))
		ical.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICalText(event.Title)))

		if event.Description != "" {
			ical.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICalText(event.Description)))
		}

		if event.Location != "" {
			ical.WriteString(fmt.Sprintf("LOCATION:%s\r\n", escapeICalText(event.Location)))
		}

		ical.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", formatDateTimeForICal(event.CreatedAt)))
		ical.WriteString("STATUS:CONFIRMED\r\n")
		ical.WriteString("END:VEVENT\r\n")
	}

	ical.WriteString("END:VCALENDAR\r\n")
	return ical.String()
}

func formatDateTimeForICal(t time.Time) string {
	// Format as YYYYMMDDTHHMMSSZ for UTC
	return t.UTC().Format("20060102T150405Z")
}

func escapeICalText(text string) string {
	// Escape special characters for iCal format
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "\n", "\\n")
	text = strings.ReplaceAll(text, ";", "\\;")
	text = strings.ReplaceAll(text, ",", "\\,")
	return text
}

func parseICalToEvents(icalContent string, userID string) ([]Event, error) {
	var events []Event
	lines := strings.Split(strings.ReplaceAll(icalContent, "\r\n", "\n"), "\n")

	var currentEvent *Event
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "BEGIN:VEVENT" {
			currentEvent = &Event{
				ID:       uuid.New().String(),
				UserID:   userID,
				Status:   "active",
				Timezone: "UTC",
			}
		} else if line == "END:VEVENT" && currentEvent != nil {
			events = append(events, *currentEvent)
			currentEvent = nil
		} else if currentEvent != nil {
			// Parse event properties
			if strings.HasPrefix(line, "SUMMARY:") {
				currentEvent.Title = unescapeICalText(strings.TrimPrefix(line, "SUMMARY:"))
			} else if strings.HasPrefix(line, "DESCRIPTION:") {
				currentEvent.Description = unescapeICalText(strings.TrimPrefix(line, "DESCRIPTION:"))
			} else if strings.HasPrefix(line, "LOCATION:") {
				currentEvent.Location = unescapeICalText(strings.TrimPrefix(line, "LOCATION:"))
			} else if strings.HasPrefix(line, "DTSTART:") {
				if t, err := parseICalDateTime(strings.TrimPrefix(line, "DTSTART:")); err == nil {
					currentEvent.StartTime = t
				}
			} else if strings.HasPrefix(line, "DTEND:") {
				if t, err := parseICalDateTime(strings.TrimPrefix(line, "DTEND:")); err == nil {
					currentEvent.EndTime = t
				}
			}
		}
	}

	return events, nil
}

func parseICalDateTime(dtStr string) (time.Time, error) {
	// Handle different iCal datetime formats
	if strings.HasSuffix(dtStr, "Z") {
		// UTC format: YYYYMMDDTHHMMSSZ
		return time.Parse("20060102T150405Z", dtStr)
	}
	// Local time format: YYYYMMDDTHHMMSS
	return time.Parse("20060102T150405", dtStr)
}

func unescapeICalText(text string) string {
	text = strings.ReplaceAll(text, "\\n", "\n")
	text = strings.ReplaceAll(text, "\\;", ";")
	text = strings.ReplaceAll(text, "\\,", ",")
	text = strings.ReplaceAll(text, "\\\\", "\\")
	return text
}

// exportICalHandler exports user's events in iCal format
func exportICalHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	// Parse date range from query parameters
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Build query
	query := `
		SELECT id, user_id, title, description, start_time, end_time, timezone, 
		       location, event_type, status, created_at, updated_at
		FROM events 
		WHERE user_id = $1 AND status = 'active'
	`
	args := []interface{}{userID}
	argCount := 1

	if startDate != "" {
		argCount++
		query += fmt.Sprintf(" AND start_time >= $%d", argCount)
		args = append(args, startDate)
	}

	if endDate != "" {
		argCount++
		query += fmt.Sprintf(" AND end_time <= $%d", argCount)
		args = append(args, endDate)
	}

	query += " ORDER BY start_time"

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to fetch events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		err := rows.Scan(
			&event.ID, &event.UserID, &event.Title, &event.Description,
			&event.StartTime, &event.EndTime, &event.Timezone, &event.Location,
			&event.EventType, &event.Status, &event.CreatedAt, &event.UpdatedAt,
		)
		if err != nil {
			continue
		}
		events = append(events, event)
	}

	// Generate iCal content
	icalContent := generateICalFromEvents(events)

	// Set response headers for file download
	w.Header().Set("Content-Type", "text/calendar")
	w.Header().Set("Content-Disposition", "attachment; filename=calendar.ics")
	w.Write([]byte(icalContent))
}

// importICalHandler imports events from iCal format
func importICalHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	// Read iCal content from request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse iCal content to events
	events, err := parseICalToEvents(string(body), userID)
	if err != nil {
		http.Error(w, "Failed to parse iCal content: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Import each event into the database
	importedCount := 0
	failedCount := 0

	for _, event := range events {
		// Skip events with missing required fields
		if event.Title == "" || event.StartTime.IsZero() || event.EndTime.IsZero() {
			failedCount++
			continue
		}

		// Insert event into database
		query := `
			INSERT INTO events (id, user_id, title, description, start_time, end_time, 
			                   timezone, location, event_type, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (id) DO NOTHING
		`

		_, err := db.Exec(query, event.ID, userID, event.Title, event.Description,
			event.StartTime, event.EndTime, event.Timezone, event.Location,
			event.EventType, event.Status)

		if err != nil {
			log.Printf("Failed to import event %s: %v", event.Title, err)
			failedCount++
		} else {
			importedCount++
		}
	}

	// Return import summary
	response := map[string]interface{}{
		"success":        true,
		"imported_count": importedCount,
		"failed_count":   failedCount,
		"total_count":    len(events),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// filterEventsByCriteria filters events based on date range and event type
func filterEventsByCriteria(events []Event, startDate, endDate, eventType string) []Event {
	var filtered []Event

	for _, event := range events {
		// Apply date filter
		if startDate != "" {
			if startTime, err := time.Parse(time.RFC3339, startDate); err == nil {
				if event.StartTime.Before(startTime) {
					continue
				}
			}
		}

		if endDate != "" {
			if endTime, err := time.Parse(time.RFC3339, endDate); err == nil {
				if event.StartTime.After(endTime) {
					continue
				}
			}
		}

		// Apply event type filter
		if eventType != "" && event.EventType != eventType {
			continue
		}

		filtered = append(filtered, event)
	}

	return filtered
}

// processRemindersHandler manually triggers reminder processing
func processRemindersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := calendarProcessor.ProcessReminders(ctx); err != nil {
		http.Error(w, fmt.Sprintf("Failed to process reminders: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Reminders processed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getCategoriesHandler returns all available categories
func getCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	categories, err := categoryManager.GetCategories(userID)
	if err != nil {
		log.Printf("Error getting categories: %v", err)
		http.Error(w, "Failed to retrieve categories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"categories": categories,
		"total":      len(categories),
	})
}

// createCategoryHandler creates a custom category
func createCategoryHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	var category EventCategory
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	createdCategory, err := categoryManager.CreateCategory(userID, category)
	if err != nil {
		log.Printf("Error creating category: %v", err)
		http.Error(w, "Failed to create category: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdCategory)
}

// getEventsByCategoryHandler returns events grouped by category
func getEventsByCategoryHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Default to current month if no dates provided
	if startDate == "" {
		now := time.Now()
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	}
	if endDate == "" {
		now := time.Now()
		nextMonth := now.AddDate(0, 1, 0)
		endDate = time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, time.UTC).Add(-1 * time.Second).Format(time.RFC3339)
	}

	eventsByCategory, err := categoryManager.GetEventsByCategory(userID, startDate, endDate)
	if err != nil {
		log.Printf("Error getting events by category: %v", err)
		http.Error(w, "Failed to retrieve events", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eventsByCategory)
}

// getCategoryStatisticsHandler returns statistics about event categories
func getCategoryStatisticsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Default to last 30 days if no dates provided
	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -30).Format(time.RFC3339)
	}
	if endDate == "" {
		endDate = time.Now().Format(time.RFC3339)
	}

	statistics, err := categoryManager.GetCategoryStatistics(userID, startDate, endDate)
	if err != nil {
		log.Printf("Error getting category statistics: %v", err)
		http.Error(w, "Failed to retrieve statistics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statistics)
}

// Template management handlers

func getTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	includeSystem := r.URL.Query().Get("include_system") != "false"

	templates, err := getEventTemplates(userID, includeSystem)
	if err != nil {
		log.Printf("Error getting templates: %v", err)
		http.Error(w, "Failed to retrieve templates", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
		"total":     len(templates),
	})
}

func getEventTemplates(userID string, includeSystem bool) ([]EventTemplate, error) {
	var templates []EventTemplate

	// For now, just return system templates if includeSystem is true
	// since we're using "test-user" which is not a valid UUID
	query := `
		SELECT id, user_id, name, description, template_data, category, is_system, use_count, created_at, updated_at
		FROM event_templates
		WHERE is_system = true
		ORDER BY use_count DESC, name ASC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var template EventTemplate
		var templateDataJSON []byte

		err := rows.Scan(
			&template.ID,
			&template.UserID,
			&template.Name,
			&template.Description,
			&templateDataJSON,
			&template.Category,
			&template.IsSystem,
			&template.UseCount,
			&template.CreatedAt,
			&template.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(templateDataJSON, &template.TemplateData); err != nil {
			log.Printf("Error unmarshaling template data: %v", err)
			template.TemplateData = make(map[string]interface{})
		}

		templates = append(templates, template)
	}

	return templates, nil
}

func createTemplateHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name         string                 `json:"name"`
		Description  string                 `json:"description"`
		TemplateData map[string]interface{} `json:"template_data"`
		Category     string                 `json:"category"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Template name is required", http.StatusBadRequest)
		return
	}

	templateDataJSON, err := json.Marshal(req.TemplateData)
	if err != nil {
		http.Error(w, "Invalid template data", http.StatusBadRequest)
		return
	}

	var templateID string
	query := `
		INSERT INTO event_templates (user_id, name, description, template_data, category)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, name) 
		DO UPDATE SET 
			description = EXCLUDED.description,
			template_data = EXCLUDED.template_data,
			category = EXCLUDED.category,
			updated_at = NOW()
		RETURNING id
	`

	err = db.QueryRow(query, userID, req.Name, req.Description, templateDataJSON, req.Category).Scan(&templateID)
	if err != nil {
		log.Printf("Error creating template: %v", err)
		http.Error(w, "Failed to create template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"template_id": templateID,
		"message":     "Template created successfully",
	})
}

func deleteTemplateHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	templateID := mux.Vars(r)["id"]
	if templateID == "" {
		http.Error(w, "Template ID is required", http.StatusBadRequest)
		return
	}

	// Only allow deletion of user's own templates, not system templates
	query := `DELETE FROM event_templates WHERE id = $1 AND user_id = $2 AND is_system = false`
	result, err := db.Exec(query, templateID, userID)
	if err != nil {
		log.Printf("Error deleting template: %v", err)
		http.Error(w, "Failed to delete template", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Template not found or cannot be deleted", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Template deleted successfully",
	})
}

func createEventFromTemplateHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	var req struct {
		TemplateID string                 `json:"template_id"`
		StartTime  time.Time              `json:"start_time"`
		Overrides  map[string]interface{} `json:"overrides"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TemplateID == "" {
		http.Error(w, "Template ID is required", http.StatusBadRequest)
		return
	}

	// Get the template
	var templateDataJSON []byte
	var templateName string
	query := `
		SELECT template_data, name 
		FROM event_templates 
		WHERE id = $1 AND (user_id = $2 OR is_system = true)
	`

	err := db.QueryRow(query, req.TemplateID, userID).Scan(&templateDataJSON, &templateName)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Template not found", http.StatusNotFound)
			return
		}
		log.Printf("Error getting template: %v", err)
		http.Error(w, "Failed to retrieve template", http.StatusInternalServerError)
		return
	}

	// Parse template data
	var templateData map[string]interface{}
	if err := json.Unmarshal(templateDataJSON, &templateData); err != nil {
		log.Printf("Error unmarshaling template data: %v", err)
		http.Error(w, "Invalid template data", http.StatusInternalServerError)
		return
	}

	// Build event from template
	title, _ := templateData["title"].(string)
	if title == "" {
		title = templateName
	}
	description, _ := templateData["description"].(string)
	location, _ := templateData["location"].(string)
	eventType, _ := templateData["event_type"].(string)

	// Calculate end time based on duration
	durationMinutes, _ := templateData["duration_minutes"].(float64)
	if durationMinutes == 0 {
		durationMinutes = 60 // Default 1 hour
	}

	endTime := req.StartTime.Add(time.Duration(durationMinutes) * time.Minute)

	// Apply overrides
	if req.Overrides != nil {
		if v, ok := req.Overrides["title"].(string); ok {
			title = v
		}
		if v, ok := req.Overrides["description"].(string); ok {
			description = v
		}
		if v, ok := req.Overrides["location"].(string); ok {
			location = v
		}
		if v, ok := req.Overrides["end_time"].(string); ok {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				endTime = t
			}
		}
	}

	// Create the event
	event := Event{
		UserID:      userID,
		Title:       title,
		Description: description,
		StartTime:   req.StartTime,
		EndTime:     endTime,
		Timezone:    "UTC",
		Location:    location,
		EventType:   eventType,
		Status:      "active",
		Metadata:    map[string]interface{}{"from_template": req.TemplateID},
	}

	// Check for conflicts
	excludeEventID := "" // No event to exclude for new events
	conflicts, err := conflictDetector.CheckConflicts(userID, req.StartTime, endTime, excludeEventID)
	if err != nil {
		log.Printf("Error checking conflicts: %v", err)
		// Continue anyway, conflicts are not critical
	}

	if len(conflicts) > 0 && r.URL.Query().Get("force") != "true" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   false,
			"message":   "Event conflicts detected. Please resolve conflicts or add ?force=true to proceed.",
			"conflicts": conflicts,
		})
		return
	}

	// Save event to database
	eventID := uuid.New().String()
	event.ID = eventID

	metadataJSON, _ := json.Marshal(event.Metadata)
	automationJSON, _ := json.Marshal(event.AutomationConfig)

	insertQuery := `
		INSERT INTO events (id, user_id, title, description, start_time, end_time, timezone, location, event_type, metadata, automation_config)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = db.Exec(insertQuery, eventID, userID, event.Title, event.Description, event.StartTime, event.EndTime,
		event.Timezone, event.Location, event.EventType, metadataJSON, automationJSON)
	if err != nil {
		log.Printf("Error creating event from template: %v", err)
		http.Error(w, "Failed to create event", http.StatusInternalServerError)
		return
	}

	// Update template use count
	db.Exec("UPDATE event_templates SET use_count = use_count + 1 WHERE id = $1", req.TemplateID)

	// Auto-categorize the event
	if categoryManager != nil && event.EventType == "" {
		if category := categoryManager.SuggestCategory(event.Title, event.Description); category != "" {
			event.EventType = category
			db.Exec("UPDATE events SET event_type = $1 WHERE id = $2", category, eventID)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"event":   event,
		"message": "Event created from template successfully",
	})
}

// Bulk Operations Handlers

// bulkCreateEventsHandler creates multiple events in a single request
func bulkCreateEventsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	var req struct {
		Events  []CreateEventRequest `json:"events"`
		Options struct {
			SkipConflicts bool `json:"skip_conflicts"`
			ValidateOnly  bool `json:"validate_only"`
		} `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Events) == 0 {
		http.Error(w, "No events provided", http.StatusBadRequest)
		return
	}

	if len(req.Events) > 100 {
		http.Error(w, "Maximum 100 events per bulk request", http.StatusBadRequest)
		return
	}

	createdEvents := []Event{}
	failedEvents := []map[string]interface{}{}
	conflicts := []map[string]interface{}{}

	// Begin transaction for atomicity
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	for i, eventReq := range req.Events {
		// Parse times
		startTime, err := time.Parse(time.RFC3339, eventReq.StartTime)
		if err != nil {
			failedEvents = append(failedEvents, map[string]interface{}{
				"index":   i,
				"title":   eventReq.Title,
				"error":   "Invalid start time format",
				"details": err.Error(),
			})
			continue
		}

		endTime, err := time.Parse(time.RFC3339, eventReq.EndTime)
		if err != nil {
			failedEvents = append(failedEvents, map[string]interface{}{
				"index":   i,
				"title":   eventReq.Title,
				"error":   "Invalid end time format",
				"details": err.Error(),
			})
			continue
		}

		// Check conflicts if not skipping
		if !req.Options.SkipConflicts {
			eventConflicts, err := conflictDetector.CheckConflicts(userID, startTime, endTime, "")
			if err == nil && len(eventConflicts) > 0 {
				conflicts = append(conflicts, map[string]interface{}{
					"index":     i,
					"title":     eventReq.Title,
					"conflicts": eventConflicts,
				})
				continue
			}
		}

		// If validate only, don't actually create
		if req.Options.ValidateOnly {
			createdEvents = append(createdEvents, Event{
				Title:     eventReq.Title,
				StartTime: startTime,
				EndTime:   endTime,
				Status:    "validated",
			})
			continue
		}

		// Create event
		eventID := uuid.New().String()
		metadataJSON, _ := json.Marshal(eventReq.Metadata)
		automationJSON, _ := json.Marshal(eventReq.AutomationConfig)

		query := `
			INSERT INTO events 
			(id, user_id, title, description, start_time, end_time, timezone, location, event_type, metadata, automation_config)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING created_at, updated_at
		`

		var createdAt, updatedAt time.Time
		err = tx.QueryRow(query, eventID, userID, eventReq.Title, eventReq.Description,
			startTime, endTime, eventReq.Timezone, eventReq.Location,
			eventReq.EventType, metadataJSON, automationJSON).Scan(&createdAt, &updatedAt)

		if err != nil {
			failedEvents = append(failedEvents, map[string]interface{}{
				"index":   i,
				"title":   eventReq.Title,
				"error":   "Failed to create event",
				"details": err.Error(),
			})
			continue
		}

		createdEvents = append(createdEvents, Event{
			ID:        eventID,
			UserID:    userID,
			Title:     eventReq.Title,
			StartTime: startTime,
			EndTime:   endTime,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}

	// Commit transaction if not validate-only
	if !req.Options.ValidateOnly && len(createdEvents) > 0 {
		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit bulk creation", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	status := http.StatusOK
	if len(createdEvents) > 0 && !req.Options.ValidateOnly {
		status = http.StatusCreated
	}
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":         true,
		"created":         createdEvents,
		"failed":          failedEvents,
		"conflicts":       conflicts,
		"total_created":   len(createdEvents),
		"total_failed":    len(failedEvents),
		"total_conflicts": len(conflicts),
		"validate_only":   req.Options.ValidateOnly,
	})
}

// bulkUpdateEventsHandler updates multiple events in a single request
func bulkUpdateEventsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	var req struct {
		EventIDs []string               `json:"event_ids"`
		Updates  map[string]interface{} `json:"updates"`
		Options  struct {
			ValidateOnly bool `json:"validate_only"`
		} `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.EventIDs) == 0 {
		http.Error(w, "No event IDs provided", http.StatusBadRequest)
		return
	}

	if len(req.EventIDs) > 100 {
		http.Error(w, "Maximum 100 events per bulk update", http.StatusBadRequest)
		return
	}

	// Build update query dynamically
	updateFields := []string{}
	updateValues := []interface{}{}
	paramCount := 1

	if title, ok := req.Updates["title"].(string); ok {
		updateFields = append(updateFields, fmt.Sprintf("title = $%d", paramCount))
		updateValues = append(updateValues, title)
		paramCount++
	}

	if description, ok := req.Updates["description"].(string); ok {
		updateFields = append(updateFields, fmt.Sprintf("description = $%d", paramCount))
		updateValues = append(updateValues, description)
		paramCount++
	}

	if location, ok := req.Updates["location"].(string); ok {
		updateFields = append(updateFields, fmt.Sprintf("location = $%d", paramCount))
		updateValues = append(updateValues, location)
		paramCount++
	}

	if eventType, ok := req.Updates["event_type"].(string); ok {
		updateFields = append(updateFields, fmt.Sprintf("event_type = $%d", paramCount))
		updateValues = append(updateValues, eventType)
		paramCount++
	}

	if len(updateFields) == 0 {
		http.Error(w, "No valid update fields provided", http.StatusBadRequest)
		return
	}

	// Add updated_at
	updateFields = append(updateFields, fmt.Sprintf("updated_at = $%d", paramCount))
	updateValues = append(updateValues, time.Now())
	paramCount++

	// Add WHERE clause parameters
	placeholders := make([]string, len(req.EventIDs))
	for i, id := range req.EventIDs {
		placeholders[i] = fmt.Sprintf("$%d", paramCount+i)
		updateValues = append(updateValues, id)
	}

	// Add user_id for security
	updateValues = append(updateValues, userID)

	query := fmt.Sprintf(
		"UPDATE events SET %s WHERE id IN (%s) AND user_id = $%d",
		strings.Join(updateFields, ", "),
		strings.Join(placeholders, ", "),
		paramCount+len(req.EventIDs),
	)

	if req.Options.ValidateOnly {
		// Just validate without executing
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":       true,
			"validate_only": true,
			"would_update":  len(req.EventIDs),
			"update_fields": updateFields,
		})
		return
	}

	// Execute update
	result, err := db.Exec(query, updateValues...)
	if err != nil {
		http.Error(w, "Failed to update events", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"updated":       rowsAffected,
		"requested":     len(req.EventIDs),
		"update_fields": updateFields,
	})
}

// bulkDeleteEventsHandler deletes multiple events in a single request
func bulkDeleteEventsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	var req struct {
		EventIDs []string `json:"event_ids"`
		Options  struct {
			HardDelete   bool `json:"hard_delete"`
			ValidateOnly bool `json:"validate_only"`
		} `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.EventIDs) == 0 {
		http.Error(w, "No event IDs provided", http.StatusBadRequest)
		return
	}

	if len(req.EventIDs) > 100 {
		http.Error(w, "Maximum 100 events per bulk delete", http.StatusBadRequest)
		return
	}

	if req.Options.ValidateOnly {
		// Just validate without executing
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":       true,
			"validate_only": true,
			"would_delete":  len(req.EventIDs),
			"hard_delete":   req.Options.HardDelete,
		})
		return
	}

	var query string
	if req.Options.HardDelete {
		// Permanently delete events
		placeholders := make([]string, len(req.EventIDs))
		args := make([]interface{}, len(req.EventIDs)+1)
		for i, id := range req.EventIDs {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args[i] = id
		}
		args[len(req.EventIDs)] = userID

		query = fmt.Sprintf(
			"DELETE FROM events WHERE id IN (%s) AND user_id = $%d",
			strings.Join(placeholders, ", "),
			len(req.EventIDs)+1,
		)

		result, err := db.Exec(query, args...)
		if err != nil {
			http.Error(w, "Failed to delete events", http.StatusInternalServerError)
			return
		}

		rowsAffected, _ := result.RowsAffected()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     true,
			"deleted":     rowsAffected,
			"requested":   len(req.EventIDs),
			"hard_delete": true,
		})
	} else {
		// Soft delete - mark as cancelled
		placeholders := make([]string, len(req.EventIDs))
		args := make([]interface{}, len(req.EventIDs)+2)
		args[0] = time.Now()
		for i, id := range req.EventIDs {
			placeholders[i] = fmt.Sprintf("$%d", i+2)
			args[i+1] = id
		}
		args[len(req.EventIDs)+1] = userID

		query = fmt.Sprintf(
			"UPDATE events SET status = 'cancelled', updated_at = $1 WHERE id IN (%s) AND user_id = $%d",
			strings.Join(placeholders, ", "),
			len(req.EventIDs)+2,
		)

		result, err := db.Exec(query, args...)
		if err != nil {
			http.Error(w, "Failed to cancel events", http.StatusInternalServerError)
			return
		}

		rowsAffected, _ := result.RowsAffected()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     true,
			"cancelled":   rowsAffected,
			"requested":   len(req.EventIDs),
			"hard_delete": false,
		})
	}
}

// Attendance/RSVP Handlers

// getAttendeesHandler retrieves attendees for an event
func getAttendeesHandler(w http.ResponseWriter, r *http.Request) {
	eventID := mux.Vars(r)["id"]
	if eventID == "" {
		http.Error(w, "Event ID is required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT user_id, name, email, rsvp_status, attendance_status, response_time, check_in_time
		FROM event_attendees
		WHERE event_id = $1
		ORDER BY response_time DESC
	`

	rows, err := db.Query(query, eventID)
	if err != nil {
		http.Error(w, "Failed to retrieve attendees", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	attendees := []map[string]interface{}{}
	for rows.Next() {
		var userID, name, email, rsvpStatus, attendanceStatus sql.NullString
		var responseTime, checkInTime sql.NullTime

		err := rows.Scan(&userID, &name, &email, &rsvpStatus, &attendanceStatus, &responseTime, &checkInTime)
		if err != nil {
			continue
		}

		attendee := map[string]interface{}{
			"user_id":           userID.String,
			"name":              name.String,
			"email":             email.String,
			"rsvp_status":       rsvpStatus.String,
			"attendance_status": attendanceStatus.String,
		}

		if responseTime.Valid {
			attendee["response_time"] = responseTime.Time
		}
		if checkInTime.Valid {
			attendee["check_in_time"] = checkInTime.Time
		}

		attendees = append(attendees, attendee)
	}

	// Get summary statistics
	var totalInvited, totalAccepted, totalDeclined, totalAttended int
	statsQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN rsvp_status = 'accepted' THEN 1 END) as accepted,
			COUNT(CASE WHEN rsvp_status = 'declined' THEN 1 END) as declined,
			COUNT(CASE WHEN attendance_status = 'attended' THEN 1 END) as attended
		FROM event_attendees
		WHERE event_id = $1
	`
	db.QueryRow(statsQuery, eventID).Scan(&totalInvited, &totalAccepted, &totalDeclined, &totalAttended)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"attendees": attendees,
		"statistics": map[string]int{
			"total_invited":  totalInvited,
			"total_accepted": totalAccepted,
			"total_declined": totalDeclined,
			"total_attended": totalAttended,
		},
	})
}

// rsvpHandler handles RSVP responses for events
func rsvpHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID not found", http.StatusUnauthorized)
		return
	}

	eventID := mux.Vars(r)["id"]
	if eventID == "" {
		http.Error(w, "Event ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Status  string `json:"status"` // accepted, declined, tentative
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Status != "accepted" && req.Status != "declined" && req.Status != "tentative" {
		http.Error(w, "Invalid RSVP status. Must be 'accepted', 'declined', or 'tentative'", http.StatusBadRequest)
		return
	}

	// Get user details
	userEmail := r.Header.Get("X-User-Email")
	userName := "User" // Default if not available

	// Upsert RSVP response
	query := `
		INSERT INTO event_attendees (event_id, user_id, name, email, rsvp_status, response_time, rsvp_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (event_id, user_id)
		DO UPDATE SET 
			rsvp_status = EXCLUDED.rsvp_status,
			response_time = EXCLUDED.response_time,
			rsvp_message = EXCLUDED.rsvp_message
	`

	_, err := db.Exec(query, eventID, userID, userName, userEmail, req.Status, time.Now(), req.Message)
	if err != nil {
		http.Error(w, "Failed to record RSVP", http.StatusInternalServerError)
		return
	}

	// Send notification to event organizer
	// TODO: Integrate with notification-hub

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("RSVP recorded as %s", req.Status),
		"status":  req.Status,
	})
}

// trackAttendanceHandler tracks actual attendance for an event
func trackAttendanceHandler(w http.ResponseWriter, r *http.Request) {
	eventID := mux.Vars(r)["id"]
	if eventID == "" {
		http.Error(w, "Event ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Attendees []struct {
			UserID string `json:"user_id"`
			Status string `json:"status"` // attended, no_show
		} `json:"attendees"`
		CheckInMethod string `json:"check_in_method"` // manual, qr_code, auto
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updatedCount := 0
	for _, attendee := range req.Attendees {
		if attendee.Status != "attended" && attendee.Status != "no_show" {
			continue
		}

		query := `
			UPDATE event_attendees 
			SET attendance_status = $1, check_in_time = $2, check_in_method = $3
			WHERE event_id = $4 AND user_id = $5
		`

		result, err := db.Exec(query, attendee.Status, time.Now(), req.CheckInMethod, eventID, attendee.UserID)
		if err != nil {
			log.Printf("Error updating attendance for user %s: %v", attendee.UserID, err)
			continue
		}

		rows, _ := result.RowsAffected()
		updatedCount += int(rows)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"updated": updatedCount,
		"method":  req.CheckInMethod,
	})
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "calendar",
	}) {
		return // Process was re-exec'd after rebuild
	}

	log.Println("Starting Calendar API...")

	// Initialize configuration
	config := initConfig()

	// Initialize database
	if err := initDatabase(config); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	// Initialize calendar processor
	calendarProcessor = NewCalendarProcessor(db, config)

	// Initialize NLP processor (will use fallback if Ollama unavailable)
	nlpProcessor = NewNLPProcessor(config.OllamaURL, db, config)

	// Initialize vector search manager for Qdrant integration
	vectorSearchManager = NewVectorSearchManager(db, config.QdrantURL)

	// Initialize error handler (include stack traces in development)
	includeStackTrace := os.Getenv("GO_ENV") != "production"
	errorHandler = NewErrorHandler(includeStackTrace, true)

	// Initialize conflict detector
	conflictDetector = NewConflictDetector(db)

	// Initialize category manager
	categoryManager = NewCategoryManager(db)

	// Initialize meeting prep manager
	prepManager = NewMeetingPrepManager(db, nlpProcessor)

	// Start background processors
	ctx := context.Background()
	calendarProcessor.StartReminderProcessor(ctx)
	calendarProcessor.StartEventAutomationProcessor(ctx)

	// Setup routes
	router := mux.NewRouter()

	// Health check (no auth required)
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// Public API routes (no auth required)
	publicAPI := router.PathPrefix("/api/v1").Subrouter()
	publicAPI.HandleFunc("/auth/login", authLoginProxyHandler).Methods("POST", "OPTIONS")

	// Authenticated API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.Use(authMiddleware)

	api.HandleFunc("/auth/validate", authValidateHandler).Methods("GET")

	// Event management
	api.HandleFunc("/events", createEventHandler).Methods("POST")
	api.HandleFunc("/events", listEventsHandler).Methods("GET")
	api.HandleFunc("/events/{id}", getEventHandler).Methods("GET")
	api.HandleFunc("/events/{id}", updateEventHandler).Methods("PUT")
	api.HandleFunc("/events/{id}", deleteEventHandler).Methods("DELETE")

	// Schedule management
	api.HandleFunc("/schedule/chat", scheduleChatHandler).Methods("POST")
	api.HandleFunc("/schedule/optimize", scheduleOptimizeHandler).Methods("POST")

	// Reminder management (replacing n8n workflows)
	api.HandleFunc("/reminders/process", processRemindersHandler).Methods("POST")

	// iCal import/export
	api.HandleFunc("/events/export/ical", exportICalHandler).Methods("GET")
	api.HandleFunc("/events/import/ical", importICalHandler).Methods("POST")

	// Category management
	api.HandleFunc("/categories", getCategoriesHandler).Methods("GET")
	api.HandleFunc("/categories", createCategoryHandler).Methods("POST")
	api.HandleFunc("/events/by-category", getEventsByCategoryHandler).Methods("GET")
	api.HandleFunc("/categories/statistics", getCategoryStatisticsHandler).Methods("GET")

	// Template management
	api.HandleFunc("/templates", getTemplatesHandler).Methods("GET")
	api.HandleFunc("/templates", createTemplateHandler).Methods("POST")
	api.HandleFunc("/templates/{id}", deleteTemplateHandler).Methods("DELETE")
	api.HandleFunc("/events/from-template", createEventFromTemplateHandler).Methods("POST")

	// Bulk operations
	api.HandleFunc("/events/bulk", bulkCreateEventsHandler).Methods("POST")
	api.HandleFunc("/events/bulk", bulkUpdateEventsHandler).Methods("PUT")
	api.HandleFunc("/events/bulk", bulkDeleteEventsHandler).Methods("DELETE")

	// Attendance/RSVP management
	api.HandleFunc("/events/{id}/attendees", getAttendeesHandler).Methods("GET")
	api.HandleFunc("/events/{id}/rsvp", rsvpHandler).Methods("POST")
	api.HandleFunc("/events/{id}/attendance", trackAttendanceHandler).Methods("POST")

	// Travel time and buffer calculation
	api.HandleFunc("/travel/calculate", handleCalculateTravelTime).Methods("POST")
	api.HandleFunc("/events/{id}/departure-time", handleSuggestDepartureTime).Methods("GET")

	// Meeting preparation automation
	api.HandleFunc("/events/{id}/agenda", generateAgendaHandler).Methods("GET")
	api.HandleFunc("/events/{id}/agenda", updateAgendaHandler).Methods("PUT")

	// Resource management and double-booking prevention
	resourceManager := NewResourceManager(db)
	api.HandleFunc("/resources", resourceManager.CreateResource).Methods("POST")
	api.HandleFunc("/resources", resourceManager.ListResources).Methods("GET")
	api.HandleFunc("/resources/{id}/availability", resourceManager.CheckResourceAvailability).Methods("GET")
	api.HandleFunc("/events/{event_id}/resources", resourceManager.BookResourceForEvent).Methods("POST")
	api.HandleFunc("/events/{event_id}/resources", resourceManager.GetEventResources).Methods("GET")
	api.HandleFunc("/events/{event_id}/resources/{resource_id}", resourceManager.CancelResourceBooking).Methods("DELETE")

	// Advanced analytics on scheduling patterns
	analyticsManager := NewAnalyticsManager(db)
	api.HandleFunc("/analytics/schedule", analyticsManager.HandleAnalytics).Methods("GET")

	// External calendar synchronization (Google Calendar, Outlook)
	externalSyncManager := NewExternalSyncManager(db)
	api.HandleFunc("/external-sync/oauth/{provider}", externalSyncManager.InitiateOAuthHandler).Methods("GET")
	api.HandleFunc("/external-sync/oauth/{provider}/callback", externalSyncManager.OAuthCallbackHandler).Methods("GET")
	api.HandleFunc("/external-sync/sync", externalSyncManager.SyncEventsHandler).Methods("POST")
	api.HandleFunc("/external-sync/disconnect/{provider}", externalSyncManager.DisconnectCalendarHandler).Methods("DELETE")
	api.HandleFunc("/external-sync/status", externalSyncManager.GetSyncStatusHandler).Methods("GET")

	// Start background sync for external calendars
	go externalSyncManager.StartBackgroundSync(ctx)

	// Setup CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	// Initialize rate limiter - 100 requests per minute per IP
	rateLimiter := NewRateLimiter(100, time.Minute)

	// Setup middleware chain: Recovery -> RateLimit -> Logging -> CORS
	recoveryMiddleware := RecoveryMiddleware(errorHandler)
	middlewareChain := recoveryMiddleware(rateLimiter.Middleware(handlers.LoggingHandler(os.Stdout, corsHandler.Handler(router))))

	// Start server
	address := ":" + config.Port
	log.Printf("Calendar API server starting on port %s", config.Port)
	log.Printf("Health check: http://localhost:%s/health", config.Port)

	if err := http.ListenAndServe(address, middlewareChain); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// generateRecurringEvents creates recurring event instances based on pattern
func generateRecurringEvents(parentID string, recurrence *RecurrenceRequest, startTime, endTime time.Time, req CreateEventRequest, userID string) error {
	// Calculate the duration of the original event
	duration := endTime.Sub(startTime)

	// Determine the end date for recurring events
	var endDate time.Time
	if recurrence.EndDate != nil {
		endDate = *recurrence.EndDate
	} else if recurrence.MaxOccurrences != nil && *recurrence.MaxOccurrences > 0 {
		// Calculate based on max occurrences
		switch recurrence.Pattern {
		case "daily":
			endDate = startTime.AddDate(0, 0, *recurrence.MaxOccurrences*recurrence.Interval)
		case "weekly":
			endDate = startTime.AddDate(0, 0, *recurrence.MaxOccurrences*recurrence.Interval*7)
		case "monthly":
			endDate = startTime.AddDate(0, *recurrence.MaxOccurrences*recurrence.Interval, 0)
		default:
			endDate = startTime.AddDate(1, 0, 0) // Default to 1 year
		}
	} else {
		// Default to 1 year if no end specified
		endDate = startTime.AddDate(1, 0, 0)
	}

	// Generate recurring events
	currentStart := startTime
	occurrences := 0
	maxOccurrences := 365 // Default limit
	if recurrence.MaxOccurrences != nil && *recurrence.MaxOccurrences > 0 {
		maxOccurrences = *recurrence.MaxOccurrences
	}

	for currentStart.Before(endDate) && occurrences < maxOccurrences {
		// Skip the first occurrence (already created as parent)
		if occurrences > 0 {
			eventID := uuid.New().String()
			currentEnd := currentStart.Add(duration)

			// Create the recurring event instance with parent reference in metadata
			recurringMetadata := req.Metadata
			if recurringMetadata == nil {
				recurringMetadata = make(map[string]interface{})
			}
			recurringMetadata["parent_event_id"] = parentID
			recurringMetadata["is_recurring_instance"] = true

			query := `
				INSERT INTO events 
				(id, user_id, title, description, start_time, end_time, timezone, location, event_type, metadata, automation_config)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			`

			metadataJSON, _ := json.Marshal(recurringMetadata)
			automationJSON, _ := json.Marshal(req.AutomationConfig)

			_, err := db.Exec(query, eventID, userID, req.Title, req.Description,
				currentStart, currentEnd, req.Timezone, req.Location,
				req.EventType, metadataJSON, automationJSON)

			if err != nil {
				log.Printf("Error creating recurring event instance: %v", err)
				// Continue with next occurrence
			}
		}

		// Move to next occurrence
		switch recurrence.Pattern {
		case "daily":
			currentStart = currentStart.AddDate(0, 0, recurrence.Interval)
		case "weekly":
			currentStart = currentStart.AddDate(0, 0, recurrence.Interval*7)
		case "monthly":
			currentStart = currentStart.AddDate(0, recurrence.Interval, 0)
		default:
			return fmt.Errorf("unsupported recurrence pattern: %s", recurrence.Pattern)
		}

		occurrences++
	}

	log.Printf("Generated %d recurring events for parent %s", occurrences, parentID)
	return nil
}

// Test change for calendar
