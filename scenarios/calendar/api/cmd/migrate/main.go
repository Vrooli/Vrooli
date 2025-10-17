package main

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const (
	migrationsDir   = "migrations"
	migrationsTable = "schema_migrations"
)

// Migration represents a database migration
type Migration struct {
	Version  string
	Filename string
	Content  string
	Checksum string
}

// MigrationRecord represents a migration record from database
type MigrationRecord struct {
	Version   string
	AppliedAt time.Time
	Checksum  string
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start calendar

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "up":
		if err := runMigrationsUp(); err != nil {
			log.Fatalf("Migration up failed: %v", err)
		}
	case "down":
		count := 1
		if len(os.Args) > 2 {
			fmt.Sscanf(os.Args[2], "%d", &count)
		}
		if err := runMigrationsDown(count); err != nil {
			log.Fatalf("Migration down failed: %v", err)
		}
	case "status":
		if err := showMigrationStatus(); err != nil {
			log.Fatalf("Migration status failed: %v", err)
		}
	case "create":
		if len(os.Args) < 3 {
			log.Fatal("Migration name required")
		}
		name := os.Args[2]
		if err := createMigration(name); err != nil {
			log.Fatalf("Create migration failed: %v", err)
		}
	case "reset":
		if err := resetMigrations(); err != nil {
			log.Fatalf("Reset migrations failed: %v", err)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Calendar Database Migration Tool")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  go run cmd/migrate/main.go <command> [args]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  up              Apply all pending migrations")
	fmt.Println("  down [count]    Rollback last N migrations (default: 1)")
	fmt.Println("  status          Show migration status")
	fmt.Println("  create <name>   Create a new migration file")
	fmt.Println("  reset           Reset all migrations (DANGER!)")
	fmt.Println("")
	fmt.Println("Environment variables:")
	fmt.Println("  POSTGRES_URL    Database connection string (required)")
	fmt.Println("")
}

