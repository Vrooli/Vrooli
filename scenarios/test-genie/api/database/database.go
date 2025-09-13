package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"time"
	
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"test-genie-api/models"
)

var db *sql.DB

// InitDB initializes the database connection with exponential backoff
func InitDB() error {
	// ALL database configuration MUST come from environment - no defaults
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	// Validate required environment variables
	if dbHost == "" || dbPort == "" || dbUser == "" || dbName == "" {
		return fmt.Errorf("missing required database configuration. Please set: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_DB")
	}

	// Build connection string - password is optional for trust auth
	var connStr string
	if dbPassword != "" {
		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
	} else {
		log.Println("⚠️  No password provided, attempting trust authentication...")
		connStr = fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbName)
	}

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
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
		return fmt.Errorf("database connection failed after %d attempts: %w", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")

	// Create tables if they don't exist
	return createTables()
}

// GetDB returns the database connection
func GetDB() *sql.DB {
	return db
}

// Close closes the database connection
func Close() {
	if db != nil {
		db.Close()
	}
}

// createTables creates the necessary database tables
func createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS test_suites (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scenario_name VARCHAR(255) NOT NULL,
			suite_type VARCHAR(50) NOT NULL,
			test_cases JSONB DEFAULT '[]',
			coverage_metrics JSONB DEFAULT '{}',
			generated_at TIMESTAMP DEFAULT NOW(),
			last_executed TIMESTAMP,
			status VARCHAR(50) DEFAULT 'active',
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS test_executions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			suite_id UUID REFERENCES test_suites(id),
			execution_type VARCHAR(50),
			start_time TIMESTAMP DEFAULT NOW(),
			end_time TIMESTAMP,
			status VARCHAR(50) DEFAULT 'running',
			results JSONB DEFAULT '[]',
			performance_metrics JSONB DEFAULT '{}',
			environment VARCHAR(100),
			execution_notes TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS test_results (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			execution_id UUID REFERENCES test_executions(id),
			test_case_id UUID,
			status VARCHAR(50),
			duration DECIMAL(10,3),
			error_message TEXT,
			stack_trace TEXT,
			assertions JSONB DEFAULT '[]',
			artifacts JSONB DEFAULT '{}',
			started_at TIMESTAMP,
			completed_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS test_vaults (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scenario_name VARCHAR(255) NOT NULL,
			vault_name VARCHAR(255) NOT NULL,
			phases JSONB DEFAULT '[]',
			phase_configurations JSONB DEFAULT '{}',
			success_criteria JSONB DEFAULT '{}',
			total_timeout INTEGER DEFAULT 3600,
			created_at TIMESTAMP DEFAULT NOW(),
			last_executed TIMESTAMP,
			status VARCHAR(50) DEFAULT 'active'
		)`,
		`CREATE TABLE IF NOT EXISTS vault_executions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			vault_id UUID REFERENCES test_vaults(id),
			execution_type VARCHAR(50),
			start_time TIMESTAMP DEFAULT NOW(),
			end_time TIMESTAMP,
			current_phase VARCHAR(100),
			completed_phases JSONB DEFAULT '[]',
			failed_phases JSONB DEFAULT '[]',
			status VARCHAR(50) DEFAULT 'running',
			phase_results JSONB DEFAULT '{}',
			environment VARCHAR(100),
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS coverage_analysis (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scenario_name VARCHAR(255) NOT NULL,
			overall_coverage DECIMAL(5,2),
			coverage_by_file JSONB DEFAULT '{}',
			coverage_gaps JSONB DEFAULT '{}',
			improvement_suggestions JSONB DEFAULT '[]',
			priority_areas JSONB DEFAULT '[]',
			analyzed_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_test_suites_scenario ON test_suites(scenario_name)`,
		`CREATE INDEX IF NOT EXISTS idx_test_executions_suite ON test_executions(suite_id)`,
		`CREATE INDEX IF NOT EXISTS idx_test_results_execution ON test_results(execution_id)`,
		`CREATE INDEX IF NOT EXISTS idx_test_vaults_scenario ON test_vaults(scenario_name)`,
		`CREATE INDEX IF NOT EXISTS idx_vault_executions_vault ON vault_executions(vault_id)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	log.Println("✅ Database tables created/verified successfully")
	return nil
}

// CheckHealth performs a health check on the database
func CheckHealth() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	
	// Check if we can query
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_suites").Scan(&count)
	if err != nil {
		return fmt.Errorf("database query failed: %w", err)
	}
	
	return nil
}

// StoreTestSuite stores a test suite in the database
func StoreTestSuite(suite *models.TestSuite) error {
	testCasesJSON, err := json.Marshal(suite.TestCases)
	if err != nil {
		return fmt.Errorf("failed to marshal test cases: %w", err)
	}
	
	coverageJSON, err := json.Marshal(suite.CoverageMetrics)
	if err != nil {
		return fmt.Errorf("failed to marshal coverage metrics: %w", err)
	}
	
	_, err = db.Exec(`
		INSERT INTO test_suites (id, scenario_name, suite_type, test_cases, coverage_metrics, generated_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, suite.ID, suite.ScenarioName, suite.SuiteType, testCasesJSON, coverageJSON, suite.GeneratedAt, suite.Status)
	
	if err != nil {
		return fmt.Errorf("failed to store test suite: %w", err)
	}
	
	log.Printf("✅ Test suite stored successfully: %s", suite.ID)
	return nil
}

// GetTestSuite retrieves a test suite from the database
func GetTestSuite(suiteID uuid.UUID) (*models.TestSuite, error) {
	var suite models.TestSuite
	var testCasesJSON, coverageJSON []byte
	
	err := db.QueryRow(`
		SELECT id, scenario_name, suite_type, test_cases, coverage_metrics, generated_at, last_executed, status
		FROM test_suites WHERE id = $1
	`, suiteID).Scan(
		&suite.ID, &suite.ScenarioName, &suite.SuiteType, 
		&testCasesJSON, &coverageJSON,
		&suite.GeneratedAt, &suite.LastExecuted, &suite.Status,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("test suite not found: %s", suiteID)
		}
		return nil, fmt.Errorf("failed to get test suite: %w", err)
	}
	
	if err := json.Unmarshal(testCasesJSON, &suite.TestCases); err != nil {
		return nil, fmt.Errorf("failed to unmarshal test cases: %w", err)
	}
	
	if err := json.Unmarshal(coverageJSON, &suite.CoverageMetrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal coverage metrics: %w", err)
	}
	
	return &suite, nil
}

// ListTestSuites lists all test suites
func ListTestSuites() ([]models.TestSuite, error) {
	rows, err := db.Query(`
		SELECT id, scenario_name, suite_type, test_cases, coverage_metrics, generated_at, last_executed, status
		FROM test_suites
		ORDER BY generated_at DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list test suites: %w", err)
	}
	defer rows.Close()
	
	var suites []models.TestSuite
	for rows.Next() {
		var suite models.TestSuite
		var testCasesJSON, coverageJSON []byte
		
		err := rows.Scan(
			&suite.ID, &suite.ScenarioName, &suite.SuiteType,
			&testCasesJSON, &coverageJSON,
			&suite.GeneratedAt, &suite.LastExecuted, &suite.Status,
		)
		if err != nil {
			log.Printf("Warning: failed to scan test suite row: %v", err)
			continue
		}
		
		if err := json.Unmarshal(testCasesJSON, &suite.TestCases); err != nil {
			log.Printf("Warning: failed to unmarshal test cases: %v", err)
			suite.TestCases = []models.TestCase{}
		}
		
		if err := json.Unmarshal(coverageJSON, &suite.CoverageMetrics); err != nil {
			log.Printf("Warning: failed to unmarshal coverage metrics: %v", err)
			suite.CoverageMetrics = models.CoverageMetrics{}
		}
		
		suites = append(suites, suite)
	}
	
	return suites, nil
}

// StoreTestExecution stores a test execution with retry logic
func StoreTestExecution(execution *models.TestExecution) error {
	resultsJSON, err := json.Marshal(execution.Results)
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}
	
	metricsJSON, err := json.Marshal(execution.PerformanceMetrics)
	if err != nil {
		return fmt.Errorf("failed to marshal performance metrics: %w", err)
	}
	
	maxRetries := 3
	var lastErr error
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		_, lastErr = db.Exec(`
			INSERT INTO test_executions (id, suite_id, execution_type, start_time, end_time, status, results, performance_metrics, environment)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, execution.ID, execution.SuiteID, execution.ExecutionType, execution.StartTime, 
			execution.EndTime, execution.Status, resultsJSON, metricsJSON, execution.Environment)
		
		if lastErr == nil {
			log.Printf("✅ Test execution stored successfully: %s", execution.ID)
			return nil
		}
		
		if attempt < maxRetries-1 {
			backoffDelay := time.Duration(attempt+1) * time.Second
			log.Printf("⚠️ Failed to store test execution (attempt %d/%d): %v. Retrying in %v...", 
				attempt+1, maxRetries, lastErr, backoffDelay)
			time.Sleep(backoffDelay)
		}
	}
	
	return fmt.Errorf("failed to store test execution after %d attempts: %w", maxRetries, lastErr)
}

// StoreTestResult stores a test result with retry logic
func StoreTestResult(result *models.TestResult) error {
	assertionsJSON, err := json.Marshal(result.Assertions)
	if err != nil {
		return fmt.Errorf("failed to marshal assertions: %w", err)
	}
	
	artifactsJSON, err := json.Marshal(result.Artifacts)
	if err != nil {
		return fmt.Errorf("failed to marshal artifacts: %w", err)
	}
	
	maxRetries := 3
	var lastErr error
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		_, lastErr = db.Exec(`
			INSERT INTO test_results (id, execution_id, test_case_id, status, duration, error_message, stack_trace, assertions, artifacts, started_at, completed_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, result.ID, result.ExecutionID, result.TestCaseID, result.Status, result.Duration, 
			result.ErrorMessage, result.StackTrace, assertionsJSON, artifactsJSON, result.StartedAt, result.CompletedAt)
		
		if lastErr == nil {
			log.Printf("✅ Successfully stored test result: %s (status: %s)", result.TestCaseID, result.Status)
			return nil
		}
		
		if attempt < maxRetries-1 {
			backoffDelay := time.Duration(attempt+1) * time.Second
			log.Printf("⚠️ Failed to store test result (attempt %d/%d): %v. Retrying in %v...", 
				attempt+1, maxRetries, lastErr, backoffDelay)
			time.Sleep(backoffDelay)
		}
	}
	
	log.Printf("❌ Failed to store test result after %d attempts: %v", maxRetries, lastErr)
	return lastErr
}

