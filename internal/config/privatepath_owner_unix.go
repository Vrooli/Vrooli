//go:build !windows

package config

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func nativePrivatePathOwner(_ string, info os.FileInfo) (string, error) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st == nil {
		return "", errors.New("private path has no native owner")
	}
	return fmt.Sprintf("uid:%d", st.Uid), nil
}
