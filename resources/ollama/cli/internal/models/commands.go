package models

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
)

// Handlers owns the runtime dependencies for the `models` subcommand group.
type Handlers struct {
	NewDaemon func() Daemon
	GetEnv    func(string) string
	Stdout    io.Writer
	Stderr    io.Writer
}

// Default returns Handlers wired to the real Ollama daemon client (the gateway
// SSOT) and the process environment.
func Default() *Handlers {
	return &Handlers{
		NewDaemon: func() Daemon { return ensure.NewClient() },
		GetEnv:    os.Getenv,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	}
}

// Commands returns the `models` subcommand group for registration.
func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "models",
		Description: "Probe and validate locally-installed Ollama models (SSOT)",
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List installed models from the live daemon",
				Usage:       "resource-ollama models list [--json]",
				Run:         h.List,
			},
			{
				Name:        "inventory",
				Description: "Report named model sizes, digests, and policy reachability",
				Usage:       "resource-ollama models inventory [--json]",
				Run:         h.Inventory,
			},
			{
				Name:        "probe-tools",
				Description: "Run a live tool-call smoke against a model or role",
				Usage:       "resource-ollama models probe-tools (--model <ref> | --role <role>) [--json]",
				Run:         h.ProbeTools,
			},
			{
				Name:        "probe-vision",
				Description: "Run a live image-input smoke against a model or role",
				Usage:       "resource-ollama models probe-vision (--model <ref> | --role <role>) --image <path> [--json]",
				Run:         h.ProbeVision,
			},
			{
				Name:        "doctor",
				Description: "Validate role capabilities + live tool-calling against the daemon",
				Usage:       "resource-ollama models doctor (--role <role> | --all) [--json]",
				Run:         h.Doctor,
			},
		},
	}
}

