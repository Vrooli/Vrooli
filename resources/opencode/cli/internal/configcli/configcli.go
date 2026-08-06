// Package configcli registers `resource-opencode config ...` — the Go
// entrypoint that generates opencode.json (replacing the bash config writer)
// and syncs OpenRouter auth.
package configcli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"resource-opencode/cli/internal/config"
	"resource-opencode/cli/internal/secrets"

	"github.com/vrooli/cli-core/cliapp"
)

// Handlers owns the runtime dependencies for the `config` subcommand group.
type Handlers struct {
	GetEnv func(string) string
	Stdout io.Writer
	Stderr io.Writer
}

// Default returns Handlers wired to the process environment.
func Default() *Handlers {
	return &Handlers{GetEnv: os.Getenv, Stdout: os.Stdout, Stderr: os.Stderr}
}

// Commands returns the `config` subcommand group for registration.
func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "config",
		Description: "Generate and self-heal opencode.json (provider, local Ollama block, sampling)",
		Subcommands: []cliapp.Command{
			{
				Name:        "ensure",
				Description: "Resolve secrets + write/self-heal opencode.json and OpenRouter auth",
				Usage:       "resource-opencode config ensure",
				Run:         h.Ensure,
			},
		},
	}
}

// Ensure resolves the ephemeral OpenRouter injection and writes non-secret
// OpenCode configuration. It never persists a provider key.
func (h *Handlers) Ensure(args []string) error {
	fs := flag.NewFlagSet("config ensure", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	getenv := h.GetEnv
	if getenv == nil {
		getenv = os.Getenv
	}
	ctx := context.Background()

	key := secrets.ResolveOpenRouterKey(secrets.Options{Getenv: getenv})
	haveOpenRouter := secrets.KeyUsable(key)

	changed, err := config.Ensure(ctx, config.EnsureOptions{
		ConfigPath:     configPath(getenv),
		Defaults:       config.DefaultDefaults(getenv),
		HaveOpenRouter: haveOpenRouter,
		Resolver:       config.ExecResolver{},
		Logf: func(format string, a ...any) {
			fmt.Fprintf(h.Stdout, format+"\n", a...)
		},
	})
	if err != nil {
		return err
	}
	if !changed {
		fmt.Fprintln(h.Stdout, "opencode.json already current")
	}
	return nil
}

func xdgConfigHome(getenv func(string) string) string {
	if v := getenv("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}

func configPath(getenv func(string) string) string {
	return filepath.Join(xdgConfigHome(getenv), "opencode", "opencode.json")
}
