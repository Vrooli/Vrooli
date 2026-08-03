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
// intentional: the safe defaults are still device-scoped and exclude virtual
// filesystems. STORAGE_CENSUS_POLICY permits an explicitly managed host
// policy without changing the repository.
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
		if decodeErr := json.Unmarshal(data, &policy); decodeErr != nil {
			return ScanPolicy{}, fmt.Errorf("decode census policy %s: %w", path, decodeErr)
		}
	}
	if policy.FloorBytes <= 0 {
		policy.FloorBytes = defaultCensusFloorBytes
	}
	for _, exclusion := range policy.Exclusions {
		if strings.TrimSpace(exclusion.Reason) == "" {
			return ScanPolicy{}, fmt.Errorf("census policy exclusion %q has no reason", exclusion.Path)
		}
	}
	return policy, nil
}

func defaultPolicy() ScanPolicy {
	return ScanPolicy{
		FloorBytes: defaultCensusFloorBytes,
		Exclusions: []PolicyExclusion{
			{Path: "$DEVICE_ROOT/proc", Reason: "Linux process pseudo-filesystem; it has no durable storage bytes."},
			{Path: "$DEVICE_ROOT/sys", Reason: "Kernel sysfs; it is virtual and must not enter disk accounting."},
			{Path: "$DEVICE_ROOT/dev", Reason: "Device filesystem; device nodes are not durable file storage."},
			{Path: "$DEVICE_ROOT/run", Reason: "Runtime tmpfs; it is ephemeral and may change during a census."},
		},
	}
}

func resolvePolicy(displayRoot string, input ScanPolicy, deviceScoped bool) (ScanPolicy, []string, []string, error) {
	policy := input
	if policy.FloorBytes <= 0 {
		policy.FloorBytes = defaultCensusFloorBytes
	}
	deviceRoot := displayRoot
	if deviceScoped {
		var err error
		deviceRoot, err = deviceRootForPath(displayRoot)
		if err != nil {
			return ScanPolicy{}, nil, nil, err
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
	roots := make([]string, 0, len(input.Roots))
	if len(input.Roots) == 0 {
		roots = append(roots, deviceRoot)
	} else {
		for _, root := range input.Roots {
			path, err := resolve(root.Path)
			if err != nil {
				return ScanPolicy{}, nil, nil, fmt.Errorf("resolve census root %q: %w", root.Path, err)
			}
			roots = append(roots, path)
		}
	}
	exclusions := make([]string, 0, len(input.Exclusions))
	for _, exclusion := range input.Exclusions {
		path, err := resolve(exclusion.Path)
		if err != nil {
			return ScanPolicy{}, nil, nil, fmt.Errorf("resolve census exclusion %q: %w", exclusion.Path, err)
		}
		exclusions = append(exclusions, path)
	}
	sort.Strings(roots)
	sort.Strings(exclusions)
	policy.Roots = make([]PolicyRoot, len(roots))
	for index, root := range roots {
		policy.Roots[index] = PolicyRoot{Path: root}
	}
	policy.Exclusions = append([]PolicyExclusion(nil), input.Exclusions...)
	for index := range policy.Exclusions {
		policy.Exclusions[index].Path, _ = resolve(policy.Exclusions[index].Path)
	}
	return policy, roots, exclusions, nil
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
