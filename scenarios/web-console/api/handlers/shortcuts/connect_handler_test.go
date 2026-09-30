package shortcuts

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	shortcutsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shortcuts"
)

type fakeShortcutsService struct{ err error }

func (fakeShortcutsService) Effective(context.Context) Effective {
	return Effective{
		ProfileID: "p1",
		Scope:     "workspace",
		Name:      "Global",
		Shortcuts: []Shortcut{{Label: "List", Command: "ls", Description: "files"}},
	}
}

func (fakeShortcutsService) List(context.Context) []Profile {
	return []Profile{{ID: "p1", Scope: "global", Name: "Global", Shortcuts: []Shortcut{{Label: "List", Command: "ls"}}}}
}

func (f fakeShortcutsService) Upsert(context.Context, UpsertRequest) (Profile, error) {
	return Profile{ID: "p1", Name: "Global"}, f.err
}
func (fakeShortcutsService) Delete(context.Context, string) {}
func TestConnectHandlerShortcuts(t *testing.T) {
	h := NewConnectHandler(Deps{Service: fakeShortcutsService{}})
	resp, err := h.GetEffective(context.Background(), connect.NewRequest(&shortcutsv1.GetEffectiveRequest{}))
	if err != nil || len(resp.Msg.Shortcuts) != 1 {
		t.Fatal(err)
	}
	// The profile identity travels with the list. A client that lets the
	// operator reorder the effective list writes it back to this id; without
	// it, that client has to re-derive scope priority for itself.
	if resp.Msg.GetProfileId() != "p1" || resp.Msg.GetScope() != "workspace" || resp.Msg.GetProfileName() != "Global" {
		t.Fatalf("GetEffective profile = %q/%q/%q, want p1/workspace/Global",
			resp.Msg.GetProfileId(), resp.Msg.GetScope(), resp.Msg.GetProfileName())
	}
	if resp, err := h.ListProfiles(context.Background(), connect.NewRequest(&shortcutsv1.ListProfilesRequest{})); err != nil || len(resp.Msg.Profiles) != 1 {
		t.Fatal(err)
	}
	if resp, err := h.UpsertProfile(context.Background(), connect.NewRequest(&shortcutsv1.UpsertProfileRequest{Id: "p1", Scope: "global", Name: "Global", Shortcuts: []*shortcutsv1.Shortcut{{Label: "List", Command: "ls"}}})); err != nil || resp.Msg.Profile.Id != "p1" {
		t.Fatalf("upsert: %#v %v", resp, err)
	}
	if _, err := h.DeleteProfile(context.Background(), connect.NewRequest(&shortcutsv1.DeleteProfileRequest{Id: "p1"})); err != nil {
		t.Fatal(err)
	}
}

func TestConnectHandlerShortcutsErrors(t *testing.T) {
	for _, in := range []error{ErrInvalidArgument, errors.New("db")} {
		h := NewConnectHandler(Deps{Service: fakeShortcutsService{err: in}})
		_, err := h.UpsertProfile(context.Background(), connect.NewRequest(&shortcutsv1.UpsertProfileRequest{}))
		if err == nil {
			t.Fatal("expected error")
		}
	}
	_ = shortcutsFromProto(nil)
}
