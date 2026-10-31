// Package ffmpegvideo records the exact paired FFmpeg artifact selected for
// the Device Studio video renderer. It is a separate managed tool from the
// host's general-purpose ffmpeg declaration.
package ffmpegvideo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/binaryfetch"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// TreeSHA256 is the reviewed complete managed-tree digest for the pinned
// Linux amd64 renderer payload. It is distinct from the archive digest.
const TreeSHA256 = "1abdef35cd8c90a168f574ec630e14708b18b304111dc4c7a03bdfb16c7ae4c2"

// VerifyTree confirms a staged payload matches its declared whole-tree
// identity using the shared executable-tree format.
func VerifyTree(ctx context.Context, root, expected string) error {
	if ctx == nil {
		return errors.New("ffmpeg-video tree context is nil")
	}
	expected = strings.ToLower(strings.TrimSpace(expected))
	if len(expected) != 64 {
		return errors.New("ffmpeg-video expected tree digest is invalid")
	}
	actual, err := binaryfetch.TreeDigestContext(ctx, root)
	if err != nil {
		return fmt.Errorf("verify ffmpeg-video tree: %w", err)
	}
	if actual != expected {
		return errors.New("ffmpeg-video tree digest does not match the bundle declaration")
	}
	return nil
}

// ArtifactIdentity is the stable identity used by the host and bundle
// projections for this pinned tool profile.
func ArtifactIdentity(artifact Artifact) string { return artifactIdentity(artifact) }

// ResolveBundleProfile verifies and probes a fixed bundle tree. The caller
// has already matched the manifest's profile ID/platform/tree declaration.
func ResolveBundleProfile(ctx context.Context, root string, declared bool) (*cliv1.RendererToolProfile, error) {
	if ctx == nil {
		return nil, errors.New("ffmpeg-video profile context is nil")
	}
	artifact := PinnedArtifact()
	response := bundleProfileBase(artifact)
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		response.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_UNSUPPORTED_PLATFORM
		response.Detail = "the inspected paired archive currently supports Linux amd64 only"
		return response, nil
	}
	if !declared {
		if _, err := os.Lstat(root); err == nil {
			response.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_UNMANAGED
			response.Detail = "a renderer payload exists but is not declared by this bundle; refusing to inspect or execute it"
		} else if errors.Is(err, os.ErrNotExist) {
			response.Detail = "the bundle does not include the optional ffmpeg-video payload"
		} else {
			return nil, fmt.Errorf("inspect undeclared ffmpeg-video path: %w", err)
		}
		return response, nil
	}
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		response.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_PARTIAL
		response.Detail = "the bundle declares ffmpeg-video but its payload is missing"
		return response, nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		response.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_TAMPERED
		response.Detail = "the bundled ffmpeg-video root is not a regular directory"
		return response, nil
	}
	if err := VerifyTree(ctx, root, TreeSHA256); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		response.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_TAMPERED
		response.Detail = "the bundled ffmpeg-video tree differs from the pinned archive content"
		return response, nil
	}
	files := []struct {
		member     string
		size       int64
		hash       string
		executable bool
	}{
		{artifact.FFmpegPath, artifact.FFmpegSize, artifact.FFmpegSHA256, true},
		{artifact.FFprobePath, artifact.FFprobeSize, artifact.FFprobeSHA256, true},
		{artifact.LicensePath, artifact.LicenseSize, artifact.LicenseSHA256, false},
	}
	paths := make([]string, len(files))
	for index, file := range files {
		paths[index] = filepath.Join(root, filepath.FromSlash(file.member))
		if err := verifyBundleMember(ctx, root, paths[index], file.size, file.hash, file.executable); err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			response.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_TAMPERED
			response.Detail = err.Error()
			return response, nil
		}
	}
	ffmpeg, err := inspectBundleBinary(ctx, paths[0], files[0].member, files[0].hash, files[0].size, artifact.BuildLabel)
	if err == nil {
		var conf, license []byte
		conf, err = bundleProbe(ctx, paths[0], "-buildconf")
		if err == nil {
			ffmpeg.buildConf, ffmpeg.buildHash = string(conf), digest(conf)
			license, err = bundleProbe(ctx, paths[0], "-L")
			if err == nil {
				ffmpeg.license = string(license)
			}
		}
	}
	ffprobe, probeErr := inspectBundleBinary(ctx, paths[1], files[1].member, files[1].hash, files[1].size, artifact.BuildLabel)
	if probeErr == nil {
		var conf []byte
		conf, probeErr = bundleProbe(ctx, paths[1], "-buildconf")
		if probeErr == nil {
			ffprobe.buildConf, ffprobe.buildHash = string(conf), digest(conf)
		}
	}
	if err != nil || probeErr != nil {
		response.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_PROBE_FAILED
		response.Detail = "a pinned binary's fixed version/build probe failed or exceeded its bound"
		return response, nil
	}
	response.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT
	response.EnvironmentIdentity = runtime.GOOS + "-" + runtime.GOARCH + "-runtime-prerequisites-unverified"
	response.Ffmpeg = bundleBinaryProto(ffmpeg)
	response.Ffprobe = bundleBinaryProto(ffprobe)
	response.LicenseDetails = ffmpeg.license
	response.Detail = "bundled tree digest, pinned members, and bounded probes match the inspected artifact; OS prerequisites and render behavior remain unqualified"
	return response, nil
}

