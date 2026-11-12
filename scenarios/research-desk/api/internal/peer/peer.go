// Package peer dials another scenario's Connect API as the desk: the base URL
// comes from discovery on each call, and a peer that needs proof of who is
// calling gets the desk's own credential when it has one. The operator's own
// session token is never forwarded. A refusal is returned as the peer said
// it, and remembered until a later call succeeds; the desk never retries with
// a wider credential.
package peer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
)

// How the desk reaches a peer, in the words the operator sees (rulings F14).
const (
	AccessCredential   = "service credential"
	AccessAvailability = "local availability policy (no desk credential)"
)

// CallTimeout bounds every call the desk makes to a peer (rulings E5).
const CallTimeout = 20 * time.Second

// Client is the HTTP client for peer calls.
func Client() *http.Client { return &http.Client{Timeout: CallTimeout} }

// Peer is one optional dependency.
type Peer struct {
	// Name is the scenario, used in errors.
	Name string
	// BaseURL resolves the API base URL on each call.
	BaseURL func(ctx context.Context) (string, error)
	// Credential returns the desk's Authorization header value for this
	// peer. Nil, or an empty value, sends none: the peer's own policy then
	// decides what a local caller may do. An error means the desk does not
	// call.
	Credential func(ctx context.Context) (string, error)
	// HTTP is the HTTP client; nil uses Client().
	HTTP *http.Client

	mu      sync.Mutex
	refusal error
}

// Access says how the desk reaches the peer now: with its service
// credential, or under the peer's local availability policy.
func (p *Peer) Access(ctx context.Context) (string, error) {
	if p.Credential == nil {
		return AccessAvailability, nil
	}
	credential, err := p.Credential(ctx)
	switch {
	case err != nil:
		return "", err
	case strings.TrimSpace(credential) == "":
		return AccessAvailability, nil
	}
	return AccessCredential, nil
}

// Refused is the peer's last refusal (permission denied or unauthenticated),
// nil once a later call succeeded.
func (p *Peer) Refused() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.refusal
}

// watch remembers a refusal and clears it on success.
func (p *Peer) watch(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		resp, err := next(ctx, req)
		var cerr *connect.Error
		refused := errors.As(err, &cerr) && (cerr.Code() == connect.CodePermissionDenied || cerr.Code() == connect.CodeUnauthenticated)
		p.mu.Lock()
		switch {
		case refused:
			p.refusal = fmt.Errorf("%s refused %s: %s", p.Name, req.Spec().Procedure, cerr.Message())
		case err == nil:
			p.refusal = nil
		}
		p.mu.Unlock()
		return resp, err
	}
}

// Dial returns what a generated Connect client constructor needs.
func (p *Peer) Dial(ctx context.Context) (connect.HTTPClient, string, connect.ClientOption, error) {
	base, err := p.BaseURL(ctx)
	if err != nil {
		return nil, "", nil, fmt.Errorf("%s is not reachable: %w", p.Name, err)
	}
	client := p.HTTP
	if client == nil {
		client = Client()
	}
	watch := connect.UnaryInterceptorFunc(p.watch)
	if p.Credential == nil {
		return client, base, connect.WithInterceptors(watch), nil
	}
	credential, err := p.Credential(ctx)
	if err != nil {
		return nil, "", nil, err
	}
	if strings.TrimSpace(credential) == "" {
		return client, base, connect.WithInterceptors(watch), nil
	}
	return client, base, connect.WithInterceptors(watch, header("Authorization", credential)), nil
}

func header(name, value string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set(name, value)
			return next(ctx, req)
		}
	}
}
