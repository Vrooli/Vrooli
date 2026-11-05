package rendering

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type inspectionStream struct {
	Index             int               `json:"index"`
	CodecType         string            `json:"codec_type"`
	CodecName         string            `json:"codec_name"`
	Width             int               `json:"width"`
	Height            int               `json:"height"`
	PixelFormat       string            `json:"pix_fmt"`
	SampleAspectRatio string            `json:"sample_aspect_ratio"`
	TimeBase          string            `json:"time_base"`
	StartPTS          json.RawMessage   `json:"start_pts"`
	ColorPrimaries    string            `json:"color_primaries"`
	ColorTransfer     string            `json:"color_transfer"`
	ColorMatrix       string            `json:"color_space"`
	ColorRange        string            `json:"color_range"`
	Tags              map[string]string `json:"tags"`
	SideDataList      []struct {
		Type     string          `json:"side_data_type"`
		Rotation json.RawMessage `json:"rotation"`
	} `json:"side_data_list"`
}

type inspectionFormat struct {
	Name string `json:"format_name"`
}

type inspectionProbeDocument struct {
	Streams []inspectionStream `json:"streams"`
	Format  inspectionFormat   `json:"format"`
}

type inspectionFrame struct {
	PTS           json.RawMessage `json:"pts"`
	DurationTicks json.RawMessage `json:"duration"`
}

type inspectionFrameDocument struct {
	Frames []inspectionFrame `json:"frames"`
}

type inspectionSchedule struct {
	PTS            []int64
	LastFrameTicks int64
}

type inspectionCommandRunner interface {
	Probe(context.Context, *os.File, []string, *os.File, int64) ([]byte, error)
	Decode(context.Context, *os.File, []string, *os.File, int64) ([]byte, error)
}

type execInspectionRunner struct{}

type boundedInspectionBuffer struct {
	data     bytes.Buffer
	limit    int64
	overflow bool
}

func (b *boundedInspectionBuffer) Write(p []byte) (int, error) {
	if int64(len(p)) > b.limit-int64(b.data.Len()) {
		b.overflow = true
		return 0, errors.New("inspection subprocess output exceeded its limit")
	}
	return b.data.Write(p)
}

func (b *boundedInspectionBuffer) Bytes() []byte { return b.data.Bytes() }

func (execInspectionRunner) Probe(ctx context.Context, tool *os.File, args []string, source *os.File, limit int64) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/proc/self/fd/3", args...)
	cmd.Env = []string{"LANG=C", "LC_ALL=C"}
	cmd.ExtraFiles = []*os.File{tool, source}
	var stdout, stderr boundedInspectionBuffer
	stdout.limit, stderr.limit = limit, inspectionMaxStderr
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if stdout.overflow || stderr.overflow {
		return nil, errors.New("inspection subprocess output exceeded its limit")
	}
	if err != nil {
		return nil, fmt.Errorf("pinned probe command failed: %w", err)
	}
	return bytes.Clone(stdout.Bytes()), nil
}

func (execInspectionRunner) Decode(ctx context.Context, tool *os.File, args []string, source *os.File, limit int64) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/proc/self/fd/3", args...)
	cmd.Env = []string{"LANG=C", "LC_ALL=C"}
	cmd.ExtraFiles = []*os.File{tool, source}
	var stdout, stderr boundedInspectionBuffer
	stdout.limit, stderr.limit = limit, inspectionMaxStderr
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if stdout.overflow || stderr.overflow {
		return nil, errors.New("inspection subprocess output exceeded its limit")
	}
	if err != nil {
		return nil, fmt.Errorf("pinned full-decode command failed: %w", err)
	}
	return bytes.Clone(stdout.Bytes()), nil
}

// IsInspectionWorkerInvocation recognizes only the private worker subcommand.
func IsInspectionWorkerInvocation(args []string) bool {
	return len(args) > 0 && args[0] == "inspect-worker"
}

// RunInspectionWorkerCommand is the fixed process entry point. The tool
// profile is staged beside the manifest by the host-profile adapter.
func RunInspectionWorkerCommand(args []string) error {
	if len(args) != 3 || args[0] != "inspect-worker" || args[1] != "--manifest" {
		return errors.New("inspect-worker requires exactly --manifest <absolute-manifest-path>")
	}
	ctx, stop := signalContext()
	defer stop()
	return RunInspectionManifest(ctx, args[2])
}

func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

// RunInspectionManifest probes and fully decodes one pinned source. The
// operation owner, not this worker, owns process-tree cancellation/recovery.
func RunInspectionManifest(ctx context.Context, manifestPath string) (retErr error) {
	return runInspectionManifest(ctx, manifestPath, execInspectionRunner{})
}

