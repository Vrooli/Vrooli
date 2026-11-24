//go:build audio_lab

package report

import (
	"sync/atomic"

	"audio-tools/internal/stt/pipeline"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

type speakerEvalCondition struct {
	ID                  string
	Config              pipeline.SpeakerConfig
	ExtractionEnabled   bool
	VerificationEnabled bool
	VerificationMode    sttv1.SpeakerMode
	Skipped             bool
	Note                string
}

type experimentProgress struct {
	total     int64
	completed atomic.Int64
	emit      func(int, string)
}
