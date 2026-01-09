package storage

import (
	"fmt"
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
	ContentType string
	ObjectName  string
	Path        string
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

func detectContentTypeFromFile(filePath string, fallback string) string {
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

	header := make([]byte, 512)
	n, _ := file.Read(header)
	if n == 0 {
		return ""
	}
	return http.DetectContentType(header[:n])
}
