package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var db *sql.DB

func initDatabase() error {
	// Get PostgreSQL connection info from environment
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		return fmt.Errorf("POSTGRES_PORT environment variable is required")
	}
	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "postgres"
	}
	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		// SECURITY: Password must be provided via environment variable in production
		// For local development, it falls back to the Vrooli default postgres setup
		slog.Warn("database authentication not configured", "environment", "development", "recommendation", "set database credentials via environment variables for production use")
	}
	dbname := os.Getenv("POSTGRES_DB")
	if dbname == "" {
		dbname = "financial_calculators_hub"
	}

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	err = db.Ping()
	if err != nil {
		// If database doesn't exist, create it
		if err.Error() == fmt.Sprintf("pq: database \"%s\" does not exist", dbname) {
			if err := createDatabase(host, port, user, password, dbname); err != nil {
				return fmt.Errorf("failed to create database: %w", err)
			}
			// Reconnect to new database
			db, err = sql.Open("postgres", psqlInfo)
			if err != nil {
				return fmt.Errorf("failed to reconnect: %w", err)
			}
		} else {
			return fmt.Errorf("failed to ping database: %w", err)
		}
	}

	// Initialize schema
	if err := initSchema(); err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	slog.Info("database connected successfully", "host", host, "port", port, "database", dbname)
	return nil
}

func createDatabase(host, port, user, password, dbname string) error {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=disable",
		host, port, user, password)

	tempDB, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return err
	}
	defer tempDB.Close()

	_, err = tempDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbname))
	if err != nil {
		return err
	}

	slog.Info("database created successfully", "database", dbname)
	return nil
}

func initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS calculations (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		calculator_type VARCHAR(50) NOT NULL,
		inputs JSONB NOT NULL,
		outputs JSONB NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		user_id UUID,
		notes TEXT
	);
	
	CREATE INDEX IF NOT EXISTS idx_calculator_type ON calculations(calculator_type);
	CREATE INDEX IF NOT EXISTS idx_created_at ON calculations(created_at);
	CREATE INDEX IF NOT EXISTS idx_user_id ON calculations(user_id);
	
	CREATE TABLE IF NOT EXISTS saved_scenarios (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		description TEXT,
		calculations UUID[] DEFAULT '{}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		share_token VARCHAR(100) UNIQUE
	);
	
	CREATE INDEX IF NOT EXISTS idx_share_token ON saved_scenarios(share_token);
	CREATE INDEX IF NOT EXISTS idx_scenarios_created_at ON saved_scenarios(created_at);
	
	CREATE TABLE IF NOT EXISTS net_worth_entries (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID,
		entry_date DATE NOT NULL,
		assets JSONB NOT NULL DEFAULT '{}',
		liabilities JSONB NOT NULL DEFAULT '{}',
		net_worth DECIMAL(15, 2) GENERATED ALWAYS AS (
			(assets->>'total')::DECIMAL - (liabilities->>'total')::DECIMAL
		) STORED,
		notes TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, entry_date)
	);
	
	CREATE INDEX IF NOT EXISTS idx_user_entry_date ON net_worth_entries(user_id, entry_date);
	CREATE INDEX IF NOT EXISTS idx_entry_date ON net_worth_entries(entry_date);
	
	CREATE TABLE IF NOT EXISTS tax_calculations (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID,
		tax_year INTEGER NOT NULL,
		income DECIMAL(15, 2) NOT NULL,
		filing_status VARCHAR(50) NOT NULL,
		deductions JSONB DEFAULT '{}',
		credits JSONB DEFAULT '{}',
		tax_owed DECIMAL(15, 2),
		effective_rate DECIMAL(5, 2),
		marginal_rate DECIMAL(5, 2),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_user_year ON tax_calculations(user_id, tax_year);
	
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = CURRENT_TIMESTAMP;
		RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;
	
	DROP TRIGGER IF EXISTS update_saved_scenarios_updated_at ON saved_scenarios;
	CREATE TRIGGER update_saved_scenarios_updated_at 
		BEFORE UPDATE ON saved_scenarios
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();
		
	DROP TRIGGER IF EXISTS update_net_worth_updated_at ON net_worth_entries;
	CREATE TRIGGER update_net_worth_updated_at
		BEFORE UPDATE ON net_worth_entries
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();
	`

	_, err := db.Exec(schema)
	return err
}

// SaveCalculation stores a calculation result in the database
func saveCalculation(calcType string, inputs interface{}, outputs interface{}, userID *string, notes *string) (string, error) {
	if db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	id := uuid.New().String()

	inputsJSON, err := json.Marshal(inputs)
	if err != nil {
		return "", err
	}

	outputsJSON, err := json.Marshal(outputs)
	if err != nil {
		return "", err
	}

	query := `
		INSERT INTO calculations (id, calculator_type, inputs, outputs, user_id, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = db.Exec(query, id, calcType, inputsJSON, outputsJSON, userID, notes)
	if err != nil {
		return "", err
	}

	return id, nil
}

// GetCalculationHistory retrieves calculation history
func getCalculationHistory(calcType string, limit int, userID *string) ([]map[string]interface{}, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var query string
	var rows *sql.Rows
	var err error

	if userID != nil && *userID != "" {
		query = `
			SELECT id, calculator_type, inputs, outputs, created_at, notes
			FROM calculations
			WHERE ($1 = '' OR calculator_type = $1)
			  AND user_id = $2
			ORDER BY created_at DESC
			LIMIT $3
		`
		rows, err = db.Query(query, calcType, *userID, limit)
	} else {
		query = `
			SELECT id, calculator_type, inputs, outputs, created_at, notes
			FROM calculations
			WHERE ($1 = '' OR calculator_type = $1)
			ORDER BY created_at DESC
			LIMIT $2
		`
		rows, err = db.Query(query, calcType, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, calcType string
		var inputs, outputs json.RawMessage
		var createdAt time.Time
		var notes sql.NullString

		err := rows.Scan(&id, &calcType, &inputs, &outputs, &createdAt, &notes)
		if err != nil {
			return nil, err
		}

		result := map[string]interface{}{
			"id":              id,
			"calculator_type": calcType,
			"inputs":          inputs,
			"outputs":         outputs,
			"created_at":      createdAt,
		}

		if notes.Valid {
			result["notes"] = notes.String
		}

		results = append(results, result)
	}

	return results, nil
}
