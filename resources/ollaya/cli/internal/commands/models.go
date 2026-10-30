package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

func ModelCommands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "models",
		Description: "Pull and inspect digest-pinned decision models in the managed daemon",
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List pinned models and whether the daemon holds the pinned digest", Usage: "resource-ollaya models list [--json]", Run: h.ListModels},
			{Name: "pull", Description: "Pull pinned models by digest (default: every default_pull model)", Usage: "resource-ollaya models pull [--model <name>] [--timeout 30m] [--json]", Run: h.PullModels},
		},
	}
}

type modelRow struct {
	Name           string `json:"name"`
	PinnedDigest   string `json:"pinned_digest"`
	LocalDigest    string `json:"local_digest,omitempty"`
	Installed      bool   `json:"installed"`
	MatchesPinned  bool   `json:"matches_pinned"`
	DefaultPull    bool   `json:"default_pull"`
	DiskBytes      int64  `json:"disk_bytes"`
	License        string `json:"license"`
	InstalledError string `json:"error,omitempty"`
}

func (h *Handlers) ListModels(args []string) error {
	fs := flag.NewFlagSet("models list", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	jsonOutput := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	p, err := h.policy()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	installed, localErr := h.client().LocalModels(ctx)
	rows := h.modelRows(p, installed, localErr)
	if *jsonOutput {
		return writeJSON(h.Stdout, map[string]any{"models": rows})
	}
	for _, row := range rows {
		fmt.Fprintf(h.Stdout, "%-20s installed=%t pinned=%t default_pull=%t disk_bytes=%d\n", row.Name, row.Installed, row.MatchesPinned, row.DefaultPull, row.DiskBytes)
	}
	return localErr
}

func (h *Handlers) PullModels(args []string) error {
	fs := flag.NewFlagSet("models pull", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	model := fs.String("model", "", "Pinned model name; omit for every default_pull model")
	timeout := fs.Duration("timeout", 30*time.Minute, "Overall pull deadline")
	jsonOutput := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	p, err := h.policy()
	if err != nil {
		return err
	}
	names := p.DefaultPulls()
	if name := strings.TrimSpace(*model); name != "" {
		pinned, ok := p.Models[name]
		if !ok {
			return fmt.Errorf("model %q is not pinned in model-policy.json", name)
		}
		// A router needs its targets; pull them after it so their own digest
		// pins replace the tags the router pull fetched.
		names = append([]string{name}, pinned.Routes...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	client := h.client()
	for _, name := range names {
		// The tag is explicit: an untagged digest pull is stored under the
		// digest name, not the name role resolution looks up.
		reference := upstream.InstalledName(name) + "@sha256:" + p.Models[name].Digest
		fmt.Fprintf(h.Stderr, "pulling %s\n", reference)
		if err := client.Pull(ctx, reference); err != nil {
			return h.fail(err)
		}
	}
	installed, err := client.LocalModels(ctx)
	if err != nil {
		return h.fail(err)
	}
	rows := h.modelRows(p, installed, nil)
	var mismatched []string
	for _, row := range rows {
		for _, name := range names {
			if row.Name == name && !row.MatchesPinned {
				mismatched = append(mismatched, fmt.Sprintf("%s local=%s pinned=%s", name, row.LocalDigest, row.PinnedDigest))
			}
		}
	}
	if *jsonOutput {
		if err := writeJSON(h.Stdout, map[string]any{"pulled": names, "models": rows}); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(h.Stdout, "pulled: %s\n", strings.Join(names, ", "))
	}
	if len(mismatched) > 0 {
		return errors.New("pulled models do not match their pinned digests: " + strings.Join(mismatched, "; "))
	}
	return nil
}

func (h *Handlers) modelRows(p policy.Policy, installed map[string]string, localErr error) []modelRow {
	rows := make([]modelRow, 0, len(p.Models))
	for _, name := range sortedModelNames(p) {
		model := p.Models[name]
		row := modelRow{Name: name, PinnedDigest: model.Digest, DefaultPull: model.DefaultPull, DiskBytes: model.DiskBytes, License: model.License}
		if localErr != nil {
			row.InstalledError = localErr.Error()
		} else if digest, ok := installed[upstream.InstalledName(name)]; ok {
			row.Installed, row.LocalDigest, row.MatchesPinned = true, digest, digest == model.Digest
		}
		rows = append(rows, row)
	}
	return rows
}

func sortedModelNames(p policy.Policy) []string {
	names := make([]string, 0, len(p.Models))
	for name := range p.Models {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
