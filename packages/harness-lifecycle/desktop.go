package agentinstall

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/cli-core/upstreamcheck"
)

// Desktop embeds the same lifecycle while replacing dependency admission and
// its host ledger with verified upstream artifacts in this user's own prefix.
func desktopHarnessDeps() harnessDeps {
	deps := defaultHarnessDeps()
	deps.desktop = true
	deps.version = func(ctx context.Context, path string) (string, error) {
		return installedVersion(ctx, path, deps.platform)
	}
	deps.installRelease = installDesktopRelease
	return deps
}

func InspectDesktopHarness(ctx context.Context, root, resource string) (HarnessReport, error) {
	h, err := LoadHarness(root, resource)
	if err != nil {
		return HarnessReport{}, err
	}
	return inspectHarness(ctx, h, desktopHarnessDeps()), nil
}

func ManageDesktopHarness(ctx context.Context, root, resource, action, actor, cleanupPlanID string) (HarnessActionResult, error) {
	return manageHarness(ctx, root, resource, action, actor, cleanupPlanID, desktopHarnessDeps())
}

type npmRelease struct {
	Name     string            `json:"name"`
	Version  string            `json:"version"`
	Optional map[string]string `json:"optionalDependencies"`
	Dist     struct {
		Tarball   string `json:"tarball"`
		Integrity string `json:"integrity"`
	} `json:"dist"`
}

const npmRegistry = "https://registry.npmjs.org"

func readNPMRelease(ctx context.Context, name, version string) (npmRelease, error) {
	var release npmRelease
	if name == "" || version == "" || strings.ContainsAny(version, "/\\") {
		return release, fmt.Errorf("invalid native release identity")
	}
	target := npmRegistry + "/" + url.PathEscape(name) + "/" + url.PathEscape(version)
	body, err := upstreamcheck.GetBody(ctx, &http.Client{Timeout: defaultPlatformDeps().controlTimeout}, target, "", 1<<20)
	if err != nil {
		return release, err
	}
	if err = json.Unmarshal(body, &release); err != nil {
		return release, err
	}
	if release.Name != name || release.Version != version {
		return release, fmt.Errorf("native registry identity mismatch for %s@%s", name, version)
	}
	return release, nil
}

func installDesktopRelease(ctx context.Context, h Harness, version string, deps harnessDeps) error {
	if err := desktopPath(h.Home, h.BinDir); err != nil {
		return err
	}
	root, err := h.versionsRoot()
	if err != nil {
		return err
	}
	if err := desktopPath(h.Home, root); err != nil {
		return err
	}
	if h.ManifestBase != "" {
		return h.installFromManifest(ctx, deps)
	}
	artifact := h.NativeArtifact
	if artifact == nil {
		return fmt.Errorf("%s has no verified native artifact for this platform", h.Resource)
	}
	parent, err := readNPMRelease(ctx, h.Package, version)
	if err != nil {
		return err
	}
	spec := parent.Optional[artifact.Dependency]
	nativeName, nativeVersion := artifact.Dependency, spec
	if strings.HasPrefix(spec, "npm:") {
		spec = strings.TrimPrefix(spec, "npm:")
		split := strings.LastIndex(spec, "@")
		if split <= 0 {
			return fmt.Errorf("invalid native optional package alias")
		}
		nativeName, nativeVersion = spec[:split], spec[split+1:]
	}
	release, err := readNPMRelease(ctx, nativeName, nativeVersion)
	if err != nil {
		return err
	}
	integrity := strings.TrimPrefix(release.Dist.Integrity, "sha512-")
	checksum, err := base64.StdEncoding.DecodeString(integrity)
	if err != nil || len(checksum) != 64 || !strings.HasPrefix(release.Dist.Integrity, "sha512-") {
		return fmt.Errorf("native artifact requires published SHA-512 integrity")
	}
	prefix, err := npmInstallPrefix(h.Home, h.Resource, version, deps.platform)
	if err != nil {
		return err
	}
	unlock, err := deps.platform.acquireLock(ctx, "resource-agent-"+h.Resource)
	if err != nil {
		return err
	}
	if _, err = deps.platform.ensureOwnedDir(root); err != nil {
		unlock()
		return err
	}
	if _, statErr := os.Lstat(prefix); os.IsNotExist(statErr) {
		err = buildDesktopPrefix(ctx, h, release, hex.EncodeToString(checksum), root, prefix, version, deps)
	} else if statErr != nil {
		err = statErr
	}
	unlock()
	if err != nil {
		return err
	}
	install := managedInstall{Version: version, Prefix: prefix, Executable: filepath.Join(prefix, h.Executable)}
	if err := h.verifyInstall(ctx, install); err != nil {
		return fmt.Errorf("preserve invalid existing native prefix: %w", err)
	}
	if err := h.selectInstall(ctx, install, deps); err != nil {
		return err
	}
	return verifySelectedUpdate(ctx, h.Binary, h.linkPath(), version, updateDeps{installed: deps.version, lookPath: deps.lookPath, stat: os.Stat})
}

