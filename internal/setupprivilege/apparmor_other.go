//go:build !linux

package setupprivilege

import (
	"context"
	"errors"
)

func RunSystemAppArmorProfile(context.Context, string) error {
	return errors.New("AppArmor profiles are supported only on Linux")
}
