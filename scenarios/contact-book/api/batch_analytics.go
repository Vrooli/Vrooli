package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/lib/pq"
)

// BatchAnalyticsProcessor handles batch analytics computations
type BatchAnalyticsProcessor struct {
	db *sql.DB
}

// NewBatchAnalyticsProcessor creates a new batch processor
func NewBatchAnalyticsProcessor(database *sql.DB) *BatchAnalyticsProcessor {
	return &BatchAnalyticsProcessor{
		db: database,
	}
}

// ProcessAnalytics runs batch analytics processing
func (b *BatchAnalyticsProcessor) ProcessAnalytics() error {
	log.Println("Starting batch analytics processing...")

	// 1. Update relationship strengths based on interaction patterns
	if err := b.updateRelationshipStrengths(); err != nil {
		log.Printf("Failed to update relationship strengths: %v", err)
	}

	// 2. Calculate closeness scores
	if err := b.calculateClosenessScores(); err != nil {
		log.Printf("Failed to calculate closeness scores: %v", err)
	}

	// 3. Calculate maintenance priority scores
	if err := b.calculateMaintenancePriority(); err != nil {
		log.Printf("Failed to calculate maintenance priority: %v", err)
	}

	// 4. Identify shared interests
	if err := b.identifySharedInterests(); err != nil {
		log.Printf("Failed to identify shared interests: %v", err)
	}

	// 5. Update computed signals
	if err := b.updateComputedSignals(); err != nil {
		log.Printf("Failed to update computed signals: %v", err)
	}

	log.Println("Batch analytics processing completed")
	return nil
}

// updateRelationshipStrengths updates relationship strength based on interaction patterns
func (b *BatchAnalyticsProcessor) updateRelationshipStrengths() error {
	query := `
		UPDATE relationships
		SET strength = CASE
			WHEN last_contact_date > NOW() - INTERVAL '7 days' THEN LEAST(strength + 0.1, 1.0)
			WHEN last_contact_date > NOW() - INTERVAL '30 days' THEN strength
			WHEN last_contact_date > NOW() - INTERVAL '90 days' THEN GREATEST(strength - 0.05, 0.1)
			ELSE GREATEST(strength - 0.1, 0.0)
		END,
		updated_at = NOW()
		WHERE deleted_at IS NULL`

	_, err := b.db.Exec(query)
	return err
}

// calculateClosenessScores calculates overall closeness scores for each person
func (b *BatchAnalyticsProcessor) calculateClosenessScores() error {
	// Get all persons
	personsQuery := `SELECT id FROM persons WHERE deleted_at IS NULL`
	rows, err := b.db.Query(personsQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var personID string
		if err := rows.Scan(&personID); err != nil {
			continue
		}

		// Calculate average relationship strength for this person
		avgQuery := `
			SELECT AVG(strength) as avg_strength
			FROM relationships
			WHERE (from_person_id = $1 OR to_person_id = $1)
			AND deleted_at IS NULL`

		var avgStrength sql.NullFloat64
		if err := b.db.QueryRow(avgQuery, personID).Scan(&avgStrength); err != nil {
			continue
		}

		closenessScore := 0.5 // Default score
		if avgStrength.Valid {
			closenessScore = avgStrength.Float64
		}

		// Update computed signals with closeness score
		updateQuery := `
			UPDATE persons
			SET computed_signals = computed_signals || $1::jsonb,
			    updated_at = NOW()
			WHERE id = $2`

		signals := map[string]interface{}{
			"overall_closeness_score": closenessScore,
			"last_computed":          time.Now(),
		}
		signalsJSON, _ := json.Marshal(signals)

		if _, err := b.db.Exec(updateQuery, signalsJSON, personID); err != nil {
			log.Printf("Failed to update closeness score for person %s: %v", personID, err)
		}
	}

	return nil
}

