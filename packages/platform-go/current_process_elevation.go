package platform

// CurrentProcessElevated reports whether the current process has native
// elevated privileges. It reports process state only; callers remain
// responsible for authorizing any operation.
func CurrentProcessElevated() (bool, error) {
	return nativeCurrentProcessElevated()
}
