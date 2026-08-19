package gates

// Differential gates consume the rendered evidence written by Experience
// Manager. Source text can suggest that an asset supports RTL or reduced
// motion, but it cannot prove that two browser contexts actually produced two
// different results. Missing context-paired evidence is therefore unmeasured.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/api-core/database"
)

type differentialObservation struct {
	AssetID     string
	ComponentID string
	ExampleName string
	StateID     string
	ViewportID  string
	ViewportW   float64
	ViewportH   float64
	Locale      string
	Direction   string
	Motion      string
	AX          axObservation
}

type axObservation struct {
	Bounds        *axBounds         `json:"bounds"`
	DOM           axDOM             `json:"dom"`
	Children      []axObservation   `json:"children"`
	ComputedStyle map[string]string `json:"computedStyle"`
}

type axBounds struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type axDOM struct {
	Tag        string            `json:"tag"`
	Attributes map[string]string `json:"attributes"`
}

type captureContext struct {
	Locale           string `json:"locale"`
	Direction        string `json:"direction"`
	MotionPreference string `json:"motionPreference"`
}

type captureMeasurement struct {
	CaptureContext captureContext `json:"captureContext"`
}

type differentialRow struct {
	ID          string
	ComponentID string
	ExampleName string
	StateID     string
	ViewportID  string
	ViewportW   float64
	ViewportH   float64
	AXJSON      string
	Measurement string
	CheckedAt   string
}

func ValidateRTL(root string) (Result, error) {
	return validateDifferentialGate(root, "rtl")
}

func ValidateReducedMotion(root string) (Result, error) {
	return validateDifferentialGate(root, "reduced-motion")
}

func validateDifferentialGate(root, gate string) (Result, error) {
	assets, err := loadAssets(root)
	if err != nil {
		return Result{}, err
	}
	rows, err := loadDifferentialRows(root)
	if err != nil {
		return Result{}, err
	}
	byAsset := map[string][]differentialObservation{}
	for _, row := range rows {
		observation, ok := decodeDifferentialObservation(row, gate)
		if ok {
			byAsset[observation.AssetID] = append(byAsset[observation.AssetID], observation)
		}
	}
	result := Result{}
	for _, asset := range assets {
		kind := asset.Asset.Kind
		if kind != "component" && kind != "primitive" && kind != "pattern" && kind != "page-template" && kind != "navigation" {
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
		observations := byAsset[asset.Asset.ID]
		verdict, message := evaluateDifferential(gate, observations)
		if verdict == "unmeasured" {
			result.UnmeasuredAssets = append(result.UnmeasuredAssets, asset.Asset.ID)
			continue
		}
		if verdict != "pass" {
			result.Findings = append(result.Findings, Finding{
				Code:        "catalog." + gate,
				AssetID:     asset.Asset.ID,
				Message:     message,
				Remediation: differentialRemediation(gate),
				DocsRef:     "docs/concepts/ARCHITECTURE.md#internationalization",
			})
		}
	}
	if len(result.UnmeasuredAssets) == result.Inspected {
		result.Status = "unmeasured"
	}
	return nonEmpty(result, gate), nil
}

func evaluateDifferential(gate string, observations []differentialObservation) (string, string) {
	if len(observations) == 0 {
		return "unmeasured", ""
	}
	if gate == "rtl" {
		return evaluateRTL(observations)
	}
	return evaluateReducedMotion(observations)
}

func evaluateRTL(observations []differentialObservation) (string, string) {
	pairs := pairObservations(observations, func(item differentialObservation) string {
		return item.Locale
	})
	for _, pair := range pairs {
		ltr, rtl := pair["ltr"], pair["rtl"]
		if ltr.Direction == "" && ltr.Locale == "en" {
			ltr.Direction = "ltr"
		}
		if rtl.Direction == "" && rtl.Locale == "ar" {
			rtl.Direction = "rtl"
		}
		if ltr.AX.Bounds == nil || rtl.AX.Bounds == nil || ltr.ViewportW <= 0 {
			continue
		}
		mirrored := nearlyEqual(ltr.AX.Bounds.X+ltr.AX.Bounds.Width, ltr.ViewportW-rtl.AX.Bounds.X, 2)
		changed := !nearlyEqual(ltr.AX.Bounds.X, rtl.AX.Bounds.X, 2)
		if mirrored && changed {
			return "pass", ""
		}
		return "renders-not-differential", "RTL capture did not mirror the stamped asset's inline position between ltr and rtl"
	}
	return "unmeasured", ""
}

func evaluateReducedMotion(observations []differentialObservation) (string, string) {
	pairs := pairObservations(observations, func(item differentialObservation) string {
		return item.Motion
	})
	for _, pair := range pairs {
		full, reduced := pair["no-preference"], pair["reduce"]
		fullDuration, fullOK := motionDuration(full.AX)
		reducedDuration, reducedOK := motionDuration(reduced.AX)
		if !fullOK || !reducedOK {
			continue
		}
		if reducedDuration == 0 && (fullDuration == 0 || fullDuration > 0) {
			if fullDuration == 0 {
				return "pass", ""
			}
			return "pass", ""
		}
		return "renders-not-differential", fmt.Sprintf("reduced-motion capture kept transition/animation duration at %s instead of zero", formatDuration(reducedDuration))
	}
	return "unmeasured", ""
}

type differentialPair map[string]differentialObservation

func pairObservations(observations []differentialObservation, context func(differentialObservation) string) []differentialPair {
	groups := map[string]differentialPair{}
	for _, observation := range observations {
		value := context(observation)
		if value != "ltr" && value != "rtl" && value != "en" && value != "ar" && value != "no-preference" && value != "reduce" {
			continue
		}
		if value == "en" {
			value = "ltr"
		}
		if value == "ar" {
			value = "rtl"
		}
		key := strings.Join([]string{observation.ComponentID, observation.ExampleName, observation.StateID, observation.ViewportID}, "\x00")
		if groups[key] == nil {
			groups[key] = differentialPair{}
		}
		groups[key][value] = observation
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]differentialPair, 0, len(keys))
	for _, key := range keys {
		pair := groups[key]
		if pair["ltr"].AssetID != "" && pair["rtl"].AssetID != "" || pair["no-preference"].AssetID != "" && pair["reduce"].AssetID != "" {
			result = append(result, pair)
		}
	}
	return result
}

