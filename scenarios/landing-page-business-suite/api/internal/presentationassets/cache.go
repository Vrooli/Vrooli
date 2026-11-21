package presentationassets

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

var contentHashPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

const (
	cacheDir             = "presentation-assets"
	proofsDir            = "proofs"
	maxCachedAssetBytes  = 32 << 20
	maxCachedRecordBytes = 1 << 20
)

// Cache stores verified bytes once per content hash and keeps each verified
// release proof under its own proof identity. Thus two releases may share
// bytes without overwriting one another's release/surface/provenance proof.
// The data root is routed per request, so a test lease cannot populate live
// data.
type Cache struct {
	roots      *filerouting.RoutedRoots
	publicPath string
	mu         sync.Mutex
}

func NewCache(roots *filerouting.RoutedRoots, publicPath string) *Cache {
	if strings.TrimSpace(publicPath) == "" {
		publicPath = defaultPublicPath
	}
	if !strings.HasPrefix(publicPath, "/") || !strings.HasSuffix(publicPath, "/") {
		publicPath = defaultPublicPath
	}
	return &Cache{roots: roots, publicPath: publicPath}
}

func (c *Cache) PublicURL(hash string) string {
	if !contentHashPattern.MatchString(hash) {
		return ""
	}
	return c.publicPath + hash + ".png"
}

func (c *Cache) Put(ctx context.Context, proof AssetProof, imageBytes []byte) (AssetProof, error) {
	if c == nil || c.roots == nil {
		return AssetProof{}, errors.New("presentation assets: durable cache is unavailable")
	}
	if !contentHashPattern.MatchString(proof.ContentHash) {
		return AssetProof{}, errors.New("presentation assets: invalid content hash")
	}
	if proof.MIME != "image/png" {
		return AssetProof{}, errors.New("presentation assets: cache proof MIME must be image/png")
	}
	if len(imageBytes) == 0 || len(imageBytes) > maxCachedAssetBytes {
		return AssetProof{}, errors.New("presentation assets: cached asset exceeds size bounds")
	}
	if err := verifyPNGBytes(imageBytes, proof.ContentHash, proof.Width, proof.Height); err != nil {
		return AssetProof{}, err
	}
	proof.PublicURL = c.PublicURL(proof.ContentHash)
	proofJSON, err := json.Marshal(proof)
	if err != nil {
		return AssetProof{}, fmt.Errorf("presentation assets: encode cache proof: %w", err)
	}
	proofKey := proofDigest(proofJSON)

	c.mu.Lock()
	defer c.mu.Unlock()
	dir, err := c.roots.PickRequired(ctx, storage.ClassData)
	if err != nil {
		return AssetProof{}, err
	}
	if err := storage.EnsureDirectory(dir, 0700); err != nil {
		return AssetProof{}, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return AssetProof{}, err
	}
	defer root.Close()
	if err := root.MkdirAll(cacheDir, 0700); err != nil {
		return AssetProof{}, err
	}
	entryDir := filepath.Join(cacheDir, proof.ContentHash)
	if err := root.MkdirAll(filepath.Join(entryDir, proofsDir), 0700); err != nil {
		return AssetProof{}, err
	}

	changed, err := c.ensureBytes(root, entryDir, proof, imageBytes)
	if err != nil {
		return AssetProof{}, err
	}
	proofChanged, err := c.ensureProof(root, entryDir, proofKey, proofJSON, proof)
	if err != nil {
		return AssetProof{}, err
	}
	if changed || proofChanged {
		c.roots.RecordWrite(ctx)
	}
	return proof, nil
}

