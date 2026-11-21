package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// loudnormReport is the JSON report loudnorm writes to ffmpeg's log.
// Every numeric field is rendered as a string by the filter, and silence
// produces "-inf", so values are parsed loosely.
type loudnormReport struct {
	InputI       string `json:"input_i"`
	InputTP      string `json:"input_tp"`
	InputLRA     string `json:"input_lra"`
	InputThresh  string `json:"input_thresh"`
	TargetOffset string `json:"target_offset"`
	OutputI      string `json:"output_i"`
}

// NormalizeMeasured normalises audio and reports the integrated loudness
// of the audio it produced, in LUFS. A measured value of 0 means unknown:
// the transform succeeded but the measurement could not be taken.
//
// EBU R128 uses two loudnorm passes. Pass 1 analyses the input; pass 2
// encodes with the measured figures and linear=true, which is the accurate
// approach (single-pass loudnorm is dynamic and its gain depends on the
// material trajectory rather than a fixed measured correction). The extra
// decode is deliberate: normalization is not a hot path, and the alternative
// cannot report what the produced audio actually measures.
func NormalizeMeasured(ctx context.Context, audio []byte, format, method string, targetLUFS float64, outFormat string) ([]byte, float64, error) {
	if outFormat == "" {
		outFormat = format
		if outFormat == "" {
			outFormat = "wav"
		}
	}
	switch method {
	case "peak", "rms":
		// dynaudnorm is not an EBU R128 filter, so the produced output is
		// measured separately (best-effort) instead of trusting a target.
		filter := "dynaudnorm"
		if method == "rms" {
			filter = "dynaudnorm=g=15"
		}
		out, err := runFfmpeg(ctx, audio,
			"-i", "pipe:0",
			"-af", filter,
			"-f", muxerFor(outFormat), "pipe:1",
		)
		if err != nil {
			return nil, 0, err
		}
		return out, measureIntegratedLoudness(ctx, out), nil
	default: // ebu r128 (including the unspecified/default method)
		lufs := targetLUFS
		if lufs == 0 {
			lufs = -16.0
		}
		if math.IsNaN(lufs) || math.IsInf(lufs, 0) {
			return nil, 0, fmt.Errorf("%w: target_lufs must be a finite value", ErrInvalidArgument)
		}
		// loudnorm's integrated-loudness target is bounded; values outside
		// this window either fail inside ffmpeg or produce unusable gain.
		if lufs < -70 || lufs > -5 {
			return nil, 0, fmt.Errorf("%w: target_lufs must be between -70 and -5", ErrInvalidArgument)
		}
		return normalizeTwoPass(ctx, audio, lufs, outFormat)
	}
}

// normalizeTwoPass performs EBU R128 normalisation. Measurement failure
// degrades to the single-pass dynamic filter: producing correctly decodable
// audio is the request's primary contract, and accuracy is secondary to it.
func normalizeTwoPass(ctx context.Context, audio []byte, target float64, outFormat string) ([]byte, float64, error) {
	report, err := measureInputLoudness(ctx, audio)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, 0, fmt.Errorf("%w: %w", ctxErr, err)
		}
		out, runErr := runFfmpeg(ctx, audio,
			"-i", "pipe:0",
			"-af", fmt.Sprintf("loudnorm=I=%g", target),
			"-f", muxerFor(outFormat), "pipe:1",
		)
		if runErr != nil {
			return nil, 0, runErr
		}
		return out, 0, nil
	}

	filter := fmt.Sprintf(
		"loudnorm=I=%g:measured_I=%s:measured_TP=%s:measured_LRA=%s:measured_thresh=%s:offset=%s:linear=true:print_format=json",
		target, report.InputI, report.InputTP, report.InputLRA, report.InputThresh, report.TargetOffset,
	)
	out, stderr, err := runFfmpegCapture(ctx, audio,
		"-i", "pipe:0",
		"-af", filter,
		"-f", muxerFor(outFormat), "pipe:1",
	)
	if err != nil {
		return nil, 0, err
	}
	// The produced audio is already in hand; an unreadable report only
	// costs us the measurement, not the response.
	if second, parseErr := parseLoudnormReport(stderr); parseErr == nil {
		if v, ok := finiteLoudness(second.OutputI); ok {
			return out, v, nil
		}
	}
	return out, 0, nil
}

// measureInputLoudness runs loudnorm's analysis pass over audio and returns
// the measured input figures. Any failure is returned so callers can fall
// back to a path that does not need the measurement.
func measureInputLoudness(ctx context.Context, audio []byte) (loudnormReport, error) {
	_, stderr, err := runFfmpegCapture(ctx, audio,
		"-i", "pipe:0",
		"-af", "loudnorm=print_format=json",
		"-f", "null", "-",
	)
	if err != nil {
		return loudnormReport{}, err
	}
	report, err := parseLoudnormReport(stderr)
	if err != nil {
		return loudnormReport{}, err
	}
	if err := validateLoudnormReport(report); err != nil {
		return loudnormReport{}, err
	}
	return report, nil
}

// validateLoudnormReport rejects reports whose analysis figures are not
// usable as second-pass inputs. Silence yields "-inf", which would make the
// first pass look successful while the linear correction is impossible.
func validateLoudnormReport(report loudnormReport) error {
	for name, value := range map[string]string{
		"input_i":       report.InputI,
		"input_tp":      report.InputTP,
		"input_lra":     report.InputLRA,
		"input_thresh":  report.InputThresh,
		"target_offset": report.TargetOffset,
	} {
		if _, ok := finiteLoudness(value); !ok {
			return fmt.Errorf("audio: loudnorm report has non-finite %s (%q)", name, value)
		}
	}
	return nil
}

// measureIntegratedLoudness reports the integrated loudness of encoded
// audio, or 0 when it cannot be measured. Never fails a caller: the audio
// it measures has already been produced successfully.
func measureIntegratedLoudness(ctx context.Context, audio []byte) float64 {
	report, err := measureInputLoudness(ctx, audio)
	if err != nil {
		return 0
	}
	v, ok := finiteLoudness(report.InputI)
	if !ok {
		return 0
	}
	return v
}

// parseLoudnormReport extracts loudnorm's JSON report from stderr. The
// report is the last balanced object ffmpeg writes: log lines, the progress
// stats, and the report itself all share stderr.
func parseLoudnormReport(stderr []byte) (loudnormReport, error) {
	block, ok := lastJSONObject(stderr)
	if !ok {
		return loudnormReport{}, fmt.Errorf("audio: no loudnorm JSON report in ffmpeg output")
	}
	var report loudnormReport
	if err := json.Unmarshal(block, &report); err != nil {
		return loudnormReport{}, fmt.Errorf("audio: parse loudnorm report: %w", err)
	}
	return report, nil
}

// lastJSONObject returns the last complete top-level {...} block in b,
// tracking string literals so braces inside quoted values do not distort
// depth. ffmpeg's surrounding log lines make both a whole-buffer decode and
// a first-object decode unreliable.
func lastJSONObject(b []byte) ([]byte, bool) {
	start, depth := -1, 0
	inString, escaped := false, false
	var last []byte
	for i, c := range b {
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			if depth > 0 {
				inString = true
			}
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			if depth > 0 {
				depth--
				if depth == 0 && start >= 0 {
					last = b[start : i+1]
				}
			}
		}
	}
	return last, last != nil
}

// finiteLoudness parses a loudnorm numeric string. "-inf" (silence) and
// other non-finite sentinels report no usable measurement.
func finiteLoudness(s string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}
