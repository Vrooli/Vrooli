package optimization

import "context"

type ManualApplier struct{}

func (ManualApplier) Apply(context.Context, Run, Candidate) (ApplyResult, error) {
	return ApplyResult{Evidence: []string{"Persistent optimization apply needs a configured resolver/router adapter; no live change was made."}}, ErrManualRequired
}

func (ManualApplier) Rollback(context.Context, Run, Candidate) (RollbackResult, error) {
	return RollbackResult{Evidence: []string{"No adapter rollback was executed; follow the saved manual recovery instructions."}}, ErrManualRequired
}
