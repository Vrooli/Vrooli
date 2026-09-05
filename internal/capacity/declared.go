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
// manifest. Reconcile needs only to identify an accelerator declaration and its
// optional claim; the resource manifest package remains the owner of full
// validation.
//
// This shape used to test for a `gpu` block, which fleet_contract.go rejected on
// every managed-service resource. The two disagreed, so ollama, whisper and
// reranker were invisible to reconciliation while kokoro, kyutai-stt and
// speaker-verification raised findings permanently, including while stopped.
// One declaration ends that.
type declaredResourceManifest struct {
	Acceleration *struct {
		Backends []string           `json:"backends"`
		Claim    *ResourceClaimSpec `json:"claim"`
	} `json:"acceleration"`
}

// declaresAccelerator reports whether the manifest asks for any backend other
// than the CPU. It is the single answer to the question two files used to
// answer differently.
func (m declaredResourceManifest) declaresAccelerator() bool {
	if m.Acceleration == nil {
		return false
	}
	for _, backend := range m.Acceleration.Backends {
		if strings.TrimSpace(strings.ToLower(backend)) != ResourceKindCPU {
			return true
		}
	}
	return false
}

// claim returns the declared reservation, if any.
func (m declaredResourceManifest) claim() *ResourceClaimSpec {
	if m.Acceleration == nil {
		return nil
	}
	return m.Acceleration.Claim
}

// InstalledPredicate answers whether a resource is installed on this host. It
// is injected because only the resource layer knows, and the ledger must not
// depend on it.
type InstalledPredicate func(resource string) bool

// DeclaredGPUWithoutClaimFindings reports accelerator resources that have no
// active broker claim. This catches a declaration/ledger mismatch even when the
// resource is currently idle and therefore absent from the host GPU process
// snapshot. It is deliberately a finding, not an automatic claim: admission
// and resident adoption remain the only claim-creation paths.
//
// installed filters out resources that are not on this host at all. Without it
// every accelerator-declaring resource the operator has never installed raises
// a permanent warning, which is what made this finding class ignorable. A nil
// predicate reports every declaration, which is correct when the caller has no
// way to tell.
func DeclaredGPUWithoutClaimFindings(root string, ledger []CapacityClaim, installed InstalledPredicate) ([]Finding, error) {
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
		if !manifest.declaresAccelerator() {
			continue
		}
		if installed != nil && !installed(entry.Name()) {
			// A resource that is not on this host cannot hold a claim, so the
			// absence of one is not a mismatch.
			continue
		}
		gpuIndex := 0
		if claim := manifest.claim(); claim != nil && claim.GPUIndex != nil {
			gpuIndex = *claim.GPUIndex
		}
		findings = append(findings, Finding{
			Class:        FindingDeclaredUnclaimed,
			OwnerKind:    OwnerKindResource,
			OwnerID:      entry.Name(),
			ResourceKind: ResourceKindVRAM,
			GPUIndex:     &gpuIndex,
			Severity:     "warn",
			Message:      fmt.Sprintf("resource %q declares an accelerator backend but holds no active capacity claim", entry.Name()),
		})
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].OwnerID < findings[j].OwnerID })
	return findings, nil
}
