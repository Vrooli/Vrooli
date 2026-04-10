package pathfilter

import "strings"

// skipDirs is the canonical set of directory base names to always skip.
// Dot-directories are handled by prefix check in SkipDir.
var skipDirs = map[string]bool{
	// Build/deployment artifacts
	"platforms": true, "dist": true, "build": true,
	"bin": true, "bundle": true, "artifacts": true,
	// Dependencies
	"node_modules": true, "vendor": true,
	// Runtime data
	"data": true, "logs": true, "coverage": true, "playwright-driver": true,
	// Language caches
	"__pycache__": true, "target": true, "obj": true,
	// Temporary
	"tmp": true, "temp": true, "storybook-static": true,
	// Python virtual envs (non-dot variant; .venv caught by dot-prefix rule)
	"venv": true,
}

// sourceExts is the canonical set of scannable source code extensions.
var sourceExts = map[string]bool{
	".go":    true,
	".ts":    true,
	".tsx":   true,
	".js":    true,
	".jsx":   true,
	".py":    true,
	".rs":    true,
	".rb":    true,
	".java":  true,
	".kt":    true,
	".cs":    true,
	".swift": true,
}

// generatedSuffixes are filename suffixes that indicate generated code.
var generatedSuffixes = []string{
	".pb.go", "_pb.go", "_pb2.go",
	"_gen.go", "_generated.go",
	"_mock.go",
	".d.ts",
	".min.js", ".min.css",
}

// generatedPrefixes are filename prefixes that indicate generated code.
var generatedPrefixes = []string{
	"mock_",
}

// SkipDir reports whether a directory with the given base name should be
// skipped during code-quality filesystem walks.
func SkipDir(name string) bool {
	if len(name) > 0 && name[0] == '.' {
		return true
	}
	return skipDirs[name]
}

// SkipDirSet returns a copy of the canonical directory name set.
// Dot-directories are not included; they are matched by the prefix rule
// in SkipDir.
func SkipDirSet() map[string]bool {
	cp := make(map[string]bool, len(skipDirs))
	for k, v := range skipDirs {
		cp[k] = v
	}
	return cp
}

// IsSourceExt reports whether ext (including leading dot, e.g. ".go")
// is a recognized source code extension for scanning purposes.
func IsSourceExt(ext string) bool {
	return sourceExts[ext]
}

// SourceExts returns a copy of the canonical source extension set.
// Keys include the leading dot.
func SourceExts() map[string]bool {
	cp := make(map[string]bool, len(sourceExts))
	for k, v := range sourceExts {
		cp[k] = v
	}
	return cp
}

// IsGeneratedFile reports whether a file with the given base name matches
// a known generated-file pattern.
func IsGeneratedFile(name string) bool {
	for _, s := range generatedSuffixes {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	for _, p := range generatedPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}
