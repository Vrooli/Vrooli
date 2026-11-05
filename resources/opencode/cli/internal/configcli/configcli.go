// Package configcli registers `resource-opencode config ...` — the Go
// entrypoint that generates opencode.json (replacing the bash config writer)
// and syncs OpenRouter auth.
package configcli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/resources/opencode/cli/internal/config"
	"github.com/vrooli/vrooli/resources/opencode/cli/internal/secrets"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"

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
	goKey := resolveKey(secretsOpts, "OPENCODE_GO_KEY", "opencode-go")
	haveGo := secrets.KeyUsable(goKey)
	orKey := resolveKey(secretsOpts, "OPENROUTER_API_KEY", "openrouter")
	haveOpenRouter := secrets.KeyUsable(orKey)
	orcaRouterKey := resolveKey(secretsOpts, "ORCAROUTER_API_KEY", "orcarouter")
	haveOrcaRouter := secrets.KeyUsable(orcaRouterKey)
	abliterationKey := resolveKey(secretsOpts, "ABLIT_KEY", "abliteration")
	haveAbliteration := secrets.KeyUsable(abliterationKey)

	changed, err := config.Ensure(ctx, config.EnsureOptions{
		ConfigPath:       configPath(getenv),
		Defaults:         config.DefaultDefaults(getenv),
		HaveOpenCodeGo:   haveGo,
		HaveOpenRouter:   haveOpenRouter,
		HaveOrcaRouter:   haveOrcaRouter,
		HaveAbliteration: haveAbliteration,
		Resolver:         config.ExecResolver{},
		ProviderModels:   providerModels,
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
	for _, provider := range []struct {
		id  string
		key string
	}{
		{id: "orcarouter", key: orcaRouterKey},
		{id: "abliteration", key: abliterationKey},
	} {
		if provider.key == "" {
			continue
		}
		changedAuth, err := secrets.SyncProviderAuth(authPath(getenv), provider.id, provider.key)
		if err != nil {
			return err
		}
		if changedAuth {
			fmt.Fprintf(h.Stdout, "Updated OpenCode auth (%s) at %s\n", provider.id, authPath(getenv))
		}
	}

	if !changed {
		fmt.Fprintln(h.Stdout, "opencode.json already current")
	}
	return nil
}

// resolveKey reads a provider key from the process environment or the OpenCode
// auth file first. If neither has it, it asks the credential authority, which is
// where the operator actually provisioned the key. Without this fallback a
// provider can be configured for Vrooli and still be absent from OpenCode until
// a lifecycle process happens to inject the value.
func resolveKey(opts secrets.Options, envVar, providerID string) string {
	if key := secrets.ResolveProviderKey(opts, envVar, providerID); key != "" {
		return key
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return ""
	}
	client, err := credentialclient.NewClient(credentialclient.ClientOptions{Authority: authority})
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	key, err := client.Resolve(ctx, "vrooli/"+providerID, "api-key")
	if err != nil || !secrets.KeyUsable(key) {
		return ""
	}
	return strings.TrimSpace(key)
}

// providerModels asks the owning provider resource for its chat-capable model
// catalog. The provider policy is the single source of truth, so OpenCode's
// custom provider block and the launcher's model picker cannot drift apart.
func providerModels(ctx context.Context, providerID string) ([]config.ProviderModel, error) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return nil, fmt.Errorf("provider id is required")
	}
	out, err := exec.CommandContext(ctx, "resource-"+providerID, "policy", "models", "--json").Output()
	if err != nil {
		return nil, fmt.Errorf("resource-%s policy models: %w", providerID, err)
	}
	var payload struct {
		Models []struct {
			Model        string   `json:"model"`
			DisplayName  string   `json:"display_name"`
			Endpoints    []string `json:"endpoints"`
			Capabilities []string `json:"capabilities"`
		} `json:"models"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil, fmt.Errorf("decode resource-%s model catalog: %w", providerID, err)
	}
	seen := map[string]bool{}
	models := make([]config.ProviderModel, 0, len(payload.Models))
	for _, item := range payload.Models {
		model := strings.TrimSpace(item.Model)
		if model == "" || seen[model] || !servesChat(item.Endpoints, item.Capabilities) {
			continue
		}
		seen[model] = true
		models = append(models, config.ProviderModel{ID: model, Name: strings.TrimSpace(item.DisplayName)})
	}
	return models, nil
}

func servesChat(endpoints, capabilities []string) bool {
	for _, values := range [][]string{endpoints, capabilities} {
		for _, value := range values {
			if strings.EqualFold(strings.TrimSpace(value), "chat") {
				return true
			}
		}
	}
	return false
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
