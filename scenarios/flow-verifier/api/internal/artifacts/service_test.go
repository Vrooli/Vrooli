package artifacts

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"flow-verifier/internal/testkit"
)

func TestFindFlowWrapsUnknownFlowAsNotFound(t *testing.T) {
	_, err := findFlow(t.TempDir(), "missing.flow")
	if !errors.Is(err, ErrFlowNotFound) {
		t.Fatalf("findFlow() error = %v, want ErrFlowNotFound", err)
	}
}

func TestFindFlowIgnoresInvalidUnrelatedContracts(t *testing.T) {
	root := t.TempDir()
	raw := testkit.ValidRawContract()
	raw.FlowID = "example.target.api"
	testkit.WriteFlowJSON(t, root, "api/target/flow/flow.json", raw)

	badPath := filepath.Join(root, "ui/unrelated/flow/flow.json")
	if err := os.MkdirAll(filepath.Dir(badPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(badPath, []byte(`{"kind":"temporal","flowId":"example.unrelated.ui"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	flow, err := findFlow(root, raw.FlowID)
	if err != nil {
		t.Fatalf("findFlow() error = %v", err)
	}
	if got, want := flow.FlowID, raw.FlowID; got != want {
		t.Fatalf("flow id = %s, want %s", got, want)
	}
}
