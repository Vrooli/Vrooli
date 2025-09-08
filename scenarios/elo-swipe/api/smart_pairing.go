package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SmartPairing struct {
	db *sql.DB
}

type PairingRequest struct {
	ListID string      `json:"list_id"`
	Items  []PairItem  `json:"items"`
}

type PairItem struct {
	ID      string          `json:"id"`
	Content json.RawMessage `json:"content"`
}

type PairingResponse struct {
	Success        bool               `json:"success"`
	PairsSuggested int                `json:"pairs_suggested"`
	Message        string             `json:"message"`
	SuggestedPairs []SuggestedPair    `json:"suggested_pairs,omitempty"`
}

type SuggestedPair struct {
	ItemAID         string  `json:"item_a_id"`
	ItemBID         string  `json:"item_b_id"`
	Priority        float64 `json:"priority"`
	UncertaintyScore float64 `json:"uncertainty_score"`
}

type AIResponse struct {
	SuggestedPairs []SuggestedPair `json:"suggested_pairs"`
}

func NewSmartPairing(db *sql.DB) *SmartPairing {
	return &SmartPairing{
		db: db,
	}
}

func (sp *SmartPairing) GenerateSmartPairs(ctx context.Context, req PairingRequest) (*PairingResponse, error) {
	if len(req.Items) < 2 {
		return &PairingResponse{
			Success: false,
			Message: "Need at least 2 items to generate pairs",
		}, fmt.Errorf("insufficient items")
	}

	// Build prompt for AI analysis
	itemsData, err := json.Marshal(req.Items)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal items: %w", err)
	}

	prompt := fmt.Sprintf(`Analyze these items for ranking comparisons. Suggest which pairs would be most informative to compare, considering similarity and importance:

Items: %s

Return a JSON object with 'suggested_pairs' array, each containing 'item_a_id', 'item_b_id', and 'priority' (0-100).`, string(itemsData))

	// Call Ollama for AI analysis
	aiResponse, err := sp.callOllamaGenerate(ctx, prompt, "llama3.2", "analysis")
	if err != nil {
		return nil, fmt.Errorf("failed to generate AI analysis: %w", err)
	}

	// Parse AI response
	suggestedPairs, err := sp.parseAIResponse(aiResponse)
	if err != nil {
		// Fallback to random pairing if AI fails
		suggestedPairs = sp.generateFallbackPairs(req.Items)
	}

	// Store pairs in database
	storedCount := 0
	for _, pair := range suggestedPairs {
		err := sp.storePairingQueue(req.ListID, pair)
		if err != nil {
			// Log error but continue with other pairs
			continue
		}
		storedCount++
	}

	return &PairingResponse{
		Success:        storedCount > 0,
		PairsSuggested: storedCount,
		Message:        "Smart pairing suggestions generated",
		SuggestedPairs: suggestedPairs,
	}, nil
}

