// Package policybridge publishes storage-manager's filesystem dispositions as
// an agent-policy provider snapshot: where the repository contract declares
// deletion safe, where it needs an owner's confirmation, and where it declares
// data durable. The shared agent policy runtime owns the decision and its
// floor; this package only states what the declarations say.
package policybridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	corestorage "github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	"storage-manager/internal/cleanup"
	"storage-manager/internal/providers"
)

const (
	ProviderID = "storage-manager"
	// PublishInterval is how often the snapshot is refreshed; snapshotLifetime
	// outlives several refreshes so one missed pass does not expire it.
	PublishInterval  = 30 * time.Minute
	snapshotLifetime = 2 * time.Hour
	publishTimeout   = 30 * time.Second
	runnerName       = "vrooli-policy-runner"
)

// Declaration precedence when two declarations name the same root: the
// contract's governed roots carry an explicit safety tier, runtime-home entries
// a cleanup owner, owner storage entries only regenerability.
const (
	priorityOwnerEntry = iota + 1
	priorityRuntimeHome
	priorityStorageRoot
)

// PathRule is the agent-policy path_rules wire shape.
type PathRule struct {
	Root     string `json:"root"`
	Action   string `json:"action"`
	Reason   string `json:"reason"`
	Source   string `json:"source"`
	Owner    string `json:"owner,omitempty"`
	priority int
}

// SnapshotSink receives a provider snapshot for the agent-policy bundle.
type SnapshotSink interface {
	PublishProviderSnapshot(context.Context, []byte) error
}

// Declarations resolves every filesystem declaration this host can see into
// path rules. warnings name declarations that could not be resolved; the
// runtime's floor governs those locations instead.
func Declarations(repoRoot string) ([]PathRule, []string, error) {
	inputs, err := corestorage.HostGovernedRootInputs(repoRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve governed root inputs: %w", err)
	}
	specs, err := providers.LoadRootSpecs(repoRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("load storage roots: %w", err)
	}
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("load repo contract: %w", err)
	}
	entries, err := contract.RuntimeHomeEntries(inputs.Home)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve runtime-home entries: %w", err)
	}
	inventory, err := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: repoRoot})
	if err != nil {
		return nil, nil, fmt.Errorf("load owner storage declarations: %w", err)
	}
	var warnings []string
	rules := rootRules(specs, inputs, &warnings)
	rules = append(rules, runtimeHomeRules(entries, inputs.RuntimeHome)...)
	rules = append(rules, ownerRules(repoRoot, inventory.Owners, &warnings)...)
	return mergeRules(rules), warnings, nil
}

// tierAction maps a storage safety tier to what an agent may do there.
func tierAction(tier cleanup.SafetyTier) string {
	switch tier {
	case cleanup.SafetyTierSafe, cleanup.SafetyTierRegenerable:
		return "allow"
	case cleanup.SafetyTierSafeWithOwner, cleanup.SafetyTierConditional:
		return "ask"
	default:
		return "deny"
	}
}

func rootRules(specs []providers.RootSpec, inputs corestorage.GovernedRootInputs, warnings *[]string) []PathRule {
	rules := make([]PathRule, 0, len(specs))
	for _, spec := range specs {
		if !spec.Applicable() {
			continue
		}
		root, err := corestorage.ResolveGovernedRootStrict(spec.Root, inputs)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("storage root %s: %v", spec.ID, err))
			continue
		}
		if strings.ContainsAny(filepath.Base(root), "*?[") {
			// A pattern root is expanded at reap time; a path rule names one
			// location, so the floor governs these instead.
			*warnings = append(*warnings, fmt.Sprintf("storage root %s is a pattern and is left to the floor", spec.ID))
			continue
		}
		rules = append(rules, PathRule{
			Root: root, Action: tierAction(spec.Tier), Reason: string(spec.Tier) + " " + spec.Class,
			Source: "repo-contract storage.roots/" + spec.ID, priority: priorityStorageRoot,
		})
	}
	return rules
}

