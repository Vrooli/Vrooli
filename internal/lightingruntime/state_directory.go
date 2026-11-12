package lightingruntime

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/internal/config"
)

// EnsureStateDirectory is the shared intent/policy directory owner. A write
// may tighten a same-owner legacy directory; observation never repairs it.
func EnsureStateDirectory(path string, write bool) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("lighting state directory must be canonical and absolute")
	}
	ancestor := path
	for {
		_, err := os.Lstat(ancestor)
		if err == nil {
			break
		}
		if !write || !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return err
		}
		ancestor = parent
	}
	resolved, err := filepath.EvalSymlinks(ancestor)
	same := filepath.Clean(resolved) == ancestor
	if runtime.GOOS == "windows" {
		same = strings.EqualFold(filepath.Clean(resolved), ancestor)
	}
	if err != nil || !same {
		return errors.Join(errors.New("lighting state directory resolves through an alias"), err)
	}
	if write {
		if _, err := config.EnsureOwnedDirMode(path, 0o700); err != nil {
			return err
		}
	}
	return verifyLightingStateDirectory(path, write)
}
