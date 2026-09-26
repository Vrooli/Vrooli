package capacity

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	engine "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/operatorstate"
)

func TestFitRunsWhenBrokerStoreIsUnavailable(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "resources", "model"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), []byte(`{"dependencies":{"resources":{"model":{"enabled":true}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "resources", "model", "resource.json"), []byte(`{"acceleration":{"backends":["cuda","cpu"],"require":"preferred","claim":{"resource_kind":"vram","default_preferred_bytes":1073741824,"floor_bytes":0,"default_priority":"service","confidence":"estimated","profile":{"steps":[{"label":"gpu","default_amount_bytes":1073741824},{"label":"cpu","default_amount_bytes":0}],"apply":{"verb":"capacity","argv":[]},"upshift":true}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	service := Service{
		SourceRoot: root,
		Source:     engine.StaticSource{Inventory: hostinventory.Snapshot{GPUs: []hostinventory.GPU{{Name: "fixture", Source: "nvidia-smi", VRAMBytes: 2 * 1024 * 1024 * 1024}}}},
		OpenStore:  func(context.Context) (Store, error) { return nil, errors.New("broker stopped") },
		LoadOperatorState: func(context.Context, string) (operatorstate.Document, error) {
			doc := operatorstate.Default()
			zero := int64(0)
			doc.Capacity = &operatorstate.CapacitySettings{TransientHeadroomReserveBytes: &zero}
			return doc, nil
		},
	}
	got, err := service.Fit(context.Background(), FitRequest{})
	if err != nil {
		t.Fatalf("Fit() with stopped broker = %v", err)
	}
	if got.Verdict != engine.FitVerdictFits || got.StaticBytes != 1024*1024*1024 {
		t.Fatalf("Fit() = %#v", got)
	}
}
