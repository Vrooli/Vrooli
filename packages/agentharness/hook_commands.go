package agentharness

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/vrooli/cli-core/cliapp"
)

// HookCommandConfig describes one provider's native hook path while the
// broker owns command parsing and mutation semantics.
type HookCommandConfig struct {
	Agent        string
	Description  string
	ScopeDefault string
	ScopeHelp    string
	Target       func(scope string) (HookTarget, error)
	Stdout       io.Writer
	Stderr       io.Writer
}

// HookCommands exposes the common hooks list/reconcile/remove surface.
func HookCommands(cfg HookCommandConfig) cliapp.SubcommandGroup {
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	if cfg.Stderr == nil {
		cfg.Stderr = os.Stderr
	}
	if cfg.ScopeHelp == "" {
		cfg.ScopeHelp = "Settings scope"
	}
	broker := NewHookBroker()
	flagSet := func(name string) *flag.FlagSet {
		fs := flag.NewFlagSet(name, flag.ContinueOnError)
		fs.SetOutput(cfg.Stderr)
		if cfg.ScopeDefault != "" {
			fs.String("scope", cfg.ScopeDefault, cfg.ScopeHelp)
		}
		return fs
	}
	scope := func(fs *flag.FlagSet) string {
		if field := fs.Lookup("scope"); field != nil {
			return field.Value.String()
		}
		return ""
	}
	reconcile := func(args []string) error {
		fs := flagSet("hooks reconcile")
		event := fs.String("event", "", "Hook event name")
		id := fs.String("id", "", "Stable hook identifier")
		hookJSON := fs.String("hook-json", "", "Hook object as JSON")
		if err := fs.Parse(args); err != nil {
			return err
		}
		if *event == "" || *id == "" || *hookJSON == "" {
			return errors.New("--event, --id, and --hook-json are required")
		}
		var hook map[string]any
		if err := json.Unmarshal([]byte(*hookJSON), &hook); err != nil {
			return fmt.Errorf("invalid --hook-json: %w", err)
		}
		target, err := cfg.Target(scope(fs))
		if err != nil {
			return err
		}
		if _, err := broker.Migrate(target); err != nil {
			return err
		}
		result, err := broker.Reconcile(target, HookRegistration{Event: *event, ID: *id, Hook: hook})
		if writeErr := writeHookJSON(cfg.Stdout, result); writeErr != nil {
			return writeErr
		}
		return err
	}
	remove := func(args []string) error {
		fs := flagSet("hooks remove")
		event := fs.String("event", "", "Hook event name")
		id := fs.String("id", "", "Stable hook identifier")
		if err := fs.Parse(args); err != nil {
			return err
		}
		if *event == "" || *id == "" {
			return errors.New("--event and --id are required")
		}
		target, err := cfg.Target(scope(fs))
		if err != nil {
			return err
		}
		if _, err := broker.Migrate(target); err != nil {
			return err
		}
		result, err := broker.Remove(target, *event, *id)
		if writeErr := writeHookJSON(cfg.Stdout, result); writeErr != nil {
			return writeErr
		}
		return err
	}
	migrate := func(args []string) error {
		fs := flagSet("hooks migrate")
		if err := fs.Parse(args); err != nil {
			return err
		}
		target, err := cfg.Target(scope(fs))
		if err != nil {
			return err
		}
		count, err := broker.Migrate(target)
		if writeErr := writeHookJSON(cfg.Stdout, map[string]any{
			"agent": target.Agent, "path": target.Path, "migrated": count,
		}); writeErr != nil {
			return writeErr
		}
		return err
	}
	list := func(args []string) error {
		fs := flagSet("hooks list")
		if err := fs.Parse(args); err != nil {
			return err
		}
		target, err := cfg.Target(scope(fs))
		if err != nil {
			return err
		}
		entries, err := broker.List([]HookTarget{target})
		if err != nil {
			return err
		}
		return writeHookJSON(cfg.Stdout, map[string]any{"agent": cfg.Agent, "hooks": entries})
	}
	return cliapp.SubcommandGroup{
		Name:        "hooks",
		Description: cfg.Description,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List Vrooli-managed hooks", Run: list},
			{Name: "reconcile", Description: "Add or update an identified hook (JSON)", Run: reconcile},
			{Name: "remove", Description: "Remove an identified hook", Run: remove},
			{Name: "migrate", Description: "Adopt legacy Vrooli hook entries", Run: migrate},
		},
	}
}

func writeHookJSON(out io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "%s\n", data)
	return err
}
