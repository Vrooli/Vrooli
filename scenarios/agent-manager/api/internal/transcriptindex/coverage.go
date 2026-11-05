package transcriptindex

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// CoverageReader is the service seam for SPEC A1 (`usage coverage`): the
// indexer's view of every transcript on disk against what it has captured.
type CoverageReader interface {
	Coverage(ctx context.Context, since time.Time) (Coverage, error)
}

// Coverage compares the transcripts on disk modified at or after Since with
// their capture state, per root kind.
type Coverage struct {
	Since  time.Time      `json:"since"`
	Roots  []RootCoverage `json:"roots"`
	Totals RootCoverage   `json:"totals"`
}

// RootCoverage counts one root kind. Captured files are indexed by the
// indexer or owned by the Agent Manager run that reads them. IndexedBytes
// over Bytes is a conservative share of bytes at stored checkpoints, not a
// content-integrity proof or evidence that a run recorded every event.
type RootCoverage struct {
	Kind         string      `json:"kind"`
	Harness      string      `json:"harness"`
	Files        int         `json:"files"`
	Indexed      int         `json:"indexed"`
	WithUsage    int         `json:"with_usage"`
	Owned        int         `json:"owned"`
	Excluded     int         `json:"excluded"`
	Pending      int         `json:"pending"`
	Unreadable   int         `json:"unreadable"`
	Bytes        int64       `json:"bytes"`
	IndexedBytes int64       `json:"indexed_bytes"`
	Turns        int64       `json:"turns"`
	UsageReports int64       `json:"usage_reports"`
	Issues       []FileIssue `json:"issues,omitempty"`
}

// FileIssue names an uncaptured source.
type FileIssue struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

const maxIssuesPerRoot = 50

// Coverage walks the roots now and joins each file with its row, so files the
// indexer has not reached yet count as pending rather than disappearing.
func (ix *Indexer) Coverage(ctx context.Context, since time.Time) (Coverage, error) {
	rows, err := ix.store.ListTranscriptFiles(ctx)
	if err != nil {
		return Coverage{}, err
	}
	return BuildCoverage(Roots(ix.cfg.Roots), rows, since), nil
}

