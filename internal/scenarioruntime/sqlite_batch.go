package scenarioruntime

import (
	"context"
	"fmt"
)

// batchIDChunkSize caps bound parameters per IN(...) query, comfortably under
// SQLite's default 999-variable limit.
const batchIDChunkSize = 500

// GetInstances returns the instances for the given IDs keyed by instance_id.
// Missing IDs are simply absent from the map (no zero-valued entries). Rows
// are fully materialized per chunk before the next query is issued (the
// store's pool size is 1 — a nested query inside an open rows loop would
// deadlock).
func (s *SQLiteStore) GetInstances(ctx context.Context, instanceIDs []string) (map[string]Instance, error) {
	out := make(map[string]Instance, len(instanceIDs))
	for _, chunk := range chunkIDs(dedupeIDs(instanceIDs), batchIDChunkSize) {
		query := instanceSelectSQL + ` WHERE instance_id IN (` + placeholders(len(chunk)) + `)`
		rows, err := s.db.QueryContext(ctx, query, idArgs(chunk)...)
		if err != nil {
			return nil, fmt.Errorf("get runtime instances: %w", err)
		}
		instances, scanErr := scanInstances(rows)
		closeErr := rows.Close()
		if scanErr != nil {
			return nil, scanErr
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close runtime instances rows: %w", closeErr)
		}
		for _, instance := range instances {
			out[instance.InstanceID] = instance
		}
	}
	return out, nil
}

// ListProcessRefsForInstances returns every process ref of the given
// instances grouped by instance_id, each group in the same order
// ListProcessRefs would return it.
func (s *SQLiteStore) ListProcessRefsForInstances(ctx context.Context, instanceIDs []string) (map[string][]ProcessRef, error) {
	out := make(map[string][]ProcessRef, len(instanceIDs))
	for _, chunk := range chunkIDs(dedupeIDs(instanceIDs), batchIDChunkSize) {
		query := processRefSelectSQL + ` WHERE instance_id IN (` + placeholders(len(chunk)) + `) ORDER BY started_at ASC, ref_id ASC`
		rows, err := s.db.QueryContext(ctx, query, idArgs(chunk)...)
		if err != nil {
			return nil, fmt.Errorf("list runtime process refs: %w", err)
		}
		refs, scanErr := scanProcessRefs(rows)
		closeErr := rows.Close()
		if scanErr != nil {
			return nil, scanErr
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close runtime process refs rows: %w", closeErr)
		}
		for _, ref := range refs {
			out[ref.InstanceID] = append(out[ref.InstanceID], ref)
		}
	}
	return out, nil
}

// GetHealthSnapshots returns the health snapshots for the given instance IDs
// keyed by instance_id; instances without a snapshot are absent from the map.
func (s *SQLiteStore) GetHealthSnapshots(ctx context.Context, instanceIDs []string) (map[string]HealthSnapshot, error) {
	out := make(map[string]HealthSnapshot, len(instanceIDs))
	for _, chunk := range chunkIDs(dedupeIDs(instanceIDs), batchIDChunkSize) {
		query := `
SELECT instance_id, scenario, status, readiness, checked_at, latency_ms, error, response_json, schema_valid
FROM runtime_health_snapshots
WHERE instance_id IN (` + placeholders(len(chunk)) + `)`
		rows, err := s.db.QueryContext(ctx, query, idArgs(chunk)...)
		if err != nil {
			return nil, fmt.Errorf("get health snapshots: %w", err)
		}
		var snapshots []HealthSnapshot
		var scanErr error
		for rows.Next() {
			snapshot, err := scanHealthSnapshot(rows)
			if err != nil {
				scanErr = err
				break
			}
			snapshots = append(snapshots, snapshot)
		}
		if scanErr == nil {
			scanErr = rows.Err()
		}
		closeErr := rows.Close()
		if scanErr != nil {
			return nil, fmt.Errorf("iterate health snapshots: %w", scanErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close health snapshot rows: %w", closeErr)
		}
		for _, snapshot := range snapshots {
			out[snapshot.InstanceID] = snapshot
		}
	}
	return out, nil
}

func dedupeIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func chunkIDs(ids []string, size int) [][]string {
	if size <= 0 || len(ids) == 0 {
		if len(ids) == 0 {
			return nil
		}
		return [][]string{ids}
	}
	out := make([][]string, 0, (len(ids)+size-1)/size)
	for start := 0; start < len(ids); start += size {
		end := start + size
		if end > len(ids) {
			end = len(ids)
		}
		out = append(out, ids[start:end])
	}
	return out
}

func idArgs(ids []string) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}
