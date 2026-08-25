//go:build windows

package config

import (
	"fmt"
	"os"
)

func validateRepairIdentity(uid, gid uint32) error {
	if uid != 0 || gid != 0 {
		return fmt.Errorf("ownership repair is unsupported on windows")
	}
	return nil
}

func invokingRepairIdentity() (uint32, uint32, bool) { return 0, 0, false }

func currentRepairIdentity() (uint32, uint32) { return 0, 0 }

func entryIdentity(path string) (fileIdentity, error) {
	if _, err := os.Lstat(path); err != nil {
		return fileIdentity{}, err
	}
	return fileIdentity{}, nil
}

func repairEntryOwnership(string, uint32, uint32) error {
	return fmt.Errorf("ownership repair is unsupported on windows")
}
