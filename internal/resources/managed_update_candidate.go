package resources

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/process"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	resourcehealth "github.com/vrooli/vrooli/internal/resources/runtime/health"
)

// AdvanceManagedUpdateCandidate advances only an already-admitted operation's
// parallel candidate. It never stops or publishes a serving primary. A resumed
// launch reconciles the persisted intent; absence or caller loss never causes
// redispatch. The complete promotion/recovery runner owns subsequent phases.
func (c *Controller) AdvanceManagedUpdateCandidate(ctx context.Context, resource, request string) (ManagedServiceUpdate, error) {
	if resource == openRGBResourceName {
		return c.runOpenRGBPublicUpdate(ctx, func(ctx context.Context) (ManagedServiceUpdate, error) {
			return c.advanceManagedUpdateCandidate(ctx, resource, request)
		})
	}
	return c.advanceManagedUpdateCandidate(ctx, resource, request)
}

func (c *Controller) advanceManagedUpdateCandidate(ctx context.Context, resource, request string) (ManagedServiceUpdate, error) {
	if !managedUpdateResource.MatchString(resource) || !managedUpdateKey.MatchString(request) {
		return ManagedServiceUpdate{}, fmt.Errorf("invalid candidate selector")
	}
	release, err := acquireManagedServiceLifecycleLock(ctx, resource)
	if err != nil {
		return ManagedServiceUpdate{}, err
	}
	defer release()
	s, err := newManagedUpdateStore(resource)
	if err != nil {
		return ManagedServiceUpdate{}, err
	}
	u, err := s.Lookup(request)
	if err != nil {
		return ManagedServiceUpdate{}, err
	}
	u, err = s.Confirm(ctx, request, u.Revision)
	if err != nil {
		return ManagedServiceUpdate{}, err
	}
	if resource == openRGBResourceName {
		if err := c.qualifyOpenRGBResumption(ctx, u); err != nil {
			return u, err
		}
		if !exclusiveStopStart(u) {
			return u, fmt.Errorf("OpenRGB profile requires exclusive native replacement")
		}
		if u.Phase != UpdatePrepared {
			return u, fmt.Errorf("exclusive OpenRGB candidate preflight requires prepared admission; resume the update for %s", u.Phase)
		}
		// Hardware access cannot be duplicated by a parallel candidate. Prove
		// retained original in place; the full runner repeats proof before stop.
		_, err := c.managedUpdateEffectOwner().original(ctx, u)
		return u, err
	}
	return c.advanceManagedUpdateCandidateLocked(ctx, u, s)
}

func (c *Controller) advanceManagedUpdateCandidateLocked(ctx context.Context, u ManagedServiceUpdate, s *managedUpdateStore) (ManagedServiceUpdate, error) {
	var err error
	u, err = c.advanceManagedUpdateRollbackProbe(ctx, u, s)
	if err != nil {
		return u, err
	}
	if u.Phase == UpdatePrepared {
		intent, err := c.prepareManagedUpdateIsolatedRole(ctx, u, u.Target, "candidate")
		if err != nil {
			return u, err
		}
		u.Candidate.Intent = &intent
		u, err = s.Commit(ctx, u, UpdateLaunchCandidate)
		if err != nil {
			return ManagedServiceUpdate{}, err
		}
		// This is the only dispatch edge, reached only by the owner that just
		// admitted the transition under the lifecycle lock. Reloads skip it.
		if _, err := platform.StartOwnedService(ctx, intent.NativeIntent, managedUpdateWrapperSpec(u, intent)); err != nil {
			return u, fmt.Errorf("candidate dispatch outcome needs reconciliation: %w", err)
		}
	}
	if u.Phase != UpdateCandidateStarting && u.Phase != UpdateCandidateVerifying && u.Phase != UpdateCandidateReady {
		return u, fmt.Errorf("phase %s is outside candidate reconciliation", u.Phase)
	}
	if u.Candidate.Intent == nil {
		return u, fmt.Errorf("candidate has no retained launch identity")
	}
	observed, err := waitManagedUpdateProcess(ctx, u, u.Candidate, u.Target)
	if err != nil {
		return u, err
	}
	if u.Phase == UpdateCandidateReady {
		// Preserve the exact durable receipt. The fresh observation above does
		// not invent a new checkpoint; promotion must refresh its own proof.
		return u, nil
	}
	u.Candidate = observed
	if u.Phase == UpdateCandidateStarting {
		u, err = s.Commit(ctx, u, UpdateCandidateStarted)
		if err != nil {
			return ManagedServiceUpdate{}, err
		}
	}
	return s.Commit(ctx, u, UpdateCandidateVerified)
}

