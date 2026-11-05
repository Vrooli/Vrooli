package ocr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecognizeTextCarriesPositionAndConfidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "page.txt")
	if err := os.WriteFile(path, []byte("INVOICE 42\nTotal 10.00\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Recognize(path, "eng")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Runs) != 2 {
		t.Fatalf("runs = %d, want 2", len(result.Runs))
	}
	for _, run := range result.Runs {
		if run.Confidence <= 0 || run.Position.Width <= 0 || run.Position.Height <= 0 {
			t.Fatalf("incomplete OCR run: %+v", run)
		}
	}
}

func TestFixtureSetReturnsAuditableRuns(t *testing.T) {
	fixtures := []string{"clean-300dpi.txt", "low-resolution.txt", "rotated.txt", "multi-column.txt", "table.txt"}
	for _, fixture := range fixtures {
		result, err := Recognize(filepath.Join("..", "..", "..", "testdata", fixture), "eng")
		if err != nil {
			t.Fatalf("%s: %v", fixture, err)
		}
		if len(result.Runs) == 0 {
			t.Fatalf("%s: no runs", fixture)
		}
		for _, run := range result.Runs {
			if run.Confidence <= 0 || run.Position.Width <= 0 || run.Position.Height <= 0 {
				t.Fatalf("%s: unauditable run %+v", fixture, run)
			}
		}
	}
}
