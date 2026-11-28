package rendering

import (
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

const (
	InspectionManifestName    = "inspection-manifest.json"
	InspectionProfileName     = "resolved-tool-profile.json"
	InspectionReceiptName     = "inspection-receipt.json"
	InspectionClockName       = "source-clock.v1.bin"
	InspectionFailureName     = "inspection-error.json"
	InspectionSourceName      = "source.media"
	inspectionInputDescriptor = "/proc/self/fd/4"
	inspectionVersion         = 1
	inspectionReceiptVersion  = 2
	inspectionProtocol        = 1
	inspectionMaxManifest     = 64 << 10
	inspectionMaxProfile      = 64 << 10
	inspectionMaxProbe        = 16 << 20
	inspectionMaxStats        = 4 << 20
	inspectionMaxSource       = 512 << 20
	inspectionMaxWorker       = 256 << 20
	inspectionMaxFrames       = 36000
	inspectionMaxDimension    = 3840
	inspectionMaxPixels       = 3840 * 2160
	inspectionMaxSeconds      = 600
	inspectionMaxWallSecs     = 300
	inspectionMaxStderr       = 64 << 10
	inspectionClockHeader     = 12
	inspectionClockRecord     = 12
	inspectionProfileID       = "btbn-ffmpeg-video-n9.0.2-28-g669f5ab083-linux64-gpl"
	inspectionColorPolicy     = "sdr-unmarked-default709-v1"
)

var (
	inspectionDigestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
	inspectionIDPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	inspectionClockMagic    = [8]byte{'D', 'S', 'C', 'L', 'K', '0', '0', '1'}
)

// InspectionManifest is the immutable, path-free input to protocol v1. Local
// executable paths are supplied separately by the host profile owner.
type InspectionManifest struct {
	Version          int              `json:"version"`
	JobID            string           `json:"job_id"`
	AttemptID        string           `json:"attempt_id"`
	InputFingerprint string           `json:"input_fingerprint"`
	WorkerSHA256     string           `json:"worker_sha256"`
	Source           InspectionSource `json:"source"`
	ProfileID        string           `json:"profile_id"`
	ProfileDigest    string           `json:"profile_digest"`
	ProtocolVersion  uint32           `json:"protocol_version"`
	Limits           InspectionLimits `json:"limits"`
}

type InspectionSource struct {
	Name      string `json:"name"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

// Limits is repeated in the manifest for auditability. The worker accepts
// only the protocol's fixed values, so a submitter cannot raise any bound.
type InspectionLimits struct {
	MaxSourceBytes       int64  `json:"max_source_bytes"`
	MaxProbeBytes        int64  `json:"max_probe_bytes"`
	MaxStatsBytes        int64  `json:"max_stats_bytes"`
	MaxFrames            uint32 `json:"max_frames"`
	MaxDimension         uint32 `json:"max_dimension"`
	MaxPixels            uint64 `json:"max_pixels"`
	MaxDurationSeconds   uint32 `json:"max_duration_seconds"`
	MaxWallSeconds       uint32 `json:"max_wall_seconds"`
	MaxMemoryBytes       uint64 `json:"max_memory_bytes"`
	SelectedVideoStreams uint32 `json:"selected_video_streams"`
}

func DefaultInspectionLimits() InspectionLimits {
	return InspectionLimits{
		MaxSourceBytes: inspectionMaxSource, MaxProbeBytes: inspectionMaxProbe, MaxStatsBytes: inspectionMaxStats,
		MaxFrames: inspectionMaxFrames, MaxDimension: inspectionMaxDimension, MaxPixels: inspectionMaxPixels,
		MaxDurationSeconds: inspectionMaxSeconds, MaxWallSeconds: inspectionMaxWallSecs,
		MaxMemoryBytes: 2 << 30, SelectedVideoStreams: 1,
	}
}

type InspectionClassification string

const (
	InspectionAccepted InspectionClassification = "accepted"
	InspectionRejected InspectionClassification = "rejected"
)

// InspectionReceipt records only facts observed and verified by protocol v1.
// Optional facts are pointers so zero and missing remain distinct.
type InspectionReceipt struct {
	Version                int                      `json:"version"`
	JobID                  string                   `json:"job_id"`
	AttemptID              string                   `json:"attempt_id"`
	InputFingerprint       string                   `json:"input_fingerprint"`
	WorkerSHA256           string                   `json:"worker_sha256"`
	ManifestSHA256         string                   `json:"manifest_sha256"`
	SourceSHA256           string                   `json:"source_sha256"`
	SourceSizeBytes        int64                    `json:"source_size_bytes"`
	ProfileID              string                   `json:"profile_id"`
	ProfileDigest          string                   `json:"profile_digest"`
	ProtocolVersion        uint32                   `json:"protocol_version"`
	Classification         InspectionClassification `json:"classification"`
	IssueCode              string                   `json:"issue_code,omitempty"`
	IssueMessage           string                   `json:"issue_message,omitempty"`
	Container              string                   `json:"container,omitempty"`
	StreamCount            uint32                   `json:"stream_count,omitempty"`
	VideoStreamIndex       *uint32                  `json:"video_stream_index,omitempty"`
	Codec                  string                   `json:"codec,omitempty"`
	CodedWidth             uint32                   `json:"coded_width,omitempty"`
	CodedHeight            uint32                   `json:"coded_height,omitempty"`
	DisplayWidth           uint32                   `json:"display_width,omitempty"`
	DisplayHeight          uint32                   `json:"display_height,omitempty"`
	SampleAspectNum        uint32                   `json:"sample_aspect_num,omitempty"`
	SampleAspectDen        uint32                   `json:"sample_aspect_den,omitempty"`
	RotationDegrees        int32                    `json:"rotation_degrees,omitempty"`
	ColorPrimaries         string                   `json:"color_primaries,omitempty"`
	ColorTransfer          string                   `json:"color_transfer,omitempty"`
	ColorMatrix            string                   `json:"color_matrix,omitempty"`
	ColorRange             string                   `json:"color_range,omitempty"`
	ColorPolicy            string                   `json:"color_policy,omitempty"`
	TimeBaseNum            int64                    `json:"time_base_num,omitempty"`
	TimeBaseDen            int64                    `json:"time_base_den,omitempty"`
	StreamStartPTS         *int64                   `json:"stream_start_pts,omitempty"`
	SourceEndPTS           *int64                   `json:"source_end_pts,omitempty"`
	LastFrameDurationTicks int64                    `json:"last_frame_duration_ticks,omitempty"`
	DurationNumerator      int64                    `json:"duration_numerator,omitempty"`
	DurationDenominator    int64                    `json:"duration_denominator,omitempty"`
	FrameCount             uint32                   `json:"frame_count,omitempty"`
	EOFPolicy              string                   `json:"eof_policy,omitempty"`
	ClockArtifactRef       string                   `json:"clock_artifact_ref,omitempty"`
	ClockArtifactSizeBytes int64                    `json:"clock_artifact_size_bytes,omitempty"`
	ClockSHA256            string                   `json:"clock_sha256,omitempty"`
	Audio                  *InspectionAudioFacts    `json:"audio,omitempty"`
}

// InspectionAudioFacts records audio streams observed in the original
// container. An empty, non-nil Streams slice means inspection found none;
// nil Audio is reserved for historical receipt v1, where the fact is unknown.
type InspectionAudioFacts struct {
	Version uint32                      `json:"version"`
	Streams []InspectionAudioStreamFact `json:"streams"`
}

type InspectionAudioStreamFact struct {
	OriginalStreamIndex uint32 `json:"original_stream_index"`
	Codec               string `json:"codec"`
}

type InspectionFailure struct {
	Version   int    `json:"version"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	JobID     string `json:"job_id,omitempty"`
	AttemptID string `json:"attempt_id,omitempty"`
	FailedAt  string `json:"failed_at"`
}

// LocalToolPair is an execution-time resolution from the host-owned profile.
// Paths are local configuration, excluded from manifests, receipts and IDs.
type LocalToolPair struct {
	ProfileID          string
	ArtifactIdentity   string
	Environment        string
	FFmpegPath         string
	FFmpegSHA256       string
	FFmpegSize         int64
	FFmpegVersion      string
	FFmpegBuildSHA256  string
	FFprobePath        string
	FFprobeSHA256      string
	FFprobeSize        int64
	FFprobeVersion     string
	FFprobeBuildSHA256 string
}

func LocalToolPairFromProfile(profile *cliv1.RendererToolProfile) (LocalToolPair, error) {
	if profile == nil || profile.GetSchemaVersion() != 1 || profile.GetState() != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT {
		return LocalToolPair{}, errors.New("ffmpeg-video profile is absent, partial, untrusted, or unsupported")
	}
	if profile.GetOs() != "linux" || profile.GetArchitecture() != "amd64" || runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return LocalToolPair{}, errors.New("ffmpeg-video inspection protocol v1 supports only Linux amd64")
	}
	if profile.GetProfileId() != inspectionProfileID || !inspectionDigestPattern.MatchString(profile.GetArtifactIdentity()) {
		return LocalToolPair{}, errors.New("ffmpeg-video profile identity is invalid")
	}
	ffmpeg, ffprobe := profile.GetFfmpeg(), profile.GetFfprobe()
	if ffmpeg == nil || ffprobe == nil {
		return LocalToolPair{}, errors.New("ffmpeg-video profile does not contain the complete pair")
	}
	if err := validateLocalTool(ffmpeg); err != nil {
		return LocalToolPair{}, fmt.Errorf("validate pinned ffmpeg profile member: %w", err)
	}
	if err := validateLocalTool(ffprobe); err != nil {
		return LocalToolPair{}, fmt.Errorf("validate pinned ffprobe profile member: %w", err)
	}
	return LocalToolPair{
		ProfileID: profile.GetProfileId(), ArtifactIdentity: profile.GetArtifactIdentity(), Environment: profile.GetEnvironmentIdentity(),
		FFmpegPath: ffmpeg.GetPath(), FFmpegSHA256: ffmpeg.GetSha256(), FFmpegSize: int64(ffmpeg.GetSizeBytes()), FFmpegVersion: ffmpeg.GetVersion(), FFmpegBuildSHA256: ffmpeg.GetBuildConfigSha256(),
		FFprobePath: ffprobe.GetPath(), FFprobeSHA256: ffprobe.GetSha256(), FFprobeSize: int64(ffprobe.GetSizeBytes()), FFprobeVersion: ffprobe.GetVersion(), FFprobeBuildSHA256: ffprobe.GetBuildConfigSha256(),
	}, nil
}

func validateLocalTool(tool *cliv1.RendererToolBinary) error {
	if tool == nil || !filepath.IsAbs(tool.GetPath()) || filepath.Clean(tool.GetPath()) != tool.GetPath() || strings.TrimSpace(tool.GetVersion()) == "" || !inspectionDigestPattern.MatchString(tool.GetSha256()) || !inspectionDigestPattern.MatchString(tool.GetBuildConfigSha256()) || tool.GetSizeBytes() == 0 || tool.GetSizeBytes() > uint64(^uint64(0)>>1) {
		return errors.New("profile member path, digest, build identity, or size is invalid")
	}
	if _, err := hex.DecodeString(tool.GetSha256()); err != nil {
		return errors.New("profile member digest is invalid")
	}
	return nil
}
