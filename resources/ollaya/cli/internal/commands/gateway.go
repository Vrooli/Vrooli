package commands

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

// maxRequestBytes mirrors the daemon's 8 MiB body ceiling.
const maxRequestBytes = 8 << 20

func GatewayCommands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "gateway",
		Description: "Answer typed decision questions (the surface ai-gateway calls)",
		Subcommands: []cliapp.Command{
			{
				Name:        "decide",
				Description: "Answer a System One request ({state, questions}) read from stdin on the role's upstream",
				Usage:       "resource-ollaya gateway decide --role <id> --input-stdin [--timeout 90s] [--json]",
				Run:         h.Decide,
			},
			{
				Name:        "health",
				Description: "Probe the local daemon (GET /api/version)",
				Usage:       "resource-ollaya gateway health [--json]",
				Run:         h.Health,
			},
		},
	}
}

func (h *Handlers) Decide(args []string) error {
	fs := flag.NewFlagSet("gateway decide", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	role := fs.String("role", "", "Decision role")
	inputStdin := fs.Bool("input-stdin", false, "Read the {state, questions} JSON body from stdin")
	timeout := fs.Duration("timeout", 90*time.Second, "Upstream deadline, model load included")
	jsonOutput := fs.Bool("json", false, "Emit the normalized result as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*role) == "" || !*inputStdin {
		return errors.New("gateway decide requires --role and --input-stdin")
	}
	raw, err := io.ReadAll(io.LimitReader(h.Stdin, maxRequestBytes+1))
	if err != nil {
		return fmt.Errorf("read decision request: %w", err)
	}
	if len(raw) > maxRequestBytes {
		return fmt.Errorf("decision request exceeds %d bytes", maxRequestBytes)
	}
	var request upstream.Request
	if err := json.Unmarshal(raw, &request); err != nil {
		return fmt.Errorf("decode decision request: %w", err)
	}
	p, err := h.policy()
	if err != nil {
		return err
	}
	resolution, err := p.Resolve(*role, h.GetEnv)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	result, err := h.client().Decide(ctx, resolution, request)
	if err != nil {
		return h.fail(err)
	}
	if *jsonOutput {
		return writeJSON(h.Stdout, result)
	}
	fmt.Fprintf(h.Stdout, "upstream=%s model=%s latency_ms=%d\n%s\n", result.Upstream, result.Model, result.LatencyMS, string(result.Answers))
	return nil
}

func (h *Handlers) Health(args []string) error {
	fs := flag.NewFlagSet("gateway health", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	jsonOutput := fs.Bool("json", false, "Emit {healthy, version}")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	version, err := h.client().Version(ctx)
	if *jsonOutput {
		payload := map[string]any{"healthy": err == nil, "version": version}
		if err != nil {
			payload["error"] = err.Error()
		}
		if encodeErr := writeJSON(h.Stdout, payload); encodeErr != nil {
			return encodeErr
		}
		return err
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(h.Stdout, "healthy (ollaya %s)\n", version)
	return err
}
