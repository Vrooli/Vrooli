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
	"github.com/vrooli/vrooli/internal/stateformat"
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

func ProbeWorkerIdentity(ctx context.Context, image string, prior *cliv1.CliVersion, priorImage string) (WorkerIdentity, error) {
	var identity WorkerIdentity
	if err := validateExecutableRole(image, "github.com/vrooli/vrooli/cmd/vrooli-watchdog"); err != nil {
		return identity, err
	}
	data, err := probeNativeMetadata(ctx, image, []string{"--core-update-capability"}, workerIdentityLimit)
	if err != nil {
		return identity, err
	}
	if err := canonicalMetadata(data, &identity); err != nil {
		return identity, err
	}
	if prior == nil || identity.Protocol != 1 || !strings.HasPrefix(identity.BuildIdentity, "managed:") || !digestPattern.MatchString(strings.TrimPrefix(identity.BuildIdentity, "managed:")) || identity.RuntimeHomeContractSHA256 != prior.GetRuntimeHomeContractSha256() {
		return identity, fmt.Errorf("retained watchdog and prior root persistence/bootstrap capabilities do not agree")
	}
	// The versioned manifests decide when both images report one. A watchdog
	// built before the last compatible install can carry a different legacy
	// whole-source identity while speaking the same state formats; refusing it
	// would make the next ordinary install impossible until the watchdog
	// self-updated. Compatibility is directional, so compare the manifests and
	// fall back to the identity rule only for an image that cannot report one.
	agrees, err := workerStateFormatAgrees(ctx, priorImage, identity.StateFormat)
	if err != nil {
		return identity, err
	}
	if !agrees && identity.PersistentStateContractSHA256 != prior.GetPersistentStateContractSha256() {
		return identity, fmt.Errorf("retained watchdog and prior root persistence/bootstrap capabilities do not agree")
	}
	return identity, nil
}

// workerStateFormatAgrees compares the prior root's state format with the one
// the watchdog reports. It reports (false, nil) when either predates the
// versioned manifest, leaving the caller to use the identity rule.
func workerStateFormatAgrees(ctx context.Context, priorImage string, workerFormat *stateformat.Manifest) (bool, error) {
	if priorImage == "" || workerFormat == nil {
		return false, nil
	}
	priorFormat, priorErr := ProbeStateFormat(ctx, priorImage)
	if priorErr != nil {
		return false, nil
	}
	return workerFormatAgreement(priorFormat, *workerFormat, true)
}

// workerFormatAgreement is the decision behind workerStateFormatAgrees, split
// out so the direction can be tested without building two executables.
func workerFormatAgreement(priorFormat, workerFormat stateformat.Manifest, haveBoth bool) (bool, error) {
	if !haveBoth {
		return false, nil
	}
	assessment := stateformat.Check(priorFormat, workerFormat)
	if assessment.AllowsAutomaticRollback() {
		return true, nil
	}
	return false, fmt.Errorf("retained watchdog cannot share the serving root's state format: %s", strings.Join(formatChanges(assessment), "; "))
}
