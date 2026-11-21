package agentinstall

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
)

// HarnessCopy is one observed copy. Found copies are never treated as owned
// merely because they are on PATH.
type HarnessCopy struct {
	Path     string `json:"path"`
	Kind     string `json:"kind"` // found, managed, or broken
	Version  string `json:"version,omitempty"`
	Selected bool   `json:"selected"`
}

// Inventory reports all relevant copies and the read-only doctor result.
type Inventory struct {
	Doctor HarnessReport `json:"doctor"`
	Copies []HarnessCopy `json:"copies"`
}

// InventoryHarness inventories the observed PATH copy and verified managed
// prefixes without changing paths or file contents.
func InventoryHarness(ctx context.Context, root, resource string) (Inventory, error) {
	h, err := LoadHarness(root, resource)
	if err != nil {
		return Inventory{}, err
	}
	deps := defaultHarnessDeps()
	doctor := inspectHarness(ctx, h, deps)
	copies := make([]HarnessCopy, 0)
	if doctor.CommandPath != "" {
		kind := "found"
		if doctor.InstallState == "managed" {
			kind = "managed"
		}
		copies = append(copies, HarnessCopy{Path: doctor.CommandPath, Kind: kind, Version: doctor.Installed, Selected: true})
	}
	installs, err := h.managedInstalls()
	if err != nil {
		return Inventory{}, err
	}
	for _, install := range installs {
		if filepath.Clean(install.Executable) == filepath.Clean(doctor.CommandPath) {
			continue
		}
		copies = append(copies, HarnessCopy{Path: install.Executable, Kind: "managed", Version: install.Version})
	}
	sort.Slice(copies, func(i, j int) bool { return copies[i].Path < copies[j].Path })
	return Inventory{Doctor: doctor, Copies: copies}, nil
}

// CleanupPlan is an immutable preview of receipt-owned lifecycle artifacts.
// Found or otherwise unowned copies appear only in Preserved.
type CleanupPlan struct {
	Resource  string   `json:"resource"`
	ID        string   `json:"id"`
	KeepCount int      `json:"keep_count,omitempty"`
	Remove    []string `json:"remove"`
	Preserved []string `json:"preserved"`
}

// PreviewCleanup lists exactly the receipt-verified managed installs that
// UninstallHarness will remove. Discovered upstream copies are preserved.
func PreviewCleanup(root, resource string) (CleanupPlan, error) {
	h, err := LoadHarness(root, resource)
	if err != nil {
		return CleanupPlan{}, err
	}
	return previewCleanup(h)
}

func previewCleanup(h Harness) (CleanupPlan, error) {
	installs, err := h.managedInstalls()
	if err != nil {
		return CleanupPlan{}, err
	}
	plan := CleanupPlan{Resource: h.Resource, Remove: make([]string, 0, len(installs)+1), Preserved: make([]string, 0)}
	for _, install := range installs {
		plan.Remove = append(plan.Remove, install.Prefix)
	}
	link, err := h.readLink()
	if err != nil {
		return CleanupPlan{}, err
	}
	if link.Kind == linkManaged && len(installs) > 0 {
		plan.Remove = append(plan.Remove, h.linkPath())
	}
	selected, _ := h.selectedCopy(defaultHarnessDeps())
	if selected != "" && !pathIn(plan.Remove, selected) {
		plan.Preserved = append(plan.Preserved, selected)
	}
	sort.Strings(plan.Remove)
	sort.Strings(plan.Preserved)
	identity, err := cleanupPlanIdentity(plan)
	if err != nil {
		return CleanupPlan{}, err
	}
	plan.ID = identity
	return plan, nil
}

// ApplyCleanup removes only the exact plan still present on disk. It refuses a
// stale preview so the operator never approves one set and deletes another.
func ApplyCleanup(ctx context.Context, root string, plan CleanupPlan) error {
	if plan.ID == "" || plan.Resource == "" {
		return fmt.Errorf("cleanup plan identity is required")
	}
	h, err := LoadHarness(root, plan.Resource)
	if err != nil {
		return err
	}
	deps := defaultHarnessDeps()
	unlock, err := deps.platform.withDefaults().acquireLock(ctx, "resource-update-"+h.Resource)
	if err != nil {
		return err
	}
	defer unlock()
	current, err := previewCleanup(h)
	if err != nil {
		return err
	}
	if !sameCleanupPlan(current, plan) {
		return fmt.Errorf("cleanup preview is stale; create a new preview")
	}
	return uninstallHarnessLocked(ctx, h)
}

