package bundleruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vrooli/binaryfetch"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	component "github.com/vrooli/vrooli/packages/desktop-component"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/api"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/components"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/deps"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/env"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/health"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/peerlookup"
)

func (s *Supervisor) ComponentCatalog(ctx context.Context) ([]component.SignedEntry, error) {
	return s.components.Catalog(ctx)
}
func (s *Supervisor) ComponentInstallations(ctx context.Context) ([]components.Installation, error) {
	return s.components.Installed(ctx)
}
func (s *Supervisor) InstallComponent(ctx context.Context, id string) (components.Installation, error) {
	rows, err := s.components.Catalog(ctx)
	if err != nil {
		return components.Installation{}, err
	}
	for _, row := range rows {
		if row.Entry.ID == id {
			return s.components.Install(ctx, row)
		}
	}
	return components.Installation{}, fmt.Errorf("component is not available in the signed catalog")
}

// RemoveComponent stops the active generation before unpublishing its code.
// Data is a separate owner directory and is removed only by explicit choice.
func (s *Supervisor) RemoveComponent(ctx context.Context, id string, keepData bool) error {
	if !component.ValidID(id) {
		return fmt.Errorf("invalid component ID")
	}
	s.componentMu.Lock()
	defer s.componentMu.Unlock()
	if err := s.stopComponent(ctx, id); err != nil {
		return err
	}
	return s.components.Remove(ctx, id, keepData)
}

func (s *Supervisor) stopComponent(ctx context.Context, id string) error {
	services := s.effectiveServices()
	order, err := deps.TopoSort(services)
	if err != nil {
		return err
	}
	for i := len(order) - 1; i >= 0; i-- {
		svc := deps.FindService(services, order[i])
		if svc == nil || svc.ComponentID != id {
			continue
		}
		if proc := s.getProc(svc.ID); proc != nil {
			if proc.proc != nil {
				s.gracefulStop(ctx, proc)
			}
			if proc.cancel != nil {
				proc.cancel()
			}
			s.setProc(svc.ID, nil)
		}
		s.setStatus(svc.ID, ServiceStatus{Skipped: true, Message: "component removed"})
	}
	return nil
}

// ResolveFeature is call-time selection. A compatible live full app wins;
// otherwise only the explicitly requested, verified installed slice may start.
// A slice never becomes a candidate for bare scenario discovery.
func (s *Supervisor) ResolveFeature(ctx context.Context, owner, feature, requestedRange string) (api.FeatureResolution, error) {
	s.componentMu.Lock()
	defer s.componentMu.Unlock()
	rows, err := s.components.Installed(ctx)
	if err != nil {
		return api.FeatureResolution{}, err
	}
	if selected, err := s.resolveFullFeature(ctx, rows, owner, feature, requestedRange); err != nil || selected.Source != "" {
		return selected, err
	}
	peers := append(append([]manifest.Peer(nil), s.opts.Manifest.Peers...), s.opts.Manifest.ProgramBindingPeers...)
	localBindings := 0
	for _, peer := range peers {
		if _, ok := peer.Features[feature]; ok && peer.Scenario == owner {
			localBindings++
		}
	}
	for i := range rows {
		if rows[i].State == components.Pending {
			if active, ok := s.activeComponents[rows[i].Signed.Entry.ID]; ok {
				rows[i] = active
			}
		}
		if rows[i].State != components.Installed || rows[i].Signed.Entry.Owner != owner {
			continue
		}
		for _, provided := range rows[i].Signed.Entry.Provides {
			if provided.ID == feature {
				localBindings++
			}
		}
	}
	if localBindings > 1 {
		return api.FeatureResolution{}, fmt.Errorf("ambiguous feature provider")
	}
	if selected, err := s.resolveEmbeddedFeature(ctx, peers, owner, feature, requestedRange); err != nil || selected.Source != "" {
		return selected, err
	}
	return s.resolveInstalledFeature(ctx, rows, owner, feature, requestedRange)
}

