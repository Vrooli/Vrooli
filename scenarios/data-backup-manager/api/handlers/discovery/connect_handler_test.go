package discovery

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"data-backup-manager/internal/discovery"
	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/sysmounts"

	destinationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations"
	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/discovery"
	sourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/sources"
)

// fakeService is a local discovery.Service double for handler translation tests.
type fakeService struct {
	targets     []discovery.TargetSuggestion
	dests       []discovery.DestinationSuggestion
	targetsErr  error
	destsErr    error
	dismissOut  bool
	dismissErr  error
	dismissedID string
}

func (f *fakeService) ListTargetSuggestions(context.Context) ([]discovery.TargetSuggestion, error) {
	return f.targets, f.targetsErr
}

func (f *fakeService) ListDestinationSuggestions(context.Context) ([]discovery.DestinationSuggestion, error) {
	return f.dests, f.destsErr
}

func (f *fakeService) DismissSuggestion(_ context.Context, id string) (bool, error) {
	f.dismissedID = id
	return f.dismissOut, f.dismissErr
}

var _ discovery.Service = (*fakeService)(nil)

func TestDiscoveryService_Contract(t *testing.T) {
	ctx := context.Background()

	t.Run("ListTargetSuggestions maps enum and fields", func(t *testing.T) {
		svc := &fakeService{targets: []discovery.TargetSuggestion{
			{ID: "abc123", TargetCandidate: discovery.TargetCandidate{
				Owner: "vrooli", Name: "runtime-db", SourceKind: sources.KindSQLite,
				Locator: "/home/u/.vrooli/runtime.db", Rationale: "db", ApproxBytes: 4096,
			}},
		}}
		h := NewConnectHandler(Deps{Service: svc})

		resp, err := h.ListTargetSuggestions(ctx, connect.NewRequest(&discoveryv1.ListTargetSuggestionsRequest{}))
		if err != nil {
			t.Fatalf("ListTargetSuggestions: %v", err)
		}
		if len(resp.Msg.Suggestions) != 1 {
			t.Fatalf("want 1 suggestion, got %d", len(resp.Msg.Suggestions))
		}
		got := resp.Msg.Suggestions[0]
		if got.Id != "abc123" || got.Owner != "vrooli" || got.Name != "runtime-db" {
			t.Fatalf("wrong identity fields: %+v", got)
		}
		if got.SourceKind != sourcesv1.SourceKind_SOURCE_KIND_SQLITE {
			t.Fatalf("source kind = %v, want sqlite", got.SourceKind)
		}
		if got.ApproxBytes != 4096 {
			t.Fatalf("approx bytes = %d, want 4096", got.ApproxBytes)
		}
	})

	t.Run("ListDestinationSuggestions maps class and backend", func(t *testing.T) {
		svc := &fakeService{dests: []discovery.DestinationSuggestion{
			{
				ID: "d1", Label: "USB", Location: "/media/u/USB", Class: sysmounts.ClassRemovable,
				FreeBytes: 50, TotalBytes: 64, Removable: true, SeparateRootOK: true, Rationale: "ok",
			},
			{
				ID: "d2", Label: "root", Location: "/", Class: sysmounts.ClassFixed,
				SeparateRootOK: false, Rationale: "overlaps",
			},
		}}
		h := NewConnectHandler(Deps{Service: svc})

		resp, err := h.ListDestinationSuggestions(ctx, connect.NewRequest(&discoveryv1.ListDestinationSuggestionsRequest{}))
		if err != nil {
			t.Fatalf("ListDestinationSuggestions: %v", err)
		}
		if len(resp.Msg.Suggestions) != 2 {
			t.Fatalf("want 2, got %d", len(resp.Msg.Suggestions))
		}
		usb := resp.Msg.Suggestions[0]
		if usb.DriveClass != discoveryv1.DriveClass_DRIVE_CLASS_REMOVABLE || !usb.Removable || !usb.SeparateRootOk {
			t.Fatalf("usb mapped wrong: %+v", usb)
		}
		if usb.BackendKind != destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM {
			t.Fatalf("backend = %v, want filesystem", usb.BackendKind)
		}
		if root := resp.Msg.Suggestions[1]; root.DriveClass != discoveryv1.DriveClass_DRIVE_CLASS_FIXED || root.SeparateRootOk {
			t.Fatalf("root mapped wrong: %+v", root)
		}
	})

	t.Run("DismissSuggestion passes id and returns flag", func(t *testing.T) {
		svc := &fakeService{dismissOut: true}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.DismissSuggestion(ctx, connect.NewRequest(&discoveryv1.DismissSuggestionRequest{Id: "xyz"}))
		if err != nil {
			t.Fatalf("DismissSuggestion: %v", err)
		}
		if !resp.Msg.Dismissed {
			t.Fatal("dismissed = false, want true")
		}
		if svc.dismissedID != "xyz" {
			t.Fatalf("service got id %q, want xyz", svc.dismissedID)
		}
	})

	t.Run("DismissSuggestion surfaces invalid-argument", func(t *testing.T) {
		svc := &fakeService{dismissErr: discovery.ErrInvalidDiscovery{Field: "id", Reason: "required"}}
		h := NewConnectHandler(Deps{Service: svc})
		_, err := h.DismissSuggestion(ctx, connect.NewRequest(&discoveryv1.DismissSuggestionRequest{Id: "x"}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
		}
	})

	t.Run("internal errors surface as internal", func(t *testing.T) {
		svc := &fakeService{targetsErr: errors.New("disk exploded")}
		h := NewConnectHandler(Deps{Service: svc})
		_, err := h.ListTargetSuggestions(ctx, connect.NewRequest(&discoveryv1.ListTargetSuggestionsRequest{}))
		if connect.CodeOf(err) != connect.CodeInternal {
			t.Fatalf("code = %v, want internal", connect.CodeOf(err))
		}
	})
}
