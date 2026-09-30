package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/encoding/protojson"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type conversationRequestCapture struct {
	mu     sync.Mutex
	paths  []string
	bodies [][]byte
}

func (c *conversationRequestCapture) add(path string, body []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paths = append(c.paths, path)
	c.bodies = append(c.bodies, append([]byte(nil), body...))
}

func newConversationTestApp(t *testing.T, responder func(string) (int, string)) (*App, *conversationRequestCapture) {
	t.Helper()
	capture := &conversationRequestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		capture.add(request.URL.Path, body)
		status, response := responder(request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		_, _ = writer.Write([]byte(response))
	}))
	t.Cleanup(server.Close)
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
	return &App{services: NewServices(api)}, capture
}

func TestParseConversationSearchArgsCoversModesFiltersAndCursor(t *testing.T) {
	for _, mode := range []string{"hybrid", "text", "regex", "semantic"} {
		query := "capacity ledger"
		if mode == "regex" {
			query = `(?i)capacity\s+ledger`
		}
		parsed, err := parseConversationSearchArgs([]string{
			query, "--mode", mode, "--sort", "newest", "--after", "2026-08-01T00:00:00Z", "--before", "2026-09-01T00:00:00Z",
			"--roles", "user,assistant,user", "--harnesses", "codex", "--provider-origins", "local", "--projects", "/workspace/project",
			"--cwds", "/workspace/project/api", "--runners", "codex", "--models", "gpt", "--profiles", "default", "--statuses", "complete",
			"--tags", "plan", "--workloads", "implementation", "--content-classes", "prose,quoted-prose", "--include-tool-events",
			"--page-size", "25", "--cursor", "cursor-v1", "--json",
		})
		if err != nil {
			t.Fatalf("mode %s: %v", mode, err)
		}
		if parsed.mode == domainpb.ConversationSearchMode_CONVERSATION_SEARCH_MODE_UNSPECIFIED || parsed.pageCursor != "cursor-v1" || parsed.pageSize != 25 || !parsed.filters.GetIncludeToolEvents() || len(parsed.filters.GetRoles()) != 2 || len(parsed.filters.GetContentClasses()) != 2 {
			t.Fatalf("mode %s parsed incompletely: %+v filters=%+v", mode, parsed, parsed.filters)
		}
	}
}

func TestParseConversationSearchArgsRejectsInvalidInput(t *testing.T) {
	cases := [][]string{
		{"query", "--mode", "unknown"},
		{"[", "--mode", "regex"},
		{"", "--mode", "semantic", "--sort", "newest", "--roles", "user"},
		{"", "--sort", "relevance", "--roles", "user"},
		{"", "--sort", "newest"},
		{"query", "--after", "not-a-time"},
		{"query", "--after", "2026-09-02T00:00:00Z", "--before", "2026-09-01T00:00:00Z"},
		{"query", "--page-size", "101"},
		{"query", "--content-classes", "secret"},
	}
	for _, args := range cases {
		if _, err := parseConversationSearchArgs(args); err == nil {
			t.Fatalf("arguments should fail: %q", args)
		}
	}
}

func TestConversationSearchCommandUsesTypedRequestAndExplainsResults(t *testing.T) {
	response := `{
  "hits": [{"stableHitId":"hit-1","runId":"run-1","role":"assistant","occurredAt":"2026-09-01T12:00:00Z","snippet":"capacity ledger","highlights":[{"startGrapheme":0,"endGrapheme":8}],"provenance":{"harness":"codex","sourceSessionId":"session-1"},"rankEvidence":[{"leg":"CONVERSATION_SEARCH_LEG_LEXICAL","rank":1,"score":0.9,"explanation":"phrase"}],"run":{"runId":"run-1","label":"Adaptive scheduler","status":"complete","runner":"codex","model":"gpt"},"deepLink":"/runs/run-1?event=hit-1"}],
  "nextPageCursor":"next-v1","coverage":{"lexicalRatio":1,"semanticRatio":0.5,"freshnessLagMs":"1200"},
  "degradations":[{"reason":"CONVERSATION_SEARCH_DEGRADATION_REASON_SEMANTIC_UNAVAILABLE","leg":"CONVERSATION_SEARCH_LEG_DENSE","detail":"embedding offline","retryable":true}]
}`
	app, capture := newConversationTestApp(t, func(string) (int, string) { return http.StatusOK, response })
	output := captureStdout(t, func() error {
		return app.conversationSearch([]string{"capacity ledger", "--mode", "hybrid", "--roles", "assistant", "--page-token", "prior-v1"})
	})
	for _, expected := range []string{"run-1", "Adaptive scheduler", "⟦capacity⟧ ledger", "lexical#1=0.9000:phrase", "harness=codex", "Degraded:", "--page-token next-v1"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("output missing %q:\n%s", expected, output)
		}
	}
	if len(capture.paths) != 1 || capture.paths[0] != "/agent_manager.v1.ConversationSearchService/SearchConversations" {
		t.Fatalf("paths=%v", capture.paths)
	}
	request := &domainpb.SearchConversationsRequest{}
	if err := protojson.Unmarshal(capture.bodies[0], request); err != nil {
		t.Fatal(err)
	}
	if request.GetQuery() != "capacity ledger" || request.GetPageCursor() != "prior-v1" || len(request.GetFilters().GetRoles()) != 1 {
		t.Fatalf("request=%s", request)
	}
}

