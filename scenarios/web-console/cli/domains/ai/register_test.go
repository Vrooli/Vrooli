package ai

import "testing"

// AI commands that mutate config or send prompts all funnel through
// support.ReadJSONFile with required=true. The error string comes from
// that helper; if it changes, update both call sites.
const missingBodyFile = "a JSON body file path is required (use --body-file)"

func TestValidation(t *testing.T) {
	t.Run("generate_requires_body_file", func(t *testing.T) {
		err := runGenerate(nil, []string{})
		if err == nil || err.Error() != missingBodyFile {
			t.Fatalf("expected missing body-file error, got %v", err)
		}
	})

	t.Run("suggest_requires_body_file", func(t *testing.T) {
		err := runSuggest(nil, []string{})
		if err == nil || err.Error() != missingBodyFile {
			t.Fatalf("expected missing body-file error, got %v", err)
		}
	})

	t.Run("config_set_requires_body_file", func(t *testing.T) {
		err := runConfigSet(nil, []string{})
		if err == nil || err.Error() != missingBodyFile {
			t.Fatalf("expected missing body-file error, got %v", err)
		}
	})
}
