package transformation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"music-tools/internal/jobs"
	musicstorage "music-tools/internal/storage"
)

const (
	MaxSourceBytes = 128 << 20
	MaxOutputBytes = 128 << 20
	MaxExcerptSecs = 120.0
	MinExcerptSecs = 0.25
)

var (
	ErrInvalidTransform = errors.New("invalid audio transformation request")
	ErrToolUnavailable  = errors.New("ffmpeg is unavailable")
)

// Request identifies one managed source excerpt and a deterministic operation.
// The source receipt may claim an inherited rights scope, but Music Library
// independently verifies that scope against its exact source attestation.
type Request struct {
	Kind                 string  `json:"kind"`
	SourceBlobRef        string  `json:"source_blob_ref"`
	SourceTrackSHA256    string  `json:"source_track_sha256"`
	SourceStartSeconds   float64 `json:"source_start_seconds"`
	SourceEndSeconds     float64 `json:"source_end_seconds"`
	RepeatCount          int     `json:"repeat_count,omitempty"`
	InheritedRightsScope string  `json:"inherited_rights_scope"`
}

// Submit accepts only an already-managed source plus a content hash and a
// bounded excerpt. It creates a CPU job and never selects a generative model.
func Submit(ctx context.Context, manager *jobs.Manager, input Request) (jobs.Job, error) {
	if manager == nil || validateTransform(input) != nil {
		return jobs.Job{}, ErrInvalidTransform
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return jobs.Job{}, err
	}
	estimatedSeconds := int(input.SourceEndSeconds - input.SourceStartSeconds)
	if input.Kind == "sample" {
		estimatedSeconds *= input.RepeatCount
	}
	if estimatedSeconds < 1 {
		estimatedSeconds = 1
	}
	return manager.Submit(ctx, jobs.Spec{Operation: "transform." + input.Kind, Lane: jobs.LaneCPU, Payload: payload, EstimatedSeconds: estimatedSeconds})
}

// Register installs the two deterministic transformation workers onto the
// existing Music Tools job manager. They share its bounded CPU lane, durable
// cancellation/recovery and job identity; there is no private worker pool.
func Register(manager *jobs.Manager, blobs *musicstorage.Store) error {
	if manager == nil || blobs == nil {
		return errors.New("transformation workers require the shared Music Tools job manager and blob store")
	}
	for _, kind := range []string{"remix", "sample"} {
		kind := kind
		if err := manager.RegisterOperation("transform."+kind, jobs.Operation{Runner: func(ctx context.Context, job jobs.Job, emit func(int, string)) (jobs.Result, error) {
			return run(ctx, job, kind, blobs, emit)
		}}); err != nil {
			return err
		}
	}
	return nil
}

func validateTransform(input Request) error {
	if input.Kind != "remix" && input.Kind != "sample" ||
		!sha256RE.MatchString(input.SourceTrackSHA256) || input.SourceBlobRef == "" ||
		path.Clean(input.SourceBlobRef) != input.SourceBlobRef ||
		(!strings.HasPrefix(input.SourceBlobRef, "out/") && !strings.HasPrefix(input.SourceBlobRef, "sources/")) ||
		!finiteNonnegative(input.SourceStartSeconds) || !finitePositiveAfter(input.SourceEndSeconds, input.SourceStartSeconds) ||
		input.SourceEndSeconds-input.SourceStartSeconds < MinExcerptSecs || input.SourceEndSeconds-input.SourceStartSeconds > MaxExcerptSecs ||
		(input.InheritedRightsScope != "personal_only" && input.InheritedRightsScope != "commercial" && input.InheritedRightsScope != "blocked" && input.InheritedRightsScope != "unknown") {
		return ErrInvalidTransform
	}
	if input.Kind == "sample" {
		if input.RepeatCount < 2 || input.RepeatCount > 8 || (input.SourceEndSeconds-input.SourceStartSeconds)*float64(input.RepeatCount) > MaxExcerptSecs*4 {
			return ErrInvalidTransform
		}
	} else if input.RepeatCount != 0 {
		return ErrInvalidTransform
	}
	return nil
}

