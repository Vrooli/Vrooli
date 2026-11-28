package projects

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strings"
	"time"
	"unicode"

	"device-studio/internal/sourceimage"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/blobstore"
)

const (
	MaxSourceImageBytes         int64 = 24 << 20
	MaxSourceVideoBytes         int64 = 512 << 20
	MaxSourceMediaBytes         int64 = MaxSourceVideoBytes
	MaxSourceMediaPixels        int64 = 24_000_000
	MaxSourceMediaSide                = 20_000
	MaxSourceMediaFilenameRunes       = 128
	mediaSniffPrefixBytes             = 4096
	cleanupTimeout                    = 5 * time.Second
)

type MediaKind string

const (
	MediaKindImage MediaKind = "image"
	MediaKindVideo MediaKind = "video"
)

type Readiness string

const (
	ReadinessReady                  Readiness = "ready"
	ReadinessPending                Readiness = "pending"
	ReadinessRejected               Readiness = "rejected"
	ReadinessReconciliationRequired Readiness = "reconciliation_required"
)

type AudioInspectionFacts struct {
	Version uint32            `json:"version"`
	Streams []AudioStreamFact `json:"streams"`
}

type AudioStreamFact struct {
	OriginalStreamIndex uint32 `json:"original_stream_index"`
	Codec               string `json:"codec"`
}

type InspectionReceipt struct {
	Audio                     *AudioInspectionFacts `json:"audio,omitempty"`
	Version                   uint32                `json:"version"`
	ID                        string                `json:"id"`
	SHA256                    string                `json:"sha256"`
	SourceSHA256              string                `json:"source_sha256"`
	SourceSizeBytes           int64                 `json:"source_size_bytes"`
	JobID                     string                `json:"job_id"`
	AttemptID                 string                `json:"attempt_id"`
	ToolProfileID             string                `json:"tool_profile_id"`
	ToolProfileDigest         string                `json:"tool_profile_digest"`
	InspectionProtocolVersion uint32                `json:"inspection_protocol_version"`
	Classification            string                `json:"classification"`
	IssueCode                 string                `json:"issue_code,omitempty"`
	IssueMessage              string                `json:"issue_message,omitempty"`
	Container                 string                `json:"container,omitempty"`
	VideoStreamIndex          *int64                `json:"video_stream_index,omitempty"`
	Codec                     string                `json:"codec,omitempty"`
	CodedWidth                uint32                `json:"coded_width,omitempty"`
	CodedHeight               uint32                `json:"coded_height,omitempty"`
	VideoWidth                uint32                `json:"video_width,omitempty"`
	VideoHeight               uint32                `json:"video_height,omitempty"`
	SampleAspectNum           uint32                `json:"sample_aspect_num,omitempty"`
	SampleAspectDen           uint32                `json:"sample_aspect_den,omitempty"`
	RotationDegrees           int32                 `json:"rotation_degrees,omitempty"`
	ColorPolicy               string                `json:"color_policy,omitempty"`
	ColorPrimaries            string                `json:"color_primaries,omitempty"`
	ColorTransfer             string                `json:"color_transfer,omitempty"`
	ColorMatrix               string                `json:"color_matrix,omitempty"`
	ColorRange                string                `json:"color_range,omitempty"`
	TimeBaseNumerator         int64                 `json:"time_base_numerator,omitempty"`
	TimeBaseDenominator       int64                 `json:"time_base_denominator,omitempty"`
	StreamStartPTS            *int64                `json:"stream_start_pts,omitempty"`
	SourceEndPTS              *int64                `json:"source_end_pts,omitempty"`
	LastFrameDurationTicks    int64                 `json:"last_frame_duration_ticks,omitempty"`
	DurationNumerator         int64                 `json:"duration_numerator,omitempty"`
	DurationDenominator       int64                 `json:"duration_denominator,omitempty"`
	FrameCount                uint32                `json:"frame_count,omitempty"`
	StreamCount               uint32                `json:"stream_count,omitempty"`
	EOFPolicy                 string                `json:"eof_policy,omitempty"`
	ClockSHA256               string                `json:"clock_sha256,omitempty"`
	ClockArtifactRef          string                `json:"clock_artifact_ref,omitempty"`
	ClockArtifactSizeBytes    int64                 `json:"clock_artifact_size_bytes,omitempty"`
}

