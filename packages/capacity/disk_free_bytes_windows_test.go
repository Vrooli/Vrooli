//go:build windows

package capacity

import (
	"testing"

	"golang.org/x/sys/windows"
)

func freeDiskBytesForTest(t *testing.T, path string) int64 {
	t.Helper()
	utf16Path, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	var available, total, free uint64
	if err := windows.GetDiskFreeSpaceEx(utf16Path, &available, &total, &free); err != nil {
		t.Fatal(err)
	}
	return int64(available)
}
