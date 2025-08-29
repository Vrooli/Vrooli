package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// BacklogWatcher monitors the backlog directories for changes
type BacklogWatcher struct {
	server       *APIServer
	lastModified map[string]time.Time
	stopChan     chan bool
}

// NewBacklogWatcher creates a new backlog watcher
func NewBacklogWatcher(server *APIServer) *BacklogWatcher {
	return &BacklogWatcher{
		server:       server,
		lastModified: make(map[string]time.Time),
		stopChan:     make(chan bool),
	}
}

// Start begins monitoring the backlog directories
func (w *BacklogWatcher) Start() {
	log.Println("📁 Starting backlog file watcher...")
	
	// Initial scan
	w.scanDirectories()
	
	// Start monitoring in background
	go w.monitor()
}

// Stop stops the watcher
func (w *BacklogWatcher) Stop() {
	w.stopChan <- true
}

// monitor continuously watches for file changes
func (w *BacklogWatcher) monitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			w.scanDirectories()
		case <-w.stopChan:
			log.Println("📁 Stopping backlog file watcher")
			return
		}
	}
}

// scanDirectories checks all backlog directories for changes
func (w *BacklogWatcher) scanDirectories() {
	folders := []string{
		backlogPending,
		backlogInProgress,
		backlogCompleted,
		backlogFailed,
	}
	
	for _, folder := range folders {
		w.scanFolder(folder)
	}
}

// scanFolder checks a single folder for changes
func (w *BacklogWatcher) scanFolder(folder string) {
	// Ensure folder exists
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		os.MkdirAll(folder, 0755)
		return
	}
	
	files, err := os.ReadDir(folder)
	if err != nil {
		log.Printf("Error reading folder %s: %v", folder, err)
		return
	}
	
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".yaml" {
			continue
		}
		
		fullPath := filepath.Join(folder, file.Name())
		info, err := file.Info()
		if err != nil {
			continue
		}
		
		// Check if file is new or modified
		lastMod, exists := w.lastModified[fullPath]
		if !exists || info.ModTime().After(lastMod) {
			w.lastModified[fullPath] = info.ModTime()
			
			if exists {
				w.handleFileChange(fullPath, folder)
			} else {
				w.handleNewFile(fullPath, folder)
			}
		}
	}
	
	// Check for deleted files
	for path, _ := range w.lastModified {
		if filepath.Dir(path) == folder {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				delete(w.lastModified, path)
				w.handleFileDeleted(path, folder)
			}
		}
	}
}

// handleNewFile processes a newly added file
func (w *BacklogWatcher) handleNewFile(filepath string, folder string) {
	folderName := getFolderName(folder)
	filename := getFilename(filepath)
	
	log.Printf("📄 New backlog item detected in %s: %s", folderName, filename)
	
	// Validate YAML structure
	item, err := w.server.loadBacklogItem(filepath)
	if err != nil {
		log.Printf("⚠️  Invalid YAML in %s: %v", filename, err)
		// Move to failed folder with error info
		w.moveInvalidFile(filepath, err)
		return
	}
	
	// Auto-generate ID if missing
	if item.ID == "" {
		log.Printf("⚠️  Missing ID in %s, auto-generating...", filename)
		// TODO: Update file with generated ID
	}
	
	// If in pending and has high priority, notify
	if folder == backlogPending && item.Priority == "high" {
		log.Printf("🔴 High priority item added to backlog: %s", item.Name)
	}
}

// handleFileChange processes a modified file
func (w *BacklogWatcher) handleFileChange(filepath string, folder string) {
	folderName := getFolderName(folder)
	filename := getFilename(filepath)
	
	log.Printf("✏️  Backlog item modified in %s: %s", folderName, filename)
	
	// Validate updated YAML
	_, err := w.server.loadBacklogItem(filepath)
	if err != nil {
		log.Printf("⚠️  Invalid YAML after edit in %s: %v", filename, err)
	}
}

