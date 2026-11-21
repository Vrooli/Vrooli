package platform

import (
	"context"
	"errors"
	"strings"
)

// PrepareOwnedServicePeerObservation verifies native peer-identity visibility
// for the workload account. Windows actual-operator peers require no ACL grant;
// account-specific native backends own any narrow observation setup. The caller
// retains and invokes cleanup for its endpoint lifetime.
func PrepareOwnedServicePeerObservation(ctx context.Context, account string) (func() error, error) {
	if ctx == nil || account == "" || strings.ContainsAny(account, "\x00\r\n") {
		return nil, errors.New("platform: peer observation requires context and exact service account")
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	return prepareOwnedServicePeerObservation(ctx, account)
}
