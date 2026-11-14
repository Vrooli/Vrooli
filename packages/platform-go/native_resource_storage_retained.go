package platform

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
)

// Immutable published descriptors are ownership evidence even after boot
// removal. Multiple code versions may agree; conflicting namespaces are unknown.
func retainedNativeResourceStorageOptionsWith(ctx context.Context, principal string, paths []string, read func(string) (NativeSystemServiceBrokerOptions, error)) (NativeSystemServiceBrokerOptions, bool, error) {
	var selected NativeSystemServiceBrokerOptions
	found := false
	if principal == "" || len(paths) > 256 {
		return selected, false, ErrOperationIdentity
	}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return NativeSystemServiceBrokerOptions{}, false, err
		}
		options, err := read(path)
		if err != nil {
			return NativeSystemServiceBrokerOptions{}, false, err
		}
		if options.OperatorPrincipal != principal {
			return NativeSystemServiceBrokerOptions{}, false, ErrOperationIdentity
		}
		if err = validateNativeGlobalResourceStorageRoot(options.GlobalResourceStorageRoot); err != nil {
			return NativeSystemServiceBrokerOptions{}, false, err
		}
		if found {
			previous, current := selected, options
			previous.ExecutableSHA256 = ""
			current.ExecutableSHA256 = ""
			if previous != current {
				return NativeSystemServiceBrokerOptions{}, false, fmt.Errorf("%w: retained native descriptors disagree on the operator resource namespace", ErrOperationIdentity)
			}
		} else {
			selected = options
			found = true
		}
	}
	return selected, found, nil
}

// Only exact private names used by windowsCreateSealedBrokerFile are known
// unpublished residue. They are never namespace/code authority.
func knownNativeBrokerUnpublishedStage(name string) bool {
	base, nonce, ok := strings.Cut(name, ".stage-")
	if !ok || len(nonce) != 32 || strings.ToLower(nonce) != nonce {
		return false
	}
	if _, err := hex.DecodeString(nonce); err != nil {
		return false
	}
	if base == "vrooli.exe" {
		return true
	}
	return strings.HasPrefix(base, "broker-") && strings.HasSuffix(base, ".json") && validNativeBrokerDigest(strings.TrimSuffix(strings.TrimPrefix(base, "broker-"), ".json"))
}
