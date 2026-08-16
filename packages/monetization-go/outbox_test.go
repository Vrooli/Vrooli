package monetization

import (
	"context"
	"errors"
	"testing"
	"time"
)

type recordingTransport struct {
	seen  []Usage
	erros []error
}

func (t *recordingTransport) Report(_ context.Context, usage Usage) error {
	t.seen = append(t.seen, usage)
	if len(t.erros) > 0 {
		err := t.erros[0]
		t.erros = t.erros[1:]
		return err
	}
	return nil
}

func testUsage() Usage {
	return Usage{
		OperationID:  "operation-1",
		UserIdentity: "user@example.com",
		BundleKey:    "business_suite",
		AppKey:       "browser-automation-studio",
		MeterKey:     "workflow_executions",
		Units:        1,
		OccurredAt:   time.Unix(100, 0).UTC(),
		Metadata:     map[string]string{"source": "test"},
	}
}

func TestOutboxEnqueueIsIdempotent(t *testing.T) {
	store := NewMemoryOutboxStore()
	outbox := NewOutbox(store, &recordingTransport{})
	usage := testUsage()
	if err := outbox.Enqueue(context.Background(), usage); err != nil {
		t.Fatal(err)
	}
	if err := outbox.Enqueue(context.Background(), usage); err != nil {
		t.Fatal(err)
	}
	records, err := store.Pending(context.Background(), 10, time.Unix(200, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one durable record, got %d", len(records))
	}
}

func TestOutboxRetriesAndDeliversAfterTransportRecovery(t *testing.T) {
	store := NewMemoryOutboxStore()
	transport := &recordingTransport{erros: []error{errors.New("offline")}}
	outbox := NewOutbox(store, transport)
	outbox.Now = func() time.Time { return time.Unix(100, 0) }
	outbox.BaseDelay = time.Second
	usage := testUsage()
	if err := outbox.Enqueue(context.Background(), usage); err != nil {
		t.Fatal(err)
	}
	delivered, err := outbox.Drain(context.Background(), 10)
	if delivered != 0 || err == nil {
		t.Fatalf("expected failed first delivery, delivered=%d err=%v", delivered, err)
	}
	records, err := store.Pending(context.Background(), 10, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Fatal("record should wait for retry delay")
	}
	outbox.Now = func() time.Time { return time.Unix(101, 0) }
	delivered, err = outbox.Drain(context.Background(), 10)
	if err != nil || delivered != 1 {
		t.Fatalf("expected recovered delivery, delivered=%d err=%v", delivered, err)
	}
	if len(transport.seen) != 2 {
		t.Fatalf("expected two transport attempts, got %d", len(transport.seen))
	}
	if transport.seen[0].OperationID != transport.seen[1].OperationID {
		t.Fatal("retry changed the idempotency key")
	}
}

func TestOutboxRejectsIncompleteUsage(t *testing.T) {
	outbox := NewOutbox(NewMemoryOutboxStore(), &recordingTransport{})
	if err := outbox.Enqueue(context.Background(), Usage{}); !errors.Is(err, ErrInvalidUsage) {
		t.Fatalf("expected ErrInvalidUsage, got %v", err)
	}
}
