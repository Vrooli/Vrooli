package buildinfo

import (
	"bytes"
	"errors"
	"fmt"
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"unicode"

	"golang.org/x/mod/module"
)

// Missing matches can be valid for an inactive build-tagged source. The Go
// compiler remains responsible for rejecting them when that source is active.
var errUnmatchedEmbed = errors.New("embed pattern has no usable matches")

// fs.Glob intentionally discards directory read errors. That is unsafe for a
// fingerprint: an unreadable directory cannot be treated as an empty match.
type embedGlobFS struct {
	fs.FS
	readErr error
}

func (f *embedGlobFS) Open(name string) (fs.File, error) {
	file, err := f.FS.Open(name)
	f.recordError(err)
	return file, err
}

func (f *embedGlobFS) recordError(err error) {
	if err != nil && !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR) && f.readErr == nil {
		f.readErr = err
	}
}

func (f *embedGlobFS) Stat(name string) (fs.FileInfo, error) {
	info, err := fs.Stat(f.FS, name)
	f.recordError(err)
	return info, err
}

func (f *embedGlobFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(f.FS, name)
	f.recordError(err)
	return entries, err
}

// IsGoBuildSource identifies compiler inputs that can change a Go package's
// compiled file set. Keep the root stamp and scenario closure cache aligned.
func IsGoBuildSource(name string) bool {
	// go build excludes test files. Assets with this suffix that are embedded
	// by production code are still collected by completeCompilerInputs.
	if strings.HasSuffix(filepath.Base(name), "_test.go") {
		return false
	}
	switch filepath.Ext(name) {
	case ".go", ".c", ".cc", ".cpp", ".cxx", ".m", ".h", ".hh", ".hpp", ".hxx", ".f", ".F", ".for", ".f90", ".s", ".S", ".sx", ".swig", ".swigcxx", ".syso":
		return true
	default:
		return false
	}
}

// completeCompilerInputs preserves the conservative portable source roots and
// adds assets named by real Go directive comments, including platform variants.
// A lexer avoids treating raw strings or documentation examples as directives.
// No Go toolchain invocation or dependency download is needed to check a stamp.
func completeCompilerInputs(root string, entries []fingerprintEntry) ([]fingerprintEntry, error) {
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		seen[entry.rel] = true
	}
	for _, entry := range slices.Clone(entries) {
		if !strings.HasSuffix(entry.rel, ".go") {
			continue
		}
		source := filepath.Join(root, filepath.FromSlash(entry.rel))
		patterns, err := sourceEmbedPatterns(source)
		if err != nil {
			return nil, err
		}
		for _, pattern := range patterns {
			files, err := embeddedFiles(filepath.Dir(source), pattern)
			if err != nil && !errors.Is(err, errUnmatchedEmbed) {
				return nil, fmt.Errorf("resolve embedded input in %s: %w", entry.rel, err)
			}
			for _, file := range files {
				rel, err := filepath.Rel(root, file)
				if err != nil {
					return nil, err
				}
				rel = filepath.ToSlash(rel)
				if seen[rel] {
					continue
				}
				asset, err := fingerprintFile(root, file)
				if err != nil {
					return nil, err
				}
				entries = append(entries, asset)
				seen[rel] = true
			}
		}
	}
	return entries, nil
}

