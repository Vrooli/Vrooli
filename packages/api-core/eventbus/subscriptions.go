package eventbus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const subscriptionsPath = "/api/v1/subscriptions"
const subscriptionRequestTimeout = 10 * time.Second
const maxSubscriptionErrorBody = 4096

var ErrInvalidSubscriptionsURL = errors.New("invalid vrooli-events subscriptions URL")

// Subscription is the consumer-facing wire projection for Vrooli Events'
// durable subscription API. Vrooli Events remains authoritative for validation
// and persistence; this type only defines the JSON fields used by API callers.
type Subscription struct {
	ID             int64     `json:"id,omitempty"`
	Name           string    `json:"name"`
	OwnerScenario  string    `json:"owner_scenario"`
	EventPattern   string    `json:"event_pattern"`
	SourceFilter   string    `json:"source_filter,omitempty"`
	DeliveryType   string    `json:"delivery_type"`
	DeliveryTarget string    `json:"delivery_target"`
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

// SubscriptionClient owns HTTP/JSON mechanics and deterministic named-route
// reconciliation for the durable subscription API. Callers resolve
// vrooli-events immediately before constructing the client and retain ownership
// of periodic retries and lifecycle.
type SubscriptionClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewSubscriptionClient creates a client for an explicit, current Events API
// base. It does not perform or cache scenario discovery. A nil HTTP client gets
// a standard client; callers may inject a configured transport. Each request
// still has the package's ten-second total deadline, capped by the caller's
// earlier deadline.
func NewSubscriptionClient(baseURL string, httpClient *http.Client) (*SubscriptionClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("%w: %q", ErrInvalidSubscriptionsURL, baseURL)
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &SubscriptionClient{baseURL: baseURL, httpClient: httpClient}, nil
}

// ListSubscriptions lists Events-owned subscriptions, optionally filtered by
// owner. An empty owner requests the complete list.
func (c *SubscriptionClient) ListSubscriptions(ctx context.Context, owner string) ([]Subscription, error) {
	endpoint := c.baseURL + subscriptionsPath
	if owner != "" {
		query := url.Values{"owner": []string{owner}}
		endpoint += "?" + query.Encode()
	}
	var subscriptions []Subscription
	err := c.do(ctx, http.MethodGet, endpoint, nil, func(response *http.Response) error {
		return json.NewDecoder(response.Body).Decode(&subscriptions)
	})
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	if subscriptions == nil {
		subscriptions = []Subscription{}
	}
	return subscriptions, nil
}

// EnsureNamedSubscription reconciles one caller-owned subscription using the
// stable (owner_scenario, name) identity. For migration compatibility it also
// adopts an unnamed row only when its complete route projection already matches
// the desired one. It creates a missing row, repairs the lowest-ID matching
// row to the full desired projection, and disables enabled duplicates. Rows
// with another name or owner are never mutated, even when they share a target.
// Caller-specific discovery and retry lifecycle stay outside this client.
func (c *SubscriptionClient) EnsureNamedSubscription(ctx context.Context, desired Subscription) error {
	_, err := c.EnsureNamedSubscriptions(ctx, []Subscription{desired})
	return err
}

// EnsureNamedSubscriptionWithExactRouteAdoption applies the same policy as
// EnsureNamedSubscription, and also adopts a differently named row whose full
// owner and route projection exactly matches desired. Use this only when the
// caller has a documented legacy identity represented by that exact tuple.
func (c *SubscriptionClient) EnsureNamedSubscriptionWithExactRouteAdoption(ctx context.Context, desired Subscription) error {
	_, err := c.ensureNamedSubscriptions(ctx, []Subscription{desired}, true)
	return err
}

// EnsureNamedSubscriptions reconciles several named routes for one owner with
// one Events inventory read. It returns the refreshed inventory for callers
// that also retire old named routes after replacements are ready. Writes are
// sequential and durable; if a later write fails, earlier writes remain and a
// retry of the full desired set converges safely.
func (c *SubscriptionClient) EnsureNamedSubscriptions(ctx context.Context, desired []Subscription) ([]Subscription, error) {
	return c.ensureNamedSubscriptions(ctx, desired, false)
}

func (c *SubscriptionClient) ensureNamedSubscriptions(ctx context.Context, desired []Subscription, adoptAnyExactRoute bool) ([]Subscription, error) {
	if len(desired) == 0 {
		return []Subscription{}, nil
	}
	owner := strings.TrimSpace(desired[0].OwnerScenario)
	if owner == "" {
		return nil, errors.New("subscription owner is required")
	}
	names := make(map[string]struct{}, len(desired))
	for _, subscription := range desired {
		if subscription.OwnerScenario != owner || strings.TrimSpace(subscription.Name) == "" {
			return nil, errors.New("named subscriptions require one owner and nonempty names")
		}
		if _, duplicate := names[subscription.Name]; duplicate {
			return nil, fmt.Errorf("duplicate desired subscription name %q", subscription.Name)
		}
		names[subscription.Name] = struct{}{}
	}
	rows, err := c.ListSubscriptions(ctx, owner)
	if err != nil {
		return nil, err
	}
	for _, subscription := range desired {
		rows, err = c.ensureNamedSubscription(ctx, rows, subscription, adoptAnyExactRoute)
		if err != nil {
			return nil, err
		}
	}
	return rows, nil
}

func (c *SubscriptionClient) ensureNamedSubscription(ctx context.Context, rows []Subscription, desired Subscription, adoptAnyExactRoute bool) ([]Subscription, error) {
	indices := make([]int, 0, 1)
	for index, row := range rows {
		namedOwner := row.OwnerScenario == desired.OwnerScenario && row.Name == desired.Name
		legacyExactRoute := row.OwnerScenario == desired.OwnerScenario && (row.Name == "" || adoptAnyExactRoute) && row.EventPattern == desired.EventPattern && row.SourceFilter == desired.SourceFilter && row.DeliveryType == desired.DeliveryType && row.DeliveryTarget == desired.DeliveryTarget
		if namedOwner || legacyExactRoute {
			indices = append(indices, index)
		}
	}
	if len(indices) == 0 {
		id, err := c.CreateSubscription(ctx, desired)
		if err != nil {
			return rows, err
		}
		if id <= 0 {
			return rows, fmt.Errorf("create named subscription %q: Events returned no row ID", desired.Name)
		}
		created := desired
		created.ID = id
		return append(rows, created), nil
	}
	seenIDs := make(map[int64]struct{}, len(indices))
	for _, index := range indices {
		row := rows[index]
		if row.ID <= 0 {
			return rows, fmt.Errorf("reconcile named subscription %q: row is missing an ID", desired.Name)
		}
		if _, exists := seenIDs[row.ID]; exists {
			return rows, fmt.Errorf("reconcile named subscription %q: duplicate row ID %d", desired.Name, row.ID)
		}
		seenIDs[row.ID] = struct{}{}
	}
	sort.Slice(indices, func(i, j int) bool { return rows[indices[i]].ID < rows[indices[j]].ID })
	for _, index := range indices[1:] {
		duplicate := rows[index]
		if !duplicate.Enabled {
			continue
		}
		duplicate.Enabled = false
		if err := c.UpdateSubscription(ctx, duplicate.ID, duplicate); err != nil {
			return rows, fmt.Errorf("disable duplicate named subscription %q: %w", desired.Name, err)
		}
		rows[index] = duplicate
	}
	canonicalIndex := indices[0]
	canonical := rows[canonicalIndex]
	if canonical.Name != desired.Name || canonical.OwnerScenario != desired.OwnerScenario || canonical.EventPattern != desired.EventPattern || canonical.SourceFilter != desired.SourceFilter || canonical.DeliveryType != desired.DeliveryType || canonical.DeliveryTarget != desired.DeliveryTarget || canonical.Enabled != desired.Enabled {
		if err := c.UpdateSubscription(ctx, canonical.ID, desired); err != nil {
			return rows, fmt.Errorf("repair named subscription %q: %w", desired.Name, err)
		}
		canonical = desired
		canonical.ID = rows[canonicalIndex].ID
		rows[canonicalIndex] = canonical
	}
	return rows, nil
}

// CreateSubscription creates one durable subscription and returns its Events
// assigned identifier.
func (c *SubscriptionClient) CreateSubscription(ctx context.Context, subscription Subscription) (int64, error) {
	var result struct {
		ID int64 `json:"id"`
	}
	if err := c.do(ctx, http.MethodPost, c.baseURL+subscriptionsPath, subscriptionRequestBody(subscription), func(response *http.Response) error {
		return json.NewDecoder(response.Body).Decode(&result)
	}); err != nil {
		return 0, fmt.Errorf("create subscription: %w", err)
	}
	return result.ID, nil
}

// UpdateSubscription replaces one durable subscription using the API's
// full-resource PUT semantics.
func (c *SubscriptionClient) UpdateSubscription(ctx context.Context, id int64, subscription Subscription) error {
	if id <= 0 {
		return fmt.Errorf("update subscription: invalid id %d", id)
	}
	endpoint := c.baseURL + subscriptionsPath + "/" + strconv.FormatInt(id, 10)
	if err := c.do(ctx, http.MethodPut, endpoint, subscriptionRequestBody(subscription), nil); err != nil {
		return fmt.Errorf("update subscription %d: %w", id, err)
	}
	return nil
}

// DeleteSubscription removes one durable subscription by its Events assigned
// identifier.
func (c *SubscriptionClient) DeleteSubscription(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("delete subscription: invalid id %d", id)
	}
	endpoint := c.baseURL + subscriptionsPath + "/" + strconv.FormatInt(id, 10)
	if err := c.do(ctx, http.MethodDelete, endpoint, nil, nil); err != nil {
		return fmt.Errorf("delete subscription %d: %w", id, err)
	}
	return nil
}

