package runannotations

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"
	"github.com/google/cel-go/common/types"
)

// A composite label is a rule over other labels. Its inputs are named in the
// rule as literal keys (labels['a.b'], present['a.b'],
// message_labels['a.b']), so the pack can be checked before it
// runs: every input exists, lives in a scope the composite can see, and the
// composites form no cycle. The same names drive three behaviours: the order a
// pass evaluates labels in, the inputs a requested composite pulls in, and the
// proof a composite observation inherits from the observations it read.

// compositeRefs are the labels a composite rule reads.
type compositeRefs struct {
	// Labels are the labels of the composite's own scope read through labels[...]
	// or present[...].
	Labels []string
	// Messages are the message-scope labels a run composite reads through
	// message_labels[...].
	Messages []string
	// Dynamic is set when the rule reads one of the label maps in a way that
	// does not name its key (a computed key, size(), a comprehension), so its
	// inputs cannot be known.
	Dynamic bool
}

func (r compositeRefs) any() bool { return len(r.Labels)+len(r.Messages) > 0 }

// labelMaps are the composite-only variables that hold labels.
var labelMaps = map[string]bool{"labels": true, "present": true, "message_labels": true}

// analyzeComposite reads which labels a composite rule names. The rule must
// already compile in the composite environment.
func analyzeComposite(env *cel.Env, rule string) (compositeRefs, error) {
	checked, issues := env.Compile(rule)
	if issues != nil && issues.Err() != nil {
		return compositeRefs{}, issues.Err()
	}
	var (
		refs    compositeRefs
		handled = map[int64]bool{}
		idents  = map[int64]bool{}
		labels  = map[string]bool{}
		message = map[string]bool{}
	)
	record := func(ident, key string) {
		if ident == "message_labels" {
			message[key] = true
		} else {
			labels[key] = true
		}
	}
	literalKey := func(expr celast.Expr) (string, bool) {
		if expr.Kind() != celast.LiteralKind {
			return "", false
		}
		text, ok := expr.AsLiteral().(types.String)
		return string(text), ok
	}
	celast.PreOrderVisit(checked.NativeRep().Expr(), celast.NewExprVisitor(func(expr celast.Expr) {
		switch expr.Kind() {
		case celast.IdentKind:
			if labelMaps[expr.AsIdent()] {
				idents[expr.ID()] = true
			}
		case celast.SelectKind:
			selected := expr.AsSelect()
			if operand := selected.Operand(); operand.Kind() == celast.IdentKind && labelMaps[operand.AsIdent()] {
				record(operand.AsIdent(), selected.FieldName())
				handled[operand.ID()] = true
			}
		case celast.CallKind:
			call := expr.AsCall()
			args := call.Args()
			switch call.FunctionName() {
			case operators.Index:
				if len(args) == 2 && args[0].Kind() == celast.IdentKind && labelMaps[args[0].AsIdent()] {
					if key, ok := literalKey(args[1]); ok {
						record(args[0].AsIdent(), key)
						handled[args[0].ID()] = true
					}
				}
			case operators.In:
				if len(args) == 2 && args[1].Kind() == celast.IdentKind && labelMaps[args[1].AsIdent()] {
					if key, ok := literalKey(args[0]); ok {
						record(args[1].AsIdent(), key)
						handled[args[1].ID()] = true
					}
				}
			}
		}
	}))
	for id := range idents {
		if !handled[id] {
			refs.Dynamic = true
		}
	}
	refs.Labels, refs.Messages = sortedKeys(labels), sortedKeys(message)
	return refs, nil
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// validateComposites checks the inputs of every composite against the pack.
func (p *Pack) validateComposites(env *cel.Env, add func(string, ...any)) {
	byID := map[string]Label{}
	for _, label := range p.Labels {
		byID[label.ID] = label
	}
	reads := map[string][]string{}
	for _, label := range p.Labels {
		if label.Method != MethodComposite || strings.TrimSpace(label.Rule) == "" {
			continue
		}
		refs, err := analyzeComposite(env, label.Rule)
		if err != nil {
			continue // reported by the label's own validation
		}
		if refs.Dynamic {
			add("%s: a composite must name each label it reads as a literal key (labels['id']); a computed key hides its inputs", label.ID)
		}
		for _, id := range refs.Labels {
			input, ok := byID[id]
			switch {
			case !ok:
				add("%s: reads unknown label %q", label.ID, id)
			case id == label.ID:
				add("%s: a composite cannot read itself", label.ID)
			case input.Scope != label.Scope:
				add("%s: reads %s, a %s label, through labels[...]; a %s composite reads %s labels there", label.ID, id, input.Scope, label.Scope, label.Scope, label.Scope)
			default:
				reads[label.ID] = append(reads[label.ID], id)
			}
		}
		for _, id := range refs.Messages {
			input, ok := byID[id]
			switch {
			case label.Scope != ScopeRun:
				add("%s: message_labels is read by a run composite; this one is %s scope", label.ID, label.Scope)
			case !ok:
				add("%s: reads unknown label %q", label.ID, id)
			case input.Scope != ScopeMessage:
				add("%s: message_labels[%q] must name a message-scope label; it is %s scope", label.ID, id, input.Scope)
			}
		}
		if refs.any() && len(label.Gate.Measurements) > 0 {
			add("%s: a composite over labels takes its proof from the observations it reads; it cannot declare gate measurements", label.ID)
		}
	}
	// A composite that reads another composite of its scope needs that one first;
	// a cycle has no first.
	state := map[string]int{}
	var visit func(id string) bool
	visit = func(id string) bool {
		switch state[id] {
		case 1:
			return false
		case 2:
			return true
		}
		state[id] = 1
		for _, next := range reads[id] {
			if !visit(next) {
				return false
			}
		}
		state[id] = 2
		return true
	}
	for _, label := range p.Labels {
		if !visit(label.ID) {
			add("%s: composites read each other in a cycle", label.ID)
			return
		}
	}
}

// compositeOrder returns the composite ids of one scope with every composite
// after the composites it reads.
func (cp *CompiledPack) compositeOrder(scope Scope) []string {
	var order []string
	placed := map[string]bool{}
	var place func(id string)
	place = func(id string) {
		if placed[id] {
			return
		}
		placed[id] = true
		compiled, ok := cp.labels[id]
		if !ok || compiled.label.Method != MethodComposite || compiled.label.Scope != scope {
			return
		}
		for _, input := range compiled.refs.Labels {
			place(input)
		}
		order = append(order, id)
	}
	for _, id := range cp.order {
		place(id)
	}
	return order
}

// requested is the set of labels a call asked for, closed over what the
// requested composites read: asking for a composite asks for its inputs. A nil
// set means every label.
func (cp *CompiledPack) requested(ids []string) map[string]bool {
	if len(ids) == 0 {
		return nil
	}
	set := map[string]bool{}
	var add func(id string)
	add = func(id string) {
		if set[id] {
			return
		}
		set[id] = true
		if compiled, ok := cp.labels[id]; ok {
			for _, input := range append(append([]string(nil), compiled.refs.Labels...), compiled.refs.Messages...) {
				add(input)
			}
		}
	}
	for _, id := range ids {
		add(id)
	}
	return set
}

// MessageStats summarizes one message-scope label over a run's messages, as a
// run composite reads it through message_labels['id'].
type MessageStats struct {
	Count     int
	Positives int
	Max       float64
	Mean      float64
}

func (m MessageStats) activation() map[string]float64 {
	return map[string]float64{"count": float64(m.Count), "positives": float64(m.Positives), "max": m.Max, "mean": m.Mean}
}

// messageStats folds a run's message observations of one label: the value of
// each on the label's own scale, tempered as the annotation view tempers it (see
// calibrationOf), and how many are friction positives. The measured friction
// threshold applies only to a proven observation; an unproven one is judged by
// the label's declared threshold.
func (cp *CompiledPack) messageStats(compiled *compiledLabel, rows []Annotation) (MessageStats, bool) {
	var (
		stats  MessageStats
		sum    float64
		valued int
	)
	for _, row := range rows {
		if row.State != StateObserved && row.State != StateCompatible {
			continue
		}
		stats.Count++
		temperature, threshold := calibrationOf(compiled.label, row)
		if cp.StatusFor(compiled, row).State != GateProven {
			threshold = 0
		}
		if isFrictionPositive(compiled.label, row, temperature, threshold) {
			stats.Positives++
		}
		if value, ok := calibratedValue(compiled.label, row, temperature); ok {
			if valued == 0 || value > stats.Max {
				stats.Max = value
			}
			sum += value
			valued++
		}
	}
	if stats.Count == 0 {
		return MessageStats{}, false
	}
	if valued > 0 {
		stats.Mean = sum / float64(valued)
	}
	return stats, true
}

// inputKey names an observation by what proves it.
func inputKey(label string, row Annotation) InputKey {
	model, role := ProofKey(row)
	return InputKey{Label: label, DefinitionHash: row.DefinitionHash, Model: model, ResourceRole: role}
}

// leafKeys are the proof keys of an input observation: its own, or, when it is
// itself a composite over labels, the keys of what it read.
func leafKeys(label string, row Annotation) []InputKey {
	if len(row.Value.Inputs) > 0 {
		return row.Value.Inputs
	}
	return []InputKey{inputKey(label, row)}
}

// mergeInputs collects keys without duplicates, in a stable order.
func mergeInputs(into []InputKey, more ...InputKey) []InputKey {
	for _, key := range more {
		found := false
		for _, existing := range into {
			if existing == key {
				found = true
				break
			}
		}
		if !found {
			into = append(into, key)
		}
	}
	sort.Slice(into, func(i, j int) bool {
		a, b := into[i], into[j]
		switch {
		case a.Label != b.Label:
			return a.Label < b.Label
		case a.DefinitionHash != b.DefinitionHash:
			return a.DefinitionHash < b.DefinitionHash
		case a.Model != b.Model:
			return a.Model < b.Model
		}
		return a.ResourceRole < b.ResourceRole
	})
	return into
}

func describeInput(key InputKey) string {
	if key.ResourceRole == "" {
		return fmt.Sprintf("%s (%s)", key.Label, key.Model)
	}
	return fmt.Sprintf("%s (%s @ %s)", key.Label, key.Model, key.ResourceRole)
}
