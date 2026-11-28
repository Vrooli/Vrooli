//go:build !audio_lab

package bootstrap

import (
	"context"

	"audio-tools/internal/audioformat"
	"audio-tools/internal/logx"
	sttpipeline "audio-tools/internal/stt/pipeline"
	inttts "audio-tools/internal/tts"
	"github.com/vrooli/api-core/database"
)

// buildExperimentSubsystem is intentionally empty in the product build. The
// product API has no experiment, corpus replay, or report-retention service;
// measures report zero when no product-owned count source exists.
func buildExperimentSubsystem(context.Context, Env, *database.RoutedDB, logx.Logger, *sttpipeline.WhisperTranscriber, *sttpipeline.SpeakerClient, *inttts.Service, *audioformat.Engine) experimentSubsystem {
	return experimentSubsystem{
		experimentCounter:     emptyExperimentCounter{},
		qualificationEvidence: emptyQualificationEvidenceStore{},
		corpusCounter:         corpusCounterProjection{},
	}
}
