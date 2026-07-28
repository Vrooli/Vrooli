package studio

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImportCanonIsIdempotentAndRejectsTemplates(t *testing.T) {
	root := t.TempDir()
	products := filepath.Join(root, "products")
	if err := os.MkdirAll(products, 0o755); err != nil {
		t.Fatal(err)
	}
	valid := `{"slug":"vrooli-console","display_name":"Vrooli Console","product_kind":"scenario","brand_element_placement_rules":{"palette_lock":"slate"}}`
	if err := os.WriteFile(filepath.Join(products, "console.json"), []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(products, "_template.json"), []byte(`{"slug":"REPLACE"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New()
	now := time.Now()
	first := s.ImportCanon(root, "import", now)
	if first.Created != 1 || len(first.Errors) != 0 {
		t.Fatalf("first import = %#v", first)
	}
	second := s.ImportCanon(root, "import", now)
	if second.Created != 0 || second.Revised != 0 {
		t.Fatalf("second import = %#v", second)
	}
	if s.Identities["vrooli-console"].Traits["finish"] != "slate" {
		t.Fatalf("imported identity = %#v", s.Identities["vrooli-console"])
	}
}
