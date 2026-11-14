package filekind

import "testing"

func TestIsTestSupportFile(t *testing.T) {
	for _, path := range []string{
		"ui/src/shared/test-utils/api-mocks.ts",
		"api/internal/testutil/helpers.go",
		"ui/src/fixtures/response.json",
	} {
		if !IsTestSupportFile(path) {
			t.Errorf("IsTestSupportFile(%q) = false, want true", path)
		}
	}
	if IsTestSupportFile("ui/src/shared/api/client.ts") {
		t.Fatal("production client must not be classified as test support")
	}
}
