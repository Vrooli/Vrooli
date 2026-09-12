package census

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// LoadPolicy reads the operator-editable policy. A missing policy file is
// intentional: the safe defaults walk the attribution universe (repository
// and runtime homes), measure the device by statfs, prune regenerable
// dependency trees by name, and exclude virtual filesystems.
// STORAGE_CENSUS_POLICY permits an explicitly managed host policy without
// changing the repository.
func LoadPolicy(repoRoot string) (ScanPolicy, error) {
	path := strings.TrimSpace(os.Getenv("STORAGE_CENSUS_POLICY"))
	if path == "" {
		path = filepath.Join(repoRoot, "scenarios", "storage-manager", "config", "storage-census-policy.json")
	}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return ScanPolicy{}, fmt.Errorf("read census policy %s: %w", path, err)
	}
	policy := defaultPolicy()
	if err == nil {
		// Decode into a fresh value. Unmarshalling over the defaults would
		// reuse their backing array, so an exclusion that omitted "path"
		// silently inherited a default path.
		var file struct {
			FloorBytes int64              `json:"floor_bytes"`
			Roots      []PolicyRoot       `json:"roots"`
			Exclusions *[]PolicyExclusion `json:"exclusions"`
		}
		if decodeErr := json.Unmarshal(data, &file); decodeErr != nil {
			return ScanPolicy{}, fmt.Errorf("decode census policy %s: %w", path, decodeErr)
		}
		policy = ScanPolicy{FloorBytes: file.FloorBytes, Roots: file.Roots, Exclusions: defaultExclusions()}
		if file.Exclusions != nil {
			policy.Exclusions = *file.Exclusions
		}
	}
	if policy.FloorBytes <= 0 {
		policy.FloorBytes = defaultCensusFloorBytes
	}
	if err := validatePolicy(policy); err != nil {
		return ScanPolicy{}, err
	}
	return policy, nil
}

