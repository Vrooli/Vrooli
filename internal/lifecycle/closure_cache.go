package lifecycle

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/tuning"
)

const closureCacheVersion = 1

// closureCache is the durable dependency-closure record for one Go component.
// It lives beside the component's build output, not in a shared global cache,
// so two scenarios cannot invalidate or consume one another's inputs.
type closureCache struct {
	Version   int      `json:"version"`
	Key       string   `json:"key"`
	Inputs    []string `json:"inputs"`
	Toolchain string   `json:"toolchain"`
}

// goListCacheKey identifies a dependency-closure query by the module files
// that can change its result. It deliberately excludes artifact mtimes and
// unrelated repository files, so a warm freshness gate reuses the closure
// until the module contract itself changes.
func goListCacheKey(dir string, deps hostProbeDeps) string {
	h := sha256.New()
	for _, name := range []string{"go.mod", "go.sum"} {
		path := filepath.Join(dir, name)
		data, err := deps.readFile(path)
		if err != nil {
			fmt.Fprintf(h, "%s:error:%v\n", name, err)
			continue
		}
		fmt.Fprintf(h, "%s:%x\n", name, sha256.Sum256(data))
	}
	return filepath.Clean(dir) + "\x00" + fmt.Sprintf("%x", h.Sum(nil))
}

func closureCachePath(dir string) string {
	return filepath.Join(dir, ".vrooli-closure-go_module.json")
}

func closureCacheKey(dir, toolchain string, deps hostProbeDeps) string {
	h := sha256.New()
	addFile := func(label, path string) {
		read := deps.readFile
		if read == nil {
			read = os.ReadFile
		}
		data, err := read(path)
		if err != nil {
			fmt.Fprintf(h, "%s:error:%v\n", label, err)
			return
		}
		fmt.Fprintf(h, "%s:%x\n", label, sha256.Sum256(data))
	}
	addFile("go.mod", filepath.Join(dir, "go.mod"))
	addFile("go.sum", filepath.Join(dir, "go.sum"))
	if replaces, err := localReplaceDirs(filepath.Join(dir, "go.mod")); err == nil {
		for _, replace := range replaces {
			root := filepath.Clean(filepath.Join(dir, replace))
			addFile("replace:go.mod:"+root, filepath.Join(root, "go.mod"))
			addFile("replace:go.sum:"+root, filepath.Join(root, "go.sum"))
		}
	}
	fmt.Fprintf(h, "toolchain:%s\n", strings.TrimSpace(toolchain))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func readClosureCache(path, key, toolchain string, read func(string) ([]byte, error)) ([]string, bool) {
	if read == nil {
		read = os.ReadFile
	}
	data, err := read(path)
	if err != nil {
		return nil, false
	}
	var cached closureCache
	if json.Unmarshal(data, &cached) != nil || cached.Version != closureCacheVersion || cached.Key != key || cached.Toolchain != toolchain || len(cached.Inputs) == 0 {
		return nil, false
	}
	return append([]string(nil), cached.Inputs...), true
}

func writeClosureCache(path string, cached closureCache) error {
	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".vrooli-closure-go_module-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(tuning.PermFile); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
