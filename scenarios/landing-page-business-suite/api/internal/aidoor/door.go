package aidoor

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/commerce"
)

type Door struct {
	ObserveTransition func(RequestTransition)
	Keys              KeyStore
	Wallet            *commerce.BusinessAccountUsageService
	ResolveGateway    func(context.Context) (string, error)
	GatewayURL        string
	Client            *http.Client
}

const agentSessionFeature = "agent.session"

var errKeyCapAdmission = errors.New("key cap admission failed")

type plan struct {
	OperationID    string `json:"operation_id"`
	FeatureID      string `json:"feature_id"`
	RequestedModel string `json:"requested_model"`
	BudgetMicros   int64  `json:"budget_micros"`
	Eligible       bool   `json:"eligible"`
	Harness        string `json:"harness,omitempty"`
	QuotedCredits  int64  `json:"quoted_credits"`
	StreamQuote    struct {
		MaxInputBytes   int    `json:"max_input_bytes"`
		MaxOutputTokens int    `json:"max_output_tokens"`
		MaxCostMicros   int64  `json:"max_cost_micros"`
		TokenLimitField string `json:"token_limit_field"`
	} `json:"stream_quote"`
	CreditMultiplier int64  `json:"credit_multiplier"`
	RouteID          string `json:"route_id,omitempty"`
	Supplier         string `json:"supplier,omitempty"`
	Model            string `json:"model,omitempty"`
	Reason           string `json:"reason,omitempty"`
}
type receipt struct {
	OperationID      string `json:"operation_id"`
	FeatureID        string `json:"feature_id"`
	Content          string `json:"content,omitempty"`
	State            string `json:"state"`
	Credits          int64  `json:"credits"`
	CostMicros       int64  `json:"cost_micros"`
	CostKnown        bool   `json:"cost_known"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	Model            string `json:"model"`
	Supplier         string `json:"supplier"`
}

func (d *Door) client() *http.Client {
	client := http.Client{}
	if d.Client != nil {
		client = *d.Client
	}
	// Private Gateway calls have exactly one target. A redirected dial failure
	// must never be mistaken for failure to dispatch to the original target.
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &client
}

func fail(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": message}})
}

func (d *Door) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key, err := d.Keys.Authenticate(r.Context(), r.Header.Get("Authorization"))
	if err != nil {
		fail(w, 401, err.Error())
		return
	}
	switch {
	case r.Method == "GET" && r.URL.Path == "/api/v1/ai/models":
		d.models(w, r, key)
	case r.Method == "POST" && r.URL.Path == "/api/v1/ai/keys/session":
		d.session(w, r, key)
	case r.Method == "DELETE" && strings.HasPrefix(r.URL.Path, "/api/v1/ai/keys/"):
		if err := d.Keys.Revoke(r.Context(), key, strings.TrimPrefix(r.URL.Path, "/api/v1/ai/keys/")); err != nil {
			fail(w, 403, err.Error())
			return
		}
		w.WriteHeader(204)
	case r.Method == "POST" && r.URL.Path == "/api/v1/ai/responses":
		d.responses(w, r, key)
	case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/api/v1/features/"):
		d.feature(w, r, key)
	case r.Method == "POST" && r.URL.Path == "/api/v1/ai/chat/completions":
		d.chat(w, r, key)
	default:
		fail(w, 404, "route not found")
	}
}

func (d *Door) session(w http.ResponseWriter, r *http.Request, key Key) {
	var input struct {
		Harness    string `json:"harness"`
		Model      string `json:"model"`
		MaxCredits int64  `json:"max_credits"`
		TTL        int64  `json:"ttl_seconds"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&input) != nil {
		fail(w, 400, "invalid session request")
		return
	}
	session, err := d.Keys.IssueSession(r.Context(), key, input.Harness, input.Model, input.MaxCredits, time.Duration(input.TTL)*time.Second)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(session)
}