func (h *Handlers) List(args []string) error {
	fs := flag.NewFlagSet("models list", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	asJSON := fs.Bool("json", false, "Emit a JSON object {\"models\":[...]}")
	if err := fs.Parse(args); err != nil {
		return err
	}
	res, err := List(context.Background(), h.NewDaemon())
	if err != nil {
		return err
	}
	if *asJSON {
		return writeJSON(h.Stdout, res)
	}
	for _, m := range res.Models {
		if _, err := fmt.Fprintln(h.Stdout, m); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handlers) Inventory(args []string) error {
	fs := flag.NewFlagSet("models inventory", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	asJSON := fs.Bool("json", false, "Emit a JSON inventory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	p, _, err := h.loadPolicy()
	if err != nil {
		return err
	}
	res, err := Inventory(context.Background(), h.NewDaemon(), p)
	if err != nil {
		return err
	}
	if *asJSON {
		return writeJSON(h.Stdout, res)
	}
	for _, model := range res.Models {
		if _, err := fmt.Fprintf(h.Stdout, "%s\t%d\t%s\tpolicy_reachable=%t\tregenerable=%t\n", model.Name, model.Size, model.Digest, model.PolicyReachable, model.Regenerable); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handlers) ProbeTools(args []string) error {
	fs := flag.NewFlagSet("models probe-tools", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	model := fs.String("model", "", "Model reference to probe")
	role := fs.String("role", "", "Role to resolve and probe")
	asJSON := fs.Bool("json", false, "Emit JSON result")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*model) == "" && strings.TrimSpace(*role) == "" {
		return errors.New("--model or --role is required")
	}
	if strings.TrimSpace(*model) != "" && strings.TrimSpace(*role) != "" {
		return errors.New("--model and --role are mutually exclusive")
	}
	ref := strings.TrimSpace(*model)
	if ref == "" {
		p, _, err := h.loadPolicy()
		if err != nil {
			return err
		}
		resolved, err := p.ResolveRole(*role)
		if err != nil {
			return err
		}
		ref = resolved.Model
	}
	res, probeErr := ProbeTools(context.Background(), h.NewDaemon(), ref)
	if *asJSON {
		if probeErr != nil && res.Error == "" {
			res.Error = probeErr.Error()
		}
		if err := writeJSON(h.Stdout, res); err != nil {
			return err
		}
		if !res.SupportsTools {
			return errQuietFail
		}
		return nil
	}
	if probeErr != nil {
		return probeErr
	}
	if _, err := fmt.Fprintf(h.Stdout, "%s supports_tools=%t — %s\n", res.Model, res.SupportsTools, res.Evidence); err != nil {
		return err
	}
	if !res.SupportsTools {
		return errQuietFail
	}
	return nil
}

func (h *Handlers) ProbeVision(args []string) error {
	fs := flag.NewFlagSet("models probe-vision", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	model := fs.String("model", "", "Model reference to probe")
	role := fs.String("role", "", "Role to resolve and probe")
	image := fs.String("image", "", "Path to a PNG/JPEG/WEBP image")
	asJSON := fs.Bool("json", false, "Emit JSON result")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*model) == "" && strings.TrimSpace(*role) == "" {
		return errors.New("--model or --role is required")
	}
	if strings.TrimSpace(*model) != "" && strings.TrimSpace(*role) != "" {
		return errors.New("--model and --role are mutually exclusive")
	}
	if strings.TrimSpace(*image) == "" {
		return errors.New("--image is required")
	}
	ref := strings.TrimSpace(*model)
	if ref == "" {
		p, _, err := h.loadPolicy()
		if err != nil {
			return err
		}
		resolved, err := p.ResolveRole(*role)
		if err != nil {
			return err
		}
		ref = resolved.Model
	}
	imageBytes, err := os.ReadFile(*image)
	if err != nil {
		return fmt.Errorf("read image: %w", err)
	}
	res, probeErr := ProbeVision(context.Background(), h.NewDaemon(), ref, imageBytes)
	if *asJSON {
		if probeErr != nil && res.Error == "" {
			res.Error = probeErr.Error()
		}
		if err := writeJSON(h.Stdout, res); err != nil {
			return err
		}
		if !res.SupportsVision {
			return errQuietFail
		}
		return nil
	}
	if probeErr != nil {
		return probeErr
	}
	if _, err := fmt.Fprintf(h.Stdout, "%s supports_vision=%t — %s\n", res.Model, res.SupportsVision, res.Evidence); err != nil {
		return err
	}
	if !res.SupportsVision {
		return errQuietFail
	}
	return nil
}

func (h *Handlers) Doctor(args []string) error {
	fs := flag.NewFlagSet("models doctor", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	role := fs.String("role", "", "Role to validate, e.g. code.local")
	all := fs.Bool("all", false, "Validate every role in policy")
	asJSON := fs.Bool("json", false, "Emit JSON result")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*role) == "" && !*all {
		return errors.New("--role or --all is required")
	}
	p, _, err := h.loadPolicy()
	if err != nil {
		return err
	}
	opts := DoctorOptions{All: *all}
	if r := strings.TrimSpace(*role); r != "" {
		opts.Roles = []string{r}
	}
	res, err := Doctor(context.Background(), h.NewDaemon(), p, opts)
	if err != nil {
		return err
	}
	if *asJSON {
		if err := writeJSON(h.Stdout, res); err != nil {
			return err
		}
		if !res.Pass {
			return errQuietFail
		}
		return nil
	}
	for _, m := range res.Models {
		verdict := "PASS"
		if !m.Pass {
			verdict = "FAIL"
		}
		if _, err := fmt.Fprintf(h.Stdout, "%s -> %s: %s\n", m.Role, m.Model, verdict); err != nil {
			return err
		}
		for _, c := range m.Checks {
			detail := ""
			if c.Detail != "" {
				detail = " (" + c.Detail + ")"
			}
			if _, err := fmt.Fprintf(h.Stdout, "  - %s: %s%s\n", c.Name, c.Status, detail); err != nil {
				return err
			}
		}
	}
	if !res.Pass {
		return errQuietFail
	}
	return nil
}

// errQuietFail signals a non-zero exit without re-printing an error message
// (the JSON/human report already conveyed the failure).
var errQuietFail = errors.New("one or more models failed validation")

func (h *Handlers) loadPolicy() (policy.Policy, string, error) {
	getenv := h.GetEnv
	if getenv == nil {
		getenv = os.Getenv
	}
	return policy.LoadDefaultFile(getenv)
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
