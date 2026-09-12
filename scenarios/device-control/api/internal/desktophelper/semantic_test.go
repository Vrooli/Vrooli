package desktophelper

import (
	"context"
	"image"
	"strings"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"

	"device-control/internal/native/atspi"
	"device-control/internal/sessions"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/targetmodel"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/encoding/protojson"
)

type semanticPixelFixture struct{}

func (semanticPixelFixture) Validate(context.Context, sessions.DesktopCommand) error { return nil }
func (semanticPixelFixture) Apply(context.Context, sessions.DesktopCommand) error    { return nil }
func (semanticPixelFixture) ReleaseHeld(context.Context) error                       { return nil }
func (semanticPixelFixture) CheckSession(context.Context) error                      { return nil }
func (semanticPixelFixture) Observe(context.Context) (sessions.DesktopObservation, error) {
	return sessions.DesktopObservation{Image: image.NewRGBA(image.Rect(0, 0, 2, 2)), DisplayID: "display", GeometryRevision: "geometry", CapturedAt: time.Now()}, nil
}

type refusingSemanticPixelFixture struct{ semanticPixelFixture }

func (refusingSemanticPixelFixture) CheckSession(context.Context) error { return atspi.ErrRefused }

func TestSemanticObservationRefusesProtectedSession(t *testing.T) { // NAT-14
	backend := &semanticBackend{pixels: refusingSemanticPixelFixture{}, access: &semanticAccessFixture{}, busID: "bus"}
	_, err := backend.ObserveProcess(context.Background(), 42, "lease-1")
	require.ErrorIs(t, err, atspi.ErrRefused)
}

type semanticAccessFixture struct {
	text      string
	mutations int
	invokes   int
	named     string
}

func (f *semanticAccessFixture) ProcessRoot(context.Context, string, uint32) (atspi.Ref, error) {
	return atspi.Ref{Owner: ":1.42", Path: "/org/a11y/atspi/accessible/root", PID: 42}, nil
}

func (f *semanticAccessFixture) Children(context.Context, atspi.Ref) ([]atspi.Ref, error) {
	return nil, nil
}
func (f *semanticAccessFixture) Name(context.Context, atspi.Ref) (string, error)   { return "entry", nil }
func (f *semanticAccessFixture) Editable(context.Context, atspi.Ref) (bool, error) { return true, nil }
func (f *semanticAccessFixture) Description(context.Context, atspi.Ref) (string, error) {
	return "", nil
}

func (f *semanticAccessFixture) State(context.Context, atspi.Ref) ([]string, error) {
	return []string{"enabled", "showing", "visible"}, nil
}

func (f *semanticAccessFixture) Bounds(context.Context, atspi.Ref) (int32, int32, uint32, uint32, bool, error) {
	return 0, 0, 100, 30, true, nil
}

func (f *semanticAccessFixture) Actions(context.Context, atspi.Ref) ([]string, bool, error) {
	return []string{"invoke", "expand"}, true, nil
}
func (f *semanticAccessFixture) Focus(context.Context, atspi.Ref) error { return nil }
func (f *semanticAccessFixture) ReadText(context.Context, atspi.Ref) (string, error) {
	return f.text, nil
}

func (f *semanticAccessFixture) InsertTextChecked(_ context.Context, _ atspi.Ref, observed string, pos int32, text string, _ func(context.Context) error) error {
	if observed != f.text {
		return atspi.ErrRefused
	}
	runes := []rune(f.text)
	f.text = string(runes[:pos]) + text + string(runes[pos:])
	f.mutations++
	return nil
}

func (f *semanticAccessFixture) Invoke(context.Context, atspi.Ref) error {
	f.invokes++
	return nil
}
func (f *semanticAccessFixture) InvokeNamed(_ context.Context, _ atspi.Ref, action string) error {
	f.invokes++
	f.named = action
	return nil
}

