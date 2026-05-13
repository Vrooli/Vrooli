package shortcuts

import "testing"

const missingBodyFile = "a JSON body file path is required (use --body-file)"

func TestValidation(t *testing.T) {
	t.Run("delete_requires_id", func(t *testing.T) {
		err := runDelete(nil, []string{})
		if err == nil || err.Error() != "usage: shortcuts delete <profile-id>" {
			t.Fatalf("expected usage error, got %v", err)
		}
	})

	t.Run("upsert_requires_body_file", func(t *testing.T) {
		err := runUpsert(nil, []string{})
		if err == nil || err.Error() != missingBodyFile {
			t.Fatalf("expected missing body-file error, got %v", err)
		}
	})
}
