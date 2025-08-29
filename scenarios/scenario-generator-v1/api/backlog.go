package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

// BacklogItem represents a scenario in the backlog queue
type BacklogItem struct {
	ID               string                 `yaml:"id" json:"id"`
	Name             string                 `yaml:"name" json:"name"`
	Description      string                 `yaml:"description" json:"description"`
	Prompt           string                 `yaml:"prompt" json:"prompt"`
	Complexity       string                 `yaml:"complexity" json:"complexity"`
	Category         string                 `yaml:"category" json:"category"`
	Priority         string                 `yaml:"priority" json:"priority"`
	EstimatedRevenue int                    `yaml:"estimated_revenue" json:"estimated_revenue"`
	Tags             []string               `yaml:"tags" json:"tags"`
	Metadata         BacklogMetadata        `yaml:"metadata" json:"metadata"`
	ResourcesRequired []string              `yaml:"resources_required" json:"resources_required"`
	ValidationCriteria []string             `yaml:"validation_criteria" json:"validation_criteria"`
	Notes            string                 `yaml:"notes" json:"notes"`
	
	// Additional fields for UI display
	Filename         string                 `json:"filename"`
	Status           string                 `json:"status"`
	CreatedAt        time.Time              `json:"created_at"`
	ModifiedAt       time.Time              `json:"modified_at"`
}

type BacklogMetadata struct {
	RequestedBy         string `yaml:"requested_by" json:"requested_by"`
	RequestedDate       string `yaml:"requested_date" json:"requested_date"`
	Deadline            string `yaml:"deadline" json:"deadline"`
	BusinessValue       string `yaml:"business_value" json:"business_value"`
	MarketOpportunity   string `yaml:"market_opportunity" json:"market_opportunity"`
	CompetitiveAdvantage string `yaml:"competitive_advantage" json:"competitive_advantage"`
}

// Backlog directory paths
const (
	backlogBase      = "./backlog"
	backlogPending   = "./backlog/pending"
	backlogInProgress = "./backlog/in-progress"
	backlogCompleted = "./backlog/completed"
	backlogFailed    = "./backlog/failed"
	backlogTemplates = "./backlog/templates"
)

// getBacklogItems retrieves all items from a specific backlog folder
func (s *APIServer) getBacklogItems(folder string) ([]BacklogItem, error) {
	items := []BacklogItem{}
	
	files, err := ioutil.ReadDir(folder)
	if err != nil {
		if os.IsNotExist(err) {
			return items, nil // Return empty list if folder doesn't exist
		}
		return nil, err
	}
	
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
			continue
		}
		
		item, err := s.loadBacklogItem(filepath.Join(folder, file.Name()))
		if err != nil {
			// Log error but continue processing other files
			fmt.Printf("Error loading backlog item %s: %v\n", file.Name(), err)
			continue
		}
		
		// Add file metadata
		item.Filename = file.Name()
		item.Status = filepath.Base(folder)
		item.ModifiedAt = file.ModTime()
		
		// Extract creation time from filename if it follows XXX- pattern
		if len(file.Name()) > 4 && file.Name()[3] == '-' {
			item.CreatedAt = file.ModTime() // Use mod time as approximation
		}
		
		items = append(items, item)
	}
	
	// Sort by filename (which includes priority prefix)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Filename < items[j].Filename
	})
	
	return items, nil
}

// loadBacklogItem loads a single backlog item from a YAML file
func (s *APIServer) loadBacklogItem(filepath string) (BacklogItem, error) {
	var item BacklogItem
	
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return item, err
	}
	
	err = yaml.Unmarshal(data, &item)
	if err != nil {
		return item, err
	}
	
	return item, nil
}

// saveBacklogItem saves a backlog item to a YAML file
func (s *APIServer) saveBacklogItem(folder string, item BacklogItem) error {
	// Generate filename if not provided
	if item.Filename == "" {
		// Find next available number
		files, _ := ioutil.ReadDir(folder)
		maxNum := 0
		for _, f := range files {
			if len(f.Name()) > 3 && f.Name()[3] == '-' {
				var num int
				fmt.Sscanf(f.Name(), "%03d-", &num)
				if num > maxNum {
					maxNum = num
				}
			}
		}
		
		// Generate filename with next number
		priority := "100" // Default medium priority
		if item.Priority == "high" {
			priority = fmt.Sprintf("%03d", maxNum+1)
			if maxNum+1 > 99 {
				priority = "099"
			}
		} else if item.Priority == "low" {
			priority = fmt.Sprintf("%03d", maxNum+200)
		}
		
		safeName := strings.ReplaceAll(strings.ToLower(item.Name), " ", "-")
		safeName = strings.ReplaceAll(safeName, "/", "-")
		item.Filename = fmt.Sprintf("%s-%s.yaml", priority, safeName)
	}
	
	filepath := filepath.Join(folder, item.Filename)
	
	// Marshal to YAML with nice formatting
	data, err := yaml.Marshal(&item)
	if err != nil {
		return err
	}
	
	// Add header comment
	header := fmt.Sprintf(`# %s
# Priority: %s
# Estimated Value: $%d

`, item.Name, strings.ToUpper(item.Priority), item.EstimatedRevenue)
	
	fullContent := header + string(data)
	
	return ioutil.WriteFile(filepath, []byte(fullContent), 0644)
}

