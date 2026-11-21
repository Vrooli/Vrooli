package hostapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	ffmpegvideo "github.com/vrooli/vrooli/internal/tools/ffmpeg-video"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func TestRendererProfileMissingAndUnsupportedNeverProbe(t *testing.T) {
	artifact := testRendererArtifact(t)
	probeCalls := 0
	probe := func(context.Context, string, ...string) ([]byte, error) {
		probeCalls++
		return nil, errors.New("must not run")
	}

	missing, err := resolveRendererProfileForPlatform(context.Background(), filepath.Join(t.TempDir(), "not-installed"), true, artifact, "linux", "amd64", probe)
	if err != nil || missing.State != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_MISSING {
		t.Fatalf("missing profile = %+v, err %v", missing, err)
	}
	unsupported, err := resolveRendererProfileForPlatform(context.Background(), filepath.Join(t.TempDir(), "not-installed"), false, artifact, "darwin", "arm64", probe)
	if err != nil || unsupported.State != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_UNSUPPORTED_PLATFORM {
		t.Fatalf("unsupported profile = %+v, err %v", unsupported, err)
	}
	if probeCalls != 0 {
		t.Fatalf("probes on missing/unsupported paths = %d", probeCalls)
	}
}

func TestRendererProfileRequiresRecordedOwnerAndCompletePair(t *testing.T) {
	artifact := testRendererArtifact(t)
	root := t.TempDir()
	writeRendererFixture(t, root, artifact, false)
	probeCalls := 0
	probe := func(context.Context, string, ...string) ([]byte, error) {
		probeCalls++
		return nil, errors.New("must not run")
	}

	unmanaged, err := resolveRendererProfile(context.Background(), root, false, artifact, probe)
	if err != nil || unmanaged.State != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_UNMANAGED {
		t.Fatalf("unmanaged profile = %+v, err %v", unmanaged, err)
	}
	partial, err := resolveRendererProfile(context.Background(), root, true, artifact, probe)
	if err != nil || partial.State != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_PARTIAL {
		t.Fatalf("partial profile = %+v, err %v", partial, err)
	}
	if probeCalls != 0 {
		t.Fatalf("probes before pair verification = %d", probeCalls)
	}
}

func TestRendererProfileVerifiesExactPairBeforeFixedProbes(t *testing.T) {
	artifact := testRendererArtifact(t)
	root := t.TempDir()
	writeRendererFixture(t, root, artifact, true)
	var calls []string
	probe := func(_ context.Context, path string, args ...string) ([]byte, error) {
		calls = append(calls, path+" "+strings.Join(args, " "))
		if len(args) == 1 && args[0] == "-version" {
			return []byte("ffmpeg version fixture-build\n"), nil
		}
		if len(args) == 1 && args[0] == "-L" {
			return []byte("FFmpeg build license details fixture\n"), nil
		}
		return []byte("configuration: --enable-gpl\n"), nil
	}

	profile, err := resolveRendererProfile(context.Background(), root, true, artifact, probe)
	if err != nil {
		t.Fatal(err)
	}
	if profile.State != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT {
		t.Fatalf("profile state = %s (%s)", profile.State, profile.Detail)
	}
	wantCalls := []string{
		filepath.Join(root, artifact.FFmpegPath) + " -version",
		filepath.Join(root, artifact.FFmpegPath) + " -buildconf",
		filepath.Join(root, artifact.FFmpegPath) + " -L",
		filepath.Join(root, artifact.FFprobePath) + " -version",
		filepath.Join(root, artifact.FFprobePath) + " -buildconf",
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("probe calls = %#v, want %#v", calls, wantCalls)
	}
	if profile.Ffmpeg.GetBuildConfigSha256() != hashString("configuration: --enable-gpl\n") || profile.Ffprobe.GetBuildConfigSha256() != profile.Ffmpeg.GetBuildConfigSha256() {
		t.Fatalf("build configuration identities = %q / %q", profile.Ffmpeg.GetBuildConfigSha256(), profile.Ffprobe.GetBuildConfigSha256())
	}
	if profile.GetLicenseDetails() != "FFmpeg build license details fixture\n" {
		t.Fatalf("license probe output = %q", profile.GetLicenseDetails())
	}
	if !strings.Contains(profile.GetQualificationScope(), "no video workflow") {
		t.Fatalf("profile overstates qualification: %q", profile.GetQualificationScope())
	}
}

func TestRendererProfileRejectsTamperingSymlinksAndUnknownBuilds(t *testing.T) {
	artifact := testRendererArtifact(t)
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string, ffmpegvideo.Artifact)
		probe  rendererProbe
		state  cliv1.RendererToolProfileState
	}{
		{
			name: "changed bytes",
			mutate: func(t *testing.T, root string, a ffmpegvideo.Artifact) {
				writeRendererFixture(t, root, a, true)
				if err := os.WriteFile(filepath.Join(root, a.FFmpegPath), []byte("tampered"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			probe: func(context.Context, string, ...string) ([]byte, error) {
				t.Fatal("tampered binary was probed")
				return nil, nil
			},
			state: cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_TAMPERED,
		},
		{
			name: "symlink",
			mutate: func(t *testing.T, root string, a ffmpegvideo.Artifact) {
				writeRendererFixture(t, root, a, true)
				if err := os.Remove(filepath.Join(root, a.FFmpegPath)); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(root, a.FFprobePath), filepath.Join(root, a.FFmpegPath)); err != nil {
					t.Fatal(err)
				}
			},
			probe: func(context.Context, string, ...string) ([]byte, error) {
				t.Fatal("symlinked binary was probed")
				return nil, nil
			},
			state: cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_TAMPERED,
		},
		{
			name:   "unrecognized build",
			mutate: func(t *testing.T, root string, a ffmpegvideo.Artifact) { writeRendererFixture(t, root, a, true) },
			probe: func(context.Context, string, ...string) ([]byte, error) {
				return []byte("ffmpeg version other-build\n"), nil
			},
			state: cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_PROBE_FAILED,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			test.mutate(t, root, artifact)
			profile, err := resolveRendererProfile(context.Background(), root, true, artifact, test.probe)
			if err != nil || profile.State != test.state {
				t.Fatalf("profile = %+v, err %v; want %s", profile, err, test.state)
			}
		})
	}
}

