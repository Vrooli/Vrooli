package aidoor

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"testing"
)

type requestSpec struct {
	States []struct {
		ID                string
		Initial, Terminal bool
	}
	Events             []struct{ ID string }
	TransitionDefaults map[string]struct {
		To        string
		WantError bool
	}
	Transitions []struct{ From, Event, To string }
	Traces      []struct {
		Name, Initial string
		Steps         []struct {
			Event, Want string
			WantError   bool
		}
	}
	CompositionInvariants []string
}

func loadRequestSpec(t *testing.T) requestSpec {
	t.Helper()
	b, err := os.ReadFile("request.spec.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec requestSpec
	if err = json.Unmarshal(b, &spec); err != nil {
		t.Fatal(err)
	}
	return spec
}

func TestRequestSpecAllStateEventPairs(t *testing.T) {
	spec := loadRequestSpec(t)
	states, events := map[string]bool{}, map[string]bool{}
	for _, s := range spec.States {
		if states[s.ID] {
			t.Fatalf("duplicate state %s", s.ID)
		}
		states[s.ID] = true
	}
	for _, e := range spec.Events {
		if events[e.ID] {
			t.Fatalf("duplicate event %s", e.ID)
		}
		events[e.ID] = true
	}
	// Read production declarations so additions cannot silently escape the spec.
	file, err := parser.ParseFile(token.NewFileSet(), "request.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	ast.Inspect(file, func(n ast.Node) bool {
		v, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		typ, ok := v.Type.(*ast.Ident)
		if !ok {
			return true
		}
		set := states
		if typ.Name == "requestEvent" {
			set = events
		} else if typ.Name != "requestStatus" {
			return true
		}
		for _, e := range v.Values {
			lit, ok := e.(*ast.BasicLit)
			if !ok {
				t.Fatal("nonliteral lifecycle declaration")
			}
			value, err := strconv.Unquote(lit.Value)
			if err != nil || !set[value] {
				t.Fatalf("production %s absent in spec: %s", typ.Name, lit.Value)
			}
			counts[typ.Name]++
		}
		return true
	})
	if counts["requestStatus"] != len(states) || counts["requestEvent"] != len(events) {
		t.Fatal("spec and production vocabulary differ")
	}
	for _, key := range []string{"invalid", "terminal"} {
		d := spec.TransitionDefaults[key]
		if d.To != "self" || !d.WantError {
			t.Fatalf("%s default is not fail-closed", key)
		}
	}
	table := map[[2]string]string{}
	for _, r := range spec.Transitions {
		k := [2]string{r.From, r.Event}
		if _, ok := table[k]; ok {
			t.Fatalf("duplicate pair %v", k)
		}
		if !states[r.From] || !states[r.To] || !events[r.Event] {
			t.Fatalf("unknown transition %v", r)
		}
		table[k] = r.To
	}
	for _, s := range spec.States {
		for _, e := range spec.Events {
			t.Run(s.ID+"/"+e.ID, func(t *testing.T) {
				want, valid := table[[2]string{s.ID, e.ID}]
				if !valid {
					want = s.ID
				}
				got, err := transitionRequest(requestStatus(s.ID), requestEvent(e.ID))
				if string(got) != want || (err != nil) == valid {
					t.Fatalf("got %s/%v, want %s error=%t", got, err, want, !valid)
				}
				if s.Terminal && (valid || string(got) != s.ID) {
					t.Fatal("terminal state escaped")
				}
			})
		}
	}
	if len(states)*len(events) != 80 {
		t.Fatalf("review the full matrix: got %d pairs", len(states)*len(events))
	}
	for _, trace := range spec.Traces {
		t.Run(trace.Name, func(t *testing.T) {
			state := requestStatus(trace.Initial)
			for _, step := range trace.Steps {
				next, err := transitionRequest(state, requestEvent(step.Event))
				if string(next) != step.Want || (err != nil) != step.WantError {
					t.Fatalf("%s/%s got %s %v", state, step.Event, next, err)
				}
				state = next
			}
		})
	}
}

func TestRequestCompositionInvariants(t *testing.T) {
	spec := loadRequestSpec(t)
	names := map[string]bool{"reserved_before_dispatch": true, "receipt_before_commit": true, "no_dispatch_after_recovery": true, "commit_once": true}
	for _, name := range spec.CompositionInvariants {
		if !names[name] {
			t.Fatalf("unchecked composition invariant %s", name)
		}
		delete(names, name)
	}
	if len(names) != 0 {
		t.Fatalf("spec omits invariants %v", names)
	}
	type snapshot struct {
		state                                    requestStatus
		reserved, dispatched, receipt, recovered bool
		commits                                  int
	}
	queue := []snapshot{{state: requestAccepted}, {state: requestReconciling, reserved: true, dispatched: true, recovered: true}}
	seen := map[snapshot]bool{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if seen[current] {
			continue
		}
		seen[current] = true
		for _, ev := range spec.Events {
			event := requestEvent(ev.ID)
			next, err := transitionRequest(current.state, event)
			if err != nil {
				continue
			}
			after := current
			after.state = next
			switch event {
			case requestReserve:
				after.reserved = true
			case requestDispatch:
				if !current.reserved || current.recovered || current.dispatched {
					t.Fatalf("illegal redispatch %+v", current)
				}
				after.dispatched = true
			case requestRecover, requestDefer:
				after.recovered = true
			case requestReceive:
				if !current.dispatched && !current.recovered {
					t.Fatalf("receipt before dispatch/recovery %+v", current)
				}
				after.receipt = true
			case requestCommit:
				if !current.receipt || current.commits != 0 {
					t.Fatalf("commit without unique receipt %+v", current)
				}
				after.commits++
			}
			queue = append(queue, after)
		}
	}
	t.Logf("checked %d reachable composite states", len(seen))
}

func TestRequestReceiptCannotSettleAnotherOperationOrRepeatCommit(t *testing.T) {
	d := &Door{}
	f := d.requestLifecycle("owned", requestStreaming)
	if err := f.settle(context.Background(), Key{}, receipt{OperationID: "other", CostKnown: true, State: "completed"}); err == nil || f.state != requestStreaming {
		t.Fatal("foreign receipt admitted")
	}
	if err := f.settle(context.Background(), Key{}, receipt{OperationID: "owned", State: "unknown"}); err == nil || f.state != requestStreaming {
		t.Fatal("unknown cost admitted")
	}
	f.state = requestSettled
	if err := f.settle(context.Background(), Key{}, receipt{OperationID: "owned", CostKnown: true, State: "completed"}); err != nil {
		t.Fatal(err)
	}
}
