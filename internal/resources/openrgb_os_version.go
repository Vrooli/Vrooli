package resources

import (
	"strconv"
	"strings"
)

func openRGBDarwinVersionAtLeast(value string, minimum uint32) bool {
	parts := strings.Split(value, ".")
	if len(parts) < 1 || len(parts) > 3 || minimum == 0 {
		return false
	}
	var numbers [3]uint64
	for i, part := range parts {
		if part == "" {
			return false
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}
		n, err := strconv.ParseUint(part, 10, 16)
		if err != nil || i > 0 && n > 255 {
			return false
		}
		numbers[i] = n
	}
	packed := uint32(numbers[0])<<16 | uint32(numbers[1])<<8 | uint32(numbers[2])
	return packed >= minimum
}
