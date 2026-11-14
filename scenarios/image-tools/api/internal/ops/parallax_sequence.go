package ops

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
)

// ParallaxSequence is a lossless nearby-view animation of an explicitly
// prepared scene. Frame count/raster work/archive size are bounded separately.
type ParallaxSequence struct {
	Start  LayeredCamera `json:"start"`
	End    LayeredCamera `json:"end"`
	Frames uint32        `json:"frames"`
	FPS    uint32        `json:"fps"`
}

type boundedArchive struct{ bytes.Buffer }

func (b *boundedArchive) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 128<<20 {
		return 0, fmt.Errorf("ops: parallax archive exceeds 128 MiB")
	}
	return b.Buffer.Write(p)
}

func parallaxSequence(in RunInput) (RunResult, error) {
	p := in.Params
	q := p.Parallax
	if q == nil || q.Frames < 2 || q.Frames > 180 || (q.FPS != 24 && q.FPS != 30) {
		return RunResult{}, fmt.Errorf("ops: parallax requires 2..180 frames and 24 or 30 fps")
	}
	if p.Camera != nil {
		return RunResult{}, fmt.Errorf("ops: parallax scene must omit camera; use explicit start/end cameras")
	}
	if p.Format != "" && p.Format != "zip" {
		return RunResult{}, fmt.Errorf("ops: parallax produces a lossless PNG sequence ZIP")
	}
	plates, layers, w, h, err := prepareComposite(p)
	if err != nil {
		return RunResult{}, err
	}
	if int64(w)*int64(h)*int64(q.Frames) > 240_000_000 {
		return RunResult{}, fmt.Errorf("ops: parallax exceeds 240 million frame pixels; reduce dimensions or frame count")
	}
	if err = validateLayeredCamera(q.Start, w, h, plates); err != nil {
		return RunResult{}, err
	}
	if err = validateLayeredCamera(q.End, w, h, plates); err != nil {
		return RunResult{}, err
	}
	// Start/end opacity policy must agree. Otherwise interpolation would hide a
	// missing-scene check during part of the path.
	if q.Start.RequireOpaque != q.End.RequireOpaque {
		return RunResult{}, fmt.Errorf("ops: parallax start/end require_opaque policies must match")
	}
	ctx := in.Context
	if ctx == nil {
		return RunResult{}, fmt.Errorf("ops: parallax context is required")
	}
	var archive boundedArchive
	zw := zip.NewWriter(&archive)
	// PNG is already compressed. ZIP Store avoids a second compressor and keeps
	// archive bytes deterministic (fixed headers, order and no current timestamp).
	create := func(name string) (io.Writer, error) {
		return zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
	}
	type frameReceipt struct {
		Name   string        `json:"name"`
		SHA256 string        `json:"sha256"`
		Camera LayeredCamera `json:"camera"`
	}
	type sourceReceipt struct {
		Name     string  `json:"name"`
		SHA256   string  `json:"sha256"`
		Distance float64 `json:"distance"`
		Depth    int     `json:"depth"`
		Blend    string  `json:"blend"`
		Opacity  float64 `json:"opacity"`
	}
	frames := make([]frameReceipt, 0, q.Frames)
	sources := make([]sourceReceipt, 0, len(plates))
	for _, plate := range plates {
		sum := sha256.Sum256(plate.Image)
		sources = append(sources, sourceReceipt{plate.Name, hex.EncodeToString(sum[:]), *plate.Distance, plate.Depth, plate.Blend, plate.Opacity})
	}
	mix := func(a, b, t float64) float64 { return a + (b-a)*t }
	for n := uint32(0); n < q.Frames; n++ {
		if err = ctx.Err(); err != nil {
			return RunResult{}, err
		}
		t := float64(n) / float64(q.Frames-1)
		t = t * t * (3 - 2*t)
		a, b := q.Start, q.End
		fa, fb := a.FocalLengthPixels, b.FocalLengthPixels
		if fa == 0 {
			fa = float64(w)
		}
		if fb == 0 {
			fb = float64(w)
		}
		camera := LayeredCamera{X: mix(a.X, b.X, t), Y: mix(a.Y, b.Y, t), Z: mix(a.Z, b.Z, t), YawDegrees: mix(a.YawDegrees, b.YawDegrees, t), PitchDegrees: mix(a.PitchDegrees, b.PitchDegrees, t), FocalLengthPixels: mix(fa, fb, t), RequireOpaque: a.RequireOpaque}
		if err = validateLayeredCamera(camera, w, h, plates); err != nil {
			return RunResult{}, err
		}
		frameParams := *p
		frameParams.Camera = &camera
		im, renderErr := compositePrepared(ctx, plates, layers, &frameParams, w, h)
		if renderErr != nil {
			return RunResult{}, fmt.Errorf("ops: parallax frame %d: %w", n, renderErr)
		}
		name := fmt.Sprintf("frames/%06d.png", n)
		writer, writeErr := create(name)
		if writeErr != nil {
			return RunResult{}, writeErr
		}
		sum := sha256.New()
		encoder := png.Encoder{CompressionLevel: png.BestSpeed}
		if err = encoder.Encode(io.MultiWriter(writer, sum), im); err != nil {
			return RunResult{}, err
		}
		frames = append(frames, frameReceipt{name, hex.EncodeToString(sum.Sum(nil)), camera})
	}
	manifest := struct {
		Profile    string          `json:"profile"`
		Width      int             `json:"width"`
		Height     int             `json:"height"`
		FPS        uint32          `json:"fps"`
		Easing     string          `json:"easing"`
		Background string          `json:"background"`
		Sources    []sourceReceipt `json:"sources"`
		Frames     []frameReceipt  `json:"frames"`
	}{"image-tools-layered-parallax-v1", w, h, q.FPS, "smoothstep-output-ordinals", p.Background, sources, frames}
	mw, err := create("manifest.json")
	if err != nil {
		return RunResult{}, err
	}
	if err = json.NewEncoder(mw).Encode(manifest); err != nil {
		return RunResult{}, err
	}
	if err = zw.Close(); err != nil {
		return RunResult{}, err
	}
	return RunResult{Bytes: archive.Bytes(), Format: "zip", Mime: "application/zip", Width: w, Height: h}, nil
}

// ValidateParallaxRequest checks preparation and bounded path endpoints before
// the owner queues work. Every intermediate frame still checks coverage.
func ValidateParallaxRequest(p *Params) error {
	if p == nil || p.Parallax == nil {
		return fmt.Errorf("ops: prepared parallax parameters are required")
	}
	q := p.Parallax
	if q.Frames < 2 || q.Frames > 180 || (q.FPS != 24 && q.FPS != 30) {
		return fmt.Errorf("ops: parallax requires 2..180 frames and 24 or 30 fps")
	}
	if p.Camera != nil {
		return fmt.Errorf("ops: parallax scene must omit camera")
	}
	if p.Format != "" && p.Format != "zip" {
		return fmt.Errorf("ops: parallax produces a lossless PNG sequence ZIP")
	}
	plates, _, w, h, err := prepareComposite(p)
	if err != nil {
		return err
	}
	if int64(w)*int64(h)*int64(q.Frames) > 240_000_000 {
		return fmt.Errorf("ops: parallax exceeds 240 million frame pixels")
	}
	if err = validateLayeredCamera(q.Start, w, h, plates); err != nil {
		return err
	}
	if err = validateLayeredCamera(q.End, w, h, plates); err != nil {
		return err
	}
	if q.Start.RequireOpaque != q.End.RequireOpaque {
		return fmt.Errorf("ops: parallax opacity policies must match")
	}
	return nil
}