// ReconcileDelivery asks Events to enqueue one retained event for a matching
// enabled webhook subscription. Delivery idempotency and event retention remain
// owned by Vrooli Events.
func (c *SubscriptionClient) ReconcileDelivery(ctx context.Context, id int64, eventID string) error {
	if id <= 0 || strings.TrimSpace(eventID) == "" || strings.TrimSpace(eventID) != eventID {
		return fmt.Errorf("reconcile subscription delivery: invalid subscription or event identity")
	}
	endpoint := c.baseURL + subscriptionsPath + "/" + strconv.FormatInt(id, 10) + "/deliveries/reconcile"
	if err := c.do(ctx, http.MethodPost, endpoint, struct {
		EventID string `json:"event_id"`
	}{EventID: eventID}, nil); err != nil {
		return fmt.Errorf("reconcile subscription %d delivery: %w", id, err)
	}
	return nil
}

func (c *SubscriptionClient) do(ctx context.Context, method, endpoint string, body any, consume func(*http.Response) error) error {
	if c == nil || c.httpClient == nil {
		return errors.New("subscription client is not configured")
	}
	if ctx == nil {
		return errors.New("subscription request context is required")
	}
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		payload = bytes.NewReader(encoded)
	}
	requestCtx, cancel := context.WithTimeout(ctx, subscriptionRequestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, method, endpoint, payload)
	if err != nil {
		return err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		errorBody, _ := io.ReadAll(io.LimitReader(response.Body, maxSubscriptionErrorBody+1))
		truncated := len(errorBody) > maxSubscriptionErrorBody
		if truncated {
			errorBody = errorBody[:maxSubscriptionErrorBody]
		}
		var apiError struct {
			Code    string `json:"code"`
			Message string `json:"error"`
		}
		_ = json.Unmarshal(errorBody, &apiError)
		return &SubscriptionHTTPError{
			Method: method, StatusCode: response.StatusCode, Status: response.Status,
			Code: apiError.Code, Message: apiError.Message, BodyTruncated: truncated,
		}
	}
	if consume != nil {
		return consume(response)
	}
	return nil
}

