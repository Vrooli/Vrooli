// Package componenttests owns the declarative, version-pinned contract used to
// validate catalog assets. It intentionally contains no executable source,
// shell command, module path, or example setup field: those values would turn
// a catalog declaration into arbitrary code execution.
package componenttests

import (
	"encoding/json"
	"fmt"
	"strings"
)

const ContractFileName = "test-contract.json"

type Contract struct {
	SchemaVersion string         `json:"schemaVersion"`
	Examples      []ExampleTrace `json:"examples"`
	Fixtures      []HookFixture  `json:"fixtures"`
}

type ExampleTrace struct {
	Example    string      `json:"example"`
	Actions    []Action    `json:"actions"`
	Assertions []Assertion `json:"assertions"`
	Claims     []string    `json:"claims"`
}

type HookFixture struct {
	Name       string         `json:"name"`
	Inputs     map[string]any `json:"inputs"`
	Assertions []Assertion    `json:"assertions"`
	Claims     []string       `json:"claims"`
}

type Action struct {
	Kind     string `json:"kind"`
	Target   string `json:"target,omitempty"`
	Key      string `json:"key,omitempty"`
	Duration int    `json:"durationMs,omitempty"`
}

type Assertion struct {
	Kind      string `json:"kind"`
	Role      string `json:"role,omitempty"`
	Name      string `json:"name,omitempty"`
	Target    string `json:"target,omitempty"`
	Value     string `json:"value,omitempty"`
	Attribute string `json:"attribute,omitempty"`
}

// ValidationError identifies the authored field instead of producing a vague
// runner failure. Its Code is stable for CLI, UI, and tests.
type ValidationError struct {
	Code, Field, Detail string
}

func (e ValidationError) Error() string {
	if e.Field == "" {
		return e.Code + ": " + e.Detail
	}
	return fmt.Sprintf("%s at %s: %s", e.Code, e.Field, e.Detail)
}

func ParseContract(data []byte, assetKind string) (Contract, error) {
	var contract Contract
	if err := json.Unmarshal(data, &contract); err != nil {
		return Contract{}, ValidationError{Code: "invalid_contract_json", Detail: err.Error()}
	}
	if err := contract.Validate(assetKind); err != nil {
		return Contract{}, err
	}
	return contract, nil
}

func (c Contract) Validate(assetKind string) error {
	if c.SchemaVersion != "1" {
		return ValidationError{Code: "unsupported_contract_version", Field: "schemaVersion", Detail: "must be \"1\""}
	}
	if assetKind != "component" && assetKind != "hook" {
		return ValidationError{Code: "invalid_asset_kind", Detail: assetKind}
	}
	if assetKind == "component" && len(c.Fixtures) > 0 {
		return ValidationError{Code: "hook_fixture_on_component", Field: "fixtures", Detail: "fixtures are only valid for hooks"}
	}
	if assetKind == "hook" && len(c.Examples) > 0 {
		return ValidationError{Code: "example_on_hook", Field: "examples", Detail: "examples are only valid for hooks"}
	}
	if assetKind == "component" && len(c.Examples) == 0 {
		return ValidationError{Code: "missing_example_trace", Field: "examples", Detail: "at least one trace is required"}
	}
	if assetKind == "hook" && len(c.Fixtures) == 0 {
		return ValidationError{Code: "missing_hook_fixture", Field: "fixtures", Detail: "at least one fixture is required"}
	}
	seen := map[string]bool{}
	for i, trace := range c.Examples {
		if err := validName(trace.Example, fmt.Sprintf("examples[%d].example", i)); err != nil {
			return err
		}
		if seen[trace.Example] {
			return ValidationError{Code: "duplicate_trace", Field: fmt.Sprintf("examples[%d].example", i), Detail: trace.Example}
		}
		seen[trace.Example] = true
		for j, action := range trace.Actions {
			if err := action.validate(fmt.Sprintf("examples[%d].actions[%d]", i, j)); err != nil {
				return err
			}
		}
		if err := validateAssertions(trace.Assertions, fmt.Sprintf("examples[%d].assertions", i)); err != nil {
			return err
		}
		if err := validateClaims(trace.Claims, fmt.Sprintf("examples[%d].claims", i)); err != nil {
			return err
		}
	}
	seen = map[string]bool{}
	for i, fixture := range c.Fixtures {
		if err := validName(fixture.Name, fmt.Sprintf("fixtures[%d].name", i)); err != nil {
			return err
		}
		if seen[fixture.Name] {
			return ValidationError{Code: "duplicate_fixture", Field: fmt.Sprintf("fixtures[%d].name", i), Detail: fixture.Name}
		}
		seen[fixture.Name] = true
		if err := validateAssertions(fixture.Assertions, fmt.Sprintf("fixtures[%d].assertions", i)); err != nil {
			return err
		}
		if err := validateClaims(fixture.Claims, fmt.Sprintf("fixtures[%d].claims", i)); err != nil {
			return err
		}
	}
	return nil
}

