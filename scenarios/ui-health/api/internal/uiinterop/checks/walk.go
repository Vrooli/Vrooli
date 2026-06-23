package checks

import (
	"os"
	"path/filepath"
)

// uiSourceFile is one scanned production source file under a UI tree.
type uiSourceFile struct {
	relPath string // path relative to scenarioRoot, slash-separated (e.g. "ui/src/App.tsx")
	absPath string // absolute path on disk
	content string // file contents
}

// walkUISource walks the given subtree under scenarioRoot (e.g. "ui" or
// "ui/src"), yielding every production source file with a scannable extension.
// Test files, snapshot/fixture dirs, build output and dependency dirs are
// skipped (via isTestFile + skipDirectories + scanExtensions). It returns nil
// when the subtree does not exist, so callers can treat "no UI" as "skip".
func walkUISource(scenarioRoot, subdir string) []uiSourceFile {
	base := filepath.Join(scenarioRoot, filepath.FromSlash(subdir))
	info, err := os.Stat(base)
	if err != nil || !info.IsDir() {
		return nil
	}

	var files []uiSourceFile
	_ = filepath.WalkDir(base, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if _, skip := skipDirectories[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if isTestFile(d.Name()) {
			return nil
		}
		if _, ok := scanExtensions[filepath.Ext(d.Name())]; !ok {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		rel, relErr := filepath.Rel(scenarioRoot, path)
		if relErr != nil {
			rel = path
		}
		files = append(files, uiSourceFile{
			relPath: filepath.ToSlash(rel),
			absPath: path,
			content: string(data),
		})
		return nil
	})
	return files
}
