package workflow

import "strings"

// SubstitutePlaceholders replaces placeholder values in a workflow definition.
// Supported placeholders:
//   - ${BASE_URL}: The scenario's UI base URL
//   - {{UI_PORT}}: The UI port number
func SubstitutePlaceholders(doc any, baseURL string) {
	substitute(doc, baseURL)
}

// substitute recursively processes the document.
func substitute(doc any, baseURL string) {
	switch value := doc.(type) {
	case map[string]any:
		for k, v := range value {
			value[k] = substituteValue(v, baseURL)
		}
	case []any:
		for idx, v := range value {
			value[idx] = substituteValue(v, baseURL)
		}
	}
}

// substituteValue processes a single value.
func substituteValue(v any, baseURL string) any {
	switch typed := v.(type) {
	case string:
		processed := strings.ReplaceAll(typed, "${BASE_URL}", baseURL)
		if port := extractPort(baseURL); port != "" {
			processed = strings.ReplaceAll(processed, "{{UI_PORT}}", port)
		}
		return processed
	case map[string]any, []any:
		substitute(typed, baseURL)
		return typed
	default:
		return v
	}
}

// extractPort extracts the port from a URL string.
func extractPort(baseURL string) string {
	if baseURL == "" {
		return ""
	}
	// Find the last colon, but make sure it's not part of the scheme (http://)
	idx := strings.LastIndex(baseURL, ":")
	if idx == -1 {
		return ""
	}
	// Check if this is a scheme colon (followed by //)
	if idx+2 < len(baseURL) && baseURL[idx+1:idx+3] == "//" {
		return ""
	}
	// Verify the part after colon looks like a port (digits only, no slashes)
	port := baseURL[idx+1:]
	// Strip any trailing path
	if slashIdx := strings.Index(port, "/"); slashIdx != -1 {
		port = port[:slashIdx]
	}
	// Verify it contains only digits
	for _, c := range port {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return port
}

// CleanDefinition prepares a workflow definition for BAS API consumption.
// It extracts the core flow_definition if wrapped, and retains only
// nodes, edges, and settings.
func CleanDefinition(doc map[string]any) map[string]any {
	root := doc
	if inner, ok := doc["flow_definition"].(map[string]any); ok {
		root = inner
	}

	result := map[string]any{}
	if nodes, ok := root["nodes"]; ok {
		result["nodes"] = cleanNodes(nodes)
	}
	if edges, ok := root["edges"]; ok {
		result["edges"] = edges
	}
	if settings, ok := root["settings"]; ok {
		result["settings"] = settings
	}
	return result
}

// cleanNodes recursively cleans nested workflow definitions in nodes.
func cleanNodes(value any) any {
	nodes, ok := value.([]any)
	if !ok {
		return value
	}

	for _, raw := range nodes {
		node, _ := raw.(map[string]any)
		if node == nil {
			continue
		}
		data, _ := node["data"].(map[string]any)
		if data == nil {
			continue
		}
		if wf, ok := data["workflowDefinition"].(map[string]any); ok {
			data["workflowDefinition"] = CleanDefinition(wf)
		}
	}
	return nodes
}
