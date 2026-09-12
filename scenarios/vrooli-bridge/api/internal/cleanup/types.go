// Package cleanup owns the durable, operator-confirmed cleanup operation.
// It deliberately contains no filesystem or host-remediation code: discovery
// and application happen in the node's privilege-separated helper through a
// typed command, while this package owns identity, leases, frozen artifacts,
// audit, and resumable receipts.
package cleanup

import (
	"context"
	"fmt"
	"time"
)

type Status int

const (
	StatusUnspecified Status = iota
	StatusQueued
	StatusPlanning
	StatusPlanned
	StatusConfirmed
	StatusApplying
	StatusCompleted
	StatusFailed
	StatusBlocked
	StatusCanceled
)

func (s Status) Terminal() bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusBlocked || s == StatusCanceled
}

func (s Status) String() string {
	switch s {
	case StatusQueued:
		return "queued"
	case StatusPlanning:
		return "planning"
	case StatusPlanned:
		return "planned"
	case StatusConfirmed:
		return "confirmed"
	case StatusApplying:
		return "applying"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusBlocked:
		return "blocked"
	case StatusCanceled:
		return "canceled"
	default:
		return "unspecified"
	}
}

type EventKind int

const (
	EventUnspecified EventKind = iota
	EventStatus
	EventLog
	EventPlan
	EventReceipt
	EventExit
)

type Operation struct {
	ID              string
	MachineID       string
	NodeID          string
	Target          string
	Scope           string
	Status          Status
	Transport       string
	TransportReason string
	Reason          string
	PlanHash        string
	PlanJSON        []byte
	ReceiptJSON     []byte
	OperatorID      string
	// These are opaque, node-bound envelopes. Keeping ciphertext makes a
	// confirmed operation resumable without retaining plaintext in the control
	// plane or asking the operator to retype a secret after a transport fault.
	SealedPassphrase []byte
	Capability       []byte
	SealingPublicKey []byte
	CreatedAt        time.Time
	UpdatedAt        time.Time
	FinishedAt       time.Time
}

type Event struct {
	OperationID string
	Kind        EventKind
	Sequence    uint64
	Status      string
	LogChunk    string
	PlanJSON    []byte
	ReceiptJSON []byte
	Reason      string
	ExitCode    int32
	EmittedAt   time.Time
}

type StartInput struct {
	MachineID string
	NodeID    string
	Target    string
	Scope     string
}

type Target struct {
	MachineID        string
	NodeID           string
	Target           string
	Scope            string
	Transport        string
	TransportReason  string
	OperatorID       string
	OperationID      string
	SealingPublicKey []byte
	Capabilities     []string
	ApprovedScopes   []string
}

type ProvisionInput struct {
	MachineID        string
	NodeID           string
	Target           string
	Scope            string
	OperationID      string
	SealedPassphrase []byte
	OperatorID       string
}

// ResetInput retires only the node-local managed break-glass material. It is a
// recovery operation for abandoned protection state; it never authorizes or
// applies cleanup.
type ResetInput struct {
	MachineID string
	NodeID    string
	Target    string
	Scope     string
}

type ConfirmInput struct {
	ID               string
	Target           string
	PlanHash         string
	SealedPassphrase []byte
	Capability       []byte
	OperatorID       string
}

type Command struct {
	Operation         string
	OpID              string
	MachineID         string
	NodeID            string
	Target            string
	Scope             string
	PlanID            string
	PlanHash          string
	SealedPassphrase  []byte
	Capability        []byte
	OperatorConfirmed bool
	OperatorID        string
}

type NodeReader interface {
	GetTarget(context.Context, string) (TargetNode, error)
}

type TargetNode struct {
	ID               string
	Kind             string
	Revoked          bool
	SealingPublicKey []byte
	Capabilities     []string
	Scopes           []string
	Endpoint         string
}

type Transport string

const (
	TransportAgent Transport = "agent"
	TransportSSH   Transport = "ssh"
)

type TransportFacts struct {
	AgentOnline      bool
	SSHManagement    bool
	SSHScopeApproved bool
	TargetReachable  bool
}

type TransportSelection struct {
	Transport Transport
	Reason    string
}

// SelectTransport is pure policy. It never repairs a provider or executes a
// command; callers must separately supply the selected typed transport.
func SelectTransport(facts TransportFacts) (TransportSelection, error) {
	if facts.AgentOnline {
		return TransportSelection{Transport: TransportAgent, Reason: "paired agent is online"}, nil
	}
	if !facts.TargetReachable {
		return TransportSelection{}, ErrBlocked{Field: "reachability", Reason: "target is unreachable"}
	}
	if facts.SSHManagement && facts.SSHScopeApproved {
		return TransportSelection{Transport: TransportSSH, Reason: "paired agent is offline; approved SSH management is available"}, nil
	}
	if !facts.SSHManagement {
		return TransportSelection{}, ErrBlocked{Field: "ssh.management", Reason: "verified SSH management capability is missing"}
	}
	return TransportSelection{}, ErrBlocked{Field: "ssh.management", Reason: "SSH management scope is not approved"}
}

type Presence interface {
	IsOnline(string) bool
}

type CommandPusher interface {
	PushCleanup(context.Context, string, Command) (int, error)
}

// SSHCommandPusher is optional until a node's verified SSH transport is
// configured. It carries the same typed command; it never accepts a shell
// string or a raw removal command.
type SSHCommandPusher interface {
	PushCleanupSSH(context.Context, string, Command) (int, error)
}

type AuditSink interface {
	Record(context.Context, AuditEntry) error
}

type AuditEntry struct {
	Actor       string
	NodeID      string
	OperationID string
	Verb        string
	Outcome     string
	Detail      string
}

type Repository interface {
	Create(context.Context, Operation) (Operation, error)
	Get(context.Context, string) (Operation, error)
	Update(context.Context, Operation) (Operation, error)
	AppendEvent(context.Context, Event) (bool, error)
	ListEvents(context.Context, string) ([]Event, error)
}

type Service interface {
	Prepare(context.Context, StartInput, string) (Target, error)
	ProvisionBreakGlass(context.Context, ProvisionInput) (Operation, error)
	ResetBreakGlass(context.Context, ResetInput, string) (Operation, error)
	Start(context.Context, StartInput, string) (Operation, error)
	Get(context.Context, string) (Operation, []Event, error)
	Plan(context.Context, string) (Operation, error)
	Confirm(context.Context, ConfirmInput) (Operation, error)
	Apply(context.Context, string) (Operation, error)
	Verify(context.Context, string) (Operation, error)
	Cancel(context.Context, string, string) (Operation, error)
	AppendEvent(context.Context, Event) (bool, error)
	Wait(context.Context, string, time.Duration) (Operation, bool, error)
	Subscribe(string) (<-chan Event, func())
}

type ErrInvalid struct{ Field, Reason string }

func (e ErrInvalid) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("cleanup operation %q not found", e.ID) }

type ErrInFlight struct{ MachineID, OperationID string }

func (e ErrInFlight) Error() string {
	return fmt.Sprintf("cleanup already in flight for machine %q (operation %q)", e.MachineID, e.OperationID)
}

type ErrBlocked struct{ Field, Reason string }

func (e ErrBlocked) Error() string { return fmt.Sprintf("cleanup blocked: %s: %s", e.Field, e.Reason) }

type ErrConflict struct{ Field, Reason string }

func (e ErrConflict) Error() string {
	return fmt.Sprintf("cleanup conflict: %s: %s", e.Field, e.Reason)
}
