// Package librarysync durably forwards completed Music Tools analysis results
// into Music Library. It only observes succeeded jobs; Library failure cannot
// change the producer job's terminal state.
package librarysync

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
)

const (
	maxResultBytes = 64 << 20
	passInterval   = 10 * time.Second
	requestTimeout = 8 * time.Second
	leaseDuration  = 45 * time.Second
	maxRetryDelay  = time.Hour
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type ResultStore interface {
	Get(context.Context, string) (io.ReadCloser, string, error)
}

type AnalysisResult struct {
	JobID        string `json:"job_id"`
	Operation    string `json:"operation"`
	SourceSHA256 string `json:"source_sha256"`
	Resource     string `json:"resource"`
}

type SourceMetadata struct {
	Title     string `json:"title,omitempty"`
	Artist    string `json:"artist,omitempty"`
	Uploader  string `json:"uploader,omitempty"`
	FileName  string `json:"file_name,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
	Acquired  string `json:"acquired_at,omitempty"`
}

type Status struct {
	Pending   int    `json:"pending"`
	Delivered int    `json:"delivered"`
	Blocked   int    `json:"blocked"`
	LastError string `json:"last_error,omitempty"`
}

type Worker struct {
	db      SQLExecutor
	blobs   ResultStore
	client  *http.Client
	resolve func(context.Context) (string, error)
	logger  *log.Logger

	mu        sync.Mutex
	started   bool
	paused    bool
	resume    chan struct{}
	activeEnd chan struct{}
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

func Schema() string { return schemaSQL }

func New(db SQLExecutor, blobs ResultStore, logger *log.Logger) (*Worker, error) {
	if db == nil || blobs == nil {
		return nil, errors.New("music library sync requires database and result blob store")
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Worker{
		db: db, blobs: blobs, client: &http.Client{Timeout: requestTimeout},
		resolve: func(ctx context.Context) (string, error) {
			var resolved string
			if override := strings.TrimSpace(os.Getenv("MUSIC_LIBRARY_URL")); override != "" {
				resolved = strings.TrimRight(override, "/")
			} else {
				var err error
				resolved, err = discovery.ResolveScenarioURLDefault(ctx, "music-library")
				if err != nil {
					return "", err
				}
			}
			return validateLocalBaseURL(resolved)
		},
		logger: logger, resume: make(chan struct{}, 1),
	}, nil
}

// Start launches the durable reconciliation and delivery loop. Existing
// succeeded jobs are reconstructed into the outbox after restart.
func (w *Worker) Start(parent context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.started {
		return errors.New("music library sync worker already started")
	}
	ctx, cancel := context.WithCancel(parent)
	w.cancel, w.started = cancel, true
	w.wg.Add(1)
	go w.run(ctx)
	return nil
}

func (w *Worker) Stop() {
	w.mu.Lock()
	cancel := w.cancel
	w.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	w.wg.Wait()
}

// Quiesce stops new outbox work and waits for any current reconciliation or
// HTTP attempt to finish before scenario replacement proceeds.
func (w *Worker) Quiesce(ctx context.Context) error {
	w.mu.Lock()
	w.paused = true
	active := w.activeEnd
	w.mu.Unlock()
	if active == nil {
		return nil
	}
	select {
	case <-active:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) Resume(context.Context) error {
	w.mu.Lock()
	w.paused = false
	w.mu.Unlock()
	select {
	case w.resume <- struct{}{}:
	default:
	}
	return nil
}

func (w *Worker) run(ctx context.Context) {
	defer w.wg.Done()
	ticker := time.NewTicker(passInterval)
	defer ticker.Stop()
	for {
		w.mu.Lock()
		if w.paused {
			w.mu.Unlock()
			select {
			case <-ctx.Done():
				return
			case <-w.resume:
				continue
			}
		}
		active := make(chan struct{})
		w.activeEnd = active
		w.mu.Unlock()

		if err := w.pass(ctx); err != nil && ctx.Err() == nil {
			w.logger.Printf("music library sync pass failed: %v", err)
		}

		w.mu.Lock()
		w.activeEnd = nil
		close(active)
		w.mu.Unlock()
		select {
		case <-ctx.Done():
			return
		case <-w.resume:
		case <-ticker.C:
		}
	}
}

func (w *Worker) pass(ctx context.Context) error {
	if err := w.reconcile(ctx); err != nil {
		return err
	}
	for sent := 0; sent < 4; sent++ {
		delivered, err := w.deliverOne(ctx)
		if err != nil {
			return err
		}
		if !delivered {
			break
		}
	}
	return nil
}

func (w *Worker) reconcile(ctx context.Context) error {
	rows, err := w.db.QueryContext(ctx, `SELECT j.id,j.result_ref,j.payload FROM jobs j LEFT JOIN music_library_sync_outbox o ON o.job_id=j.id WHERE j.state='succeeded' AND j.operation IN ('analyze','separate') AND o.job_id IS NULL ORDER BY j.finished_at ASC LIMIT 100`)
	if err != nil {
		return fmt.Errorf("scan completed analysis jobs: %w", err)
	}
	type source struct {
		id, ref string
		payload []byte
	}
	var sources []source
	for rows.Next() {
		var item source
		if err := rows.Scan(&item.id, &item.ref, &item.payload); err != nil {
			rows.Close()
			return err
		}
		sources = append(sources, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, item := range sources {
		if err := ctx.Err(); err != nil {
			return err
		}
		if item.ref == "" {
			now := stamp(time.Now())
			note := "succeeded analysis job has no result reference"
			if _, insertErr := w.db.ExecContext(ctx, `INSERT OR IGNORE INTO music_library_sync_outbox (job_id,result_ref,state,available_at,last_error,created_at) VALUES (?,?,?,?,?,?)`, item.id, item.ref, "blocked", now, note, now); insertErr != nil {
				return fmt.Errorf("record missing result reference for job %s: %w", item.id, insertErr)
			}
			w.logger.Printf("music library sync blocked job %s: %s", item.id, note)
			continue
		}
		var request struct {
			SourceSHA256 string `json:"source_sha256"`
		}
		if err := json.Unmarshal(item.payload, &request); err != nil || !validSHA256(request.SourceSHA256) {
			note := "succeeded analysis job has invalid source SHA-256 in its request"
			if err != nil {
				note = "invalid queued analysis request: " + err.Error()
			}
			if _, insertErr := w.db.ExecContext(ctx, `INSERT OR IGNORE INTO music_library_sync_outbox (job_id,result_ref,state,available_at,last_error,created_at) VALUES (?,?,?,?,?,?)`, item.id, item.ref, "blocked", stamp(time.Now()), note, stamp(time.Now())); insertErr != nil {
				return fmt.Errorf("record invalid result for job %s: %w", item.id, insertErr)
			}
			w.logger.Printf("music library sync blocked invalid result for job %s: %s", item.id, note)
			continue
		}
		now := stamp(time.Now())
		if _, err := w.db.ExecContext(ctx, `INSERT OR IGNORE INTO music_library_sync_outbox (job_id,track_sha256,result_ref,state,available_at,created_at) VALUES (?,?,?,?,?,?)`, item.id, request.SourceSHA256, item.ref, "pending", now, now); err != nil {
			return fmt.Errorf("queue completed analysis job %s: %w", item.id, err)
		}
	}
	return nil
}

func (w *Worker) deliverOne(ctx context.Context) (bool, error) {
	item, ok, err := w.claim(ctx)
	if err != nil || !ok {
		return false, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	baseURL, err := w.resolve(requestCtx)
	if err != nil {
		return true, w.retry(ctx, item, fmt.Errorf("resolve music-library: %w", err))
	}
	blob, _, err := w.blobs.Get(requestCtx, item.resultRef)
	if err != nil {
		return true, w.retry(ctx, item, fmt.Errorf("read retained result: %w", err))
	}
	data, readErr := io.ReadAll(io.LimitReader(blob, maxResultBytes+1))
	closeErr := blob.Close()
	if readErr != nil || closeErr != nil {
		return true, w.retry(ctx, item, errors.Join(readErr, closeErr))
	}
	if len(data) > maxResultBytes {
		return true, w.retry(ctx, item, errors.New("retained result exceeds configured byte limit"))
	}
	var result AnalysisResult
	if err := json.Unmarshal(data, &result); err != nil || result.JobID != item.jobID || (result.Operation != "analyze" && result.Operation != "separate") || result.Resource != "music-mir" || result.SourceSHA256 != item.trackSHA {
		note := "retained result envelope does not match its durable job and source identity"
		if err != nil {
			note = "invalid retained result envelope: " + err.Error()
		}
		_, updateErr := w.db.ExecContext(ctx, `UPDATE music_library_sync_outbox SET state='blocked',lease_token='',lease_until='',last_error=? WHERE job_id=? AND lease_token=?`, note, item.jobID, item.token)
		if updateErr != nil {
			return true, fmt.Errorf("mark invalid result blocked for job %s: %w", item.jobID, updateErr)
		}
		w.logger.Printf("music library sync blocked invalid result for job %s: %s", item.jobID, note)
		return true, nil
	}
	var source SourceMetadata
	err = w.db.QueryRowContext(requestCtx, `SELECT title,artist,uploader,file_name,media_type,size_bytes,acquired_at FROM music_sources WHERE sha256=?`, item.trackSHA).Scan(&source.Title, &source.Artist, &source.Uploader, &source.FileName, &source.MediaType, &source.SizeBytes, &source.Acquired)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return true, w.retry(ctx, item, fmt.Errorf("read source provenance: %w", err))
	}
	payload, err := json.Marshal(struct {
		AnalysisJob string          `json:"analysis_job_id"`
		SourceSHA   string          `json:"source_sha256"`
		Source      SourceMetadata  `json:"source,omitempty"`
		ToolResult  json.RawMessage `json:"tool_result"`
	}{AnalysisJob: item.jobID, SourceSHA: item.trackSHA, Source: source, ToolResult: data})
	if err != nil {
		return true, w.retry(ctx, item, err)
	}
	url := strings.TrimRight(baseURL, "/") + "/api/v1/decompositions/" + item.trackSHA
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return true, w.retry(ctx, item, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.client.Do(req)
	if err != nil {
		return true, w.retry(ctx, item, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return true, w.retry(ctx, item, fmt.Errorf("music-library returned %s: %s", resp.Status, strings.TrimSpace(string(message))))
	}
	now := stamp(time.Now())
	_, err = w.db.ExecContext(ctx, `UPDATE music_library_sync_outbox SET state='delivered',delivered_at=?,lease_token='',lease_until='',last_error='' WHERE job_id=? AND lease_token=?`, now, item.jobID, item.token)
	if err != nil {
		return true, fmt.Errorf("mark job %s delivered: %w", item.jobID, err)
	}
	return true, nil
}

type claimed struct {
	jobID, trackSHA, resultRef, token string
	attempts                          int
}

func (w *Worker) claim(ctx context.Context) (claimed, bool, error) {
	item := claimed{token: uuid.NewString()}
	now := time.Now().UTC()
	_, err := w.db.ExecContext(ctx, `UPDATE music_library_sync_outbox SET lease_token=?,lease_until=?,attempts=attempts+1 WHERE job_id=(SELECT job_id FROM music_library_sync_outbox WHERE state='pending' AND available_at<=? AND (lease_until='' OR lease_until<=?) ORDER BY created_at LIMIT 1) AND state='pending' AND available_at<=? AND (lease_until='' OR lease_until<=?)`, item.token, stamp(now.Add(leaseDuration)), stamp(now), stamp(now), stamp(now), stamp(now))
	if err != nil {
		return claimed{}, false, err
	}
	err = w.db.QueryRowContext(ctx, `SELECT job_id,track_sha256,result_ref,attempts FROM music_library_sync_outbox WHERE lease_token=?`, item.token).Scan(&item.jobID, &item.trackSHA, &item.resultRef, &item.attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return claimed{}, false, nil
	}
	if err != nil {
		return claimed{}, false, err
	}
	return item, true, nil
}

func (w *Worker) retry(ctx context.Context, item claimed, cause error) error {
	delay := 10 * time.Second * time.Duration(1<<min(item.attempts-1, 8))
	if delay > maxRetryDelay {
		delay = maxRetryDelay
	}
	now := time.Now().UTC()
	_, err := w.db.ExecContext(ctx, `UPDATE music_library_sync_outbox SET available_at=?,lease_token='',lease_until='',last_error=? WHERE job_id=? AND lease_token=?`, stamp(now.Add(delay)), truncate(cause.Error(), 2000), item.jobID, item.token)
	if err != nil {
		return fmt.Errorf("retry job %s after %v (original error %v): %w", item.jobID, delay, cause, err)
	}
	w.logger.Printf("music library sync delivery deferred for job %s (attempt %d, retry in %s): %v", item.jobID, item.attempts, delay, cause)
	return nil
}

func (w *Worker) Status(ctx context.Context) (Status, error) {
	var out Status
	rows, err := w.db.QueryContext(ctx, `SELECT state,COUNT(*) FROM music_library_sync_outbox GROUP BY state`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var state string
		var n int
		if err := rows.Scan(&state, &n); err != nil {
			rows.Close()
			return out, err
		}
		switch state {
		case "pending":
			out.Pending = n
		case "delivered":
			out.Delivered = n
		case "blocked":
			out.Blocked = n
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	if err := rows.Close(); err != nil {
		return out, err
	}
	if err := w.db.QueryRowContext(ctx, `SELECT last_error FROM music_library_sync_outbox WHERE last_error<>'' ORDER BY created_at DESC LIMIT 1`).Scan(&out.LastError); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return out, err
	}
	return out, nil
}

func validSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func validateLocalBaseURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("invalid Music Library URL: %w", err)
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	ip := net.ParseIP(host)
	loopback := host == "localhost" || (ip != nil && ip.IsLoopback())
	if u.Scheme != "http" || !loopback || u.Port() == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("Music Library URL must be an explicit loopback HTTP address with no credentials or path")
	}
	return strings.TrimRight(u.String(), "/"), nil
}

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
func truncate(s string, maxBytes int) string {
	if len(s) > maxBytes {
		return s[:maxBytes]
	}
	return s
}
