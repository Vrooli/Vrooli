package aisearch

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/measures-go/manifestscan"

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
	ListExternalCLIs() []ExternalCLI
	DiscoverExternal(ctx context.Context, cli ExternalCLI) ([]CommandRecord, error)
}

// ExternalCLI is a non-scenario CLI (e.g. the top-level `vrooli` binary)
// that should be indexed alongside scenario CLIs. Origin records under
// these entries carry the configured Name (not a scenario id).
type ExternalCLI struct {
	Name   string // origin display name (e.g. "vrooli")
	Binary string // exec.LookPath name or absolute path
}

// FilesystemDiscoverySource walks the repo's scenarios/ tree, reads
// cli/manifest.json when present, and falls back to invoking the scenario
// CLI binary with `--help` when not. Help-parse failures emit a single
// help-failed record so the scenario remains discoverable by name (per
// plan §11 — "never crash indexing").
type FilesystemDiscoverySource struct {
	RepoRoot      string
	HelpTimeout   time.Duration
	HelpBinaryEnv string        // optional override; when set, this env var holds the binary path
	ExternalCLIs  []ExternalCLI // non-scenario CLIs (e.g. vrooli) indexed alongside scenarios

	// MeasureSchema resolves proto param schemas for measure blocks. Optional:
	// when nil it defaults (lazily) to a descriptor reader rooted at RepoRoot.
	// Tests inject a stub to avoid touching the committed descriptor image.
	MeasureSchema manifestscan.SchemaSource

	mu        sync.Mutex
	helpCache map[string]helpCacheEntry // keyed by absolute binary path

	measureOnce sync.Once
	measureSrc  manifestscan.SchemaSource
}

// measureSchemaSource returns the configured SchemaSource, lazily defaulting to
// a descriptor reader on RepoRoot (which itself loads the image lazily, so a
// missing descriptor degrades gracefully rather than crashing indexing).
func (d *FilesystemDiscoverySource) measureSchemaSource() manifestscan.SchemaSource {
	if d.MeasureSchema != nil {
		return d.MeasureSchema
	}
	d.measureOnce.Do(func() {
		d.measureSrc = manifestscan.NewDescriptorSchemaReader(d.RepoRoot)
	})
	return d.measureSrc
}

// helpCacheEntry holds the parsed help tree for a binary at a given mtime.
// On reindex the entry is reused when the binary file has not changed since
// it was cached — avoids the recursive subprocess fan-out on every reindex.
type helpCacheEntry struct {
	mtime   time.Time
	origin  string
	records []CommandRecord
}

// NewFilesystemDiscoverySource returns a discovery source rooted at repoRoot.
func NewFilesystemDiscoverySource(repoRoot string) *FilesystemDiscoverySource {
	return &FilesystemDiscoverySource{
		RepoRoot:    repoRoot,
		HelpTimeout: 5 * time.Second,
	}
}

// ListExternalCLIs returns a copy of the configured ExternalCLIs slice.
// Returned slice is safe to mutate.
func (d *FilesystemDiscoverySource) ListExternalCLIs() []ExternalCLI {
	if len(d.ExternalCLIs) == 0 {
		return nil
	}
	out := make([]ExternalCLI, len(d.ExternalCLIs))
	copy(out, d.ExternalCLIs)
	return out
}

// DiscoverExternal walks the given ExternalCLI's --help tree and emits one
// CommandRecord per leaf, with Origin = cli.Name. Mirrors
// helpFallback but does not consult the scenarios/ tree or a manifest.
func (d *FilesystemDiscoverySource) DiscoverExternal(ctx context.Context, cli ExternalCLI) ([]CommandRecord, error) {
	name := strings.TrimSpace(cli.Name)
	if name == "" {
		return nil, fmt.Errorf("external CLI name is required")
	}
	bin := strings.TrimSpace(cli.Binary)
	if bin == "" {
		bin = name
	}
	if resolved, err := exec.LookPath(bin); err == nil {
		bin = resolved
	} else {
		return []CommandRecord{{
			Origin:      name,
			Name:        name,
			FullPath:    name,
			Source:      SourceHelpFailed,
			Description: fmt.Sprintf("External CLI %s: binary %q not found on PATH; index entry is a stub.", name, cli.Binary),
		}}, nil
	}
	return d.parseHelpTreeCached(ctx, bin, name), nil
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
		records, perr := parseManifestRecords(scenario, raw)
		if perr != nil {
			return nil, perr
		}
		attachMeasures(records, raw, d.measureSchemaSource())
		return records, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	return d.helpFallback(ctx, scenario), nil
}

// attachMeasures parses the manifest's measure blocks and joins each onto its
// matching CommandRecord (by group+command). Best-effort: a parse failure or an
// unresolvable proto schema leaves the records untouched / the tier ungraded so
// indexing never crashes (plan §11).
func attachMeasures(records []CommandRecord, raw []byte, src manifestscan.SchemaSource) {
	parsed, err := manifestscan.Parse(raw)
	if err != nil || len(parsed.Commands) == 0 {
		return
	}
	byKey := make(map[string]int, len(records))
	for i := range records {
		byKey[records[i].Group+"\x00"+records[i].Name] = i
	}
	for _, cm := range parsed.Commands {
		idx, ok := byKey[cm.Group+"\x00"+cm.Command]
		if !ok {
			continue
		}
		records[idx].Measure = buildMeasureRecord(cm, src)
	}
}

