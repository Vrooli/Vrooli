package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
)

// InitializeDatabase creates tables if they don't exist
func InitializeDatabase(db *sql.DB) error {
	log.Println("🔧 Checking database schema...")
	
	// Check if schedules table exists
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'schedules'
		)
	`).Scan(&exists)
	
	if err != nil {
		return fmt.Errorf("failed to check if tables exist: %v", err)
	}
	
	if exists {
		log.Println("✅ Database schema already initialized")
		return nil
	}
	
	log.Println("📝 Initializing database schema...")
	
	// Try to read and execute schema.sql
	schemaPath := filepath.Join("..", "initialization", "storage", "postgres", "schema.sql")
	schemaBytes, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		// If schema file not found, use embedded minimal schema
		log.Println("⚠️ Schema file not found, using minimal embedded schema")
		return createMinimalSchema(db)
	}
	
	// Execute schema
	schemaSQL := string(schemaBytes)
	
	// Split by semicolon and execute each statement
	statements := strings.Split(schemaSQL, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		
		_, err = db.Exec(stmt)
		if err != nil {
			// Log but continue - some statements might fail if already exist
			log.Printf("⚠️ Schema statement warning: %v", err)
		}
	}
	
	log.Println("✅ Database schema initialized successfully")
	return nil
}

// createMinimalSchema creates a minimal working schema
func createMinimalSchema(db *sql.DB) error {
	schema := `
	-- Enable UUID extension
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
	
	-- Create schedules table with minimal required fields
	CREATE TABLE IF NOT EXISTS schedules (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		name VARCHAR(255) NOT NULL,
		description TEXT,
		cron_expression VARCHAR(100) NOT NULL,
		timezone VARCHAR(50) DEFAULT 'UTC',
		target_type VARCHAR(50) DEFAULT 'webhook',
		target_url TEXT,
		target_method VARCHAR(10) DEFAULT 'POST',
		target_headers JSONB DEFAULT '{}',
		target_payload JSONB DEFAULT '{}',
		enabled BOOLEAN DEFAULT true,
		status VARCHAR(20) DEFAULT 'active',
		overlap_policy VARCHAR(20) DEFAULT 'skip',
		max_retries INTEGER DEFAULT 3,
		retry_strategy VARCHAR(20) DEFAULT 'exponential',
		timeout_seconds INTEGER DEFAULT 300,
		catch_up_missed BOOLEAN DEFAULT true,
		tags TEXT[] DEFAULT '{}',
		priority INTEGER DEFAULT 5,
		owner VARCHAR(255),
		team VARCHAR(255),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		last_executed_at TIMESTAMP WITH TIME ZONE,
		next_execution_at TIMESTAMP WITH TIME ZONE
	);
	
	-- Create schedule_executions table (separate from n8n workflow executions)
	CREATE TABLE IF NOT EXISTS schedule_executions (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		schedule_id UUID REFERENCES schedules(id) ON DELETE CASCADE,
		scheduled_time TIMESTAMP WITH TIME ZONE NOT NULL,
		start_time TIMESTAMP WITH TIME ZONE,
		end_time TIMESTAMP WITH TIME ZONE,
		duration_ms INTEGER,
		status VARCHAR(20) DEFAULT 'pending',
		attempt_count INTEGER DEFAULT 1,
		response_code INTEGER,
		response_body TEXT,
		error_message TEXT,
		is_manual_trigger BOOLEAN DEFAULT false,
		is_catch_up BOOLEAN DEFAULT false,
		triggered_by VARCHAR(255),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	-- Create indexes for performance
	CREATE INDEX IF NOT EXISTS idx_schedules_enabled ON schedules(enabled);
	CREATE INDEX IF NOT EXISTS idx_schedules_next_execution ON schedules(next_execution_at);
	CREATE INDEX IF NOT EXISTS idx_schedule_executions_schedule_id ON schedule_executions(schedule_id);
	CREATE INDEX IF NOT EXISTS idx_schedule_executions_status ON schedule_executions(status);
	
	-- Create cron_presets table
	CREATE TABLE IF NOT EXISTS cron_presets (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		name VARCHAR(255) NOT NULL,
		description TEXT,
		expression VARCHAR(100) NOT NULL,
		category VARCHAR(50),
		is_system BOOLEAN DEFAULT false,
		usage_count INTEGER DEFAULT 0
	);
	
	-- Insert default cron presets
	INSERT INTO cron_presets (name, description, expression, category, is_system) VALUES
		('Every minute', 'Runs every minute', '* * * * *', 'Frequent', true),
		('Every 5 minutes', 'Runs every 5 minutes', '*/5 * * * *', 'Frequent', true),
		('Every 15 minutes', 'Runs every 15 minutes', '*/15 * * * *', 'Frequent', true),
		('Every 30 minutes', 'Runs every 30 minutes', '*/30 * * * *', 'Frequent', true),
		('Every hour', 'Runs at the start of every hour', '0 * * * *', 'Hourly', true),
		('Every day at midnight', 'Runs daily at midnight', '0 0 * * *', 'Daily', true),
		('Every day at 9 AM', 'Runs daily at 9:00 AM', '0 9 * * *', 'Daily', true),
		('Every Monday', 'Runs weekly on Monday at midnight', '0 0 * * 1', 'Weekly', true),
		('First day of month', 'Runs monthly on the 1st at midnight', '0 0 1 * *', 'Monthly', true)
	ON CONFLICT DO NOTHING;
	`
	
	// Execute minimal schema
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create minimal schema: %v", err)
	}
	
	return nil
}