// Package assets is the application layer for uploaded media: validated
// multipart uploads, catalog CRUD, and best-effort image derivative generation
// (logo/favicon/OG sizes) using only the standard library's image codecs. Bytes
// live on disk under UPLOAD_DIR; the assets table records the catalog. The
// Connect handler in handlers/assets adapts this Service, and the raw multipart
// upload + static /uploads file server are mounted as REST exceptions there.
package assets

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"  // register GIF decoder
	_ "image/jpeg" // register JPEG decoder
	"image/png"
	"io"
	"log"
	"math"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Domain errors surfaced to the handler, which maps them to Connect codes / HTTP status.
var (
	ErrAssetNotFound   = errors.New("asset not found")
	ErrInvalidFileType = errors.New("invalid file type")
	ErrFileTooLarge    = errors.New("file exceeds maximum size")
	ErrUploadFailed    = errors.New("failed to save uploaded file")
)

// MaxUploadSize is the maximum accepted upload size (10 MB).
const MaxUploadSize = 10 * 1024 * 1024

// Asset is an uploaded file's catalog record.
type Asset struct {
	ID               int
	Filename         string
	OriginalFilename string
	MimeType         string
	SizeBytes        int64
	StoragePath      string
	ThumbnailPath    *string
	AltText          *string
	Category         string
	UploadedBy       *string
	CreatedAt        time.Time
	URL              string
	Derivatives      map[string]string
}

// UploadRequest carries a multipart upload plus its metadata.
type UploadRequest struct {
	File       multipart.File
	Header     *multipart.FileHeader
	Category   string
	AltText    string
	UploadedBy string
}

// Service handles upload, catalog, and derivative operations.
type Service struct {
	db           *sql.DB
	uploadDir    string
	maxSize      int64
	baseURL      string
	allowedTypes map[string]bool
}

// NewService constructs the assets Service and ensures the upload directory tree
// exists. UPLOAD_DIR overrides the default ./uploads location.
func NewService(db *sql.DB) *Service {
	uploadDir := strings.TrimSpace(os.Getenv("UPLOAD_DIR"))
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		log.Printf("assets: create upload dir %s failed: %v", uploadDir, err)
	}
	for _, subdir := range []string{"logos", "favicons", "og-images", "general"} {
		if err := os.MkdirAll(filepath.Join(uploadDir, subdir), 0o755); err != nil {
			log.Printf("assets: create upload subdir failed: %v", err)
		}
	}
	return &Service{
		db:        db,
		uploadDir: uploadDir,
		maxSize:   MaxUploadSize,
		baseURL:   "/api/v1/uploads",
		allowedTypes: map[string]bool{
			"image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true,
			"image/svg+xml": true, "image/x-icon": true, "image/vnd.microsoft.icon": true,
		},
	}
}

