//go:build unix || !windows

package infra

import "os"

func assignProcessContainment(_ *os.Process) (func(), error) { return func() {}, nil }