func validatePolicy(policy ScanPolicy) error {
	for _, exclusion := range policy.Exclusions {
		hasPath := strings.TrimSpace(exclusion.Path) != ""
		hasName := strings.TrimSpace(exclusion.Name) != ""
		if hasPath == hasName {
			return fmt.Errorf("census policy exclusion %q/%q must set exactly one of path or name", exclusion.Path, exclusion.Name)
		}
		if hasName && strings.ContainsAny(exclusion.Name, `/\`) {
			return fmt.Errorf("census policy name exclusion %q must be a directory base name", exclusion.Name)
		}
		if strings.TrimSpace(exclusion.Reason) == "" {
			return fmt.Errorf("census policy exclusion %q%s has no reason", exclusion.Path, exclusion.Name)
		}
	}
	return nil
}

// defaultRoots is the attribution universe: every location an owner
// declaration or class root can resolve to. The census walks these trees
// file-by-file; the rest of the device is measured by statfs and reported as
// one unscanned remainder at the device root. Walking the whole device was
// the previous default and cost 13M inodes per pass on a single-partition
// host, most of it operating-system and unrelated user data the census can
// never attribute.
func defaultRoots() []PolicyRoot {
	return []PolicyRoot{
		{Path: "$REPO_ROOT", Reason: "Repository checkout; scenario, resource, tool, and safeguard manifests live here."},
		{Path: "$RUNTIME_HOME/.vrooli", Reason: "Runtime home; state, binaries, and per-scenario data.", Optional: true},
		{Path: "$RUNTIME_HOME/.local/share/vrooli", Reason: "XDG data class root for every owner.", Optional: true},
		{Path: "$RUNTIME_HOME/.config/vrooli", Reason: "XDG config class root for every owner.", Optional: true},
		{Path: "$RUNTIME_HOME/.cache/vrooli", Reason: "XDG cache class root for every owner.", Optional: true},
		{Path: "$RUNTIME_HOME/.local/state/vrooli", Reason: "XDG state and logs class root for every owner.", Optional: true},
	}
}

// defaultExclusions prunes virtual filesystems by path and regenerable
// dependency trees by name. On the measured host node_modules alone held 4.4M
// of the device's 13.2M inodes; walking it every census attributes nothing an
// operator can act on, because it is rebuilt from a lockfile.
func defaultExclusions() []PolicyExclusion {
	return []PolicyExclusion{
		{Path: "$DEVICE_ROOT/proc", Reason: "Linux process pseudo-filesystem; it has no durable storage bytes."},
		{Path: "$DEVICE_ROOT/sys", Reason: "Kernel sysfs; it is virtual and must not enter disk accounting."},
		{Path: "$DEVICE_ROOT/dev", Reason: "Device filesystem; device nodes are not durable file storage."},
		{Path: "$DEVICE_ROOT/run", Reason: "Runtime tmpfs; it is ephemeral and may change during a census."},
		{Name: "node_modules", Reason: "Package-manager dependency tree; regenerable from the lockfile and millions of inodes per host."},
	}
}

func defaultPolicy() ScanPolicy {
	return ScanPolicy{
		FloorBytes: defaultCensusFloorBytes,
		Roots:      defaultRoots(),
		Exclusions: defaultExclusions(),
	}
}

// DefaultPolicy is the policy a host runs with when no policy file exists or
// when the file leaves roots empty.
func DefaultPolicy() ScanPolicy { return defaultPolicy() }

// resolvedPolicy is the concrete scan plan derived from an operator policy.
type resolvedPolicy struct {
	policy     ScanPolicy
	roots      []string
	exclusions []string
	// deviceRoot is the mount point that owns the display root. It is the
	// accounting root and the durable snapshot key for device-scoped scans.
	deviceRoot string
}

func resolvePolicy(displayRoot string, input ScanPolicy, deviceScoped bool, filesystem FileSystem) (resolvedPolicy, error) {
	policy := input
	if policy.FloorBytes <= 0 {
		policy.FloorBytes = defaultCensusFloorBytes
	}
	deviceRoot := displayRoot
	if deviceScoped {
		var err error
		deviceRoot, err = deviceRootForPathWith(filesystem, displayRoot)
		if err != nil {
			return resolvedPolicy{}, err
		}
	}
	resolve := func(raw string) (string, error) {
		value := strings.TrimSpace(raw)
		if value == "" {
			return "", fmt.Errorf("empty census policy path")
		}
		value = strings.ReplaceAll(value, "$DEVICE_ROOT", deviceRoot)
		value = strings.ReplaceAll(value, "$RUNTIME_HOME", runtimeHome())
		value = strings.ReplaceAll(value, "$REPO_ROOT", displayRoot)
		if !filepath.IsAbs(value) {
			value = filepath.Join(deviceRoot, value)
		}
		return filepath.Abs(filepath.Clean(value))
	}
	// An empty root list means "the attribution universe", never the whole
	// device. Operators who want a device walk declare $DEVICE_ROOT explicitly
	// and carry its cost knowingly.
	policyRoots := input.Roots
	if len(policyRoots) == 0 {
		policyRoots = defaultRoots()
		if len(input.Exclusions) == 0 {
			input.Exclusions = defaultExclusions()
		}
	}
	resolvedRoots := make([]PolicyRoot, 0, len(policyRoots))
	for _, root := range policyRoots {
		path, err := resolve(root.Path)
		if err != nil {
			return resolvedPolicy{}, fmt.Errorf("resolve census root %q: %w", root.Path, err)
		}
		resolvedRoots = append(resolvedRoots, PolicyRoot{Path: path, Reason: root.Reason, Optional: root.Optional})
	}
	sort.Slice(resolvedRoots, func(i, j int) bool { return resolvedRoots[i].Path < resolvedRoots[j].Path })
	roots := make([]string, 0, len(resolvedRoots))
	for _, root := range resolvedRoots {
		roots = append(roots, root.Path)
	}
	roots = outermostRoots(roots)
	exclusions := make([]string, 0, len(input.Exclusions))
	policy.Exclusions = append([]PolicyExclusion(nil), input.Exclusions...)
	for index, exclusion := range input.Exclusions {
		if strings.TrimSpace(exclusion.Path) == "" {
			continue
		}
		path, err := resolve(exclusion.Path)
		if err != nil {
			return resolvedPolicy{}, fmt.Errorf("resolve census exclusion %q: %w", exclusion.Path, err)
		}
		exclusions = append(exclusions, path)
		policy.Exclusions[index].Path = path
	}
	sort.Strings(exclusions)
	policy.Roots = resolvedRoots
	return resolvedPolicy{policy: policy, roots: roots, exclusions: exclusions, deviceRoot: deviceRoot}, nil
}

// outermostRoots drops any root nested inside another so a file is walked at
// most once per census. The result is sorted and duplicate-free.
func outermostRoots(roots []string) []string {
	cleaned := make([]string, 0, len(roots))
	seen := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		root = filepath.Clean(root)
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		cleaned = append(cleaned, root)
	}
	sort.Strings(cleaned)
	result := make([]string, 0, len(cleaned))
	for _, root := range cleaned {
		nested := false
		for _, outer := range result {
			if isWithin(root, outer) {
				nested = true
				break
			}
		}
		if !nested {
			result = append(result, root)
		}
	}
	return result
}

func runtimeHome() string {
	home, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(home) != "" {
		return home
	}
	if runtime.GOOS == "windows" {
		return `C:\Users\vrooli`
	}
	return string(filepath.Separator)
}

func excluded(path string, exclusions []string) bool {
	for _, exclusion := range exclusions {
		if isWithinOrEqual(path, exclusion) {
			return true
		}
	}
	return false
}

func isWithinOrEqual(path, root string) bool {
	if filepath.Clean(path) == filepath.Clean(root) {
		return true
	}
	return isWithin(path, root)
}
