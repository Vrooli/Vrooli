//go:build linux

package resources

import "golang.org/x/sys/unix"

func readAddressSpaceLimit(pid int) ([2]uint64, error) {
	var limit unix.Rlimit
	if err := unix.Prlimit(pid, unix.RLIMIT_AS, nil, &limit); err != nil {
		return [2]uint64{}, err
	}
	return [2]uint64{limit.Cur, limit.Max}, nil
}
