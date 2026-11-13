package registry

import (
	"context"
	"errors"
)

// providerDescriptionWriteGate serializes Search Hub registry writes with the
// brief embedding-index cutover window. This is an in-process owner fence for
// the scenario's registered writer path; direct database or external writers
// remain outside this boundary and are detected through the durable change
// feed where possible.
var providerDescriptionWriteGate = func() chan struct{} {
	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	return gate
}()

// AcquireProviderDescriptionWrite grants one registry mutation or an exclusive
// source-owner cutover. The returned release function must be called once.
func AcquireProviderDescriptionWrite(ctx context.Context) (func(), error) {
	if ctx == nil {
		return nil, errors.New("provider description write gate requires a context")
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-providerDescriptionWriteGate:
	}
	return func() { providerDescriptionWriteGate <- struct{}{} }, nil
}