func sourceEmbedPatterns(source string) ([]string, error) {
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}
	if !bytes.Contains(data, []byte("//go:embed")) {
		return nil, nil
	}
	var lex scanner.Scanner
	lex.Init(token.NewFileSet().AddFile(source, -1, len(data)), data, nil, scanner.ScanComments)
	var patterns []string
	for {
		_, kind, literal := lex.Scan()
		if kind == token.EOF {
			return patterns, nil
		}
		if kind != token.COMMENT || !strings.HasPrefix(literal, "//go:embed") {
			continue
		}
		rest := strings.TrimPrefix(literal, "//go:embed")
		if rest == "" || (rest[0] != ' ' && rest[0] != '\t') {
			continue
		}
		for rest = strings.TrimSpace(rest); rest != ""; rest = strings.TrimSpace(rest) {
			var pattern string
			if rest[0] == '"' || rest[0] == '`' {
				quoted, err := strconv.QuotedPrefix(rest)
				if err != nil {
					return nil, fmt.Errorf("parse embedded input in %s: %w", source, err)
				}
				pattern, err = strconv.Unquote(quoted)
				if err != nil {
					return nil, err
				}
				rest = rest[len(quoted):]
				if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
					return nil, fmt.Errorf("embedded input in %s has no separator after %q", source, quoted)
				}
			} else {
				end := strings.IndexFunc(rest, unicode.IsSpace)
				if end < 0 {
					end = len(rest)
				}
				pattern, rest = rest[:end], rest[end:]
			}
			patterns = append(patterns, pattern)
		}
	}
}

func embeddedFiles(dir, pattern string) ([]string, error) {
	includeHidden := strings.HasPrefix(pattern, "all:")
	pattern = strings.TrimPrefix(pattern, "all:")
	if !fs.ValidPath(pattern) || pattern == "." {
		return nil, fmt.Errorf("invalid embed pattern %q", pattern)
	}
	if _, err := path.Match(pattern, ""); err != nil {
		return nil, err
	}
	filesystem := &embedGlobFS{FS: os.DirFS(dir)}
	matches, err := fs.Glob(filesystem, pattern)
	if err != nil {
		return nil, err
	}
	if filesystem.readErr != nil {
		return nil, fmt.Errorf("read embed glob directory: %w", filesystem.readErr)
	}
	var files []string
	var unmatched error
	for _, match := range matches {
		matchedPath := filepath.Join(dir, filepath.FromSlash(match))
		for current := matchedPath; current != dir; current = filepath.Dir(current) {
			info, err := os.Lstat(current)
			if err != nil {
				return nil, err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("embedded input %s traverses a symlink", matchedPath)
			}
			if invalidEmbedName(filepath.Base(current)) {
				return nil, fmt.Errorf("embedded input %s has an invalid module file name", matchedPath)
			}
			if info.IsDir() {
				if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
					return nil, fmt.Errorf("embedded input %s crosses a module boundary", matchedPath)
				}
			}
		}
		matchedFiles := 0
		err := filepath.WalkDir(matchedPath, func(file string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// Glob and recursive directory matching must never follow an alias
			// out of the admitted package or stamp bytes the compiler rejects.
			if entry.Type()&os.ModeSymlink != 0 {
				if file == matchedPath {
					return fmt.Errorf("embedded input %s is a symlink", file)
				}
				return nil
			}
			if invalidEmbedName(entry.Name()) {
				if entry.IsDir() {
					return fs.SkipDir
				}
				if strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") {
					return nil
				}
				return fmt.Errorf("embedded input %s has an invalid module file name", file)
			}
			if entry.IsDir() {
				if file != matchedPath {
					if !includeHidden && (strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_")) {
						return fs.SkipDir
					}
					if _, err := os.Stat(filepath.Join(file, "go.mod")); err == nil {
						return fs.SkipDir
					}
				}
				return nil
			}
			if file != matchedPath && !includeHidden && (strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_")) {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				if file == matchedPath {
					return fmt.Errorf("embedded input %s is not a regular file", file)
				}
				return nil
			}
			files = append(files, file)
			matchedFiles++
			return nil
		})
		if err != nil {
			return nil, err
		}
		if matchedFiles == 0 {
			unmatched = fmt.Errorf("%w: embed pattern %q matches an empty directory %q", errUnmatchedEmbed, pattern, match)
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%w: embed pattern %q matches no files", errUnmatchedEmbed, pattern)
	}
	return files, unmatched
}

func invalidEmbedName(name string) bool {
	if module.CheckFilePath(name) != nil {
		return true
	}
	return name == ".git" || name == ".hg" || name == ".svn" || name == ".bzr"
}
