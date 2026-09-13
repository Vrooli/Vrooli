package protogen

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/packages/proto/genmanifest"
)

// CleanupOptions controls the deterministic protobuf-footprint reconciliation.
// Cleanup never guesses that an individual message is dead: semantic message
// candidates remain an advisory owned by proto-health.
type CleanupOptions struct {
	RepoRoot  string
	ProtoRoot string
	Scenarios []string
	DryRun    bool
}

type CleanupAction struct {
	Kind    string `json:"kind"`
	Owner   string `json:"owner,omitempty"`
	Path    string `json:"path"`
	Reason  string `json:"reason"`
	Safe    bool   `json:"safe"`
	Applied bool   `json:"applied"`
}

type CleanupPlan struct {
	DryRun    bool            `json:"dry_run"`
	Scenarios []string        `json:"scenarios,omitempty"`
	Actions   []CleanupAction `json:"actions,omitempty"`
	Notes     []string        `json:"notes,omitempty"`
}

type CleanupResult struct {
	Plan          CleanupPlan `json:"plan"`
	Applied       bool        `json:"applied"`
	GenerationRan bool        `json:"generation_ran"`
}

// PlanCleanup discovers generated protobuf footprints that no longer have a
// schema owner. It also identifies empty schema-owner directories, which are a
// common remnant after a retired owner is deleted through Git. All paths are
// derived from packages/proto and are safe to remove only as part of the full
// generation reconciliation.
func PlanCleanup(opts CleanupOptions) (CleanupPlan, error) {
	opts.ProtoRoot = filepath.Clean(strings.TrimSpace(opts.ProtoRoot))
	if opts.ProtoRoot == "." || opts.ProtoRoot == "" {
		return CleanupPlan{}, fmt.Errorf("proto root is required")
	}
	if opts.RepoRoot == "" {
		opts.RepoRoot = filepath.Clean(filepath.Join(opts.ProtoRoot, "..", ".."))
	}
	requested, err := normalizeCleanupScenarios(opts.Scenarios)
	if err != nil {
		return CleanupPlan{}, err
	}
	active, err := genmanifest.ScenarioNames(opts.ProtoRoot)
	if err != nil {
		return CleanupPlan{}, fmt.Errorf("list schema owners: %w", err)
	}
	activeSet := make(map[string]bool, len(active))
	for _, owner := range active {
		activeSet[owner] = true
	}

	plan := CleanupPlan{DryRun: opts.DryRun, Scenarios: requested}
	add := func(action CleanupAction) {
		if action.Owner != "" && !cleanupScenarioSelected(action.Owner, requested) {
			return
		}
		action.Path = cleanupRelativePath(opts.RepoRoot, action.Path)
		plan.Actions = append(plan.Actions, action)
	}

	schemaRoot := filepath.Join(opts.ProtoRoot, "schemas")
	entries, err := os.ReadDir(schemaRoot)
	if err != nil {
		return CleanupPlan{}, fmt.Errorf("read schema owners: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() || activeSet[entry.Name()] {
			continue
		}
		ownerRoot := filepath.Join(schemaRoot, entry.Name())
		hasFiles, fileErr := schemaOwnerHasFiles(ownerRoot)
		if fileErr != nil {
			return CleanupPlan{}, fmt.Errorf("read schema owner %s: %w", entry.Name(), fileErr)
		}
		if !hasFiles {
			add(CleanupAction{
				Kind: "empty_schema_owner", Owner: entry.Name(), Path: ownerRoot,
				Reason: "schema owner contains no files and is not part of the active generation set", Safe: true,
			})
		} else {
			plan.Notes = append(plan.Notes, fmt.Sprintf("schema owner %q has no .proto files; review its non-proto files before retiring it", entry.Name()))
		}
	}

	staleOwners, manifestActions := orphanManifestActions(opts.ProtoRoot, activeSet)
	for _, action := range orphanGeneratedOwnerActions(opts.ProtoRoot, activeSet, staleOwners) {
		if action.Safe {
			add(action)
		} else if action.Reason != "" {
			plan.Notes = append(plan.Notes, action.Reason)
		}
	}
	for _, action := range manifestActions {
		add(action)
	}

	if len(plan.Actions) > 0 {
		add(CleanupAction{
			Kind: "regenerate_generated_tree", Owner: "fleet",
			Path:   filepath.Join(opts.ProtoRoot, "gen"),
			Reason: "the descriptor image and remaining manifests must be rebuilt after orphan removal", Safe: true,
		})
	}
	if len(plan.Actions) > 0 {
		plan.Notes = append(plan.Notes,
			"semantic possibly-unused messages are advisory; review them with proto-health before deleting source contracts",
			"apply performs a full staged generation so the descriptor, language outputs, and manifests converge atomically",
		)
	}
	sort.Slice(plan.Actions, func(i, j int) bool {
		if plan.Actions[i].Path != plan.Actions[j].Path {
			return plan.Actions[i].Path < plan.Actions[j].Path
		}
		return plan.Actions[i].Kind < plan.Actions[j].Kind
	})
	return plan, nil
}

