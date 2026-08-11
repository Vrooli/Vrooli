package programs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
)

type fakeRunner struct {
	result Result
	err    error
}

func (r fakeRunner) Execute(context.Context, string, string, bool) (Result, error) {
	return r.result, r.err
}

func TestRetainsProgramSourceAndFailureDetail(t *testing.T) { // [REQ:PRT-P1-006]
	s := NewService(Options{Runner: fakeRunner{err: errors.New("field title: invalid")}})
	p, err := s.Submit(context.Background(), "s1", "raise ValueError()", programsv1.Provenance_PROVENANCE_AGENT, false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(context.Background(), p.Id)
	if err != nil || got.Source != "raise ValueError()" || got.FailureDetail == "" {
		t.Fatalf("program=%+v err=%v", got, err)
	}
}

func TestRecurringFailureShapesAreDerivable(t *testing.T) { // [REQ:PRT-P1-006]
	s := NewService(Options{Runner: fakeRunner{err: errors.New("field title: invalid")}})
	for i := 0; i < 3; i++ {
		if _, err := s.Submit(context.Background(), "s1", "x", programsv1.Provenance_PROVENANCE_AGENT, false); err != nil {
			t.Fatal(err)
		}
	}
	shapes := s.MineFailures(context.Background(), false)
	if len(shapes) != 1 || shapes[0].Count != 3 {
		t.Fatalf("shapes=%v", shapes)
	}
}

func TestSubmissionRecordsProvenance(t *testing.T) { // [REQ:PRT-P1-008]
	s := NewService(Options{})
	p, err := s.Submit(context.Background(), "s1", "x=1", programsv1.Provenance_PROVENANCE_OPERATOR, false)
	if err != nil {
		t.Fatal(err)
	}
	if p.Provenance != programsv1.Provenance_PROVENANCE_OPERATOR {
		t.Fatalf("provenance=%v", p.Provenance)
	}
}

func TestMiningExcludesOperatorProgramsByDefault(t *testing.T) { // [REQ:PRT-P1-008]
	s := NewService(Options{Runner: fakeRunner{err: errors.New("same")}})
	_, _ = s.Submit(context.Background(), "s1", "x", programsv1.Provenance_PROVENANCE_OPERATOR, false)
	_, _ = s.Submit(context.Background(), "s1", "x", programsv1.Provenance_PROVENANCE_AGENT, false)
	shapes := s.MineFailures(context.Background(), false)
	if len(shapes) != 1 || shapes[0].Count != 1 {
		t.Fatalf("shapes=%v", shapes)
	}
}

func TestDeadlineFailureUsesStableFailureShape(t *testing.T) { // [REQ:PRT-P1-005]
	s := NewService(Options{Runner: fakeRunner{err: &DeadlineExceededError{Limit: 2 * time.Second}}})
	p, err := s.Submit(context.Background(), "s1", "while True: pass", programsv1.Provenance_PROVENANCE_AGENT, false)
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != "failed" || p.FailureShape != "deadline_exceeded" || !strings.Contains(p.FailureDetail, "2s") {
		t.Fatalf("program=%+v", p)
	}
}
