package rendering

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"device-studio/internal/composition"
)

const (
	videoQualificationProfileSHA  = "3737acd3d9e5f5fee3b4986d17d003af51fea3fea32c5a8c51310cb95fba7900"
	videoQualificationFFmpegSHA   = "dcf865aada2dc5f80fcf57f97c7ab34aa6ea651faaa20375833933b16e9ad372"
	videoQualificationFFprobeSHA  = "b517813b5de7c5b837a2a1d8ac052b79e17938137694ef99f2e7c42aa21468ce"
	videoQualificationPhoneSHA    = "4bf3e1663ce60116e85e4172f359ac9ce7a2b39a99bf4fa5d8b2e0fe040db857"
	videoQualificationPhoneBytes  = int64(473562)
	videoQualificationOutputLimit = int64(160 << 20)
)

type videoQualificationResult struct {
	ProfileID        string `json:"profile_id"`
	ProfileDigest    string `json:"profile_artifact_sha256"`
	FFmpegSHA256     string `json:"ffmpeg_sha256"`
	FFprobeSHA256    string `json:"ffprobe_sha256"`
	SourceSHA256     string `json:"source_sha256,omitempty"`
	ColorPolicy      string `json:"color_policy,omitempty"`
	ColorFrameBytes  int    `json:"color_frame_bytes,omitempty"`
	ColorFrameSHA256 string `json:"color_frame_sha256,omitempty"`
	EncoderWidth     int    `json:"encoder_width,omitempty"`
	EncoderHeight    int    `json:"encoder_height,omitempty"`
	EncoderFrames    int    `json:"encoder_frames,omitempty"`
	MP4Bytes         int64  `json:"mp4_bytes,omitempty"`
	MP4SHA256        string `json:"mp4_sha256,omitempty"`
	Probe            string `json:"probe_json,omitempty"`
	DecodeBytes      int    `json:"decoded_rgb_bytes,omitempty"`
	DecodeSHA256     string `json:"decoded_rgb_sha256,omitempty"`
	RawRGBFrameCount int    `json:"raw_rgb_frame_count,omitempty"`
	RawRGBPTSCount   int    `json:"raw_rgb_pts_count,omitempty"`
	RawRGBStreamSHA  string `json:"raw_rgb_stream_sha256,omitempty"`
	ElapsedMillis    int64  `json:"elapsed_millis,omitempty"`
	ClockFrames      int    `json:"clock_frame_count,omitempty"`
	ClockSHA256      string `json:"clock_sha256,omitempty"`
}

