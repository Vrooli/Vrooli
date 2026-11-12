package n8n

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSyntheticWebhookValidationIsStaticAndTriggerBound(t *testing.T) {
	workflowID := "0123456789abcdefghij"
	base := `[{"id":"` + workflowID + `","active":false,"nodes":[{"type":"n8n-nodes-base.webhook","parameters":{"httpMethod":"POST","path":"events","responseMode":"lastNode"}},{"type":"n8n-nodes-base.httpRequest","parameters":{"url":"http://api.synthetic.invalid:18080/route"}}]}]`
	route, err := validateExportedSyntheticWebhook([]byte(base), workflowID)
	if err != nil || route.Method != http.MethodPost || route.Path != "/webhook/events" {
		t.Fatalf("validated route = %#v, %v", route, err)
	}
	cases := []struct {
		name string
		json string
	}{
		{"wrong owner ID", strings.Replace(base, workflowID, "differentworkflowid", 1)},
		{"published workflow", strings.Replace(base, `"active":false`, `"active":true`, 1)},
		{"dynamic path", strings.Replace(base, `"events"`, `"events/:id"`, 1)},
		{"multiple methods", strings.Replace(base, `"parameters":{"httpMethod":"POST"`, `"parameters":{"multipleMethods":true,"httpMethod":"POST"`, 1)},
		{"additional activation trigger", strings.Replace(base, `{"type":"n8n-nodes-base.httpRequest","parameters":{"url":"http://api.synthetic.invalid:18080/route"}}`, `{"type":"n8n-nodes-base.scheduleTrigger","parameters":{}}`, 1)},
		{"unknown node type", strings.Replace(base, `"type":"n8n-nodes-base.httpRequest"`, `"type":"community.poller"`, 1)},
		{"external HTTP destination", strings.Replace(base, "http://api.synthetic.invalid:18080/route", "http://example.com/route", 1)},
		{"wrong mock port", strings.Replace(base, ":18080/route", ":8080/route", 1)},
		{"duplicate webhook", strings.Replace(base, `{"type":"n8n-nodes-base.httpRequest","parameters":{"url":"http://api.synthetic.invalid:18080/route"}}`, `{"type":"n8n-nodes-base.webhook","parameters":{"httpMethod":"POST","path":"other"}}`, 1)},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := validateExportedSyntheticWebhook([]byte(test.json), workflowID); err == nil {
				t.Fatalf("accepted unsafe workflow: %s", test.json)
			}
		})
	}
}

func TestSyntheticWebhookEventRejectsClientLikeAndUnboundedInput(t *testing.T) {
	for _, input := range []string{
		`{"access_token":"stand-in"}`,
		`{"email":"person@example.com"}`,
		`{"value":"Bearer abcdefghijklmnop"}`,
		`{"duplicate":1,"duplicate":2}`,
	} {
		if _, err := validateSyntheticWebhookEvent([]byte(input)); err == nil {
			t.Fatalf("accepted unsafe event %q", input)
		}
	}
	if _, err := validateSyntheticWebhookEvent([]byte(`{"email":"synthetic@example.invalid"}`)); err != nil {
		t.Fatalf("rejected synthetic event: %v", err)
	}
	if _, err := validateSyntheticWebhookEvent(make([]byte, MaxSyntheticWebhookEventBytes+1)); err == nil {
		t.Fatal("accepted oversized event")
	}
}

type webhookRoundTripper func(*http.Request) (*http.Response, error)

func (f webhookRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestDispatchSyntheticWebhookUsesExportedRouteAndAlwaysUnpublishes(t *testing.T) {
	operationID := uuid.MustParse(workflowOperationID)
	workflowID := workflowIDFor(testSandboxID, runtimeIDFor(testSandboxID), operationID)
	exported := `[{"id":"` + workflowID + `","active":false,"nodes":[{"type":"n8n-nodes-base.webhook","parameters":{"httpMethod":"POST","path":"events","responseMode":"lastNode"}},{"type":"n8n-nodes-base.httpRequest","parameters":{"url":"http://api.synthetic.invalid:18080/route"}}]}]`
	runner := ownedRuntimeRunner(t, runtimeIDFor(testSandboxID))
	baseCall := runner.call
	var lifecycle []string
	var exportArgs []string
	runner.call = func(ctx context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, " n8n export:workflow "):
			lifecycle = append(lifecycle, "lookup")
			exportArgs = append([]string(nil), args...)
			return nil, nil
		case len(args) > 5 && args[0] == "exec" && args[4] == "node" && args[5] == "-e":
			return []byte(exported), nil
		case strings.Contains(joined, " n8n publish:workflow "):
			lifecycle = append(lifecycle, "publish")
			return nil, nil
		case strings.Contains(joined, " n8n unpublish:workflow "):
			lifecycle = append(lifecycle, "unpublish")
			return nil, nil
		case strings.Contains(joined, " rm -f -- /tmp/.vrooli-webhook-export-"):
			return nil, nil
		default:
			return baseCall(ctx, args...)
		}
	}
	body := `{"record":{"email":"synthetic@example.invalid"}}`
	var observed *http.Request
	owner := &Owner{
		Runner: runner, WebhookPublishWait: time.Millisecond,
		HTTPClient: &http.Client{Transport: webhookRoundTripper(func(request *http.Request) (*http.Response, error) {
			observed = request
			body, _ := io.ReadAll(request.Body)
			request.Body.Close()
			if string(body) != "{\"record\":{\"email\":\"synthetic@example.invalid\"}}" {
				return nil, fmt.Errorf("unexpected event body")
			}
			return &http.Response{StatusCode: 202, Body: io.NopCloser(strings.NewReader("secret response body")), Header: make(http.Header), Request: request}, nil
		})},
	}
	receipt, err := owner.DispatchSyntheticWebhook(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), operationID, []byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if observed == nil || observed.Method != http.MethodPost || observed.URL.String() != "http://127.0.0.1:15678/webhook/events" {
		t.Fatalf("request was not routed to owner-derived loopback URL: %#v", observed)
	}
	if receipt.Status != "completed" || receipt.HTTPStatus != 202 || !receipt.Published || receipt.PublicationState != "unpublish-requested" || receipt.CleanupRequired {
		t.Fatalf("unexpected receipt: %#v", receipt)
	}
	if strings.Join(lifecycle, ",") != "lookup,publish,unpublish" {
		t.Fatalf("workflow lifecycle = %v", lifecycle)
	}
	if len(exportArgs) != 8 || exportArgs[0] != "exec" || exportArgs[4] != "n8n" || exportArgs[5] != "export:workflow" || exportArgs[6] != "--id="+workflowID || !strings.HasPrefix(exportArgs[7], "--output=/tmp/.vrooli-webhook-export-") {
		t.Fatalf("workflow lookup did not use fixed owner-derived export args: %#v", exportArgs)
	}
	encoded, _ := json.Marshal(receipt)
	if strings.Contains(string(encoded), body) || strings.Contains(string(encoded), "secret response body") || strings.Contains(string(encoded), workflowID) || strings.Contains(string(encoded), "/webhook/events") {
		t.Fatalf("receipt leaked request, response, route, or workflow identity: %s", encoded)
	}
}

