package cleanup

import (
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/cleanup"
	"github.com/vrooli/vrooli/packages/proto/privilegedops"
	"vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/channelsign"
	internalcleanup "vrooli-bridge/internal/cleanup"
	internalmachines "vrooli-bridge/internal/machines"
	onboardssh "vrooli-bridge/internal/onboard/ssh"
	"vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/registry"
)

func opToProto(op internalcleanup.Operation) *cleanupv1.CleanupOperation {
	out := &cleanupv1.CleanupOperation{Id: op.ID, MachineId: op.MachineID, NodeId: op.NodeID, Target: op.Target, Scope: op.Scope, Status: statusToProto(op.Status), Transport: op.Transport, TransportReason: op.TransportReason, Reason: op.Reason, PlanHash: op.PlanHash, PlanJson: append([]byte(nil), op.PlanJSON...), ReceiptJson: append([]byte(nil), op.ReceiptJSON...), OperatorId: op.OperatorID, SealingPublicKey: append([]byte(nil), op.SealingPublicKey...), CreatedAt: timestamppb.New(op.CreatedAt), UpdatedAt: timestamppb.New(op.UpdatedAt)}
	if !op.FinishedAt.IsZero() {
		out.FinishedAt = timestamppb.New(op.FinishedAt)
	}
	return out
}

func statusToProto(s internalcleanup.Status) cleanupv1.CleanupStatus {
	return cleanupv1.CleanupStatus(s)
}

func eventToProto(ev internalcleanup.Event) *cleanupv1.CleanupEvent {
	return &cleanupv1.CleanupEvent{OperationId: ev.OperationID, Kind: cleanupv1.CleanupEventKind(ev.Kind), Sequence: ev.Sequence, Status: ev.Status, LogChunk: ev.LogChunk, PlanJson: append([]byte(nil), ev.PlanJSON...), ReceiptJson: append([]byte(nil), ev.ReceiptJSON...), Reason: ev.Reason, ExitCode: ev.ExitCode, EmittedAt: timestamppb.New(ev.EmittedAt)}
}

func eventFromProto(ev *cleanupv1.CleanupEvent) internalcleanup.Event {
	out := internalcleanup.Event{OperationID: ev.GetOperationId(), Kind: internalcleanup.EventKind(ev.GetKind()), Sequence: ev.GetSequence(), Status: ev.GetStatus(), LogChunk: ev.GetLogChunk(), PlanJSON: append([]byte(nil), ev.GetPlanJson()...), ReceiptJSON: append([]byte(nil), ev.GetReceiptJson()...), Reason: ev.GetReason(), ExitCode: ev.GetExitCode()}
	if ev.GetEmittedAt() != nil {
		out.EmittedAt = ev.GetEmittedAt().AsTime()
	}
	return out
}

func operationToProto(name string) channelv1.PrivilegedOperation {
	if operation, ok := privilegedops.Parse(name); ok {
		return operation
	}
	return channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_UNSPECIFIED
}

type sealingKeySource interface {
	SealingPublicKey(context.Context, string) ([]byte, error)
}

type nodeReader struct {
	svc  registry.Service
	keys sealingKeySource
}

func (a nodeReader) GetTarget(ctx context.Context, id string) (internalcleanup.TargetNode, error) {
	n, err := a.svc.Get(ctx, id)
	if err != nil {
		return internalcleanup.TargetNode{}, err
	}
	var key []byte
	if a.keys != nil {
		key, err = a.keys.SealingPublicKey(ctx, id)
		if err != nil {
			return internalcleanup.TargetNode{}, err
		}
	}
	return internalcleanup.TargetNode{ID: n.ID, Kind: n.Kind, Revoked: n.Revoked(), SealingPublicKey: key, Capabilities: append([]string(nil), n.Capabilities...), Scopes: append([]string(nil), n.Scopes...), Endpoint: n.Endpoint}, nil
}

type auditSinkAdapter struct{ sink audit.Sink }

func (a auditSinkAdapter) Record(ctx context.Context, e internalcleanup.AuditEntry) error {
	outcome := audit.OutcomeRejected
	if e.Outcome == "accepted" {
		outcome = audit.OutcomeAccepted
	}
	_, err := a.sink.Append(ctx, audit.Record{Action: audit.ActionCleanup, Actor: e.Actor, NodeID: e.NodeID, Verb: e.Verb, Args: []string{e.OperationID}, Outcome: outcome, Detail: e.Detail, RunID: e.OperationID})
	return err
}

