//go:build darwin && cgo

package platform

/*
#cgo LDFLAGS: -lsandbox
#include <sandbox.h>
#include <dlfcn.h>
#include <unistd.h>
#include <stdlib.h>
#include <errno.h>
// Query the inherited kernel process-fork rule used by fork/spawn. This SPI
// has no SDK availability guarantee; missing symbols/unknown replies fail
// closed. NONE=0 is the operation's unfiltered check.
typedef int (*vrooli_check_fn)(pid_t,const char*,int,...);
static int vrooli_nochildren_check(int pid,const char*op){
 vrooli_check_fn check=(vrooli_check_fn)dlsym(RTLD_DEFAULT,"sandbox_check");
 if(!check)return -ENOTSUP;return check(pid,op,0);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func init() {
	verifyDarwinNoChildren = verifyDarwinNoChildrenNative
}
func verifyDarwinNoChildrenNative(pid int) error {
	before, e := darwinObservationStamp(pid)
	if e != nil {
		return e
	}
	for _, op := range []string{"process-fork"} {
		name := C.CString(op)
		r := int(C.vrooli_nochildren_check(C.int(pid), name))
		C.free(unsafe.Pointer(name))
		if r != 1 {
			return fmt.Errorf("%w: native no-children policy %s unverified (%d)", ErrOperationIdentity, op, r)
		}
	}
	after, e := darwinObservationStamp(pid)
	if e != nil || before != after {
		return ErrOperationIdentity
	}
	return nil
}
