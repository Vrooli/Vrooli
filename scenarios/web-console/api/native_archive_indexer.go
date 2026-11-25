package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/vrooli/cli-core/agenttranscript"
	"web-console/internal/tailer"
)

// NativeArchiveFile binds an already discovered native archive file to the
// stable Web Console session that owns its local projection. Discovery stays
// with the harness/catalog adapter; every byte is decoded by the shared
// agenttranscript package and tailed by the existing checkpoint engine.
type NativeArchiveFile struct {
	Harness             string
	Path                string
	SourceKey           string
	CatalogKey          string
	Generation          string
	StatSignature       string
	ContentDigest       string
	Size                int64
	WebConsoleSessionID string
	NativeSessionID     string
}

// NativeArchiveIndexer builds the Web Console-owned archive backfill/tail
// engine. It has no Agent Manager dependency; callers may start this engine
// while Agent Manager is stopped.
type NativeArchiveIndexer struct {
	conversations *ConversationStore
	checkpoints   AgentTranscriptCheckpointStore
	catalog       *nativeArchiveCatalog
	logger        *log.Logger
}

const nativeArchiveOccurrenceTimeReplayMigration = "native_archive_occurred_at_v1"

func NewNativeArchiveIndexer(conversations *ConversationStore, checkpoints AgentTranscriptCheckpointStore, logger *log.Logger) *NativeArchiveIndexer {
	catalog := newMemoryNativeArchiveCatalog()
	if provider, ok := checkpoints.(interface{ nativeArchiveCatalog() *nativeArchiveCatalog }); ok {
		catalog = provider.nativeArchiveCatalog()
	}
	return &NativeArchiveIndexer{conversations: conversations, checkpoints: checkpoints, catalog: catalog, logger: logger}
}

// NewEngine creates a resumable local indexer for the current native archive
// catalog snapshot. The shared tailer owns initial backfill, append polling,
// byte checkpoints and restart behavior. Stable event identities are derived
// from native session, source-file byte offset, raw record bytes, and message
// position within the record; replay after a failed checkpoint therefore does
// not append duplicates while identical records at separate offsets stay distinct.
func (i *NativeArchiveIndexer) NewEngine(files []NativeArchiveFile) (*tailer.Engine, error) {
	if i == nil || i.conversations == nil || i.checkpoints == nil {
		return nil, errors.New("native archive indexer requires the local conversation and checkpoint stores")
	}
	source, err := newNativeArchiveSource(files)
	if err != nil {
		return nil, err
	}
	return i.engineForSource(source), nil
}

func (i *NativeArchiveIndexer) engineForSource(source *nativeArchiveSource) *tailer.Engine {
	return tailer.New(tailer.Config{
		Name:             "native-archive-index",
		Source:           source,
		Checkpoints:      transcriptCheckpointAdapter{store: i.checkpoints},
		StopUndiscovered: true,
		Logger:           i.logger,
		Dispatch: func(event tailer.Event, sessionID string) bool {
			return source.dispatchCurrent(event.Source, func(provenance *NativeProvenance) bool {
				projectionSessionID := sessionID
				if provenance.Provider == "cursor" {
					projectionSessionID = cursorComposerSessionID(sessionID, provenance.SessionID)
				}
				_, result := i.conversations.AppendNativeEventAt(context.Background(), projectionSessionID, provenance.Provider+"_archive", event.Role, event.Text, provenance, event.OccurredAt)
				return result.Appended
			})
		},
	})
}

// NewDiscoveredEngine catalogs the supplied native roots and wires the
// resulting files into the same local checkpointed search index. The returned
// catalog is useful for reporting which files were admitted and their stable
// native identities; roots must be explicitly selected by the caller.
func (i *NativeArchiveIndexer) NewDiscoveredEngine(ctx context.Context, roots NativeArchiveRoots) (*tailer.Engine, []NativeArchiveFile, error) {
	files, err := DiscoverNativeArchiveFiles(ctx, roots)
	if err != nil {
		return nil, nil, err
	}
	if i == nil || i.conversations == nil || i.checkpoints == nil {
		return nil, nil, errors.New("native archive indexer requires the local conversation and checkpoint stores")
	}
	files, err = i.refreshArchiveCatalog(ctx, files)
	if err != nil {
		return nil, nil, err
	}
	source, err := newNativeArchiveSource(files)
	if err != nil {
		return nil, nil, err
	}
	rootCopy := roots.Clone()
	source.roots = &rootCopy
	source.refresh = func(refreshCtx context.Context) ([]NativeArchiveFile, error) {
		nativeFiles, err := DiscoverNativeArchiveFiles(refreshCtx, rootCopy)
		if err != nil {
			return nil, err
		}
		return i.refreshArchiveCatalog(refreshCtx, nativeFiles)
	}
	engine := i.engineForSource(source)
	return engine, files, nil
}

