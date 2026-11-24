package bindings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	programbindings "program-runtime/internal/bindings"
)

const (
	browserGrantReceiptHeader = "X-Vrooli-Browser-Grant-Receipt"
	browserContextPath        = "/api/v1/browser/resolve-context"
	maxBrowserContextBytes    = 16 << 10
)

// browserSessionBindingIDs is the only governed Program Runtime surface that
// receives a browser grant receipt. Keep it synchronized with BAS's manifest.
var browserSessionBindingIDs = map[string]struct{}{
	"browser-automation-studio/session/open":          {},
	"browser-automation-studio/session/observe":       {},
	"browser-automation-studio/session/act":           {},
	"browser-automation-studio/session/attach":        {},
	"browser-automation-studio/session/read-trace":    {},
	"browser-automation-studio/session/draft-trace":   {},
	"browser-automation-studio/session/promote-trace": {},
	"browser-automation-studio/session/close":         {},
	"browser-automation-studio/task/run-task":         {},
	"browser-automation-studio/task/replay-flow":      {},
}

const BrowserRunTaskBindingID = "browser-automation-studio/task/run-task"

func IsBrowserSessionBindingID(id string) bool {
	_, ok := browserSessionBindingIDs[id]
	return ok
}

type browserContext struct {
	CallerID     string          `json:"caller_id"`
	RunID        string          `json:"run_id"`
	GrantReceipt string          `json:"grant_receipt"`
	Harness      string          `json:"harness"`
	Capability   json.RawMessage `json:"capability"`
}

// browserCapability is the minimum persisted AM contract required to relay
// the opaque receipt. The grant remains invocation-local and BAS revalidates
// it on every call; this check prevents a generic or incomplete AM context
// response from being treated as browser authority.
type browserCapability struct {
	Enabled      bool   `json:"enabled"`
	GrantReceipt string `json:"grantReceipt"`
	Mode         string `json:"mode"`
	Permissions  struct {
		ProfileID   string `json:"profileId"`
		ProfileMode string `json:"profileMode"`
	} `json:"permissions"`
}

// NewBrowserContextRequestPreparer returns a Program Runtime invocation hook.
// For one exact BAS binding ID it resolves a fresh grant from the verified
// identity, then adds both values only to the outbound HTTP headers. The
// resolver call has no identity or grant in its body and never follows redirects.
func NewBrowserContextRequestPreparer(client *http.Client, resolveURL func(context.Context, string) (string, error)) programbindings.OutboundRequestPreparer {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	if resolveURL == nil {
		return func(_ context.Context, id string, _ *http.Request) error {
			if IsBrowserSessionBindingID(id) {
				return errors.New("browser grant resolver is unavailable")
			}
			return nil
		}
	}
	return func(ctx context.Context, bindingID string, downstream *http.Request) error {
		if !IsBrowserSessionBindingID(bindingID) {
			return nil
		}
		if downstream == nil {
			return errors.New("browser request is missing")
		}
		if len(downstream.Header.Values(cliutil.HeaderAgentIdentityToken)) != 0 || len(downstream.Header.Values(browserGrantReceiptHeader)) != 0 {
			return errors.New("browser proof headers must be server-derived")
		}
		caller := provenance.FromContext(ctx)
		token := provenance.ForwardedIdentityToken(ctx)
		if token == "" || !caller.IsVerifiedAgent() || strings.TrimSpace(caller.RunID) == "" {
			return errors.New("verified browser caller identity is required")
		}
		base, err := resolveURL(ctx, "agent-manager")
		if err != nil || strings.TrimSpace(base) == "" {
			return errors.New("Agent Manager browser context is unavailable")
		}
		resolveReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+browserContextPath, http.NoBody)
		if err != nil {
			return errors.New("could not create Agent Manager browser context request")
		}
		resolveReq.Header.Set(cliutil.HeaderAgentIdentityToken, token)
		resolveReq.Header.Set("Accept", "application/json")
		resolverClient := *client
		resolverClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		response, err := resolverClient.Do(resolveReq)
		if err != nil {
			return errors.New("Agent Manager browser context request failed")
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("Agent Manager browser context denied with status %d", response.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(response.Body, maxBrowserContextBytes+1))
		if err != nil || len(body) > maxBrowserContextBytes {
			return errors.New("Agent Manager browser context response is invalid")
		}
		var resolved browserContext
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&resolved); err != nil {
			return errors.New("Agent Manager browser context response is invalid")
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			return errors.New("Agent Manager browser context response is invalid")
		}
		if strings.TrimSpace(resolved.CallerID) == "" || strings.TrimSpace(resolved.RunID) == "" || strings.TrimSpace(resolved.GrantReceipt) == "" {
			return errors.New("Agent Manager browser context is incomplete")
		}
		if resolved.RunID != caller.RunID || resolved.CallerID != "run:"+caller.RunID {
			return errors.New("Agent Manager browser context does not match the verified caller")
		}
		var capability browserCapability
		if len(resolved.Capability) == 0 || json.Unmarshal(resolved.Capability, &capability) != nil ||
			!capability.Enabled || strings.TrimSpace(capability.GrantReceipt) == "" ||
			capability.GrantReceipt != resolved.GrantReceipt || strings.TrimSpace(capability.Mode) == "" ||
			strings.TrimSpace(capability.Permissions.ProfileID) == "" || strings.TrimSpace(capability.Permissions.ProfileMode) == "" {
			return errors.New("Agent Manager browser capability is incomplete or mismatched")
		}
		downstream.Header.Set(cliutil.HeaderAgentIdentityToken, token)
		downstream.Header.Set(browserGrantReceiptHeader, resolved.GrantReceipt)
		return nil
	}
}
