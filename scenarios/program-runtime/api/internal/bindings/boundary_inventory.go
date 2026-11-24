package bindings

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
)

// BoundarySourceDigests fingerprints the exact descriptor and manifest inputs
// used by the admitted registry, keyed by repository-relative path.
func (r *Registry) BoundarySourceDigests(repoRoot string) (map[string]string, error) {
	if r == nil || r.dynamic == nil {
		return nil, nil
	}
	d := r.dynamic
	d.refresh()
	d.mu.Lock()
	defer d.mu.Unlock()
	current := d.current
	initial, err := d.source.Snapshot()
	if err != nil {
		return nil, err
	}
	if current == nil || d.generation != initial.Generation {
		return nil, fmt.Errorf("Program Runtime registry changed during source digest")
	}
	digests, err := boundarySourceDigests(d.source.WatchedPaths(), repoRoot)
	if err != nil {
		return nil, err
	}
	final, err := d.source.Snapshot()
	if err != nil {
		return nil, err
	}
	if final.Generation != initial.Generation {
		return nil, fmt.Errorf("Program Runtime inputs changed during source digest")
	}
	return digests, nil
}

func boundarySourceDigests(inputs []string, repoRoot string) (map[string]string, error) {
	out := make(map[string]string, len(inputs))
	for _, path := range inputs {
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			rel, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				return nil, relErr
			}
			out[filepath.ToSlash(rel)] = "missing"
			continue
		}
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(data)
		out[filepath.ToSlash(rel)] = "sha256:" + hex.EncodeToString(digest[:])
	}
	return out, nil
}

// BoundaryExportSnapshot returns rows and source digests from one registry
// generation. It rejects a watched-input change observed during projection.
func (r *Registry) BoundaryExportSnapshot(repoRoot string) ([]Boundary, map[string]string, []BoundaryTarget, string, error) {
	if r == nil {
		return nil, nil, nil, "", fmt.Errorf("Program Runtime registry unavailable")
	}
	if r.dynamic == nil {
		return boundaryInventory(r), nil, nil, "", nil
	}
	d := r.dynamic
	d.refresh()
	d.mu.Lock()
	defer d.mu.Unlock()
	current := d.current
	initial, err := d.source.Snapshot()
	if err != nil {
		return nil, nil, nil, "", err
	}
	if current == nil || d.generation != initial.Generation {
		return nil, nil, nil, "", fmt.Errorf("Program Runtime registry changed during export")
	}
	rows := boundaryInventory(current)
	digests, err := boundarySourceDigests(d.source.WatchedPaths(), repoRoot)
	if err != nil {
		return nil, nil, nil, "", err
	}
	final, err := d.source.Snapshot()
	if err != nil {
		return nil, nil, nil, "", err
	}
	if final.Generation != initial.Generation {
		return nil, nil, nil, "", fmt.Errorf("Program Runtime inputs changed during export")
	}
	confirmed, err := boundarySourceDigests(d.source.WatchedPaths(), repoRoot)
	if err != nil {
		return nil, nil, nil, "", err
	}
	if !reflect.DeepEqual(digests, confirmed) {
		return nil, nil, nil, "", fmt.Errorf("Program Runtime source contents changed during export")
	}
	targets := append([]BoundaryTarget(nil), r.boundaryTargets...)
	return rows, digests, targets, r.boundaryContractDigest, nil
}

// Boundary is one admitted dynamic inter-scenario call in the Program Runtime
// registry. The Python kernel's private HTTP bridge is a separate transport
// hop and is intentionally not represented as the peer boundary here.
type Boundary struct {
	ID           string `json:"id"`
	Provenance   string `json:"provenance"`
	Caller       string `json:"caller"`
	Target       string `json:"target"`
	Service      string `json:"service"`
	Method       string `json:"method"`
	RequestType  string `json:"request_type"`
	ResponseType string `json:"response_type"`
	Signature    string `json:"signature"`
}

type BoundaryTarget struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	Root        string `json:"root"`
	RuntimeName string `json:"runtime_name"`
}

// BoundaryInventory projects admitted capability declarations without probing
// target reachability. It describes admitted capabilities, not observed calls.
func (r *Registry) BoundaryInventory() []Boundary {
	r = r.active()
	return boundaryInventory(r)
}

func boundaryInventory(r *Registry) []Boundary {
	if r == nil {
		return nil
	}
	out := make([]Boundary, 0, len(r.bindings))
	for _, binding := range r.bindings {
		if binding.GetScenario() == "program-runtime" || binding.GetService() == "" || binding.GetMethod() == "" {
			continue
		}
		out = append(out, Boundary{
			ID: binding.GetId(), Provenance: "program-runtime-admitted-capability", Caller: "program-runtime",
			Target: binding.GetScenario(), Service: binding.GetService(), Method: binding.GetMethod(),
			RequestType: binding.GetRequestType(), ResponseType: binding.GetResponseType(), Signature: binding.GetSignature(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
