package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// OwnerKind identifies the repository artifact that owns a storage declaration.
// Keeping this vocabulary in api-core prevents each consumer from inventing a
// slightly different census or retention owner model.
type OwnerKind string

const (
	OwnerScenario  OwnerKind = "scenario"
	OwnerResource  OwnerKind = "resource"
	OwnerTool      OwnerKind = "tool"
	OwnerSafeguard OwnerKind = "safeguard"
)

// OwnerManifest is the normalized, owner-neutral view of one native manifest.
// The original manifest remains authoritative; this model exists so storage-
// manager, retention, placement, and health consumers read the same shape.
type OwnerManifest struct {
	Kind             OwnerKind                `json:"kind"`
	ID               string                   `json:"id"`
	Platforms        []Platform               `json:"platforms,omitempty"`
	ManifestPath     string                   `json:"manifest_path"`
	StorageDeclared  bool                     `json:"storage_declared"`
	StorageRationale string                   `json:"storage_rationale,omitempty"`
	StorageEntries   []StorageEntry           `json:"storage_entries,omitempty"`
	Retention        []RetentionDeclaration   `json:"retention,omitempty"`
	DurableData      []DurableDataDeclaration `json:"durable_data,omitempty"`
}

// StorageEntry is the common portable storage contract used by all owner
// kinds. Budget and durable data are retained alongside the location because
// those declarations answer different questions and must not be conflated.
type StorageEntry struct {
	Name        string                 `json:"name"`
	Platforms   []Platform             `json:"platforms,omitempty"`
	Rung        Rung                   `json:"rung,omitempty"`
	Path        PortablePath           `json:"path"`
	Kind        string                 `json:"kind"`
	Class       Class                  `json:"class,omitempty"`
	Format      string                 `json:"format,omitempty"`
	Regenerable bool                   `json:"regenerable,omitempty"`
	Sensitive   bool                   `json:"sensitive,omitempty"`
	Relocation  *RelocationDeclaration `json:"relocation,omitempty"`
	Reclaim     *ReclaimDeclaration    `json:"reclaim,omitempty"`
	Budget      *BudgetDeclaration     `json:"budget,omitempty"`
	Rationale   string                 `json:"rationale,omitempty"`
}

// EffectivePlatforms returns the platforms where an entry has a declared
// location. An entry-level declaration narrows its owner's scope; omitted
// declarations mean all supported platforms.
func EffectivePlatforms(owner OwnerManifest, entry StorageEntry) []Platform {
	if len(entry.Platforms) > 0 {
		return append([]Platform(nil), entry.Platforms...)
	}
	if len(owner.Platforms) > 0 {
		return append([]Platform(nil), owner.Platforms...)
	}
	return []Platform{PlatformLinux, PlatformMacOS, PlatformWindows}
}

type RelocationDeclaration struct {
	Key    string `json:"key,omitempty"`
	Scope  string `json:"scope,omitempty"`
	Config string `json:"config,omitempty"`
}

type ReclaimDeclaration struct {
	Command string `json:"command,omitempty"`
	Pruner  string `json:"pruner,omitempty"`
}

// BudgetDeclaration is the legacy/storage-surface budget shape. Retention's
// enforcement model is deliberately separate, but inventory must retain both
// forms so migration and parity reports can explain what was found.
type BudgetDeclaration struct {
	MaxAge    string `json:"max_age,omitempty"`
	MaxBytes  string `json:"max_bytes,omitempty"`
	Rationale string `json:"rationale,omitempty"`
}

type RetentionDeclaration struct {
	Name      string          `json:"name"`
	Target    RetentionTarget `json:"target"`
	MaxAge    string          `json:"max_age,omitempty"`
	MaxBytes  string          `json:"max_bytes,omitempty"`
	Pruner    string          `json:"pruner,omitempty"`
	Rationale string          `json:"rationale,omitempty"`
}

