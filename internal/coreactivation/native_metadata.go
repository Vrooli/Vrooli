package coreactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func InstalledWatchdogPath(home string) (string, error) {
	root, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "libexec", "vrooli-watchdog"), nil
}

func probeNativeMetadata(ctx context.Context, path string, args []string, limit int) ([]byte, error) {
	if ctx == nil {
		return nil, ErrConflict
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	home, err := os.MkdirTemp("", "vrooli-native-metadata-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(home)
	name, wrapped := hostreqkit.InvokingUserCommand(path, args...)
	cmd := exec.CommandContext(ctx, name, wrapped...)
	cmd.Dir = home
	cmd.WaitDelay = time.Second
	diagnostic := rootIdentityOutput{limit: 1024}
	cmd.Stderr = &diagnostic
	missing := filepath.Join(home, "source-unavailable")
	cmd.Env = []string{"HOME=" + home, "PATH=/usr/local/bin:/usr/bin:/bin", "VROOLI_SOURCE_ROOT=" + missing, "VROOLI_ROOT=" + missing}
	out := rootIdentityOutput{limit: limit}
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil || out.overflow {
		// The executable's own first diagnostic line names why it refused.
		reason, _, _ := strings.Cut(strings.TrimSpace(diagnostic.buffer.String()), "\n")
		if reason != "" {
			return nil, fmt.Errorf("native metadata probe failed: %s", reason)
		}
		return nil, fmt.Errorf("native metadata probe failed or exceeded its bound")
	}
	entries, err := os.ReadDir(home)
	if err != nil || len(entries) != 0 {
		return nil, fmt.Errorf("native metadata probe initialized runtime files")
	}
	return bytes.TrimSpace(out.buffer.Bytes()), nil
}

// ReadDefinition retains the same regular-file and byte bounds as frozen
// definitions when the admission owner captures the installed prior unit.
func ReadDefinition(path string) ([]byte, error) { return readFrozen(path, definitionLimit) }

func canonicalMetadata(data []byte, target any) error {
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("invalid native metadata response")
	}
	canonical, err := json.Marshal(target)
	if err != nil || !bytes.Equal(canonical, data) {
		return fmt.Errorf("noncanonical native metadata response")
	}
	return nil
}

func ProbeRootDefinition(ctx context.Context, image string, l Layout, sourceRoot string) ([]byte, error) {
	if err := ValidateRootExecutableRole(image); err != nil {
		return nil, err
	}
	if err := l.validate(l.Home); err != nil {
		return nil, err
	}
	data, err := probeNativeMetadata(ctx, image, []string{"--native-supervisor-definition", l.Home, sourceRoot, l.Executable, l.LogPath}, definitionLimit)
	if err != nil {
		return nil, err
	}
	var artifact platform.RenderedArtifact
	if err := canonicalMetadata(data, &artifact); err != nil {
		return nil, err
	}
	if artifact.Target != "linux" || len(artifact.Files) != 1 || artifact.Files[0].Name != l.UnitName || len(artifact.Files[0].Content) == 0 {
		return nil, fmt.Errorf("candidate does not export the fixed root native definition")
	}
	return []byte(artifact.Files[0].Content), nil
}

func ProbeWorkerIdentity(ctx context.Context, image string, prior *cliv1.CliVersion) (WorkerIdentity, error) {
	var identity WorkerIdentity
	if err := validateExecutableRole(image, "github.com/vrooli/vrooli/cmd/vrooli-watchdog"); err != nil {
		return identity, err
	}
	data, err := probeNativeMetadata(ctx, image, []string{"--core-update-capability"}, 8192)
	if err != nil {
		return identity, err
	}
	if err := canonicalMetadata(data, &identity); err != nil {
		return identity, err
	}
	if prior == nil || identity.Protocol != 1 || !strings.HasPrefix(identity.BuildIdentity, "managed:") || !digestPattern.MatchString(strings.TrimPrefix(identity.BuildIdentity, "managed:")) || identity.RuntimeHomeContractSHA256 != prior.GetRuntimeHomeContractSha256() || identity.PersistentStateContractSHA256 != prior.GetPersistentStateContractSha256() {
		return identity, fmt.Errorf("retained watchdog and prior root persistence/bootstrap capabilities do not agree")
	}
	return identity, nil
}
