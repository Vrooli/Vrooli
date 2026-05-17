//go:build !linux && !darwin && !windows

package hwprobe

import "fmt"

func readSystemRAM() (uint64, error) {
	return 0, fmt.Errorf("hwprobe: unsupported OS")
}
