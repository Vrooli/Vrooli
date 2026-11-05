package agentinstall

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/cli-core/upstreamcheck"
)

// releaseDownloadLimit bounds a manifest-released archive, so a hostile host
// cannot fill the disk before the checksum fails. A variable only for tests.
var releaseDownloadLimit int64 = 1 << 30

// releaseArtifactPrefixes are the locations a manifest may send a download to,
// apart from its own host. A prefix, not a host: anyone can publish a bucket on
// storage.googleapis.com. See "Version sources" in docs/configuration/host/tools.md.
var releaseArtifactPrefixes = map[string][]string{
	AntigravityManifestBase: {"https://storage.googleapis.com/antigravity-public/"},
}

// releaseManifest is the flat per-platform descriptor upstream publishes.
type releaseManifest struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA512  string `json:"sha512"`
}

// releasePlatform is the manifest key for this host, such as "linux_amd64".
func releasePlatform() string { return releasePlatformWith(defaultPlatformDeps()) }

func releasePlatformWith(platform platformDeps) string {
	platform = platform.withDefaults()
	return platform.goos + "_" + platform.goarch
}

func fetchReleaseManifest(ctx context.Context, base, platform string) (releaseManifest, error) {
	return fetchReleaseManifestWithDeps(ctx, base, platform, defaultPlatformDeps())
}

func fetchReleaseManifestWithDeps(ctx context.Context, base, platform string, deps platformDeps) (releaseManifest, error) {
	target := strings.TrimRight(base, "/") + "/manifests/" + platform + ".json"
	if err := requireTransportSecurity(target); err != nil {
		return releaseManifest{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return releaseManifest{}, err
	}
	response, err := (&http.Client{Timeout: deps.withDefaults().controlTimeout}).Do(request)
	if err != nil {
		return releaseManifest{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return releaseManifest{}, fmt.Errorf("GET %s: status %d", target, response.StatusCode)
	}
	var release releaseManifest
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<16)).Decode(&release); err != nil {
		return releaseManifest{}, fmt.Errorf("parse %s: %w", target, err)
	}
	release.Version = strings.TrimSpace(release.Version)
	if upstreamcheck.ExtractSemver(release.Version) != release.Version || release.Version == "" {
		return releaseManifest{}, fmt.Errorf("%s has no usable version: %q", target, release.Version)
	}
	return release, nil
}

// requireTransportSecurity keeps the checksum meaningful: a digest read over an
// unauthenticated channel proves nothing. Loopback is local and always allowed.
func requireTransportSecurity(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if ip := net.ParseIP(parsed.Hostname()); ip != nil && ip.IsLoopback() {
		return nil
	}
	return fmt.Errorf("refusing insecure release URL %s", raw)
}

// requireArtifactLocation limits where a manifest can send the download. The
// manifest supplies both the URL and its checksum, so the checksum alone proves
// nothing about who published the bytes.
func requireArtifactLocation(base, artifact string) error {
	if err := requireTransportSecurity(artifact); err != nil {
		return err
	}
	parsed, err := url.Parse(artifact)
	if err != nil {
		return err
	}
	manifestHost, err := url.Parse(base)
	if err != nil {
		return err
	}
	if parsed.Host == manifestHost.Host && parsed.Scheme == manifestHost.Scheme {
		return nil
	}
	clean := parsed.Scheme + "://" + parsed.Host + path.Clean(parsed.Path)
	for _, prefix := range releaseArtifactPrefixes[base] {
		if strings.HasPrefix(clean, prefix) {
			return nil
		}
	}
	return fmt.Errorf("refusing release download from %s: it is not on the manifest host or the allowlist", artifact)
}