func runInspectionManifest(ctx context.Context, manifestPath string, runner inspectionCommandRunner) (retErr error) {
	if ctx == nil || runner == nil {
		return errors.New("inspection worker context and command runner are required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !filepath.IsAbs(manifestPath) || filepath.Clean(manifestPath) != manifestPath || filepath.Base(manifestPath) != InspectionManifestName {
		return errors.New("inspection manifest must be the fixed filename in an absolute attempt workspace")
	}
	workspace := filepath.Dir(manifestPath)
	if err := validateWorkspacePath(workspace); err != nil {
		return err
	}
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return fmt.Errorf("open inspection workspace: %w", err)
	}
	defer root.Close()
	var manifest InspectionManifest
	defer func() {
		if retErr != nil && manifest.JobID != "" && manifest.AttemptID != "" {
			if diagnosticErr := writeInspectionFailure(root, manifest, retErr); diagnosticErr != nil {
				retErr = errors.Join(retErr, fmt.Errorf("write inspection failure receipt: %w", diagnosticErr))
			}
		}
	}()
	manifestBytes, err := readBoundedRegular(ctx, root, InspectionManifestName, inspectionMaxManifest, 0)
	if err != nil {
		return fmt.Errorf("read inspection manifest: %w", err)
	}
	manifestSHA := digest(manifestBytes)
	if err := decodeCanonicalInspectionManifest(manifestBytes, &manifest); err != nil {
		return err
	}
	if err := validateInspectionManifest(manifest); err != nil {
		return err
	}
	if err := verifyCurrentWorker(ctx, manifest.WorkerSHA256); err != nil {
		return fmt.Errorf("verify pinned inspection worker: %w", err)
	}
	for _, name := range []string{InspectionReceiptName, InspectionClockName, InspectionFailureName} {
		if _, err := root.Lstat(name); err == nil {
			return fmt.Errorf("inspection output %q already exists", name)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect reserved output %q: %w", name, err)
		}
	}
	var profile cliv1.RendererToolProfile
	profileBytes, err := readBoundedRegular(ctx, root, InspectionProfileName, inspectionMaxProfile, 0)
	if err != nil {
		return fmt.Errorf("read resolved host tool profile: %w", err)
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(profileBytes, &profile); err != nil {
		return errors.New("resolved host tool profile is not valid renderer-profile JSON")
	}
	pair, err := LocalToolPairFromProfile(&profile)
	if err != nil {
		return err
	}
	if pair.ProfileID != manifest.ProfileID || pair.ArtifactIdentity != manifest.ProfileDigest || profileArtifactIdentity(&profile) != manifest.ProfileDigest {
		return errors.New("resolved host tool profile does not match immutable inspection identity")
	}
	ffmpeg, err := openVerifiedTool(ctx, pair.FFmpegPath, pair.FFmpegSize, pair.FFmpegSHA256)
	if err != nil {
		return fmt.Errorf("verify pinned ffmpeg executable: %w", err)
	}
	defer ffmpeg.Close()
	ffprobe, err := openVerifiedTool(ctx, pair.FFprobePath, pair.FFprobeSize, pair.FFprobeSHA256)
	if err != nil {
		return fmt.Errorf("verify pinned ffprobe executable: %w", err)
	}
	defer ffprobe.Close()
	source, err := openInspectionSource(ctx, root, manifest.Source)
	if err != nil {
		return fmt.Errorf("verify inspection source: %w", err)
	}
	defer source.Close()
	deadlineCtx, cancel := context.WithTimeout(ctx, time.Duration(inspectionMaxWallSecs)*time.Second)
	defer cancel()
	base := InspectionReceipt{
		Version: inspectionReceiptVersion, JobID: manifest.JobID, AttemptID: manifest.AttemptID, InputFingerprint: manifest.InputFingerprint,
		WorkerSHA256:   manifest.WorkerSHA256,
		ManifestSHA256: manifestSHA, SourceSHA256: manifest.Source.SHA256, SourceSizeBytes: manifest.Source.SizeBytes,
		ProfileID: manifest.ProfileID, ProfileDigest: manifest.ProfileDigest, ProtocolVersion: manifest.ProtocolVersion,
	}
	meta, rejection, err := probeInspectionSource(deadlineCtx, runner, ffprobe, source)
	if err != nil {
		return err
	}
	if rejection != nil {
		return publishInspectionRejection(deadlineCtx, root, source, manifest.Source, base, rejection)
	}
	audioFacts, rejection := inspectionAudioFacts(meta)
	if rejection != nil {
		return publishInspectionRejection(deadlineCtx, root, source, manifest.Source, base, rejection)
	}
	schedule, rejection, err := probeInspectionFrames(deadlineCtx, runner, ffprobe, source)
	if err != nil {
		return err
	}
	if rejection != nil {
		return publishInspectionRejection(deadlineCtx, root, source, manifest.Source, base, rejection)
	}
	clock, accepted, rejection, err := verifyInspectionDecode(deadlineCtx, root, runner, ffmpeg, source, manifest, meta, schedule)
	if err != nil {
		return err
	}
	if rejection != nil {
		return publishInspectionRejection(deadlineCtx, root, source, manifest.Source, base, rejection)
	}
	accepted.Version, accepted.JobID, accepted.AttemptID = base.Version, base.JobID, base.AttemptID
	accepted.InputFingerprint, accepted.WorkerSHA256, accepted.ManifestSHA256 = base.InputFingerprint, base.WorkerSHA256, base.ManifestSHA256
	accepted.SourceSHA256, accepted.SourceSizeBytes = base.SourceSHA256, base.SourceSizeBytes
	accepted.ProfileID, accepted.ProfileDigest, accepted.ProtocolVersion = base.ProfileID, base.ProfileDigest, base.ProtocolVersion
	accepted.Classification = InspectionAccepted
	accepted.Audio = audioFacts
	accepted.ClockArtifactRef, accepted.ClockArtifactSizeBytes, accepted.ClockSHA256 = InspectionClockName, int64(len(clock)), digest(clock)
	return publishInspectionSuccess(deadlineCtx, root, accepted, clock)
}

func publishInspectionRejection(ctx context.Context, root *os.Root, source *os.File, binding InspectionSource, base InspectionReceipt, rejection *inspectionRejection) error {
	if err := verifyInspectionSourceUnchanged(ctx, source, binding); err != nil {
		return fmt.Errorf("source changed during rejected inspection: %w", err)
	}
	return publishInspectionReceipt(ctx, root, base, rejection.Code, rejection.Message, nil)
}

type inspectionRejection struct{ Code, Message string }

func decodeCanonicalInspectionManifest(data []byte, out *InspectionManifest) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode inspection manifest: %w", err)
	}
	if decoder.Decode(new(any)) != io.EOF {
		return errors.New("inspection manifest has trailing JSON")
	}
	canonical, err := json.Marshal(*out)
	if err != nil || !bytes.Equal(data, canonical) && !bytes.Equal(data, append(canonical, '\n')) {
		return errors.New("inspection manifest is not canonical JSON")
	}
	return nil
}

