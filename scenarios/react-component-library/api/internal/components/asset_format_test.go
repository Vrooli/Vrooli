package components

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanonicalJSONUsesStableIndentAndNewline(t *testing.T) {
	formatted, err := canonicalJSONText(`{"z":1,"nested":{"ok":true}}`)
	require.NoError(t, err)
	require.Equal(t, "{\n  \"nested\": {\n    \"ok\": true\n  },\n  \"z\": 1\n}\n", formatted)
}

func TestValidateExperienceContractMatchesStoryStates(t *testing.T) {
	story := StoryContract{Stories: []StoryDefinition{{ID: "idle"}, {ID: "error"}}}
	valid := []byte(`{"contract":{"kind":"rcl-component-experience-contract"},"states":[{"id":"idle","example":"idle"},{"id":"error","example":"error"}],"claims":[{"id":"visible","states":["idle"]}]}`)
	require.Empty(t, validateExperienceContract(valid, story))

	invalid := []byte(`{"kind":"experience-component","states":[{"id":"missing","example":"missing"}]}`)
	require.Equal(t, []string{`experience contract state "missing" has no matching story`}, validateExperienceContract(invalid, story))
}

func TestValidateExperienceContractAcceptsHistoricalSchemaVersion(t *testing.T) {
	legacy := []byte(`{"schemaVersion":1,"componentId":"react-component-library:Button","states":["default"]}`)
	require.Equal(t, []string{"experience contract kind must be experience-component or rcl-component-experience-contract"}, validateExperienceContract(legacy, StoryContract{}))
}
