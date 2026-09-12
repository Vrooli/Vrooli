//go:build windows

package androidsdk

import "syscall"

func androidSDKDetachedProcessAttrs() *syscall.SysProcAttr {
	return nil
}
