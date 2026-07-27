package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/filerouting"
	corestorage "github.com/vrooli/api-core/storage"
	"landing-page-business-suite-api/internal/envx"
)

var (
	ErrAssetNotFound    = errors.New("asset not found")
	ErrInvalidFileType  = errors.New("invalid file type")
	ErrFileTooLarge     = errors.New("file exceeds maximum size")
	ErrUploadFailed     = errors.New("failed to save uploaded file")
	ErrInvalidAssetPath = errors.New("invalid asset storage path")
)

// Asset represents an uploaded file
type Asset struct {
	ID               int               `json:"id"`
	Filename         string            `json:"filename"`
	OriginalFilename string            `json:"original_filename"`
	MimeType         string            `json:"mime_type"`
	SizeBytes        int64             `json:"size_bytes"`
	StoragePath      string            `json:"storage_path"`
	ThumbnailPath    *string           `json:"thumbnail_path,omitempty"`
	AltText          *string           `json:"alt_text,omitempty"`
	Category         string            `json:"category"`
	UploadedBy       *string           `json:"uploaded_by,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	URL              string            `json:"url"`
	Derivatives      map[string]string `json:"derivatives,omitempty"`
}

// AssetUploadRequest contains upload parameters
type AssetUploadRequest struct {
	File       multipart.File
	Header     *multipart.FileHeader
	Category   string
	AltText    string
	UploadedBy string
}

// AssetsService handles file upload operations
type AssetsService struct {
	db           AssetStore
	uploadDir    string
	maxSize      int64
	baseURL      string
	allowedTypes map[string]bool
	fileRoots    *filerouting.RoutedRoots
}

// AssetStore is the persistence boundary for asset metadata.
type AssetStore interface {
	QueryRow(string, ...any) *sql.Row
	Query(string, ...any) (*sql.Rows, error)
	Exec(string, ...any) (sql.Result, error)
}

// NewAssetsService creates a new assets service
func NewAssetsService(db AssetStore) *AssetsService {
	uploadDir := envx.Get("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	// Ensure upload directory exists
	if err := os.MkdirAll(uploadDir, 0o750); err != nil {
		logStructuredError("create_upload_dir_failed", map[string]interface{}{
			"dir":   uploadDir,
			"error": err.Error(),
		})
	}

	// Create subdirectories for organization
	for _, subdir := range []string{"logos", "favicons", "og-images", "general"} {
		path := filepath.Join(uploadDir, subdir)
		if err := os.MkdirAll(path, 0o750); err != nil {
			logStructuredError("create_upload_subdir_failed", map[string]interface{}{
				"dir":   path,
				"error": err.Error(),
			})
		}
	}

	return &AssetsService{
		db:        db,
		uploadDir: uploadDir,
		maxSize:   10 * 1024 * 1024, // 10MB default
		baseURL:   "/api/v1/uploads",
		allowedTypes: map[string]bool{
			"image/png":                true,
			"image/jpeg":               true,
			"image/gif":                true,
			"image/webp":               true,
			"image/svg+xml":            true,
			"image/x-icon":             true,
			"image/vnd.microsoft.icon": true,
		},
	}
}

// SetFileRoots enables request-scoped file routing for live HTTP traffic.
// Unit tests and command paths that do not install roots retain the existing
// uploadDir behavior.
func (s *AssetsService) SetFileRoots(roots *filerouting.RoutedRoots) {
	s.fileRoots = roots
}

func (s *AssetsService) uploadRoot(ctx context.Context) (string, error) {
	if s.fileRoots == nil {
		return s.uploadDir, nil
	}
	return s.fileRoots.Pick(ctx, corestorage.ClassData)
}

// Upload handles file upload, validation, and storage
func (s *AssetsService) Upload(req *AssetUploadRequest) (*Asset, error) {
	return s.UploadContext(context.Background(), req)
}

// UploadContext persists an asset in the root selected for the request. A
// Test-Genie test-mode request therefore writes only to its leased data root.
func (s *AssetsService) UploadContext(ctx context.Context, req *AssetUploadRequest) (*Asset, error) {
	if req.File == nil || req.Header == nil {
		return nil, errors.New("no file provided")
	}

	// Validate file size
	if req.Header.Size > s.maxSize {
		return nil, fmt.Errorf("%w: max %d bytes", ErrFileTooLarge, s.maxSize)
	}

	// Detect and validate MIME type
	mimeType := req.Header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = detectMimeType(req.Header.Filename)
	}

	if !s.allowedTypes[mimeType] {
		return nil, fmt.Errorf("%w: %s not allowed", ErrInvalidFileType, mimeType)
	}

	// Generate unique filename
	ext := filepath.Ext(req.Header.Filename)
	if ext == "" {
		ext = mimeTypeToExt(mimeType)
	}
	uniqueName := generateUniqueFilename(ext)

	// Determine storage subdirectory based on category
	category := req.Category
	if category == "" {
		category = "general"
	}
	subdir := categoryToSubdir(category)

	// Create full path
	storagePath := filepath.Join(subdir, uniqueName)
	uploadRoot, err := s.uploadRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve upload root: %w", err)
	}
	fullPath := filepath.Join(uploadRoot, storagePath)

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}

	// Create destination file
	// #nosec G304 -- fullPath is rooted at trusted configured storage and storagePath uses a fixed category plus a generated filename.
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}
	defer dst.Close()

	// Copy file content
	written, err := io.Copy(dst, req.File)
	if err != nil {
		if removeErr := os.Remove(fullPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			logStructuredError("asset_upload_cleanup_failed", map[string]interface{}{"path": fullPath, "error": removeErr.Error()})
		}
		return nil, fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}
	if s.fileRoots != nil {
		s.fileRoots.RecordWrite(ctx)
	}

	// Insert into database
	query := `
		INSERT INTO assets (filename, original_filename, mime_type, size_bytes, storage_path, alt_text, category, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`

	var altText, uploadedBy *string
	if req.AltText != "" {
		altText = &req.AltText
	}
	if req.UploadedBy != "" {
		uploadedBy = &req.UploadedBy
	}

	asset := &Asset{
		Filename:         uniqueName,
		OriginalFilename: req.Header.Filename,
		MimeType:         mimeType,
		SizeBytes:        written,
		StoragePath:      storagePath,
		AltText:          altText,
		Category:         category,
		UploadedBy:       uploadedBy,
	}

	err = s.db.QueryRow(query,
		asset.Filename,
		asset.OriginalFilename,
		asset.MimeType,
		asset.SizeBytes,
		asset.StoragePath,
		altText,
		category,
		uploadedBy,
	).Scan(&asset.ID, &asset.CreatedAt)
	if err != nil {
		if removeErr := os.Remove(fullPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			logStructuredError("asset_metadata_cleanup_failed", map[string]interface{}{"path": fullPath, "error": removeErr.Error()})
		}
		return nil, fmt.Errorf("failed to save asset metadata: %w", err)
	}

	asset.URL = s.baseURL + "/" + storagePath

	// Generate derivatives for known categories (best-effort)
	if strings.HasPrefix(mimeType, "image/") && mimeType != "image/svg+xml" {
		derivatives, derr := s.generateDerivativesAtRoot(uploadRoot, fullPath, storagePath, mimeType, category)
		if derr != nil {
			logStructuredError("asset_derivative_failed", map[string]interface{}{
				"error":    derr.Error(),
				"category": category,
				"mime":     mimeType,
			})
		} else if len(derivatives) > 0 {
			asset.Derivatives = derivatives
			if thumb, ok := derivatives["logo_256"]; ok && asset.ThumbnailPath == nil {
				asset.ThumbnailPath = stringPtr(thumb)
			}
			if thumb, ok := derivatives["favicon_64"]; ok && asset.ThumbnailPath == nil {
				asset.ThumbnailPath = stringPtr(thumb)
			}
			// Ensure logo_icon always exists for logo uploads
			if category == "logo" {
				if _, ok := asset.Derivatives["logo_icon"]; !ok {
					if alias, ok := asset.Derivatives["logo_256"]; ok {
						asset.Derivatives["logo_icon"] = alias
					} else {
						asset.Derivatives["logo_icon"] = storagePath
					}
				}
			}
		}
	}

	return asset, nil
}

// Get retrieves an asset by ID
func (s *AssetsService) Get(id int) (*Asset, error) {
	query := `
		SELECT id, filename, original_filename, mime_type, size_bytes, storage_path,
		       thumbnail_path, alt_text, category, uploaded_by, created_at
		FROM assets
		WHERE id = $1
	`

	var asset Asset
	err := s.db.QueryRow(query, id).Scan(
		&asset.ID, &asset.Filename, &asset.OriginalFilename, &asset.MimeType,
		&asset.SizeBytes, &asset.StoragePath, &asset.ThumbnailPath,
		&asset.AltText, &asset.Category, &asset.UploadedBy, &asset.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrAssetNotFound
	}
	if err != nil {
		return nil, err
	}

	asset.URL = s.baseURL + "/" + asset.StoragePath
	return &asset, nil
}

// List retrieves assets, optionally filtered by category
func (s *AssetsService) List(category string) ([]Asset, error) {
	query := `
		SELECT id, filename, original_filename, mime_type, size_bytes, storage_path,
		       thumbnail_path, alt_text, category, uploaded_by, created_at
		FROM assets
	`
	args := []interface{}{}

	if category != "" {
		query += " WHERE category = $1"
		args = append(args, category)
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var asset Asset
		err := rows.Scan(
			&asset.ID, &asset.Filename, &asset.OriginalFilename, &asset.MimeType,
			&asset.SizeBytes, &asset.StoragePath, &asset.ThumbnailPath,
			&asset.AltText, &asset.Category, &asset.UploadedBy, &asset.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		asset.URL = s.baseURL + "/" + asset.StoragePath
		assets = append(assets, asset)
	}

	return assets, nil
}

// Delete removes an asset by ID
func (s *AssetsService) Delete(id int) error {
	return s.DeleteContext(context.Background(), id)
}

// DeleteContext removes an asset from the request-selected data root.
func (s *AssetsService) DeleteContext(ctx context.Context, id int) error {
	// Get asset to find file path
	asset, err := s.Get(id)
	if err != nil {
		return err
	}

	// Delete from database first
	_, err = s.db.Exec("DELETE FROM assets WHERE id = $1", id)
	if err != nil {
		return err
	}

	// Delete file from disk
	fullPath, pathErr := s.ResolveStoragePathContext(ctx, asset.StoragePath)
	if pathErr != nil {
		logStructuredError("delete_asset_path_invalid", map[string]interface{}{"id": id, "path": asset.StoragePath, "error": pathErr.Error()})
	} else if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		logStructuredError("delete_asset_file_failed", map[string]interface{}{
			"id":    id,
			"path":  fullPath,
			"error": err.Error(),
		})
		// Don't return error - DB record is already deleted
	} else if s.fileRoots != nil {
		s.fileRoots.RecordWrite(ctx)
	}

	// Delete thumbnail if exists
	if asset.ThumbnailPath != nil {
		if thumbPath, err := s.ResolveStoragePathContext(ctx, *asset.ThumbnailPath); err == nil {
			_ = os.Remove(thumbPath)
		}
	}

	return nil
}

// GetFilePath returns the full filesystem path for an asset
func (s *AssetsService) GetFilePath(storagePath string) string {
	path, _ := s.ResolveStoragePath(storagePath)
	return path
}

// ResolveStoragePath returns a path that is guaranteed to remain under the
// configured upload root. Storage paths can originate from a URL or database,
// so every filesystem operation must resolve them through this boundary.
func (s *AssetsService) ResolveStoragePath(storagePath string) (string, error) {
	return s.ResolveStoragePathContext(context.Background(), storagePath)
}

// ResolveStoragePathContext resolves a storage-relative asset path under the
// root selected for this request and rejects traversal before touching disk.
func (s *AssetsService) ResolveStoragePathContext(ctx context.Context, storagePath string) (string, error) {
	cleanPath := filepath.Clean(strings.TrimSpace(storagePath))
	if cleanPath == "." || filepath.IsAbs(cleanPath) || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", ErrInvalidAssetPath
	}
	uploadRoot, err := s.uploadRoot(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve upload root: %w", err)
	}
	root, err := filepath.Abs(uploadRoot)
	if err != nil {
		return "", fmt.Errorf("resolve upload root: %w", err)
	}
	fullPath := filepath.Join(root, cleanPath)
	relative, err := filepath.Rel(root, fullPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", ErrInvalidAssetPath
	}
	return fullPath, nil
}

// Helper functions

func generateUniqueFilename(ext string) string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		logStructuredError("asset_filename_random_failed", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	}
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%d_%s%s", timestamp, hex.EncodeToString(bytes), ext)
}

func detectMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeTypes := map[string]string{
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
	}
	if mt, ok := mimeTypes[ext]; ok {
		return mt
	}
	return "application/octet-stream"
}

func mimeTypeToExt(mimeType string) string {
	extensions := map[string]string{
		"image/png":                ".png",
		"image/jpeg":               ".jpg",
		"image/gif":                ".gif",
		"image/webp":               ".webp",
		"image/svg+xml":            ".svg",
		"image/x-icon":             ".ico",
		"image/vnd.microsoft.icon": ".ico",
	}
	if ext, ok := extensions[mimeType]; ok {
		return ext
	}
	return ".bin"
}

func categoryToSubdir(category string) string {
	switch category {
	case "logo":
		return "logos"
	case "favicon":
		return "favicons"
	case "og_image":
		return "og-images"
	default:
		return "general"
	}
}

// GetUploadDir returns the base upload directory path
func (s *AssetsService) GetUploadDir() string {
	return s.uploadDir
}

func (s *AssetsService) generateDerivatives(fullPath, storagePath, mimeType, category string) (map[string]string, error) {
	return s.generateDerivativesAtRoot(s.uploadDir, fullPath, storagePath, mimeType, category)
}

func (s *AssetsService) generateDerivativesAtRoot(uploadRoot, fullPath, storagePath, mimeType, category string) (map[string]string, error) {
	// SVGs can't be rasterized without extra deps; fallback to reusing the original for all slots.
	if mimeType == "image/svg+xml" {
		return map[string]string{
			"logo_512":          storagePath,
			"logo_256":          storagePath,
			"logo_128":          storagePath,
			"logo_icon":         storagePath,
			"favicon":           storagePath,
			"favicon_64":        storagePath,
			"favicon_32":        storagePath,
			"favicon_16":        storagePath,
			"apple_touch_180":   storagePath,
			"og_image_1200x630": storagePath,
		}, nil
	}

	srcImg, err := s.decodeImage(fullPath, mimeType)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	baseDir := filepath.Dir(storagePath)
	baseName := strings.TrimSuffix(filepath.Base(storagePath), filepath.Ext(storagePath))
	derivatives := make(map[string]string)

	switch category {
	case "logo":
		sizes := []struct {
			key string
			w   int
			h   int
		}{
			{key: "logo_512", w: 512, h: 512},
			{key: "logo_256", w: 256, h: 256},
			{key: "logo_128", w: 128, h: 128},
			{key: "apple_touch_180", w: 180, h: 180},
			{key: "favicon_64", w: 64, h: 64},
			{key: "favicon_32", w: 32, h: 32},
			{key: "favicon_16", w: 16, h: 16},
		}
		for _, size := range sizes {
			path := filepath.Join(baseDir, fmt.Sprintf("%s-%s.png", baseName, size.key))
			if err := saveResizedPNG(srcImg, filepath.Join(uploadRoot, path), size.w, size.h); err != nil {
				return nil, err
			}
			derivatives[size.key] = path
		}
		// Icon alias for convenience
		if path, ok := derivatives["logo_256"]; ok {
			derivatives["logo_icon"] = path
		}
		if path, ok := derivatives["favicon_32"]; ok {
			derivatives["favicon"] = path
		}
	case "favicon":
		sizes := []struct {
			key string
			w   int
			h   int
		}{
			{key: "favicon_64", w: 64, h: 64},
			{key: "favicon_32", w: 32, h: 32},
			{key: "favicon_16", w: 16, h: 16},
			{key: "apple_touch_180", w: 180, h: 180},
		}
		for _, size := range sizes {
			path := filepath.Join(baseDir, fmt.Sprintf("%s-%s.png", baseName, size.key))
			if err := saveResizedPNG(srcImg, filepath.Join(uploadRoot, path), size.w, size.h); err != nil {
				return nil, err
			}
			derivatives[size.key] = path
		}
	case "og_image":
		path := filepath.Join(baseDir, fmt.Sprintf("%s-og-1200x630.png", baseName))
		if err := saveResizedPNG(srcImg, filepath.Join(uploadRoot, path), 1200, 630); err != nil {
			return nil, err
		}
		derivatives["og_image_1200x630"] = path
	}

	for key, path := range derivatives {
		derivatives[key] = strings.TrimPrefix(filepath.Clean(path), string(os.PathSeparator))
	}

	return derivatives, nil
}

func (s *AssetsService) decodeImage(path string, mimeType string) (image.Image, error) {
	// #nosec G304 -- derivative paths are generated beneath the selected trusted storage root.
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func saveResizedPNG(src image.Image, outputPath string, targetW, targetH int) error {
	dst := resizeContain(src, targetW, targetH)
	if dst == nil {
		return fmt.Errorf("resize failed for %s", outputPath)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	// #nosec G304 -- derivative output paths are generated beneath the selected trusted storage root.
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer f.Close()

	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(f, dst); err != nil {
		return fmt.Errorf("encode png: %w", err)
	}

	return nil
}

func resizeContain(src image.Image, targetW, targetH int) *image.RGBA {
	if targetW <= 0 || targetH <= 0 {
		return nil
	}

	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	if srcW == 0 || srcH == 0 {
		return nil
	}

	scale := math.Min(float64(targetW)/float64(srcW), float64(targetH)/float64(srcH))
	if scale > 1 {
		scale = 1 // avoid unnecessary upscaling
	}

	dstW := int(float64(srcW) * scale)
	dstH := int(float64(srcH) * scale)
	if dstW == 0 || dstH == 0 {
		return nil
	}

	dstImg := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			dstImg.Set(x, y, color.RGBA{0, 0, 0, 0})
		}
	}

	xOffset := (targetW - dstW) / 2
	yOffset := (targetH - dstH) / 2

	for y := 0; y < dstH; y++ {
		for x := 0; x < dstW; x++ {
			srcX := int(float64(x)/scale + 0.5)
			srcY := int(float64(y)/scale + 0.5)
			dstImg.Set(x+xOffset, y+yOffset, src.At(srcBounds.Min.X+srcX, srcBounds.Min.Y+srcY))
		}
	}

	return dstImg
}

func stringPtr(v string) *string {
	return &v
}
