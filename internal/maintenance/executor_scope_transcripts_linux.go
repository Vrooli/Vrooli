//go:build linux

package maintenance

import (
	"os"
	"strconv"
)

// readProcessOpenFiles resolves the descriptor links of one same-user
// process. Another user's table is unreadable and returns an error, which the
// caller treats as no claim. At most maxFDs descriptors are resolved.
func readProcessOpenFiles(pid int) ([]string, error) {
	const maxFDs = 65536
	dir := "/proc/" + strconv.Itoa(pid) + "/fd"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	targets := make([]string, 0, min(len(entries), maxFDs))
	for i, entry := range entries {
		if i >= maxFDs {
			break
		}
		target, err := os.Readlink(dir + "/" + entry.Name())
		if err != nil || len(target) == 0 || target[0] != '/' {
			continue
		}
		targets = append(targets, target)
	}
	return targets, nil
}
