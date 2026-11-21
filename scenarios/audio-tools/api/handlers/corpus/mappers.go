package corpus

import (
	intcorpus "audio-tools/internal/corpus"
	"audio-tools/internal/protoint"
	"audio-tools/internal/protomap"

	corpusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/corpus"
)

func sourceToProto(s intcorpus.Source) corpusv1.ClipSource {
	switch s.Normalize() {
	case intcorpus.SourceScripted:
		return corpusv1.ClipSource_CLIP_SOURCE_SCRIPTED
	default:
		return corpusv1.ClipSource_CLIP_SOURCE_FREE_FORM
	}
}

func sourceFromProto(s corpusv1.ClipSource) intcorpus.Source {
	switch s {
	case corpusv1.ClipSource_CLIP_SOURCE_SCRIPTED:
		return intcorpus.SourceScripted
	default:
		return intcorpus.SourceFreeForm
	}
}

func splitFromProto(s corpusv1.CorpusSplit) intcorpus.Split {
	switch s {
	case corpusv1.CorpusSplit_CORPUS_SPLIT_DEV:
		return intcorpus.SplitDev
	case corpusv1.CorpusSplit_CORPUS_SPLIT_HELD_OUT:
		return intcorpus.SplitHeldOut
	default:
		return intcorpus.SplitUnspecified
	}
}

func noiseConditionFromProto(s corpusv1.NoiseCondition) intcorpus.NoiseCondition {
	switch s {
	case corpusv1.NoiseCondition_NOISE_CONDITION_CLEAN:
		return intcorpus.NoiseClean
	case corpusv1.NoiseCondition_NOISE_CONDITION_NOISY:
		return intcorpus.NoiseNoisy
	default:
		return intcorpus.NoiseUnspecified
	}
}

func clipToProto(c intcorpus.Clip) *corpusv1.Clip {
	tags := c.Tags
	if tags == nil {
		tags = []string{}
	}
	return &corpusv1.Clip{
		Id:               c.ID,
		ReferenceText:    c.ReferenceText,
		Tags:             tags,
		DurationMs:       c.DurationMs,
		SampleRateHz:     protoint.FromInt(c.SampleRateHz),
		Format:           c.Format,
		BlobKey:          c.BlobKey,
		Source:           sourceToProto(c.Source),
		CreatedAt:        protomap.TimeToProto(c.CreatedAt),
		CorpusRevision:   c.CorpusRevision,
		LicenseReference: c.LicenseReference,
		SpeakerPseudonym: c.SpeakerPseudonym,
		Split:            splitToProto(c.Split),
		NoiseCondition:   noiseConditionToProto(c.NoiseCondition),
		SnrDb:            c.SNRDB,
		ContentSha256:    c.ContentSHA256,
	}
}

func splitToProto(s intcorpus.Split) corpusv1.CorpusSplit {
	switch s {
	case intcorpus.SplitDev:
		return corpusv1.CorpusSplit_CORPUS_SPLIT_DEV
	case intcorpus.SplitHeldOut:
		return corpusv1.CorpusSplit_CORPUS_SPLIT_HELD_OUT
	default:
		return corpusv1.CorpusSplit_CORPUS_SPLIT_UNSPECIFIED
	}
}

func noiseConditionToProto(s intcorpus.NoiseCondition) corpusv1.NoiseCondition {
	switch s {
	case intcorpus.NoiseClean:
		return corpusv1.NoiseCondition_NOISE_CONDITION_CLEAN
	case intcorpus.NoiseNoisy:
		return corpusv1.NoiseCondition_NOISE_CONDITION_NOISY
	default:
		return corpusv1.NoiseCondition_NOISE_CONDITION_UNSPECIFIED
	}
}