func (s *Supervisor) resolveInstalledFeature(ctx context.Context, rows []components.Installation, owner, feature, requestedRange string) (api.FeatureResolution, error) {
	var candidate *components.Installation
	var provides component.Feature
	for i := range rows {
		if rows[i].Signed.Entry.Owner != owner {
			continue
		}
		for _, f := range rows[i].Signed.Entry.Provides {
			if f.ID == feature {
				if candidate != nil {
					return api.FeatureResolution{}, fmt.Errorf("ambiguous feature provider")
				}
				candidate, provides = &rows[i], f
			}
		}
	}
	if candidate == nil {
		return api.FeatureResolution{}, fmt.Errorf("feature is not installed")
	}
	if s.components.Source == nil || candidate.Signed.Verify(s.components.Source.PublicKeyPEM) != nil {
		return api.FeatureResolution{}, fmt.Errorf("feature signature invalid")
	}
	if candidate.State != components.Installed {
		return api.FeatureResolution{}, fmt.Errorf("feature is %s", candidate.State)
	}
	if !manifest.VersionSatisfiesRange(candidate.Signed.Entry.Version, requestedRange) || !manifest.VersionSatisfiesRange(candidate.Signed.Entry.Version, peerlookup.FeatureVersionRange(s.opts.Manifest, owner)) {
		return api.FeatureResolution{}, fmt.Errorf("feature owner version is outside the consumer range")
	}
	root := filepath.Join(s.components.Root, candidate.Signed.Entry.ID, "generations", candidate.Generation)
	if digest, err := binaryfetch.TreeDigest(root); err != nil || digest != candidate.Signed.Entry.TreeSHA256 {
		return api.FeatureResolution{}, fmt.Errorf("installed feature generation failed verification")
	}
	activated, err := s.activateComponent(ctx, *candidate)
	if err != nil {
		if ctx.Err() == nil {
			s.disableFailedComponent(ctx, activated, err)
		}
		return api.FeatureResolution{}, err
	}
	candidate = &activated
	if !manifest.VersionSatisfiesRange(candidate.Signed.Entry.Version, requestedRange) || !manifest.VersionSatisfiesRange(candidate.Signed.Entry.Version, peerlookup.FeatureVersionRange(s.opts.Manifest, owner)) {
		return api.FeatureResolution{}, fmt.Errorf("restored feature owner is outside the consumer range")
	}
	provides = component.Feature{}
	for _, declared := range candidate.Signed.Entry.Provides {
		if declared.ID == feature {
			provides = declared
		}
	}
	if provides.ID == "" {
		return api.FeatureResolution{}, fmt.Errorf("restored generation does not provide requested feature")
	}
	svc := deps.FindService(s.effectiveServices(), provides.Service)
	if svc == nil || svc.ComponentID != candidate.Signed.Entry.ID || svc.ComponentVersion != candidate.Signed.Entry.Version {
		return api.FeatureResolution{}, fmt.Errorf("feature does not belong to selected component")
	}
	declaredPort := false
	if svc.Ports != nil {
		for _, port := range svc.Ports.Requested {
			if port.Name == provides.Port {
				declaredPort = true
			}
		}
	}
	if !declaredPort {
		return api.FeatureResolution{}, fmt.Errorf("feature port is not declared by selected component")
	}
	port := s.portAllocator.Map()[provides.Service][provides.Port]
	if port <= 0 {
		return api.FeatureResolution{}, fmt.Errorf("feature port unavailable")
	}
	return api.FeatureResolution{URL: fmt.Sprintf("http://127.0.0.1:%d", port), Source: "component", ComponentID: candidate.Signed.Entry.ID, Owner: owner, Feature: feature, OwnerVersion: candidate.Signed.Entry.Version, RuntimeInstance: s.instanceID}, nil
}

// effectiveServices is read while componentMu is held. Rollback overlays are
// immutable service snapshots; the original host manifest is never rewritten.
func (s *Supervisor) effectiveServices() []manifest.Service {
	result := make([]manifest.Service, 0, len(s.opts.Manifest.Services))
	for _, svc := range s.opts.Manifest.Services {
		if _, replaced := s.componentOverrides[svc.ComponentID]; !replaced {
			result = append(result, svc)
		}
	}
	for _, services := range s.componentOverrides {
		result = append(result, services...)
	}
	return result
}

func (s *Supervisor) serviceHealthChecker(svc manifest.Service) HealthChecker {
	if svc.ComponentID == "" || s.opts.HealthChecker != nil {
		return s.healthChecker
	}
	snapshot := *s.opts.Manifest
	snapshot.Services = []manifest.Service{svc}
	return health.NewMonitor(health.MonitorConfig{Manifest: &snapshot, Ports: s.portAllocator, Dialer: s.dialer, CmdRunner: s.cmdRunner, FS: s.fs, Clock: s.clock, AppData: s.serviceDataRoot(svc), StatusGetter: func(id string) (health.Status, bool) { status, ok := s.ServiceStatuses()[id]; return status, ok }})
}

