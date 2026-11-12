// Package lightingruntime owns opt-in headless lighting lifecycle intent.
package lightingruntime

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/config"
)

const (
	lightingStateVersion = 1
	lightingStateDir     = "host-lighting"
	lightingStateFile    = "intent.json"
	lightingLockFile     = "operation.lock"
	maxLightingStateSize = 4096
	quiesceTimeout       = 5 * time.Second
)

type Phase string

const (
	Disabled  Phase = "disabled"
	Enabled   Phase = "enabled"
	Disabling Phase = "disabling"
)

// Selection freezes the artifact and machine-local device permission
// policy qualified by the resource owner during explicit setup.
type Selection struct {
	Version        string `json:"version"`
	SHA256         string `json:"sha256"`
	DevicePolicyID string `json:"device_policy_id"`
}

// Intent describes the operator's durable choice. It makes no claim
// that either process is currently running or that the lights were applied.
type Intent struct {
	SchemaVersion int          `json:"schema_version"`
	Phase         Phase        `json:"phase"`
	Selection     *Selection   `json:"selection,omitempty"`
	Preparation   *Preparation `json:"preparation,omitempty"`
}

// Preparation retains the last explicit installation/static-qualification result.
// It never claims that runtime access or the installed bytes remain observed.
type Preparation struct {
	Source        string    `json:"source"`
	Selection     Selection `json:"selection"`
	Qualified     bool      `json:"qualified"`
	LaunchBlocker string    `json:"launch_blocker,omitempty"`
}

const (
	PreparationManaged  = "managed"
	PreparationExisting = "existing"
)

// Backend is returned only for a ready, verified resource-owned
// process. Incarnation changes when that exact backend process is replaced.
type Backend struct {
	Endpoint    netip.AddrPort
	Incarnation string
}

type ResourceOwner interface {
	// VerifyLightingSetup verifies staged bytes, resource enablement, platform
	// support and the selected machine/account enrollment without installing or
	// launching. Native daemon HID access is proved in its owned launch context;
	// prospective setup admission does not claim that access was observed.
	VerifyLightingSetup(context.Context, Selection) error
	// EnsureOpenRGB starts or adopts the exact ready, selected backend. It
	// checks resource enablement, the selection and the device permission policy
	// at the launch boundary, never acquires bytes, and honors the managed-service
	// update fence.
	EnsureOpenRGB(context.Context, Selection) (Backend, error)
	// DisableOpenRGB confirms the owned backend stopped or absent and clears
	// its ordinary resource enablement. It preserves the installed artifact
	// and source selection. Repeating the same disable is safe.
	DisableOpenRGB(context.Context) error
}

type ScenarioOwner interface {
	// EnsureLightingReconciler connects the scenario to the owned endpoint and
	// delivers a changed incarnation as a reapply trigger. It is idempotent
	// and restores a quiesced binding even when the incarnation is unchanged.
	EnsureLightingReconciler(context.Context, Backend) error
	// QuiesceLightingReconciler waits until the scenario cannot write lighting
	// state through its prior binding. It succeeds if already quiesced.
	QuiesceLightingReconciler(context.Context) error
}

type Owners struct {
	Resource ResourceOwner
	Scenario ScenarioOwner
}

type Controller struct {
	path   string
	home   string
	owners Owners
}

func NewController(home string, owners Owners) (*Controller, error) {
	if !filepath.IsAbs(home) || filepath.Clean(home) != home {
		return nil, fmt.Errorf("headless lighting requires a canonical absolute operator home")
	}
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return nil, err
	}
	return &Controller{path: filepath.Join(state, lightingStateDir, lightingStateFile), home: home, owners: owners}, nil
}

// Enable admits the exact selection for the next native reconciliation. If
// the current selection is already admitted, it can be retried unchanged.
func (c *Controller) Enable(ctx context.Context, selection Selection) error {
	return c.EnablePrepared(ctx, selection, nil)
}

// EnablePrepared serializes resource enablement with the durable intent fence.
func (c *Controller) EnablePrepared(ctx context.Context, selection Selection, prepare func() error) error {
	if err := selection.validate(); err != nil {
		return err
	}
	if err := c.requireOwners(); err != nil {
		return err
	}
	release, err := c.lock(ctx)
	if err != nil {
		return err
	}
	defer release()
	current, err := c.Current()
	if err != nil {
		return err
	}
	if current.Phase == Disabling {
		return fmt.Errorf("headless lighting disable is pending; finish it before enabling")
	}
	if err := c.owners.Resource.VerifyLightingSetup(ctx, selection); err != nil {
		return fmt.Errorf("verify headless lighting setup: %w", err)
	}
	if prepare != nil {
		if err := prepare(); err != nil {
			return err
		}
	}
	return c.write(Intent{SchemaVersion: lightingStateVersion, Phase: Enabled, Selection: &selection, Preparation: current.Preparation})
}