func validateInspectionManifest(m InspectionManifest) error {
	if m.Version != inspectionVersion || m.ProtocolVersion != inspectionProtocol || !inspectionIDPattern.MatchString(m.JobID) || !inspectionIDPattern.MatchString(m.AttemptID) || !inspectionDigestPattern.MatchString(m.InputFingerprint) || !inspectionDigestPattern.MatchString(m.WorkerSHA256) || m.ProfileID != inspectionProfileID || !inspectionDigestPattern.MatchString(m.ProfileDigest) {
		return errors.New("inspection manifest identity or protocol version is invalid")
	}
	if m.Source.Name != InspectionSourceName || !inspectionDigestPattern.MatchString(m.Source.SHA256) || m.Source.SizeBytes <= 0 || m.Source.SizeBytes > inspectionMaxSource {
		return errors.New("inspection source binding is invalid or exceeds its byte limit")
	}
	if m.Limits != DefaultInspectionLimits() {
		return errors.New("inspection manifest attempts to change fixed protocol resource limits")
	}
	return nil
}

func openInspectionSource(ctx context.Context, root *os.Root, binding InspectionSource) (*os.File, error) {
	info, err := root.Lstat(binding.Name)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() != binding.SizeBytes || info.Size() > inspectionMaxSource {
		return nil, errors.New("source is not the declared regular file or its size differs")
	}
	f, err := root.Open(binding.Name)
	if err != nil {
		return nil, err
	}
	opened, err := f.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		f.Close()
		return nil, errors.New("source changed while opening")
	}
	h := sha256.New()
	n, err := io.Copy(h, contextReader{ctx: ctx, src: io.LimitReader(f, inspectionMaxSource+1)})
	if err != nil || n != binding.SizeBytes || hex.EncodeToString(h.Sum(nil)) != binding.SHA256 {
		f.Close()
		return nil, errors.New("source bytes differ from immutable digest or size")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return nil, errors.New("source cannot be rewound for native inspection")
	}
	return f, nil
}

func verifyCurrentWorker(ctx context.Context, expected string) error {
	if !inspectionDigestPattern.MatchString(expected) {
		return errors.New("worker digest is missing or invalid")
	}
	// P4b is Linux-only. /proc/self/exe binds the bytes mapped by this process,
	// even if the staged pathname is replaced after the native start.
	file, err := os.Open("/proc/self/exe")
	if err != nil {
		return errors.New("running worker executable cannot be opened by identity")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > inspectionMaxWorker {
		return errors.New("running worker executable size or type is outside protocol limits")
	}
	h := sha256.New()
	n, err := io.Copy(h, contextReader{ctx: ctx, src: io.LimitReader(file, inspectionMaxWorker+1)})
	if err != nil || n != info.Size() || hex.EncodeToString(h.Sum(nil)) != expected {
		return errors.New("running worker bytes differ from immutable job identity")
	}
	return nil
}

func openVerifiedTool(ctx context.Context, path string, size int64, expected string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() != size || info.Mode().Perm()&0o111 == 0 {
		return nil, errors.New("pinned renderer executable is missing, replaced, or not a regular executable")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("pinned renderer executable cannot be opened")
	}
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		f.Close()
		return nil, errors.New("pinned renderer executable changed while opening")
	}
	h := sha256.New()
	if _, err := io.Copy(h, contextReader{ctx: ctx, src: f}); err != nil || hex.EncodeToString(h.Sum(nil)) != expected {
		f.Close()
		return nil, errors.New("pinned renderer executable digest differs from profile")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return nil, errors.New("pinned renderer executable cannot be rewound")
	}
	return f, nil
}