func updateOperationRoleDigest(operation, role string) string {
	return managedServiceTokenHash(operation + "\x00" + role)[:40]
}

func observeManagedUpdateProcess(ctx context.Context, u ManagedServiceUpdate, p ManagedUpdateProcess, c ManagedUpdateContract) (ManagedUpdateProcess, error) {
	if !validUpdateIntent(p.Intent, c) {
		return p, fmt.Errorf("invalid role launch identity")
	}
	i := *p.Intent
	ref, err := platform.LookupOwnedService(ctx, i.NativeIntent)
	if err != nil {
		return p, err
	}
	if ref.RootPID <= 0 {
		return p, fmt.Errorf("native root is not observable")
	}
	if p.NativeRef != nil && !reflect.DeepEqual(*p.NativeRef, ref) {
		return p, fmt.Errorf("native invocation changed")
	}
	native, err := platform.InspectOwnedService(ctx, ref)
	if err != nil {
		return p, err
	}
	if native.State != platform.OperationRunning || !native.RootAlive || !native.Populated {
		return p, fmt.Errorf("native role is not running")
	}
	incarnation, err := process.Incarnation(ref.RootPID)
	if err != nil {
		return p, err
	}
	_, ticks, ok := strings.Cut(incarnation, ":")
	if !ok || ticks != ref.RootStartTime {
		return p, fmt.Errorf("native root incarnation changed")
	}
	fences, err := newManagedUpdateExecStore(u.Resource, u.ID, i.InstanceID)
	if err != nil {
		return p, err
	}
	fence, err := fences.read()
	if err != nil {
		return p, err
	}
	if fence.Intent != i || fence.State != managedUpdateExecFenceClaimed || fence.Claim == nil || fence.ClaimedAt == nil || !reflect.DeepEqual(*fence.Claim, ref) {
		return p, fmt.Errorf("resource exec claim is absent, revoked or belongs to another native identity")
	}
	root, err := i.Runtime.roleRoot(u.ID, i.InstanceID)
	if err != nil {
		return p, err
	}
	token, err := readManagedUpdateToken(filepath.Join(root, "ownership-token"))
	if err != nil {
		return p, err
	}
	if managedServiceTokenHash(token) != i.OwnershipTokenHash {
		return p, fmt.Errorf("role token differs from admission")
	}
	invocation, err := renderManagedUpdateInvocation(c, i.Runtime, u.ID, i.InstanceID, token)
	if err != nil {
		return p, err
	}
	executing := invocation
	expectedImageSHA := c.Artifact.SHA256
	var appImageStage *openRGBAppImageStage
	if u.Resource == openRGBResourceName {
		owner := &Controller{Root: i.Runtime.Root, Home: i.Runtime.Home}
		appImageStage, err = owner.verifyOpenRGBRoleArtifact(ctx, c.Artifact)
		if err != nil {
			return p, err
		}
		if appImageStage != nil {
			executing, err = appImageStage.servingInvocation(invocation, i.Runtime.Port)
			if err != nil {
				return p, err
			}
			expectedImageSHA = appImageStage.payloadSHA256
		}
	}
	actualEnv, err := process.ReadEnvironment(ref.RootPID)
	if err != nil {
		return p, err
	}
	if !reflect.DeepEqual(actualEnv, managedServiceEnvValues(executing.Env)) {
		return p, fmt.Errorf("running role environment differs from frozen inputs")
	}
	args, err := platform.ProcessArguments(ref.RootPID)
	if err != nil {
		return p, err
	}
	if !reflect.DeepEqual(args, append([]string{executing.Path}, executing.Args...)) {
		return p, fmt.Errorf("running role arguments differ from frozen inputs")
	}
	image, err := platform.OpenProcessExecutable(ref.RootPID)
	if err != nil {
		return p, err
	}
	imageInfo, err := image.Stat()
	if err != nil {
		_ = image.Close()
		return p, err
	}
	digest, err := managedUpdateImageDigest(ctx, image)
	_ = image.Close()
	if err != nil {
		return p, err
	}
	if digest != expectedImageSHA {
		return p, fmt.Errorf("running role image differs from frozen checksum")
	}
	if err := managedUpdateOwnedListener(i.Runtime.Port, ref.RootPID); err != nil {
		return p, err
	}
	var openRGBServing *openRGBServingObservation
	if u.Resource == openRGBResourceName {
		observation, err := newOpenRGBServingObservation(fence, i, c, invocation, ref, incarnation)
		if err != nil {
			return p, err
		}
		if err := observation.verifyRoleImage(ctx, appImageStage, imageInfo, digest); err != nil {
			return p, fmt.Errorf("OpenRGB launch witness is not serving in its admitted context: %w", err)
		}
		openRGBServing = &observation
	}
	values := managedServiceEnvValues(invocation.Env)
	client, err := observeManagedUpdateReadiness(ctx, c, i.Runtime.Port, values, invocation.Env)
	if err != nil {
		return p, err
	}
	defer client.CloseIdleConnections()
	metrics, err := observeManagedUpdateMetrics(ctx, client, i.Runtime.Port, c)
	if err != nil {
		return p, err
	}
	if err := managedUpdateOwnedListener(i.Runtime.Port, ref.RootPID); err != nil {
		return p, err
	}
	after, err := platform.InspectOwnedService(ctx, ref)
	if err != nil {
		return p, err
	}
	finalIncarnation, err := process.Incarnation(ref.RootPID)
	if err != nil {
		return p, err
	}
	if after.State != platform.OperationRunning || !after.RootAlive || !after.Populated || finalIncarnation != incarnation {
		return p, fmt.Errorf("native role changed during readiness observation")
	}
	finalImage, err := platform.OpenProcessExecutable(ref.RootPID)
	if err != nil {
		return p, err
	}
	finalImageInfo, err := finalImage.Stat()
	_ = finalImage.Close()
	if err != nil {
		return p, err
	}
	if !os.SameFile(imageInfo, finalImageInfo) {
		return p, fmt.Errorf("native role executed another image during readiness observation")
	}
	if openRGBServing != nil {
		if err := openRGBServing.verifyRoleImage(ctx, appImageStage, finalImageInfo, digest); err != nil {
			return p, fmt.Errorf("OpenRGB launch witness changed during readiness observation: %w", err)
		}
	}
	state := ManagedServiceState{InstanceID: i.InstanceID, OwnershipTokenHash: i.OwnershipTokenHash, PID: ref.RootPID, Incarnation: incarnation, ArtifactPath: c.Artifact.Path, ArtifactSHA256: c.Artifact.SHA256, ArtifactVersion: c.Artifact.Version, StartedAt: *fence.ClaimedAt}
	if p.State != nil && !reflect.DeepEqual(*p.State, state) {
		return p, fmt.Errorf("recorded process identity differs from native observation")
	}
	p.State, p.NativeRef = &state, &ref
	p.Proof = &ManagedUpdateProof{PID: state.PID, Incarnation: incarnation, InstanceID: i.InstanceID, ArtifactSHA256: state.ArtifactSHA256, ObservedAt: time.Now().UTC(), OwnedListener: true, Ready: true}
	p.Proof.Metrics = metrics
	return p, nil
}

