package configcli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

type Handlers struct {
	GetEnv func(string) string
	Stdout io.Writer
	Stderr io.Writer
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
		Description: "Resolve credential sources and ensure Claude Code configuration",
		Subcommands: []cliapp.Command{
			{
				Name:        "ensure",
				Description: "Check available credential sources and print active source",
				Usage:       "resource-claude-code config ensure",
				Run:         h.Ensure,
			},
		},
	}
}

type credentialSource struct {
	ID         string
	Label      string
	EnvVar     string
	BillingMode string
}

var sources = []credentialSource{
	{ID: "claude-subscription", Label: "Claude Subscription", EnvVar: "ANTHROPIC_API_KEY", BillingMode: "subscription"},
	{ID: "opencode-go", Label: "OpenCode Go", EnvVar: "OPENCODE_GO_KEY", BillingMode: "subscription"},
	{ID: "openrouter", Label: "OpenRouter API", EnvVar: "OPENROUTER_API_KEY", BillingMode: "metered"},
}

func keyUsable(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	return len(key) >= 20
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

	activeSource := ""
	for _, s := range sources {
		if key := strings.TrimSpace(getenv(s.EnvVar)); keyUsable(key) {
			activeSource = s.ID
			break
		}
	}

	model := strings.TrimSpace(getenv("CLAUDE_MODEL"))
	if model == "" {
		model = "sonnet"
	}

	switch *format {
	case "json":
		fmt.Fprintf(h.Stdout, `{"active_source":"%s","role":"%s","model":"%s"}`, activeSource, *role, model)
	default:
		if activeSource == "" {
			fmt.Fprintf(h.Stdout, "No usable credential source found. Set ANTHROPIC_API_KEY, OPENCODE_GO_KEY, or OPENROUTER_API_KEY.\n")
		} else {
			fmt.Fprintf(h.Stdout, "Active source: %s (%s)\nModel: %s\nRole: %s\n", activeSource, sourceLabel(activeSource), model, *role)
		}
	}
	return nil
}

func sourceLabel(id string) string {
	for _, s := range sources {
		if s.ID == id {
			return s.Label
		}
	}
	return id
}