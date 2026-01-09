package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// getBacklogPath returns the filesystem path for a backlog entry
func getBacklogPath(id string) string {
	return filepath.Join("../data/backlog", id+".md")
}

// saveBacklogEntryToFile persists a backlog entry to filesystem with frontmatter metadata
func saveBacklogEntryToFile(entry BacklogEntry) error {
	backlogPath := getBacklogPath(entry.ID)

	// Create directory if it doesn't exist
	backlogDir := filepath.Dir(backlogPath)
	if err := os.MkdirAll(backlogDir, 0755); err != nil {
		return fmt.Errorf("failed to create backlog directory: %w", err)
	}

	// Build file content with frontmatter
	var content strings.Builder
	content.WriteString("---\n")
	content.WriteString(fmt.Sprintf("id: %s\n", entry.ID))
	content.WriteString(fmt.Sprintf("entity_type: %s\n", entry.EntityType))
	content.WriteString(fmt.Sprintf("suggested_name: %s\n", entry.SuggestedName))
	if entry.Notes != "" {
		content.WriteString(fmt.Sprintf("notes: %s\n", strconv.Quote(entry.Notes)))
	}
	content.WriteString(fmt.Sprintf("status: %s\n", entry.Status))
	content.WriteString(fmt.Sprintf("created_at: %s\n", entry.CreatedAt.Format(time.RFC3339)))
	content.WriteString(fmt.Sprintf("updated_at: %s\n", entry.UpdatedAt.Format(time.RFC3339)))
	if entry.ConvertedDraftID != nil && *entry.ConvertedDraftID != "" {
		content.WriteString(fmt.Sprintf("converted_draft_id: %s\n", *entry.ConvertedDraftID))
	}
	content.WriteString("---\n\n")
	content.WriteString(entry.IdeaText)
	content.WriteString("\n")

	// Write file
	if err := os.WriteFile(backlogPath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write backlog file: %w", err)
	}

	return nil
}

// loadBacklogEntryFromFile reads a backlog entry from filesystem
func loadBacklogEntryFromFile(path string) (*BacklogEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open backlog file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Parse frontmatter
	var inFrontmatter bool
	var ideaTextBuilder strings.Builder
	entry := &BacklogEntry{}

	for scanner.Scan() {
		line := scanner.Text()

		// Start of frontmatter
		if line == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				// End of frontmatter
				inFrontmatter = false
				continue
			}
		}

		if inFrontmatter {
			// Parse frontmatter key-value pairs
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "id":
				entry.ID = value
			case "entity_type":
				entry.EntityType = value
			case "suggested_name":
				entry.SuggestedName = value
			case "notes":
				if decoded, err := strconv.Unquote(value); err == nil {
					entry.Notes = decoded
				} else {
					entry.Notes = value
				}
			case "status":
				entry.Status = value
			case "created_at":
				if t, err := time.Parse(time.RFC3339, value); err == nil {
					entry.CreatedAt = t
				}
			case "updated_at":
				if t, err := time.Parse(time.RFC3339, value); err == nil {
					entry.UpdatedAt = t
				}
			case "converted_draft_id":
				entry.ConvertedDraftID = &value
			}
		} else {
			// Content after frontmatter
			ideaTextBuilder.WriteString(line)
			ideaTextBuilder.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan backlog file: %w", err)
	}

	// Trim the idea text
	entry.IdeaText = strings.TrimSpace(ideaTextBuilder.String())

	// Validate required fields
	if entry.ID == "" {
		// Generate ID from filename if missing
		filename := filepath.Base(path)
		entry.ID = strings.TrimSuffix(filename, filepath.Ext(filename))
	}
	if entry.EntityType == "" {
		entry.EntityType = EntityTypeScenario // default
	}
	if entry.Status == "" {
		entry.Status = BacklogStatusPending // default
	}

	return entry, nil
}

// syncBacklogFilesystemWithDatabase syncs backlog files from filesystem to database
// Filesystem is the source of truth for git-backed persistence
func syncBacklogFilesystemWithDatabase(exec dbExecutor) error {
	if exec == nil {
		return fmt.Errorf("backlog executor is nil")
	}

	backlogRoot := filepath.Join("../data/backlog")
	info, err := os.Stat(backlogRoot)
	if err != nil {
		if os.IsNotExist(err) {
			// Create directory if it doesn't exist
			if err := os.MkdirAll(backlogRoot, 0755); err != nil {
				return fmt.Errorf("failed to create backlog directory: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to stat backlog directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("backlog path is not a directory: %s", backlogRoot)
	}

	// Walk through backlog directory
	entries, err := os.ReadDir(backlogRoot)
	if err != nil {
		return fmt.Errorf("failed to read backlog directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		path := filepath.Join(backlogRoot, entry.Name())
		backlogEntry, err := loadBacklogEntryFromFile(path)
		if err != nil {
			// Log error but continue processing other files
			fmt.Printf("Warning: failed to load backlog entry from %s: %v\n", path, err)
			continue
		}

		// Ensure valid UUID
		if _, err := uuid.Parse(backlogEntry.ID); err != nil {
			fmt.Printf("Warning: invalid UUID in backlog file %s: %v\n", path, err)
			continue
		}

		// Prepare converted_draft_id for SQL
		var convertedDraftID *string
		if backlogEntry.ConvertedDraftID != nil && *backlogEntry.ConvertedDraftID != "" {
			convertedDraftID = backlogEntry.ConvertedDraftID
		}

		// Upsert into database (filesystem is source of truth)
		_, err = exec.Exec(`
			INSERT INTO backlog_entries (id, idea_text, entity_type, suggested_name, notes, status, converted_draft_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id)
			DO UPDATE SET
				idea_text = EXCLUDED.idea_text,
				entity_type = EXCLUDED.entity_type,
				suggested_name = EXCLUDED.suggested_name,
				notes = EXCLUDED.notes,
				status = EXCLUDED.status,
				converted_draft_id = EXCLUDED.converted_draft_id,
				updated_at = EXCLUDED.updated_at
		`, backlogEntry.ID, backlogEntry.IdeaText, backlogEntry.EntityType, backlogEntry.SuggestedName, nullIfEmpty(backlogEntry.Notes),
			backlogEntry.Status, convertedDraftID, backlogEntry.CreatedAt, backlogEntry.UpdatedAt)

		if err != nil {
			return fmt.Errorf("failed to sync backlog entry %s: %w", backlogEntry.ID, err)
		}
	}

	return nil
}

// deleteBacklogFile removes a backlog entry file from filesystem
func deleteBacklogFile(id string) error {
	backlogPath := getBacklogPath(id)
	if err := os.Remove(backlogPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete backlog file: %w", err)
	}
	return nil
}
