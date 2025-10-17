package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

// Check for duplicate files
func (app *App) checkDuplicatesJob(job ProcessingJob) {
	req := job.Payload.(UploadRequest)
	
	// Check by file hash
	duplicates := app.findDuplicatesByHash(req.FileHash, job.FileID)
	
	if len(duplicates) > 0 {
		// Create suggestion for duplicate
		app.createDuplicateSuggestion(job.FileID, duplicates)
	}
	
	// For images, also check visual similarity
	if strings.HasPrefix(req.MimeType, "image/") {
		similarImages := app.findSimilarImages(job.FileID)
		if len(similarImages) > 0 {
			app.createSimilaritySuggestion(job.FileID, similarImages)
		}
	}
}

// Find duplicates by hash
func (app *App) findDuplicatesByHash(fileHash, excludeID string) []string {
	query := `SELECT id FROM files WHERE file_hash = $1 AND id != $2`
	rows, err := app.DB.Query(query, fileHash, excludeID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	
	var duplicates []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		duplicates = append(duplicates, id)
	}
	return duplicates
}

// Find similar images using Qdrant vector search
func (app *App) findSimilarImages(fileID string) []string {
	// First, get the embedding for this file
	var embedding []float32
	
	// Query Qdrant for the file's vector
	resp, err := http.Get(app.QdrantURL + "/collections/files/points/" + fileID)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	
	var pointResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&pointResult)
	
	if result, ok := pointResult["result"].(map[string]interface{}); ok {
		if vector, ok := result["vector"].([]interface{}); ok {
			embedding = make([]float32, len(vector))
			for i, v := range vector {
				if f, ok := v.(float64); ok {
					embedding[i] = float32(f)
				}
			}
		}
	}
	
	if len(embedding) == 0 {
		return nil
	}
	
	// Search for similar vectors
	payload := map[string]interface{}{
		"vector": embedding,
		"limit": 10,
		"score_threshold": 0.85,
		"filter": map[string]interface{}{
			"must_not": []map[string]interface{}{
				{"has_id": fileID},
			},
		},
	}
	
	jsonData, _ := json.Marshal(payload)
	searchResp, err := http.Post(app.QdrantURL+"/collections/files/points/search", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil
	}
	defer searchResp.Body.Close()
	
	var searchResult map[string]interface{}
	json.NewDecoder(searchResp.Body).Decode(&searchResult)
	
	var similar []string
	if results, ok := searchResult["result"].([]interface{}); ok {
		for _, r := range results {
			if item, ok := r.(map[string]interface{}); ok {
				if id, ok := item["id"].(string); ok {
					similar = append(similar, id)
				}
			}
		}
	}
	return similar
}

// Store embedding in Qdrant
func (app *App) storeEmbedding(fileID string, embedding []interface{}) {
	// Convert embedding to float array
	vector := make([]float32, len(embedding))
	for i, v := range embedding {
		if f, ok := v.(float64); ok {
			vector[i] = float32(f)
		}
	}
	
	// Store in Qdrant
	payload := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id": fileID,
				"vector": vector,
				"payload": map[string]interface{}{
					"file_id": fileID,
				},
			},
		},
	}
	
	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", app.QdrantURL+"/collections/files/points", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	client.Do(req)
}

// Create duplicate suggestion
func (app *App) createDuplicateSuggestion(fileID string, duplicates []string) {
	query := `
		INSERT INTO suggestions (file_id, type, status, suggested_value, reason, similar_file_ids, confidence, created_at)
		VALUES ($1, 'duplicate', 'pending', 'merge_or_delete', 'Exact duplicate files found', $2, 1.0, CURRENT_TIMESTAMP)
	`
	duplicatesJSON, _ := json.Marshal(duplicates)
	app.DB.Exec(query, fileID, duplicatesJSON)
}

// Create similarity suggestion
func (app *App) createSimilaritySuggestion(fileID string, similar []string) {
	query := `
		INSERT INTO suggestions (file_id, type, status, suggested_value, reason, similar_file_ids, confidence, created_at)
		VALUES ($1, 'similar', 'pending', 'review_similarity', 'Similar images found', $2, 0.85, CURRENT_TIMESTAMP)
	`
	similarJSON, _ := json.Marshal(similar)
	app.DB.Exec(query, fileID, similarJSON)
}

// Create organization suggestion
func (app *App) createOrganizationSuggestion(fileID, folderPath, reason string) {
	query := `
		INSERT INTO suggestions (file_id, type, status, suggested_value, reason, confidence, created_at)
		VALUES ($1, 'organization', 'pending', $2, $3, 0.75, CURRENT_TIMESTAMP)
	`
	app.DB.Exec(query, fileID, folderPath, reason)
}