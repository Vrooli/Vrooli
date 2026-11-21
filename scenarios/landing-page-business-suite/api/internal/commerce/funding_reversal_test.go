package commerce

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The explicit matrix is the oracle; never derive expectations with the same
// branching rules as nextDisputeStatus. The AST vocabulary check catches a newly
// accepted Stripe state even when nobody remembered to extend the contract.
func TestFundingDisputeWorkflowConformsToSpec(t *testing.T) {
	data, err := os.ReadFile("funding_reversal.spec.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		FlowID, Domain, Initial, FormalModel string
		States, Events, Terminal, Invariants []string
		WebhookEventKinds                    []string
		Transitions                          []struct{ FromState, Event, To string }
		Traces                               []struct {
			ID, Initial        string
			Incoming, Expected []string
		}
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	if spec.FlowID != "commerce.funding-payment-reversal" || spec.Domain != "commerce" || spec.Initial != "" || spec.FormalModel != "not_adopted" || len(spec.Invariants) == 0 {
		t.Fatal("incomplete contract identity/invariants")
	}
	file, err := parser.ParseFile(token.NewFileSet(), "funding_reversal.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var vocabulary []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "nextDisputeStatus" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			sw, ok := n.(*ast.SwitchStmt)
			if !ok {
				return true
			}
			tag, ok := sw.Tag.(*ast.Ident)
			if !ok || tag.Name != "incoming" {
				return true
			}
			for _, item := range sw.Body.List {
				for _, expr := range item.(*ast.CaseClause).List {
					literal, ok := expr.(*ast.BasicLit)
					if !ok || literal.Kind != token.STRING {
						t.Fatal("dispute vocabulary declaration changed; update extractor")
					}
					value, err := strconv.Unquote(literal.Value)
					if err != nil {
						t.Fatal(err)
					}
					vocabulary = append(vocabulary, value)
				}
			}
			return false
		})
	}
	assertSet := func(label string, got, want []string) {
		t.Helper()
		seen := map[string]bool{}
		for _, v := range got {
			if seen[v] {
				t.Fatalf("duplicate %s %q", label, v)
			}
			seen[v] = true
		}
		got = slices.Clone(got)
		want = slices.Clone(want)
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Fatalf("%s drift: got %v want %v", label, got, want)
		}
	}
	if len(vocabulary) == 0 {
		t.Fatal("production vocabulary not found")
	}
	assertSet("terminal", spec.Terminal, []string{"won", "warning_closed", "prevented"})
	webhook, err := parser.ParseFile(token.NewFileSet(), "stripe_webhook_service.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, decl := range webhook.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "HandleWebhook" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			clause, ok := n.(*ast.CaseClause)
			if !ok {
				return true
			}
			for _, expr := range clause.List {
				literal, ok := expr.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				value, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatal(err)
				}
				if strings.HasPrefix(value, "checkout.") || strings.HasPrefix(value, "charge.") || strings.HasPrefix(value, "radar.") {
					kinds = append(kinds, value)
				}
			}
			return true
		})
	}
	assertSet("handled payment webhook kinds", spec.WebhookEventKinds, kinds)
	assertSet("events", spec.Events, vocabulary)
	assertSet("states", spec.States, append([]string{""}, vocabulary...))
	pairs := map[[2]string]string{}
	for _, row := range spec.Transitions {
		key := [2]string{row.FromState, row.Event}
		if _, exists := pairs[key]; exists || !slices.Contains(spec.States, row.FromState) || !slices.Contains(spec.Events, row.Event) || !slices.Contains(spec.States, row.To) {
			t.Fatalf("duplicate or unknown matrix row: %+v", row)
		}
		if slices.Contains(spec.Terminal, row.FromState) && row.To != row.FromState {
			t.Fatalf("terminal escape: %+v", row)
		}
		pairs[key] = row.To
	}
	for _, current := range spec.States {
		for _, event := range spec.Events {
			want, exists := pairs[[2]string{current, event}]
			if !exists {
				t.Fatalf("missing pair %q + %q", current, event)
			}
			got, err := nextDisputeStatus(current, event)
			if err != nil || got != want {
				t.Errorf("%q + %q => %q %v want %q", current, event, got, err, want)
			}
		}
		got, err := nextDisputeStatus(current, "unknown")
		if err == nil || got != current {
			t.Fatal("unknown event must reject without mutation")
		}
	}
	seenTraces := map[string]bool{}
	for _, trace := range spec.Traces {
		if trace.ID == "" || seenTraces[trace.ID] || len(trace.Incoming) == 0 || len(trace.Incoming) != len(trace.Expected) || !slices.Contains(spec.States, trace.Initial) {
			t.Fatalf("invalid trace: %+v", trace)
		}
		seenTraces[trace.ID] = true
		state := trace.Initial
		for i, event := range trace.Incoming {
			want, ok := pairs[[2]string{state, event}]
			if !ok || want != trace.Expected[i] {
				t.Fatalf("trace %s step %d drifts from matrix", trace.ID, i)
			}
			state, err = nextDisputeStatus(state, event)
			if err != nil || state != trace.Expected[i] {
				t.Fatalf("trace %s step %d: %s %v", trace.ID, i, state, err)
			}
		}
	}
	t.Logf("SM4 dispute contract: %d states, %d events, %d pairs, %d traces", len(spec.States), len(spec.Events), len(pairs), len(spec.Traces))
}
func TestFundingRefundRoundingPreservesMoneyBounds(t *testing.T) {
	for _, test := range []struct{ credits, part, total, want int64 }{{100, 1, 3, 34}, {100, 0, 3, 0}, {math.MaxInt64, math.MaxInt64, math.MaxInt64, math.MaxInt64}, {math.MaxInt64, 1, math.MaxInt64, 1}} {
		if got := proportionalCredits(test.credits, test.part, test.total); got != test.want {
			t.Errorf("refund credits=%d part=%d total=%d =>%d want%d", test.credits, test.part, test.total, got, test.want)
		}
	}
	if roundedCostMinor(math.MaxInt64) < 0 {
		t.Fatal("minor currency overflow")
	}
	for _, invalid := range []any{nil, "100", -1.0, 1.5, math.Inf(1), 9007199254740992.0} {
		if _, err := validStripeMinorAmount(invalid); err == nil {
			t.Errorf("invalid Stripe amount accepted: %v", invalid)
		}
	}
}
