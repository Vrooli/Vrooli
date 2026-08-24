package models

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"resource-reranker/cli/internal/client"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/packages/capacity/companion"
)

type Handlers struct {
	GetEnv  func(string) string
	Stdout  io.Writer
	Stderr  io.Writer
	Now     func() time.Time
	Restart func(context.Context) error
	Info    func(context.Context) (map[string]any, error)
}

func Default() *Handlers {
	return &Handlers{
		GetEnv: os.Getenv,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Now:    time.Now,
		Restart: func(ctx context.Context) error {
			cmd := exec.CommandContext(ctx, "vrooli", "resource", "restart", "reranker")
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
			return cmd.Run()
		},
		Info: func(ctx context.Context) (map[string]any, error) {
			return client.NewClient().Info(ctx)
		},
	}
}

func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "models",
		Description: "Inspect and switch measured reranker models",
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List the measured reranker catalog", Usage: "resource-reranker models list [--json]", Run: h.List},
			{Name: "status", Description: "Show the active model and live TEI identity", Usage: "resource-reranker models status [--json]", Run: h.Status},
			{Name: "activate", Description: "Activate a measured model and restart the managed service", Usage: "resource-reranker models activate --model <id> [--role <role>] [--no-restart] [--json]", Run: h.Activate},
		},
	}
}

// CapacityCommands returns the `capacity` subcommand group every accelerated
// resource exposes. The broker calls `resource-reranker capacity degrade --to
// <model-id>`; for the reranker, a rung IS a model, so the degrade handler is
// the existing measured-model activation. The verb and its flags come from the
// fleet-wide contract in packages/capacity/companion.
func CapacityCommands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	verbs := companion.Verbs{
		Resource: "reranker",
		Degrade: func(_ context.Context, label string) error {
			return h.Activate([]string{"--model", label})
		},
	}
	return cliapp.SubcommandGroup{
		Name:        "capacity",
		Description: "Respond to the capacity broker",
		Subcommands: []cliapp.Command{
			{
				Name:        "degrade",
				Description: "Activate a smaller measured model at the capacity broker's request",
				Usage:       "resource-reranker capacity degrade --to <model-id>",
				Run:         func(args []string) error { return verbs.Run(append([]string{"degrade"}, args...)) },
			},
			{
				Name:        "upshift",
				Description: "Activate a larger measured model when headroom returns",
				Usage:       "resource-reranker capacity upshift --to <model-id>",
				Run:         func(args []string) error { return verbs.Run(append([]string{"upshift"}, args...)) },
			},
		},
	}
}

func StatusCommand(h *Handlers) cliapp.CommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.CommandGroup{Title: "Model status", Commands: []cliapp.Command{{
		Name:        "status",
		Description: "Show resource status including the active reranker model",
		Usage:       "resource-reranker status [--json]",
		Run:         h.Status,
	}}}
}

