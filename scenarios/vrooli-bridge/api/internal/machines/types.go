package machines

import (
	"fmt"
	"strings"
	"time"
)

type Lifecycle string

const (
	LifecycleActive   Lifecycle = "active"
	LifecycleArchived Lifecycle = "archived"
	LifecycleRemoved  Lifecycle = "removed"
)

type Locator struct {
	Kind    string
	Value   string
	Ordinal int
}
type NodeLineage struct {
	NodeID        string
	Current       bool
	LinkedAt      time.Time
	SupersededAt  time.Time
	CorrelationID string
}
type Machine struct {
	ID                    string
	Lifecycle             Lifecycle
	Version               int64
	DesiredProfileID      string
	DesiredProfileVersion string
	TrustRef              string
	Locators              []Locator
	Lineage               []NodeLineage
	CreatedAt             time.Time
	UpdatedAt             time.Time
	ArchivedAt            time.Time
	RemovedAt             time.Time
}
type MigrationReview struct {
	ID           string
	LegacySource string
	LegacyID     string
	Status       string
	Confidence   string
	Reason       string
	CreatedAt    time.Time
	ReviewedAt   time.Time
}
type CleanupStatus string

const (
	CleanupPending       CleanupStatus = "pending"
	CleanupConfirmed     CleanupStatus = "confirmed"
	CleanupNotApplicable CleanupStatus = "not_applicable"
	CleanupAbandoned     CleanupStatus = "abandoned_with_acknowledgement"
)

type CleanupTombstone struct {
	ID             string
	MachineID      string
	Action         string
	Status         CleanupStatus
	Detail         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	AcknowledgedAt time.Time
}

// AuditEvent is append-only evidence for an operator-visible Machine effect.
// It never carries private keys, SSH passwords, pairing codes, or raw commands.
type AuditEvent struct {
	ID        string
	MachineID string
	Action    string
	Actor     string
	Detail    string
	CreatedAt time.Time
}
type HostKeyState string

const (
	HostKeyUnverified     HostKeyState = "unverified"
	HostKeyVerified       HostKeyState = "verified"
	HostKeyReviewRequired HostKeyState = "review_required"
)

// ConnectionState is the durable SSH trust lifecycle used by Bridge's
// one-command connect flow. It is deliberately separate from host-key state:
// a host key can be verified while the Bridge client key is missing or
// untrusted.
type ConnectionState string

const (
	ConnectionUntrusted ConnectionState = "untrusted"
	ConnectionTrusted   ConnectionState = "trusted"
	ConnectionRecovery  ConnectionState = "recovery_required"
)

type TrustRecord struct {
	MachineID            string
	ClientKeyRef         string
	ClientKeyFingerprint string
	HostKeyFingerprint   string
	HostKeyState         HostKeyState
	SSHUser              string
	SSHPort              int
	ConnectionState      ConnectionState
	UpdatedAt            time.Time
}
type CreateInput struct {
	ID                    string
	Locators              []Locator
	DesiredProfileID      string
	DesiredProfileVersion string
	TrustRef              string
}

// IdentityQuery is the ordered evidence used to reconnect an enrollment to
// an existing Machine. Callers should provide every fact they have; Resolve
// applies the documented strength order and never guesses across conflicting
// evidence.
type IdentityQuery struct {
	MachineID             string
	NodeID                string
	SSHHostKeyFingerprint string
	Hostname              string
}

type MergeInput struct {
	FromMachineID string
	IntoMachineID string
	Actor         string
}

// PolicyChangeInput is an explicit, optimistic-concurrency policy decision.
// Profiles can suggest setup/scopes but cannot grant Registry authorization.
type PolicyChangeInput struct {
	MachineID       string
	ExpectedVersion int64
	ProfileID       string
	ProfileVersion  string
	Overrides       map[string]string
	Actor           string
	Reason          string
	ConfirmRemoval  bool
}
type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("machine %q not found", e.ID) }

type ErrConflict struct {
	ID      string
	Version int64
}

func (e ErrConflict) Error() string { return fmt.Sprintf("machine %q version conflict", e.ID) }

type ErrInvalid struct{ Field, Reason string }

func (e ErrInvalid) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

type ErrAmbiguous struct{ Evidence string }

func (e ErrAmbiguous) Error() string {
	return fmt.Sprintf("machine identity is ambiguous for %s; use an explicit machine id or machines merge", e.Evidence)
}

func normalizeLocator(kind, value string) (string, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	v := strings.TrimSpace(value)
	if kind == "" {
		return "", ErrInvalid{"locator.kind", "required"}
	}
	if v == "" {
		return "", ErrInvalid{"locator.value", "required"}
	}
	switch kind {
	case "hostname":
		return strings.ToLower(strings.TrimSuffix(v, ".")), nil
	case "ip":
		return strings.ToLower(v), nil
	case "ssh":
		return strings.ToLower(v), nil
	case "ssh-host-key":
		return v, nil
	default:
		return "", ErrInvalid{"locator.kind", "must be hostname, ip, ssh, or ssh-host-key"}
	}
}
