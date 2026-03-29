package sqlfiles

import (
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"strings"
)

// ExecFile reads a SQL file, strips line comments, splits it into individual
// statements, and executes them sequentially.
func ExecFile(db *sql.DB, path string) error {
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

	for _, stmt := range SplitStatements(builder.String()) {
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

// SplitStatements splits SQL content by semicolons while respecting
// dollar-quoted strings (e.g., DO $$ ... END $$;).
func SplitStatements(content string) []string {
	var statements []string
	var current strings.Builder
	inDollarQuote := false
	dollarTag := ""

	i := 0
	for i < len(content) {
		if content[i] == '$' {
			j := i + 1
			for j < len(content) && (content[j] == '_' || (content[j] >= 'a' && content[j] <= 'z') || (content[j] >= 'A' && content[j] <= 'Z') || (content[j] >= '0' && content[j] <= '9')) {
				j++
			}
			if j < len(content) && content[j] == '$' {
				tag := content[i : j+1]
				if !inDollarQuote {
					inDollarQuote = true
					dollarTag = tag
					current.WriteString(tag)
					i = j + 1
					continue
				} else if tag == dollarTag {
					inDollarQuote = false
					dollarTag = ""
					current.WriteString(tag)
					i = j + 1
					continue
				}
			}
		}

		if content[i] == ';' && !inDollarQuote {
			current.WriteByte(';')
			statements = append(statements, current.String())
			current.Reset()
			i++
			continue
		}

		current.WriteByte(content[i])
		i++
	}

	if current.Len() > 0 {
		statements = append(statements, current.String())
	}

	return statements
}
