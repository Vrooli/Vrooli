package quint

import (
	"fmt"
	"strings"

	"flow-verifier/internal/flows/kinds/temporal/contract"
	"flow-verifier/internal/flows/kinds/temporal/model"
)

type productRender struct {
	Definitions      string
	ApplyAssignments string
}

const TransitionTableRun = GeneratedCheckTransitionTable

func Render(flow model.Flow) string {
	stateTags := make([]string, 0, len(flow.States))
	for _, state := range flow.States {
		stateTags = append(stateTags, state.Quint)
	}
	eventTags := make([]string, 0, len(flow.Events))
	for _, event := range flow.Events {
		eventTags = append(eventTags, event.Quint)
	}

	transitionClauses := make([]string, 0)
	validClauses := make([]string, 0)
	for _, t := range flow.Matrix.Rows() {
		from := quintState(flow, t.From)
		event := quintEvent(flow, t.Event)
		to := quintState(flow, t.To)
		if !t.WantError {
			validClauses = append(validClauses, fmt.Sprintf("(s == %s and e == %s)", from, event))
			if t.From != t.To {
				transitionClauses = append(transitionClauses, fmt.Sprintf("    else if (s == %s and e == %s) %s", from, event, to))
			}
		}
	}
	if len(transitionClauses) > 0 {
		transitionClauses[0] = strings.Replace(transitionClauses[0], "else if", "if", 1)
		transitionClauses = append(transitionClauses, "    else s")
	}

	rows := flow.Matrix.Rows()
	transitionAssertions := make([]string, 0, len(rows))
	for i, t := range rows {
		from := quintState(flow, t.From)
		event := quintEvent(flow, t.Event)
		to := quintState(flow, t.To)
		expectation := fmt.Sprintf("isValid(%s, %s) and nextStatus(%s, %s) == %s", from, event, from, event, to)
		if t.WantError {
			expectation = fmt.Sprintf("not(isValid(%s, %s)) and nextStatus(%s, %s) == %s", from, event, from, event, to)
		}
		if i == 0 {
			transitionAssertions = append(transitionAssertions, fmt.Sprintf("    assert(%s)", expectation))
		} else {
			transitionAssertions = append(transitionAssertions, fmt.Sprintf("      .then(assert(%s))", expectation))
		}
	}

	terminalStates := make([]string, 0)
	for _, state := range flow.States {
		if state.Terminal {
			terminalStates = append(terminalStates, state.Quint)
		}
	}
	terminalClosure := "true"
	if len(terminalStates) > 0 {
		clauses := make([]string, 0, len(terminalStates))
		for _, state := range terminalStates {
			events := make([]string, 0, len(eventTags))
			for _, event := range eventTags {
				events = append(events, fmt.Sprintf("nextStatus(%s, %s) == %s", state, event, state))
			}
			clauses = append(clauses, fmt.Sprintf("(status != %s or (%s))", state, strings.Join(events, " and ")))
		}
		terminalClosure = strings.Join(clauses, " and\n    ")
	}

	customInvariants := make([]string, 0)
	builtins := map[string]bool{
		"TypeOK": true, "TerminalClosure": true, "IllegalTransitionsPreserveState": true,
	}
	for _, invariant := range flow.Invariants {
		if builtins[invariant.Quint] {
			continue
		}
		expr := invariant.Expression
		if expr == "" {
			expr = "true"
		}
		customInvariants = append(customInvariants, fmt.Sprintf("\n  val %s =\n    %s", invariant.Quint, expr))
	}
	productDecls, productInit, productUpdates, productTypeOK, productInvariants, productChecks := renderProduct(flow)

	body := fmt.Sprintf(`%smodule %s {
  type Status = | %s
  type Event = | %s

  var status: Status
  var event: Event
  var rejected: bool
%s

  pure def isValid(s: Status, e: Event): bool =
    %s

  pure def nextStatus(s: Status, e: Event): Status =
%s

%s

  action init = all {
    status' = %s,
    event' = %s,
    rejected' = false,
%s
  }

  action apply(e: Event): bool = all {
    event' = e,
    status' = nextStatus(status, e),
    rejected' = not(isValid(status, e)),
%s
  }

  action step = any {
%s
  }

  val TypeOK = status.in(Set(%s)) and
    event.in(Set(%s))%s

  val TerminalClosure =
    %s

  val IllegalTransitionsPreserveState =
    not(rejected) or nextStatus(status, event) == status

%s

%s

  run %s = {
%s
  }
%s
}

`,
		Header(flow),
		flow.Model.Module,
		strings.Join(stateTags, " | "),
		strings.Join(eventTags, " | "),
		productDecls,
		joinOrFalse(validClauses),
		joinOrSelf(transitionClauses),
		productUpdates.Definitions,
		flow.Initial.Quint,
		eventTags[0],
		productInit,
		productUpdates.ApplyAssignments,
		joinEventApplies(eventTags),
		strings.Join(stateTags, ", "),
		strings.Join(eventTags, ", "),
		productTypeOK,
		terminalClosure,
		strings.Join(customInvariants, ""),
		productInvariants,
		TransitionTableRun,
		strings.Join(transitionAssertions, "\n"),
		productChecks,
	)
	return strings.TrimRight(body, "\n") + "\n"
}

