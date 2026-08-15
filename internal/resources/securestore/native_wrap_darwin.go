//go:build darwin && cgo

package securestore

/*
#cgo darwin LDFLAGS: -framework Security
#include <Security/Security.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>

static const char *vrooli_service = "Vrooli encrypted credential store";
static const char *vrooli_account = "vrooli-securestore-data-key";

static int vrooli_keychain_put(const uint8_t *data, size_t len) {
  SecKeychainItemRef existing = NULL;
  OSStatus find = SecKeychainFindGenericPassword(NULL, strlen(vrooli_service), vrooli_service,
    strlen(vrooli_account), vrooli_account, NULL, NULL, &existing);
  if (find == errSecSuccess && existing != NULL) {
    OSStatus update = SecKeychainItemModifyAttributesAndData(existing, NULL, (UInt32)len, data);
    CFRelease(existing);
    if (update != errSecSuccess) return (int)update;
    return 0;
  }
  OSStatus add = SecKeychainAddGenericPassword(NULL, strlen(vrooli_service), vrooli_service,
    strlen(vrooli_account), vrooli_account, (UInt32)len, data, NULL);
  return (int)add;
}

static int vrooli_keychain_get(uint8_t **data, size_t *len) {
  UInt32 password_len = 0;
  void *password = NULL;
  OSStatus status = SecKeychainFindGenericPassword(NULL, strlen(vrooli_service), vrooli_service,
    strlen(vrooli_account), vrooli_account, &password_len, &password, NULL);
  if (status != errSecSuccess) return (int)status;
  *data = malloc(password_len);
  if (*data == NULL) { SecKeychainItemFreeContent(NULL, password); return -1; }
  memcpy(*data, password, password_len);
  *len = password_len;
  SecKeychainItemFreeContent(NULL, password);
  return 0;
}
*/
import "C"

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"unsafe"
)

func nativeWrapAvailable() (string, error) {
	// A zero-length probe does not prove the login Keychain is reachable. The
	// actual encrypt/decrypt canary in the provider does; this only identifies
	// the backend for that proof.
	return keyStoreKeychain, nil
}

func nativeWrapProtect(value []byte) ([]byte, error) {
	if len(value) == 0 {
		return nil, fmt.Errorf("%w: empty data key", errKeyProviderUnavailable)
	}
	kek, err := nativeKeychainKey()
	if err != nil {
		return nil, err
	}
	sealed, err := sealDataKey(kek, value, providerNativeWrap)
	if err != nil {
		return nil, err
	}
	return []byte(sealed), nil
}

func nativeWrapUnprotect(value []byte) ([]byte, error) {
	kek, err := nativeKeychainKey()
	if err != nil {
		return nil, err
	}
	return openDataKey(kek, base64.StdEncoding.EncodeToString(value), providerNativeWrap, errWrongPassphrase)
}

func nativeKeychainKey() ([]byte, error) {
	var data *C.uint8_t
	var length C.size_t
	if status := C.vrooli_keychain_get(&data, &length); status == 0 {
		defer C.free(unsafe.Pointer(data))
		key := C.GoBytes(unsafe.Pointer(data), C.int(length))
		if len(key) == dataKeyLen {
			return key, nil
		}
		return nil, fmt.Errorf("%w: macOS Keychain wrap key has invalid length", errSealedCorrupt)
	}
	key := make([]byte, dataKeyLen)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate Keychain wrap key: %w", err)
	}
	if status := C.vrooli_keychain_put((*C.uint8_t)(unsafe.Pointer(&key[0])), C.size_t(len(key))); status != 0 {
		return nil, fmt.Errorf("%w: macOS Keychain write failed with status %d", errKeyProviderUnavailable, int(status))
	}
	return key, nil
}
