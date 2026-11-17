package deletiongrant

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

func TestReconcileSubscriptionUsesExistingTargetSemantics(t *testing.T) {
	tests := []struct {
		name         string
		listed       string
		wantMethod   string
		wantPath     string
		wantMutation bool
	}{
		{
			name:   "matching subscription is retained",
			listed: fmt.Sprintf(`[{"id":3,"name":%q,"owner_scenario":%q,"event_pattern":%q,"delivery_type":"webhook","delivery_target":"https://storage/hook","enabled":true}]`, subscriptionName, subscriptionOwner, operatordecision.AskResolvedEventType),
		},
		{
			name:         "owned route is repaired after event pattern changes",
			listed:       fmt.Sprintf(`[{"id":3,"name":%q,"owner_scenario":%q,"event_pattern":"old.event","delivery_type":"webhook","delivery_target":"https://old/hook","enabled":true}]`, subscriptionName, subscriptionOwner),
			wantMethod:   http.MethodPut,
			wantPath:     "/api/v1/subscriptions/3",
			wantMutation: true,
		},
		{
			name:         "unrelated same-target subscription is preserved",
			listed:       `[{"id":3,"name":"other-integration","owner_scenario":"storage-manager","event_pattern":"old.event","delivery_type":"webhook","delivery_target":"https://storage/hook","enabled":true}]`,
			wantMethod:   http.MethodPost,
			wantPath:     "/api/v1/subscriptions",
			wantMutation: true,
		},
		{
			name:         "disabled owned subscription is reenabled",
			listed:       fmt.Sprintf(`[{"id":3,"name":%q,"owner_scenario":%q,"event_pattern":"old.event","delivery_type":"webhook","delivery_target":"https://storage/hook","enabled":false}]`, subscriptionName, subscriptionOwner),
			wantMethod:   http.MethodPut,
			wantPath:     "/api/v1/subscriptions/3",
			wantMutation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var method, path, body string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					if r.URL.Path != "/api/v1/subscriptions" || r.URL.Query().Get("owner") != subscriptionOwner {
						t.Errorf("list request = %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
					}
					_, _ = w.Write([]byte(tt.listed))
					return
				}
				method, path = r.Method, r.URL.Path
				data, _ := io.ReadAll(r.Body)
				body = string(data)
				if r.Method == http.MethodPost {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"id":4}`))
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			client, err := eventbus.NewSubscriptionClient(server.URL, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			if err := reconcileSubscription(context.Background(), client, "https://storage/hook"); err != nil {
				t.Fatal(err)
			}
			if tt.wantMutation {
				if method != tt.wantMethod || path != tt.wantPath {
					t.Fatalf("mutation = %s %s; want %s %s", method, path, tt.wantMethod, tt.wantPath)
				}
				for _, field := range []string{subscriptionName, subscriptionOwner, operatordecision.AskResolvedEventType, "https://storage/hook", `"enabled":true`} {
					if !strings.Contains(body, field) {
						t.Errorf("mutation body %q does not contain %q", body, field)
					}
				}
			} else if method != "" {
				t.Fatalf("matching subscription was mutated with %s %s", method, path)
			}
		})
	}
}
