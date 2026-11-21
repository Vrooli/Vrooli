// Responsibility: lease the in-app browser pane to one run and verify its return receipt.
package orchestration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/browsercapability"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

const (
	BrowserPaneControlLeaseTTL  = 5 * time.Minute
	maxBrowserPaneControlLeases = 1024
)

// BrowserPaneControlBinding is the full server-side identity for one human
// control yield. BAS separately binds the stream receipt to its immutable lease.
type BrowserPaneControlBinding struct {
	RunID               uuid.UUID
	OwnerSubject        string
	WebConsoleSessionID string
	ThreadID            string
	BASSessionID        string
	GrantReceipt        string
	ProfileID           string
}

type BrowserPaneControlYield struct {
	Receipt   string
	ExpiresAt time.Time
}

type browserPaneControlLease struct {
	binding                BrowserPaneControlBinding
	tokenHash              [sha256.Size]byte
	token                  string
	awaitKey               string
	expires                time.Time
	closedAt               time.Time
	parkedLifecycleVersion int64
}

func (o *Orchestrator) YieldBrowserPaneControl(ctx context.Context, binding BrowserPaneControlBinding) (BrowserPaneControlYield, error) {
	if o == nil || o.runs == nil {
		return BrowserPaneControlYield{}, errors.New("Agent Manager browser control owner unavailable")
	}
	if err := validateBrowserPaneControlBinding(binding); err != nil {
		return BrowserPaneControlYield{}, err
	}
	o.browserPaneControlMu.Lock()
	defer o.browserPaneControlMu.Unlock()
	o.pruneBrowserPaneControlsLocked()
	if current := o.browserPaneControls[binding.RunID]; current != nil {
		if !sameBrowserPaneControlBinding(current.binding, binding) {
			return BrowserPaneControlYield{}, errors.New("another browser pane controls this run")
		}
		run, err := o.runs.Get(ctx, binding.RunID)
		if err != nil || run == nil {
			return BrowserPaneControlYield{}, errors.New("browser control run is unavailable")
		}
		if current.closedAt.IsZero() && current.expires.After(o.now()) && run.Status == domain.RunStatusParked && run.AwaitHandle != nil && run.AwaitHandle.Producer == ProducerBrowserPaneControl && run.AwaitHandle.Key == current.awaitKey {
			return BrowserPaneControlYield{Receipt: current.token, ExpiresAt: current.expires}, nil
		}
		if current.closedAt.IsZero() {
			return BrowserPaneControlYield{}, errors.New("browser pane control yield expired or changed; wait for Agent Manager to recover the run")
		}
		if run.Status == domain.RunStatusRunning && run.SessionID == binding.ThreadID {
			delete(o.browserPaneControls, binding.RunID)
		} else {
			return BrowserPaneControlYield{}, errors.New("prior browser control yield has not returned to the same active run")
		}
	}
	if len(o.browserPaneControls) >= maxBrowserPaneControlLeases {
		return BrowserPaneControlYield{}, errors.New("browser control lease capacity reached")
	}
	run, err := o.GetRun(ctx, binding.RunID)
	if err != nil || run == nil || run.ID != binding.RunID || run.Status != domain.RunStatusRunning || run.ExecutionMode.Normalized() != domain.ExecutionModeInteractive ||
		run.OwnerSubject != binding.OwnerSubject || run.WebConsoleSessionID != binding.WebConsoleSessionID || run.SessionID != binding.ThreadID || run.ResolvedConfig == nil || run.ResolvedConfig.Features.BrowserCapability == nil {
		return BrowserPaneControlYield{}, errors.New("active interactive run does not match the human browser control binding")
	}
	capability := run.ResolvedConfig.Features.BrowserCapability
	if !capability.Enabled || !capability.Permissions.View || !capability.Permissions.Control ||
		strings.TrimSpace(capability.Permissions.ProfileID) != binding.ProfileID ||
		subtle.ConstantTimeCompare([]byte(capability.GrantReceipt), []byte(binding.GrantReceipt)) != 1 ||
		(run.ApprovalState != "" && run.ApprovalState != domain.ApprovalStateNone) {
		return BrowserPaneControlYield{}, errors.New("persisted browser permission or approval state refuses control")
	}
	if o.awaitRegistry == nil || !o.awaitRegistry.SupportsProducer(ProducerBrowserPaneControl) {
		return BrowserPaneControlYield{}, errors.New("browser pane control recovery waiter unavailable")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return BrowserPaneControlYield{}, fmt.Errorf("create browser pane control receipt: %w", err)
	}
	key := uuid.NewString()
	deadline := o.now().Add(BrowserPaneControlLeaseTTL)
	barrier, err := o.prepareParkedTurnEnd(run)
	if err != nil {
		return BrowserPaneControlYield{}, err
	}
	parked, err := o.parkRun(ctx, ParkRunInput{RunID: binding.RunID, Producer: ProducerBrowserPaneControl, Key: key, Deadline: &deadline}, false)
	if err != nil && (parked == nil || parked.Status != domain.RunStatusParked || parked.AwaitHandle == nil || parked.AwaitHandle.Producer != ProducerBrowserPaneControl || parked.AwaitHandle.Key != key) {
		o.failParkedTurnEnd(binding.RunID, barrier, err)
		return BrowserPaneControlYield{}, err
	}
	if parked == nil || parked.Status != domain.RunStatusParked || parked.AwaitHandle == nil || parked.AwaitHandle.Key != key {
		o.failParkedTurnEnd(binding.RunID, barrier, errors.New("park transition did not persist the browser control handle"))
		return BrowserPaneControlYield{}, errors.New("browser control run yield failed")
	}
	o.endParkedTurn(parked, barrier)
	if o.awaitRegistry != nil {
		o.awaitRegistry.Register(parked.ID, parked.AwaitHandle)
	}
	if err := o.awaitParkedTurnEnd(ctx, binding.RunID); err != nil {
		return BrowserPaneControlYield{}, fmt.Errorf("Agent Manager could not confirm the agent turn stopped before browser control: %w", err)
	}
	hash := sha256.Sum256(secret)
	if o.browserPaneControls == nil {
		o.browserPaneControls = make(map[uuid.UUID]*browserPaneControlLease)
	}
	lease := &browserPaneControlLease{binding: binding, tokenHash: hash, token: base64.RawURLEncoding.EncodeToString(secret), awaitKey: key, expires: deadline, parkedLifecycleVersion: parked.LifecycleVersion}
	o.browserPaneControls[binding.RunID] = lease
	return BrowserPaneControlYield{Receipt: lease.token, ExpiresAt: deadline}, err
}