// TestPinnedVideoProtocolQualification is deliberately opt-in. It exercises
// only the current Linux host pair, authentic read-only source, and bounded
// synthetic output; it does not launch the scenario worker or touch a lease.
func TestPinnedVideoProtocolQualification(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_VIDEO_PROTOCOL_QUALIFY") != "1" {
		t.Skip("set DEVICE_STUDIO_VIDEO_PROTOCOL_QUALIFY=1 for the admitted bounded host qualification")
	}
	sourcePath := os.Getenv("DEVICE_STUDIO_VIDEO_PROTOCOL_SOURCE")
	outDir := os.Getenv("DEVICE_STUDIO_VIDEO_PROTOCOL_OUTPUT")
	if sourcePath == "" || outDir == "" {
		t.Fatal("source and evidence output directory are required")
	}
	cleanOut := filepath.Clean(outDir)
	if filepath.Base(cleanOut) != "video-protocol" || !strings.Contains(filepath.ToSlash(cleanOut), "/goal/evidence/E2V/video-protocol") {
		t.Fatalf("evidence output must be the admitted E2V/video-protocol directory: %q", outDir)
	}
	profilePath := "../../../docs/internal/goal/evidence/E2V/tool-profile/current-host/profile.json"
	profileBytes, err := os.ReadFile(filepath.Clean(profilePath))
	if err != nil {
		t.Fatal(err)
	}
	if digest(profileBytes) != videoQualificationProfileSHA {
		t.Fatal("current-host tool profile artifact changed from the admitted identity")
	}
	var profile struct {
		ProfileID        string `json:"profile_id"`
		ArtifactIdentity string `json:"artifact_identity"`
		FFmpeg           struct {
			Path, SHA256 string
			SizeBytes    string `json:"size_bytes"`
		} `json:"ffmpeg"`
		FFprobe struct {
			Path, SHA256 string
			SizeBytes    string `json:"size_bytes"`
		} `json:"ffprobe"`
	}
	if err := json.Unmarshal(profileBytes, &profile); err != nil {
		t.Fatal(err)
	}
	if profile.ProfileID != inspectionProfileID || profile.FFmpeg.SHA256 != videoQualificationFFmpegSHA || profile.FFprobe.SHA256 != videoQualificationFFprobeSHA {
		t.Fatal("profile member identities differ from the admitted pair")
	}
	ffmpegSize, err := strconv.ParseInt(profile.FFmpeg.SizeBytes, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	ffprobeSize, err := strconv.ParseInt(profile.FFprobe.SizeBytes, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	ffmpeg, err := openVerifiedTool(ctx, profile.FFmpeg.Path, ffmpegSize, profile.FFmpeg.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	defer ffmpeg.Close()
	ffprobe, err := openVerifiedTool(ctx, profile.FFprobe.Path, ffprobeSize, profile.FFprobe.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	defer ffprobe.Close()
	started := time.Now()
	result := videoQualificationResult{ProfileID: profile.ProfileID, ProfileDigest: profile.ArtifactIdentity, FFmpegSHA256: profile.FFmpeg.SHA256, FFprobeSHA256: profile.FFprobe.SHA256}

	source, err := openExactQualificationSource(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	result.SourceSHA256 = videoQualificationPhoneSHA
	if manifestPath := os.Getenv("DEVICE_STUDIO_VIDEO_PROTOCOL_MANIFEST"); manifestPath != "" {
		clockPath := os.Getenv("DEVICE_STUDIO_VIDEO_PROTOCOL_CLOCK")
		receiptPath := os.Getenv("DEVICE_STUDIO_VIDEO_PROTOCOL_RECEIPT")
		if clockPath == "" || receiptPath == "" {
			t.Fatal("clock and receipt paths must be supplied with the manifest")
		}
		if err := compareFullDecodeToRetainedClock(ctx, ffmpeg, ffprobe, source, manifestPath, receiptPath, clockPath, &result); err != nil {
			t.Fatalf("full authentic decode/clock replay: %v", err)
		}
		if _, err := source.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}
	}
	colorArgs := []string{"-hide_banner", "-nostdin", "-v", "error", "-protocol_whitelist", "file,pipe", "-format_whitelist", "mov,matroska,webm", "-max_streams", "8", "-i", inspectionInputDescriptor, "-map", "0:0", "-frames:v", "1", "-vf", "colorspace=ispace=bt470bg:irange=tv:iprimaries=bt709:itrc=bt709:space=gbr:range=pc:primaries=bt709:trc=iec61966-2-1,format=rgb24", "-f", "rawvideo", "pipe:1"}
	colorFrame, err := runPinnedOutput(ctx, ffmpeg, source, colorArgs, 430*880*3)
	if err != nil {
		t.Fatalf("pinned SDR conversion failed: %v", err)
	}
	if len(colorFrame) != 430*880*3 {
		t.Fatalf("converted RGB frame bytes=%d, want %d", len(colorFrame), 430*880*3)
	}
	result.ColorPolicy = "unmarked primaries/transfer default BT.709; observed bt470bg matrix + limited range; colorspace output BT.709 primaries + IEC 61966-2-1 sRGB transfer + GBR/full range"
	result.ColorFrameBytes = len(colorFrame)
	result.ColorFrameSHA256 = canonicalRGBDigest(430, 880, "bt709-primaries/srgb-transfer/gbr-full", colorFrame)
	if err := verifyExactQualificationSource(source, sourcePath); err != nil {
		t.Fatal(err)
	}
	colorFrame = nil

	const width, height, frames = 1920, 1080, 2
	rgb := syntheticRGBFrame(width, height)
	tmp, err := os.CreateTemp("", "device-studio-p5-*.mp4")
	if err != nil {
		t.Fatal(err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	encodeArgs := []string{"-hide_banner", "-nostdin", "-v", "error", "-f", "rawvideo", "-pix_fmt", "rgb24", "-video_size", "1920x1080", "-framerate", "24", "-i", "pipe:0", "-frames:v", "2", "-vf", "zscale=primariesin=bt709:transferin=iec61966-2-1:matrixin=gbr:rangein=full:primaries=bt709:transfer=bt709:matrix=bt709:range=limited,format=yuv420p", "-c:v", "libx264", "-preset", "ultrafast", "-crf", "24", "-color_primaries", "bt709", "-color_trc", "bt709", "-colorspace", "bt709", "-color_range", "tv", "-an", "-movflags", "+frag_keyframe+empty_moov+default_base_moof", "-f", "mp4", "pipe:1"}
	if err := encodeRawMP4(ctx, ffmpeg, rgb, frames, tmp, encodeArgs, videoQualificationOutputLimit); err != nil {
		t.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(tmpPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() <= 0 || info.Size() > videoQualificationOutputLimit {
		t.Fatalf("MP4 size %d outside hard bound", info.Size())
	}
	tmpRead, err := os.Open(tmpPath)
	if err != nil {
		t.Fatal(err)
	}
	outputHash := sha256.New()
	if _, err := io.Copy(outputHash, io.LimitReader(tmpRead, videoQualificationOutputLimit+1)); err != nil {
		t.Fatal(err)
	}
	if err := tmpRead.Close(); err != nil {
		t.Fatal(err)
	}
	result.EncoderWidth, result.EncoderHeight, result.EncoderFrames = width, height, frames
	result.MP4Bytes, result.MP4SHA256 = info.Size(), hex.EncodeToString(outputHash.Sum(nil))

	probeFile, err := os.Open(tmpPath)
	if err != nil {
		t.Fatal(err)
	}
	probeArgs := []string{"-hide_banner", "-v", "error", "-protocol_whitelist", "file,pipe", "-format_whitelist", "mov,matroska,webm", "-max_streams", "8", "-count_frames", "-show_entries", "format=format_name:stream=codec_name,width,height,pix_fmt,avg_frame_rate,color_range,color_space,color_transfer,color_primaries,nb_frames,nb_read_frames", "-of", "json", "/proc/self/fd/4"}
	probeJSON, err := runPinnedOutput(ctx, ffprobe, probeFile, probeArgs, 1<<20)
	_ = probeFile.Close()
	if err != nil {
		t.Fatalf("fragmented MP4 probe failed: %v", err)
	}
	var parsed struct {
		Streams []struct {
			Codec       string `json:"codec_name"`
			Width       int    `json:"width"`
			Height      int    `json:"height"`
			PixelFormat string `json:"pix_fmt"`
			AvgRate     string `json:"avg_frame_rate"`
			Range       string `json:"color_range"`
			Matrix      string `json:"color_space"`
			Transfer    string `json:"color_transfer"`
			Primaries   string `json:"color_primaries"`
			Frames      string `json:"nb_read_frames"`
		} `json:"streams"`
		Format struct {
			Name string `json:"format_name"`
		} `json:"format"`
	}
	if err := json.Unmarshal(probeJSON, &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Streams) != 1 || parsed.Streams[0].Codec != "h264" || parsed.Streams[0].Width != width || parsed.Streams[0].Height != height || parsed.Streams[0].PixelFormat != "yuv420p" || parsed.Streams[0].AvgRate != "24/1" || parsed.Streams[0].Frames != "2" || parsed.Streams[0].Range != "tv" || parsed.Streams[0].Matrix != "bt709" || parsed.Streams[0].Transfer != "bt709" || parsed.Streams[0].Primaries != "bt709" || !strings.Contains(parsed.Format.Name, "mp4") {
		t.Fatalf("fragmented MP4 metadata differs from the qualified contract: %s", probeJSON)
	}
	result.Probe = string(probeJSON)

	decodeFile, err := os.Open(tmpPath)
	if err != nil {
		t.Fatal(err)
	}
	decodeArgs := []string{"-hide_banner", "-nostdin", "-v", "error", "-protocol_whitelist", "file,pipe", "-format_whitelist", "mov,matroska,webm", "-max_streams", "8", "-i", inspectionInputDescriptor, "-map", "0:0", "-fps_mode", "passthrough", "-pix_fmt", "rgb24", "-f", "rawvideo", "pipe:1"}
	decoded, err := runPinnedOutput(ctx, ffmpeg, decodeFile, decodeArgs, int64(width*height*3*frames))
	_ = decodeFile.Close()
	if err != nil {
		t.Fatalf("fragmented MP4 full decode failed: %v", err)
	}
	if len(decoded) != width*height*3*frames {
		t.Fatalf("decoded RGB bytes=%d, want %d", len(decoded), width*height*3*frames)
	}
	result.DecodeBytes = len(decoded)
	result.DecodeSHA256 = canonicalRGBDigest(width, height, "bt709-primaries/bt709-transfer/gbr-full", decoded)
	result.ElapsedMillis = time.Since(started).Milliseconds()
	if result.ElapsedMillis > 60_000 {
		t.Fatalf("bounded qualification took %d ms", result.ElapsedMillis)
	}
	if err := writeQualificationJSON(filepath.Join(cleanOut, "qualification.json"), result); err != nil {
		t.Fatal(err)
	}
}

func openExactQualificationSource(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() != videoQualificationPhoneBytes {
		return nil, fmt.Errorf("authentic phone source is not the exact regular file: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		f.Close()
		return nil, fmt.Errorf("source changed while opening")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, videoQualificationPhoneBytes+1))
	if err != nil || n != videoQualificationPhoneBytes || hex.EncodeToString(h.Sum(nil)) != videoQualificationPhoneSHA {
		f.Close()
		return nil, fmt.Errorf("source identity differs from retained authentic fixture")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func verifyExactQualificationSource(f *os.File, path string) error {
	before, err := f.Stat()
	if err != nil {
		return err
	}
	h := sha256.New()
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if _, err := io.Copy(h, io.LimitReader(f, videoQualificationPhoneBytes+1)); err != nil {
		return err
	}
	after, err := f.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(before, after) || before.Size() != videoQualificationPhoneBytes || hex.EncodeToString(h.Sum(nil)) != videoQualificationPhoneSHA {
		return fmt.Errorf("source changed during tool use")
	}
	pathInfo, err := os.Lstat(path)
	if err != nil || pathInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, pathInfo) {
		return fmt.Errorf("source path no longer names the verified file")
	}
	return nil
}

func compareFullDecodeToRetainedClock(ctx context.Context, ffmpeg, ffprobe, original *os.File, manifestPath, receiptPath, clockPath string, result *videoQualificationResult) error {
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	receiptBytes, err := os.ReadFile(receiptPath)
	if err != nil {
		return err
	}
	clockBytes, err := os.ReadFile(clockPath)
	if err != nil {
		return err
	}
	stored, err := VerifyInspectionArtifacts(manifestBytes, receiptBytes, clockBytes)
	if err != nil {
		return fmt.Errorf("retained successful P4b artifacts fail verification: %w", err)
	}
	var manifest InspectionManifest
	if err := decodeCanonicalInspectionManifest(manifestBytes, &manifest); err != nil {
		return err
	}
	if manifest.Source.SHA256 != videoQualificationPhoneSHA || manifest.Source.SizeBytes != videoQualificationPhoneBytes || stored.Classification != InspectionAccepted {
		return fmt.Errorf("retained artifacts do not bind the admitted authentic phone source")
	}
	dir, err := os.MkdirTemp("", "device-studio-p5-clock-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if _, err := original.Seek(0, io.SeekStart); err != nil {
		return err
	}
	copyFile, err := os.Create(filepath.Join(dir, InspectionSourceName))
	if err != nil {
		return err
	}
	if _, err := io.Copy(copyFile, io.LimitReader(original, videoQualificationPhoneBytes+1)); err != nil {
		copyFile.Close()
		return err
	}
	if err := copyFile.Close(); err != nil {
		return err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	source, err := root.Open(InspectionSourceName)
	if err != nil {
		return err
	}
	defer source.Close()
	metadata, rejection, err := probeInspectionSource(ctx, execInspectionRunner{}, ffprobe, source)
	if err != nil || rejection != nil {
		return fmt.Errorf("metadata probe: rejection=%v error=%v", rejection, err)
	}
	schedule, rejection, err := probeInspectionFrames(ctx, execInspectionRunner{}, ffprobe, source)
	if err != nil || rejection != nil {
		return fmt.Errorf("frame probe: rejection=%v error=%v", rejection, err)
	}
	storedPTS, err := decodeInspectionClock(clockBytes, stored.FrameCount)
	if err != nil {
		return err
	}
	if !equalInt64s(schedule.PTS, storedPTS) || schedule.LastFrameTicks != stored.LastFrameDurationTicks {
		return fmt.Errorf("fresh probe schedule differs from retained P4b clock")
	}
	var video inspectionStream
	for _, candidate := range metadata.Streams {
		if candidate.Index == int(*stored.VideoStreamIndex) {
			video = candidate
			break
		}
	}
	if video.CodecType != "video" {
		return fmt.Errorf("retained video stream index is missing from fresh probe")
	}
	if err := streamRGBAndComparePTS(ctx, ffmpeg, source, video, stored.TimeBaseNum, stored.TimeBaseDen, storedPTS, result); err != nil {
		return err
	}
	if err := verifyInspectionSourceUnchanged(ctx, source, manifest.Source); err != nil {
		return err
	}
	result.ClockFrames, result.ClockSHA256 = int(stored.FrameCount), digest(clockBytes)
	return nil
}

func streamRGBAndComparePTS(ctx context.Context, tool, source *os.File, video inspectionStream, timeNum, timeDen int64, expected []int64, result *videoQualificationResult) error {
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return err
	}
	statsRead, statsWrite, err := os.Pipe()
	if err != nil {
		return err
	}
	defer statsRead.Close()
	defer statsWrite.Close()
	args := []string{"-hide_banner", "-nostdin", "-v", "error", "-protocol_whitelist", "file,pipe", "-format_whitelist", "mov,matroska,webm", "-max_streams", "8", "-xerror", "-err_detect", "explode", "-copyts", "-i", inspectionInputDescriptor, "-map", fmt.Sprintf("0:%d", video.Index), "-an", "-sn", "-dn", "-fps_mode", "passthrough", "-enc_time_base:v", fmt.Sprintf("%d/%d", timeNum, timeDen), "-vf", "colorspace=ispace=bt470bg:irange=tv:iprimaries=bt709:itrc=bt709:space=gbr:range=pc:primaries=bt709:trc=iec61966-2-1,format=rgb24", "-stats_enc_pre", "pipe:5", "-stats_enc_pre_fmt", "{n} {pts} {tb}", "-c:v", "rawvideo", "-pix_fmt", "rgb24", "-f", "rawvideo", "pipe:1"}
	cmd := exec.CommandContext(ctx, "/proc/self/fd/3", args...)
	cmd.Env = []string{"LANG=C", "LC_ALL=C"}
	cmd.ExtraFiles = []*os.File{tool, source, statsWrite}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr, stats boundedInspectionBuffer
	stderr.limit, stats.limit = inspectionMaxStderr, inspectionMaxStats
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = statsWrite.Close()
	statsDone := make(chan error, 1)
	go func() { _, copyErr := io.Copy(&stats, statsRead); statsDone <- copyErr }()
	frameBytes := int64(video.Width) * int64(video.Height) * 3
	if frameBytes <= 0 || frameBytes > int64(inspectionMaxPixels)*3 {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("raw frame dimensions exceed the fixed bound")
	}
	streamHash := sha256.New()
	if err := hashRGBFrameStream(stdout, video.Width, video.Height, len(expected), streamHash); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("pinned raw RGB decode failed: %w: %s", err, truncateUTF8(string(stderr.Bytes()), 2048))
	}
	if err := <-statsDone; err != nil || stats.overflow {
		return fmt.Errorf("encoder-pre PTS stream exceeded its bound or failed: %v", err)
	}
	if stderr.overflow {
		return fmt.Errorf("native stderr exceeded its fixed bound")
	}
	if err := compareDecodePTS(stats.Bytes(), expected, timeNum, timeDen); err != nil {
		return err
	}
	result.RawRGBFrameCount, result.RawRGBPTSCount = len(expected), len(bytes.Split(bytes.TrimSpace(stats.Bytes()), []byte{'\n'}))
	result.RawRGBStreamSHA = hex.EncodeToString(streamHash.Sum(nil))
	return nil
}

func hashRGBFrameStream(r io.Reader, width, height, frames int, h io.Writer) error {
	if width <= 0 || height <= 0 || frames <= 0 || int64(width)*int64(height)*3 > int64(inspectionMaxPixels)*3 {
		return fmt.Errorf("raw RGB frame geometry is outside the fixed bound")
	}
	frame := make([]byte, width*height*3)
	for ordinal := 0; ordinal < frames; ordinal++ {
		if _, err := io.ReadFull(r, frame); err != nil {
			return fmt.Errorf("raw RGB frame %d is short: %w", ordinal, err)
		}
		frameHash := canonicalRGBDigest(width, height, "bt709-primaries/srgb-transfer/gbr-full", frame)
		if _, err := fmt.Fprintf(h, "%d:%s\n", ordinal, frameHash); err != nil {
			return err
		}
	}
	var extra [1]byte
	if n, err := io.ReadFull(r, extra[:]); n != 0 || err != io.EOF {
		return fmt.Errorf("raw RGB output has trailing bytes or unexpected EOF state: n=%d err=%v", n, err)
	}
	return nil
}

func TestRawRGBFrameStreamRejectsPartialAndTrailingOutput(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{name: "complete frames", input: []byte{1, 2, 3, 4, 5, 6}},
		{name: "partial last frame", input: []byte{1, 2, 3, 4, 5}, wantErr: true},
		{name: "unexpected trailing byte", input: []byte{1, 2, 3, 4, 5, 6, 7}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := hashRGBFrameStream(bytes.NewReader(tc.input), 1, 1, 2, sha256.New())
			if (err != nil) != tc.wantErr {
				t.Fatalf("stream result error=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func equalInt64s(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func runPinnedOutput(ctx context.Context, tool, input *os.File, args []string, capBytes int64) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/proc/self/fd/3", args...)
	cmd.Env = []string{"LANG=C", "LC_ALL=C"}
	cmd.ExtraFiles = []*os.File{tool, input}
	var stdout, stderr boundedInspectionBuffer
	stdout.limit, stderr.limit = capBytes, inspectionMaxStderr
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pinned command failed: %w: %s", err, truncateUTF8(string(stderr.Bytes()), 2048))
	}
	if stdout.overflow || stderr.overflow {
		return nil, fmt.Errorf("pinned command output exceeded %d bytes", capBytes)
	}
	return bytes.Clone(stdout.Bytes()), nil
}

func encodeRawMP4(ctx context.Context, tool *os.File, rgb []byte, frames int, dst *os.File, args []string, capBytes int64) error {
	cmd := exec.CommandContext(ctx, "/proc/self/fd/3", args...)
	cmd.Env = []string{"LANG=C", "LC_ALL=C"}
	cmd.ExtraFiles = []*os.File{tool}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	var stderr boundedInspectionBuffer
	stderr.limit = inspectionMaxStderr
	limited := &qualificationLimitedWriter{dst: dst, remaining: capBytes}
	cmd.Stdout, cmd.Stderr = limited, &stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	writeErr := make(chan error, 1)
	go func() {
		defer stdin.Close()
		for i := 0; i < frames; i++ {
			if _, err := stdin.Write(rgb); err != nil {
				writeErr <- err
				return
			}
		}
		writeErr <- nil
	}()
	runErr := cmd.Wait()
	inErr := <-writeErr
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if limited.overflow {
		return fmt.Errorf("fragmented MP4 exceeded hard output cap %d", capBytes)
	}
	if inErr != nil {
		return fmt.Errorf("write raw RGB encoder input: %w", inErr)
	}
	if stderr.overflow {
		return fmt.Errorf("encoder stderr exceeded fixed bound")
	}
	if runErr != nil {
		return fmt.Errorf("pinned encoder failed: %w: %s", runErr, truncateUTF8(string(stderr.Bytes()), 2048))
	}
	return nil
}

type qualificationLimitedWriter struct {
	dst       *os.File
	remaining int64
	overflow  bool
}

func (w *qualificationLimitedWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		w.overflow = true
		return 0, fmt.Errorf("output limit reached")
	}
	n, err := w.dst.Write(p)
	w.remaining -= int64(n)
	return n, err
}

func syntheticRGBFrame(width, height int) []byte {
	b := make([]byte, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := (y*width + x) * 3
			b[i], b[i+1], b[i+2] = byte(x%256), byte(y%256), byte((x+y)%256)
		}
	}
	return b
}

func canonicalRGBDigest(width, height int, profile string, rgb []byte) string {
	h := sha256.New()
	fmt.Fprintf(h, "device-studio-rgb-frame-v1\x00%d\x00%d\x00%s\x00", width, height, profile)
	h.Write(rgb)
	return hex.EncodeToString(h.Sum(nil))
}

func writeQualificationJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func BenchmarkPhotographicComposition1080p(b *testing.B) {
	const width, height = 1920, 1080
	plate := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			plate.SetRGBA(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 80, A: 255})
		}
	}
	app := image.NewRGBA(image.Rect(0, 0, 800, 450))
	for y := 0; y < 450; y++ {
		for x := 0; x < 800; x++ {
			app.SetRGBA(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 180, A: 255})
		}
	}
	mask := image.NewGray(plate.Bounds())
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			mask.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	in := composition.Inputs{Plate: plate, Screen: app, SurfaceSize: image.Pt(800, 450), Quad: [4]composition.Point{{560, 220}, {1360, 220}, {1360, 670}, {560, 670}}, ActiveMask: mask, Samples: 1}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := composition.Compose(context.Background(), in); err != nil {
			b.Fatal(err)
		}
	}
}

var _ io.Writer = (*qualificationLimitedWriter)(nil)
