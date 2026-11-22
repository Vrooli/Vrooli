package transcriptindex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"time"
)

// A transcript's lineage is the set of files that record the same harness
// session. Codex names each rollout after its thread, and the same rollout can
// exist as several files: web-console recovery copies a session's Codex home
// into the reopened session's home (where Codex resumes the thread and appends
// to the copy), and Codex compresses finished rollouts to .jsonl.zst under a
// new identity. Every such file belongs to one run, and each byte of the
// transcript is read once: a file that repeats another file's indexed bytes
// is read only from where it stops repeating them. Usage is therefore counted
// once per session at the source, and every reader of transcript usage agrees.

// lineageFamilies are the harness families whose session id names one
// transcript wherever its files are. Claude project files are not copied
// between homes, so they are not listed.
var lineageFamilies = map[string]bool{HarnessCodex: true}

// lineageKey is the transcript a file records, or "" when the file has no
// lineage (an unlisted harness, an unnamed session or a run-owned file, whose
// run reads it itself).
func lineageKey(row *FileRow) string {
	family := Family(row.Harness)
	if row.Owner != OwnerIndexer || row.SessionID == "" || !lineageFamilies[family] {
		return ""
	}
	return family + "\x00" + row.SessionID
}

// lineageOrder puts a lineage's files in the order they were first seen; the
// earliest file's run is the lineage's run.
func lineageOrder(rows []*FileRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		if !rows[i].FirstSeenAt.Equal(rows[j].FirstSeenAt) {
			return rows[i].FirstSeenAt.Before(rows[j].FirstSeenAt)
		}
		return rows[i].Key < rows[j].Key
	})
}

// continueLineage makes a new file part of its lineage's run, starting its
// cursor where its bytes stop repeating the indexed bytes of a file already
// in that lineage. A file with no lineage, or the first of its lineage, is
// left to adoption.
func (ix *Indexer) continueLineage(ctx context.Context, t *tick, row *FileRow, candidate Candidate) error {
	key := lineageKey(row)
	if key == "" || row.RunID != "" {
		return nil
	}
	var predecessors []*FileRow
	for _, other := range t.byKey {
		if other.Key != row.Key && other.RunID != "" && other.Status != StatusExcluded && lineageKey(other) == key {
			predecessors = append(predecessors, other)
		}
	}
	if len(predecessors) == 0 {
		return nil
	}
	lineageOrder(predecessors)
	best, bestShared := predecessors[0], int64(0)
	for _, predecessor := range predecessors {
		shared, err := ix.sharedPrefix(ctx, candidate, predecessor)
		if err != nil {
			return fmt.Errorf("compare with %s: %w", predecessor.Path, err)
		}
		if shared > bestShared {
			best, bestShared = predecessor, shared
		}
	}
	row.RunID, row.ParentFileKey = best.RunID, best.ParentFileKey
	if bestShared > 0 {
		head, err := HeadHash(candidate.Path, bestShared)
		if err != nil {
			return fmt.Errorf("read transcript head: %w", err)
		}
		row.Offset, row.HeadHash = bestShared, head
		t.budget -= bestShared
	}
	return nil
}

