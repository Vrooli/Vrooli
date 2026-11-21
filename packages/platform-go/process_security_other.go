//go:build !darwin && !windows

package platform

func nativeProcessSecurity(int) (ProcessSecurityContext, error) {
	return ProcessSecurityContext{}, ErrUnsupported
}
