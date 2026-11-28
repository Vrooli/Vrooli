//go:build darwin && !cgo

package platform

import "os"

func darwinSystemBrokerACL(string, os.FileInfo) error { return ErrUnsupported }

func darwinOperatorPathACL(string, os.FileInfo) error { return ErrUnsupported }
