package driverid

import (
	"fmt"
	"strings"
)

// ID is the canonical driver identifier across DB, wire payloads,
// preference files, launcher decisions, and driver implementations.
type ID string

const (
	OverlayfsUserNS ID = "overlayfs-userns"
	OverlayfsRoot   ID = "overlayfs-root"
	FuseOverlayfs   ID = "fuse-overlayfs"
	Copy            ID = "copy"
)

// Known reports whether id is one of the supported canonical driver IDs.
func Known(id ID) bool {
	switch id {
	case OverlayfsUserNS, OverlayfsRoot, FuseOverlayfs, Copy:
		return true
	default:
		return false
	}
}

// Parse validates and returns a canonical driver ID.
func Parse(value string) (ID, error) {
	id := ID(strings.TrimSpace(value))
	if id == "" {
		return "", fmt.Errorf("driver ID is required")
	}
	if !Known(id) {
		return "", fmt.Errorf("unknown driver ID: %s", id)
	}
	return id, nil
}
