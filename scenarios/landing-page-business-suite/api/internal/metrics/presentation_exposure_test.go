package metrics

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/vrooli/api-core/database"
)

type presentationExposureLegacyStore struct{ execCalls int }

func (s *presentationExposureLegacyStore) QueryRow(string, ...any) *sql.Row        { return nil }
func (s *presentationExposureLegacyStore) Query(string, ...any) (*sql.Rows, error) { return nil, nil }
func (s *presentationExposureLegacyStore) Exec(string, ...any) (sql.Result, error) {
	s.execCalls++
	return nil, errors.New("legacy Store.Exec must not receive presentation exposure")
}

type presentationExposureContextStore struct {
	ctx       context.Context
	rows      int64
	err       error
	execCalls int
}

func (s *presentationExposureContextStore) ExecContext(ctx context.Context, _ string, _ ...any) (sql.Result, error) {
	s.ctx = ctx
	s.execCalls++
	if s.err != nil {
		return nil, s.err
	}
	return presentationExposureResult(s.rows), nil
}

type presentationExposureResult int64

func (r presentationExposureResult) LastInsertId() (int64, error) { return 0, nil }
func (r presentationExposureResult) RowsAffected() (int64, error) { return int64(r), nil }

func TestRecordPresentationExposureUsesLeaseContextStoreAndDeduplicates(t *testing.T) {
	legacy := &presentationExposureLegacyStore{}
	contextStore := &presentationExposureContextStore{rows: 1}
	service := NewServiceWithContextStore(legacy, contextStore)
	ctx := database.WithTestMode(context.Background())

	recorded, err := service.RecordPresentationExposure(ctx, "visitor", "control", "rev-1", "/", "en", "digest", "weights")
	if err != nil || !recorded {
		t.Fatalf("first presentation exposure = recorded:%v error:%v", recorded, err)
	}
	if contextStore.execCalls != 1 || contextStore.ctx != ctx {
		t.Fatalf("lease context was not preserved: calls=%d same=%v", contextStore.execCalls, contextStore.ctx == ctx)
	}
	if !database.IsTestMode(contextStore.ctx) {
		t.Fatal("presentation exposure lost test-lease routing marker")
	}
	if legacy.execCalls != 0 {
		t.Fatalf("presentation exposure fell back to primary Store.Exec: %d", legacy.execCalls)
	}

	contextStore.rows = 0
	recorded, err = service.RecordPresentationExposure(ctx, "visitor", "control", "rev-1", "/", "en", "digest", "weights")
	if err != nil || recorded {
		t.Fatalf("deduplicated presentation exposure = recorded:%v error:%v", recorded, err)
	}
}

func TestRecordPresentationExposureFailsClosedWithoutContextStore(t *testing.T) {
	service := NewServiceWithContextStore(&presentationExposureLegacyStore{}, nil)
	_, err := service.RecordPresentationExposure(context.Background(), "visitor", "control", "rev-1", "/", "en", "digest", "weights")
	if !errors.Is(err, ErrPresentationExposureUnavailable) {
		t.Fatalf("missing context store error = %v, want ErrPresentationExposureUnavailable", err)
	}
}

func TestRecordPresentationExposurePropagatesCanceledLeaseContext(t *testing.T) {
	legacy := &presentationExposureLegacyStore{}
	contextStore := &presentationExposureContextStore{err: context.Canceled}
	service := NewServiceWithContextStore(legacy, contextStore)
	ctx, cancel := context.WithCancel(database.WithTestMode(context.Background()))
	cancel()
	if _, err := service.RecordPresentationExposure(ctx, "visitor", "control", "rev-1", "/", "en", "digest", "weights"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled lease error = %v, want context.Canceled", err)
	}
	if legacy.execCalls != 0 {
		t.Fatal("canceled presentation exposure used primary Store.Exec")
	}
}
