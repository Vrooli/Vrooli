package debt

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"connectrpc.com/connect"
	debtv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/debt"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"
)

// fakeDebtRepo implements catalog.Repository by embedding the interface (so any
// method the handler does not exercise panics if called) and overriding just the
// two debt accessors under test.
type fakeDebtRepo struct {
	catalog.Repository
	list    []catalog.DebtEntry
	listErr error
	get     catalog.DebtEntry
	getErr  error

	gotTemplateID string
	gotStatus     string
	gotKey        string
}

func (f *fakeDebtRepo) ListDebt(_ context.Context, templateID, status string) ([]catalog.DebtEntry, error) {
	f.gotTemplateID = templateID
	f.gotStatus = status
	return f.list, f.listErr
}

func (f *fakeDebtRepo) GetDebt(_ context.Context, key string) (catalog.DebtEntry, error) {
	f.gotKey = key
	return f.get, f.getErr
}

func quietLogger() *log.Logger { return log.New(io.Discard, "", 0) }

func TestListDebtMapsEntriesAndForwardsFilters(t *testing.T) {
	now := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	repo := &fakeDebtRepo{list: []catalog.DebtEntry{{
		Key:         "react-vite.deep.failure",
		TemplateID:  "react-vite",
		Source:      "deep-validation",
		Severity:    "warning",
		Status:      "open",
		Title:       "deep validation failed",
		Detail:      "phase X failed",
		FirstSeenAt: now,
		LastSeenAt:  now,
	}}}
	handler := NewConnectHandler(Deps{Repository: repo, Logger: quietLogger()})

	resp, err := handler.ListDebt(context.Background(), connect.NewRequest(&debtv1.ListDebtRequest{
		TemplateId: "react-vite",
		Status:     "open",
	}))
	if err != nil {
		t.Fatalf("ListDebt: %v", err)
	}
	if repo.gotTemplateID != "react-vite" || repo.gotStatus != "open" {
		t.Fatalf("filters not forwarded: template=%q status=%q", repo.gotTemplateID, repo.gotStatus)
	}
	if len(resp.Msg.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(resp.Msg.Entries))
	}
	got := resp.Msg.Entries[0]
	if got.Key != "react-vite.deep.failure" || got.TemplateId != "react-vite" || got.Status != "open" {
		t.Fatalf("entry mapped wrong: %#v", got)
	}
	if got.FirstSeenAt == nil || !got.FirstSeenAt.AsTime().Equal(now) {
		t.Fatalf("first_seen_at not mapped: %v", got.FirstSeenAt)
	}
}

func TestListDebtRepositoryErrorMapsToInternal(t *testing.T) {
	repo := &fakeDebtRepo{listErr: errors.New("db exploded")}
	handler := NewConnectHandler(Deps{Repository: repo, Logger: quietLogger()})

	_, err := handler.ListDebt(context.Background(), connect.NewRequest(&debtv1.ListDebtRequest{}))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v, want Internal", connect.CodeOf(err))
	}
}

func TestGetDebtReturnsEntry(t *testing.T) {
	repo := &fakeDebtRepo{get: catalog.DebtEntry{Key: "k1", TemplateID: "react-vite", Status: "resolved"}}
	handler := NewConnectHandler(Deps{Repository: repo, Logger: quietLogger()})

	resp, err := handler.GetDebt(context.Background(), connect.NewRequest(&debtv1.GetDebtRequest{Key: "k1"}))
	if err != nil {
		t.Fatalf("GetDebt: %v", err)
	}
	if repo.gotKey != "k1" {
		t.Fatalf("key not forwarded: %q", repo.gotKey)
	}
	if resp.Msg.Entry == nil || resp.Msg.Entry.Key != "k1" || resp.Msg.Entry.Status != "resolved" {
		t.Fatalf("entry mapped wrong: %#v", resp.Msg.Entry)
	}
}

func TestGetDebtNotFoundMapsToNotFound(t *testing.T) {
	repo := &fakeDebtRepo{getErr: catalog.ErrNotFound{Kind: "debt entry", ID: "missing"}}
	handler := NewConnectHandler(Deps{Repository: repo, Logger: quietLogger()})

	_, err := handler.GetDebt(context.Background(), connect.NewRequest(&debtv1.GetDebtRequest{Key: "missing"}))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code = %v, want NotFound", connect.CodeOf(err))
	}
}

func TestGetDebtOtherErrorMapsToInternal(t *testing.T) {
	repo := &fakeDebtRepo{getErr: errors.New("db exploded")}
	handler := NewConnectHandler(Deps{Repository: repo, Logger: quietLogger()})

	_, err := handler.GetDebt(context.Background(), connect.NewRequest(&debtv1.GetDebtRequest{Key: "k1"}))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v, want Internal", connect.CodeOf(err))
	}
}
