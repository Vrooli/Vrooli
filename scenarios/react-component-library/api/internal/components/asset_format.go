package components

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// canonicalJSON is the registry's wire format for JSON assets: two-space
// indentation and one trailing newline. Keeping this in the API makes ingest
// and version promotion deterministic even when their caller is not the UI.
func canonicalJSON(raw []byte) ([]byte, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return nil, fmt.Errorf("multiple JSON values")
	} else if err != io.EOF {
		return nil, fmt.Errorf("trailing JSON: %w", err)
	}
	formatted, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(formatted, '\n'), nil
}

func canonicalJSONText(raw string) (string, error) {
	formatted, err := canonicalJSON([]byte(raw))
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}

// validateExperienceContract checks the common contract vocabulary shared by
// the RCL wrapper and the experience-component document.
func validateExperienceContract(raw []byte, story StoryContract) []string {
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return []string{"experience-contract.json is not valid JSON: " + err.Error()}
	}
	kind := stringValue(document["kind"])
	if nested, ok := document["contract"].(map[string]any); ok {
		kind = stringValue(nested["kind"])
	}
	if kind != "experience-component" && kind != "rcl-component-experience-contract" {
		return []string{"experience contract kind must be experience-component or rcl-component-experience-contract"}
	}

	stateIDs := map[string]bool{}
	if states, ok := document["states"].([]any); ok {
		for _, item := range states {
			switch state := item.(type) {
			case string:
				stateIDs[state] = true
			case map[string]any:
				id := stringValue(state["id"])
				if id == "" {
					return append([]string{}, "experience contract state is missing id")
				}
				stateIDs[id] = true
			}
		}
	}
	if len(stateIDs) > 0 && len(story.Stories) > 0 {
		storyIDs := map[string]bool{}
		for _, item := range story.Stories {
			storyIDs[item.ID] = true
		}
		for id := range stateIDs {
			if !storyIDs[id] {
				return []string{fmt.Sprintf("experience contract state %q has no matching story", id)}
			}
		}
	}
	if claims, ok := document["claims"].([]any); ok {
		seen := map[string]bool{}
		for _, item := range claims {
			claim, ok := item.(map[string]any)
			if !ok {
				return []string{"experience contract claim must be an object"}
			}
			id := stringValue(claim["id"])
			if id == "" || seen[id] {
				return []string{"experience contract claims require unique ids"}
			}
			seen[id] = true
			if refs, ok := claim["states"].([]any); ok {
				for _, ref := range refs {
					if stateID := stringValue(ref); stateID != "" && len(stateIDs) > 0 && !stateIDs[stateID] {
						return []string{fmt.Sprintf("experience contract claim %q references unknown state %q", id, stateID)}
					}
				}
			}
		}
	}
	return nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}
