package resources

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/internal/config"
)

func nativeOpenRGBPrivatePath(path string) (bool, error) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return false, nil
	}
	root, err := OpenRGBNativeResourceRoot()
	if err != nil {
		return false, err
	}
	if !filepath.IsAbs(path) || !strings.EqualFold(filepath.VolumeName(root), filepath.VolumeName(path)) {
		return false, nil
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false, err
	}
	return rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}

func resourcePrivatePathPolicy(path string) (config.PrivatePathPolicy, error) {
	native, err := nativeOpenRGBPrivatePath(path)
	if err != nil || !native {
		return config.PrivatePathPolicy{}, err
	}
	owner, err := readOpenRGBNativeOwner()
	if err != nil {
		return config.PrivatePathPolicy{}, err
	}
	return owner.policyFor(path)
}

func verifyManagedUpdatePrivatePath(path string, info os.FileInfo) error {
	p, err := resourcePrivatePathPolicy(path)
	if err != nil {
		return err
	}
	return config.VerifyPrivatePath(path, info, p)
}

func writeResourceOwnedFileDurable(path string, data []byte, mode os.FileMode) error {
	native, err := nativeOpenRGBPrivatePath(path)
	if err != nil {
		return err
	}
	if !native {
		return config.WriteOwnedFileDurable(path, data, mode)
	}
	p, err := resourcePrivatePathPolicy(path)
	if err != nil {
		return err
	}
	return config.WritePrivateFileDurable(path, data, mode, p)
}
func writeResourceOwnedFileDurableStream(path string, mode os.FileMode, write func(io.Writer) error) error {
	native, err := nativeOpenRGBPrivatePath(path)
	if err != nil {
		return err
	}
	if !native {
		return config.WriteOwnedFileDurableStream(path, mode, write)
	}
	p, err := resourcePrivatePathPolicy(path)
	if err != nil {
		return err
	}
	return config.WritePrivateFileDurableStream(path, mode, p, write)
}