// PreviewPruneOldManaged retains the active version and at least the requested
// number of managed versions. Storage Manager uses two: current plus one rollback.
// Found copies and unmanaged files are reported as preserved, never removed.
func PreviewPruneOldManaged(root, resource string, keepCount int) (CleanupPlan, error) {
	h, err := LoadHarness(root, resource)
	if err != nil {
		return CleanupPlan{}, err
	}
	return previewPruneOldManaged(h, keepCount)
}

func previewPruneOldManaged(h Harness, keepCount int) (CleanupPlan, error) {
	if keepCount < 1 {
		return CleanupPlan{}, fmt.Errorf("cleanup must retain at least one managed version")
	}
	installs, err := h.managedInstalls()
	if err != nil {
		return CleanupPlan{}, err
	}
	link, err := h.readLink()
	if err != nil {
		return CleanupPlan{}, err
	}
	retained := map[string]bool{}
	if link.Kind == linkManaged && pathIn(installPrefixes(installs), link.Prefix) {
		retained[filepath.Clean(link.Prefix)] = true
	}
	for _, install := range installs {
		if len(retained) >= keepCount {
			break
		}
		retained[filepath.Clean(install.Prefix)] = true
	}
	plan := CleanupPlan{Resource: h.Resource, KeepCount: keepCount, Remove: make([]string, 0), Preserved: make([]string, 0, len(retained)+1)}
	for _, install := range installs {
		if retained[filepath.Clean(install.Prefix)] {
			plan.Preserved = append(plan.Preserved, install.Prefix)
		} else {
			plan.Remove = append(plan.Remove, install.Prefix)
		}
	}
	selected, _ := h.selectedCopy(defaultHarnessDeps())
	if selected != "" {
		plan.Preserved = append(plan.Preserved, selected)
	}
	sort.Strings(plan.Remove)
	sort.Strings(plan.Preserved)
	plan.ID, err = cleanupPlanIdentity(plan)
	if err != nil {
		return CleanupPlan{}, err
	}
	return plan, nil
}

func installPrefixes(installs []managedInstall) []string {
	out := make([]string, 0, len(installs))
	for _, install := range installs {
		out = append(out, install.Prefix)
	}
	return out
}

// ApplyPruneOldManaged applies only the versions named by an unchanged preview.
// The per-harness lock covers the recheck and safe root-relative removals.
func ApplyPruneOldManaged(ctx context.Context, root string, plan CleanupPlan) error {
	if plan.Resource == "" || plan.ID == "" || plan.KeepCount < 1 {
		return fmt.Errorf("valid cleanup preview is required")
	}
	h, err := LoadHarness(root, plan.Resource)
	if err != nil {
		return err
	}
	deps := defaultHarnessDeps()
	unlock, err := deps.platform.withDefaults().acquireLock(ctx, "resource-update-"+h.Resource)
	if err != nil {
		return err
	}
	defer unlock()
	current, err := previewPruneOldManaged(h, plan.KeepCount)
	if err != nil {
		return err
	}
	if !sameCleanupPlan(current, plan) {
		return fmt.Errorf("cleanup preview is stale; create a new preview")
	}
	installs, err := h.managedInstalls()
	if err != nil {
		return err
	}
	remove := make([]managedInstall, 0, len(plan.Remove))
	for _, install := range installs {
		if pathIn(plan.Remove, install.Prefix) {
			remove = append(remove, install)
		}
	}
	link, err := h.readLink()
	if err != nil {
		return err
	}
	return removeManagedInstalls(ctx, h, remove, link, false)
}

// The digest is useful as a compact approval token, but caller supplied fields
// must still equal the authoritative canonical preview. This also rejects
// legacy or hand-built plans whose ID happens to be copied from a preview.
func sameCleanupPlan(current, supplied CleanupPlan) bool {
	if current.ID != supplied.ID || current.Resource != supplied.Resource || current.KeepCount != supplied.KeepCount || len(current.Remove) != len(supplied.Remove) || len(current.Preserved) != len(supplied.Preserved) {
		return false
	}
	for i := range current.Remove {
		if current.Remove[i] != supplied.Remove[i] {
			return false
		}
	}
	for i := range current.Preserved {
		if current.Preserved[i] != supplied.Preserved[i] {
			return false
		}
	}
	return true
}

func cleanupPlanIdentity(plan CleanupPlan) (string, error) {
	identity, err := json.Marshal(struct {
		Resource          string
		KeepCount         int
		Remove, Preserved []string
	}{plan.Resource, plan.KeepCount, plan.Remove, plan.Preserved})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(identity)
	return hex.EncodeToString(digest[:]), nil
}

func pathIn(paths []string, candidate string) bool {
	for _, path := range paths {
		if filepath.Clean(path) == filepath.Clean(candidate) {
			return true
		}
	}
	return false
}
