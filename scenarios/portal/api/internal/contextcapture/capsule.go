// Package contextcapture owns bounded image-context documents. Source metadata
// is provenance supplied at import, never authority to actuate a desktop.
package contextcapture

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image/png"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/targetmodel"
)

const (
	MaxImageBytes = 32 * 1024 * 1024
	MaxPixels     = 16 * 1024 * 1024
)

var ErrInvalid = errors.New("invalid captured context")

type Bounds struct {
	X      int32  `json:"x"`
	Y      int32  `json:"y"`
	Width  uint32 `json:"width"`
	Height uint32 `json:"height"`
}
type Region struct {
	X      uint32 `json:"x"`
	Y      uint32 `json:"y"`
	Width  uint32 `json:"width"`
	Height uint32 `json:"height"`
}
type (
	Point struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	Source struct {
		Surface          targetmodel.SurfaceRef `json:"surface"`
		CaptureID        string                 `json:"capture_id"`
		DisplayID        string                 `json:"display_id"`
		GeometryRevision string                 `json:"geometry_revision"`
		CapturedAt       time.Time              `json:"captured_at"`
		Bounds           Bounds                 `json:"bounds"`
	}
)

type Import struct {
	RequestID string
	Source    Source
	Region    Region
	Strokes   [][]Point
	PNG       []byte
}

// Document retains the full original raster separately from the crop/marks.
// Coordinates always refer to that raster, including strokes outside the crop.
// Owner is supplied by the calling service's authenticated principal.
type Document struct {
	RequestID      string    `json:"request_id"`
	ImportDigest   string    `json:"import_digest"`
	ID             string    `json:"id"`
	Owner          string    `json:"owner"`
	Source         Source    `json:"source"`
	Region         Region    `json:"region"`
	Strokes        [][]Point `json:"strokes"`
	OriginalSHA256 string    `json:"original_sha256"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}
type boundedPNG struct{ bytes.Buffer }

func (b *boundedPNG) Write(p []byte) (int, error) {
	if len(p) > MaxImageBytes-b.Len() {
		return 0, ErrInvalid
	}
	return b.Buffer.Write(p)
}

// Prepare validates and normalizes imported pixels before any durable write.
// Retention is explicit and bounded independently from the desktop input lease.
// The returned byte slice and strokes do not alias the caller's buffers.
func Prepare(owner string, input Import, now time.Time, retention time.Duration) (Document, []byte, error) {
	if owner == "" || len(owner) > 256 || now.IsZero() || retention <= 0 || retention > 24*time.Hour || input.Source.Surface.Validate() != nil {
		return Document{}, nil, ErrInvalid
	}
	requestID := input.RequestID
	if requestID == "" {
		requestID = uuid.NewString()
	}
	requestUUID, requestErr := uuid.Parse(requestID)
	if requestErr != nil || requestUUID == uuid.Nil || requestUUID.String() != requestID {
		return Document{}, nil, ErrInvalid
	}
	source := input.Source
	b := source.Bounds
	r := input.Region
	id, err := uuid.Parse(source.CaptureID)
	if err != nil || id == uuid.Nil || id.String() != source.CaptureID || source.DisplayID == "" || len(source.DisplayID) > 128 || source.GeometryRevision == "" || len(source.GeometryRevision) > 128 || source.CapturedAt.IsZero() || source.CapturedAt.After(now) || now.Sub(source.CapturedAt) > 30*time.Second {
		return Document{}, nil, ErrInvalid
	}
	if b.Width == 0 || b.Height == 0 || b.Width > 65535 || b.Height > 65535 || uint64(b.Width)*uint64(b.Height) > MaxPixels || int64(b.X)+int64(b.Width) > math.MaxInt32 || int64(b.Y)+int64(b.Height) > math.MaxInt32 || r.Width == 0 || r.Height == 0 || uint64(r.X)+uint64(r.Width) > uint64(b.Width) || uint64(r.Y)+uint64(r.Height) > uint64(b.Height) || len(input.Strokes) > 32 {
		return Document{}, nil, ErrInvalid
	}
	strokes := make([][]Point, len(input.Strokes))
	for i, stroke := range input.Strokes {
		if len(stroke) < 2 || len(stroke) > 256 {
			return Document{}, nil, ErrInvalid
		}
		strokes[i] = make([]Point, len(stroke))
		for j, p := range stroke {
			if math.IsNaN(p.X) || math.IsNaN(p.Y) || math.IsInf(p.X, 0) || math.IsInf(p.Y, 0) || p.X < 0 || p.Y < 0 || p.X > float64(b.Width) || p.Y > float64(b.Height) {
				return Document{}, nil, ErrInvalid
			}
			strokes[i][j] = p
		}
	}
	if len(input.PNG) == 0 || len(input.PNG) > MaxImageBytes {
		return Document{}, nil, ErrInvalid
	}
	config, err := png.DecodeConfig(bytes.NewReader(input.PNG))
	if err != nil || config.Width != int(b.Width) || config.Height != int(b.Height) {
		return Document{}, nil, ErrInvalid
	}
	raster, err := png.Decode(bytes.NewReader(input.PNG))
	if err != nil {
		return Document{}, nil, ErrInvalid
	}
	// Re-encoding strips untrusted ancillary/trailing payloads while preserving
	// source pixels. Hash and retention cover this normalized original image.
	var normalized boundedPNG
	if err = png.Encode(&normalized, raster); err != nil {
		return Document{}, nil, ErrInvalid
	}
	digest := sha256.Sum256(normalized.Bytes())
	source.CapturedAt = source.CapturedAt.UTC()
	canonical, err := json.Marshal(struct {
		Source    Source
		Region    Region
		Strokes   [][]Point
		Image     string
		Retention int64
	}{source, r, strokes, hex.EncodeToString(digest[:]), int64(retention)})
	if err != nil {
		return Document{}, nil, ErrInvalid
	}
	intentDigest := sha256.Sum256(canonical)
	return Document{RequestID: requestID, ImportDigest: hex.EncodeToString(intentDigest[:]), ID: uuid.NewString(), Owner: owner, Source: source, Region: r, Strokes: strokes, OriginalSHA256: hex.EncodeToString(digest[:]), CreatedAt: now, ExpiresAt: now.Add(retention)}, normalized.Bytes(), nil
}
