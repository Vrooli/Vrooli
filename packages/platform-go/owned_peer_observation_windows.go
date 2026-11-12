//go:build windows

package platform

import (
	"context"
	"golang.org/x/sys/windows"
)

// Peers share the actual operator's standard token. No virtual-account process
// ACL grant, token/DACL mutation or reference-counted privileged roster exists.
func prepareOwnedServicePeerObservation(ctx context.Context, account string) (func() error, error) {
	if _, e := resolveOwnedServiceAccount(ctx, account); e != nil {
		return nil, e
	}
	if _, e := windowsProcessOperatorSecurityProof(windows.CurrentProcess(), account); e != nil {
		return nil, e
	}
	return func() error { return nil }, nil
}
