//go:build darwin && cgo

package hostreqkit

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#cgo LDFLAGS: -lsandbox
#include <IOKit/IOKitLib.h>
#include <IOKit/hid/IOHIDLib.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <sandbox.h>

typedef struct { char path[1024]; uint64_t birth; int vendor, product, page, usage; } vrooli_hid_node;
static int vrooli_hid_number(IOHIDDeviceRef dev, CFStringRef key) {
 CFTypeRef value=IOHIDDeviceGetProperty(dev,key); int n=0;
 if(value && CFGetTypeID(value)==CFNumberGetTypeID()) CFNumberGetValue((CFNumberRef)value,kCFNumberIntType,&n);
 return n;
}
static int vrooli_hid_list(vrooli_hid_node *out,int cap) {
 io_iterator_t it=0; if(IOServiceGetMatchingServices(kIOMainPortDefault,IOServiceMatching("IOHIDDevice"),&it)!=KERN_SUCCESS) return -1;
 io_service_t entry; int n=0;
 while((entry=IOIteratorNext(it))) {
  IOHIDDeviceRef dev=IOHIDDeviceCreate(kCFAllocatorDefault,entry);
  if(dev) {
   int vid=vrooli_hid_number(dev,CFSTR("VendorID")),pid=vrooli_hid_number(dev,CFSTR("ProductID"));
   if(vid==0x0b05 && (pid==0x18f3 || pid==0x1939 || pid==0x19af || pid==0x1aa6 || pid==0x1bed || pid==0x1889)) {
    if(n==cap){CFRelease(dev);IOObjectRelease(entry);IOObjectRelease(it);return -2;}
    vrooli_hid_node *p=&out[n]; p->vendor=vid;p->product=pid;
    p->page=vrooli_hid_number(dev,CFSTR("PrimaryUsagePage"));p->usage=vrooli_hid_number(dev,CFSTR("PrimaryUsage"));
    if(IORegistryEntryGetPath(entry,kIOServicePlane,p->path)!=KERN_SUCCESS || IORegistryEntryGetRegistryEntryID(entry,&p->birth)!=KERN_SUCCESS){CFRelease(dev);IOObjectRelease(entry);IOObjectRelease(it);return -3;}
    n++;
   }
   CFRelease(dev);
  }
  IOObjectRelease(entry);
 }
 IOObjectRelease(it);return n;
}
static int vrooli_hid_access(const char *path,uint64_t birth) {
 io_registry_entry_t entry=IORegistryEntryFromPath(kIOMainPortDefault,path); if(!entry)return -1;
 uint64_t actual=0; if(IORegistryEntryGetRegistryEntryID(entry,&actual)!=KERN_SUCCESS || actual!=birth){IOObjectRelease(entry);return -2;}
 IOHIDDeviceRef dev=IOHIDDeviceCreate(kCFAllocatorDefault,entry);IOObjectRelease(entry);if(!dev)return -3;
 IOReturn ret=IOHIDDeviceOpen(dev,kIOHIDOptionsTypeNone);
 if(ret==kIOReturnSuccess)ret=IOHIDDeviceClose(dev,kIOHIDOptionsTypeNone);
 CFRelease(dev);return ret;
}
static int vrooli_machine_uuid(char *out,int cap) {
 io_service_t service=IOServiceGetMatchingService(kIOMainPortDefault,IOServiceMatching("IOPlatformExpertDevice"));if(!service)return -1;
 CFTypeRef uuid=IORegistryEntryCreateCFProperty(service,CFSTR("IOPlatformUUID"),kCFAllocatorDefault,0);IOObjectRelease(service);
 if(!uuid)return -2;
 int ok=CFGetTypeID(uuid)==CFStringGetTypeID() && CFStringGetCString((CFStringRef)uuid,out,cap,kCFStringEncodingUTF8);CFRelease(uuid);return ok?0:-3;
}
static int vrooli_hid_transport_sandbox(const char *profile) {
 char *error=NULL;int ret=sandbox_init(profile,0,&error);if(error)sandbox_free_error(error);if(ret!=0)return -1;
 // This checks only a denied open after installing the profile. No scalar
 // method or port/register transaction is issued even if isolation fails.
 io_service_t service=IOServiceGetMatchingService(kIOMainPortDefault,IOServiceMatching("macUSPCIO"));
 if(!service)return 0;
 io_name_t service_class;
 if(IOObjectGetClass(service,service_class)!=KERN_SUCCESS||strcmp(service_class,"macUSPCIO")!=0){IOObjectRelease(service);return -3;}
 CFTypeRef user_class=IORegistryEntryCreateCFProperty(service,CFSTR("IOUserClientClass"),kCFAllocatorDefault,0);
 int qualified=user_class&&CFGetTypeID(user_class)==CFStringGetTypeID()&&CFStringCompare((CFStringRef)user_class,CFSTR("macUSPCIOUserClient"),0)==kCFCompareEqualTo;
 if(user_class)CFRelease(user_class);
 if(!qualified){IOObjectRelease(service);return -3;}
 io_connect_t connection=0;ret=IOServiceOpen(service,mach_task_self(),0,&connection);IOObjectRelease(service);
 if(ret==KERN_SUCCESS){IOServiceClose(connection);return -2;}
 return 0;
}
*/
import "C"

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	platform "github.com/vrooli/platform-go"
	"os"
	"strconv"
	"unsafe"
)

