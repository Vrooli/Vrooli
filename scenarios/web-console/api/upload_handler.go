package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vrooli/api-core/storage"
)

// maxUploadBodySize is the hard ceiling for the whole multipart body. It sits
// above the largest per-category cap so the socket is closed on an absurd
// request even before the file's category is known. Per-category caps are what
// actually govern ordinary uploads; this is only a backstop.
const maxUploadBodySize int64 = 512<<20 + 1<<20 // 513 MiB

// uploadCategory groups accepted file types so one size cap can govern a whole
// family. The file extension — not the client-declared Content-Type — decides
// the category, because the preview layer also classifies by extension. A file
// with no recognized extension falls through to uploadOther.
type uploadCategory string

const (
	uploadImage   uploadCategory = "image"
	uploadVideo   uploadCategory = "video"
	uploadAudio   uploadCategory = "audio"
	uploadPDF     uploadCategory = "pdf"
	uploadArchive uploadCategory = "archive"
	uploadText    uploadCategory = "text"
	uploadOther   uploadCategory = "file"
)

// uploadCategoryCaps bounds the bytes accepted for one file part, per category.
// Video is the outlier by necessity; everything else stays in the tens of MiB.
var uploadCategoryCaps = map[uploadCategory]int64{
	uploadImage:   25 << 20,
	uploadVideo:   512 << 20,
	uploadAudio:   128 << 20,
	uploadPDF:     64 << 20,
	uploadArchive: 128 << 20,
	uploadText:    25 << 20,
	uploadOther:   64 << 20,
}

// uploadCategoryExt maps a lower-case extension (with the leading dot) to a
// category. Extensions not listed are accepted as uploadOther unless blocked.
var uploadCategoryExt = map[string]uploadCategory{
	// Images
	".png": uploadImage, ".jpg": uploadImage, ".jpeg": uploadImage, ".gif": uploadImage,
	".webp": uploadImage, ".bmp": uploadImage, ".ico": uploadImage, ".avif": uploadImage,
	".tiff": uploadImage, ".tif": uploadImage, ".svg": uploadImage,
	// Video
	".mp4": uploadVideo, ".webm": uploadVideo, ".ogv": uploadVideo,
	".mov": uploadVideo, ".m4v": uploadVideo,
	// Audio
	".mp3": uploadAudio, ".wav": uploadAudio, ".ogg": uploadAudio, ".oga": uploadAudio,
	".m4a": uploadAudio, ".aac": uploadAudio, ".flac": uploadAudio, ".opus": uploadAudio,
	// Documents
	".pdf": uploadPDF,
	// Archives
	".zip": uploadArchive, ".tar": uploadArchive, ".gz": uploadArchive, ".tgz": uploadArchive,
	".bz2": uploadArchive, ".xz": uploadArchive, ".7z": uploadArchive, ".rar": uploadArchive,
	// Text, code, and data
	".md": uploadText, ".mdx": uploadText, ".markdown": uploadText, ".txt": uploadText,
	".csv": uploadText, ".tsv": uploadText, ".json": uploadText, ".jsonl": uploadText,
	".yaml": uploadText, ".yml": uploadText, ".toml": uploadText, ".ini": uploadText,
	".env": uploadText, ".xml": uploadText, ".html": uploadText, ".htm": uploadText,
	".css": uploadText, ".scss": uploadText, ".less": uploadText, ".sh": uploadText,
	".bash": uploadText, ".zsh": uploadText, ".fish": uploadText, ".sql": uploadText,
	".go": uploadText, ".ts": uploadText, ".tsx": uploadText, ".js": uploadText,
	".jsx": uploadText, ".mjs": uploadText, ".cjs": uploadText, ".py": uploadText,
	".rb": uploadText, ".rs": uploadText, ".java": uploadText, ".kt": uploadText,
	".swift": uploadText, ".c": uploadText, ".h": uploadText, ".cpp": uploadText,
	".hpp": uploadText, ".cc": uploadText, ".cs": uploadText, ".php": uploadText,
	".lua": uploadText, ".r": uploadText, ".pl": uploadText, ".proto": uploadText,
	".mod": uploadText, ".sum": uploadText, ".log": uploadText, ".diff": uploadText,
	".patch": uploadText,
}

