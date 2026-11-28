package agenttranscript

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// CursorConversationSnapshot is the transport-neutral input for one Cursor
// composer. Composer and bubble values are native JSON bytes regardless of
// whether their enclosing snapshot encoded them as JSON objects or byte slices.
type CursorConversationSnapshot struct {
	Composer []byte            `json:"composer"`
	Bubbles  map[string][]byte `json:"bubbles"`
}

// CursorWorkspaceSnapshot and CursorWorkspaceComposerSnapshot retain the
// Web Console archive's embedded-JSON representation while it groups the
// native key/value rows into one source snapshot.
type CursorWorkspaceSnapshot struct {
	Composers map[string]CursorWorkspaceComposerSnapshot `json:"composers"`
}

type CursorWorkspaceComposerSnapshot struct {
	Composer json.RawMessage            `json:"composer"`
	Bubbles  map[string]json.RawMessage `json:"bubbles"`
}

// CursorWorkspaceSnapshotBuilder groups native cursorDiskKV rows into the
// Web Console workspace wire shape or an Agent Manager per-composer snapshot.
// It owns key-prefix and bubble-ID parsing shared by both database adapters.
type CursorWorkspaceSnapshotBuilder struct {
	composers map[string][]byte
	bubbles   map[string]map[string][]byte
}

// CursorKVRows is the row subset returned by database/sql query results.
// Keeping this small interface lets the shared codec own row decoding without
// selecting a SQLite driver or owning a consumer's transaction.
type CursorKVRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close() error
}

func NewCursorWorkspaceSnapshotBuilder() *CursorWorkspaceSnapshotBuilder {
	return &CursorWorkspaceSnapshotBuilder{
		composers: make(map[string][]byte),
		bubbles:   make(map[string]map[string][]byte),
	}
}

// AddKV groups one native Cursor key/value row. Unknown and malformed keys are
// ignored, matching the adapters' former prefix checks.
func (b *CursorWorkspaceSnapshotBuilder) AddKV(key string, value []byte) {
	if b == nil {
		return
	}
	if composerID, ok := strings.CutPrefix(key, "composerData:"); ok && composerID != "" {
		b.composers[composerID] = append([]byte(nil), value...)
		return
	}
	composerID, remainder, ok := strings.Cut(strings.TrimPrefix(key, "bubbleId:"), ":")
	if !strings.HasPrefix(key, "bubbleId:") || !ok || composerID == "" {
		return
	}
	if b.bubbles[composerID] == nil {
		b.bubbles[composerID] = make(map[string][]byte)
	}
	b.bubbles[composerID][remainder] = append([]byte(nil), value...)
}

// AddRows decodes ordered cursorDiskKV rows, accounts their source bytes, and
// groups them in this builder. The caller owns the query and transaction; this
// method always closes rows, including on errors and size-limit overflow.
func (b *CursorWorkspaceSnapshotBuilder) AddRows(rows CursorKVRows, initialBytes, maxBytes int64) (size int64, exceeded bool, err error) {
	size = initialBytes
	for rows.Next() {
		var key string
		var value []byte
		if err := rows.Scan(&key, &value); err != nil {
			_ = rows.Close()
			return size, false, err
		}
		size += int64(len(key) + len(value))
		if size > maxBytes {
			_ = rows.Close()
			return size, true, nil
		}
		b.AddKV(key, value)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return size, false, err
	}
	if err := rows.Close(); err != nil {
		return size, false, err
	}
	return size, false, nil
}

