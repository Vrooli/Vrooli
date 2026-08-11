package evidence

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"time"
)

// Video is a transport-neutral capture artifact. Native adapters may provide
// their own encoded bytes; synthesized captures use the deterministic GIF
// encoder below until a platform-native recorder is available.
type Video struct {
	Bytes           []byte  `json:"-"`
	RecordingMethod string  `json:"recording_method"`
	EffectiveFPS    float64 `json:"effective_fps"`
	FrameCount      int     `json:"frame_count"`
}

func EncodeFrames(frames []image.Image, fps float64) (Video, error) {
	if len(frames) == 0 {
		return Video{}, fmt.Errorf("at least one frame is required")
	}
	if err := ValidateEffectiveFPS(fps); err != nil {
		return Video{}, err
	}
	delay := int(100 / fps)
	if delay < 1 {
		delay = 1
	}
	encoded := &gif.GIF{}
	for _, frame := range frames {
		if frame == nil {
			return Video{}, fmt.Errorf("nil frame")
		}
		encoded.Image = append(encoded.Image, imageToPaletted(frame))
		encoded.Delay = append(encoded.Delay, delay)
	}
	var out bytes.Buffer
	if err := gif.EncodeAll(&out, encoded); err != nil {
		return Video{}, fmt.Errorf("encode synthesized capture: %w", err)
	}
	return Video{Bytes: out.Bytes(), RecordingMethod: "synthesized", EffectiveFPS: fps, FrameCount: len(frames)}, nil
}

func imageToPaletted(src image.Image) *image.Paletted {
	b := src.Bounds()
	dst := image.NewPaletted(b, palette)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(x, y, src.At(x, y))
		}
	}
	return dst
}

var palette = func() color.Palette {
	return color.Palette{color.Black, color.White, color.RGBA{R: 220, G: 38, B: 38, A: 255}, color.RGBA{R: 37, G: 99, B: 235, A: 255}}
}()

func NativeMetadata(effectiveFPS float64) (RecorderMetadata, error) {
	if err := ValidateEffectiveFPS(effectiveFPS); err != nil {
		return RecorderMetadata{}, err
	}
	return RecorderMetadata{Method: "native", EffectiveFPS: effectiveFPS, CreatedAt: time.Now().UTC()}, nil
}
