package recovery

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/process"
)

// CodingAgentExecution is a versioned native execution sidecar. Historical
// admission receipts keep their existing format and remain readable by prior
// CLIs. This owner never infers execution identity from their age or runner.
type CodingAgentExecution struct {
	Version           int                      `json:"version"`
	ID                string                   `json:"id"`
	Request           CodingAgentRequest       `json:"request"`
	WorkerSHA         string                   `json:"worker_sha256"`
	BootID            string                   `json:"boot_id"`
	Deadline          time.Time                `json:"deadline"`
	Intent            platform.OperationIntent `json:"intent"`
	DispatchCommitted bool                     `json:"dispatch_committed"`
	Fence             string                   `json:"fence"`
	Ref               *platform.OperationRef   `json:"ref,omitempty"`
	LeaseID           string                   `json:"lease_id,omitempty"`
	LeaseRegistered   bool                     `json:"lease_registered"`
	ExitProven        bool                     `json:"exit_proven"`
	Retired           bool                     `json:"retired"`
	AttachBase        string                   `json:"attach_base,omitempty"`
}

type codingAgentNativeOwner struct {
	worker   string
	start    func(context.Context, platform.OperationIntent, platform.OperationSpec) (*platform.OwnedOperation, error)
	lookup   func(context.Context, platform.OperationIntent) (platform.OperationRef, error)
	inspect  func(context.Context, platform.OperationRef) (platform.OperationObservation, error)
	absent   func(context.Context, platform.OperationIntent) (bool, error)
	stop     func(context.Context, platform.OperationRef) (platform.OperationStopReceipt, error)
	release  func(context.Context, platform.OperationRef) error
	released func(context.Context, platform.OperationRef, string) (bool, error)
}

func nativeCodingOwner(worker string) *codingAgentNativeOwner {
	return &codingAgentNativeOwner{worker: worker, start: platform.StartOwnedOperation, lookup: platform.LookupOwnedOperation, inspect: platform.InspectOwnedOperation, absent: platform.OwnedOperationIntentAbsent, stop: platform.StopOwnedOperation, release: platform.ReleaseOwnedOperation, released: platform.OwnedOperationReleasedForIncarnation}
}

type codingWorkerMetadata struct {
	data     bytes.Buffer
	overflow bool
}

func (w *codingWorkerMetadata) Write(p []byte) (int, error) {
	n := len(p)
	remaining := 256 - w.data.Len()
	if len(p) > remaining {
		w.overflow = true
		p = p[:remaining]
	}
	_, _ = w.data.Write(p)
	return n, nil
}

func qualifyNativeCodingWorker(ctx context.Context, path string) error {
	if err := coreactivation.ValidateRootExecutableRole(path); err != nil {
		return fmt.Errorf("native recovery requires the governed root worker")
	}
	identity, err := coreactivation.ProbeRootIdentity(ctx, path)
	if err != nil || coreactivation.QualifyRollbackIdentities(identity, identity) != nil {
		return fmt.Errorf("native recovery worker lacks governed identity metadata")
	}
	dir, err := os.MkdirTemp("", "vrooli-recovery-capability-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--coding-agent-recovery-capability")
	cmd.Dir = dir
	cmd.Env = []string{"HOME=" + dir, "PATH=/usr/bin:/bin"}
	cmd.WaitDelay = time.Second
	var output codingWorkerMetadata
	cmd.Stdout = &output
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil || output.overflow || output.data.String() != "{\"protocol\":1}\n" {
		return fmt.Errorf("native recovery worker protocol is unavailable")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		return fmt.Errorf("native recovery capability probe initialized runtime state")
	}
	return nil
}

