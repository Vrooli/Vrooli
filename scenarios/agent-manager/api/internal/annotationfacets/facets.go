// Package annotationfacets projects gated annotation labels into the
// conversation-search catalog's existing tags. It deliberately does not add a
// new search filter: `--tags label:agent.spinning` already works, and the
// catalog is the one place conversation search reads.
//
// The projection is a convenience, not the source of truth. The annotation
// store remains authoritative; a missing tag only means the run was not
// re-tagged yet. Unknown is never written as a label.
package annotationfacets

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"agent-manager/internal/runannotations"
	"agent-manager/internal/sqlcompat"
)

// Refresher projects one or more runs' latest annotations into catalog tags.
type Refresher struct {
	DB      sqlcompat.DB
	Service *runannotations.Service
}

// Refresh re-tags every catalog document of each run. It is idempotent and
// bounded by the caller's run list.
func (r *Refresher) Refresh(ctx context.Context, runIDs []string) error {
	if r == nil || r.DB == nil || r.Service == nil {
		return nil
	}
	labels := map[string]runannotations.Label{}
	for _, label := range r.Service.Pack().Pack.Labels {
		labels[label.ID] = label
	}
	for _, runID := range runIDs {
		view, err := r.Service.Run(ctx, runID)
		if err != nil {
			// A run with no catalog rows is not an error for this projection.
			if err == runannotations.ErrScopeNotFound {
				continue
			}
			return fmt.Errorf("read annotations for run %s: %w", runID, err)
		}
		if err := r.refreshView(ctx, view, labels); err != nil {
			return fmt.Errorf("refresh annotation facets for run %s: %w", runID, err)
		}
	}
	return nil
}

// RefreshViews re-tags views the caller already holds, avoiding a second
// annotation read after Evaluate.
func (r *Refresher) RefreshViews(ctx context.Context, views []runannotations.RunView) error {
	if r == nil || r.DB == nil || r.Service == nil {
		return nil
	}
	labels := map[string]runannotations.Label{}
	for _, label := range r.Service.Pack().Pack.Labels {
		labels[label.ID] = label
	}
	for _, view := range views {
		if err := r.refreshView(ctx, view, labels); err != nil {
			return fmt.Errorf("refresh annotation facets for run %s: %w", view.RunID, err)
		}
	}
	return nil
}

func (r *Refresher) refreshView(ctx context.Context, view runannotations.RunView, labels map[string]runannotations.Label) error {
	runID := view.RunID
	rows, err := r.DB.QueryxContext(ctx, `SELECT DISTINCT tags_json FROM conversation_search_catalog WHERE source_run_id = ?`, runID)
	if err != nil {
		return err
	}
	defer rows.Close()
	tags := map[string]bool{}
	seen := false
	for rows.Next() {
		seen = true
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		for _, tag := range decodeTags(raw) {
			// Drop stale facet tags; keep every non-facet tag the importer set.
			// The union across rows avoids dropping a per-chunk tag if the run's
			// chunks ever disagree.
			if strings.HasPrefix(tag, "label:") {
				continue
			}
			tags[tag] = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !seen {
		return nil
	}
	for _, annotation := range view.Labels {
		label, ok := labels[annotation.ID]
		if !ok || strings.TrimSpace(label.Search.Facet) == "" {
			continue
		}
		if !crossesSearchThreshold(label, annotation) {
			continue
		}
		tags[label.Search.Facet] = true
		if annotation.Value.Choice != "" {
			tags[label.Search.Facet+":"+annotation.Value.Choice] = true
		}
		if annotation.Value.Score != nil {
			tags[fmt.Sprintf("%s:score=%s", label.Search.Facet, trimFloat(*annotation.Value.Score))] = true
		}
	}
	encoded, err := json.Marshal(sortedKeys(tags))
	if err != nil {
		return err
	}
	_, err = r.DB.ExecContext(ctx, `UPDATE conversation_search_catalog SET tags_json = ? WHERE source_run_id = ?`, string(encoded), runID)
	return err
}

// crossesSearchThreshold reports whether an annotation is strong enough to
// become a searchable facet. A missing probability never crosses a positive
// threshold: an evaluator that cannot express confidence (a deterministic rule
// or the LLM enum fallback) must not be treated as certain.
func crossesSearchThreshold(label runannotations.Label, annotation runannotations.AnnotationView) bool {
	if annotation.State != runannotations.StateObserved && annotation.State != runannotations.StateCompatible {
		return false
	}
	threshold := 0.0
	if label.Search.MinProbability != nil {
		threshold = *label.Search.MinProbability
	}
	if threshold <= 0 {
		return true
	}
	return annotation.Probability != nil && *annotation.Probability >= threshold
}

func decodeTags(raw string) []string {
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return nil
	}
	return tags
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func trimFloat(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
}