// NewLazyDiscoveredEngine defers the potentially large home-directory walk and
// archive fingerprint pass until the tailer's first poll. Production startup
// therefore remains independent of transcript corpus size.
func (i *NativeArchiveIndexer) NewLazyDiscoveredEngine(roots NativeArchiveRoots) (*tailer.Engine, error) {
	if i == nil || i.conversations == nil || i.checkpoints == nil {
		return nil, errors.New("native archive indexer requires the local conversation and checkpoint stores")
	}
	source, err := newNativeArchiveSource(nil)
	if err != nil {
		return nil, err
	}
	rootCopy := roots.Clone()
	source.roots = &rootCopy
	source.refresh = func(ctx context.Context) ([]NativeArchiveFile, error) {
		files, err := DiscoverNativeArchiveFiles(ctx, rootCopy)
		if err != nil {
			return nil, err
		}
		return i.refreshArchiveCatalog(ctx, files)
	}
	return i.engineForSource(source), nil
}

func (i *NativeArchiveIndexer) refreshArchiveCatalog(ctx context.Context, files []NativeArchiveFile) ([]NativeArchiveFile, error) {
	if i.catalog != nil {
		var err error
		files, err = i.catalog.Reconcile(ctx, files, func(sessionID string) error {
			if strings.HasPrefix(sessionID, "wc-archive-cursor-") {
				if err := i.conversations.DeleteSessionsByPrefix(ctx, sessionID+"-composer-"); err != nil {
					return err
				}
			}
			if err := i.conversations.DeleteSession(ctx, sessionID); err != nil {
				return err
			}
			return i.checkpoints.DeleteSession(ctx, sessionID)
		})
		if err != nil {
			return nil, err
		}
	}
	if replay, ok := i.checkpoints.(interface {
		ResetNativeArchiveOccurrenceCursors(context.Context, string, string) error
	}); ok {
		if err := replay.ResetNativeArchiveOccurrenceCursors(ctx, "native-archive-index", nativeArchiveOccurrenceTimeReplayMigration); err != nil {
			return nil, fmt.Errorf("prepare native occurrence-time replay: %w", err)
		}
	}
	return files, nil
}

func cursorComposerSessionID(databaseSessionID, composerID string) string {
	digest := sha256.Sum256([]byte(composerID))
	return databaseSessionID + "-composer-" + hex.EncodeToString(digest[:12])
}

type nativeArchiveSource struct {
	files      []NativeArchiveFile
	refs       []tailer.FileRef
	byPath     map[string]NativeArchiveFile
	identityMu sync.RWMutex
	roots      *NativeArchiveRoots
	refresh    func(context.Context) ([]NativeArchiveFile, error)
	activeKeys map[string]struct{}
}

func newNativeArchiveSource(files []NativeArchiveFile) (*nativeArchiveSource, error) {
	source := &nativeArchiveSource{
		files: append([]NativeArchiveFile(nil), files...), byPath: make(map[string]NativeArchiveFile, len(files)),
		activeKeys: make(map[string]struct{}),
	}
	for _, file := range files {
		format, supported := agenttranscript.FormatFor(file.Harness)
		if !supported || format.MessageDialect == "" {
			return nil, fmt.Errorf("unsupported native archive harness %q", file.Harness)
		}
		if file.Path == "" || file.WebConsoleSessionID == "" || file.NativeSessionID == "" {
			return nil, fmt.Errorf("native archive %q needs path and both session identities", file.Harness)
		}
		file.Harness = strings.ToLower(strings.TrimSpace(file.Harness))
		file.Path = filepath.Clean(file.Path)
		if file.SourceKey == "" {
			file.SourceKey = "native-archive/" + file.Harness + "/" + file.NativeSessionID
		}
		if file.CatalogKey == "" {
			file.CatalogKey = file.SourceKey
		}
		if _, duplicate := source.byPath[file.Path]; duplicate {
			return nil, fmt.Errorf("native archive path appears more than once: %s", file.Path)
		}
		source.byPath[file.Path] = file
		source.refs = append(source.refs, tailer.FileRef{Path: file.Path, SessionID: file.WebConsoleSessionID, SourceKey: file.SourceKey})
		source.activeKeys[file.SourceKey] = struct{}{}
	}
	return source, nil
}

