package resources

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/credentialauthority"
	"github.com/vrooli/vrooli/internal/process"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// ManagedUpdateLegacyObservation is an observation of a historical launch,
// never a claim that today's declaration was its complete historical input.
// Contract retains the observed image with today's verified observable launch
// declarations. It requires isolated qualification before rollback admission.
// No actual arguments, environment or credential values enter this receipt.
type ManagedUpdateLegacyObservation struct {
	Resource                      string                           `json:"resource"`
	SourceRevision                string                           `json:"source_revision"`
	Process                       ManagedServiceState              `json:"process"`
	Scope                         platform.ScopeRef                `json:"scope"`
	NativeScope                   platform.ServiceScopeRef         `json:"native_scope"`
	Contract                      ManagedUpdateContract            `json:"contract"`
	ObservedAt                    time.Time                        `json:"observed_at"`
	ExactArgumentsVerified        bool                             `json:"exact_arguments_verified"`
	DeclaredEnvironmentVerified   bool                             `json:"declared_environment_verified"`
	CredentialBindingsVerified    bool                             `json:"credential_bindings_verified"`
	RequiresIsolatedQualification bool                             `json:"requires_isolated_qualification"`
	CredentialVersions            []ManagedUpdateCredentialVersion `json:"credential_versions,omitempty"`
}

type ManagedUpdateCredentialVersion struct {
	LogicalID string `json:"logical_id"`
	Field     string `json:"field"`
	Version   string `json:"version,omitempty"`
}

// ObserveManagedServiceLegacyLaunch verifies current ownership, exact argv,
// declared environment and logical credential-file binding, retains the actual
// running image, and reconfirms the process and supervisor record. It performs
// no serving mutation, ownership adoption or update admission.
func (c *Controller) ObserveManagedServiceLegacyLaunch(ctx context.Context, name string) (ManagedUpdateLegacyObservation, error) {
	if c == nil || ctx == nil {
		return ManagedUpdateLegacyObservation{}, fmt.Errorf("legacy observation requires an owner and context")
	}
	m, err := c.managedServiceArtifactManifest(name)
	if err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	if err := manifestpkg.ValidateManagedServiceUpdate(m); err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	release, err := acquireManagedLifecycleLocks(ctx, m)
	if err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	defer release()
	if err := ensureManagedUpdateLifecycleAvailable(name, "legacy launch observation"); err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	return c.observeManagedServiceLegacyLaunchLocked(ctx, m)
}

