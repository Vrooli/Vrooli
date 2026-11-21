//go:build !(aix || android || darwin || dragonfly || freebsd || hurd || illumos || linux || netbsd || openbsd || solaris)

package storage

import "fmt"

type ollamaBlobPhysicalInventory struct {
	Count             int64
	LogicalBytes      int64
	PhysicalBytes     int64
	UnrecognizedFiles int64
}

func directoryAllocatedBytes(root string) (int64, error) {
	return 0, fmt.Errorf("allocated-block measurement is unsupported on this platform for %q", root)
}

func inspectOllamaBlobs(root string) (ollamaBlobPhysicalInventory, error) {
	return ollamaBlobPhysicalInventory{}, fmt.Errorf("allocated-block measurement is unsupported on this platform for %q", root)
}
