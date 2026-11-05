package measures

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func backlogDecl() MeasureDeclaration {
	return MeasureDeclaration{
		Name:      "backlog.completed",
		Domain:    "backlog",
		Intent:    "How many backlog items completed in a window.",
		Questions: []string{"how many backlog items did we complete this week"},
		Params: map[string]Param{
			"window": {Name: "window", Type: ParamTypeTimeWindow, Default: "this_week"},
			"status": {Name: "status", Type: ParamTypeEnum, EnumValues: []string{"done", "archived"}},
		},
		Result:      Result{Kind: ResultScalar, ValueField: "count", Unit: "items"},
		Effect:      EffectRead,
		RunEligible: true,
	}
}

func TestRegistryExecuteRoundTripWithProvenance(t *testing.T) {
	fixedNow := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	reg := NewRegistry(WithClock(func() time.Time { return fixedNow }))

	err := reg.Register(backlogDecl(), func(_ context.Context, req MeasureRequest) (MeasureResult, error) {
		return MeasureResult{
			Value:      "42",
			Provenance: Provenance{ExecutedQuery: "SELECT count(*) ... window=" + req.Params["window"]},
		}, nil
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	out, err := reg.Execute(context.Background(), MeasureRequest{
		Measure: "backlog.completed",
		Params:  map[string]string{"window": "this_week"},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Value != "42" {
		t.Errorf("value = %q, want 42", out.Value)
	}
	if out.Provenance.ExecutedQuery == "" {
		t.Error("provenance.executed_query must be populated")
	}
	if !out.Provenance.ComputedAt.Equal(fixedNow) {
		t.Errorf("provenance.computed_at = %s, want stamped %s", out.Provenance.ComputedAt, fixedNow)
	}
}

func TestRegistryValidation(t *testing.T) {
	reg := NewRegistry()
	noop := func(context.Context, MeasureRequest) (MeasureResult, error) { return MeasureResult{}, nil }

	// Duplicate registration rejected.
	if err := reg.Register(backlogDecl(), noop); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(backlogDecl(), noop); err == nil {
		t.Error("expected duplicate registration error")
	}

	// Unknown measure.
	if _, err := reg.Execute(context.Background(), MeasureRequest{Measure: "nope"}); err == nil {
		t.Error("expected unknown-measure error")
	}

	// Out-of-enum status rejected.
	if _, err := reg.Execute(context.Background(), MeasureRequest{
		Measure: "backlog.completed",
		Params:  map[string]string{"status": "bogus"},
	}); err == nil {
		t.Error("expected out-of-enum validation error")
	}

	// Invalid time-window token rejected.
	if _, err := reg.Execute(context.Background(), MeasureRequest{
		Measure: "backlog.completed",
		Params:  map[string]string{"window": "not_a_token"},
	}); err == nil {
		t.Error("expected invalid time-window token error")
	}
}

func TestRegistryRequiredParamMissing(t *testing.T) {
	decl := backlogDecl()
	decl.Params["status"] = Param{Name: "status", Type: ParamTypeEnum, EnumValues: []string{"done"}, Required: true}
	reg := NewRegistry()
	_ = reg.Register(decl, func(context.Context, MeasureRequest) (MeasureResult, error) {
		return MeasureResult{Value: "1"}, nil
	})
	if _, err := reg.Execute(context.Background(), MeasureRequest{Measure: "backlog.completed"}); err == nil {
		t.Error("expected required-param-missing error")
	}
}

func TestRegistryHandler(t *testing.T) {
	reg := NewRegistry(WithClock(func() time.Time { return time.Unix(1700000000, 0).UTC() }))
	_ = reg.Register(backlogDecl(), func(_ context.Context, req MeasureRequest) (MeasureResult, error) {
		return MeasureResult{Value: "7", Provenance: Provenance{ExecutedQuery: "q"}}, nil
	})
	srv := httptest.NewServer(reg.Handler())
	defer srv.Close()

	// declarations
	resp, err := http.Get(srv.URL + "/declarations")
	if err != nil {
		t.Fatal(err)
	}
	var decls []MeasureDeclaration
	if err := json.NewDecoder(resp.Body).Decode(&decls); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if len(decls) != 1 || decls[0].Name != "backlog.completed" {
		t.Fatalf("declarations = %+v", decls)
	}

	// execute
	body, _ := json.Marshal(MeasureRequest{Measure: "backlog.completed", Params: map[string]string{"window": "this_week"}})
	resp2, err := http.Post(srv.URL+"/execute", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("execute status = %d", resp2.StatusCode)
	}
	var out MeasureResult
	if err := json.NewDecoder(resp2.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Value != "7" || out.Provenance.ComputedAt.IsZero() {
		t.Errorf("execute result = %+v", out)
	}
}
