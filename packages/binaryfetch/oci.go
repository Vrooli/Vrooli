package binaryfetch

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// FetchOCI retrieves a digest-pinned OCI blob over HTTPS and extracts its tar
// layer into optDir. It never invokes Docker, skopeo, or another local daemon.
// The image reference is registry/repository@sha256:<digest>; an explicit
// http:// scheme is accepted for local test registries only.
func FetchOCI(ctx context.Context, target AcquisitionTarget, optDir string, onProgress ProgressFunc) (string, error) {
	return fetchOCIForPlatform(ctx, target, optDir, runtime.GOOS, runtime.GOARCH, onProgress)
}

// FetchOCIForPlatform extracts the platform member selected by a release
// stager. Runtime acquisition should use FetchOCI; cross-target release
// staging must pass the declared target platform explicitly rather than
// accidentally extracting the build host's image member.
func FetchOCIForPlatform(ctx context.Context, target AcquisitionTarget, optDir, goos, goarch string, onProgress ProgressFunc) (string, error) {
	return fetchOCIForPlatform(ctx, target, optDir, goos, goarch, onProgress)
}

func fetchOCIForPlatform(ctx context.Context, target AcquisitionTarget, optDir, goos, goarch string, onProgress ProgressFunc) (string, error) {
	registryURL, repository, digest, err := ociReference(target.Image)
	if err != nil {
		return "", err
	}
	tmpDir, err := os.MkdirTemp("", "binaryfetch-oci-*")
	if err != nil {
		return "", fmt.Errorf("binaryfetch: create OCI temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	layers, err := ociLayers(ctx, registryURL, repository, digest, goos, goarch)
	if err != nil {
		// Keep the local test registry and simple OCI-blob embedding contract
		// useful: a server that exposes only /blobs/<digest> is treated as one
		// already-selected uncompressed layer. Real registries must expose a
		// manifest/index and therefore take the authenticated path above.
		legacy := path.Join(tmpDir, "layer")
		legacyURL := strings.TrimRight(registryURL, "/") + "/blobs/sha256:" + digest
		if err := download(ctx, legacyURL, legacy, onProgress); err != nil {
			return "", fmt.Errorf("binaryfetch: resolve OCI image manifest: %w", err)
		}
		if err := verifyChecksum(legacy, digest); err != nil {
			return "", fmt.Errorf("binaryfetch: verify OCI blob: %w", err)
		}
		if err := extractTarLayer(legacy, optDir); err != nil {
			return "", err
		}
		return ociEntry(target, optDir)
	}
	for index, layer := range layers {
		layerPath := filepath.Join(tmpDir, fmt.Sprintf("layer-%03d", index))
		if err := downloadOCI(ctx, layer.URL, repository, layer.Digest, layerPath, onProgress); err != nil {
			return "", err
		}
		onProgress.emit(Progress{Stage: StageExtracting})
		if err := extractOCILayer(layerPath, layer.MediaType, optDir); err != nil {
			return "", err
		}
	}
	if err := removeDanglingOCISymlinks(optDir); err != nil {
		return "", err
	}
	return ociEntry(target, optDir)
}

// FetchOCIFile extracts one executable from an OCI image into destFile. It is
// used only for artifacts whose upstream image is itself the provenance source
// and whose runtime contract needs the verified executable rather than the
// complete container filesystem.
func FetchOCIFile(ctx context.Context, target AcquisitionTarget, destFile string, onProgress ProgressFunc) (string, error) {
	return fetchOCIFileForPlatform(ctx, target, destFile, runtime.GOOS, runtime.GOARCH, onProgress)
}

// FetchOCIFileForPlatform is the cross-target counterpart used by release
// staging when the selected OCI member is not the stager's own platform.
func FetchOCIFileForPlatform(ctx context.Context, target AcquisitionTarget, destFile, goos, goarch string, onProgress ProgressFunc) (string, error) {
	return fetchOCIFileForPlatform(ctx, target, destFile, goos, goarch, onProgress)
}

func fetchOCIFileForPlatform(ctx context.Context, target AcquisitionTarget, destFile, goos, goarch string, onProgress ProgressFunc) (string, error) {
	tmpDir, err := os.MkdirTemp("", "binaryfetch-oci-file-*")
	if err != nil {
		return "", fmt.Errorf("binaryfetch: create OCI temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	entry, err := fetchOCIForPlatform(ctx, target, tmpDir, goos, goarch, onProgress)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(destFile), 0o755); err != nil {
		return "", err
	}
	input, err := os.Open(entry)
	if err != nil {
		return "", err
	}
	defer input.Close()
	output, err := os.OpenFile(destFile+".tmp", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return "", err
	}
	if err := output.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(destFile+".tmp", destFile); err != nil {
		return "", err
	}
	return destFile, nil
}

type ociLayer struct {
	URL       string
	Digest    string
	MediaType string
}

type ociManifest struct {
	Layers []struct {
		Digest    string `json:"digest"`
		MediaType string `json:"mediaType"`
	} `json:"layers"`
}

type ociIndex struct {
	Manifests []struct {
		Digest   string `json:"digest"`
		Platform struct {
			OS           string `json:"os"`
			Architecture string `json:"architecture"`
		} `json:"platform"`
	} `json:"manifests"`
}

func ociLayers(ctx context.Context, registryURL, repository, digest, goos, goarch string) ([]ociLayer, error) {
	manifestURL := strings.TrimRight(registryURL, "/") + "/manifests/sha256:" + digest
	body, mediaType, err := fetchOCIJSON(ctx, manifestURL, repository)
	if err != nil {
		return nil, err
	}
	if strings.Contains(mediaType, "index") || strings.Contains(mediaType, "manifest.list") {
		var index ociIndex
		if err := json.Unmarshal(body, &index); err != nil {
			return nil, fmt.Errorf("binaryfetch: parse OCI index: %w", err)
		}
		wantOS, wantArch := strings.ToLower(strings.TrimSpace(goos)), strings.ToLower(strings.TrimSpace(goarch))
		if wantOS == "darwin" {
			wantOS = "darwin"
		}
		selected := ""
		for _, item := range index.Manifests {
			if item.Platform.OS == wantOS && item.Platform.Architecture == wantArch {
				selected = item.Digest
				break
			}
		}
		if selected == "" {
			return nil, fmt.Errorf("binaryfetch: OCI image has no manifest for %s/%s", wantOS, wantArch)
		}
		body, mediaType, err = fetchOCIJSON(ctx, strings.TrimRight(registryURL, "/")+"/manifests/"+selected, repository)
		if err != nil {
			return nil, err
		}
	}
	if !strings.Contains(mediaType, "manifest") {
		return nil, fmt.Errorf("binaryfetch: OCI reference returned unsupported media type %q", mediaType)
	}
	var manifest ociManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("binaryfetch: parse OCI manifest: %w", err)
	}
	layers := make([]ociLayer, 0, len(manifest.Layers))
	for _, layer := range manifest.Layers {
		if !strings.HasPrefix(layer.Digest, "sha256:") {
			return nil, fmt.Errorf("binaryfetch: OCI layer digest is not sha256: %q", layer.Digest)
		}
		layers = append(layers, ociLayer{
			URL:    strings.TrimRight(registryURL, "/") + "/blobs/" + layer.Digest,
			Digest: strings.TrimPrefix(layer.Digest, "sha256:"), MediaType: layer.MediaType,
		})
	}
	return layers, nil
}

func fetchOCIJSON(ctx context.Context, endpoint, repository string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json")
	if !strings.HasPrefix(endpoint, "http://127.0.0.1") && !strings.HasPrefix(endpoint, "http://localhost") {
		if token, tokenErr := ociBearerToken(ctx, endpoint, repository); tokenErr == nil {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	response, err := HTTPClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("OCI manifest request returned %s", response.Status)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", err
	}
	return body, response.Header.Get("Content-Type"), nil
}

func ociBearerToken(ctx context.Context, endpoint, repository string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	authHost := parsed.Host
	service := parsed.Host
	if parsed.Host == "registry-1.docker.io" {
		authHost = "auth.docker.io"
		service = "registry.docker.io"
	}
	tokenURL := "https://" + authHost + "/token?service=" + url.QueryEscape(service) + "&scope=repository:" + url.QueryEscape(repository) + ":pull"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", err
	}
	response, err := HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OCI token request returned %s", response.Status)
	}
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.Token, nil
}

func downloadOCI(ctx context.Context, endpoint, repository, digest, destination string, onProgress ProgressFunc) error {
	if strings.TrimSpace(digest) == "" {
		return fmt.Errorf("binaryfetch: OCI layer digest is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(endpoint, "http://127.0.0.1") && !strings.HasPrefix(endpoint, "http://localhost") {
		if token, tokenErr := ociBearerToken(ctx, endpoint, repository); tokenErr == nil {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	response, err := HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("OCI layer request returned %s", response.Status)
	}
	output, err := os.Create(destination)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, response.Body); err != nil {
		output.Close()
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	if err := verifyChecksum(destination, strings.TrimPrefix(digest, "sha256:")); err != nil {
		return fmt.Errorf("binaryfetch: verify OCI layer: %w", err)
	}
	return nil
}

func extractOCILayer(archivePath, mediaType, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	var reader io.Reader = file
	if strings.Contains(mediaType, "gzip") || strings.HasSuffix(archivePath, ".gz") {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}
	return applyOCITar(tar.NewReader(reader), destDir)
}

func applyOCITar(reader *tar.Reader, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("binaryfetch: read OCI layer: %w", err)
		}
		if header.Name == "." || header.Name == "./" || header.Name == "./"+string(filepath.Separator) {
			continue
		}
		name := filepath.Clean(filepath.FromSlash(strings.TrimPrefix(header.Name, "./")))
		if name == "." || filepath.IsAbs(name) || strings.HasPrefix(name, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("binaryfetch: OCI layer path escapes root: %q", header.Name)
		}
		// Container images carry pseudo-filesystem mount points that are not
		// executable artifact bytes. Materializing them would either create
		// host-sensitive paths or leave deliberately dangling links such as
		// /etc/mtab -> /proc/mounts in a tree that must be self-contained.
		first := strings.Split(filepath.ToSlash(name), "/")[0]
		linkTarget := filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(name), filepath.FromSlash(header.Linkname))))
		if first == "proc" || first == "sys" || first == "dev" || first == "run" || (header.Typeflag == tar.TypeSymlink && (strings.HasPrefix(header.Linkname, "/proc/") || linkTarget == "proc" || strings.HasPrefix(linkTarget, "proc/"))) {
			continue
		}
		if strings.HasPrefix(filepath.Base(name), ".wh.") {
			target := strings.TrimPrefix(filepath.Base(name), ".wh.")
			if target == ".opq" {
				entries, _ := os.ReadDir(filepath.Join(destDir, filepath.Dir(name)))
				for _, entry := range entries {
					_ = os.RemoveAll(filepath.Join(destDir, filepath.Dir(name), entry.Name()))
				}
			} else {
				_ = os.RemoveAll(filepath.Join(destDir, filepath.Dir(name), target))
			}
			continue
		}
		path := filepath.Join(destDir, name)
		switch header.Typeflag {
		case tar.TypeDir:
			// OCI layers are commonly authored with root-only directory modes.
			// The artifact is extracted into a user-owned store, so preserve the
			// declared mode while ensuring the extracting owner can traverse it.
			mode := os.FileMode(header.Mode)&0o777 | 0o700
			if err := os.MkdirAll(path, mode); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			// The extraction process owns the staged artifact. Root-owned images
			// may mark configuration files 000, which would make verification
			// fail before the managed-service driver can launch them. Keep the
			// upstream permission bits and grant the extracting owner read/write.
			mode := os.FileMode(header.Mode)&0o777 | 0o600
			if err := writeReaderMode(reader, path, mode); err != nil {
				return err
			}
		case tar.TypeSymlink:
			linkname := header.Linkname
			if filepath.IsAbs(filepath.FromSlash(linkname)) {
				// OCI images use absolute links relative to the image root. Keep
				// that meaning without creating a host-root link in the extracted
				// artifact tree.
				imageTarget := filepath.Join(destDir, filepath.FromSlash(strings.TrimPrefix(linkname, "/")))
				relative, relErr := filepath.Rel(filepath.Dir(path), imageTarget)
				if relErr != nil {
					return fmt.Errorf("binaryfetch: OCI symlink %q escapes image root", header.Name)
				}
				linkname = filepath.ToSlash(relative)
			}
			if err := createSafeSymlink(destDir, name, linkname, path); err != nil {
				return err
			}
		case tar.TypeLink:
			linkTarget := filepath.Join(destDir, filepath.Clean(filepath.FromSlash(header.Linkname)))
			if rel, err := filepath.Rel(destDir, linkTarget); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
				return fmt.Errorf("binaryfetch: OCI hard link escapes root: %q", header.Linkname)
			}
			// OCI layers may emit a regular placeholder before the canonical
			// hardlink name. Replace it so extraction remains faithful to the
			// layer instead of failing with EEXIST.
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("binaryfetch: replace OCI hard link target %q: %w", header.Name, err)
			}
			if err := os.Link(linkTarget, path); err != nil {
				return fmt.Errorf("binaryfetch: create OCI hard link %q: %w", header.Name, err)
			}
		default:
			return fmt.Errorf("binaryfetch: unsupported OCI entry type %d for %q", header.Typeflag, header.Name)
		}
	}
}