type RetentionTarget struct {
	Kind       string `json:"kind"`
	Class      Class  `json:"class,omitempty"`
	Database   string `json:"database,omitempty"`
	Table      string `json:"table,omitempty"`
	TimeColumn string `json:"time_column,omitempty"`
	Path       string `json:"path,omitempty"`
}

type DurableDataDeclaration struct {
	Name        string `json:"name"`
	Base        string `json:"base,omitempty"`
	Path        string `json:"path,omitempty"`
	Regenerable bool   `json:"regenerable"`
	HostOnly    bool   `json:"host_only"`
	Rationale   string `json:"rationale,omitempty"`
}

// InventoryFinding is a typed declaration gap. Findings are data, not errors:
// a fleet inventory should remain useful when one owner is malformed or lacks
// a manifest, while callers can still fail closed for strict validation.
type InventoryFinding struct {
	Code         string    `json:"code"`
	Severity     string    `json:"severity"`
	OwnerKind    OwnerKind `json:"owner_kind"`
	OwnerID      string    `json:"owner_id"`
	ManifestPath string    `json:"manifest_path"`
	Message      string    `json:"message"`
}

// OwnerInventory is deterministic: owners and findings are sorted by kind,
// id, and path. Counts therefore make stable baseline and drift comparisons.
type OwnerInventory struct {
	RepoRoot string             `json:"repo_root"`
	Owners   []OwnerManifest    `json:"owners"`
	Findings []InventoryFinding `json:"findings,omitempty"`
}

// InventoryOptions controls repository discovery and platform resolution.
type InventoryOptions struct {
	RepoRoot      string
	Platform      Platform
	PlatformSeams PlatformSeams
}

