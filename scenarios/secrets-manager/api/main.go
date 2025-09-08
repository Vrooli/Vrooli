package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Data models
type ResourceSecret struct {
	ID                string     `json:"id" db:"id"`
	ResourceName      string     `json:"resource_name" db:"resource_name"`
	SecretKey         string     `json:"secret_key" db:"secret_key"`
	SecretType        string     `json:"secret_type" db:"secret_type"`
	Required          bool       `json:"required" db:"required"`
	Description       *string    `json:"description" db:"description"`
	ValidationPattern *string    `json:"validation_pattern" db:"validation_pattern"`
	DocumentationURL  *string    `json:"documentation_url" db:"documentation_url"`
	DefaultValue      *string    `json:"default_value" db:"default_value"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

type SecretValidation struct {
	ID                 string     `json:"id" db:"id"`
	ResourceSecretID   string     `json:"resource_secret_id" db:"resource_secret_id"`
	ValidationStatus   string     `json:"validation_status" db:"validation_status"`
	ValidationMethod   string     `json:"validation_method" db:"validation_method"`
	ValidationTimestamp time.Time `json:"validation_timestamp" db:"validation_timestamp"`
	ErrorMessage       *string    `json:"error_message" db:"error_message"`
	ValidationDetails  *string    `json:"validation_details" db:"validation_details"`
}

type SecretScan struct {
	ID                 string     `json:"id" db:"id"`
	ScanType          string     `json:"scan_type" db:"scan_type"`
	ResourcesScanned  []string   `json:"resources_scanned" db:"resources_scanned"`
	SecretsDiscovered int        `json:"secrets_discovered" db:"secrets_discovered"`
	ScanDurationMs    int        `json:"scan_duration_ms" db:"scan_duration_ms"`
	ScanTimestamp     time.Time  `json:"scan_timestamp" db:"scan_timestamp"`
	ScanStatus        string     `json:"scan_status" db:"scan_status"`
	ErrorMessage      *string    `json:"error_message" db:"error_message"`
	ScanMetadata      *string    `json:"scan_metadata" db:"scan_metadata"`
}

type SecretHealthSummary struct {
	ResourceName         string     `json:"resource_name"`
	TotalSecrets         int        `json:"total_secrets"`
	RequiredSecrets      int        `json:"required_secrets"`
	ValidSecrets         int        `json:"valid_secrets"`
	MissingRequiredSecrets int      `json:"missing_required_secrets"`
	InvalidSecrets       int        `json:"invalid_secrets"`
	LastValidation       *time.Time `json:"last_validation"`
}

// API request/response types
type ScanRequest struct {
	Resources []string `json:"resources"`
	ScanType  string   `json:"scan_type"`
}

type ScanResponse struct {
	ScanID           string           `json:"scan_id"`
	DiscoveredSecrets []ResourceSecret `json:"discovered_secrets"`
	ScanDurationMs   int              `json:"scan_duration_ms"`
	ResourcesScanned []string         `json:"resources_scanned"`
}

type ValidationRequest struct {
	Resource string `json:"resource"`
}

type ValidationResponse struct {
	ValidationID     string              `json:"validation_id"`
	TotalSecrets     int                 `json:"total_secrets"`
	ValidSecrets     int                 `json:"valid_secrets"`
	MissingSecrets   []SecretValidation  `json:"missing_secrets"`
	InvalidSecrets   []SecretValidation  `json:"invalid_secrets"`
	HealthSummary    []SecretHealthSummary `json:"health_summary"`
}

type ProvisionRequest struct {
	SecretKey      string `json:"secret_key"`
	SecretValue    string `json:"secret_value"`
	StorageMethod  string `json:"storage_method"`
}

type ProvisionResponse struct {
	Success            bool              `json:"success"`
	StorageLocation    string            `json:"storage_location"`
	ValidationResult   SecretValidation  `json:"validation_result"`
}

// Database connection
var db *sql.DB
var scanner *SecretScanner
var validator *SecretValidator

func initDB() {
	var err error
	
	// Database configuration - support both POSTGRES_URL and individual components
	connStr := os.Getenv("POSTGRES_URL")
	if connStr == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		
		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
	}
	
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
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
	
	// Initialize scanner and validator components
	scanner = NewSecretScanner(db)
	validator = NewSecretValidator(db)
}

// getEnvOrDefault removed to prevent hardcoded defaults

// Resource scanner - scans Vrooli resources for secret requirements
func scanResourceSecrets(resources []string) ([]ResourceSecret, error) {
	var discoveredSecrets []ResourceSecret
	
	// Path to Vrooli resources directory
	resourcesPath := "../../../resources"
	
	// If no specific resources requested, scan all
	if len(resources) == 0 {
		entries, err := os.ReadDir(resourcesPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read resources directory: %v", err)
		}
		
		for _, entry := range entries {
			if entry.IsDir() {
				resources = append(resources, entry.Name())
			}
		}
	}
	
	// Scan each resource
	for _, resourceName := range resources {
		resourceDir := filepath.Join(resourcesPath, resourceName)
		secrets, err := scanResourceDirectory(resourceName, resourceDir)
		if err != nil {
			log.Printf("Warning: failed to scan resource %s: %v", resourceName, err)
			continue
		}
		discoveredSecrets = append(discoveredSecrets, secrets...)
	}
	
	return discoveredSecrets, nil
}

func scanResourceDirectory(resourceName, resourceDir string) ([]ResourceSecret, error) {
	var secrets []ResourceSecret
	
	// Patterns to look for environment variables and credentials
	envVarPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\$\{([A-Z_]+[A-Z0-9_]*)\}`),        // ${VAR_NAME}
		regexp.MustCompile(`\$([A-Z_]+[A-Z0-9_]*)`),            // $VAR_NAME
		regexp.MustCompile(`([A-Z_]+[A-Z0-9_]*)=`),             // VAR_NAME=
		regexp.MustCompile(`env\.([A-Z_]+[A-Z0-9_]*)`),         // env.VAR_NAME
		regexp.MustCompile(`getenv\("([A-Z_]+[A-Z0-9_]*)"\)`),  // getenv("VAR_NAME")
		regexp.MustCompile(`os\.Getenv\("([A-Z_]+[A-Z0-9_]*)"\)`), // os.Getenv("VAR_NAME")
	}
	
	// Walk through resource directory
	err := filepath.WalkDir(resourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip files we can't read
		}
		
		// Skip directories and non-text files
		if d.IsDir() || !isTextFile(path) {
			return nil
		}
		
		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}
		
		// Search for environment variables
		foundVars := make(map[string]bool)
		for _, pattern := range envVarPatterns {
			matches := pattern.FindAllStringSubmatch(string(content), -1)
			for _, match := range matches {
				if len(match) > 1 {
					varName := match[1]
					if !foundVars[varName] && isLikelySecret(varName) {
						foundVars[varName] = true
						
						secret := ResourceSecret{
							ID:                uuid.New().String(),
							ResourceName:      resourceName,
							SecretKey:         varName,
							SecretType:        classifySecretType(varName),
							Required:          isLikelyRequired(varName),
							Description:       stringPtr(fmt.Sprintf("Environment variable found in %s", filepath.Base(path))),
							ValidationPattern: nil,
							DocumentationURL:  nil,
							DefaultValue:      nil,
							CreatedAt:         time.Now(),
							UpdatedAt:         time.Now(),
						}
						secrets = append(secrets, secret)
					}
				}
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return secrets, nil
}

func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	textExtensions := []string{".sh", ".bash", ".yml", ".yaml", ".json", ".env", ".conf", ".config", ".md", ".txt", ".go", ".js", ".py", ".dockerfile", ".sql"}
	
	for _, textExt := range textExtensions {
		if ext == textExt {
			return true
		}
	}
	
	return false
}

