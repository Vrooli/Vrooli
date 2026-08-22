package system

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/reporoot"
)

// CLIPlacementReporter reads accelerator placement from the control plane,
// which owns accelerator truth. autoheal never probes a device itself.
//
// It queries one resource at a time with --no-fast, because the fleet sweep is
// deliberately fast: reading placement means reading the host once per
// resource, which is right for a question about one resource and wasteful for
// twenty-nine. Resources is therefore the small set that declares an
// accelerator, not the whole fleet.
type CLIPlacementReporter struct {
	Executor  checks.CommandExecutor
	Resources []string
}

// resourceStatus is the slice of `vrooli resource status <name> --no-fast
// --json` this reporter needs.
type resourceStatus struct {
	Resource struct {
		Running      bool   `json:"running"`
		DeclaredMode string `json:"declared_mode"`
		ObservedMode string `json:"observed_mode"`
		ModeDrift    bool   `json:"mode_drift"`
	} `json:"resource"`
	ModeDrift bool `json:"mode_drift"`
}

// DriftedResources lists every accelerator-declaring resource serving below the
// backend it declared.
func (r CLIPlacementReporter) DriftedResources(ctx context.Context) ([]DriftedResource, error) {
	executor := r.Executor
	if executor == nil {
		executor = checks.DefaultExecutor
	}
	names := r.Resources
	if len(names) == 0 {
		names = acceleratorDeclaringResources(reporoot.ResolveFromOS())
	}
	var drifted []DriftedResource
	var failures []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		output, err := executor.CombinedOutput(ctx, "vrooli", "resource", "status", name, "--no-fast", "--json")
		if err != nil {
			failures = append(failures, name)
			continue
		}
		var status resourceStatus
		if err := json.Unmarshal(output, &status); err != nil {
			failures = append(failures, name)
			continue
		}
		if !status.Resource.Running || !(status.ModeDrift || status.Resource.ModeDrift) {
			continue
		}
		drifted = append(drifted, DriftedResource{
			Name:     name,
			Declared: status.Resource.DeclaredMode,
			Observed: status.Resource.ObservedMode,
		})
	}
	if len(failures) > 0 && len(drifted) == 0 {
		// Every read failed and none succeeded: reporting "nothing drifted"
		// would be a claim this reporter cannot support.
		return nil, fmt.Errorf("could not read placement for %s", strings.Join(failures, ", "))
	}
	return drifted, nil
}

// acceleratorDeclaringResources reads the repository's manifests and returns
// the resources that declare a non-CPU backend.
//
// The reporter resolves this itself rather than being handed a list, so the
// check is self-contained and does not depend on the order its factory builds
// things in.
func acceleratorDeclaringResources(root string) []string {
	if strings.TrimSpace(root) == "" {
		return nil
	}
	entries, err := os.ReadDir(filepath.Join(root, "resources"))
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, "resources", entry.Name(), "resource.json"))
		if err != nil {
			continue
		}
		var manifest struct {
			Acceleration *struct {
				Backends []string `json:"backends"`
			} `json:"acceleration"`
		}
		if json.Unmarshal(data, &manifest) != nil || manifest.Acceleration == nil {
			continue
		}
		for _, backend := range manifest.Acceleration.Backends {
			if strings.TrimSpace(strings.ToLower(backend)) != "cpu" {
				out = append(out, entry.Name())
				break
			}
		}
	}
	sort.Strings(out)
	return out
}
