package filepreview

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

// DefaultMaxTextBytes caps the inline UTF-8 payload returned by GetTextContent.
// Media/image/pdf are never read inline — they stream through the blob route —
// so this only bounds markdown/code/text/csv/diff.
const DefaultMaxTextBytes int64 = 1 << 20 // 1 MiB

// sniffBytes is how many leading bytes classification inspects for unknown
// extensions.
const sniffBytes = 512

// Code enumerates the resolver failure classes. Package main maps these onto
// Connect codes (and the blob handler onto HTTP statuses).
type Code string

const (
	CodeInvalid        Code = "invalid"
	CodeNotAllowed     Code = "not_allowed"
	CodeNotPreviewable Code = "not_previewable"
	CodeNotFound       Code = "not_found"
	CodeUnresolvable   Code = "unresolvable"
	CodeInternal       Code = "internal"
	// CodeStale means the target changed underneath a multi-step read (a
	// directory mutated between listing pages). The client should restart the
	// read rather than treat it as a hard failure.
	CodeStale Code = "stale"
)

// Error is a typed resolver failure carrying a stable Code plus a user-safe
// message (never leaks more path detail than the already-visible input).
type Error struct {
	Code    Code
	Message string
}

func (e *Error) Error() string { return e.Message }

func newError(code Code, message string) *Error { return &Error{Code: code, Message: message} }

// Target is the rich preview metadata for one resolved file. It is the single
// shape the Connect resolve response, the preview-id store, and the UI model
// are all projected from.
type Target struct {
	SourceInfo           os.FileInfo
	InputPath            string
	ResolvedPath         string
	Basename             string
	Line                 int
	HasLine              bool
	ResolutionBasis      string // "absolute" | "session_cwd" | "project_root"
	Kind                 Kind
	MIMEType             string
	SizeBytes            int64
	ModTimeUnixNano      int64
	CanPreview           bool
	CanDownload          bool
	SupportsRange        bool
	TextContentAvailable bool
	ListingAvailable     bool
	Warnings             []string
}

// Resolver translates raw path strings into resolved Targets. It probes
// candidate absolute paths (absolute, then session cwd, then project root) and
// classifies the first one that exists. Filesystem permissions are the only
// access gate — web-console runs locally as the operator and may preview any
// file they can read.
type Resolver struct {
	// ProjectRoot is the fallback root for relative paths.
	ProjectRoot string
	// MaxTextBytes caps inline text payloads (0 → DefaultMaxTextBytes).
	MaxTextBytes int64
	// StatSortLimit is the largest directory a stat-requiring sort may order
	// (0 → StatSortLimit). Configurable so the downgrade path is testable
	// without building a directory of thousands of files.
	StatSortLimit int
	// MaxEntriesScanned caps entries read from one directory
	// (0 → MaxEntriesScanned).
	MaxEntriesScanned int
}

func (r *Resolver) maxText() int64 {
	if r.MaxTextBytes > 0 {
		return r.MaxTextBytes
	}
	return DefaultMaxTextBytes
}

// ResolveCandidate is one filesystem location a relative path matched, or one
// base directory offered for browsing when it matched nowhere.
type ResolveCandidate struct {
	ResolvedPath    string
	ResolutionBasis string
	Kind            Kind
	CanPreview      bool
	Basename        string
}

// ResolveResult carries the primary (default) target plus every location the
// input matched and, when nothing matched a relative path, the directories a
// client may browse to locate it manually.
type ResolveResult struct {
	// Selected is the default target (the first match), nil when Unresolved.
	Selected *Target
	// Candidates lists every match (including Selected). len > 1 means a
	// relative path exists under more than one base — the client disambiguates.
	Candidates []ResolveCandidate
	// Unresolved is true when a relative path matched no base directory.
	Unresolved bool
	// SearchBases are existing directories to browse when Unresolved.
	SearchBases []ResolveCandidate
}

