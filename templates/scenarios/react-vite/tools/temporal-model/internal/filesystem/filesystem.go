package filesystem

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ignoredDirs = map[string]bool{
	".git":          true,
	"node_modules":  true,
	"dist":          true,
	"build":         true,
	"coverage":      true,
	"_apalache-out": true,
}

func Slash(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}

func Abs(root string, rel string) string {
	return filepath.Join(root, filepath.FromSlash(rel))
}

func Rel(root string, abs string) (string, error) {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", err
	}
	return Slash(rel), nil
}

func Find(root string, suffix string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && ignoredDirs[entry.Name()] && path != root {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), suffix) {
			return nil
		}
		rel, err := Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, rel)
		return nil
	})
	sort.Strings(out)
	return out, err
}

func SHA256Bytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func SHA256String(data string) string {
	return SHA256Bytes([]byte(data))
}

func FileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return SHA256Bytes(data), nil
}

func TreeSHA256(root string) (string, error) {
	var parts []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if ignoredDirs[entry.Name()] && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(rel)
		if strings.HasPrefix(slash, "testdata/") || strings.HasSuffix(slash, "_test.go") || strings.HasSuffix(slash, ".formal.generated.json") || strings.HasSuffix(slash, ".qnt") {
			return nil
		}
		switch {
		case strings.HasSuffix(slash, ".go"),
			slash == "go.mod",
			strings.HasSuffix(slash, ".schema.json"):
			hash, err := FileSHA256(path)
			if err != nil {
				return err
			}
			parts = append(parts, slash+"\x00"+hash)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(parts)
	return SHA256String(strings.Join(parts, "\n")), nil
}