// calculateMaintenancePriority calculates which relationships need attention
func (b *BatchAnalyticsProcessor) calculateMaintenancePriority() error {
	// Get all persons with their last contact dates
	query := `
		SELECT p.id,
		       MIN(r.last_contact_date) as oldest_contact,
		       AVG(r.strength) as avg_strength,
		       COUNT(r.id) as relationship_count
		FROM persons p
		LEFT JOIN relationships r ON (r.from_person_id = p.id OR r.to_person_id = p.id)
		WHERE p.deleted_at IS NULL AND r.deleted_at IS NULL
		GROUP BY p.id`

	rows, err := b.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var personID string
		var oldestContact sql.NullTime
		var avgStrength sql.NullFloat64
		var relationshipCount int

		if err := rows.Scan(&personID, &oldestContact, &avgStrength, &relationshipCount); err != nil {
			continue
		}

		// Calculate maintenance priority score
		priorityScore := 0.0

		if oldestContact.Valid {
			daysSinceContact := time.Since(oldestContact.Time).Hours() / 24
			if daysSinceContact > 90 {
				priorityScore += 0.5
			} else if daysSinceContact > 30 {
				priorityScore += 0.3
			} else if daysSinceContact > 7 {
				priorityScore += 0.1
			}
		}

		if avgStrength.Valid && avgStrength.Float64 < 0.5 {
			priorityScore += 0.3
		}

		if relationshipCount == 0 {
			priorityScore += 0.2
		}

		// Update computed signals
		updateQuery := `
			UPDATE persons
			SET computed_signals = jsonb_set(
				COALESCE(computed_signals, '{}'::jsonb),
				'{maintenance_priority_score}',
				to_jsonb($1::float)
			),
			updated_at = NOW()
			WHERE id = $2`

		if _, err := b.db.Exec(updateQuery, priorityScore, personID); err != nil {
			log.Printf("Failed to update maintenance priority for person %s: %v", personID, err)
		}
	}

	return nil
}

// identifySharedInterests finds common tags and interests between connected persons
func (b *BatchAnalyticsProcessor) identifySharedInterests() error {
	query := `
		SELECT r.id, r.from_person_id, r.to_person_id,
		       p1.tags as tags1, p2.tags as tags2
		FROM relationships r
		JOIN persons p1 ON p1.id = r.from_person_id
		JOIN persons p2 ON p2.id = r.to_person_id
		WHERE r.deleted_at IS NULL
		AND p1.deleted_at IS NULL
		AND p2.deleted_at IS NULL`

	rows, err := b.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var relID, fromID, toID string
		var tags1, tags2 []string

		if err := rows.Scan(&relID, &fromID, &toID, pq.Array(&tags1), pq.Array(&tags2)); err != nil {
			continue
		}

		// Find common tags
		sharedInterests := findCommonStrings(tags1, tags2)

		if len(sharedInterests) > 0 {
			updateQuery := `
				UPDATE relationships
				SET shared_interests = $1,
				    updated_at = NOW()
				WHERE id = $2`

			if _, err := b.db.Exec(updateQuery, pq.Array(sharedInterests), relID); err != nil {
				log.Printf("Failed to update shared interests for relationship %s: %v", relID, err)
			}
		}
	}

	return nil
}

// updateComputedSignals aggregates all analytics into computed signals
func (b *BatchAnalyticsProcessor) updateComputedSignals() error {
	query := `
		UPDATE persons
		SET computed_signals = jsonb_set(
			COALESCE(computed_signals, '{}'::jsonb),
			'{last_batch_update}',
			to_jsonb(NOW())
		),
		updated_at = NOW()
		WHERE deleted_at IS NULL`

	_, err := b.db.Exec(query)
	return err
}

// Helper function to find common strings between two slices
func findCommonStrings(slice1, slice2 []string) []string {
	common := []string{}
	set := make(map[string]bool)

	for _, s := range slice1 {
		set[s] = true
	}

	for _, s := range slice2 {
		if set[s] {
			common = append(common, s)
		}
	}

	return common
}

// RunBatchProcessing starts the batch processing in a goroutine
func RunBatchProcessing(database *sql.DB, interval time.Duration) {
	processor := NewBatchAnalyticsProcessor(database)

	// Run immediately on startup
	go processor.ProcessAnalytics()

	// Then run periodically
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := processor.ProcessAnalytics(); err != nil {
				log.Printf("Batch processing failed: %v", err)
			}
		}
	}()
}