func TestConversationCommandsHandleNoResultsContextStatusAndReindex(t *testing.T) {
	app, capture := newConversationTestApp(t, func(path string) (int, string) {
		switch path {
		case "/agent_manager.v1.ConversationSearchService/SearchConversations":
			return http.StatusOK, `{}`
		case "/agent_manager.v1.ConversationSearchService/GetConversationContext":
			return http.StatusOK, `{"hit":{"stableHitId":"hit-1","runId":"run-1","run":{"label":"Context run"}},"events":[{"eventId":"event-1","eventSequence":"4","role":"user","boundedContent":"bounded","matched":true}],"truncated":true}`
		case "/agent_manager.v1.ConversationSearchService/GetConversationIndexStatus":
			return http.StatusOK, `{"state":"CONVERSATION_INDEX_STATE_DEGRADED","activeGeneration":"generation-1","collectionLayout":"dense+sparse","coverage":{"canonicalVisibleMessages":"5","lexicalDocuments":"5","semanticDocuments":"3","freshnessLagMs":"50"}}`
		case "/agent_manager.v1.ConversationSearchControlService/PlanConversationReindex":
			return http.StatusOK, `{"operationId":"plan-1","state":"CONVERSATION_REINDEX_STATE_PLANNED","dryRun":true,"plannedDocuments":"5"}`
		case "/agent_manager.v1.ConversationSearchControlService/ReindexConversations":
			return http.StatusOK, `{"operationId":"job-1","state":"CONVERSATION_REINDEX_STATE_QUEUED","plannedDocuments":"5"}`
		case "/agent_manager.v1.ConversationSearchControlService/CancelConversationReindex":
			return http.StatusOK, `{"operationId":"job-1","state":"CONVERSATION_REINDEX_STATE_CANCELLED"}`
		default:
			return http.StatusNotFound, `{}`
		}
	})
	checks := []struct {
		run  func() error
		want string
	}{
		{func() error { return app.conversationSearch([]string{"missing"}) }, "No conversation matches"},
		{func() error { return app.conversationContext([]string{"hit-1", "--before", "1", "--after", "1"}) }, "bounded"},
		{func() error { return app.conversationIndexStatus(nil) }, "dense+sparse"},
		{func() error { return app.conversationReindex([]string{"--dry-run", "--control-token", "token"}) }, "planned"},
		{func() error {
			return app.conversationReindex([]string{"--control-token", "token", "--idempotency-key", "retry-1"})
		}, "job-1"},
		{func() error { return app.conversationReindexCancel([]string{"job-1", "--control-token", "token"}) }, "cancelled"},
	}
	for _, check := range checks {
		output := captureStdout(t, check.run)
		if !strings.Contains(strings.ToLower(output), strings.ToLower(check.want)) {
			t.Fatalf("output %q missing %q", output, check.want)
		}
	}
	if len(capture.paths) != len(checks) {
		t.Fatalf("captured %d requests, want %d", len(capture.paths), len(checks))
	}
}

func TestConversationCommandPreservesAPIErrors(t *testing.T) {
	app, _ := newConversationTestApp(t, func(string) (int, string) {
		return http.StatusBadRequest, `{"code":"invalid_argument","message":"filters.occurred_after is invalid"}`
	})
	if err := app.conversationSearch([]string{"query"}); err == nil || !strings.Contains(err.Error(), "filters.occurred_after") {
		t.Fatalf("error=%v", err)
	}
}

func TestConversationJSONUsesStableProtoFieldNames(t *testing.T) {
	output := captureStdout(t, func() error {
		return printConversationJSON(&domainpb.SearchConversationsResponse{NextPageCursor: "cursor-v1"})
	})
	if !strings.Contains(output, `"next_page_cursor"`) || strings.Contains(output, `"nextPageCursor"`) {
		t.Fatalf("output=%s", output)
	}
}

func TestConversationHelpDocumentsFiltersAndIndexOperations(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	searchHelp := captureStdout(t, func() error { return app.Run([]string{"conversation", "search", "--help"}) })
	for _, want := range []string{"--mode hybrid|text|regex|semantic", "--project-scopes", "Examples:", "--page-token"} {
		if !strings.Contains(searchHelp, want) {
			t.Fatalf("search help missing %q:\n%s", want, searchHelp)
		}
	}
	indexHelp := captureStdout(t, func() error { return app.Run([]string{"conversation", "index", "status", "--help"}) })
	for _, want := range []string{"status [--json]", "reindex [--dry-run]", "cancel <operation-id>"} {
		if !strings.Contains(indexHelp, want) {
			t.Fatalf("index help missing %q:\n%s", want, indexHelp)
		}
	}
}

func TestConversationHumanTextStripsTerminalControls(t *testing.T) {
	got := safeTerminalText("safe\x1b[31mred\nnext")
	if strings.ContainsRune(got, '\x1b') || strings.ContainsRune(got, '\n') || !strings.Contains(got, "safe [31mred next") {
		t.Fatalf("sanitized text=%q", got)
	}
}
