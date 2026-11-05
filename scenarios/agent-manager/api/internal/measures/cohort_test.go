package measures

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/runreport"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

type fakeCohortSource struct {
	definitions map[string]string
}

func (f fakeCohortSource) ShowCohort(_ context.Context, name string, _ int) (invocationreadmodel.CohortDefinition, invocationreadmodel.Cohort, error) {
	filter, ok := f.definitions[name]
	if !ok {
		return invocationreadmodel.CohortDefinition{}, invocationreadmodel.Cohort{}, errors.New("cohort not found")
	}
	return invocationreadmodel.CohortDefinition{Name: name, FilterJSON: filter}, invocationreadmodel.Cohort{}, nil
}

func (fakeCohortSource) BuildRunReport(context.Context, uuid.UUID) (*runreport.RunReport, error) {
	return nil, errors.New("not used")
}

func newCohortHandler() *Handler {
	h := NewHandler(nil, func() time.Time { return time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC) })
	h.SetCohortSource(fakeCohortSource{definitions: map[string]string{"baseline": `{"TagPrefix":"plan-x"}`}})
	return h
}

func TestResolveCohortRejectsMoreThanOneSelector(t *testing.T) {
	h := newCohortHandler()
	id := uuid.NewString()
	filter := &measurepb.InvocationFilter{TagPrefix: "plan-y"}
	for name, args := range map[string]struct {
		cohort string
		runIDs []string
		filter *measurepb.InvocationFilter
	}{
		"cohort and run ids": {cohort: "baseline", runIDs: []string{id}},
		"cohort and filter":  {cohort: "baseline", filter: filter},
		"run ids and filter": {runIDs: []string{id}, filter: filter},
		"all three":          {cohort: "baseline", runIDs: []string{id}, filter: filter},
	} {
		_, err := h.resolveCohort(context.Background(), nil, args.filter, args.cohort, args.runIDs)
		if connect.CodeOf(err) != connect.CodeInvalidArgument || !strings.Contains(err.Error(), "one of cohort, run_ids or filter") {
			t.Errorf("%s: err = %v, want invalid-argument naming the one-selector rule", name, err)
		}
	}
}

func TestResolveCohortExplicitRunIDsAreDedupedAndValidated(t *testing.T) {
	h := newCohortHandler()
	a, b := uuid.NewString(), uuid.NewString()
	selection, err := h.resolveCohort(context.Background(), nil, nil, "", []string{a, " " + a + " ", b})
	if err != nil {
		t.Fatal(err)
	}
	if !selection.explicit() || len(selection.runIDs) != 2 || selection.runIDs[0] != a || selection.runIDs[1] != b {
		t.Fatalf("run ids = %v, want [%s %s]", selection.runIDs, a, b)
	}

	if _, err := h.resolveCohort(context.Background(), nil, nil, "", []string{"not-a-uuid"}); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("non-UUID run id: err = %v, want invalid-argument", err)
	}

	tooMany := make([]string, maxCohortRunIDs+1)
	for i := range tooMany {
		tooMany[i] = uuid.NewString()
	}
	if _, err := h.resolveCohort(context.Background(), nil, nil, "", tooMany); connect.CodeOf(err) != connect.CodeInvalidArgument || !strings.Contains(err.Error(), fmt.Sprint(maxCohortRunIDs)) {
		t.Fatalf("over-limit run ids: err = %v, want invalid-argument naming the %d cap", err, maxCohortRunIDs)
	}
}

func TestResolveCohortNamedUsesDurableDefinition(t *testing.T) {
	h := newCohortHandler()
	selection, err := h.resolveCohort(context.Background(), nil, nil, "baseline", nil)
	if err != nil {
		t.Fatal(err)
	}
	if selection.name != "baseline" || selection.explicit() || selection.filter.TagPrefix != "plan-x" {
		t.Fatalf("selection = %+v, want the stored baseline predicate", selection)
	}

	if _, err := h.resolveCohort(context.Background(), nil, nil, "missing", nil); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unknown cohort: err = %v, want invalid-argument", err)
	}

	unconfigured := NewHandler(nil, nil)
	if _, err := unconfigured.resolveCohort(context.Background(), nil, nil, "baseline", nil); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("no cohort source: err = %v, want internal", err)
	}
}

func TestResolveCohortFilterAppliesWindowAndDefaultsToThisWeek(t *testing.T) {
	h := newCohortHandler()
	selection, err := h.resolveCohort(context.Background(), nil, &measurepb.InvocationFilter{TagPrefix: "plan-y"}, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if selection.explicit() || selection.filter.TagPrefix != "plan-y" || selection.filter.From == nil || selection.filter.To == nil {
		t.Fatalf("selection = %+v, want a windowed predicate", selection)
	}
	// An empty selector is the default population: windowed, no predicate.
	empty, err := h.resolveCohort(context.Background(), nil, nil, "", nil)
	if err != nil || empty.explicit() || empty.filter.From == nil {
		t.Fatalf("empty selector = %+v, %v", empty, err)
	}
}
