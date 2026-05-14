package main

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"web-console/internal/config"
)

const maxFilePreviewBytes int64 = 256 * 1024

type fileReferenceResolution struct {
	inputPath       string
	resolvedPath    string
	line            *int
	exists          bool
	resolutionBasis string
	category        string
	canPreview      bool
	sizeBytes       int64
}

type fileReferenceError struct {
	code    string
	message string
}

func (e *fileReferenceError) Error() string { return e.message }

func asFileReferenceError(err error, target **fileReferenceError) bool {
	refErr, ok := err.(*fileReferenceError)
	if !ok {
		return false
	}
	*target = refErr
	return true
}

func newFileReferenceError(code, message string) error {
	return &fileReferenceError{code: code, message: message}
}

func (s *Server) resolveFileReference(ctx context.Context, sess *Session, rawPath string) (*fileReferenceResolution, error) {
	inputPath, line, normalizeErr := normalizeFileReferencePath(rawPath)
	if normalizeErr != nil {
		return nil, normalizeErr
	}
	if inputPath == "" {
		return nil, newFileReferenceError("file_reference_invalid", "Referenced path is empty")
	}

	projectRoot, err := filepath.Abs(config.ResolveWorkingDir())
	if err != nil {
		projectRoot = filepath.Clean(config.ResolveWorkingDir())
	}

	liveCwd, cwdErr := sess.CurrentDir(ctx)
	if cwdErr == nil && liveCwd != "" {
		if absCwd, absErr := filepath.Abs(liveCwd); absErr == nil {
			liveCwd = absCwd
		} else {
			liveCwd = filepath.Clean(liveCwd)
		}
	}

	// Build candidate absolute paths to probe. Filesystem permissions are the
	// only access gate — the web-console runs locally as the user and may
	// preview any file they can read (plans in ~/.vrooli, /etc configs, etc.).
	candidates := make([]struct {
		path  string
		basis string
	}, 0, 3)

	if filepath.IsAbs(inputPath) {
		candidates = append(candidates, struct {
			path  string
			basis string
		}{path: filepath.Clean(inputPath), basis: "absolute"})
	} else {
		if liveCwd != "" {
			candidates = append(candidates, struct {
				path  string
				basis string
			}{path: filepath.Join(liveCwd, inputPath), basis: "session_cwd"})
		}
		candidates = append(candidates, struct {
			path  string
			basis string
		}{path: filepath.Join(projectRoot, inputPath), basis: "project_root"})
	}

	for _, candidate := range candidates {
		resolvedPath, err := filepath.Abs(candidate.path)
		if err != nil {
			return nil, newFileReferenceError("file_reference_invalid", "Referenced path is invalid")
		}
		resolvedPath = filepath.Clean(resolvedPath)

		info, statErr := os.Stat(resolvedPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			if os.IsPermission(statErr) {
				return nil, newFileReferenceError("file_reference_not_allowed", "Referenced file is not readable")
			}
			return nil, newFileReferenceError("internal_error", "Failed to stat referenced file")
		}
		if info.IsDir() {
			return nil, newFileReferenceError("file_reference_not_previewable", "Directories cannot be previewed")
		}

		category, canPreview := classifyFileReference(resolvedPath, info.Size())
		return &fileReferenceResolution{
			inputPath:       rawPath,
			resolvedPath:    resolvedPath,
			line:            line,
			exists:          true,
			resolutionBasis: candidate.basis,
			category:        category,
			canPreview:      canPreview,
			sizeBytes:       info.Size(),
		}, nil
	}

	if cwdErr != nil && !filepath.IsAbs(inputPath) {
		return nil, newFileReferenceError("file_reference_unresolvable", "Relative path could not be resolved from the current session context")
	}
	return nil, newFileReferenceError("file_reference_not_found", "Referenced file was not found")
}

