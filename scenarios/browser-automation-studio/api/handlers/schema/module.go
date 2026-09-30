// Package schema hosts the BAS SchemaService Connect-RPC handler.
//
// SchemaService exposes the workflow JSON-Schema and the step-definition
// catalogue used by UI authoring tools and CLI documentation.
package schema

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"

	"github.com/vrooli/browser-automation-studio/workflow/validator"
	schemaconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/schema/schemaconnect"
)

// Provider is the narrow seam the schema handler depends on. The concrete
// implementation wraps validator.SchemaProvider + the validator package-level
// step-definition helpers; tests supply an in-memory fake.
type Provider interface {
	GetFullSchema() (json.RawMessage, error)
	GetFilteredSchema(nodeTypes []string) (json.RawMessage, error)
	AvailableNodeTypes() []string
	StepDefinitions(cliOnly bool) []validator.StepDefinition
}

// Deps wires the schema handler. Logger is required; Provider defaults to
// the validator-backed implementation when nil.
type Deps struct {
	Provider Provider
	Logger   *logrus.Logger
}

// Module builds the SchemaService Connect handler.
func Module(d Deps) (connectx.ServiceMount, error) {
	if d.Logger == nil {
		panic("schema.Module requires Deps.Logger")
	}
	if d.Provider == nil {
		p, err := newDefaultProvider()
		if err != nil {
			return connectx.ServiceMount{}, err
		}
		d.Provider = p
	}
	path, handler := schemaconnect.NewSchemaServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}, nil
}

type defaultProvider struct {
	sp validator.SchemaProvider
}

func newDefaultProvider() (*defaultProvider, error) {
	sp, err := validator.NewSchemaProvider()
	if err != nil {
		return nil, err
	}
	return &defaultProvider{sp: sp}, nil
}

func (p *defaultProvider) GetFullSchema() (json.RawMessage, error) {
	return p.sp.GetFullSchema()
}

func (p *defaultProvider) GetFilteredSchema(nodeTypes []string) (json.RawMessage, error) {
	return p.sp.GetFilteredSchema(nodeTypes)
}

func (p *defaultProvider) AvailableNodeTypes() []string {
	return validator.GetAvailableNodeTypes()
}

func (p *defaultProvider) StepDefinitions(cliOnly bool) []validator.StepDefinition {
	if cliOnly {
		return validator.GetCLISupportedStepDefinitions()
	}
	return validator.GetStepDefinitions()
}