func getDBConnection() (*sql.DB, error) {
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components
		host := os.Getenv("POSTGRES_HOST")
		port := os.Getenv("POSTGRES_PORT")
		user := os.Getenv("POSTGRES_USER")
		password := os.Getenv("POSTGRES_PASSWORD")
		dbname := os.Getenv("POSTGRES_DB")

		if host == "" || port == "" || user == "" || password == "" || dbname == "" {
			return nil, fmt.Errorf("database configuration missing. Provide POSTGRES_URL or all individual components")
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			user, password, host, port, dbname)
	}

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func ensureMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS ` + migrationsTable + ` (
			version VARCHAR(50) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			checksum VARCHAR(32) NOT NULL
		)
	`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	return nil
}

func loadMigrations() ([]Migration, error) {
	var migrations []Migration

	err := filepath.WalkDir(migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", path, err)
		}

		filename := filepath.Base(path)
		version := strings.TrimSuffix(filename, ".sql")

		// Calculate checksum
		checksum := fmt.Sprintf("%x", md5.Sum(content))

		migrations = append(migrations, Migration{
			Version:  version,
			Filename: filename,
			Content:  string(content),
			Checksum: checksum,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func getAppliedMigrations(db *sql.DB) (map[string]MigrationRecord, error) {
	query := `SELECT version, applied_at, checksum FROM ` + migrationsTable + ` ORDER BY version`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]MigrationRecord)

	for rows.Next() {
		var record MigrationRecord
		if err := rows.Scan(&record.Version, &record.AppliedAt, &record.Checksum); err != nil {
			return nil, fmt.Errorf("failed to scan migration record: %w", err)
		}
		applied[record.Version] = record
	}

	return applied, nil
}

func runMigrationsUp() error {
	db, err := getDBConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := ensureMigrationsTable(db); err != nil {
		return err
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	log.Printf("Found %d migration files", len(migrations))
	log.Printf("Found %d applied migrations", len(applied))

	pendingCount := 0
	for _, migration := range migrations {
		if record, exists := applied[migration.Version]; exists {
			// Check checksum
			if record.Checksum != migration.Checksum {
				return fmt.Errorf("migration %s has been modified (checksum mismatch)", migration.Version)
			}
			log.Printf("✓ %s (already applied)", migration.Version)
			continue
		}

		pendingCount++
		log.Printf("→ Applying %s...", migration.Version)

		// Begin transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction for %s: %w", migration.Version, err)
		}

		// Execute migration
		if _, err := tx.Exec(migration.Content); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", migration.Version, err)
		}

		// Record migration as applied (if not already recorded by migration itself)
		_, err = tx.Exec(`
			INSERT INTO `+migrationsTable+` (version, checksum) 
			VALUES ($1, $2) 
			ON CONFLICT (version) DO NOTHING
		`, migration.Version, migration.Checksum)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", migration.Version, err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", migration.Version, err)
		}

		log.Printf("✓ %s applied successfully", migration.Version)
	}

	if pendingCount == 0 {
		log.Println("✓ Database is up to date")
	} else {
		log.Printf("✓ Applied %d migrations successfully", pendingCount)
	}

	return nil
}

func runMigrationsDown(count int) error {
	db, err := getDBConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	log.Printf("Rolling back %d migration(s)...", count)

	// For now, just log that down migrations aren't implemented
	// In a production system, you'd want to implement proper down migrations
	log.Println("⚠️  Down migrations not implemented yet")
	log.Println("   Consider implementing rollback scripts or manual database changes")

	return nil
}

func showMigrationStatus() error {
	db, err := getDBConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := ensureMigrationsTable(db); err != nil {
		return err
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	fmt.Println("Migration Status")
	fmt.Println("================")
	fmt.Printf("Available migrations: %d\n", len(migrations))
	fmt.Printf("Applied migrations: %d\n", len(applied))
	fmt.Println("")

	fmt.Println("Status  Version                    Applied At")
	fmt.Println("------  -------------------------  ---------------------------")

	for _, migration := range migrations {
		if record, exists := applied[migration.Version]; exists {
			status := "✓ UP  "
			if record.Checksum != migration.Checksum {
				status = "⚠ MOD "
			}
			fmt.Printf("%s  %-25s  %s\n", status, migration.Version, record.AppliedAt.Format("2006-01-02 15:04:05 UTC"))
		} else {
			fmt.Printf("✗ DOWN  %-25s  (not applied)\n", migration.Version)
		}
	}

	// Check for orphaned migrations (applied but file missing)
	fmt.Println("")
	orphaned := false
	for version := range applied {
		found := false
		for _, migration := range migrations {
			if migration.Version == version {
				found = true
				break
			}
		}
		if !found {
			if !orphaned {
				fmt.Println("Orphaned migrations (applied but file missing):")
				orphaned = true
			}
			fmt.Printf("⚠ ORPHAN  %-25s\n", version)
		}
	}

	return nil
}

func createMigration(name string) error {
	// Generate timestamp
	timestamp := time.Now().Format("20060102_150405")

	// Clean name
	cleanName := strings.ReplaceAll(strings.ToLower(name), " ", "_")

	// Generate filename
	filename := fmt.Sprintf("%s_%s.sql", timestamp, cleanName)
	filepath := filepath.Join(migrationsDir, filename)

	// Ensure migrations directory exists
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Create migration file with template
	template := fmt.Sprintf(`-- Migration: %s_%s
-- Description: %s
-- Author: Developer
-- Date: %s

-- Add your migration SQL here
-- Example:
-- CREATE TABLE example (
--     id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
--     name VARCHAR(255) NOT NULL,
--     created_at TIMESTAMPTZ DEFAULT NOW()
-- );

-- Record this migration as applied
-- INSERT INTO schema_migrations (version, checksum) 
-- VALUES ('%s_%s', 'update_this_checksum_after_writing_sql');
`,
		timestamp, cleanName, name, time.Now().Format("2006-01-02"), timestamp, cleanName)

	if err := os.WriteFile(filepath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to create migration file: %w", err)
	}

	log.Printf("✓ Created migration: %s", filename)
	log.Printf("  Edit the file and update the checksum before applying")

	return nil
}

func resetMigrations() error {
	fmt.Print("⚠️  This will delete all data and reset the database. Are you sure? (yes/no): ")

	var confirmation string
	fmt.Scanln(&confirmation)

	if confirmation != "yes" {
		log.Println("Reset cancelled")
		return nil
	}

	db, err := getDBConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	log.Println("Resetting database...")

	// Drop all tables (in reverse dependency order)
	tables := []string{
		"event_embeddings",
		"event_reminders",
		"recurring_patterns",
		"events",
		"users",
		"schema_migrations",
	}

	for _, table := range tables {
		if _, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)); err != nil {
			log.Printf("Warning: failed to drop table %s: %v", table, err)
		} else {
			log.Printf("✓ Dropped table %s", table)
		}
	}

	log.Println("✓ Database reset complete")
	log.Println("Run 'go run cmd/migrate/main.go up' to apply migrations")

	return nil
}
