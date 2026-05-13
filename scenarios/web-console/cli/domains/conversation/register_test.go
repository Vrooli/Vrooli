package conversation

import "testing"

func TestValidation(t *testing.T) {
	t.Run("get_requires_session", func(t *testing.T) {
		err := runGet(nil, []string{})
		if err == nil || err.Error() != "--session is required" {
			t.Fatalf("expected missing session error, got %v", err)
		}
	})

	t.Run("cursor_set_requires_session", func(t *testing.T) {
		err := runCursorSet(nil, []string{})
		if err == nil || err.Error() != "--session is required" {
			t.Fatalf("expected missing session error, got %v", err)
		}
	})

	t.Run("summarize_requires_session_and_event", func(t *testing.T) {
		err := runSummarize(nil, []string{})
		if err == nil || err.Error() != "--session and --event are required" {
			t.Fatalf("expected missing session/event error, got %v", err)
		}
	})

	t.Run("file_resolve_requires_session_and_path", func(t *testing.T) {
		err := runFileResolve(nil, []string{})
		if err == nil || err.Error() != "--session and --path are required" {
			t.Fatalf("expected missing session/path error, got %v", err)
		}
	})

	t.Run("file_content_requires_session_and_path", func(t *testing.T) {
		err := runFileContent(nil, []string{})
		if err == nil || err.Error() != "--session and --path are required" {
			t.Fatalf("expected missing session/path error, got %v", err)
		}
	})
}
