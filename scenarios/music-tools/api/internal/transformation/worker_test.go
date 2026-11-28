package transformation

import (
	"math"
	"strings"
	"testing"
)

func validTransformRequest() Request {
	return Request{
		Kind: "sample", SourceBlobRef: "out/take-job/take-01.wav",
		SourceTrackSHA256:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SourceStartSeconds: 2, SourceEndSeconds: 6, RepeatCount: 3,
		InheritedRightsScope: "personal_only",
	}
}

func TestValidateTransformAcceptsBoundedManagedSampleAndRemix(t *testing.T) {
	sample := validTransformRequest()
	if err := validateTransform(sample); err != nil {
		t.Fatalf("valid sample rejected: %v", err)
	}
	remix := sample
	remix.Kind = "remix"
	remix.RepeatCount = 0
	if err := validateTransform(remix); err != nil {
		t.Fatalf("valid remix rejected: %v", err)
	}
}

func TestValidateTransformRejectsUnboundOrUnsafeRequests(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Request)
	}{
		{name: "unsupported operation", mutate: func(r *Request) { r.Kind = "compose" }},
		{name: "bad source hash", mutate: func(r *Request) { r.SourceTrackSHA256 = "not-a-hash" }},
		{name: "unmanaged path", mutate: func(r *Request) { r.SourceBlobRef = "/tmp/operator.wav" }},
		{name: "path traversal", mutate: func(r *Request) { r.SourceBlobRef = "out/../private.wav" }},
		{name: "zero range", mutate: func(r *Request) { r.SourceEndSeconds = r.SourceStartSeconds }},
		{name: "excerpt too short for stable edit", mutate: func(r *Request) { r.SourceEndSeconds = r.SourceStartSeconds + MinExcerptSecs - 0.01 }},
		{name: "non-finite range", mutate: func(r *Request) { r.SourceEndSeconds = math.Inf(1) }},
		{name: "range above maximum", mutate: func(r *Request) { r.SourceStartSeconds = 0; r.SourceEndSeconds = MaxExcerptSecs + 1 }},
		{name: "sample repeats below minimum", mutate: func(r *Request) { r.RepeatCount = 1 }},
		{name: "sample repeats above maximum", mutate: func(r *Request) { r.RepeatCount = 9 }},
		{name: "remix cannot repeat", mutate: func(r *Request) { r.Kind = "remix" }},
		{name: "invalid rights scope", mutate: func(r *Request) { r.InheritedRightsScope = "unrestricted" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := validTransformRequest()
			tc.mutate(&request)
			if err := validateTransform(request); err == nil {
				t.Fatal("unsafe request accepted")
			}
		})
	}
}

func TestParseAudioDurationRejectsMissingOrNonFiniteStreamBounds(t *testing.T) {
	for _, input := range []string{"", "N/A", "NaN", "Inf", "0", "12.5\n20.0"} {
		if _, err := parseAudioDuration(input); err == nil {
			t.Errorf("duration %q unexpectedly accepted", input)
		}
	}
	duration, err := parseAudioDuration("12.500000\n")
	if err != nil || duration != 12.5 {
		t.Fatalf("valid stream duration rejected: duration=%v err=%v", duration, err)
	}
}

func TestTransformationArgsSelectOnlyTheRequestedExcerpt(t *testing.T) {
	input := validTransformRequest()
	sampleArgs := strings.Join(transformationArgs(input, "source.audio", "derived.wav"), " ")
	for _, required := range []string{"atrim=start=2.000000:end=6.000000", "asplit=3", "sample0_raw]afade=t=in", "concat=n=3", "afade=t=out", "-map_metadata -1", "-ac 2", "-ar 48000", "pcm_s16le"} {
		if !strings.Contains(sampleArgs, required) {
			t.Fatalf("sample command omitted %q: %s", required, sampleArgs)
		}
	}
	input.Kind, input.RepeatCount = "remix", 0
	remixArgs := strings.Join(transformationArgs(input, "source.audio", "derived.wav"), " ")
	for _, required := range []string{"atrim=start=2.000000:end=6.000000", "highpass=f=80", "aecho=", "alimiter=", "-ac 2", "-ar 48000", "pcm_s16le"} {
		if !strings.Contains(remixArgs, required) {
			t.Fatalf("remix command omitted %q: %s", required, remixArgs)
		}
	}
}
