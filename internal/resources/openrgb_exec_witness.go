package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

const (
	openRGBPlatformDarwin  = "darwin"
	openRGBPlatformWindows = "windows"
)

// The owner freezes this binding in the exec fence before admitting or
// dispatching a role. The policy ID binds the host, grant and physical selectors.
type openRGBExecPolicy struct {
	Platform              string `json:"platform,omitempty"`
	Account               string `json:"account,omitempty"`
	ID                    string `json:"id"`
	ArtifactSHA256        string `json:"artifact_sha256"`
	DetectorCatalogSHA256 string `json:"detector_catalog_sha256"`
	ConfigSHA256          string `json:"config_sha256"`
}

type openRGBExecWitness struct {
	TransportSHA256        string                       `json:"transport_sha256,omitempty"`
	ExecutionChild         *openRGBNativeExecutionChild `json:"execution_child,omitempty"`
	DeviceGenerationSHA256 string                       `json:"device_generation_sha256,omitempty"`
	Policy                 openRGBExecPolicy            `json:"policy"`
	Incarnation            string                       `json:"incarnation"`
	ClaimSHA256            string                       `json:"claim_sha256"`
	ContextSHA256          string                       `json:"context_sha256"`
	SelectedNodes          []string                     `json:"selected_nodes"`
}

type openRGBExecOwner struct {
	readFence         func() (managedUpdateExecFence, error)
	claim             func(context.Context, managedUpdateExecFence, platform.ServiceRef, openRGBExecWitness) (managedUpdateExecFence, error)
	observe           func(context.Context, ManagedUpdateContract, managedUpdateInvocation, managedUpdateExecFence, platform.ServiceRef, string) (openRGBExecWitness, error)
	confineExecThread func() (func(), error)
}

func (o *openRGBExecOwner) claimVerified(ctx context.Context, contract ManagedUpdateContract, invocation managedUpdateInvocation, expected managedUpdateExecFence, ref platform.ServiceRef, incarnation string, lookup func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error)) error {
	f, err := o.readFence()
	if err != nil {
		return err
	}
	if !sameManagedExecIdentity(f, expected) {
		return ErrManagedUpdateConflict
	}
	switch f.State {
	case managedUpdateExecFenceClaimed:
		return ErrManagedUpdateExecClaimed
	case managedUpdateExecFenceRevoked:
		return ErrManagedUpdateExecRevoked
	}
	if f.OpenRGBPolicy == nil {
		return errors.New("OpenRGB launch has no frozen device policy admission")
	}
	if !ref.NativeOwnedIdentityComplete() {
		return errors.New("OpenRGB launch has no complete exact native containment identity")
	}
	witness, err := o.observe(ctx, contract, invocation, f, ref, incarnation)
	if err != nil {
		return err
	}
	confirmed, err := lookup(ctx, f.Intent.NativeIntent)
	if err != nil || !confirmed.Equal(ref) {
		return errors.Join(errors.New("OpenRGB native ownership changed during device observation"), err)
	}
	_, err = o.claim(ctx, f, ref, witness)
	return err
}

var openRGBWitnessNode = regexp.MustCompile(`^/dev/(i2c-|hidraw)[0-9]+$`)

func (p openRGBExecPolicy) valid() bool {
	if !updateDigest(p.ID) || !updateDigest(p.ArtifactSHA256) || !updateDigest(p.DetectorCatalogSHA256) || !updateDigest(p.ConfigSHA256) {
		return false
	}
	if p.Platform == "" {
		return p.Account == ""
	}
	return (p.Platform == openRGBPlatformDarwin || p.Platform == openRGBPlatformWindows) && hostreqkit.ValidateNativeOpenRGBAccount(p.Platform, p.Account) == nil
}

func sameOpenRGBExecPolicy(a, b *openRGBExecPolicy) bool {
	return a == b || a != nil && b != nil && *a == *b
}