func profileArtifactIdentity(profile *cliv1.RendererToolProfile) string {
	type identity struct {
		ProfileID   string `json:"profile_id"`
		ArchiveSHA  string `json:"archive_sha256"`
		ArchiveSize int64  `json:"archive_size_bytes"`
		FFmpegPath  string `json:"ffmpeg_member"`
		FFmpegSHA   string `json:"ffmpeg_sha256"`
		FFmpegSize  int64  `json:"ffmpeg_size_bytes"`
		FFprobePath string `json:"ffprobe_member"`
		FFprobeSHA  string `json:"ffprobe_sha256"`
		FFprobeSize int64  `json:"ffprobe_size_bytes"`
		LicenseSHA  string `json:"license_sha256"`
		LicenseID   string `json:"license_id"`
	}
	ffmpeg, ffprobe := profile.GetFfmpeg(), profile.GetFfprobe()
	if ffmpeg == nil || ffprobe == nil || profile.GetArchiveSizeBytes() > uint64(^uint64(0)>>1) || ffmpeg.GetSizeBytes() > uint64(^uint64(0)>>1) || ffprobe.GetSizeBytes() > uint64(^uint64(0)>>1) {
		return ""
	}
	b, err := json.Marshal(identity{profile.GetProfileId(), profile.GetArchiveSha256(), int64(profile.GetArchiveSizeBytes()), ffmpeg.GetArchiveMemberPath(), ffmpeg.GetSha256(), int64(ffmpeg.GetSizeBytes()), ffprobe.GetArchiveMemberPath(), ffprobe.GetSha256(), int64(ffprobe.GetSizeBytes()), profile.GetLicenseSha256(), profile.GetLicenseId()})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func publishInspectionSuccess(ctx context.Context, root *os.Root, receipt InspectionReceipt, clock []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := publishExclusive(ctx, root, InspectionClockName, clock, int64(inspectionClockHeader+inspectionMaxFrames*inspectionClockRecord)); err != nil {
		return fmt.Errorf("publish source clock: %w", err)
	}
	return publishInspectionReceipt(ctx, root, receipt, "", "", clock)
}

func publishInspectionReceipt(ctx context.Context, root *os.Root, base InspectionReceipt, issueCode, issueMessage string, clock []byte) error {
	if issueCode != "" {
		base.Classification, base.IssueCode, base.IssueMessage = InspectionRejected, issueCode, issueMessage
	} else {
		base.Classification = InspectionAccepted
		base.ClockArtifactRef, base.ClockArtifactSizeBytes, base.ClockSHA256 = InspectionClockName, int64(len(clock)), digest(clock)
	}
	b, err := json.Marshal(base)
	if err != nil {
		return err
	}
	if int64(len(b)) > maxReceiptBytes {
		return errors.New("inspection receipt exceeds its fixed size limit")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := publishExclusive(ctx, root, InspectionReceiptName, b, maxReceiptBytes); err != nil {
		return fmt.Errorf("publish inspection receipt: %w", err)
	}
	return nil
}

func decodeInspectionClock(data []byte, expectedCount uint32) ([]int64, error) {
	expectedSize := inspectionClockHeader + int64(expectedCount)*inspectionClockRecord
	if expectedCount == 0 || expectedCount > inspectionMaxFrames || int64(len(data)) != expectedSize || !bytes.Equal(data[:8], inspectionClockMagic[:]) || binary.BigEndian.Uint32(data[8:12]) != expectedCount {
		return nil, errors.New("source clock header or byte size is invalid")
	}
	pts := make([]int64, expectedCount)
	for i := range pts {
		offset := inspectionClockHeader + i*inspectionClockRecord
		if binary.BigEndian.Uint32(data[offset:offset+4]) != uint32(i) {
			return nil, errors.New("source clock ordinal sequence is invalid")
		}
		pts[i] = int64(binary.BigEndian.Uint64(data[offset+4 : offset+12]))
		if i > 0 && pts[i] <= pts[i-1] {
			return nil, errors.New("source clock PTS must be strictly increasing in v1")
		}
	}
	return pts, nil
}

func buildInspectionClock(pts []int64) ([]byte, error) {
	if len(pts) == 0 || len(pts) > inspectionMaxFrames {
		return nil, errors.New("source clock frame count is outside v1 limits")
	}
	data := make([]byte, inspectionClockHeader+len(pts)*inspectionClockRecord)
	copy(data[:8], inspectionClockMagic[:])
	binary.BigEndian.PutUint32(data[8:12], uint32(len(pts)))
	for i, value := range pts {
		if i > 0 && value <= pts[i-1] {
			return nil, errors.New("source clock PTS must be strictly increasing in v1")
		}
		offset := inspectionClockHeader + i*inspectionClockRecord
		binary.BigEndian.PutUint32(data[offset:offset+4], uint32(i))
		binary.BigEndian.PutUint64(data[offset+4:offset+12], uint64(value))
	}
	return data, nil
}

func writeInspectionFailure(root *os.Root, manifest InspectionManifest, cause error) error {
	code := "inspection_execution_failed"
	message := strings.ReplaceAll(cause.Error(), root.Name(), "<attempt-workspace>")
	message = truncateUTF8(strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' {
			return -1
		}
		return r
	}, message), 768)
	b, err := json.Marshal(InspectionFailure{Version: inspectionVersion, Code: code, Message: message, JobID: manifest.JobID, AttemptID: manifest.AttemptID, FailedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	if err != nil {
		return err
	}
	return publishExclusive(context.Background(), root, InspectionFailureName, b, maxDiagnosticBytes)
}

func probeInspectionSource(ctx context.Context, runner inspectionCommandRunner, tool, source *os.File) (inspectionProbeDocument, *inspectionRejection, error) {
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return inspectionProbeDocument{}, nil, err
	}
	args := []string{"-v", "error", "-protocol_whitelist", "file,pipe", "-format_whitelist", "mov,matroska,webm", "-max_streams", "8", "-show_streams", "-show_format", "-of", "json", inspectionInputDescriptor}
	data, err := runner.Probe(ctx, tool, args, source, inspectionMaxProbe)
	if err != nil {
		return inspectionProbeDocument{}, nil, fmt.Errorf("inspect source metadata: %w", err)
	}
	var document inspectionProbeDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return document, nil, fmt.Errorf("decode bounded source metadata: %w", err)
	}
	if len(document.Streams) == 0 || len(document.Streams) > 8 {
		return document, &inspectionRejection{"stream_count_unsupported", "source must contain between one and eight streams"}, nil
	}
	streamIndexes := make(map[int]struct{}, len(document.Streams))
	for _, stream := range document.Streams {
		if stream.Index < 0 || stream.Index >= len(document.Streams) {
			return document, &inspectionRejection{"stream_index_invalid", "container stream indexes are inconsistent with the bounded stream list"}, nil
		}
		if _, exists := streamIndexes[stream.Index]; exists {
			return document, &inspectionRejection{"stream_index_ambiguous", "container contains duplicate stream indexes"}, nil
		}
		streamIndexes[stream.Index] = struct{}{}
	}
	var videos []inspectionStream
	for _, stream := range document.Streams {
		if stream.CodecType == "video" {
			videos = append(videos, stream)
		}
	}
	if len(videos) != 1 {
		return document, &inspectionRejection{"video_stream_selection_ambiguous", "protocol v1 requires exactly one video stream"}, nil
	}
	stream := videos[0]
	if stream.CodecName == "" || stream.Width <= 0 || stream.Height <= 0 || stream.Width > inspectionMaxDimension || stream.Height > inspectionMaxDimension || int64(stream.Width)*int64(stream.Height) > inspectionMaxPixels {
		return document, &inspectionRejection{"video_raster_unsupported", "video stream dimensions or codec metadata are outside protocol v1 limits"}, nil
	}
	if stream.SampleAspectRatio != "1:1" {
		return document, &inspectionRejection{"sample_aspect_ratio_unsupported", "protocol v1 accepts only square-pixel video"}, nil
	}
	if inspectionRotation(stream) != 0 {
		return document, &inspectionRejection{"rotation_unsupported", "protocol v1 accepts only sources with zero display rotation"}, nil
	}
	if !supportedSDR(stream) {
		return document, &inspectionRejection{"color_profile_unsupported", "protocol v1 accepts only its pinned SDR matrix, range, and transfer policy"}, nil
	}
	if !supportedContainer(document.Format.Name) {
		return document, &inspectionRejection{"container_unsupported", "container is not accepted by the qualified inspection protocol"}, nil
	}
	if _, _, ok := parseRational(stream.TimeBase); !ok {
		return document, &inspectionRejection{"time_base_unavailable", "video stream has no valid exact rational time base"}, nil
	}
	if _, ok := parseRawInt(stream.StartPTS); !ok {
		return document, &inspectionRejection{"stream_start_unavailable", "video stream has no exact integer start PTS"}, nil
	}
	return document, nil, nil
}

