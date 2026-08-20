package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/interfacegraph"
)

type InterfaceGraphCacheEntry struct {
	Signature  string
	Graph      interfacegraph.Graph
	ComputedAt time.Time
}

func (s *Store) LoadInterfaceGraphCache(signature string) (InterfaceGraphCacheEntry, bool, error) {
	if s == nil || s.db == nil {
		return InterfaceGraphCacheEntry{}, false, nil
	}
	var payload string
	var computedAtRaw string
	err := s.db.QueryRow(`
		SELECT graph_json, computed_at
		FROM interface_graph_cache
		WHERE fleet_signature = ?`, signature).Scan(&payload, &computedAtRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return InterfaceGraphCacheEntry{}, false, nil
	}
	if err != nil {
		return InterfaceGraphCacheEntry{}, false, fmt.Errorf("load interface graph cache: %w", err)
	}
	computedAt, err := time.Parse(time.RFC3339Nano, computedAtRaw)
	if err != nil {
		return InterfaceGraphCacheEntry{}, false, fmt.Errorf("parse interface graph cache timestamp: %w", err)
	}
	var graph interfacegraph.Graph
	if err := json.Unmarshal([]byte(payload), &graph); err != nil {
		return InterfaceGraphCacheEntry{}, false, fmt.Errorf("decode interface graph cache: %w", err)
	}
	return InterfaceGraphCacheEntry{Signature: signature, Graph: graph, ComputedAt: computedAt}, true, nil
}

func (s *Store) StoreInterfaceGraphCache(entry InterfaceGraphCacheEntry) error {
	if s == nil || s.db == nil {
		return nil
	}
	if entry.Signature == "" {
		return errors.New("interface graph cache signature is required")
	}
	if entry.ComputedAt.IsZero() {
		entry.ComputedAt = time.Now().UTC()
	}
	payload, err := json.Marshal(entry.Graph)
	if err != nil {
		return fmt.Errorf("encode interface graph cache: %w", err)
	}
	_, err = s.db.Exec(`
		INSERT INTO interface_graph_cache (fleet_signature, graph_json, computed_at)
		VALUES (?, ?, ?)
		ON CONFLICT(fleet_signature) DO UPDATE SET
			graph_json = excluded.graph_json,
			computed_at = excluded.computed_at`,
		entry.Signature,
		string(payload),
		entry.ComputedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("store interface graph cache: %w", err)
	}
	return nil
}