func observeManagedUpdateReadiness(ctx context.Context, c ManagedUpdateContract, port int, values map[string]string, env []string) (*http.Client, error) {
	var ready *ResourceHealthCheck
	for i := range c.Manifest.HealthChecks {
		if c.Manifest.HealthChecks[i].Kind == manifestpkg.HealthCheckReadiness {
			ready = &c.Manifest.HealthChecks[i]
			break
		}
	}
	if ready == nil {
		return nil, fmt.Errorf("update contract has no readiness check")
	}
	isOpenRGB := c.Manifest.Name == openRGBResourceName
	isOpenRGBSDKCheck := ready.Type == manifestpkg.HealthCheckOpenRGBV6
	if isOpenRGB != isOpenRGBSDKCheck {
		return nil, fmt.Errorf("managed update readiness type does not match its resource")
	}
	if isOpenRGB {
		rendered := *ready
		rendered.Target = os.Expand(rendered.Target, func(k string) string { return values[k] })
		host, actualPort, err := net.SplitHostPort(rendered.Target)
		if err != nil || host != "127.0.0.1" || actualPort != strconv.Itoa(port) {
			return nil, fmt.Errorf("OpenRGB SDK readiness escaped the exact role loopback endpoint")
		}
		result, err := resourcehealth.RunCheck(ctx, rendered, resourcehealth.Config{Env: env})
		if err != nil {
			return nil, fmt.Errorf("managed update readiness check failed: %w", err)
		}
		if !result.Healthy {
			return nil, fmt.Errorf("managed update readiness failed: %s", result.Message)
		}
		return &http.Client{Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true}, Timeout: 2 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, nil
	}
	readyURL := os.Expand(ready.Target, func(k string) string { return values[k] })
	endpoint, err := url.Parse(readyURL)
	if err != nil || endpoint.Scheme != "http" || endpoint.Host != "127.0.0.1:"+strconv.Itoa(port) || endpoint.User != nil {
		return nil, fmt.Errorf("readiness escaped the exact role loopback endpoint")
	}
	transport := &http.Transport{Proxy: nil, DisableKeepAlives: true}
	client := &http.Client{Transport: transport, Timeout: 2 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, readyURL, nil)
	if err != nil {
		transport.CloseIdleConnections()
		return nil, err
	}
	response, err := client.Do(req)
	if err != nil {
		transport.CloseIdleConnections()
		return nil, err
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		transport.CloseIdleConnections()
		return nil, fmt.Errorf("role readiness returned HTTP %d", response.StatusCode)
	}
	return client, nil
}

