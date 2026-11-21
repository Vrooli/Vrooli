package resources

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/process"
)

func (c *Controller) prepareOpenRGBInitialAttempt(ctx context.Context, launch openRGBHeadlessLaunch) (openRGBInitialAttempt, error) {
	var a openRGBInitialAttempt
	manifest := launch.manifest
	catalogManifest := manifest
	var policy *OpenRGBDevicePolicyRecord
	var catalog []byte
	var err error
	if launch.retained != nil {
		retainedPolicy := launch.retained.record()
		policy = &retainedPolicy
		catalog = launch.retained.Catalog
	} else {
		// Effective serving settings are distinct from the source detector catalog.
		// Reload the complete declaration without acquiring bytes at startup.
		catalogManifest, err = c.managedServiceArtifactManifest(openRGBResourceName)
		if err != nil {
			return a, err
		}
		policy, err = c.loadOpenRGBDevicePolicy(catalogManifest)
		if err != nil || policy == nil {
			return a, errors.Join(errors.New("initial OpenRGB launch requires selected device policy"), err)
		}
		catalog, err = readManagedUpdateConfigFile(filepath.Join(c.Root, "resources", openRGBResourceName), catalogManifest.ManagedService.Config.ContentFile)
		if err != nil {
			return a, err
		}
	}
	decoded, err := decodeOpenRGBDetectorCatalog(catalog)
	if err != nil {
		return a, err
	}
	names, err := policy.selectedDetectors(decoded)
	if err != nil {
		return a, err
	}
	var settings string
	if launch.settings != nil {
		settings, err = mergeOpenRGBConfigContentWithSelection(launch.settings, string(catalog), names)
	} else {
		settings, err = c.freezeOpenRGBInitialSettings(catalogManifest, string(catalog), names)
	}
	if err != nil {
		return a, err
	}
	if policy.NativePolicy != nil {
		safe, e := enforceNativeOpenRGBSettings([]byte(settings))
		if e != nil {
			return a, e
		}
		settings = string(safe)
	}
	// Preserve the complete source acquisition declaration in the frozen contract.
	// This is metadata retention, not startup acquisition.
	frozen := catalogManifest
	service := *frozen.ManagedService
	declaration := *service.Config
	declaration.Content = ""
	declaration.ContentFile = catalogManifest.ManagedService.Config.ContentFile
	service.Config = &declaration
	frozen.ManagedService = &service
	contract, err := newManagedUpdateContractWithConfig(frozen, launch.policy.Artifact, declaration.ContentFile, []byte(settings))
	if err != nil {
		return a, err
	}
	binding, err := policy.freezeExecPolicy(contract.Artifact.SHA256, catalog, contract.ConfigBytes)
	if err != nil {
		return a, err
	}
	paths, err := resourceStoragePaths(openRGBResourceName)
	if err != nil {
		return a, err
	}
	id, err := newManagedServiceInstanceID()
	if err != nil {
		return a, err
	}
	var retainedLauncher *ManagedServiceState
	if policy.NativePolicy != nil && launch.retained != nil {
		supervisor, _, err := managedServiceSupervisorFor(openRGBResourceName)
		if err != nil {
			return a, err
		}
		state, err := supervisor.readState()
		if err != nil {
			return a, err
		}
		role, err := c.openRGBFrozenOriginalRole(ctx, state)
		if err != nil {
			return a, err
		}
		selectedID, _, _, err := launch.retained.policyIdentity()
		if err != nil {
			return a, err
		}
		recordedID, _, _, err := role.policyIdentity()
		if err != nil || recordedID != selectedID || role.ContractDigest != launch.retained.ContractDigest {
			return a, errors.Join(errors.New("retained native launcher differs from selected role"), err)
		}
		retainedLauncher = &state
	}
	image, err := c.openOpenRGBLauncherImage(ctx, policy.NativePolicy != nil, retainedLauncher)
	if err != nil {
		return a, err
	}
	defer image.Close()
	digest, err := managedUpdateImageDigest(ctx, image)
	if err != nil {
		return a, err
	}
	launcher := filepath.Join(paths.StateDir, "native-start", id, "launcher", digest, "vrooli")
	if policy.NativePolicy != nil {
		launcher, err = c.stageOpenRGBNativeLauncher(ctx, image, digest)
		if err != nil {
			return a, err
		}
	} else {
		if err := ensureManagedUpdatePrivateDirectory(paths.StateDir, filepath.Dir(launcher)); err != nil {
			return a, err
		}
		if _, err := image.Seek(0, io.SeekStart); err != nil {
			return a, err
		}
		if err := writeResourceOwnedFileDurableStream(launcher, 0755, func(dst io.Writer) error {
			_, err := io.Copy(dst, &managedUpdateContextReader{ctx: ctx, reader: image})
			return err
		}); err != nil {
			return a, err
		}
	}
	nativeAccount := ""
	ownerPrincipal := ""
	if policy.NativePolicy != nil {
		account, e := c.openRGBNativeServiceAccount(ctx)
		if e != nil || account.Principal != policy.NativePolicy.Account {
			return a, errors.Join(errors.New("OpenRGB native account differs from selected policy"), e)
		}
		nativeAccount = account.Name
		owner, e := readOpenRGBNativeOwner()
		if e != nil {
			return a, e
		}
		ownerPrincipal = owner.OwnerPrincipal
	}
	a = openRGBInitialAttempt{SchemaVersion: openRGBInitialSchemaVersion, ID: id, State: openRGBInitialPrepared, DevicePolicy: &policy.Policy, Catalog: catalog, Contract: contract, Policy: binding, Runtime: ManagedUpdateRuntime{OwnerPrincipal: ownerPrincipal, NativeAccount: nativeAccount, Root: c.Root, Home: c.Home, StateHome: os.Getenv("XDG_STATE_HOME"), StateDir: paths.StateDir, Launcher: launcher, LauncherSHA256: digest, Port: int(launch.endpoint.Port()), Limits: platform.ServiceLimits{MemoryMaxBytes: managedUpdateRoleMemory, TasksMax: 128, CPUWeight: 100}}}
	if policy.NativePolicy != nil {
		a.DevicePolicy = nil
		a.NativeDevicePolicy = policy.NativePolicy
	}
	if a.Runtime.NativeAccount != "" {
		a.Runtime.StateHome = ""
		a.Runtime.NativeLauncherProfile = platform.NativeLauncherProfileOpenRGB
		switch runtime.GOOS {
		case "darwin":
			a.Runtime.Limits.Enforcement = "native-advisory"
		case "windows":
			a.Runtime.Limits.Enforcement = "native-process-limits"
		}
	}
	tokenParent := a.root()
	if a.Runtime.NativeAccount != "" {
		tokenParent = filepath.Dir(a.tokenPath())
	}
	if err := ensureManagedUpdatePrivateDirectory(paths.StateDir, tokenParent); err != nil {
		return a, err
	}
	token, err := newManagedServiceToken()
	if err != nil {
		return a, err
	}
	if err := writeResourceOwnedFileDurable(a.tokenPath(), []byte(token), 0600); err != nil {
		return a, err
	}
	a.TokenHash = managedServiceTokenHash(token)
	if _, err := renderManagedServiceInvocation(contract, a.Runtime, a.root(), token); err != nil {
		return a, err
	}
	a.Intent, err = platform.NewServiceIntent("resource-initial", id, a.wrapperSpec())
	if err != nil {
		return a, err
	}
	return a, a.validate()
}