func (c *Cache) ensureBytes(root *os.Root, entryDir string, proof AssetProof, imageBytes []byte) (bool, error) {
	assetPath := filepath.Join(entryDir, "asset.png")
	if existing, err := readRootFile(root, assetPath, maxCachedAssetBytes); err == nil {
		if err := verifyPNGBytes(existing, proof.ContentHash, proof.Width, proof.Height); err != nil {
			return false, fmt.Errorf("presentation assets: immutable cached bytes conflict: %w", err)
		}
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	stage, cleanup, err := newStagingDir(root, cacheDir, proof.ContentHash)
	if err != nil {
		return false, err
	}
	defer cleanup()
	if err := storage.WriteFileAtomicInRoot(root, filepath.Join(stage, "asset.png"), imageBytes, 0600); err != nil {
		return false, fmt.Errorf("presentation assets: write cache bytes: %w", err)
	}
	if err := root.Rename(filepath.Join(stage, "asset.png"), assetPath); err != nil {
		if existing, readErr := readRootFile(root, assetPath, maxCachedAssetBytes); readErr == nil {
			if verifyErr := verifyPNGBytes(existing, proof.ContentHash, proof.Width, proof.Height); verifyErr == nil {
				return false, nil
			}
		}
		return false, fmt.Errorf("presentation assets: commit cache bytes: %w", err)
	}
	return true, nil
}

func (c *Cache) ensureProof(root *os.Root, entryDir, proofKey string, proofJSON []byte, proof AssetProof) (bool, error) {
	proofPath := filepath.Join(entryDir, proofsDir, proofKey)
	if existing, err := readProof(root, proofPath, proof.ContentHash, c.PublicURL(proof.ContentHash)); err == nil {
		encoded, marshalErr := json.Marshal(existing)
		if marshalErr != nil || !bytes.Equal(encoded, proofJSON) {
			return false, errors.New("presentation assets: immutable cache proof conflict")
		}
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, errCacheNotFound) {
		return false, err
	}
	stage, cleanup, err := newStagingDir(root, filepath.Join(entryDir, proofsDir), proofKey)
	if err != nil {
		return false, err
	}
	defer cleanup()
	digest := sha256.Sum256(proofJSON)
	checksum := []byte(hex.EncodeToString(digest[:]) + "\n")
	if err := storage.WriteFileAtomicInRoot(root, filepath.Join(stage, "metadata.json"), proofJSON, 0600); err != nil {
		return false, fmt.Errorf("presentation assets: write cache proof: %w", err)
	}
	if err := storage.WriteFileAtomicInRoot(root, filepath.Join(stage, "metadata.sha256"), checksum, 0600); err != nil {
		return false, fmt.Errorf("presentation assets: write cache proof integrity: %w", err)
	}
	if err := root.Rename(stage, proofPath); err != nil {
		if existing, readErr := readProof(root, proofPath, proof.ContentHash, c.PublicURL(proof.ContentHash)); readErr == nil {
			encoded, marshalErr := json.Marshal(existing)
			if marshalErr == nil && bytes.Equal(encoded, proofJSON) {
				return false, nil
			}
		}
		return false, fmt.Errorf("presentation assets: commit cache proof: %w", err)
	}
	return true, nil
}

var errCacheNotFound = errors.New("presentation asset cache entry not found")

func (c *Cache) Read(ctx context.Context, hash string) (AssetProof, []byte, error) {
	if c == nil || c.roots == nil {
		return AssetProof{}, nil, errors.New("presentation assets: durable cache is unavailable")
	}
	if !contentHashPattern.MatchString(hash) {
		return AssetProof{}, nil, errCacheNotFound
	}
	dir, err := c.roots.PickRequired(ctx, storage.ClassData)
	if err != nil {
		return AssetProof{}, nil, err
	}
	root, err := os.OpenRoot(dir)
	if errors.Is(err, os.ErrNotExist) {
		return AssetProof{}, nil, errCacheNotFound
	}
	if err != nil {
		return AssetProof{}, nil, err
	}
	defer root.Close()
	entry, err := c.readRoot(root, hash)
	if err != nil {
		return AssetProof{}, nil, err
	}
	return entry.Proof, entry.bytes, nil
}

type cachedEntry struct {
	Proof AssetProof
	bytes []byte
}

func (c *Cache) readRoot(root *os.Root, hash string) (cachedEntry, error) {
	entryDir := filepath.Join(cacheDir, hash)
	imageBytes, err := readRootFile(root, filepath.Join(entryDir, "asset.png"), maxCachedAssetBytes)
	if errors.Is(err, os.ErrNotExist) {
		return cachedEntry{}, errCacheNotFound
	}
	if err != nil {
		return cachedEntry{}, fmt.Errorf("presentation assets: cached bytes unavailable: %w", err)
	}
	proofDirectory, err := root.Open(filepath.Join(entryDir, proofsDir))
	if errors.Is(err, os.ErrNotExist) {
		return cachedEntry{}, errCacheNotFound
	}
	if err != nil {
		return cachedEntry{}, err
	}
	defer proofDirectory.Close()
	entries, err := proofDirectory.ReadDir(-1)
	if err != nil {
		return cachedEntry{}, err
	}
	proofKeys := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && contentHashPattern.MatchString(entry.Name()) {
			proofKeys = append(proofKeys, entry.Name())
		}
	}
	sort.Strings(proofKeys)
	for _, proofKey := range proofKeys {
		proof, proofErr := readProof(root, filepath.Join(entryDir, proofsDir, proofKey), hash, c.PublicURL(hash))
		if proofErr != nil {
			return cachedEntry{}, proofErr
		}
		if err := verifyPNGBytes(imageBytes, hash, proof.Width, proof.Height); err != nil {
			return cachedEntry{}, err
		}
		return cachedEntry{Proof: proof, bytes: imageBytes}, nil
	}
	return cachedEntry{}, errCacheNotFound
}

