package main

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	codexrollout "github.com/vrooli/vrooli/packages/codex-rollout"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	"web-console/backends/codex"
)

const (
	nativeUsageMaxThreads   = 16
	nativeUsageMaxFileBytes = 256 << 20
)

type nativeUsageRequest struct {
	SessionID string    `json:"session_id"`
	ThreadIDs []string  `json:"thread_ids"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
}
type nativeThreadUsage struct {
	ThreadID string                         `json:"thread_id"`
	Status   string                         `json:"status"`
	Report   codexrollout.NativeUsageReport `json:"report"`
	Model    string                         `json:"observed_model,omitempty"`
	Provider string                         `json:"observed_provider,omitempty"`
	Effort   string                         `json:"observed_effort,omitempty"`
}
type nativeUsageResponse struct {
	Origin  string                         `json:"origin"`
	Pricing string                         `json:"pricing"`
	Status  string                         `json:"status"`
	Threads []nativeThreadUsage            `json:"threads"`
	Total   codexrollout.NativeUsageReport `json:"total"`
}

// readSessionNativeUsage resolves only selected threads owned by an existing session.
func (s *Server) readSessionNativeUsage(ctx context.Context, in *sessionsv1.ReadNativeUsageRequest) (*sessionsv1.ReadNativeUsageResponse, error) {
	start, e1 := time.Parse(time.RFC3339Nano, in.GetStart())
	end, e2 := time.Parse(time.RFC3339Nano, in.GetEnd())
	req := nativeUsageRequest{SessionID: in.GetSessionId(), ThreadIDs: in.GetThreadIds(), Start: start, End: end}
	if e1 != nil || e2 != nil || uuid.Validate(req.SessionID) != nil || len(req.ThreadIDs) == 0 || len(req.ThreadIDs) > nativeUsageMaxThreads || !req.End.After(req.Start) || req.End.Sub(req.Start) > 31*24*time.Hour {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid session, threads, or bounded time window"))
	}
	seen := map[string]bool{}
	for _, id := range req.ThreadIDs {
		if uuid.Validate(id) != nil || seen[id] {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid or duplicate thread id"))
		}
		seen[id] = true
	}
	if s.sessionStore == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("session store unavailable"))
	}
	if _, err := s.sessionStore.Get(ctx, req.SessionID); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("session unavailable"))
	}
	result, err := readNativeUsage(ctx, codex.SessionHomePath(resolveSessionStateRoot(), req.SessionID), req)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("native session unavailable"))
	}
	out := &sessionsv1.ReadNativeUsageResponse{Origin: result.Origin, Pricing: result.Pricing, Status: result.Status, Total: nativeReportProto(result.Total)}
	for _, row := range result.Threads {
		thread := &sessionsv1.NativeThreadUsage{ThreadId: row.ThreadID, Status: row.Status, Report: nativeReportProto(row.Report), ObservedModel: row.Model, ObservedProvider: row.Provider, ObservedEffort: row.Effort}
		for _, setting := range row.Report.ModelContexts {
			thread.ModelContexts = append(thread.ModelContexts, &sessionsv1.NativeModelContext{Model: setting.Model, Effort: setting.Effort, Count: int32(setting.Count)})
		}
		out.Threads = append(out.Threads, thread)
	}
	return out, nil
}

func nativeReportProto(r codexrollout.NativeUsageReport) *sessionsv1.NativeUsageReport {
	out := &sessionsv1.NativeUsageReport{Responses: int32(r.Responses), Duplicates: int32(r.Duplicates), Incomplete: int32(r.Incomplete)}
	if r.Usage != nil {
		out.Usage = &sessionsv1.NativeUsageTotals{InputTokens: r.Usage.InputTokens, CachedInputTokens: r.Usage.CachedInputTokens, UncachedInputTokens: r.Usage.UncachedInputTokens, OutputTokens: r.Usage.OutputTokens}
	}
	return out
}

func readNativeUsage(ctx context.Context, home string, req nativeUsageRequest) (nativeUsageResponse, error) {
	result := nativeUsageResponse{Origin: "native_interactive", Pricing: "unknown", Status: "ok", Threads: make([]nativeThreadUsage, 0, len(req.ThreadIDs))}
	uri := (&url.URL{Scheme: "file", Path: filepath.Join(home, "state_5.sqlite"), RawQuery: "mode=ro"}).String()
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return result, err
	}
	defer db.Close()
	if err = db.PingContext(ctx); err != nil {
		return result, err
	}
	// Codex versions predating these columns remain valid usage sources. Keep
	// missing settings unknown rather than inferring them from launch requests.
	columns, err := db.QueryContext(ctx, "PRAGMA table_info(threads)")
	if err != nil {
		return result, err
	}
	available := map[string]bool{}
	for columns.Next() {
		var cid, notNull, primaryKey int
		var name, kind string
		var defaultValue sql.NullString
		if err := columns.Scan(&cid, &name, &kind, &notNull, &defaultValue, &primaryKey); err != nil {
			columns.Close()
			return result, err
		}
		available[name] = true
	}
	if err := columns.Err(); err != nil {
		columns.Close()
		return result, err
	}
	columns.Close()
	fields := []string{"rollout_path"}
	for _, name := range []string{"model", "model_provider", "reasoning_effort"} {
		if available[name] {
			fields = append(fields, name)
		} else {
			fields = append(fields, "NULL")
		}
	}
	query := "SELECT " + strings.Join(fields, ", ") + " FROM threads WHERE id = ?"

	for _, id := range req.ThreadIDs {
		row := nativeThreadUsage{ThreadID: id, Status: "unknown"}
		var path string
		var model, provider, effort sql.NullString
		err := db.QueryRowContext(ctx, query, id).Scan(&path, &model, &provider, &effort)
		if err == nil {
			row.Model, row.Provider, row.Effort = model.String, provider.String, effort.String
			var f *os.File
			f, err = openNativeRollout(filepath.Join(home, "sessions"), path)
			if err == nil {
				var one codexrollout.CodexNativeUsageAccumulator
				err = one.Read(ctx, f, id, req.Start, req.End, nativeUsageMaxFileBytes)
				_ = f.Close()
				if err == nil {
					row.Report = one.Report()
					row.Status = "ok"
					if row.Report.Incomplete > 0 || row.Report.Usage == nil {
						row.Status = "partial"
					}
					addNativeUsage(&result.Total, row.Report)
				}
			}
		}
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		if row.Status != "ok" {
			result.Status = "partial"
		}
		result.Threads = append(result.Threads, row)
	}
	return result, nil
}

// OpenRoot confines symlink resolution to the session directory at open time.
// Nonblocking open followed by regular-file admission rejects FIFOs and devices.
func openNativeRollout(root, path string) (*os.File, error) {
	if !filepath.IsAbs(path) || !strings.HasSuffix(path, ".jsonl") {
		return nil, errors.New("invalid rollout")
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, errors.New("rollout outside session")
	}
	dir, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	f, err := dir.OpenFile(rel, os.O_RDONLY|nativeNonblockingOpenFlag, 0)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		f.Close()
		return nil, errors.New("rollout is not a regular file")
	}
	return f, nil
}

// Threads are unique in the request; each report already deduplicates its
// response IDs. Missing reports never manufacture zero observations.
func addNativeUsage(total *codexrollout.NativeUsageReport, part codexrollout.NativeUsageReport) {
	total.Responses += part.Responses
	total.Duplicates += part.Duplicates
	total.Incomplete += part.Incomplete
	if part.Usage == nil {
		return
	}
	if total.Usage == nil {
		total.Usage = &codexrollout.NativeUsage{}
	}
	total.Usage.InputTokens += part.Usage.InputTokens
	total.Usage.CachedInputTokens += part.Usage.CachedInputTokens
	total.Usage.UncachedInputTokens += part.Usage.UncachedInputTokens
	total.Usage.OutputTokens += part.Usage.OutputTokens
}
