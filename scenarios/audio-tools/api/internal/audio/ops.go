package audio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
func Trim(ctx context.Context, audio []byte, format string, start, end float64, outFormat string) ([]byte, error) {
	if start < 0 {
		return nil, errors.New("audio: trim start_seconds must be >= 0")
	}
	if outFormat == "" {
		outFormat = format
		if outFormat == "" {
			outFormat = "wav"
		}
	}
	args := []string{"-ss", strconv.FormatFloat(start, 'f', -1, 64), "-i", "pipe:0"}
	if end > start {
		args = append(args, "-to", strconv.FormatFloat(end, 'f', -1, 64))
	}
	args = append(args, "-f", outFormat, "pipe:1")
	return runFfmpeg(ctx, audio, args...)
}

// Fade applies fade-in and fade-out windows.
func Fade(ctx context.Context, audio []byte, format string, fadeIn, fadeOut float64, outFormat string) ([]byte, error) {
	if outFormat == "" {
		outFormat = format
		if outFormat == "" {
			outFormat = "wav"
		}
	}
	filter := ""
	if fadeIn > 0 {
		filter = fmt.Sprintf("afade=t=in:st=0:d=%g", fadeIn)
	}
	if fadeOut > 0 {
		if filter != "" {
			filter += ","
		}
		// We don't know total duration without ffprobe; use afade with
		// curve duration only — clients responsible for trimming first if
		// they need a precise end time.
		filter += fmt.Sprintf("afade=t=out:d=%g", fadeOut)
	}
	args := []string{"-i", "pipe:0"}
	if filter != "" {
		args = append(args, "-af", filter)
	}
	args = append(args, "-f", outFormat, "pipe:1")
	return runFfmpeg(ctx, audio, args...)
}

// Volume scales the input by gainDB.
func Volume(ctx context.Context, audio []byte, format string, gainDB float64, outFormat string) ([]byte, error) {
	if outFormat == "" {
		outFormat = format
		if outFormat == "" {
			outFormat = "wav"
		}
	}
	return runFfmpeg(ctx, audio,
		"-i", "pipe:0",
		"-af", fmt.Sprintf("volume=%gdB", gainDB),
		"-f", outFormat, "pipe:1",
	)
}

// Normalize applies loudnorm (EBU R128 by default) or peak/rms normalisation.
func Normalize(ctx context.Context, audio []byte, format, method string, targetLUFS float64, outFormat string) ([]byte, error) {
	if outFormat == "" {
		outFormat = format
		if outFormat == "" {
			outFormat = "wav"
		}
	}
	var filter string
	switch method {
	case "peak":
		filter = "dynaudnorm"
	case "rms":
		filter = "dynaudnorm=g=15"
	default: // ebu r128
		lufs := targetLUFS
		if lufs == 0 {
			lufs = -16.0
		}
		filter = fmt.Sprintf("loudnorm=I=%g", lufs)
	}
	return runFfmpeg(ctx, audio,
		"-i", "pipe:0",
		"-af", filter,
		"-f", outFormat, "pipe:1",
	)
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
			Duration string            `json:"duration"`
			FormatName string          `json:"format_name"`
			BitRate    string          `json:"bit_rate"`
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
	if doc.Format.Duration != "" {
		m.DurationSeconds, _ = strconv.ParseFloat(doc.Format.Duration, 64)
	}
	if doc.Format.BitRate != "" {
		v, _ := strconv.Atoi(doc.Format.BitRate)
		m.Bitrate = int32(v)
	}
	if len(doc.Streams) > 0 {
		s := doc.Streams[0]
		if s.SampleRate != "" {
			v, _ := strconv.Atoi(s.SampleRate)
			m.SampleRate = int32(v)
		}
		m.Channels = s.Channels
		m.Codec = s.CodecName
		if m.Bitrate == 0 && s.BitRate != "" {
			v, _ := strconv.Atoi(s.BitRate)
			m.Bitrate = int32(v)
		}
	}
	return m, nil
}

// Split slices the input into chunks. If chunkSeconds > 0 the audio is
// cut into equal slices; otherwise boundariesSeconds defines explicit
// cut points (always combined with start=0 and end=EOF).
func Split(ctx context.Context, audio []byte, format string, chunkSeconds float64, boundaries []float64, outFormat string) ([]SplitChunk, error) {
	if outFormat == "" {
		outFormat = format
		if outFormat == "" {
			outFormat = "wav"
		}
	}

	var cuts []float64
	if chunkSeconds > 0 {
		meta, err := Probe(ctx, audio)
		if err != nil {
			return nil, fmt.Errorf("probe: %w", err)
		}
		for t := chunkSeconds; t < meta.DurationSeconds; t += chunkSeconds {
			cuts = append(cuts, t)
		}
	} else {
		cuts = append(cuts, boundaries...)
	}
	starts := append([]float64{0}, cuts...)
	ends := append(append([]float64{}, cuts...), 0) // 0 sentinel = EOF
	out := make([]SplitChunk, 0, len(starts))
	for i, s := range starts {
		e := ends[i]
		var bytes []byte
		var err error
		if e > 0 {
			bytes, err = Trim(ctx, audio, format, s, e, outFormat)
		} else {
			bytes, err = Trim(ctx, audio, format, s, 0, outFormat)
		}
		if err != nil {
			return nil, err
		}
		dur := 0.0
		if e > 0 {
			dur = e - s
		}
		out = append(out, SplitChunk{Audio: bytes, Start: s, Duration: dur, ContentType: contentTypeFor(outFormat)})
	}
	return out, nil
}

type SplitChunk struct {
	Audio       []byte
	ContentType string
	Start       float64
	Duration    float64
}

// Merge concatenates the input sources. Crossfade is honoured when > 0.
func Merge(ctx context.Context, sources [][]byte, _ []string, outFormat string, crossfade float64) ([]byte, error) {
	if len(sources) == 0 {
		return nil, errors.New("audio: merge requires at least one source")
	}
	if outFormat == "" {
		outFormat = "wav"
	}
	// For greenfield simplicity we use the concat demuxer via pipe is
	// awkward; instead we read each input as a separate filter graph
	// input and concat. Sources are written to fifo-like /dev/fd args.
	// Implementation uses temp files because the concat filter needs
	// seekable input.
	tmp, err := writeTempInputs(sources)
	if err != nil {
		return nil, err
	}
	defer tmp.cleanup()

	args := []string{}
	for _, p := range tmp.paths {
		args = append(args, "-i", p)
	}
	var filter string
	if crossfade > 0 && len(sources) > 1 {
		filter = ""
		prev := "[0:a]"
		for i := 1; i < len(sources); i++ {
			out := fmt.Sprintf("[a%d]", i)
			filter += fmt.Sprintf("%s[%d:a]acrossfade=d=%g%s;", prev, i, crossfade, out)
			prev = out
		}
		// Trim trailing semicolon
		filter = filter[:len(filter)-1]
	} else {
		concat := ""
		for i := range sources {
			concat += fmt.Sprintf("[%d:a]", i)
		}
		filter = fmt.Sprintf("%sconcat=n=%d:v=0:a=1[a]", concat, len(sources))
	}
	args = append(args, "-filter_complex", filter, "-map", "[a]", "-f", outFormat, "pipe:1")
	return runFfmpeg(ctx, nil, args...)
}

func contentTypeFor(format string) string {
	switch format {
	case "wav":
		return "audio/wav"
	case "mp3":
		return "audio/mpeg"
	case "flac":
		return "audio/flac"
	case "aac":
		return "audio/aac"
	case "ogg":
		return "audio/ogg"
	}
	return "application/octet-stream"
}