func isLikelySecret(varName string) bool {
	secretKeywords := []string{
		"PASSWORD", "PASS", "PWD", "SECRET", "KEY", "TOKEN", "AUTH", "API", "CREDENTIAL", "CERT", "TLS", "SSL", "PRIVATE",
	}
	
	upperVar := strings.ToUpper(varName)
	for _, keyword := range secretKeywords {
		if strings.Contains(upperVar, keyword) {
			return true
		}
	}
	
	// Also include common configuration variables that might be sensitive
	configKeywords := []string{"HOST", "PORT", "URL", "ADDR", "DATABASE", "DB", "USER", "NAMESPACE"}
	for _, keyword := range configKeywords {
		if strings.Contains(upperVar, keyword) {
			return true
		}
	}
	
	return false
}

func classifySecretType(varName string) string {
	upperVar := strings.ToUpper(varName)
	
	if strings.Contains(upperVar, "PASSWORD") || strings.Contains(upperVar, "PASS") || strings.Contains(upperVar, "PWD") {
		return "password"
	}
	if strings.Contains(upperVar, "TOKEN") {
		return "token"
	}
	if strings.Contains(upperVar, "KEY") && (strings.Contains(upperVar, "API") || strings.Contains(upperVar, "ACCESS")) {
		return "api_key"
	}
	if strings.Contains(upperVar, "SECRET") || strings.Contains(upperVar, "CREDENTIAL") {
		return "credential"
	}
	if strings.Contains(upperVar, "CERT") || strings.Contains(upperVar, "TLS") || strings.Contains(upperVar, "SSL") {
		return "certificate"
	}
	
	return "env_var"
}

