// Package contract parses navigation.json files and validates them
// against the embedded navigation JSON schema. It owns the raw on-disk
// contract type; structural cross-reference validation lives one layer
// up in the compile package.
package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"flow-verifier/internal/flows/schemas"
)

// SchemaVersion is the on-disk schemaVersion every navigation.json must
// declare. Bumped on breaking schema changes.
const SchemaVersion = 1

// Kind is the wire-stable identifier for navigation contracts.
const Kind = "navigation"

// Contract is the raw, schema-validated navigation graph.
type Contract struct {
	SchemaVersion          int                     `json:"schemaVersion"`
	Kind                   string                  `json:"kind"`
	FlowID                 string                  `json:"flowId"`
	Domain                 string                  `json:"domain"`
	Description            string                  `json:"description"`
	Contexts               map[string]Context      `json:"contexts"`
	Routes                 []Route                 `json:"routes"`
	Containers             []Container             `json:"containers,omitempty"`
	Affordances            []Affordance            `json:"affordances,omitempty"`
	Overlays               []Overlay               `json:"overlays,omitempty"`
	ReturnPaths            []ReturnPath            `json:"return_paths,omitempty"`
	Shortcuts              []Shortcut              `json:"shortcuts,omitempty"`
	ReachabilityInvariants []ReachabilityInvariant `json:"reachability_invariants,omitempty"`
	DeepLinkPolicy         []DeepLinkRule          `json:"deep_link_policy,omitempty"`
	ContractPath           string                  `json:"-"`
	Schema                 string                  `json:"$schema,omitempty"`
}

// Context is a single declared conditional dimension. Default is a
// json.RawMessage so the same struct serves both enum (string) and bool
// kinds without a discriminated union on the wire.
type Context struct {
	Kind      string          `json:"kind"`
	Values    []string        `json:"values,omitempty"`
	Default   json.RawMessage `json:"default"`
	ValidWhen string          `json:"valid_when,omitempty"`
}

type Route struct {
	ID              string           `json:"id"`
	Path            string           `json:"path"`
	Page            string           `json:"page"`
	Requires        string           `json:"requires,omitempty"`
	RedirectIfUnmet *RedirectIfUnmet `json:"redirect_if_unmet,omitempty"`
	DeepLink        string           `json:"deep_link,omitempty"`
	Parents         []string         `json:"parents,omitempty"`
}

type RedirectIfUnmet struct {
	To             string `json:"to"`
	PreserveTarget bool   `json:"preserve_target,omitempty"`
}

type Container struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	HostRoutes []string `json:"host_routes"`
	ShowWhen   string   `json:"show_when,omitempty"`
	Disclosure string   `json:"disclosure"`
	Trigger    *Trigger `json:"trigger,omitempty"`
	Scope      string   `json:"scope,omitempty"`
	FocusTrap  bool     `json:"focus_trap,omitempty"`
	Dismiss    []string `json:"dismiss,omitempty"`
}

type Trigger struct {
	Label        string   `json:"label"`
	TestID       string   `json:"test_id"`
	ReachableVia []string `json:"reachable_via"`
}

type Affordance struct {
	ID            string         `json:"id"`
	To            string         `json:"to"`
	ShowWhen      string         `json:"show_when,omitempty"`
	SideEffect    string         `json:"side_effect,omitempty"`
	Presentations []Presentation `json:"presentations"`
}

type Presentation struct {
	In           string   `json:"in"`
	Label        string   `json:"label"`
	Icon         string   `json:"icon,omitempty"`
	TestID       string   `json:"test_id"`
	ReachableVia []string `json:"reachable_via"`
}

type Overlay struct {
	ID         string   `json:"id"`
	HostRoutes []string `json:"host_routes"`
	Scope      string   `json:"scope"`
	FocusTrap  bool     `json:"focus_trap,omitempty"`
	Dismiss    []string `json:"dismiss,omitempty"`
	ShowWhen   string   `json:"show_when,omitempty"`
	TestID     string   `json:"test_id"`
}

type ReturnPath struct {
	From     string `json:"from"`
	Rule     string `json:"rule"`
	Fallback string `json:"fallback,omitempty"`
}

type Shortcut struct {
	Binding        string   `json:"binding"`
	Action         string   `json:"action"`
	To             string   `json:"to,omitempty"`
	Scope          string   `json:"scope"`
	ExcludedRoutes []string `json:"excluded_routes,omitempty"`
	ShowWhen       string   `json:"show_when,omitempty"`
}

// MaxClicks carries either a scalar budget (Scalar set, ByViewport nil)
// or a per-viewport budget (Scalar=-1, ByViewport populated). The custom
// UnmarshalJSON disambiguates.
type MaxClicks struct {
	Scalar     int
	ByViewport map[string]int
}

func (m *MaxClicks) UnmarshalJSON(data []byte) error {
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		m.Scalar = n
		m.ByViewport = nil
		return nil
	}
	var mp map[string]int
	if err := json.Unmarshal(data, &mp); err != nil {
		return fmt.Errorf("max_clicks must be an integer or {desktop?,tablet?,mobile?} object: %w", err)
	}
	m.Scalar = -1
	m.ByViewport = mp
	return nil
}

func (m MaxClicks) MarshalJSON() ([]byte, error) {
	if m.ByViewport != nil {
		return json.Marshal(m.ByViewport)
	}
	return json.Marshal(m.Scalar)
}

// IsSet reports whether MaxClicks carries any budget. The zero value
// (Scalar=0, ByViewport=nil) is treated as "no budget set" because the
// JSON Schema forbids a scalar 0 alongside missing field via omitempty
// on the parent.
func (m MaxClicks) IsSet() bool {
	return m.ByViewport != nil || m.Scalar > 0
}

type ReachabilityInvariant struct {
	ID           string     `json:"id"`
	Given        string     `json:"given"`
	From         string     `json:"from"`
	MustReach    []string   `json:"must_reach,omitempty"`
	MustNotReach []string   `json:"must_not_reach,omitempty"`
	MaxClicks    *MaxClicks `json:"max_clicks,omitempty"`
}

type DeepLinkRule struct {
	ID             string   `json:"id"`
	ForRoutes      []string `json:"for_routes,omitempty"`
	ForRoutesWhere string   `json:"for_routes_where,omitempty"`
	Given          string   `json:"given"`
	Must           string   `json:"must"`
}

// LoadBytes parses and validates a navigation contract. relPath is the
// repo-relative path used in error messages; it is not consulted for IO.
func LoadBytes(data []byte, relPath string) (Contract, error) {
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
	schemaOnce sync.Once
	schema     *jsonschema.Schema
	schemaErr  error
)

func validateJSONSchema(data []byte) error {
	s, err := compiledSchema()
	if err != nil {
		return err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	return s.Validate(value)
}

func compiledSchema() (*jsonschema.Schema, error) {
	schemaOnce.Do(func() {
		var schemaDoc any
		if err := json.Unmarshal(schemas.Navigation, &schemaDoc); err != nil {
			schemaErr = fmt.Errorf("parse embedded navigation.schema.json: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		const schemaURL = "https://vrooli.dev/schemas/navigation.v1.json"
		if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
			schemaErr = fmt.Errorf("load embedded navigation.schema.json: %w", err)
			return
		}
		schema, schemaErr = compiler.Compile(schemaURL)
	})
	return schema, schemaErr
}
