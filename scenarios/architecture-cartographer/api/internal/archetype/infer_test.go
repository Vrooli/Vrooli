package archetype

import "testing"

func TestInfer(t *testing.T) {
	cases := []struct {
		name string
		in   Input
		want string
	}{
		{"mutation", Input{Name: "apply", Paths: []string{"api/internal/apply/"}}, "mutation"},
		{"reporting", Input{Name: "analytics", Paths: []string{"api/internal/analytics/"}}, "reporting"},
		{"service", Input{Name: "graph", Paths: []string{"api/internal/graph/"}}, "service"},
		{"unknown", Input{Name: "docs", Paths: []string{"docs/concepts/"}}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Infer(tc.in)
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("Infer() = %+v, want none", got)
				}
				return
			}
			if len(got) == 0 || got[0].Name != tc.want || got[0].Confidence == 0 || len(got[0].Evidence) == 0 {
				t.Fatalf("Infer() = %+v, want primary %q result with evidence", got, tc.want)
			}
		})
	}
}

func TestInfer_CodeSignals(t *testing.T) {
	// File-writing + storage + handler: mutation (specific) is primary over the
	// generic service signals.
	got := Infer(Input{
		Name:           "apply",
		Paths:          []string{"api/internal/apply/"},
		WritesFiles:    true,
		OwnsStorage:    true,
		HasHTTPHandler: true,
	})
	if len(got) == 0 || got[0].Name != string(Mutation) {
		t.Fatalf("expected mutation primary, got %+v", got)
	}
	// Corroboration boost: two service signals (storage + handler + api path)
	// should raise service confidence above a single generic 0.6.
	var serviceConf float64
	for _, r := range got {
		if r.Name == string(Service) {
			serviceConf = r.Confidence
		}
	}
	if serviceConf <= 0.6 {
		t.Fatalf("expected corroboration boost on service, got %v (%+v)", serviceConf, got)
	}
}

func TestInfer_WorkflowOrchestration(t *testing.T) {
	got := Infer(Input{Name: "audit", Paths: []string{"api/internal/audit/"}, HasWorkflow: true})
	found := false
	for _, r := range got {
		if r.Name == string(Orchestration) {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected orchestration from workflow signal, got %+v", got)
	}
}
