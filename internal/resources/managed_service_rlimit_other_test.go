//go:build !linux

package resources

func readAddressSpaceLimit(int) ([2]uint64, error) {
	return [2]uint64{}, nil
}
