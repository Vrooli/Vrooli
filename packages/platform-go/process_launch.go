package platform

import (
	"bytes"
	"errors"
	"os"
)

const (
	maxProcessArgumentBytes = 1 << 20
	maxProcessArguments     = 4096
)

// ProcessArguments returns the observed argv, including argv[0], preserving
// argument boundaries and empty arguments. Flattened command lines cannot
// establish a launch contract. Values may contain credentials: compare them
// in memory and never put the returned arguments in receipts or diagnostics.
// Callers must separately bind the observation to the process incarnation.
func ProcessArguments(pid int) ([]string, error) { return processArguments(pid) }

// OpenProcessExecutable opens the running executable image where the native
// backend can retain that exact file. Opening its pathname instead could read
// replacement bytes. The caller closes the file and separately verifies the
// process incarnation before/after its observation. Unsupported backends fail
// closed rather than opening a possibly replaced pathname.
func OpenProcessExecutable(pid int) (*os.File, error) { return openProcessExecutable(pid) }

func parseNULProcessArguments(data []byte) ([]string, error) {
	if len(data) == 0 || len(data) > maxProcessArgumentBytes || data[len(data)-1] != 0 {
		return nil, errors.New("platform: process argument block is missing, truncated or exceeds its bound")
	}
	if data[0] == 0 || bytes.Count(data, []byte{0}) > maxProcessArguments {
		return nil, errors.New("platform: process argument block has no executable or exceeds its argument count")
	}
	parts := bytes.Split(data[:len(data)-1], []byte{0})
	args := make([]string, len(parts))
	for i, part := range parts {
		args[i] = string(part)
	}
	return args, nil
}
