package reconcile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const defaultCaptureBudget = 12

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
		return CaptureProfile{
			ID:               viewport,
			MatrixID:         viewport + "-" + color + "-" + locale + "-" + motion + "-" + interaction,
			Width:            width,
			Height:           height,
			ColorScheme:      color,
			Locale:           locale,
			MotionPreference: motion,
			InteractionState: interaction,
		}
	}

	// These rows are the vrooli-default baseline matrix. The first row for
	// every viewport establishes responsive coverage, then orthogonal values
	// are paired with representative viewports. In particular, desktop-dark
	// is an explicit row rather than an alias of a mobile profile.
	rows := []CaptureProfile{}
	for _, viewport := range values["viewport"] {
		rows = append(rows, profile(viewport, values["color-scheme"][0], values["locale"][0], values["motion-preference"][0], values["interaction-state"][0]))
	}
	baselineViewport := axisValue(values["viewport"], "desktop")
	if baselineViewport == "" {
		baselineViewport = values["viewport"][0]
	}
	rows = append(rows,
		profile(baselineViewport, values["color-scheme"][1], values["locale"][0], values["motion-preference"][0], values["interaction-state"][0]),
		profile(baselineViewport, values["color-scheme"][0], values["locale"][1], values["motion-preference"][0], values["interaction-state"][0]),
		profile(baselineViewport, values["color-scheme"][0], values["locale"][0], values["motion-preference"][1], values["interaction-state"][0]),
	)
	for index, interaction := range values["interaction-state"][1:] {
		rows = append(rows, profile(baselineViewport, values["color-scheme"][0], values["locale"][0], values["motion-preference"][0], interaction))
		if index+len(values["viewport"])+4 >= budget {
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
