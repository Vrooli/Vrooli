package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// Analyze image with AI
func (app *App) analyzeImageWithAI(fileID, storagePath string) {
	// Call Ollama vision model for image analysis
	payload := map[string]interface{}{
		"model": "llava:13b",
		"prompt": "Describe this image in detail. Identify objects, people, scenes, and any text visible.",
		"images": []string{storagePath},
	}
	
	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(app.OllamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to analyze image %s: %v", fileID, err)
		return
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	if description, ok := result["response"].(string); ok {
		// Update file with AI description
		app.updateFileDescription(fileID, description)
		
		// Extract detected objects
		objects := app.extractObjectsFromDescription(description)
		app.updateFileObjects(fileID, objects)
	}
}

// Generate embeddings for semantic search
func (app *App) generateEmbeddingsJob(job ProcessingJob) {
	// Get file description and content
	var description, ocrText string
	query := `SELECT description, ocr_text FROM files WHERE id = $1`
	app.DB.QueryRow(query, job.FileID).Scan(&description, &ocrText)
	
	content := description
	if ocrText != "" {
		content += " " + ocrText
	}
	
	if content == "" {
		return
	}
	
	// Generate embedding with Ollama
	payload := map[string]interface{}{
		"model": "nomic-embed-text",
		"prompt": content,
	}
	
	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(app.OllamaURL+"/api/embeddings", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to generate embeddings for %s: %v", job.FileID, err)
		return
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	if embedding, ok := result["embedding"].([]interface{}); ok {
		// Store in Qdrant
		app.storeEmbedding(job.FileID, embedding)
	}
}

// Generate document summary with AI
func (app *App) generateDocumentSummary(fileID string) {
	// Get document text
	var ocrText string
	query := `SELECT ocr_text FROM files WHERE id = $1`
	app.DB.QueryRow(query, fileID).Scan(&ocrText)
	
	if ocrText == "" {
		return
	}
	
	// Truncate if too long
	if len(ocrText) > 4000 {
		ocrText = ocrText[:4000]
	}
	
	// Generate summary with Ollama
	payload := map[string]interface{}{
		"model": "llama3.2",
		"prompt": "Summarize this document in 2-3 sentences: " + ocrText,
	}
	
	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(app.OllamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to generate summary for %s: %v", fileID, err)
		return
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	if summary, ok := result["response"].(string); ok {
		app.updateFileDescription(fileID, summary)
	}
}

// Organize files using AI suggestions
func (app *App) organizeSmartAI(fileID string) {
	// Get file info
	var filename, description string
	var fileType *string
	query := `SELECT original_name, description, file_type FROM files WHERE id = $1`
	app.DB.QueryRow(query, fileID).Scan(&filename, &description, &fileType)
	
	// Build context for AI
	context := "Filename: " + filename
	if description != "" {
		context += "\nDescription: " + description
	}
	if fileType != nil {
		context += "\nType: " + *fileType
	}
	
	// Ask AI for organization suggestion
	payload := map[string]interface{}{
		"model": "llama3.2",
		"prompt": "Suggest a folder path for this file. Reply with only the path, no explanation. " + context,
	}
	
	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(app.OllamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to get organization suggestion for %s: %v", fileID, err)
		return
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	
	if suggestion, ok := result["response"].(string); ok {
		// Clean up the suggestion
		suggestion = strings.TrimSpace(suggestion)
		suggestion = strings.Trim(suggestion, "\"'")
		
		// Update file folder path
		updateQuery := `UPDATE files SET folder_path = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
		app.DB.Exec(updateQuery, suggestion, fileID)
		
		// Create organization suggestion
		app.createOrganizationSuggestion(fileID, suggestion, "AI recommendation based on content analysis")
	}
}

// Organize by file type
func (app *App) organizeByType(fileID string) {
	var fileType *string
	query := `SELECT file_type FROM files WHERE id = $1`
	app.DB.QueryRow(query, fileID).Scan(&fileType)
	
	folder := "/Uncategorized"
	if fileType != nil {
		switch *fileType {
		case "image":
			folder = "/Images"
		case "document":
			folder = "/Documents"
		case "video":
			folder = "/Videos"
		case "audio":
			folder = "/Audio"
		default:
			folder = "/Other"
		}
	}
	
	updateQuery := `UPDATE files SET folder_path = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	app.DB.Exec(updateQuery, folder, fileID)
}

// Organize by date
func (app *App) organizeByDate(fileID string) {
	var uploadedAt time.Time
	query := `SELECT uploaded_at FROM files WHERE id = $1`
	app.DB.QueryRow(query, fileID).Scan(&uploadedAt)
	
	folder := fmt.Sprintf("/%d/%02d", uploadedAt.Year(), uploadedAt.Month())
	
	updateQuery := `UPDATE files SET folder_path = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	app.DB.Exec(updateQuery, folder, fileID)
}

// Organize by content analysis
func (app *App) organizeByContent(fileID string) {
	// Use detected objects and tags to organize
	var detectedObjects json.RawMessage
	var tags []string
	query := `SELECT detected_objects, tags FROM files WHERE id = $1`
	app.DB.QueryRow(query, fileID).Scan(&detectedObjects, &tags)
	
	folder := "/Content"
	
	// Parse detected objects
	var objects []string
	json.Unmarshal(detectedObjects, &objects)
	
	// Determine folder based on content
	if contains(objects, "person") || contains(objects, "face") {
		folder = "/People"
	} else if contains(objects, "animal") || contains(objects, "pet") {
		folder = "/Animals"
	} else if contains(objects, "food") {
		folder = "/Food"
	} else if contains(objects, "landscape") || contains(objects, "nature") {
		folder = "/Nature"
	} else if contains(objects, "building") || contains(objects, "architecture") {
		folder = "/Architecture"
	} else if contains(tags, "document") || contains(objects, "text") {
		folder = "/Documents"
	}
	
	updateQuery := `UPDATE files SET folder_path = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	app.DB.Exec(updateQuery, folder, fileID)
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}