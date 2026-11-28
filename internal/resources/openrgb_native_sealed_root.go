package resources

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// OpenRGBNativeSealedRoot is separate from mutable operator admissions. Native
// staging must prove its OS-owner/ancestry contract before writing. An explicit
// artifact-class override preserves fixture isolation, never bypasses root/ACL
// checks and never establishes native qualification by itself.
func OpenRGBNativeSealedRoot() (string, error) {
	if override := strings.TrimSpace(os.Getenv("VROOLI_RESOURCE_ARTIFACT_DIR")); override != "" {
		if !filepath.IsAbs(override) || filepath.Clean(override) != override {
			return "", errors.New("native sealed artifact override must be canonical and absolute")
		}
		return filepath.Join(override, "openrgb-native-sealed"), nil
	}
	return nativeOpenRGBSealedRoot()
}

func requireCanonicalOpenRGBPublicationRoot() error {
	selected, err := OpenRGBNativeSealedRoot()
	if err != nil {
		return err
	}
	canonical, err := nativeOpenRGBSealedRoot()
	if err != nil {
		return err
	}
	same := selected == canonical
	if runtime.GOOS == "windows" {
		same = strings.EqualFold(selected, canonical)
	}
	if !same {
		return errors.New("fixture or alternate artifact roots cannot be promoted into the privileged native code namespace")
	}
	return nil
}
