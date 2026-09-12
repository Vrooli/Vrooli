package apply_test

import (
	"context"
	"errors"
	"testing"

	"architecture-cartographer/internal/apply"
	"architecture-cartographer/internal/apply/mocks"
	supmocks "architecture-cartographer/internal/suppressions/mocks"
)

type fixedLocator struct{ dir string }

func (l fixedLocator) Locate(string) (string, error) { return l.dir, nil }

func newApplyWithWriter(w *supmocks.FakeWriter) apply.Service {
	return apply.NewService(
		&mocks.FakeRepository{}, &mocks.FakeConflictLister{}, apply.NewRecipeRegistry(),
		apply.WithSuppressionWriter(w, fixedLocator{"/repo/scenarios/demo"}),
	)
}

func TestWriteSuppression_WritesMarker(t *testing.T) {
	w := &supmocks.FakeWriter{}
	svc := newApplyWithWriter(w)
	res, err := svc.WriteSuppression(context.Background(), apply.SuppressionInput{
		Scenario: "demo",
		File:     "api/internal/graph/service.go",
		ID:       "cycle",
		Reason:   "known legacy hub",
		Line:     12,
	})
	if err != nil {
		t.Fatalf("WriteSuppression: %v", err)
	}
	if len(w.Written) != 1 {
		t.Fatalf("expected 1 write, got %d", len(w.Written))
	}
	got := w.Written[0]
	if got.Path != "/repo/scenarios/demo/api/internal/graph/service.go" || got.Line != 12 {
		t.Fatalf("unexpected write target %+v", got)
	}
	if got.Marker.ID != "cycle" || got.Marker.Reason != "known legacy hub" {
		t.Fatalf("unexpected marker %+v", got.Marker)
	}
	if res.File != "api/internal/graph/service.go" {
		t.Fatalf("unexpected result file %q", res.File)
	}
}

func TestWriteSuppression_RequiresReason(t *testing.T) {
	svc := newApplyWithWriter(&supmocks.FakeWriter{})
	_, err := svc.WriteSuppression(context.Background(), apply.SuppressionInput{
		Scenario: "demo", File: "x.go", ID: "cycle",
	})
	var inv apply.ErrInvalidPlanRequest
	if !errors.As(err, &inv) {
		t.Fatalf("want ErrInvalidPlanRequest for missing reason, got %v", err)
	}
}

func TestWriteSuppression_RejectsPathEscape(t *testing.T) {
	svc := newApplyWithWriter(&supmocks.FakeWriter{})
	_, err := svc.WriteSuppression(context.Background(), apply.SuppressionInput{
		Scenario: "demo", File: "../../etc/passwd", ID: "cycle", Reason: "nope",
	})
	if err == nil {
		t.Fatal("expected path-escape rejection")
	}
}

func TestWriteSuppression_Unconfigured(t *testing.T) {
	svc := apply.NewService(&mocks.FakeRepository{}, &mocks.FakeConflictLister{}, apply.NewRecipeRegistry())
	_, err := svc.WriteSuppression(context.Background(), apply.SuppressionInput{
		Scenario: "demo", File: "x.go", ID: "cycle", Reason: "y",
	})
	var unconf apply.ErrSuppressionUnconfigured
	if !errors.As(err, &unconf) {
		t.Fatalf("want ErrSuppressionUnconfigured, got %v", err)
	}
}
