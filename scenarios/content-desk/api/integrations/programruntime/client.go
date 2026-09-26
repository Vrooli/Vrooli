// Package programruntime is Content Desk's outbound boundary to the governed
// program runtime. It runs a scenario-declared program and returns its
// envelope unchanged; it never re-implements the program's logic.
package programruntime

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library/library_v1connect"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	scenarioID = "program-runtime"
	// The declared board contract budgets 60s of wall time; the transport
	// deadline is set above it so the runtime, not the client, decides when
	// the wait is over.
	requestTimeout = 90 * time.Second
)

// Result is the terminal declared-program outcome. Envelope is the program's
// stdout, which every declared contract prints exactly once.
type Result struct {
	Envelope  []byte
	ProgramID string
	Terminal  bool
	Status    string
}

// URLResolver resolves a sibling scenario to its live base URL.
type URLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

// Runner runs one declared program and returns its envelope.
type Runner interface {
	RunDeclared(context.Context, string, map[string]any) (Result, error)
}

// Client resolves the dependency for each call so a restart or test-shadow
// port change cannot leave Content Desk pointing at a stale process.
type Client struct {
	resolver URLResolver
	http     *http.Client
}

func NewClient() *Client {
	return &Client{resolver: discovery.NewResolver(discovery.ResolverConfig{}), http: &http.Client{Timeout: requestTimeout}}
}

// RunDeclared executes name with the given declared inputs and returns the
// terminal envelope. A non-terminal run is an explicit error carrying the
// durable program id; it is never reported as a failed or empty result.
func (c *Client) RunDeclared(ctx context.Context, name string, inputs map[string]any) (Result, error) {
	if strings.TrimSpace(name) == "" {
		return Result{}, fmt.Errorf("declared program name is required")
	}
	if c == nil || c.resolver == nil || c.http == nil {
		return Result{}, fmt.Errorf("program runtime integration is not configured")
	}
	structured, err := structpb.NewStruct(inputs)
	if err != nil {
		return Result{}, fmt.Errorf("encode declared program inputs: %w", err)
	}
	callCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	baseURL, err := c.resolver.ResolveScenarioURLDefault(callCtx, scenarioID)
	if err != nil {
		return Result{}, fmt.Errorf("resolve program-runtime: %w", err)
	}
	client := libraryconnect.NewLibraryServiceClient(c.http, strings.TrimRight(baseURL, "/"))
	response, err := client.RunDeclaredProgram(callCtx, connect.NewRequest(&libraryv1.RunDeclaredProgramRequest{
		Name:       name,
		Inputs:     structured,
		Provenance: programsv1.Provenance_PROVENANCE_AGENT,
		Caller:     &programsv1.Caller{Harness: "content-desk-api"},
	}))
	if err != nil {
		return Result{}, fmt.Errorf("run declared program %s: %w", name, err)
	}
	program := response.Msg.GetProgram()
	if program == nil {
		return Result{}, fmt.Errorf("declared program %s returned no program", name)
	}
	result := Result{
		ProgramID: program.GetId(),
		Terminal:  response.Msg.GetTerminal(),
		Status:    program.GetStatus().String(),
	}
	if !result.Terminal {
		return result, fmt.Errorf("declared program %s did not reach a terminal state; inspect program %s", name, result.ProgramID)
	}
	result.Envelope = []byte(program.GetStdout())
	return result, nil
}