// Resolve probes the candidate paths and returns metadata for the first file
// that exists. sessionCwd is the live working directory of the session (may be
// ""); cwdErr is non-nil when the cwd could not be determined, which only
// affects the error message for unresolved relative paths. It preserves the
// legacy error-on-not-found contract; callers wanting candidate lists and a
// browseable fallback use ResolveDetailed.
func (r *Resolver) Resolve(sessionCwd string, cwdErr error, rawPath string) (*Target, error) {
	res, err := r.ResolveDetailed(sessionCwd, cwdErr, rawPath)
	if err != nil {
		return nil, err
	}
	if res.Unresolved {
		if cwdErr != nil {
			return nil, newError(CodeUnresolvable, "Relative path could not be resolved from the current session context")
		}
		return nil, newError(CodeNotFound, "Referenced file was not found")
	}
	return res.Selected, nil
}

// ResolveDetailed probes every candidate path, classifies the first match as the
// selected target, and reports all matches plus — when a relative path matched
// nowhere — the existing base directories as browseable search bases.
func (r *Resolver) ResolveDetailed(sessionCwd string, cwdErr error, rawPath string) (*ResolveResult, error) {
	inputPath, line, err := normalizePath(rawPath)
	if err != nil {
		return nil, err
	}
	if inputPath == "" {
		return nil, newError(CodeInvalid, "Referenced path is empty")
	}

	projectRoot, absErr := filepath.Abs(r.ProjectRoot)
	if absErr != nil {
		projectRoot = filepath.Clean(r.ProjectRoot)
	}
	if sessionCwd != "" {
		if abs, e := filepath.Abs(sessionCwd); e == nil {
			sessionCwd = abs
		} else {
			sessionCwd = filepath.Clean(sessionCwd)
		}
	}

	type candidate struct {
		path    string
		basis   string
		warning string
	}
	candidates := make([]candidate, 0, 3)
	if filepath.IsAbs(inputPath) {
		candidates = append(candidates, candidate{path: filepath.Clean(inputPath), basis: "absolute"})
	} else {
		if sessionCwd != "" {
			candidates = append(candidates, candidate{path: filepath.Join(sessionCwd, inputPath), basis: "session_cwd"})
		}
		candidates = append(candidates, candidate{path: filepath.Join(projectRoot, inputPath), basis: "project_root"})
	}
	// Some authored Markdown links substitute full-width slashes throughout an
	// absolute path. Try recovery only after literal candidates, preserving real
	// Unicode filenames. Mixed paths and other Unicode characters stay untouched.
	if strings.HasPrefix(inputPath, "／") && !strings.Contains(inputPath, "/") {
		candidates = append(candidates, candidate{
			path: strings.ReplaceAll(inputPath, "／", "/"), basis: "absolute",
			warning: "Recovered an absolute path whose separators were full-width slashes.",
		})
	}

	type match struct {
		resolvedPath string
		basis        string
		warning      string
		info         os.FileInfo
	}
	matches := make([]match, 0, len(candidates))
	for _, c := range candidates {
		resolvedPath, e := filepath.Abs(c.path)
		if e != nil {
			return nil, newError(CodeInvalid, "Referenced path is invalid")
		}
		resolvedPath = filepath.Clean(resolvedPath)

		info, statErr := os.Stat(resolvedPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			if os.IsPermission(statErr) {
				return nil, newError(CodeNotAllowed, "Referenced file is not readable")
			}
			return nil, newError(CodeInternal, "Failed to stat referenced file")
		}
		matches = append(matches, match{resolvedPath: resolvedPath, basis: c.basis, warning: c.warning, info: info})
	}

	if len(matches) == 0 {
		// A relative path that matched nowhere is offered to the client as a
		// manual browse rooted at the directories we tried. An absolute miss
		// has no meaningful base to browse, so it stays an error.
		if filepath.IsAbs(inputPath) {
			return nil, newError(CodeNotFound, "Referenced file was not found")
		}
		bases := make([]ResolveCandidate, 0, 2)
		for _, b := range []struct{ path, basis string }{{sessionCwd, "session_cwd"}, {projectRoot, "project_root"}} {
			if b.path == "" {
				continue
			}
			if info, e := os.Stat(b.path); e == nil && info.IsDir() {
				bases = append(bases, ResolveCandidate{
					ResolvedPath: b.path, ResolutionBasis: b.basis, Kind: KindDirectory,
					CanPreview: true, Basename: filepath.Base(b.path),
				})
			}
		}
		return &ResolveResult{Unresolved: true, SearchBases: bases}, nil
	}

	selected, err := r.buildTarget(rawPath, matches[0].resolvedPath, matches[0].basis, matches[0].warning, matches[0].info, line)
	if err != nil {
		return nil, err
	}

	cands := make([]ResolveCandidate, 0, len(matches))
	for _, m := range matches {
		kind := KindDirectory
		canPreview := true
		if !m.info.IsDir() {
			if k, ok := classifyByExtension(m.resolvedPath); ok {
				kind = k
				canPreview = k.CanPreview()
			} else {
				kind = KindUnsupported
				canPreview = false
			}
		}
		cands = append(cands, ResolveCandidate{
			ResolvedPath: m.resolvedPath, ResolutionBasis: m.basis, Kind: kind,
			CanPreview: canPreview, Basename: filepath.Base(m.resolvedPath),
		})
	}
	return &ResolveResult{Selected: selected, Candidates: cands}, nil
}

