package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// ExtractionCache is deliberately small so the service can be tested with an
// in-memory fake and production can use a filesystem-backed cache. A cache is
// an optimization only: misses and cache failures must preserve extraction
// correctness.
type ExtractionCache interface {
	Get(key string) (Graph, []Warning, bool)
	Put(key string, graph Graph, warnings []Warning) error
}

const (
	extractionCacheVersion                = 1
	defaultExtractionCacheMaxBytes  int64 = 512 << 20
	defaultExtractionMemoryEntries        = 8
	defaultExtractionMemoryMaxBytes int64 = 128 << 20
)

type fileCacheEntry struct {
	Version  int       `json:"version"`
	Graph    Graph     `json:"graph"`
	Warnings []Warning `json:"warnings,omitempty"`
}

// FileExtractionCache stores immutable extraction results as atomically
// replaced JSON files. The cache directory is disposable and may be removed
// without affecting source data or the operation log.
type FileExtractionCache struct {
	dir         string
	maxBytes    int64
	mu          sync.Mutex
	memory      map[string]fileCacheEntry
	order       []string
	memoryBytes int64
}

func NewFileExtractionCache(dir string) (*FileExtractionCache, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, fmt.Errorf("cache directory is empty")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create extraction cache directory: %w", err)
	}
	return &FileExtractionCache{
		dir:      dir,
		maxBytes: defaultExtractionCacheMaxBytes,
		memory:   make(map[string]fileCacheEntry),
	}, nil
}

// NewFileExtractionCacheWithLimit is useful for deployments that need to
// bound disposable cache storage explicitly. A non-positive limit disables
// eviction while retaining the cache's atomic-write behavior.
func NewFileExtractionCacheWithLimit(dir string, maxBytes int64) (*FileExtractionCache, error) {
	cache, err := NewFileExtractionCache(dir)
	if err != nil {
		return nil, err
	}
	cache.maxBytes = maxBytes
	return cache, nil
}

func (c *FileExtractionCache) Get(key string) (Graph, []Warning, bool) {
	if c == nil || c.dir == "" {
		return Graph{}, nil, false
	}
	c.mu.Lock()
	if entry, ok := c.memory[key]; ok {
		c.touchMemoryKey(key)
		c.mu.Unlock()
		return entry.Graph, entry.Warnings, true
	}
	c.mu.Unlock()

	payload, err := os.ReadFile(filepath.Join(c.dir, key+".json"))
	if err != nil {
		return Graph{}, nil, false
	}
	var entry fileCacheEntry
	if err := json.Unmarshal(payload, &entry); err != nil || entry.Version != extractionCacheVersion {
		return Graph{}, nil, false
	}
	c.mu.Lock()
	c.rememberMemoryEntry(key, entry, int64(len(payload)))
	c.mu.Unlock()
	return entry.Graph, entry.Warnings, true
}

func (c *FileExtractionCache) Put(key string, graph Graph, warnings []Warning) error {
	if c == nil || c.dir == "" {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry := fileCacheEntry{
		Version:  extractionCacheVersion,
		Graph:    graph,
		Warnings: warnings,
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encode extraction cache entry: %w", err)
	}
	tmp, err := os.CreateTemp(c.dir, ".extraction-*.tmp")
	if err != nil {
		return fmt.Errorf("create extraction cache temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write extraction cache entry: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close extraction cache entry: %w", err)
	}
	if err := os.Rename(tmpName, filepath.Join(c.dir, key+".json")); err != nil {
		return fmt.Errorf("publish extraction cache entry: %w", err)
	}
	c.rememberMemoryEntry(key, entry, int64(len(payload)))
	// Cache entries are disposable. Eviction is deliberately best effort so a
	// full or read-only cache never turns a successful extraction into a
	// failed request.
	_ = c.evictIfNeeded()
	return nil
}

func (c *FileExtractionCache) rememberMemoryEntry(key string, entry fileCacheEntry, size int64) {
	if c.memory == nil {
		c.memory = make(map[string]fileCacheEntry)
	}
	if previous, ok := c.memory[key]; ok {
		c.memoryBytes -= cacheEntrySize(previous)
	}
	c.memory[key] = entry
	c.memoryBytes += size
	c.touchMemoryKey(key)
	for len(c.order) > defaultExtractionMemoryEntries || c.memoryBytes > defaultExtractionMemoryMaxBytes {
		oldest := c.order[0]
		c.order = c.order[1:]
		c.memoryBytes -= cacheEntrySize(c.memory[oldest])
		delete(c.memory, oldest)
	}
}

func cacheEntrySize(entry fileCacheEntry) int64 {
	payload, err := json.Marshal(entry)
	if err != nil {
		return 0
	}
	return int64(len(payload))
}

func (c *FileExtractionCache) touchMemoryKey(key string) {
	for i, current := range c.order {
		if current != key {
			continue
		}
		c.order = append(c.order[:i], c.order[i+1:]...)
		break
	}
	c.order = append(c.order, key)
}

func (c *FileExtractionCache) evictIfNeeded() error {
	if c.maxBytes <= 0 {
		return nil
	}
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}
	type cacheFile struct {
		path    string
		size    int64
		modTime int64
	}
	files := make([]cacheFile, 0, len(entries))
	var total int64
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, cacheFile{
			path:    filepath.Join(c.dir, entry.Name()),
			size:    info.Size(),
			modTime: info.ModTime().UnixNano(),
		})
		total += info.Size()
	}
	if total <= c.maxBytes {
		return nil
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime < files[j].modTime
	})
	for _, file := range files {
		if total <= c.maxBytes {
			break
		}
		if err := os.Remove(file.path); err != nil {
			continue
		}
		total -= file.size
	}
	return nil
}

func extractionCacheKey(modulePath string, opts LoadOptions, fingerprint string) string {
	parts := []string{
		fmt.Sprintf("version=%d", extractionCacheVersion),
		"module=" + filepath.Clean(modulePath),
		"fingerprint=" + fingerprint,
		"profile=" + string(opts.Profile.normalized()),
		fmt.Sprintf("include_vendor=%t", opts.IncludeVendor),
		"patterns=" + strings.Join(opts.PackagePatterns, "\x00"),
		"go=" + runtime.Version(),
		"goos=" + runtime.GOOS,
		"goarch=" + runtime.GOARCH,
	}
	parts = append(parts, "environment="+opts.EnvironmentFingerprint)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}

func moduleFingerprint(modulePath string, opts LoadOptions) (string, error) {
	var paths []string
	err := filepath.WalkDir(modulePath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != modulePath && shouldSkipFingerprintDir(entry.Name(), opts.IncludeVendor) {
				return filepath.SkipDir
			}
			return nil
		}
		if isFingerprintFile(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		contentHash := sha256.Sum256(contents)
		rel, err := filepath.Rel(modulePath, path)
		if err != nil {
			return "", err
		}
		_, _ = fmt.Fprintf(h, "%s:%d:%x\n", filepath.ToSlash(rel), info.Size(), contentHash)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func isFingerprintFile(path string) bool {
	base := filepath.Base(path)
	if base == "go.mod" || base == "go.sum" || base == "go.work.sum" {
		return true
	}
	switch filepath.Ext(path) {
	case ".go":
		return true
	default:
		return false
	}
}

func shouldSkipFingerprintDir(name string, includeVendor bool) bool {
	if name == ".git" || name == "testdata" || name == "node_modules" {
		return true
	}
	return name == "vendor" && !includeVendor
}