// UpdateExecutionStatus updates the status of a test execution
func UpdateExecutionStatus(executionID uuid.UUID, status string, errorMessage string) error {
	maxRetries := 3
	var lastErr error
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		_, lastErr = db.Exec(`
			UPDATE test_executions 
			SET status = $1, end_time = $2, execution_notes = $3
			WHERE id = $4
		`, status, time.Now(), errorMessage, executionID)
		
		if lastErr == nil {
			log.Printf("✅ Successfully updated execution status: %s -> %s", executionID, status)
			return nil
		}
		
		if attempt < maxRetries-1 {
			backoffDelay := time.Duration(attempt+1) * time.Second
			log.Printf("⚠️ Failed to update execution status (attempt %d/%d): %v. Retrying in %v...", 
				attempt+1, maxRetries, lastErr, backoffDelay)
			time.Sleep(backoffDelay)
		}
	}
	
	log.Printf("❌ Failed to update execution status after %d attempts: %v", maxRetries, lastErr)
	return lastErr
}

// StoreTestVault stores a test vault
func StoreTestVault(vault *models.TestVault) error {
	phasesJSON, _ := json.Marshal(vault.Phases)
	phaseConfigJSON, _ := json.Marshal(vault.PhaseConfigurations)
	criteriaJSON, _ := json.Marshal(vault.SuccessCriteria)
	
	_, err := db.Exec(`
		INSERT INTO test_vaults (id, scenario_name, vault_name, phases, phase_configurations, success_criteria, total_timeout, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, vault.ID, vault.ScenarioName, vault.VaultName, phasesJSON, phaseConfigJSON, criteriaJSON, vault.TotalTimeout, vault.Status)
	
	if err != nil {
		return fmt.Errorf("failed to store test vault: %w", err)
	}
	
	log.Printf("✅ Test vault stored successfully: %s", vault.ID)
	return nil
}

// GetTestVault retrieves a test vault
func GetTestVault(vaultID uuid.UUID) (*models.TestVault, error) {
	var vault models.TestVault
	var phasesJSON, phaseConfigJSON, criteriaJSON []byte
	
	err := db.QueryRow(`
		SELECT id, scenario_name, vault_name, phases, phase_configurations, success_criteria, total_timeout, created_at, last_executed, status
		FROM test_vaults WHERE id = $1
	`, vaultID).Scan(
		&vault.ID, &vault.ScenarioName, &vault.VaultName,
		&phasesJSON, &phaseConfigJSON, &criteriaJSON,
		&vault.TotalTimeout, &vault.CreatedAt, &vault.LastExecuted, &vault.Status,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get test vault: %w", err)
	}
	
	json.Unmarshal(phasesJSON, &vault.Phases)
	json.Unmarshal(phaseConfigJSON, &vault.PhaseConfigurations)
	json.Unmarshal(criteriaJSON, &vault.SuccessCriteria)
	
	return &vault, nil
}

// ListTestVaults lists all test vaults
func ListTestVaults() ([]models.TestVault, error) {
	rows, err := db.Query(`
		SELECT id, scenario_name, vault_name, phases, phase_configurations, success_criteria, total_timeout, created_at, last_executed, status
		FROM test_vaults
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list test vaults: %w", err)
	}
	defer rows.Close()
	
	var vaults []models.TestVault
	for rows.Next() {
		var vault models.TestVault
		var phasesJSON, phaseConfigJSON, criteriaJSON []byte
		
		err := rows.Scan(
			&vault.ID, &vault.ScenarioName, &vault.VaultName,
			&phasesJSON, &phaseConfigJSON, &criteriaJSON,
			&vault.TotalTimeout, &vault.CreatedAt, &vault.LastExecuted, &vault.Status,
		)
		if err != nil {
			continue
		}
		
		json.Unmarshal(phasesJSON, &vault.Phases)
		json.Unmarshal(phaseConfigJSON, &vault.PhaseConfigurations)
		json.Unmarshal(criteriaJSON, &vault.SuccessCriteria)
		
		vaults = append(vaults, vault)
	}
	
	return vaults, nil
}