// Freeze only the resource-owned preferences. The role receives its own copy;
// preparation never modifies the desktop or previous resource configuration.
func (c *Controller) freezeOpenRGBInitialSettings(manifest ResourceManifest, catalog string, names []string) (string, error) {
	paths, err := resourceStoragePaths(openRGBResourceName)
	if err != nil {
		return "", err
	}
	if err := rejectOpenRGBAmbientConfigRoot(paths.ConfigDir, c.Home); err != nil {
		return "", err
	}
	path := filepath.Join(paths.ConfigDir, filepath.FromSlash(manifest.ManagedService.Config.Path))
	resolvedRoot, err := resolvedPathWithExistingPrefix(paths.ConfigDir)
	if err != nil {
		return "", err
	}
	resolvedPath, err := resolvedPathWithExistingPrefix(path)
	if err != nil {
		return "", err
	}
	if rel, err := filepath.Rel(resolvedRoot, resolvedPath); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("initial OpenRGB preferences escape resource config owner")
	}
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return "", errors.New("initial OpenRGB preferences must be a regular file")
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return mergeOpenRGBConfigWithSelection(path, catalog, names)
}

type openRGBInitialExecOwner struct {
	lookup      func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error)
	selfImage   func(int) (*os.File, error)
	incarnation func(int) (string, error)
	confine     func() (func(), error)
	image       func(context.Context, ManagedServiceArtifact, managedUpdateInvocation, int) (openRGBAppImageStage, managedUpdateInvocation, *os.File, error)
	witness     func(context.Context, openRGBInitialAttempt, managedUpdateInvocation, platform.ServiceRef, string) (openRGBExecWitness, error)
	claim       func(context.Context, managedUpdateExecFence, platform.ServiceRef, openRGBExecWitness) (managedUpdateExecFence, error)
	chdir       func(string) error
	replace     func(*os.File, []string, []string) error
}