// sharedPrefix is the length of the leading complete lines of candidate that
// repeat predecessor's indexed bytes. It is proven from the predecessor's
// checkpoint alone, so the predecessor's file may be gone (compressed, moved
// or deleted): the head fingerprint covers the first bytes, and every usage
// report the predecessor recorded is an anchor — the candidate must have a
// line at the same offset with the same timestamp. A candidate whose lines
// pass every anchor and reach the predecessor's cursor shares all of it. A
// usage report is never read twice, because each one is an anchor.
func (ix *Indexer) sharedPrefix(ctx context.Context, candidate Candidate, predecessor *FileRow) (int64, error) {
	if predecessor.Offset <= 0 || predecessor.HeadHash == "" {
		return 0, nil
	}
	headLen := min(int64(headHashBytes), predecessor.Offset)
	if candidate.Size < headLen {
		return 0, nil
	}
	head, err := HeadHash(candidate.Path, headLen)
	if err != nil {
		return 0, err
	}
	if head != predecessor.HeadHash {
		return 0, nil
	}
	usage, err := ix.store.TranscriptUsageForRun(ctx, predecessor.RunID)
	if err != nil {
		return 0, err
	}
	anchors := lineAnchors(usage, predecessor.Key)
	source, err := OpenTranscript(candidate.Path)
	if err != nil {
		return 0, err
	}
	defer source.Close()
	return scanSharedPrefix(bufio.NewReaderSize(source, 64<<10), min(candidate.Size, predecessor.Offset), predecessor.Offset, headLen, anchors)
}

// lineAnchor is a line a predecessor recorded a usage report from.
type lineAnchor struct {
	offset int64
	at     time.Time
}

func lineAnchors(usage []UsageRow, fileKey string) []lineAnchor {
	var anchors []lineAnchor
	for _, row := range usage {
		if row.FileKey == fileKey {
			anchors = append(anchors, lineAnchor{offset: row.LineOffset, at: row.ObservedAt})
		}
	}
	sort.Slice(anchors, func(i, j int) bool { return anchors[i].offset < anchors[j].offset })
	out := anchors[:0]
	for _, anchor := range anchors {
		if len(out) == 0 || out[len(out)-1].offset != anchor.offset {
			out = append(out, anchor)
		}
	}
	return out
}

// lineTimestampPattern finds a JSONL record's timestamp in its first bytes,
// where Codex writes it.
var lineTimestampPattern = regexp.MustCompile(`"timestamp"\s*:\s*"([^"]+)"`)

// anchorPrefixBytes is how much of a line is kept to read its timestamp.
const anchorPrefixBytes = 4096

// scanSharedPrefix walks complete lines up to limit and returns the end of
// the last line proven to repeat the predecessor: a line inside the verified
// head, an anchor line whose timestamp matches, or — once every anchor
// matched — the line that reaches the predecessor's cursor. Lines after the
// last anchor carry no usage report in the predecessor; a predecessor with no
// anchors at all is trusted only as far as its verified head.
func scanSharedPrefix(reader *bufio.Reader, limit, cursor, headLen int64, anchors []lineAnchor) (int64, error) {
	var shared, offset int64
	next := 0
	prefix := make([]byte, 0, anchorPrefixBytes)
	for offset < limit {
		length, complete, err := readLinePrefix(reader, &prefix)
		if err != nil && !errors.Is(err, io.EOF) {
			return shared, err
		}
		end := offset + length
		if !complete || end > cursor {
			return shared, nil
		}
		if next < len(anchors) && anchors[next].offset < offset {
			return shared, nil // the predecessor had a line boundary here and this file does not
		}
		switch {
		case next < len(anchors) && anchors[next].offset == offset:
			if !sameInstant(prefix, anchors[next].at) {
				return shared, nil
			}
			next++
			shared = end
		case end <= headLen:
			shared = end
		}
		if end == cursor && next == len(anchors) && (len(anchors) > 0 || end <= headLen) {
			return end, nil
		}
		offset = end
	}
	return shared, nil
}

// readLinePrefix consumes one line, keeping its first bytes in prefix. It
// reports the line's length and whether it ended in a newline.
func readLinePrefix(reader *bufio.Reader, prefix *[]byte) (int64, bool, error) {
	*prefix = (*prefix)[:0]
	var length int64
	for {
		chunk, err := reader.ReadSlice('\n')
		length += int64(len(chunk))
		if room := anchorPrefixBytes - len(*prefix); room > 0 {
			*prefix = append(*prefix, chunk[:min(room, len(chunk))]...)
		}
		switch {
		case err == nil:
			return length, true, nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		default:
			return length, false, err
		}
	}
}

