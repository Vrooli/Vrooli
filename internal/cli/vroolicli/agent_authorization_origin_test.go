package vroolicli

import (
	"context"
	"errors"
	"testing"
)

// A refusal's request command carries no --approval-base unless the refusing
// scenario is configured with one, so `vrooli agent authorize` must find the
// origin itself, and only ever accept an authenticated public (HTTPS) origin.
func TestResolveApprovalBaseAcceptsOnlyAPublicHTTPSOrigin(t *testing.T) {
	restore := externalOriginOf
	t.Cleanup(func() { externalOriginOf = restore })
	for _, tc := range []struct {
		name, resolved, want string
		err                  error
	}{
		{name: "public tunnel origin", resolved: "https://agent-manager.itsagitime.com", want: "https://agent-manager.itsagitime.com"},
		{name: "trailing slash trimmed", resolved: " https://agent-manager.itsagitime.com/ ", want: "https://agent-manager.itsagitime.com"},
		{name: "localhost fallback is not an approval origin", resolved: "http://localhost:36000"},
		{name: "plain http is not accepted", resolved: "http://agent-manager.itsagitime.com"},
		{name: "path is not an origin", resolved: "https://agent-manager.itsagitime.com/app"},
		{name: "credentials are refused", resolved: "https://owner:secret@agent-manager.itsagitime.com"},
		{name: "query is refused", resolved: "https://agent-manager.itsagitime.com/?x=1"},
		{name: "resolver failure", err: errors.New("tunnel manager unavailable")},
		{name: "empty", resolved: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			externalOriginOf = func(_ context.Context, slug string) (string, error) {
				if slug != "agent-manager" {
					t.Fatalf("resolved %q, want agent-manager", slug)
				}
				return tc.resolved, tc.err
			}
			if got := resolveApprovalBase(context.Background()); got != tc.want {
				t.Fatalf("resolveApprovalBase = %q, want %q", got, tc.want)
			}
		})
	}
}