func (d *Door) post(ctx context.Context, path string, input any) (*http.Response, error) {
	base := d.GatewayURL
	if d.ResolveGateway != nil {
		var err error
		base, err = d.ResolveGateway(ctx)
		if err != nil {
			return nil, err
		}
	}
	body, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(base, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return d.client().Do(req)
}

func (d *Door) chat(w http.ResponseWriter, r *http.Request, key Key) {
	var input map[string]json.RawMessage
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if decoder.Decode(&input) != nil {
		fail(w, 400, "invalid chat request")
		return
	}
	var model string
	if json.Unmarshal(input["model"], &model) != nil || model == "" || (key.Model != "" && key.Model != model) {
		fail(w, 403, "model is outside key scope")
		return
	}
	for _, name := range []string{"outside_funded", "business_account_id", "funded", "metadata", "use_class", "harness"} {
		delete(input, name)
	}
	d.funded(w, r, key, agentSessionFeature, model, input, nil)
}

func (d *Door) funded(w http.ResponseWriter, r *http.Request, key Key, featureID, model string, input map[string]json.RawMessage, feature json.RawMessage) {
	operation := r.Header.Get("Idempotency-Key")
	if operation == "" {
		operation = randomID()
	}
	digest := sha256.Sum256([]byte(key.ID + ":" + operation))
	operation = fmt.Sprintf("%x-%x-%x-%x-%x", digest[:4], digest[4:6], digest[6:8], digest[8:10], digest[10:16])
	flow := d.requestLifecycle(operation, requestAccepted)
	defer flow.finish()
	tier, multiplier, err := d.Wallet.AccountCreditScope(r.Context(), key.Account, key.App)
	if err != nil {
		fail(w, 503, "account credit scope unavailable")
		return
	}
	budget := new(big.Int).Mul(big.NewInt(key.MaxCredits), big.NewInt(10000))
	budget.Div(budget, big.NewInt(multiplier))
	if !budget.IsInt64() {
		fail(w, 400, "credit cap exceeds supported quote range")
		return
	}
	harness := key.Harness
	planResponse, err := d.post(r.Context(), "/api/v1/private/provider-routes/plan", map[string]any{"operation_id": operation, "feature_id": featureID, "model": model, "budget_micros": budget.Int64(), "harness": harness, "credit_multiplier": multiplier})
	if err != nil {
		fail(w, 503, "AI Gateway unavailable")
		return
	}
	defer planResponse.Body.Close()
	var selected plan
	if json.NewDecoder(planResponse.Body).Decode(&selected) != nil || planResponse.StatusCode != 200 {
		fail(w, 502, "provider plan unavailable")
		return
	}
	if !selected.Eligible || selected.QuotedCredits <= 0 || selected.QuotedCredits > key.MaxCredits {
		fail(w, 403, "no eligible provider within key cap")
		return
	}
	if err := flow.advance(requestPlan); err != nil {
		fail(w, 500, err.Error())
		return
	}
	_, created, err := d.Wallet.ReserveScopedCreditsWithAdmission(r.Context(), key.Account, key.Actor, tier, "ai_credits", key.App, selected.QuotedCredits, operation, &commerce.ReservationAdmission{
		Reserve: func(ctx context.Context, tx *sql.Tx) error {
			if err := d.Keys.ReserveCap(ctx, tx, key, operation, selected.QuotedCredits); err != nil {
				return fmt.Errorf("%w: %w", errKeyCapAdmission, err)
			}
			return nil
		},
		Reject: func(ctx context.Context, tx *sql.Tx) error { return d.Keys.RejectCap(ctx, tx, operation) },
	})
	if err != nil {
		status := http.StatusConflict
		if errors.Is(err, errKeyCapAdmission) && !errors.Is(err, ErrKeyOperationTerminal) {
			status = http.StatusPaymentRequired
		}
		if strings.Contains(err.Error(), "insufficient") {
			fail(w, 402, "insufficient credits")
		} else {
			fail(w, status, err.Error())
		}
		return
	}
	if err := flow.advance(requestReserve); err != nil {
		fail(w, 500, err.Error())
		return
	}
	if !created {
		if err := flow.recover(r.Context(), key); err != nil {
			fail(w, 409, "operation already dispatched; receipt pending")
		} else {
			fail(w, 409, "operation already settled")
		}
		return
	}
	if err := d.Wallet.RetainUncertainReservation(r.Context(), key.Account, operation); err != nil {
		fail(w, 503, "reservation durability unavailable")
		return
	}
	chat, _ := json.Marshal(input)
	// Provider receipt ownership survives client cancellation; never retry an
	// unknown dispatch. The same durable operation identifies the wallet hold.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 16*time.Minute)
	defer cancel()
	if err := flow.advance(requestDispatch); err != nil {
		fail(w, 500, err.Error())
		return
	}
	upstream, err := d.post(ctx, "/api/v1/private/provider-routes/stream", map[string]any{"operation_id": operation, "plan": selected, "chat": json.RawMessage(chat), "feature": feature})
	if err != nil {
		var dial *net.OpError
		var dns *net.DNSError
		if (errors.As(err, &dial) && dial.Op == "dial") || errors.As(err, &dns) {
			// No connection was established, so Gateway could not claim/dispatch
			// this operation. Other transport errors retain the uncertain hold.
			if releaseErr := d.Wallet.ReleaseReservation(ctx, key.Account, operation); releaseErr == nil {
				_ = d.Keys.SettleCap(ctx, operation, 0)
				_ = flow.advance(requestRelease)
			}
		}
		fail(w, 502, "provider outcome pending reconciliation")
		return
	}
	defer upstream.Body.Close()
	if upstream.StatusCode == 409 {
		if d.Wallet.ReleaseReservation(ctx, key.Account, operation) == nil {
			_ = d.Keys.SettleCap(ctx, operation, 0)
			_ = flow.advance(requestRelease)
		}
	}
	if upstream.StatusCode != 200 {
		fail(w, 502, "provider refused execution; receipt pending reconciliation")
		return
	}
	if !strings.Contains(upstream.Header.Get("Content-Type"), "text/event-stream") {
		var result receipt
		if json.NewDecoder(upstream.Body).Decode(&result) == nil {
			if err := flow.settle(ctx, key, result); err == nil && feature != nil && result.State == "completed" {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Operation-ID", operation)
				_ = json.NewEncoder(w).Encode(result)
				return
			}
		}
		fail(w, 502, "provider did not stream")
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Operation-ID", operation)
	scanner := bufio.NewScanner(upstream.Body)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	privateEvent := false
	delivery := true
	for scanner.Scan() {
		line := scanner.Text()
		if line == "event: receipt" {
			privateEvent = true
			continue
		}
		if privateEvent {
			if strings.HasPrefix(line, "data:") {
				var result receipt
				if json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), &result) == nil {
					_ = flow.settle(ctx, key, result)
				}
			}
			if line == "" {
				privateEvent = false
			}
			continue
		}
		if strings.TrimSpace(line) == "data: [DONE]" && flow.state != requestSettled {
			continue
		}
		if delivery {
			_, err = io.WriteString(w, line+"\n")
			delivery = err == nil
			if !delivery && flow.state == requestStreaming {
				_ = flow.advance(requestDisconnect)
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
	if flow.state != requestSettled {
		_ = flow.recover(context.Background(), key)
	}
}

func (d *Door) settle(ctx context.Context, key Key, result receipt) error {
	if !result.CostKnown {
		return errors.New("cost unknown")
	}
	if result.State == "failed" && result.Credits == 0 {
		if err := d.Wallet.ReleaseReservation(ctx, key.Account, result.OperationID); err != nil {
			return err
		}
		return d.Keys.SettleCap(ctx, result.OperationID, 0)
	}
	if result.State != "completed" && result.State != "unknown" {
		return errors.New("nonterminal receipt")
	}
	if err := d.Wallet.FinalizeFeatureReservationWithCost(ctx, key.Account, result.OperationID, result.Credits, key.App, result.Model, result.CostMicros, result.PromptTokens, result.CompletionTokens, result.Supplier, result.FeatureID); err != nil {
		return err
	}
	var charged int64
	if err := d.Keys.DB.QueryRowContext(ctx, `SELECT charged_amount FROM business_account_credit_reservations WHERE id=$1 AND business_account_id=$2`, result.OperationID, key.Account).Scan(&charged); err != nil {
		return err
	}
	return d.Keys.SettleCap(ctx, result.OperationID, charged)
}

func (d *Door) recover(ctx context.Context, key Key, operation string) error {
	return d.requestLifecycle(operation, requestReconciling).recover(ctx, key)
}

func (f *requestLifecycle) recover(ctx context.Context, key Key) error {
	if f.state == requestReserved {
		if err := f.advance(requestRecover); err != nil {
			return err
		}
	} else if f.state != requestReconciling {
		if err := f.advance(requestDefer); err != nil {
			return err
		}
	}
	d := f.door
	base := d.GatewayURL
	if d.ResolveGateway != nil {
		var err error
		base, err = d.ResolveGateway(ctx)
		if err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(base, "/")+"/api/v1/private/provider-routes/receipt/"+f.operation, nil)
	if err != nil {
		return err
	}
	resp, err := d.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("receipt HTTP %d", resp.StatusCode)
	}
	var result receipt
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	return f.settle(ctx, key, result)
}

func (d *Door) models(w http.ResponseWriter, r *http.Request, key Key) {
	harness := key.Harness
	if harness == "" {
		harness = r.URL.Query().Get("harness")
	}
	base := d.GatewayURL
	if d.ResolveGateway != nil {
		var err error
		base, err = d.ResolveGateway(r.Context())
		if err != nil {
			fail(w, 503, "Gateway unavailable")
			return
		}
	}
	req, err := http.NewRequestWithContext(r.Context(), "GET", strings.TrimRight(base, "/")+"/api/v1/private/provider-routes/models?harness="+url.QueryEscape(harness), nil)
	if err != nil {
		fail(w, 503, "Gateway unavailable")
		return
	}
	response, err := d.client().Do(req)
	if err != nil {
		fail(w, 503, "Gateway unavailable")
		return
	}
	defer response.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(response.Body, 1<<20))
}