func schemaOwnerHasFiles(root string) (bool, error) {
	found := false
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found, err
}

// ApplyCleanup reconciles the generated tree only after a plan has been
// produced. Generation is staged and published atomically by Generator, so a
// failed Buf/plugin run leaves the existing generated tree untouched.
func ApplyCleanup(ctx context.Context, opts CleanupOptions, plan CleanupPlan, logger io.Writer) (CleanupResult, error) {
	result := CleanupResult{Plan: plan}
	if plan.DryRun {
		return result, nil
	}
	if len(plan.Actions) == 0 {
		return result, nil
	}
	config := DefaultConfig(opts.RepoRoot)
	config.ProtoRoot = filepath.Clean(opts.ProtoRoot)
	config.Logger = logger
	config.PublishArtifacts = true
	generator, err := New(config)
	if err != nil {
		return result, err
	}
	if err := generator.Generate(ctx); err != nil {
		return result, fmt.Errorf("reconcile generated protobuf tree: %w", err)
	}
	result.GenerationRan = true
	for i := range result.Plan.Actions {
		action := &result.Plan.Actions[i]
		if action.Kind == "empty_schema_owner" {
			path := filepath.Join(opts.RepoRoot, filepath.FromSlash(action.Path))
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return result, fmt.Errorf("remove empty schema owner %s: %w", action.Path, err)
			}
		}
		action.Applied = true
	}
	result.Applied = true
	return result, nil
}

func orphanGeneratedOwnerActions(protoRoot string, active, stale map[string]bool) []CleanupAction {
	var actions []CleanupAction
	addOwner := func(kind, owner, path string) {
		if stale[owner] {
			actions = append(actions, CleanupAction{
				Kind: kind, Owner: owner, Path: path,
				Reason: "generated output has no active schema owner", Safe: true,
			})
		} else if !active[owner] {
			actions = append(actions, CleanupAction{
				Kind: "unrecognized_generated_directory", Owner: owner, Path: path,
				Reason: fmt.Sprintf("generated directory %s has no active schema owner or stale manifest; review before removing", cleanupRelativePath(protoRoot, path)), Safe: false,
			})
		}
	}
	for _, language := range []string{"go", "typescript"} {
		root := filepath.Join(protoRoot, "gen", language)
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if language == "typescript" && (entry.Name() == "buf" || entry.Name() == "google" || entry.Name() == "node_modules") {
				continue
			}
			if language == "typescript" && entry.Name() == "js" {
				jsEntries, readErr := os.ReadDir(filepath.Join(root, entry.Name()))
				if readErr != nil {
					continue
				}
				for _, jsEntry := range jsEntries {
					if jsEntry.IsDir() {
						addOwner("orphan_generated_owner", jsEntry.Name(), filepath.Join(root, entry.Name(), jsEntry.Name()))
					}
				}
				continue
			}
			addOwner("orphan_generated_owner", entry.Name(), filepath.Join(root, entry.Name()))
		}
	}
	pythonRoot := filepath.Join(protoRoot, "gen", "python")
	pythonEntries, err := os.ReadDir(pythonRoot)
	if err == nil {
		for _, entry := range pythonEntries {
			if entry.IsDir() {
				owner := entry.Name()
				if !active[owner] {
					for activeOwner := range active {
						if strings.ReplaceAll(activeOwner, "-", "_") == owner {
							owner = activeOwner
							break
						}
					}
				}
				if !active[owner] {
					for staleOwner := range stale {
						if strings.ReplaceAll(staleOwner, "-", "_") == owner {
							owner = staleOwner
							break
						}
					}
				}
				if !active[owner] {
					addOwner("orphan_generated_owner", owner, filepath.Join(pythonRoot, entry.Name()))
				}
			}
		}
	}
	return actions
}

func orphanManifestActions(protoRoot string, active map[string]bool) (map[string]bool, []CleanupAction) {
	root := filepath.Join(protoRoot, "gen", "manifests")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, nil
	}
	staleOwners := make(map[string]bool)
	var actions []CleanupAction
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".lock.json") {
			continue
		}
		owner := strings.TrimSuffix(entry.Name(), ".lock.json")
		if !active[owner] {
			staleOwners[owner] = true
			actions = append(actions, CleanupAction{
				Kind: "orphan_manifest", Owner: owner, Path: filepath.Join(root, entry.Name()),
				Reason: "generation manifest has no active schema owner", Safe: true,
			})
		}
	}
	return staleOwners, actions
}

func normalizeCleanupScenarios(values []string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("scenario cannot be empty")
		}
		if strings.ContainsAny(value, `/\\`) || value == "." || value == ".." {
			return nil, fmt.Errorf("scenario %q must be a bare owner", value)
		}
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out, nil
}

func cleanupScenarioSelected(owner string, requested []string) bool {
	if owner == "fleet" || len(requested) == 0 {
		return true
	}
	for _, value := range requested {
		if value == owner {
			return true
		}
	}
	return false
}

func cleanupRelativePath(repoRoot, path string) string {
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
