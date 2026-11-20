package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Metadata mirrors the proto AudioMetadata shape so handlers can copy
// fields without an extra DTO.
type Metadata struct {
	DurationSeconds float64
	SampleRate      int32
	Channels        int32
	Bitrate         int32
	Codec           string
	Format          string
	Tags            map[string]string
}

// Trim returns audio[start:end] re-encoded into outFormat (defaults to
// the input format when outFormat is empty). end==0 means EOF.
//
// Seeking semantics: `-ss` sits before `-i`, so it is an input seek
// (ffmpeg decodes and discards for non-seekable pipes). The remaining
// window is expressed as an output `-t` *duration* rather than `-to`,
// because an output `-to` is measured on the output timeline, which is
// reset by the input seek and would therefore extend the clip instead of
// ending it at end_seconds.
func Trim(ctx context.Context, audio []byte, format string, start, end float64, outFormat string) ([]byte, error) {
	// A start beyond the real end otherwise yields a header-only WAV with a
	// successful status. Probe best-effort: when ffprobe is unavailable,
	// ffmpeg remains the authority and trimming still works.
	inputDuration := 0.0
	if meta, err := Probe(ctx, audio); err == nil && isFinitePositive(meta.DurationSeconds) {
		inputDuration = meta.DurationSeconds
	}
	return trimWithDuration(ctx, audio, format, start, end, outFormat, inputDuration)
}

// trimWithDuration is Trim with the input duration already measured (0 =
// unknown). Split calls it once per chunk so a hundred-chunk request does not
// re-probe the same input a hundred times.
func trimWithDuration(ctx context.Context, audio []byte, format string, start, end float64, outFormat string, inputDuration float64) ([]byte, error) {
	if err := validTime(start, "start_seconds"); err != nil {
		return nil, err
	}
	if err := validTime(end, "end_seconds"); err != nil {
		return nil, err
	}
	if end > 0 && end <= start {
		return nil, fmt.Errorf("%w: end_seconds (%g) must be greater than start_seconds (%g)", ErrInvalidArgument, end, start)
	}
	if outFormat == "" {
		outFormat = format
		if outFormat == "" {
			outFormat = "wav"
		}
	}
	if isFinitePositive(inputDuration) {
		if start >= inputDuration {
			return nil, fmt.Errorf("%w: start_seconds (%g) is beyond the input duration (%g s)", ErrInvalidArgument, start, inputDuration)
		}
		if end > inputDuration {
			end = inputDuration
		}
	}
	args := []string{"-ss", strconv.FormatFloat(start, 'f', -1, 64), "-i", "pipe:0"}
	if end > start {
		duration := end - start
		args = append(args, "-t", strconv.FormatFloat(duration, 'f', -1, 64))
	}
	args = append(args, "-f", muxerFor(outFormat), "pipe:1")
	return runFfmpeg(ctx, audio, args...)
}

// Fade applies fade-in and fade-out windows.
//
// A fade-out needs the stream's duration to know where to start; the
// `afade` filter otherwise anchors at t=0 and silences everything after
// the fade window. When ffprobe reports a duration we anchor the fade at
// duration-fadeOut. When duration is unavailable (ffprobe missing, or a
// stream without duration metadata) we fall back to the dependency-free
// reverse/fade/reverse idiom, which buffers the decoded stream but is
// always correct.
func Fade(ctx context.Context, audio []byte, format string, fadeIn, fadeOut float64, outFormat string) ([]byte, error) {
	if err := validTime(fadeIn, "fade_in_seconds"); err != nil {
		return nil, err
	}
	if err := validTime(fadeOut, "fade_out_seconds"); err != nil {
		return nil, err
	}
	if outFormat == "" {
		outFormat = format
		if outFormat == "" {
			outFormat = "wav"
		}
	}
	var filters []string
	if fadeIn > 0 {
		filters = append(filters, fmt.Sprintf("afade=t=in:st=0:d=%g", fadeIn))
	}
	if fadeOut > 0 {
		start, ok := fadeOutStart(ctx, audio, fadeOut)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if ok {
			filters = append(filters, fmt.Sprintf("afade=t=out:st=%g:d=%g", start, fadeOut))
		} else {
			filters = append(filters,
				"areverse",
				fmt.Sprintf("afade=t=in:st=0:d=%g", fadeOut),
				"areverse",
			)
		}
	}
	args := []string{"-i", "pipe:0"}
	if len(filters) > 0 {
		args = append(args, "-af", strings.Join(filters, ","))
	}
	args = append(args, "-f", muxerFor(outFormat), "pipe:1")
	return runFfmpeg(ctx, audio, args...)
}

