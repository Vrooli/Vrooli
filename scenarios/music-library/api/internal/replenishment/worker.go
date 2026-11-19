// Package replenishment forwards committed catalog uses to Music Tools as
// durable candidate-refill requests. It never generates music itself.
package replenishment

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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
	compositionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/music-tools/v1/composition/composition_v1connect"
)

const (
	passInterval   = 10 * time.Second
	requestTimeout = 8 * time.Second
	leaseDuration  = 45 * time.Second
	maxRetryDelay  = time.Hour
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Request struct {
	RequestID     string `json:"request_id"`
	ReservationID string `json:"reservation_id"`
	SHA256        string `json:"sha256"`
	Purpose       string `json:"purpose"`
	Project       string `json:"project"`
	Revision      string `json:"revision"`
	StyleID       string `json:"style_id,omitempty"`
}

type Worker struct {
	db      SQLExecutor
	client  *http.Client
	resolve func(context.Context) (string, error)
	logger  *log.Logger

	mu     sync.Mutex
	start  bool
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New(db SQLExecutor, logger *log.Logger) (*Worker, error) {
	if db == nil {
		return nil, errors.New("music replenishment worker requires a database")
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Worker{
		db: db, client: &http.Client{Timeout: requestTimeout}, logger: logger,
		resolve: func(ctx context.Context) (string, error) {
			base := strings.TrimRight(strings.TrimSpace(os.Getenv("MUSIC_TOOLS_URL")), "/")
			if base == "" {
				var err error
				base, err = discovery.ResolveScenarioURLDefault(ctx, "music-tools")
				if err != nil {
					return "", err
				}
			}
			return validateBaseURL(base)
		},
	}, nil
}

func (w *Worker) Start(parent context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.start {
		return errors.New("music replenishment worker already started")
	}
	ctx, cancel := context.WithCancel(parent)
	w.cancel, w.start = cancel, true
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

func (w *Worker) run(ctx context.Context) {
	defer w.wg.Done()
	ticker := time.NewTicker(passInterval)
	defer ticker.Stop()
	for {
		for sent := 0; sent < 4; sent++ {
			didWork, err := w.deliverOne(ctx)
			if err != nil && ctx.Err() == nil {
				w.logger.Printf("music candidate refill delivery failed: %v", err)
			}
			if !didWork {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

type claimed struct {
	Request
	state, token string
	attempts     int
}

func (w *Worker) claim(ctx context.Context) (claimed, bool, error) {
	item := claimed{token: uuid.NewString()}
	now := time.Now().UTC()
	_, err := w.db.ExecContext(ctx, `UPDATE music_catalog_replenish_outbox SET lease_token=?,lease_until=?,attempts=attempts+1 WHERE event_id=(SELECT event_id FROM music_catalog_replenish_outbox WHERE state IN ('pending','cancel_pending') AND available_at<=? AND (lease_until='' OR lease_until<=?) ORDER BY created_at LIMIT 1) AND state IN ('pending','cancel_pending') AND available_at<=? AND (lease_until='' OR lease_until<=?)`, item.token, stamp(now.Add(leaseDuration)), stamp(now), stamp(now), stamp(now), stamp(now))
	if err != nil {
		return claimed{}, false, err
	}
	err = w.db.QueryRowContext(ctx, `SELECT event_id,reservation_id,sha256,purpose,project,revision,style_id,state,attempts FROM music_catalog_replenish_outbox WHERE lease_token=?`, item.token).Scan(&item.RequestID, &item.ReservationID, &item.SHA256, &item.Purpose, &item.Project, &item.Revision, &item.StyleID, &item.state, &item.attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return claimed{}, false, nil
	}
	if err != nil {
		return claimed{}, false, err
	}
	return item, true, nil
}

func (w *Worker) deliverOne(ctx context.Context) (bool, error) {
	item, ok, err := w.claim(ctx)
	if err != nil || !ok {
		return false, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	base, err := w.resolve(requestCtx)
	if err != nil {
		return true, w.retry(ctx, item, fmt.Errorf("resolve music-tools: %w", err), false)
	}
	path, method := compositionconnect.CandidateInventoryServiceRequestRefillProcedure, http.MethodPost
	body := any(item.Request)
	if item.state == "cancel_pending" {
		path, method = compositionconnect.CandidateInventoryServiceCancelRefillProcedure, http.MethodPost
		body = map[string]string{"request_id": item.RequestID}
	}
	data, err := json.Marshal(body)
	if err != nil {
		return true, w.retry(ctx, item, err, false)
	}
	req, err := http.NewRequestWithContext(requestCtx, method, base+path, bytes.NewReader(data))
	if err != nil {
		return true, w.retry(ctx, item, err, false)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.client.Do(req)
	if err != nil {
		return true, w.retry(ctx, item, err, false)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		permanent := resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests
		return true, w.retry(ctx, item, fmt.Errorf("music-tools returned %s", resp.Status), permanent)
	}
	state := "delivered"
	if item.state == "cancel_pending" {
		state = "cancelled"
	}
	_, err = w.db.ExecContext(ctx, `UPDATE music_catalog_replenish_outbox SET state=?,delivered_at=?,lease_token='',lease_until='',last_error='' WHERE event_id=? AND lease_token=? AND state=?`, state, stamp(time.Now().UTC()), item.RequestID, item.token, item.state)
	if err != nil {
		return true, fmt.Errorf("record candidate refill delivery %s: %w", item.RequestID, err)
	}
	return true, nil
}

func (w *Worker) retry(ctx context.Context, item claimed, cause error, permanent bool) error {
	state := item.state
	if permanent {
		state = "blocked"
	}
	delay := 10 * time.Second * time.Duration(1<<min(item.attempts-1, 8))
	if delay > maxRetryDelay {
		delay = maxRetryDelay
	}
	_, err := w.db.ExecContext(ctx, `UPDATE music_catalog_replenish_outbox SET state=?,available_at=?,lease_token='',lease_until='',last_error=? WHERE event_id=? AND lease_token=? AND state=?`, state, stamp(time.Now().UTC().Add(delay)), truncate(cause.Error(), 2000), item.RequestID, item.token, item.state)
	if err != nil {
		return fmt.Errorf("schedule candidate refill retry %s: %v: %w", item.RequestID, cause, err)
	}
	return nil
}

func validateBaseURL(raw string) (string, error) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("invalid music-tools base URL")
	}
	if host := parsed.Hostname(); !strings.EqualFold(host, "localhost") && !net.ParseIP(host).IsLoopback() {
		return "", fmt.Errorf("music-tools URL must resolve to loopback")
	}
	return strings.TrimRight(raw, "/"), nil
}

func stamp(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