func TestSemanticElementCacheIsLeaseScopedAndSingleUse(t *testing.T) {
	access := &semanticAccessFixture{text: "é|tail"}
	backend := &semanticBackend{pixels: semanticPixelFixture{}, access: access, busID: "bus"}
	snapshot, err := backend.ObserveProcess(context.Background(), 42, "lease-1")
	require.NoError(t, err)
	element := snapshot.Semantic.Elements[0]
	require.NotEqual(t, "/org/a11y/atspi/accessible/root", element.ID)
	payload, err := protojson.Marshal(&desktopv1.Action{Action: &desktopv1.Action_Text{Text: &desktopv1.TextAction{Text: "日本語 🧪", ElementId: element.ID, ObservationRevision: snapshot.Semantic.Revision, Position: 2}}})
	require.NoError(t, err)
	command := sessions.DesktopCommand{Lease: sessions.DesktopLease{Ref: targetmodel.SessionRef{SessionID: "other-lease"}}, GeometryRevision: "geometry", Payload: payload}
	require.Error(t, backend.Validate(context.Background(), command))
	require.Error(t, backend.Apply(context.Background(), command))
	require.Zero(t, access.mutations)
	command.Lease.Ref.SessionID = "lease-1"
	require.NoError(t, backend.Validate(context.Background(), command))
	access.text = "changed"
	require.Error(t, backend.Validate(context.Background(), command))
	access.text = "é|tail"
	require.NoError(t, backend.Apply(context.Background(), command))
	require.Equal(t, "é|日本語 🧪tail", access.text)
	require.Error(t, backend.Apply(context.Background(), command))
	require.Equal(t, 1, access.mutations)
	snapshot, err = backend.ObserveProcess(context.Background(), 42, "lease-1")
	require.NoError(t, err)
	require.NoError(t, backend.ReleaseHeld(context.Background()))
	require.Empty(t, backend.entries)
	require.NotEmpty(t, snapshot.Semantic.Revision)
	require.NotZero(t, snapshot.Semantic.RefreshEpoch)
	require.NotEmpty(t, snapshot.Semantic.Elements[0].Fingerprint)
}

func TestSemanticInvokeUsesObservedElementAndInvalidatesLease(t *testing.T) {
	access := &semanticAccessFixture{text: "button"}
	backend := &semanticBackend{pixels: semanticPixelFixture{}, access: access, busID: "bus"}
	snapshot, err := backend.ObserveProcess(context.Background(), 42, "lease-1")
	require.NoError(t, err)
	element := snapshot.Semantic.Elements[0]
	action := &desktopv1.Action{Action: &desktopv1.Action_Invoke{Invoke: &desktopv1.InvokeAction{ElementId: element.ID, ObservationRevision: snapshot.Semantic.Revision}}}
	payload, err := protojson.Marshal(action)
	require.NoError(t, err)
	command := sessions.DesktopCommand{Lease: sessions.DesktopLease{Ref: targetmodel.SessionRef{SessionID: "lease-1"}}, GeometryRevision: snapshot.GeometryRevision, Payload: payload}
	require.NoError(t, backend.Validate(context.Background(), command))
	require.NoError(t, backend.Apply(context.Background(), command))
	require.Equal(t, 1, access.invokes)
	require.Error(t, backend.Apply(context.Background(), command))
}

func TestSemanticInvokeUsesExplicitNativeActionName(t *testing.T) {
	access := &semanticAccessFixture{text: "button"}
	backend := &semanticBackend{pixels: semanticPixelFixture{}, access: access, busID: "bus"}
	snapshot, err := backend.ObserveProcess(context.Background(), 42, "lease-1")
	require.NoError(t, err)
	element := snapshot.Semantic.Elements[0]
	action := &desktopv1.Action{Action: &desktopv1.Action_Invoke{Invoke: &desktopv1.InvokeAction{ElementId: element.ID, ObservationRevision: snapshot.Semantic.Revision, ActionName: "expand"}}}
	payload, err := protojson.Marshal(action)
	require.NoError(t, err)
	command := sessions.DesktopCommand{Lease: sessions.DesktopLease{Ref: targetmodel.SessionRef{SessionID: "lease-1"}}, GeometryRevision: snapshot.GeometryRevision, Payload: payload}
	require.NoError(t, backend.Validate(context.Background(), command))
	require.NoError(t, backend.Apply(context.Background(), command))
	require.Equal(t, "expand", access.named)
}

type applicationAccessFixture struct {
	semanticAccessFixture
	owner       string
	selected    atspi.Ref
	resolutions int
}

func (f *applicationAccessFixture) Applications(context.Context, string) ([]atspi.Application, error) {
	return []atspi.Application{{Ref: atspi.Ref{Owner: f.owner, Path: "/org/a11y/atspi/accessible/root", PID: 42}, Name: "Editor"}}, nil
}

