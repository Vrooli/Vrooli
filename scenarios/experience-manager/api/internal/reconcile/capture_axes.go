package reconcile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// The baseline covers every declared value for the five axes currently
// transmitted by BAS: four viewports, two color schemes, four locales, two
// motion preferences, and five interaction states. These are deliberately
// paired into a bounded covering set rather than a Cartesian product.
const defaultCaptureBudget = 16

type axisRegistryDocument struct {
	Axes []struct {
		ID     string `json:"id"`
		Values []struct {
			ID     string         `json:"id"`
			Params map[string]any `json:"params"`
		} `json:"values"`
	} `json:"axes"`
}

// CaptureProfilesFromAxes builds the bounded baseline matrix from the axis
// registry. The matrix is intentionally a covering set rather than an
// unbounded Cartesian product: every declared value is exercised, while the
// design kit's baseline budget keeps a reconciliation run predictable.
func CaptureProfilesFromAxes(path string, budget int) ([]CaptureProfile, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read capture axes: %w", err)
	}
	var document axisRegistryDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse capture axes: %w", err)
	}
	values := map[string][]string{}
	params := map[string]map[string]map[string]any{}
	for _, axis := range document.Axes {
		for _, value := range axis.Values {
			values[axis.ID] = append(values[axis.ID], value.ID)
			if params[axis.ID] == nil {
				params[axis.ID] = map[string]map[string]any{}
			}
			params[axis.ID][value.ID] = value.Params
		}
	}
	for _, axis := range []string{"viewport", "color-scheme", "locale", "motion-preference", "interaction-state"} {
		if len(values[axis]) == 0 {
			return nil, fmt.Errorf("capture axes missing values for %q", axis)
		}
	}
	if budget <= 0 {
		budget = defaultCaptureBudget
	}

	profile := func(viewport, color, locale, motion, interaction string) CaptureProfile {
		width, height := axisDimensions(params["viewport"][viewport])
		direction, _ := params["locale"][locale]["direction"].(string)
		if direction == "" {
			direction = "ltr"
		}
		return CaptureProfile{
			ID:               viewport,
			MatrixID:         viewport + "-" + color + "-" + locale + "-" + motion + "-" + interaction,
			Width:            width,
			Height:           height,
			ColorScheme:      color,
			Locale:           locale,
			Direction:        direction,
			MotionPreference: motion,
			InteractionState: interaction,
		}
	}

	// These rows are the baseline matrix. The first row for every viewport
	// establishes responsive coverage, then every value on each orthogonal
	// axis is paired with the representative desktop viewport. In particular,
	// desktop-dark and every declared locale are explicit rows rather than
	// aliases of a mobile or English profile.
	rows := []CaptureProfile{}
	for _, viewport := range values["viewport"] {
		rows = append(rows, profile(viewport, values["color-scheme"][0], values["locale"][0], values["motion-preference"][0], values["interaction-state"][0]))
	}
	baselineViewport := axisValue(values["viewport"], "desktop")
	if baselineViewport == "" {
		baselineViewport = values["viewport"][0]
	}
	for _, color := range values["color-scheme"][1:] {
		rows = append(rows, profile(baselineViewport, color, values["locale"][0], values["motion-preference"][0], values["interaction-state"][0]))
	}
	for _, locale := range values["locale"][1:] {
		rows = append(rows, profile(baselineViewport, values["color-scheme"][0], locale, values["motion-preference"][0], values["interaction-state"][0]))
	}
	for _, motion := range values["motion-preference"][1:] {
		rows = append(rows, profile(baselineViewport, values["color-scheme"][0], values["locale"][0], motion, values["interaction-state"][0]))
	}
	for index, interaction := range values["interaction-state"][1:] {
		rows = append(rows, profile(baselineViewport, values["color-scheme"][0], values["locale"][0], values["motion-preference"][0], interaction))
		if index+len(values["viewport"])+len(values["color-scheme"])+len(values["locale"])+len(values["motion-preference"])+1 >= budget {
			break
		}
	}
	if len(rows) > budget {
		rows = rows[:budget]
	}
	return rows, nil
}

func axisValue(values []string, wanted string) string {
	for _, value := range values {
		if value == wanted {
			return value
		}
	}
	return ""
}

func axisDimensions(params map[string]any) (int, int) {
	width, _ := params["width"].(float64)
	height, _ := params["height"].(float64)
	return int(width), int(height)
}
