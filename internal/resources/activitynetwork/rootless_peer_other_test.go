//go:build !linux

package activitynetwork

import "fmt"

func startRootlessPeerUIDProbe(string, string) error {
	return fmt.Errorf("rootless peer credential probe is Linux-only")
}