func sameInstant(line []byte, at time.Time) bool {
	var record struct {
		Timestamp string `json:"timestamp"`
	}
	text := ""
	if json.Unmarshal(line, &record) == nil {
		text = record.Timestamp
	} else if match := lineTimestampPattern.FindSubmatch(line); match != nil {
		text = string(match[1])
	}
	parsed, err := time.Parse(time.RFC3339Nano, text)
	return err == nil && parsed.Equal(at)
}

// lineageMerger moves a file, and the usage it alone recorded, onto its
// lineage's run, retiring the run the file had when no file names it any more.
type lineageMerger interface {
	MergeTranscriptRun(ctx context.Context, store Store, file FileRow, runID string) (FileRow, error)
}

// reconcileLineage enforces one run per lineage over files adopted before
// lineage existed (or whose predecessor had no run yet when they were
// adopted): each later file joins the earliest file's run, and its usage
// reports that repeat a report of that run are dropped.
func (ix *Indexer) reconcileLineage(ctx context.Context, t *tick) {
	groups := map[string][]*FileRow{}
	for _, row := range t.byKey {
		if key := lineageKey(row); key != "" && row.RunID != "" && row.Status != StatusExcluded {
			groups[key] = append(groups[key], row)
		}
	}
	var merger lineageMerger
	for _, rows := range groups {
		if ctx.Err() != nil {
			return
		}
		if !splitAcrossRuns(rows) {
			continue
		}
		if merger == nil {
			var ok bool
			if merger, ok = ix.host.(lineageMerger); !ok {
				t.summary.Failures = append(t.summary.Failures, "merge transcript copies: lineage owner unavailable")
				return
			}
		}
		lineageOrder(rows)
		runID := rows[0].RunID
		for _, row := range rows[1:] {
			if row.RunID == runID || captureInFlight(row) {
				continue
			}
			merged, err := merger.MergeTranscriptRun(ctx, ix.store, *row, runID)
			if err != nil {
				t.summary.Failures = append(t.summary.Failures, row.Path+": merge transcript copy: "+err.Error())
				continue
			}
			delete(ix.tails, row.Key)
			*row = merged
			t.summary.Merged++
		}
	}
}

func splitAcrossRuns(rows []*FileRow) bool {
	for _, row := range rows[1:] {
		if row.RunID != rows[0].RunID {
			return true
		}
	}
	return false
}

// captureInFlight reports a checkpoint whose capture or replacement has not
// settled; it is merged once it has.
func captureInFlight(row *FileRow) bool {
	return row.Error == ReplacementEffectsPending || (row.Status == StatusPending && row.Error != "")
}

// runActiveElsewhere reports another observed, active file of row's run.
func (t *tick) runActiveElsewhere(row *FileRow) bool {
	for _, other := range t.byKey {
		if other != row && other.RunID == row.RunID && other.Owner == OwnerIndexer && other.Active {
			if _, observed := t.candidates[other.Key]; observed {
				return true
			}
		}
	}
	return false
}

// runObservedElsewhere reports another file of row's run present this tick.
func (t *tick) runObservedElsewhere(row *FileRow) bool {
	for _, other := range t.byKey {
		if other != row && other.RunID == row.RunID && other.Owner == OwnerIndexer {
			if _, observed := t.candidates[other.Key]; observed {
				return true
			}
		}
	}
	return false
}

// runLastObserved is the latest event any file of row's run recorded.
func (t *tick) runLastObserved(row *FileRow) time.Time {
	var latest time.Time
	if row.LastObservedEventAt != nil {
		latest = *row.LastObservedEventAt
	}
	for _, other := range t.byKey {
		if other != row && other.RunID == row.RunID && other.LastObservedEventAt != nil && other.LastObservedEventAt.After(latest) {
			latest = *other.LastObservedEventAt
		}
	}
	return latest
}
