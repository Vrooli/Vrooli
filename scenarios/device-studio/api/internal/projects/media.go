package projects

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"runtime"
	"strings"
	"time"
	"unicode"

	"device-studio/internal/sourceimage"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/blobstore"
)

const (
	MaxSourceMediaBytes         int64 = 24 << 20
	MaxSourceMediaPixels        int64 = 24_000_000
	MaxSourceMediaSide                = 20_000
	MaxSourceMediaFilenameRunes       = 128
	cleanupTimeout                    = 5 * time.Second
)

type SourceMedia struct {
	ID                 string
	ProjectID          string
	SHA256             string
	SizeBytes          int64
	MediaType          string
	Width              uint32
	Height             uint32
	Orientation        string
	ColorModel         string
	ColorProfileStatus string
	OriginalFilename   string
	ImportedAt         time.Time
	ImportMethod       string
	Decoder            string
	storageKey         string
}

type SourceMediaRepository interface {
	CreateSourceMedia(context.Context, SourceMedia) (SourceMedia, bool, error)
	GetSourceMedia(context.Context, string, string) (SourceMedia, error)
	GetSourceMediaByDigest(context.Context, string, string) (SourceMedia, error)
	ListSourceMedia(context.Context, string, int, *SourceMedia) ([]SourceMedia, error)
}

type SourceMediaService interface {
	Import(context.Context, string, string, int64, io.Reader) (SourceMedia, error)
	Get(context.Context, string, string) (SourceMedia, error)
	Open(context.Context, string, string) (SourceMedia, io.ReadCloser, error)
	List(context.Context, string, int32, string) ([]SourceMedia, string, error)
}

type SourceMediaDeps struct {
	Projects   Service
	Repository SourceMediaRepository
	Store      blobstore.BlobStore
	Now        func() time.Time
	NewID      func() string
	Limits     SourceMediaLimits
}

type SourceMediaLimits struct {
	MaxBytes    int64
	MaxPixels   int64
	MaxSide     int
	MaxFilename int
}

func DefaultSourceMediaLimits() SourceMediaLimits {
	return SourceMediaLimits{MaxBytes: MaxSourceMediaBytes, MaxPixels: MaxSourceMediaPixels, MaxSide: MaxSourceMediaSide, MaxFilename: MaxSourceMediaFilenameRunes}
}

type sourceMediaService struct {
	projects Service
	repo     SourceMediaRepository
	store    blobstore.BlobStore
	now      func() time.Time
	newID    func() string
	limits   SourceMediaLimits
}

func NewSourceMediaService(deps SourceMediaDeps) SourceMediaService {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.NewID == nil {
		deps.NewID = uuid.NewString
	}
	defaults := DefaultSourceMediaLimits()
	if deps.Limits.MaxBytes <= 0 {
		deps.Limits.MaxBytes = defaults.MaxBytes
	}
	if deps.Limits.MaxPixels <= 0 {
		deps.Limits.MaxPixels = defaults.MaxPixels
	}
	if deps.Limits.MaxSide <= 0 {
		deps.Limits.MaxSide = defaults.MaxSide
	}
	if deps.Limits.MaxFilename <= 0 {
		deps.Limits.MaxFilename = defaults.MaxFilename
	}
	return &sourceMediaService{projects: deps.Projects, repo: deps.Repository, store: deps.Store, now: deps.Now, newID: deps.NewID, limits: deps.Limits}
}

type ErrInvalidSourceMedia struct{ Reason string }

func (e ErrInvalidSourceMedia) Error() string { return "source media: " + e.Reason }

type ErrSourceMediaTooLarge struct{ Limit int64 }

func (e ErrSourceMediaTooLarge) Error() string {
	return fmt.Sprintf("source media exceeds %d bytes", e.Limit)
}

type ErrSourceMediaNotFound struct{ ID string }

func (e ErrSourceMediaNotFound) Error() string { return fmt.Sprintf("source media %q not found", e.ID) }

type ErrSourceMediaDigestCollision struct{ SHA256 string }

func (e ErrSourceMediaDigestCollision) Error() string {
	return "source media digest collision for " + e.SHA256
}

