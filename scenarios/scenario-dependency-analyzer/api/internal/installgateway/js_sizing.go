package installgateway

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Sizing a staged pnpm install: pnpm already resolved the closure into the
// staged lockfile, so admission only has to size the packages that lock adds
// and node_modules does not already hold. Registry packages are sized from
// their published metadata, a tarball URL dependency is downloaded once into
// memory so its bytes are checked against the lock's integrity, and a local
// `file:` package is measured on disk.

const (
	maxPnpmVersionDocument = 2 << 20
	maxPnpmMetadataBudget  = 128 << 20
	maxPnpmRegistryCalls   = 8192
	maxPnpmArchiveProbe    = 32 << 20
	pnpmSizingWorkers      = 8
	// Registry records that omit dist.unpackedSize are sized from the archive.
	// Real packages expand a few times; the factor leaves wide margin and the
	// lease's observed-growth monitor still protects the filesystem floor.
	pnpmUnpackedFactor = 16
	// Staged files are written twice (temporary copy, then rename) and the
	// previous inputs are retained for rollback.
	jsInputAllowanceBytes = 1 << 20
)

// JSMeasurement is what admission sized for a staged install.
type JSMeasurement struct {
	PackageCount int
	PackageBytes int64
	InputBytes   int64
	ProbeBytes   int64
}

type artifactSize struct {
	Unpacked int64
	Archive  int64
	// Integrity is the sha512 of a tarball downloaded without a recorded
	// integrity, so a lock can pin what was measured.
	Integrity string
}

// registrySizer sizes one locked artifact. The production implementation talks to
// the public registry; tests substitute a fake.
type registrySizer interface {
	Registry(ctx context.Context, name, version, integrity string) (artifactSize, error)
	Tarball(ctx context.Context, rawURL, integrity string) (artifactSize, error)
}

var newRegistrySizer = func() (registrySizer, error) { return newRegistryHTTP() }

// measure fills s.Measured for the resolved inputs.
// runSizingJobs runs count jobs on a small worker pool and returns the first
// error, cancelling the rest.
func runSizingJobs(ctx context.Context, count int, run func(context.Context, int) error) error {
	if count == 0 {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	next := make(chan int)
	var once sync.Once
	var firstErr error
	var wg sync.WaitGroup
	for w := 0; w < pnpmSizingWorkers && w < count; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				if err := run(ctx, i); err != nil {
					once.Do(func() { firstErr = err; cancel() })
				}
			}
		}()
	}
	for i := 0; i < count; i++ {
		next <- i
	}
	close(next)
	wg.Wait()
	return firstErr
}

// measureLocalPnpmPackage validates a repository-relative file: specifier and
// measures the package's regular source files without following symlinks or
// counting node_modules.
func measureLocalPnpmPackage(repositoryRoot, baseRoot, expectedName, specifier string) (int64, error) {
	rootPath, err := verifyLocalPnpmPackage(repositoryRoot, baseRoot, expectedName, specifier)
	if err != nil {
		return 0, err
	}
	return localPnpmPackageSize(rootPath)
}