func isLikelyRequired(varName string) bool {
	requiredKeywords := []string{"PASSWORD", "SECRET", "TOKEN", "KEY", "DATABASE", "DB", "HOST"}
	upperVar := strings.ToUpper(varName)
	
	for _, keyword := range requiredKeywords {
		if strings.Contains(upperVar, keyword) {
			return true
		}
	}
	
	return false
}

// Validation functions
func validateSecrets(resource string) (ValidationResponse, error) {
	// Get all secrets for the resource (or all if empty)
	query := `
		SELECT rs.id, rs.resource_name, rs.secret_key, rs.secret_type, rs.required,
		       rs.description, rs.validation_pattern
		FROM resource_secrets rs
	`
	args := []interface{}{}
	
	if resource != "" {
		query += " WHERE rs.resource_name = $1"
		args = append(args, resource)
	}
	
	rows, err := db.Query(query, args...)
	if err != nil {
		return ValidationResponse{}, err
	}
	defer rows.Close()
	
	var secrets []ResourceSecret
	for rows.Next() {
		var secret ResourceSecret
		err := rows.Scan(&secret.ID, &secret.ResourceName, &secret.SecretKey, 
			&secret.SecretType, &secret.Required, &secret.Description, &secret.ValidationPattern)
		if err != nil {
			continue
		}
		secrets = append(secrets, secret)
	}
	
	// Validate each secret
	var validSecrets, totalSecrets int
	var missingSecrets, invalidSecrets []SecretValidation
	
	for _, secret := range secrets {
		totalSecrets++
		validation := validateSingleSecret(secret)
		
		// Store validation result
		storeValidationResult(validation)
		
		switch validation.ValidationStatus {
		case "valid":
			validSecrets++
		case "missing":
			missingSecrets = append(missingSecrets, validation)
		case "invalid":
			invalidSecrets = append(invalidSecrets, validation)
		}
	}
	
	// Get health summary
	healthSummary, _ := getHealthSummary()
	
	return ValidationResponse{
		ValidationID:   uuid.New().String(),
		TotalSecrets:   totalSecrets,
		ValidSecrets:   validSecrets,
		MissingSecrets: missingSecrets,
		InvalidSecrets: invalidSecrets,
		HealthSummary:  healthSummary,
	}, nil
}

func validateSingleSecret(secret ResourceSecret) SecretValidation {
	validation := SecretValidation{
		ID:                  uuid.New().String(),
		ResourceSecretID:    secret.ID,
		ValidationTimestamp: time.Now(),
	}
	
	// Check environment variable
	envValue := os.Getenv(secret.SecretKey)
	if envValue != "" {
		validation.ValidationMethod = "env"
		
		// Validate against pattern if provided
		if secret.ValidationPattern != nil {
			if matched, _ := regexp.MatchString(*secret.ValidationPattern, envValue); matched {
				validation.ValidationStatus = "valid"
			} else {
				validation.ValidationStatus = "invalid"
				validation.ErrorMessage = stringPtr("Value does not match required pattern")
			}
		} else {
			validation.ValidationStatus = "valid"
		}
	} else {
		// Check vault if available
		vaultValue, err := getVaultSecret(secret.SecretKey)
		if err == nil && vaultValue != "" {
			validation.ValidationMethod = "vault"
			validation.ValidationStatus = "valid"
		} else {
			validation.ValidationStatus = "missing"
			validation.ValidationMethod = "env"
			validation.ErrorMessage = stringPtr("Environment variable not set and not found in vault")
		}
	}
	
	return validation
}

func getVaultSecret(key string) (string, error) {
	// TODO: Implement vault integration using resource-vault CLI
	// For now, return empty to indicate not found
	return "", fmt.Errorf("vault integration not implemented")
}