func (s *sourceMediaService) Import(ctx context.Context, projectID, filename string, declaredSize int64, body io.Reader) (SourceMedia, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return SourceMedia{}, ErrInvalidSourceMedia{Reason: "project_id is required"}
	}
	if s.projects == nil || s.repo == nil || s.store == nil {
		return SourceMedia{}, errors.New("source media service is not configured")
	}
	if _, err := s.projects.Get(ctx, projectID); err != nil {
		return SourceMedia{}, err
	}
	if body == nil {
		return SourceMedia{}, ErrInvalidSourceMedia{Reason: "file is required"}
	}
	if declaredSize <= 0 {
		return SourceMedia{}, ErrInvalidSourceMedia{Reason: "file is empty"}
	}
	if declaredSize > s.limits.MaxBytes {
		return SourceMedia{}, ErrSourceMediaTooLarge{Limit: s.limits.MaxBytes}
	}
	data, err := io.ReadAll(io.LimitReader(body, s.limits.MaxBytes+1))
	if err != nil {
		return SourceMedia{}, fmt.Errorf("read source media: %w", err)
	}
	if int64(len(data)) > s.limits.MaxBytes {
		return SourceMedia{}, ErrSourceMediaTooLarge{Limit: s.limits.MaxBytes}
	}
	if int64(len(data)) != declaredSize {
		return SourceMedia{}, ErrInvalidSourceMedia{Reason: "multipart file size did not match bytes read"}
	}
	info, err := inspectSourceImage(data, s.limits)
	if err != nil {
		return SourceMedia{}, err
	}
	filename, err = sanitizeMediaFilename(filename, s.limits.MaxFilename)
	if err != nil {
		return SourceMedia{}, err
	}
	digest := sha256.Sum256(data)
	id := s.newID()
	if strings.TrimSpace(id) == "" {
		return SourceMedia{}, errors.New("source media id source returned an empty id")
	}
	objectID := s.newID()
	if strings.TrimSpace(objectID) == "" {
		return SourceMedia{}, errors.New("source media object id source returned an empty id")
	}
	key := "projects/" + projectID + "/media/" + objectID
	media := SourceMedia{
		ID:                 id,
		ProjectID:          projectID,
		SHA256:             hex.EncodeToString(digest[:]),
		SizeBytes:          int64(len(data)),
		MediaType:          info.MediaType,
		Width:              uint32(info.Width),
		Height:             uint32(info.Height),
		Orientation:        info.Orientation,
		ColorModel:         info.ColorModel,
		ColorProfileStatus: info.ColorProfileStatus,
		OriginalFilename:   filename,
		ImportedAt:         s.now().UTC(),
		ImportMethod:       "local_upload",
		Decoder:            "go-image/" + runtime.Version(),
		storageKey:         key,
	}
	if err := s.store.Put(ctx, key, bytes.NewReader(data), media.MediaType); err != nil {
		cleanupErr := s.cleanupObject(ctx, key)
		return SourceMedia{}, errors.Join(fmt.Errorf("store source media bytes: %w", err), cleanupErr)
	}
	created, inserted, err := s.repo.CreateSourceMedia(ctx, media)
	if err != nil {
		cleanupErr := s.cleanupObject(ctx, key)
		return SourceMedia{}, errors.Join(fmt.Errorf("persist source media metadata: %w", err), cleanupErr)
	}
	if inserted {
		return created, nil
	}
	cleanup := func() error { return s.cleanupObject(ctx, key) }
	reader, _, err := s.store.Get(ctx, created.storageKey)
	if err != nil {
		return SourceMedia{}, errors.Join(fmt.Errorf("read existing source media bytes: %w", err), cleanup())
	}
	winner, readErr := io.ReadAll(io.LimitReader(reader, s.limits.MaxBytes+1))
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		return SourceMedia{}, errors.Join(fmt.Errorf("verify duplicate source media bytes: %w", errors.Join(readErr, closeErr)), cleanup())
	}
	if !bytes.Equal(data, winner) {
		return SourceMedia{}, errors.Join(ErrSourceMediaDigestCollision{SHA256: media.SHA256}, cleanup())
	}
	if err := cleanup(); err != nil {
		return SourceMedia{}, fmt.Errorf("remove duplicate staging bytes: %w", err)
	}
	return created, nil
}

func (s *sourceMediaService) cleanupObject(ctx context.Context, key string) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()
	return s.store.Delete(cleanupCtx, key)
}

func (s *sourceMediaService) Get(ctx context.Context, projectID, id string) (SourceMedia, error) {
	if s.projects == nil || s.repo == nil {
		return SourceMedia{}, errors.New("source media service is not configured")
	}
	if _, err := s.projects.Get(ctx, strings.TrimSpace(projectID)); err != nil {
		return SourceMedia{}, err
	}
	return s.repo.GetSourceMedia(ctx, strings.TrimSpace(projectID), strings.TrimSpace(id))
}

func (s *sourceMediaService) Open(ctx context.Context, projectID, id string) (SourceMedia, io.ReadCloser, error) {
	media, err := s.Get(ctx, projectID, id)
	if err != nil {
		return SourceMedia{}, nil, err
	}
	reader, _, err := s.store.Get(ctx, media.storageKey)
	if err != nil {
		return SourceMedia{}, nil, fmt.Errorf("read source media object: %w", err)
	}
	data, readErr := io.ReadAll(io.LimitReader(reader, s.limits.MaxBytes+1))
	closeErr := reader.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return SourceMedia{}, nil, fmt.Errorf("read source media object: %w", err)
	}
	if int64(len(data)) != media.SizeBytes || int64(len(data)) > s.limits.MaxBytes {
		return SourceMedia{}, nil, errors.New("source media object size does not match metadata")
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != media.SHA256 {
		return SourceMedia{}, nil, errors.New("source media object digest does not match metadata")
	}
	return media, io.NopCloser(bytes.NewReader(data)), nil
}

