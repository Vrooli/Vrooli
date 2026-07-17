package workflowruntime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"agent-manager/internal/domain"
	"agent-manager/internal/workflowexpr"
)

const MaxRenderedPromptBytes = 64 << 10

type BindingContext struct {
	Input   json.RawMessage
	Journal []*domain.WorkflowJournalEntry
}

// BindingDiagnostic is deterministic renderer evidence that the engine journals
// alongside the attempt that consumed the binding.
type BindingDiagnostic struct {
	Binding      string `json:"binding"`
	Code         string `json:"code"`
	DroppedBytes int    `json:"droppedBytes,omitempty"`
	ElidedItems  int    `json:"elidedItems,omitempty"`
}

type selectedBindingValue struct {
	Value    any
	NodeID   string
	Sequence int64
}

func EvaluateBindings(bindings []domain.WorkflowInputBinding, ctx BindingContext) (map[string]any, error) {
	values, _, err := EvaluateBindingsWithDiagnostics(bindings, ctx)
	return values, err
}

func EvaluateBindingsWithDiagnostics(bindings []domain.WorkflowInputBinding, ctx BindingContext) (map[string]any, []BindingDiagnostic, error) {
	out := make(map[string]any, len(bindings))
	diagnostics := []BindingDiagnostic{}
	for _, binding := range bindings {
		selected, err := selectBinding(binding, ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("binding %s: %w", binding.Name, err)
		}
		if len(selected) == 0 {
			switch binding.MissingPolicy {
			case "omit":
				continue
			case "null":
				out[binding.Name] = nil
			default:
				return nil, nil, fmt.Errorf("binding %s selected no values", binding.Name)
			}
			continue
		}
		values := make([]any, len(selected))
		for i := range selected {
			values[i] = selected[i].Value
		}
		var value any
		if len(values) == 1 {
			value = values[0]
		} else {
			value = values
		}
		var rendered string
		if len(selected) > 1 && binding.RenderAs == "xml" && binding.EvictionPolicy != "" {
			var listDiagnostics []BindingDiagnostic
			rendered, listDiagnostics, err = renderXMLList(binding, selected)
			diagnostics = append(diagnostics, listDiagnostics...)
		} else {
			rendered, err = renderBinding(binding, value)
		}
		if err != nil {
			return nil, nil, err
		}
		if binding.RenderAs == "json" && len(rendered) > binding.MaxBytes {
			return nil, nil, fmt.Errorf("binding %s exceeds %d bytes", binding.Name, binding.MaxBytes)
		}
		if len(rendered) > binding.MaxBytes {
			if binding.Overflow != "truncate" {
				return nil, nil, fmt.Errorf("binding %s exceeds %d bytes", binding.Name, binding.MaxBytes)
			}
			diagnostics = append(diagnostics, BindingDiagnostic{Binding: binding.Name, Code: "binding_truncated", DroppedBytes: len(rendered) - binding.MaxBytes})
			rendered = truncateRendered(rendered, binding.MaxBytes)
		}
		// json is the long-standing structured binding form used by end nodes to
		// assemble schema-validated execution output. The presentation forms are
		// strings for text/template; json_pretty is the explicit prompt-facing
		// JSON renderer when formatting matters.
		if binding.RenderAs == "json" {
			out[binding.Name] = value
		} else {
			out[binding.Name] = rendered
		}
	}
	return out, diagnostics, nil
}

func renderBinding(binding domain.WorkflowInputBinding, value any) (string, error) {
	compact, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	text := string(compact)
	if s, ok := value.(string); ok {
		text = s
	}
	switch binding.RenderAs {
	case "", "text":
		return text, nil
	case "json":
		return string(compact), nil
	case "json_pretty":
		var out bytes.Buffer
		if err := json.Indent(&out, compact, "", "  "); err != nil {
			return "", err
		}
		return out.String(), nil
	case "xml":
		tag := binding.WrapTag
		if tag == "" {
			tag = binding.Name
		}
		return "<" + tag + ">\n" + text + "\n</" + tag + ">", nil
	case "markdown":
		return "## " + binding.Name + "\n\n" + text, nil
	case "fenced":
		return "```" + binding.Lang + "\n" + text + "\n```", nil
	default:
		return "", fmt.Errorf("unsupported renderAs %q", binding.RenderAs)
	}
}

