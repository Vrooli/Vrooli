package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"react-vite-temporal-model/internal/spec"
)

const SchemaVersion = spec.SchemaVersion

const (
	ReplayKindGoTest = spec.ReplayKindGoTest
	ReplayKindVitest = spec.ReplayKindVitest
)

type Contract struct {
	SchemaVersion      int                `json:"schemaVersion"`
	FlowID             string             `json:"flowId"`
	Domain             string             `json:"domain"`
	Description        string             `json:"description"`
	Model              Model              `json:"model"`
	Outputs            Outputs            `json:"outputs"`
	States             []State            `json:"states"`
	Events             []Event            `json:"events"`
	TransitionDefaults TransitionDefaults `json:"transitionDefaults"`
	Transitions        []Transition       `json:"transitions"`
	Invariants         []Invariant        `json:"invariants"`
	Traces             []Trace            `json:"traces"`
	Runtime            Runtime            `json:"runtime,omitempty"`
	Replay             Replay             `json:"replay"`
	ContractPath       string             `json:"-"`
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

type Outputs struct {
	ModelPath        string `json:"modelPath"`
	ArtifactPath     string `json:"artifactPath"`
	DeclarationsPath string `json:"declarationsPath"`
	ReplayHelperPath string `json:"replayHelperPath,omitempty"`
	ReplayTestPath   string `json:"replayTestPath"`
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
	Kind          string           `json:"kind"`
	HelperPath    string           `json:"helperPath,omitempty"`
	TestPath      string           `json:"testPath"`
	FixtureModule string           `json:"fixtureModule,omitempty"`
	FixtureExport string           `json:"fixtureExport,omitempty"`
	Transition    ReplayTransition `json:"transition"`
}

type ReplayTransition struct {
	Module         string `json:"module,omitempty"`
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

func LoadRaw(path string, relPath string) (Contract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
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
	return c, nil
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

func ValidateReplayFixture(c Contract, root string) error {
	return ValidateReplayFixturePaths(root, c.FlowID, c.Replay.Kind, c.Outputs.ReplayTestPath, c.Replay.FixtureModule, c.Replay.FixtureExport)
}

func ValidateReplayFixturePaths(root string, flowID string, replayKind string, replayTestPath string, fixtureModule string, fixtureExport string) error {
	var errs []string
	if replayKind != ReplayKindVitest {
		return nil
	}
	path, err := ResolveTypeScriptImport(replayTestPath, fixtureModule)
	if err != nil {
		return fmt.Errorf("invalid replay fixture module for %s: %w", flowID, err)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		errs = append(errs, fmt.Sprintf("replay fixture module %s for %s is missing or unreadable: %v", path, flowID, err))
	} else if !hasTypeScriptExport(data, fixtureExport) {
		errs = append(errs, fmt.Sprintf("replay fixture module %s for %s does not export %s", path, flowID, fixtureExport))
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("invalid replay fixture for %s:\n  - %s", flowID, strings.Join(errs, "\n  - "))
	}
	return nil
}

func ResolveTypeScriptImport(fromPath string, module string) (string, error) {
	if !strings.HasPrefix(module, ".") {
		return "", fmt.Errorf("module must be relative")
	}
	base := filepath.Dir(filepath.ToSlash(fromPath))
	clean := filepath.ToSlash(filepath.Clean(filepath.Join(base, filepath.FromSlash(module))))
	if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
		return "", fmt.Errorf("module must resolve inside the scenario root")
	}
	return clean + ".ts", nil
}

func hasTypeScriptExport(data []byte, name string) bool {
	quoted := regexp.QuoteMeta(name)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\s*export\s+(const|let|var|function|class)\s+` + quoted + `\b`),
		regexp.MustCompile(`(?m)^\s*export\s*\{[^}]*\b` + quoted + `\b[^}]*\}`),
	}
	for _, pattern := range patterns {
		if pattern.Match(data) {
			return true
		}
	}
	return false
}
