package resources

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

const openRGBInitialSchemaVersion = 1
const openRGBInitialExecCommand = "managed-service-exec"

type openRGBInitialState string

const (
	openRGBInitialPrepared  openRGBInitialState = managedUpdateExecFencePrepared
	openRGBInitialSubmitted openRGBInitialState = "submitted"
	openRGBInitialServing   openRGBInitialState = "serving"
	openRGBInitialQuiescent openRGBInitialState = "quiescent"
	openRGBInitialStopped   openRGBInitialState = "stopped"
)

// Initial launch admission is separate from the managed-update ledger. Each
// attempt retains its own fence even after a newer attempt becomes current.
type openRGBInitialAttempt struct {
	SchemaVersion      int                             `json:"schema_version"`
	ID                 string                          `json:"id"`
	State              openRGBInitialState             `json:"state"`
	Contract           ManagedUpdateContract           `json:"contract"`
	Runtime            ManagedUpdateRuntime            `json:"runtime"`
	Intent             platform.ServiceIntent          `json:"intent"`
	TokenHash          string                          `json:"token_hash"`
	DevicePolicy       *hostreqkit.OpenRGBLinuxPolicy  `json:"device_policy,omitempty"`
	NativeDevicePolicy *hostreqkit.OpenRGBNativePolicy `json:"native_device_policy,omitempty"`
	Catalog            []byte                          `json:"detector_catalog,omitempty"`
	Policy             openRGBExecPolicy               `json:"policy"`

	NativeRef   *platform.ServiceRef `json:"native_ref,omitempty"`
	QuiescentAt *time.Time           `json:"quiescent_at,omitempty"`
	StoppedAt   *time.Time           `json:"stopped_at,omitempty"`
}

// ManagedInitialServingLaunch identifies the canonical native initial launch.
// The exact contract and witness remain in its owner-derived attempt record.
type ManagedInitialServingLaunch struct {
	AttemptID string              `json:"attempt_id"`
	NativeRef platform.ServiceRef `json:"native_ref"`
}

func (a openRGBInitialAttempt) root() string {
	if a.Runtime.NativeAccount != "" {
		return filepath.Join(filepath.Dir(a.Runtime.StateDir), "runtime", "initial", a.ID)
	}
	return filepath.Join(a.Runtime.StateDir, "native-start", a.ID, "runtime")
}
func (a openRGBInitialAttempt) tokenPath() string {
	if a.Runtime.NativeAccount != "" {
		return filepath.Join(a.Runtime.StateDir, "native-start", a.ID, "ownership-token")
	}
	return filepath.Join(a.root(), "ownership-token")
}
func (a openRGBInitialAttempt) wrapperSpec() platform.OwnedServiceSpec {
	if a.Runtime.NativeAccount != "" {
		return a.Runtime.nativeWrapperSpec(a.ID)
	}
	return platform.OwnedServiceSpec{Path: a.Runtime.Launcher, Dir: a.Runtime.Root, NativeAccount: a.Runtime.NativeAccount, Args: []string{openRGBInitialExecCommand, openRGBResourceName, a.ID}, Env: []string{"HOME=" + a.Runtime.Home, "PATH=/usr/local/bin:/usr/bin:/bin", "XDG_STATE_HOME=" + a.Runtime.StateHome}, Limits: a.Runtime.Limits}
}
func (a openRGBInitialAttempt) fence() managedUpdateExecFence {
	return managedUpdateExecFence{AdmissionKind: managedExecAdmissionInitial, SchemaVersion: managedUpdateExecFenceSchemaVersion, Resource: openRGBResourceName, OperationID: a.ID, Intent: ManagedUpdateLaunchIntent{InstanceID: a.ID, OwnershipTokenHash: a.TokenHash, ContractDigest: a.Contract.Digest, Runtime: a.Runtime, NativeIntent: a.Intent}, State: managedUpdateExecFencePrepared, OpenRGBPolicy: &a.Policy}
}
func (a openRGBInitialAttempt) claimDigest(ref platform.ServiceRef) string {
	return a.fence().openRGBClaimDigest(ref)
}

