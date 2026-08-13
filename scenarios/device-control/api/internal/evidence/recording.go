package evidence

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type ClaimClass string

const (
	ClaimStatic     ClaimClass = "static"
	ClaimTransition ClaimClass = "transition"
	ClaimAnimation  ClaimClass = "animation"
)

type Disposition string

const (
	DispositionPassed   Disposition = "passed"
	DispositionDegraded Disposition = "degraded"
)

func MinimumUsefulFPSFor(class ClaimClass) float64 {
	switch class {
	case ClaimStatic:
		return 1
	case ClaimAnimation:
		return 15
	case ClaimTransition:
		fallthrough
	default:
		return 5
	}
}

type Assessment struct {
	ClaimClass       ClaimClass  `json:"claim_class"`
	EffectiveFPS     float64     `json:"effective_fps"`
	MinimumUsefulFPS float64     `json:"minimum_useful_fps"`
	Disposition      Disposition `json:"disposition"`
	Reason           string      `json:"reason,omitempty"`
}

// MeasureVideo derives the delivered frame rate from the encoded artifact.
// Requested recorder settings are intentionally not accepted as evidence.
func MeasureVideo(raw []byte) (frameCount int, duration time.Duration, effectiveFPS float64, err error) {
	file, err := os.CreateTemp("", "device-control-measure-*.mp4")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("create measurement input: %w", err)
	}
	path := file.Name()
	defer os.Remove(path)
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return 0, 0, 0, fmt.Errorf("protect measurement input: %w", err)
	}
	if _, err := file.Write(raw); err != nil {
		_ = file.Close()
		return 0, 0, 0, fmt.Errorf("write measurement input: %w", err)
	}
	if err := file.Close(); err != nil {
		return 0, 0, 0, fmt.Errorf("close measurement input: %w", err)
	}
	out, err := exec.Command("ffprobe", "-v", "error", "-count_frames", "-select_streams", "v:0", "-show_entries", "stream=nb_read_frames,duration", "-of", "csv=p=0:s=,", path).Output()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("measure video with ffprobe: %w", err)
	}
	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) != 2 {
		return 0, 0, 0, fmt.Errorf("measure video returned malformed output %q", strings.TrimSpace(string(out)))
	}
	durationSeconds, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil || durationSeconds <= 0 {
		return 0, 0, 0, fmt.Errorf("measure video returned invalid duration %q", parts[0])
	}
	frameCount, err = strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || frameCount <= 0 {
		return 0, 0, 0, fmt.Errorf("measure video returned invalid frame count %q", parts[1])
	}
	duration = time.Duration(durationSeconds * float64(time.Second))
	return frameCount, duration, float64(frameCount) / durationSeconds, nil
}

func Assess(class ClaimClass, effectiveFPS float64) Assessment {
	if class == "" {
		class = ClaimTransition
	}
	minimum := MinimumUsefulFPSFor(class)
	a := Assessment{ClaimClass: class, EffectiveFPS: effectiveFPS, MinimumUsefulFPS: minimum, Disposition: DispositionPassed}
	if effectiveFPS < minimum || math.IsNaN(effectiveFPS) || math.IsInf(effectiveFPS, 0) {
		a.Disposition = DispositionDegraded
		a.Reason = fmt.Sprintf("effective frame rate %.2f is below the %s claim minimum of %.2f fps", effectiveFPS, class, minimum)
	}
	return a
}

// Video is a transport-neutral capture artifact. Native adapters may provide
// their own encoded bytes; synthesized captures use the deterministic GIF
// encoder below until a platform-native recorder is available.
type Video struct {
	Bytes           []byte     `json:"-"`
	RecordingMethod string     `json:"recording_method"`
	EffectiveFPS    float64    `json:"effective_fps"`
	FrameCount      int        `json:"frame_count"`
	ClaimClass      ClaimClass `json:"claim_class"`
	Assessment      Assessment `json:"assessment"`
}

func EncodeFrames(frames []image.Image, fps float64) (Video, error) {
	return EncodeFramesForClaim(frames, fps, ClaimTransition)
}

func EncodeFramesForClaim(frames []image.Image, fps float64, class ClaimClass) (Video, error) {
	if len(frames) == 0 {
		return Video{}, fmt.Errorf("at least one frame is required")
	}
	if fps <= 0 || math.IsNaN(fps) || math.IsInf(fps, 0) {
		return Video{}, fmt.Errorf("effective frame rate must be positive")
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
	return Video{Bytes: out.Bytes(), RecordingMethod: "synthesized", EffectiveFPS: fps, FrameCount: len(frames), ClaimClass: class, Assessment: Assess(class, fps)}, nil
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
	return NativeMetadataForClaim(effectiveFPS, ClaimTransition)
}

func NativeMetadataForClaim(effectiveFPS float64, class ClaimClass) (RecorderMetadata, error) {
	if effectiveFPS <= 0 || math.IsNaN(effectiveFPS) || math.IsInf(effectiveFPS, 0) {
		return RecorderMetadata{}, fmt.Errorf("effective frame rate must be positive")
	}
	a := Assess(class, effectiveFPS)
	return RecorderMetadata{Method: "native", EffectiveFPS: effectiveFPS, CreatedAt: time.Now().UTC(), ClaimClass: class, Assessment: a}, nil
}
