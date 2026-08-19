package gates

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/api-core/database"
)

// compositionCapture is the newest persisted rendered tree for one catalog
// asset. The tree is the oracle: source imports and JSX declarations cannot
// prove that an asset actually rendered the primitives it claims to compose.
type compositionCapture struct {
	AssetID   string
	Tree      axObservation
	CheckedAt string
}

// compositionPassThreshold was derived from the measured Phase 13 corpus:
// the seven cockpit assets had a median of 1.0 after shared primitive markers
// and reasoned native escapes were captured. A floor of 0.8 leaves room for a
// small composed exception while preventing raw-heavy assets from reaching
// production-ready.
const compositionPassThreshold = 0.8

// ValidateComposition scores the stamped DOM authored by each built asset.
// Nested stamped assets are excluded from the parent denominator so a parent
// is measured on its own nodes. A data-bespoke node is an explicit, reasoned
// exception rather than a raw-node failure.
func ValidateComposition(root string) (Result, error) {
	assets, err := loadAssets(root)
	if err != nil {
		return Result{}, err
	}
	captures, err := loadCompositionCaptures(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{CompositionScores: map[string]float64{}}
	values := make([]float64, 0)
	for _, asset := range assets {
		if !compositionAssetKind(asset.Asset.Kind) {
			continue
		}
		_, _, implemented, err := implementationSource(root, asset.Asset.ID)
		if err != nil {
			return Result{}, err
		}
		if !implemented {
			continue
		}
		result.Inspected++
		result.InspectedAssets = appendUnique(result.InspectedAssets, asset.Asset.ID)
		capture, ok := captures[asset.Asset.ID]
		if !ok {
			result.UnmeasuredAssets = append(result.UnmeasuredAssets, asset.Asset.ID)
			continue
		}
		rootNode, ok := stampedNode(&capture.Tree, asset.Asset.ID)
		if !ok {
			result.UnmeasuredAssets = append(result.UnmeasuredAssets, asset.Asset.ID)
			continue
		}
		score, total, raw, escapes, missingReasons := scoreComposition(rootNode, asset.Asset.ID)
		if total == 0 {
			result.UnmeasuredAssets = append(result.UnmeasuredAssets, asset.Asset.ID)
			continue
		}
		result.CompositionScores[asset.Asset.ID] = score
		values = append(values, score)
		for _, escape := range escapes {
			result.BespokeEscapes = append(result.BespokeEscapes, escape)
		}
		for range missingReasons {
			result.Findings = append(result.Findings, Finding{
				Code:        "catalog.composition_bespoke_reason",
				AssetID:     asset.Asset.ID,
				Message:     "data-bespoke node has no non-empty reason",
				Remediation: "Give every data-bespoke node a concise reason in the attribute value. Bespoke escapes are allowed only when the rendered tree records why the shared composition primitives do not apply.",
				DocsRef:     "docs/internal/TESTING.md",
			})
		}
		if score < compositionPassThreshold {
			result.Findings = append(result.Findings, Finding{
				Code:        "catalog.composition_low_score",
				AssetID:     asset.Asset.ID,
				Message:     fmt.Sprintf("rendered composition score %.3f below production threshold %.3f (%d raw of %d own DOM nodes)", score, compositionPassThreshold, raw, total),
				Remediation: "Compose the asset from shared Stack, Surface, or other library primitives, or mark a genuinely bespoke node with data-bespoke and a reason. The score is derived from the stamped rendered tree, not imports.",
				DocsRef:     "docs/internal/TESTING.md",
			})
		}
	}
	if len(values) > 0 {
		sort.Float64s(values)
		middle := len(values) / 2
		if len(values)%2 == 0 {
			result.CompositionMedian = (values[middle-1] + values[middle]) / 2
		} else {
			result.CompositionMedian = values[middle]
		}
	}
	if len(result.UnmeasuredAssets) == result.Inspected {
		result.Status = "unmeasured"
	}
	return nonEmpty(result, "composition"), nil
}

func compositionAssetKind(kind string) bool {
	switch kind {
	case "component", "primitive", "pattern", "page-template", "navigation":
		return true
	default:
		return false
	}
}

func loadCompositionCaptures(root string) (map[string]compositionCapture, error) {
	path := filepath.Join(root, "scenarios", "experience-manager", "data", "experience-manager.db")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return map[string]compositionCapture{}, nil
		}
		return nil, err
	}
	db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: "file:" + path + "?_pragma=busy_timeout(10000)", MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return queryCompositionCaptures(db)
}

type compositionQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func queryCompositionCaptures(db compositionQueryer) (map[string]compositionCapture, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT ax_node_json, checked_at FROM reconcile_evidence WHERE scenario = 'react-component-library' AND document_kind = 'component' AND ax_node_json LIKE '%data-rcl-asset%' ORDER BY checked_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]compositionCapture{}
	for rows.Next() {
		var raw, checkedAt string
		if err := rows.Scan(&raw, &checkedAt); err != nil {
			return nil, err
		}
		var tree axObservation
		if json.Unmarshal([]byte(raw), &tree) != nil {
			continue
		}
		assetID := stampedAsset(tree)
		if assetID == "" {
			continue
		}
		if _, exists := result[assetID]; exists {
			continue
		}
		result[assetID] = compositionCapture{AssetID: assetID, Tree: tree, CheckedAt: checkedAt}
	}
	return result, rows.Err()
}

func stampedNode(node *axObservation, assetID string) (*axObservation, bool) {
	if node == nil {
		return nil, false
	}
	if strings.TrimSpace(node.DOM.Attributes["data-rcl-asset"]) == assetID {
		return node, true
	}
	for index := range node.Children {
		if found, ok := stampedNode(&node.Children[index], assetID); ok {
			return found, true
		}
	}
	return nil, false
}

func scoreComposition(root *axObservation, assetID string) (score float64, total, raw int, escapes []CompositionEscape, missingReasons []string) {
	var walk func(*axObservation)
	walk = func(node *axObservation) {
		if node == nil {
			return
		}
		stamp := strings.TrimSpace(node.DOM.Attributes["data-rcl-asset"])
		if stamp != "" && stamp != assetID {
			return
		}
		if node.DOM.Tag != "" {
			total++
			if reason, bespoke := node.DOM.Attributes["data-bespoke"]; bespoke {
				reason = strings.TrimSpace(reason)
				escapes = append(escapes, CompositionEscape{AssetID: assetID, Reason: reason})
				if reason == "" {
					missingReasons = append(missingReasons, reason)
				}
			} else if stamp == "" && !sharedPrimitiveNode(node) {
				raw++
			}
		}
		for index := range node.Children {
			walk(&node.Children[index])
		}
	}
	walk(root)
	if total > 0 {
		score = 1 - float64(raw)/float64(total)
		if score < 0 {
			score = 0
		}
		if score > 1 {
			score = 1
		}
	}
	return score, total, raw, escapes, missingReasons
}

// sharedPrimitiveNode recognizes the explicit DOM markers emitted by the
// library's shared primitives. The rendered oracle intentionally relies on
// markers rather than tag names: a native span or div is not composed merely
// because it happens to use a familiar HTML element.
func sharedPrimitiveNode(node *axObservation) bool {
	if node == nil {
		return false
	}
	attributes := node.DOM.Attributes
	for _, key := range []string{
		"data-elevation",
		"data-rcl-control",
		"data-rcl-progress",
		"data-text-style",
		"data-tone",
	} {
		if strings.TrimSpace(attributes[key]) != "" {
			return true
		}
	}
	return false
}

func compositionMetadata(result Result, assetID string) string {
	score, ok := result.CompositionScores[assetID]
	if !ok {
		return ""
	}
	escapes := make([]string, 0)
	for _, escape := range result.BespokeEscapes {
		if escape.AssetID == assetID {
			escapes = append(escapes, escape.Reason)
		}
	}
	payload, _ := json.Marshal(struct {
		Score         float64  `json:"score"`
		BespokeReason []string `json:"bespokeReasons,omitempty"`
	}{Score: score, BespokeReason: escapes})
	return string(payload)
}

// CompositionScoreMetadata is the small read-side adapter used by catalog
// health. It keeps the evidence payload private while allowing the coverage
// projection to publish the same score and escape census as the gate runner.
func CompositionScoreMetadata(raw string) (float64, bool, []string) {
	return compositionScoreFromMetadata(raw)
}

// CompositionScoreMetadataJSON serializes one asset's measured score for the
// catalog evidence row. Empty means the asset had no rendered measurement.
func CompositionScoreMetadataJSON(result Result, assetID string) string {
	return compositionMetadata(result, assetID)
}

func compositionScoreFromMetadata(raw string) (float64, bool, []string) {
	var payload struct {
		Score         float64  `json:"score"`
		BespokeReason []string `json:"bespokeReasons"`
	}
	if json.Unmarshal([]byte(raw), &payload) != nil {
		return 0, false, nil
	}
	return payload.Score, true, payload.BespokeReason
}
