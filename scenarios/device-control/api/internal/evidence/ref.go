package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type Reference struct {
	ID                string    `json:"id"`
	SHA256            string    `json:"sha256"`
	SizeBytes         int64     `json:"size_bytes"`
	CreatedAt         time.Time `json:"created_at"`
	RedactionVerified bool      `json:"redaction_verified"`
	RecordingMethod   string    `json:"recording_method,omitempty"`
	EffectiveFPS      float64   `json:"effective_fps,omitempty"`
}

func NewReference(id string, raw []byte, redaction Result) (Reference, error) {
	if !redaction.Verified {
		return Reference{}, fmt.Errorf("capture redaction has not been verified")
	}
	sum := sha256.Sum256(raw)
	return Reference{ID: id, SHA256: hex.EncodeToString(sum[:]), SizeBytes: int64(len(raw)), CreatedAt: time.Now().UTC(), RedactionVerified: true}, nil
}

type Recorder struct {
	Native bool
	FPS    float64
}

type RecorderMetadata struct {
	Method       string    `json:"recording_method"`
	EffectiveFPS float64   `json:"effective_fps"`
	CreatedAt    time.Time `json:"created_at"`
}

const MinimumUsefulFPS = 5.0

func ValidateEffectiveFPS(fps float64) error {
	if fps < MinimumUsefulFPS {
		return fmt.Errorf("recording effective frame rate %.2f is below the useful evidence threshold %.2f", fps, MinimumUsefulFPS)
	}
	return nil
}

func (r Recorder) Metadata() (method string, fps float64) {
	if r.Native {
		return "native", r.FPS
	}
	if r.FPS <= 0 {
		return "synthesized", 5
	}
	return "synthesized", r.FPS
}
