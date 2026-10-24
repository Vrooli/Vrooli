package corpus

import (
	"testing"

	corpusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/corpus"
)

func TestSourceLabel(t *testing.T) {
	if got := sourceLabel(corpusv1.ClipSource_CLIP_SOURCE_SCRIPTED); got != "scripted" {
		t.Fatalf("scripted label = %q", got)
	}
	if got := sourceLabel(corpusv1.ClipSource_CLIP_SOURCE_FREE_FORM); got != "free_form" {
		t.Fatalf("free-form label = %q", got)
	}
}
