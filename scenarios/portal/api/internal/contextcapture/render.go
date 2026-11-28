package contextcapture

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
)

// Render produces only the selected pixels and clipped red annotations. It
// never changes the retained original. Annotation width is two source pixels.
func (s *Service) Render(ctx context.Context, owner, id string) (Document, []byte, string, error) {
	doc, original, err := s.Read(ctx, owner, id)
	if err != nil {
		return Document{}, nil, "", err
	}
	pixels, err := renderImage(ctx, doc, original)
	if err != nil {
		return Document{}, nil, "", err
	}
	// A deletion or expiry during rendering must not publish derived pixels.
	err = s.repo.WithRecord(ctx, owner, id, func(record Record) (Action, error) {
		if record.State != Ready || !s.now().Before(record.Document.ExpiresAt) || record.Document.OriginalSHA256 != doc.OriginalSHA256 {
			return Keep, ErrUnavailable
		}
		return Keep, ctx.Err()
	})
	if err != nil {
		return Document{}, nil, "", err
	}
	digest := sha256.Sum256(pixels)
	return doc, pixels, hex.EncodeToString(digest[:]), nil
}

func renderImage(ctx context.Context, doc Document, original []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	config, err := png.DecodeConfig(bytes.NewReader(original))
	if err != nil {
		return nil, ErrInvalid
	}
	region := doc.Region
	if config.Width < 1 || config.Height < 1 || config.Width*config.Height > MaxPixels || config.Width != int(doc.Source.Bounds.Width) || config.Height != int(doc.Source.Bounds.Height) || region.Width == 0 || region.Height == 0 || uint64(region.X)+uint64(region.Width) > uint64(config.Width) || uint64(region.Y)+uint64(region.Height) > uint64(config.Height) {
		return nil, ErrInvalid
	}
	source, err := png.Decode(bytes.NewReader(original))
	if err != nil {
		return nil, ErrInvalid
	}
	result := image.NewRGBA(image.Rect(0, 0, int(region.Width), int(region.Height)))
	draw.Draw(result, result.Bounds(), source, image.Pt(int(region.X), int(region.Y)), draw.Src)
	const maxSteps = 4 * 1024 * 1024
	stepsUsed := 0
	red := color.RGBA{R: 220, G: 38, B: 38, A: 255}
	if len(doc.Strokes) > 32 {
		return nil, ErrInvalid
	}
	for _, stroke := range doc.Strokes {
		if len(stroke) < 2 || len(stroke) > 256 {
			return nil, ErrInvalid
		}
		for _, p := range stroke {
			if math.IsNaN(p.X) || math.IsNaN(p.Y) || math.IsInf(p.X, 0) || math.IsInf(p.Y, 0) || p.X < 0 || p.Y < 0 || p.X > float64(config.Width) || p.Y > float64(config.Height) {
				return nil, ErrInvalid
			}
		}
		for i := 1; i < len(stroke); i++ {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			a, b := stroke[i-1], stroke[i]
			a.X -= float64(region.X)
			a.Y -= float64(region.Y)
			b.X -= float64(region.X)
			b.Y -= float64(region.Y)
			a, b, visible := clipSegment(a, b, -1, -1, float64(region.Width), float64(region.Height))
			if !visible {
				continue
			}
			steps := int(math.Ceil(math.Max(math.Abs(b.X-a.X), math.Abs(b.Y-a.Y))))
			stepsUsed += steps + 1
			if stepsUsed > maxSteps {
				return nil, ErrQuota
			}
			for step := 0; step <= steps; step++ {
				if step%1024 == 0 {
					if err := ctx.Err(); err != nil {
						return nil, err
					}
				}
				t := 0.0
				if steps > 0 {
					t = float64(step) / float64(steps)
				}
				x, y := int(math.Floor(a.X+(b.X-a.X)*t)), int(math.Floor(a.Y+(b.Y-a.Y)*t))
				for dy := -1; dy <= 0; dy++ {
					for dx := -1; dx <= 0; dx++ {
						result.SetRGBA(x+dx, y+dy, red)
					}
				}
			}
		}
	}
	var encoded boundedPNG
	if err = png.Encode(&encoded, result); err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	return encoded.Bytes(), nil
}

// Liang–Barsky clipping keeps work proportional to the selected region.
func clipSegment(a, b Point, left, top, right, bottom float64) (Point, Point, bool) {
	dx, dy := b.X-a.X, b.Y-a.Y
	lo, hi := 0.0, 1.0
	p := [4]float64{-dx, dx, -dy, dy}
	q := [4]float64{a.X - left, right - a.X, a.Y - top, bottom - a.Y}
	for i := range p {
		if p[i] == 0 {
			if q[i] < 0 {
				return a, b, false
			}
			continue
		}
		r := q[i] / p[i]
		if p[i] < 0 {
			lo = math.Max(lo, r)
		} else {
			hi = math.Min(hi, r)
		}
		if lo > hi {
			return a, b, false
		}
	}
	return Point{X: a.X + lo*dx, Y: a.Y + lo*dy}, Point{X: a.X + hi*dx, Y: a.Y + hi*dy}, true
}
