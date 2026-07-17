package workflowruntime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/workflowexpr"
)

const MaxRenderedPromptBytes = 64 << 10

type BindingContext struct {
	Input   json.RawMessage
	Journal []*domain.WorkflowJournalEntry
}

func EvaluateBindings(bindings []domain.WorkflowInputBinding, ctx BindingContext) (map[string]any, error) {
	out := make(map[string]any, len(bindings))
	for _, binding := range bindings {
		values, err := selectBinding(binding, ctx)
		if err != nil {
			return nil, fmt.Errorf("binding %s: %w", binding.Name, err)
		}
		if len(values) == 0 {
			switch binding.MissingPolicy {
			case "omit":
				continue
			case "null":
				out[binding.Name] = nil
			default:
				return nil, fmt.Errorf("binding %s selected no values", binding.Name)
			}
			continue
		}
		var value any
		if len(values) == 1 {
			value = values[0]
		} else {
			value = values
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		if len(encoded) > binding.MaxBytes {
			return nil, fmt.Errorf("binding %s exceeds %d bytes", binding.Name, binding.MaxBytes)
		}
		if binding.RenderAs == "text" {
			if s, ok := value.(string); ok {
				out[binding.Name] = s
			} else {
				out[binding.Name] = string(encoded)
			}
		} else {
			out[binding.Name] = value
		}
	}
	return out, nil
}

func selectBinding(binding domain.WorkflowInputBinding, ctx BindingContext) ([]any, error) {
	if binding.Source == domain.WorkflowBindingInput {
		var value any
		if err := json.Unmarshal(ctx.Input, &value); err != nil {
			return nil, err
		}
		selected, ok := selectPath(value, binding.Selector)
		if !ok {
			return nil, nil
		}
		return []any{selected}, nil
	}
	kind := map[domain.WorkflowBindingSource]domain.WorkflowJournalKind{domain.WorkflowBindingAttempts: domain.WorkflowJournalAttempt, domain.WorkflowBindingRunResult: domain.WorkflowJournalRunResult, domain.WorkflowBindingStructured: domain.WorkflowJournalStructured, domain.WorkflowBindingHandoff: domain.WorkflowJournalHandoff, domain.WorkflowBindingSignal: domain.WorkflowJournalSignal, domain.WorkflowBindingCounter: domain.WorkflowJournalCounter, domain.WorkflowBindingChild: domain.WorkflowJournalChild}[binding.Source]
	node, path := parseSelector(binding.Selector)
	entries := append([]*domain.WorkflowJournalEntry(nil), ctx.Journal...)
	if binding.Order == "desc" {
		sort.SliceStable(entries, func(i, j int) bool { return entries[i].Sequence > entries[j].Sequence })
	}
	var values []any
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
		values = append(values, selected)
		if len(values) >= binding.Limit {
			break
		}
	}
	return values, nil
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