// handleFileDeleted processes a deleted file
func (w *BacklogWatcher) handleFileDeleted(filepath string, folder string) {
	folderName := getFolderName(folder)
	filename := getFilename(filepath)
	
	log.Printf("🗑️  Backlog item deleted from %s: %s", folderName, filename)
}

// moveInvalidFile moves an invalid YAML file to a special folder
func (w *BacklogWatcher) moveInvalidFile(filepath string, err error) {
	invalidFolder := "./backlog/invalid"
	os.MkdirAll(invalidFolder, 0755)
	
	filename := getFilename(filepath)
	newPath := fmt.Sprintf("%s/%s.error-%d.yaml", invalidFolder, filename, time.Now().Unix())
	
	// Read original file
	content, readErr := os.ReadFile(filepath)
	if readErr != nil {
		log.Printf("Failed to read invalid file: %v", readErr)
		return
	}
	
	// Write with error info
	errorContent := fmt.Sprintf("# ERROR: %v\n# Original file from: %s\n# Time: %s\n\n%s",
		err, filepath, time.Now().Format(time.RFC3339), string(content))
	
	os.WriteFile(newPath, []byte(errorContent), 0644)
	os.Remove(filepath)
	
	log.Printf("Moved invalid file to %s", newPath)
}

// Helper functions
func getFolderName(path string) string {
	return filepath.Base(path)
}

func getFilename(path string) string {
	return filepath.Base(path)
}

// AutoMoveCompleted automatically moves completed scenarios from in-progress to completed
func (w *BacklogWatcher) AutoMoveCompleted() {
	// Query database for completed scenarios
	query := `
		SELECT id, name, status 
		FROM scenarios 
		WHERE status = 'completed' 
		AND created_at > NOW() - INTERVAL '24 hours'`
	
	rows, err := w.server.db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()
	
	completedIDs := make(map[string]bool)
	for rows.Next() {
		var id, name, status string
		if err := rows.Scan(&id, &name, &status); err == nil {
			completedIDs[id] = true
		}
	}
	
	// Check in-progress folder for matching items
	items, err := w.server.getBacklogItems(backlogInProgress)
	if err != nil {
		return
	}
	
	for _, item := range items {
		if completedIDs[item.ID] {
			// Move to completed
			err := w.server.moveBacklogItem(item, backlogInProgress, backlogCompleted)
			if err == nil {
				log.Printf("✅ Auto-moved completed scenario to backlog/completed: %s", item.Name)
			}
		}
	}
}

// ValidateBacklogIntegrity checks all backlog items for validity
func (w *BacklogWatcher) ValidateBacklogIntegrity() error {
	folders := []string{backlogPending, backlogInProgress, backlogCompleted, backlogFailed}
	totalIssues := 0
	
	log.Println("🔍 Validating backlog integrity...")
	
	for _, folder := range folders {
		files, err := os.ReadDir(folder)
		if err != nil {
			continue
		}
		
		for _, file := range files {
			if file.IsDir() || filepath.Ext(file.Name()) != ".yaml" {
				continue
			}
			
			fullPath := filepath.Join(folder, file.Name())
			item, err := w.server.loadBacklogItem(fullPath)
			if err != nil {
				log.Printf("❌ Invalid YAML in %s: %v", fullPath, err)
				totalIssues++
				continue
			}
			
			// Validate required fields
			issues := []string{}
			if item.ID == "" {
				issues = append(issues, "missing ID")
			}
			if item.Name == "" {
				issues = append(issues, "missing name")
			}
			if item.Description == "" {
				issues = append(issues, "missing description")
			}
			if item.Prompt == "" {
				issues = append(issues, "missing prompt")
			}
			
			if len(issues) > 0 {
				log.Printf("⚠️  Issues in %s: %v", file.Name(), issues)
				totalIssues++
			}
		}
	}
	
	if totalIssues == 0 {
		log.Println("✅ Backlog validation complete - no issues found")
		return nil
	}
	
	return fmt.Errorf("found %d issues in backlog files", totalIssues)
}