// blockedUploadExt are executable and installer types that are never accepted.
// The agent already owns a shell, so this is defense-in-depth against the
// upload surface being used to plant runnable binaries — not a capability
// boundary. It is paired with magic-byte sniffing so renaming an executable to
// a benign extension does not defeat it.
var blockedUploadExt = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true, ".msi": true,
	".com": true, ".scr": true, ".bat": true, ".cmd": true, ".cpl": true,
	".sys": true, ".drv": true, ".efi": true, ".apk": true, ".dmg": true,
	".pkg": true, ".deb": true, ".rpm": true, ".jar": true,
}

// executableMagics are the leading bytes of native executables. MZ (Windows
// PE), ELF, and the Mach-O / fat-binary / Java-class cafebabe family are all
// refused regardless of the extension the client supplied.
var executableMagics = [][]byte{
	{0x4D, 0x5A},
	{0x7F, 'E', 'L', 'F'},
	{0xFE, 0xED, 0xFA, 0xCE},
	{0xFE, 0xED, 0xFA, 0xCF},
	{0xCE, 0xFA, 0xED, 0xFE},
	{0xCF, 0xFA, 0xED, 0xFE},
	{0xCA, 0xFE, 0xBA, 0xBE},
}

const maxUploadSniffBytes = 512

var (
	errUploadTooLarge = errors.New("upload too large")
	errUploadBlocked  = errors.New("executable upload blocked")
)

// uploadCategoryFor resolves a filename to its category. The second return is
// false when the extension is explicitly blocked.
func uploadCategoryFor(filename string) (uploadCategory, bool) {
	ext := strings.ToLower(filepath.Ext(filename))
	if blockedUploadExt[ext] {
		return "", false
	}
	if cat, ok := uploadCategoryExt[ext]; ok {
		return cat, true
	}
	return uploadOther, true
}

func uploadCapFor(cat uploadCategory) int64 {
	if limit, ok := uploadCategoryCaps[cat]; ok {
		return limit
	}
	return uploadCategoryCaps[uploadOther]
}

// sniffExecutable reports whether the leading bytes identify a native
// executable. It is deliberately narrow: a false positive would reject a
// legitimate file, so only unambiguous magic numbers are matched.
func sniffExecutable(header []byte) bool {
	for _, magic := range executableMagics {
		if bytes.HasPrefix(header, magic) {
			return true
		}
	}
	return false
}

func resolveUploadDir() string {
	return mustResolveScenarioStorageDir(storage.ClassCache, "uploads")
}

// resolveUploadDirFor resolves the uploads root for one request. Under a test
// lease the request context routes to the leased Cache root, so a BAS upload
// never lands in the operator's real uploads tree; without a lease it resolves
// to exactly the same path as resolveUploadDir.
func (s *Server) resolveUploadDirFor(ctx context.Context) string {
	if s.roots == nil {
		return resolveUploadDir()
	}
	root, err := s.roots.Pick(ctx, storage.ClassCache)
	if err != nil || strings.TrimSpace(root) == "" {
		return resolveUploadDir()
	}
	return ensureDir(filepath.Join(root, "uploads"))
}

// unsafeFilenameChars matches characters unsafe for filenames.
var unsafeFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

// sanitizeFilename strips path components and dangerous characters from a
// user-supplied filename to prevent path traversal and other attacks.
func sanitizeFilename(name string) string {
	// Strip directory components
	name = filepath.Base(name)
	// Replace unsafe characters
	name = unsafeFilenameChars.ReplaceAllString(name, "_")
	// Collapse runs of underscores
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	name = strings.Trim(name, "_.")
	if name == "" || name == "." || name == ".." {
		name = "upload"
	}
	// Limit length
	if len(name) > 200 {
		ext := filepath.Ext(name)
		name = name[:200-len(ext)] + ext
	}
	return name
}