func inspectionAudioFacts(document inspectionProbeDocument) (*InspectionAudioFacts, *inspectionRejection) {
	facts := &InspectionAudioFacts{Version: 1, Streams: make([]InspectionAudioStreamFact, 0)}
	for _, stream := range document.Streams {
		if stream.CodecType != "audio" {
			continue
		}
		if stream.Index < 0 || stream.Index >= len(document.Streams) || !validAudioCodec(stream.CodecName) {
			return nil, &inspectionRejection{"audio_stream_unsupported", "audio stream index or codec metadata is outside receipt v2 limits"}
		}
		facts.Streams = append(facts.Streams, InspectionAudioStreamFact{OriginalStreamIndex: uint32(stream.Index), Codec: stream.CodecName})
	}
	sort.Slice(facts.Streams, func(i, j int) bool {
		return facts.Streams[i].OriginalStreamIndex < facts.Streams[j].OriginalStreamIndex
	})
	return facts, nil
}

func validAudioCodec(codec string) bool {
	if len(codec) == 0 || len(codec) > 64 {
		return false
	}
	for i, c := range codec {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || (i > 0 && (c == '_' || c == '-' || c == '.'))) {
			return false
		}
	}
	return true
}

func probeInspectionFrames(ctx context.Context, runner inspectionCommandRunner, tool, source *os.File) (inspectionSchedule, *inspectionRejection, error) {
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return inspectionSchedule{}, nil, err
	}
	args := []string{"-v", "error", "-protocol_whitelist", "file,pipe", "-format_whitelist", "mov,matroska,webm", "-max_streams", "8", "-select_streams", "v:0", "-show_frames", "-show_entries", "frame=pts,duration", "-of", "json", inspectionInputDescriptor}
	data, err := runner.Probe(ctx, tool, args, source, inspectionMaxProbe)
	if err != nil {
		return inspectionSchedule{}, nil, fmt.Errorf("inspect source frame timestamps: %w", err)
	}
	var document inspectionFrameDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return inspectionSchedule{}, nil, fmt.Errorf("decode bounded source timestamp schedule: %w", err)
	}
	if len(document.Frames) == 0 || len(document.Frames) > inspectionMaxFrames {
		return inspectionSchedule{}, &inspectionRejection{"frame_count_unsupported", "source frame count is empty or exceeds the fixed protocol limit"}, nil
	}
	schedule := inspectionSchedule{PTS: make([]int64, len(document.Frames))}
	for i, frame := range document.Frames {
		value, ok := parseRawInt(frame.PTS)
		if !ok {
			return inspectionSchedule{}, &inspectionRejection{"frame_pts_unavailable", "one or more decoded frames lack exact integer PTS"}, nil
		}
		if i > 0 && value <= schedule.PTS[i-1] {
			return inspectionSchedule{}, &inspectionRejection{"frame_pts_ambiguous", "protocol v1 requires strictly increasing source PTS"}, nil
		}
		schedule.PTS[i] = value
	}
	lastDuration, ok := parseRawInt(document.Frames[len(document.Frames)-1].DurationTicks)
	if !ok || lastDuration <= 0 || schedule.PTS[len(schedule.PTS)-1] > math.MaxInt64-lastDuration {
		return inspectionSchedule{}, &inspectionRejection{"source_eof_unavailable", "last frame has no valid exact duration for an exclusive EOF PTS"}, nil
	}
	schedule.LastFrameTicks = lastDuration
	return schedule, nil, nil
}

