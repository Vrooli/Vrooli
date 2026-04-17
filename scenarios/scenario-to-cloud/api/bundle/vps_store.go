package bundle

import (
	"context"
	"fmt"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/shellutil"
	"scenario-to-cloud/ssh"
	"strings"
	"time"
)

// ListVPSBundles lists bundles under <workdir>/.vrooli/cloud/bundles on the target VPS.
func ListVPSBundles(ctx context.Context, sshRunner ssh.Runner, cfg ssh.Config, workdir string) ([]domain.VPSBundleInfo, int64, error) {
	bundlesPath := shellutil.SafeRemoteJoin(workdir, ".vrooli/cloud/bundles")
	return listVPSBundlesByPath(ctx, sshRunner, cfg, bundlesPath)
}

// listVPSBundlesByPath lists bundles under the given remote bundlesPath via SSH.
// bundlesPath should be an absolute path (typically <workdir>/.vrooli/cloud/bundles).
func listVPSBundlesByPath(ctx context.Context, sshRunner ssh.Runner, cfg ssh.Config, bundlesPath string) ([]domain.VPSBundleInfo, int64, error) {
	// List bundles with size and modification time (format: size_bytes\tfilename\tmod_time_unix)
	listCmd := fmt.Sprintf(
		`cd %s 2>/dev/null && ls -1 mini-vrooli_*.tar.gz 2>/dev/null | while read f; do stat --printf="%%s\t%%n\t%%Y\n" "$f" 2>/dev/null; done || true`,
		shellutil.QuoteSingle(bundlesPath),
	)

	res, err := sshRunner.Run(ctx, cfg, listCmd, ssh.DefaultRunOptions())
	if err != nil {
		return nil, 0, err
	}

	bundles, totalSize, parseErr := ParseVPSBundleOutput(res.Stdout)
	if parseErr != nil {
		return nil, 0, parseErr
	}

	return bundles, totalSize, nil
}

// ParseVPSBundleOutput parses the output from listing VPS bundles.
// Expected format: size_bytes\tfilename\tmod_time_unix (one per line)
func ParseVPSBundleOutput(output string) ([]domain.VPSBundleInfo, int64, error) {
	var bundles []domain.VPSBundleInfo
	var totalSize int64

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			continue
		}

		sizeBytes := ParseBytes(parts[0])
		filename := parts[1]
		modTimeUnix := ParseBytes(parts[2])

		scenarioID, sha256Hash := ParseBundleFilename(filename)

		bundles = append(bundles, domain.VPSBundleInfo{
			Filename:   filename,
			ScenarioID: scenarioID,
			Sha256:     sha256Hash,
			SizeBytes:  sizeBytes,
			ModTime:    time.Unix(modTimeUnix, 0).UTC().Format(time.RFC3339),
		})
		totalSize += sizeBytes
	}

	return bundles, totalSize, nil
}

// ParseBytes parses a byte count string, returning 0 on error.
func ParseBytes(s string) int64 {
	var n int64
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0
	}
	return n
}
