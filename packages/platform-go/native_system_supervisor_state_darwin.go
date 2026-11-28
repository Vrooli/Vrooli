//go:build darwin

package platform

import (
	"context"
	"os"
)

func nativeSystemSupervisorStateRoot(ctx context.Context) (string, error) {
	if os.Getuid() == 0 || os.Getuid() != os.Geteuid() || len(os.Args) != 3 || os.Args[1] != NativeSystemSupervisorVerb {
		return "", ErrOperationIdentity
	}
	o, err := nativeDarwinSystemBrokerEngine().runtime(ctx, os.Args[2], os.Getpid())
	if err != nil {
		return "", err
	}
	return o.OperatorHome, nil
}