type bundleBinary struct {
	path, member, hash, version, buildHash, buildConf, license string
	size                                                       int64
}

func bundleProfileBase(artifact Artifact) *cliv1.RendererToolProfile {
	return &cliv1.RendererToolProfile{SchemaVersion: 1, ProfileId: artifact.ProfileID,
		State:            cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_MISSING,
		ArtifactIdentity: artifactIdentity(artifact), Os: runtime.GOOS, Architecture: runtime.GOARCH,
		ArchiveSha256: artifact.ArchiveSHA256, ArchiveSizeBytes: uint64(artifact.ArchiveSize),
		LicenseId: artifact.LicenseID, LicenseSha256: artifact.LicenseSHA256,
		RuntimePrerequisites: append([]string(nil), artifact.Prerequisites...),
		QualificationScope:   "artifact-and-version-probe-only; no video workflow or redistribution qualification",
		ObservedAt:           time.Now().UTC().Format(time.RFC3339Nano)}
}

func artifactIdentity(artifact Artifact) string {
	identity := struct {
		ProfileID     string `json:"profile_id"`
		ArchiveSHA256 string `json:"archive_sha256"`
		ArchiveSize   int64  `json:"archive_size_bytes"`
		FFmpegPath    string `json:"ffmpeg_member"`
		FFmpegSHA256  string `json:"ffmpeg_sha256"`
		FFmpegSize    int64  `json:"ffmpeg_size_bytes"`
		FFprobePath   string `json:"ffprobe_member"`
		FFprobeSHA256 string `json:"ffprobe_sha256"`
		FFprobeSize   int64  `json:"ffprobe_size_bytes"`
		LicenseSHA256 string `json:"license_sha256"`
		LicenseID     string `json:"license_id"`
	}{artifact.ProfileID, artifact.ArchiveSHA256, artifact.ArchiveSize, artifact.FFmpegPath, artifact.FFmpegSHA256, artifact.FFmpegSize, artifact.FFprobePath, artifact.FFprobeSHA256, artifact.FFprobeSize, artifact.LicenseSHA256, artifact.LicenseID}
	data, _ := json.Marshal(identity)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func verifyBundleMember(ctx context.Context, root, path string, size int64, expected string, executable bool) error {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("pinned ffmpeg-video member escapes its root")
	}
	current := root
	var previous os.FileInfo
	for index, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if statErr != nil || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("pinned ffmpeg-video member path is missing or contains a symlink")
		}
		if index < len(strings.Split(rel, string(filepath.Separator)))-1 && !info.IsDir() {
			return errors.New("pinned ffmpeg-video member parent is not a directory")
		}
		previous = info
	}
	if !previous.Mode().IsRegular() || previous.Size() != size || (executable && previous.Mode().Perm()&0o111 == 0) {
		return errors.New("pinned ffmpeg-video member type, mode, or size is unexpected")
	}
	file, err := os.Open(path)
	if err != nil {
		return errors.New("pinned ffmpeg-video member could not be opened")
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(previous, opened) {
		return errors.New("pinned ffmpeg-video member changed while opening")
	}
	hash := sha256.New()
	if _, err := io.CopyN(hash, bundleContextReader{ctx, file}, size+1); err != nil && !errors.Is(err, io.EOF) {
		return errors.New("pinned ffmpeg-video member could not be hashed")
	}
	if hex.EncodeToString(hash.Sum(nil)) != expected {
		return errors.New("pinned ffmpeg-video member digest differs from the inspected artifact")
	}
	return nil
}

type bundleContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r bundleContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func inspectBundleBinary(ctx context.Context, path, member, hash string, size int64, build string) (bundleBinary, error) {
	out, err := bundleProbe(ctx, path, "-version")
	if err != nil {
		return bundleBinary{}, err
	}
	version := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if version == "" || !strings.Contains(version, build) {
		return bundleBinary{}, errors.New("version probe did not report the pinned build label")
	}
	return bundleBinary{path: path, member: member, hash: hash, size: size, version: version}, nil
}

func bundleProbe(ctx context.Context, path string, args ...string) ([]byte, error) {
	if len(args) != 1 || (args[0] != "-version" && args[0] != "-buildconf" && args[0] != "-L") {
		return nil, errors.New("unsupported ffmpeg-video profile probe")
	}
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, path, args...)
	cmd.Env = []string{"LANG=C", "LC_ALL=C"}
	output := &bundleProbeOutput{limit: 32 << 10}
	cmd.Stdout, cmd.Stderr = output, output
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	if output.overflow {
		return nil, errors.New("ffmpeg-video probe output exceeded its bound")
	}
	return bytes.Clone(output.data.Bytes()), nil
}

type bundleProbeOutput struct {
	data     bytes.Buffer
	limit    int
	overflow bool
}

func (w *bundleProbeOutput) Write(p []byte) (int, error) {
	if len(p) > w.limit-w.data.Len() {
		w.overflow = true
		return 0, errors.New("ffmpeg-video probe output exceeded its bound")
	}
	return w.data.Write(p)
}
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func bundleBinaryProto(value bundleBinary) *cliv1.RendererToolBinary {
	return &cliv1.RendererToolBinary{Path: value.path, ArchiveMemberPath: value.member, Sha256: value.hash, SizeBytes: uint64(value.size), Version: value.version, BuildConfigSha256: value.buildHash, BuildConfig: value.buildConf}
}

// Artifact contains content identity only. Install paths and observation time
// are deliberately resolved by the host profile reader and do not affect ID.
type Artifact struct {
	ProfileID     string
	ArchiveURL    string
	ArchiveSHA256 string
	ArchiveSize   int64
	ArchiveRoot   string
	FFmpegPath    string
	FFmpegSHA256  string
	FFmpegSize    int64
	FFprobePath   string
	FFprobeSHA256 string
	FFprobeSize   int64
	LicensePath   string
	LicenseSHA256 string
	LicenseSize   int64
	LicenseID     string
	BuildLabel    string
	Prerequisites []string
}

// PinnedArtifact returns the reviewed Linux amd64 candidate. This identifies
// an inspected candidate only; it does not assert installation, workflow
// qualification, or redistribution approval.
func PinnedArtifact() Artifact {
	return Artifact{
		ProfileID:     "btbn-ffmpeg-video-n9.0.2-28-g669f5ab083-linux64-gpl",
		ArchiveURL:    "https://github.com/BtbN/FFmpeg-Builds/releases/download/autobuild-2026-10-19-13-06/ffmpeg-n9.0.2-28-g669f5ab083-linux64-gpl-9.0.tar.xz",
		ArchiveSHA256: "6668d9fda6278375751f3272f5a6799829b424eedf1934f32d4ee784e646a66e",
		ArchiveSize:   151219048,
		ArchiveRoot:   "ffmpeg-n9.0.2-28-g669f5ab083-linux64-gpl-9.0",
		FFmpegPath:    "ffmpeg-n9.0.2-28-g669f5ab083-linux64-gpl-9.0/bin/ffmpeg",
		FFmpegSHA256:  "dcf865aada2dc5f80fcf57f97c7ab34aa6ea651faaa20375833933b16e9ad372",
		FFmpegSize:    172864808,
		FFprobePath:   "ffmpeg-n9.0.2-28-g669f5ab083-linux64-gpl-9.0/bin/ffprobe",
		FFprobeSHA256: "b517813b5de7c5b837a2a1d8ac052b79e17938137694ef99f2e7c42aa21468ce",
		FFprobeSize:   172631272,
		LicensePath:   "ffmpeg-n9.0.2-28-g669f5ab083-linux64-gpl-9.0/LICENSE.txt",
		LicenseSHA256: "8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903",
		LicenseSize:   35147,
		LicenseID:     "GPLv3 pack notice; binary and component expressions pending review",
		BuildLabel:    "n9.0.2-28-g669f5ab083",
		Prerequisites: []string{
			"Linux x86_64; publisher states glibc >= 2.28 and Linux kernel >= 4.18",
			"This candidate is dynamically linked and requires the host ELF loader and declared system libraries",
		},
	}
}