func (f managedUpdateExecFence) validateOpenRGBWitness() error {
	if f.OpenRGBPolicy == nil && f.OpenRGBWitness == nil {
		// Legacy fences must remain readable for exact stop and revocation.
		return nil
	}
	if f.Resource != openRGBResourceName || f.OpenRGBPolicy == nil || !f.OpenRGBPolicy.valid() {
		return errors.New("invalid OpenRGB policy on resource exec fence")
	}
	if (f.OpenRGBWitness == nil) != (f.Claim == nil) {
		return errors.New("OpenRGB exec claim and device witness must be recorded together")
	}
	if f.OpenRGBWitness == nil {
		return nil
	}
	w := f.OpenRGBWitness
	if w.Policy.Platform != "" {
		expected, err := hostreqkit.NativeOpenRGBTransportPolicySHA256(w.Policy.Platform)
		if err != nil || w.TransportSHA256 != expected {
			return errors.New("OpenRGB native transport confinement witness differs from qualified policy")
		}
	} else if w.TransportSHA256 != "" {
		return errors.New("legacy OpenRGB witness cannot carry native transport confinement")
	}

	if w.Policy.Platform == openRGBPlatformWindows {
		if w.ExecutionChild == nil || w.ExecutionChild.validateFor(*f.Claim) != nil {
			return errors.New("OpenRGB Windows witness lacks an exact admitted native child")
		}
	} else if w.ExecutionChild != nil {
		return errors.New("OpenRGB same-root execution cannot carry a native child receipt")
	}
	if w.DeviceGenerationSHA256 != "" && !updateDigest(w.DeviceGenerationSHA256) {
		return errors.New("invalid OpenRGB device generation witness")
	}
	if w.Policy != *f.OpenRGBPolicy || w.ClaimSHA256 != f.openRGBClaimDigest(*f.Claim) || !f.Claim.MatchesIncarnation(w.Incarnation) || !f.Claim.NativeOwnedIdentityComplete() || !updateDigest(w.ContextSHA256) || len(w.SelectedNodes) == 0 || len(w.SelectedNodes) > 16 {
		return errors.New("OpenRGB device witness differs from its exact launch claim")
	}
	if w.Policy.Platform != "" && (f.Claim.Native == nil || f.Claim.Native.Platform != w.Policy.Platform || !updateDigest(w.DeviceGenerationSHA256)) {
		return errors.New("OpenRGB native device witness differs from its platform claim")
	}
	for i, node := range w.SelectedNodes {
		validNode := openRGBWitnessNode.MatchString(node)
		if w.Policy.Platform != "" {
			validNode = node != "" && len(node) <= 4096 && !strings.ContainsAny(node, "\x00\r\n")
		}
		if !validNode || i > 0 && w.SelectedNodes[i-1] >= node {
			return errors.New("OpenRGB device witness contains an invalid or duplicate selected node")
		}
	}
	return nil
}

