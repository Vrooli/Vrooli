package hostinventory

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type StorageCandidate struct {
	ID                   string `json:"id"`
	Kind                 string `json:"kind"`
	Location             string `json:"location"`
	StableIdentity       string `json:"stable_identity"`
	DeviceIdentity       string `json:"device_identity,omitempty"`
	Filesystem           string `json:"filesystem,omitempty"`
	Writable             bool   `json:"writable"`
	PhysicalIndependence string `json:"physical_independence"`
	Status               string `json:"status"`
	Risk                 string `json:"risk,omitempty"`
	Remediation          string `json:"remediation,omitempty"`
}

type StoragePolicy struct {
	ProtectedRoots            []string
	RepositoryRoots           []string
	RequirePhysicalSeparation bool
}

var ErrUnsafeStorageCandidate = errors.New("storage candidate does not satisfy escrow policy")

// DiscoverStorageCandidates enumerates existing mount roots through the
// platform seam and validates each candidate without selecting one. The
// caller must still require an operator choice when more than one candidate is
// ready; discovery never chooses the first writable path.
func DiscoverStorageCandidates(policy StoragePolicy) ([]StorageCandidate, error) {
	mounts, err := platformStorageMounts()
	if err != nil {
		return nil, err
	}
	candidates := make([]StorageCandidate, 0, len(mounts))
	for _, mount := range mounts {
		candidate := inspectStorageMount(mount, policy)
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Location < candidates[j].Location })
	return candidates, nil
}

type storageMount struct {
	Location   string
	Kind       string
	Filesystem string
}

func inspectStorageMount(mount storageMount, policy StoragePolicy) StorageCandidate {
	location := filepath.Clean(strings.TrimSpace(mount.Location))
	candidate := StorageCandidate{Kind: mount.Kind, Location: location, Filesystem: mount.Filesystem, Status: "degraded", PhysicalIndependence: "unknown"}
	candidate.StableIdentity = stableStorageIdentity(mount, "")
	if location == "" || location == "." {
		candidate.Status = "rejected"
		candidate.Remediation = "the storage location is not a usable mount root"
		return candidate
	}
	if containedByAny(location, policy.ProtectedRoots) {
		candidate.Status = "rejected"
		candidate.Remediation = "choose a destination outside the protected credential and state roots"
		return candidate
	}
	if containedByAny(location, policy.RepositoryRoots) {
		candidate.Status = "rejected"
		candidate.Remediation = "choose a destination outside every registered backup repository"
		return candidate
	}
	device, known := physicalDeviceIdentity(location)
	candidate.DeviceIdentity = device
	if known {
		candidate.PhysicalIndependence = "observed"
		// Repository roots are containment boundaries, not the source whose
		// physical independence this candidate must prove. A removable volume
		// may already contain a Kopia repository and still be a valid escrow
		// sink outside that repository; compare device identity with the
		// protected credential/state roots only.
		for _, root := range policy.ProtectedRoots {
			if rootDevice, rootKnown := physicalDeviceIdentity(root); rootKnown && rootDevice == device {
				candidate.PhysicalIndependence = "same-device"
				candidate.Status = "rejected"
				candidate.Risk = "same physical device as a protected source"
				candidate.Remediation = "choose a destination on a different physical device"
				return candidate
			}
		}
	} else if policy.RequirePhysicalSeparation {
		candidate.Risk = "the operating system did not expose a physical-device identity"
		candidate.Remediation = "select a candidate whose physical device can be verified, or explicitly acknowledge degraded independence"
		candidate.Status = "degraded"
		return candidate
	}
	if err := probeWritableDirectory(location); err != nil {
		candidate.Status = "rejected"
		candidate.Remediation = "choose a writable mounted destination and retry"
		return candidate
	}
	candidate.Writable = true
	candidate.Status = "ready"
	candidate.StableIdentity = stableStorageIdentity(mount, device)
	candidate.ID = candidate.StableIdentity
	return candidate
}

func ValidateStorageCandidate(candidate StorageCandidate, policy StoragePolicy) error {
	if strings.TrimSpace(candidate.Location) == "" {
		return fmt.Errorf("%w: location is required", ErrUnsafeStorageCandidate)
	}
	if containedByAny(candidate.Location, policy.ProtectedRoots) || containedByAny(candidate.Location, policy.RepositoryRoots) {
		return fmt.Errorf("%w: destination is inside a protected root", ErrUnsafeStorageCandidate)
	}
	if !candidate.Writable {
		return fmt.Errorf("%w: destination is not writable", ErrUnsafeStorageCandidate)
	}
	if policy.RequirePhysicalSeparation && candidate.PhysicalIndependence != "observed" {
		return fmt.Errorf("%w: physical independence is %s", ErrUnsafeStorageCandidate, candidate.PhysicalIndependence)
	}
	return nil
}

func containedByAny(path string, roots []string) bool {
	for _, root := range roots {
		if pathWithin(path, root) {
			return true
		}
	}
	return false
}

func pathWithin(path, root string) bool {
	path = filepath.Clean(strings.TrimSpace(path))
	root = filepath.Clean(strings.TrimSpace(root))
	if path == "" || root == "" || path == "." || root == "." {
		return false
	}
	resolvedPath, pathErr := resolveExistingOrParent(path)
	resolvedRoot, rootErr := resolveExistingOrParent(root)
	if pathErr == nil && rootErr == nil {
		path, root = resolvedPath, resolvedRoot
	}
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func resolveExistingOrParent(path string) (string, error) {
	return resolvePathForStorage(path)
}

func stableStorageIdentity(mount storageMount, device string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(mount.Kind + "|" + filepath.Clean(mount.Location) + "|" + device + "|" + mount.Filesystem))
	return hex.EncodeToString(h.Sum(nil))
}
