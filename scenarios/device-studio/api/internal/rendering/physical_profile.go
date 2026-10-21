package rendering

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/repo-contract-go/cliinvoke"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	physicalProfileOutputMax = 64 << 10
	physicalProfileErrorMax  = 8 << 10
)

type physicalProfileBuffer struct {
	data  bytes.Buffer
	limit int
}

func (b *physicalProfileBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.data.Len() {
		return 0, errors.New("Blender profile output exceeded its bound")
	}
	return b.data.Write(p)
}

// ResolvePhysicalBlenderProfile uses the supported control-plane profile
// owner. PATH is deliberately unavailable; an explicit VROOLI_BIN or the
// user's runtime-home installation is required.
func ResolvePhysicalBlenderProfile(ctx context.Context) ([]byte, *cliv1.BlenderToolProfile, error) {
	if ctx == nil || runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return nil, nil, errors.New("physical Blender profile requires a context on Linux/amd64")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, fmt.Errorf("resolve user home for control-plane CLI: %w", err)
	}
	binary, err := resolvePhysicalBlenderCLI(os.Getenv, home)
	if err != nil {
		return nil, nil, err
	}
	callCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	var stdout, stderr physicalProfileBuffer
	stdout.limit, stderr.limit = physicalProfileOutputMax, physicalProfileErrorMax
	run := cliinvoke.Run(callCtx, cliinvoke.Invocation{
		Binary: binary, Args: []string{"host", "renderer-profile", "blender", "--json"},
		Timeout: 20 * time.Second, WaitDelay: 2 * time.Second, Stdout: &stdout, Stderr: &stderr,
	})
	if run.Class != cliinvoke.OK {
		return nil, nil, fmt.Errorf("governed Blender profile query failed (%s)", run.Class)
	}
	profile := new(cliv1.BlenderToolProfile)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(stdout.data.Bytes(), profile); err != nil {
		return nil, nil, errors.New("governed Blender profile response is not typed JSON")
	}
	if err := validatePhysicalBlenderProfile(profile); err != nil {
		return nil, nil, err
	}
	return bytes.Clone(stdout.data.Bytes()), profile, nil
}

func resolvePhysicalBlenderCLI(getenv func(string) string, home string) (string, error) {
	if getenv == nil || strings.TrimSpace(home) == "" {
		return "", errors.New("physical renderer CLI resolution requires the user home and environment")
	}
	binary, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{
		Explicit: getenv(cliinvoke.BinaryEnvVar), RuntimeHome: home,
		Getenv: func(key string) string {
			if key == cliinvoke.BinaryEnvVar {
				return getenv(cliinvoke.BinaryEnvVar)
			}
			return ""
		},
		LookPath: func(string) (string, error) { return "", exec.ErrNotFound },
	})
	if err != nil {
		return "", fmt.Errorf("resolve governed control-plane CLI: %w", err)
	}
	return binary, nil
}

func validatePhysicalBlenderProfile(profile *cliv1.BlenderToolProfile) error {
	if profile == nil || profile.GetSchemaVersion() != 1 || profile.GetProfileId() != PhysicalBlenderProfileID ||
		profile.GetState() != cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT ||
		profile.GetArtifactIdentity() != PhysicalBlenderArtifact || profile.GetExpectedTreeSha256() != PhysicalBlenderTreeSHA ||
		profile.GetOs() != "linux" || profile.GetArchitecture() != "amd64" {
		return errors.New("managed Blender profile is missing, stale, or outside the pinned Linux/amd64 artifact")
	}
	bin := profile.GetBlender()
	if bin == nil || bin.GetVersion() != PhysicalBlenderVersion || bin.GetSha256() != PhysicalBlenderBinarySHA ||
		bin.GetArchiveMemberPath() != PhysicalBlenderArchive+"/blender" || bin.GetSizeBytes() == 0 ||
		!filepath.IsAbs(bin.GetPath()) || filepath.Clean(bin.GetPath()) != bin.GetPath() || filepath.Base(bin.GetPath()) != "blender" ||
		filepath.Base(filepath.Dir(bin.GetPath())) != PhysicalBlenderArchive {
		return errors.New("managed Blender executable facts do not match the pinned archive profile")
	}
	if strings.ContainsAny(bin.GetPath(), "\x00\n\r") {
		return errors.New("managed Blender executable path contains invalid characters")
	}
	return nil
}
