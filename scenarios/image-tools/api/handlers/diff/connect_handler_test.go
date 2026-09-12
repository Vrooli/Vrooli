package diff

import (
	"testing"

	"connectrpc.com/connect"

	diffv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/diff"
)

func TestListDiffModes(t *testing.T) {
	h := NewConnectHandler()
	resp, err := h.ListDiffModes(t.Context(), connect.NewRequest(&diffv1.ListDiffModesRequest{}))
	if err != nil {
		t.Fatalf("ListDiffModes: %v", err)
	}
	modes := resp.Msg.GetModes()
	if len(modes) != 2 {
		t.Fatalf("want 2 modes, got %d", len(modes))
	}
	names := map[string]bool{}
	for _, m := range modes {
		names[m.GetName()] = true
		if m.GetSummary() == "" {
			t.Errorf("mode %q missing summary", m.GetName())
		}
	}
	if !names["pixel"] || !names["perceptual"] {
		t.Errorf("missing expected modes: %v", names)
	}
}
