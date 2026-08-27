// Package daemonreload centralizes Vrooli's systemd daemon-reload operation.
// A reload can revoke NVIDIA device access from live containers on affected
// hosts, so the operation owns the verify-and-restart repair transaction.
package daemonreload

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/gpuaccess"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/shell"
)

// gpuManifest is the smallest shape that answers "does this resource run a
// container on an accelerator device". It reads the acceleration block, which
// is the fleet's only accelerator declaration.
type gpuManifest struct {
	Name         string `json:"name"`
	Acceleration *struct {
		Backends []string `json:"backends"`
	} `json:"acceleration"`
	Runtime struct {
		ContainerName string `json:"container_name"`
	} `json:"runtime"`
}

// containerDeviceProbe returns the device probe name for the resource's
// highest-preference non-CPU backend, or "" when it declares none.
//
// The probe vocabulary belongs to internal/gpuaccess, which opens the device
// node inside the container; this maps a declared backend onto it.
func (m gpuManifest) containerDeviceProbe() string {
	if m.Acceleration == nil {
		return ""
	}
	for _, backend := range m.Acceleration.Backends {
		switch strings.TrimSpace(strings.ToLower(backend)) {
		case "cuda":
			return "nvidia"
		case "cpu":
			// The CPU floor is reached before any device backend, so this
			// resource declares no container device to repair.
			return ""
		}
	}
	return ""
}

type Repair struct {
	Container string `json:"container"`
	Resource  string `json:"resource"`
	State     string `json:"state"`
	Reason    string `json:"reason,omitempty"`
}

type Result struct {
	Reloaded bool     `json:"reloaded"`
	Repairs  []Repair `json:"repairs,omitempty"`
	Notes    []string `json:"notes,omitempty"`
}

var (
	listGPUContainersFn = gpuContainers
	verifyGPUFn         = gpuaccess.Verify
	reloadFn            = hostreqkit.RunPrivilegedCommand
	restartGPUFn        = docker
)

func CurrentRoot() string {
	root, _ := buildinfo.ResolveSourceRoot()
	return root
}

// Reload performs daemon-reload and repairs GPU containers whose verified
// access changed from ok to revoked. Unknown results are reported, never
// treated as a repairable success.
func Reload(ctx context.Context, root string, opts hostreqkit.EnsureOptions) (Result, error) {
	result := Result{}
	targets, err := listGPUContainersFn(ctx, root)
	if err != nil {
		// Docker may be the service being started or repaired by this reload.
		// Preserve the daemon-reload operation and report that the GPU inventory
		// could not be verified instead of blocking Docker recovery.
		result.Notes = append(result.Notes, "GPU container inventory unavailable before daemon-reload: "+err.Error())
		targets = nil
	}
	before := make(map[string]Repair, len(targets))
	for _, target := range targets {
		state, reason := verifyGPUFn(ctx, target.container, target.probe)
		before[target.container] = Repair{Container: target.container, Resource: target.resource, State: string(state), Reason: reason}
	}
	if err := reloadFn(opts.SudoMode, "systemctl", []string{"daemon-reload"}, opts); err != nil {
		return result, fmt.Errorf("systemctl daemon-reload: %w", err)
	}
	result.Reloaded = true
	for _, target := range targets {
		prior := before[target.container]
		after, reason := verifyGPUFn(ctx, target.container, target.probe)
		if prior.State == string(gpuaccess.OK) && after == gpuaccess.Revoked {
			if opts.DryRun {
				result.Notes = append(result.Notes, "dry-run: would restart "+target.container)
				continue
			}
			if err := restartGPUFn(ctx, root, "restart", target.container); err != nil {
				return result, fmt.Errorf("repair GPU container %s: %w", target.container, err)
			}
			repaired, repairReason := verifyGPUFn(ctx, target.container, target.probe)
			result.Repairs = append(result.Repairs, Repair{Container: target.container, Resource: target.resource, State: string(repaired), Reason: repairReason})
			if repaired != gpuaccess.OK {
				return result, fmt.Errorf("GPU container %s remained %s after restart: %s", target.container, repaired, repairReason)
			}
		} else if after == gpuaccess.Unknown {
			result.Notes = append(result.Notes, target.container+" GPU access is unknown after daemon-reload: "+reason)
		}
	}
	return result, nil
}

type target struct{ resource, container, probe string }

func gpuContainers(ctx context.Context, root string) ([]target, error) {
	entries, err := filepath.Glob(filepath.Join(root, "resources", "*", "resource.json"))
	if err != nil {
		return nil, err
	}
	manifests := make([]gpuManifest, 0)
	for _, path := range entries {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var manifest gpuManifest
		if json.Unmarshal(data, &manifest) != nil || manifest.containerDeviceProbe() == "" {
			continue
		}
		manifests = append(manifests, manifest)
	}
	if len(manifests) == 0 {
		return nil, nil
	}
	running, err := dockerOutput(ctx, root, "ps", "--format", "{{.Names}}")
	if err != nil {
		return nil, err
	}
	names := strings.Fields(string(running))
	var out []target
	for _, manifest := range manifests {
		preferred := strings.TrimSpace(manifest.Runtime.ContainerName)
		for _, name := range names {
			if (preferred != "" && name == preferred) || (preferred == "" && strings.Contains(name, manifest.Name)) || (preferred == "" && name == "vrooli-"+manifest.Name) {
				out = append(out, target{resource: manifest.Name, container: name, probe: manifest.containerDeviceProbe()})
			}
		}
	}
	return out, nil
}

func docker(ctx context.Context, root string, args ...string) error {
	cmd := shell.NewCommandContext(ctx, "docker", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func dockerOutput(ctx context.Context, root string, args ...string) ([]byte, error) {
	cmd := shell.NewCommandContext(ctx, "docker", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}
