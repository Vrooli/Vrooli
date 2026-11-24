package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultStripeEventPageSize = 100
	defaultStripeEventMaxPages = 1000
	stripeEventRetrieveWorkers = 8
	stripeEventRetrieveTimeout = 24 * time.Second
)

// StripeEventReconciliationSource reads the provider's durable Events API and
// returns a deterministic ascending slice. Stripe lists newest-first; this
// source scans back to the persisted cursor (including every event sharing its
// second) before sorting, so pagination boundaries cannot skip a tied event.
type StripeEventReconciliationSource struct {
	requester StripeRequester
	liveMode  bool
	pageSize  int
	maxPages  int
	backfill  *SQLStripeEventBackfillStore
}

// NewDurableStripeEventReconciliationSource persists page progress and event
// references while scanning Stripe's newest-first API. It stores no event
// payloads; payloads are retrieved by ID only after the scan is complete and
// are kept in memory for the shared webhook processor.
func NewDurableStripeEventReconciliationSource(requester StripeRequester, db stripeEventBackfillDB, liveMode bool) (*StripeEventReconciliationSource, error) {
	if requester == nil || db == nil {
		return nil, errors.New("Stripe event reconciliation requester and backfill store are required")
	}
	return &StripeEventReconciliationSource{requester: requester, liveMode: liveMode, pageSize: defaultStripeEventPageSize, maxPages: 1, backfill: &SQLStripeEventBackfillStore{db: db}}, nil
}

func NewStripeEventReconciliationSource(requester StripeRequester, liveMode bool) (*StripeEventReconciliationSource, error) {
	if requester == nil {
		return nil, errors.New("Stripe event reconciliation requester is required")
	}
	return &StripeEventReconciliationSource{requester: requester, liveMode: liveMode, pageSize: defaultStripeEventPageSize, maxPages: defaultStripeEventMaxPages}, nil
}

func (s *StripeEventReconciliationSource) ListAfter(ctx context.Context, cursor PaymentReconciliationCursor, hasCursor bool, limit int) ([]PaymentReconciliationEvent, error) {
	if s == nil || s.requester == nil || ctx == nil || limit <= 0 || limit > 1000 || hasCursor && !cursor.Valid() {
		return nil, errors.New("invalid Stripe event reconciliation query")
	}
	if s.backfill != nil {
		return s.listDurable(ctx, cursor, hasCursor, limit)
	}
	pageSize, maxPages := s.pageSize, s.maxPages
	if pageSize <= 0 || pageSize > defaultStripeEventPageSize {
		pageSize = defaultStripeEventPageSize
	}
	if maxPages <= 0 {
		maxPages = defaultStripeEventMaxPages
	}
	query := url.Values{"limit": {strconv.Itoa(pageSize)}}
	if hasCursor {
		query.Set("created[gte]", strconv.FormatInt(cursor.CreatedAt.UTC().Unix(), 10))
	}
	var events []PaymentReconciliationEvent
	startingAfter := ""
	for page := 0; page < maxPages; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pageQuery := cloneValues(query)
		if startingAfter != "" {
			pageQuery.Set("starting_after", startingAfter)
		}
		body, err := s.requester.Request(ctx, http.MethodGet, "/v1/events?"+pageQuery.Encode(), nil, "")
		if err != nil {
			return nil, fmt.Errorf("list Stripe events: %w", err)
		}
		var pageResult struct {
			Data    []json.RawMessage `json:"data"`
			HasMore bool              `json:"has_more"`
		}
		if err := json.Unmarshal(body, &pageResult); err != nil {
			return nil, fmt.Errorf("decode Stripe event page: %w", err)
		}
		if len(pageResult.Data) == 0 {
			return sortStripeReconciliationEvents(events, limit), nil
		}
		stop := false
		for _, raw := range pageResult.Data {
			event, err := decodeStripeReconciliationEvent(raw)
			if err != nil {
				return nil, err
			}
			startingAfter = event.Cursor.EventID
			if event.LiveMode != s.liveMode {
				return nil, fmt.Errorf("Stripe event %q mode does not match configured reconciliation mode", event.Cursor.EventID)
			}
			if hasCursor && event.Cursor.CreatedAt.Unix() < cursor.CreatedAt.UTC().Unix() {
				stop = true
				break
			}
			if !hasCursor || event.Cursor.Compare(cursor) > 0 {
				events = append(events, event)
			}
		}
		if stop || !pageResult.HasMore {
			return sortStripeReconciliationEvents(events, limit), nil
		}
	}
	return nil, fmt.Errorf("Stripe event reconciliation exceeded %d provider pages without reaching its cursor", maxPages)
}

