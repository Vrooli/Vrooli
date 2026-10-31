package monetization

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCommittedMeterInventoryMatchesDeclarations(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := BuildMeterInventory(root)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	expected = append(expected, '\n')
	actual, err := os.ReadFile(filepath.Join(root, "packages", "monetization-go", "meter-inventory.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatalf("committed meter inventory is stale; run go run ./cmd/meter-inventory\nwant:\n%s\ngot:\n%s", expected, actual)
	}
}
