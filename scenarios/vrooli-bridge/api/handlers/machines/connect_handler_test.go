package machines

import (
	"context"
	"testing"

	"vrooli-bridge/internal/auth"
	internalmachines "vrooli-bridge/internal/machines"
	internalonboard "vrooli-bridge/internal/onboard"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines"
)

type fakeService struct {
	createInput internalmachines.CreateInput
	createOut   internalmachines.Machine
	createErr   error
	getOut      internalmachines.Machine
	getErr      error
	archiveOut  internalmachines.Machine
	archiveErr  error
	removeOut   internalmachines.Machine
	removeErr   error
	mergeOut    internalmachines.Machine
	mergeErr    error
	trustOut    internalmachines.TrustRecord
	reviewOut   internalmachines.TrustRecord
	cleanupOut  []internalmachines.CleanupTombstone
	policyInput internalmachines.PolicyChangeInput
}

func (f *fakeService) Create(_ context.Context, in internalmachines.CreateInput) (internalmachines.Machine, error) {
	f.createInput = in
	return f.createOut, f.createErr
}

func (f *fakeService) Get(context.Context, string) (internalmachines.Machine, error) {
	return f.getOut, f.getErr
}
func (f *fakeService) List(context.Context) ([]internalmachines.Machine, error) { return nil, nil }
func (f *fakeService) Archive(context.Context, string, int64) (internalmachines.Machine, error) {
	return f.archiveOut, f.archiveErr
}

func (f *fakeService) Remove(context.Context, string, int64) (internalmachines.Machine, error) {
	return f.removeOut, f.removeErr
}

func (f *fakeService) Merge(context.Context, internalmachines.MergeInput) (internalmachines.Machine, error) {
	return f.mergeOut, f.mergeErr
}

func (f *fakeService) GetTrust(context.Context, string) (internalmachines.TrustRecord, error) {
	return f.trustOut, nil
}

func (f *fakeService) ReviewHostKey(context.Context, string, string) (internalmachines.TrustRecord, error) {
	return f.reviewOut, nil
}

type fakeHostKeyResetter struct {
	host string
	port int
}

type fakeNodeRevoker struct{ nodeID string }

type fakeRepairer struct {
	opID, attemptID string
	err             error
}

func (f fakeRepairer) Repair(context.Context, internalmachines.Machine) (string, string, error) {
	return f.opID, f.attemptID, f.err
}

type fakeEnrollmentStarter struct {
	in internalonboard.StartInput
}

func (f *fakeEnrollmentStarter) StartMachineEnrollment(_ context.Context, _ string, in internalonboard.StartInput) (internalonboard.MachineEnrollmentDecision, error) {
	f.in = in
	return internalonboard.MachineEnrollmentDecision{
		Decision: internalonboard.Decision{OpID: "op-1"},
		Attempt:  internalonboard.EnrollmentAttempt{ID: "attempt-1"},
	}, nil
}

type fakeTrustReader struct {
	trust internalmachines.TrustRecord
}

func (f fakeTrustReader) GetTrust(context.Context, string) (internalmachines.TrustRecord, error) {
	return f.trust, nil
}

func TestMachineRepairerUsesPersistedSSHTrust(t *testing.T) {
	starter := &fakeEnrollmentStarter{}
	repairer := machineRepairer{
		service: starter,
		trust:   fakeTrustReader{trust: internalmachines.TrustRecord{SSHUser: "matthalloran8", SSHPort: 2222}},
	}

	opID, attemptID, err := repairer.Repair(context.Background(), internalmachines.Machine{
		ID:       "m1",
		Locators: []internalmachines.Locator{{Kind: "hostname", Value: "mini.local"}},
	})
	require.NoError(t, err)
	require.Equal(t, "op-1", opID)
	require.Equal(t, "attempt-1", attemptID)
	require.Equal(t, "mini.local", starter.in.Host)
	require.Equal(t, 2222, starter.in.Port)
	require.Equal(t, "matthalloran8", starter.in.User)
}

type fakeAttemptReader struct {
	attempts []internalonboard.EnrollmentAttempt
	machine  string
}

type fakeProjectionReader struct{ projection internalmachines.Projection }