type commandPusher struct {
	hub       *presence.Hub
	signer    channelsign.Signer
	sshPusher *typedSSHPusher
}

func (p commandPusher) PushCleanup(_ context.Context, nodeID string, cmd internalcleanup.Command) (int, error) {
	frame := &channelv1.ServerFrame{FrameId: uuid.NewString(), Payload: &channelv1.ServerFrame_Cleanup{Cleanup: &channelv1.CleanupCommand{Operation: operationToProto(cmd.Operation), OpId: cmd.OpID, MachineId: cmd.MachineID, NodeId: cmd.NodeID, Target: cmd.Target, Scope: cmd.Scope, PlanId: cmd.PlanID, PlanHash: cmd.PlanHash, SealedPassphrase: append([]byte(nil), cmd.SealedPassphrase...), Capability: append([]byte(nil), cmd.Capability...), OperatorConfirmed: cmd.OperatorConfirmed, OperatorId: cmd.OperatorID}}}
	payload, err := channelsign.Marshal(p.signer, frame)
	if err != nil {
		return 0, err
	}
	delivered := p.hub.PushFrame(nodeID, frame.GetFrameId(), payload)
	if delivered == 0 {
		return 0, internalcleanup.ErrBlocked{Field: "reachability", Reason: "cleanup command was not delivered to the paired agent"}
	}
	return delivered, nil
}

func (p commandPusher) PushCleanupSSH(ctx context.Context, machineID string, cmd internalcleanup.Command) (int, error) {
	if p.sshPusher == nil {
		return 0, internalcleanup.ErrBlocked{Field: "ssh.management", Reason: "verified SSH transport is not configured for typed cleanup"}
	}
	return p.sshPusher.PushCleanupSSH(ctx, machineID, cmd)
}

// typedSSHPusher is the verified fallback transport. It reuses the Bridge-owned
// machine key and host-key store from onboarding, invokes the stable agent
// launcher with one fixed --cleanup-stdin entrypoint, and persists the returned
// typed events through the same cleanup service callback used by the paired
// channel. The remote command never contains a removal verb or caller argv.
type typedSSHPusher struct {
	machines internalmachines.Service
	ssh      *onboardssh.Service
	report   func(context.Context, internalcleanup.Event) (bool, error)
}

func (p *typedSSHPusher) PushCleanupSSH(ctx context.Context, _ string, cmd internalcleanup.Command) (int, error) {
	if p == nil || p.machines == nil || p.ssh == nil || p.report == nil {
		return 0, internalcleanup.ErrBlocked{Field: "ssh.management", Reason: "typed SSH cleanup transport is not configured"}
	}
	machine, err := p.machines.Get(ctx, cmd.MachineID)
	if err != nil {
		return 0, fmt.Errorf("resolve SSH cleanup machine: %w", err)
	}
	trust, err := p.machines.GetTrust(ctx, cmd.MachineID)
	if err != nil {
		return 0, fmt.Errorf("resolve SSH cleanup trust: %w", err)
	}
	if trust.HostKeyState != internalmachines.HostKeyVerified || trust.ConnectionState != internalmachines.ConnectionTrusted {
		return 0, internalcleanup.ErrBlocked{Field: "ssh.management", Reason: "machine SSH trust is not verified"}
	}
	host := machineHost(machine)
	if host == "" {
		return 0, internalcleanup.ErrBlocked{Field: "reachability", Reason: "machine has no hostname or IP locator"}
	}
	keyName := strings.TrimPrefix(strings.TrimSpace(trust.ClientKeyRef), "ssh-key://")
	if keyName == "" || filepath.Base(keyName) != keyName {
		return 0, internalcleanup.ErrBlocked{Field: "ssh.management", Reason: "machine trust has no Bridge-owned client key"}
	}
	port := trust.SSHPort
	if port == 0 {
		port = 22
	}
	user := strings.TrimSpace(trust.SSHUser)
	if user == "" {
		return 0, internalcleanup.ErrBlocked{Field: "ssh.management", Reason: "machine trust has no SSH user"}
	}
	payload, err := protojson.Marshal(cleanupCommandProto(cmd))
	if err != nil {
		return 0, fmt.Errorf("encode typed SSH cleanup command: %w", err)
	}
	var eventErr error
	cfg := onboardssh.NewConfig(host, port, user, filepath.Join(p.ssh.StateDir(), keyName), p.ssh.KnownHostsPath())
	remote := typedCleanupRemoteCommand()
	result, runErr := p.ssh.RunStreaming(ctx, cfg, remote, onboardssh.StreamOptions{
		Run:   onboardssh.RunOptions{ConnectTimeout: 10 * time.Second, StrictHostKey: true, IdentitiesOnly: true, MaxOutputBytes: 2 * 1024 * 1024, CommandTimeout: 30 * time.Minute},
		Stdin: payload,
		OnStdoutLine: func(line string) {
			if eventErr != nil {
				return
			}
			var wire cleanupv1.CleanupEvent
			if err := protojson.Unmarshal([]byte(line), &wire); err != nil {
				eventErr = fmt.Errorf("decode typed SSH cleanup event: %w", err)
				return
			}
			_, eventErr = p.report(ctx, eventFromProto(&wire))
		},
	})
	for i := range payload {
		payload[i] = 0
	}
	if runErr != nil {
		return 0, fmt.Errorf("run typed SSH cleanup helper: %w", runErr)
	}
	if eventErr != nil {
		return 0, eventErr
	}
	if result.ExitCode != 0 {
		return 0, fmt.Errorf("typed SSH cleanup helper exited %d: %s", result.ExitCode, strings.TrimSpace(result.Stderr))
	}
	return 1, nil
}