func storeValidationResult(validation SecretValidation) {
	query := `
		INSERT INTO secret_validations (id, resource_secret_id, validation_status, 
			validation_method, validation_timestamp, error_message)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.Exec(query, validation.ID, validation.ResourceSecretID, 
		validation.ValidationStatus, validation.ValidationMethod, 
		validation.ValidationTimestamp, validation.ErrorMessage)
	if err != nil {
		log.Printf("Failed to store validation result: %v", err)
	}
}

func getHealthSummary() ([]SecretHealthSummary, error) {
	query := `SELECT * FROM secret_health_summary ORDER BY resource_name`
	
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var summaries []SecretHealthSummary
	for rows.Next() {
		var summary SecretHealthSummary
		err := rows.Scan(&summary.ResourceName, &summary.TotalSecrets,
			&summary.RequiredSecrets, &summary.ValidSecrets,
			&summary.MissingRequiredSecrets, &summary.InvalidSecrets,
			&summary.LastValidation)
		if err != nil {
			continue
		}
		summaries = append(summaries, summary)
	}
	
	return summaries, nil
}

// HTTP Handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "secrets-manager-api",
		"version":   "1.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func scanHandler(w http.ResponseWriter, r *http.Request) {
	var req ScanRequest
	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	}
	
	// Set defaults
	if req.ScanType == "" {
		req.ScanType = "full"
	}
	
	// Perform resource scan using new scanner
	response, err := scanner.ScanResources(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Scan failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	var req ValidationRequest
	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	}
	
	response, err := validator.ValidateSecrets(req.Resource)
	if err != nil {
		http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func provisionHandler(w http.ResponseWriter, r *http.Request) {
	var req ProvisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// TODO: Implement secret provisioning to vault or environment
	response := ProvisionResponse{
		Success:         false,
		StorageLocation: "",
		ValidationResult: SecretValidation{
			ValidationStatus: "error",
			ErrorMessage:     stringPtr("Provisioning not yet implemented"),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func storeDiscoveredSecret(secret ResourceSecret) {
	query := `
		INSERT INTO resource_secrets (id, resource_name, secret_key, secret_type, 
			required, description, validation_pattern, documentation_url, default_value,
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (resource_name, secret_key) 
		DO UPDATE SET 
			secret_type = EXCLUDED.secret_type,
			required = EXCLUDED.required,
			description = EXCLUDED.description,
			updated_at = CURRENT_TIMESTAMP
	`
	
	_, err := db.Exec(query, secret.ID, secret.ResourceName, secret.SecretKey,
		secret.SecretType, secret.Required, secret.Description, 
		secret.ValidationPattern, secret.DocumentationURL, secret.DefaultValue,
		secret.CreatedAt, secret.UpdatedAt)
	if err != nil {
		log.Printf("Failed to store discovered secret: %v", err)
	}
}

func storeScanRecord(scan SecretScan) {
	// TODO: Implement scan record storage
	log.Printf("Scan completed: %d secrets discovered in %dms", scan.SecretsDiscovered, scan.ScanDurationMs)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func main() {
	// Initialize database
	initDB()
	defer db.Close()
	
	// Create router
	r := mux.NewRouter()
	
	// Health check
	r.HandleFunc("/health", healthHandler).Methods("GET")
	
	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/secrets/scan", scanHandler).Methods("GET", "POST")
	api.HandleFunc("/secrets/validate", validateHandler).Methods("GET", "POST")
	api.HandleFunc("/secrets/provision", provisionHandler).Methods("POST")
	
	// CORS headers
	corsHeaders := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	corsMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	corsOrigins := handlers.AllowedOrigins([]string{"*"})
	
	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT") // Fallback to PORT
		if port == "" {
			log.Fatal("❌ API_PORT or PORT environment variable is required")
		}
	}
	
	log.Printf("🔐 Secrets Manager API starting on port %s", port)
	log.Printf("   📊 Health check: http://localhost:%s/health", port)
	log.Printf("   🔍 Scan endpoint: http://localhost:%s/api/v1/secrets/scan", port)
	log.Printf("   ✅ Validate endpoint: http://localhost:%s/api/v1/secrets/validate", port)
	
	// Start server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: handlers.CORS(corsHeaders, corsMethods, corsOrigins)(r),
	}
	
	log.Fatal(server.ListenAndServe())
}