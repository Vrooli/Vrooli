package feedback

import (
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// AllowedAttachmentTypes lists Content-Types accepted for feedback round
// attachments. Image-only because the agent's vision pass is the sole
// consumer; broaden this when other modalities land.
var AllowedAttachmentTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// MaxAttachmentSize is the per-file byte cap. Enforced on multipart parse,
// which bounds the combined form size via http.Request.ParseMultipartForm.
const MaxAttachmentSize = 20 * 1024 * 1024 // 20 MiB

// SaveAttachmentsToDir reads the multipart "files" field and stores each
// upload under `{roundDir}/attachments/`. Returns the relative IDs
// ("attachments/{uuid}{ext}") to persist on the round. Caller must
// pre-create roundDir; passing the reserved dir directly avoids a race
// with concurrent submitters predicting the same round number.
func (s *Store) SaveAttachmentsToDir(roundDir string, r *http.Request) ([]string, error) {
	if r.MultipartForm == nil {
		return nil, nil
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		return nil, nil
	}

	attDir := filepath.Join(roundDir, "attachments")
	if err := os.MkdirAll(attDir, 0o750); err != nil {
		return nil, fmt.Errorf("create attachments dir: %w", err)
	}

	ids := make([]string, 0, len(files))
	for _, fh := range files {
		id, err := s.saveSingleAttachment(attDir, fh)
		if err != nil {
			return nil, err
		}
		ids = append(ids, filepath.Join("attachments", id))
	}
	return ids, nil
}

func (s *Store) saveSingleAttachment(attDir string, fh *multipart.FileHeader) (string, error) {
	mediaType, _, _ := mime.ParseMediaType(fh.Header.Get("Content-Type"))
	if !AllowedAttachmentTypes[mediaType] {
		return "", fmt.Errorf("unsupported file type %q (expected jpeg/png/gif/webp)", mediaType)
	}
	if fh.Size > MaxAttachmentSize {
		return "", fmt.Errorf("file %q exceeds %d bytes", fh.Filename, MaxAttachmentSize)
	}

	attID := uuid.New().String()
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	destName := attID + ext
	destPath := filepath.Join(attDir, destName)

	src, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("open uploaded file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create attachment file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		if rmErr := os.Remove(destPath); rmErr != nil && !os.IsNotExist(rmErr) {
			slog.Debug("feedback: remove partial attachment failed", "err", rmErr, "path", destPath)
		}
		return "", fmt.Errorf("write attachment file: %w", err)
	}
	return destName, nil
}

// ResolveAttachment returns the absolute disk path for the attachment with
// the given ID on the given round. Returns ("", false) if the id is
// malformed, escapes the round directory, or does not resolve to a file.
//
// The security posture: IDs are opaque UUID-prefixed filenames the store
// itself wrote, so callers should receive them back from persisted round
// records rather than from untrusted user input. Even so, this resolver
// defensively rejects path traversal (..) and absolute IDs.
func (s *Store) ResolveAttachment(initiativeName string, number int, slug, id string) (string, bool) {
	clean := filepath.Clean(id)
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") || strings.Contains(clean, string(filepath.Separator)+"..") {
		return "", false
	}
	// Accept either the full "attachments/{uuid}.ext" form or just the leaf.
	if !strings.HasPrefix(clean, "attachments"+string(filepath.Separator)) && !strings.HasPrefix(clean, "attachments/") {
		clean = filepath.Join("attachments", clean)
	}
	path := filepath.Join(s.RoundDir(initiativeName, number, slug), clean)

	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", false
	}
	return path, true
}

// ContentTypeForAttachment returns a best-effort Content-Type for an
// attachment ID, inferred from the file extension. Used by handler code
// so we don't have to maintain a separate magic-byte detector.
func ContentTypeForAttachment(id string) string {
	ext := strings.ToLower(filepath.Ext(id))
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
