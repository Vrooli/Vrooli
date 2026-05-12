package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"react-vite-temporal-model/internal/layout"
	"react-vite-temporal-model/internal/spec"
)

const SchemaVersion = spec.SchemaVersion

type Contract struct {
	SchemaVersion      int                `json:"schemaVersion"`
	FlowID             string             `json:"flowId"`
	Domain             string             `json:"domain"`
	Description        string             `json:"description"`
	Model              Model              `json:"model"`
	States             []State            `json:"states"`
	Events             []Event            `json:"events"`
	TransitionDefaults TransitionDefaults `json:"transitionDefaults"`
	Transitions        []Transition       `json:"transitions"`
	Invariants         []Invariant        `json:"invariants"`
	Traces             []Trace            `json:"traces"`
	Runtime            Runtime            `json:"runtime"`
	Replay             Replay             `json:"replay"`
	ContractPath       string             `json:"-"`
	Layout             layout.Layout      `json:"-"`
}

type Model struct {
	Module     string `json:"module"`
	Seed       string `json:"seed"`
	MaxSteps   int    `json:"maxSteps"`
	TraceCount int    `json:"traceCount"`
	Verify     Verify `json:"verify"`
}

type Verify struct {
	Invariants []string `json:"invariants"`
}

type Runtime struct {
	Go              *GoRuntime         `json:"go,omitempty"`
	TypeScript      *TypeScriptRuntime `json:"typescript,omitempty"`
	SideEffects     []string           `json:"sideEffects,omitempty"`
	StaleCompletion string             `json:"staleCompletion,omitempty"`
}

type GoRuntime struct {
	Package        string `json:"package"`
	StatusType     string `json:"statusType"`
	EventType      string `json:"eventType"`
	ConstantPrefix string `json:"constantPrefix"`
}

type TypeScriptRuntime struct {
	StatusType             string                       `json:"statusType"`
	EventType              string                       `json:"eventType"`
	StatusesConst          string                       `json:"statusesConst"`
	EventsConst            string                       `json:"eventsConst"`
	FormalExpectationConst string                       `json:"formalExpectationConst"`
	StateUnionType         string                       `json:"stateUnionType,omitempty"`
	EventUnionType         string                       `json:"eventUnionType,omitempty"`
	PayloadTypes           map[string]string            `json:"payloadTypes,omitempty"`
	StateVariants          map[string]map[string]string `json:"stateVariants,omitempty"`
	EventVariants          map[string]map[string]string `json:"eventVariants,omitempty"`
}

type State struct {
	ID       string `json:"id"`
	Quint    string `json:"quint"`
	Initial  bool   `json:"initial,omitempty"`
	Terminal bool   `json:"terminal,omitempty"`
}

type Event struct {
	ID    string `json:"id"`
	Quint string `json:"quint"`
}

type TransitionDefaults struct {
	Invalid  *DefaultTransition `json:"invalid,omitempty"`
	Terminal *DefaultTransition `json:"terminal,omitempty"`
}

type DefaultTransition struct {
	To        string `json:"to"`
	WantError bool   `json:"wantError"`
}

type Transition struct {
	From      StringList `json:"from"`
	Event     StringList `json:"event"`
	To        string     `json:"to"`
	WantError *bool      `json:"wantError,omitempty"`
}

type Invariant struct {
	ID          string `json:"id"`
	Quint       string `json:"quint"`
	Description string `json:"description"`
	Expression  string `json:"expression,omitempty"`
}

type Trace struct {
	Name    string      `json:"name"`
	Initial string      `json:"initial"`
	Steps   []TraceStep `json:"steps"`
}

type TraceStep struct {
	Event     string `json:"event"`
	Want      string `json:"want"`
	WantError bool   `json:"wantError"`
}

type Replay struct {
	Transition ReplayTransition `json:"transition"`
}

type ReplayTransition struct {
	Function       string `json:"function"`
	StateType      string `json:"stateType,omitempty"`
	StatusField    string `json:"statusField,omitempty"`
	StatusAccessor string `json:"statusAccessor,omitempty"`
}

type StringList []string

func (s *StringList) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*s = []string{single}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*s = many
	return nil
}

