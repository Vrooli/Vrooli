package recording

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/storage"
)

// selectRecordingStorage picks a storage implementation for recording artifacts.
// Order of precedence:
// 1) If BAS_RECORDING_STORAGE=local, force local file store under recordingsRoot.
// 2) If a storage client is provided, use it.
// 3) Fallback to local file store.
func selectRecordingStorage(provided storage.StorageInterface, recordingsRoot string, log *logrus.Logger) storage.StorageInterface {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("BAS_RECORDING_STORAGE")))
	if mode == "local" {
		return newRecordingFileStore(recordingsRoot, log)
	}
	if provided != nil {
		return provided
	}
	if log != nil {
		log.WithField("recordings_root", recordingsRoot).Warn("Recording storage not configured; using local filesystem fallback")
	}
	return newRecordingFileStore(recordingsRoot, log)
}

func deriveWorkflowName(manifest *recordingManifest, opts RecordingImportOptions) string {
	if opts.WorkflowName != "" {
		return opts.WorkflowName
	}
	if manifest.WorkflowName != "" {
		return manifest.WorkflowName
	}
	if manifest.RunID != "" {
		return fmt.Sprintf("Recording %s", manifest.RunID)
	}
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05")
	return fmt.Sprintf("Extension Recording %s", timestamp)
}

func loadRecordingManifest(zr *zip.Reader) (*recordingManifest, error) {
	var manifestFile *zip.File
	for i := range zr.File {
		entry := zr.File[i]
		name := strings.ToLower(entry.Name)
		if strings.HasSuffix(name, "manifest.json") || strings.HasSuffix(name, "recording.json") {
			manifestFile = entry
			break
		}
	}

	if manifestFile == nil {
		return nil, errors.New("recording archive is missing manifest.json")
	}

	data, _, err := readZipFile(manifestFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest recordingManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("invalid recording manifest: %w", err)
	}

	manifest.normalise()
	return &manifest, nil
}

func readZipFile(entry *zip.File) ([]byte, string, error) {
	reader, err := entry.Open()
	if err != nil {
		return nil, "", err
	}
	defer reader.Close()

	buf := bytes.NewBuffer(make([]byte, 0, entry.UncompressedSize64))
	if _, err := io.Copy(buf, io.LimitReader(reader, maxRecordingAssetBytes+1)); err != nil {
		return nil, "", err
	}

	data := buf.Bytes()
	contentType := http.DetectContentType(firstN(data, 512))
	return data, contentType, nil
}

func firstN(data []byte, n int) []byte {
	if len(data) <= n {
		return data
	}
	return data[:n]
}

func decodeDimensions(data []byte) (int, int) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

func extFromContentType(contentType string) string {
	switch strings.ToLower(contentType) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}

func normalizeArchiveName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	normalized := filepath.ToSlash(trimmed)
	normalized = strings.TrimPrefix(normalized, "./")
	normalized = strings.Trim(normalized, "/")
	if strings.Contains(normalized, "..") {
		return ""
	}
	return normalized
}
