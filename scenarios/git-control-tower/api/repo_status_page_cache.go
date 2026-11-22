package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"connectrpc.com/connect"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
	"google.golang.org/protobuf/proto"
)

const (
	repoStatusPageCacheMaxBytes   = 64 << 20
	repoStatusPageCacheMaxEntries = 128
)

type signedRepoStatusCursor struct {
	Snapshot string `json:"s"`
	RepoPath string `json:"r"`
	Offset   int    `json:"o"`
	MAC      string `json:"m"`
}

type repoStatusPageSnapshot struct {
	id       string
	repoPath string
	response *repov1.GetRepoStatusResponse
	bytes    int
	lastUsed uint64
}

// RepoStatusPageCache keeps one immutable materialized response for active page
// cursors. Continuations read that snapshot without re-running Git: a cursor
// traversal is snapshot-consistent even while the live worktree is changing.
// The next status refresh captures the newer worktree state.
type RepoStatusPageCache struct {
	mu         sync.Mutex
	secret     [32]byte
	entries    map[string]*repoStatusPageSnapshot
	bytes      int
	sequence   uint64
	maxBytes   int
	maxEntries int
}

func NewRepoStatusPageCache(maxBytes, maxEntries int) *RepoStatusPageCache {
	cache := &RepoStatusPageCache{entries: map[string]*repoStatusPageSnapshot{}, maxBytes: maxBytes, maxEntries: maxEntries}
	if cache.maxBytes <= 0 {
		cache.maxBytes = repoStatusPageCacheMaxBytes
	}
	if cache.maxEntries <= 0 {
		cache.maxEntries = repoStatusPageCacheMaxEntries
	}
	if _, err := rand.Read(cache.secret[:]); err != nil {
		panic("crypto/rand unavailable for status page cursors: " + err.Error())
	}
	return cache
}

var defaultRepoStatusPageCache = NewRepoStatusPageCache(repoStatusPageCacheMaxBytes, repoStatusPageCacheMaxEntries)

func (c *RepoStatusPageCache) Store(ctx context.Context, repoPath string, response *repov1.GetRepoStatusResponse, pageSize uint32) (*repov1.GetRepoStatusResponse, error) {
	semantic := proto.Clone(response).(*repov1.GetRepoStatusResponse)
	semantic.Timestamp = ""
	semantic.Page = nil
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(semantic)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	h := sha256.New()
	_, _ = h.Write([]byte(repoPath + "\x00"))
	_, _ = h.Write(encoded)
	id := hex.EncodeToString(h.Sum(nil))
	full := proto.Clone(response).(*repov1.GetRepoStatusResponse)
	entryBytes := proto.Size(full) + len(repoPath) + 256
	if entryBytes > c.maxBytes {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("repository status snapshot (%d bytes) exceeds the %d-byte continuation cache budget; reduce repository scope and retry", entryBytes, c.maxBytes))
	}
	c.mu.Lock()
	if prior := c.entries[id]; prior != nil {
		c.sequence++
		prior.lastUsed = c.sequence
		full = prior.response
		entryBytes = prior.bytes
	} else {
		c.sequence++
		entry := &repoStatusPageSnapshot{id: id, repoPath: repoPath, response: full, bytes: entryBytes, lastUsed: c.sequence}
		c.entries[id] = entry
		c.bytes += entryBytes
	}
	c.evictLocked(id)
	c.mu.Unlock()
	return c.page(full, repoPath, id, 0, pageSize)
}

func (c *RepoStatusPageCache) Continue(ctx context.Context, repoPath, token string, pageSize uint32) (*repov1.GetRepoStatusResponse, error) {
	cursor, err := c.decode(token)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if cursor.RepoPath != repoPath {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("repo status page token is bound to a different repository"))
	}
	c.mu.Lock()
	entry := c.entries[cursor.Snapshot]
	if entry == nil {
		c.mu.Unlock()
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("repository status continuation expired or was invalidated; restart from the first page"))
	}
	c.sequence++
	entry.lastUsed = c.sequence
	c.mu.Unlock()
	return c.page(entry.response, repoPath, cursor.Snapshot, cursor.Offset, pageSize)
}

