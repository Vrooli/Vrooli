//go:build !windows

package config

import (
	"fmt"
	"os"
	"syscall"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func validateRepairIdentity(uid, gid uint32) error {
	if invokingUID, invokingGID, ok := invokingRepairIdentity(); ok {
		if uid != invokingUID || gid != invokingGID {
			return fmt.Errorf("repair identity must match invoking user")
		}
		return nil
	}
	if uid != uint32(os.Getuid()) || gid != uint32(os.Getgid()) {
		return fmt.Errorf("repair identity must match current user")
	}
	return nil
}

func invokingRepairIdentity() (uint32, uint32, bool) {
	uid, gid, ok := hostreqkit.InvokingUserIDs()
	return uint32(uid), uint32(gid), ok
}

func currentRepairIdentity() (uint32, uint32) {
	return uint32(os.Getuid()), uint32(os.Getgid())
}

func entryIdentity(path string) (fileIdentity, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return fileIdentity{}, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fileIdentity{}, fmt.Errorf("unsupported file identity")
	}
	return fileIdentity{UID: uint32(stat.Uid), GID: uint32(stat.Gid)}, nil
}

func repairEntryOwnership(path string, uid, gid uint32) error {
	return os.Lchown(path, int(uid), int(gid))
}
