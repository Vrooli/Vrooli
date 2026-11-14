package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// A capsule contains only nonsecret frozen metadata. The owning resource
// process retains its ledger and raw token. The capsule must be published in
// the private canonical launch directory. Vendor confinement excludes owner state.
type openRGBNativeCapsule struct {
	SchemaVersion    int                    `json:"version"`
	ID               string                 `json:"id"`
	Intent           platform.ServiceIntent `json:"intent"`
	Account          string                 `json:"account"`
	LauncherSHA256   string                 `json:"launcher_sha256"`
	RuntimeRoot      string                 `json:"runtime_root"`
	MaterialSHA256   string                 `json:"material_sha256"`
	Endpoint         string                 `json:"endpoint"`
	OwnerPID         int                    `json:"owner_pid"`
	OwnerIncarnation string                 `json:"owner_incarnation"`
	OwnerPrincipal   string                 `json:"owner_principal"`
}

type openRGBNativeExecutionChild struct {
	PID           int    `json:"pid"`
	Incarnation   string `json:"incarnation"`
	ImageID       string `json:"image_id"`
	ImageHandle   uint64 `json:"image_handle"`
	SecurityProof string `json:"security_proof"`
	NativeAccount string `json:"native_account"`
	ContainmentID string `json:"containment_id"`
}

// Material comes exclusively from the authenticated owner after exact native
// peer observation. Paths and invocation are already frozen; the protocol has
// no caller-selected executable, credentials, policy, or shell command.
type openRGBNativeMaterial struct {
	Contract     ManagedUpdateContract          `json:"contract"`
	Runtime      ManagedUpdateRuntime           `json:"runtime"`
	Invocation   managedUpdateInvocation        `json:"invocation"`
	StagePath    string                         `json:"stage_path"`
	NativePolicy hostreqkit.OpenRGBNativePolicy `json:"native_policy"`
	Catalog      []byte                         `json:"catalog"`
	Policy       openRGBExecPolicy              `json:"policy"`
	ClaimSHA256  string                         `json:"claim_sha256"`
	TokenHash    string                         `json:"token_hash"`
}

type openRGBNativeClaimCallbacks struct {
	Lookup         func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error)
	Inspect        func(context.Context, platform.ServiceRef) (platform.OperationObservation, error)
	Incarnation    func(int) (string, error)
	ResolveAccount func(context.Context, string) (platform.OwnedServiceAccount, error)
	Prepare        func(context.Context, platform.ServiceRef) (openRGBNativeMaterial, error)
	Claim          func(context.Context, platform.ServiceRef, openRGBExecWitness) error
}

