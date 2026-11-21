package coreactivation

import (
	"bytes"
	"context"
	debugbuildinfo "debug/buildinfo"
	"fmt"
	"os"
	"runtime"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// ValidateRootExecutableRole establishes the native Go entrypoint role before
// a read-only identity invocation. It does not establish health or byte identity.
func ValidateRootExecutableRole(path string) error {
	return validateExecutableRole(path, "github.com/vrooli/vrooli/cmd/vrooli")
}

func validateExecutableRole(path, role string) error {
	file, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !file.Mode().IsRegular() || file.Size() <= 0 || file.Size() > imageLimit {
		return fmt.Errorf("root CLI must be a bounded native regular image")
	}
	info, err := debugbuildinfo.ReadFile(path)
	if err != nil || info.Path != role {
		return fmt.Errorf("candidate is not the expected Go-built native role")
	}
	settings := map[string]string{}
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	if settings["GOOS"] != runtime.GOOS || settings["GOARCH"] != runtime.GOARCH {
		return fmt.Errorf("root CLI candidate targets a different native platform")
	}
	return nil
}

type rootIdentityOutput struct {
	buffer   bytes.Buffer
	overflow bool
	limit    int
}

func (o *rootIdentityOutput) Write(p []byte) (int, error) {
	n := len(p)
	limit := o.limit
	if limit == 0 {
		limit = 8192
	}
	remaining := limit - o.buffer.Len()
	if len(p) > remaining {
		o.overflow = true
		p = p[:remaining]
	}
	_, _ = o.buffer.Write(p)
	return n, nil
}

// ProbeRootIdentity validates the bounded read-only identity protocol. Its
// caller verifies native Go role/target and frozen image identity separately.
func ProbeRootIdentity(ctx context.Context, path string) (*cliv1.CliVersion, error) {
	if ctx == nil {
		return nil, fmt.Errorf("root identity context is required")
	}
	data, err := probeNativeMetadata(ctx, path, []string{"version", "--json"}, 8192)
	if err != nil {
		return nil, fmt.Errorf("root CLI candidate version probe failed or exceeded its bounds")
	}
	var identity cliv1.CliVersion
	if err := protojson.Unmarshal(data, &identity); err != nil {
		return nil, fmt.Errorf("root CLI candidate returned an invalid identity response")
	}
	if identity.GetCliVersion() == "" || identity.GetPlatformVersion() == "" || !digestPattern.MatchString(identity.GetBuildIdentity()) || !digestPattern.MatchString(identity.GetRuntimeHomeContractSha256()) {
		return nil, fmt.Errorf("root CLI candidate lacks a stamped executable or runtime-home authority identity")
	}
	return &identity, nil
}
