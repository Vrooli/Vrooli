// Package eventbus supplies the bounded, best-effort Vrooli Events client used
// by the standard API Core receipt runtime.
package eventbus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const ReceiptEventType = "vrooli.events.receipt.v1"

type Correlation struct {
	RequestID           string
	RunID               string
	TaskID              string
	WorkflowExecutionID string
	WorkflowNodeID      string
	Attempt             uint32
}

type Receipt struct {
	Source, Target, Operation string
	Outcome                   string
	StatusCode                int
	Duration                  time.Duration
	PolicyVer                 string
	Projection                map[string]any
	Correlation               Correlation
	SubjectID                 string
	ActorKind                 string
	IdentityToken             string
}

func (r Receipt) IdempotencyKey() string {
	b := fmt.Sprintf("%s\x00%s\x00%s\x00%d\x00%s\x00%d", r.Target, r.Operation, r.Correlation.RunID, r.StatusCode, r.Correlation.RequestID, r.Correlation.Attempt)
	s := sha256.Sum256([]byte(b))
	return "evt_" + hex.EncodeToString(s[:])
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	endpoint   *endpointRef
}

type endpointRef struct {
	mu  sync.RWMutex
	url string
}

func newDynamicClient(baseURL string) (Client, func(string)) {
	endpoint := &endpointRef{url: strings.TrimSpace(baseURL)}
	return Client{endpoint: endpoint}, func(next string) {
		endpoint.mu.Lock()
		endpoint.url = strings.TrimSpace(next)
		endpoint.mu.Unlock()
	}
}

func (c Client) baseURL() string {
	if c.endpoint == nil {
		return strings.TrimSpace(c.BaseURL)
	}
	c.endpoint.mu.RLock()
	defer c.endpoint.mu.RUnlock()
	return c.endpoint.url
}

func (c Client) Enabled() bool { return c.baseURL() != "" }

func (c Client) PublishAsync(receipt Receipt) <-chan error {
	done := make(chan error, 1)
	if !c.Enabled() {
		done <- nil
		close(done)
		return done
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		err := c.Publish(ctx, receipt)
		if err != nil {
			log.Printf("vrooli-events receipt delivery degraded: target=%s operation=%s error=%v", receipt.Target, receipt.Operation, err)
		}
		done <- err
		close(done)
	}()
	return done
}

func (c Client) Publish(ctx context.Context, receipt Receipt) error {
	if !c.Enabled() {
		return nil
	}
	projection, err := structpb.NewStruct(receipt.Projection)
	if err != nil {
		return fmt.Errorf("receipt projection: %w", err)
	}
	data, err := anypb.New(&domain.ReceiptData{Outcome: receipt.Outcome, StatusCode: uint32(receipt.StatusCode), DurationMs: uint64(receipt.Duration.Milliseconds()), PolicyVersion: receipt.PolicyVer, IdempotencyKey: receipt.IdempotencyKey(), Projection: projection})
	if err != nil {
		return fmt.Errorf("pack receipt data: %w", err)
	}
	actorKind := receipt.ActorKind
	if actorKind == "" {
		actorKind = "system"
	}
	subjectKind := ""
	if receipt.SubjectID != "" {
		subjectKind = "agent"
	}
	env := &domain.EventEnvelope{EventId: receipt.IdempotencyKey(), EventType: ReceiptEventType, OccurredAt: timestamppb.Now(), Source: &domain.EventSource{Scenario: receipt.Source, ActorKind: actorKind}, Target: &domain.EventTarget{Scenario: receipt.Target, Operation: receipt.Operation, Protocol: "connect"}, Correlation: &domain.EventCorrelation{RequestId: receipt.Correlation.RequestID, AgentRunId: receipt.Correlation.RunID, TaskId: receipt.Correlation.TaskID, WorkflowExecutionId: receipt.Correlation.WorkflowExecutionID, WorkflowNodeId: receipt.Correlation.WorkflowNodeID, Attempt: receipt.Correlation.Attempt}, Data: data}
	// Attribution is absent, not an empty object, when the request did not carry
	// verified Agent Manager provenance. Events uses that distinction to prevent
	// callers from creating an apparently attributed receipt without proof.
	if receipt.SubjectID != "" {
		env.Attribution = &domain.EventAttribution{SubjectKind: subjectKind, SubjectId: receipt.SubjectID, Verified: true}
	}
	body, err := (protojson.MarshalOptions{}).Marshal(env)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.baseURL(), "/")+"/api/v1/events", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token := strings.TrimSpace(receipt.IdentityToken); token != "" {
		req.Header.Set(cliutil.HeaderAgentIdentityToken, token)
	}
	h := c.HTTPClient
	if h == nil {
		h = http.DefaultClient
	}
	resp, err := h.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("vrooli-events receipt publish: %s", resp.Status)
	}
	return nil
}

func (c Client) ReceiptQuery(ctx context.Context, runID string, limit int) ([]json.RawMessage, error) {
	if !c.Enabled() {
		return []json.RawMessage{}, nil
	}
	if limit < 1 || limit > 100 {
		limit = 100
	}
	u, err := url.Parse(strings.TrimRight(c.baseURL(), "/") + "/api/v1/events")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("type", ReceiptEventType)
	q.Set("agent_run_id", runID)
	q.Set("limit", fmt.Sprint(limit))
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	h := c.HTTPClient
	if h == nil {
		h = http.DefaultClient
	}
	resp, err := h.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vrooli-events receipt query: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var observations []json.RawMessage
	if err := json.Unmarshal(body, &observations); err != nil {
		return nil, fmt.Errorf("decode vrooli-events receipt query: %w", err)
	}
	return observations, nil
}
