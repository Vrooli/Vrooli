package operatorsession

import (
	"context"
	"strings"
)

// LocalOwnerTokenProvider returns a short-lived owner credential from the
// local enrollment, or an empty credential when this host is not enrolled.
// Consumers can pass it directly to nodereach without duplicating the local
// store and session-minting policy.
func LocalOwnerTokenProvider() func(context.Context) (string, error) {
	return func(context.Context) (string, error) {
		store, err := DefaultFileStore()
		if err != nil {
			return "", nil
		}
		resolution, err := (LocalResolver{Store: store}).Resolve()
		if err != nil || strings.TrimSpace(resolution.Token) == "" {
			return "", nil
		}
		return LocalSessionScheme + " " + resolution.Token, nil
	}
}