func (a openRGBInitialAttempt) validate() error {
	if a.NativeDevicePolicy == nil && (a.Policy.Platform != "" || a.Policy.Account != "") {
		return errors.New("initial native HID admission has no retained native selection")
	}
	if a.NativeDevicePolicy != nil {
		if a.DevicePolicy != nil || len(a.Catalog) == 0 || len(a.Catalog) > managedUpdateConfigMaxBytes {
			return errors.New("initial native HID policy/catalog is incomplete or ambiguous")
		}
		p := a.NativeDevicePolicy
		id, err := p.ID()
		if err != nil || id != a.Policy.ID || p.ArtifactSHA256 != a.Policy.ArtifactSHA256 || p.DetectorSHA256 != a.Policy.DetectorCatalogSHA256 || p.Platform != a.Policy.Platform || p.Account != a.Policy.Account || fmt.Sprintf("%x", sha256.Sum256(a.Catalog)) != p.DetectorSHA256 {
			return errors.Join(errors.New("initial native HID policy differs from admission"), err)
		}
	} else if a.DevicePolicy != nil || len(a.Catalog) != 0 {
		if a.DevicePolicy == nil || len(a.Catalog) == 0 || len(a.Catalog) > managedUpdateConfigMaxBytes {
			return errors.New("initial OpenRGB retained policy/catalog is incomplete")
		}
		id, err := a.DevicePolicy.ID()
		if err != nil || id != a.Policy.ID || a.DevicePolicy.ArtifactSHA256 != a.Policy.ArtifactSHA256 || a.DevicePolicy.DetectorSHA256 != a.Policy.DetectorCatalogSHA256 || fmt.Sprintf("%x", sha256.Sum256(a.Catalog)) != a.Policy.DetectorCatalogSHA256 {
			return errors.Join(errors.New("initial OpenRGB retained policy differs from native admission"), err)
		}
	}

	if a.SchemaVersion != openRGBInitialSchemaVersion || !managedUpdateKey.MatchString(a.ID) || a.Contract.Manifest.Name != openRGBResourceName || a.Contract.validate() != nil || a.Runtime.validate() != nil || a.Intent.Owner != "resource-initial" || a.Intent.AttemptID != a.ID || a.Intent.ValidateSpec(a.wrapperSpec()) != nil || !updateDigest(a.TokenHash) || !a.Policy.valid() || a.Policy.ArtifactSHA256 != a.Contract.Artifact.SHA256 || a.Policy.ConfigSHA256 != a.Contract.ConfigSHA256 {
		return errors.New("invalid initial OpenRGB native admission")
	}
	switch a.State {
	case openRGBInitialPrepared, openRGBInitialSubmitted, openRGBInitialServing, openRGBInitialQuiescent, openRGBInitialStopped:
	default:
		return errors.New("unknown initial OpenRGB admission state")
	}
	if a.NativeRef != nil && (a.NativeRef.ValidateIdentity() != nil || a.NativeRef.Intent != platform.OperationIntent(a.Intent)) {
		return errors.New("initial OpenRGB cleanup native identity changed")
	}
	if a.QuiescentAt != nil && (a.QuiescentAt.IsZero() || a.NativeRef == nil || a.State != openRGBInitialQuiescent && a.State != openRGBInitialStopped) {
		return errors.New("invalid initial OpenRGB quiescence evidence")
	}
	if a.State == openRGBInitialQuiescent && a.QuiescentAt == nil {
		return errors.New("initial OpenRGB cleanup has no quiescence evidence")
	}
	if (a.State == openRGBInitialStopped) != (a.StoppedAt != nil) {
		return errors.New("initial OpenRGB stop evidence is incomplete")
	}
	return nil
}

type openRGBInitialStore struct {
	dir   string
	write func(string, []byte, os.FileMode) error
}

