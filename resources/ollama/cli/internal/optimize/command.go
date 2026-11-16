package optimize

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
)

// Handlers owns the dependencies for the resource optimize adapter command.
type Handlers struct {
	NewLoopbackDaemon func() (*ensure.Client, error)
	GetEnv            func(string) string
	Stdout            io.Writer
	Stderr            io.Writer
}

func DefaultHandlers() *Handlers {
	return &Handlers{
		NewLoopbackDaemon: ensure.NewLoopbackClient,
		GetEnv:            os.Getenv,
		Stdout:            os.Stdout,
		Stderr:            os.Stderr,
	}
}

// Commands registers a callable owner operation for LM6. Its scope is
// intentionally read-only until all downstream effect owners are available.
func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = DefaultHandlers()
	}
	return cliapp.SubcommandGroup{
		Name:        "optimize",
		Description: "Inspect candidate optimization requests through Ollama-owned facts",
		Subcommands: []cliapp.Command{{
			Name:        "role-adapter",
			Description: "Attest installed role candidates and return an explicit LM6 receipt",
			Usage:       "resource-ollama optimize role-adapter --request <json-file> --json",
			Run:         h.RoleAdapter,
		}},
	}
}

func (h *Handlers) RoleAdapter(args []string) error {
	fs := flag.NewFlagSet("optimize role-adapter", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	requestPath := fs.String("request", "", "Path to an ollama.optimize.role-request.v1 JSON request")
	asJSON := fs.Bool("json", false, "Emit an ollama.optimize.receipt.v1 JSON receipt")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *requestPath == "" || !*asJSON {
		return errors.New("--request and --json are required")
	}
	raw, err := os.ReadFile(*requestPath)
	if err != nil {
		return fmt.Errorf("read optimization request: %w", err)
	}
	request, err := decodeRoleRequest(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("decode optimization request: %w", err)
	}
	if err := request.Validate(); err != nil {
		if encodeErr := cliout.NewEncoder(h.Stdout).Encode(NewRefusal(request, "invalid request: "+err.Error())); encodeErr != nil {
			return encodeErr
		}
		return err
	}
	client, err := h.NewLoopbackDaemon()
	if err != nil {
		return fmt.Errorf("verify local Ollama endpoint: %w", err)
	}
	p, _, err := policy.LoadDefaultFile(h.GetEnv)
	if err != nil {
		return err
	}
	inventory, inventoryErr := client.ListModelInventory(context.Background())
	if inventoryErr != nil {
		return fmt.Errorf("read resource-owned Ollama inventory: %w", inventoryErr)
	}
	receipt, inspectErr := Inspect(context.Background(), request, inventory, p, h.GetEnv)
	if err := cliout.NewEncoder(h.Stdout).Encode(receipt); err != nil {
		return err
	}
	return inspectErr
}

func decodeRoleRequest(r io.Reader) (RoleRequest, error) {
	var request RoleRequest
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return RoleRequest{}, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return RoleRequest{}, errors.New("optimization request must contain exactly one JSON object")
	}
	return request, nil
}
