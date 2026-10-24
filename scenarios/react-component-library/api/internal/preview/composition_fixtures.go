package preview

import (
	"encoding/json"
	"fmt"
	"strings"
)

type CompositionFixture struct {
	Target, Asset, Version, State, Field string
	Prop                                 []string
}
type PreparedComposition struct {
	Composition Composition
	Bindings    map[string]any
	Fixtures    []DeterministicFixturePayload
	Gaps        []CompositionGap
}

// PrepareComposition resolves only declared deterministic fixture families.
// Function bindings use the existing story harness's inert $handler vocabulary.
func PrepareComposition(c Composition, bindings map[string]any, fixtures []CompositionFixture) (PreparedComposition, error) {
	out := PreparedComposition{Composition: c}
	if err := validateComposition(c); err != nil {
		return out, err
	}
	raw, err := json.Marshal(bindings)
	if err != nil {
		return out, err
	}
	if len(raw) > 512*1024 {
		return out, fmt.Errorf("composition bindings exceed 512 KiB")
	}
	if err := json.Unmarshal(raw, &out.Bindings); err != nil {
		return out, err
	}
	if out.Bindings == nil {
		out.Bindings = map[string]any{}
	}
	allowed := map[string]bool{"$template": true, "$labels": true, "$preview": true}
	for _, region := range c.Regions {
		allowed[region.ID] = true
	}
	for key := range out.Bindings {
		if !allowed[key] {
			return out, fmt.Errorf("binding target %q is not declared", key)
		}
	}
	if len(fixtures) > 64 {
		return out, fmt.Errorf("composition exceeds 64 fixture bindings")
	}
	for _, reference := range fixtures {
		if !allowed[reference.Target] || (reference.Target == "$labels" || reference.Target == "$preview") {
			return out, fmt.Errorf("fixture target %q is not an asset binding", reference.Target)
		}
		if len(reference.Prop) == 0 || len(reference.Prop) > 4 {
			return out, fmt.Errorf("fixture prop path must contain one to four fields")
		}
		for _, part := range reference.Prop {
			if !compositionIdentifier.MatchString(part) || unsafeCompositionKey(part) {
				return out, fmt.Errorf("invalid fixture prop path")
			}
		}
		fixture, err := ResolveDeterministicFixture(reference.Asset, reference.Version, reference.State)
		if err != nil {
			return out, err
		}
		encoded, err := json.Marshal(fixture)
		if err != nil {
			return out, err
		}
		var fields map[string]any
		if err := json.Unmarshal(encoded, &fields); err != nil {
			return out, err
		}
		switch reference.Field {
		case "records", "series", "nodes", "error", "clock", "seed":
		default:
			return out, fmt.Errorf("unsupported fixture field %q", reference.Field)
		}
		value := fields[reference.Field]
		if value == nil {
			if reference.Field == "records" || reference.Field == "series" || reference.Field == "nodes" {
				value = []any{}
			} else {
				value = ""
			}
		}
		props, ok := out.Bindings[reference.Target].(map[string]any)
		if !ok {
			if out.Bindings[reference.Target] != nil {
				return out, fmt.Errorf("asset bindings must be objects")
			}
			props = map[string]any{}
			out.Bindings[reference.Target] = props
		}
		for _, part := range reference.Prop[:len(reference.Prop)-1] {
			child, ok := props[part].(map[string]any)
			if !ok {
				if props[part] != nil {
					return out, fmt.Errorf("fixture prop path overlaps a scalar binding")
				}
				child = map[string]any{}
				props[part] = child
			}
			props = child
		}
		leaf := reference.Prop[len(reference.Prop)-1]
		if _, exists := props[leaf]; exists {
			return out, fmt.Errorf("fixture overwrites existing prop %q", leaf)
		}
		props[leaf] = value
		out.Fixtures = append(out.Fixtures, fixture)
	}
	if _, ok := out.Bindings["$template"].(map[string]any); !ok {
		return out, fmt.Errorf("template fixture bindings are required")
	}
	labels, ok := out.Bindings["$labels"].(map[string]any)
	if !ok {
		return out, fmt.Errorf("localized missing and failed labels are required")
	}
	for _, key := range []string{"missing", "failed"} {
		value, ok := labels[key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return out, fmt.Errorf("localized label %s is required", key)
		}
	}
	interaction, interactive := out.Bindings["$preview"]
	delete(out.Bindings, "$preview")
	if interactive {
		if err := validatePreviewInteraction(interaction, allowed); err != nil {
			return out, err
		}
	}
	if err := validateFixtureValues(out.Bindings, 0); err != nil {
		return out, err
	}
	if interactive {
		out.Bindings["$preview"] = interaction
	}
	out.Composition.Regions = append([]CompositionRegion(nil), c.Regions...)
	for i, region := range c.Regions {
		if region.Asset == nil {
			continue
		}
		if _, ok := out.Bindings[region.ID].(map[string]any); !ok {
			out.Gaps = append(out.Gaps, CompositionGap{region.ID, "fixture_missing", "Region has no deterministic prop bindings", region.Required})
			out.Composition.Regions[i].Asset = nil
		}
	}
	return out, nil
}
func unsafeCompositionKey(key string) bool {
	return key == "__proto__" || key == "constructor" || key == "prototype"
}
func validateFixtureValues(value any, depth int) error {
	if depth > 16 {
		return fmt.Errorf("fixture bindings exceed nesting limit")
	}
	switch node := value.(type) {
	case []any:
		for _, child := range node {
			if err := validateFixtureValues(child, depth+1); err != nil {
				return err
			}
		}
	case map[string]any:
		if marker, ok := node["$handler"]; ok {
			if name, valid := marker.(string); !valid || name == "" || len(node) != 1 {
				return fmt.Errorf("action fixture must contain only a named $handler")
			}
			return nil
		}
		for key, child := range node {
			if unsafeCompositionKey(key) || key == "dangerouslySetInnerHTML" || key == "srcDoc" {
				return fmt.Errorf("unsupported fixture prop %q", key)
			}
			if strings.HasPrefix(key, "$") && key != "$template" && key != "$labels" {
				return fmt.Errorf("unsupported fixture directive %q", key)
			}
			if strings.HasPrefix(key, "on") && len(key) > 2 && key[2] >= 'A' && key[2] <= 'Z' {
				handler, ok := child.(map[string]any)
				if !ok || handler["$handler"] == nil {
					return fmt.Errorf("action prop %s requires an inert handler fixture", key)
				}
			}
			// Component action props also describe decisions (ApprovalPrompt). Native
			// form destinations are blocked by the composition document form-action CSP,
			// rather than guessing DOM semantics from this component prop name.
			if key == "href" || key == "formAction" {
				if s, ok := child.(string); ok && s != "" && !strings.HasPrefix(s, "#") {
					return fmt.Errorf("fixture navigation must remain within the preview")
				}
			}
			if err := validateFixtureValues(child, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// Preview interactions are finite fixture transitions, never executable code or
// scenario actions. Overrides are shallow prop merges per declared asset target.
func validatePreviewInteraction(value any, allowed map[string]bool) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var spec struct {
		Initial string                               `json:"initial"`
		States  map[string]map[string]map[string]any `json:"states"`
		Actions map[string][]struct {
			Result   *bool    `json:"result,omitempty"`
			State    string   `json:"state"`
			Argument []string `json:"argument,omitempty"`
			Equals   any      `json:"equals,omitempty"`
			Set      []struct {
				Scope    string    `json:"scope,omitempty"`
				Target   string    `json:"target"`
				Prop     string    `json:"prop"`
				Argument *[]string `json:"argument,omitempty"`
				Value    any       `json:"value,omitempty"`
			} `json:"set,omitempty"`
		} `json:"actions"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&spec); err != nil {
		return fmt.Errorf("invalid preview interaction: %w", err)
	}
	if len(spec.States) == 0 || len(spec.States) > 32 || len(spec.Actions) > 64 {
		return fmt.Errorf("preview interaction exceeds state or action bounds")
	}
	if _, ok := spec.States[spec.Initial]; !ok {
		return fmt.Errorf("preview initial state is not declared")
	}
	for state, targets := range spec.States {
		if targets == nil {
			return fmt.Errorf("preview state must be an object")
		}
		if !compositionIdentifier.MatchString(state) || unsafeCompositionKey(state) {
			return fmt.Errorf("invalid preview state name")
		}
		for target, props := range targets {
			if !allowed[target] || target == "$labels" || target == "$preview" {
				return fmt.Errorf("preview state targets undeclared asset %q", target)
			}
			if props == nil {
				return fmt.Errorf("preview state props must be objects")
			}
			if err := validateFixtureValues(props, 0); err != nil {
				return err
			}
		}
	}
	for action, rules := range spec.Actions {
		if !compositionIdentifier.MatchString(action) || unsafeCompositionKey(action) || len(rules) == 0 || len(rules) > 32 {
			return fmt.Errorf("invalid preview action")
		}
		for i, rule := range rules {
			if len(rule.Set) > 32 {
				return fmt.Errorf("preview action exceeds 32 assignments")
			}
			for _, assignment := range rule.Set {
				if assignment.Scope != "" && assignment.Scope != "state" {
					return fmt.Errorf("preview assignment scope must be state or omitted")
				}
				if !allowed[assignment.Target] || assignment.Target == "$labels" || assignment.Target == "$preview" {
					return fmt.Errorf("preview assignment target is not an asset")
				}
				prop := assignment.Prop
				if !compositionIdentifier.MatchString(prop) || unsafeCompositionKey(prop) || strings.HasPrefix(prop, "on") {
					return fmt.Errorf("invalid preview assignment prop")
				}
				switch strings.ToLower(prop) {
				case "href", "src", "action", "formaction", "srcdoc", "dangerouslysetinnerhtml":
					return fmt.Errorf("preview assignments cannot navigate or inject markup")
				}
				if assignment.Argument != nil {
					if len(*assignment.Argument) > 4 || assignment.Value != nil {
						return fmt.Errorf("invalid preview assignment source")
					}
					for _, field := range *assignment.Argument {
						if !compositionIdentifier.MatchString(field) || unsafeCompositionKey(field) {
							return fmt.Errorf("invalid preview assignment argument path")
						}
					}
				} else {
					if value, ok := assignment.Value.(string); ok && len(value) > 4096 {
						return fmt.Errorf("preview assigned string exceeds 4096 bytes")
					}
					switch assignment.Value.(type) {
					case nil, string, bool, float64:
					default:
						return fmt.Errorf("preview assigned value must be scalar")
					}
				}
			}

			if _, ok := spec.States[rule.State]; !ok {
				return fmt.Errorf("preview transition state is not declared")
			}
			if len(rule.Argument) > 4 {
				return fmt.Errorf("preview argument path exceeds four fields")
			}
			for _, field := range rule.Argument {
				if !compositionIdentifier.MatchString(field) || unsafeCompositionKey(field) {
					return fmt.Errorf("invalid preview argument field")
				}
			}
			if len(rule.Argument) == 0 && rule.Equals == nil && i != len(rules)-1 {
				return fmt.Errorf("unconditional preview transition must be last")
			}
			switch rule.Equals.(type) {
			case nil, string, bool, float64:
			default:
				return fmt.Errorf("preview argument comparison must be a scalar")
			}
		}
	}
	return nil
}

// SelectCompositionPreviewState makes a declared state the reproducible initial
// state of a render without mutating the candidate or the prepared input.
func SelectCompositionPreviewState(p PreparedComposition, state string) (PreparedComposition, error) {
	if state == "" {
		return p, nil
	}
	graph, ok := p.Bindings["$preview"].(map[string]any)
	if !ok {
		return p, fmt.Errorf("candidate has no preview states")
	}
	states, ok := graph["states"].(map[string]any)
	if !ok || states[state] == nil {
		return p, fmt.Errorf("preview state %q is not declared", state)
	}
	bindings := make(map[string]any, len(p.Bindings))
	for key, value := range p.Bindings {
		bindings[key] = value
	}
	selected := make(map[string]any, len(graph))
	for key, value := range graph {
		selected[key] = value
	}
	selected["initial"] = state
	bindings["$preview"] = selected
	p.Bindings = bindings
	return p, nil
}