func verifyInspectionDecode(ctx context.Context, root *os.Root, runner inspectionCommandRunner, tool, source *os.File, manifest InspectionManifest, metadata inspectionProbeDocument, schedule inspectionSchedule) ([]byte, InspectionReceipt, *inspectionRejection, error) {
	var video inspectionStream
	for _, candidate := range metadata.Streams {
		if candidate.CodecType == "video" {
			video = candidate
			break
		}
	}
	timeNum, timeDen, ok := parseRational(video.TimeBase)
	if !ok {
		return nil, InspectionReceipt{}, nil, errors.New("validated stream time base disappeared")
	}
	start, ok := parseRawInt(video.StartPTS)
	if !ok {
		return nil, InspectionReceipt{}, nil, errors.New("validated stream start PTS disappeared")
	}
	end := schedule.PTS[len(schedule.PTS)-1] + schedule.LastFrameTicks
	durationNum, durationDen, ok := exactDuration(timeNum, timeDen, schedule.PTS[0], end)
	if !ok {
		return nil, InspectionReceipt{}, &inspectionRejection{"duration_unsupported", "exact decoded duration exceeds protocol v1 bounds"}, nil
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return nil, InspectionReceipt{}, nil, err
	}
	args := []string{"-hide_banner", "-nostdin", "-v", "error", "-protocol_whitelist", "file,pipe", "-format_whitelist", "mov,matroska,webm", "-max_streams", "8", "-xerror", "-err_detect", "explode", "-copyts", "-i", inspectionInputDescriptor, "-map", fmt.Sprintf("0:%d", video.Index), "-an", "-sn", "-dn", "-fps_mode", "passthrough", "-enc_time_base:v", fmt.Sprintf("%d/%d", timeNum, timeDen), "-stats_enc_pre", "pipe:1", "-stats_enc_pre_fmt", "{n} {pts} {tb}", "-c:v", "rawvideo", "-pix_fmt", "rgb24", "-f", "null", "-"}
	stats, err := runner.Decode(ctx, tool, args, source, inspectionMaxStats)
	if err != nil {
		return nil, InspectionReceipt{}, nil, fmt.Errorf("full source decode failed: %w", err)
	}
	if err := compareDecodePTS(stats, schedule.PTS, timeNum, timeDen); err != nil {
		return nil, InspectionReceipt{}, &inspectionRejection{"decode_timestamp_mismatch", "decoder frame order or exact PTS differs from the pinned source schedule"}, nil
	}
	clock, err := buildInspectionClock(schedule.PTS)
	if err != nil {
		return nil, InspectionReceipt{}, nil, err
	}
	if err := verifyInspectionSourceUnchanged(ctx, source, manifest.Source); err != nil {
		return nil, InspectionReceipt{}, nil, fmt.Errorf("source changed during native inspection: %w", err)
	}
	index := uint32(video.Index)
	receipt := InspectionReceipt{
		Classification: InspectionAccepted, Container: metadata.Format.Name, StreamCount: uint32(len(metadata.Streams)), VideoStreamIndex: &index,
		Codec: video.CodecName, CodedWidth: uint32(video.Width), CodedHeight: uint32(video.Height), DisplayWidth: uint32(video.Width), DisplayHeight: uint32(video.Height),
		SampleAspectNum: 1, SampleAspectDen: 1, RotationDegrees: 0,
		ColorPrimaries: video.ColorPrimaries, ColorTransfer: video.ColorTransfer, ColorMatrix: video.ColorMatrix, ColorRange: video.ColorRange, ColorPolicy: inspectionColorPolicy,
		TimeBaseNum: timeNum, TimeBaseDen: timeDen, StreamStartPTS: &start, SourceEndPTS: &end,
		LastFrameDurationTicks: schedule.LastFrameTicks,
		DurationNumerator:      durationNum, DurationDenominator: durationDen, FrameCount: uint32(len(schedule.PTS)), EOFPolicy: "last_frame_duration",
	}
	return clock, receipt, nil, nil
}

func compareDecodePTS(data []byte, expected []int64, timeNum, timeDen int64) error {
	lines := bytes.Split(bytes.TrimSpace(data), []byte{'\n'})
	if len(lines) != len(expected) {
		return errors.New("decoder timestamp count differs")
	}
	timeBase := fmt.Sprintf("%d/%d", timeNum, timeDen)
	for i, line := range lines {
		parts := strings.Fields(string(line))
		if len(parts) != 3 {
			return errors.New("decoder timestamp row is malformed")
		}
		ordinal, err1 := strconv.ParseUint(parts[0], 10, 32)
		pts, err2 := strconv.ParseInt(parts[1], 10, 64)
		if err1 != nil || err2 != nil || ordinal != uint64(i) || pts != expected[i] || parts[2] != timeBase {
			return errors.New("decoder timestamp differs from source schedule")
		}
	}
	return nil
}

func parseRawInt(raw json.RawMessage) (int64, bool) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return 0, false
	}
	value := string(raw)
	if raw[0] == '"' {
		if json.Unmarshal(raw, &value) != nil {
			return 0, false
		}
	}
	n, err := strconv.ParseInt(value, 10, 64)
	return n, err == nil
}

func parseRational(value string) (int64, int64, bool) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return 0, 0, false
	}
	num, err1 := strconv.ParseInt(parts[0], 10, 64)
	den, err2 := strconv.ParseInt(parts[1], 10, 64)
	return num, den, err1 == nil && err2 == nil && num > 0 && den > 0
}

