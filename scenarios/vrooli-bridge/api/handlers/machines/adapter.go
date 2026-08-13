package machines

import (
	internalmachines "vrooli-bridge/internal/machines"
	internalonboard "vrooli-bridge/internal/onboard"

	"google.golang.org/protobuf/types/known/timestamppb"

	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines"
)

func attemptToProto(attempt internalonboard.EnrollmentAttempt) *machinesv1.EnrollmentAttempt {
	out := &machinesv1.EnrollmentAttempt{
		Id:               attempt.ID,
		RetryOfAttemptId: attempt.RetryOfAttemptID,
		CorrelationId:    attempt.CorrelationID,
		State:            string(attempt.State),
		TerminalResult:   attempt.TerminalResult,
		Diagnostics:      attempt.Diagnostics,
		CreatedAt:        timestamppb.New(attempt.CreatedAt),
	}
	if !attempt.TerminalAt.IsZero() {
		out.TerminalAt = timestamppb.New(attempt.TerminalAt)
	}
	return out
}

func projectionToProto(projection internalmachines.Projection) *machinesv1.CurrentNodeProjection {
	if !projection.HasNode {
		return nil
	}
	return &machinesv1.CurrentNodeProjection{NodeId: projection.Node.ID, Name: projection.Node.Name, Capabilities: append([]string(nil), projection.Node.Capabilities...), Online: projection.Presence.Connected}
}

func auditToProto(event internalmachines.AuditEvent) *machinesv1.MachineAuditEvent {
	return &machinesv1.MachineAuditEvent{Id: event.ID, Action: event.Action, Actor: event.Actor, Detail: event.Detail, CreatedAt: timestamppb.New(event.CreatedAt)}
}

func readinessToProto(readiness internalmachines.Readiness) *machinesv1.MachineReadiness {
	return &machinesv1.MachineReadiness{Ready: readiness.Ready, Reasons: append([]string(nil), readiness.Reasons...)}
}

// domainToProto is the sole translation boundary between Machine persistence
// and its public contract. Nodes remain referenced as lineage, never copied.
func domainToProto(m internalmachines.Machine) *machinesv1.Machine {
	out := &machinesv1.Machine{
		Id:                    m.ID,
		Lifecycle:             string(m.Lifecycle),
		Version:               m.Version,
		DesiredProfileId:      m.DesiredProfileID,
		DesiredProfileVersion: m.DesiredProfileVersion,
		CreatedAt:             timestamppb.New(m.CreatedAt),
		UpdatedAt:             timestamppb.New(m.UpdatedAt),
		Locators:              make([]*machinesv1.ConnectionLocator, 0, len(m.Locators)),
		NodeLineage:           make([]*machinesv1.NodeLineage, 0, len(m.Lineage)),
	}
	for _, locator := range m.Locators {
		out.Locators = append(out.Locators, &machinesv1.ConnectionLocator{Kind: locator.Kind, Value: locator.Value, Ordinal: int32(locator.Ordinal)})
	}
	for _, lineage := range m.Lineage {
		entry := &machinesv1.NodeLineage{
			NodeId:        lineage.NodeID,
			Current:       lineage.Current,
			LinkedAt:      timestamppb.New(lineage.LinkedAt),
			CorrelationId: lineage.CorrelationID,
		}
		if !lineage.SupersededAt.IsZero() {
			entry.SupersededAt = timestamppb.New(lineage.SupersededAt)
		}
		out.NodeLineage = append(out.NodeLineage, entry)
	}
	if !m.ArchivedAt.IsZero() {
		out.ArchivedAt = timestamppb.New(m.ArchivedAt)
	}
	if !m.RemovedAt.IsZero() {
		out.RemovedAt = timestamppb.New(m.RemovedAt)
	}
	return out
}

func trustToProto(trust internalmachines.TrustRecord) *machinesv1.MachineTrust {
	return &machinesv1.MachineTrust{ClientKeyFingerprint: trust.ClientKeyFingerprint, HostKeyFingerprint: trust.HostKeyFingerprint, HostKeyState: string(trust.HostKeyState), SshUser: trust.SSHUser, SshPort: int32(trust.SSHPort), ConnectionState: string(trust.ConnectionState), UpdatedAt: timestamppb.New(trust.UpdatedAt)}
}

func cleanupToProto(cleanup internalmachines.CleanupTombstone) *machinesv1.MachineCleanup {
	out := &machinesv1.MachineCleanup{Id: cleanup.ID, MachineId: cleanup.MachineID, Action: cleanup.Action, Status: string(cleanup.Status), Detail: cleanup.Detail, CreatedAt: timestamppb.New(cleanup.CreatedAt), UpdatedAt: timestamppb.New(cleanup.UpdatedAt)}
	if !cleanup.AcknowledgedAt.IsZero() {
		out.AcknowledgedAt = timestamppb.New(cleanup.AcknowledgedAt)
	}
	return out
}

func policyToProto(snapshot internalmachines.PolicySnapshot) *machinesv1.EffectivePolicy {
	return &machinesv1.EffectivePolicy{ProfileId: snapshot.ProfileID, ProfileVersion: snapshot.ProfileVersion, SetupEnvironment: snapshot.SetupEnvironment, SuggestedScopes: append([]string(nil), snapshot.SuggestedScopes...), RequiredCapabilities: append([]string(nil), snapshot.RequiredCapabilities...), SnapshotJson: snapshot.JSON}
}