func truncateRendered(rendered string, maxBytes int) string {
	if len(rendered) <= maxBytes {
		return rendered
	}
	marker := fmt.Sprintf("…[truncated %d bytes]", len(rendered)-maxBytes)
	if len(marker) >= maxBytes {
		return truncateUTF8(marker, maxBytes)
	}
	prefixBytes := maxBytes - len(marker)
	for {
		prefix := truncateUTF8(rendered, prefixBytes)
		candidate := prefix + fmt.Sprintf("…[truncated %d bytes]", len(rendered)-len(prefix))
		if len(candidate) <= maxBytes || prefixBytes == 0 {
			return candidate
		}
		prefixBytes--
	}
}

func truncateUTF8(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	end := maxBytes
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end]
}

func selectBinding(binding domain.WorkflowInputBinding, ctx BindingContext) ([]selectedBindingValue, error) {
	if binding.Source == domain.WorkflowBindingInput {
		var value any
		if err := json.Unmarshal(ctx.Input, &value); err != nil {
			return nil, err
		}
		selected, ok := selectPath(value, binding.Selector)
		if !ok {
			return nil, nil
		}
		return []selectedBindingValue{{Value: selected}}, nil
	}
	kind := map[domain.WorkflowBindingSource]domain.WorkflowJournalKind{domain.WorkflowBindingAttempts: domain.WorkflowJournalAttempt, domain.WorkflowBindingRunResult: domain.WorkflowJournalRunResult, domain.WorkflowBindingStructured: domain.WorkflowJournalStructured, domain.WorkflowBindingHandoff: domain.WorkflowJournalHandoff, domain.WorkflowBindingSignal: domain.WorkflowJournalSignal, domain.WorkflowBindingCounter: domain.WorkflowJournalCounter, domain.WorkflowBindingChild: domain.WorkflowJournalChild}[binding.Source]
	node, path := parseSelector(binding.Selector)
	entries := append([]*domain.WorkflowJournalEntry(nil), ctx.Journal...)
	if binding.Order == "desc" {
		sort.SliceStable(entries, func(i, j int) bool { return entries[i].Sequence > entries[j].Sequence })
	}
	var values []selectedBindingValue
	for _, entry := range entries {
		if entry.Kind != kind || node != "" && entry.NodeID != node {
			continue
		}
		var value any
		if err := json.Unmarshal(entry.Payload, &value); err != nil {
			return nil, err
		}
		selected, ok := selectPath(value, path)
		if !ok {
			continue
		}
		values = append(values, selectedBindingValue{Value: selected, NodeID: entry.NodeID, Sequence: entry.Sequence})
		if len(values) >= binding.Limit {
			break
		}
	}
	return values, nil
}

func renderXMLList(binding domain.WorkflowInputBinding, selected []selectedBindingValue) (string, []BindingDiagnostic, error) {
	diagnostics := []BindingDiagnostic{}
	tag := binding.WrapTag
	if tag == "" {
		tag = binding.Name
	}
	itemTag := binding.ItemTag
	if itemTag == "" {
		itemTag = strings.TrimSuffix(binding.Name, "s")
		if itemTag == binding.Name || itemTag == "" {
			itemTag = "item"
		}
	}
	items := make([]string, len(selected))
	for i, item := range selected {
		content, err := renderBinding(domain.WorkflowInputBinding{Name: binding.Name, RenderAs: "text"}, item.Value)
		if err != nil {
			return "", nil, err
		}
		if binding.ItemMaxBytes > 0 && len(content) > binding.ItemMaxBytes {
			diagnostics = append(diagnostics, BindingDiagnostic{Binding: binding.Name, Code: "binding_item_truncated", DroppedBytes: len(content) - binding.ItemMaxBytes})
			content = truncateRendered(content, binding.ItemMaxBytes)
		}
		items[i] = fmt.Sprintf("<%s nodeId=%q sequence=%q ordinal=%q>\n%s\n</%s>", itemTag, item.NodeID, fmt.Sprint(item.Sequence), fmt.Sprint(i+1), content, itemTag)
	}
	indices := packXMLItems(binding, tag, items)
	rendered := formatXMLList(tag, items, indices)
	if len(rendered) > binding.MaxBytes {
		return "", nil, fmt.Errorf("binding %s maxBytes is too small for its XML list envelope", binding.Name)
	}
	if len(indices) < len(items) {
		diagnostics = append(diagnostics, BindingDiagnostic{Binding: binding.Name, Code: "binding_items_evicted", ElidedItems: len(items) - len(indices)})
	}
	return rendered, diagnostics, nil
}