// moveBacklogItem moves an item between backlog folders
func (s *APIServer) moveBacklogItem(item BacklogItem, fromFolder, toFolder string) error {
	oldPath := filepath.Join(fromFolder, item.Filename)
	newPath := filepath.Join(toFolder, item.Filename)
	
	// Read the file
	data, err := ioutil.ReadFile(oldPath)
	if err != nil {
		return err
	}
	
	// Write to new location
	err = ioutil.WriteFile(newPath, data, 0644)
	if err != nil {
		return err
	}
	
	// Remove from old location
	return os.Remove(oldPath)
}

// API Handlers

// handleGetBacklog returns all backlog items across all states
func (s *APIServer) handleGetBacklog(w http.ResponseWriter, r *http.Request) {
	result := map[string][]BacklogItem{}
	
	// Get items from each folder
	pending, err := s.getBacklogItems(backlogPending)
	if err != nil {
		http.Error(w, "Error loading pending items: "+err.Error(), http.StatusInternalServerError)
		return
	}
	result["pending"] = pending
	
	inProgress, err := s.getBacklogItems(backlogInProgress)
	if err != nil {
		http.Error(w, "Error loading in-progress items: "+err.Error(), http.StatusInternalServerError)
		return
	}
	result["in_progress"] = inProgress
	
	completed, err := s.getBacklogItems(backlogCompleted)
	if err != nil {
		http.Error(w, "Error loading completed items: "+err.Error(), http.StatusInternalServerError)
		return
	}
	result["completed"] = completed
	
	failed, err := s.getBacklogItems(backlogFailed)
	if err != nil {
		http.Error(w, "Error loading failed items: "+err.Error(), http.StatusInternalServerError)
		return
	}
	result["failed"] = failed
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleGetBacklogItem returns a specific backlog item
func (s *APIServer) handleGetBacklogItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	// Search all folders for the item
	folders := []string{backlogPending, backlogInProgress, backlogCompleted, backlogFailed}
	
	for _, folder := range folders {
		items, err := s.getBacklogItems(folder)
		if err != nil {
			continue
		}
		
		for _, item := range items {
			if item.ID == id || strings.Contains(item.Filename, id) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(item)
				return
			}
		}
	}
	
	http.Error(w, "Backlog item not found", http.StatusNotFound)
}

