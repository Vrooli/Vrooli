//go:build darwin

package platform

import (
	"context"
	"os"
	"runtime"
	"syscall"
)

// Darwin has no supported fexecve syscall. This is an explicit native
// equivalent: an immutable root-published pathname, the caller's verified
// vnode, and the complete ACL-checked ancestor chain must agree before exec.
// Privileged publishers must copy to new inodes and never mutate a sealed
// version in place; domain profile ownership still verifies its full closure.
func replaceProcessImage(file *os.File, argv, env []string) error {
	engine := darwinImageExecEngine{check: darwinSystemBrokerCheck, path: verifyImmutableNativePath, format: func(file *os.File) error { return validateDarwinNativeExecutable(file, runtime.GOARCH) }, exec: syscall.Exec}
	return engine.replace(file, argv, env)
}
func verifyImmutableNativePath(ctx context.Context, path string, directory bool) error {
	return verifyDarwinImagePath(ctx, path, directory, darwinSystemBrokerCheck)
}
func frozenOwnedServiceEnvironment(ctx context.Context, ref ServiceRef) ([]string, error) {
	return nativeDarwinOwnedBackend().frozenEnvironment(ctx, OperationRef(ref))
}