func run(ctx context.Context, job jobs.Job, kind string, blobs *musicstorage.Store, emit func(int, string)) (jobs.Result, error) {
	var input Request
	decoder := json.NewDecoder(strings.NewReader(string(job.Payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || validateTransform(input) != nil || input.Kind != kind {
		return jobs.Result{}, ErrInvalidTransform
	}
	ffmpeg, err := ffmpegPath()
	if err != nil {
		return jobs.Result{}, err
	}
	version, err := ffmpegVersion(ctx, ffmpeg)
	if err != nil {
		return jobs.Result{}, err
	}
	jobCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	tempDir, err := os.MkdirTemp("", "music-tools-transform-")
	if err != nil {
		return jobs.Result{}, err
	}
	defer os.RemoveAll(tempDir)
	sourcePath := tempDir + string(os.PathSeparator) + "source.audio"
	outputPath := tempDir + string(os.PathSeparator) + "derived.wav"
	sourceHash, err := copyVerifiedSource(jobCtx, blobs, input.SourceBlobRef, input.SourceTrackSHA256, sourcePath)
	if err != nil {
		return jobs.Result{}, err
	}
	if emit != nil {
		emit(10, "verified exact source bytes")
	}
	probe, err := ffprobePath()
	if err != nil {
		return jobs.Result{}, err
	}
	sourceDuration, err := audioDuration(jobCtx, probe, sourcePath)
	if err != nil {
		return jobs.Result{}, err
	}
	if input.SourceEndSeconds > sourceDuration {
		return jobs.Result{}, fmt.Errorf("requested source range ends at %.6f seconds but the audio stream ends at %.6f seconds", input.SourceEndSeconds, sourceDuration)
	}
	args := transformationArgs(input, sourcePath, outputPath)
	commandCtx, commandCancel := context.WithTimeout(jobCtx, 10*time.Minute)
	defer commandCancel()
	command := exec.CommandContext(commandCtx, ffmpeg, args...)
	command.Stdin = nil
	if output, commandErr := command.CombinedOutput(); commandErr != nil {
		return jobs.Result{}, fmt.Errorf("ffmpeg %s failed: %w: %s", kind, commandErr, safeToolOutput(output))
	}
	if err := jobCtx.Err(); err != nil {
		return jobs.Result{}, err
	}
	outputDigest, outputSize, err := digestFile(outputPath)
	if err != nil {
		return jobs.Result{}, err
	}
	if outputSize == 0 || outputSize > MaxOutputBytes {
		return jobs.Result{}, fmt.Errorf("transformed output size is outside 1..%d bytes", MaxOutputBytes)
	}
	resultRef := "derived/" + job.ID + "/output.wav"
	derivedEnd := input.SourceEndSeconds - input.SourceStartSeconds
	if kind == "sample" {
		derivedEnd *= float64(input.RepeatCount)
	}
	receipt := DerivationReceipt{
		SchemaVersion: "1", JobID: job.ID, Kind: kind,
		SourceTrackSHA256: sourceHash, DerivedTrackSHA256: outputDigest,
		SourceStartSeconds: input.SourceStartSeconds, SourceEndSeconds: input.SourceEndSeconds,
		DerivedStartSeconds: 0, DerivedEndSeconds: derivedEnd,
		ResultRef: resultRef, OperationVersion: "1", ToolName: "ffmpeg", ToolVersion: version,
		InheritedRightsScope: input.InheritedRightsScope,
	}
	encoded, err := EncodeJobReceipt(job.ID, job.Operation, resultRef, receipt)
	if err != nil {
		return jobs.Result{}, err
	}
	outputFile, err := os.Open(outputPath)
	if err != nil {
		return jobs.Result{}, err
	}
	putErr := blobs.PutOwned(jobCtx, resultRef, "music-tools-transform:"+job.ID, outputFile, "audio/wav", false)
	closeErr := outputFile.Close()
	if putErr != nil || closeErr != nil {
		return jobs.Result{}, errors.Join(putErr, closeErr)
	}
	if emit != nil {
		emit(100, "stored hash-verified transformation and lineage receipt")
	}
	return jobs.Result{Ref: resultRef, Meta: map[string]string{ReceiptMetaKey: encoded, "source_sha256": sourceHash, "derived_sha256": outputDigest}}, nil
}

func copyVerifiedSource(ctx context.Context, blobs *musicstorage.Store, key, expectedHash, target string) (string, error) {
	reader, _, err := blobs.Get(ctx, key)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	limited := io.LimitReader(reader, MaxSourceBytes+1)
	size, copyErr := io.Copy(io.MultiWriter(out, h), limited)
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil {
		return "", errors.Join(copyErr, closeErr)
	}
	if size <= 0 || size > MaxSourceBytes {
		return "", fmt.Errorf("source size is outside 1..%d bytes", MaxSourceBytes)
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expectedHash {
		return "", errors.New("source bytes no longer match requested track SHA-256")
	}
	return actual, nil
}

func transformationArgs(input Request, sourcePath, outputPath string) []string {
	start := ffNumber(input.SourceStartSeconds)
	end := ffNumber(input.SourceEndSeconds)
	duration := ffNumber(input.SourceEndSeconds - input.SourceStartSeconds)
	filter := "[0:a]atrim=start=" + start + ":end=" + end + ",asetpts=PTS-STARTPTS"
	if input.Kind == "sample" {
		durationValue := input.SourceEndSeconds - input.SourceStartSeconds
		rawLabels := make([]string, input.RepeatCount)
		outputLabels := make([]string, input.RepeatCount)
		for i := range rawLabels {
			rawLabels[i] = fmt.Sprintf("[sample%d_raw]", i)
			outputLabels[i] = fmt.Sprintf("[sample%d]", i)
		}
		parts := []string{filter + ",asplit=" + strconv.Itoa(input.RepeatCount) + strings.Join(rawLabels, "")}
		fade := 0.01
		fadeOutStart := durationValue - fade
		for i := range rawLabels {
			parts = append(parts, rawLabels[i]+"afade=t=in:st=0:d="+ffNumber(fade)+",afade=t=out:st="+ffNumber(fadeOutStart)+":d="+ffNumber(fade)+outputLabels[i])
		}
		parts = append(parts, strings.Join(outputLabels, "")+"concat=n="+strconv.Itoa(input.RepeatCount)+":v=0:a=1,atrim=duration="+ffNumber(durationValue*float64(input.RepeatCount))+"[outa]")
		filter = strings.Join(parts, ";")
	} else {
		filter += ",highpass=f=80,equalizer=f=3200:t=q:w=1:g=1.5,aecho=0.8:0.85:75|120:0.18|0.12,alimiter=limit=0.95,atrim=duration=" + duration + "[outa]"
	}
	return []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-y", "-i", sourcePath, "-filter_complex", filter, "-map", "[outa]", "-map_metadata", "-1", "-ac", "2", "-ar", "48000", "-c:a", "pcm_s16le", "-f", "wav", outputPath}
}

func ffNumber(value float64) string { return strconv.FormatFloat(value, 'f', 6, 64) }

func ffmpegPath() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("FFMPEG_BIN")); configured != "" {
		path, err := exec.LookPath(configured)
		if err != nil {
			return "", fmt.Errorf("%w: configured FFMPEG_BIN was not found", ErrToolUnavailable)
		}
		return path, nil
	}
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", ErrToolUnavailable
	}
	return path, nil
}

func ffprobePath() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("FFPROBE_BIN")); configured != "" {
		path, err := exec.LookPath(configured)
		if err != nil {
			return "", fmt.Errorf("configured FFPROBE_BIN was not found")
		}
		return path, nil
	}
	path, err := exec.LookPath("ffprobe")
	if err != nil {
		return "", fmt.Errorf("ffprobe is unavailable; exact excerpt verification requires it")
	}
	return path, nil
}