func inspectionRotation(stream inspectionStream) int64 {
	if stream.Tags != nil && stream.Tags["rotate"] != "" {
		rotation, err := strconv.ParseInt(stream.Tags["rotate"], 10, 32)
		if err != nil {
			return math.MaxInt64
		}
		if rotation != 0 {
			return rotation
		}
	}
	for _, side := range stream.SideDataList {
		if len(side.Rotation) == 0 {
			return math.MaxInt64
		}
		rotation, ok := parseRawInt(side.Rotation)
		if !ok {
			return math.MaxInt64
		}
		if rotation != 0 {
			return rotation
		}
	}
	return 0
}

func supportedSDR(stream inspectionStream) bool {
	return allowedSDRTag(stream.ColorPrimaries, "bt709", "bt470bg", "smpte170m") &&
		allowedSDRTag(stream.ColorTransfer, "bt709", "bt470bg", "smpte170m") &&
		(stream.ColorMatrix == "bt709" || stream.ColorMatrix == "bt470bg" || stream.ColorMatrix == "smpte170m") &&
		(stream.ColorRange == "tv" || stream.ColorRange == "mpeg" || stream.ColorRange == "pc" || stream.ColorRange == "jpeg")
}

func allowedSDRTag(value string, accepted ...string) bool {
	if value == "" {
		return true
	}
	for _, candidate := range accepted {
		if value == candidate {
			return true
		}
	}
	return false
}

func verifyInspectionSourceUnchanged(ctx context.Context, source *os.File, binding InspectionSource) error {
	info, err := source.Stat()
	if err != nil || info.Size() != binding.SizeBytes || !info.Mode().IsRegular() {
		return errors.New("source file facts changed during inspection")
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return err
	}
	h := sha256.New()
	n, err := io.Copy(h, contextReader{ctx: ctx, src: io.LimitReader(source, inspectionMaxSource+1)})
	if err != nil || n != binding.SizeBytes || hex.EncodeToString(h.Sum(nil)) != binding.SHA256 {
		return errors.New("source bytes changed during inspection")
	}
	return nil
}

func supportedContainer(value string) bool {
	for _, name := range strings.Split(value, ",") {
		if name == "mov" || name == "matroska" || name == "webm" {
			return true
		}
	}
	return false
}

func exactDuration(timeNum, timeDen, start, end int64) (int64, int64, bool) {
	if timeNum <= 0 || timeDen <= 0 || end <= start {
		return 0, 0, false
	}
	delta := new(big.Int).Sub(big.NewInt(end), big.NewInt(start))
	r := new(big.Rat).SetFrac(new(big.Int).Mul(delta, big.NewInt(timeNum)), big.NewInt(timeDen))
	if r.Sign() <= 0 || r.Cmp(big.NewRat(inspectionMaxSeconds, 1)) > 0 || !r.Num().IsInt64() || !r.Denom().IsInt64() {
		return 0, 0, false
	}
	return r.Num().Int64(), r.Denom().Int64(), true
}