func (s *Supervisor) startComponentGraph(ctx context.Context, id string) error {
	services := s.effectiveServices()
	order, err := deps.TopoSort(services)
	if err != nil {
		return err
	}
	for _, name := range order {
		svc := deps.FindService(services, name)
		if svc == nil || svc.ComponentID != id {
			continue
		}
		if s.getProc(name) == nil {
			if err := s.startService(s.runtimeCtx, *svc); err != nil {
				return err
			}
		}
		if err := s.serviceHealthChecker(*svc).WaitForReadiness(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

func (s *Supervisor) activateComponent(ctx context.Context, row components.Installation) (components.Installation, error) {
	if s.activeComponents == nil {
		s.activeComponents = map[string]components.Installation{}
	}
	err := s.startComponentGraph(ctx, row.Signed.Entry.ID)
	if err == nil {
		if row.ActivationPending || row.Previous != nil {
			if err := s.components.CompleteActivation(ctx, row.Signed.Entry.ID, row.Generation); err != nil && !errors.Is(err, components.ErrActivationSuperseded) {
				return row, err
			}
			row.Previous = nil
			row.ActivationPending = false
		}
		s.activeComponents[row.Signed.Entry.ID] = row
		return row, nil
	}
	if row.Previous == nil {
		return row, err
	}
	if stopErr := s.stopComponent(ctx, row.Signed.Entry.ID); stopErr != nil {
		return row, stopErr
	}
	old, rollbackErr := s.components.Rollback(ctx, row.Signed.Entry.ID, row.Generation)
	if rollbackErr != nil {
		return row, fmt.Errorf("component activation failed: %v; rollback: %w", err, rollbackErr)
	}
	var previous manifest.Manifest
	if err := json.Unmarshal(old.Signed.Entry.Manifest, &previous); err != nil {
		return old, err
	}
	services, err := manifest.ProjectComponent(old.Signed.Entry.Owner, previous.Services)
	if err != nil {
		return old, err
	}
	for i := range services {
		services[i].BundleRoot = filepath.Join(s.components.Root, old.Signed.Entry.ID, "generations", old.Generation)
		services[i].ComponentID, services[i].ComponentVersion, services[i].ComponentSlice = old.Signed.Entry.ID, old.Signed.Entry.Version, old.Signed.Entry.Slice
	}
	if s.componentOverrides == nil {
		s.componentOverrides = map[string][]manifest.Service{}
	}
	s.componentOverrides[old.Signed.Entry.ID] = services
	allocator, ok := s.portAllocator.(interface {
		AllocateServices([]manifest.Service) error
	})
	if !ok {
		return old, fmt.Errorf("runtime port allocator cannot restore component services")
	}
	if err := allocator.AllocateServices(services); err != nil {
		return old, err
	}
	s.activeComponents[old.Signed.Entry.ID] = old
	if err := s.startComponentGraph(ctx, old.Signed.Entry.ID); err != nil {
		return old, fmt.Errorf("restored component failed readiness: %w", err)
	}
	_ = s.recordTelemetry("component_update_rolled_back", map[string]interface{}{"component_id": old.Signed.Entry.ID, "version": old.Signed.Entry.Version})
	return old, nil
}

func (s *Supervisor) disableFailedComponent(ctx context.Context, row components.Installation, cause error) {
	const reason = "The dependency could not start and no healthy compatible version is available. Update or remove it."
	_ = s.stopComponent(ctx, row.Signed.Entry.ID)
	if s.componentOverrides == nil {
		s.componentOverrides = map[string][]manifest.Service{}
	}
	s.componentOverrides[row.Signed.Entry.ID] = nil
	delete(s.activeComponents, row.Signed.Entry.ID)
	err := s.components.Disable(ctx, row.Signed.Entry.ID, row.Generation, reason)
	_ = s.recordTelemetry("component_activation_disabled", map[string]interface{}{"component_id": row.Signed.Entry.ID, "reason": reason, "activation_error": cause.Error(), "persistence_error": fmt.Sprint(err)})
}

func (s *Supervisor) activatePendingUpdates(ctx context.Context) error {
	if s.components == nil {
		return nil
	}
	rows, err := s.components.Installed(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.State != components.Installed || !row.ActivationPending || row.Signed.Entry.Slice != "" {
			continue
		}
		if active, err := s.activateComponent(ctx, row); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			s.disableFailedComponent(ctx, active, err)
		}
	}
	return nil
}

func (s *Supervisor) resolveFullFeature(ctx context.Context, rows []components.Installation, owner, feature, requestedRange string) (api.FeatureResolution, error) {
	home, err := s.resolvePeerHome()
	if err != nil {
		return api.FeatureResolution{}, err
	}
	if admitted, port, err := peerlookup.FeaturePeer(s.opts.Manifest, home, owner, feature, requestedRange); err == nil && port > 0 {
		url := fmt.Sprintf("http://127.0.0.1:%d", port)
		probe, cancel := context.WithTimeout(ctx, 2*time.Second)
		req, _ := http.NewRequestWithContext(probe, http.MethodGet, url+"/health", nil)
		resp, probeErr := http.DefaultClient.Do(req)
		cancel()
		if probeErr == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				if err := s.stopEmbeddedFeature(ctx, owner); err != nil {
					return api.FeatureResolution{}, err
				}
				for _, row := range rows {
					if row.Signed.Entry.Owner != owner {
						continue
					}
					for _, f := range row.Signed.Entry.Provides {
						if f.ID == feature {
							if err := s.stopComponent(ctx, row.Signed.Entry.ID); err != nil {
								return api.FeatureResolution{}, err
							}
						}
					}
				}
				record, currentPort, err := peerlookup.FeaturePeer(s.opts.Manifest, home, owner, feature, requestedRange)
				if err != nil || currentPort != port || record.OwnerPID != admitted.OwnerPID || record.Incarnation != admitted.Incarnation || record.Instance != admitted.Instance || !record.StartedAt.Equal(admitted.StartedAt) || record.Features[feature] != admitted.Features[feature] {
					return api.FeatureResolution{}, fmt.Errorf("full feature provider changed during selection")
				}
				advertised := record.Features[feature]
				return api.FeatureResolution{URL: url, Source: "full_app", Owner: owner, Feature: feature, OwnerVersion: advertised.OwnerVersion, RuntimeInstance: fmt.Sprintf("%s/%d/%d", record.Instance, record.OwnerPID, record.StartedAt.UnixNano())}, nil
			}
		}
	}
	return api.FeatureResolution{}, nil
}