// uniquePath returns a path that doesn't conflict with existing files by
// appending a numeric suffix before the extension when necessary.
func uniquePath(dir, name string) string {
	candidate := filepath.Join(dir, name)
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; i < 10*1000; i++ {
		candidate = filepath.Join(dir, fmt.Sprintf("%s_%d%s", base, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return candidate
}

// uploadResponse is the JSON body returned by a successful upload. `path` is
// the terminal-injection handle; the remaining fields let a caller render a
// name/size without re-deriving them (older callers only read `path`).
type uploadResponse struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	SizeBytes int64  `json:"sizeBytes"`
	Category  string `json:"category"`
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	sess := s.lookupSession(w, r)
	if sess == nil {
		return
	}

	if sess.IsDead() {
		writeCatalogError(w, "session_terminated", "Cannot upload to terminated session "+sanitizeID(sess.ID))
		return
	}

	// Parse the multipart stream directly rather than ParseMultipartForm so a
	// large video streams to disk instead of buffering the whole body in memory.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBodySize)
	reader, err := r.MultipartReader()
	if err != nil {
		writeCatalogError(w, "invalid_body", "Expected a multipart/form-data upload")
		return
	}

	part, err := nextFilePart(reader)
	if err != nil {
		writeCatalogError(w, "invalid_body", "Missing file field")
		return
	}
	defer part.Close()

	category, ok := uploadCategoryFor(part.FileName())
	if !ok {
		writeCatalogError(w, "invalid_upload_type", "Executable files are not accepted")
		return
	}
	limit := uploadCapFor(category)

	// Create session-scoped upload directory
	sessionDir := filepath.Join(s.resolveUploadDirFor(r.Context()), sess.ID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		writeCatalogError(w, "internal_error", "Failed to create upload directory")
		return
	}

	destPath := uniquePath(sessionDir, sanitizeFilename(part.FileName()))
	written, writeErr := writeUploadPart(destPath, part, limit)
	if writeErr != nil {
		os.Remove(destPath)
		var maxBytesErr *http.MaxBytesError
		switch {
		case errors.Is(writeErr, errUploadTooLarge):
			writeCatalogError(w, "upload_too_large", fmt.Sprintf("%s files may be at most %s", categoryLabel(category), humanBytes(limit)))
		case errors.Is(writeErr, errUploadBlocked):
			writeCatalogError(w, "invalid_upload_type", "Executable files are not accepted")
		case errors.As(writeErr, &maxBytesErr):
			writeCatalogError(w, "upload_too_large", "Request exceeds the maximum upload size of "+humanBytes(maxUploadBodySize))
		default:
			writeCatalogError(w, "internal_error", "Failed to write file")
		}
		return
	}

	// Records which root the write actually landed in. The lease reports a
	// primary write during test mode as an isolation leak, so this call is
	// what makes the evidence real rather than assumed.
	s.roots.RecordWrite(r.Context())

	writeJSON(w, http.StatusOK, uploadResponse{
		Path:      destPath,
		Name:      filepath.Base(destPath),
		SizeBytes: written,
		Category:  string(category),
	})
}

// nextFilePart advances the multipart reader to the first part named "file",
// closing and discarding any parts before it.
func nextFilePart(reader *multipart.Reader) (*multipart.Part, error) {
	for {
		part, err := reader.NextPart()
		if err != nil {
			return nil, err
		}
		if part.FormName() == "file" {
			return part, nil
		}
		_ = part.Close()
	}
}

// writeUploadPart streams one multipart part to destPath, enforcing limit bytes
// and refusing native executables detected by magic bytes. It returns the
// number of bytes written; on failure the caller removes destPath.
func writeUploadPart(destPath string, part *multipart.Part, limit int64) (int64, error) {
	dst, err := os.Create(destPath)
	if err != nil {
		return 0, err
	}
	defer dst.Close()

	header := make([]byte, maxUploadSniffBytes)
	n, err := io.ReadFull(part, header)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return 0, err
	}
	if sniffExecutable(header[:n]) {
		return 0, errUploadBlocked
	}

	var written int64
	if n > 0 {
		if _, err := dst.Write(header[:n]); err != nil {
			return 0, err
		}
		written = int64(n)
	}

	buf := make([]byte, 64<<10)
	for {
		read, readErr := part.Read(buf)
		if read > 0 {
			written += int64(read)
			if written > limit {
				return written, errUploadTooLarge
			}
			if _, err := dst.Write(buf[:read]); err != nil {
				return written, err
			}
		}
		if errors.Is(readErr, io.EOF) {
			return written, nil
		}
		if readErr != nil {
			return written, readErr
		}
	}
}

func categoryLabel(cat uploadCategory) string {
	switch cat {
	case uploadImage:
		return "Image"
	case uploadVideo:
		return "Video"
	case uploadAudio:
		return "Audio"
	case uploadPDF:
		return "PDF"
	case uploadArchive:
		return "Archive"
	case uploadText:
		return "Text"
	default:
		return "This"
	}
}

// humanBytes renders a byte count with binary units (e.g. "25 MiB").
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.0f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