func normalizeFileReferencePath(raw string) (string, *int, error) {
	trimmed := stripWrappers(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", nil, nil
	}

	if strings.HasPrefix(strings.ToLower(trimmed), "file://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return "", nil, newFileReferenceError("file_reference_invalid", "Referenced file URL is invalid")
		}
		if parsed.Host != "" && parsed.Host != "localhost" {
			return "", nil, newFileReferenceError("file_reference_not_allowed", "Only local file URLs can be previewed")
		}
		decoded, err := url.PathUnescape(parsed.Path)
		if err != nil {
			return "", nil, newFileReferenceError("file_reference_invalid", "Referenced file URL path is invalid")
		}
		path, line := splitFileReferenceLine(decoded)
		return strings.TrimSpace(path), line, nil
	}

	if decoded, err := url.PathUnescape(trimmed); err == nil {
		trimmed = decoded
	}
	path, line := splitFileReferenceLine(trimmed)
	return expandHome(stripWrappers(strings.TrimSpace(path))), line, nil
}

// stripWrappers removes surrounding backticks, ASCII/curly quotes, and matched
// bracket-like wrappers that agents sometimes include around a path in prose
// (“ `path` “, "path", 'path', <path>, [path]). Only trims when the same
// character (or its match) appears on both ends so we don't eat real path
// content.
func stripWrappers(s string) string {
	for {
		if len(s) < 2 {
			return s
		}
		first, last := s[0], s[len(s)-1]
		// Same-char wrappers.
		if first == last && (first == '`' || first == '"' || first == '\'') {
			s = s[1 : len(s)-1]
			continue
		}
		// Matched bracket wrappers.
		if (first == '<' && last == '>') || (first == '[' && last == ']') || (first == '(' && last == ')') {
			s = s[1 : len(s)-1]
			continue
		}
		// Curly quotes (multibyte) — handle as runes.
		if strings.HasPrefix(s, "“") && strings.HasSuffix(s, "”") {
			s = strings.TrimPrefix(strings.TrimSuffix(s, "”"), "“")
			continue
		}
		if strings.HasPrefix(s, "‘") && strings.HasSuffix(s, "’") {
			s = strings.TrimPrefix(strings.TrimSuffix(s, "’"), "‘")
			continue
		}
		return strings.TrimSpace(s)
	}
}

// expandHome expands a leading ~ or ~/ to the current user's home directory.
// Bare ~user/... forms are not supported (the path is returned unchanged).
func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func splitFileReferenceLine(raw string) (string, *int) {
	trimmed := strings.TrimSpace(raw)
	// Strip trailing wrapper/punctuation chars (e.g. `path:42`. or path:42),
	// then look for a :<digits> suffix.
	end := len(trimmed)
	for end > 0 {
		c := trimmed[end-1]
		if c == '`' || c == '"' || c == '\'' || c == ')' || c == ']' || c == '>' || c == '.' || c == ',' || c == ';' {
			end--
			continue
		}
		break
	}
	stripped := trimmed[:end]
	lastColon := strings.LastIndex(stripped, ":")
	if lastColon <= 0 || lastColon == len(stripped)-1 {
		return trimmed, nil
	}
	lineValue, err := strconv.Atoi(stripped[lastColon+1:])
	if err != nil || lineValue <= 0 {
		return trimmed, nil
	}
	return stripped[:lastColon], &lineValue
}

func classifyFileReference(path string, sizeBytes int64) (category string, canPreview bool) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".mdx", ".markdown":
		return "markdown", true
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico", ".bmp", ".tiff", ".pdf":
		return "binary", false
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".json", ".yml", ".yaml", ".sh", ".bash", ".zsh", ".sql", ".css", ".html", ".txt", ".proto", ".toml", ".ini", ".env":
		return "code", true
	default:
		if sizeBytes > maxFilePreviewBytes {
			return "text", true
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "text", false
		}
		if !utf8.Valid(data) || bytesContainNull(data) {
			return "binary", false
		}
		return "text", true
	}
}

func fileReferenceContentType(category string) string {
	switch category {
	case "markdown":
		return "text/markdown; charset=utf-8"
	default:
		return "text/plain; charset=utf-8"
	}
}

func bytesContainNull(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}