func nativeOpenRGBPlatform() string { return "darwin" }
func confineNativeOpenRGBTransport(ctx context.Context, _ string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	profile := C.CString(openRGBNativeDarwinTransportProfile)
	defer C.free(unsafe.Pointer(profile))
	if ret := C.vrooli_hid_transport_sandbox(profile); ret != 0 {
		return "", fmt.Errorf("native HID route could not deny unsupported macUSPCIO bus transport (%d)", int(ret))
	}
	if err := platform.VerifyDarwinNoChildrenProcess(os.Getpid()); err != nil {
		return "", fmt.Errorf("native HID route lacks kernel no-children containment: %w", err)
	}
	return NativeOpenRGBTransportPolicySHA256("darwin")
}
func nativeOpenRGBIdentity() (string, string, error) {
	if os.Geteuid() <= 0 {
		return "", "", errors.New("native OpenRGB HID access requires a non-root account")
	}
	machine, err := nativeOpenRGBMachineID()
	if err != nil {
		return "", "", err
	}
	return machine, fmt.Sprintf("uid:%d", os.Geteuid()), nil
}
func nativeOpenRGBMachineID() (string, error) {
	var uuid [128]C.char
	if C.vrooli_machine_uuid(&uuid[0], C.int(len(uuid))) != 0 {
		return "", errors.New("read native macOS machine identity")
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(C.GoString(&uuid[0])))), nil
}
func nativeOpenRGBDiscover(ctx context.Context) ([]OpenRGBNativeNode, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const limit = 1024
	ptr := C.calloc(limit, C.size_t(C.sizeof_vrooli_hid_node))
	if ptr == nil {
		return nil, errors.New("allocate native HID discovery")
	}
	defer C.free(ptr)
	n := int(C.vrooli_hid_list((*C.vrooli_hid_node)(ptr), limit))
	if n < 0 {
		return nil, fmt.Errorf("macOS HID metadata enumeration failed (%d)", n)
	}
	var result []OpenRGBNativeNode
	for _, v := range unsafe.Slice((*C.vrooli_hid_node)(ptr), n) {
		path := C.GoString(&v.path[0])
		d := OpenRGBNativeDevice{Detector: nativeOpenRGBHIDDetector(fmt.Sprintf("%04x:%04x", int(v.vendor), int(v.product))), Kind: "hid", StableID: path, HardwareID: fmt.Sprintf("%04x:%04x", int(v.vendor), int(v.product)), UsagePage: uint16(v.page), Usage: uint16(v.usage)}
		if qualifiedNativeHID(d) {
			result = append(result, OpenRGBNativeNode{Selector: d, Path: path, Generation: strconv.FormatUint(uint64(v.birth), 10)})
		}
	}
	return result, ctx.Err()
}
func nativeOpenRGBAccess(ctx context.Context, node OpenRGBNativeNode) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !qualifiedNativeHID(node.Selector) || node.Path != node.Selector.StableID {
		return errors.New("unqualified macOS HID access request")
	}
	birth, err := strconv.ParseUint(node.Generation, 10, 64)
	if err != nil || birth == 0 {
		return errors.New("invalid macOS HID generation")
	}
	path := C.CString(node.Path)
	defer C.free(unsafe.Pointer(path))
	if ret := C.vrooli_hid_access(path, C.uint64_t(birth)); ret != 0 {
		return fmt.Errorf("selected macOS HID account access unavailable (%d); no privilege fallback", int(ret))
	}
	return ctx.Err()
}
