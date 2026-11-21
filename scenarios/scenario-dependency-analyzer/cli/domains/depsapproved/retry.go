package depsapproved

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
)

// A restarting API closes connections mid-request, which reaches the CLI as
// "unavailable: unexpected EOF" (or connection refused) from a call that was
// fine a moment earlier. A read-only call is safe to repeat, so the client
// repeats it a bounded number of times instead of handing the agent a failure
// that a retry always cleared. Mutating calls are never repeated: an install
// or upsert that may have been applied must be reported, not replayed.
const readRetryAttempts = 3

var readRetryBackoff = []time.Duration{2 * time.Second, 5 * time.Second}

// readOnlyProcedure reports whether a governance RPC only reads state.
func readOnlyProcedure(procedure string) bool {
	name := procedure[strings.LastIndex(procedure, "/")+1:]
	for _, prefix := range []string{"List", "Search", "Explain", "Validate", "Get", "Preview"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// retryTransient runs call, and for a read-only procedure repeats it when the
// API reports itself unavailable. sleep and note are seams for tests.
func retryTransient(ctx context.Context, procedure string, call func() error, sleep func(time.Duration), note func(string)) error {
	attempts := 1
	if readOnlyProcedure(procedure) {
		attempts = readRetryAttempts
	}
	var err error
	for attempt := 1; ; attempt++ {
		err = call()
		if err == nil || attempt >= attempts || connect.CodeOf(err) != connect.CodeUnavailable || ctx.Err() != nil {
			return err
		}
		wait := readRetryBackoff[min(attempt-1, len(readRetryBackoff)-1)]
		note(fmt.Sprintf("dependency governance API unavailable (%v); retrying %s in %s (attempt %d of %d)", errors.Unwrap(err), procedure[strings.LastIndex(procedure, "/")+1:], wait, attempt+1, attempts))
		sleep(wait)
	}
}

func retryReadOnlyInterceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error) {
			err = retryTransient(ctx, req.Spec().Procedure, func() error {
				resp, err = next(ctx, req)
				return err
			}, time.Sleep, func(message string) { fmt.Fprintln(os.Stderr, message) })
			return resp, err
		}
	})
}