func (f fakeProjectionReader) Compose(context.Context, internalmachines.Machine) (internalmachines.Projection, error) {
	return f.projection, nil
}

type fakeAuditAppender struct{ events []internalmachines.AuditEvent }

func (f *fakeAuditAppender) AppendAudit(_ context.Context, event internalmachines.AuditEvent) error {
	f.events = append(f.events, event)
	return nil
}

func (f *fakeAuditAppender) ListAudit(_ context.Context, _ string) ([]internalmachines.AuditEvent, error) {
	return append([]internalmachines.AuditEvent(nil), f.events...), nil
}

func (f *fakeAttemptReader) ListAttemptsForMachine(_ context.Context, machineID string) ([]internalonboard.EnrollmentAttempt, error) {
	f.machine = machineID
	return f.attempts, nil
}

func (f *fakeNodeRevoker) RevokeMachineNode(_ context.Context, nodeID string) error {
	f.nodeID = nodeID
	return nil
}

func (f *fakeHostKeyResetter) ForgetHostKey(host string, port int) error {
	f.host, f.port = host, port
	return nil
}

func (f *fakeService) CreateCleanupTombstone(_ context.Context, cleanup internalmachines.CleanupTombstone) (internalmachines.CleanupTombstone, error) {
	return cleanup, nil
}

func (f *fakeService) ListCleanupTombstones(context.Context, string) ([]internalmachines.CleanupTombstone, error) {
	return f.cleanupOut, nil
}

func (f *fakeService) UpdateCleanupTombstone(context.Context, string, internalmachines.CleanupStatus, string) (internalmachines.CleanupTombstone, error) {
	return internalmachines.CleanupTombstone{}, nil
}

func (f *fakeService) ApplyPolicy(_ context.Context, input internalmachines.PolicyChangeInput) (internalmachines.Machine, internalmachines.PolicySnapshot, error) {
	f.policyInput = input
	return internalmachines.Machine{ID: input.MachineID}, internalmachines.PolicySnapshot{ProfileID: input.ProfileID, ProfileVersion: "v1"}, nil
}

func machineOwnerContext() context.Context {
	return auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})
}

