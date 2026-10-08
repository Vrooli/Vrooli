package metrics

import (
	"context"
	"database/sql"
	"testing"
)

type contextRecordingFeedbackStore struct{ got context.Context }

func (s *contextRecordingFeedbackStore) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, nil
}

func (s *contextRecordingFeedbackStore) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return nil
}

func (s *contextRecordingFeedbackStore) ExecContext(ctx context.Context, _ string, _ ...any) (sql.Result, error) {
	s.got = ctx
	return feedbackTestResult(1), nil
}

type feedbackTestResult int64

func (r feedbackTestResult) LastInsertId() (int64, error) { return 0, nil }
func (r feedbackTestResult) RowsAffected() (int64, error) { return int64(r), nil }

func TestPostgresIntArray(t *testing.T) {
	if got := postgresIntArray([]int{3, 12, -1}); got != "{3,12,-1}" {
		t.Fatalf("postgresIntArray() = %q", got)
	}
}

func TestFeedbackDeleteBulkEmptyIsANoOp(t *testing.T) {
	t.Parallel()

	deleted, err := (&FeedbackService{}).DeleteBulk(context.Background(), nil)
	if err != nil {
		t.Fatalf("DeleteBulk(nil) error = %v", err)
	}
	if deleted != 0 {
		t.Fatalf("DeleteBulk(nil) deleted = %d, want 0", deleted)
	}
}

func TestFeedbackDeleteBulkPreservesRequestContextForPersistence(t *testing.T) {
	t.Parallel()

	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "request-scope")
	store := &contextRecordingFeedbackStore{}
	deleted, err := NewFeedbackService(store).DeleteBulk(ctx, []int{1})
	if err != nil {
		t.Fatalf("DeleteBulk() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("DeleteBulk() deleted = %d, want 1", deleted)
	}
	if store.got != ctx {
		t.Fatal("DeleteBulk() did not pass the request context to persistence")
	}
}