// buildTarget performs the full (content-sniffing) classification of one
// matched path into a rich Target. Directories and special files are handled
// specially, matching the one-syscall-per-kind resolve contract.
func (r *Resolver) buildTarget(rawPath, resolvedPath, basis, warning string, info os.FileInfo, line *int) (*Target, error) {
	// A directory is a preview target in its own right: its contents come from
	// ListDirectory rather than from inline text or a byte stream. Resolve
	// deliberately does not count entries — that would cost a full directory
	// read on every resolve. The count arrives with the first listing page.
	if info.IsDir() {
		var warnings []string
		if warning != "" {
			warnings = append(warnings, warning)
		}
		return &Target{
			InputPath:        rawPath,
			ResolvedPath:     resolvedPath,
			Basename:         filepath.Base(resolvedPath),
			ResolutionBasis:  basis,
			Kind:             KindDirectory,
			MIMEType:         MIMEDirectory,
			SizeBytes:        0, // A directory's own byte size is not meaningful.
			ModTimeUnixNano:  info.ModTime().UnixNano(),
			CanPreview:       true,
			CanDownload:      false, // No archive stream in this version.
			SupportsRange:    false,
			ListingAvailable: true,
			Warnings:         warnings,
			// A :line suffix on a directory is meaningless; drop it rather than
			// failing a reference an agent wrote loosely.
		}, nil
	}

	// Reject sockets, FIFOs, and device nodes before anything opens them.
	// Opening a FIFO blocks until a writer appears, so classification's content
	// sniff must never reach one.
	if !info.Mode().IsRegular() {
		return nil, newError(CodeNotPreviewable, "Special files cannot be previewed")
	}

	cl := classify(resolvedPath, func() ([]byte, error) {
		return sniffFile(resolvedPath)
	})

	target := &Target{
		InputPath:            rawPath,
		ResolvedPath:         resolvedPath,
		Basename:             filepath.Base(resolvedPath),
		ResolutionBasis:      basis,
		Kind:                 cl.kind,
		MIMEType:             cl.mimeType,
		SourceInfo:           info,
		SizeBytes:            info.Size(),
		ModTimeUnixNano:      info.ModTime().UnixNano(),
		CanPreview:           cl.kind.CanPreview(),
		CanDownload:          true,
		SupportsRange:        cl.kind.UsesBlob(),
		TextContentAvailable: cl.kind.TextContentAvailable(),
	}
	if warning != "" {
		target.Warnings = append(target.Warnings, warning)
	}
	if line != nil {
		target.Line = *line
		target.HasLine = true
	}
	if target.TextContentAvailable && target.SizeBytes > r.maxText() {
		// Oversize text can't be inlined; offer download instead.
		target.TextContentAvailable = false
		target.Warnings = append(target.Warnings, fmt.Sprintf(
			"File is %d bytes; inline text preview is capped at %d bytes — download to view the full file.",
			target.SizeBytes, r.maxText()))
	}
	return target, nil
}