// BuildCoverage is the pure coverage computation over discovered roots and
// stored rows. A nil rows slice reports the disk alone (the dry run).
func BuildCoverage(roots []Root, rows []FileRow, since time.Time) Coverage {
	byKey := make(map[string]FileRow, len(rows))
	byReconciliationPath := make(map[string]FileRow)
	for _, row := range rows {
		byKey[row.Key] = row
		if row.Status == StatusUnreadable && strings.HasPrefix(row.Error, sourceReconcilePrefix) {
			byReconciliationPath[row.Path] = row
		}
	}
	byKind := map[string]*RootCoverage{}
	entry := func(kind string) *RootCoverage {
		if byKind[kind] == nil {
			byKind[kind] = &RootCoverage{Kind: kind}
		}
		return byKind[kind]
	}
	for _, root := range roots {
		found, walkErr := Walk(root)
		cov := entry(root.Kind)
		switch {
		case cov.Harness == "":
			cov.Harness = root.Harness
		case cov.Harness != root.Harness:
			cov.Harness = HarnessMixed
		}
		if walkErr != nil {
			cov.Unreadable++
			cov.issue(root.Path, walkErr.Error())
		}
		for _, candidate := range found {
			if candidate.MtimeUnixNano < since.UnixNano() {
				continue
			}
			cov.Files++
			cov.Bytes += candidate.Size
			row, known := byKey[candidate.Key]
			if !known {
				// A replacement inode has a different key. Surface the prior
				// checkpoint's unresolved state without counting its bytes as
				// capture of the new file.
				if prior, ok := byReconciliationPath[candidate.Path]; ok {
					cov.Unreadable++
					cov.issue(candidate.Path, prior.Error)
					continue
				}
			}
			switch {
			case !known:
				cov.Pending++
				cov.issue(candidate.Path, "not indexed yet")
			case row.Status == StatusExcluded:
				cov.Excluded++
			case row.Status == StatusUnreadable:
				cov.Unreadable++
				cov.issue(candidate.Path, row.Error)
			default:
				// Metadata changes after the checkpoint make the relationship
				// between its bytes and the current source unknown, including a
				// same-size rewrite. A pending/error row is never all-clear just
				// because its saved offset equals the current file length.
				unchanged := row.Path == candidate.Path && row.Size == candidate.Size && row.MtimeUnixNano == candidate.MtimeUnixNano
				checkpoint := row.Status == StatusIndexed || row.Status == StatusOwned || (row.Status == StatusPending && row.Offset < candidate.Size)
				if unchanged && row.IndexedAt != nil && row.Error == "" && checkpoint {
					cov.IndexedBytes += min(row.Offset, candidate.Size)
				}
				complete := unchanged && row.IndexedAt != nil && row.Error == "" && row.Offset >= candidate.Size
				if complete && row.Status == StatusOwned {
					cov.Owned++
				} else if complete && row.Status == StatusIndexed {
					cov.Indexed++
				} else {
					cov.Pending++
					reason := "partially indexed"
					if row.Error != "" {
						reason = row.Error
					} else if !unchanged {
						reason = "source changed since index checkpoint"
					}
					cov.issue(candidate.Path, reason)
				}
				if row.UsageReports > 0 {
					cov.WithUsage++
				}
				cov.Turns += row.Turns
				cov.UsageReports += row.UsageReports
			}
		}
	}
	out := Coverage{Since: since, Totals: RootCoverage{Kind: "total", Harness: HarnessMixed}}
	for _, cov := range byKind {
		out.Roots = append(out.Roots, *cov)
		out.Totals.add(*cov)
	}
	sort.Slice(out.Roots, func(i, j int) bool { return out.Roots[i].Kind < out.Roots[j].Kind })
	return out
}

// CapturedShare is the checkpoint byte share; 1 for an empty denominator.
// A positive share does not prove source integrity or complete event capture.
func (c RootCoverage) CapturedShare() float64 {
	if c.Bytes == 0 {
		return 1
	}
	return float64(c.IndexedBytes) / float64(c.Bytes)
}

func (c *RootCoverage) issue(path, reason string) {
	if len(c.Issues) < maxIssuesPerRoot {
		c.Issues = append(c.Issues, FileIssue{Path: path, Reason: reason})
	}
}

func (c *RootCoverage) add(other RootCoverage) {
	c.Files += other.Files
	c.Indexed += other.Indexed
	c.WithUsage += other.WithUsage
	c.Owned += other.Owned
	c.Excluded += other.Excluded
	c.Pending += other.Pending
	c.Unreadable += other.Unreadable
	c.Bytes += other.Bytes
	c.IndexedBytes += other.IndexedBytes
	c.Turns += other.Turns
	c.UsageReports += other.UsageReports
}

// UsageEvents rebuilds usage events from durable rows so the one accounting
// function (runsignal.AccountUsage) derives a run's totals after its full
// events left the recall layer.
func UsageEvents(rows []UsageRow) []*domain.RunEvent {
	events := make([]*domain.RunEvent, 0, len(rows))
	for _, row := range rows {
		runID, _ := uuid.Parse(row.RunID)
		usage := row.Usage
		usage.PayloadKind = domain.PayloadKindUsage
		events = append(events, &domain.RunEvent{
			ID:        uuid.NewSHA1(runID, []byte(row.FileKey+":"+strconv.FormatInt(row.LineOffset, 10)+":"+strconv.Itoa(row.Ordinal))),
			RunID:     runID,
			EventType: domain.EventTypeMetric,
			Timestamp: row.ObservedAt,
			Data:      &usage,
		})
	}
	return events
}
