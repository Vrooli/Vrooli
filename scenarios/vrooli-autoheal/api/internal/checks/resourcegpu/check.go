// Package resourcegpu monitors GPU access from the resource containers that
// consume it. It is deliberately separate from system-gpu: host health and
// container device access are different facts and must not mask each other.
package resourcegpu

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	sharedhost "github.com/vrooli/vrooli/internal/hostinventory"
	sharedresources "github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/reporoot"
)

type Target struct {
	Name      string
	Container string
	Probe     string
	Running   bool
}

type (
	VerifyFunc  func(context.Context, string, string) (sharedresources.GPUAccessState, string)
	TargetsFunc func(context.Context) ([]Target, error)
	RestartFunc func(context.Context, string) error
	HostFunc    func(context.Context) (sharedhost.Snapshot, error)
)

type Check struct {
	verify  VerifyFunc
	targets TargetsFunc
	restart RestartFunc
	host    HostFunc
	clock   func() time.Time
}

type Option func(*Check)

func WithVerifier(fn VerifyFunc) Option    { return func(c *Check) { c.verify = fn } }
func WithTargets(fn TargetsFunc) Option    { return func(c *Check) { c.targets = fn } }
func WithRestarter(fn RestartFunc) Option  { return func(c *Check) { c.restart = fn } }
func WithHostCollector(fn HostFunc) Option { return func(c *Check) { c.host = fn } }
func WithClock(fn func() time.Time) Option { return func(c *Check) { c.clock = fn } }

func New(opts ...Option) *Check {
	c := &Check{verify: sharedresources.VerifyContainerGPU, host: sharedhost.Collect, clock: time.Now}
	for _, opt := range opts {
		opt(c)
	}
	if c.targets == nil {
		c.targets = defaultTargets
	}
	if c.restart == nil {
		c.restart = defaultRestart
	}
	return c
}

func (c *Check) ID() string    { return "resource-gpu-access" }
func (c *Check) Title() string { return "Resource Container GPU Access" }
func (c *Check) Description() string {
	return "Verifies that every running NVIDIA GPU resource container can open its GPU device"
}

func (c *Check) Importance() string {
	return "A green host GPU does not prove a container can use it; revoked access silently moves inference to CPU"
}
func (c *Check) Category() checks.Category  { return checks.CategoryResource }
func (c *Check) IntervalSeconds() int       { return 60 }
func (c *Check) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *Check) Run(ctx context.Context) checks.Result {
	result := checks.Result{CheckID: c.ID(), Details: map[string]interface{}{}, Timestamp: c.clock()}
	if checkOS != "linux" {
		result.Status = checks.StatusNotApplicable
		result.Message = "container GPU access check is not implemented on this platform"
		result.Details["platform"] = checkOS
		return result
	}
	snap, err := c.host(ctx)
	if err != nil {
		result.Status = checks.StatusWarning
		result.Message = "container GPU access is unknown: host inventory failed"
		result.Details["error"] = err.Error()
		return result
	}
	if !snap.HasDockerAddressableNvidiaGPU() {
		result.Status = checks.StatusNotApplicable
		result.Message = "not-applicable: no Docker-addressable NVIDIA GPU on this host"
		result.Details["has_host_gpu"] = snap.HasNvidiaGPU()
		result.Details["has_docker_nvidia_runtime"] = snap.HasDockerNvidiaRuntime()
		return result
	}
	targets, err := c.targets(ctx)
	if err != nil {
		result.Status = checks.StatusWarning
		result.Message = "container GPU access is unknown: resource discovery failed"
		result.Details["error"] = err.Error()
		return result
	}
	observed := make([]map[string]string, 0, len(targets))
	var revoked, unknown []string
	for _, target := range targets {
		if !target.Running {
			continue
		}
		state, reason := c.verify(ctx, target.Container, target.Probe)
		observed = append(observed, map[string]string{"resource": target.Name, "container": target.Container, "state": string(state), "reason": reason})
		switch state {
		case sharedresources.GPUAccessRevoked:
			revoked = append(revoked, target.Name)
		case sharedresources.GPUAccessUnknown:
			unknown = append(unknown, target.Name)
		}
	}
	result.Details["resources"] = observed
	result.Details["revoked"] = revoked
	result.Details["unknown"] = unknown
	switch {
	case len(revoked) > 0:
		result.Status = checks.StatusCritical
		result.Message = "GPU access revoked for resource container(s): " + strings.Join(revoked, ", ")
	case len(unknown) > 0:
		result.Status = checks.StatusWarning
		result.Message = "GPU access unknown for resource container(s): " + strings.Join(unknown, ", ")
	default:
		result.Status = checks.StatusOK
		result.Message = "all running GPU resource containers can open /dev/nvidiactl"
	}
	return result
}

