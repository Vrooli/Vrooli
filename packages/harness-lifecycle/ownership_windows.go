//go:build windows

package agentinstall

import "os"

// Windows has no root-owned install case; ACL failures surface when the action runs.
func ownedByRoot(os.FileInfo) bool { return false }

func writableByCaller(string) bool { return true }

func handBackToBinDirOwner(string, ...string) error { return nil }
