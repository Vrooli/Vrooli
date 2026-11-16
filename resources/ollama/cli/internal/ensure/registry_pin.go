package ensure

// This file implements a one-pull, loopback registry view over an already
// fetched Ollama manifest. The manifest bytes are frozen at construction;
// blobs remain content-addressed and are redirected to the canonical registry.
// The caller is responsible for obtaining the manifest from the trusted
// registry before opening this bridge.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	RegistryHost     = "registry.ollama.ai"
	maxManifestBytes = 4 << 20
)

type descriptor struct {
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}
type manifest struct {
	SchemaVersion int          `json:"schemaVersion"`
	Config        descriptor   `json:"config"`
	Layers        []descriptor `json:"layers"`
}

// Pin is a temporary registry exposing precisely one validated manifest.
// Target is the Ollama model argument to use with Pull(insecure=true).
type Pin struct {
	Target    string
	URL       string
	Repo      string
	Tag       string
	PeakBytes int64
	server    *http.Server
	listener  net.Listener
	once      sync.Once
}

// PreflightClient retrieves a manifest from the default public registry. Its
// implementation must use HTTPS and must not silently fall back to another
// registry or credential source.
type PreflightClient interface {
	FetchManifest(ctx context.Context, repository, tag string) ([]byte, error)
	ValidateBlobSize(ctx context.Context, repository, digest string, expectedSize int64) error
}

// PinnedPull is the caller-facing immutable pull plan.
type PinnedPull struct {
	pin        *Pin
	digest     string
	modelBytes int64
	layers     []string
	manifest   []byte
	repository string
	tag        string
}

// PreparePinnedPull resolves a plain Ollama model reference against the default
// registry, validates the returned manifest, and installs its frozen loopback
// view. Accepted refs are model[:tag] and library/model[:tag], with optional
// explicit registry.ollama.ai/ prefix. Credentials and alternate registries
// are intentionally unsupported.
func PreparePinnedPull(ctx context.Context, originalRef string, preflight PreflightClient) (*PinnedPull, error) {
	if preflight == nil {
		return nil, errors.New("pinned pull requires a registry preflight client")
	}
	repo, tag, err := parseOriginalRef(originalRef)
	if err != nil {
		return nil, err
	}
	manifestBytes, err := preflight.FetchManifest(ctx, repo, tag)
	if err != nil {
		return nil, fmt.Errorf("fetch default registry manifest: %w", err)
	}
	pin, err := OpenRegistryPin(repo, tag, manifestBytes)
	if err != nil {
		return nil, err
	}
	var m manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		_ = pin.Close(context.Background())
		return nil, err
	}
	// Ollama fetches the config descriptor and each layer into the same blob
	// store. Include config in cleanup inspection as well as every layer.
	layers := make([]string, 0, len(m.Layers)+1)
	layers = append(layers, m.Config.Digest)
	var modelBytes int64
	for _, d := range append([]descriptor{m.Config}, m.Layers...) {
		if err := preflight.ValidateBlobSize(ctx, repo, d.Digest, d.Size); err != nil {
			_ = pin.Close(context.Background())
			return nil, fmt.Errorf("validate pinned blob size %s: %w", d.Digest, err)
		}
	}
	for _, d := range m.Layers {
		layers = append(layers, d.Digest)
		if d.Size > math.MaxInt64-modelBytes {
			_ = pin.Close(context.Background())
			return nil, errors.New("pinned model size overflows")
		}
		modelBytes += d.Size
	}
	return &PinnedPull{pin: pin, digest: "sha256:" + sha256Hex(manifestBytes), modelBytes: modelBytes, layers: layers, manifest: append([]byte(nil), manifestBytes...), repository: repo, tag: tag}, nil
}

