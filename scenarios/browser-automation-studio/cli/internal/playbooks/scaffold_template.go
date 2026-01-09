package playbooks

import (
	"encoding/json"
)

type WorkflowTemplate struct {
	Metadata map[string]any   `json:"metadata"`
	Settings map[string]any   `json:"settings"`
	Nodes    []map[string]any `json:"nodes"`
	Edges    []map[string]any `json:"edges"`
}

func BuildScaffoldTemplate(name, description, reset string) ([]byte, error) {
	template := WorkflowTemplate{
		Metadata: map[string]any{
			"name":        name,
			"description": description,
			"version":     1,
			"reset":       reset,
		},
		Settings: map[string]any{
			"executionViewport": map[string]any{
				"width":  1440,
				"height": 900,
				"preset": "desktop",
			},
		},
		Nodes: []map[string]any{
			{
				"id":   "navigate-app",
				"type": "navigate",
				"data": map[string]any{
					"label":           "Open BAS homepage",
					"destinationType": "scenario",
					"scenario":        "browser-automation-studio",
					"scenarioPath":    "/",
					"waitUntil":       "networkidle",
					"timeoutMs":       45000,
					"waitForMs":       2000,
				},
			},
			{
				"id":   "wait-for-ui",
				"type": "wait",
				"data": map[string]any{
					"label":      "Wait for UI to settle",
					"waitType":   "duration",
					"durationMs": 1000,
				},
			},
		},
		Edges: []map[string]any{
			{
				"id":     "edge-initial",
				"source": "navigate-app",
				"target": "wait-for-ui",
				"type":   "smoothstep",
			},
		},
	}

	return json.MarshalIndent(template, "", "  ")
}