// verifyLocalPnpmPackage proves that a file: specifier names a first-party
// workspace package and returns its directory. The package must be a real,
// non-linked directory beneath the repository's packages directory and must
// carry the expected name and an exact semantic version.
func verifyLocalPnpmPackage(repositoryRoot, baseRoot, expectedName, specifier string) (string, error) {
	if repositoryRoot == "" || baseRoot == "" || !strings.HasPrefix(specifier, "file:") {
		return "", fmt.Errorf("local pnpm package requires a repository-relative file: specifier")
	}
	relative := strings.TrimPrefix(specifier, "file:")
	if relative == "" || filepath.IsAbs(relative) || strings.ContainsAny(relative, "?#\\\x00\r\n") {
		return "", fmt.Errorf("local pnpm package path must be a relative file: path")
	}
	rootPath := filepath.Clean(filepath.Join(baseRoot, filepath.FromSlash(relative)))
	resolvedRepo, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return "", fmt.Errorf("resolve repository root for local pnpm package: %w", err)
	}
	resolvedPackages, err := filepath.EvalSymlinks(filepath.Join(resolvedRepo, "packages"))
	if err != nil {
		return "", fmt.Errorf("resolve repository packages directory: %w", err)
	}
	within := func(path string) bool {
		rel, err := filepath.Rel(resolvedPackages, path)
		return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	}
	if !within(rootPath) {
		return "", fmt.Errorf("local pnpm package %s must resolve beneath the repository packages directory", specifier)
	}
	resolvedRoot, err := filepath.EvalSymlinks(rootPath)
	if err != nil {
		return "", fmt.Errorf("resolve local pnpm package: %w", err)
	}
	rootInfo, err := os.Lstat(rootPath)
	if err != nil || !within(resolvedRoot) || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 || filepath.Clean(resolvedRoot) != filepath.Clean(rootPath) {
		return "", fmt.Errorf("local pnpm package root must be a real, non-linked directory beneath the repository packages directory")
	}
	manifestBytes, err := readRegularBounded(filepath.Join(rootPath, "package.json"), maxPnpmManifestBytes)
	if err != nil {
		return "", fmt.Errorf("read local pnpm package manifest: %w", err)
	}
	var manifest struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil || manifest.Name != expectedName || !exactNpmVersion.MatchString(manifest.Version) {
		return "", fmt.Errorf("local pnpm package at %s must be named %q with an exact semantic version", specifier, expectedName)
	}
	return rootPath, nil
}

func localPnpmPackageSize(root string) (int64, error) {
	var size int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path != root && entry.IsDir() && (entry.Name() == "node_modules" || entry.Name() == ".git") {
			return filepath.SkipDir
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("local pnpm package contains a symlink at %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("local pnpm package contains a non-regular file at %s", path)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() <= 0 {
			return nil
		}
		if info.Size() > DiskJobPlanningCeilingBytes/3-size {
			return fmt.Errorf("local pnpm package exceeds its bounded source size")
		}
		size += info.Size()
		return nil
	})
	if err != nil {
		return 0, err
	}
	if size <= 0 {
		return 0, fmt.Errorf("local pnpm package contains no measurable regular files")
	}
	return size, nil
}

// registryHTTP sizes artifacts from the public npm registry under a shared
// request and byte budget.
type registryHTTP struct {
	base   string
	client *http.Client

	mu        sync.Mutex
	usedBytes int64
	requests  int
}

func newRegistryHTTP() (*registryHTTP, error) {
	if err := validatePnpmRegistryEnvironment(); err != nil {
		return nil, err
	}
	base := strings.TrimSpace(os.Getenv("npm_config_registry"))
	if base == "" {
		base = strings.TrimSpace(os.Getenv("NPM_CONFIG_REGISTRY"))
	}
	if base == "" {
		base = "https://registry.npmjs.org"
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "registry.npmjs.org" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("package registry is not the approved public HTTPS registry; refusing an unbounded pnpm size probe")
	}
	return &registryHTTP{base: strings.TrimRight(base, "/"), client: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func validatePnpmRegistryEnvironment() error {
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		lower := strings.ToLower(key)
		if !ok || !strings.HasPrefix(lower, "npm_config_") || !strings.Contains(lower, "registry") {
			continue
		}
		parsed, err := url.Parse(strings.TrimSpace(value))
		if err != nil || parsed.Scheme != "https" || parsed.Host != "registry.npmjs.org" || parsed.User != nil {
			return fmt.Errorf("custom npm registry environment setting cannot be covered by the approved bounded pnpm size probe")
		}
	}
	return nil
}

func (r *registryHTTP) beginRequest() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.requests >= maxPnpmRegistryCalls {
		return fmt.Errorf("bounded pnpm registry request budget (%d) exceeded", maxPnpmRegistryCalls)
	}
	r.requests++
	return nil
}

func (r *registryHTTP) recordResponse(size int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if size < 0 || int64(size) > maxPnpmMetadataBudget-r.usedBytes {
		return fmt.Errorf("bounded pnpm registry byte budget (%d bytes) exceeded", int64(maxPnpmMetadataBudget))
	}
	r.usedBytes += int64(size)
	return nil
}

