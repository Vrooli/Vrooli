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

func clipToProto(c intcorpus.Clip) *corpusv1.Clip {
	tags := c.Tags
	if tags == nil {
		tags = []string{}
	}
	return &corpusv1.Clip{
		Id:            c.ID,
		ReferenceText: c.ReferenceText,
		Tags:          tags,
		DurationMs:    c.DurationMs,
		SampleRateHz:  protoint.FromInt(c.SampleRateHz),
		Format:        c.Format,
		BlobKey:       c.BlobKey,
		Source:        sourceToProto(c.Source),
		CreatedAt:     protomap.TimeToProto(c.CreatedAt),
	}
}
