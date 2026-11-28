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

func TestSyntheticWebhookValidationAcceptsEveryFixedDispatchMethod(t *testing.T) {
	workflowID := "0123456789abcdefghij"
	base := `[{"id":"` + workflowID + `","active":false,"nodes":[{"type":"n8n-nodes-base.webhook","parameters":{"httpMethod":"METHOD","path":"events","responseMode":"lastNode"}}]}]`
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead} {
		t.Run(method, func(t *testing.T) {
			exported := strings.Replace(base, "METHOD", method, 1)
			route, err := validateExportedSyntheticWebhook([]byte(exported), workflowID)
			if err != nil || route.Method != method || route.Path != "/webhook/events" {
				t.Fatalf("valid fixed dispatch method rejected: route=%#v err=%v", route, err)
			}
		})
	}
	for _, method := range []string{http.MethodOptions, "CONNECT", "TRACE"} {
		t.Run("reject-"+method, func(t *testing.T) {
			exported := strings.Replace(base, "METHOD", method, 1)
			if _, err := validateExportedSyntheticWebhook([]byte(exported), workflowID); err == nil {
				t.Fatalf("unsupported webhook method %q accepted", method)
			}
		})
	}
}

func TestSyntheticWebhookAdmissionAllowsBoundedGoogleWorkflowAndResponseNodeMode(t *testing.T) {
	workflowID := "0123456789abcdefghij"
	valid := `[{"id":"` + workflowID + `","active":false,"nodes":[{"type":"n8n-nodes-base.webhook","parameters":{"httpMethod":"POST","path":"lead-intake","responseMode":"responseNode"}},{"type":"n8n-nodes-base.googleSheets","parameters":{"resource":"sheet","operation":"append","documentId":{"value":"synthetic-sheet"},"sheetName":{"value":"Leads"}}},{"type":"n8n-nodes-base.gmail","parameters":{"resource":"message","operation":"send","sendTo":"lead@example.invalid","subject":"Synthetic lead received","message":"Thanks for your request.","emailType":"text"}},{"type":"n8n-nodes-base.respondToWebhook","parameters":{"respondWith":"json"}}]}]`
	if route, err := validateExportedSyntheticWebhook([]byte(valid), workflowID); err != nil || route.Method != http.MethodPost || route.Path != "/webhook/lead-intake" {
		t.Fatalf("bounded public-style webhook graph rejected: route=%#v err=%v", route, err)
	}
	for name, invalid := range map[string]string{
		"missing responder":             strings.Replace(valid, `,{"type":"n8n-nodes-base.respondToWebhook","parameters":{"respondWith":"json"}}`, "", 1),
		"dynamic sheets selector":       strings.Replace(valid, `"value":"synthetic-sheet"`, `"value":"={{ $json.documentId }}"`, 1),
		"unsupported sheets operation":  strings.Replace(valid, `"operation":"append"`, `"operation":"delete"`, 1),
		"unsupported gmail operation":   strings.Replace(valid, `"operation":"send"`, `"operation":"delete"`, 1),
		"non-synthetic gmail recipient": strings.Replace(valid, `lead@example.invalid`, `lead@example.com`, 1),
		"dynamic gmail subject":         strings.Replace(valid, `Synthetic lead received`, `={{ $json.subject }}`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := validateExportedSyntheticWebhook([]byte(invalid), workflowID); err == nil {
				t.Fatalf("unsafe native webhook profile accepted: %s", name)
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

func TestDispatchSyntheticWebhookUsesOwnerLocalRequestWhenNoHostPortExists(t *testing.T) {
	containerName, networkName := objectNames(testSandboxID)
	runtimeID := runtimeIDFor(testSandboxID)
	ownerOnlyContainer := []byte(`[ {"Id":"owned-n8n","Config":{"Image":"` + ImageRef + `","Labels":{"com.vrooli.resource":"n8n","com.vrooli.sandbox-id":"` + testSandboxID + `","com.vrooli.runtime-id":"` + runtimeID + `","com.vrooli.object-kind":"container"}},"State":{"Running":true},"NetworkSettings":{"Ports":{},"Networks":{"` + networkName + `":{"Aliases":["` + n8nAlias(testSandboxID) + `"]}}},"HostConfig":{"PortBindings":{"5678/tcp":[{"HostIp":"127.0.0.1","HostPort":"15678"}]}}} ]`)
	operationID := uuid.MustParse(workflowOperationID)
	workflowID := workflowIDFor(testSandboxID, runtimeID, operationID)
	exported := `[{"id":"` + workflowID + `","active":false,"nodes":[{"type":"n8n-nodes-base.webhook","parameters":{"httpMethod":"POST","path":"events","responseMode":"lastNode"}}]}]`
	runner := ownedRuntimeRunner(t, runtimeID)
	baseCall := runner.call
	var lifecycle []string
	var staged []byte
	responseHash := strings.Repeat("b", 64)
	runner.call = func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "container" && args[1] == "inspect" && args[len(args)-1] == containerName {
			return ownerOnlyContainer, nil
		}
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, " n8n export:workflow "):
			lifecycle = append(lifecycle, "lookup")
			return nil, nil
		case strings.Contains(joined, " n8n publish:workflow "):
			lifecycle = append(lifecycle, "publish")
			return nil, nil
		case strings.Contains(joined, " n8n unpublish:workflow "):
			lifecycle = append(lifecycle, "unpublish")
			return nil, nil
		case len(args) > 6 && args[0] == "exec" && args[4] == "node" && args[5] == "-e" && args[6] == readSyntheticWorkflowProgram:
			return []byte(exported), nil
		case len(args) > 6 && args[0] == "exec" && args[4] == "node" && args[5] == "-e" && args[6] == dispatchSyntheticWebhookProgram:
			return []byte(fmt.Sprintf(`{"httpStatus":202,"responseBytes":12,"responseSha256":%q}`, responseHash)), nil
		case strings.Contains(joined, " rm -f -- /tmp/.vrooli-webhook-export-") || strings.Contains(joined, " rm -f -- /tmp/.vrooli-webhook-event-"):
			return nil, nil
		}
		return baseCall(ctx, args...)
	}
	runner.inputCall = func(_ context.Context, input []byte, args ...string) ([]byte, error) {
		if !containsArgPair(args, "--interactive", "--user") || !containsArg(args, stageWorkflowFileProgram) {
			t.Fatalf("webhook event was not staged through the fixed stdin writer: %#v", args)
		}
		staged = append([]byte(nil), input...)
		return nil, nil
	}
	owner := &Owner{Runner: runner, WebhookPublishWait: time.Millisecond}
	receipt, err := owner.DispatchSyntheticWebhook(context.Background(), testSandboxID, runtimeID, operationID, []byte(`{"event":"synthetic"}`))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "completed" || receipt.HTTPStatus != 202 || receipt.ResponseSHA256 != responseHash || receipt.PublicationState != "unpublish-requested" || receipt.CleanupRequired {
		t.Fatalf("unexpected owner-local receipt: %#v", receipt)
	}
	var event struct {
		Path   string         `json:"path"`
		Method string         `json:"method"`
		Event  map[string]any `json:"event"`
	}
	if err := json.Unmarshal(staged, &event); err != nil || event.Path != "/webhook/events" || event.Method != http.MethodPost || event.Event["event"] != "synthetic" {
		t.Fatalf("staged owner-local event = %+v err=%v", event, err)
	}
	if strings.Join(lifecycle, ",") != "lookup,publish,unpublish" {
		t.Fatalf("workflow lifecycle = %v", lifecycle)
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

func TestSyntheticWebhookPublishFailureClassifierDoesNotReturnDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name     string
		stdout   string
		stderr   string
		exceeded bool
		want     string
	}{
		{name: "unavailable command", stderr: "Unknown command publish:workflow", want: "workflow-publish-cli-unavailable"},
		{name: "missing workflow", stderr: "Workflow not found", want: "workflow-publish-workflow-missing"},
		{name: "project ownership", stderr: "workflow has no project owner", want: "workflow-publish-ownership-rejected"},
		{name: "bounded generic failure", stderr: "private workflow fields", want: "workflow-publish-runtime-error"},
		{name: "output limit", stdout: "ignored", exceeded: true, want: "workflow-publish-output-limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifySyntheticWebhookPublishFailure([]byte(tc.stdout), []byte(tc.stderr), tc.exceeded); got != tc.want {
				t.Fatalf("failure class = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSyntheticWebhookDispatcherReconcilesPublication404WithoutSurfacingBody(t *testing.T) {
	for _, marker := range []string{`response.statusCode===404`, `attempt<20`, `setTimeout(()=>send(attempt+1),500)`, `responseSha256:hash.digest("hex")`} {
		if !strings.Contains(dispatchSyntheticWebhookProgram, marker) {
			t.Errorf("owner-local dispatcher lacks bounded publication reconciliation marker %q", marker)
		}
	}
	if strings.Contains(dispatchSyntheticWebhookProgram, "response body") || strings.Contains(dispatchSyntheticWebhookProgram, "responseBody") {
		t.Fatal("webhook dispatcher must hash response content without retaining or returning its body")
	}
}
