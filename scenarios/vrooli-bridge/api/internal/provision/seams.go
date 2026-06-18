package provision

import "context"

// NodeReader is the registry read seam: provisioning projects a node down to
// the TargetNode it needs (id + revocation). The handler adapter wraps the
// registry service. A missing node surfaces as ErrNodeNotFound.
type NodeReader interface {
	GetTarget(ctx context.Context, id string) (TargetNode, error)
}

// Presence is the live online/offline seam (the presence hub satisfies it).
type Presence interface {
	IsOnline(nodeID string) bool
}

// AuditSink is the append-only accountability seam (the audit store satisfies
// it via the handler adapter). Every provisioning op — accepted or rejected —
// is recorded, because remote provisioning must be reconstructable after the
// fact.
type AuditSink interface {
	Record(ctx context.Context, e Entry) error
}

// Entry is the provision-local audit DTO. Accepted distinguishes a dispatched
// op from a denied one; Detail carries the rejection reason or acceptance note.
type Entry struct {
	Actor            string
	NodeID           string
	TargetRevision   string
	RollbackRevision string
	Accepted         bool
	Detail           string
	OpID             string
}

// CommandPusher is the channel push seam: deliver the privileged
// ProvisionCommand to the node's held dial-out channel. The handler adapter
// translates PushedCommand into a channel.ProvisionCommand ServerFrame and
// calls the presence hub's push. delivered is the number of live connections
// the frame reached (0 means the node dropped between the online check and the
// push).
type CommandPusher interface {
	PushProvision(ctx context.Context, nodeID string, cmd PushedCommand) (delivered int, err error)
}

// PushedCommand is the provision-local DTO for the pushed privileged command
// (proto-free).
type PushedCommand struct {
	OpID             string
	TargetRevision   string
	RollbackRevision string
}
