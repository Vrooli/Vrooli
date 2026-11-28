package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// ArtifactInfo represents stored artifact information.
type ArtifactInfo struct {
	URL         string
	SizeBytes   int64
	SHA256      string
	ContentType string
	ObjectName  string
	Path        string
}

func artifactDigest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func artifactObjectName(executionID uuid.UUID, label string, ext string) string {
	safeLabel := sanitizeArtifactLabel(label)
	if ext == "" {
		ext = ".bin"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return fmt.Sprintf("artifacts/%s/%s-%s%s", executionID.String(), safeLabel, uuid.New().String(), ext)
}

func sanitizeArtifactLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		label = "artifact"
	}
	label = strings.ReplaceAll(label, "/", "-")
	label = strings.ReplaceAll(label, "\\", "-")
	return label
}

func artifactURL(objectName string) string {
	return fmt.Sprintf("/api/v1/artifacts/%s", strings.TrimPrefix(objectName, "/"))
}

// DetectContentTypeFromFile uses an explicit fallback first, then the path
// extension, and finally a bounded prefix sniff. Callers that need content
// type inference over file-backed artifacts share this policy.
func DetectContentTypeFromFile(filePath string, fallback string) string {
	if fallback != "" {
		return fallback
	}
	ext := filepath.Ext(filePath)
	if ext != "" {
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			return contentType
		}
	}
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()
	return SniffContentType(file)
}

// SniffContentType matches net/http's content-type detection using at most its
// documented 512-byte sample from the caller's stream.
func SniffContentType(reader io.Reader) string {
	header := make([]byte, 512)
	n, _ := reader.Read(header)
	if n == 0 {
		return ""
	}
	return http.DetectContentType(header[:n])
}
