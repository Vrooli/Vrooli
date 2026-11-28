package aisearch

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

// Payload field keys written by the reconciler. They are part of the on-disk
// contract: ScrollIDs projects them back out to detect drift and ghosts.
const (
	// payloadHashKey gates chunk-level drift (text + payload, both hashes
	// excluded). Stable across the cutover — kept byte-identical to cli-health
	// so a migrated collection's existing points are recognized, not re-embedded.
	payloadHashKey = "payload_hash"
	// sourceHashKey gates source-level drift: an unchanged file is skipped
	// wholesale before it is read/chunked/embedded (§4.1). Excluded from the
	// chunk hash so a same-body chunk in an edited file is not re-embedded.
	sourceHashKey = "source_hash"
	// sourceIDKey groups a source's chunks for the source-level skip and ghost
	// deletion.
	sourceIDKey = "source_id"
	// chunkIndexKey records the chunk's ordinal within its source.
	chunkIndexKey = "chunk_index"
	// chunkTotalKey records how many chunks the source fanned out into. The
	// source-level skip only fires when every expected chunk is present
	// (stored count == chunk_total) — so a budget-deferred partial index is
	// never mistaken for a complete, up-to-date one. Excluded from the chunk
	// hash (a same-body chunk in a re-split doc refreshes, not re-embeds).
	chunkTotalKey = "chunk_total"
	// bodyKey stores the raw, retrievable chunk text — fixing the legacy KO
	// content:"" defect.
	bodyKey = "body"
)

// composePayloadHash returns a stable identifier for the (embedding text,
// payload) pair so the reconciler can skip embedding when neither changed. The
// two reconciler-managed hashes (payload_hash, source_hash) are excluded so the
// chunk hash reflects only the chunk's own content + metadata.
func composePayloadHash(text string, payload map[string]any) string {
	canon, _ := canonicalJSON(stripHashFields(payload))
	h := sha256.New()
	_, _ = h.Write([]byte(text))
	_, _ = h.Write([]byte{'|'})
	_, _ = h.Write(canon)
	sum := h.Sum(nil)
	return "sha256:" + hex.EncodeToString(sum[:8])
}

func stripHashFields(payload map[string]any) map[string]any {
	out := make(map[string]any, len(payload))
	for k, v := range payload {
		if k == payloadHashKey || k == sourceHashKey || k == chunkTotalKey {
			continue
		}
		out[k] = v
	}
	return out
}

func canonicalJSON(v any) ([]byte, error) {
	return json.Marshal(canonicalize(v))
}

func canonicalize(v any) any {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make(map[string]any, len(x))
		for _, k := range keys {
			out[k] = canonicalize(x[k])
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = canonicalize(e)
		}
		return out
	default:
		return v
	}
}

// qdrantNamespace is the fixed UUID namespace used to derive point IDs from a
// source's natural key. Kept identical to cli-health/prompt-manager so a
// migrated consumer's IDs are unchanged (collisions across consumers are
// avoided by the per-consumer ID prefix, not the namespace).
var qdrantNamespace = [16]byte{
	0x6b, 0xa7, 0xb8, 0x10,
	0x9d, 0xad, 0x11, 0xd1,
	0x80, 0xb4, 0x00, 0xc0,
	0x4f, 0xd4, 0x30, 0xc8,
}

// PointIDFor returns the deterministic UUIDv5 point ID for a chunk. The natural
// key is "<prefix><sourceID>" for a single-chunk source (so 1:1 consumers like
// cli-health keep their existing IDs across the migration) and
// "<prefix><sourceID>#<index>" when a source fans out into many chunks.
func PointIDFor(prefix, sourceID string, index, total int) string {
	key := strings.TrimSpace(prefix) + strings.TrimSpace(sourceID)
	if total > 1 || index > 0 {
		key += "#" + strconv.Itoa(index)
	}
	return uuidV5(qdrantNamespace, key)
}

func uuidV5(namespace [16]byte, name string) string {
	hash := sha1.New()
	_, _ = hash.Write(namespace[:])
	_, _ = hash.Write([]byte(name))
	sum := hash.Sum(nil)

	var uuid [16]byte
	copy(uuid[:], sum[:16])
	uuid[6] = (uuid[6] & 0x0f) | 0x50
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	hexStr := hex.EncodeToString(uuid[:])
	return hexStr[0:8] + "-" + hexStr[8:12] + "-" + hexStr[12:16] + "-" + hexStr[16:20] + "-" + hexStr[20:32]
}