func buildDesktopPrefix(ctx context.Context, h Harness, release npmRelease, checksum, root, prefix, version string, deps harnessDeps) error {
	candidate, err := os.MkdirTemp(root, ".candidate-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(candidate)
	archive := filepath.Join(candidate, "download.tgz")
	if err := downloadVerifiedSHA512WithDeps(ctx, npmRegistry, release.Dist.Tarball, archive, checksum, deps.platform); err != nil {
		return err
	}
	extracted := filepath.Join(candidate, "unpacked")
	if _, err := binaryfetch.ExtractArchiveBounded(archive, "tar.gz", extracted, binaryfetch.ExtractOptions{MaxEntries: 10000, MaxExpandedBytes: 2 << 30}); err != nil {
		return err
	}
	nativeRoot := filepath.Join(candidate, "node_modules", filepath.FromSlash(h.NativeArtifact.Dependency))
	if err := os.MkdirAll(filepath.Dir(nativeRoot), 0o700); err != nil {
		return err
	}
	if err := os.Rename(filepath.Join(extracted, "package"), nativeRoot); err != nil {
		return err
	}
	if err := os.RemoveAll(extracted); err != nil {
		return err
	}
	if err := os.Remove(archive); err != nil {
		return err
	}
	expected, err := prepareNativeCandidate(ctx, h, candidate)
	if err != nil {
		return err
	}
	raw, err := installedVersion(ctx, expected, deps.platform)
	if err != nil || upstreamcheck.ExtractSemver(raw) != version {
		return fmt.Errorf("native candidate identity: got %q, want %s: %v", raw, version, err)
	}
	digest, err := binaryfetch.TreeDigestContext(ctx, filepath.Join(candidate, "node_modules"))
	if err != nil {
		return err
	}
	record, err := json.Marshal(installationRecord{Resource: h.Resource, Package: h.Package, Version: version, TreeSHA256: digest})
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(candidate, "installation.json"), record, 0o600); err != nil {
		return err
	}
	return deps.platform.commitOwnedPath(candidate, prefix)
}

// PrepareNativeHarnessCandidate materializes a declared compressed executable
// in a private npm candidate without running its installer scripts. The caller
// still owns package identity checks, offline qualification and publication.
func PrepareNativeHarnessCandidate(ctx context.Context, root, resource, candidate string) error {
	h, err := LoadHarness(root, resource)
	if err != nil {
		return err
	}
	if h.NativeArtifact == nil || h.NativeArtifact.Compression == "" {
		return nil
	}
	_, err = prepareNativeCandidate(ctx, h, candidate)
	return err
}

func prepareNativeCandidate(ctx context.Context, h Harness, candidate string) (string, error) {
	nativeRoot := filepath.Join(candidate, "node_modules", filepath.FromSlash(h.NativeArtifact.Dependency))
	executable := filepath.Join(nativeRoot, filepath.FromSlash(h.NativeArtifact.Executable))
	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil || !withinDir(resolved, nativeRoot) || !withinDir(resolved, candidate) {
		return "", fmt.Errorf("native executable is outside its verified candidate")
	}
	if h.NativeArtifact.Compression == "brotli" {
		executable, err = decodeNativeExecutable(ctx, executable, 512<<20)
		if err != nil {
			return "", err
		}
	}
	if err := os.Chmod(executable, 0o755); err != nil {
		return "", err
	}
	expected := filepath.Join(candidate, h.Executable)
	if expected == executable {
		return expected, nil
	}
	if info, err := os.Lstat(expected); err == nil {
		resolved, resolveErr := filepath.EvalSymlinks(expected)
		if info.Mode()&os.ModeSymlink == 0 || resolveErr != nil || !withinDir(resolved, candidate) {
			return "", fmt.Errorf("preserve unexpected native candidate entry point")
		}
		if err := os.Remove(expected); err != nil {
			return "", err
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(expected), 0o700); err != nil {
		return "", err
	}
	relative, err := filepath.Rel(filepath.Dir(expected), executable)
	if err != nil {
		return "", err
	}
	return expected, os.Symlink(relative, expected)
}

// Decode only inside the verified candidate tree; an error leaves no runnable
// output. The caller still verifies --version and receipts before publication.
func decodeNativeExecutable(ctx context.Context, source string, limit int64) (string, error) {
	input, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer input.Close()
	destination := source + ".decoded"
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	reader := brotli.NewReader(input)
	buffer := make([]byte, 64<<10)
	var total int64
	for err == nil {
		if err = ctx.Err(); err != nil {
			break
		}
		var n int
		n, err = reader.Read(buffer)
		total += int64(n)
		if total > limit {
			err = fmt.Errorf("decoded native artifact exceeds %d bytes", limit)
			break
		}
		if n > 0 {
			if _, writeErr := output.Write(buffer[:n]); writeErr != nil {
				err = writeErr
				break
			}
		}
	}
	closeErr := output.Close()
	if err == io.EOF {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(destination)
		return "", err
	}
	return destination, nil
}

// Validate the nearest existing ancestor before creating any path. A lexical
// prefix check alone would accept a user-home symlink into another directory.
func desktopPath(home, target string) error {
	base, err := filepath.EvalSymlinks(home)
	if err != nil {
		return err
	}
	if !withinDir(target, home) {
		return fmt.Errorf("desktop install path is outside your home")
	}
	for ancestor := target; ; ancestor = filepath.Dir(ancestor) {
		resolved, err := filepath.EvalSymlinks(ancestor)
		if err == nil {
			if resolved == base || withinDir(resolved, base) {
				return nil
			}
			return fmt.Errorf("desktop install path escapes your home through a symlink")
		}
		if !os.IsNotExist(err) {
			return err
		}
		if filepath.Dir(ancestor) == ancestor {
			return fmt.Errorf("desktop install path has no user-owned ancestor")
		}
	}
}
