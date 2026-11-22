package main

import "testing"

func TestHandleVisualCaptureScreenshot_PathTraversal(t *testing.T) {
	t.Parallel()

	err := validateFilename("../../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal")
	}

	err = validateFilename("normal_file.png")
	if err != nil {
		t.Errorf("unexpected error for valid filename: %v", err)
	}

	err = validateFilename("file\\with\\backslash")
	if err == nil {
		t.Error("expected error for backslash in filename")
	}
}
