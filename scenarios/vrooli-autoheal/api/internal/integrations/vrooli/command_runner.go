package vrooli

import (
	"context"
	"encoding/json"
	"strings"

	vroolicli "github.com/vrooli/vrooli-cli-go"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Client adapts the typed Vrooli CLI client (github.com/vrooli/vrooli-cli-go)
// to autoheal's check infrastructure. Typed reads decode the vrooli.cli.v1
// proto contracts, so a CLI output change is a compile error here rather than a
// silently wrong heal decision; raw text mutations still flow through the
// shared CommandExecutor.
type Client struct {
	executor checks.CommandExecutor
	cli      *vroolicli.Client
}

// NewClient builds a Client over the given executor. The executor resolves
// the vrooli binary path and passes no global flags: the CLI's argv surface is
// a contract every supervisor consumes, and behavior switches travel as
// VROOLI_* environment variables instead. WithRunner makes the typed client
// and the executor share one execution (and test-mock) seam.
func NewClient(executor checks.CommandExecutor) *Client {
	if executor == nil {
		executor = checks.DefaultExecutor
	}
	return &Client{
		executor: executor,
		cli:      vroolicli.New(vroolicli.WithRunner(executorRunner{exec: executor})),
	}
}

// executorRunner adapts checks.CommandExecutor to vroolicli.Runner.
type executorRunner struct{ exec checks.CommandExecutor }

func (r executorRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return r.exec.Output(ctx, name, args...)
}

func (r executorRunner) RunCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	return r.exec.CombinedOutput(ctx, name, args...)
}

// CombinedOutput runs an arbitrary `vrooli` command and returns combined
// stdout/stderr. It is the raw escape hatch for plain-text commands (start,
// stop, logs, …) that have no typed JSON contract.
func (c *Client) CombinedOutput(ctx context.Context, args ...string) ([]byte, error) {
	return c.executor.CombinedOutput(ctx, "vrooli", args...)
}

func requiredString(value string, field string) error {
	if strings.TrimSpace(value) == "" {
		return ContractError{Field: field, Message: "missing required field"}
	}
	return nil
}

type ContractError struct {
	Field   string
	Message string
}

func (e ContractError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

// diagnosticJSON renders a decoded proto message back to indented JSON for the
// `[]byte` diagnostic that checks surface in their Result. It is presentation
// only; callers parse the typed value, never these bytes.
func diagnosticJSON(msg proto.Message) []byte {
	out, err := protojson.MarshalOptions{UseProtoNames: true, Indent: "  "}.Marshal(msg)
	if err != nil {
		return nil
	}
	return out
}

// valueToRawJSON renders a protobuf Value (used for string-or-null fields like
// health_status) back to raw JSON so domain helpers can interpret it.
func valueToRawJSON(v *structpb.Value) json.RawMessage {
	if v == nil {
		return nil
	}
	out, err := protojson.Marshal(v)
	if err != nil {
		return nil
	}
	return json.RawMessage(out)
}