func motionDuration(node axObservation) (float64, bool) {
	for _, key := range []string{"transitionDuration", "transition-duration", "animationDuration", "animation-duration"} {
		if value := strings.TrimSpace(node.ComputedStyle[key]); value != "" {
			parts := strings.Split(value, ",")
			max := 0.0
			for _, part := range parts {
				seconds, ok := parseCSSDuration(strings.TrimSpace(part))
				if !ok {
					return 0, false
				}
				if seconds > max {
					max = seconds
				}
			}
			return max, true
		}
	}
	return 0, false
}

func parseCSSDuration(value string) (float64, bool) {
	if value == "0" || value == "0s" || value == "0ms" {
		return 0, true
	}
	if strings.HasSuffix(value, "ms") {
		n, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "ms")), 64)
		return n / 1000, err == nil
	}
	if strings.HasSuffix(value, "s") {
		n, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "s")), 64)
		return n, err == nil
	}
	return 0, false
}

func nearlyEqual(a, b, tolerance float64) bool { return a >= b-tolerance && a <= b+tolerance }

func formatDuration(value float64) string { return strconv.FormatFloat(value, 'f', 3, 64) + "s" }

func differentialRemediation(gate string) string {
	if gate == "rtl" {
		return "Use logical inline properties or direction-aware geometry so the rendered asset mirrors between the captured ltr and rtl contexts. The gate reads the stamped DOM bounds, not a source annotation."
	}
	return "Honor prefers-reduced-motion in the rendered asset. The reduced context must collapse transition and animation durations while the full-motion context remains independently captured."
}

func loadDifferentialRows(root string) ([]differentialRow, error) {
	dbPath := filepath.Join(root, "scenarios", "experience-manager", "data", "experience-manager.db")
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)", dbPath), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(context.Background(), `SELECT r.id,r.component_id,r.example_name,r.state_id,COALESCE(v.viewport_id,''),COALESCE(v.viewport_width,0),COALESCE(v.viewport_height,0),r.ax_node_json,r.measurement_json,r.checked_at FROM reconcile_evidence r LEFT JOIN reconcile_evidence_viewports v ON v.evidence_id=r.id WHERE r.scenario='react-component-library' AND r.document_kind='component' AND r.ax_node_json LIKE '%data-rcl-asset%' ORDER BY r.checked_at DESC, r.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []differentialRow
	seen := map[string]bool{}
	for rows.Next() {
		var row differentialRow
		if err := rows.Scan(&row.ID, &row.ComponentID, &row.ExampleName, &row.StateID, &row.ViewportID, &row.ViewportW, &row.ViewportH, &row.AXJSON, &row.Measurement, &row.CheckedAt); err != nil {
			return nil, err
		}
		// Evidence is append-only. Keep the newest row for one asset/context so
		// a stale baseline cannot pair with a current capture.
		var node axObservation
		if json.Unmarshal([]byte(row.AXJSON), &node) != nil {
			continue
		}
		assetID := stampedAsset(node)
		if assetID == "" {
			continue
		}
		key := strings.Join([]string{assetID, row.ComponentID, row.ExampleName, row.StateID, row.ViewportID, row.Measurement}, "\x00")
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, row)
	}
	return result, rows.Err()
}

func decodeDifferentialObservation(row differentialRow, gate string) (differentialObservation, bool) {
	var node axObservation
	if json.Unmarshal([]byte(row.AXJSON), &node) != nil {
		return differentialObservation{}, false
	}
	assetID := stampedAsset(node)
	if assetID == "" {
		return differentialObservation{}, false
	}
	var measurement captureMeasurement
	if json.Unmarshal([]byte(row.Measurement), &measurement) != nil {
		return differentialObservation{}, false
	}
	ctx := measurement.CaptureContext
	if ctx.Direction == "" && ctx.Locale == "en" {
		ctx.Direction = "ltr"
	}
	if ctx.Direction == "" && ctx.Locale == "ar" {
		ctx.Direction = "rtl"
	}
	if gate == "rtl" && ctx.Direction == "" {
		return differentialObservation{}, false
	}
	if gate == "reduced-motion" && ctx.MotionPreference == "" {
		return differentialObservation{}, false
	}
	return differentialObservation{AssetID: assetID, ComponentID: row.ComponentID, ExampleName: row.ExampleName, StateID: row.StateID, ViewportID: row.ViewportID, ViewportW: row.ViewportW, ViewportH: row.ViewportH, Locale: ctx.Locale, Direction: ctx.Direction, Motion: ctx.MotionPreference, AX: node}, true
}

func stampedAsset(node axObservation) string {
	if value := strings.TrimSpace(node.DOM.Attributes["data-rcl-asset"]); value != "" {
		return value
	}
	for _, child := range node.Children {
		if value := stampedAsset(child); value != "" {
			return value
		}
	}
	return ""
}