func renderProduct(flow model.Flow) (declarations, initAssignments string, updates productRender, typeOK, invariants, checks string) {
	if flow.Product == nil {
		return "", "", productRender{}, "", "", ""
	}
	p := flow.Product
	var decl, init, defs, assignments strings.Builder
	for _, variable := range p.Variables {
		values := make([]string, 0, len(variable.Values))
		for _, value := range variable.Values {
			values = append(values, value.Quint)
		}
		fmt.Fprintf(&decl, "  type %s = | %s\n  var %s: %s\n", productType(variable), strings.Join(values, " | "), variable.Quint, productType(variable))
		fmt.Fprintf(&init, "    %s' = %s,\n", variable.Quint, productValue(variable, p.Initial[variable.ID]))
		valuesIn := make([]string, 0, len(p.Variables)+2)
		valuesIn = append(valuesIn, "s: Status", "e: Event")
		for _, other := range p.Variables {
			valuesIn = append(valuesIn, other.Quint+": "+productType(other))
		}
		fmt.Fprintf(&defs, "  pure def next%s(%s): %s =\n", pascal(variable.ID), strings.Join(valuesIn, ", "), productType(variable))
		clauses := make([]string, 0)
		for _, transition := range p.Transitions {
			conditions := []string{"s == " + quintState(flow, transition.FromStatus), "e == " + quintEvent(flow, transition.Event)}
			for _, other := range p.Variables {
				conditions = append(conditions, other.Quint+" == "+productValue(other, transition.From[other.ID]))
			}
			clauses = append(clauses, fmt.Sprintf("(%s) %s", strings.Join(conditions, " and "), productValue(variable, transition.To[variable.ID])))
		}
		for i, clause := range clauses {
			predicate, result, _ := strings.Cut(strings.TrimPrefix(clause, "("), ") ")
			if i == 0 {
				fmt.Fprintf(&defs, "    if (%s) %s\n", predicate, result)
			} else {
				fmt.Fprintf(&defs, "    else if (%s) %s\n", predicate, result)
			}
		}
		if len(clauses) > 0 {
			defs.WriteString("    else " + variable.Quint + "\n")
		} else {
			defs.WriteString("    " + variable.Quint + "\n")
		}
		args := []string{"status", "e"}
		for _, other := range p.Variables {
			args = append(args, other.Quint)
		}
		fmt.Fprintf(&assignments, "    %s' = next%s(%s),\n", variable.Quint, pascal(variable.ID), strings.Join(args, ", "))
	}
	var typeChecks []string
	for _, variable := range p.Variables {
		values := make([]string, 0, len(variable.Values))
		for _, value := range variable.Values {
			values = append(values, value.Quint)
		}
		typeChecks = append(typeChecks, variable.Quint+".in(Set("+strings.Join(values, ", ")+"))")
	}
	var invariantBody strings.Builder
	for _, invariant := range p.Invariants {
		fmt.Fprintf(&invariantBody, "\n  val %s =\n    %s\n", invariant.Quint, invariant.Expression)
	}
	var traceChecks strings.Builder
	for i, trace := range p.Traces {
		if i > 0 {
			traceChecks.WriteString("\n\n")
		}
		fmt.Fprintf(&traceChecks, "  run productTrace%s = {\n", pascal(trace.Name))
		status, tuple := trace.InitialStatus, trace.Initial
		for j, step := range trace.Steps {
			parts := []string{"nextStatus(" + quintState(flow, status) + ", " + quintEvent(flow, step.Event) + ") == " + quintState(flow, step.WantStatus)}
			for _, variable := range p.Variables {
				args := []string{quintState(flow, status), quintEvent(flow, step.Event)}
				for _, input := range p.Variables {
					args = append(args, productValue(input, tuple[input.ID]))
				}
				parts = append(parts, "next"+pascal(variable.ID)+"("+strings.Join(args, ", ")+") == "+productValue(variable, step.State[variable.ID]))
			}
			if step.WantError {
				parts = append([]string{"not(isValid(" + quintState(flow, status) + ", " + quintEvent(flow, step.Event) + "))"}, parts...)
			}
			if j == 0 {
				fmt.Fprintf(&traceChecks, "    assert(%s)\n", strings.Join(parts, " and "))
			} else {
				fmt.Fprintf(&traceChecks, "      .then(assert(%s))\n", strings.Join(parts, " and "))
			}
			status, tuple = step.WantStatus, step.State
		}
		traceChecks.WriteString("  }")
	}
	typeOK = " and\n    " + strings.Join(typeChecks, " and\n    ")
	return decl.String(), init.String(), productRender{Definitions: defs.String(), ApplyAssignments: assignments.String()}, typeOK, invariantBody.String(), "\n\n" + traceChecks.String()
}

func productType(variable contract.ProductVariable) string { return pascal(variable.ID) + "Domain" }

func productValue(variable contract.ProductVariable, id string) string {
	for _, value := range variable.Values {
		if value.ID == id {
			return value.Quint
		}
	}
	return id
}

func pascal(value string) string {
	var out strings.Builder
	upper := true
	for _, r := range value {
		if r == '_' || r == '-' || r == ' ' {
			upper = true
			continue
		}
		if upper {
			r = []rune(strings.ToUpper(string(r)))[0]
			upper = false
		}
		out.WriteRune(r)
	}
	return out.String()
}

func Header(flow model.Flow) string {
	return fmt.Sprintf("// Code generated by flow-verifier from %s; DO NOT EDIT.\n", flow.ContractPath)
}

func joinOrFalse(values []string) string {
	if len(values) == 0 {
		return "false"
	}
	return strings.Join(values, " or\n    ")
}

func joinOrSelf(values []string) string {
	if len(values) == 0 {
		return "    s"
	}
	return strings.Join(values, "\n")
}

func joinEventApplies(events []string) string {
	lines := make([]string, 0, len(events))
	for _, event := range events {
		lines = append(lines, "    apply("+event+"),")
	}
	return strings.Join(lines, "\n")
}

func quintState(flow model.Flow, id string) string {
	if state, ok := flow.StateByID[id]; ok {
		return state.Quint
	}
	return id
}

func quintEvent(flow model.Flow, id string) string {
	if event, ok := flow.EventByID[id]; ok {
		return event.Quint
	}
	return id
}
