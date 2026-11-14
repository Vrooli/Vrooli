package hostapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/config"
	ffmpegvideo "github.com/vrooli/vrooli/internal/tools/ffmpeg-video"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

const (
	rendererProfileToolName = "ffmpeg-video"
	rendererProbeTimeout    = 5 * time.Second
	rendererProbeOutputMax  = 32 << 10
)

type rendererProbe func(context.Context, string, ...string) ([]byte, error)

type rendererBinaryResult struct {
	Path      string
	Member    string
	SHA256    string
	Size      int64
	Version   string
	BuildHash string
	BuildConf string
	License   string
}

// RendererProfile inspects only the exact managed FFmpeg-video installation.
// It never provisions a tool, consults PATH, or accepts executable paths from
// the caller.
func (Service) RendererProfile(ctx context.Context) (*cliv1.RendererToolProfile, error) {
	if ctx == nil {
		return nil, errors.New("renderer profile context is nil")
	}
	home, err := config.HomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve renderer profile owner home: %w", err)
	}
	root, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return nil, fmt.Errorf("resolve renderer profile runtime home: %w", err)
	}
	optDir := filepath.Join(root, "opt", rendererProfileToolName)
	record, err := cliinstall.LoadInstallRecord(home)
	if err != nil {
		return nil, fmt.Errorf("read managed renderer ownership: %w", err)
	}
	owned := false
	for _, entry := range record.Entries {
		if entry.Scope == cliinstall.ScopeRuntime && entry.Kind == cliinstall.EntryDirectory && filepath.Clean(entry.Path) == filepath.Clean(optDir) {
			owned = true
			break
		}
	}

	artifact := ffmpegvideo.PinnedArtifact()
	return resolveRendererProfileForPlatform(ctx, optDir, owned, artifact, runtime.GOOS, runtime.GOARCH, rendererProfileProbe)
}

func rendererProfileBase(artifact ffmpegvideo.Artifact, goos, arch string) *cliv1.RendererToolProfile {
	return &cliv1.RendererToolProfile{
		SchemaVersion:        1,
		ProfileId:            artifact.ProfileID,
		State:                cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_MISSING,
		ArtifactIdentity:     rendererArtifactIdentity(artifact),
		Os:                   goos,
		Architecture:         arch,
		ArchiveSha256:        artifact.ArchiveSHA256,
		ArchiveSizeBytes:     uint64(artifact.ArchiveSize),
		LicenseId:            artifact.LicenseID,
		LicenseSha256:        artifact.LicenseSHA256,
		RuntimePrerequisites: append([]string(nil), artifact.Prerequisites...),
		QualificationScope:   "artifact-and-version-probe-only; no video workflow or redistribution qualification",
		ObservedAt:           time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func rendererArtifactIdentity(artifact ffmpegvideo.Artifact) string {
	return ffmpegvideo.ArtifactIdentity(artifact)
}

func resolveRendererProfile(ctx context.Context, optDir string, owned bool, artifact ffmpegvideo.Artifact, probe rendererProbe) (*cliv1.RendererToolProfile, error) {
	return resolveRendererProfileForPlatform(ctx, optDir, owned, artifact, "linux", "amd64", probe)
}

func resolveRendererProfileForPlatform(ctx context.Context, optDir string, owned bool, artifact ffmpegvideo.Artifact, goos, arch string, probe rendererProbe) (*cliv1.RendererToolProfile, error) {
	response := rendererProfileBase(artifact, goos, arch)
	if goos != "linux" || arch != "amd64" {
		response.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_UNSUPPORTED_PLATFORM
		response.Detail = "the inspected paired archive currently supports Linux amd64 only"
		return response, nil
	}
	if _, err := os.Lstat(optDir); errors.Is(err, os.ErrNotExist) {
		response.Detail = "the managed ffmpeg-video payload is not installed"
		return response, nil
	} else if err != nil {
		return nil, fmt.Errorf("inspect managed renderer directory: %w", err)
	}
	if !owned {
		response.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_UNMANAGED
		response.Detail = "payload exists outside the recorded managed-tool directory; refusing to inspect or execute it"
		return response, nil
	}
	if err := validateManagedDirectory(optDir); err != nil {
		response.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_TAMPERED
		response.Detail = err.Error()
		return response, nil
	}
	files := []struct {
		path string
		size int64
		hash string
	}{
		{artifact.FFmpegPath, artifact.FFmpegSize, artifact.FFmpegSHA256},
		{artifact.FFprobePath, artifact.FFprobeSize, artifact.FFprobeSHA256},
		{artifact.LicensePath, artifact.LicenseSize, artifact.LicenseSHA256},
	}
	paths := make([]string, len(files))
	present := 0
	for index, file := range files {
		path := filepath.Join(optDir, filepath.FromSlash(file.path))
		paths[index] = path
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			response.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_TAMPERED
			response.Detail = "could not inspect a pinned renderer member"
			return response, nil
		}
		present++
	}
	if present == 0 {
		response.Detail = "the managed renderer payload is not installed"
		return response, nil
	}
	if present != len(files) {
		response.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_PARTIAL
		response.Detail = "one or more pinned renderer members are missing"
		return response, nil
	}
	for index, file := range files {
		if err := verifyRendererMember(ctx, optDir, paths[index], file.size, file.hash, index < 2); err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			response.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_TAMPERED
			response.Detail = err.Error()
			return response, nil
		}
	}
	if probe == nil {
		probe = rendererProfileProbe
	}
	ffmpeg, err := inspectRendererBinary(ctx, probe, paths[0], artifact.FFmpegPath, files[0].hash, files[0].size, artifact.BuildLabel)
	if err == nil {
		var probeConf []byte
		probeConf, err = probe(ctx, paths[0], "-buildconf")
		if err == nil {
			ffmpeg.BuildConf = string(probeConf)
			ffmpeg.BuildHash = digestBytes(probeConf)
			var licenseDetails []byte
			licenseDetails, err = probe(ctx, paths[0], "-L")
			if err == nil {
				ffmpeg.License = string(licenseDetails)
			}
		}
	}
	ffprobe, probeErr := inspectRendererBinary(ctx, probe, paths[1], artifact.FFprobePath, files[1].hash, files[1].size, artifact.BuildLabel)
	if probeErr == nil {
		var probeConf []byte
		probeConf, probeErr = probe(ctx, paths[1], "-buildconf")
		if probeErr == nil {
			ffprobe.BuildConf = string(probeConf)
			ffprobe.BuildHash = digestBytes(probeConf)
		}
	}
	if err != nil || probeErr != nil {
		response.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_PROBE_FAILED
		response.Detail = "a pinned binary's fixed version/build probe failed or exceeded its bound"
		return response, nil
	}
	response.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT
	response.EnvironmentIdentity = rendererEnvironmentIdentity(runtime.GOOS, runtime.GOARCH)
	response.Ffmpeg = rendererBinaryProto(ffmpeg)
	response.Ffprobe = rendererBinaryProto(ffprobe)
	response.LicenseDetails = ffmpeg.License
	response.Detail = "both managed binary hashes and bounded probes match the inspected artifact; OS prerequisites and render behavior remain unqualified"
	return response, nil
}