// RunManagedServiceExec consumes only an exact durable initial admission. The
// selectors do not grant authority to run arbitrary resource payloads.
func (c *Controller) RunManagedServiceExec(ctx context.Context, resource, attempt string) error {
	if resource != openRGBResourceName || !managedUpdateKey.MatchString(attempt) {
		return errors.New("initial native exec requires an admitted OpenRGB attempt")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	store, err := newOpenRGBInitialStore()
	if err != nil {
		return err
	}
	a, err := store.read(attempt)
	if err != nil {
		return err
	}
	current, err := store.current()
	if err != nil || current != attempt || a.State != openRGBInitialSubmitted {
		return errors.New("initial OpenRGB attempt is stale, consumed or revoked")
	}
	if a.Runtime.Root != c.Root || a.Runtime.Home != c.Home {
		return errors.New("initial OpenRGB runtime owner changed")
	}
	state, err := c.operatorStateService().Load(ctx)
	if err != nil {
		return err
	}
	choice := state.Resources[openRGBResourceName]
	if choice.Enabled == nil || !*choice.Enabled || choice.Source == nil || (choice.Source.Mode != "managed" && choice.Source.Mode != "host") {
		return errors.New("initial OpenRGB resource is disabled or source changed")
	}
	selected, err := NewOpenRGBHeadless(c, c.OpenRGBDevicePolicyVerifier()).inspectSelection(ctx, lightingruntime.Selection{Version: a.Contract.Artifact.Version, SHA256: a.Contract.Artifact.SHA256, DevicePolicyID: a.Policy.ID})
	if err != nil {
		return err
	}
	if selected.policy.Artifact != a.Contract.Artifact || int(selected.endpoint.Port()) != a.Runtime.Port {
		return errors.New("initial OpenRGB source selection differs from frozen admission")
	}
	fences, err := newManagedInitialExecStore(a.ID)
	if err != nil {
		return err
	}
	fence, err := fences.read()
	if err != nil {
		return err
	}
	if !sameManagedExecIdentity(fence, a.fence()) || fence.State != managedUpdateExecFencePrepared {
		return errors.New("initial OpenRGB execution fence is consumed or revoked")
	}
	return executeOpenRGBInitial(ctx, a, openRGBInitialExecOwner{lookup: platform.LookupOwnedService, selfImage: platform.OpenProcessExecutable, incarnation: process.Incarnation, confine: confineOpenRGBExecThread, image: c.prepareOpenRGBAppImageExecution, witness: c.observeOpenRGBInitialWitness, claim: fences.ClaimOpenRGB, chdir: os.Chdir, replace: platform.ReplaceProcessImage})
}
func executeOpenRGBInitial(ctx context.Context, a openRGBInitialAttempt, o openRGBInitialExecOwner) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.validate(); err != nil {
		return err
	}
	if a.State != openRGBInitialSubmitted {
		return errors.New("initial OpenRGB execution has no unused submission")
	}
	self, err := o.selfImage(os.Getpid())
	if err != nil {
		return err
	}
	digest, err := managedUpdateImageDigest(ctx, self)
	_ = self.Close()
	if err != nil {
		return err
	}
	if digest != a.Runtime.LauncherSHA256 {
		return errors.New("initial OpenRGB launcher image changed")
	}
	ref, err := o.lookup(ctx, a.Intent)
	if err != nil {
		return err
	}
	incarnation, err := o.incarnation(os.Getpid())
	if err != nil {
		return err
	}
	if ref.ValidateIdentity() != nil || ref.Intent != platform.OperationIntent(a.Intent) || ref.RootPID != os.Getpid() || !ref.MatchesIncarnation(incarnation) {
		return errors.New("initial OpenRGB wrapper is outside its exact native root")
	}
	token, err := readManagedUpdateToken(a.tokenPath())
	if err != nil {
		return err
	}
	if managedServiceTokenHash(token) != a.TokenHash {
		return errors.New("initial OpenRGB ownership token changed")
	}
	base, err := renderManagedServiceInvocation(a.Contract, a.Runtime, a.root(), token)
	if err != nil {
		return err
	}
	release, err := o.confine()
	if err != nil {
		return err
	}
	defer release()
	_, derived, image, err := o.image(ctx, a.Contract.Artifact, base, a.Runtime.Port)
	if err != nil {
		return err
	}
	defer image.Close()
	witness, err := o.witness(ctx, a, base, ref, incarnation)
	if err != nil {
		return err
	}
	confirmed, err := o.lookup(ctx, a.Intent)
	if err != nil || !confirmed.Equal(ref) {
		return errors.Join(errors.New("initial OpenRGB native owner changed during witness"), err)
	}
	if _, err := o.claim(ctx, a.fence(), ref, witness); err != nil {
		return err
	}
	// Claim consumption precedes config publication and FD exec. A failed or
	// uncertain exec cannot authorize a second execution of this attempt.
	if err := materializeOpenRGBInitial(base); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := o.chdir(derived.Dir); err != nil {
		return err
	}
	return o.replace(image, append([]string{derived.Path}, derived.Args...), derived.Env)
}
func materializeOpenRGBInitial(invocation managedUpdateInvocation) error {
	values := managedServiceEnvValues(invocation.Env)
	if err := ensureManagedUpdatePrivateDirectory(invocation.StateRoot, invocation.RuntimeRoot); err != nil {
		return err
	}
	for _, key := range []string{"HOME", "RESOURCE_CONFIG_DIR", "RESOURCE_CACHE_DIR", "RESOURCE_LOGS_DIR", "RESOURCE_STATE_DIR"} {
		if err := ensureManagedUpdatePrivateDirectory(invocation.RuntimeRoot, values[key]); err != nil {
			return err
		}
	}
	if err := ensureManagedUpdatePrivateDirectory(invocation.RuntimeRoot, filepath.Dir(invocation.ConfigPath)); err != nil {
		return err
	}
	return writeResourceOwnedFileDurable(invocation.ConfigPath, []byte(invocation.ConfigContent), 0600)
}
func (c *Controller) observeOpenRGBInitialWitness(ctx context.Context, a openRGBInitialAttempt, base managedUpdateInvocation, ref platform.ServiceRef, incarnation string) (openRGBExecWitness, error) {
	if a.NativeDevicePolicy != nil {
		return observeOpenRGBNativeAdmittedWitness(ctx, *a.NativeDevicePolicy, a.Contract, base, a.Catalog, a.Contract.ConfigBytes, openRGBWitnessAdmission{Policy: &a.Policy, Port: a.Runtime.Port, ContractDigest: a.Contract.Digest, ClaimSHA256: a.claimDigest(ref)}, ref, incarnation, openRGBExecContext, hostreqkit.ObserveNativeOpenRGBDevicePolicy)
	}
	var policy hostreqkit.OpenRGBLinuxPolicy
	catalog := a.Catalog
	if a.DevicePolicy != nil {
		policy = *a.DevicePolicy
	} else {
		manifest, err := c.managedServiceArtifactManifest(openRGBResourceName)
		if err != nil {
			return openRGBExecWitness{}, err
		}
		record, err := c.loadOpenRGBDevicePolicy(manifest)
		if err != nil || record == nil {
			return openRGBExecWitness{}, errors.Join(errors.New("initial OpenRGB policy is unavailable"), err)
		}
		policy = record.Policy
		catalog, err = readManagedUpdateConfigFile(filepath.Join(c.Root, "resources", openRGBResourceName), a.Contract.ConfigFile)
		if err != nil {
			return openRGBExecWitness{}, err
		}
	}
	return observeOpenRGBAdmittedWitness(ctx, policy, a.Contract, base, catalog, a.Contract.ConfigBytes, openRGBWitnessAdmission{Policy: &a.Policy, Port: a.Runtime.Port, ContractDigest: a.Contract.Digest, ClaimSHA256: a.claimDigest(ref)}, ref, incarnation, os.Geteuid(), openRGBExecContext, hostreqkit.ObserveLinuxOpenRGBDevicePolicy)
}
