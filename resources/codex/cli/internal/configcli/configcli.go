package configcli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/vrooli/agentharness"
	"github.com/vrooli/cli-core/agentcatalog"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/resources/codex/cli/internal/permissions"
	"github.com/vrooli/vrooli/resources/internal/agentapp"
)

type Handlers struct {
	CatalogPath string
	GetEnv      func(string) string
	Stdout      io.Writer
	Stderr      io.Writer
}

func Default() *Handlers {
	return &Handlers{GetEnv: os.Getenv, Stdout: os.Stdout, Stderr: os.Stderr}
}

func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "config",
		Description: "Resolve credential sources and ensure Codex configuration",
		Subcommands: []cliapp.Command{
			{
				Name:        "ensure",
				Description: "Check available credential sources and print active source",
				Usage:       "resource-codex config ensure",
				Run:         h.Ensure,
			},
		},
	}
}

func (h *Handlers) Ensure(args []string) error {
	fs := flag.NewFlagSet("config ensure", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	format := fs.String("format", "text", "Output format: text or json")
	role := fs.String("role", "code.default", "Role to resolve model for")
	if err := fs.Parse(args); err != nil {
		return err
	}

	getenv := h.GetEnv
	if getenv == nil {
		getenv = os.Getenv
	}

	path := h.CatalogPath
	if path == "" {
		path = agentharness.ResourceCatalogPath("codex")
	}
	catalog, err := agentcatalog.ReadCodingRoleCatalog("codex", path)
	if err != nil {
		return err
	}
	selected, ok := catalog.Roles[*role]
	if !ok {
		return fmt.Errorf("unknown resource role %q", *role)
	}
	model := selected.Model
	if override := strings.TrimSpace(getenv("CODEX_MODEL")); override != "" && override != model && override != selected.CanonicalModel {
		return fmt.Errorf("CODEX_MODEL %q differs from role %s model %s; select the matching resource role", override, *role, model)
	}
	effort := strings.TrimSpace(getenv("CODEX_EFFORT"))
	if effort == "" {
		for _, entry := range selected.Models {
			if entry.Model == model && entry.Effort != nil {
				effort = *entry.Effort
				break
			}
		}
	}
	activeSource := ""
	if catalog.Billing != nil {
		for _, source := range catalog.Billing.Sources {
			if key := strings.TrimSpace(getenv(source.CredentialKey)); len(key) >= 20 {
				activeSource = source.ID
				break
			}
		}
	}

	switch *format {
	case "json":
		return json.NewEncoder(h.Stdout).Encode(map[string]string{"active_source": activeSource, "role": *role, "model": model, "effort": effort})
	default:
		if activeSource == "" {
			fmt.Fprintf(h.Stdout, "No credential source exported; native Codex authentication may be available.\n")
		} else {
			fmt.Fprintf(h.Stdout, "Active source: %s\nModel: %s\nEffort: %s\nRole: %s\n", activeSource, model, effort, *role)
		}
	}
	return nil
}

// NativeSettings binds resources/codex/native-settings.json to the shared
// Codex config. In a Web Console session CODEX_HOME is a per-session home
// whose config.toml links to ~/.codex/config.toml; the shared reconciler
// follows that link so every session sees one managed file.
func NativeSettings() *agentapp.NativeSettings {
	return &agentapp.NativeSettings{
		SettingsPath: func() (string, error) {
			adapter, err := permissions.DefaultAdapter(permissions.ScopeUser)
			if err != nil {
				return "", err
			}
			return adapter.SettingsPath, nil
		},
		Decode: func(data []byte) (map[string]any, error) {
			doc := map[string]any{}
			return doc, toml.Unmarshal(data, &doc)
		},
		Encode: func(doc map[string]any) ([]byte, error) { return toml.Marshal(doc) },
	}
}
