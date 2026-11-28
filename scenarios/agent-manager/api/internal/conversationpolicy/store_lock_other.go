//go:build windows

package conversationpolicy

// lockPolicyFile is a no-op on platforms without flock(2). The policy CLI is an
// operator-driven, low-concurrency command and the host that serves the policy
// is Unix; a desktop cross-build must still compile, so the lock degrades to a
// no-op here rather than pulling a Windows locking dependency.
func lockPolicyFile(path string) (func(), error) {
	return func() {}, nil
}
