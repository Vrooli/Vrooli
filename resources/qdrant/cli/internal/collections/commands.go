package collections

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
)

// Handlers contains the environment and output dependencies for the inventory
// command. Tests inject these values and a fake HTTPDoer; no test reaches Qdrant.
type Handlers struct {
	Client             Client
	GetEnv             func(string) string
	RepoRoot           string
	UserHome           string
	ResolveEnvironment func(root, home, resource string) (map[string]string, error)
	Stdout             io.Writer
	Stderr             io.Writer
	Timeout            time.Duration
}

func Default() *Handlers {
	return &Handlers{GetEnv: os.Getenv, Stdout: os.Stdout, Stderr: os.Stderr, Timeout: 10 * time.Second}
}

func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "collections",
		Description: "Read Qdrant collection and alias inventory",
		Subcommands: []cliapp.Command{{
			Name:        "list",
			Description: "List collection names, aliases, vector dimensions, and point counts",
			Usage:       "resource-qdrant collections list [--base-url <url>] [--json]",
			Run:         h.List,
		}},
	}
}

func (h *Handlers) List(args []string) error {
	if h == nil {
		return errors.New("Qdrant collection inventory handlers are required")
	}
	stdout := h.Stdout
	if stdout == nil {
		stdout = io.Discard
	}
	stderr := h.Stderr
	if stderr == nil {
		stderr = io.Discard
	}
	fs := flag.NewFlagSet("collections list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	baseURL := fs.String("base-url", "", "Explicit Qdrant API base URL (otherwise resolve the registered resource endpoint)")
	jsonOutput := fs.Bool("json", false, "Write machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	getenv := h.GetEnv
	if getenv == nil {
		getenv = os.Getenv
	}
	endpoint := strings.TrimSpace(*baseURL)
	ownerEnvironment := map[string]string{}
	if endpoint == "" {
		root := strings.TrimSpace(h.RepoRoot)
		if root == "" {
			var err error
			root, err = findQdrantRepoRoot(getenv)
			if err != nil {
				return err
			}
		}
		home := strings.TrimSpace(h.UserHome)
		if home == "" {
			home = strings.TrimSpace(getenv("VROOLI_HOME"))
		}
		if home == "" {
			var err error
			home, err = os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("resolve Vrooli home for registered Qdrant endpoint: %w", err)
			}
		}
		resolveEnvironment := h.ResolveEnvironment
		if resolveEnvironment == nil {
			resolveEnvironment = func(root, home, name string) (map[string]string, error) {
				report, err := resourceenv.ResolveResource(root, home, name, resourceenv.ResolveOptions{})
				if err != nil {
					return nil, err
				}
				return report.Values, nil
			}
		}
		var err error
		ownerEnvironment, err = resolveEnvironment(root, home, "qdrant")
		if err != nil {
			return fmt.Errorf("resolve registered Qdrant resource environment: %w", err)
		}
		endpoint = strings.TrimSpace(ownerEnvironment["QDRANT_URL"])
		if endpoint == "" {
			endpoint = strings.TrimSpace(ownerEnvironment["QDRANT_BASE_URL"])
		}
		if endpoint == "" {
			return errors.New("registered Qdrant resource environment has no API endpoint")
		}
	}
	client := h.Client
	client.BaseURL = endpoint
	if strings.TrimSpace(client.APIKey) == "" {
		client.APIKey = strings.TrimSpace(ownerEnvironment["QDRANT_API_KEY"])
		if *baseURL != "" {
			client.APIKey = strings.TrimSpace(getenv("QDRANT_API_KEY"))
		}
	}
	timeout := h.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	inventory, err := client.List(ctx)
	if err != nil {
		return err
	}
	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(inventory)
	}
	for _, collection := range inventory.Collections {
		aliases := "-"
		if len(collection.Aliases) > 0 {
			aliases = strings.Join(collection.Aliases, ",")
		}
		points := "unknown"
		if collection.PointsCount != nil {
			points = fmt.Sprint(*collection.PointsCount)
		}
		fmt.Fprintf(stdout, "%s\taliases=%s\tstatus=%s\tpoints=%s", collection.Name, aliases, valueOrUnknown(collection.Status), points)
		for _, vector := range collection.Vectors {
			fmt.Fprintf(stdout, "\tvector=%s:%d/%s", vector.Name, vector.Size, valueOrUnknown(vector.Distance))
		}
		fmt.Fprintln(stdout)
	}
	return nil
}

func findQdrantRepoRoot(getenv func(string) string) (string, error) {
	candidates := []string{
		strings.TrimSpace(getenv("VROOLI_ROOT")),
		strings.TrimSpace(getenv("VROOLI_SOURCE_ROOT")),
	}
	for _, key := range []string{"VROOLI_CLI_SOURCE_ROOT", "QDRANT_CLI_SOURCE_ROOT"} {
		if candidate := strings.TrimSpace(getenv(key)); candidate != "" {
			candidates = append(candidates, candidate)
		}
	}
	if current, err := os.Getwd(); err == nil {
		candidates = append(candidates, current)
	}
	for _, candidate := range candidates {
		for path := filepath.Clean(candidate); path != "." && path != string(filepath.Separator); path = filepath.Dir(path) {
			if fileExists(filepath.Join(path, "resources", "qdrant", "resource.json")) {
				return path, nil
			}
			parent := filepath.Dir(path)
			if parent == path {
				break
			}
		}
	}
	return "", errors.New("cannot resolve registered Qdrant endpoint: set VROOLI_ROOT to the Vrooli checkout")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}