func parseOriginalRef(ref string) (string, string, error) {
	if strings.HasPrefix(ref, "registry.ollama.ai/") {
		ref = strings.TrimPrefix(ref, "registry.ollama.ai/")
	}
	if strings.Contains(ref, "://") || strings.Contains(ref, "@") || strings.Contains(ref, "\\") {
		return "", "", errors.New("unsupported pinned model reference")
	}
	parts := strings.Split(ref, ":")
	if len(parts) > 2 {
		return "", "", errors.New("unsupported pinned model reference")
	}
	name, tag := parts[0], "latest"
	if len(parts) == 2 {
		tag = parts[1]
	}
	if !strings.Contains(name, "/") {
		name = "library/" + name
	}
	if !validRepo(name) || !validSegment(tag) {
		return "", "", errors.New("unsupported pinned model reference")
	}
	return name, tag, nil
}

func normalizedDefaultModelRef(ref string) string {
	repo, tag, err := parseOriginalRef(ref)
	if err != nil {
		return withLatestTag(ref)
	}
	return strings.TrimPrefix(repo, "library/") + ":" + tag
}

func modelReferenceInstalled(tags map[string]bool, ref string) bool {
	if tags[ref] || tags[withLatestTag(ref)] {
		return true
	}
	canonical := normalizedDefaultModelRef(ref)
	return tags[canonical] || tags[withLatestTag(canonical)]
}

func (p *PinnedPull) PullReference() string {
	if p == nil || p.pin == nil {
		return ""
	}
	return strings.TrimPrefix(p.pin.Target, "http://")
}
func (p *PinnedPull) PeakBytes() int64 {
	if p == nil || p.pin == nil {
		return 0
	}
	return p.pin.PeakBytes
}
func (p *PinnedPull) ManifestDigest() string {
	if p == nil {
		return ""
	}
	return p.digest
}
func (p *PinnedPull) ModelBytes() int64 {
	if p == nil {
		return 0
	}
	return p.modelBytes
}
func (p *PinnedPull) LayerDigests() []string {
	if p == nil {
		return nil
	}
	return append([]string(nil), p.layers...)
}

// InstallTarget atomically publishes the pinned manifest only if the requested
// tag is still absent. Ollama's /api/copy uses os.Create and can overwrite a
// tag installed concurrently, so the ensure path uses a same-filesystem hard
// link as its no-clobber commit point.
func (p *PinnedPull) InstallTarget(ctx context.Context, modelRoot, targetRef string) error {
	if p == nil || len(p.manifest) == 0 || !validRepo(p.repository) || !validSegment(p.tag) {
		return errors.New("pinned manifest target is unavailable")
	}
	repo, tag, err := parseOriginalRef(targetRef)
	if err != nil || repo != p.repository || tag != p.tag {
		return errors.New("requested install tag does not match the pinned manifest")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	parts := strings.Split(p.repository, "/")
	parts = append([]string{modelRoot, "manifests", RegistryHost}, parts...)
	parts = append(parts, p.tag)
	target := filepath.Join(parts...)
	root, err := filepath.Abs(modelRoot)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("pinned manifest target escapes Ollama model root")
	}
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create Ollama manifest directory: %w", err)
	}
	for path := dir; path != root && path != filepath.Dir(path); path = filepath.Dir(path) {
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("Ollama manifest path contains an invalid directory: %s", path)
		}
	}
	tmp, err := os.CreateTemp(dir, ".ensure-manifest-*")
	if err != nil {
		return fmt.Errorf("stage pinned Ollama manifest: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(p.manifest); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write pinned Ollama manifest: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync pinned Ollama manifest: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close staged Ollama manifest: %w", err)
	}
	if err := os.Link(tmpPath, target); err != nil {
		return fmt.Errorf("publish pinned Ollama manifest without replacing an existing tag: %w", err)
	}
	return nil
}
func (p *PinnedPull) Close() error {
	if p == nil || p.pin == nil {
		return nil
	}
	return p.pin.Close(context.Background())
}

