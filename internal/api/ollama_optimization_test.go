package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"connectrpc.com/connect"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	cliv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1/cliv1connect"
)

func TestOllamaOptimizationConnectIsFixedTypedOwnerCall(t *testing.T) {
	called := false
	path, serviceHandler := cliv1connect.NewOllamaOptimizationServiceHandler(&ollamaOptimizationServiceHandler{
		runResourceCLI: func(resource string, args []string, stdout, _ io.Writer) error {
			called = true
			if resource != "ollama" || len(args) != 5 || args[0] != "optimize" || args[1] != "role-adapter" || args[2] != "--request" || args[4] != "--json" {
				t.Fatalf("resource operation = %q %q", resource, args)
			}
			requestBytes, err := os.ReadFile(args[3])
			if err != nil {
				t.Fatalf("read owner request file: %v", err)
			}
			var request map[string]any
			if err := json.Unmarshal(requestBytes, &request); err != nil {
				t.Fatalf("decode owner request: %v", err)
			}
			if request["request_id"] != "stable-run" || request["target"] != "ollama-role" || request["role"] != "chat.default" {
				t.Fatalf("request identity = %#v", request)
			}
			if len(request["candidate_models"].([]any)) != 3 || len(request["fixed_case_ids"].([]any)) != 20 {
				t.Fatalf("request denominator = %#v", request)
			}
			_, err = io.WriteString(stdout, `{"schema_version":"ollama.optimize.receipt.v1","receipt_id":"stable-run:resource-ollama","status":"refused"}`)
			return err
		},
	})
	mux := http.NewServeMux()
	mux.Handle(path, serviceHandler)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := cliv1connect.NewOllamaOptimizationServiceClient(server.Client(), server.URL)
	caseIDs := make([]string, 20)
	for i := range caseIDs {
		caseIDs[i] = string(rune('a' + i))
	}
	response, err := client.InspectRoleCandidates(context.Background(), connect.NewRequest(&cliv1.InspectOllamaRoleCandidatesRequest{
		RequestId: "stable-run", Target: "ollama-role", Role: "chat.default",
		CandidateModels: []string{"one", "two", "three"}, FixedSuiteId: "suite", FixedCaseIds: caseIDs,
		EvaluationReceiptId: "eval-receipt",
	}))
	if err != nil {
		t.Fatalf("InspectRoleCandidates() error = %v", err)
	}
	if !called {
		t.Fatal("resource owner was not called")
	}
	var receipt map[string]any
	if err := json.Unmarshal([]byte(response.Msg.GetReceiptJson()), &receipt); err != nil {
		t.Fatalf("decode receipt: %v", err)
	}
	if receipt["receipt_id"] != "stable-run:resource-ollama" || receipt["status"] != "refused" {
		t.Fatalf("receipt = %#v", receipt)
	}
}

func TestOllamaOptimizationConnectReportsUnavailableOwner(t *testing.T) {
	handler := &ollamaOptimizationServiceHandler{runResourceCLI: func(string, []string, io.Writer, io.Writer) error {
		return context.DeadlineExceeded
	}}
	_, err := handler.InspectRoleCandidates(context.Background(), connect.NewRequest(&cliv1.InspectOllamaRoleCandidatesRequest{}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("error code = %s, want unavailable: %v", connect.CodeOf(err), err)
	}
}