// [REQ:BRG-ME-001] The public Machine surface is owner-gated, including
// lifecycle mutation, so Node-facing credentials cannot create identities.
func TestHandler_RequiresOwner(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &fakeService{}})
	_, err := h.CreateMachine(context.Background(), connect.NewRequest(&machinesv1.CreateMachineRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.RemoveMachine(context.Background(), connect.NewRequest(&machinesv1.RemoveMachineRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.MergeMachines(context.Background(), connect.NewRequest(&machinesv1.MergeMachinesRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestHandler_RepairStartsInPlaceEnrollment(t *testing.T) {
	svc := &fakeService{getOut: internalmachines.Machine{ID: "m1", Lifecycle: internalmachines.LifecycleActive, Locators: []internalmachines.Locator{{Kind: "hostname", Value: "mini.local"}}}}
	h := NewConnectHandler(Deps{Service: svc, Repairer: fakeRepairer{opID: "op-1", attemptID: "attempt-1"}})
	resp, err := h.RepairMachine(machineOwnerContext(), machineOwnerRequest(&machinesv1.RepairMachineRequest{MachineId: "m1"}))
	require.NoError(t, err)
	require.Equal(t, "op-1", resp.Msg.OnboardingOpId)
	require.Equal(t, "attempt-1", resp.Msg.EnrollmentAttemptId)
	require.Equal(t, "m1", resp.Msg.Machine.Id)
}

func TestHandler_MergeArchivesExplicitSource(t *testing.T) {
	svc := &fakeService{mergeOut: internalmachines.Machine{ID: "target", Lifecycle: internalmachines.LifecycleActive}}
	h := NewConnectHandler(Deps{Service: svc})
	resp, err := h.MergeMachines(machineOwnerContext(), machineOwnerRequest(&machinesv1.MergeMachinesRequest{FromMachineId: "source", IntoMachineId: "target"}))
	require.NoError(t, err)
	require.Equal(t, "target", resp.Msg.Machine.Id)
	require.Equal(t, "source", resp.Msg.ArchivedMachineId)
}

func machineOwnerRequest[T any](msg *T) *connect.Request[T] {
	return connect.NewRequest(msg)
}

func TestHandler_Create_MapsLocatorsAndDoesNotExposeTrustReferences(t *testing.T) {
	svc := &fakeService{createOut: internalmachines.Machine{ID: "m1", Lifecycle: internalmachines.LifecycleActive, Version: 1, TrustRef: "vrooli/bridge:client-key"}}
	h := NewConnectHandler(Deps{Service: svc})
	resp, err := h.CreateMachine(machineOwnerContext(), connect.NewRequest(&machinesv1.CreateMachineRequest{
		Locators:         []*machinesv1.ConnectionLocator{{Kind: "hostname", Value: "Host.Example.", Ordinal: 2}},
		DesiredProfileId: "standard", DesiredProfileVersion: "v1",
	}))
	require.NoError(t, err)
	require.Equal(t, "m1", resp.Msg.Machine.Id)
	require.Equal(t, internalmachines.Locator{Kind: "hostname", Value: "Host.Example.", Ordinal: 2}, svc.createInput.Locators[0])
	require.Equal(t, "standard", svc.createInput.DesiredProfileID)
	// The wire contract has no TrustRef: secret manager locations remain opaque.
	require.NotContains(t, resp.Msg.Machine.String(), "vrooli/bridge:client-key")
}

func TestHandler_Create_DryRunDoesNotPersistMachine(t *testing.T) {
	svc := &fakeService{}
	h := NewConnectHandler(Deps{Service: svc})
	req := connect.NewRequest(&machinesv1.CreateMachineRequest{Locators: []*machinesv1.ConnectionLocator{{Kind: "hostname", Value: "host.example"}}})
	req.Header().Set(dryRunHeader, "true")
	resp, err := h.CreateMachine(machineOwnerContext(), req)
	require.NoError(t, err)
	require.Equal(t, "dry-run-machine", resp.Msg.Machine.Id)
	require.Equal(t, "", svc.createInput.ID)
	require.Empty(t, svc.createInput.Locators)
}

func TestHandler_GetMachineComposesImmutableEnrollmentHistory(t *testing.T) {
	svc := &fakeService{getOut: internalmachines.Machine{ID: "m1"}, cleanupOut: []internalmachines.CleanupTombstone{{ID: "cleanup-1", MachineID: "m1", Action: "remove_ssh_access", Status: internalmachines.CleanupPending}}}
	attempts := &fakeAttemptReader{attempts: []internalonboard.EnrollmentAttempt{{ID: "attempt-2", MachineID: "m1", RetryOfAttemptID: "attempt-1", CorrelationID: "corr-2", State: internalonboard.AttemptFailed, TerminalResult: "ssh_setup_failed"}}}
	audit := &fakeAuditAppender{events: []internalmachines.AuditEvent{{ID: "audit-1", MachineID: "m1", Action: "archive", Actor: "owner-1"}}}
	h := NewConnectHandler(Deps{Service: svc, Attempts: attempts, Audit: audit, Projection: fakeProjectionReader{projection: internalmachines.Projection{HasNode: true, Node: internalmachines.NodeSnapshot{ID: "node-1", Name: "mac-mini", Capabilities: []string{"presence"}}, Presence: internalmachines.PresenceSnapshot{Connected: true}}}})
	resp, err := h.GetMachine(machineOwnerContext(), connect.NewRequest(&machinesv1.GetMachineRequest{Id: "m1"}))
	require.NoError(t, err)
	require.Equal(t, "m1", attempts.machine)
	require.Len(t, resp.Msg.EnrollmentAttempts, 1)
	require.Equal(t, "attempt-2", resp.Msg.EnrollmentAttempts[0].Id)
	require.Equal(t, "attempt-1", resp.Msg.EnrollmentAttempts[0].RetryOfAttemptId)
	require.Equal(t, "ssh_setup_failed", resp.Msg.EnrollmentAttempts[0].TerminalResult)
	require.Equal(t, "node-1", resp.Msg.CurrentNode.NodeId)
	require.True(t, resp.Msg.CurrentNode.Online)
	require.Equal(t, "archive", resp.Msg.AuditEvents[0].Action)
	require.Len(t, resp.Msg.CleanupTombstones, 1)
	require.Equal(t, "pending", resp.Msg.CleanupTombstones[0].Status)
}

func TestHandler_MapsExpectedDomainErrors(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &fakeService{getErr: internalmachines.ErrNotFound{ID: "gone"}}})
	_, err := h.GetMachine(machineOwnerContext(), connect.NewRequest(&machinesv1.GetMachineRequest{Id: "gone"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	h = NewConnectHandler(Deps{Service: &fakeService{archiveErr: internalmachines.ErrConflict{ID: "m1", Version: 2}}})
	_, err = h.ArchiveMachine(machineOwnerContext(), connect.NewRequest(&machinesv1.ArchiveMachineRequest{Id: "m1", Version: 1}))
	require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
}

func TestHandler_ApplyPolicyPassesExplicitDowngradeConfirmation(t *testing.T) {
	svc := &fakeService{}
	h := NewConnectHandler(Deps{Service: svc})
	_, err := h.ApplyMachinePolicy(machineOwnerContext(), connect.NewRequest(&machinesv1.ApplyMachinePolicyRequest{MachineId: "m1", Version: 2, ProfileId: "managed-connection", ConfirmRemoval: true, Reason: "reduce posture"}))
	require.NoError(t, err)
	require.True(t, svc.policyInput.ConfirmRemoval)
	require.Equal(t, "owner-1", svc.policyInput.Actor)
}

func TestHandler_ArchiveRecordsMachineScopedAudit(t *testing.T) {
	svc := &fakeService{archiveOut: internalmachines.Machine{ID: "m1", Lifecycle: internalmachines.LifecycleArchived}}
	audit := &fakeAuditAppender{}
	h := NewConnectHandler(Deps{Service: svc, Audit: audit})
	_, err := h.ArchiveMachine(machineOwnerContext(), connect.NewRequest(&machinesv1.ArchiveMachineRequest{Id: "m1", Version: 1}))
	require.NoError(t, err)
	require.Len(t, audit.events, 1)
	require.Equal(t, "m1", audit.events[0].MachineID)
	require.Equal(t, "archive", audit.events[0].Action)
	require.Equal(t, "owner-1", audit.events[0].Actor)
}

func TestHandler_ReviewHostKeyResetsOnlyMachineLocatorPin(t *testing.T) {
	svc := &fakeService{getOut: internalmachines.Machine{ID: "m1", Locators: []internalmachines.Locator{{Kind: "hostname", Value: "host.example"}}}, reviewOut: internalmachines.TrustRecord{MachineID: "m1", HostKeyFingerprint: "SHA256:new", HostKeyState: internalmachines.HostKeyVerified}}
	resetter := &fakeHostKeyResetter{}
	h := NewConnectHandler(Deps{Service: svc, HostKeyResetter: resetter})
	resp, err := h.ReviewMachineHostKey(machineOwnerContext(), connect.NewRequest(&machinesv1.ReviewMachineHostKeyRequest{MachineId: "m1", ReplacementHostKeyFingerprint: "SHA256:new"}))
	require.NoError(t, err)
	require.Equal(t, "host.example", resetter.host)
	require.Equal(t, 22, resetter.port)
	require.Equal(t, "SHA256:new", resp.Msg.Trust.HostKeyFingerprint)
}

func TestHandler_RevokeMachineNodeUsesCurrentLineageOnly(t *testing.T) {
	svc := &fakeService{getOut: internalmachines.Machine{ID: "m1", Lineage: []internalmachines.NodeLineage{{NodeID: "old", Current: false}, {NodeID: "current", Current: true}}}}
	revoker := &fakeNodeRevoker{}
	h := NewConnectHandler(Deps{Service: svc, NodeRevoker: revoker})
	resp, err := h.RevokeMachineNode(machineOwnerContext(), connect.NewRequest(&machinesv1.RevokeMachineNodeRequest{MachineId: "m1"}))
	require.NoError(t, err)
	require.Equal(t, "current", revoker.nodeID)
	require.Equal(t, "current", resp.Msg.RevokedNodeId)
}
