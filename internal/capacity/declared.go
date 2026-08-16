package capacity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// declaredResourceManifest is intentionally smaller than the full resource
// manifest. Reconcile needs only to identify a GPU declaration and its optional
// claim; the resource manifest package remains the owner of full validation.
type declaredResourceManifest struct {
	GPU      json.RawMessage    `json:"gpu"`
	Capacity *ResourceClaimSpec `json:"capacity"`
}

// DeclaredGPUWithoutClaimFindings reports GPU resources that have no active
// broker claim. This catches a declaration/ledger mismatch even when the
// resource is currently idle and therefore absent from the host GPU process
// snapshot. It is deliberately a finding, not an automatic claim: admission
// and resident adoption remain the only claim-creation paths.
func DeclaredGPUWithoutClaimFindings(root string, ledger []CapacityClaim) ([]Finding, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(filepath.Join(root, "resources"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read resource declarations: %w", err)
	}
	active := make(map[string]bool)
	for _, claim := range ledger {
		if claim.OwnerKind == OwnerKindResource && IsActiveClaimStatus(claim.Status) {
			active[strings.TrimSpace(claim.OwnerID)] = true
		}
	}

	var findings []Finding
	for _, entry := range entries {
		if !entry.IsDir() || active[entry.Name()] {
			continue
		}
		path := filepath.Join(root, "resources", entry.Name(), "resource.json")
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", path, readErr)
		}
		var manifest declaredResourceManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if len(manifest.GPU) == 0 || string(manifest.GPU) == "null" {
			continue
		}
		gpuIndex := 0
		if manifest.Capacity != nil && manifest.Capacity.GPUIndex != nil {
			gpuIndex = *manifest.Capacity.GPUIndex
		}
		findings = append(findings, Finding{
			Class:        FindingDeclaredUnclaimed,
			OwnerKind:    OwnerKindResource,
			OwnerID:      entry.Name(),
			ResourceKind: ResourceKindVRAM,
			GPUIndex:     &gpuIndex,
			Severity:     "warn",
			Message:      fmt.Sprintf("resource %q declares GPU usage but holds no active capacity claim", entry.Name()),
		})
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].OwnerID < findings[j].OwnerID })
	return findings, nil
}
