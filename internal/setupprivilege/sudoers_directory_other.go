//go:build !linux

package setupprivilege

import "errors"

func ensureSudoersDirectory() error {
	return errors.New("onboarding sudoers grants are supported only on Linux")
}
