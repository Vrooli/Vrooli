// Package browsergrant carries the verified agent browser-grant receipt only
// through the active Test Genie run context and exposes it to the Audio Tools
// soak provider call. It deliberately has no persistence or serialization API.
package browsergrant

import (
	"context"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
)

const HeaderBrowserGrantReceipt = "X-Vrooli-Browser-Grant-Receipt"

type receiptContextKey struct{}
type testGenieLeaseReceiptContextKey struct{}

// AttachRequestReceipt accepts exactly one opaque browser grant only when the
// request already carries middleware-verified Agent Manager identity.
func AttachRequestReceipt(ctx context.Context, values []string) context.Context {
	if ctx == nil || !provenance.FromContext(ctx).IsVerifiedAgent() || len(values) != 1 {
		return ctx
	}
	receipt := strings.TrimSpace(values[0])
	if receipt == "" {
		return ctx
	}
	return context.WithValue(ctx, receiptContextKey{}, receipt)
}

// CarryVerifiedReceipt copies only the request-scoped receipt onto the
// server-owned run context after the same verified identity has been carried.
func CarryVerifiedReceipt(runCtx, initiator context.Context) context.Context {
	if runCtx == nil || initiator == nil ||
		!provenance.FromContext(initiator).IsVerifiedAgent() ||
		!provenance.FromContext(runCtx).IsVerifiedAgent() {
		return runCtx
	}
	if provenance.ForwardedIdentityToken(runCtx) == "" || provenance.ForwardedIdentityToken(runCtx) != provenance.ForwardedIdentityToken(initiator) {
		return runCtx
	}
	receipt := Receipt(initiator)
	if receipt == "" {
		return runCtx
	}
	return context.WithValue(runCtx, receiptContextKey{}, receipt)
}

// Receipt is the narrow accessor for an opaque request-scoped receipt. It
// returns empty unless provenance is still verified on this context.
func Receipt(ctx context.Context) string {
	if ctx == nil || !provenance.FromContext(ctx).IsVerifiedAgent() {
		return ""
	}
	receipt, _ := ctx.Value(receiptContextKey{}).(string)
	return strings.TrimSpace(receipt)
}

// AudioToolsSoakContext is the narrow providerRunner accessor. It scopes
// Test Genie's separate owner-issued runtime lease receipt only when this run
// also has verified agent identity and exactly one browser grant receipt.
func AudioToolsSoakContext(ctx context.Context, receipt string) context.Context {
	if ctx == nil || !provenance.FromContext(ctx).IsVerifiedAgent() || Receipt(ctx) == "" || strings.TrimSpace(receipt) == "" {
		return ctx
	}
	return context.WithValue(ctx, testGenieLeaseReceiptContextKey{}, strings.TrimSpace(receipt))
}

// AudioToolsSoakHeaders returns the three exact bearer headers needed by the
// Audio Tools soak admission. It fails closed as a unit; no partial identity,
// browser grant, or Test Genie lease receipt is forwarded.
func AudioToolsSoakHeaders(ctx context.Context) http.Header {
	if ctx == nil || !provenance.FromContext(ctx).IsVerifiedAgent() {
		return nil
	}
	identity := provenance.ForwardedIdentityToken(ctx)
	grant := Receipt(ctx)
	lease, _ := ctx.Value(testGenieLeaseReceiptContextKey{}).(string)
	if identity == "" || grant == "" || strings.TrimSpace(lease) == "" {
		return nil
	}
	headers := make(http.Header)
	headers.Set(cliutil.HeaderAgentIdentityToken, identity)
	headers.Set(HeaderBrowserGrantReceipt, grant)
	headers.Set("X-Test-Genie-Lease-Receipt", strings.TrimSpace(lease))
	return headers
}