func (f *applicationAccessFixture) ProcessRoot(context.Context, string, uint32) (atspi.Ref, error) {
	f.resolutions++
	return atspi.Ref{}, atspi.ErrRefused
}

func (f *applicationAccessFixture) Name(_ context.Context, ref atspi.Ref) (string, error) {
	f.selected = ref
	if ref.Owner != f.owner {
		return "", atspi.ErrRefused
	}
	return "Editor", nil
}

func TestApplicationSelectionRetainsExactOwnerAndLease(t *testing.T) {
	ctx := context.Background()
	access := &applicationAccessFixture{owner: ":1.42"}
	backend := &semanticBackend{pixels: semanticPixelFixture{}, access: access, busID: "bus"}
	catalog, err := backend.Applications(ctx, "lease-1")
	require.NoError(t, err)
	require.Len(t, catalog.Applications, 1)
	id := catalog.Applications[0].ID
	require.NotEqual(t, access.owner, id)
	for _, selection := range [][3]string{{id, catalog.Revision, "lease-2"}, {id, "wrong-revision", "lease-1"}, {"missing", catalog.Revision, "lease-1"}} {
		_, err := backend.ObserveApplication(ctx, selection[0], selection[1], selection[2])
		require.Error(t, err)
	}
	snapshot, err := backend.ObserveApplication(ctx, id, catalog.Revision, "lease-1")
	require.NoError(t, err)
	require.Equal(t, uint32(42), snapshot.Semantic.ProcessID)
	require.Equal(t, access.owner, access.selected.Owner)
	access.owner = ":1.99" // Replacement may have the same PID and display name.
	_, err = backend.ObserveApplication(ctx, id, catalog.Revision, "lease-1")
	require.Error(t, err)
	require.Empty(t, backend.entries)
	require.Zero(t, access.resolutions, "selection must not re-resolve by PID")
	access.owner = ":1.42"
	backend.applicationExpires = time.Now().Add(-time.Second)
	_, err = backend.ObserveApplication(ctx, id, catalog.Revision, "lease-1")
	require.Error(t, err)
	next, err := backend.Applications(ctx, "lease-1")
	require.NoError(t, err)
	_, err = backend.ObserveApplication(ctx, id, catalog.Revision, "lease-1")
	require.Error(t, err, "refresh invalidates prior references")
	require.NoError(t, backend.ReleaseHeld(ctx))
	_, err = backend.ObserveApplication(ctx, next.Applications[0].ID, next.Revision, "lease-1")
	require.Error(t, err, "Stop clears application references")
}

func (f *semanticAccessFixture) Role(context.Context, atspi.Ref) (uint32, error) { return 61, nil }

type windowAccessFixture struct{ semanticAccessFixture }

func (f *windowAccessFixture) Children(_ context.Context, r atspi.Ref) ([]atspi.Ref, error) {
	paths := map[string][]string{"/org/a11y/atspi/accessible/root": {"/org/a11y/atspi/accessible/window1", "/org/a11y/atspi/accessible/window2"}, "/org/a11y/atspi/accessible/window1": {"/org/a11y/atspi/accessible/field1"}, "/org/a11y/atspi/accessible/window2": {"/org/a11y/atspi/accessible/field2"}}
	var out []atspi.Ref
	for _, p := range paths[string(r.Path)] {
		child := r
		child.Path = dbus.ObjectPath(p)
		out = append(out, child)
	}
	return out, nil
}

func (f *windowAccessFixture) Role(_ context.Context, r atspi.Ref) (uint32, error) {
	if strings.Contains(string(r.Path), "window") {
		return 23, nil
	}
	return 61, nil
}

func TestSemanticWindowsKeepDuplicateFieldNamesDistinct(t *testing.T) {
	backend := &semanticBackend{pixels: semanticPixelFixture{}, access: &windowAccessFixture{}, busID: "bus"}
	got, err := backend.ObserveProcess(context.Background(), 42, "lease")
	require.NoError(t, err)
	require.Len(t, got.Semantic.Elements, 5)
	first, second := got.Semantic.Elements[3], got.Semantic.Elements[4]
	require.Equal(t, first.Name, second.Name)
	require.NotEqual(t, first.ID, second.ID)
	require.NotEqual(t, first.WindowID, second.WindowID)
	require.Equal(t, got.Semantic.Elements[1].ID, first.WindowID)
	require.Equal(t, got.Semantic.Elements[2].ID, second.WindowID)
	require.Equal(t, first.WindowID, first.ParentID)
}