// Caller holds the resource lifecycle lock; this helper grants no stop authority.
func (c *Controller) observeManagedServiceLegacyLaunchLocked(ctx context.Context, m ResourceManifest) (ManagedUpdateLegacyObservation, error) {
	name := m.Name
	sup, paths, err := managedServiceSupervisorFor(name)
	if err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	state, running, err := sup.Status()
	if err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	if !running {
		return ManagedUpdateLegacyObservation{}, fmt.Errorf("legacy serving ownership is unavailable")
	}
	incarnation, err := process.Incarnation(state.PID)
	if err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	observed := state
	observed.Incarnation = incarnation
	if err := verifyManagedServiceOwnership(observed); err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	scopePath, err := platform.ProcessScope(state.PID)
	if err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	unit := resourceServiceScopeName(name, "service") + ".scope"
	if filepath.Base(scopePath) != unit || filepath.Base(filepath.Dir(scopePath)) != platform.ServicesSlice {
		return ManagedUpdateLegacyObservation{}, fmt.Errorf("legacy process is outside its exact resource service scope")
	}
	scope := platform.ScopeRef{Kind: platform.ScopeKindCgroup, Name: strings.TrimSuffix(unit, ".scope"), Path: scopePath}
	_, start, found := strings.Cut(incarnation, ":")
	if !found {
		return ManagedUpdateLegacyObservation{}, fmt.Errorf("legacy native root incarnation is not qualified")
	}
	nativeScope, err := platform.CaptureServiceScope(ctx, scope, state.PID, start)
	if err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	// Render only controlled owner paths and explicitly declared inputs. This
	// comparison does not import ambient caller variables into a new launch.
	env := []string{
		"HOME=" + c.Home, "ROOT=" + c.Root, "PATH=/usr/local/bin:/usr/bin:/bin",
		"RESOURCE_ROOT=" + filepath.Join(c.Root, "resources", name), "RESOURCE_CONFIG_DIR=" + paths.ConfigDir, "RESOURCE_CACHE_DIR=" + paths.CacheDir, "RESOURCE_LOGS_DIR=" + paths.LogsDir, "RESOURCE_STATE_DIR=" + paths.StateDir,
		"RESOURCE_ARTIFACT_DIR=" + filepath.Dir(state.ArtifactPath), "VROOLI_MANAGED_SERVICE_ARTIFACT=" + state.ArtifactPath,
	}
	for _, port := range m.Ports {
		env = append(env, resourcedeployment.ServicePortEnvironmentName(port.Name)+"="+strconv.Itoa(port.Host))
	}
	values := managedServiceEnvValues(env)
	expand, err := managedUpdateTemplateResolver(values, m.ManagedService.Environment)
	if err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	args, err := platform.ProcessArguments(state.PID)
	if err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	expected := []string{state.ArtifactPath}
	for _, arg := range m.ManagedService.ArgumentsFor(runtime.GOOS, runtime.GOARCH) {
		v, err := expand(arg)
		if err != nil {
			return ManagedUpdateLegacyObservation{}, err
		}
		expected = append(expected, v)
	}
	if !reflect.DeepEqual(args, expected) {
		return ManagedUpdateLegacyObservation{}, fmt.Errorf("legacy arguments differ from the exact declared launch boundaries or order")
	}
	actualEnv, err := process.ReadEnvironment(state.PID)
	if err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	for key := range m.ManagedService.Environment {
		if actualEnv[key] != values[key] {
			return ManagedUpdateLegacyObservation{}, fmt.Errorf("legacy declared environment differs from the verified declaration")
		}
	}
	var versions []ManagedUpdateCredentialVersion
	if len(m.ManagedService.CredentialFiles) > 0 {
		authority, err := credentialauthority.DefaultAuthority()
		if err != nil {
			return ManagedUpdateLegacyObservation{}, err
		}
		for _, declaration := range m.ManagedService.CredentialFiles {
			identity, err := credentialauthority.ParseIdentity(declaration.LogicalID)
			if err != nil {
				return ManagedUpdateLegacyObservation{}, err
			}
			field := declaration.Field
			if field == "" {
				field = managedUpdateCredentialDefaultField
			}
			beforeStatus := authority.Status(identity, field)
			if beforeStatus.ProviderState != credentialauthority.ProviderAvailable || !beforeStatus.Configured {
				return ManagedUpdateLegacyObservation{}, fmt.Errorf("legacy logical credential metadata is unavailable")
			}
			resolve := func() (string, error) {
				if err := ctx.Err(); err != nil {
					return "", err
				}
				return authority.Resolve(identity, field)
			}
			if err := verifyManagedUpdateCredentialBinding(filepath.Join(paths.StateDir, filepath.FromSlash(declaration.Path)), resolve); err != nil {
				return ManagedUpdateLegacyObservation{}, fmt.Errorf("legacy credential %s:%s binding unavailable or changed: %w", declaration.LogicalID, field, err)
			}
			afterStatus := authority.Status(identity, field)
			if afterStatus.ProviderState != credentialauthority.ProviderAvailable || !afterStatus.Configured || afterStatus.Version != beforeStatus.Version {
				return ManagedUpdateLegacyObservation{}, fmt.Errorf("legacy logical credential metadata changed during verification")
			}
			versions = append(versions, ManagedUpdateCredentialVersion{LogicalID: declaration.LogicalID, Field: field, Version: afterStatus.Version})
		}
	}
	image, err := platform.OpenProcessExecutable(state.PID)
	if err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	defer image.Close()
	retained, err := retainManagedUpdateFile(ctx, c, name, state.ArtifactVersion, state.ArtifactSHA256, image)
	if err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	source := sha256.Sum256(encoded)
	var frozen ResourceManifest
	if err := json.Unmarshal(encoded, &frozen); err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	frozen.ManagedService.Artifact.Version = state.ArtifactVersion
	frozen.ManagedService.Artifact.SHA256ByPlatform = nil
	frozen.ManagedService.Artifact.SHA256 = state.ArtifactSHA256
	contract, err := newManagedUpdateContractForResource(frozen, retained, filepath.Join(c.Root, "resources", name))
	if err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	after, live, err := sup.Status()
	if err != nil || !live || !reflect.DeepEqual(state, after) {
		return ManagedUpdateLegacyObservation{}, fmt.Errorf("legacy serving record changed during observation")
	}
	finalScope, scopeErr := platform.ProcessScope(state.PID)
	finalIncarnation, incErr := process.Incarnation(state.PID)
	finalArgs, argErr := platform.ProcessArguments(state.PID)
	if scopeErr != nil || incErr != nil || argErr != nil || finalScope != scopePath || finalIncarnation != incarnation || !reflect.DeepEqual(finalArgs, args) {
		return ManagedUpdateLegacyObservation{}, fmt.Errorf("legacy native identity or launch arguments changed during observation")
	}
	if err := verifyManagedServiceOwnership(observed); err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	currentNative, nativeErr := platform.InspectServiceScope(ctx, nativeScope)
	if nativeErr != nil || !currentNative.NativePresent || !currentNative.RootAlive || !currentNative.TreePopulated {
		return ManagedUpdateLegacyObservation{}, fmt.Errorf("legacy native scope changed during observation: %v", nativeErr)
	}
	if err := ctx.Err(); err != nil {
		return ManagedUpdateLegacyObservation{}, err
	}
	return ManagedUpdateLegacyObservation{Resource: name, SourceRevision: hex.EncodeToString(source[:]), Process: observed, Scope: scope, NativeScope: nativeScope, Contract: contract, ObservedAt: time.Now().UTC(), ExactArgumentsVerified: true, DeclaredEnvironmentVerified: true, CredentialBindingsVerified: true, RequiresIsolatedQualification: true, CredentialVersions: versions}, nil
}

func verifyManagedUpdateCredentialBinding(path string, resolve func() (string, error)) error {
	value, err := resolve()
	if err != nil {
		return fmt.Errorf("logical credential resolution failed")
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("materialized credential file is unavailable")
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() || before.Size() > 1<<20 {
		return fmt.Errorf("materialized credential file is not a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil {
		return fmt.Errorf("materialized credential read failed")
	}
	if len(data) > 1<<20 || subtle.ConstantTimeCompare(data, []byte(value+"\n")) != 1 {
		return fmt.Errorf("materialized credential differs from logical authority")
	}
	after, err := os.Stat(path)
	if err != nil || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return fmt.Errorf("materialized credential file changed during verification")
	}
	fresh, err := resolve()
	if err != nil || subtle.ConstantTimeCompare([]byte(value), []byte(fresh)) != 1 {
		return fmt.Errorf("logical credential changed during verification")
	}
	finalInfo, err := os.Stat(path)
	if err != nil || !os.SameFile(before, finalInfo) || before.Size() != finalInfo.Size() || !before.ModTime().Equal(finalInfo.ModTime()) {
		return fmt.Errorf("materialized credential changed during authority recheck")
	}
	return nil
}
