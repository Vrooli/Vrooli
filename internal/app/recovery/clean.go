package recovery

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// ServingInstance is a registered instance that has not reached a terminal
// state, with the directory it runs from.
type ServingInstance struct {
	// Key is the instance key: "<scenario>" for live, "<scenario>@<variant>".
	Key        string
	WorkingDir string
}

// LiveControl is the recovery service's seam onto the running instances of a
// scenario. The production implementation reads the runtime registry and
// drives the lifecycle; tests inject a fake.
type LiveControl interface {
	// Instances lists the registered, non-terminal instances of scenario.
	Instances(scenario string) ([]ServingInstance, error)
	// Restart rebuilds and restarts one instance. The lifecycle builds the
	// replacement before it stops the running instance, so a failed build
	// leaves the running instance serving.
	Restart(instanceKey string) error
	// Stop stops one instance.
	Stop(instanceKey string) error
}

// ErrServingFromEngagement is returned when a clean would delete the tree a
// registered instance runs from.
var ErrServingFromEngagement = errors.New("recovery: an instance runs from the engagement")

// CleanRequest tears down one engagement.
type CleanRequest struct {
	Scenario string
	Slug     string
	// RestartLive moves an instance that runs from the engagement's serving
	// tree onto the working tree first: the engagement stops routing it to
	// the frozen copy, and the instance is restarted (build before stop). If
	// the restart fails, the engagement is kept and the clean fails, so the
	// instance never runs from a deleted tree. abandon uses it after restoring
	// the working tree.
	RestartLive bool
	// Force removes the engagement even when an instance runs from it: the
	// instance is restarted from the working tree, and stopped if that
	// restart fails. It is also the override when the runtime registry
	// cannot be read.
	Force bool
}

// CleanOutput reports a torn-down engagement.
type CleanOutput struct {
	Scenario      string `json:"scenario"`
	Slug          string `json:"slug"`
	EngagementDir string `json:"engagementDir"`
	// Restarted and Stopped name the instances moved off the engagement.
	Restarted []string `json:"restarted,omitempty"`
	Stopped   []string `json:"stopped,omitempty"`
}

// Clean removes an entire engagement directory (restore point, manifest,
// capture record, shared sources and serving trees). It is idempotent:
// tearing down a missing engagement is not an error.
//
// It refuses while a registered instance runs from inside the engagement
// (an engaged live instance runs from its serving tree): deleting the tree
// would break the running process, and quietly moving it to the working tree
// would serve unvalidated candidate code. RestartLive and Force move the
// instance first.
func (s Service) Clean(req CleanRequest) (CleanOutput, error) {
	if err := requireRef(req.Scenario, req.Slug); err != nil {
		return CleanOutput{}, err
	}
	dir := s.Store.EngagementDir(req.Scenario, req.Slug)
	out := CleanOutput{Scenario: req.Scenario, Slug: req.Slug, EngagementDir: dir}
	if s.Live != nil {
		if err := s.moveInstancesOffEngagement(req, dir, &out); err != nil {
			return out, err
		}
	}
	if err := s.Store.Clean(req.Scenario, req.Slug); err != nil {
		return out, err
	}
	return out, nil
}

func (s Service) moveInstancesOffEngagement(req CleanRequest, dir string, out *CleanOutput) error {
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	serving, err := s.instancesInside(req.Scenario, dir)
	if err != nil {
		if req.Force {
			return nil
		}
		return fmt.Errorf("recovery: cannot tell whether an instance of %s runs from engagement %s: %w (pass --force to clean anyway)", req.Scenario, req.Slug, err)
	}
	if len(serving) == 0 {
		return nil
	}
	if !req.RestartLive && !req.Force {
		return fmt.Errorf("%w: %s runs from %s; pass --restart-live to restart it from the working tree first, or --force", ErrServingFromEngagement,
			strings.Join(instanceKeys(serving), ", "), dir)
	}

	// End the split: the lifecycle routes live to the frozen copy only while
	// a shadow-mode manifest exists. The manifest is moved, not deleted, so a
	// failed restart can put the engagement back.
	manifest := s.Store.ManifestPath(req.Scenario, req.Slug)
	parked := manifest + ".closing"
	hadManifest := false
	if err := os.Rename(manifest, parked); err == nil {
		hadManifest = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("recovery: end engagement split: %w", err)
	}
	reinstate := func() {
		if hadManifest {
			_ = os.Rename(parked, manifest)
		}
	}

	for _, instance := range serving {
		restartErr := s.Live.Restart(instance.Key)
		if restartErr == nil {
			out.Restarted = append(out.Restarted, instance.Key)
			continue
		}
		if !req.Force {
			reinstate()
			return fmt.Errorf("recovery: restart %s from the working tree: %w; the engagement is kept and %s still runs from it", instance.Key, restartErr, instance.Key)
		}
		if stopErr := s.Live.Stop(instance.Key); stopErr != nil {
			reinstate()
			return fmt.Errorf("recovery: restart %s failed (%v) and stopping it failed: %w; the engagement is kept", instance.Key, restartErr, stopErr)
		}
		out.Stopped = append(out.Stopped, instance.Key)
	}

	// Never delete a tree that is still in use: a restart that resolved the
	// frozen copy again (a concurrent engagement write) leaves it in place.
	still, err := s.instancesInside(req.Scenario, dir)
	if err == nil && len(still) > 0 {
		reinstate()
		return fmt.Errorf("%w: %s still runs from %s after the restart", ErrServingFromEngagement, strings.Join(instanceKeys(still), ", "), dir)
	}
	return nil
}

func (s Service) instancesInside(scenario, dir string) ([]ServingInstance, error) {
	instances, err := s.Live.Instances(scenario)
	if err != nil {
		return nil, err
	}
	var inside []ServingInstance
	for _, instance := range instances {
		if pathWithin(dir, instance.WorkingDir) {
			inside = append(inside, instance)
		}
	}
	return inside, nil
}

func instanceKeys(instances []ServingInstance) []string {
	keys := make([]string, 0, len(instances))
	for _, instance := range instances {
		keys = append(keys, instance.Key)
	}
	return keys
}

func pathWithin(root, target string) bool {
	if strings.TrimSpace(target) == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(target))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// InstanceKey renders the instance key the lifecycle accepts for a
// registered instance.
func InstanceKey(scenario, variant string) string {
	return scenarioruntime.InstanceKey{Scenario: scenario, Variant: variant}.Normalize().Slug()
}
