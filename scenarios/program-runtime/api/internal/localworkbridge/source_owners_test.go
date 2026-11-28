package localworkbridge

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type selectedReaderFixture struct {
	calls int
	raw   []byte
	read  func(context.Context, func(context.Context) error, func(context.Context, string, string, string) error) error
}

func (r *selectedReaderFixture) ReadSelected(ctx context.Context, id string, check func(context.Context) error, validate func(context.Context, string, string, string) error) ([]byte, error) {
	r.calls++
	if id != "event-1" {
		return nil, errors.New("bad id")
	}
	if r.read != nil {
		if err := r.read(ctx, check, validate); err != nil {
			return nil, err
		}
	}
	return r.raw, nil
}
func TestFiniteSourceOwnersEventComposition(t *testing.T) {
	b, _, _, old, _, _ := brokerFixture(t)
	reader := &selectedReaderFixture{raw: old.event}
	o, err := NewFiniteSourceOwners(true, reader, old)
	if err != nil {
		t.Fatal(err)
	}
	b.owners = o
	raw, err := b.Execute(context.Background(), brokerRequest("read_event"))
	if err != nil || !json.Valid(raw) || reader.calls != 1 {
		t.Fatalf("read=%s %v calls=%d", raw, err, reader.calls)
	}
	reader.raw = []byte(`{"eventId":"wrong","eventType":"agent_manager.run.notification.v1","source":{"scenario":"agent-manager"}}`)
	if raw, err = b.Execute(context.Background(), brokerRequest("read_event")); err == nil || raw != nil {
		t.Fatal("invalid event released")
	}
}
func TestFiniteSourceOwnersFinalOwnerRevocation(t *testing.T) {
	b, _, _, old, bindings, _ := brokerFixture(t)
	reader := &selectedReaderFixture{raw: old.event}
	reader.read = func(ctx context.Context, check func(context.Context) error, validate func(context.Context, string, string, string) error) error {
		bindings.b.Revoked = true
		return check(ctx)
	}
	o, _ := NewFiniteSourceOwners(true, reader, old)
	b.owners = o
	if raw, err := b.Execute(context.Background(), brokerRequest("read_event")); err == nil || raw != nil {
		t.Fatal("revocation released data")
	}
}
func TestFiniteSourceOwnersDisabledNoIO(t *testing.T) {
	reader := &selectedReaderFixture{}
	collector := &brokerOwners{}
	o, err := NewFiniteSourceOwners(false, reader, collector)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = o.ReadEvent(nil, nil); !errors.Is(err, ErrDisabled) {
		t.Fatal(err)
	}
	if _, err = o.AdmitCollector(nil, nil); !errors.Is(err, ErrDisabled) {
		t.Fatal(err)
	}
	if reader.calls != 0 || collector.calls != 0 {
		t.Fatal("disabled IO")
	}
	if _, err = NewFiniteSourceOwners(true, nil, collector); err == nil {
		t.Fatal("nil event owner")
	}
}
func TestFiniteSourceOwnersCollectorRoutes(t *testing.T) {
	for _, op := range []string{"admit_collector", "observe_collector", "wait_collector"} {
		t.Run(op, func(t *testing.T) {
			b, _, _, old, _, _ := brokerFixture(t)
			o, _ := NewFiniteSourceOwners(true, &selectedReaderFixture{}, old)
			b.owners = o
			if _, err := b.Execute(context.Background(), brokerRequest(op)); err != nil {
				t.Fatal(err)
			}
			if old.operation != op {
				t.Fatal(old.operation)
			}
		})
	}
}
func TestCollectorSpecDefensivePolicyCopy(t *testing.T) {
	g, _, _ := sourceFixture(t, SourceCollector)
	p, err := g.Admit(context.Background(), sourceRequest("POST", sourceLibrary+"GetDeclaredExecution", sourceObserveBody()))
	if err != nil {
		t.Fatal(err)
	}
	name, digest, inputs, err := p.CollectorSpec()
	if err != nil || name != SourceCollectorName || digest != g.policy.CollectorDigest {
		t.Fatal("wrong spec")
	}
	inputs[0] = 'x'
	_, _, again, _ := p.CollectorSpec()
	if !json.Valid(again) {
		t.Fatal("mutated policy")
	}
	if _, _, _, err = (*SourcePermit)(nil).CollectorSpec(); err == nil {
		t.Fatal("nil permit")
	}
}
