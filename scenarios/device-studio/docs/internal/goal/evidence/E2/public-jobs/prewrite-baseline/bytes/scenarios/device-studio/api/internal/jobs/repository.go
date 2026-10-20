package jobs

import "context"

// Repository is the engine-neutral persistence seam. Each mutating method is
// one durable, serialized transition; implementations must compare the
// current state/version and commit all related attempt/job receipts atomically.
type Repository interface {
	Submit(context.Context, Job) (Job, bool, error)
	Get(context.Context, string) (Job, error)
	ListNonterminal(context.Context) ([]Job, error)
	Start(context.Context, ExecutionClaim, int64, int64) (Job, error) // claim, now, expected version
	Heartbeat(context.Context, string, string, int64, int64) (Job, error)
	Checkpoint(context.Context, string, string, Checkpoint, QuiescenceProof, int64) (Job, error)
	Complete(context.Context, string, string, CompletionEvidence, int64) (Job, error)
	Fail(context.Context, string, string, string, int64, int64) (Job, error)
	RequestCancel(context.Context, string, int64, int64) (Job, error)
	ConfirmCancelled(context.Context, string, string, QuiescenceProof, int64) (Job, error)
	Settle(context.Context, string, SettlementState, int64, int64) (Job, error)
}