// Disable durably fences starts before touching either process. An explicit
// retry also retires backend records left after lighting became disabled.
// Passive reconciliation of disabled intent continues to have no effects.
func (c *Controller) Disable(ctx context.Context) error {
	release, err := c.lock(ctx)
	if err != nil {
		return err
	}
	defer release()
	current, err := c.Current()
	if err != nil {
		return err
	}
	current.Phase = Disabling
	return c.finishDisable(ctx, current)
}

// Reconcile is safe to run repeatedly and after a new native supervisor owns
// the runtime. A fresh or disabled host makes no owner calls or state writes.
func (c *Controller) Reconcile(ctx context.Context) error {
	current, err := c.Current()
	if err != nil || current.Phase == Disabled {
		return err
	}
	release, err := c.lock(ctx)
	if err != nil {
		return err
	}
	defer release()
	current, err = c.Current()
	if err != nil || current.Phase == Disabled {
		return err
	}
	if current.Phase == Disabling {
		return c.finishDisable(ctx, current)
	}
	if err := c.requireOwners(); err != nil {
		return err
	}
	backend, err := c.owners.Resource.EnsureOpenRGB(ctx, *current.Selection)
	if err != nil {
		return c.quiesceAfterFailure(ctx, fmt.Errorf("ensure staged OpenRGB backend: %w", err))
	}
	if !backend.Endpoint.IsValid() || !backend.Endpoint.Addr().Unmap().IsLoopback() || backend.Endpoint.Addr().Zone() != "" || backend.Endpoint.Port() == 0 || strings.TrimSpace(backend.Incarnation) == "" {
		return c.quiesceAfterFailure(ctx, errors.New("OpenRGB owner returned no verified loopback process binding"))
	}
	if err := c.owners.Scenario.EnsureLightingReconciler(ctx, backend); err != nil {
		return c.quiesceAfterFailure(ctx, fmt.Errorf("ensure lighting reconciler: %w", err))
	}
	return nil
}

func (c *Controller) quiesceAfterFailure(ctx context.Context, cause error) error {
	// Losing a check to cancellation still leaves the old writer live. Give
	// quiescing a bounded cleanup context while the operation lock is held.
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), quiesceTimeout)
	defer cancel()
	if err := c.owners.Scenario.QuiesceLightingReconciler(cleanup); err != nil {
		return errors.Join(cause, fmt.Errorf("quiesce lighting reconciler after failed binding: %w", err))
	}
	return cause
}

func (c *Controller) finishDisable(ctx context.Context, intent Intent) error {
	// A failed publication can leave a visible rename with unconfirmed
	// directory durability. Confirm the same fence before effects on retries.
	if err := c.write(intent); err != nil {
		return err
	}
	if err := c.requireOwners(); err != nil {
		return err
	}
	if err := c.owners.Scenario.QuiesceLightingReconciler(ctx); err != nil {
		return fmt.Errorf("quiesce lighting reconciler: %w", err)
	}
	if err := c.owners.Resource.DisableOpenRGB(ctx); err != nil {
		return fmt.Errorf("disable OpenRGB backend: %w", err)
	}
	return c.write(Intent{SchemaVersion: lightingStateVersion, Phase: Disabled, Preparation: intent.Preparation})
}

func (c *Controller) requireOwners() error {
	if c.owners.Resource == nil || c.owners.Scenario == nil {
		return fmt.Errorf("headless lighting requires both the OpenRGB resource and lighting scenario owners")
	}
	return nil
}

func (c *Controller) lock(ctx context.Context) (func(), error) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		policy, err := c.nativeStatePolicy()
		if err != nil {
			return nil, err
		}
		if err = c.prepareNativeStateDirectory(policy); err != nil {
			return nil, err
		}
		return config.AcquirePrivateFileLock(ctx, filepath.Join(filepath.Dir(c.path), lightingLockFile), policy)
	}
	if err := EnsureStateDirectory(filepath.Dir(c.path), true); err != nil {
		return nil, err
	}
	return artifactlock.AcquireWithPathContext(ctx, c.path, func(string) (string, error) {
		return filepath.Join(filepath.Dir(c.path), lightingLockFile), nil
	})
}