func (s *nativeArchiveSource) DiscoverFiles(ctx context.Context) ([]tailer.FileRef, error) {
	if s.refresh != nil {
		s.identityMu.Lock()
		files, err := s.refresh(ctx)
		if err != nil {
			s.identityMu.Unlock()
			return nil, err
		}
		discovered, err := newNativeArchiveSource(files)
		if err != nil {
			s.identityMu.Unlock()
			return nil, err
		}
		s.files, s.refs, s.byPath, s.activeKeys = discovered.files, discovered.refs, discovered.byPath, discovered.activeKeys
		s.identityMu.Unlock()
	}
	s.identityMu.RLock()
	defer s.identityMu.RUnlock()
	return append([]tailer.FileRef(nil), s.refs...), nil
}

func (s *nativeArchiveSource) DecodeLine(path string, line []byte) ([]tailer.Event, error) {
	return s.DecodeLineAt(path, line, -1)
}

func (s *nativeArchiveSource) DecodeLineAt(path string, line []byte, byteOffset int64) ([]tailer.Event, error) {
	s.identityMu.RLock()
	file, ok := s.byPath[filepath.Clean(path)]
	s.identityMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("native archive path is not in the active catalog: %s", path)
	}
	messages := agenttranscript.ParseMessages(file.Harness, line)
	if len(messages) == 0 {
		return nil, nil
	}
	lineHash := sha256.Sum256(line)
	lineID := hex.EncodeToString(lineHash[:])
	position := strconv.FormatInt(byteOffset, 10)
	events := make([]tailer.Event, 0, len(messages))
	for index, message := range messages {
		if message.Role != "user" && message.Role != "assistant" {
			continue
		}
		identity := strings.Join([]string{file.SourceKey, file.CatalogKey, file.Harness, file.NativeSessionID, position, lineID, strconv.Itoa(index)}, "\x1f")
		events = append(events, tailer.Event{Source: identity, Role: message.Role, Text: message.Text, OccurredAt: message.OccurredAt, Commit: true})
	}
	return events, nil
}

func (s *nativeArchiveSource) HandlesDocument(path string) bool {
	s.identityMu.RLock()
	defer s.identityMu.RUnlock()
	file, ok := s.byPath[filepath.Clean(path)]
	format, supported := agenttranscript.FormatFor(file.Harness)
	return ok && supported && (format.MessageDialect == "gemini-cli-chat" || format.ArchiveSnapshot)
}

func (s *nativeArchiveSource) DecodeDocument(path string, document []byte) ([]tailer.Event, bool, error) {
	s.identityMu.RLock()
	file, ok := s.byPath[filepath.Clean(path)]
	s.identityMu.RUnlock()
	format, supported := agenttranscript.FormatFor(file.Harness)
	if !ok || !supported || (format.MessageDialect != "gemini-cli-chat" && !format.ArchiveSnapshot) {
		return nil, false, fmt.Errorf("native document archive path is not in the active catalog: %s", path)
	}
	if format.ArchiveSnapshot {
		return s.decodeCursorSnapshot(document, file)
	}
	conversation, valid := agenttranscript.ParseGeminiConversationDocument(document)
	if !valid || conversation.SessionID != file.NativeSessionID {
		return nil, false, nil
	}
	messages := conversation.Messages
	events := make([]tailer.Event, 0, len(messages))
	for index, message := range messages {
		if message.Role != "user" && message.Role != "assistant" {
			continue
		}
		messageHash := sha256.Sum256([]byte(message.Role + "\x00" + message.Text))
		identity := strings.Join([]string{file.SourceKey, file.CatalogKey, file.Harness, file.NativeSessionID, strconv.Itoa(index), hex.EncodeToString(messageHash[:])}, "\x1f")
		events = append(events, tailer.Event{Source: identity, Role: message.Role, Text: message.Text, OccurredAt: message.OccurredAt, Commit: true})
	}
	return events, true, nil
}

const maxCursorSnapshotBytes = 64 << 20

// Cursor stores conversations as SQLite key/value snapshots rather than
// append-only files. Decode a consistent, read-only database snapshot through
// the shared codec; the tailer watches both the database and its WAL version.
func (s *nativeArchiveSource) decodeCursorSnapshot(document []byte, file NativeArchiveFile) ([]tailer.Event, bool, error) {
	snapshots, err := agenttranscript.DecodeCursorWorkspaceSnapshot(document)
	if err != nil {
		return nil, false, err
	}
	events := []tailer.Event{}
	for composerID, composer := range snapshots {
		_, messages, ok := agenttranscript.ParseCursorConversationForID(composer.Composer, composer.Bubbles, composerID)
		if !ok {
			continue
		}
		for index, message := range messages {
			messageHash := sha256.Sum256([]byte(message.Role + "\x00" + message.Text))
			identity := strings.Join([]string{file.SourceKey, file.CatalogKey, composerID, strconv.Itoa(index), hex.EncodeToString(messageHash[:])}, "\x1f")
			events = append(events, tailer.Event{Source: identity, Role: message.Role, Text: message.Text, OccurredAt: message.OccurredAt, Commit: true})
		}
	}
	return events, true, nil
}