func (s *sourceMediaService) List(ctx context.Context, projectID string, pageSize int32, token string) ([]SourceMedia, string, error) {
	if s.projects == nil || s.repo == nil {
		return nil, "", errors.New("source media service is not configured")
	}
	projectID = strings.TrimSpace(projectID)
	if _, err := s.projects.Get(ctx, projectID); err != nil {
		return nil, "", err
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}
	var after *SourceMedia
	if token != "" {
		raw, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil {
			return nil, "", ErrInvalidSourceMedia{Reason: "page_token is invalid"}
		}
		var cursor sourceMediaCursor
		if err := json.Unmarshal(raw, &cursor); err != nil || cursor.ID == "" || cursor.ImportedAt.IsZero() {
			return nil, "", ErrInvalidSourceMedia{Reason: "page_token is invalid"}
		}
		after = &SourceMedia{ID: cursor.ID, ImportedAt: cursor.ImportedAt}
	}
	rows, err := s.repo.ListSourceMedia(ctx, projectID, int(pageSize)+1, after)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(rows) > int(pageSize) {
		rows = rows[:pageSize]
		last := rows[len(rows)-1]
		raw, _ := json.Marshal(sourceMediaCursor{ID: last.ID, ImportedAt: last.ImportedAt})
		next = base64.RawURLEncoding.EncodeToString(raw)
	}
	return rows, next, nil
}

type sourceMediaCursor struct {
	ID         string    `json:"id"`
	ImportedAt time.Time `json:"imported_at"`
}

type sourceImageInfo struct {
	MediaType          string
	Width              int
	Height             int
	Orientation        string
	ColorModel         string
	ColorProfileStatus string
}

func inspectSourceImage(data []byte, limits SourceMediaLimits) (sourceImageInfo, error) {
	if len(data) == 0 {
		return sourceImageInfo{}, ErrInvalidSourceMedia{Reason: "file is empty"}
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return sourceImageInfo{}, ErrInvalidSourceMedia{Reason: "image is corrupt or truncated"}
	}
	mediaType := ""
	switch format {
	case "png":
		mediaType = "image/png"
	case "jpeg":
		mediaType = "image/jpeg"
	default:
		return sourceImageInfo{}, ErrInvalidSourceMedia{Reason: "only PNG and JPEG images are supported"}
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > limits.MaxSide || config.Height > limits.MaxSide || int64(config.Width)*int64(config.Height) > limits.MaxPixels {
		return sourceImageInfo{}, ErrInvalidSourceMedia{Reason: "image dimensions exceed the supported pixel limits"}
	}
	metadata, err := sourceimage.Inspect(data, format)
	if err != nil {
		if errors.Is(err, sourceimage.ErrUnsupportedImageMetadata) {
			return sourceImageInfo{}, ErrInvalidSourceMedia{Reason: "unsupported color metadata"}
		}
		return sourceImageInfo{}, ErrInvalidSourceMedia{Reason: "image orientation or profile metadata is malformed"}
	}
	if metadata.Orientation != 1 {
		return sourceImageInfo{}, ErrInvalidSourceMedia{Reason: "non-identity EXIF orientation is not supported"}
	}
	if metadata.HasProfile {
		return sourceImageInfo{}, ErrInvalidSourceMedia{Reason: "embedded color profiles are not supported"}
	}
	decoded, decodedFormat, err := image.Decode(bytes.NewReader(data))
	if err != nil || decoded == nil || decodedFormat != format {
		return sourceImageInfo{}, ErrInvalidSourceMedia{Reason: "image is corrupt or truncated"}
	}
	if decoded.Bounds().Dx() != config.Width || decoded.Bounds().Dy() != config.Height {
		return sourceImageInfo{}, ErrInvalidSourceMedia{Reason: "decoded image dimensions do not match the header"}
	}
	return sourceImageInfo{MediaType: mediaType, Width: config.Width, Height: config.Height, Orientation: "normal", ColorModel: sourceColorModel(decoded), ColorProfileStatus: metadata.ProfileStatus}, nil
}

func sourceColorModel(img image.Image) string {
	switch img.(type) {
	case *image.Gray, *image.Gray16:
		return "grayscale"
	case *image.Alpha:
		return "grayscale-alpha"
	case *image.Alpha16:
		return "grayscale-alpha-16"
	case *image.RGBA, *image.NRGBA:
		return "rgba"
	case *image.RGBA64, *image.NRGBA64:
		return "rgba-16"
	case *image.CMYK:
		return "cmyk"
	case *image.YCbCr:
		return "ycbcr"
	default:
		return "unknown"
	}
}

func sanitizeMediaFilename(filename string, maxRunes int) (string, error) {
	if strings.ContainsAny(filename, `/\\`) {
		return "", ErrInvalidSourceMedia{Reason: "filename must not contain path separators"}
	}
	var clean strings.Builder
	for _, r := range strings.TrimSpace(filename) {
		if unicode.IsControl(r) {
			clean.WriteRune('_')
			continue
		}
		clean.WriteRune(r)
	}
	out := strings.TrimSpace(clean.String())
	if out == "" {
		out = "upload"
	}
	runes := []rune(out)
	if len(runes) > maxRunes {
		return "", ErrInvalidSourceMedia{Reason: "filename exceeds the supported length"}
	}
	return out, nil
}
