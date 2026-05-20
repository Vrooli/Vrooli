package aisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	repocontract "github.com/vrooli/repo-contract-go"
)

// CommandSource enumerates record provenance. Match the values in
// CommandRecord.Source so payload-hash inputs stay consistent.
const (
	SourceManifest   = "manifest"
	SourceHelp       = "help"
	SourceHelpFailed = "help-failed"
)

// DiscoverySource produces the canonical command set for indexing. Tests
// substitute fakes via NewDiscoveryServiceWithSource.
type DiscoverySource interface {
	Discover(ctx context.Context, scenario string) ([]CommandRecord, error)
	ListScenarios(ctx context.Context) ([]string, error)
}

// FilesystemDiscoverySource walks the repo's scenarios/ tree, reads
// cli/manifest.json when present, and falls back to invoking the scenario
// CLI binary with `--help` when not. Help-parse failures emit a single
// help-failed record so the scenario remains discoverable by name (per
// plan §11 — "never crash indexing").
type FilesystemDiscoverySource struct {
	RepoRoot      string
	HelpTimeout   time.Duration
	HelpBinaryEnv string // optional override; when set, this env var holds the binary path
}

// NewFilesystemDiscoverySource returns a discovery source rooted at repoRoot.
func NewFilesystemDiscoverySource(repoRoot string) *FilesystemDiscoverySource {
	return &FilesystemDiscoverySource{
		RepoRoot:    repoRoot,
		HelpTimeout: 5 * time.Second,
	}
}

// ListScenarios returns every directory under scenarios/ in the repo.
func (d *FilesystemDiscoverySource) ListScenarios(_ context.Context) ([]string, error) {
	if strings.TrimSpace(d.RepoRoot) == "" {
		return nil, fmt.Errorf("repo root is required")
	}
	entries, err := os.ReadDir(filepath.Join(d.RepoRoot, "scenarios"))
	if err != nil {
		return nil, fmt.Errorf("read scenarios/: %w", err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

// Discover returns the canonical command records for a single scenario.
// Manifest-first; help fallback only when no manifest exists.
func (d *FilesystemDiscoverySource) Discover(ctx context.Context, scenario string) ([]CommandRecord, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, fmt.Errorf("scenario is required")
	}

	path, err := repocontract.ScenarioCLIManifestPath(d.RepoRoot, scenario)
	if err != nil {
		return nil, fmt.Errorf("resolve manifest path: %w", err)
	}

	raw, err := os.ReadFile(path)
	if err == nil {
		return parseManifestRecords(scenario, raw)
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	return d.helpFallback(ctx, scenario), nil
}

func parseManifestRecords(scenario string, raw []byte) ([]CommandRecord, error) {
	m, err := cliapp.ParseManifest(raw)
	if err != nil {
		return nil, fmt.Errorf("parse manifest for %s: %w", scenario, err)
	}
	if m == nil {
		return nil, nil
	}
	records := make([]CommandRecord, 0, 16)
	for _, group := range m.Groups {
		groupName := strings.TrimSpace(group.Name)
		for _, cmd := range group.Commands {
			rec := CommandRecord{
				Scenario:    scenario,
				Group:       groupName,
				Name:        strings.TrimSpace(cmd.Name),
				Description: strings.TrimSpace(cmd.Description),
				Source:      SourceManifest,
			}
			rec.FullPath = canonicalFullPath(scenario, groupName, rec.Name)

			for _, f := range cmd.Flags {
				if name := strings.TrimSpace(f.Name); name != "" {
					rec.Flags = append(rec.Flags, name)
				}
			}
			for _, p := range cmd.Positionals {
				if name := strings.TrimSpace(p.Name); name != "" {
					rec.Positionals = append(rec.Positionals, name)
				}
			}

			if svc := strings.TrimSpace(cmd.Binding.Service); svc != "" {
				rec.Binding = svc + "." + strings.TrimSpace(cmd.Binding.Method)
			}

			rec.Tags = composeTags(groupName, cmd)
			records = append(records, rec)
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].FullPath < records[j].FullPath })
	return records, nil
}

func composeTags(group string, cmd cliapp.ManifestCommand) []string {
	tags := make([]string, 0, 4)
	if group != "" {
		tags = append(tags, group)
	}
	if eff := strings.TrimSpace(cmd.Governance.Effect); eff != "" {
		tags = append(tags, "effect:"+eff)
	}
	if cmd.Governance.RunEligible {
		tags = append(tags, "run-eligible")
	}
	return tags
}

func canonicalFullPath(scenario, group, name string) string {
	parts := []string{scenario}
	if group != "" {
		parts = append(parts, group)
	}
	if name != "" {
		parts = append(parts, name)
	}
	return strings.Join(parts, " ")
}

// helpFallback returns a single record per scenario, derived from a
// best-effort invocation of `<binary> --help`. On any failure the record is
// emitted with Source=help-failed so the scenario is still discoverable.
func (d *FilesystemDiscoverySource) helpFallback(ctx context.Context, scenario string) []CommandRecord {
	bin := d.resolveBinary(scenario)
	rec := CommandRecord{
		Scenario: scenario,
		Name:     scenario,
		FullPath: scenario,
		Source:   SourceHelpFailed,
	}
	if bin == "" {
		rec.Description = fmt.Sprintf("Scenario %s has no CLI manifest and no binary on PATH; index entry is a stub.", scenario)
		return []CommandRecord{rec}
	}
	timeout := d.HelpTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, bin, "--help")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		rec.Description = fmt.Sprintf("Help invocation failed: %v", err)
		return []CommandRecord{rec}
	}
	out := strings.TrimSpace(stdout.String())
	if out == "" {
		rec.Description = "Help invocation returned no output"
		return []CommandRecord{rec}
	}
	rec.Source = SourceHelp
	rec.Description = truncateForEmbedding(firstNonEmptyLine(out)+"\n"+out, 1800)
	return []CommandRecord{rec}
}

func (d *FilesystemDiscoverySource) resolveBinary(scenario string) string {
	if d.HelpBinaryEnv != "" {
		if p := strings.TrimSpace(os.Getenv(d.HelpBinaryEnv)); p != "" {
			return p
		}
	}
	if p, err := exec.LookPath(scenario); err == nil {
		return p
	}
	return ""
}

func firstNonEmptyLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if t != "" {
			return t
		}
	}
	return ""
}

func truncateForEmbedding(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// MarshalForDebug is a small helper used by handlers when serializing the
// payload for logs without leaking the raw embedding text.
func MarshalForDebug(r CommandRecord) string {
	b, _ := json.Marshal(r)
	return string(b)
}