// buildMeasureRecord projects a CommandMeasure into the discovery MeasureRecord.
// When assembly against the proto schema succeeds it carries the authoritative
// param schema + graded tier; otherwise it degrades to the manifest-authored
// params (names/annotations only) with an empty (ungraded) tier.
func buildMeasureRecord(cm manifestscan.CommandMeasure, src manifestscan.SchemaSource) *MeasureRecord {
	mr := &MeasureRecord{
		Name:       cm.MeasureName(),
		Domain:     cm.Domain,
		Intent:     cm.Measure.Intent,
		Questions:  cm.Measure.Questions,
		ResultKind: string(cm.Measure.Result.Kind),
		ValueField: cm.Measure.Result.ValueField,
		Unit:       cm.Measure.Result.Unit,
		Effect:     string(cm.Governance.Effect),
	}
	if decl, err := cm.Assemble(src); err == nil {
		mr.Tier = string(manifestscan.GradeTier(decl))
		for _, name := range decl.ParamNames() {
			p := decl.Params[name]
			mr.Params = append(mr.Params, MeasureParamRecord{
				Name:         p.Name,
				Type:         p.Type,
				Required:     p.Required,
				EnumValues:   p.EnumValues,
				Default:      p.Default,
				ValuesSource: p.ValuesSource,
			})
		}
		return mr
	}
	// Degraded: surface manifest-authored params so the command stays
	// discoverable; tier left empty (ungraded).
	names := make([]string, 0, len(cm.Measure.Params))
	for n := range cm.Measure.Params {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		mp := cm.Measure.Params[n]
		mr.Params = append(mr.Params, MeasureParamRecord{
			Name:         n,
			Type:         mp.Type,
			Default:      mp.Default,
			ValuesSource: mp.ValuesSource,
		})
	}
	return mr
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
		groupDesc := strings.TrimSpace(group.Description)
		for _, cmd := range group.Commands {
			rec := CommandRecord{
				Origin:      scenario,
				Group:       groupName,
				Name:        strings.TrimSpace(cmd.Name),
				Description: strings.TrimSpace(cmd.Description),
				Source:      SourceManifest,
			}
			// Fold the group's prose into the leaf (same rule as the --help
			// path: dropped when empty or a pure repeat of the leaf desc) so
			// manifest-discovered commands carry the same query vocabulary.
			rec.GroupDescription = groupContext(groupDesc, rec.Description)
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

// helpFallback walks the CLI's `--help` tree recursively and emits one
// CommandRecord per leaf command. On any failure (binary missing, exec
// error, unparseable output) a single Source=help-failed stub is emitted so
// the CLI remains discoverable by name.
func (d *FilesystemDiscoverySource) helpFallback(ctx context.Context, origin string) []CommandRecord {
	bin := d.resolveBinary(origin)
	if bin == "" {
		return []CommandRecord{{
			Origin:      origin,
			Name:        origin,
			FullPath:    origin,
			Source:      SourceHelpFailed,
			Description: fmt.Sprintf("Scenario %s has no CLI manifest and no binary on PATH; index entry is a stub.", origin),
		}}
	}
	return d.parseHelpTreeCached(ctx, bin, origin)
}

// parseHelpTreeCached returns the help-derived CommandRecords for bin,
// reusing a cached parse when the binary's mtime is unchanged. The cache
// lives for the lifetime of the FilesystemDiscoverySource (typically the
// process lifetime).
func (d *FilesystemDiscoverySource) parseHelpTreeCached(ctx context.Context, bin, origin string) []CommandRecord {
	mtime, ok := binaryMtime(bin)
	if ok {
		d.mu.Lock()
		entry, hit := d.helpCache[bin]
		d.mu.Unlock()
		if hit && entry.mtime.Equal(mtime) && entry.origin == origin {
			return cloneRecords(entry.records)
		}
	}

	records := ParseHelpTree(ctx, d.helpRunner(), bin, HelpTreeOptions{Origin: origin, MaxDepth: defaultHelpMaxDepth})

	if ok {
		d.mu.Lock()
		if d.helpCache == nil {
			d.helpCache = make(map[string]helpCacheEntry)
		}
		d.helpCache[bin] = helpCacheEntry{mtime: mtime, origin: origin, records: cloneRecords(records)}
		d.mu.Unlock()
	}
	return records
}

func binaryMtime(bin string) (time.Time, bool) {
	info, err := os.Stat(bin)
	if err != nil {
		return time.Time{}, false
	}
	return info.ModTime(), true
}

func cloneRecords(in []CommandRecord) []CommandRecord {
	out := make([]CommandRecord, len(in))
	copy(out, in)
	return out
}

// helpRunner returns the production helpRunner that shells out to the
// binary, applying HelpTimeout per invocation.
func (d *FilesystemDiscoverySource) helpRunner() helpRunner {
	timeout := d.HelpTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return func(ctx context.Context, bin string, args []string) ([]byte, error) {
		cctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		argv := append(append([]string{}, args...), "--help")
		cmd := exec.CommandContext(cctx, bin, argv...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg != "" {
				return nil, fmt.Errorf("%s: %w", msg, err)
			}
			return nil, err
		}
		return stdout.Bytes(), nil
	}
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

func truncateForEmbedding(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