func (c *Controller) Current() (Intent, error) {
	info, err := os.Lstat(c.path)
	if errors.Is(err, os.ErrNotExist) {
		return Intent{SchemaVersion: lightingStateVersion, Phase: Disabled}, nil
	}
	if err != nil {
		return Intent{}, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxLightingStateSize {
		return Intent{}, fmt.Errorf("invalid headless lighting intent file")
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		policy, err := c.nativeStatePolicy()
		if err != nil {
			return Intent{}, err
		}
		if err = config.VerifyPrivatePath(c.path, info, policy); err != nil {
			return Intent{}, err
		}
	}
	file, err := os.Open(c.path)
	if err != nil {
		return Intent{}, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return Intent{}, err
	}
	if !os.SameFile(info, opened) {
		return Intent{}, fmt.Errorf("headless lighting intent changed during read")
	}
	decoder := json.NewDecoder(io.LimitReader(file, maxLightingStateSize+1))
	decoder.DisallowUnknownFields()
	var current Intent
	if err := decoder.Decode(&current); err != nil {
		return Intent{}, fmt.Errorf("decode headless lighting intent: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Intent{}, fmt.Errorf("headless lighting intent has trailing data")
	}
	if err := current.validate(); err != nil {
		return Intent{}, err
	}
	return current, nil
}

func (c *Controller) write(intent Intent) error {
	if runtime.GOOS == "linux" {
		if err := EnsureStateDirectory(filepath.Dir(c.path), true); err != nil {
			return err
		}
	}
	data, err := json.Marshal(intent)
	if err != nil {
		return err
	}
	write := config.WriteOwnedFileDurable
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		policy, err := c.nativeStatePolicy()
		if err != nil {
			return err
		}
		write = func(path string, data []byte, mode os.FileMode) error {
			return config.WritePrivateFileDurable(path, data, mode, policy)
		}
	}
	if err := write(c.path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("persist headless lighting intent: %w", err)
	}
	return nil
}

func (i Intent) validate() error {
	if p := i.Preparation; p != nil {
		selection := p.Selection
		if selection.DevicePolicyID == "" && !p.Qualified {
			selection.DevicePolicyID = "unqualified"
		}
		if (p.Source != PreparationManaged && p.Source != PreparationExisting) || len(p.LaunchBlocker) > 1024 || selection.validate() != nil {
			return errors.New("invalid lighting setup preparation")
		}
	}

	if i.SchemaVersion == lightingStateVersion {
		switch i.Phase {
		case Disabled:
			if i.Selection == nil {
				return nil
			}
		case Disabling:
			// Retiring old backend debt does not require a selected artifact.
			// This phase cannot admit any startup and remains retryable.
			if i.Selection == nil {
				return nil
			}
			return i.Selection.validate()
		case Enabled:
			if i.Selection != nil {
				return i.Selection.validate()
			}
		}
	}
	return fmt.Errorf("invalid headless lighting intent version, phase, or selection")
}

func (s Selection) validate() error {
	digest, err := hex.DecodeString(s.SHA256)
	if err != nil || len(digest) != 32 || s.SHA256 != strings.ToLower(s.SHA256) {
		return fmt.Errorf("headless lighting requires a lowercase SHA-256 artifact digest")
	}
	if !validSelectionValue(s.Version) || !validSelectionValue(s.DevicePolicyID) {
		return fmt.Errorf("headless lighting requires an artifact version and a device policy identity")
	}
	return nil
}

func validSelectionValue(value string) bool {
	return value != "" && len(value) <= 128 && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n")
}

// PrepareDisabled serializes explicit installation changes with enable/disable
// and supervisor reconciliation. The callback must not reenter the controller.
func (c *Controller) PrepareDisabled(ctx context.Context, prepare func() error) error {
	release, err := c.lock(ctx)
	if err != nil {
		return err
	}
	defer release()
	current, err := c.Current()
	if err != nil {
		return err
	}
	if current.Phase != Disabled {
		return errors.New("disable lighting before changing its installation")
	}
	return prepare()
}

// PrepareDisabledResult persists the explicit setup result under the existing
// intent fence. Status can project this receipt without running setup again.
func (c *Controller) PrepareDisabledResult(ctx context.Context, prepare func() (*Preparation, error)) error {
	return c.PrepareDisabled(ctx, func() error {
		result, err := prepare()
		if err != nil {
			// A failed explicit setup can reveal missing/tampered bytes or
			// change the selected source. Do not retain an older qualification.
			return errors.Join(err, c.write(Intent{SchemaVersion: lightingStateVersion, Phase: Disabled}))
		}
		return c.write(Intent{SchemaVersion: lightingStateVersion, Phase: Disabled, Preparation: result})
	})
}
