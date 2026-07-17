// Package mocks holds the onboard domain's co-located test fakes for its
// Repository and seams (SSHDriver, CodeIssuer, OnlineConfirmer). Deleting
// internal/onboard/ takes these with it.
package mocks

import (
	"context"
	"sort"
	"sync"
	"time"

	"vrooli-bridge/internal/onboard"

	"github.com/google/uuid"
)

// FakeRepository is an in-memory onboard.Repository. Service tests drive the
// service against a controllable persistence layer without sqlite.
type FakeRepository struct {
	mu     sync.Mutex
	ops    map[string]onboard.Op
	events map[string][]onboard.StepEvent

	CreateErr error
	Now       time.Time
}

// NewFakeRepository constructs an empty fake.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{
		ops:    make(map[string]onboard.Op),
		events: make(map[string][]onboard.StepEvent),
	}
}

var _ onboard.Repository = (*FakeRepository)(nil)

func (f *FakeRepository) now() time.Time {
	if !f.Now.IsZero() {
		return f.Now
	}
	return time.Unix(0, 0).UTC()
}

func (f *FakeRepository) Create(_ context.Context, op onboard.Op) (onboard.Op, error) {
	if f.CreateErr != nil {
		return onboard.Op{}, f.CreateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if op.ID == "" {
		op.ID = uuid.NewString()
	}
	if op.CreatedAt.IsZero() {
		op.CreatedAt = f.now()
	}
	if op.State == onboard.StateUnspecified {
		op.State = onboard.StatePending
	}
	f.ops[op.ID] = op
	return op, nil
}

func (f *FakeRepository) Get(_ context.Context, id string) (onboard.Op, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	op, ok := f.ops[id]
	if !ok {
		return onboard.Op{}, onboard.ErrOpNotFound{ID: id}
	}
	return op, nil
}

func (f *FakeRepository) List(_ context.Context, filter onboard.ListFilter) ([]onboard.Op, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]onboard.Op, 0, len(f.ops))
	for _, op := range f.ops {
		if filter.Host != "" && op.Host != filter.Host {
			continue
		}
		out = append(out, op)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

func (f *FakeRepository) ListNonTerminal(_ context.Context) ([]onboard.Op, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []onboard.Op
	for _, op := range f.ops {
		if !op.State.Terminal() {
			out = append(out, op)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (f *FakeRepository) DeleteFailed(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	op, ok := f.ops[id]
	if !ok {
		return onboard.ErrOpNotFound{ID: id}
	}
	if op.State != onboard.StateFailed {
		return onboard.ErrInvalid{Field: "id", Reason: "only failed onboarding attempts can be removed"}
	}
	delete(f.ops, id)
	delete(f.events, id)
	return nil
}

func (f *FakeRepository) Update(_ context.Context, op onboard.Op) (onboard.Op, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	existing, ok := f.ops[op.ID]
	if !ok {
		return onboard.Op{}, onboard.ErrOpNotFound{ID: op.ID}
	}
	existing.State = op.State
	existing.NodeID = op.NodeID
	existing.FailureReason = op.FailureReason
	existing.FailureDetail = op.FailureDetail
	existing.ExitCode = op.ExitCode
	existing.StartedAt = op.StartedAt
	existing.FinishedAt = op.FinishedAt
	existing.SourceMode = op.SourceMode
	existing.BaseRevision = op.BaseRevision
	existing.WorkingTreeDigest = op.WorkingTreeDigest
	existing.ControlPlaneURL = op.ControlPlaneURL
	existing.ReachabilityMode = op.ReachabilityMode
	f.ops[op.ID] = existing
	return existing, nil
}

func (f *FakeRepository) AppendEvent(_ context.Context, ev onboard.StepEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	// Mirror the sqlite INSERT OR IGNORE de-dup on (op_id, sequence).
	for _, existing := range f.events[ev.OpID] {
		if existing.Sequence == ev.Sequence {
			return nil
		}
	}
	f.events[ev.OpID] = append(f.events[ev.OpID], ev)
	return nil
}

func (f *FakeRepository) ListEvents(_ context.Context, opID string) ([]onboard.StepEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	evs := append([]onboard.StepEvent(nil), f.events[opID]...)
	sort.Slice(evs, func(i, j int) bool { return evs[i].Sequence < evs[j].Sequence })
	return evs, nil
}

// Seed inserts an op directly for test setup, bypassing Create's stamping.
func (f *FakeRepository) Seed(op onboard.Op) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ops[op.ID] = op
}

// ---- seam fakes ----

// FakeSSHDriver is a scriptable SSHDriver. Per-phase error/blocking knobs let a
// test fail (or hang-until-cancel) at any step; RunBootstrap replays a canned
// marker script and records the pairing code + args it was handed.
type FakeSSHDriver struct {
	mu sync.Mutex

	FirstTouchErr           error
	FirstTouchBlock         bool
	PushScriptErr           error
	PushScriptBlock         bool
	RunBootstrapBlock       bool
	RunBootstrapErr         error
	RunBootstrapExit        int
	RunBootstrapMarkers     []onboard.Marker
	RunBootstrapDiagnostics string

	// FirstTouchSudoState, when set, is echoed back on the returned Conn so a test
	// can assert the orchestrator surfaces the sudo outcome in the step detail.
	FirstTouchSudoState string

	// Working-tree ship (SyncTree) knobs + capture.
	SyncTreeErr         error
	SyncTreeBlock       bool
	SyncTreeResult      onboard.SyncResult
	SyncTreeCalls       int
	CapturedSyncDest    string
	CapturedSyncTree    onboard.SyncParams
	DetectPlatformErr   error
	DetectedPlatform    onboard.NodePlatform
	DetectPlatformCalls int
	PushArtifactsErr    error
	RemoteArtifacts     onboard.RemoteArtifacts
	PushArtifactsCalls  int
	CapturedArtifacts   onboard.ArtifactPushParams

	// Captured for assertions.
	CapturedPairingCode   []byte
	CapturedArgs          []string
	FirstTouchCalls       int
	CapturedProvisionSudo bool
	AdmissionResult       onboard.AdmissionResult
	AdmissionResults      []onboard.AdmissionResult
	AdmissionErr          error
	AdmissionCalls        int
}

var _ onboard.SSHDriver = (*FakeSSHDriver)(nil)

func (d *FakeSSHDriver) FirstTouch(ctx context.Context, p onboard.FirstTouchParams) (onboard.Conn, error) {
	d.mu.Lock()
	d.FirstTouchCalls++
	d.CapturedProvisionSudo = p.ProvisionSudo
	d.mu.Unlock()
	if d.FirstTouchBlock {
		<-ctx.Done()
		return onboard.Conn{}, ctx.Err()
	}
	if d.FirstTouchErr != nil {
		return onboard.Conn{}, d.FirstTouchErr
	}
	return onboard.Conn{Host: p.Host, Port: p.Port, User: p.User, KeyPath: "/state/bridge-onboard", SudoState: d.FirstTouchSudoState}, nil
}

func (d *FakeSSHDriver) PushScript(ctx context.Context, _ onboard.Conn) (string, error) {
	if d.PushScriptBlock {
		<-ctx.Done()
		return "", ctx.Err()
	}
	if d.PushScriptErr != nil {
		return "", d.PushScriptErr
	}
	return "/tmp/bootstrap.sh", nil
}

// ProbeEndpoint defaults to a successful candidate proof so existing focused
// onboarding tests describe a reachable control plane unless they opt into a
// specific admission failure.
func (d *FakeSSHDriver) ProbeEndpoint(_ context.Context, _ onboard.Conn, endpoint string) (onboard.AdmissionResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.AdmissionCalls++
	if d.AdmissionErr != nil {
		return d.AdmissionResult, d.AdmissionErr
	}
	result := d.AdmissionResult
	if index := d.AdmissionCalls - 1; index >= 0 && index < len(d.AdmissionResults) {
		result = d.AdmissionResults[index]
	}
	if result.Category == "" {
		result.Category = onboard.AdmissionPassed
	}
	if result.Endpoint == "" {
		result.Endpoint = endpoint
	}
	return result, nil
}

func (d *FakeSSHDriver) SyncTree(ctx context.Context, p onboard.SyncParams) (onboard.SyncResult, error) {
	d.mu.Lock()
	d.SyncTreeCalls++
	d.CapturedSyncTree = p
	d.CapturedSyncDest = p.DestDir
	d.mu.Unlock()
	if d.SyncTreeBlock {
		<-ctx.Done()
		return onboard.SyncResult{}, ctx.Err()
	}
	if d.SyncTreeErr != nil {
		return onboard.SyncResult{}, d.SyncTreeErr
	}
	res := d.SyncTreeResult
	if res.ResolvedDestDir == "" {
		res.ResolvedDestDir = p.DestDir
		if res.ResolvedDestDir == "" {
			res.ResolvedDestDir = "/home/node/vrooli"
		}
	}
	return res, nil
}

func (d *FakeSSHDriver) DetectPlatform(_ context.Context, _ onboard.Conn) (onboard.NodePlatform, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.DetectPlatformCalls++
	if d.DetectPlatformErr != nil {
		return onboard.NodePlatform{}, d.DetectPlatformErr
	}
	if d.DetectedPlatform.OS == "" {
		return onboard.NodePlatform{OS: "linux", Arch: "amd64"}, nil
	}
	return d.DetectedPlatform, nil
}

func (d *FakeSSHDriver) PushArtifacts(_ context.Context, p onboard.ArtifactPushParams) (onboard.RemoteArtifacts, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.PushArtifactsCalls++
	d.CapturedArtifacts = p
	if d.PushArtifactsErr != nil {
		return onboard.RemoteArtifacts{}, d.PushArtifactsErr
	}
	if d.RemoteArtifacts.Vrooli == "" {
		return onboard.RemoteArtifacts{
			Vrooli: "/tmp/artifacts/vrooli", BridgeCLI: "/tmp/artifacts/vrooli-bridge", Agent: "/tmp/artifacts/vrooli-bridge-agent",
		}, nil
	}
	return d.RemoteArtifacts, nil
}

func (d *FakeSSHDriver) RunBootstrap(ctx context.Context, p onboard.RunParams, onMarker func(onboard.Marker)) (onboard.BootstrapResult, error) {
	// Record a COPY of the injected code + args before any zeroing by the caller.
	d.mu.Lock()
	d.CapturedPairingCode = append([]byte(nil), p.PairingCode...)
	d.CapturedArgs = append([]string(nil), p.Args...)
	d.mu.Unlock()

	if d.RunBootstrapBlock {
		<-ctx.Done()
		return onboard.BootstrapResult{}, ctx.Err()
	}
	for _, m := range d.RunBootstrapMarkers {
		onMarker(m)
	}
	if d.RunBootstrapErr != nil {
		return onboard.BootstrapResult{ExitCode: d.RunBootstrapExit, Diagnostics: d.RunBootstrapDiagnostics}, d.RunBootstrapErr
	}
	return onboard.BootstrapResult{ExitCode: d.RunBootstrapExit, Diagnostics: d.RunBootstrapDiagnostics}, nil
}

// PairingCode returns a copy of the code the driver was handed.
func (d *FakeSSHDriver) PairingCode() []byte {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]byte(nil), d.CapturedPairingCode...)
}

// FakeCodeIssuer returns a canned pairing code and records the params.
type FakeCodeIssuer struct {
	Code       string
	Err        error
	LastParams onboard.IssueParams
	Calls      int
}

var _ onboard.CodeIssuer = (*FakeCodeIssuer)(nil)

func (f *FakeCodeIssuer) Issue(_ context.Context, p onboard.IssueParams) ([]byte, error) {
	f.Calls++
	f.LastParams = p
	if f.Err != nil {
		return nil, f.Err
	}
	return []byte(f.Code), nil
}

// FakeWorkingTreeSource returns a canned working-tree snapshot and records calls.
type FakeWorkingTreeSource struct {
	Snapshot_ onboard.WorkingTreeSnapshot
	Err       error
	Calls     int
}

var _ onboard.WorkingTreeSource = (*FakeWorkingTreeSource)(nil)

func (f *FakeWorkingTreeSource) Snapshot(_ context.Context) (onboard.WorkingTreeSnapshot, error) {
	f.Calls++
	if f.Err != nil {
		return onboard.WorkingTreeSnapshot{}, f.Err
	}
	return f.Snapshot_, nil
}

// FakeArtifactBuilder records the one target/root requested by working-tree
// onboarding and returns a canned bundle without running a toolchain.
type FakeArtifactBuilder struct {
	Result onboard.PrebuiltArtifacts
	Err    error
	Calls  int
	Last   onboard.ArtifactBuildParams
}

var _ onboard.ArtifactBuilder = (*FakeArtifactBuilder)(nil)

func (f *FakeArtifactBuilder) Build(_ context.Context, p onboard.ArtifactBuildParams) (onboard.PrebuiltArtifacts, error) {
	f.Calls++
	f.Last = p
	if f.Err != nil {
		return onboard.PrebuiltArtifacts{}, f.Err
	}
	if f.Result.Vrooli == "" {
		f.Result = onboard.PrebuiltArtifacts{
			Vrooli: "/cp/artifacts/vrooli", VrooliSidecar: "/cp/artifacts/vrooli.fp",
			BridgeCLI: "/cp/artifacts/vrooli-bridge", BridgeSidecar: "/cp/artifacts/vrooli-bridge.fp",
			Agent: "/cp/artifacts/vrooli-bridge-agent", AgentSidecar: "/cp/artifacts/vrooli-bridge-agent.fp",
			Fingerprint: "test-fingerprint", Target: p.Target,
		}
	}
	return f.Result, nil
}

// FakeNodeRevisionRecorder records the revision it was asked to stamp on a node.
type FakeNodeRevisionRecorder struct {
	Err          error
	Calls        int
	LastNodeID   string
	LastRevision string
}

var _ onboard.NodeRevisionRecorder = (*FakeNodeRevisionRecorder)(nil)

func (f *FakeNodeRevisionRecorder) RecordRevision(_ context.Context, nodeID, revision string) error {
	f.Calls++
	f.LastNodeID = nodeID
	f.LastRevision = revision
	return f.Err
}

// FakeOnlineConfirmer reports a canned online result.
type FakeOnlineConfirmer struct {
	Online bool
	Err    error
	LastID string
	Calls  int
}

var _ onboard.OnlineConfirmer = (*FakeOnlineConfirmer)(nil)

func (f *FakeOnlineConfirmer) ConfirmOnline(_ context.Context, nodeID string, _ time.Duration) (bool, error) {
	f.LastID = nodeID
	f.Calls++
	return f.Online, f.Err
}