func (h *Handlers) List(args []string) error {
	fs := flag.NewFlagSet("models list", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	jsonOutput := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	path := policyPath(h.GetEnv)
	policy, err := Load(path)
	if err != nil {
		return err
	}
	type row struct {
		Name string `json:"name"`
		Model
	}
	rows := make([]row, 0, len(policy.Models))
	for _, name := range policy.ModelNames() {
		rows = append(rows, row{Name: name, Model: policy.Models[name]})
	}
	payload := map[string]any{"schema_version": policy.SchemaVersion, "policy_path": path, "role": defaultRole, "models": rows}
	return writeResult(h.Stdout, *jsonOutput, payload, func() {
		for _, item := range rows {
			fmt.Fprintf(h.Stdout, "%s top1=%.3f top3=%.3f resident_bytes=%d\n", item.Name, item.Measurement.Top1, item.Measurement.Top3, item.ResidentBytes)
		}
	})
}

func (h *Handlers) Status(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	jsonOutput := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	path := policyPath(h.GetEnv)
	_, policyErr := Load(path)
	active := readActiveModel(h.GetEnv)
	info := map[string]any{}
	infoErr := error(nil)
	if h.Info != nil {
		info, infoErr = h.Info(context.Background())
	}
	if active == "" {
		if model, ok := info["model_id"].(string); ok {
			active = model
		}
	}
	payload := map[string]any{"active_model": active, "role": defaultRole, "policy_path": path, "info": info}
	if policyErr != nil {
		payload["policy_error"] = policyErr.Error()
	}
	if infoErr != nil {
		payload["info_error"] = infoErr.Error()
	}
	err := writeResult(h.Stdout, *jsonOutput, payload, func() {
		fmt.Fprintf(h.Stdout, "active_model: %s\n", active)
		if model, ok := info["model_id"].(string); ok {
			fmt.Fprintf(h.Stdout, "served_model: %s\n", model)
		}
	})
	if err != nil {
		return err
	}
	return nil
}

func (h *Handlers) Activate(args []string) error {
	fs := flag.NewFlagSet("models activate", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	model := fs.String("model", "", "Measured model id")
	role := fs.String("role", defaultRole, "Policy role")
	noRestart := fs.Bool("no-restart", false, "Only write the durable selection; do not restart")
	jsonOutput := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || strings.TrimSpace(*model) == "" {
		return errors.New("models activate requires --model <id> and no positional arguments")
	}
	path := policyPath(h.GetEnv)
	policy, err := Load(path)
	if err != nil {
		return err
	}
	if !policy.Allowed(*role, *model) {
		return fmt.Errorf("model %q is not an allowed measured member of role %q", *model, *role)
	}
	statePath := activeEnvPath(h.GetEnv)
	if err := writeActiveModel(statePath, *model); err != nil {
		return err
	}
	if !*noRestart {
		if h.Restart == nil {
			return errors.New("reranker restart handler is not configured")
		}
		if err := h.Restart(context.Background()); err != nil {
			return fmt.Errorf("restart reranker for model activation: %w", err)
		}
	}
	payload := map[string]any{"active_model": *model, "role": *role, "policy_path": path, "state_path": statePath, "restarted": !*noRestart}
	return writeResult(h.Stdout, *jsonOutput, payload, func() {
		fmt.Fprintf(h.Stdout, "active_model: %s\nrestarted: %t\n", *model, !*noRestart)
	})
}

func policyPath(getEnv func(string) string) string {
	if path := strings.TrimSpace(getEnv("RERANKER_MODEL_POLICY_PATH")); path != "" {
		return path
	}
	for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_CLI_SOURCE_ROOT", "RESOURCE_ROOT"} {
		if root := strings.TrimSpace(getEnv(key)); root != "" {
			return filepath.Join(root, "resources", "reranker", "model-policy.json")
		}
	}
	return "resources/reranker/model-policy.json"
}

func activeEnvPath(getEnv func(string) string) string {
	if path := strings.TrimSpace(getEnv("RERANKER_ACTIVE_ENV_PATH")); path != "" {
		return path
	}
	if root := strings.TrimSpace(getEnv("RESOURCE_DATA_DIR")); root != "" {
		return filepath.Join(root, "active.env")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "vrooli", "resources", "reranker", "active.env")
	}
	return filepath.Join("resources", "reranker", "active.env")
}

func readActiveModel(getEnv func(string) string) string {
	data, err := os.ReadFile(activeEnvPath(getEnv))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if ok && key == "RERANKER_MODEL" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func writeActiveModel(path, model string) error {
	if strings.TrimSpace(model) == "" {
		return errors.New("active reranker model cannot be empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create reranker state directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".active-*.tmp")
	if err != nil {
		return fmt.Errorf("create reranker state file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := fmt.Fprintf(tmp, "RERANKER_MODEL=%s\n", model); err != nil {
		tmp.Close()
		return fmt.Errorf("write reranker state file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("commit reranker model selection: %w", err)
	}
	return nil
}

func writeResult(out io.Writer, jsonOutput bool, payload any, text func()) error {
	if jsonOutput {
		return json.NewEncoder(out).Encode(payload)
	}
	text()
	return nil
}
