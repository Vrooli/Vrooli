package eventbus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

const (
	TerminalAdmissionPath             = "/api/v1/internal/terminal-admissions"
	TerminalAdmissionSchema           = "vrooli-events.terminal-admission.v1"
	TerminalAdmissionMaxRequestBytes  = 32 * 1024
	TerminalAdmissionMaxResponseBytes = 1024
)

// TerminalAdmissionRequest carries retained facts only. Neither a source claim
// in the seed nor the presence of this request confers admission authority.
type TerminalAdmissionRequest struct {
	Schema     string `json:"schema"`
	EventID    string `json:"event_id"`
	IntentJSON string `json:"intent_json"`
}

// DecodeTerminalAdmissionRequest requires the exact bounded encoding emitted by
// the client, rejecting duplicate/unknown fields and alternate interpretations.
func DecodeTerminalAdmissionRequest(body []byte) (TerminalAdmissionRequest, error) {
	var request TerminalAdmissionRequest
	if len(body) == 0 || len(body) > TerminalAdmissionMaxRequestBytes {
		return request, errors.New("invalid terminal admission request size")
	}
	if err := json.Unmarshal(body, &request); err != nil {
		return TerminalAdmissionRequest{}, errors.New("invalid terminal admission request")
	}
	canonical, err := json.Marshal(request)
	if err != nil || !bytes.Equal(body, canonical) || request.Schema != TerminalAdmissionSchema {
		return TerminalAdmissionRequest{}, errors.New("invalid terminal admission encoding or schema")
	}
	if _, _, err := RetainedDomainEventEnvelope(request.IntentJSON, request.EventID); err != nil {
		return TerminalAdmissionRequest{}, err
	}
	return request, nil
}

// TerminalAdmissionClient has no discovery, credential provider, cookies,
// redirects, automatic retry, or default transport. Plain HTTP is restricted to
// explicit loopback; remote origins require HTTPS. Construction performs no I/O.
// The injected transport is a trusted owner capability: it MUST honor request
// cancellation, use only the fixed request destination, and never add ambient
// credentials/proxy routing. A custom RoundTripper is responsible for these same
// constraints; it is not sandboxed by this client. No production one is installed.
type TerminalAdmissionClient struct {
	destination string
	client      *http.Client
}

func NewTerminalAdmissionClient(origin string, transport http.RoundTripper, timeout time.Duration) (*TerminalAdmissionClient, error) {
	u, err := url.Parse(origin)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.String() != origin || timeout <= 0 || timeout > 30*time.Second || nilTerminalTransport(transport) || reflect.ValueOf(transport) == reflect.ValueOf(http.DefaultTransport) {
		return nil, errors.New("terminal admission requires explicit origin, transport and bounded timeout")
	}
	if u.Scheme == "http" {
		host := strings.ToLower(u.Hostname())
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return nil, errors.New("remote terminal admission requires HTTPS")
		}
	}
	if standard, ok := transport.(*http.Transport); ok {
		if standard.Proxy != nil || (standard.TLSClientConfig != nil && standard.TLSClientConfig.InsecureSkipVerify) {
			return nil, errors.New("terminal admission transport must not use proxies or insecure TLS")
		}
		copy := standard.Clone()
		copy.DisableCompression = true
		copy.DisableKeepAlives = true
		copy.MaxResponseHeaderBytes = 8192
		copy.ResponseHeaderTimeout = timeout
		copy.TLSHandshakeTimeout = timeout
		transport = copy
	}
	return &TerminalAdmissionClient{destination: origin + TerminalAdmissionPath, client: &http.Client{
		Transport: transport, Timeout: timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

func nilTerminalTransport(transport http.RoundTripper) bool {
	if transport == nil {
		return true
	}
	value := reflect.ValueOf(transport)
	switch value.Kind() {
	case reflect.Pointer, reflect.Func, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan:
		return value.IsNil()
	}
	return false
}

// AcceptTerminalEventIntent returns success only for the exact durable owner
// acknowledgment. All other results, including a lost response, generic 2xx,
// duplicate 409 or cancellation, leave the immutable native intent pending.
func (c *TerminalAdmissionClient) AcceptTerminalEventIntent(ctx context.Context, intentJSON, eventID string) (DomainEventAcknowledgment, error) {
	var zero DomainEventAcknowledgment
	if c == nil || c.client == nil || ctx == nil {
		return zero, errors.New("terminal admission client and context required")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	body, err := json.Marshal(TerminalAdmissionRequest{Schema: TerminalAdmissionSchema, EventID: eventID, IntentJSON: intentJSON})
	if err != nil {
		return zero, err
	}
	if _, err := DecodeTerminalAdmissionRequest(body); err != nil {
		return zero, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.destination, bytes.NewReader(body))
	if err != nil {
		return zero, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return zero, errors.New("terminal admission exchange failed")
	}
	defer response.Body.Close()
	encoded, err := io.ReadAll(io.LimitReader(response.Body, TerminalAdmissionMaxResponseBytes+1))
	if err != nil || len(encoded) > TerminalAdmissionMaxResponseBytes {
		return zero, errors.New("terminal admission response unreadable or too large")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if response.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("terminal admission status %d is not an acknowledgment", response.StatusCode)
	}
	media, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return zero, errors.New("invalid terminal admission response type")
	}
	var ack DomainEventAcknowledgment
	if json.Unmarshal(encoded, &ack) != nil {
		return zero, errors.New("invalid terminal admission acknowledgment")
	}
	canonical, err := json.Marshal(ack)
	if err != nil || !bytes.Equal(bytes.TrimSpace(encoded), canonical) {
		return zero, errors.New("ambiguous terminal admission acknowledgment")
	}
	if err := ValidateDomainEventAcknowledgment(intentJSON, eventID, ack); err != nil {
		return zero, err
	}
	return ack, nil
}