func typedCleanupRemoteCommand() string {
	return "sudo -n -- ~/.local/bin/vrooli-bridge-agent --cleanup-stdin --provision-client-home \"$HOME\" --state-dir ~/.local/state/vrooli-bridge-agent --work-dir ~/Vrooli --vrooli-bin ~/.vrooli/bin/vrooli"
}

func cleanupCommandProto(cmd internalcleanup.Command) *channelv1.CleanupCommand {
	return &channelv1.CleanupCommand{
		Operation:         operationToProto(cmd.Operation),
		OpId:              cmd.OpID,
		MachineId:         cmd.MachineID,
		NodeId:            cmd.NodeID,
		Target:            cmd.Target,
		Scope:             cmd.Scope,
		PlanId:            cmd.PlanID,
		PlanHash:          cmd.PlanHash,
		SealedPassphrase:  append([]byte(nil), cmd.SealedPassphrase...),
		Capability:        append([]byte(nil), cmd.Capability...),
		OperatorConfirmed: cmd.OperatorConfirmed,
		OperatorId:        cmd.OperatorID,
	}
}

func machineHost(machine internalmachines.Machine) string {
	for _, locator := range machine.Locators {
		host := strings.TrimSpace(locator.Value)
		switch locator.Kind {
		case "hostname", "ip":
		case "ssh":
			host = strings.TrimPrefix(host, "ssh://")
			if at := strings.LastIndex(host, "@"); at >= 0 {
				host = host[at+1:]
			}
			if parsed, _, err := net.SplitHostPort(host); err == nil {
				host = parsed
			}
			host = strings.Trim(host, "[]")
		default:
			continue
		}
		if host != "" {
			return host
		}
	}
	return ""
}

var (
	_ internalcleanup.CommandPusher    = commandPusher{}
	_ internalcleanup.SSHCommandPusher = commandPusher{}
	_ internalcleanup.NodeReader       = nodeReader{}
	_ internalcleanup.AuditSink        = auditSinkAdapter{}
)

func mapCleanupError(err error) error {
	var invalid internalcleanup.ErrInvalid
	if errors.As(err, &invalid) {
		return connectErrorInvalid(invalid)
	}
	var conflict internalcleanup.ErrConflict
	if errors.As(err, &conflict) {
		return connectErrorFailedPrecondition(conflict)
	}
	var blocked internalcleanup.ErrBlocked
	if errors.As(err, &blocked) {
		return connectErrorUnavailable(blocked)
	}
	var inflight internalcleanup.ErrInFlight
	if errors.As(err, &inflight) {
		return connectErrorAlreadyExists(inflight)
	}
	return err
}