func (s *Supervisor) resolveEmbeddedFeature(ctx context.Context, peers []manifest.Peer, owner, feature, requestedRange string) (api.FeatureResolution, error) {
	for _, peer := range peers {
		provided, ok := peer.Features[feature]
		if peer.Scenario != owner || !ok || (peer.BundlePolicy != "embed" && peer.BundlePolicy != "either") {
			continue
		}
		if !manifest.VersionSatisfiesRange(provided.OwnerVersion, requestedRange) || !manifest.VersionSatisfiesRange(provided.OwnerVersion, peerlookup.FeatureVersionRange(s.opts.Manifest, owner)) {
			return api.FeatureResolution{}, fmt.Errorf("embedded feature owner version is outside the consumer range")
		}
		order, err := deps.TopoSort(s.opts.Manifest.Services)
		if err != nil {
			return api.FeatureResolution{}, err
		}
		for _, id := range order {
			svc := deps.FindService(s.opts.Manifest.Services, id)
			if svc == nil || !strings.HasPrefix(id, owner+"--") || svc.ComponentID != "" {
				continue
			}
			if s.getProc(id) == nil {
				if err := s.startService(s.runtimeCtx, *svc); err != nil {
					return api.FeatureResolution{}, err
				}
			}
			if err := s.healthChecker.WaitForReadiness(ctx, id); err != nil {
				return api.FeatureResolution{}, err
			}
		}
		port := s.portAllocator.Map()[provided.Service][provided.Port]
		if port <= 0 {
			return api.FeatureResolution{}, fmt.Errorf("embedded feature port unavailable")
		}
		return api.FeatureResolution{URL: fmt.Sprintf("http://127.0.0.1:%d", port), Source: "embedded", Owner: owner, Feature: feature, OwnerVersion: provided.OwnerVersion, RuntimeInstance: s.instanceID}, nil
	}
	return api.FeatureResolution{}, nil
}

func (s *Supervisor) stopEmbeddedFeature(ctx context.Context, owner string) error {
	order, err := deps.TopoSort(s.opts.Manifest.Services)
	if err != nil {
		return err
	}
	for i := len(order) - 1; i >= 0; i-- {
		svc := deps.FindService(s.opts.Manifest.Services, order[i])
		if svc == nil || !strings.HasPrefix(svc.ID, owner+"--") || svc.ComponentID != "" {
			continue
		}
		if proc := s.getProc(svc.ID); proc != nil {
			if proc.proc != nil {
				s.gracefulStop(ctx, proc)
			}
			if proc.cancel != nil {
				proc.cancel()
			}
			s.setProc(svc.ID, nil)
		}
		s.setStatus(svc.ID, ServiceStatus{Skipped: true, Message: "full feature provider selected"})
	}
	return nil
}

func (s *Supervisor) serviceBundleRoot(svc manifest.Service) string {
	if svc.BundleRoot != "" {
		return svc.BundleRoot
	}
	return s.opts.BundlePath
}
func (s *Supervisor) serviceDataRoot(svc manifest.Service) string {
	if svc.ComponentID != "" {
		return filepath.Join(s.appData, "components", svc.ComponentID, "data")
	}
	return s.appData
}
func (s *Supervisor) serviceRenderer(svc manifest.Service) *env.Renderer {
	if svc.ComponentID == "" {
		return s.envRenderer
	}
	return env.NewRenderer(s.serviceDataRoot(svc), s.serviceBundleRoot(svc), s.portAllocator, s.envReader)
}
