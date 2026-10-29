package transcriptindex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

// Usage rows written before 2026-10-29 carry the indexing time as
// observed_at and "unknown" as model: the codecs stamped usage with the parse
// time and did not read the turn's model (Codex turn_context, Claude
// message.model). RepairUsage rewrites both from the transcript itself, by
// line offset, without re-indexing (no recall or quota re-emission).

// usageRepairName is the completion marker of this repair.
const usageRepairName = "usage-line-time-and-model-2026-10-29"

// UsageRepairStore is the optional store side of RepairUsage.
type UsageRepairStore interface {
	// UsageRepairDone reports whether the named repair completed.
	UsageRepairDone(ctx context.Context, name string) (bool, error)
	// MarkUsageRepairDone records the named repair's completion.
	MarkUsageRepairDone(ctx context.Context, name string, report RepairReport) error
	// UsageFilesToRepair lists the indexer's files that still hold usage rows
	// with an unknown model.
	UsageFilesToRepair(ctx context.Context) ([]FileRow, error)
	UsageRowOffsets(ctx context.Context, fileKey string) ([]int64, error)
	// RepairUsageRows sets observed_at, and model where it is known, on every
	// row at each fix's line offset, in one short transaction: callers pass
	// at most repairBatch fixes so the serving database is never held long.
	RepairUsageRows(ctx context.Context, fileKey string, fixes []UsageFix) error
}

// UsageFix is the transcript's own time and model for the usage row(s) at
// LineOffset. An empty Model leaves the stored model unchanged.
type UsageFix struct {
	LineOffset int64
	ObservedAt time.Time
	Model      string
}

// RepairReport counts one repair pass.
type RepairReport struct {
	Files    int      `json:"files"`
	Rows     int      `json:"rows"`
	Failures []string `json:"failures,omitempty"`
}

// repairBatch bounds one repair transaction.
const repairBatch = 500

// repairRetries is how often a batch is retried when the database is busy.
var repairRetries, repairRetryDelay = 5, time.Second

// RepairUsage repairs every file the store lists once, then records its
// completion so it never runs again. A file that cannot be read is reported
// and skipped; a busy database is retried, and if it stays busy the pass
// stops unmarked and resumes on the next start (each batch is idempotent).
func RepairUsage(ctx context.Context, store UsageRepairStore) (RepairReport, error) {
	var report RepairReport
	if done, err := store.UsageRepairDone(ctx, usageRepairName); err != nil || done {
		return report, err
	}
	files, err := store.UsageFilesToRepair(ctx)
	if err != nil {
		return report, err
	}
	for _, file := range files {
		if ctx.Err() != nil {
			return report, ctx.Err()
		}
		offsets, err := store.UsageRowOffsets(ctx, file.Key)
		if err != nil {
			return report, err
		}
		fixes, err := ScanUsageFixes(file.Path, offsets)
		if err != nil {
			report.Failures = append(report.Failures, fmt.Sprintf("%s: %v", file.Path, err))
			continue
		}
		for start := 0; start < len(fixes); start += repairBatch {
			batch := fixes[start:min(len(fixes), start+repairBatch)]
			if err := retryBusy(ctx, func() error { return store.RepairUsageRows(ctx, file.Key, batch) }); err != nil {
				return report, err
			}
		}
		report.Files++
		report.Rows += len(fixes)
	}
	return report, store.MarkUsageRepairDone(ctx, usageRepairName, report)
}

func retryBusy(ctx context.Context, write func() error) error {
	err := write()
	for attempt := 0; attempt < repairRetries && err != nil && TransientStoreFailure(err.Error()); attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(repairRetryDelay):
		}
		err = write()
	}
	return err
}

// TransientStoreFailure reports whether a store error is SQLite contention
// (busy or locked) that a later attempt can succeed past, as opposed to a
// file the indexer cannot read.
func TransientStoreFailure(message string) bool {
	for _, marker := range []string{"SQLITE_BUSY", "SQLITE_LOCKED", "database is locked", "database table is locked"} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

// repairHead is how much of a line is inspected to classify it; whole lines
// are read only for usage rows and model lines.
const repairHead = 4 << 10

var (
	modelLineMarkers = [][]byte{[]byte(`"type":"turn_context"`), []byte(`"type":"assistant"`)}
	modelKey         = []byte(`"model":"`)
)

// ScanUsageFixes reads the file once. A row's model is the model its own line
// names (Claude assistant message) or else the latest model line before it
// (Codex turn_context); rows before the first model line take the file's
// first model. A row's time is its line's timestamp.
func ScanUsageFixes(path string, offsets []int64) ([]UsageFix, error) {
	handle, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	wanted := append([]int64(nil), offsets...)
	sort.Slice(wanted, func(i, j int) bool { return wanted[i] < wanted[j] })
	reader := bufio.NewReaderSize(handle, 1<<20)
	var (
		fixes      []UsageFix
		offset     int64
		current    string
		firstModel string
		next       int
		unmodelled []int
	)
	for next < len(wanted) {
		for next < len(wanted) && wanted[next] < offset {
			next++ // an offset inside a line: the file changed under the row
		}
		isRow := next < len(wanted) && wanted[next] == offset
		line, length, err := readLine(reader, func(head []byte) bool { return isRow || modelLine(head) })
		if length == 0 && err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if line != nil {
			if model := lineModel(line); model != "" && modelLine(line) {
				current = model
				if firstModel == "" {
					firstModel = model
				}
			}
			if isRow {
				fix := UsageFix{LineOffset: offset, ObservedAt: lineTime(line), Model: current}
				if fix.Model == "" {
					unmodelled = append(unmodelled, len(fixes))
				}
				fixes = append(fixes, fix)
				next++
			}
		}
		offset += length
		if err != nil {
			break
		}
	}
	for _, index := range unmodelled {
		fixes[index].Model = firstModel
	}
	return fixes, nil
}

// readLine consumes one line. It returns the whole line when want(head) is
// true for its first bytes, and its length either way.
func readLine(reader *bufio.Reader, want func(head []byte) bool) ([]byte, int64, error) {
	chunk, err := reader.ReadSlice('\n')
	length := int64(len(chunk))
	head := chunk
	if len(head) > repairHead {
		head = head[:repairHead]
	}
	keep := want(head)
	var line []byte
	if keep {
		line = append(line, chunk...)
	}
	for errors.Is(err, bufio.ErrBufferFull) {
		chunk, err = reader.ReadSlice('\n')
		length += int64(len(chunk))
		if keep {
			line = append(line, chunk...)
		}
	}
	return line, length, err
}

func modelLine(head []byte) bool {
	for _, marker := range modelLineMarkers {
		if bytes.Contains(head, marker) {
			return true
		}
	}
	return false
}

// lineModel is the first "model" string of the line: Codex turn_context
// payload.model, Claude message.model.
func lineModel(line []byte) string {
	start := bytes.Index(line, modelKey)
	if start < 0 {
		return ""
	}
	rest := line[start+len(modelKey):]
	end := bytes.IndexByte(rest, '"')
	if end <= 0 {
		return ""
	}
	model := strings.TrimSpace(string(rest[:end]))
	if model == "<synthetic>" {
		return ""
	}
	return model
}

func lineTime(line []byte) time.Time {
	var envelope struct {
		Timestamp string `json:"timestamp"`
	}
	if json.Unmarshal(bytes.TrimSpace(line), &envelope) != nil {
		return time.Time{}
	}
	at, err := time.Parse(time.RFC3339Nano, envelope.Timestamp)
	if err != nil {
		return time.Time{}
	}
	return at
}
