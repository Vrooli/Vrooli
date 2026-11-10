package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type Reference struct {
	ID                 string      `json:"id"`
	Producer           string      `json:"producer"`
	Kind               string      `json:"kind"`
	SHA256             string      `json:"sha256"`
	Checksum           string      `json:"checksum"`
	SizeBytes          int64       `json:"size_bytes"`
	CreatedAt          time.Time   `json:"created_at"`
	RedactionVerified  bool        `json:"redaction_verified"`
	RecordingMethod    string      `json:"recording_method,omitempty"`
	EffectiveFPS       float64     `json:"effective_fps,omitempty"`
	AppliedRules       []string    `json:"applied_rules,omitempty"`
	OptedOut           bool        `json:"opted_out,omitempty"`
	ClaimClass         ClaimClass  `json:"claim_class,omitempty"`
	MinimumUsefulFPS   float64     `json:"minimum_useful_fps,omitempty"`
	Disposition        Disposition `json:"disposition,omitempty"`
	DispositionReason  string      `json:"disposition_reason,omitempty"`
	ContentVerified    bool        `json:"content_verified"`
	AudioTrackPresent  *bool       `json:"audio_track_present,omitempty"`
	AudioTrackCoverage string      `json:"audio_track_coverage,omitempty"`
}

// EvidenceSink is the producer boundary for evidence references. Callers must
// supply the result of the redaction policy; a claimed verification bit alone
// is not enough when the result was produced under a different policy.
type EvidenceSink struct {
	Policy Policy
}

func NewEvidenceSink(policy Policy) EvidenceSink { return EvidenceSink{Policy: policy} }

func (s EvidenceSink) Put(id string, raw []byte, redaction Result) (Reference, error) {
	if err := ValidatePolicy(s.Policy); err != nil {
		return Reference{}, err
	}
	if redaction.Requested && !redaction.Verified {
		return Reference{}, fmt.Errorf("capture redaction has not been verified")
	}
	if redaction.Policy != s.Policy {
		return Reference{}, fmt.Errorf("capture redaction policy does not match the producer policy")
	}
	return newReference(id, raw, redaction, "image", "", 0)
}

// EvidenceSink is a stateless value struct holding only its Policy: there is no
// cache, pool, or connection to amortize, so constructing one per call is free.
// ast-grep-ignore: no-discarded-stateful-helper
func NewReference(id string, raw []byte, redaction Result) (Reference, error) {
	return NewEvidenceSink(DefaultPolicy).Put(id, raw, redaction)
}

func NewClaimedVideoReference(id string, raw []byte, redaction Result, method string, fps float64, class ClaimClass) (Reference, error) {
	if redaction.Requested && !redaction.Verified {
		return Reference{}, fmt.Errorf("capture redaction has not been verified")
	}
	if redaction.Policy != DefaultPolicy {
		return Reference{}, fmt.Errorf("capture redaction policy does not match the producer policy")
	}
	content, err := ValidateVideoContent(raw)
	if err != nil {
		return Reference{}, err
	}
	if content.SampledFrames == 0 {
		return Reference{}, fmt.Errorf("video content verification failed: %s", content.Reason)
	}
	ref, err := newReference(id, raw, redaction, "video", method, fps)
	if err != nil {
		return Reference{}, err
	}
	a := Assess(class, fps)
	ref.ClaimClass = a.ClaimClass
	ref.MinimumUsefulFPS = a.MinimumUsefulFPS
	ref.Disposition = a.Disposition
	ref.DispositionReason = a.Reason
	ref.ContentVerified = content.Verified
	if !content.Verified {
		ref.Disposition = DispositionDegraded
		reason := content.Reason + "; visible content cannot be verified"
		if ref.DispositionReason != "" {
			reason += "; " + ref.DispositionReason
		}
		ref.DispositionReason = reason
	}
	// Track presence is a container fact, not proof of audible, synchronized,
	// or complete sound. Leave it unknown when inspection is unavailable.
	if present, err := InspectAudioTrack(raw); err == nil {
		ref.AudioTrackPresent = &present
	}
	return ref, nil
}

func NewLogReference(id string, raw []byte, redaction Result) (Reference, error) {
	return newReference(id, raw, redaction, "log", "", 0)
}

func newReference(id string, raw []byte, redaction Result, kind, method string, fps float64) (Reference, error) {
	if redaction.Requested && !redaction.Verified {
		return Reference{}, fmt.Errorf("capture redaction has not been verified")
	}
	sum := sha256.Sum256(raw)
	checksum := hex.EncodeToString(sum[:])
	return Reference{ID: id, Producer: "device-control", Kind: kind, SHA256: checksum, Checksum: checksum, SizeBytes: int64(len(raw)), CreatedAt: time.Now().UTC(), RedactionVerified: redaction.Verified, RecordingMethod: method, EffectiveFPS: fps, AppliedRules: redaction.Rules, OptedOut: redaction.OptedOut}, nil
}

type Recorder struct {
	Native bool
	FPS    float64
}

type RecorderMetadata struct {
	Method       string     `json:"recording_method"`
	EffectiveFPS float64    `json:"effective_fps"`
	CreatedAt    time.Time  `json:"created_at"`
	ClaimClass   ClaimClass `json:"claim_class"`
	Assessment   Assessment `json:"assessment"`
}

const MinimumUsefulFPS = 5.0

func (r Recorder) Metadata() (method string, fps float64) {
	if r.Native {
		return "native", r.FPS
	}
	if r.FPS <= 0 {
		return "synthesized", 5
	}
	return "synthesized", r.FPS
}