func (o *Orchestrator) ReturnBrowserPaneControl(ctx context.Context, binding BrowserPaneControlBinding, receipt string) error {
	if o == nil || o.runs == nil || validateBrowserPaneControlBinding(binding) != nil {
		return errors.New("Agent Manager browser control return binding unavailable")
	}
	o.browserPaneControlMu.Lock()
	defer o.browserPaneControlMu.Unlock()
	lease := o.browserPaneControls[binding.RunID]
	if lease == nil || !sameBrowserPaneControlBinding(lease.binding, binding) || !validBrowserPaneControlReceipt(lease, receipt) {
		return errors.New("browser control return receipt does not match the active human and run binding")
	}
	run, err := o.runs.Get(ctx, binding.RunID)
	if err != nil || run == nil {
		return errors.New("browser control return run unavailable")
	}
	if !lease.closedAt.IsZero() && run.Status == domain.RunStatusRunning && run.SessionID == binding.ThreadID && run.LifecycleVersion > lease.parkedLifecycleVersion {
		return nil // exact retry after the wake committed but its response was lost
	}
	if !lease.expires.After(o.now()) || run.Status != domain.RunStatusParked || run.AwaitHandle == nil || run.AwaitHandle.Producer != ProducerBrowserPaneControl || run.AwaitHandle.Key != lease.awaitKey ||
		run.OwnerSubject != binding.OwnerSubject || run.WebConsoleSessionID != binding.WebConsoleSessionID || run.SessionID != binding.ThreadID {
		return errors.New("browser control yield expired or no longer owns this parked run")
	}
	var capability *browsercapability.Capability
	if run.ResolvedConfig != nil {
		capability = run.ResolvedConfig.Features.BrowserCapability
	}
	if capability == nil || !capability.Enabled || !capability.Permissions.Control || strings.TrimSpace(capability.Permissions.ProfileID) != binding.ProfileID || subtle.ConstantTimeCompare([]byte(capability.GrantReceipt), []byte(binding.GrantReceipt)) != 1 || (run.ApprovalState != "" && run.ApprovalState != domain.ApprovalStateNone) {
		return errors.New("browser control permission changed before hand-back")
	}
	result := "The human released browser control. Continue the same authorized run and browser session."
	if _, err := o.WakeRun(ctx, WakeRunInput{RunID: binding.RunID, Result: result}); err != nil {
		return fmt.Errorf("resume exact browser run after control release: %w", err)
	}
	lease.closedAt = o.now()
	return nil
}

func (o *Orchestrator) pruneBrowserPaneControlsLocked() {
	if o.browserPaneControls == nil {
		return
	}
	now := o.now()
	for id, lease := range o.browserPaneControls {
		if lease == nil || !lease.expires.After(now) && (lease.closedAt.IsZero() || now.Sub(lease.closedAt) > time.Minute) {
			delete(o.browserPaneControls, id)
		}
	}
}

func validateBrowserPaneControlBinding(b BrowserPaneControlBinding) error {
	if b.RunID == uuid.Nil || strings.TrimSpace(b.OwnerSubject) == "" || strings.TrimSpace(b.WebConsoleSessionID) == "" || strings.TrimSpace(b.ThreadID) == "" || strings.TrimSpace(b.BASSessionID) == "" || strings.TrimSpace(b.GrantReceipt) == "" || strings.TrimSpace(b.ProfileID) == "" {
		return errors.New("browser control binding is incomplete")
	}
	return nil
}

func sameBrowserPaneControlBinding(a, b BrowserPaneControlBinding) bool {
	return a.RunID == b.RunID && a.OwnerSubject == b.OwnerSubject && a.WebConsoleSessionID == b.WebConsoleSessionID && a.ThreadID == b.ThreadID && a.BASSessionID == b.BASSessionID && a.ProfileID == b.ProfileID && subtle.ConstantTimeCompare([]byte(a.GrantReceipt), []byte(b.GrantReceipt)) == 1
}

func validBrowserPaneControlReceipt(lease *browserPaneControlLease, receipt string) bool {
	secret, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(receipt))
	if err != nil || len(secret) != 32 {
		return false
	}
	hash := sha256.Sum256(secret)
	return subtle.ConstantTimeCompare(hash[:], lease.tokenHash[:]) == 1
}