// NewNativeCodingAgentBroker selects the production execution owner. Native
// unavailability is an explicit error; it never falls back into an API child.
func NewNativeCodingAgentBroker(path, worker string) (*CodingAgentBroker, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("native coding-agent recovery is unsupported on this platform")
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || !filepath.IsAbs(worker) || filepath.Clean(worker) != worker {
		return nil, fmt.Errorf("native recovery requires canonical absolute paths")
	}
	b, err := NewCodingAgentBroker(path, nil)
	if err != nil {
		return nil, err
	}
	b.native = nativeCodingOwner(worker)
	if err := qualifyNativeCodingWorker(context.Background(), worker); err != nil {
		return nil, err
	}
	return b, nil
}

var codingScenarioName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,95}$`)
var codingImageDigest = regexp.MustCompile(`^[a-f0-9]{64}$`)

func validNativeCodingRequest(r CodingAgentRequest) bool {
	return codingScenarioName.MatchString(r.Scenario) && r.Requester != "" && len(r.Requester) <= 512 && utf8.ValidString(r.Requester) && !strings.ContainsRune(r.Requester, '\x00') && len(r.Reason) <= 517 && utf8.ValidString(r.Reason) && !strings.ContainsRune(r.Reason, '\x00') && len(r.WorkingDir) <= 4096 && utf8.ValidString(r.WorkingDir) && !strings.ContainsRune(r.WorkingDir, '\x00') && filepath.IsAbs(r.WorkingDir) && filepath.Clean(r.WorkingDir) == r.WorkingDir && r.Claim == filepath.Join("scenarios", r.Scenario)
}

func validRecoveryAttachBase(raw string) bool {
	if raw == "" {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || u.RawPath != "" {
		return false
	}
	ip := net.ParseIP(u.Hostname())
	return (u.Hostname() == "localhost" || ip != nil && ip.IsLoopback()) && u.Port() != ""
}

func (b *CodingAgentBroker) executionDir(id string) (string, error) {
	u, err := uuid.Parse(id)
	if err != nil || u.String() != id || u == uuid.Nil {
		return "", fmt.Errorf("invalid native recovery identity")
	}
	return filepath.Join(filepath.Dir(b.path), "coding-agent-executions", id), nil
}

func (b *CodingAgentBroker) workerPath(id string) (string, error) {
	d, e := b.executionDir(id)
	return filepath.Join(d, "worker"), e
}

func (b *CodingAgentBroker) writeExecution(e CodingAgentExecution) error {
	d, err := b.executionDir(e.ID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if len(data) > 64<<10 {
		return fmt.Errorf("native recovery receipt exceeds bound")
	}
	return config.WriteOwnedFileDurable(filepath.Join(d, "execution.json"), append(data, '\n'), 0o600)
}

func (b *CodingAgentBroker) readExecution(id string) (CodingAgentExecution, error) {
	var e CodingAgentExecution
	d, err := b.executionDir(id)
	if err != nil {
		return e, err
	}
	p := filepath.Join(d, "execution.json")
	info, err := os.Lstat(p)
	if err != nil {
		return e, err
	}
	if !info.Mode().IsRegular() || info.Size() > 64<<10 {
		return e, fmt.Errorf("native recovery receipt is not a bounded regular file")
	}
	f, err := os.Open(p)
	if err != nil {
		return e, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (64<<10)+1))
	if err != nil || len(data) > 64<<10 {
		return e, fmt.Errorf("native recovery receipt unreadable or oversized")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&e); err != nil {
		return e, fmt.Errorf("invalid native recovery receipt")
	}
	canonical, err := json.Marshal(e)
	if err != nil || !bytes.Equal(data, append(canonical, '\n')) {
		return e, fmt.Errorf("noncanonical native recovery receipt")
	}
	boot, bootErr := uuid.Parse(e.BootID)
	if e.Version != 1 || e.ID != id || !validNativeCodingRequest(e.Request) || !validRecoveryAttachBase(e.AttachBase) || !codingImageDigest.MatchString(e.WorkerSHA) || bootErr != nil || boot.String() != e.BootID || boot == uuid.Nil || e.Deadline.IsZero() || e.Intent.Owner != "coding-agent-recovery" || e.Intent.AttemptID != id || e.Intent.Kind != "" || e.Intent.ValidateIdentity() != nil {
		return e, fmt.Errorf("native recovery identity conflict")
	}
	if e.Fence != "ready" && e.Fence != "claimed" && e.Fence != "revoked" {
		return e, fmt.Errorf("invalid native recovery fence")
	}
	if e.Ref != nil && (e.Ref.Intent != e.Intent || e.Ref.ValidateIdentity() != nil) {
		return e, fmt.Errorf("invalid native recovery reference")
	}
	if e.Fence == "claimed" && (!e.DispatchCommitted || e.Ref == nil) {
		return e, fmt.Errorf("native recovery claim lacks dispatch identity")
	}
	if e.ExitProven && e.Fence != "revoked" || e.Retired && !e.ExitProven {
		return e, fmt.Errorf("invalid native recovery retirement proof")
	}
	return e, nil
}

func (b *CodingAgentBroker) nativeSpec(e CodingAgentExecution) (platform.OperationSpec, error) {
	p, err := b.workerPath(e.ID)
	if err != nil {
		return platform.OperationSpec{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return platform.OperationSpec{}, err
	}
	return platform.OperationSpec{Path: p, Args: []string{"--coding-agent-recovery-execute", e.ID, b.path}, Dir: home, Env: []string{"HOME=" + home, "PATH=" + platform.DefaultPath("linux", home)}, Limits: platform.OperationLimits{MemoryMaxBytes: 2 << 30, TasksMax: 1024, CPUWeight: 50, WallTime: CodingAgentRecoveryTimeout}}, nil
}

func (b *CodingAgentBroker) prepareNative(ctx context.Context, r CodingAgentRecord, req CodingAgentRequest) (CodingAgentExecution, error) {
	e := CodingAgentExecution{Version: 1, ID: r.ID, Request: req, Deadline: r.DeadlineAt, Fence: "ready"}
	e.AttachBase = os.Getenv("AGENT_MANAGER_API_BASE")
	if !validRecoveryAttachBase(e.AttachBase) {
		return e, fmt.Errorf("native recovery attachment requires a credential-free loopback endpoint")
	}
	e.Request.Reason = r.Reason
	if !validNativeCodingRequest(e.Request) {
		return e, fmt.Errorf("invalid native recovery request")
	}
	active := 0
	for _, previous := range b.records {
		if _, err := uuid.Parse(previous.ID); err != nil {
			continue
		}
		old, err := b.readExecution(previous.ID)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return e, err
		}
		if !old.ExitProven {
			active++
		}
	}
	if active >= 2 {
		return e, fmt.Errorf("native coding-agent recovery concurrency limit reached")
	}
	if err := coreactivation.ValidateRootExecutableRole(b.native.worker); err != nil {
		return e, fmt.Errorf("native recovery worker is not a root executable")
	}
	if err := qualifyNativeCodingWorker(ctx, b.native.worker); err != nil {
		return e, err
	}
	stamp, err := process.Incarnation(os.Getpid())
	if err != nil {
		return e, err
	}
	e.BootID, _, _ = strings.Cut(stamp, ":")
	dir, err := b.executionDir(r.ID)
	if err != nil {
		return e, err
	}
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return e, err
	}
	directory, err := os.Open(parent)
	if err != nil {
		return e, err
	}
	entries, err := directory.ReadDir(codingAgentMaxRecords + 1)
	_ = directory.Close()
	if err != nil && !errors.Is(err, io.EOF) {
		return e, err
	}
	if len(entries) >= codingAgentMaxRecords {
		return e, fmt.Errorf("native recovery metadata inventory requires review")
	}
	images := 0
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return e, err
		}
		_, err := os.Lstat(filepath.Join(parent, entry.Name(), "worker"))
		if err == nil {
			images++
		} else if !errors.Is(err, os.ErrNotExist) {
			return e, fmt.Errorf("native recovery input inventory is unreadable")
		}
	}
	if images >= 128 {
		return e, fmt.Errorf("native recovery input inventory requires retirement")
	}
	f, err := os.Open(b.native.worker)
	if err != nil {
		return e, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 512<<20 {
		return e, fmt.Errorf("invalid native recovery image")
	}
	hash := sha256.New()
	n, err := io.Copy(hash, io.LimitReader(f, (512<<20)+1))
	if err != nil || n != info.Size() {
		return e, fmt.Errorf("native recovery source image changed or is unreadable")
	}
	e.WorkerSHA = hex.EncodeToString(hash.Sum(nil))
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return e, err
	}
	spec, err := b.nativeSpec(e)
	if err != nil {
		return e, err
	}
	e.Intent, err = platform.NewOperationIntent("coding-agent-recovery", e.ID, spec)
	if err != nil {
		return e, err
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		return e, err
	}
	// Recovery identity precedes the large image copy. A crash cannot leave
	// a published worker image with no descriptor or admitted native intent.
	if err := b.writeExecution(e); err != nil {
		return e, err
	}
	worker, _ := b.workerPath(e.ID)
	err = config.WriteOwnedFileDurableStream(worker, 0o700, func(w io.Writer) error {
		h := sha256.New()
		n, err := io.Copy(io.MultiWriter(w, h), io.LimitReader(f, (512<<20)+1))
		if err != nil {
			return err
		}
		if n != info.Size() || hex.EncodeToString(h.Sum(nil)) != e.WorkerSHA {
			return fmt.Errorf("native recovery image changed")
		}
		return ctx.Err()
	})
	if err != nil {
		return e, err
	}
	if err := coreactivation.ValidateRootExecutableRole(worker); err != nil {
		return e, err
	}
	if err := qualifyNativeCodingWorker(ctx, worker); err != nil {
		return e, err
	}
	return e, nil
}

func (b *CodingAgentBroker) dispatchNative(ctx context.Context, e *CodingAgentExecution) error {
	if e.DispatchCommitted || e.Fence != "ready" {
		return fmt.Errorf("native recovery dispatch is already consumed")
	}
	e.DispatchCommitted = true
	if err := b.writeExecution(*e); err != nil {
		return err
	}
	spec, err := b.nativeSpec(*e)
	if err != nil {
		return err
	}
	op, err := b.native.start(ctx, e.Intent, spec)
	if err != nil {
		return fmt.Errorf("native dispatch acknowledgement unconfirmed")
	}
	e.Ref = &op.Ref
	return b.writeExecution(*e)
}

// ClaimNativeExecution is called only by the fixed frozen worker entrypoint.
// The once-only claim is committed under the same ledger lock as admission.
func (b *CodingAgentBroker) ClaimNativeExecution(ctx context.Context, id string) (CodingAgentExecution, error) {
	if ctx == nil {
		return CodingAgentExecution{}, fmt.Errorf("native recovery context is required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	unlock, err := b.lockLedger(ctx)
	if err != nil {
		return CodingAgentExecution{}, err
	}
	defer unlock()
	if err := b.loadLocked(); err != nil {
		return CodingAgentExecution{}, err
	}
	e, err := b.readExecution(id)
	if err != nil {
		return e, err
	}
	admitted := false
	for _, r := range b.records {
		if r.Outcome == "scheduled" && nativeCodingAdmissionMatches(r, e) {
			admitted = true
		}
	}
	if !admitted || !e.DispatchCommitted || e.Fence != "ready" || !b.now().Before(e.Deadline) {
		return e, fmt.Errorf("native recovery claim is not eligible")
	}
	ref, err := platform.LookupOwnedOperation(ctx, e.Intent)
	if err != nil {
		return e, err
	}
	if err := verifyNativeCodingSelf(ctx, e, ref); err != nil {
		return e, err
	}
	e.Ref = &ref
	e.Fence = "claimed"
	return e, b.writeExecution(e)
}

func verifyNativeCodingSelf(ctx context.Context, e CodingAgentExecution, ref platform.OperationRef) error {
	if ref.ValidateIdentity() != nil || ref.Intent != e.Intent || ref.RootPID != os.Getpid() {
		return fmt.Errorf("native recovery worker root identity mismatch")
	}
	stamp, err := process.Incarnation(os.Getpid())
	if err != nil || stamp != e.BootID+":"+ref.RootStartTime {
		return fmt.Errorf("native recovery worker incarnation mismatch")
	}
	ob, err := platform.InspectOwnedOperation(ctx, ref)
	if err != nil || ob.State != platform.OperationRunning || !ob.RootAlive || !ob.Populated {
		return fmt.Errorf("native recovery worker is not live")
	}
	members, err := platform.ScopeProcessesContext(ctx, ref.Scope)
	if err != nil {
		return err
	}
	found := false
	for _, pid := range members {
		if pid == os.Getpid() {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("native recovery worker is outside its bound tree")
	}
	sha, err := coreactivation.RunningImageDigest(ctx, os.Getpid())
	if err != nil || sha != e.WorkerSHA {
		return fmt.Errorf("native recovery worker image mismatch")
	}
	return nil
}

// BindNativeLease durably links the launcher-created lease before its child
// starts. The actual native cgroup, rather than a proposed sibling scope, owns
// this session. Registry unavailability may degrade observation, not custody.
func (b *CodingAgentBroker) BindNativeLease(ctx context.Context, id, lease string, registered bool) error {
	if ctx == nil {
		return fmt.Errorf("native recovery context is required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	unlock, err := b.lockLedger(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	if err := b.loadLocked(); err != nil {
		return err
	}
	e, err := b.readExecution(id)
	if err != nil {
		return err
	}
	if e.Fence != "claimed" || e.Ref == nil || !b.now().Before(e.Deadline) || e.LeaseID != "" && e.LeaseID != lease {
		return fmt.Errorf("native recovery lease fence refused")
	}
	if !b.nativeRunningAdmission(e) {
		return fmt.Errorf("native recovery lease admission changed")
	}
	if lease == "" || len(lease) > 256 {
		return fmt.Errorf("invalid native recovery lease")
	}
	if err := verifyNativeCodingSelf(ctx, e, *e.Ref); err != nil {
		return err
	}
	e.LeaseID = lease
	e.LeaseRegistered = registered
	return b.writeExecution(e)
}

func (b *CodingAgentBroker) GuardNativeChild(ctx context.Context, id string) error {
	if ctx == nil {
		return fmt.Errorf("native recovery context is required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	unlock, err := b.lockLedger(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	if err := b.loadLocked(); err != nil {
		return err
	}
	e, err := b.readExecution(id)
	if err != nil {
		return err
	}
	if e.Fence != "claimed" || e.Ref == nil || e.LeaseID == "" || !b.now().Before(e.Deadline) {
		return fmt.Errorf("native recovery child fence refused")
	}
	if !b.nativeRunningAdmission(e) {
		return fmt.Errorf("native recovery child admission changed")
	}
	return verifyNativeCodingSelf(ctx, e, *e.Ref)
}

func (b *CodingAgentBroker) nativeRunningAdmission(e CodingAgentExecution) bool {
	for _, r := range b.records {
		if r.Outcome == "running" && nativeCodingAdmissionMatches(r, e) {
			return true
		}
	}
	return false
}

// RunNativeExecution retains the deadline admitted by the original caller.
// It does not create another native job or extend time after caller loss.
func (b *CodingAgentBroker) RunNativeExecution(ctx context.Context, e CodingAgentExecution, run CodingAgentRunner) error {
	if ctx == nil || run == nil {
		return fmt.Errorf("native recovery runner is required")
	}
	ctx, cancel := context.WithDeadline(ctx, e.Deadline)
	defer cancel()
	b.run = run
	b.executeContext(ctx, e.ID, e.Request)
	return nil
}

// NativeExecutions reads only sidecars bound to existing ledger receipts.
func (b *CodingAgentBroker) NativeExecutions() ([]CodingAgentExecution, error) {
	records, err := b.Snapshot()
	if err != nil {
		return nil, err
	}
	out := []CodingAgentExecution{}
	for _, r := range records {
		if _, err := uuid.Parse(r.ID); err != nil {
			continue // Historical receipt identities cannot name native inputs.
		}
		e, err := b.readExecution(r.ID)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

type CodingAgentOwnership struct {
	Record        CodingAgentRecord
	NativePending bool
}

// NativeOwnershipSnapshot projects only executions bound to their admission
// through the canonical owner evaluator. Historical receipts without a native
// sidecar still suppress duplicate coding cost; they do not claim a live writer.
// Receipt completion alone is not native exit proof.
func (b *CodingAgentBroker) NativeOwnershipSnapshot(ctx context.Context) ([]CodingAgentOwnership, error) {
	if ctx == nil {
		return nil, fmt.Errorf("native recovery ownership context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Observe the authoritative files under the existing cross-process owner
	// lock. Avoid the legacy process-cache mutex and its background lock wait;
	// an optional observer must honor its monitoring caller's cancellation.
	unlock, err := b.lockLedger(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()
	records, err := readCodingAgentRecords(b.path)
	if err != nil {
		return nil, err
	}
	out := make([]CodingAgentOwnership, 0)
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, err := uuid.Parse(record.ID); err != nil {
			continue
		}
		execution, err := b.readExecution(record.ID)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !nativeCodingAdmissionMatches(record, execution) {
			return nil, fmt.Errorf("native recovery ownership does not match its admission")
		}
		out = append(out, CodingAgentOwnership{Record: record, NativePending: !execution.ExitProven})
	}
	return out, nil
}

// RevokeNativeExecution closes only this admitted execution's effect fence.
// It does not signal a process or turn cancellation into proof of exit;
// ReconcileNative must still exclude the exact tree before closing the receipt.
func (b *CodingAgentBroker) RevokeNativeExecution(ctx context.Context, id string) error {
	if ctx == nil {
		return fmt.Errorf("native recovery context is required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	unlock, err := b.lockLedger(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	if err := b.loadLocked(); err != nil {
		return err
	}
	found := false
	for _, r := range b.records {
		if r.ID == id {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("native recovery admission is missing")
	}
	e, err := b.readExecution(id)
	if err != nil {
		return err
	}
	e.Fence = "revoked"
	return b.writeExecution(e)
}

// RunNativeReconciliation performs bounded control-plane reconciliation while
// its API host is alive. Losing this observer never removes native limits or
// transfers job custody back to the API process.
func (b *CodingAgentBroker) RunNativeReconciliation(ctx context.Context, report func(error)) {
	if ctx == nil {
		return
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		operation, cancel := context.WithTimeout(ctx, 10*time.Second)
		_, err := b.ReconcileNative(operation, 32)
		cancel()
		if err != nil && ctx.Err() == nil && report != nil {
			report(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (b *CodingAgentBroker) retireNativeWorker(ctx context.Context, e CodingAgentExecution) error {
	if e.Fence != "revoked" || !e.ExitProven {
		return fmt.Errorf("native recovery inputs still in use")
	}
	p, err := b.workerPath(e.ID)
	if err != nil {
		return err
	}
	entries, err := boundedCodingInputEntries(filepath.Dir(p), 64)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() > 512<<20 {
			return fmt.Errorf("native recovery retirement contains nonregular or oversized input")
		}
		if entry.Name() != "worker" && entry.Name() != "execution.json" && !strings.HasPrefix(entry.Name(), ".vrooli-owned-") {
			return fmt.Errorf("native recovery retirement contains unexpected input; preserving it")
		}
	}
	sha, err := coreactivation.ImageDigest(ctx, p)
	if !errors.Is(err, os.ErrNotExist) {
		if err != nil || sha != e.WorkerSHA {
			return fmt.Errorf("native recovery retained image changed; preserving it")
		}
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".vrooli-owned-") {
			if err := os.Remove(filepath.Join(filepath.Dir(p), entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	dir, err := os.Open(filepath.Dir(p))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