func (c *Check) RecoveryActions(last *checks.Result) []checks.RecoveryAction {
	if last == nil || last.Status != checks.StatusCritical {
		return nil
	}
	actions := []checks.RecoveryAction{{ID: "restart-all-revoked", Name: "Restart All Revoked GPU Resources", Description: "Restart every resource container reported with revoked GPU access", Dangerous: true, Available: true}}
	if names := revokedNames(last); len(names) > 0 {
		for _, name := range names {
			actions = append(actions, checks.RecoveryAction{ID: "restart-" + name, Name: "Restart " + name, Description: "Restart the " + name + " resource to restore container GPU access", Dangerous: true, Available: true})
		}
	}
	return actions
}

func (c *Check) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	started := c.clock()
	result := checks.ActionResult{ActionID: actionID, CheckID: c.ID(), Timestamp: started}
	var names []string
	switch {
	case actionID == "restart-all-revoked":
		targets, err := c.targets(ctx)
		if err != nil {
			result.Error, result.Message = err.Error(), "could not enumerate GPU resources"
			return finish(result, started, c.clock)
		}
		for _, target := range targets {
			if target.Running {
				state, _ := c.verify(ctx, target.Container, target.Probe)
				if state != sharedresources.GPUAccessRevoked {
					continue
				}
				names = append(names, target.Name)
			}
		}
	case strings.HasPrefix(actionID, "restart-"):
		names = []string{strings.TrimPrefix(actionID, "restart-")}
	default:
		result.Error, result.Message = "unknown action: "+actionID, "Action not recognized"
		return finish(result, started, c.clock)
	}
	for _, name := range names {
		if err := c.restart(ctx, name); err != nil {
			result.Error, result.Message = err.Error(), "GPU resource restart failed for "+name
			return finish(result, started, c.clock)
		}
	}
	result.Success = true
	result.Message = "restarted GPU resource(s): " + strings.Join(names, ", ")
	return finish(result, started, c.clock)
}

func finish(result checks.ActionResult, started time.Time, now func() time.Time) checks.ActionResult {
	result.Duration = now().Sub(started)
	return result
}

func revokedNames(result *checks.Result) []string {
	values, ok := result.Details["revoked"].([]string)
	if ok {
		return append([]string(nil), values...)
	}
	return nil
}

func defaultTargets(ctx context.Context) ([]Target, error) {
	root := reporoot.ResolveFromOS()
	if root == "" {
		return nil, fmt.Errorf("Vrooli repository root is unavailable")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	controller := sharedresources.NewController(root, home)
	items, err := controller.Discover()
	if err != nil {
		return nil, err
	}
	out := make([]Target, 0)
	for _, item := range items {
		manifestPath := item.ManifestPath
		if manifestPath == "" {
			manifestPath = filepath.Join(root, "resources", item.Name, "resource.json")
		}
		manifest, err := controller.LoadManifest(manifestPath)
		if err != nil || manifest.GPU == nil {
			continue
		}
		status, err := controller.Status(item.Name, true)
		if err != nil {
			continue
		}
		container := strings.TrimSpace(manifest.Runtime.ContainerName)
		if container == "" {
			container = "vrooli-" + manifest.Name
		}
		out = append(out, Target{Name: manifest.Name, Container: container, Probe: manifest.GPU.Probe, Running: status.Running})
	}
	return out, nil
}

func defaultRestart(ctx context.Context, name string) error {
	root := reporoot.ResolveFromOS()
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return sharedresources.NewController(root, home).Run(name, []string{"restart"}, io.Discard, io.Discard)
}
