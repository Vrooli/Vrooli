package uxcapture

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	bundle "github.com/vrooli/browser-automation-studio/services/uxcapture"
)

func bundleDirectory(capturesRoot, bundleID string) (string, error) {
	parsedID, err := uuid.Parse(bundleID)
	if err != nil || parsedID.String() != strings.ToLower(bundleID) {
		return "", errors.New("bundle id must be a canonical UUID")
	}
	return filepath.Join(capturesRoot, "ux", parsedID.String()), nil
}

// MountArtifactHTTP exposes only artifacts declared by a bundle manifest. The
// caller supplies BAS's configured private capture root; request paths never
// choose a filesystem root.
func MountArtifactHTTP(r chi.Router, capturesRoot string) {
	r.Get("/ux-capture/bundles/{bundleID}/artifacts/*", func(w http.ResponseWriter, req *http.Request) {
		bundleID := chi.URLParam(req, "bundleID")
		parsedID, err := uuid.Parse(bundleID)
		if err != nil || parsedID.String() != strings.ToLower(bundleID) {
			http.NotFound(w, req)
			return
		}
		artifactPath := chi.URLParam(req, "*")
		if !safeArtifactRequestPath(artifactPath) {
			http.Error(w, "invalid artifact path", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(capturesRoot) == "" {
			http.Error(w, "capture storage unavailable", http.StatusServiceUnavailable)
			return
		}
		root, err := filepath.Abs(filepath.Join(capturesRoot, "ux"))
		if err != nil {
			http.NotFound(w, req)
			return
		}
		bundleDir := filepath.Join(root, parsedID.String())
		manifestPath := filepath.Join(bundleDir, "manifest.json")
		manifestBytes, err := readRegularNoSymlink(manifestPath)
		if err != nil {
			http.NotFound(w, req)
			return
		}
		var manifest bundle.Bundle
		if json.Unmarshal(manifestBytes, &manifest) != nil || manifest.Validate() != nil {
			http.Error(w, "bundle manifest is invalid", http.StatusUnprocessableEntity)
			return
		}
		declared, ok := findDeclaredArtifact(manifest.Artifacts, artifactPath)
		if !ok {
			http.NotFound(w, req)
			return
		}
		artifactFile := filepath.Join(bundleDir, filepath.FromSlash(declared.Path))
		content, err := readRegularNoSymlink(artifactFile)
		if err != nil {
			http.NotFound(w, req)
			return
		}
		if int64(len(content)) != declared.SizeBytes {
			http.Error(w, "artifact integrity check failed", http.StatusUnprocessableEntity)
			return
		}
		digest := sha256.Sum256(content)
		if hex.EncodeToString(digest[:]) != strings.ToLower(declared.SHA256) {
			http.Error(w, "artifact integrity check failed", http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", safeArtifactMediaType(declared))
		w.Header().Set("Content-Length", itoa(len(content)))
		w.Header().Set("Content-Disposition", "attachment")
		w.Header().Set("Cache-Control", "private, no-store, max-age=0")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	})
}

func safeArtifactRequestPath(value string) bool {
	if value == "" || strings.ContainsRune(value, '\x00') || strings.Contains(value, `\`) || strings.HasPrefix(value, "/") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(value))) == value
}

func findDeclaredArtifact(artifacts []bundle.Artifact, requested string) (bundle.Artifact, bool) {
	for _, artifact := range artifacts {
		if artifact.Path == requested {
			return artifact, true
		}
	}
	return bundle.Artifact{}, false
}

// readRegularNoSymlink rejects symlinks in every path component and verifies
// the opened target is a regular file. Bundle IDs and artifact paths are
// separately constrained before this helper is reached.
func readRegularNoSymlink(path string) ([]byte, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	volume := filepath.VolumeName(absolute)
	remainder := strings.TrimPrefix(absolute, volume+string(filepath.Separator))
	current := volume + string(filepath.Separator)
	for _, part := range strings.Split(remainder, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			return nil, statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("symlink path component")
		}
	}
	file, err := os.Open(absolute)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		if err == nil {
			err = errors.New("not a regular file")
		}
		return nil, err
	}
	return io.ReadAll(file)
}

func safeArtifactMediaType(artifact bundle.Artifact) string {
	ext := strings.ToLower(filepath.Ext(artifact.Path))
	allowed := map[string]string{
		".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".webp": "image/webp",
		".json": "application/json", ".txt": "text/plain; charset=utf-8", ".html": "text/html; charset=utf-8",
		".css": "text/css; charset=utf-8", ".svg": "image/svg+xml",
	}
	mediaType := allowed[ext]
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	// The manifest hint is intentionally ignored: suffix allowlisting and
	// attachment + nosniff headers prevent an untrusted type from becoming active.
	return mediaType
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