// handleCreateBacklogItem adds a new item to the backlog
func (s *APIServer) handleCreateBacklogItem(w http.ResponseWriter, r *http.Request) {
	var item BacklogItem
	
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if item.Name == "" || item.Description == "" || item.Prompt == "" {
		http.Error(w, "Missing required fields: name, description, or prompt", http.StatusBadRequest)
		return
	}
	
	// Set defaults
	if item.ID == "" {
		item.ID = strings.ReplaceAll(strings.ToLower(item.Name), " ", "-")
	}
	if item.Complexity == "" {
		item.Complexity = "intermediate"
	}
	if item.Category == "" {
		item.Category = "business-tool"
	}
	if item.Priority == "" {
		item.Priority = "medium"
	}
	if item.EstimatedRevenue == 0 {
		item.EstimatedRevenue = 25000
	}
	
	// Save to pending folder
	err = s.saveBacklogItem(backlogPending, item)
	if err != nil {
		http.Error(w, "Failed to save backlog item: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

// handleUpdateBacklogItem updates an existing backlog item
func (s *APIServer) handleUpdateBacklogItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	var updatedItem BacklogItem
	err := json.NewDecoder(r.Body).Decode(&updatedItem)
	if err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// Find the item in any folder
	folders := []string{backlogPending, backlogInProgress, backlogCompleted, backlogFailed}
	found := false
	var currentFolder string
	var existingItem BacklogItem
	
	for _, folder := range folders {
		items, err := s.getBacklogItems(folder)
		if err != nil {
			continue
		}
		
		for _, item := range items {
			if item.ID == id || strings.Contains(item.Filename, id) {
				found = true
				currentFolder = folder
				existingItem = item
				break
			}
		}
		if found {
			break
		}
	}
	
	if !found {
		http.Error(w, "Backlog item not found", http.StatusNotFound)
		return
	}
	
	// Preserve filename if not changed
	if updatedItem.Filename == "" {
		updatedItem.Filename = existingItem.Filename
	}
	
	// Delete old file
	oldPath := filepath.Join(currentFolder, existingItem.Filename)
	os.Remove(oldPath)
	
	// Save updated item
	err = s.saveBacklogItem(currentFolder, updatedItem)
	if err != nil {
		http.Error(w, "Failed to update backlog item: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedItem)
}

// handleDeleteBacklogItem removes an item from the backlog
func (s *APIServer) handleDeleteBacklogItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	// Find and delete the item
	folders := []string{backlogPending, backlogInProgress, backlogCompleted, backlogFailed}
	
	for _, folder := range folders {
		items, err := s.getBacklogItems(folder)
		if err != nil {
			continue
		}
		
		for _, item := range items {
			if item.ID == id || strings.Contains(item.Filename, id) {
				filepath := filepath.Join(folder, item.Filename)
				err := os.Remove(filepath)
				if err != nil {
					http.Error(w, "Failed to delete item: "+err.Error(), http.StatusInternalServerError)
					return
				}
				
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
	}
	
	http.Error(w, "Backlog item not found", http.StatusNotFound)
}

// handleGenerateFromBacklog moves an item to generation queue and starts processing
func (s *APIServer) handleGenerateFromBacklog(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	// Find the item in pending folder
	items, err := s.getBacklogItems(backlogPending)
	if err != nil {
		http.Error(w, "Error loading backlog: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	var foundItem BacklogItem
	found := false
	
	for _, item := range items {
		if item.ID == id || strings.Contains(item.Filename, id) {
			foundItem = item
			found = true
			break
		}
	}
	
	if !found {
		http.Error(w, "Backlog item not found in pending", http.StatusNotFound)
		return
	}
	
	// Move to in-progress
	err = s.moveBacklogItem(foundItem, backlogPending, backlogInProgress)
	if err != nil {
		http.Error(w, "Failed to move item to in-progress: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Create scenario in database and trigger generation
	scenario := Scenario{
		ID:               foundItem.ID,
		Name:             foundItem.Name,
		Description:      foundItem.Description,
		Prompt:           foundItem.Prompt,
		Complexity:       foundItem.Complexity,
		Category:         foundItem.Category,
		EstimatedRevenue: foundItem.EstimatedRevenue,
		ResourcesUsed:    foundItem.ResourcesRequired,
		Status:           "generating",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	
	// Insert into database
	query := `
		INSERT INTO scenarios (id, name, description, prompt, complexity, category, 
		                       estimated_revenue, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			prompt = EXCLUDED.prompt,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
		RETURNING id`
	
	var insertedID string
	err = s.db.QueryRow(query, 
		scenario.ID, scenario.Name, scenario.Description, scenario.Prompt,
		scenario.Complexity, scenario.Category, scenario.EstimatedRevenue,
		scenario.Status, scenario.CreatedAt, scenario.UpdatedAt).Scan(&insertedID)
	
	if err != nil {
		// Move back to pending on error
		s.moveBacklogItem(foundItem, backlogInProgress, backlogPending)
		http.Error(w, "Failed to create scenario: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// TODO: Trigger actual generation via n8n or Claude Code
	// For now, just return success
	
	response := map[string]interface{}{
		"message": "Generation started",
		"scenario_id": insertedID,
		"backlog_item": foundItem,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMoveBacklogItem moves an item between backlog states
func (s *APIServer) handleMoveBacklogItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	// Get target state from request body
	var moveRequest struct {
		ToState string `json:"to_state"`
	}
	
	err := json.NewDecoder(r.Body).Decode(&moveRequest)
	if err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// Map state to folder
	stateToFolder := map[string]string{
		"pending":     backlogPending,
		"in-progress": backlogInProgress,
		"in_progress": backlogInProgress,
		"completed":   backlogCompleted,
		"failed":      backlogFailed,
	}
	
	toFolder, ok := stateToFolder[moveRequest.ToState]
	if !ok {
		http.Error(w, "Invalid target state", http.StatusBadRequest)
		return
	}
	
	// Find the item
	folders := []string{backlogPending, backlogInProgress, backlogCompleted, backlogFailed}
	found := false
	var currentFolder string
	var foundItem BacklogItem
	
	for _, folder := range folders {
		items, err := s.getBacklogItems(folder)
		if err != nil {
			continue
		}
		
		for _, item := range items {
			if item.ID == id || strings.Contains(item.Filename, id) {
				found = true
				currentFolder = folder
				foundItem = item
				break
			}
		}
		if found {
			break
		}
	}
	
	if !found {
		http.Error(w, "Backlog item not found", http.StatusNotFound)
		return
	}
	
	// Move the item
	err = s.moveBacklogItem(foundItem, currentFolder, toFolder)
	if err != nil {
		http.Error(w, "Failed to move item: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	foundItem.Status = moveRequest.ToState
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(foundItem)
}