// Language returns the runtime language declared by the contract.
func (c Contract) Language() (layout.Language, error) {
	if c.Runtime.Go != nil && c.Runtime.TypeScript != nil {
		return "", fmt.Errorf("contract %s declares both go and typescript runtimes", c.FlowID)
	}
	if c.Runtime.Go != nil {
		return layout.LanguageGo, nil
	}
	if c.Runtime.TypeScript != nil {
		return layout.LanguageTypeScript, nil
	}
	return "", fmt.Errorf("contract %s declares no runtime", c.FlowID)
}

func LoadRaw(path string, relPath string) (Contract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Contract{}, err
	}
	if err := checkSchemaVersion(data, relPath); err != nil {
		return Contract{}, err
	}
	if err := validateJSONSchema(data); err != nil {
		return Contract{}, fmt.Errorf("schema validation failed for %s:\n%s", relPath, err)
	}
	var c Contract
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil {
		return Contract{}, fmt.Errorf("parse %s: %w", relPath, err)
	}
	c.ContractPath = relPath
	lang, err := c.Language()
	if err != nil {
		return Contract{}, err
	}
	lay, err := layout.Derive(relPath, lang)
	if err != nil {
		return Contract{}, fmt.Errorf("derive layout for %s: %w", relPath, err)
	}
	c.Layout = lay
	return c, nil
}

// checkSchemaVersion runs before JSON-schema validation so that v5
// contracts get a clear migration message instead of a "const: 6"
// validation error.
func checkSchemaVersion(data []byte, relPath string) error {
	var probe struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil // let schema validator produce the real error
	}
	if probe.SchemaVersion == 5 {
		return fmt.Errorf("contract %s: schemaVersion 5 is no longer supported. Move this flow into a flow/ subdirectory (flow.json, transition.{ts,go}, fixtures.ts for TS, flow.test.{ts,go}) and bump schemaVersion to 6. See tools/temporal-model/README.md for the migration steps", relPath)
	}
	return nil
}

var (
	flowSchemaOnce sync.Once
	flowSchema     *jsonschema.Schema
	flowSchemaErr  error
)

func validateJSONSchema(data []byte) error {
	schema, err := compiledFlowSchema()
	if err != nil {
		return err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if err := schema.Validate(value); err != nil {
		return err
	}
	return nil
}

func compiledFlowSchema() (*jsonschema.Schema, error) {
	flowSchemaOnce.Do(func() {
		_, current, _, ok := runtime.Caller(0)
		if !ok {
			flowSchemaErr = fmt.Errorf("locate flow.schema.json: runtime caller unavailable")
			return
		}
		schemaPath := filepath.Join(filepath.Dir(current), "..", "..", "flow.schema.json")
		data, err := os.ReadFile(schemaPath)
		if err != nil {
			flowSchemaErr = fmt.Errorf("read flow.schema.json: %w", err)
			return
		}
		var schemaDoc any
		if err := json.Unmarshal(data, &schemaDoc); err != nil {
			flowSchemaErr = fmt.Errorf("parse flow.schema.json: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		const schemaURL = "https://vrooli.dev/schemas/react-vite/temporal-flow.json"
		if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
			flowSchemaErr = fmt.Errorf("load flow.schema.json: %w", err)
			return
		}
		flowSchema, flowSchemaErr = compiler.Compile(schemaURL)
	})
	return flowSchema, flowSchemaErr
}

// ValidateConventionalFiles verifies that the hand-authored files
// required by the v6 layout convention exist on disk for this flow.
// The check is structural: the lint pass in package lint verifies the
// file contents.
func ValidateConventionalFiles(root string, c Contract) error {
	var errs []string
	required := []string{c.Layout.TransitionPath, c.Layout.TestPath}
	if c.Layout.FixturesPath != "" {
		required = append(required, c.Layout.FixturesPath)
	}
	for _, rel := range required {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(abs); err != nil {
			errs = append(errs, fmt.Sprintf("%s is missing (expected by the flow/ convention)", rel))
		}
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf(
			"flow %s is missing hand-authored files:\n  - %s\nRun `temporal-model new %s --flow-id %s` to scaffold a fresh flow, or add the missing files manually.",
			c.FlowID,
			strings.Join(errs, "\n  - "),
			parentOfFlow(c.Layout.BaseDir),
			c.FlowID,
		)
	}
	return nil
}

func parentOfFlow(baseDir string) string {
	parent := filepath.Dir(baseDir)
	if parent == "." || parent == "" {
		return ""
	}
	return parent
}
