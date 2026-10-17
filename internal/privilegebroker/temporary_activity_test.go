package privilegebroker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/storageactivity"
)

func TestTemporaryObservationBindsPeerAndNeverExecutesCommand(t *testing.T) {
	fake := &fakeExecutor{}
	var observedUID uint32
	b, err := New(Config{SocketPath: "/tmp/test.sock", AllowedUID: 1000, Executor: fake, TemporaryActivity: func(ctx context.Context, uid uint32) (storageactivity.Snapshot, error) {
		observedUID = uid
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > temporaryObservationTimeout {
			t.Fatal("observation is not bounded")
		}
		return storageactivity.Snapshot{UID: uid, Complete: true, ObservedAt: time.Now(), Paths: []string{"/tmp/held"}}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	req := Request{Version: ProtocolVersion, RequestID: "observe-1", Action: ActionTemporaryActivityObserve}
	denied := b.Execute(context.Background(), req, 1001)
	if denied.Code != "caller_not_authorized" || observedUID != 0 {
		t.Fatalf("denied=%+v uid=%d", denied, observedUID)
	}
	got := b.Execute(context.Background(), req, 1000)
	if got.Status != "completed" || got.Changed || got.RequestID != req.RequestID || got.TemporaryActivity == nil || observedUID != 1000 || len(fake.calls) != 0 {
		t.Fatalf("result=%+v observedUID=%d commands=%v", got, observedUID, fake.calls)
	}
}

func TestTemporaryObservationRefusesPartialOrWrongOwner(t *testing.T) {
	for _, test := range []struct {
		name     string
		snapshot storageactivity.Snapshot
		err      error
	}{
		{"partial", storageactivity.Snapshot{UID: 1000}, nil},
		{"other owner", storageactivity.Snapshot{UID: 1001, Complete: true}, nil},
		{"unreadable", storageactivity.Snapshot{UID: 1000, Complete: true}, errors.New("private kernel data")},
	} {
		t.Run(test.name, func(t *testing.T) {
			b, _ := New(Config{SocketPath: "/tmp/test.sock", AllowedUID: 1000, TemporaryActivity: func(context.Context, uint32) (storageactivity.Snapshot, error) { return test.snapshot, test.err }})
			got := b.Execute(context.Background(), Request{Version: ProtocolVersion, RequestID: "observe-2", Action: ActionTemporaryActivityObserve}, 1000)
			if got.Status != "failed" || got.Code != "temporary_activity_incomplete" || got.TemporaryActivity != nil {
				t.Fatalf("result=%+v", got)
			}
		})
	}
}

func TestCapabilitiesInspectReportsExecutableRegistryWithoutEffect(t *testing.T) {
	fake := &fakeExecutor{}
	b, _ := New(Config{SocketPath: "/tmp/test.sock", AllowedUID: 1000, Executor: fake})
	result := b.Execute(context.Background(), Request{Version: ProtocolVersion, RequestID: "cap-1", Action: ActionCapabilitiesInspect}, 1000)
	if result.Status != "completed" || result.Changed || len(fake.calls) > 0 {
		t.Fatalf("result=%+v", result)
	}
	for _, action := range result.Capabilities {
		switch action {
		case ActionTemporaryActivityObserve, ActionCapabilitiesInspect:
			if err := Validate(Request{Version: ProtocolVersion, RequestID: "validate", Action: action}); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(result.Capabilities) != len(capabilities()) {
		t.Fatalf("capabilities=%v", result.Capabilities)
	}
}
