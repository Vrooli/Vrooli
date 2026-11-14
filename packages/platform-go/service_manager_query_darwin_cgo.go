//go:build darwin && cgo

package platform

/*
#cgo LDFLAGS: -framework ServiceManagement -framework CoreFoundation
#include <ServiceManagement/ServiceManagement.h>
#include <CoreFoundation/CoreFoundation.h>
#include <errno.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
// ServiceManagement is deprecated but remains an SDK API. Its dictionaries are
// not a copy of the on-disk plist. Missing or changed required facts are unknown,
// not permission to parse launchctl's explicitly non-API print diagnostics.
static int vrooli_manager_query(int system, const char *label, const char *expected, int *present, int *pid) {
 *present=0;*pid=0;
 CFStringRef name=CFStringCreateWithCString(NULL,label,kCFStringEncodingUTF8);
 if(!name)return ENOMEM;
 #pragma clang diagnostic push
 #pragma clang diagnostic ignored "-Wdeprecated-declarations"
 CFArrayRef jobs=SMCopyAllJobDictionaries(system?kSMDomainSystemLaunchd:kSMDomainUserLaunchd);
 #pragma clang diagnostic pop
 if(!jobs){CFRelease(name);return EIO;}
 int result=0;
 if(CFGetTypeID(jobs)!=CFArrayGetTypeID()||CFArrayGetCount(jobs)>65536){result=EPROTO;goto done;}
 for(CFIndex i=0;i<CFArrayGetCount(jobs);i++) {
 CFDictionaryRef job=CFArrayGetValueAtIndex(jobs,i);
 if(!job||CFGetTypeID(job)!=CFDictionaryGetTypeID()){result=EPROTO;goto done;}
 CFTypeRef found=CFDictionaryGetValue(job,CFSTR("Label"));
 if(!found||CFGetTypeID(found)!=CFStringGetTypeID()){result=EPROTO;goto done;}
 if(!CFEqual(name,found))continue;
 if(*present){result=EPROTO;goto done;}*present=1;
 if(expected&&expected[0]) {
 CFDataRef bytes=CFDataCreate(NULL,(const UInt8*)expected,strlen(expected));
 CFPropertyListRef frozen=bytes?CFPropertyListCreateWithData(NULL,bytes,kCFPropertyListImmutable,NULL,NULL):NULL;
 if(bytes)CFRelease(bytes);
 if(!frozen||CFGetTypeID(frozen)!=CFDictionaryGetTypeID()){if(frozen)CFRelease(frozen);result=EINVAL;goto done;}
 // These are published launchd.plist keys. Compare exact configured identity;
 // defaults, timestamps and undocumented diagnostic keys convey no authority.
 // Apple job_export does not promise env/cwd/credentials in these dictionaries.
 // Those observations come from the kernel at the boot/resource image owner.
 const CFStringRef keys[]={CFSTR("Label"),CFSTR("ProgramArguments")};
 for(unsigned int k=0;k<sizeof(keys)/sizeof(keys[0]);k++) {
 CFTypeRef want=CFDictionaryGetValue((CFDictionaryRef)frozen,keys[k]);
 CFTypeRef actual=CFDictionaryGetValue(job,keys[k]);
 if(want&&(!actual||!CFEqual(want,actual))){result=ESTALE;break;}
 }
 CFRelease(frozen);if(result)goto done;
 }
 CFTypeRef process=CFDictionaryGetValue(job,CFSTR("PID"));
 if(process){int64_t value=0;if(CFGetTypeID(process)!=CFNumberGetTypeID()||!CFNumberGetValue(process,kCFNumberSInt64Type,&value)||value<=0||value>INT32_MAX){result=EPROTO;goto done;}*pid=(int)value;}
 }
 done:CFRelease(jobs);CFRelease(name);return result;
}
*/
import "C"

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

func darwinManagerSnapshot(system bool, label, expected string) (darwinJobSnapshot, bool, error) {
	if label == "" || len(label) > 255 || strings.ContainsAny(label, "\x00/\r\n") || len(expected) > darwinOwnedRecordLimit || strings.ContainsRune(expected, 0) {
		return darwinJobSnapshot{}, false, ErrOperationIdentity
	}
	name := C.CString(label)
	defer C.free(unsafe.Pointer(name))
	frozen := C.CString(expected)
	defer C.free(unsafe.Pointer(frozen))
	var present, pid C.int
	sys := C.int(0)
	if system {
		sys = 1
	}
	if e := C.vrooli_manager_query(sys, name, frozen, &present, &pid); e != 0 {
		return darwinJobSnapshot{}, false, fmt.Errorf("platform: native launchd dictionary query unavailable or inconsistent: %w", syscall.Errno(e))
	}
	p := darwinJobSnapshot{PID: int(pid), State: "not running"}
	if p.PID > 0 {
		p.State = "running"
	}
	return p, present != 0, nil
}