// OpenRegistryPin validates a manifest and starts an ephemeral IPv4 loopback
// registry. repo must be a canonical library model path. No upstream HTTP
// request is made by this object.
func OpenRegistryPin(repo, tag string, manifestBytes []byte) (*Pin, error) {
	if !validRepo(repo) || !validSegment(tag) {
		return nil, errors.New("invalid pinned registry repository or tag")
	}
	if len(manifestBytes) == 0 || len(manifestBytes) > maxManifestBytes {
		return nil, errors.New("pinned manifest size is invalid")
	}
	var m manifest
	dec := json.NewDecoder(strings.NewReader(string(manifestBytes)))
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("invalid pinned manifest: %w", err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, errors.New("pinned manifest contains trailing data")
	}
	if m.SchemaVersion != 2 || len(m.Layers) == 0 {
		return nil, errors.New("pinned manifest has unsupported or empty schema")
	}
	var peak int64
	for _, d := range append([]descriptor{m.Config}, m.Layers...) {
		if !validDigest(d.Digest) || d.Size <= 0 || d.Size > math.MaxInt64-peak {
			return nil, errors.New("pinned manifest contains invalid descriptor size or digest")
		}
		peak += d.Size
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, fmt.Errorf("create pin identity: %w", err)
	}
	localRepo := "library/pin-" + hex.EncodeToString(nonce[:])
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for local registry pin: %w", err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	base := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(addr.Port))
	p := &Pin{Target: strings.TrimPrefix(base, "http://") + "/" + localRepo + ":" + tag, URL: base, Repo: localRepo, Tag: tag, PeakBytes: peak, listener: ln}
	frozen := append([]byte(nil), manifestBytes...)
	mux := http.NewServeMux()
	manifestPath := "/v2/" + localRepo + "/manifests/" + url.PathEscape(tag)
	mux.HandleFunc("/v2/", func(w http.ResponseWriter, r *http.Request) {
		if r.Host != ln.Addr().String() {
			http.Error(w, "invalid registry host", http.StatusMisdirectedRequest)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == manifestPath {
			w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
			w.Header().Set("Content-Length", strconv.Itoa(len(frozen)))
			w.Header().Set("Docker-Content-Digest", "sha256:"+sha256Hex(frozen))
			w.WriteHeader(http.StatusOK)
			if r.Method == http.MethodGet {
				_, _ = w.Write(frozen)
			}
			return
		}
		prefix := "/v2/" + localRepo + "/blobs/"
		if strings.HasPrefix(r.URL.Path, prefix) {
			digest := strings.TrimPrefix(r.URL.Path, prefix)
			if size, ok := descriptorSize(m, digest); ok {
				if r.Method == http.MethodHead {
					// Ollama v0.30.10's standard downloader truncates its sparse
					// partial from this HEAD response. Serve the preflighted descriptor
					// size locally so a changed upstream header cannot grow past the
					// admitted peak. GET remains a content-addressed HTTPS redirect.
					w.Header().Set("Content-Type", "application/octet-stream")
					w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
					w.Header().Set("Docker-Content-Digest", digest)
					w.WriteHeader(http.StatusOK)
					return
				}
				w.Header().Set("Location", "https://"+RegistryHost+"/v2/"+repo+"/blobs/"+digest)
				http.Redirect(w, r, w.Header().Get("Location"), http.StatusTemporaryRedirect)
				return
			}
		}
		http.NotFound(w, r)
	})
	p.server = &http.Server{Handler: mux, ReadHeaderTimeout: 3 * time.Second}
	go func() { _ = p.server.Serve(ln) }()
	return p, nil
}

func validRepo(s string) bool {
	if !strings.HasPrefix(s, "library/") || strings.Count(s, "/") != 1 {
		return false
	}
	return validSegment(strings.TrimPrefix(s, "library/"))
}
func validSegment(s string) bool {
	if s == "" || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._-", r)) {
			return false
		}
	}
	return true
}
func validDigest(s string) bool {
	if !strings.HasPrefix(s, "sha256:") || len(s) != 71 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(s, "sha256:"))
	return err == nil
}
func descriptorSize(m manifest, digest string) (int64, bool) {
	if m.Config.Digest == digest {
		return m.Config.Size, true
	}
	for _, d := range m.Layers {
		if d.Digest == digest {
			return d.Size, true
		}
	}
	return 0, false
}
func sha256Hex(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }

// Close stops the listener and is safe to call repeatedly.
func (p *Pin) Close(ctx context.Context) error {
	if p == nil {
		return nil
	}
	var closeErr error
	p.once.Do(func() { closeErr = p.server.Shutdown(ctx) })
	return closeErr
}
