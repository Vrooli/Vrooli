//go:build darwin && cgo

package resources

/*
#include <mach-o/dyld.h>
#include <stdlib.h>
static int vrooli_openrgb_dyld_contains(const char *path) {
#if __MAC_OS_X_VERSION_MAX_ALLOWED >= 110000
 if (__builtin_available(macOS 11.0, *)) return _dyld_shared_cache_contains_path(path) ? 1 : 0;
#endif
 return -1;
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

func openRGBDarwinSharedCacheContains(path string) (bool, error) {
	p := C.CString(path)
	defer C.free(unsafe.Pointer(p))
	result := C.vrooli_openrgb_dyld_contains(p)
	if result < 0 {
		return false, errors.New("native dyld cache metadata requires a qualified macOS SDK/runtime")
	}
	return result == 1, nil
}