// Upload validates, stores, records, and (best-effort) derives an uploaded file.
func (s *Service) Upload(req *UploadRequest) (*Asset, error) {
	if req.File == nil || req.Header == nil {
		return nil, errors.New("no file provided")
	}
	if req.Header.Size > s.maxSize {
		return nil, fmt.Errorf("%w: max %d bytes", ErrFileTooLarge, s.maxSize)
	}

	mimeType := req.Header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = detectMimeType(req.Header.Filename)
	}
	if !s.allowedTypes[mimeType] {
		return nil, fmt.Errorf("%w: %s not allowed", ErrInvalidFileType, mimeType)
	}

	ext := filepath.Ext(req.Header.Filename)
	if ext == "" {
		ext = mimeTypeToExt(mimeType)
	}
	uniqueName := generateUniqueFilename(ext)

	category := req.Category
	if category == "" {
		category = "general"
	}
	storagePath := filepath.Join(categoryToSubdir(category), uniqueName)
	fullPath := filepath.Join(s.uploadDir, storagePath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, req.File)
	if err != nil {
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}

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
	if err := s.db.QueryRow(`
		INSERT INTO assets (filename, original_filename, mime_type, size_bytes, storage_path, alt_text, category, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`, asset.Filename, asset.OriginalFilename, asset.MimeType, asset.SizeBytes, asset.StoragePath,
		altText, category, uploadedBy).Scan(&asset.ID, &asset.CreatedAt); err != nil {
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("failed to save asset metadata: %w", err)
	}
	asset.URL = s.baseURL + "/" + storagePath

	if strings.HasPrefix(mimeType, "image/") && mimeType != "image/svg+xml" {
		derivatives, derr := s.generateDerivatives(fullPath, storagePath, mimeType, category)
		if derr != nil {
			log.Printf("assets: derivative generation failed (category=%s mime=%s): %v", category, mimeType, derr)
		} else if len(derivatives) > 0 {
			asset.Derivatives = derivatives
			if thumb, ok := derivatives["logo_256"]; ok && asset.ThumbnailPath == nil {
				asset.ThumbnailPath = ptr(thumb)
			}
			if thumb, ok := derivatives["favicon_64"]; ok && asset.ThumbnailPath == nil {
				asset.ThumbnailPath = ptr(thumb)
			}
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

const assetColumns = `id, filename, original_filename, mime_type, size_bytes, storage_path,
	thumbnail_path, alt_text, category, uploaded_by, created_at`

func scanAsset(row interface{ Scan(...any) error }) (*Asset, error) {
	var a Asset
	if err := row.Scan(&a.ID, &a.Filename, &a.OriginalFilename, &a.MimeType, &a.SizeBytes,
		&a.StoragePath, &a.ThumbnailPath, &a.AltText, &a.Category, &a.UploadedBy, &a.CreatedAt); err != nil {
		return nil, err
	}
	return &a, nil
}

// Get retrieves an asset by id.
func (s *Service) Get(id int) (*Asset, error) {
	asset, err := scanAsset(s.db.QueryRow(`SELECT `+assetColumns+` FROM assets WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAssetNotFound
	}
	if err != nil {
		return nil, err
	}
	asset.URL = s.baseURL + "/" + asset.StoragePath
	return asset, nil
}

// List retrieves assets, optionally filtered by category, newest first.
func (s *Service) List(category string) ([]Asset, error) {
	query := `SELECT ` + assetColumns + ` FROM assets`
	var args []interface{}
	if category != "" {
		query += ` WHERE category = $1`
		args = append(args, category)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		asset, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		asset.URL = s.baseURL + "/" + asset.StoragePath
		assets = append(assets, *asset)
	}
	return assets, rows.Err()
}

// Delete removes an asset record and its files (best-effort on disk).
func (s *Service) Delete(id int) error {
	asset, err := s.Get(id)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM assets WHERE id = $1`, id); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(s.uploadDir, asset.StoragePath)); err != nil && !os.IsNotExist(err) {
		log.Printf("assets: delete file %s failed: %v", asset.StoragePath, err)
	}
	if asset.ThumbnailPath != nil {
		_ = os.Remove(filepath.Join(s.uploadDir, *asset.ThumbnailPath))
	}
	return nil
}

// GetFilePath returns the on-disk path for a stored asset path.
func (s *Service) GetFilePath(storagePath string) string {
	return filepath.Join(s.uploadDir, storagePath)
}

// GetUploadDir returns the base upload directory.
func (s *Service) GetUploadDir() string { return s.uploadDir }

func generateUniqueFilename(ext string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%d_%s%s", time.Now().Unix(), hex.EncodeToString(b), ext)
}

func detectMimeType(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}

func mimeTypeToExt(mimeType string) string {
	switch mimeType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "image/x-icon", "image/vnd.microsoft.icon":
		return ".ico"
	default:
		return ".bin"
	}
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

func (s *Service) generateDerivatives(fullPath, storagePath, mimeType, category string) (map[string]string, error) {
	if mimeType == "image/svg+xml" {
		return map[string]string{
			"logo_512": storagePath, "logo_256": storagePath, "logo_128": storagePath,
			"logo_icon": storagePath, "favicon": storagePath, "favicon_64": storagePath,
			"favicon_32": storagePath, "favicon_16": storagePath, "apple_touch_180": storagePath,
			"og_image_1200x630": storagePath,
		}, nil
	}

	srcImg, err := decodeImage(fullPath)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	baseDir := filepath.Dir(storagePath)
	baseName := strings.TrimSuffix(filepath.Base(storagePath), filepath.Ext(storagePath))
	derivatives := make(map[string]string)

	type size struct {
		key  string
		w, h int
	}
	writeSizes := func(sizes []size) error {
		for _, sz := range sizes {
			path := filepath.Join(baseDir, fmt.Sprintf("%s-%s.png", baseName, sz.key))
			if err := s.saveResizedPNG(srcImg, filepath.Join(s.uploadDir, path), sz.w, sz.h); err != nil {
				return err
			}
			derivatives[sz.key] = path
		}
		return nil
	}

	switch category {
	case "logo":
		if err := writeSizes([]size{
			{"logo_512", 512, 512},
			{"logo_256", 256, 256},
			{"logo_128", 128, 128},
			{"apple_touch_180", 180, 180},
			{"favicon_64", 64, 64},
			{"favicon_32", 32, 32},
			{"favicon_16", 16, 16},
		}); err != nil {
			return nil, err
		}
		if path, ok := derivatives["logo_256"]; ok {
			derivatives["logo_icon"] = path
		}
		if path, ok := derivatives["favicon_32"]; ok {
			derivatives["favicon"] = path
		}
	case "favicon":
		if err := writeSizes([]size{
			{"favicon_64", 64, 64}, {"favicon_32", 32, 32}, {"favicon_16", 16, 16}, {"apple_touch_180", 180, 180},
		}); err != nil {
			return nil, err
		}
	case "og_image":
		path := filepath.Join(baseDir, fmt.Sprintf("%s-og-1200x630.png", baseName))
		if err := s.saveResizedPNG(srcImg, filepath.Join(s.uploadDir, path), 1200, 630); err != nil {
			return nil, err
		}
		derivatives["og_image_1200x630"] = path
	}

	for key, path := range derivatives {
		derivatives[key] = strings.TrimPrefix(filepath.Clean(path), string(os.PathSeparator))
	}
	return derivatives, nil
}

func decodeImage(path string) (image.Image, error) {
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

func (s *Service) saveResizedPNG(src image.Image, outputPath string, targetW, targetH int) error {
	dst := resizeContain(src, targetW, targetH)
	if dst == nil {
		return fmt.Errorf("resize failed for %s", outputPath)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
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
	srcW, srcH := srcBounds.Dx(), srcBounds.Dy()
	if srcW == 0 || srcH == 0 {
		return nil
	}
	scale := math.Min(float64(targetW)/float64(srcW), float64(targetH)/float64(srcH))
	if scale > 1 {
		scale = 1 // never upscale
	}
	dstW, dstH := int(float64(srcW)*scale), int(float64(srcH)*scale)
	if dstW == 0 || dstH == 0 {
		return nil
	}

	dstImg := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			dstImg.Set(x, y, color.RGBA{0, 0, 0, 0})
		}
	}
	xOffset, yOffset := (targetW-dstW)/2, (targetH-dstH)/2
	for y := 0; y < dstH; y++ {
		for x := 0; x < dstW; x++ {
			srcX := int(float64(x)/scale + 0.5)
			srcY := int(float64(y)/scale + 0.5)
			dstImg.Set(x+xOffset, y+yOffset, src.At(srcBounds.Min.X+srcX, srcBounds.Min.Y+srcY))
		}
	}
	return dstImg
}

func ptr(v string) *string { return &v }
