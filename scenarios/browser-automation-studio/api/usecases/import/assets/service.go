package assets

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/usecases/import/shared"
)

// AssetsDir is the standard directory for assets within a project.
const AssetsDir = "assets"

// Service handles asset import business logic.
type Service struct {
	scanner      shared.DirectoryScanner
	projecter    shared.ProjectIndexer
	assetIndexer shared.AssetIndexer
	log          *logrus.Logger
}

// NewService creates a new Service.
func NewService(
	scanner shared.DirectoryScanner,
	projecter shared.ProjectIndexer,
	assetIndexer shared.AssetIndexer,
	log *logrus.Logger,
) *Service {
	return &Service{
		scanner:      scanner,
		projecter:    projecter,
		assetIndexer: assetIndexer,
		log:          log,
	}
}

// Upload uploads an asset file to a project.
func (s *Service) Upload(ctx context.Context, projectID uuid.UUID, path string, content io.Reader, size int64) (*UploadAssetResponse, error) {
	// Get project
	project, err := s.projecter.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// Validate and prepare path
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("path is required")
	}

	// Ensure path is under assets directory
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, AssetsDir+string(os.PathSeparator)) && !strings.HasPrefix(cleanPath, AssetsDir+"/") {
		cleanPath = filepath.Join(AssetsDir, cleanPath)
	}

	// Build full path
	fullPath, err := shared.SafeJoin(project.FolderPath, cleanPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Create parent directory
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write content
	written, err := io.Copy(file, content)
	if err != nil {
		os.Remove(fullPath) // Clean up on error
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Get file info
	name := filepath.Base(fullPath)
	mimeType := getMimeType(name)

	// Index in database
	if s.assetIndexer != nil {
		assetData := &shared.AssetIndexData{
			ID:        uuid.New(),
			ProjectID: projectID,
			FilePath:  filepath.ToSlash(cleanPath),
			FileName:  name,
			FileSize:  written,
			MimeType:  mimeType,
		}
		if err := s.assetIndexer.CreateAsset(ctx, assetData); err != nil {
			s.log.WithError(err).WithField("path", cleanPath).Warn("Failed to index uploaded asset")
			// Continue - file exists, just not indexed
		}
	}

	return &UploadAssetResponse{
		Path:      cleanPath,
		Name:      name,
		SizeBytes: written,
		MimeType:  mimeType,
	}, nil
}

// List lists assets in a project directory.
func (s *Service) List(ctx context.Context, projectID uuid.UUID, subPath string) (*ListAssetsResponse, error) {
	// Get project
	project, err := s.projecter.GetProjectByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// Build path to scan
	scanPath := filepath.Join(project.FolderPath, AssetsDir)
	if subPath != "" {
		scanPath = filepath.Join(scanPath, subPath)
	}

	// Verify path is within project
	isSubPath, err := shared.IsSubPath(project.FolderPath, scanPath)
	if err != nil || !isSubPath {
		return nil, errors.New("path must be within project directory")
	}

	// Check if path exists
	exists, _ := s.scanner.Exists(ctx, scanPath)
	if !exists {
		return &ListAssetsResponse{
			Path:    subPath,
			Entries: []AssetEntry{},
		}, nil
	}

	// Scan directory
	entries, err := s.scanner.ScanDirectory(ctx, scanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	// Build response
	result := &ListAssetsResponse{
		Path:    subPath,
		Entries: make([]AssetEntry, 0, len(entries)),
	}

	for _, entry := range entries {
		// Calculate relative path from assets root
		relPath, _ := shared.RelativePath(filepath.Join(project.FolderPath, AssetsDir), entry.Path)

		assetEntry := AssetEntry{
			Name:      entry.Name,
			Path:      filepath.ToSlash(relPath),
			SizeBytes: entry.Size,
			IsDir:     entry.IsDir,
		}

		if !entry.IsDir {
			assetEntry.MimeType = getMimeType(entry.Name)
		}

		result.Entries = append(result.Entries, assetEntry)
	}

	return result, nil
}

// Delete deletes an asset from a project.
func (s *Service) Delete(ctx context.Context, projectID uuid.UUID, path string) error {
	// Get project
	project, err := s.projecter.GetProjectByID(ctx, projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	// Build full path
	fullPath, err := shared.SafeJoin(project.FolderPath, AssetsDir, path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Remove file or directory
	if err := os.RemoveAll(fullPath); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	// Remove from database index
	if s.assetIndexer != nil {
		// Build the relative path as stored in the database
		assetPath := filepath.ToSlash(filepath.Join(AssetsDir, path))
		if err := s.assetIndexer.DeleteAssetByPath(ctx, projectID, assetPath); err != nil {
			s.log.WithError(err).WithField("path", assetPath).Debug("Failed to delete asset from index")
			// Continue - file is deleted, index will be cleaned up on next sync
		}
	}

	return nil
}

// getMimeType guesses MIME type from filename.
func getMimeType(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return "application/octet-stream"
	}
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}