func (c *RepoStatusPageCache) page(full *repov1.GetRepoStatusResponse, repoPath, id string, offset int, requested uint32) (*repov1.GetRepoStatusResponse, error) {
	total := statusEntryCount(full.GetFiles())
	if offset < 0 || offset > total {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid repo status page offset"))
	}
	limit := int(requested)
	if limit <= 0 || limit > 512 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("page_size must be between 1 and 512"))
	}
	if limit > total-offset {
		limit = total - offset
	}
	for {
		page := projectRepoStatusPage(full, id, offset, limit, total)
		page.Page.NextPageToken = ""
		if offset+limit < total {
			page.Page.NextPageToken = c.encode(signedRepoStatusCursor{Snapshot: id, RepoPath: repoPath, Offset: offset + limit})
		}
		page.Page.ResponseBytes = repoStatusResponseBudget
		pageBytes := statusResponseBytes(page)
		if pageBytes <= repoStatusResponseBudget {
			page.Page.ResponseBytes = uint32(pageBytes)
			return page, nil
		}
		if limit <= 1 {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("a single repository status entry exceeds the 256 KiB response budget"))
		}
		limit /= 2
	}
}

func (c *RepoStatusPageCache) encode(cursor signedRepoStatusCursor) string {
	unsigned := cursor
	unsigned.MAC = ""
	payload, _ := json.Marshal(unsigned)
	mac := hmac.New(sha256.New, c.secret[:])
	_, _ = mac.Write(payload)
	cursor.MAC = base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	encoded, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func (c *RepoStatusPageCache) decode(token string) (signedRepoStatusCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return signedRepoStatusCursor{}, errors.New("invalid repo status page token")
	}
	var cursor signedRepoStatusCursor
	if json.Unmarshal(raw, &cursor) != nil || cursor.Snapshot == "" || cursor.RepoPath == "" || cursor.Offset < 0 {
		return signedRepoStatusCursor{}, errors.New("invalid repo status page token")
	}
	provided, err := base64.RawURLEncoding.DecodeString(cursor.MAC)
	if err != nil {
		return signedRepoStatusCursor{}, errors.New("invalid repo status page token")
	}
	unsigned := cursor
	unsigned.MAC = ""
	payload, _ := json.Marshal(unsigned)
	mac := hmac.New(sha256.New, c.secret[:])
	_, _ = mac.Write(payload)
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return signedRepoStatusCursor{}, errors.New("invalid repo status page token signature")
	}
	return cursor, nil
}

func (c *RepoStatusPageCache) InvalidateRepo(repoPath string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, entry := range c.entries {
		if entry.repoPath == repoPath {
			c.bytes -= entry.bytes
			delete(c.entries, id)
		}
	}
}
func (c *RepoStatusPageCache) InvalidateSnapshot(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry := c.entries[id]; entry != nil {
		c.bytes -= entry.bytes
		delete(c.entries, id)
	}
}
func (c *RepoStatusPageCache) evictLocked(keep string) {
	for c.bytes > c.maxBytes || len(c.entries) > c.maxEntries {
		var oldest *repoStatusPageSnapshot
		for id, e := range c.entries {
			if id == keep {
				continue
			}
			if oldest == nil || e.lastUsed < oldest.lastUsed {
				oldest = e
			}
		}
		if oldest == nil {
			return
		}
		delete(c.entries, oldest.id)
		c.bytes -= oldest.bytes
	}
}
func (c *RepoStatusPageCache) EntryCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// Usage reports the current bounded in-memory snapshot count and estimated
// retained bytes. It is used by owner benchmarks to keep capacity behavior
// visible as repository size grows.
func (c *RepoStatusPageCache) Usage() (entries, bytes int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries), c.bytes
}