func (sp *SmartPairing) callOllamaGenerate(ctx context.Context, prompt, model, taskType string) (string, error) {
	cmd := exec.CommandContext(ctx, "bash", "/vrooli/cli/vrooli", "resource", "ollama", "generate", prompt, "--model", model, "--type", taskType, "--quiet")
	
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("ollama command failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

func (sp *SmartPairing) parseAIResponse(response string) ([]SuggestedPair, error) {
	var suggestedPairs []SuggestedPair

	// Clean response and extract JSON
	cleanResponse := strings.TrimSpace(response)
	cleanResponse = regexp.MustCompile("```json\\n?|```\\n?").ReplaceAllString(cleanResponse, "")
	
	// Try to find JSON object
	jsonMatch := regexp.MustCompile(`\{[\s\S]*\}`).FindString(cleanResponse)
	if jsonMatch == "" {
		return nil, fmt.Errorf("no JSON object found in AI response")
	}

	var aiResp AIResponse
	if err := json.Unmarshal([]byte(jsonMatch), &aiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AI response: %w", err)
	}

	// Validate and process pairs
	for _, pair := range aiResp.SuggestedPairs {
		if pair.ItemAID != "" && pair.ItemBID != "" && pair.ItemAID != pair.ItemBID {
			// Calculate uncertainty score from priority
			pair.UncertaintyScore = pair.Priority / 100.0
			suggestedPairs = append(suggestedPairs, pair)
		}
	}

	return suggestedPairs, nil
}

func (sp *SmartPairing) generateFallbackPairs(items []PairItem) []SuggestedPair {
	var pairs []SuggestedPair
	
	if len(items) >= 2 {
		pair := SuggestedPair{
			ItemAID:          items[0].ID,
			ItemBID:          items[1].ID,
			Priority:         50.0,
			UncertaintyScore: 0.5,
		}
		pairs = append(pairs, pair)
	}

	// Generate a few more random pairs if we have enough items
	for i := 0; i < len(items)-1 && i < 3; i++ {
		for j := i + 1; j < len(items) && j < i+3; j++ {
			if i == 0 && j == 1 {
				continue // Already added this pair
			}
			
			pair := SuggestedPair{
				ItemAID:          items[i].ID,
				ItemBID:          items[j].ID,
				Priority:         40.0,
				UncertaintyScore: 0.4,
			}
			pairs = append(pairs, pair)
		}
	}

	return pairs
}

func (sp *SmartPairing) storePairingQueue(listID string, pair SuggestedPair) error {
	if sp.db == nil {
		return fmt.Errorf("database connection not available")
	}

	query := `
		INSERT INTO elo_swipe.pairing_queue (list_id, item_a_id, item_b_id, priority, uncertainty_score)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (list_id, item_a_id, item_b_id) DO UPDATE
		SET priority = EXCLUDED.priority,
		    uncertainty_score = EXCLUDED.uncertainty_score,
		    suggested_at = CURRENT_TIMESTAMP
		RETURNING id;
	`

	var pairID string
	err := sp.db.QueryRow(query, listID, pair.ItemAID, pair.ItemBID, pair.Priority, pair.UncertaintyScore).Scan(&pairID)
	if err != nil {
		return fmt.Errorf("failed to store pairing queue item: %w", err)
	}

	return nil
}

func (sp *SmartPairing) GetQueuedPairs(ctx context.Context, listID string, limit int) ([]SuggestedPair, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT item_a_id, item_b_id, priority, uncertainty_score
		FROM elo_swipe.pairing_queue
		WHERE list_id = $1
		ORDER BY priority DESC, suggested_at DESC
		LIMIT $2
	`

	rows, err := sp.db.Query(query, listID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pairing queue: %w", err)
	}
	defer rows.Close()

	var pairs []SuggestedPair
	for rows.Next() {
		var pair SuggestedPair
		err := rows.Scan(&pair.ItemAID, &pair.ItemBID, &pair.Priority, &pair.UncertaintyScore)
		if err != nil {
			continue
		}
		pairs = append(pairs, pair)
	}

	return pairs, nil
}

func (sp *SmartPairing) ClearQueue(ctx context.Context, listID string) error {
	query := `DELETE FROM elo_swipe.pairing_queue WHERE list_id = $1`
	
	_, err := sp.db.Exec(query, listID)
	if err != nil {
		return fmt.Errorf("failed to clear pairing queue: %w", err)
	}

	return nil
}

func (sp *SmartPairing) RefreshSmartPairs(ctx context.Context, listID string) (*PairingResponse, error) {
	// Get all items for the list
	items, err := sp.getListItems(listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list items: %w", err)
	}

	// Clear existing queue
	if err := sp.ClearQueue(ctx, listID); err != nil {
		return nil, fmt.Errorf("failed to clear existing queue: %w", err)
	}

	// Generate new pairs
	req := PairingRequest{
		ListID: listID,
		Items:  items,
	}

	return sp.GenerateSmartPairs(ctx, req)
}

func (sp *SmartPairing) getListItems(listID string) ([]PairItem, error) {
	query := `
		SELECT id, content
		FROM elo_swipe.items
		WHERE list_id = $1
		ORDER BY created_at
	`

	rows, err := sp.db.Query(query, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PairItem
	for rows.Next() {
		var item PairItem
		err := rows.Scan(&item.ID, &item.Content)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	return items, nil
}