type SourceMedia struct {
	ID                     string
	ProjectID              string
	SHA256                 string
	SizeBytes              int64
	MediaType              string
	Kind                   MediaKind
	Readiness              Readiness
	Width                  uint32
	Height                 uint32
	RasterKnown            bool
	Orientation            string
	OrientationKnown       bool
	ColorModel             string
	ColorModelKnown        bool
	ColorProfileStatus     string
	ColorProfileKnown      bool
	InspectionJobID        string
	InspectionAttemptID    string
	InspectionIssueCode    string
	InspectionIssueMessage string
	InspectionReceipt      *InspectionReceipt
	OriginalFilename       string
	ImportedAt             time.Time
	ImportMethod           string
	Decoder                string
	storageKey             string
	inspectionReceiptKey   string
}

type SourceMediaRepository interface {
	CreateSourceMedia(context.Context, SourceMedia) (SourceMedia, bool, error)
	GetSourceMedia(context.Context, string, string) (SourceMedia, error)
	GetSourceMediaByDigest(context.Context, string, string) (SourceMedia, error)
	ListSourceMedia(context.Context, string, int, *SourceMedia) ([]SourceMedia, error)
	BindInspectionJob(context.Context, string, string, string, string) (SourceMedia, error)
	BindInspectionAttempt(context.Context, string, string, string, string, string, string) (SourceMedia, error)
	ProjectInspection(context.Context, InspectionProjection) error
}

type SourceMediaService interface {
	Import(context.Context, string, string, int64, io.Reader) (SourceMedia, error)
	BeginImport(context.Context, string, string, int64, io.Reader) (StagedSourceMediaImport, error)
	BindInspectionJob(context.Context, string, string, string, string) (SourceMedia, error)
	BindInspectionAttempt(context.Context, string, string, string, string, string, string) (SourceMedia, error)
	StoreInspectionArtifacts(context.Context, string, string, string, string, string, string, []byte, []byte) (string, string, error)
	OpenInspectionArtifacts(context.Context, string, string, string, string, string, string) ([]byte, []byte, error)
	ProjectInspection(context.Context, InspectionProjection) error
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
	MaxBytes      int64
	MaxVideoBytes int64
	MaxPixels     int64
	MaxSide       int
	MaxFilename   int
}

func DefaultSourceMediaLimits() SourceMediaLimits {
	return SourceMediaLimits{MaxBytes: MaxSourceImageBytes, MaxVideoBytes: MaxSourceVideoBytes, MaxPixels: MaxSourceMediaPixels, MaxSide: MaxSourceMediaSide, MaxFilename: MaxSourceMediaFilenameRunes}
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
	if deps.Limits.MaxVideoBytes <= 0 {
		deps.Limits.MaxVideoBytes = defaults.MaxVideoBytes
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
	staged, err := s.BeginImport(ctx, projectID, filename, declaredSize, body)
	if err != nil {
		return SourceMedia{}, err
	}
	return staged.Commit(ctx)
}

func (s *sourceMediaService) cleanupObject(ctx context.Context, key string) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()
	return s.store.Delete(cleanupCtx, key)
}

func (s *sourceMediaService) BindInspectionJob(ctx context.Context, projectID, mediaID, sourceSHA256, jobID string) (SourceMedia, error) {
	if s.repo == nil {
		return SourceMedia{}, errors.New("source media service is not configured")
	}
	if !validPathComponent(strings.TrimSpace(projectID)) || !validPathComponent(strings.TrimSpace(mediaID)) || !validPathComponent(strings.TrimSpace(jobID)) || !validSourceDigest(sourceSHA256) || strings.IndexFunc(projectID+mediaID+jobID, unicode.IsControl) >= 0 {
		return SourceMedia{}, ErrInspectionInvalid
	}
	return s.repo.BindInspectionJob(ctx, projectID, mediaID, sourceSHA256, jobID)
}

func (s *sourceMediaService) ProjectInspection(ctx context.Context, result InspectionProjection) error {
	if s.repo == nil {
		return errors.New("source media service is not configured")
	}
	if err := validateInspectionProjection(result); err != nil {
		return err
	}
	return s.repo.ProjectInspection(ctx, result)
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
	maxBytes := s.limits.MaxBytes
	if media.Kind == MediaKindVideo {
		maxBytes = s.limits.MaxVideoBytes
	}
	if media.SizeBytes <= 0 || media.SizeBytes > maxBytes || !validSourceDigest(media.SHA256) {
		_ = reader.Close()
		return SourceMedia{}, nil, errors.New("source media metadata has invalid size or digest")
	}
	return media, &verifiedMediaReader{reader: reader, expectedSize: media.SizeBytes, expectedSHA: media.SHA256, maxBytes: maxBytes}, nil
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
