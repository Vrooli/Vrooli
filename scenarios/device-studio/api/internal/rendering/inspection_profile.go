package rendering

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/repo-contract-go/cliinvoke"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	profileStdoutLimit = 64 << 10
	profileStderrLimit = 8 << 10
)

// ResolveInspectionProfile asks the control-plane owner for the current
// read-only profile. Resolution has no PATH fallback; the operator must
// configure VROOLI_BIN or install the CLI in the contract runtime home.
func ResolveInspectionProfile(ctx context.Context) (*cliv1.RendererToolProfile, error) {
	if ctx == nil {
		return nil, errors.New("renderer profile context is required")
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return nil, errors.New("cannot resolve the configured Vrooli runtime home")
	}
	return resolveInspectionProfile(ctx, os.Getenv, home, invokeInspectionProfile)
}

type inspectionProfileInvoker func(context.Context, string, string, []string) ([]byte, error)

func resolveInspectionProfile(ctx context.Context, getenv func(string) string, home string, invoke inspectionProfileInvoker) (*cliv1.RendererToolProfile, error) {
	if ctx == nil || getenv == nil || invoke == nil {
		return nil, errors.New("renderer profile resolution dependencies are required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{
		Explicit: strings.TrimSpace(getenv(cliinvoke.BinaryEnvVar)), RuntimeHome: home,
		Getenv:   func(string) string { return "" },
		LookPath: func(string) (string, error) { return "", exec.ErrNotFound },
	})
	if err != nil {
		return nil, errors.New("configured Vrooli CLI was not found in VROOLI_BIN or the contract runtime home")
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, errors.New("resolved Vrooli CLI path must be absolute and canonical")
	}
	data, err := invoke(ctx, path, "host", []string{"renderer-profile", "ffmpeg-video", "--json"})
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > profileStdoutLimit {
		return nil, errors.New("renderer profile response is empty or exceeds its size limit")
	}
	profile := new(cliv1.RendererToolProfile)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, profile); err != nil {
		return nil, errors.New("control-plane renderer profile is not valid typed JSON")
	}
	if _, err := LocalToolPairFromProfile(profile); err != nil {
		return nil, fmt.Errorf("control-plane renderer profile is not usable: %w", err)
	}
	if profileArtifactIdentity(profile) != profile.GetArtifactIdentity() {
		return nil, errors.New("control-plane renderer profile artifact identity does not match its content")
	}
	return profile, nil
}

func invokeInspectionProfile(ctx context.Context, path, command string, args []string) ([]byte, error) {
	var stdout, stderr boundedProfileBuffer
	stdout.limit, stderr.limit = profileStdoutLimit, profileStderrLimit
	result := cliinvoke.Run(ctx, cliinvoke.Invocation{
		Binary: path, Args: append([]string{command}, args...), Timeout: 20 * time.Second, WaitDelay: 2 * time.Second,
		Stdout: &stdout, Stderr: &stderr,
	})
	if result.Class != cliinvoke.OK {
		return nil, fmt.Errorf("control-plane renderer profile query failed (%s)", result.Class)
	}
	if stdout.overflow || stderr.overflow {
		return nil, errors.New("control-plane renderer profile output exceeded its limit")
	}
	return bytes.Clone(stdout.Bytes()), nil
}

type boundedProfileBuffer struct {
	data     bytes.Buffer
	limit    int
	overflow bool
}

func (b *boundedProfileBuffer) Write(data []byte) (int, error) {
	if len(data) > b.limit-b.data.Len() {
		b.overflow = true
		return 0, errors.New("renderer profile command output exceeded its limit")
	}
	return b.data.Write(data)
}

func (b *boundedProfileBuffer) Bytes() []byte { return b.data.Bytes() }