func runtimeHomeRules(entries []repocontract.HomeEntry, runtimeHome string) []PathRule {
	rules := make([]PathRule, 0, len(entries))
	for _, entry := range entries {
		rule := PathRule{
			Root:   filepath.Join(runtimeHome, filepath.FromSlash(entry.RelPath)),
			Source: "repo-contract runtime_home/" + entry.Key, Owner: entry.Owner, priority: priorityRuntimeHome,
		}
		switch {
		case entry.Protected || entry.Sensitive || entry.Cleanup == "never":
			rule.Action, rule.Reason = "deny", "protected runtime-home "+entry.Key
		case entry.Cleanup == "storage_manager" && entry.Regenerable:
			rule.Action, rule.Reason = "allow", "regenerable runtime-home "+entry.Key+" that storage-manager reaps"
		default:
			rule.Action, rule.Reason = "ask", "runtime-home "+entry.Key+" has no declared cleanup owner"
		}
		rules = append(rules, rule)
	}
	return rules
}

func ownerRules(repoRoot string, owners []corestorage.OwnerManifest, warnings *[]string) []PathRule {
	platform := corestorage.HostPlatform()
	var rules []PathRule
	for _, owner := range owners {
		ownerName := string(owner.Kind) + "/" + owner.ID
		for _, entry := range owner.StorageEntries {
			if !containsPlatform(corestorage.EffectivePlatforms(owner, entry), platform) {
				continue
			}
			path, err := corestorage.ResolveOwnerStoragePath(repoRoot, owner, entry, platform, corestorage.PlatformSeams{})
			var notApplicable *corestorage.NotApplicable
			if errors.As(err, &notApplicable) {
				continue
			}
			if err != nil || !filepath.IsAbs(path) {
				*warnings = append(*warnings, fmt.Sprintf("owner %s storage %s did not resolve to an absolute path", ownerName, entry.Name))
				continue
			}
			for _, root := range expandDeclaredRoot(filepath.Clean(path), entry.Kind) {
				rule := PathRule{Root: root, Source: ownerName + " storage/" + entry.Name, Owner: ownerName, priority: priorityOwnerEntry}
				switch {
				case entry.AgentRemoval != "":
					// The owner's explicit answer outranks every inference below.
					rule.Action, rule.Reason = entry.AgentRemoval, "agent_removal "+entry.AgentRemoval+" declared by "+ownerName
				case isDatabaseSidecar(rule.Root):
					// A WAL or journal regenerates only after a clean checkpoint;
					// deleting one under a live database loses committed writes.
					rule.Action, rule.Reason = "deny", "live database sidecar declared by "+ownerName
				case entry.Regenerable && !entry.Sensitive:
					rule.Action, rule.Reason = "allow", "regenerable "+string(entry.Class)+" declared by "+ownerName
				default:
					rule.Action, rule.Reason = "deny", "durable "+string(entry.Class)+" declared by "+ownerName
				}
				rules = append(rules, rule)
			}
		}
	}
	return rules
}

// expandDeclaredRoot turns a declaration whose path has a wildcard segment,
// such as one directory per project, into the locations that exist now. Path
// rules name concrete roots, so the policy runtime never matches patterns; a
// location created later is covered by the next publication. A plain path is
// returned as is, existing or not.
func expandDeclaredRoot(path, kind string) []string {
	if !strings.ContainsAny(path, "*?[") {
		return []string{path}
	}
	matches, err := filepath.Glob(path)
	if err != nil {
		return nil
	}
	roots := make([]string, 0, len(matches))
	for _, match := range matches {
		info, statErr := os.Lstat(match)
		if statErr != nil || info.Mode()&os.ModeSymlink != 0 || info.IsDir() != (kind == "dir") {
			continue
		}
		roots = append(roots, filepath.Clean(match))
	}
	sort.Strings(roots)
	return roots
}

// isDatabaseSidecar reports SQLite's write-ahead log, shared-memory index,
// and rollback journal, which owners may declare regenerable for storage
// accounting but which an agent must never delete out from under a database.
func isDatabaseSidecar(path string) bool {
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if strings.HasSuffix(strings.ToLower(filepath.Base(path)), suffix) {
			return true
		}
	}
	return false
}

func containsPlatform(platforms []corestorage.Platform, want corestorage.Platform) bool {
	for _, platform := range platforms {
		if corestorage.NormalizePlatform(string(platform)) == want {
			return true
		}
	}
	return false
}

// mergeRules keeps one rule per root: the most authoritative declaration, and
// between equals the stricter disposition.
func mergeRules(rules []PathRule) []PathRule {
	byRoot := map[string]PathRule{}
	for _, rule := range rules {
		key := rule.Root
		if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
			key = strings.ToLower(key)
		}
		current, seen := byRoot[key]
		if !seen || rule.priority > current.priority || (rule.priority == current.priority && actionRank(rule.Action) > actionRank(current.Action)) {
			byRoot[key] = rule
		}
	}
	merged := make([]PathRule, 0, len(byRoot))
	for _, rule := range byRoot {
		merged = append(merged, rule)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].Root < merged[j].Root })
	return merged
}

