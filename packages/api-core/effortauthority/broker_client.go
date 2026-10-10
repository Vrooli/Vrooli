package effortauthority

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

// BrokerClient uses an enrolled, verified TLS client handle. Setup must isolate
// that handle from native model processes. It performs no enrollment or refresh.
type BrokerClient struct {
	frozen    *BrokerClientConfig
	URL       string
	HTTP      *http.Client
	ServerPin string
	Bindings  map[string]Binding
}

func (c *BrokerClient) call(ctx context.Context, op string, in transaction) (transactionResult, error) {
	c = c.snapshot()
	var out transactionResult
	if c == nil || c.HTTP == nil || c.HTTP.Timeout <= 0 || c.ServerPin == "" {
		return out, ErrRefused
	}
	u, e := url.Parse(c.URL)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.Path != BrokerEndpoint || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return out, ErrRefused
	}
	tr, ok := c.HTTP.Transport.(*http.Transport)
	if !ok || c.HTTP.Jar != nil || tr.DialTLS != nil || tr.DialTLSContext != nil || tr.TLSClientConfig == nil || tr.TLSClientConfig.InsecureSkipVerify || len(tr.TLSClientConfig.Certificates) == 0 {
		return out, ErrRefused
	}
	raw, e := json.Marshal(in)
	if e != nil {
		return out, ErrRefused
	}
	req, e := http.NewRequestWithContext(ctx, "POST", c.URL+op, bytes.NewReader(raw))
	if e != nil {
		return out, ErrRefused
	}
	req.Header.Set("Content-Type", "application/json")
	client := *c.HTTP
	protectedTransport := tr.Clone()
	protectedTLS := tr.TLSClientConfig.Clone()
	previousVerify := protectedTLS.VerifyConnection
	protectedTLS.VerifyConnection = func(state tls.ConnectionState) error {
		if len(state.VerifiedChains) == 0 || len(state.PeerCertificates) == 0 {
			return ErrRefused
		}
		peer := sha256.Sum256(state.PeerCertificates[0].Raw)
		if hex.EncodeToString(peer[:]) != c.ServerPin {
			return ErrRefused
		}
		if previousVerify != nil {
			return previousVerify(state)
		}
		return nil
	}
	protectedTransport.TLSClientConfig = protectedTLS
	client.Transport = protectedTransport
	defer protectedTransport.CloseIdleConnections()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, e := client.Do(req)
	if e != nil {
		return out, ErrRefused
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || resp.TLS == nil || len(resp.TLS.VerifiedChains) == 0 || len(resp.TLS.PeerCertificates) == 0 {
		return out, ErrRefused
	}
	pin := sha256.Sum256(resp.TLS.PeerCertificates[0].Raw)
	if hex.EncodeToString(pin[:]) != c.ServerPin {
		return out, ErrRefused
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if err != nil || len(body) > 65536 {
		return out, ErrRefused
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if dec.Decode(&out) != nil || dec.Decode(new(any)) != io.EOF {
		return transactionResult{}, ErrRefused
	}
	return out, nil
}
func (c *BrokerClient) CheckBinding(ctx context.Context, b Binding) (Policy, error) {
	out, e := c.call(ctx, "check-binding", transaction{Binding: b})
	if e == nil && (out.Policy.Validate() != nil || binding(out.Policy) != b) {
		e = ErrRefused
	}
	return out.Policy, e
}
func (c *BrokerClient) CheckProof(ctx context.Context, p Proof) (Binding, error) {
	out, e := c.call(ctx, "check-proof", transaction{Proof: p})
	return out.Binding, e
}
func (c *BrokerClient) Prepare(ctx context.Context, p Proof, i *Proof) (Reservation, bool, error) {
	out, e := c.call(ctx, "prepare", transaction{Proof: p, Ingress: i})
	return out.Reservation, out.Replay, e
}
func (c *BrokerClient) Reserve(ctx context.Context, b Binding, i Intent, p *Proof) (Reservation, bool, error) {
	out, e := c.call(ctx, "reserve", transaction{Binding: b, Intent: i, Ingress: p})
	return out.Reservation, out.Replay, e
}
func (c *BrokerClient) Accept(ctx context.Context, b Binding, key, runID string) error {
	_, e := c.call(ctx, "accept", transaction{Binding: b, Key: key, RunID: runID})
	return e
}
func (c *BrokerClient) Settle(ctx context.Context, b Binding, key, runID string) error {
	_, e := c.call(ctx, "settle", transaction{Binding: b, Key: key, RunID: runID})
	return e
}
func (c *BrokerClient) ObserveIngress(ctx context.Context, p Proof) (Reservation, bool, error) {
	out, e := c.call(ctx, "observe-ingress", transaction{Proof: p})
	return out.Reservation, out.Replay, e
}
func (c *BrokerClient) ReadReservation(ctx context.Context, b Binding, key string) (Reservation, error) {
	out, e := c.call(ctx, "read-reservation", transaction{Binding: b, Key: key})
	return out.Reservation, e
}
func (c *BrokerClient) SignEffort(ctx context.Context, client string, p Proof) (Proof, error) {
	c = c.snapshot()
	if c == nil || p.Intent.Client != client {
		return Proof{}, ErrRefused
	}
	b, exists := c.Bindings[p.Intent.PolicyID]
	if !exists || b.Client != client {
		return Proof{}, ErrRefused
	}
	out, e := c.call(ctx, "sign", transaction{Binding: b, Intent: p.Intent})
	return out.Proof, e
}

var _ Engine = (*BrokerClient)(nil)
var _ Engine = (*Authority)(nil)

func (c *BrokerClient) PrepareSerialHandoff(ctx context.Context, b Binding, sourceKey string, next Intent) (SerialHandoff, bool, error) {
	out, err := c.call(ctx, "prepare-serial-handoff", transaction{Binding: b, Key: sourceKey, Intent: next})
	if err != nil || out.Handoff == nil || out.Handoff.SourceKey != sourceKey || Digest(out.Handoff.Next) != Digest(next) || out.Handoff.SourceRunID != next.ParentRunID {
		return SerialHandoff{}, false, ErrRefused
	}
	return *out.Handoff, out.Replay, nil
}
func (c *BrokerClient) ReserveSerialEpisode(ctx context.Context, b Binding, key string) (Reservation, bool, error) {
	out, err := c.call(ctx, "reserve-serial-episode", transaction{Binding: b, Key: key})
	if err != nil || !out.Reservation.NativeBound || out.Reservation.Intent.IdempotencyKey != key || out.Reservation.Intent.PolicyID != b.PolicyID || out.Reservation.Intent.Epoch != b.Epoch || out.Reservation.Intent.Client != b.Client || out.Reservation.IntentDigest != Digest(out.Reservation.Intent) {
		return Reservation{}, false, ErrRefused
	}
	return out.Reservation, out.Replay, nil
}
func (c *BrokerClient) ReadSerialHandoff(ctx context.Context, b Binding, key string) (SerialHandoff, error) {
	out, err := c.call(ctx, "read-serial-handoff", transaction{Binding: b, Key: key})
	if err != nil || out.Handoff == nil || out.Handoff.SourceKey == "" || out.Handoff.Next.IdempotencyKey != key || out.Handoff.Next.PolicyID != b.PolicyID || out.Handoff.Next.Epoch != b.Epoch || out.Handoff.Next.Client != b.Client || out.Handoff.SourceRunID != out.Handoff.Next.ParentRunID {
		return SerialHandoff{}, ErrRefused
	}
	return *out.Handoff, nil
}

var _ SerialEngine = (*BrokerClient)(nil)
var _ SerialEngine = (*Authority)(nil)

func (c *BrokerClient) PrepareSerialHandoffWithPayload(ctx context.Context, b Binding, sourceKey string, next Intent, payload json.RawMessage) (SerialHandoff, bool, error) {
	if len(payload) == 0 || len(payload) > 24576 || !json.Valid(payload) || next.SerialPayloadDigest != Digest(string(payload)) {
		return SerialHandoff{}, false, ErrRefused
	}
	out, err := c.call(ctx, "prepare-serial-handoff", transaction{Binding: b, Key: sourceKey, Intent: next, Payload: payload})
	if err != nil || out.Handoff == nil || out.Handoff.SourceKey != sourceKey || Digest(out.Handoff.Next) != Digest(next) || string(out.Handoff.Payload) != string(payload) || out.Handoff.SourceRunID != next.ParentRunID {
		return SerialHandoff{}, false, ErrRefused
	}
	return *out.Handoff, out.Replay, nil
}

var _ SerialPayloadEngine = (*BrokerClient)(nil)
var _ SerialPayloadEngine = (*Authority)(nil)