type pnpmVersionDocument struct {
	Name                 string            `json:"name"`
	Version              string            `json:"version"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
	Dist                 struct {
		UnpackedSize int64  `json:"unpackedSize"`
		Tarball      string `json:"tarball"`
		Integrity    string `json:"integrity"`
	} `json:"dist"`
}

func (r *registryHTTP) Registry(ctx context.Context, name, version, integrity string) (artifactSize, error) {
	doc, err := r.versionDocument(ctx, name, version)
	if err != nil {
		return artifactSize{}, err
	}
	if doc.Dist.Integrity != "" && integrity != "" && doc.Dist.Integrity != integrity {
		return artifactSize{}, fmt.Errorf("registry integrity %s differs from the locked integrity", doc.Dist.Integrity)
	}
	archive, err := probePnpmArchiveSize(ctx, r.client, doc.Dist.Tarball, "registry.npmjs.org", r.beginRequest, r.recordResponse)
	if err != nil {
		return artifactSize{}, err
	}
	unpacked := doc.Dist.UnpackedSize
	if unpacked <= 0 {
		unpacked = archive * pnpmUnpackedFactor
	}
	if unpacked <= 0 || unpacked > DiskJobPlanningCeilingBytes/3 {
		return artifactSize{}, fmt.Errorf("unpacked size %d is unavailable or exceeds the bounded package limit", unpacked)
	}
	return artifactSize{Unpacked: unpacked, Archive: archive}, nil
}

func (r *registryHTTP) OptionalDependencies(ctx context.Context, name, version string) (map[string]string, error) {
	doc, err := r.versionDocument(ctx, name, version)
	if err != nil {
		return nil, err
	}
	deps := make(map[string]string, len(doc.OptionalDependencies))
	for dependency, pinned := range doc.OptionalDependencies {
		deps[dependency] = pinned
	}
	return deps, nil
}

func (r *registryHTTP) versionDocument(ctx context.Context, name, version string) (pnpmVersionDocument, error) {
	if err := r.beginRequest(); err != nil {
		return pnpmVersionDocument{}, err
	}
	probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(probeCtx, http.MethodGet, r.base+"/"+url.PathEscape(name)+"/"+url.PathEscape(version), nil)
	if err != nil {
		return pnpmVersionDocument{}, err
	}
	// The compact install-v1 document omits dist.unpackedSize for some
	// packages, so ask for the full version document.
	request.Header.Set("Accept", "application/json")
	response, err := r.client.Do(request)
	if err != nil {
		return pnpmVersionDocument{}, fmt.Errorf("registry metadata request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return pnpmVersionDocument{}, fmt.Errorf("registry metadata request returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxPnpmVersionDocument+1))
	if err != nil || len(body) == 0 || len(body) > maxPnpmVersionDocument {
		return pnpmVersionDocument{}, fmt.Errorf("registry metadata exceeded its %d-byte limit", maxPnpmVersionDocument)
	}
	if err := r.recordResponse(len(body)); err != nil {
		return pnpmVersionDocument{}, err
	}
	var doc pnpmVersionDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return pnpmVersionDocument{}, fmt.Errorf("decode registry metadata: %w", err)
	}
	if doc.Name != name || doc.Version != version {
		return pnpmVersionDocument{}, fmt.Errorf("registry metadata describes %s@%s, not the locked package", doc.Name, doc.Version)
	}
	return doc, nil
}

// Tarball downloads a tarball URL dependency into memory, checks the bytes
// against the lock's sha512 integrity, and measures the unpacked size. The
// download is bounded and never staged on disk.
// Tarball downloads a tarball URL dependency once into memory and sizes it. With
// an integrity the bytes must match it; without one the digest is computed so
// the lock can record it.
func (r *registryHTTP) Tarball(ctx context.Context, rawURL, integrity string) (artifactSize, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return artifactSize{}, fmt.Errorf("tarball URL must be an https URL")
	}
	if err := r.beginRequest(); err != nil {
		return artifactSize{}, err
	}
	archive, err := probePnpmArchiveContent(ctx, r.client, rawURL, parsed.Host, integrity)
	if err != nil {
		return artifactSize{}, err
	}
	if err := r.recordResponse(len(archive)); err != nil {
		return artifactSize{}, err
	}
	unpacked, err := measurePnpmArchive(archive)
	if err != nil {
		return artifactSize{}, err
	}
	size := artifactSize{Unpacked: unpacked, Archive: int64(len(archive))}
	if integrity == "" {
		size.Integrity = pnpmIntegrity(archive)
	}
	return size, nil
}

// pnpmIntegrity formats the lockfile integrity string for archive bytes.
func pnpmIntegrity(archive []byte) string {
	digest := sha512.Sum512(archive)
	return "sha512-" + base64.StdEncoding.EncodeToString(digest[:])
}

func probePnpmArchiveContent(ctx context.Context, client *http.Client, tarball, approvedHost, integrity string) ([]byte, error) {
	current, err := validatePnpmArchiveURL(tarball, approvedHost)
	if err != nil {
		return nil, err
	}
	probeCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	for redirects := 0; ; redirects++ {
		request, err := http.NewRequestWithContext(probeCtx, http.MethodGet, current.String(), nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Accept-Encoding", "identity")
		response, err := client.Do(request)
		if err != nil {
			return nil, fmt.Errorf("bounded pnpm archive content probe failed: %w", err)
		}
		if isRedirectStatus(response.StatusCode) {
			location := response.Header.Get("Location")
			response.Body.Close()
			if redirects >= 2 || location == "" {
				return nil, fmt.Errorf("pnpm archive content redirect chain exceeds the bounded limit")
			}
			next, err := current.Parse(location)
			if err != nil {
				return nil, fmt.Errorf("pnpm archive content redirect URL is invalid")
			}
			current, err = validatePnpmArchiveURL(next.String(), approvedHost)
			if err != nil {
				return nil, fmt.Errorf("pnpm archive content redirected outside the approved HTTPS host")
			}
			continue
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("pnpm archive content probe returned HTTP %d", response.StatusCode)
		}
		if response.ContentLength > maxPnpmArchiveProbe {
			return nil, fmt.Errorf("pnpm archive content exceeds the bounded in-memory probe limit")
		}
		archive, err := io.ReadAll(io.LimitReader(response.Body, maxPnpmArchiveProbe+1))
		if err != nil || len(archive) == 0 || len(archive) > maxPnpmArchiveProbe {
			return nil, fmt.Errorf("pnpm archive content exceeds the bounded in-memory probe limit")
		}
		if response.ContentLength >= 0 && int64(len(archive)) != response.ContentLength {
			return nil, fmt.Errorf("pnpm archive content length changed during the bounded probe")
		}
		if integrity != "" && !verifyPnpmIntegrity(archive, integrity) {
			return nil, fmt.Errorf("pnpm archive content does not match the lock's integrity; the artifact at this URL changed or the lock was altered")
		}
		return archive, nil
	}
}

func verifyPnpmIntegrity(archive []byte, integrity string) bool {
	for _, token := range strings.Fields(integrity) {
		algorithm, expected, ok := strings.Cut(token, "-")
		if !ok || algorithm != "sha512" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(expected)
		if err != nil {
			continue
		}
		digest := sha512.Sum512(archive)
		return bytes.Equal(decoded, digest[:])
	}
	return false
}

func measurePnpmArchive(archive []byte) (int64, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return 0, fmt.Errorf("decode bounded pnpm gzip archive: %w", err)
	}
	defer gz.Close()
	limited := &pnpmArchiveCounter{r: io.LimitReader(gz, DiskJobPlanningCeilingBytes/3+1)}
	tr := tar.NewReader(limited)
	var unpacked int64
	for entries := 0; ; entries++ {
		if entries >= 100_000 {
			return 0, fmt.Errorf("pnpm archive exceeds the bounded file count")
		}
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("read bounded pnpm tar archive: %w", err)
		}
		if header.Size < 0 || header.Size > DiskJobPlanningCeilingBytes/3-unpacked {
			return 0, fmt.Errorf("pnpm archive unpacked size exceeds the bounded package limit")
		}
		if header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA {
			unpacked += header.Size
		}
		if _, err := io.Copy(io.Discard, tr); err != nil {
			return 0, fmt.Errorf("read bounded pnpm tar member: %w", err)
		}
		if limited.n > DiskJobPlanningCeilingBytes/3 {
			return 0, fmt.Errorf("pnpm archive decompressed data exceeds the bounded package limit")
		}
	}
	if unpacked <= 0 {
		return 0, fmt.Errorf("pnpm archive has no measurable unpacked files")
	}
	return unpacked, nil
}

type pnpmArchiveCounter struct {
	r io.Reader
	n int64
}

func (c *pnpmArchiveCounter) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

func probePnpmArchiveSize(ctx context.Context, client *http.Client, tarball, approvedHost string, beginRequest func() error, recordResponse func(int) error) (int64, error) {
	parsed, err := validatePnpmArchiveURL(tarball, approvedHost)
	if err != nil {
		return 0, err
	}
	probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	requestArchive := func(method string) (*http.Response, error) {
		current := parsed
		for redirects := 0; ; redirects++ {
			if err := beginRequest(); err != nil {
				return nil, err
			}
			request, err := http.NewRequestWithContext(probeCtx, method, current.String(), nil)
			if err != nil {
				return nil, err
			}
			if method == http.MethodGet {
				request.Header.Set("Range", "bytes=0-0")
			}
			response, err := client.Do(request)
			if err != nil {
				return nil, fmt.Errorf("bounded pnpm archive-size probe failed: %w", err)
			}
			if !isRedirectStatus(response.StatusCode) {
				return response, nil
			}
			location := response.Header.Get("Location")
			response.Body.Close()
			if redirects >= 2 || location == "" {
				return nil, fmt.Errorf("pnpm package tarball redirect chain exceeds the bounded limit")
			}
			next, err := current.Parse(location)
			if err != nil {
				return nil, fmt.Errorf("pnpm package tarball redirect URL is invalid")
			}
			if _, err := validatePnpmArchiveURL(next.String(), approvedHost); err != nil {
				return nil, fmt.Errorf("pnpm package tarball redirected outside the approved HTTPS registry")
			}
			current = next
		}
	}

	head, err := requestArchive(http.MethodHead)
	if err != nil {
		return 0, err
	}
	if head.StatusCode >= 200 && head.StatusCode < 300 && head.ContentLength > 0 && head.ContentLength <= DiskJobPlanningCeilingBytes/3 {
		head.Body.Close()
		return head.ContentLength, nil
	}
	head.Body.Close()

	response, err := requestArchive(http.MethodGet)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusPartialContent {
		return 0, fmt.Errorf("pnpm package tarball server did not honor the bounded one-byte range request")
	}
	match := pnpmContentRange.FindStringSubmatch(strings.TrimSpace(response.Header.Get("Content-Range")))
	if match == nil {
		return 0, fmt.Errorf("pnpm package tarball returned an invalid bounded Content-Range")
	}
	total, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil || total <= 0 || total > DiskJobPlanningCeilingBytes/3 {
		return 0, fmt.Errorf("pnpm package tarball size is unavailable or exceeds the bounded probe limit")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 2))
	if err != nil || len(body) != 1 || (response.ContentLength >= 0 && response.ContentLength != 1) {
		return 0, fmt.Errorf("pnpm package tarball exceeded the bounded one-byte range response")
	}
	if recordResponse != nil {
		if err := recordResponse(len(body)); err != nil {
			return 0, err
		}
	}
	return total, nil
}

var pnpmContentRange = regexp.MustCompile(`^bytes 0-0/([0-9]+)$`)

func validatePnpmArchiveURL(raw, approvedHost string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Host, approvedHost) || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("pnpm package tarball is outside the approved HTTPS host")
	}
	return parsed, nil
}

func isRedirectStatus(status int) bool {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

func addBoundedBytes(current, addition, limit int64) (int64, error) {
	if addition < 0 || current < 0 || current > limit-addition {
		return 0, fmt.Errorf("size exceeds %d-byte bound", limit)
	}
	return current + addition, nil
}