func newOpenRGBInitialStore() (*openRGBInitialStore, error) {
	paths, err := resourceStoragePaths(openRGBResourceName)
	if err != nil {
		return nil, err
	}
	return &openRGBInitialStore{dir: paths.StateDir, write: writeResourceOwnedFileDurable}, nil
}
func (s *openRGBInitialStore) attemptPath(id string) string {
	return filepath.Join(s.dir, "native-start", id, "admission.json")
}
func (s *openRGBInitialStore) currentPath() string {
	return filepath.Join(s.dir, "native-start", "current.json")
}
func (s *openRGBInitialStore) lock(ctx context.Context) (func(), error) {
	if nativeOpenRGBUsesPrivateLocks(openRGBResourceName) {
		return acquireOpenRGBNativeOwnerLock(ctx, "resource-initial-state")
	}
	return artifactlock.AcquireWithPathContext(ctx, "openrgb-initial", func(string) (string, error) { return filepath.Join(s.dir, "native-start", ".admission.lock"), nil })
}
func readOpenRGBInitialJSON(path string, output any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > 2<<20 {
		return errors.New("invalid initial OpenRGB admission file")
	}
	data, err := io.ReadAll(io.LimitReader(f, (2<<20)+1))
	if err != nil {
		return err
	}
	if len(data) > 2<<20 {
		return errors.New("initial OpenRGB admission exceeds size bound")
	}
	if err := validateManagedUpdateJSON(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("initial OpenRGB admission has trailing data")
	}
	return nil
}
func (s *openRGBInitialStore) current() (string, error) {
	var id string
	err := readOpenRGBInitialJSON(s.currentPath(), &id)
	if err == nil && !managedUpdateKey.MatchString(id) {
		err = errors.New("invalid current initial OpenRGB attempt")
	}
	return id, err
}
func (s *openRGBInitialStore) read(id string) (openRGBInitialAttempt, error) {
	var a openRGBInitialAttempt
	if !managedUpdateKey.MatchString(id) {
		return a, errors.New("invalid initial OpenRGB attempt selector")
	}
	if err := readOpenRGBInitialJSON(s.attemptPath(id), &a); err != nil {
		return a, err
	}
	if a.ID != id || a.Runtime.StateDir != s.dir {
		return a, errors.New("initial OpenRGB admission moved outside its owner")
	}
	return a, a.validate()
}
func (s *openRGBInitialStore) save(a openRGBInitialAttempt) error {
	if err := a.validate(); err != nil {
		return err
	}
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}
	if len(data) > 2<<20 {
		return errors.New("initial OpenRGB admission exceeds size bound")
	}
	if err := s.write(s.attemptPath(a.ID), data, 0600); err != nil {
		return fmt.Errorf("initial OpenRGB durable admission outcome needs reconciliation: %w", err)
	}
	return nil
}
func (s *openRGBInitialStore) prepare(ctx context.Context, a openRGBInitialAttempt) error {
	unlock, err := s.lock(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	if id, err := s.current(); err == nil {
		old, err := s.read(id)
		if err != nil {
			return err
		}
		if old.State != openRGBInitialStopped {
			return errors.New("initial OpenRGB attempt requires reconciliation")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if a.State != openRGBInitialPrepared {
		return errors.New("initial OpenRGB attempt is already used")
	}
	if _, err := os.Lstat(s.attemptPath(a.ID)); err == nil {
		return errors.New("initial OpenRGB attempt identity is already retained")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := s.save(a); err != nil {
		return err
	}
	data, _ := json.Marshal(a.ID)
	return s.write(s.currentPath(), data, 0600)
}
func (s *openRGBInitialStore) change(ctx context.Context, expected openRGBInitialAttempt, change func(*openRGBInitialAttempt) error) (openRGBInitialAttempt, error) {
	unlock, err := s.lock(ctx)
	if err != nil {
		return expected, err
	}
	defer unlock()
	current, err := s.read(expected.ID)
	if err != nil {
		return expected, err
	}
	if !reflect.DeepEqual(current, expected) {
		return current, errors.New("initial OpenRGB admission changed")
	}
	if err := change(&current); err != nil {
		return current, err
	}
	return current, s.save(current)
}
func (s *openRGBInitialStore) submit(ctx context.Context, a openRGBInitialAttempt) (openRGBInitialAttempt, error) {
	return s.change(ctx, a, func(a *openRGBInitialAttempt) error {
		if a.State != openRGBInitialPrepared {
			return errors.New("initial OpenRGB dispatch already admitted")
		}
		id, err := s.current()
		if err != nil || id != a.ID {
			return errors.New("initial OpenRGB attempt is not current")
		}
		a.State = openRGBInitialSubmitted
		return nil
	})
}

// compareJSON is used only for exact retained immutable values in tests and
// publication checks; pointers and maps compare through their canonical JSON.
func sameOpenRGBInitialState(a, b ManagedServiceState) bool {
	x, e := json.Marshal(a)
	y, f := json.Marshal(b)
	return e == nil && f == nil && bytes.Equal(x, y)
}
