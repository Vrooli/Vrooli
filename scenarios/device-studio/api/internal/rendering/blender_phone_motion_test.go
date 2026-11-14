package rendering

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"device-studio/internal/composition"
	"github.com/vrooli/platform-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const phoneMotionWorkerMarker = "owned-phone-motion-encoder-v1"
const phoneMotionFFmpegArtifact = "2ad074849ce8635ce03d623865a26fdb1636a9a44346d21d4af08c1b191a9789"

type phoneMotionFrameBinding struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type phoneMotionEncodeReceipt struct {
	Schema              int                       `json:"schema_version"`
	SourceManifestSHA   string                    `json:"source_manifest_sha256"`
	StagedManifestSHA   string                    `json:"staged_manifest_sha256"`
	ScriptSHA           string                    `json:"script_sha256"`
	WorkerSHA           string                    `json:"worker_sha256"`
	WorkerBytes         int64                     `json:"worker_bytes"`
	ProfileID           string                    `json:"profile_id"`
	ProfileArtifact     string                    `json:"profile_artifact_identity"`
	FFmpegSHA           string                    `json:"ffmpeg_sha256"`
	FFprobeSHA          string                    `json:"ffprobe_sha256"`
	FFprobeJSONSHA      string                    `json:"ffprobe_json_sha256"`
	FFprobeJSONBytes    int64                     `json:"ffprobe_json_bytes"`
	StreamDurationTicks int64                     `json:"stream_duration_ticks"`
	StreamTimeBase      composition.Rational      `json:"stream_time_base"`
	Frames              []phoneMotionFrameBinding `json:"frames"`
	OutputName          string                    `json:"output_name"`
	OutputBytes         int64                     `json:"output_bytes"`
	OutputSHA           string                    `json:"output_sha256"`
	Native              VideoNativeFacts          `json:"native"`
	DurationNumerator   int                       `json:"duration_numerator"`
	DurationDenominator int                       `json:"duration_denominator"`
}

type phoneMotionSourceBinding struct {
	SourceManifestSHA string `json:"source_manifest_sha256"`
	StagedManifestSHA string `json:"staged_manifest_sha256"`
	ScriptSHA         string `json:"script_sha256"`
	Backend           string `json:"backend"`
	Profile           string `json:"profile"`
}

type phoneMotionProbeDocument struct {
	Streams []struct {
		Index      int             `json:"index"`
		Codec      string          `json:"codec_name"`
		DurationTS json.RawMessage `json:"duration_ts"`
		TimeBase   string          `json:"time_base"`
	} `json:"streams"`
}