func (s *StripeEventReconciliationSource) listDurable(ctx context.Context, cursor PaymentReconciliationCursor, hasCursor bool, limit int) ([]PaymentReconciliationEvent, error) {
	if err := s.backfill.PruneThrough(ctx, cursor, hasCursor); err != nil {
		return nil, err
	}
	state, err := s.backfill.State(ctx)
	if err != nil {
		return nil, err
	}
	if state.Complete {
		refs, err := s.backfill.Pending(ctx, cursor, hasCursor, limit)
		if err != nil {
			return nil, err
		}
		if len(refs) == 0 {
			if err := s.backfill.Finish(ctx); err != nil {
				return nil, err
			}
			return nil, nil
		}
		events, err := s.retrieveBatch(ctx, refs)
		if err != nil {
			return nil, err
		}
		return events, nil
	}
	if !state.Active {
		if err := s.backfill.Begin(ctx); err != nil {
			return nil, err
		}
		state = stripeEventBackfillState{Active: true}
	}
	pageSize := s.pageSize
	if pageSize <= 0 || pageSize > defaultStripeEventPageSize {
		pageSize = defaultStripeEventPageSize
	}
	query := url.Values{"limit": {strconv.Itoa(pageSize)}}
	if hasCursor {
		query.Set("created[gte]", strconv.FormatInt(cursor.CreatedAt.UTC().Unix(), 10))
	}
	if state.StartingAfter != "" {
		query.Set("starting_after", state.StartingAfter)
	}
	body, err := s.requester.Request(ctx, http.MethodGet, "/v1/events?"+query.Encode(), nil, "")
	if err != nil {
		return nil, fmt.Errorf("list Stripe event backfill page: %w", err)
	}
	var page struct {
		Data    []json.RawMessage `json:"data"`
		HasMore bool              `json:"has_more"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("decode Stripe event backfill page: %w", err)
	}
	refs := make([]stripeEventBackfillRef, 0, len(page.Data))
	for _, raw := range page.Data {
		event, err := decodeStripeReconciliationEvent(raw)
		if err != nil {
			return nil, err
		}
		if event.LiveMode != s.liveMode {
			return nil, fmt.Errorf("Stripe event %q mode does not match configured reconciliation mode", event.Cursor.EventID)
		}
		if hasCursor && event.Cursor.CreatedAt.Unix() < cursor.CreatedAt.UTC().Unix() {
			page.HasMore = false
			continue
		}
		if !hasCursor || event.Cursor.Compare(cursor) > 0 {
			refs = append(refs, stripeEventBackfillRef{Cursor: event.Cursor, EventID: event.Cursor.EventID, Type: event.Type, ObjectID: event.ObjectID})
		}
		state.StartingAfter = event.Cursor.EventID
	}
	complete := !page.HasMore || len(page.Data) == 0
	if err := s.backfill.AddPage(ctx, refs, state.StartingAfter, complete); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *StripeEventReconciliationSource) retrieveBatch(ctx context.Context, refs []stripeEventBackfillRef) ([]PaymentReconciliationEvent, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	fetchCtx, cancel := context.WithTimeout(ctx, stripeEventRetrieveTimeout)
	defer cancel()
	events := make([]PaymentReconciliationEvent, len(refs))
	jobs := make(chan int)
	errs := make(chan error, len(refs))
	workers := stripeEventRetrieveWorkers
	if workers > len(refs) {
		workers = len(refs)
	}
	var group sync.WaitGroup
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range jobs {
				ref := refs[index]
				body, err := s.requester.Request(fetchCtx, http.MethodGet, "/v1/events/"+url.PathEscape(ref.EventID), nil, "")
				if err != nil {
					errs <- fmt.Errorf("retrieve Stripe reconciliation event %q: %w", ref.EventID, err)
					cancel()
					continue
				}
				event, err := decodeStripeReconciliationEvent(body)
				if err != nil {
					errs <- err
					cancel()
					continue
				}
				if event.LiveMode != s.liveMode || event.Cursor.Compare(ref.Cursor) != 0 || event.Type != ref.Type || event.ObjectID != ref.ObjectID {
					errs <- fmt.Errorf("Stripe event %q changed identity or mode after backfill scan", ref.EventID)
					cancel()
					continue
				}
				events[index] = event
			}
		}()
	}
sendLoop:
	for i := range refs {
		select {
		case jobs <- i:
		case <-fetchCtx.Done():
			break sendLoop
		}
	}
	close(jobs)
	group.Wait()
	close(errs)
	for err := range errs {
		return nil, err
	}
	if err := fetchCtx.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func decodeStripeReconciliationEvent(raw []byte) (PaymentReconciliationEvent, error) {
	var envelope struct {
		ID      string          `json:"id"`
		Type    string          `json:"type"`
		Created int64           `json:"created"`
		Live    bool            `json:"livemode"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return PaymentReconciliationEvent{}, fmt.Errorf("decode Stripe event envelope: %w", err)
	}
	var data struct {
		Object struct {
			ID string `json:"id"`
		} `json:"object"`
	}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return PaymentReconciliationEvent{}, fmt.Errorf("decode Stripe event object identity: %w", err)
	}
	if envelope.ID == "" || envelope.Type == "" || envelope.Created <= 0 || strings.TrimSpace(data.Object.ID) == "" {
		return PaymentReconciliationEvent{}, errors.New("Stripe event is missing its ID, type, creation time or object ID")
	}
	return PaymentReconciliationEvent{
		Cursor:   PaymentReconciliationCursor{CreatedAt: time.Unix(envelope.Created, 0).UTC(), EventID: envelope.ID},
		LiveMode: envelope.Live, Type: envelope.Type, ObjectID: data.Object.ID, Payload: append([]byte(nil), raw...),
	}, nil
}

func sortStripeReconciliationEvents(events []PaymentReconciliationEvent, limit int) []PaymentReconciliationEvent {
	sort.Slice(events, func(i, j int) bool { return events[i].Cursor.Compare(events[j].Cursor) < 0 })
	if len(events) > limit {
		events = events[:limit]
	}
	return events
}

func cloneValues(values url.Values) url.Values {
	cloned := make(url.Values, len(values))
	for key, value := range values {
		cloned[key] = append([]string(nil), value...)
	}
	return cloned
}

// StripeWebhookReconciliationProcessor reuses the exact handler and event
// receipt path used for provider webhooks.
type StripeWebhookReconciliationProcessor struct {
	Webhook *StripeWebhookService
}

func (p StripeWebhookReconciliationProcessor) Apply(ctx context.Context, event PaymentReconciliationEvent) (PaymentReconciliationOutcome, error) {
	if p.Webhook == nil {
		return "", errors.New("Stripe webhook reconciliation processor is unavailable")
	}
	return p.Webhook.ApplyReconciliationEvent(ctx, event)
}