func (s *nativeArchiveSource) SnapshotVersion(path string) (string, error) {
	return cursorArchiveStatSignature(path)
}

func (s *nativeArchiveSource) UsesSnapshot(path string) bool {
	s.identityMu.RLock()
	defer s.identityMu.RUnlock()
	file, ok := s.byPath[filepath.Clean(path)]
	format, supported := agenttranscript.FormatFor(file.Harness)
	return ok && supported && format.ArchiveSnapshot
}

func (s *nativeArchiveSource) ReadSnapshot(path string) ([]byte, string, error) {
	before, err := s.SnapshotVersion(path)
	if err != nil {
		return nil, "", err
	}
	encoded, exceeded, err := agenttranscript.ReadCursorWorkspaceSnapshotFile(context.Background(), path, maxCursorSnapshotBytes)
	if err != nil {
		return nil, "", err
	}
	if exceeded {
		return nil, "", fmt.Errorf("Cursor conversation snapshot exceeds %d-byte limit", maxCursorSnapshotBytes)
	}
	after, err := s.SnapshotVersion(path)
	if err != nil {
		return nil, "", err
	}
	if before != after {
		return nil, "", errors.New("Cursor database changed while its snapshot was read")
	}
	return encoded, after, nil
}

func cursorArchiveStatSignature(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	base := path + ":" + nativeArchiveStatSignature(info)
	wal := path + "-wal"
	walInfo, walErr := os.Stat(wal)
	if walErr == nil && walInfo.Size() > 0 {
		return base + "|" + wal + ":" + nativeArchiveStatSignature(walInfo), nil
	}
	if walErr == nil || os.IsNotExist(walErr) {
		return base + "|wal:inactive", nil
	}
	return "", walErr
}

func (s *nativeArchiveSource) CompleteEOFLine(path string, line []byte) bool {
	s.identityMu.RLock()
	file, ok := s.byPath[filepath.Clean(path)]
	s.identityMu.RUnlock()
	return ok && file.Harness == "opencode" && json.Valid(line)
}

func (s *nativeArchiveSource) CaptureAgentInfo(string, string) {}

func (s *nativeArchiveSource) dispatchCurrent(identity string, dispatch func(*NativeProvenance) bool) bool {
	provenance, ok := nativeArchiveProvenance(identity)
	if !ok {
		return false
	}
	parts := strings.Split(identity, "\x1f")
	s.identityMu.RLock()
	defer s.identityMu.RUnlock()
	if _, current := s.activeKeys[parts[0]]; !current {
		return false
	}
	return dispatch(provenance)
}

// nativeArchiveProvenance reconstructs the persisted native fields from the
// deterministic deduplication identity. Keeping this out of a long-lived
// identity map prevents archive backfill and old generations from growing the
// source's memory footprint for the lifetime of Web Console.
func nativeArchiveProvenance(identity string) (*NativeProvenance, bool) {
	parts := strings.Split(identity, "\x1f")
	if len(parts) < 3 || parts[0] == "" {
		return nil, false
	}
	if len(parts) == 5 {
		if parts[1] == "" || parts[2] == "" || parts[3] == "" || parts[4] == "" {
			return nil, false
		}
		turn := sha256.Sum256([]byte(parts[1] + "\x00" + parts[2] + "\x00" + parts[3] + "\x00" + parts[4]))
		return &NativeProvenance{Provider: "cursor", SessionID: parts[2], TurnID: "composer:" + hex.EncodeToString(turn[:]), MessageID: parts[3]}, true
	}
	switch parts[2] {
	case "claude", "codex", "grok", "opencode", "qwen-code", "antigravity":
		if len(parts) != 7 || parts[1] == "" || parts[2] == "" || parts[3] == "" || parts[4] == "" || parts[5] == "" || parts[6] == "" {
			return nil, false
		}
		turn := sha256.Sum256([]byte(parts[1] + "\x00" + parts[4] + "\x00" + parts[5]))
		return &NativeProvenance{Provider: parts[2], SessionID: parts[3], TurnID: "record:" + hex.EncodeToString(turn[:]), MessageID: parts[6]}, true
	case "gemini-cli":
		if len(parts) != 6 || parts[1] == "" || parts[3] == "" || parts[4] == "" || parts[5] == "" {
			return nil, false
		}
		turn := sha256.Sum256([]byte(parts[1] + "\x00" + parts[3] + "\x00" + parts[4] + "\x00" + parts[5]))
		return &NativeProvenance{Provider: parts[2], SessionID: parts[3], TurnID: "document:" + hex.EncodeToString(turn[:]), MessageID: parts[4]}, true
	default:
		return nil, false
	}
}