func (c openRGBNativeCapsule) validate() error {
	if c.SchemaVersion != openRGBNativeCapsuleVersion || !managedUpdateKey.MatchString(c.ID) || c.ID != c.Intent.AttemptID || c.Intent.ValidateIdentity() != nil || c.Intent.NativeAccount != c.Account || c.Account == "" || !updateDigest(c.LauncherSHA256) || !updateDigest(c.MaterialSHA256) || c.OwnerPID <= 0 || c.OwnerIncarnation == "" || c.OwnerPrincipal == "" {
		return errors.New("invalid native OpenRGB capsule identity")
	}
	for _, p := range []string{c.Endpoint, c.RuntimeRoot} {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p || strings.ContainsAny(p, "\x00\r\n") {
			return errors.New("native capsule requires canonical absolute paths")
		}
	}
	if len(c.Endpoint) > 100 || len(c.OwnerIncarnation) > 256 || len(c.OwnerPrincipal) > 256 || strings.ContainsAny(c.OwnerPrincipal+c.OwnerIncarnation, "\x00\r\n") {
		return errors.New("native capsule identity exceeds bounds")
	}
	return nil
}
func (m openRGBNativeMaterial) digest() string {
	// The claim binds the native root and is unavailable before dispatch. It is
	// authenticated after lookup and independently recomputed by durable Claim.
	m.ClaimSHA256 = ""
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", sha256.Sum256(b))
}
func (m openRGBNativeMaterial) validate(c openRGBNativeCapsule) error {
	if err := c.validate(); err != nil {
		return err
	}
	if m.digest() != c.MaterialSHA256 || m.Contract.validate() != nil || m.Runtime.validate() != nil || m.Runtime.NativeAccount != c.Account || m.Runtime.LauncherSHA256 != c.LauncherSHA256 || !m.Policy.valid() || !updateDigest(m.ClaimSHA256) || !updateDigest(m.TokenHash) {
		return errors.New("native material differs from frozen capsule")
	}
	frozenRoot, rootErr := openRGBNativeResourceRootFromRuntime(c.RuntimeRoot)
	if rootErr != nil || m.Runtime.StateDir != filepath.Join(frozenRoot, "state") {
		return errors.Join(errors.New("native material owner state differs from capsule runtime namespace"), rootErr)
	}
	if m.Invocation.RuntimeRoot != c.RuntimeRoot || !filepath.IsAbs(m.StagePath) || filepath.Clean(m.StagePath) != m.StagePath || nativeCapsulePathWithin(m.Runtime.Home, m.StagePath) || nativeCapsulePathWithin(m.Runtime.Home, c.RuntimeRoot) {
		return errors.New("native launch paths must use machine-owned code and the canonical resource runtime")
	}
	if m.Policy.Platform != m.NativePolicy.Platform || m.Policy.Account != m.NativePolicy.Account || m.Policy.ArtifactSHA256 != m.Contract.Artifact.SHA256 || m.Policy.ConfigSHA256 != m.Contract.ConfigSHA256 || m.Invocation.ConfigContent != string(m.Contract.ConfigBytes) {
		return errors.New("native material policy/configuration binding changed")
	}
	policyID, err := m.NativePolicy.ID()
	if err != nil || policyID != m.Policy.ID {
		return errors.New("native material policy changed")
	}
	values, err := validateOpenRGBNativeCapsuleEnvironment(m.Invocation.Env, m.Policy.Platform)
	if err != nil {
		return err
	}
	token := values[managedServiceOwnershipTokenEnv]
	if token == "" || managedServiceTokenHash(token) != m.TokenHash {
		return errors.New("native material ownership token changed")
	}
	if err := verifyOpenRGBExecInvocation(m.Invocation, m.Runtime.Port); err != nil {
		return err
	}
	return nil
}
func nativeCapsulePathWithin(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func (c openRGBNativeExecutionChild) validateFor(ref platform.ServiceRef) error {
	if ref.ValidateIdentity() != nil || ref.Native == nil || ref.Native.Platform != "windows" || c.PID <= 0 || c.PID == ref.RootPID || c.ImageHandle == 0 || !updateDigest(c.SecurityProof) || c.ContainmentID != ref.Native.ContainmentID || c.NativeAccount != ref.Native.Account || !strings.HasPrefix(c.Incarnation, fmt.Sprintf("windows:%d:", c.PID)) || len(c.Incarnation) > 128 || len(c.ImageID) != 38 || !strings.HasPrefix(c.ImageID, "windows-file:") {
		return errors.New("native execution child differs from exact owned service")
	}
	suffix := strings.TrimPrefix(c.Incarnation, fmt.Sprintf("windows:%d:", c.PID))
	if value, err := strconv.ParseUint(suffix, 10, 64); err != nil || value == 0 {
		return errors.New("invalid native execution child incarnation")
	}
	for _, ch := range strings.TrimPrefix(c.ImageID, "windows-file:") {
		if ch != ':' && (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return errors.New("invalid native execution image identity")
		}
	}
	return nil
}

func validateOpenRGBNativeCapsuleEnvironment(env []string, target string) (map[string]string, error) {
	if env == nil || len(env) > 1024 {
		return nil, errors.New("native capsule environment must be explicit and bounded")
	}
	values := make(map[string]string, len(env))
	keys := make([]string, 0, len(env))
	size := 0
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		size += len(entry)
		if !ok || key == "" || strings.ContainsRune(entry, 0) || size > 64<<10 {
			return nil, errors.New("malformed native capsule environment")
		}
		for _, prior := range keys {
			if prior == key || target == "windows" && strings.EqualFold(prior, key) {
				return nil, errors.New("ambiguous native capsule environment")
			}
		}
		keys = append(keys, key)
		values[key] = value
	}
	return values, nil
}