func VerifyInspectionArtifacts(manifestBytes, receiptBytes, clockBytes []byte) (InspectionReceipt, error) {
	var receipt InspectionReceipt
	var manifest InspectionManifest
	if len(manifestBytes) == 0 || len(manifestBytes) > inspectionMaxManifest || len(clockBytes) > inspectionClockHeader+inspectionMaxFrames*inspectionClockRecord {
		return receipt, errors.New("inspection manifest or clock exceeds its byte limit")
	}
	if err := decodeCanonicalInspectionManifest(manifestBytes, &manifest); err != nil {
		return receipt, err
	}
	if err := validateInspectionManifest(manifest); err != nil {
		return receipt, err
	}
	if len(receiptBytes) == 0 || len(receiptBytes) > maxReceiptBytes {
		return receipt, errors.New("inspection receipt size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(receiptBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return receipt, err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return receipt, errors.New("inspection receipt has trailing JSON")
	}
	canonical, err := json.Marshal(receipt)
	if err != nil || !bytes.Equal(receiptBytes, canonical) {
		return receipt, errors.New("inspection receipt is not canonical JSON")
	}
	if (receipt.Version != inspectionVersion && receipt.Version != inspectionReceiptVersion) || receipt.JobID != manifest.JobID || receipt.AttemptID != manifest.AttemptID || receipt.InputFingerprint != manifest.InputFingerprint || receipt.WorkerSHA256 != manifest.WorkerSHA256 || receipt.ManifestSHA256 != digest(manifestBytes) || receipt.SourceSHA256 != manifest.Source.SHA256 || receipt.SourceSizeBytes != manifest.Source.SizeBytes || receipt.ProfileID != manifest.ProfileID || receipt.ProfileDigest != manifest.ProfileDigest || receipt.ProtocolVersion != manifest.ProtocolVersion {
		return receipt, errors.New("inspection receipt identity differs from immutable manifest")
	}
	if receipt.Classification == InspectionRejected {
		if !validInspectionIssueCode(receipt.IssueCode) || receipt.IssueMessage == "" || len(receipt.IssueMessage) > 1024 || len(clockBytes) != 0 || receipt.ClockArtifactRef != "" || receipt.ClockSHA256 != "" || receipt.ClockArtifactSizeBytes != 0 || hasRejectedInspectionFacts(receipt) {
			return receipt, errors.New("rejected inspection receipt contains missing issue or fabricated accepted facts")
		}
		return receipt, nil
	}
	if receipt.Classification != InspectionAccepted || receipt.IssueCode != "" || receipt.IssueMessage != "" || receipt.VideoStreamIndex == nil || receipt.SourceEndPTS == nil || receipt.StreamStartPTS == nil || receipt.LastFrameDurationTicks <= 0 || receipt.EOFPolicy != "last_frame_duration" || receipt.ClockArtifactRef != InspectionClockName || int64(len(clockBytes)) != receipt.ClockArtifactSizeBytes || digest(clockBytes) != receipt.ClockSHA256 || !inspectionDigestPattern.MatchString(receipt.ClockSHA256) {
		return receipt, errors.New("accepted inspection receipt lacks complete identity-bound clock facts")
	}
	pts, err := decodeInspectionClock(clockBytes, receipt.FrameCount)
	if err != nil {
		return receipt, err
	}
	if (receipt.Version == inspectionVersion && receipt.Audio != nil) || (receipt.Version == inspectionReceiptVersion && !validInspectionAudioFacts(receipt.Audio, receipt.StreamCount, *receipt.VideoStreamIndex)) || !validAcceptedInspectionFacts(receipt, pts) {
		return receipt, errors.New("accepted inspection receipt has invalid clock range or rational facts")
	}
	return receipt, nil
}

func validInspectionIssueCode(code string) bool {
	switch code {
	case "stream_count_unsupported", "stream_index_invalid", "stream_index_ambiguous", "video_stream_selection_ambiguous", "video_raster_unsupported", "sample_aspect_ratio_unsupported", "rotation_unsupported", "color_profile_unsupported", "container_unsupported", "time_base_unavailable", "stream_start_unavailable", "frame_count_unsupported", "frame_pts_unavailable", "frame_pts_ambiguous", "source_eof_unavailable", "duration_unsupported", "decode_timestamp_mismatch", "audio_stream_unsupported":
		return true
	default:
		return false
	}
}

func hasRejectedInspectionFacts(r InspectionReceipt) bool {
	return r.Audio != nil || r.Container != "" || r.StreamCount != 0 || r.VideoStreamIndex != nil || r.Codec != "" || r.CodedWidth != 0 || r.CodedHeight != 0 || r.DisplayWidth != 0 || r.DisplayHeight != 0 || r.SampleAspectNum != 0 || r.SampleAspectDen != 0 || r.RotationDegrees != 0 || r.ColorPrimaries != "" || r.ColorTransfer != "" || r.ColorMatrix != "" || r.ColorRange != "" || r.ColorPolicy != "" || r.TimeBaseNum != 0 || r.TimeBaseDen != 0 || r.StreamStartPTS != nil || r.SourceEndPTS != nil || r.LastFrameDurationTicks != 0 || r.DurationNumerator != 0 || r.DurationDenominator != 0 || r.FrameCount != 0 || r.EOFPolicy != ""
}

func validInspectionAudioFacts(facts *InspectionAudioFacts, streamCount, videoIndex uint32) bool {
	if facts == nil || facts.Version != 1 || facts.Streams == nil || len(facts.Streams) > 7 {
		return false
	}
	seen := make(map[uint32]struct{}, len(facts.Streams))
	for _, stream := range facts.Streams {
		if stream.OriginalStreamIndex >= streamCount || stream.OriginalStreamIndex == videoIndex || !validAudioCodec(stream.Codec) {
			return false
		}
		if _, exists := seen[stream.OriginalStreamIndex]; exists {
			return false
		}
		seen[stream.OriginalStreamIndex] = struct{}{}
	}
	return true
}

func validAcceptedInspectionFacts(r InspectionReceipt, pts []int64) bool {
	if r.StreamCount == 0 || r.StreamCount > 8 || *r.VideoStreamIndex >= r.StreamCount || r.Codec == "" || !supportedContainer(r.Container) {
		return false
	}
	if r.CodedWidth == 0 || r.CodedHeight == 0 || r.CodedWidth > inspectionMaxDimension || r.CodedHeight > inspectionMaxDimension || uint64(r.CodedWidth)*uint64(r.CodedHeight) > inspectionMaxPixels || r.DisplayWidth != r.CodedWidth || r.DisplayHeight != r.CodedHeight {
		return false
	}
	if r.SampleAspectNum != 1 || r.SampleAspectDen != 1 || r.RotationDegrees != 0 || !allowedSDRTag(r.ColorPrimaries, "bt709", "bt470bg", "smpte170m") || !allowedSDRTag(r.ColorTransfer, "bt709", "bt470bg", "smpte170m") || (r.ColorMatrix != "bt709" && r.ColorMatrix != "bt470bg" && r.ColorMatrix != "smpte170m") || (r.ColorRange != "tv" && r.ColorRange != "mpeg" && r.ColorRange != "pc" && r.ColorRange != "jpeg") || r.ColorPolicy != inspectionColorPolicy {
		return false
	}
	if r.FrameCount == 0 || r.FrameCount > inspectionMaxFrames || int64(r.FrameCount) != int64(len(pts)) || r.TimeBaseNum <= 0 || r.TimeBaseDen <= 0 || pts[0] < *r.StreamStartPTS || pts[len(pts)-1] > math.MaxInt64-r.LastFrameDurationTicks || pts[len(pts)-1]+r.LastFrameDurationTicks != *r.SourceEndPTS {
		return false
	}
	num, den, ok := exactDuration(r.TimeBaseNum, r.TimeBaseDen, pts[0], *r.SourceEndPTS)
	return ok && r.DurationNumerator == num && r.DurationDenominator == den
}