func TestRendererArtifactIdentityExcludesPathsAndObservationTime(t *testing.T) {
	artifact := testRendererArtifact(t)
	probe := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if args[0] == "-version" {
			return []byte("ffmpeg version fixture-build\n"), nil
		}
		return []byte("configuration: fixture\n"), nil
	}
	firstRoot, secondRoot := t.TempDir(), t.TempDir()
	writeRendererFixture(t, firstRoot, artifact, true)
	writeRendererFixture(t, secondRoot, artifact, true)
	first, err := resolveRendererProfile(context.Background(), firstRoot, true, artifact, probe)
	if err != nil {
		t.Fatal(err)
	}
	second, err := resolveRendererProfile(context.Background(), secondRoot, true, artifact, probe)
	if err != nil {
		t.Fatal(err)
	}
	if first.GetFfmpeg().GetPath() == second.GetFfmpeg().GetPath() {
		t.Fatal("fixture relocation did not change the resolved executable path")
	}
	if first.GetArtifactIdentity() != second.GetArtifactIdentity() {
		t.Fatal("same content at a different install path produced a different artifact identity")
	}
	artifact.FFmpegPath = "relocated/bin/ffmpeg"
	if rendererArtifactIdentity(artifact) == first.GetArtifactIdentity() {
		t.Fatal("member layout was omitted from content identity")
	}
}

func TestBoundedRendererOutputRejectsOverflow(t *testing.T) {
	var output boundedRendererOutput
	output.limit = 3
	if _, err := output.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if _, err := output.Write([]byte("d")); err == nil {
		t.Fatal("oversized probe output was accepted")
	}
}

func TestRendererProfileProbeRejectsActualSubprocessOverflow(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("renderer profile subprocess fixture is qualified on Linux")
	}
	probe := filepath.Join(t.TempDir(), "profile-probe")
	if err := os.WriteFile(probe, []byte("#!/bin/sh\nprintf '%40000s' x\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	output, err := rendererProfileProbe(context.Background(), probe, "-version")
	if err == nil || output != nil {
		t.Fatalf("actual renderer probe returned output beyond 32 KiB bound: bytes=%d err=%v", len(output), err)
	}
}

func TestRendererProfileHashingHonorsCancellation(t *testing.T) {
	artifact := testRendererArtifact(t)
	root := t.TempDir()
	writeRendererFixture(t, root, artifact, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	probeCalls := 0
	profile, err := resolveRendererProfile(ctx, root, true, artifact, func(context.Context, string, ...string) ([]byte, error) {
		probeCalls++
		return nil, errors.New("cancelled input must not be probed")
	})
	if !errors.Is(err, context.Canceled) || profile != nil {
		t.Fatalf("cancelled resolve = %+v, %v", profile, err)
	}
	if probeCalls != 0 {
		t.Fatalf("cancelled profile ran %d probes", probeCalls)
	}
}

func testRendererArtifact(t *testing.T) ffmpegvideo.Artifact {
	t.Helper()
	artifact := ffmpegvideo.PinnedArtifact()
	artifact.ProfileID = "fixture-profile"
	artifact.ArchiveSize = 3
	artifact.ArchiveSHA256 = hashString("arc")
	artifact.FFmpegPath = "root/bin/ffmpeg"
	artifact.FFmpegSHA256 = hashString("ffmpeg-bytes")
	artifact.FFmpegSize = int64(len("ffmpeg-bytes"))
	artifact.FFprobePath = "root/bin/ffprobe"
	artifact.FFprobeSHA256 = hashString("ffprobe-bytes")
	artifact.FFprobeSize = int64(len("ffprobe-bytes"))
	artifact.LicensePath = "root/LICENSE.txt"
	artifact.LicenseSHA256 = hashString("license")
	artifact.LicenseSize = int64(len("license"))
	artifact.BuildLabel = "fixture-build"
	return artifact
}

func writeRendererFixture(t *testing.T, root string, artifact ffmpegvideo.Artifact, complete bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, artifact.FFmpegPath)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, artifact.FFprobePath)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, artifact.LicensePath)), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, file := range []struct {
		path, contents string
		mode           os.FileMode
	}{
		{artifact.FFmpegPath, "ffmpeg-bytes", 0o755},
		{artifact.FFprobePath, "ffprobe-bytes", 0o755},
		{artifact.LicensePath, "license", 0o644},
	} {
		if !complete && file.path == artifact.FFprobePath {
			continue
		}
		if err := os.WriteFile(filepath.Join(root, file.path), []byte(file.contents), file.mode); err != nil {
			t.Fatal(err)
		}
	}
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