// This exact test-binary entrypoint runs only inside a platform-owned operation.
func TestPhoneMotionEncoderWorker(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_E3_PHONE_MOTION_ENCODER") != phoneMotionWorkerMarker {
		t.Skip("private owned phone-motion encoder worker")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Fatal("pinned video profile supports Linux/amd64 only")
	}
	attempt, err := os.Getwd()
	if err != nil || !filepath.IsAbs(attempt) || filepath.Clean(attempt) != attempt || !strings.Contains(filepath.ToSlash(attempt), "/goal/evidence/E3/scene-studies/phone/runs/") {
		t.Fatalf("worker cwd is outside fixed phone-study run tree: %q", attempt)
	}
	expectedWorkerSHA := os.Getenv("DEVICE_STUDIO_E3_PHONE_MOTION_WORKER_SHA256")
	workerBytes, err := readMotionRegular(attempt, "phone-motion-encoder-worker", 256<<20)
	if err != nil || blenderDigest(workerBytes) != expectedWorkerSHA {
		t.Fatal("worker executable does not match its pre-dispatch pinned digest")
	}
	executableSHA := currentExecutableSHA(t)
	if executableSHA != expectedWorkerSHA {
		t.Fatal("executed test worker differs from its pinned bytes")
	}
	bindings, err := readPhoneMotionBindings(attempt)
	if err != nil || bindings.Profile != "motion-yaw-24" || bindings.Backend != "OPTIX" || bindings.SourceManifestSHA != phoneStudyManifestSHA || bindings.ScriptSHA != phoneStudyScriptSHA {
		t.Fatalf("motion source bindings differ: %+v (%v)", bindings, err)
	}
	if err = verifyPhoneStudy(attempt, bindings.Profile, bindings.Backend, bindings.StagedManifestSHA, bindings.ScriptSHA, bindings.SourceManifestSHA); err != nil {
		t.Fatalf("verify rendered motion frames before encode: %v", err)
	}
	profileBytes, err := readMotionRegular(attempt, "resolved-video-profile.json", 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	var profile cliv1.RendererToolProfile
	if err = (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(profileBytes, &profile); err != nil {
		t.Fatal("invalid pinned FFmpeg profile JSON")
	}
	pair, err := LocalToolPairFromProfile(&profile)
	if err != nil || profileArtifactIdentity(&profile) != profile.GetArtifactIdentity() || profile.GetArtifactIdentity() != phoneMotionFFmpegArtifact {
		t.Fatalf("pinned FFmpeg pair is invalid: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	ffmpeg, err := openVerifiedTool(ctx, pair.FFmpegPath, pair.FFmpegSize, pair.FFmpegSHA256)
	if err != nil {
		t.Fatal(err)
	}
	defer ffmpeg.Close()
	ffprobe, err := openVerifiedTool(ctx, pair.FFprobePath, pair.FFprobeSize, pair.FFprobeSHA256)
	if err != nil {
		t.Fatal(err)
	}
	defer ffprobe.Close()
	root, err := os.OpenRoot(attempt)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	manifest := VideoManifest{Width: 1920, Height: 1080, OutputRate: composition.Rational{Num: 24, Den: 1}}
	encoder, err := startVideoEncoder(ctx, root, ffmpeg, manifest, 48)
	if err != nil {
		t.Fatal(err)
	}
	finished := false
	defer func() {
		if !finished {
			encoder.abort()
		}
	}()
	_, _, _, names, _ := phoneStudyProfile("motion-yaw-24")
	frames := make([]phoneMotionFrameBinding, 0, len(names))
	for _, name := range names {
		data, readErr := readMotionRegular(filepath.Join(attempt, "outputs"), name, 7<<20)
		if readErr != nil {
			t.Fatal(readErr)
		}
		frames = append(frames, phoneMotionFrameBinding{Name: name, SHA256: blenderDigest(data), Bytes: int64(len(data))})
		img, decodeErr := png.Decode(bytes.NewReader(data))
		if decodeErr != nil || img.Bounds().Dx() != 1920 || img.Bounds().Dy() != 1080 {
			t.Fatalf("decode bounded motion frame %s: %v", name, decodeErr)
		}
		rgb, encodeErr := phoneImageRGB24(img)
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		if err = encoder.writeFrame(ctx, rgb); err != nil {
			t.Fatalf("feed frame %s to pinned encoder: %v", name, err)
		}
	}
	outputBytes, outputSHA, err := encoder.finish(ctx)
	finished = true
	if err != nil {
		t.Fatal(err)
	}
	facts, err := verifyEncodedVideo(ctx, ffprobe, ffmpeg, root, manifest, 48)
	if err != nil {
		t.Fatal(err)
	}
	wantPTS, err := plannedOutputPTS(manifest, facts.TimeBase, 48)
	if err != nil || len(wantPTS) != 48 || facts.Width != 1920 || facts.Height != 1080 || facts.FrameRate.Num != 24 || facts.FrameRate.Den != 1 || facts.FrameCount != 48 || facts.FirstPTS != 0 || facts.LastPTS != wantPTS[47] || !facts.FullDecodeVerified {
		t.Fatalf("encoded motion schedule differs: facts=%+v err=%v", facts, err)
	}
	rawProbe, durationTicks, timeBase, err := inspectPhoneMotionEncodedDuration(ctx, ffprobe, root)
	if err != nil {
		t.Fatal(err)
	}
	if err = writeDurable(attempt, "phone-motion-output-ffprobe.json", rawProbe); err != nil {
		t.Fatal(err)
	}
	receipt := phoneMotionEncodeReceipt{Schema: 1, SourceManifestSHA: bindings.SourceManifestSHA, StagedManifestSHA: bindings.StagedManifestSHA, ScriptSHA: bindings.ScriptSHA, WorkerSHA: currentExecutableSHA(t), WorkerBytes: int64(len(workerBytes)), ProfileID: pair.ProfileID, ProfileArtifact: pair.ArtifactIdentity, FFmpegSHA: pair.FFmpegSHA256, FFprobeSHA: pair.FFprobeSHA256, FFprobeJSONSHA: blenderDigest(rawProbe), FFprobeJSONBytes: int64(len(rawProbe)), StreamDurationTicks: durationTicks, StreamTimeBase: timeBase, Frames: frames, OutputName: VideoOutputName, OutputBytes: outputBytes, OutputSHA: outputSHA, Native: facts, DurationNumerator: 2, DurationDenominator: 1}
	receiptBytes, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = writeDurable(attempt, "phone-motion-video-receipt.json", append(receiptBytes, '\n')); err != nil {
		t.Fatal(err)
	}
}

func runOwnedPhoneMotionEncodeOperation(t *testing.T, attempt string) {
	t.Helper()
	bindings, err := readPhoneMotionBindings(attempt)
	if err != nil || bindings.Profile != "motion-yaw-24" || bindings.Backend != "OPTIX" || bindings.SourceManifestSHA != phoneStudyManifestSHA || bindings.ScriptSHA != phoneStudyScriptSHA {
		t.Fatalf("motion source bindings differ before encoder dispatch: %+v (%v)", bindings, err)
	}
	if err = verifyPhoneStudy(attempt, bindings.Profile, bindings.Backend, bindings.StagedManifestSHA, bindings.ScriptSHA, bindings.SourceManifestSHA); err != nil {
		t.Fatalf("verify motion PNGs before encoder dispatch: %v", err)
	}
	frameReceiptBytes, err := readMotionRegular(filepath.Join(attempt, "outputs"), "qualification-receipt.json", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	var frameReceipt phoneStudyReceipt
	if err = json.Unmarshal(frameReceiptBytes, &frameReceipt); err != nil {
		t.Fatal(err)
	}
	frameBindings := make([]phoneMotionFrameBinding, 0, len(frameReceipt.Outputs))
	for _, output := range frameReceipt.Outputs {
		frameBindings = append(frameBindings, phoneMotionFrameBinding{Name: output.Path, SHA256: output.SHA, Bytes: output.Bytes})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	profile, err := ResolveInspectionProfile(ctx)
	cancel()
	if err != nil {
		t.Fatal(err)
	}
	pair, err := LocalToolPairFromProfile(profile)
	if err != nil || profileArtifactIdentity(profile) != profile.GetArtifactIdentity() || profile.GetArtifactIdentity() != phoneMotionFFmpegArtifact {
		t.Fatalf("managed FFmpeg profile is invalid: %v", err)
	}
	profileBytes, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	if err = writeDurable(attempt, "resolved-video-profile.json", profileBytes); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Lstat(filepath.Join(attempt, VideoOutputName)); err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("motion MP4 destination already exists or cannot be inspected: %v", err)
	}
	workerSource, err := os.Executable()
	if err != nil || !filepath.IsAbs(workerSource) {
		t.Fatalf("resolve exact test worker executable: %v", err)
	}
	workerBytes, err := os.ReadFile(workerSource)
	if err != nil || len(workerBytes) == 0 || len(workerBytes) > 256<<20 {
		t.Fatalf("test worker binary is unavailable or too large: %v", err)
	}
	workerSHA := blenderDigest(workerBytes)
	workerSize := int64(len(workerBytes))
	workerName := "phone-motion-encoder-worker"
	workerPath := filepath.Join(attempt, workerName)
	workerFile, err := os.OpenFile(workerPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o500)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = workerFile.Write(workerBytes); err == nil {
		err = workerFile.Sync()
	}
	closeErr := workerFile.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("pin exact motion worker bytes: %v / %v", err, closeErr)
	}
	if err = syncDirectory(attempt); err != nil {
		t.Fatal(err)
	}
	stagedWorker, err := readMotionRegular(attempt, workerName, 256<<20)
	if err != nil || int64(len(stagedWorker)) != workerSize || blenderDigest(stagedWorker) != workerSHA {
		t.Fatal("staged motion worker changed before owned dispatch")
	}
	for _, name := range []string{"home-video", "tmp-video"} {
		if err = os.Mkdir(filepath.Join(attempt, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	wall := 120 * time.Second
	spec := platform.OperationSpec{
		Path: workerPath, Args: []string{"-test.run=^TestPhoneMotionEncoderWorker$", "-test.count=1"},
		Env: []string{"DEVICE_STUDIO_E3_PHONE_MOTION_ENCODER=" + phoneMotionWorkerMarker, "DEVICE_STUDIO_E3_PHONE_MOTION_WORKER_SHA256=" + workerSHA, "HOME=" + filepath.Join(attempt, "home-video"), "TMPDIR=" + filepath.Join(attempt, "tmp-video"), "LANG=C", "LC_ALL=C"},
		Dir: attempt, Limits: platform.OperationLimits{MemoryMaxBytes: 8 << 30, TasksMax: 128, CPUWeight: 100, WallTime: wall},
	}
	attemptID := fmt.Sprintf("e3-phone-motion-ffmpeg-%d", time.Now().UnixNano())
	intent, err := platform.NewOperationIntent("device-studio", attemptID, spec)
	if err != nil {
		t.Fatal(err)
	}
	intentRecord := map[string]any{"intent": intent, "spec": spec, "worker_sha256": workerSHA, "worker_bytes": workerSize, "profile_id": pair.ProfileID, "profile_artifact_identity": pair.ArtifactIdentity, "ffmpeg_sha256": pair.FFmpegSHA256, "ffprobe_sha256": pair.FFprobeSHA256, "source_manifest_sha256": bindings.SourceManifestSHA, "staged_manifest_sha256": bindings.StagedManifestSHA, "script_sha256": bindings.ScriptSHA, "frames": frameBindings, "frame_rate": "24/1", "width": 1920, "height": 1080}
	intentBytes, err := json.MarshalIndent(intentRecord, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = writeDurable(attempt, "phone-motion-operation-intent.json", append(intentBytes, '\n')); err != nil {
		t.Fatal(err)
	}
	opCtx, opCancel := context.WithTimeout(context.Background(), wall+10*time.Second)
	defer opCancel()
	op, err := platform.StartOwnedOperation(opCtx, intent, spec)
	if err != nil {
		ref, lookupErr := platform.LookupOwnedOperation(opCtx, intent)
		if lookupErr != nil {
			record, _ := json.MarshalIndent(map[string]any{"intent": intent, "worker_sha256": workerSHA, "worker_bytes": workerSize, "start_error": err.Error(), "lookup_error": lookupErr.Error()}, "", "  ")
			if writeErr := writeDurable(attempt, "phone-motion-dispatch-unknown.json", append(record, '\n')); writeErr != nil {
				t.Errorf("persist ambiguous motion dispatch evidence: %v", writeErr)
			}
			t.Fatalf("ambiguous video dispatch; exact lookup failed and no redispatch was attempted: %v / %v", err, lookupErr)
		}
		op = &platform.OwnedOperation{Ref: ref}
	}
	settled := false
	t.Cleanup(func() {
		if settled {
			return
		}
		cleanup, done := context.WithTimeout(context.Background(), 15*time.Second)
		defer done()
		stop, stopErr := platform.StopOwnedOperation(cleanup, op.Ref)
		obs, inspectErr := platform.InspectOwnedOperation(cleanup, op.Ref)
		record := map[string]any{"ref": op.Ref, "stop": stop, "observation": obs, "released": false}
		if stopErr != nil {
			record["stop_error"] = stopErr.Error()
		}
		if inspectErr != nil {
			record["inspect_error"] = inspectErr.Error()
		}
		if stopErr == nil && inspectErr == nil && stop.NativeQuiescent && obs.ExitKnown && !obs.Populated {
			if releaseErr := platform.ReleaseOwnedOperation(cleanup, op.Ref); releaseErr == nil {
				record["released"] = true
			} else {
				record["release_error"] = releaseErr.Error()
			}
		}
		b, _ := json.MarshalIndent(record, "", "  ")
		if writeErr := writeDurable(attempt, "phone-motion-cleanup-attempt.json", append(b, '\n')); writeErr != nil {
			t.Errorf("persist phone-motion cleanup observation: %v", writeErr)
		}
		if record["released"] != true {
			t.Errorf("phone-motion exact native scope could not be released: %v / %v", stopErr, inspectErr)
		}
	})
	refBytes, _ := json.MarshalIndent(op.Ref, "", "  ")
	if err = writeDurable(attempt, "phone-motion-operation-ref.json", append(refBytes, '\n')); err != nil {
		t.Fatal(err)
	}
	waitErr := op.Wait(opCtx)
	stop, stopErr := platform.StopOwnedOperation(opCtx, op.Ref)
	obs, inspectErr := platform.InspectOwnedOperation(opCtx, op.Ref)
	terminal, _ := json.MarshalIndent(map[string]any{"ref": op.Ref, "wait_error": errorText(waitErr), "stop": stop, "stop_error": errorText(stopErr), "observation": obs, "inspect_error": errorText(inspectErr)}, "", "  ")
	if err = writeDurable(attempt, "phone-motion-operation-terminal.json", append(terminal, '\n')); err != nil {
		t.Fatal(err)
	}
	if waitErr != nil || stopErr != nil || inspectErr != nil || !stop.NativeQuiescent || !obs.ExitKnown || obs.Populated || obs.Result != "success" || obs.ExecMainStatus != 0 {
		t.Fatalf("owned video encode/decode failed or did not quiesce: wait=%v stop=%v inspect=%v result=%s status=%d", waitErr, stopErr, inspectErr, obs.Result, obs.ExecMainStatus)
	}
	afterCtx, afterCancel := context.WithTimeout(context.Background(), 20*time.Second)
	afterProfile, afterErr := ResolveInspectionProfile(afterCtx)
	afterCancel()
	afterPair, pairErr := LocalToolPairFromProfile(afterProfile)
	afterBytes, marshalErr := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(afterProfile)
	if marshalErr == nil {
		marshalErr = writeDurable(attempt, "resolved-video-profile-after.json", afterBytes)
	}
	if afterErr != nil || pairErr != nil || marshalErr != nil || afterPair != pair || profileArtifactIdentity(afterProfile) != pair.ArtifactIdentity {
		t.Fatalf("FFmpeg profile or exact tool paths changed during motion encode: %v / %v / %v", afterErr, pairErr, marshalErr)
	}
	if err = verifyPhoneMotionEncodeOutput(attempt, workerSHA, pair, bindings); err != nil {
		t.Fatal(err)
	}
	if err = platform.ReleaseOwnedOperation(opCtx, op.Ref); err != nil {
		t.Fatal(err)
	}
	settled = true
	released, _ := json.MarshalIndent(map[string]any{"ref": op.Ref, "released": true}, "", "  ")
	if err = writeDurable(attempt, "phone-motion-operation-released.json", append(released, '\n')); err != nil {
		t.Fatal(err)
	}
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func inspectPhoneMotionEncodedDuration(ctx context.Context, ffprobe *os.File, root *os.Root) ([]byte, int64, composition.Rational, error) {
	var zero composition.Rational
	output, err := root.Open(VideoOutputName)
	if err != nil {
		return nil, 0, zero, err
	}
	defer output.Close()
	args := []string{"-v", "error", "-protocol_whitelist", "file,pipe", "-format_whitelist", "mov,matroska,webm", "-max_streams", "8", "-select_streams", "v:0", "-show_entries", "stream=index,codec_name,duration_ts,time_base", "-show_streams", "-of", "json", inspectionInputDescriptor}
	raw, err := (execInspectionRunner{}).Probe(ctx, ffprobe, args, output, inspectionMaxProbe)
	if err != nil {
		return nil, 0, zero, err
	}
	ticks, timeBase, err := phoneMotionDurationFromProbe(raw)
	return raw, ticks, timeBase, err
}

func phoneMotionDurationFromProbe(raw []byte) (int64, composition.Rational, error) {
	var zero composition.Rational
	var document phoneMotionProbeDocument
	if err := json.Unmarshal(raw, &document); err != nil || len(document.Streams) != 1 {
		return 0, zero, errors.New("motion output probe must contain one video stream")
	}
	stream := document.Streams[0]
	ticks, ok := parseRawInt(stream.DurationTS)
	baseNum, baseDen, rateOK := parseRational(stream.TimeBase)
	if !ok || ticks <= 0 || !rateOK || baseNum <= 0 || baseDen <= 0 || stream.Index != 0 || stream.Codec != "h264" {
		return 0, zero, errors.New("motion output has no exact positive H.264 stream duration")
	}
	left := new(big.Int).Mul(big.NewInt(ticks), big.NewInt(baseNum))
	right := new(big.Int).Mul(big.NewInt(2), big.NewInt(baseDen))
	if left.Cmp(right) != 0 {
		return 0, zero, errors.New("motion output stream duration does not equal exactly two seconds")
	}
	return ticks, composition.Rational{Num: baseNum, Den: baseDen}, nil
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func verifyPhoneMotionEncodeOutput(attempt, workerSHA string, pair LocalToolPair, bindings phoneMotionSourceBinding) error {
	receiptBytes, err := readMotionRegular(attempt, "phone-motion-video-receipt.json", 1<<20)
	if err != nil {
		return err
	}
	var receipt phoneMotionEncodeReceipt
	if err = json.Unmarshal(receiptBytes, &receipt); err != nil {
		return err
	}
	if receipt.Schema != 1 || receipt.SourceManifestSHA != bindings.SourceManifestSHA || receipt.StagedManifestSHA != bindings.StagedManifestSHA || receipt.ScriptSHA != bindings.ScriptSHA || receipt.WorkerSHA != workerSHA || receipt.WorkerBytes <= 0 || receipt.WorkerBytes > 256<<20 || receipt.ProfileID != pair.ProfileID || receipt.ProfileArtifact != pair.ArtifactIdentity || receipt.FFmpegSHA != pair.FFmpegSHA256 || receipt.FFprobeSHA != pair.FFprobeSHA256 || receipt.OutputName != VideoOutputName || receipt.OutputBytes <= 0 || receipt.OutputBytes > videoMaxOutputBytes || receipt.DurationNumerator != 2 || receipt.DurationDenominator != 1 {
		return errors.New("motion video receipt does not bind the fixed source, worker, tools, or two-second profile")
	}
	rawProbe, err := readMotionRegular(attempt, "phone-motion-output-ffprobe.json", inspectionMaxProbe)
	if err != nil || int64(len(rawProbe)) != receipt.FFprobeJSONBytes || blenderDigest(rawProbe) != receipt.FFprobeJSONSHA {
		return errors.New("retained raw output probe is missing or differs from its receipt binding")
	}
	probeTicks, probeTimeBase, err := phoneMotionDurationFromProbe(rawProbe)
	if err != nil || probeTicks != receipt.StreamDurationTicks || probeTimeBase != receipt.StreamTimeBase || probeTimeBase != receipt.Native.TimeBase {
		return errors.New("retained output probe duration facts differ from native video receipt")
	}
	if receipt.Native.Width != 1920 || receipt.Native.Height != 1080 || receipt.Native.FrameRate.Num != 24 || receipt.Native.FrameRate.Den != 1 || receipt.Native.FrameCount != 48 || receipt.Native.FirstPTS != 0 || !receipt.Native.FullDecodeVerified || receipt.Native.Codec != "h264" || receipt.Native.Container != "mov,mp4,m4a,3gp,3g2,mj2" || receipt.Native.PixelFormat != "yuv420p" || receipt.Native.ColorPrimaries != "bt709" || receipt.Native.ColorTransfer != "bt709" || receipt.Native.ColorMatrix != "bt709" || (receipt.Native.ColorRange != "tv" && receipt.Native.ColorRange != "mpeg") {
		return errors.New("motion video native facts differ from the pinned output profile")
	}
	wantPTS, err := plannedOutputPTS(VideoManifest{Width: 1920, Height: 1080, OutputRate: composition.Rational{Num: 24, Den: 1}}, receipt.Native.TimeBase, 48)
	if err != nil || len(wantPTS) != 48 || receipt.Native.LastPTS != wantPTS[47] {
		return errors.New("motion video final PTS does not terminate the exact 48-tick schedule")
	}
	_, _, _, names, _ := phoneStudyProfile("motion-yaw-24")
	var source phoneStudyReceipt
	frameBytes, err := readMotionRegular(filepath.Join(attempt, "outputs"), "qualification-receipt.json", 1<<20)
	if err != nil || json.Unmarshal(frameBytes, &source) != nil || len(source.Outputs) != len(names) || len(receipt.Frames) != len(names) {
		return errors.New("motion encode receipt does not match the verified frame receipt")
	}
	var total int64
	for i, frame := range receipt.Frames {
		original := source.Outputs[i]
		if frame.Name != names[i] || frame.Name != original.Path || frame.SHA256 != original.SHA || frame.Bytes != original.Bytes || frame.Bytes <= 0 || frame.Bytes > 7<<20 {
			return fmt.Errorf("motion encode receipt frame binding %d differs", i)
		}
		total += frame.Bytes
	}
	if total > 336<<20 {
		return errors.New("motion frame input aggregate exceeds its bound")
	}
	output, err := readMotionRegular(attempt, VideoOutputName, videoMaxOutputBytes)
	if err != nil || int64(len(output)) != receipt.OutputBytes || blenderDigest(output) != receipt.OutputSHA {
		return errors.New("encoded motion MP4 size or digest differs from its verified receipt")
	}
	worker, err := readMotionRegular(attempt, "phone-motion-encoder-worker", 256<<20)
	if err != nil || int64(len(worker)) != receipt.WorkerBytes || blenderDigest(worker) != workerSHA {
		return errors.New("owned video worker bytes changed during execution")
	}
	return nil
}

func phoneImageRGB24(img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	if bounds.Dx() != 1920 || bounds.Dy() != 1080 {
		return nil, errors.New("motion PNG raster is outside fixed RGB24 bounds")
	}
	rgb := make([]byte, bounds.Dx()*bounds.Dy()*3)
	index := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a != 0xffff {
				return nil, errors.New("motion PNG unexpectedly contains transparency")
			}
			rgb[index], rgb[index+1], rgb[index+2] = byte(r>>8), byte(g>>8), byte(b>>8)
			index += 3
		}
	}
	return rgb, nil
}

func readPhoneMotionBindings(attempt string) (phoneMotionSourceBinding, error) {
	var bindings phoneMotionSourceBinding
	data, err := readMotionRegular(attempt, "source-bindings.json", 32<<10)
	if err != nil {
		return bindings, err
	}
	err = json.Unmarshal(data, &bindings)
	return bindings, err
}

func readMotionRegular(root, name string, max int64) ([]byte, error) {
	if filepath.Base(name) != name {
		return nil, errors.New("motion worker filename must be a fixed basename")
	}
	path := filepath.Join(root, name)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > max {
		return nil, fmt.Errorf("motion worker input %q is missing, unsafe, or oversized", name)
	}
	data, err := os.ReadFile(path)
	if err != nil || int64(len(data)) != info.Size() {
		return nil, fmt.Errorf("motion worker input %q changed while reading", name)
	}
	return data, nil
}

func currentExecutableSHA(t *testing.T) string {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return blenderDigest(data)
}

func TestPhoneMotionScheduleRejectsTimingPoseAndContactDrift(t *testing.T) {
	_, _, _, names, yaws := phoneStudyProfile("motion-yaw-24")
	if len(names) != 48 || names[0] != "motion-yaw-frame-0000.png" || names[47] != "motion-yaw-frame-0047.png" || yaws[0] != -13 || yaws[47] != 32 {
		t.Fatalf("fixed motion schedule endpoints differ: names=%d yaws=%v", len(names), []float64{yaws[0], yaws[len(yaws)-1]})
	}
	if math.Abs(yaws[23]-8.782023251110065) > 1e-12 || math.Abs(yaws[24]-10.217976748889935) > 1e-12 {
		t.Fatalf("middle motion poses differ from pinned smoothstep sequence: %.15f %.15f", yaws[23], yaws[24])
	}
	for i := 1; i < len(yaws); i++ {
		if yaws[i] <= yaws[i-1] {
			t.Fatalf("motion yaw did not progress at ordinal %d", i)
		}
	}
	valid := validPhoneMotionReceipt(yaws)
	if err := verifyPhoneMotionTimeline(valid, yaws); err != nil {
		t.Fatal(err)
	}
	*valid.Outputs[17].PTSNum++
	if err := verifyPhoneMotionTimeline(valid, yaws); err == nil {
		t.Fatal("duplicate/noncanonical frame timestamp was accepted")
	}
	valid = validPhoneMotionReceipt(yaws)
	valid.Outputs[3].Contacts.ConnectorEnd.Count = 0
	if err := validatePhoneMotionOutput(valid.Outputs[3], 3, yaws[3]); err == nil {
		t.Fatal("frame without connector support was accepted")
	}
	valid = validPhoneMotionReceipt(yaws)
	valid.Outputs[4].Bounds.Min[2] = 0.749
	if err := validatePhoneMotionOutput(valid.Outputs[4], 4, yaws[4]); err == nil {
		t.Fatal("table penetration beyond tolerance was accepted")
	}
	valid = validPhoneMotionReceipt(yaws)
	valid.Outputs[12].AnchorTranslation[2] += 0.01
	if err := verifyPhoneMotionTimeline(valid, yaws); err == nil {
		t.Fatal("per-frame placement correction was accepted")
	}
}

func TestPhoneMotionProfileLeavesApprovedStaticProfilesUnchanged(t *testing.T) {
	profiles := []struct {
		name   string
		width  int
		height int
		sample int
		files  []string
		yaws   []float64
	}{
		{"study", 960, 540, 64, []string{"study-stationary.png"}, []float64{-13}},
		{"hero", 2560, 1440, 256, []string{"hero-stationary.png"}, []float64{-13}},
		{"posecontactsheet", 960, 540, 96, []string{"pose-yaw-m13.png", "pose-yaw-p32.png", "pose-yaw-p77.png"}, []float64{-13, 32, 77}},
	}
	for _, want := range profiles {
		w, h, samples, files, yaws := phoneStudyProfile(want.name)
		if w != want.width || h != want.height || samples != want.sample || !reflect.DeepEqual(files, want.files) || !reflect.DeepEqual(yaws, want.yaws) {
			t.Errorf("approved static profile %q changed: %dx%d samples=%d files=%v yaws=%v", want.name, w, h, samples, files, yaws)
		}
	}
}

func TestPhoneMotionProbeRequiresExactTwoSecondVideoDuration(t *testing.T) {
	valid := []byte(`{"streams":[{"index":0,"codec_name":"h264","duration_ts":24576,"time_base":"1/12288"}]}`)
	ticks, base, err := phoneMotionDurationFromProbe(valid)
	if err != nil || ticks != 24576 || base != (composition.Rational{Num: 1, Den: 12288}) {
		t.Fatalf("valid exact duration was rejected: ticks=%d base=%+v err=%v", ticks, base, err)
	}
	for _, invalid := range [][]byte{
		[]byte(`{"streams":[{"index":0,"codec_name":"h264","duration_ts":24575,"time_base":"1/12288"}]}`),
		[]byte(`{"streams":[{"index":0,"codec_name":"h264","duration_ts":24576,"time_base":"1/12000"}]}`),
		[]byte(`{"streams":[{"index":0,"codec_name":"h264","duration_ts":24576,"time_base":"1/12288"},{"index":1,"codec_name":"h264","duration_ts":24576,"time_base":"1/12288"}]}`),
	} {
		if _, _, err := phoneMotionDurationFromProbe(invalid); err == nil {
			t.Fatalf("invalid two-second stream facts were accepted: %s", invalid)
		}
	}
}

func TestPhoneStudyCameraReceiptRequiresBoundedWorldTransforms(t *testing.T) {
	valid := phoneStudyCamera{SensorWidthMM: 36, SensorHeightMM: 24, SensorFit: "AUTO", ClipStartMeters: 0.1, ClipEndMeters: 100, TargetedPhoneAnchor: true, DOFFStop: 11}
	valid.CameraMatrixWorld = identityPhoneMatrix()
	valid.FocusMatrixWorld = identityPhoneMatrix()
	if !validPhoneCameraFacts(valid) {
		t.Fatal("complete camera and focus transforms were rejected")
	}
	invalid := valid
	invalid.CameraMatrixWorld = [][]float64{{math.NaN()}}
	if validPhoneCameraFacts(invalid) {
		t.Fatal("non-finite or incomplete camera transform was accepted")
	}
}

func identityPhoneMatrix() [][]float64 {
	return [][]float64{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, 1}}
}

func validPhoneMotionReceipt(yaws []float64) phoneStudyReceipt {
	r := phoneStudyReceipt{Timeline: &phoneStudyTimeline{Profile: "rigid-world-z-smoothstep-v1", FrameCount: 48, RateNum: 24, RateDen: 1, DurationTicks: 48, LastIndex: 47, FinalHold: 1}}
	r.Outputs = make([]phoneStudyOutput, len(yaws))
	for i, yaw := range yaws {
		index, num, den := i, i, 24
		frame := phoneStudyOutput{Yaw: yaw, FrameIndex: &index, PTSNum: &num, PTSDen: &den, AnchorTranslation: []float64{0, 0, 0.75}, ScreenNormal: []float64{0, 0, 1}}
		frame.Bounds.Min = []float64{-0.1, -0.2, 0.75}
		frame.Bounds.Max = []float64{0.1, 0.2, 0.9}
		frame.Contacts.CameraEnd.Count, frame.Contacts.CameraEnd.Z = 20, []float64{0.75, 0.75}
		frame.Contacts.ConnectorEnd.Count, frame.Contacts.ConnectorEnd.Z = 22, []float64{0.75, 0.75}
		r.Outputs[i] = frame
	}
	return r
}