func validateManagedDirectory(optDir string) error {
	info, err := os.Lstat(filepath.Clean(optDir))
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("managed renderer root is not a regular directory")
	}
	return nil
}

func verifyRendererMember(ctx context.Context, root, path string, expectedSize int64, expectedHash string, executable bool) error {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return errors.New("pinned renderer member escapes its managed root")
	}
	current := root
	var memberInfo os.FileInfo
	parts := strings.Split(relative, string(filepath.Separator))
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			return errors.New("pinned renderer member is missing")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("pinned renderer member path contains a symlink")
		}
		if index < len(parts)-1 && !info.IsDir() {
			return errors.New("pinned renderer member parent is not a directory")
		}
		if index == len(parts)-1 {
			if !info.Mode().IsRegular() || (executable && info.Mode().Perm()&0o111 == 0) {
				return errors.New("pinned renderer member is not a regular expected file")
			}
			if info.Size() != expectedSize {
				return errors.New("pinned renderer member size differs from the inspected artifact")
			}
			memberInfo = info
		}
	}
	file, err := os.Open(path)
	if err != nil {
		return errors.New("pinned renderer member could not be opened")
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || opened.Size() != expectedSize || !os.SameFile(memberInfo, opened) {
		return errors.New("pinned renderer member changed while opening")
	}
	hash := sha256.New()
	if _, err := io.CopyN(hash, contextReader{ctx: ctx, reader: file}, expectedSize+1); err != nil && !errors.Is(err, io.EOF) {
		return errors.New("pinned renderer member could not be hashed")
	}
	if hex.EncodeToString(hash.Sum(nil)) != expectedHash {
		return errors.New("pinned renderer member digest differs from the inspected artifact")
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader contextReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(buffer)
}

func inspectRendererBinary(ctx context.Context, probe rendererProbe, path, member, digest string, size int64, expectedBuild string) (rendererBinaryResult, error) {
	output, err := probe(ctx, path, "-version")
	if err != nil {
		return rendererBinaryResult{}, err
	}
	version := strings.TrimSpace(strings.SplitN(string(output), "\n", 2)[0])
	if version == "" || !strings.Contains(version, expectedBuild) {
		return rendererBinaryResult{}, errors.New("version probe did not report the pinned build label")
	}
	return rendererBinaryResult{Path: path, Member: member, SHA256: digest, Size: size, Version: version}, nil
}

func rendererProfileProbe(ctx context.Context, path string, args ...string) ([]byte, error) {
	probeCtx, cancel := context.WithTimeout(ctx, rendererProbeTimeout)
	defer cancel()
	command := exec.CommandContext(probeCtx, path, args...)
	command.Env = []string{"LANG=C", "LC_ALL=C"}
	var output boundedRendererOutput
	output.limit = rendererProbeOutputMax
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	if output.overflow {
		return nil, errors.New("renderer version probe output exceeded its bound")
	}
	if err != nil {
		return nil, err
	}
	return bytes.Clone(output.Bytes()), nil
}

type boundedRendererOutput struct {
	data     bytes.Buffer
	limit    int
	overflow bool
}

func (output *boundedRendererOutput) Write(value []byte) (int, error) {
	if len(value) > output.limit-output.data.Len() {
		output.overflow = true
		return 0, errors.New("renderer version probe output exceeded its bound")
	}
	return output.data.Write(value)
}

func (output *boundedRendererOutput) Bytes() []byte { return output.data.Bytes() }

func rendererBinaryProto(result rendererBinaryResult) *cliv1.RendererToolBinary {
	return &cliv1.RendererToolBinary{Path: result.Path, ArchiveMemberPath: result.Member, Sha256: result.SHA256, SizeBytes: uint64(result.Size), Version: result.Version, BuildConfigSha256: result.BuildHash, BuildConfig: result.BuildConf}
}

func rendererEnvironmentIdentity(goos, arch string) string {
	return goos + "-" + arch + "-runtime-prerequisites-unverified"
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