// fadeOutStart reports the offset at which a fade-out of the given
// duration should begin. ok is false when the duration cannot be
// determined, in which case callers must use a duration-independent
// fallback instead of misplacing the fade at t=0.
func fadeOutStart(ctx context.Context, audio []byte, fadeOut float64) (start float64, ok bool) {
	meta, err := Probe(ctx, audio)
	if err != nil || meta.DurationSeconds <= 0 {
		return 0, false
	}
	start = meta.DurationSeconds - fadeOut
	if start < 0 {
		start = 0
	}
	return start, true
}

// Volume scales the input by gainDB.
func Volume(ctx context.Context, audio []byte, format string, gainDB float64, outFormat string) ([]byte, error) {
	if math.IsNaN(gainDB) || math.IsInf(gainDB, 0) {
		return nil, fmt.Errorf("%w: gain_db must be a finite value", ErrInvalidArgument)
	}
	if outFormat == "" {
		outFormat = format
		if outFormat == "" {
			outFormat = "wav"
		}
	}
	return runFfmpeg(ctx, audio,
		"-i", "pipe:0",
		"-af", fmt.Sprintf("volume=%gdB", gainDB),
		"-f", muxerFor(outFormat), "pipe:1",
	)
}

// Normalize applies loudnorm (EBU R128 by default) or peak/rms
// normalisation. It keeps the original signature; callers that also need
// the measured integrated loudness use NormalizeMeasured.
func Normalize(ctx context.Context, audio []byte, format, method string, targetLUFS float64, outFormat string) ([]byte, error) {
	out, _, err := NormalizeMeasured(ctx, audio, format, method, targetLUFS, outFormat)
	return out, err
}