func TestDispatchSyntheticWebhookFailurePathsAttemptCleanup(t *testing.T) {
	operationID := uuid.MustParse(workflowOperationID)
	workflowID := workflowIDFor(testSandboxID, runtimeIDFor(testSandboxID), operationID)
	event := []byte(`{"ok":true}`)
	exported := `[{"id":"` + workflowID + `","active":false,"nodes":[{"type":"n8n-nodes-base.webhook","parameters":{"httpMethod":"POST","path":"events","responseMode":"lastNode"}}]}]`
	for _, test := range []struct {
		name                 string
		failPublish          bool
		failUnpublish        bool
		failTmpfsCleanup     bool
		wantCleanup          bool
		wantPublished        bool
		wantPublicationState string
		wantHTTPRequest      bool
	}{
		{name: "publish error is cleaned", failPublish: true, wantPublished: true, wantPublicationState: "unpublish-requested"},
		{name: "unpublish error is surfaced", failUnpublish: true, wantCleanup: true, wantPublished: true, wantPublicationState: "unpublish-failed", wantHTTPRequest: true},
		{name: "tmpfs export cleanup is surfaced", failTmpfsCleanup: true, wantCleanup: true, wantPublished: true, wantPublicationState: "unpublish-requested", wantHTTPRequest: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := ownedRuntimeRunner(t, runtimeIDFor(testSandboxID))
			baseCall := runner.call
			var lifecycle []string
			tmpfsRemoveCount := 0
			runner.call = func(ctx context.Context, args ...string) ([]byte, error) {
				joined := strings.Join(args, " ")
				switch {
				case strings.Contains(joined, " n8n export:workflow "):
					lifecycle = append(lifecycle, "lookup")
					return nil, nil
				case len(args) > 5 && args[0] == "exec" && args[4] == "node" && args[5] == "-e":
					return []byte(exported), nil
				case strings.Contains(joined, " n8n publish:workflow "):
					lifecycle = append(lifecycle, "publish")
					if test.failPublish {
						return nil, fmt.Errorf("secret publish diagnostic")
					}
					return nil, nil
				case strings.Contains(joined, " n8n unpublish:workflow "):
					lifecycle = append(lifecycle, "unpublish")
					if test.failUnpublish {
						return nil, fmt.Errorf("secret cleanup diagnostic")
					}
					return nil, nil
				case strings.Contains(joined, " rm -f -- /tmp/.vrooli-webhook-export-"):
					tmpfsRemoveCount++
					if test.failTmpfsCleanup && tmpfsRemoveCount > 1 {
						return nil, fmt.Errorf("secret tmpfs cleanup diagnostic")
					}
					return nil, nil
				default:
					return baseCall(ctx, args...)
				}
			}
			var requestObserved bool
			owner := &Owner{Runner: runner, WebhookPublishWait: time.Millisecond, HTTPClient: &http.Client{Transport: webhookRoundTripper(func(request *http.Request) (*http.Response, error) {
				requestObserved = true
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: request}, nil
			})}}
			receipt, err := owner.DispatchSyntheticWebhook(context.Background(), testSandboxID, runtimeIDFor(testSandboxID), operationID, event)
			if err == nil {
				t.Fatal("expected failure-path error")
			}
			if receipt.CleanupRequired != test.wantCleanup || receipt.Published != test.wantPublished || receipt.PublicationState != test.wantPublicationState || requestObserved != test.wantHTTPRequest {
				t.Fatalf("receipt/request = %#v / %t", receipt, requestObserved)
			}
			if test.failTmpfsCleanup && receipt.Status != "completed-cleanup-required" {
				t.Fatalf("tmpfs cleanup status = %q", receipt.Status)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatalf("runtime diagnostic leaked: %v", err)
			}
			if strings.Join(lifecycle, ",") != "lookup,publish,unpublish" {
				t.Fatalf("cleanup command sequence = %v", lifecycle)
			}
		})
	}
}