// ReadText returns the bounded UTF-8 content of a text-kind target. It
// re-validates UTF-8 and the size cap; callers must only invoke it when
// Target.TextContentAvailable is true.
func (r *Resolver) ReadText(t *Target) (content string, truncated bool, err error) {
	if !t.TextContentAvailable {
		return "", false, newError(CodeNotPreviewable, "File type cannot be previewed as text")
	}
	info, statErr := os.Stat(t.ResolvedPath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return "", false, newError(CodeNotFound, "Referenced file not found")
		}
		return "", false, newError(CodeInternal, "Failed to stat referenced file")
	}
	if info.Size() > r.maxText() {
		return "", false, newError(CodeNotPreviewable, "File exceeds the inline text preview limit")
	}
	data, readErr := os.ReadFile(t.ResolvedPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return "", false, newError(CodeNotFound, "Referenced file not found")
		}
		return "", false, newError(CodeInternal, "Failed to read referenced file")
	}
	if !utf8.Valid(data) || containsNUL(data) {
		return "", false, newError(CodeNotPreviewable, "File is not valid UTF-8 text")
	}
	return string(data), false, nil
}

// sniffFile reads up to sniffBytes leading bytes for content classification.
func sniffFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, sniffBytes)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		// EOF on an empty file is not a failure; return empty.
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, err
	}
	return buf[:n], nil
}

func containsNUL(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Path normalization (extracted from the legacy conversation resolver). Handles
// surrounding wrappers agents add in prose, file:// URLs, ~ expansion, percent-
// decoding, and a trailing :line suffix.
// ---------------------------------------------------------------------------

// normalizePath cleans a raw path string and splits off an optional :line
// suffix. Returns the cleaned path, the line (nil if absent), and a typed error
// for malformed file:// URLs.
func normalizePath(raw string) (string, *int, error) {
	trimmed := stripWrappers(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", nil, nil
	}

	if strings.HasPrefix(strings.ToLower(trimmed), "file://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return "", nil, newError(CodeInvalid, "Referenced file URL is invalid")
		}
		if parsed.Host != "" && parsed.Host != "localhost" {
			return "", nil, newError(CodeNotAllowed, "Only local file URLs can be previewed")
		}
		decoded, err := url.PathUnescape(parsed.Path)
		if err != nil {
			return "", nil, newError(CodeInvalid, "Referenced file URL path is invalid")
		}
		path, line := splitLineSuffix(decoded)
		return strings.TrimSpace(path), line, nil
	}

	if decoded, err := url.PathUnescape(trimmed); err == nil {
		trimmed = decoded
	}
	path, line := splitLineSuffix(trimmed)
	return expandHome(stripWrappers(strings.TrimSpace(path))), line, nil
}

// stripWrappers removes surrounding backticks, ASCII/curly quotes, and matched
// bracket-like wrappers agents include around a path in prose.
func stripWrappers(s string) string {
	for {
		if len(s) < 2 {
			return s
		}
		first, last := s[0], s[len(s)-1]
		if first == last && (first == '`' || first == '"' || first == '\'') {
			s = s[1 : len(s)-1]
			continue
		}
		if (first == '<' && last == '>') || (first == '[' && last == ']') || (first == '(' && last == ')') {
			s = s[1 : len(s)-1]
			continue
		}
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

// splitLineSuffix peels a trailing :<digits> suffix off a path, tolerating
// trailing wrapper/punctuation chars after the number.
func splitLineSuffix(raw string) (string, *int) {
	trimmed := strings.TrimSpace(raw)
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
