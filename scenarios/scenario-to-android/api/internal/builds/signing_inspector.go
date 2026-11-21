package builds

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var certificateSHA256Pattern = regexp.MustCompile(`(?i)Signer #[0-9]+ certificate SHA-256 digest:\s*([0-9a-f]{64})`)

func inspectAPKSigningCertificate(ctx context.Context, apkPath string) (string, error) {
	tool, err := resolveAPKSigTool()
	if err != nil {
		return "", err
	}
	output, err := exec.CommandContext(ctx, tool, "verify", "--print-certs", apkPath).CombinedOutput() // #nosec G204 -- resolved governed Android SDK tool and immutable build output
	if err != nil {
		return "", fmt.Errorf("apksigner verify failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	match := certificateSHA256Pattern.FindSubmatch(output)
	if len(match) != 2 {
		return "", fmt.Errorf("apksigner did not report a signer certificate SHA-256 digest")
	}
	return strings.ToLower(string(match[1])), nil
}

func resolveAPKSigTool() (string, error) {
	sdkRoot := governedSDKRoot()
	if sdkRoot != "" {
		buildToolsRoot := filepath.Join(sdkRoot, "build-tools")
		entries, err := os.ReadDir(buildToolsRoot)
		if err != nil {
			return "", fmt.Errorf("read Android build-tools: %w", err)
		}
		versions := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() {
				versions = append(versions, entry.Name())
			}
		}
		sort.Sort(sort.Reverse(sort.StringSlice(versions)))
		for _, version := range versions {
			candidate := filepath.Join(buildToolsRoot, version, "apksigner")
			if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
				return candidate, nil
			}
		}
	}
	if path, err := exec.LookPath("apksigner"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("apksigner is unavailable in Android SDK build-tools")
}

func validSHA256Digest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}
