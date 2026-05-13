package tts

import "testing"

const missingBodyFile = "a JSON body file path is required (use --body-file)"

func TestValidation(t *testing.T) {
	t.Run("cache_get_requires_event_id", func(t *testing.T) {
		err := runCacheGet(nil, []string{})
		if err == nil || err.Error() != "--event-id is required" {
			t.Fatalf("expected missing event-id error, got %v", err)
		}
	})

	t.Run("config_set_requires_body_file", func(t *testing.T) {
		err := runConfigSet(nil, []string{})
		if err == nil || err.Error() != missingBodyFile {
			t.Fatalf("expected missing body-file error, got %v", err)
		}
	})

	t.Run("summarize_config_set_requires_body_file", func(t *testing.T) {
		err := runSummarizeSet(nil, []string{})
		if err == nil || err.Error() != missingBodyFile {
			t.Fatalf("expected missing body-file error, got %v", err)
		}
	})

	t.Run("synthesize_requires_body_file", func(t *testing.T) {
		err := runSynthesize(nil, []string{})
		if err == nil || err.Error() != missingBodyFile {
			t.Fatalf("expected missing body-file error, got %v", err)
		}
	})

	t.Run("post_event_requires_body_file", func(t *testing.T) {
		err := runPostEvent(nil, []string{})
		if err == nil || err.Error() != missingBodyFile {
			t.Fatalf("expected missing body-file error, got %v", err)
		}
	})
}