func removeDanglingOCISymlinks(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink == 0 {
			return nil
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return os.Remove(path)
		} else if err != nil {
			return fmt.Errorf("binaryfetch: inspect OCI symlink %q: %w", path, err)
		}
		return nil
	})
}

func ociEntry(target AcquisitionTarget, root string) (string, error) {
	entry := strings.TrimSpace(target.BinPath)
	if entry == "" {
		return "", fmt.Errorf("binaryfetch: OCI target bin_path is required")
	}
	entryPath := filepathJoinWithin(root, strings.TrimPrefix(entry, string(os.PathSeparator)))
	if entryPath == "" {
		return "", fmt.Errorf("binaryfetch: OCI bin_path %q escapes the layer root", entry)
	}
	if info, err := os.Stat(entryPath); err != nil || info.IsDir() {
		return "", fmt.Errorf("%w: OCI entry %q not found", ErrNoBinaryInArchive, entry)
	}
	if err := ValidateNotHTMLAndSized(entryPath, path.Base(entry), 0); err != nil {
		return "", err
	}
	return entryPath, nil
}

func ociReference(image string) (string, string, string, error) {
	image = strings.TrimSpace(image)
	if image == "" {
		return "", "", "", fmt.Errorf("binaryfetch: OCI image reference is required")
	}
	parts := strings.SplitN(image, "@", 2)
	if len(parts) != 2 || !strings.HasPrefix(parts[1], "sha256:") {
		return "", "", "", fmt.Errorf("binaryfetch: OCI image must be pinned by sha256 digest")
	}
	digest := strings.TrimPrefix(parts[1], "sha256:")
	if err := validateDigest(digest); err != nil {
		return "", "", "", fmt.Errorf("binaryfetch: OCI image digest: %w", err)
	}
	ref := parts[0]
	scheme := "https"
	if strings.Contains(ref, "://") {
		parsed, err := url.Parse(ref)
		if err != nil || parsed.Host == "" {
			return "", "", "", fmt.Errorf("binaryfetch: invalid OCI registry reference %q", image)
		}
		scheme = parsed.Scheme
		ref = strings.TrimPrefix(parsed.Host+parsed.Path, "/")
	}
	registry, repository, ok := strings.Cut(ref, "/")
	if !ok {
		registry = "registry-1.docker.io"
		repository = "library/" + ref
	} else if registry == "library" {
		// The conventional Docker Hub library namespace is accepted without
		// spelling the registry in every resource manifest.
		registry = "registry-1.docker.io"
		repository = "library/" + repository
	}
	if registry == "" || repository == "" {
		return "", "", "", fmt.Errorf("binaryfetch: OCI image must include a registry or a default library image")
	}
	if registry == "docker.io" {
		registry = "registry-1.docker.io"
	}
	return fmt.Sprintf("%s://%s/v2/%s", scheme, registry, repository), repository, digest, nil
}

func filepathJoinWithin(root, relative string) string {
	clean := filepath.Clean(filepath.FromSlash(relative))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return ""
	}
	joined := filepath.Join(root, clean)
	if joined != filepath.Clean(root) && !strings.HasPrefix(joined, filepath.Clean(root)+string(os.PathSeparator)) {
		return ""
	}
	return joined
}
