package selection

import (
	"context"
	"testing"

	internalselection "image-tools/internal/selection"

	"connectrpc.com/connect"

	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/selection"
)

func TestListRegionClasses(t *testing.T) {
	h := NewConnectHandler()
	resp, err := h.ListRegionClasses(context.Background(), connect.NewRequest(&selectionv1.ListRegionClassesRequest{}))
	if err != nil {
		t.Fatalf("ListRegionClasses: %v", err)
	}
	if len(resp.Msg.Classes) == 0 {
		t.Fatal("no region classes returned")
	}
	for _, c := range resp.Msg.Classes {
		if c.Name == "" || len(c.Edits) == 0 {
			t.Errorf("class %q has no edits", c.Name)
		}
	}
}

func TestSuggestEditsKnownAndUnknown(t *testing.T) {
	h := NewConnectHandler()
	known, err := h.SuggestEdits(context.Background(), connect.NewRequest(&selectionv1.SuggestEditsRequest{RegionClass: internalselection.ClassPerson}))
	if err != nil {
		t.Fatalf("SuggestEdits(person): %v", err)
	}
	if known.Msg.RegionClass != internalselection.ClassPerson || len(known.Msg.Edits) == 0 {
		t.Errorf("person: class=%q edits=%d", known.Msg.RegionClass, len(known.Msg.Edits))
	}

	unknown, err := h.SuggestEdits(context.Background(), connect.NewRequest(&selectionv1.SuggestEditsRequest{RegionClass: "bogus"}))
	if err != nil {
		t.Fatalf("SuggestEdits(bogus): %v", err)
	}
	if unknown.Msg.RegionClass != internalselection.ClassObject {
		t.Errorf("unknown class should resolve to object, got %q", unknown.Msg.RegionClass)
	}
}