// installFromManifest installs a manifest-released harness into a version prefix
// and selects it. The manifest pin is bypassed the way npm update bypasses it.
// The prefix commits only after the archive checksum and the candidate's own
// --version agree, and the selected link only moves after that.
func (h Harness) installFromManifest(ctx context.Context, deps harnessDeps) error {
	deps.platform = deps.platform.withDefaults()
	release, err := fetchReleaseManifestWithDeps(ctx, h.ManifestBase, releasePlatformWith(deps.platform), deps.platform)
	if err != nil {
		return err
	}
	root, err := h.versionsRoot()
	if err != nil {
		return err
	}
	lock, err := deps.platform.acquireLock(ctx, "resource-agent-"+h.Resource)
	if err != nil {
		return err
	}
	if _, err := deps.platform.ensureOwnedDir(root); err != nil {
		lock()
		return err
	}
	if err := deps.recordArtifact(h.Home, h.Resource, root, true); err != nil {
		lock()
		return err
	}
	prefix, err := npmInstallPrefix(h.Home, h.Resource, release.Version, deps.platform)
	if err != nil {
		lock()
		return err
	}
	if _, statErr := os.Lstat(prefix); os.IsNotExist(statErr) {
		if err := h.buildManifestPrefix(ctx, release, root, prefix, deps); err != nil {
			lock()
			return err
		}
	} else if statErr != nil {
		lock()
		return statErr
	}
	lock()
	install := managedInstall{Version: release.Version, Prefix: prefix, Executable: filepath.Join(prefix, h.Binary)}
	if err := h.verifyInstall(ctx, install); err != nil {
		return fmt.Errorf("preserve invalid existing %s prefix: %w", release.Version, err)
	}
	if err := h.selectInstall(ctx, install, deps); err != nil {
		return err
	}
	return verifySelectedUpdate(ctx, h.Binary, h.linkPath(), release.Version, updateDeps{installed: deps.version, lookPath: deps.lookPath, stat: os.Stat})
}

func (h Harness) buildManifestPrefix(ctx context.Context, release releaseManifest, root, destination string, deps harnessDeps) error {
	candidate, err := os.MkdirTemp(root, ".candidate-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(candidate)
	archive := filepath.Join(candidate, "download")
	if err := downloadVerifiedSHA512WithDeps(ctx, h.ManifestBase, release.URL, archive, release.SHA512, deps.platform); err != nil {
		return err
	}
	executable := filepath.Join(candidate, h.Binary)
	if err := extractTarGzWithDeps(archive, executable, h.ArchiveEntry, deps.platform); err != nil {
		return err
	}
	if err := os.Remove(archive); err != nil {
		return err
	}
	raw, err := deps.version(ctx, executable)
	if err != nil {
		return fmt.Errorf("candidate %s does not run: %w", release.Version, err)
	}
	if got := upstreamcheck.ExtractSemver(raw); got != release.Version {
		return fmt.Errorf("candidate reports %q, manifest says %s", raw, release.Version)
	}
	digest, err := binaryfetch.FileDigestContext(ctx, executable)
	if err != nil {
		return err
	}
	data, err := json.Marshal(installationRecord{Resource: h.Resource, Package: h.Package, Version: release.Version, FileSHA256: digest})
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(candidate, "installation.json"), data, 0o600); err != nil {
		return err
	}
	if err := os.Chmod(candidate, deps.platform.permDir); err != nil {
		return err
	}
	return deps.platform.commitOwnedPath(candidate, destination)
}

// downloadVerifiedSHA512 keeps the download only when its whole body matches. A
// redirect is held to the same location rule as the URL it replaces.
func downloadVerifiedSHA512(ctx context.Context, base, source, destination, want string) error {
	return downloadVerifiedSHA512WithDeps(ctx, base, source, destination, want, defaultPlatformDeps())
}

func downloadVerifiedSHA512WithDeps(ctx context.Context, base, source, destination, want string, deps platformDeps) error {
	want = strings.ToLower(strings.TrimSpace(want))
	if want == "" {
		return fmt.Errorf("release manifest has no sha512 for %s", source)
	}
	if err := requireArtifactLocation(base, source); err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: deps.withDefaults().downloadTimeout, CheckRedirect: func(next *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return requireArtifactLocation(base, next.URL.String())
	}}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download %s: %w", source, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("download %s returned HTTP %d", source, response.StatusCode)
	}
	if response.ContentLength > releaseDownloadLimit {
		return fmt.Errorf("download %s is %d bytes, over the %d limit", source, response.ContentLength, releaseDownloadLimit)
	}
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	hash := sha512.New()
	written, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, releaseDownloadLimit+1))
	if err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if written > releaseDownloadLimit {
		return fmt.Errorf("download %s exceeds the %d byte limit", source, releaseDownloadLimit)
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != want {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", source, got, want)
	}
	return nil
}