func managedUpdateOwnedListener(port, pid int) error {
	listeners := network.InspectPortListeners(port)
	if !listeners.Inspection.Available || len(listeners.Listeners) == 0 {
		return fmt.Errorf("role listener ownership is unavailable")
	}
	for _, listener := range listeners.Listeners {
		if listener.PID != pid || listener.Zombie {
			return fmt.Errorf("role listener is not exclusively owned by its exact native root")
		}
	}
	return nil
}

func (c *Controller) prepareManagedUpdateIsolatedRole(ctx context.Context, u ManagedServiceUpdate, contract ManagedUpdateContract, role string) (ManagedUpdateLaunchIntent, error) {
	instance := "mu-" + updateOperationRoleDigest(u.ID, role)
	fences, err := newManagedUpdateExecStore(u.Resource, u.ID, instance)
	if err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	port := 0
	if prior, err := fences.read(); err == nil {
		port = prior.Intent.Runtime.Port
	} else if !os.IsNotExist(err) {
		return ManagedUpdateLaunchIntent{}, err
	}
	if port == 0 {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return ManagedUpdateLaunchIntent{}, err
		}
		port = listener.Addr().(*net.TCPAddr).Port
		_ = listener.Close()
	}
	if len(contract.Manifest.Ports) != 1 || port == contract.Manifest.Ports[0].Host {
		return ManagedUpdateLaunchIntent{}, fmt.Errorf("candidate requires a distinct free loopback port")
	}
	return c.prepareManagedUpdateLaunch(ctx, u, contract, instance, port)
}

func waitManagedUpdateProcess(ctx context.Context, u ManagedServiceUpdate, process ManagedUpdateProcess, contract ManagedUpdateContract) (ManagedUpdateProcess, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	var observed ManagedUpdateProcess
	var last error
	for {
		observed, last = observeManagedUpdateProcess(probeCtx, u, process, contract)
		if last == nil {
			break
		}
		select {
		case <-probeCtx.Done():
			return ManagedUpdateProcess{}, fmt.Errorf("candidate reconciliation remains unresolved: %w; last observation: %v", probeCtx.Err(), last)
		case <-time.After(250 * time.Millisecond):
		}
	}
	return observed, nil
}

func (c *Controller) advanceManagedUpdateRollbackProbe(ctx context.Context, u ManagedServiceUpdate, s *managedUpdateStore) (ManagedServiceUpdate, error) {
	if exclusiveStopStart(u) {
		return u, nil
	}
	if !u.RollbackQualificationRequired || u.RollbackQualifiedAt != nil {
		return u, nil
	}
	if u.Phase == UpdatePrepared {
		intent, err := c.prepareManagedUpdateIsolatedRole(ctx, u, u.Rollback, "rollback-qualification")
		if err != nil {
			return u, err
		}
		u.RollbackProbe.Intent = &intent
		u, err = s.Commit(ctx, u, UpdateLaunchRollbackProbe)
		if err != nil {
			return ManagedServiceUpdate{}, err
		}
		if _, err := platform.StartOwnedService(ctx, intent.NativeIntent, managedUpdateWrapperSpec(u, intent)); err != nil {
			return u, fmt.Errorf("rollback qualification dispatch needs reconciliation: %w", err)
		}
	}
	if u.Phase != UpdateRollbackProbeStarting && u.Phase != UpdateRollbackProbeVerifying {
		return u, fmt.Errorf("rollback qualification cannot advance from %s", u.Phase)
	}
	observed, err := waitManagedUpdateProcess(ctx, u, u.RollbackProbe, u.Rollback)
	if err != nil {
		return u, err
	}
	u.RollbackProbe = observed
	if u.Phase == UpdateRollbackProbeStarting {
		u, err = s.Commit(ctx, u, UpdateRollbackProbeStarted)
		if err != nil {
			return ManagedServiceUpdate{}, err
		}
	}
	return s.Commit(ctx, u, UpdateRollbackProbeVerified)
}
