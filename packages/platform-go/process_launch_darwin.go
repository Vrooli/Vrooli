//go:build darwin

package platform

import "os"

func processArguments(pid int) ([]string, error) {
	block, err := readProcArgs(pid)
	if err != nil {
		return nil, err
	}
	return block.Args, nil
}

func openProcessExecutable(int) (*os.File, error) { return nil, ErrUnsupported }
