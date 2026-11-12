package platform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// EnrolledNativeOperatorResourceStorageRoot reads the existing protected boot
// descriptor. An enrolled default ("", true) differs from no enrollment ("", false).
// It accepts no descriptor path and never provisions or changes enrollment.
func EnrolledNativeOperatorResourceStorageRoot(ctx context.Context, home string) (string, bool, error) {
	return enrolledNativeOperatorResourceStorageRootWith(ctx, home, enrolledNativeOperatorResourceStorageOptions)
}

func enrolledNativeOperatorResourceStorageRootWith(ctx context.Context, home string, read func(context.Context) (NativeSystemServiceBrokerOptions, bool, error)) (string, bool, error) {
	if ctx == nil || home == "" || !filepath.IsAbs(home) || filepath.Clean(home) != home || strings.ContainsAny(home, "\x00\r\n") {
		return "", false, ErrOperationIdentity
	}
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	options, configured, err := read(ctx)
	if err != nil {
		return "", false, err
	}
	if !configured {
		return "", false, nil
	}
	if options.OperatorHome != home {
		return "", false, ErrOperationIdentity
	}
	if err := validateNativeGlobalResourceStorageRoot(options.GlobalResourceStorageRoot); err != nil {
		return "", false, err
	}
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	return options.GlobalResourceStorageRoot, true, nil
}

func validateNativeGlobalResourceStorageRoot(root string) error {
	if root != "" && (len(root) > 4096 || !filepath.IsAbs(root) || filepath.Clean(root) != root || strings.TrimSpace(root) != root || strings.ContainsAny(root, "\x00\r\n")) {
		return errors.New("platform: enrolled global resource storage root must be canonical and absolute")
	}
	return nil
}

// Called only inside the native adapter's already-verified actual-operator
// callback. The manager process uses frozen enrollment rather than its ambient
// machine environment; elevated setup callers do not call this function.
func applyNativeOperatorResourceStorageRoot(root string) error {
	if err := validateNativeGlobalResourceStorageRoot(root); err != nil {
		return err
	}
	if root == "" {
		return os.Unsetenv("VROOLI_RESOURCE_STORAGE_ROOT")
	}
	return os.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", root)
}