func (f managedUpdateExecFence) openRGBClaimDigest(ref platform.ServiceRef) string {
	// The native identity and admitted runtime contain only serializable values.
	// The token is represented by its existing hash in Intent.
	data, _ := json.Marshal(struct {
		Resource      string
		OperationID   string
		Intent        ManagedUpdateLaunchIntent
		Ref           platform.ServiceRef
		AdmissionKind managedExecAdmissionKind `json:",omitempty"`
	}{f.Resource, f.OperationID, f.Intent, ref, f.AdmissionKind})
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

// freezeOpenRGBExecPolicy binds the actual frozen settings bytes to the exact
// artifact, pinned detector catalog and explicit machine-local selection.
func freezeOpenRGBExecPolicy(policy hostreqkit.OpenRGBLinuxPolicy, artifactSHA string, catalog, config []byte) (openRGBExecPolicy, error) {
	id, err := policy.ID()
	if err != nil {
		return openRGBExecPolicy{}, err
	}
	catalogSHA := fmt.Sprintf("%x", sha256.Sum256(catalog))
	if artifactSHA != policy.ArtifactSHA256 || catalogSHA != policy.DetectorSHA256 {
		return openRGBExecPolicy{}, errors.New("OpenRGB device policy does not match the frozen artifact and detector catalog")
	}
	catalogMap, err := decodeOpenRGBDetectorCatalog(catalog)
	if err != nil {
		return openRGBExecPolicy{}, err
	}
	selected, err := selectedOpenRGBDetectors(policy, catalogMap)
	if err != nil {
		return openRGBExecPolicy{}, err
	}
	if err := verifyOpenRGBExecDetectors(config, catalogMap, selected); err != nil {
		return openRGBExecPolicy{}, err
	}
	return openRGBExecPolicy{
		ID: id, ArtifactSHA256: artifactSHA, DetectorCatalogSHA256: catalogSHA,
		ConfigSHA256: fmt.Sprintf("%x", sha256.Sum256(config)),
	}, nil
}

func verifyOpenRGBExecDetectors(config []byte, catalog map[string]bool, selected []string) error {
	enabledDetectors, err := readOpenRGBDetectorState(config)
	if err != nil {
		return err
	}
	if len(enabledDetectors) != len(catalog) {
		return errors.New("OpenRGB frozen configuration must include the full pinned detector catalog")
	}
	for name := range catalog {
		enabled, ok := enabledDetectors[name]
		if !ok || enabled != slices.Contains(selected, name) {
			return fmt.Errorf("OpenRGB frozen detector %q differs from the physical selection", name)
		}
	}
	return nil
}

func readOpenRGBDetectorState(config []byte) (map[string]bool, error) {
	if len(config) == 0 || len(config) > managedUpdateConfigMaxBytes {
		return nil, errors.New("OpenRGB detector configuration is absent or oversized")
	}
	if err := validateManagedUpdateJSON(config); err != nil {
		return nil, fmt.Errorf("invalid OpenRGB detector configuration: %w", err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(config, &root); err != nil {
		return nil, fmt.Errorf("invalid OpenRGB detector settings: %w", err)
	}
	if err := verifyOpenRGBLocalSettings(root); err != nil {
		return nil, err
	}
	var detectors map[string]json.RawMessage
	var values map[string]*bool
	if json.Unmarshal(root["Detectors"], &detectors) != nil || json.Unmarshal(detectors["detectors"], &values) != nil || values == nil {
		return nil, errors.New("OpenRGB configuration has no exact static detector map")
	}
	for _, section := range []string{"QMKOpenRGBDevices", "QMKVialRGBDevices"} {
		var qmk map[string]json.RawMessage
		var devices []json.RawMessage
		if json.Unmarshal(root[section], &qmk) != nil || json.Unmarshal(qmk["devices"], &devices) != nil || devices == nil || len(devices) != 0 {
			return nil, errors.New("OpenRGB configuration must explicitly disable dynamic HID registrations")
		}
	}
	enabled := make(map[string]bool, len(values))
	for name, value := range values {
		if value == nil {
			return nil, fmt.Errorf("OpenRGB detector %q does not have an explicit enabled state", name)
		}
		enabled[name] = *value
	}
	return enabled, nil
}

func verifyOpenRGBExecInvocation(invocation managedUpdateInvocation, port int) error {
	configDir := filepath.Join(invocation.RuntimeRoot, "config")
	openRGBConfig := filepath.Join(configDir, "OpenRGB")
	args := invocation.Args
	if len(args) != 7 {
		return errors.New("OpenRGB invocation requires the isolated loopback server arguments")
	}
	address, err := netip.ParseAddr(args[2])
	if err != nil || !address.IsLoopback() || !slices.Equal(args, []string{"--server", "--server-host", args[2], "--server-port", strconv.Itoa(port), "--config", openRGBConfig}) || invocation.ConfigPath != filepath.Join(openRGBConfig, "OpenRGB.json") {
		return errors.New("OpenRGB invocation does not use the admitted loopback listener and private settings")
	}
	env := managedServiceEnvValues(invocation.Env)
	if env["HOME"] != filepath.Join(invocation.RuntimeRoot, "home") || env["XDG_CONFIG_HOME"] != configDir || env["QT_QPA_PLATFORM"] != "offscreen" || (runtime.GOOS == "windows" && env["APPDATA"] != configDir) {
		return errors.New("OpenRGB invocation does not use the admitted private environment")
	}
	for name := range env {
		switch name {
		case "HOME", "ROOT", "PATH", "RESOURCE_ROOT", "RESOURCE_CONFIG_DIR", "RESOURCE_CACHE_DIR", "RESOURCE_LOGS_DIR", "RESOURCE_STATE_DIR", "RESOURCE_ARTIFACT_DIR", "VROOLI_MANAGED_SERVICE_ARTIFACT", managedServiceOwnershipTokenEnv, "RESOURCE_PORT_SDK", "QT_QPA_PLATFORM", "XDG_CONFIG_HOME", "APPDATA":
		default:
			return fmt.Errorf("unexpected environment field %q in OpenRGB invocation", name)
		}
	}
	return nil
}

func observeOpenRGBExecWitness(ctx context.Context, policy hostreqkit.OpenRGBLinuxPolicy, contract ManagedUpdateContract, invocation managedUpdateInvocation, catalog, config []byte, f managedUpdateExecFence, ref platform.ServiceRef, incarnation string, uid int, observeContext func(platform.ServiceRef, string) (string, error), observeAccess func(context.Context, hostreqkit.OpenRGBLinuxPolicy, hostreqkit.OpenRGBLinuxCheck, func(int) (string, error)) (hostreqkit.OpenRGBLinuxEvidence, error)) (openRGBExecWitness, error) {
	return observeOpenRGBAdmittedWitness(ctx, policy, contract, invocation, catalog, config, openRGBWitnessAdmission{
		Policy: f.OpenRGBPolicy, Port: f.Intent.Runtime.Port, ContractDigest: f.Intent.ContractDigest, ClaimSHA256: f.openRGBClaimDigest(ref),
	}, ref, incarnation, uid, observeContext, observeAccess)
}

type openRGBWitnessAdmission struct {
	Policy                      *openRGBExecPolicy
	Port                        int
	ContractDigest, ClaimSHA256 string
}

func observeOpenRGBAdmittedWitness(ctx context.Context, policy hostreqkit.OpenRGBLinuxPolicy, contract ManagedUpdateContract, invocation managedUpdateInvocation, catalog, config []byte, admission openRGBWitnessAdmission, ref platform.ServiceRef, incarnation string, uid int, observeContext func(platform.ServiceRef, string) (string, error), observeAccess func(context.Context, hostreqkit.OpenRGBLinuxPolicy, hostreqkit.OpenRGBLinuxCheck, func(int) (string, error)) (hostreqkit.OpenRGBLinuxEvidence, error)) (openRGBExecWitness, error) {
	if err := verifyOpenRGBExecInvocation(invocation, admission.Port); err != nil {
		return openRGBExecWitness{}, err
	}
	binding, err := freezeOpenRGBExecPolicy(policy, contract.Artifact.SHA256, catalog, config)
	if err != nil {
		return openRGBExecWitness{}, err
	}
	if admission.Policy == nil || binding != *admission.Policy || binding.ConfigSHA256 != contract.ConfigSHA256 || string(config) != invocation.ConfigContent || admission.ContractDigest != contract.Digest {
		return openRGBExecWitness{}, errors.New("OpenRGB frozen settings and policy differ from the admitted launch")
	}
	before, err := observeContext(ref, incarnation)
	if err != nil {
		return openRGBExecWitness{}, err
	}
	evidence, err := observeAccess(ctx, policy, hostreqkit.OpenRGBLinuxCheck{PolicyID: binding.ID, ArtifactSHA256: binding.ArtifactSHA256, DetectorSHA256: binding.DetectorCatalogSHA256, EffectiveUID: uid}, nil)
	if err != nil {
		return openRGBExecWitness{}, fmt.Errorf("verify OpenRGB access in the native launch context: %w", err)
	}
	if !evidence.StaticGrantConfigured || !evidence.CurrentScopeVerified || len(evidence.SelectedNodes) != len(policy.Devices) || !updateDigest(evidence.DeviceGenerationSHA256) {
		return openRGBExecWitness{}, errors.New("OpenRGB launch context did not verify the full selected device policy")
	}
	after, err := observeContext(ref, incarnation)
	if err != nil || before != after {
		return openRGBExecWitness{}, errors.Join(errors.New("OpenRGB launch context changed during device observation"), err)
	}
	return openRGBExecWitness{DeviceGenerationSHA256: evidence.DeviceGenerationSHA256, Policy: binding, Incarnation: incarnation, ClaimSHA256: admission.ClaimSHA256, ContextSHA256: before, SelectedNodes: slices.Clone(evidence.SelectedNodes)}, nil
}