func (f *windowAccessFixture) Editable(_ context.Context, r atspi.Ref) (bool, error) {
	return strings.Contains(string(r.Path), "field"), nil
}

func TestResolutionDistinguishesAbsentUniqueAndAmbiguousWithinWindow(t *testing.T) {
	ctx := context.Background()
	b := &semanticBackend{pixels: semanticPixelFixture{}, access: &windowAccessFixture{}, busID: "bus"}
	snapshot, err := b.ObserveProcess(ctx, 42, "lease")
	require.NoError(t, err)
	selector := sessions.DesktopSelector{Revision: snapshot.Semantic.Revision, WindowID: snapshot.Semantic.Elements[1].ID, Name: "entry", EditableOnly: true}
	got, err := b.Resolve(ctx, selector, "lease")
	require.NoError(t, err)
	require.Equal(t, "unique", got.Disposition)
	require.Equal(t, []string{snapshot.Semantic.Elements[3].ID}, got.ElementIDs)
	require.Equal(t, snapshot.GeometryRevision, got.GeometryRevision)
	selector.Name = "absent"
	got, err = b.Resolve(ctx, selector, "lease")
	require.NoError(t, err)
	require.Equal(t, "absent", got.Disposition)
	require.Empty(t, got.ElementIDs)
	selector.Name = "entry"
	duplicate := snapshot.Semantic.Elements[3]
	duplicate.ID = "another-field"
	b.observed = append(b.observed, duplicate)
	b.entries[duplicate.ID] = b.entries[snapshot.Semantic.Elements[3].ID]
	got, err = b.Resolve(ctx, selector, "lease")
	require.NoError(t, err)
	require.Equal(t, "ambiguous", got.Disposition)
	require.Len(t, got.ElementIDs, 2)
	_, err = b.Resolve(ctx, selector, "other-lease")
	require.Error(t, err)
	selector.WindowID = "missing"
	_, err = b.Resolve(ctx, selector, "lease")
	require.Error(t, err)
	selector.WindowID = snapshot.Semantic.Elements[1].ID
	b.expires = time.Now().Add(-time.Second)
	_, err = b.Resolve(ctx, selector, "lease")
	require.Error(t, err)
	require.NoError(t, b.ReleaseHeld(ctx))
	require.Empty(t, b.observed)
}

func TestResolutionRequiresExplicitMatchModeAndSupportsBoundedVariants(t *testing.T) {
	ctx := context.Background()
	b := &semanticBackend{pixels: semanticPixelFixture{}, access: &windowAccessFixture{}, busID: "bus"}
	snapshot, err := b.ObserveProcess(ctx, 42, "lease")
	require.NoError(t, err)
	window := snapshot.Semantic.Elements[1].ID

	exact := sessions.DesktopSelector{Revision: snapshot.Semantic.Revision, WindowID: window, Name: "ENTRY", EditableOnly: true}
	exact.RefreshEpoch = snapshot.Semantic.RefreshEpoch + 1
	_, err = b.Resolve(ctx, exact, "lease")
	require.Error(t, err, "selectors bound to a stale refresh epoch must refuse")
	exact.RefreshEpoch = snapshot.Semantic.RefreshEpoch
	got, err := b.Resolve(ctx, exact, "lease")
	require.NoError(t, err)
	require.Equal(t, "absent", got.Disposition, "exact remains the safe default")

	exact.MatchMode = sessions.SemanticMatchNormalized
	got, err = b.Resolve(ctx, exact, "lease")
	require.NoError(t, err)
	require.Equal(t, "unique", got.Disposition)

	exact.Name = "entr"
	exact.MatchMode = sessions.SemanticMatchFuzzy
	exact.Role = 61
	got, err = b.Resolve(ctx, exact, "lease")
	require.NoError(t, err)
	require.Equal(t, "unique", got.Disposition)

	exact.Role = 23
	got, err = b.Resolve(ctx, exact, "lease")
	require.NoError(t, err)
	require.Equal(t, "absent", got.Disposition, "role is a hard fuzzy constraint")
}

