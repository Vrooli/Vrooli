//go:build darwin && !cgo

package platform

func darwinManagerSnapshot(bool, string, string) (darwinJobSnapshot, bool, error) {
	return darwinJobSnapshot{}, false, ErrUnsupported
}