// WorkspaceSnapshot returns known composer headers with their grouped native
// bubble JSON. Orphan bubble rows are ignored as before.
func (b *CursorWorkspaceSnapshotBuilder) WorkspaceSnapshot() CursorWorkspaceSnapshot {
	snapshot := CursorWorkspaceSnapshot{Composers: make(map[string]CursorWorkspaceComposerSnapshot, len(b.composers))}
	for composerID, composer := range b.composers {
		bubbles := make(map[string]json.RawMessage, len(b.bubbles[composerID]))
		for bubbleID, value := range b.bubbles[composerID] {
			bubbles[bubbleID] = append(json.RawMessage(nil), value...)
		}
		snapshot.Composers[composerID] = CursorWorkspaceComposerSnapshot{
			Composer: append(json.RawMessage(nil), composer...), Bubbles: bubbles,
		}
	}
	return snapshot
}

// ConversationSnapshot returns one composer in transport-neutral native bytes.
func (b *CursorWorkspaceSnapshotBuilder) ConversationSnapshot(composerID string) CursorConversationSnapshot {
	composer := b.composers[composerID]
	bubbles := make(map[string][]byte, len(b.bubbles[composerID]))
	for bubbleID, value := range b.bubbles[composerID] {
		bubbles[bubbleID] = append([]byte(nil), value...)
	}
	return CursorConversationSnapshot{Composer: append([]byte(nil), composer...), Bubbles: bubbles}
}

// DecodeCursorConversationSnapshot decodes one composer+bubbles snapshot.
// The Web Console archive path embeds native JSON objects, while Agent
// Manager's snapshot transport encodes []byte fields as base64 JSON strings.
// This shared boundary normalizes both representations before codec parsing.
func DecodeCursorConversationSnapshot(data []byte) (CursorConversationSnapshot, error) {
	var wire struct {
		Composer json.RawMessage            `json:"composer"`
		Bubbles  map[string]json.RawMessage `json:"bubbles"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return CursorConversationSnapshot{}, err
	}
	return decodeCursorConversationSnapshotValues(wire.Composer, wire.Bubbles)
}

func decodeCursorConversationSnapshotValues(composerRaw json.RawMessage, bubblesRaw map[string]json.RawMessage) (CursorConversationSnapshot, error) {
	composer, err := decodeCursorSnapshotValue(composerRaw)
	if err != nil {
		return CursorConversationSnapshot{}, fmt.Errorf("decode composer: %w", err)
	}
	snapshot := CursorConversationSnapshot{Composer: composer, Bubbles: make(map[string][]byte, len(bubblesRaw))}
	for id, raw := range bubblesRaw {
		value, err := decodeCursorSnapshotValue(raw)
		if err != nil {
			return CursorConversationSnapshot{}, fmt.Errorf("decode bubble %q: %w", id, err)
		}
		snapshot.Bubbles[id] = value
	}
	return snapshot, nil
}

// DecodeCursorWorkspaceSnapshot decodes the Web Console transport that groups
// composer snapshots by their native composer ID.
func DecodeCursorWorkspaceSnapshot(data []byte) (map[string]CursorConversationSnapshot, error) {
	var wire CursorWorkspaceSnapshot
	if err := json.Unmarshal(data, &wire); err != nil {
		return nil, err
	}
	snapshots := make(map[string]CursorConversationSnapshot, len(wire.Composers))
	for id, native := range wire.Composers {
		// Web Console's former RawMessage decoder tolerated unsupported native
		// composer/bubble shapes and let ParseCursorConversation skip them.
		// Preserve that per-message policy while centralizing the envelope.
		snapshot := CursorConversationSnapshot{
			Composer: append([]byte(nil), native.Composer...),
			Bubbles:  make(map[string][]byte, len(native.Bubbles)),
		}
		for bubbleID, raw := range native.Bubbles {
			snapshot.Bubbles[bubbleID] = append([]byte(nil), raw...)
		}
		snapshots[id] = snapshot
	}
	return snapshots, nil
}

func decodeCursorSnapshotValue(raw json.RawMessage) ([]byte, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}
	switch raw[0] {
	case '{':
		return append([]byte(nil), raw...), nil
	case '"':
		var value []byte
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		return value, nil
	default:
		return nil, fmt.Errorf("snapshot value must be a JSON object or base64 byte string")
	}
}
