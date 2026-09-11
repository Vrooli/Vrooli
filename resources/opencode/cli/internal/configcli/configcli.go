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

	"github.com/vrooli/vrooli/resources/opencode/cli/internal/config"
	"github.com/vrooli/vrooli/resources/opencode/cli/internal/secrets"

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

	secretsOpts := secrets.Options{Getenv: getenv, AuthPath: authPath(getenv)}
	goKey := secrets.ResolveOpenCodeGoKey(secretsOpts)
	haveGo := secrets.KeyUsable(goKey)
	orKey := secrets.ResolveOpenRouterKey(secretsOpts)
	haveOpenRouter := secrets.KeyUsable(orKey)

	changed, err := config.Ensure(ctx, config.EnsureOptions{
		ConfigPath:      configPath(getenv),
		Defaults:        config.DefaultDefaults(getenv),
		HaveOpenCodeGo:  haveGo,
		HaveOpenRouter:  haveOpenRouter,
		Resolver:        config.ExecResolver{},
		Logf: func(format string, a ...any) {
			fmt.Fprintf(h.Stdout, format+"\n", a...)
		},
	})
	if err != nil {
		return err
	}

	// Sync available keys into the auth file.
	if goKey != "" {
		changedAuth, err := secrets.SyncOpenCodeGoAuth(authPath(getenv), goKey)
		if err != nil {
			return err
		}
		if changedAuth {
			fmt.Fprintf(h.Stdout, "Updated OpenCode auth (opencode-go) at %s\n", authPath(getenv))
		}
	}
	if orKey != "" {
		changedAuth, err := secrets.SyncOpenRouterAuth(authPath(getenv), orKey)
		if err != nil {
			return err
		}
		if changedAuth {
			fmt.Fprintf(h.Stdout, "Updated OpenCode auth (openrouter) at %s\n", authPath(getenv))
		}
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
	if dir := getenv("OPENCODE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "opencode.json")
	}
	return filepath.Join(xdgConfigHome(getenv), "opencode", "opencode.json")
}

func authPath(getenv func(string) string) string {
	if dataHome := getenv("OPENCODE_XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "opencode", "auth.json")
	}
	dataHome := getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home := getenv("HOME")
		if home == "" {
			home, _ = os.UserHomeDir()
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "opencode", "auth.json")
}