// LoadOwnerInventory discovers every native storage owner manifest. It does
// not require an owner to have storage entries; an empty declaration is an
// explicit adoption gap rather than an invisible owner.
func LoadOwnerInventory(opts InventoryOptions) (OwnerInventory, error) {
	root := strings.TrimSpace(opts.RepoRoot)
	if root == "" {
		return OwnerInventory{}, fmt.Errorf("owner inventory: repository root is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return OwnerInventory{}, fmt.Errorf("owner inventory: resolve repository root: %w", err)
	}
	platform := opts.Platform
	if platform == "" {
		platform = Platform(runtimePlatform())
	}

	type candidate struct {
		kind OwnerKind
		path string
		id   string
	}
	candidates := make([]candidate, 0)
	addTree := func(kind OwnerKind, relativeRoot, manifestName string) error {
		base := filepath.Join(root, relativeRoot)
		if _, statErr := os.Stat(base); os.IsNotExist(statErr) {
			return nil
		} else if statErr != nil {
			return fmt.Errorf("owner inventory: inspect %s root: %w", kind, statErr)
		}
		walkErr := filepath.WalkDir(base, func(path string, entry os.DirEntry, entryErr error) error {
			if entryErr != nil {
				return fmt.Errorf("read %s: %w", path, entryErr)
			}
			if entry.IsDir() {
				return nil
			}
			if entry.Name() != manifestName {
				return nil
			}
			if kind == OwnerScenario && filepath.Base(filepath.Dir(path)) != ".vrooli" {
				return nil
			}
			if kind == OwnerScenario {
				relative, relErr := filepath.Rel(root, path)
				parts := strings.Split(filepath.ToSlash(relative), "/")
				// Only a canonical scenario root is an owner. Bundled scenario
				// catalogs under platforms/electron and dist trees are copies,
				// not independent owners and would otherwise inflate adoption and
				// generate false duplicate-owner findings.
				if relErr != nil || len(parts) != 4 || parts[0] != "scenarios" || parts[2] != ".vrooli" {
					return nil
				}
			}
			candidates = append(candidates, candidate{kind: kind, path: path})
			return nil
		})
		if walkErr != nil {
			return fmt.Errorf("owner inventory: discover %s: %w", kind, walkErr)
		}
		return nil
	}
	if err := addTree(OwnerScenario, "scenarios", "service.json"); err != nil {
		return OwnerInventory{}, err
	}
	if err := addTree(OwnerResource, "resources", "resource.json"); err != nil {
		return OwnerInventory{}, err
	}
	if err := addTree(OwnerTool, filepath.Join("internal", "tools"), "tool.json"); err != nil {
		return OwnerInventory{}, err
	}
	if err := addTree(OwnerSafeguard, filepath.Join("internal", "safeguards"), "safeguard.json"); err != nil {
		return OwnerInventory{}, err
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].kind != candidates[j].kind {
			return candidates[i].kind < candidates[j].kind
		}
		return candidates[i].path < candidates[j].path
	})

	result := OwnerInventory{RepoRoot: root}
	seen := make(map[string]string, len(candidates))
	for _, c := range candidates {
		owner, findings := parseOwnerManifest(c.kind, c.path, platform, opts.PlatformSeams)
		if owner.ID != "" {
			key := string(owner.Kind) + "/" + owner.ID
			if prior, exists := seen[key]; exists {
				findings = append(findings, InventoryFinding{
					Code: "duplicate_owner", Severity: "error", OwnerKind: owner.Kind,
					OwnerID: owner.ID, ManifestPath: c.path,
					Message: fmt.Sprintf("owner also declared by %s", prior),
				})
			}
			seen[key] = c.path
		}
		result.Owners = append(result.Owners, owner)
		result.Findings = append(result.Findings, findings...)
	}

	// Safeguards are registered as directories, so a missing native manifest is
	// still an attributable finding instead of silently shrinking the denominator.
	if dirs, readErr := os.ReadDir(filepath.Join(root, "internal", "safeguards")); readErr == nil {
		for _, dir := range dirs {
			if !dir.IsDir() {
				continue
			}
			path := filepath.Join(root, "internal", "safeguards", dir.Name(), "safeguard.json")
			if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
				result.Findings = append(result.Findings, InventoryFinding{
					Code: "missing_manifest", Severity: "error", OwnerKind: OwnerSafeguard,
					OwnerID: dir.Name(), ManifestPath: path,
					Message: "safeguard directory has no safeguard.json manifest",
				})
			}
		}
	}
	sort.Slice(result.Owners, func(i, j int) bool {
		if result.Owners[i].Kind != result.Owners[j].Kind {
			return result.Owners[i].Kind < result.Owners[j].Kind
		}
		if result.Owners[i].ID != result.Owners[j].ID {
			return result.Owners[i].ID < result.Owners[j].ID
		}
		return result.Owners[i].ManifestPath < result.Owners[j].ManifestPath
	})
	sort.Slice(result.Findings, func(i, j int) bool {
		if result.Findings[i].OwnerKind != result.Findings[j].OwnerKind {
			return result.Findings[i].OwnerKind < result.Findings[j].OwnerKind
		}
		if result.Findings[i].OwnerID != result.Findings[j].OwnerID {
			return result.Findings[i].OwnerID < result.Findings[j].OwnerID
		}
		return result.Findings[i].Code < result.Findings[j].Code
	})
	return result, nil
}