func readProof(root *os.Root, proofPath, hash, publicURL string) (AssetProof, error) {
	metadata, err := readRootFile(root, filepath.Join(proofPath, "metadata.json"), maxCachedRecordBytes)
	if errors.Is(err, os.ErrNotExist) {
		return AssetProof{}, errCacheNotFound
	}
	if err != nil {
		return AssetProof{}, err
	}
	checksum, err := readRootFile(root, filepath.Join(proofPath, "metadata.sha256"), 128)
	if err != nil {
		return AssetProof{}, fmt.Errorf("presentation assets: cache proof integrity unavailable: %w", err)
	}
	digest := sha256.Sum256(metadata)
	if strings.TrimSpace(string(checksum)) != hex.EncodeToString(digest[:]) {
		return AssetProof{}, errors.New("presentation assets: cache proof integrity failure")
	}
	var proof AssetProof
	decoder := json.NewDecoder(bytes.NewReader(metadata))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&proof); err != nil {
		return AssetProof{}, fmt.Errorf("presentation assets: decode cache proof: %w", err)
	}
	if proof.ContentHash != hash || proof.PublicURL != publicURL || proofDigest(metadata) != filepath.Base(proofPath) {
		return AssetProof{}, errors.New("presentation assets: cache proof identity mismatch")
	}
	if proof.MIME != "image/png" {
		return AssetProof{}, errors.New("presentation assets: cached proof MIME mismatch")
	}
	return proof, nil
}

func readRootFile(root *os.Root, name string, limit int64) ([]byte, error) {
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("presentation assets: cached record exceeds size bounds")
	}
	return data, nil
}

func proofDigest(proofJSON []byte) string {
	digest := sha256.Sum256(proofJSON)
	return hex.EncodeToString(digest[:])
}

func newStagingDir(root *os.Root, parent, label string) (string, func(), error) {
	for attempt := 0; attempt < 8; attempt++ {
		var random [16]byte
		if _, err := cryptorand.Read(random[:]); err != nil {
			return "", func() {}, fmt.Errorf("presentation assets: create secure staging name: %w", err)
		}
		name := filepath.Join(parent, fmt.Sprintf(".pending-%s-%s", label, hex.EncodeToString(random[:])))
		if err := root.Mkdir(name, 0700); err == nil {
			return name, func() { _ = root.RemoveAll(name) }, nil
		} else if !os.IsExist(err) {
			return "", func() {}, err
		}
	}
	return "", func() {}, errors.New("presentation assets: unable to allocate unique staging directory")
}

func verifyPNGBytes(imageBytes []byte, expectedHash string, expectedWidth, expectedHeight int) error {
	digest := sha256.Sum256(imageBytes)
	if hex.EncodeToString(digest[:]) != expectedHash {
		return errors.New("presentation assets: cached bytes hash mismatch")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(imageBytes))
	if err != nil || format != "png" || config.Width != expectedWidth || config.Height != expectedHeight {
		return errors.New("presentation assets: cached bytes metadata mismatch")
	}
	return nil
}

// Handler serves only exact content-addressed cache paths. It is intentionally
// limited to retrieval verbs and emits private/no-store responses under a
// routed test lease so validation artifacts cannot become public cache entries.
func (c *Cache) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if c == nil || r.URL.RawQuery != "" || !strings.HasPrefix(r.URL.Path, c.publicPath) {
			http.NotFound(w, r)
			return
		}
		requestName := strings.TrimPrefix(r.URL.Path, c.publicPath)
		hash := strings.TrimSuffix(requestName, ".png")
		if requestName != hash+".png" || !contentHashPattern.MatchString(hash) {
			http.NotFound(w, r)
			return
		}
		proof, imageBytes, err := c.Read(r.Context(), hash)
		if errors.Is(err, errCacheNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "presentation asset integrity failure", http.StatusInternalServerError)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Type", proof.MIME)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(imageBytes)))
		w.Header().Set("ETag", fmt.Sprintf("\"%s\"", proof.ContentHash))
		w.Header().Set("X-Content-SHA256", proof.ContentHash)
		if database.IsTestMode(r.Context()) {
			w.Header().Set("Cache-Control", "private, no-store")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write(imageBytes)
	})
}