func subscriptionRequestBody(subscription Subscription) any {
	return struct {
		Name           string `json:"name"`
		OwnerScenario  string `json:"owner_scenario"`
		EventPattern   string `json:"event_pattern"`
		SourceFilter   string `json:"source_filter,omitempty"`
		DeliveryType   string `json:"delivery_type"`
		DeliveryTarget string `json:"delivery_target"`
		Enabled        bool   `json:"enabled"`
	}{
		Name: subscription.Name, OwnerScenario: subscription.OwnerScenario,
		EventPattern: subscription.EventPattern, SourceFilter: subscription.SourceFilter,
		DeliveryType: subscription.DeliveryType, DeliveryTarget: subscription.DeliveryTarget,
		Enabled: subscription.Enabled,
	}
}

// SubscriptionHTTPError reports a non-success response from Vrooli Events.
type SubscriptionHTTPError struct {
	Method        string
	StatusCode    int
	Status        string
	Code          string
	Message       string
	BodyTruncated bool
}

func (e *SubscriptionHTTPError) Error() string {
	message := fmt.Sprintf("vrooli-events %s %s", e.Method, e.Status)
	if e.Code != "" {
		message += " (" + e.Code + ")"
	}
	if e.Message != "" {
		message += ": " + strconv.Quote(e.Message)
	}
	if e.BodyTruncated {
		message += " [error body truncated]"
	}
	return message
}
