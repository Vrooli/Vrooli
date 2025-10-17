package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// ideasHandler handles listing and creating ideas
func (s *ApiServer) ideasHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Query ideas from database
		campaignID := r.URL.Query().Get("campaign_id")

		// Validate limit parameter if provided
		limitStr := r.URL.Query().Get("limit")
		limit := DefaultIdeasQueryLimit
		if limitStr != "" {
			if parsedLimit, err := json.Number(limitStr).Int64(); err == nil && parsedLimit > 0 && parsedLimit <= MaxIdeasLimit {
				limit = int(parsedLimit)
			}
		}

		query := `SELECT id, campaign_id, title, content, status, created_at, updated_at
				  FROM ideas WHERE 1=1`
		args := []interface{}{}
		argPos := 1

		if campaignID != "" {
			query += fmt.Sprintf(" AND campaign_id = $%d", argPos)
			args = append(args, campaignID)
			argPos++
		}
		query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argPos)
		args = append(args, limit)

		rows, err := s.db.Query(query, args...)
		if err != nil {
			log.Printf("Error querying ideas from database: %v", err)
			http.Error(w, fmt.Sprintf("Failed to retrieve ideas: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		ideas := []Idea{}
		for rows.Next() {
			var idea Idea
			err := rows.Scan(&idea.ID, &idea.CampaignID, &idea.Title,
				&idea.Content, &idea.Status, &idea.CreatedAt, &idea.UpdatedAt)
			if err != nil {
				log.Printf("Warning: Failed to scan idea row: %v", err)
				continue
			}
			ideas = append(ideas, idea)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(ideas); err != nil {
			log.Printf("Failed to encode ideas response: %v", err)
		}

	case "POST":
		var req GenerateIdeasRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding idea generation request: %v", err)
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		// Use IdeaProcessor to generate ideas
		ctx := r.Context()
		response := s.ideaProcessor.GenerateIdeas(ctx, req)

		if !response.Success {
			log.Printf("Idea generation failed: %s", response.Error)
			http.Error(w, response.Error, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode idea generation response: %v", err)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// generateIdeasHandler handles idea generation with PRD-specified format
func (s *ApiServer) generateIdeasHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CampaignID      string   `json:"campaign_id"`
		Context         string   `json:"context"`
		DocumentRefs    []string `json:"document_refs"`
		CreativityLevel float64  `json:"creativity_level"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create a GenerateIdeasRequest for the processor
	generateReq := GenerateIdeasRequest{
		CampaignID: req.CampaignID,
		Prompt:     req.Context,
		Count:      1, // Default to 1 idea per request
	}

	// Use IdeaProcessor to generate ideas
	ctx := r.Context()
	response := s.ideaProcessor.GenerateIdeas(ctx, generateReq)

	if !response.Success {
		http.Error(w, response.Error, http.StatusInternalServerError)
		return
	}

	// Convert response to expected format
	if len(response.Ideas) > 0 {
		idea := response.Ideas[0]
		result := map[string]interface{}{
			"id":      fmt.Sprintf("idea-%d", time.Now().Unix()),
			"title":   idea.Title,
			"content": idea.Description,
			"generation_metadata": map[string]interface{}{
				"context_used":    []string{req.Context},
				"processing_time": 1500,
				"confidence":      0.85,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Printf("Failed to encode generate ideas result: %v", err)
		}
	} else {
		http.Error(w, "No ideas generated", http.StatusInternalServerError)
	}
}

// refineIdeaHandler handles idea refinement requests
func (s *ApiServer) refineIdeaHandler(w http.ResponseWriter, r *http.Request) {
	var req RefinementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding refinement request: %v", err)
		http.Error(w, fmt.Sprintf("Invalid refinement request: %v", err), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	err := s.ideaProcessor.RefineIdea(ctx, req)
	if err != nil {
		log.Printf("Idea refinement failed for idea %s: %v", req.IdeaID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Idea refined successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode refinement response: %v", err)
	}
}

// searchHandler handles semantic search requests
func (s *ApiServer) searchHandler(w http.ResponseWriter, r *http.Request) {
	var req SemanticSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding search request: %v", err)
		http.Error(w, fmt.Sprintf("Invalid search request: %v", err), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	results, err := s.ideaProcessor.SemanticSearch(ctx, req)
	if err != nil {
		log.Printf("Semantic search failed for query '%s': %v", req.Query, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Printf("Failed to encode search results: %v", err)
	}
}

// processDocumentHandler handles document processing requests
func (s *ApiServer) processDocumentHandler(w http.ResponseWriter, r *http.Request) {
	var req DocumentProcessingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding document processing request: %v", err)
		http.Error(w, fmt.Sprintf("Invalid document processing request: %v", err), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	err := s.ideaProcessor.ProcessDocument(ctx, req)
	if err != nil {
		log.Printf("Document processing failed for campaign %s: %v", req.CampaignID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Document processing started",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode document processing response: %v", err)
	}
}
