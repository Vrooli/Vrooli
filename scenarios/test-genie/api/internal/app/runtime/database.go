package runtime

import (
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ensureDatabaseSchema(db *sql.DB) error {
	schemaPath, err := resolveInitializationFile("schema.sql")
	if err != nil {
		return fmt.Errorf("initialization schema lookup failed: %w", err)
	}
	if err := execSQLFile(db, schemaPath); err != nil {
		return err
	}
	if seedPath, err := resolveInitializationFile("seed.sql"); err == nil {
		if err := execSQLFile(db, seedPath); err != nil {
			return err
		}
	}
	return nil
}

func resolveInitializationFile(name string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to determine working directory: %w", err)
	}
	scenarioDir := filepath.Dir(wd)
	target := filepath.Join(scenarioDir, "initialization", "postgres", name)
	if _, err := os.Stat(target); err != nil {
		return "", fmt.Errorf("initialization file not accessible (%s): %w", target, err)
	}
	return target, nil
}

func execSQLFile(db *sql.DB, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	var builder strings.Builder
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan SQL file %s: %w", path, err)
	}

	statements := strings.Split(builder.String(), ";")
	for _, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" {
			continue
		}
		if _, err := db.Exec(trimmed); err != nil {
			return fmt.Errorf("failed to execute SQL statement from %s: %w", path, err)
		}
	}
	return nil
}
