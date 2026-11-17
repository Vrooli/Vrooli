//go:build !unix

package gocache

import (
	"errors"
	"os"
)

const protocolSupported = false

var errLocksUnsupported = errors.New("the Go build-cache protocol requires advisory file locks, which this platform does not provide")

func lockShared(*os.File) error            { return errLocksUnsupported }
func lockExclusive(*os.File) error         { return errLocksUnsupported }
func unlock(*os.File) error                { return errLocksUnsupported }
func tryLock(*os.File, bool) (bool, error) { return false, errLocksUnsupported }

func allocatedBytes(info os.FileInfo) int64 { return info.Size() }

func fileIdentity(os.FileInfo) (device, inode uint64, ok bool) { return 0, 0, false }
