package content

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// defaultField is the secret field used when --key is omitted, matching the
// single-value convention in the vault QUICKSTART docs.
const defaultField = "value"

// Handlers owns the runtime dependencies for the content command group. Tests
// inject a fake Runner and capture Stdout/Stderr.
type Handlers struct {
	Runner Runner
	Stdout io.Writer
	Stderr io.Writer
}

// Default returns Handlers wired to the real docker-backed Runner.
func Default() *Handlers {
	return &Handlers{
		Runner: NewDockerRunner(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// Commands returns the `content` subcommand group for registration with
// cli-core.
func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "content",
		Description: "Read and write secrets in the Vault KV store (the supported secret interface; never call Vault's HTTP API directly)",
		Subcommands: []cliapp.Command{
			{
				Name:        "get",
				Description: "Read a secret field (raw) or the whole secret (json)",
				Usage:       "resource-vault content get --path <kv-path> [--key <field>] [--format raw|json]",
				Run:         h.Get,
			},
			{
				Name:        "set",
				Aliases:     []string{"add"},
				Description: "Write/update a secret field, preserving sibling fields",
				Usage:       "resource-vault content set --path <kv-path> [--key <field>] --value <value>",
				Run:         h.Set,
			},
			{
				Name:        "add",
				Description: "Alias for set",
				Usage:       "resource-vault content add --path <kv-path> [--key <field>] --value <value>",
				Run:         h.Set,
			},
			{
				Name:        "delete",
				Aliases:     []string{"remove"},
				Description: "Delete a secret at a path",
				Usage:       "resource-vault content delete --path <kv-path>",
				Run:         h.Delete,
			},
			{
				Name:        "remove",
				Description: "Alias for delete",
				Usage:       "resource-vault content remove --path <kv-path>",
				Run:         h.Delete,
			},
			{
				Name:        "list",
				Description: "List secret keys under a path prefix",
				Usage:       "resource-vault content list --path <kv-prefix>",
				Run:         h.List,
			},
		},
	}
}

func (h *Handlers) flagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	return fs
}

// Get reads a secret. With --format raw (default) it prints the bare value of
// --key (exit non-zero if the path/field is absent, so callers can treat a
// non-zero exit as "not found"). With --format json it prints Vault's JSON for
// the whole secret.
func (h *Handlers) Get(args []string) error {
	fs := h.flagSet("content get")
	path := fs.String("path", "", "KV path (e.g. secret/resources/kopia/...)")
	key := fs.String("key", "", "Field within the secret (required for --format raw)")
	format := fs.String("format", "raw", "Output format: raw|json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*path) == "" {
		return fmt.Errorf("--path is required")
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	ctx := context.Background()
	switch *format {
	case "json":
		stdout, stderr, err := h.Runner.Run(ctx, []string{"kv", "get", "-format=json", *path}, nil)
		if err != nil {
			return fmt.Errorf("vault kv get %s: %w%s", *path, err, formatStderr(stderr))
		}
		_, err = h.Stdout.Write(stdout)
		return err
	case "raw":
		field := strings.TrimSpace(*key)
		if field == "" {
			field = defaultField
		}
		stdout, stderr, err := h.Runner.Run(ctx, []string{"kv", "get", "-field=" + field, *path}, nil)
		if err != nil {
			return fmt.Errorf("vault kv get %s/%s: %w%s", *path, field, err, formatStderr(stderr))
		}
		_, err = h.Stdout.Write(stdout)
		return err
	default:
		return fmt.Errorf("unsupported --format %q (want raw|json)", *format)
	}
}

// Set writes a single field, preserving any sibling fields at the same path. It
// attempts `kv patch` (which merges into an existing secret) and falls back to
// `kv put` when the secret does not yet exist.
func (h *Handlers) Set(args []string) error {
	fs := h.flagSet("content set")
	path := fs.String("path", "", "KV path (e.g. secret/resources/kopia/...)")
	key := fs.String("key", "", "Field to write (defaults to \"value\")")
	value := fs.String("value", "", "Value to store")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*path) == "" {
		return fmt.Errorf("--path is required")
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	if strings.HasPrefix(*value, "@") {
		// Vault treats a leading @ as a file reference; refuse rather than read a file.
		return fmt.Errorf("values beginning with '@' are not supported")
	}
	field := strings.TrimSpace(*key)
	if field == "" {
		field = defaultField
	}
	ctx := context.Background()
	kv := field + "=" + *value
	// Merge into an existing secret if present.
	if _, _, err := h.Runner.Run(ctx, []string{"kv", "patch", *path, kv}, nil); err == nil {
		fmt.Fprintf(h.Stdout, "updated %s at %s\n", field, *path)
		return nil
	}
	// Otherwise create the secret.
	if _, stderr, err := h.Runner.Run(ctx, []string{"kv", "put", *path, kv}, nil); err != nil {
		return fmt.Errorf("vault kv put %s: %w%s", *path, err, formatStderr(stderr))
	}
	fmt.Fprintf(h.Stdout, "stored %s at %s\n", field, *path)
	return nil
}

// Delete removes the secret at a path.
func (h *Handlers) Delete(args []string) error {
	fs := h.flagSet("content delete")
	path := fs.String("path", "", "KV path to delete")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*path) == "" {
		return fmt.Errorf("--path is required")
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	ctx := context.Background()
	if _, stderr, err := h.Runner.Run(ctx, []string{"kv", "delete", *path}, nil); err != nil {
		return fmt.Errorf("vault kv delete %s: %w%s", *path, err, formatStderr(stderr))
	}
	fmt.Fprintf(h.Stdout, "deleted %s\n", *path)
	return nil
}

// List lists secret keys under a path prefix.
func (h *Handlers) List(args []string) error {
	fs := h.flagSet("content list")
	path := fs.String("path", "", "KV path prefix to list")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*path) == "" {
		return fmt.Errorf("--path is required")
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	ctx := context.Background()
	stdout, stderr, err := h.Runner.Run(ctx, []string{"kv", "list", *path}, nil)
	if err != nil {
		return fmt.Errorf("vault kv list %s: %w%s", *path, err, formatStderr(stderr))
	}
	_, err = h.Stdout.Write(stdout)
	return err
}

func formatStderr(stderr []byte) string {
	msg := strings.TrimSpace(string(stderr))
	if msg == "" {
		return ""
	}
	return ": " + msg
}
