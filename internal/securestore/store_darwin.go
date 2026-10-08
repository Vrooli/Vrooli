//go:build darwin && cgo

package securestore

/*
#cgo LDFLAGS: -framework CoreFoundation -framework Security
#include <CoreFoundation/CoreFoundation.h>
#include <Security/Security.h>
#include <stdlib.h>
#include <string.h>

static CFStringRef vrooliString(const char *bytes, int length) {
	return CFStringCreateWithBytes(kCFAllocatorDefault, (const UInt8 *)bytes, length, kCFStringEncodingUTF8, false);
}

// vrooliQuery builds the generic-password item selector shared by every
// operation. Service and account together are the primary key.
static CFMutableDictionaryRef vrooliQuery(const char *service, int serviceLen, const char *account, int accountLen) {
	CFMutableDictionaryRef query = CFDictionaryCreateMutable(kCFAllocatorDefault, 0,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	CFStringRef serviceValue = vrooliString(service, serviceLen);
	CFStringRef accountValue = vrooliString(account, accountLen);
	CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword);
	CFDictionarySetValue(query, kSecAttrService, serviceValue);
	CFDictionarySetValue(query, kSecAttrAccount, accountValue);
	CFRelease(serviceValue);
	CFRelease(accountValue);
	return query;
}

// vrooliPut overwrites an existing item and adds one otherwise, so Put is
// idempotent the way every adapter's Put must be.
static OSStatus vrooliPut(const char *service, int serviceLen, const char *account, int accountLen,
                          const char *value, int valueLen) {
	CFMutableDictionaryRef query = vrooliQuery(service, serviceLen, account, accountLen);
	CFDataRef data = CFDataCreate(kCFAllocatorDefault, (const UInt8 *)value, valueLen);
	CFMutableDictionaryRef update = CFDictionaryCreateMutable(kCFAllocatorDefault, 0,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	CFDictionarySetValue(update, kSecValueData, data);

	OSStatus status = SecItemUpdate(query, update);
	if (status == errSecItemNotFound) {
		CFDictionarySetValue(query, kSecValueData, data);
		status = SecItemAdd(query, NULL);
	}

	CFRelease(update);
	CFRelease(data);
	CFRelease(query);
	return status;
}

static OSStatus vrooliGet(const char *service, int serviceLen, const char *account, int accountLen,
                          void **out, int *outLen) {
	CFMutableDictionaryRef query = vrooliQuery(service, serviceLen, account, accountLen);
	CFDictionarySetValue(query, kSecReturnData, kCFBooleanTrue);
	CFDictionarySetValue(query, kSecMatchLimit, kSecMatchLimitOne);

	CFTypeRef result = NULL;
	OSStatus status = SecItemCopyMatching(query, &result);
	CFRelease(query);
	if (status != errSecSuccess) {
		return status;
	}
	CFDataRef data = (CFDataRef)result;
	CFIndex length = CFDataGetLength(data);
	void *buffer = malloc((size_t)length + 1);
	if (buffer == NULL) {
		CFRelease(result);
		return errSecAllocate;
	}
	if (length > 0) {
		memcpy(buffer, CFDataGetBytePtr(data), (size_t)length);
	}
	((char *)buffer)[length] = 0;
	*out = buffer;
	*outLen = (int)length;
	CFRelease(result);
	return errSecSuccess;
}

static OSStatus vrooliDelete(const char *service, int serviceLen, const char *account, int accountLen) {
	CFMutableDictionaryRef query = vrooliQuery(service, serviceLen, account, accountLen);
	OSStatus status = SecItemDelete(query);
	CFRelease(query);
	return status;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// nativeDefault uses the macOS Keychain through the Security framework. The `security`
// command accepts a password through -w, which puts a credential value in a
// process argument; the native API takes it inside a CFData, which is exactly
// why this adapter exists.
func nativeDefault() Store { return keychainStore{} }

type keychainStore struct{}

func (keychainStore) AdapterName() string { return "macos-keychain" }

// macOS OSStatus values used by the taxonomy. Named here rather than inlined so
// the mapping from Apple's codes to Vrooli's three conditions is readable.
const (
	errSecSuccess               = 0
	errSecItemNotFound          = -25300
	errSecInteractionNotAllowed = -25308
	errSecAuthFailed            = -25293
	errSecNotAvailable          = -25291
	errSecUserCanceled          = -128
)

func (keychainStore) Put(service, key, value string) error {
	cService, cKey := C.CString(service), C.CString(key)
	cValue := C.CString(value)
	defer C.free(unsafe.Pointer(cService))
	defer C.free(unsafe.Pointer(cKey))
	defer C.free(unsafe.Pointer(cValue))

	status := C.vrooliPut(cService, C.int(len(service)), cKey, C.int(len(key)), cValue, C.int(len(value)))
	return classifyKeychainStatus("store secure resource material", int(status))
}

func (keychainStore) Get(service, key string) (string, error) {
	cService, cKey := C.CString(service), C.CString(key)
	defer C.free(unsafe.Pointer(cService))
	defer C.free(unsafe.Pointer(cKey))

	var buffer unsafe.Pointer
	var length C.int
	status := C.vrooliGet(cService, C.int(len(service)), cKey, C.int(len(key)), &buffer, &length)
	if err := classifyKeychainStatus("read secure resource material", int(status)); err != nil {
		return "", err
	}
	defer C.free(buffer)
	return C.GoStringN((*C.char)(buffer), length), nil
}

func (keychainStore) Delete(service, key string) error {
	cService, cKey := C.CString(service), C.CString(key)
	defer C.free(unsafe.Pointer(cService))
	defer C.free(unsafe.Pointer(cKey))

	status := int(C.vrooliDelete(cService, C.int(len(service)), cKey, C.int(len(key))))
	if status == errSecItemNotFound {
		// Deleting an absent credential is the state the caller asked for.
		return nil
	}
	return classifyKeychainStatus("delete secure resource material", status)
}

// classifyKeychainStatus maps Apple's OSStatus onto the shared taxonomy. A
// locked keychain or a denied interaction is a provider condition an operator
// repairs, never an unset value they should be told to provision.
func classifyKeychainStatus(stage string, status int) error {
	switch status {
	case errSecSuccess:
		return nil
	case errSecItemNotFound:
		return fmt.Errorf("%w: %s", ErrNotFound, stage)
	case errSecInteractionNotAllowed, errSecAuthFailed, errSecNotAvailable, errSecUserCanceled:
		return fmt.Errorf("%w: %s: keychain is locked or access was denied (OSStatus %d); unlock the login keychain", ErrUnavailable, stage, status)
	default:
		return fmt.Errorf("%w: %s: OSStatus %d", ErrUnavailable, stage, status)
	}
}