func packXMLItems(binding domain.WorkflowInputBinding, tag string, items []string) []int {
	if len(formatXMLList(tag, items, nil)) > binding.MaxBytes {
		return nil
	}
	try := func(indices []int) bool { return len(formatXMLList(tag, items, indices)) <= binding.MaxBytes }
	chosen := []int{}
	switch binding.EvictionPolicy {
	case "keep_first":
		for i := range items {
			candidate := append(append([]int(nil), chosen...), i)
			if try(candidate) {
				chosen = candidate
			}
		}
	case "keep_ends":
		head := min(binding.KeepFirst, len(items))
		for i := 0; i < head; i++ {
			if try(append(chosen, i)) {
				chosen = append(chosen, i)
			}
		}
		for i := len(items) - 1; i >= head; i-- {
			candidate := append([]int{i}, chosen...)
			sort.Ints(candidate)
			if try(candidate) {
				chosen = candidate
			}
		}
	default: // keep_last
		for i := len(items) - 1; i >= 0; i-- {
			candidate := append([]int{i}, chosen...)
			if try(candidate) {
				chosen = candidate
			}
		}
	}
	sort.Ints(chosen)
	return chosen
}

func formatXMLList(tag string, items []string, indices []int) string {
	shown := make(map[int]bool, len(indices))
	for _, index := range indices {
		shown[index] = true
	}
	var body []string
	elided := 0
	for i, item := range items {
		if shown[i] {
			if elided > 0 {
				body = append(body, fmt.Sprintf("<elided count=%q reason=\"byte budget\"/>", fmt.Sprint(elided)))
				elided = 0
			}
			body = append(body, item)
		} else {
			elided++
		}
	}
	if elided > 0 {
		body = append(body, fmt.Sprintf("<elided count=%q reason=\"byte budget\"/>", fmt.Sprint(elided)))
	}
	return fmt.Sprintf("<%s count=%q showing=%q>\n%s\n</%s>", tag, fmt.Sprint(len(items)), fmt.Sprint(len(indices)), strings.Join(body, "\n"), tag)
}

func parseSelector(selector string) (node, path string) {
	selector = strings.TrimSpace(selector)
	if strings.HasPrefix(selector, "node=") {
		parts := strings.SplitN(selector, ";", 2)
		node = strings.TrimPrefix(parts[0], "node=")
		if len(parts) == 2 {
			path = parts[1]
		}
		return
	}
	return "", selector
}

func selectPath(value any, path string) (any, bool) {
	path = strings.TrimSpace(path)
	if path == "" || path == "$" {
		return value, true
	}
	if !strings.HasPrefix(path, "$.") {
		return nil, false
	}
	current := value
	for _, part := range strings.Split(strings.TrimPrefix(path, "$."), ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func RenderPrompt(source string, values map[string]any) (string, error) {
	parsed, err := workflowexpr.ParsePrompt(source)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := parsed.Execute(&out, values); err != nil {
		return "", err
	}
	if out.Len() > MaxRenderedPromptBytes {
		return "", fmt.Errorf("rendered prompt exceeds %d bytes", MaxRenderedPromptBytes)
	}
	return out.String(), nil
}
