package schema

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/browser-automation-studio/workflow/validator"
	schemav1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/schema"
	schemaconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/schema/schemaconnect"
)

type fakeProvider struct {
	fullSchema     json.RawMessage
	fullErr        error
	filteredSchema json.RawMessage
	filteredErr    error
	lastFilter     []string

	nodeTypes []string
	stepDefs  []validator.StepDefinition
	cliDefs   []validator.StepDefinition
	lastCLI   bool
}

func (f *fakeProvider) GetFullSchema() (json.RawMessage, error) {
	return f.fullSchema, f.fullErr
}

func (f *fakeProvider) GetFilteredSchema(nodeTypes []string) (json.RawMessage, error) {
	f.lastFilter = nodeTypes
	return f.filteredSchema, f.filteredErr
}

func (f *fakeProvider) AvailableNodeTypes() []string { return f.nodeTypes }

func (f *fakeProvider) StepDefinitions(cliOnly bool) []validator.StepDefinition {
	f.lastCLI = cliOnly
	if cliOnly {
		return f.cliDefs
	}
	return f.stepDefs
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { w.t.Log(string(p)); return len(p), nil }

func newTestClient(t *testing.T, p *fakeProvider) schemaconnect.SchemaServiceClient {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(testWriter{t})
	mount, err := Module(Deps{Provider: p, Logger: logger})
	require.NoError(t, err)
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return schemaconnect.NewSchemaServiceClient(srv.Client(), srv.URL)
}

func TestGetWorkflowSchema_FullSchema(t *testing.T) {
	p := &fakeProvider{fullSchema: json.RawMessage(`{"title":"Workflow","type":"object"}`)}
	client := newTestClient(t, p)

	resp, err := client.GetWorkflowSchema(context.Background(),
		connect.NewRequest(&schemav1.GetWorkflowSchemaRequest{}))
	require.NoError(t, err)
	m := resp.Msg.Schema.AsMap()
	require.Equal(t, "Workflow", m["title"])
	require.Equal(t, "object", m["type"])
	require.Nil(t, p.lastFilter)
}

func TestGetWorkflowSchema_Filtered(t *testing.T) {
	p := &fakeProvider{filteredSchema: json.RawMessage(`{"title":"Filtered"}`)}
	client := newTestClient(t, p)

	resp, err := client.GetWorkflowSchema(context.Background(),
		connect.NewRequest(&schemav1.GetWorkflowSchemaRequest{NodeTypes: []string{"navigate", "click"}}))
	require.NoError(t, err)
	require.Equal(t, "Filtered", resp.Msg.Schema.AsMap()["title"])
	require.Equal(t, []string{"navigate", "click"}, p.lastFilter)
}

func TestGetWorkflowSchema_ProviderErrorMapsToInternal(t *testing.T) {
	p := &fakeProvider{fullErr: errors.New("boom")}
	client := newTestClient(t, p)

	_, err := client.GetWorkflowSchema(context.Background(),
		connect.NewRequest(&schemav1.GetWorkflowSchemaRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestGetNodeTypes_HappyPath(t *testing.T) {
	p := &fakeProvider{nodeTypes: []string{"navigate", "click", "wait"}}
	client := newTestClient(t, p)

	resp, err := client.GetNodeTypes(context.Background(),
		connect.NewRequest(&schemav1.GetNodeTypesRequest{}))
	require.NoError(t, err)
	require.Equal(t, []string{"navigate", "click", "wait"}, resp.Msg.NodeTypes)
}

func TestGetStepDefinitions_All(t *testing.T) {
	pos := validator.PositionalDef{Name: "url", MapsTo: "url", Description: "Target URL"}
	p := &fakeProvider{stepDefs: []validator.StepDefinition{
		{
			Type: "navigate", Description: "Go", Positional: &pos, CLISupported: true,
			RequiredKVs: []validator.KVDef{{Key: "k", Type: "string", Description: "d"}},
			Examples:    []validator.StepExample{{Description: "e", CLI: "c"}},
		},
		{
			Type: "wait", Description: "Wait", CLISupported: false,
			RequireOneOf: [][]string{{"a", "b"}},
		},
	}}
	client := newTestClient(t, p)

	resp, err := client.GetStepDefinitions(context.Background(),
		connect.NewRequest(&schemav1.GetStepDefinitionsRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Steps, 2)
	require.Equal(t, "navigate", resp.Msg.Steps[0].Type)
	require.Equal(t, "url", resp.Msg.Steps[0].Positional.Name)
	require.Equal(t, "k", resp.Msg.Steps[0].RequiredKvs[0].Key)
	require.Equal(t, "e", resp.Msg.Steps[0].Examples[0].Description)
	require.Equal(t, "wait", resp.Msg.Steps[1].Type)
	require.False(t, resp.Msg.Steps[1].CliSupported)
	require.Equal(t, []string{"a", "b"}, resp.Msg.Steps[1].RequireOneOf[0].Keys)
	require.False(t, p.lastCLI)
}

func TestGetStepDefinitions_CLIOnly(t *testing.T) {
	p := &fakeProvider{cliDefs: []validator.StepDefinition{
		{Type: "navigate", CLISupported: true},
	}}
	client := newTestClient(t, p)

	resp, err := client.GetStepDefinitions(context.Background(),
		connect.NewRequest(&schemav1.GetStepDefinitionsRequest{CliOnly: true}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Steps, 1)
	require.Equal(t, "navigate", resp.Msg.Steps[0].Type)
	require.True(t, p.lastCLI)
}

func TestGetStepDefinitions_TypeFilter(t *testing.T) {
	p := &fakeProvider{stepDefs: []validator.StepDefinition{
		{Type: "navigate", CLISupported: true},
		{Type: "click", CLISupported: true},
		{Type: "wait", CLISupported: false},
	}}
	client := newTestClient(t, p)

	resp, err := client.GetStepDefinitions(context.Background(),
		connect.NewRequest(&schemav1.GetStepDefinitionsRequest{Types: []string{"click", "wait"}}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Steps, 2)
	require.Equal(t, "click", resp.Msg.Steps[0].Type)
	require.Equal(t, "wait", resp.Msg.Steps[1].Type)
}

func TestModule_RequiresLogger(t *testing.T) {
	require.Panics(t, func() { _, _ = Module(Deps{Provider: &fakeProvider{}}) })
}
