package journal

import "context"

// Projector derives holder balances from events. Rebuild only replaces the
// disposable projection cache; event rows remain untouched.
type Projector interface {
	BalanceAt(context.Context, string, string) (Balance, error)
	Rebuild(context.Context) error
}