func actionRank(action string) int {
	switch action {
	case "deny":
		return 3
	case "ask":
		return 2
	case "allow":
		return 1
	}
	return 0
}

// BuildSnapshot renders the provider snapshot. It is scoped to filesystem
// removal so these declarations never vouch for any other kind of action.
func BuildSnapshot(now time.Time, rules []PathRule, warnings []string) ([]byte, error) {
	if now.IsZero() {
		return nil, errors.New("snapshot time is required")
	}
	now = now.UTC()
	message := fmt.Sprintf("%d declared locations", len(rules))
	if len(warnings) > 0 {
		message += fmt.Sprintf("; %d declarations left to the floor", len(warnings))
	}
	return json.MarshalIndent(map[string]any{
		"provider_id": ProviderID, "version": "agent-policy/v1",
		"capabilities": []map[string]any{{
			"id": "filesystem-dispositions", "ideal_posture": "agents delete only where the repository contract declares deletion safe",
			"declared_maturity": "enforcing", "supports_analysis": true, "supports_enforcement": true,
		}},
		"scope":          map[string]any{"risks": []string{"filesystem_removal"}},
		"health":         map[string]any{"state": "healthy", "checked_at": now, "expires_at": now.Add(snapshotLifetime), "message": message},
		"readiness":      map[string]any{"state": "ready", "rollback_plan": "withdraw the storage-manager snapshot; the built-in floor then decides alone"},
		"evidence_state": "clean", "captured_at": now, "expires_at": now.Add(snapshotLifetime),
		"path_rules": rules,
		"provenance": map[string]string{"provider": ProviderID, "declarations": "repo-contract storage.roots and runtime_home; owner storage entries"},
	}, "", "  ")
}

// RunnerSink publishes through vrooli-policy-runner, the only writer of the
// agent-policy snapshot bundle.
type RunnerSink struct {
	Binary string
}

func (s RunnerSink) PublishProviderSnapshot(ctx context.Context, data []byte) error {
	file, err := os.CreateTemp("", "storage-manager-agent-policy-*.json")
	if err != nil {
		return fmt.Errorf("stage snapshot: %w", err)
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("stage snapshot: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("stage snapshot: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, s.Binary, "publish", "--snapshot-file", file.Name()).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s publish: %w: %s", s.Binary, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// FindRunner locates vrooli-policy-runner on PATH, then in the runtime home's
// bin directory where setup installs it.
func FindRunner(runtimeHome string) (string, error) {
	if path, err := exec.LookPath(runnerName); err == nil {
		return path, nil
	}
	name := runnerName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	candidate := filepath.Join(runtimeHome, "bin", name)
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		return candidate, nil
	}
	return "", fmt.Errorf("%s is not on PATH or in %s", runnerName, filepath.Dir(candidate))
}

// PublishOnce resolves the declarations and publishes them. It returns the
// number of rules and the declarations that were left to the floor.
func PublishOnce(ctx context.Context, repoRoot string, now time.Time) (int, []string, error) {
	inputs, err := corestorage.HostGovernedRootInputs(repoRoot)
	if err != nil {
		return 0, nil, err
	}
	binary, err := FindRunner(inputs.RuntimeHome)
	if err != nil {
		return 0, nil, err
	}
	rules, warnings, err := Declarations(repoRoot)
	if err != nil {
		return 0, nil, err
	}
	data, err := BuildSnapshot(now, rules, warnings)
	if err != nil {
		return 0, nil, err
	}
	return len(rules), warnings, RunnerSink{Binary: binary}.PublishProviderSnapshot(ctx, data)
}

// Start publishes now and every PublishInterval until ctx ends. A failed pass
// is logged; the previous snapshot stays until it expires, and then the floor
// decides alone, which is stricter, never looser.
func Start(ctx context.Context, logger *log.Logger, repoRoot string) {
	go func() {
		publish := func() {
			count, warnings, err := PublishOnce(ctx, repoRoot, time.Now())
			if err != nil {
				logger.Printf("agent-policy path rules not published: %v", err)
				return
			}
			logger.Printf("agent-policy path rules published: %d declared locations", count)
			for _, warning := range warnings {
				logger.Printf("agent-policy path rules: %s", warning)
			}
		}
		publish()
		ticker := time.NewTicker(PublishInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				publish()
			}
		}
	}()
}