func validateClaims(claims []string, field string) error {
	seen := map[string]bool{}
	for i, claim := range claims {
		if err := validName(claim, fmt.Sprintf("%s[%d]", field, i)); err != nil {
			return ValidationError{Code: "invalid_claim_reference", Field: fmt.Sprintf("%s[%d]", field, i), Detail: err.Error()}
		}
		if seen[claim] {
			return ValidationError{Code: "duplicate_claim_reference", Field: fmt.Sprintf("%s[%d]", field, i), Detail: claim}
		}
		seen[claim] = true
	}
	return nil
}

func validName(value, field string) error {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "/\\") {
		return ValidationError{Code: "invalid_trace_name", Field: field, Detail: "must be a non-empty logical name"}
	}
	return nil
}

func (a Action) validate(field string) error {
	switch a.Kind {
	case "click":
		if a.Target == "" {
			return ValidationError{Code: "missing_action_target", Field: field, Detail: "click requires target"}
		}
		if err := validName(a.Target, field+".target"); err != nil {
			return err
		}
	case "key":
		if a.Target == "" || a.Key == "" {
			return ValidationError{Code: "invalid_key_action", Field: field, Detail: "key requires target and key"}
		}
		if err := validName(a.Target, field+".target"); err != nil {
			return err
		}
	case "wait":
		if a.Duration < 0 || a.Duration > 5000 {
			return ValidationError{Code: "invalid_wait_duration", Field: field, Detail: "must be between 0 and 5000ms"}
		}
	default:
		return ValidationError{Code: "unsupported_action", Field: field, Detail: a.Kind}
	}
	return nil
}

func validateAssertions(assertions []Assertion, field string) error {
	if len(assertions) == 0 {
		return ValidationError{Code: "missing_assertion", Field: field, Detail: "at least one assertion is required"}
	}
	for i, assertion := range assertions {
		at := fmt.Sprintf("%s[%d]", field, i)
		switch assertion.Kind {
		case "role":
			if assertion.Role == "" || assertion.Name == "" {
				return ValidationError{Code: "invalid_role_assertion", Field: at, Detail: "role and name are required"}
			}
		case "text":
			if assertion.Value == "" {
				return ValidationError{Code: "invalid_text_assertion", Field: at, Detail: "value is required"}
			}
		case "attribute":
			if assertion.Target == "" || assertion.Attribute == "" {
				return ValidationError{Code: "invalid_attribute_assertion", Field: at, Detail: "target and attribute are required"}
			}
		case "state":
			if assertion.Target == "" || assertion.Value == "" {
				return ValidationError{Code: "invalid_state_assertion", Field: at, Detail: "target and value are required"}
			}
		default:
			return ValidationError{Code: "unsupported_assertion", Field: at, Detail: assertion.Kind}
		}
	}
	return nil
}
