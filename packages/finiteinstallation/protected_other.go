//go:build !linux

package finiteinstallation

import "github.com/vrooli/api-core/effortauthority"

func readProtected(root, name string, owner uint32) ([]byte, error) {
	return nil, effortauthority.ErrRefused
}

func executingBuild() (string, error) { return "", effortauthority.ErrRefused }
