package facts

import "testing"

func TestFileMetricTestClassificationAndCompanions(t *testing.T) {
	paths := []string{
		"api/math.go", "api/math_test.go", "api/record.go", "api/record_details.go",
		"api/record_details_test.go", "ui/math.ts", "ui/math.test.ts", "ui/__tests__/other.spec.ts",
		"worker.py", "test_worker.py", "src/lib.rs", "src/tests/lib_test.rs",
	}
	tests, sources := splitTestSourcePaths(paths)
	for _, path := range []string{"api/math_test.go", "api/record_details_test.go", "ui/math.test.ts", "ui/__tests__/other.spec.ts", "test_worker.py", "src/tests/lib_test.rs"} {
		if !tests[path] {
			t.Errorf("%q was not classified as a test file", path)
		}
	}
	for _, path := range []string{"api/math.go", "api/record.go", "api/record_details.go", "ui/math.ts", "worker.py", "src/lib.rs"} {
		if tests[path] || !sources[path] {
			t.Errorf("%q classification: test=%t source=%t", path, tests[path], sources[path])
		}
	}
	for path, want := range map[string]bool{
		"api/math.go": true, "api/record.go": false, "api/record_details.go": true,
		"ui/math.ts": true, "worker.py": true, "src/lib.rs": true,
	} {
		if got := hasTestCompanion(path, tests, sources); got != want {
			t.Errorf("hasTestCompanion(%q) = %t, want %t", path, got, want)
		}
	}
}