// Probe runs ffprobe -of json and returns parsed metadata.
func Probe(ctx context.Context, audio []byte) (Metadata, error) {
	if !hasFfprobe() {
		return Metadata{}, ErrFFmpegMissing
	}
	out, err := runFfprobeJSON(ctx, audio)
	if err != nil {
		return Metadata{}, err
	}
	var doc struct {
		Streams []struct {
			SampleRate string `json:"sample_rate"`
			Channels   int32  `json:"channels"`
			BitRate    string `json:"bit_rate"`
			CodecName  string `json:"codec_name"`
		} `json:"streams"`
		Format struct {
			Duration   string            `json:"duration"`
			FormatName string            `json:"format_name"`
			BitRate    string            `json:"bit_rate"`
			Tags       map[string]string `json:"tags"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return Metadata{}, err
	}
	m := Metadata{
		Format: doc.Format.FormatName,
		Tags:   doc.Format.Tags,
	}
	m.DurationSeconds = looseFloat(doc.Format.Duration)
	if !isFinitePositive(m.DurationSeconds) {
		// "N/A" or a non-finite sentinel: report unknown rather than
		// letting an Inf duration drive downstream cut loops.
		m.DurationSeconds = 0
	}
	m.Bitrate = looseInt32(doc.Format.BitRate)
	if len(doc.Streams) > 0 {
		s := doc.Streams[0]
		m.SampleRate = looseInt32(s.SampleRate)
		m.Channels = s.Channels
		m.Codec = s.CodecName
		if m.Bitrate == 0 {
			m.Bitrate = looseInt32(s.BitRate)
		}
	}
	return m, nil
}

// DurationSeconds reports the duration of an encoded payload, or 0 when it
// cannot be determined.
//
// It probes the OUTPUT bytes rather than deriving the value from input
// metadata because the final duration belongs to the produced
// container/encoder pair: encoder priming and padding, and length-changing
// filter graphs, mean input timing is not the output truth. The probe is
// best-effort — a missing ffprobe, a probe failure, or a container without
// duration metadata yields 0 and never fails the transform that already
// succeeded.
func DurationSeconds(ctx context.Context, audio []byte) float64 {
	meta, err := Probe(ctx, audio)
	if err != nil || !isFinitePositive(meta.DurationSeconds) {
		return 0
	}
	return meta.DurationSeconds
}

// looseFloat parses an ffprobe numeric field. ffprobe emits "N/A" for
// values it cannot determine; treating that as zero keeps metadata
// extraction working instead of failing the whole request.
func looseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// looseInt32 is looseFloat for integer-valued fields (sample rate,
// bitrate). Values outside int32 or unparseable strings collapse to 0.
func looseInt32(s string) int32 {
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0
	}
	return int32(v)
}

func isFinitePositive(v float64) bool {
	return v > 0 && !math.IsInf(v, 0) && !math.IsNaN(v)
}

// maxSplitChunks bounds subprocess fan-out. Split issues one ffmpeg
// process per chunk, so a tiny chunk_seconds over a long file must be
// rejected rather than turning one request into tens of thousands of
// subprocesses.
const maxSplitChunks = 10000

// Split slices the input into chunks. If chunkSeconds > 0 the audio is
// cut into equal slices; otherwise boundariesSeconds defines explicit
// cut points (always combined with start=0 and end=EOF). When neither is
// supplied the whole input is returned as a single chunk.
func Split(ctx context.Context, audio []byte, format string, chunkSeconds float64, boundaries []float64, outFormat string) ([]SplitChunk, error) {
	if math.IsNaN(chunkSeconds) || math.IsInf(chunkSeconds, 0) || chunkSeconds < 0 {
		return nil, fmt.Errorf("%w: chunk_seconds must be a finite value >= 0", ErrInvalidArgument)
	}
	if outFormat == "" {
		outFormat = format
		if outFormat == "" {
			outFormat = "wav"
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var cuts []float64
	var inputDuration float64
	if chunkSeconds > 0 {
		meta, err := Probe(ctx, audio)
		if err != nil {
			return nil, fmt.Errorf("probe: %w", err)
		}
		if isFinitePositive(meta.DurationSeconds) {
			inputDuration = meta.DurationSeconds
			for t := chunkSeconds; t < meta.DurationSeconds; t += chunkSeconds {
				if len(cuts) >= maxSplitChunks {
					return nil, fmt.Errorf("%w: chunk_seconds too small (more than %d chunks)", ErrInvalidArgument, maxSplitChunks)
				}
				cuts = append(cuts, t)
			}
		}
		// Unknown/zero duration: fall through with no cuts so the input
		// is returned as one chunk instead of looping on a bad duration.
	} else {
		if err := validateBoundaries(boundaries); err != nil {
			return nil, err
		}
		cuts = append(cuts, boundaries...)
		// Boundaries do not need a duration to cut, but the final chunk's
		// reported duration does. Probe best-effort: an unavailable ffprobe
		// must not turn a valid explicit-boundary split into a failure. With
		// no boundaries the whole input is one chunk, so skip the probe.
		if len(boundaries) > 0 {
			if meta, err := Probe(ctx, audio); err == nil && isFinitePositive(meta.DurationSeconds) {
				inputDuration = meta.DurationSeconds
				for i, b := range boundaries {
					if b >= inputDuration {
						return nil, fmt.Errorf("%w: boundaries_seconds[%d] (%g) is not before the input duration (%g s)",
							ErrInvalidArgument, i, b, inputDuration)
					}
				}
			}
		}
	}
	starts := append([]float64{0}, cuts...)
	ends := append(append([]float64{}, cuts...), 0) // 0 sentinel = EOF
	out := make([]SplitChunk, 0, len(starts))
	for i, s := range starts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		e := ends[i]
		var bytes []byte
		var err error
		if e > 0 {
			bytes, err = trimWithDuration(ctx, audio, format, s, e, outFormat, inputDuration)
		} else {
			bytes, err = trimWithDuration(ctx, audio, format, s, 0, outFormat, inputDuration)
		}
		if err != nil {
			return nil, err
		}
		dur := 0.0
		switch {
		case e > 0 && isFinitePositive(inputDuration):
			// A boundary past the real end produces a shorter chunk; report
			// what ffmpeg actually emitted rather than the requested window.
			if end := math.Min(e, inputDuration); end > s {
				dur = end - s
			}
		case e > 0:
			dur = e - s
		case isFinitePositive(inputDuration) && inputDuration > s:
			dur = inputDuration - s
		}
		out = append(out, SplitChunk{Audio: bytes, Start: s, Duration: dur, ContentType: ContentTypeFor(outFormat)})
	}
	return out, nil
}

// validateBoundaries rejects the boundary sets that would otherwise
// produce empty, backwards, or negative-start chunks.
func validateBoundaries(boundaries []float64) error {
	prev := 0.0
	for i, v := range boundaries {
		if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
			return fmt.Errorf("%w: boundaries_seconds[%d] must be a finite value > 0", ErrInvalidArgument, i)
		}
		if v <= prev {
			return fmt.Errorf("%w: boundaries_seconds must be strictly increasing (index %d)", ErrInvalidArgument, i)
		}
		prev = v
	}
	return nil
}

// shortestSourceDuration returns the shortest measurable source duration.
// known is false when any source cannot be probed or reports no duration, in
// which case callers must not reject the request: ffmpeg remains the final
// authority and the empty-output guard reports a failure.
func shortestSourceDuration(ctx context.Context, sources [][]byte) (float64, bool) {
	shortest := 0.0
	for _, src := range sources {
		meta, err := Probe(ctx, src)
		if err != nil || !isFinitePositive(meta.DurationSeconds) {
			return 0, false
		}
		if shortest == 0 || meta.DurationSeconds < shortest {
			shortest = meta.DurationSeconds
		}
	}
	return shortest, shortest > 0
}

// validTime rejects negative and non-finite durations before they reach
// ffmpeg's time parser (NaN/Inf would otherwise be formatted into argv).
func validTime(v float64, field string) error {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return fmt.Errorf("%w: %s must be a finite value >= 0", ErrInvalidArgument, field)
	}
	return nil
}

type SplitChunk struct {
	Audio       []byte
	ContentType string
	Start       float64
	Duration    float64
}

// Merge concatenates the input sources. Crossfade is honoured when > 0.
// formats carries the caller-declared input format per source; when set
// the matching demuxer is forced for that input so temp files (which all
// share a .bin suffix) decode the way the client declared them.
func Merge(ctx context.Context, sources [][]byte, formats []string, outFormat string, crossfade float64) ([]byte, error) {
	if len(sources) == 0 {
		return nil, fmt.Errorf("%w: merge requires at least one source", ErrInvalidArgument)
	}
	for i, src := range sources {
		if len(src) == 0 {
			return nil, fmt.Errorf("%w: merge source %d is empty", ErrInvalidArgument, i)
		}
	}
	if math.IsNaN(crossfade) || math.IsInf(crossfade, 0) || crossfade < 0 {
		return nil, fmt.Errorf("%w: crossfade_seconds must be a finite value >= 0", ErrInvalidArgument)
	}
	if outFormat == "" {
		outFormat = "wav"
	}
	if crossfade > 0 && len(sources) > 1 {
		if shortest, known := shortestSourceDuration(ctx, sources); known && crossfade >= shortest {
			return nil, fmt.Errorf("%w: crossfade_seconds (%g) must be shorter than the shortest source (%g s); ffmpeg otherwise writes an empty stream",
				ErrInvalidArgument, crossfade, shortest)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// The concat filter and the crossfade chain both need seekable
	// per-source input, so each source is materialized to a temp file.
	// Cleanup is deferred before any argv work so every failure path
	// below removes the files.
	tmp, err := writeTempInputs(sources)
	if err != nil {
		return nil, err
	}
	defer tmp.cleanup()

	args := []string{}
	for i, p := range tmp.paths {
		if i < len(formats) && formats[i] != "" {
			args = append(args, "-f", demuxerFor(formats[i]))
		}
		args = append(args, "-i", p)
	}

	var filter, mapLabel string
	if crossfade > 0 && len(sources) > 1 {
		var graph strings.Builder
		prev := "[0:a]"
		for i := 1; i < len(sources); i++ {
			out := fmt.Sprintf("[a%d]", i)
			if graph.Len() > 0 {
				graph.WriteString(";")
			}
			fmt.Fprintf(&graph, "%s[%d:a]acrossfade=d=%g%s", prev, i, crossfade, out)
			prev = out
		}
		// The final acrossfade output is the one to map; mapping a
		// hard-coded "[a]" left every crossfade merge with an
		// unmatched-stream error.
		filter, mapLabel = graph.String(), prev
	} else {
		var inputs strings.Builder
		for i := range sources {
			fmt.Fprintf(&inputs, "[%d:a]", i)
		}
		filter = fmt.Sprintf("%sconcat=n=%d:v=0:a=1[a]", inputs.String(), len(sources))
		mapLabel = "[a]"
	}
	args = append(args, "-filter_complex", filter, "-map", mapLabel, "-f", muxerFor(outFormat), "pipe:1")
	return runFfmpeg(ctx, nil, args...)
}

func ContentTypeFor(format string) string {
	switch format {
	case "", "wav":
		return "audio/wav"
	case "mp3":
		return "audio/mpeg"
	case "flac":
		return "audio/flac"
	case "aac":
		return "audio/aac"
	case "ogg":
		return "audio/ogg"
	case "webm":
		return "audio/webm"
	case "opus":
		return "audio/opus"
	}
	return "application/octet-stream"
}