type gatedSemanticAccessFixture struct {
	windowAccessFixture
	mode string
}

func (f *gatedSemanticAccessFixture) State(ctx context.Context, ref atspi.Ref) ([]string, error) {
	if strings.Contains(string(ref.Path), "field1") {
		switch f.mode {
		case "hidden":
			return []string{"enabled"}, nil
		case "disabled":
			return []string{"showing", "visible"}, nil
		}
	}
	return f.windowAccessFixture.State(ctx, ref)
}

func (f *gatedSemanticAccessFixture) Bounds(ctx context.Context, ref atspi.Ref) (int32, int32, uint32, uint32, bool, error) {
	if f.mode == "offscreen" && strings.Contains(string(ref.Path), "field1") {
		return 0, 0, 0, 30, true, nil
	}
	return f.windowAccessFixture.Bounds(ctx, ref)
}

func TestResolutionRejectsUnavailableSemanticStatesUnlessExplicitlyAllowed(t *testing.T) {
	for _, mode := range []string{"hidden", "disabled", "offscreen"} {
		t.Run(mode, func(t *testing.T) {
			access := &gatedSemanticAccessFixture{mode: mode}
			backend := &semanticBackend{pixels: semanticPixelFixture{}, access: access, busID: "bus"}
			snapshot, err := backend.ObserveProcess(context.Background(), 42, "lease")
			require.NoError(t, err)
			selector := sessions.DesktopSelector{Revision: snapshot.Semantic.Revision, WindowID: snapshot.Semantic.Elements[1].ID, Name: "entry", EditableOnly: true}
			got, err := backend.Resolve(context.Background(), selector, "lease")
			require.NoError(t, err)
			require.Equal(t, "absent", got.Disposition)
			switch mode {
			case "hidden":
				selector.AllowHidden = true
			case "disabled":
				selector.AllowDisabled = true
			case "offscreen":
				selector.AllowOffscreen = true
			}
			got, err = backend.Resolve(context.Background(), selector, "lease")
			require.NoError(t, err)
			require.Equal(t, "unique", got.Disposition)
		})
	}
}

type changingWindowFixture struct {
	windowAccessFixture
	change string
}

func (f *changingWindowFixture) Children(ctx context.Context, r atspi.Ref) ([]atspi.Ref, error) {
	children, err := f.windowAccessFixture.Children(ctx, r)
	if f.change == "children" && strings.Contains(string(r.Path), "window1") {
		child := r
		child.Path = "/org/a11y/atspi/accessible/new-field"
		children = append(children, child)
	}
	return children, err
}

func (f *changingWindowFixture) Name(ctx context.Context, r atspi.Ref) (string, error) {
	if f.change == "name" {
		return "renamed", nil
	}
	return f.windowAccessFixture.Name(ctx, r)
}

func (f *changingWindowFixture) Role(ctx context.Context, r atspi.Ref) (uint32, error) {
	if f.change == "role" {
		return 0, nil
	}
	return f.windowAccessFixture.Role(ctx, r)
}

func (f *changingWindowFixture) Editable(ctx context.Context, r atspi.Ref) (bool, error) {
	if f.change == "editable" {
		return false, nil
	}
	return f.windowAccessFixture.Editable(ctx, r)
}

func TestResolutionRefusesChangedNativeTree(t *testing.T) {
	for _, change := range []string{"children", "name", "role", "editable"} {
		t.Run(change, func(t *testing.T) {
			ctx := context.Background()
			access := &changingWindowFixture{}
			b := &semanticBackend{pixels: semanticPixelFixture{}, access: access, busID: "bus"}
			snapshot, err := b.ObserveProcess(ctx, 42, "lease")
			require.NoError(t, err)
			selector := sessions.DesktopSelector{Revision: snapshot.Semantic.Revision, WindowID: snapshot.Semantic.Elements[1].ID, Name: "entry", EditableOnly: true}
			got, err := b.Resolve(ctx, selector, "lease")
			require.NoError(t, err)
			require.Equal(t, "unique", got.Disposition)
			access.change = change
			got, err = b.Resolve(ctx, selector, "lease")
			require.Error(t, err)
			require.Empty(t, got.ElementIDs)
		})
	}
}