func audioDuration(ctx context.Context, probe, filename string) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, probe, "-v", "error", "-select_streams", "a:0", "-show_entries", "stream=duration", "-of", "csv=p=0", filename).Output()
	if err != nil {
		return 0, fmt.Errorf("probe source audio duration: %w", err)
	}
	duration, err := parseAudioDuration(string(output))
	if err != nil {
		return 0, err
	}
	return duration, nil
}

func parseAudioDuration(output string) (float64, error) {
	duration, err := strconv.ParseFloat(strings.TrimSpace(output), 64)
	if err != nil || !finitePositiveAfter(duration, 0) {
		return 0, errors.New("source audio duration is missing or invalid")
	}
	return duration, nil
}

func ffmpegVersion(ctx context.Context, binary string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, binary, "-version").Output()
	if err != nil {
		return "", fmt.Errorf("read ffmpeg version: %w", err)
	}
	line := strings.TrimSpace(strings.SplitN(string(output), "\n", 2)[0])
	if len(line) == 0 || len(line) > 256 {
		return "", errors.New("ffmpeg returned invalid version metadata")
	}
	return line, nil
}

func digestFile(filename string) (string, int64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	h := sha256.New()
	size, err := io.Copy(h, io.LimitReader(file, MaxOutputBytes+1))
	if err != nil {
		return "", size, err
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}

func safeToolOutput(output []byte) string {
	message := strings.TrimSpace(string(output))
	if len(message) > 512 {
		message = message[:512]
	}
	return message
}