func parseOwnerManifest(kind OwnerKind, path string, platform Platform, seams PlatformSeams) (OwnerManifest, []InventoryFinding) {
	owner := OwnerManifest{Kind: kind, ManifestPath: path}
	data, err := os.ReadFile(path)
	if err != nil {
		return owner, []InventoryFinding{{Code: "manifest_unreadable", Severity: "error", OwnerKind: kind, ManifestPath: path, Message: err.Error()}}
	}
	var raw struct {
		ID        string          `json:"id"`
		Name      string          `json:"name"`
		Platforms json.RawMessage `json:"platforms"`
		Service   struct {
			Name string `json:"name"`
		} `json:"service"`
		Storage *struct {
			Entries   map[string]json.RawMessage `json:"entries"`
			Rationale string                     `json:"rationale"`
		} `json:"storage"`
		Retention *struct {
			Budgets map[string]json.RawMessage `json:"budgets"`
		} `json:"retention"`
		DurableData *struct {
			Base     string                     `json:"base"`
			HostOnly *bool                      `json:"host_only"`
			Entries  map[string]json.RawMessage `json:"entries"`
		} `json:"durable_data"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return owner, []InventoryFinding{{Code: "malformed_manifest", Severity: "error", OwnerKind: kind, ManifestPath: path, Message: err.Error()}}
	}
	owner.ID = firstNonEmpty(raw.ID, raw.Name, raw.Service.Name)
	var findings []InventoryFinding
	owner.Platforms, findings = decodePlatforms(owner, raw.Platforms, "manifest")
	if owner.ID == "" {
		findings = append(findings, InventoryFinding{Code: "missing_owner_id", Severity: "error", OwnerKind: kind, ManifestPath: path, Message: "manifest has no id, name, or service.name"})
	}
	if raw.Storage != nil {
		owner.StorageDeclared = true
		owner.StorageRationale = strings.TrimSpace(raw.Storage.Rationale)
		names := sortedRawKeys(raw.Storage.Entries)
		for _, name := range names {
			var entry struct {
				Platforms   []string               `json:"platforms"`
				Rung        Rung                   `json:"rung"`
				Path        json.RawMessage        `json:"path"`
				Kind        string                 `json:"kind"`
				Class       Class                  `json:"class"`
				Format      string                 `json:"format"`
				Regenerable bool                   `json:"regenerable"`
				Sensitive   bool                   `json:"sensitive"`
				Relocation  *RelocationDeclaration `json:"relocation"`
				Reclaim     *ReclaimDeclaration    `json:"reclaim"`
				Budget      *BudgetDeclaration     `json:"budget"`
				Rationale   string                 `json:"rationale"`
			}
			if err := json.Unmarshal(raw.Storage.Entries[name], &entry); err != nil {
				findings = append(findings, ownerFinding(owner, "malformed_storage_entry", "storage entry "+name+": "+err.Error()))
				continue
			}
			entryPlatforms, platformFindings := normalizePlatforms(owner, entry.Platforms, "storage entry "+name)
			findings = append(findings, platformFindings...)
			if len(owner.Platforms) > 0 {
				for _, declared := range entryPlatforms {
					if !platformIncluded(owner.Platforms, declared) {
						findings = append(findings, ownerFinding(owner, "contradictory_storage_platforms", fmt.Sprintf("storage entry %s declares platform %s outside the owner's platforms %v", name, declared, owner.Platforms)))
					}
				}
			}
			var portable PortablePath
			if len(entry.Path) == 0 || string(entry.Path) == "null" {
				findings = append(findings, ownerFinding(owner, "missing_storage_path", "storage entry "+name+" has no path"))
				continue
			}
			if err := json.Unmarshal(entry.Path, &portable); err != nil {
				findings = append(findings, ownerFinding(owner, "invalid_storage_path", "storage entry "+name+": "+err.Error()))
				continue
			}
			if strings.TrimSpace(entry.Kind) == "" {
				findings = append(findings, ownerFinding(owner, "missing_storage_kind", "storage entry "+name+" has no kind"))
			}
			if entry.Rung == "" {
				findings = append(findings, ownerFinding(owner, "missing_storage_rung", "storage entry "+name+" has no rung"))
			}
			if entry.Class == "" && entry.Rung == RungOwned {
				findings = append(findings, ownerFinding(owner, "missing_storage_class", "owned storage entry "+name+" has no class"))
			}
			if entry.Rung == RungRelocatable && entry.Relocation == nil {
				findings = append(findings, ownerFinding(owner, "missing_relocation", "relocatable storage entry "+name+" has no relocation lever"))
			}
			storageEntry := StorageEntry{Name: name, Platforms: entryPlatforms, Rung: entry.Rung, Path: portable, Kind: entry.Kind, Class: entry.Class, Format: entry.Format, Regenerable: entry.Regenerable, Sensitive: entry.Sensitive, Relocation: entry.Relocation, Reclaim: entry.Reclaim, Budget: entry.Budget, Rationale: entry.Rationale}
			if platform != "" && platformIncluded(EffectivePlatforms(owner, storageEntry), platform) {
				// Native scenario manifests intentionally use paths relative to
				// the api-core storage class root. Portable host manifests use
				// absolute paths or platform tokens. Defer relative resolution to
				// the owner-aware storage resolver instead of misclassifying a
				// valid scenario declaration as an invalid host path.
				if owner.Kind == OwnerScenario && portable.ByOS == nil && !filepath.IsAbs(portable.Value) && !containsPortableToken(portable.Value) {
					owner.StorageEntries = append(owner.StorageEntries, storageEntry)
					continue
				}
				if _, resolveErr := ResolvePortablePath(name, portable, platform, seams); resolveErr != nil {
					if _, notApplicable := resolveErr.(*NotApplicable); !notApplicable {
						findings = append(findings, ownerFinding(owner, "unresolvable_storage_path", "storage entry "+name+": "+resolveErr.Error()))
					}
				}
			}
			owner.StorageEntries = append(owner.StorageEntries, storageEntry)
		}
	}
	if raw.Retention != nil {
		for _, name := range sortedRawKeys(raw.Retention.Budgets) {
			var budget RetentionDeclaration
			if err := json.Unmarshal(raw.Retention.Budgets[name], &budget); err != nil {
				findings = append(findings, ownerFinding(owner, "malformed_retention_budget", "retention budget "+name+": "+err.Error()))
				continue
			}
			budget.Name = name
			owner.Retention = append(owner.Retention, budget)
		}
	}
	if raw.DurableData != nil {
		hostOnly := true
		if raw.DurableData.HostOnly != nil {
			hostOnly = *raw.DurableData.HostOnly
		}
		for _, name := range sortedRawKeys(raw.DurableData.Entries) {
			var entry struct {
				Path        string `json:"path"`
				Regenerable bool   `json:"regenerable"`
				Rationale   string `json:"rationale"`
			}
			if err := json.Unmarshal(raw.DurableData.Entries[name], &entry); err != nil {
				findings = append(findings, ownerFinding(owner, "malformed_durable_data", "durable_data entry "+name+": "+err.Error()))
				continue
			}
			owner.DurableData = append(owner.DurableData, DurableDataDeclaration{Name: name, Base: raw.DurableData.Base, Path: entry.Path, Regenerable: entry.Regenerable, HostOnly: hostOnly, Rationale: entry.Rationale})
		}
	}
	return owner, findings
}

func decodePlatforms(owner OwnerManifest, raw json.RawMessage, scope string) ([]Platform, []InventoryFinding) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		// Resource manifests use a separate platform-status object. It is not an
		// applicability declaration and must not narrow storage entries.
		return nil, nil
	}
	return normalizePlatforms(owner, values, scope)
}

func normalizePlatforms(owner OwnerManifest, values []string, scope string) ([]Platform, []InventoryFinding) {
	platforms := make([]Platform, 0, len(values))
	var findings []InventoryFinding
	for _, value := range values {
		platform := NormalizePlatform(value)
		if platform == "" {
			findings = append(findings, ownerFinding(owner, "invalid_storage_platform", fmt.Sprintf("%s declares unsupported platform %q", scope, value)))
			continue
		}
		if !platformIncluded(platforms, platform) {
			platforms = append(platforms, platform)
		}
	}
	return platforms, findings
}

func ownerFinding(owner OwnerManifest, code, message string) InventoryFinding {
	return InventoryFinding{Code: code, Severity: "error", OwnerKind: owner.Kind, OwnerID: owner.ID, ManifestPath: owner.ManifestPath, Message: message}
}

func sortedRawKeys(values map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// runtimePlatform is kept in one small seam so inventory tests do not need to
// mutate process-wide GOOS state. The platform option should be preferred by
// callers that need deterministic cross-platform reports.
func runtimePlatform() string {
	if override := strings.TrimSpace(os.Getenv("VROOLI_STORAGE_PLATFORM")); override != "" {
		return override
	}
	return runtime.GOOS
}
