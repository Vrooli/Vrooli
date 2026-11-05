package workflows

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestNormalizeAdhocRequestMapsShortExecutionModeBeforeConnectDecode(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
	handler := normalizeAdhocRequest(next)
	req := httptest.NewRequest(http.MethodPost, "/browser_automation_studio.v1.workflows.WorkflowsService/ExecuteAdhocWorkflow", strings.NewReader(`{"flow_definition":{"metadata":{"execution_mode":"mutating"}}}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "EXECUTION_MODE_MUTATING") {
		t.Fatalf("normalized body = %s", response.Body.String())
	}
}

func TestNormalizeAdhocRequestLeavesOtherProceduresUntouched(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(body)
	})
	req := httptest.NewRequest(http.MethodPost, "/browser_automation_studio.v1.workflows.WorkflowsService/ValidateWorkflow", strings.NewReader(`{"execution_mode":"mutating"}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	normalizeAdhocRequest(next).ServeHTTP(response, req)
	if response.Body.String() != `{"execution_mode":"mutating"}` {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestNormalizeAdhocRequestPreservesExecutionControlsAndCustomParams(t *testing.T) {
	body := `{"flow_definition":{"metadata":{},"nodes":[],"edges":[]},"parameters":{"initialParams":{"baseUrl":{"stringValue":"http://127.0.0.1:43187"},"base_url":{"stringValue":"distinct snake key"},"existing":{"stringValue":"kept"},"collision":{"stringValue":"nested wins"}},"startUrl":"http://127.0.0.1:43187","projectRoot":"/tmp/project","headless":true,"viewportWidth":390,"viewportHeight":844,"locale":"en-US","continueOnError":false,"custom_flag":false,"collision":"flat loses"}}`
	var decoded *basexecution.ExecuteAdhocRequest
	var decodeErr error
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		decoded = &basexecution.ExecuteAdhocRequest{}
		decodeErr = protojson.Unmarshal(body, decoded)
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "/browser_automation_studio.v1.workflows.WorkflowsService/ExecuteAdhocWorkflow", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	normalizeAdhocRequest(next).ServeHTTP(response, req)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	if decodeErr != nil {
		t.Fatalf("normalized request failed generated ProtoJSON decoding: %v", decodeErr)
	}
	if decoded == nil || decoded.Parameters == nil {
		t.Fatal("decoded request omitted parameters")
	}
	params := decoded.Parameters
	if params.Headless == nil || !*params.Headless || params.ViewportWidth == nil || *params.ViewportWidth != 390 || params.ViewportHeight == nil || *params.ViewportHeight != 844 || params.Locale == nil || *params.Locale != "en-US" || params.ContinueOnError == nil || *params.ContinueOnError {
		t.Fatalf("execution controls changed during normalization: %+v", params)
	}
	if params.StartUrl == nil || *params.StartUrl != "http://127.0.0.1:43187" || params.ProjectRoot == nil || *params.ProjectRoot != "/tmp/project" {
		t.Fatalf("known string controls changed during normalization: %+v", params)
	}
	customFlag, ok := params.GetInitialParams()["custom_flag"]
	if _, boolValue := customFlag.GetKind().(*commonv1.JsonValue_BoolValue); !ok || !boolValue || customFlag.GetBoolValue() {
		t.Fatalf("custom boolean value = %v, want explicit false", customFlag)
	}
	if got := params.GetInitialParams()["collision"].GetStringValue(); got != "nested wins" {
		t.Fatalf("existing initialParams collision = %q, want nested value", got)
	}
	if got := params.GetInitialParams()["baseUrl"].GetStringValue(); got != "http://127.0.0.1:43187" {
		t.Fatalf("existing initialParams baseUrl = %q, want caller value", got)
	}
	if got := params.GetInitialParams()["base_url"].GetStringValue(); got != "distinct snake key" {
		t.Fatalf("dynamic initialParams key base_url = %q, want unchanged key and value", got)
	}
}

func TestNormalizeAdhocRequestDoesNotAcceptMalformedProtoJSON(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var decoded basexecution.ExecuteAdhocRequest
		if err := protojson.Unmarshal(body, &decoded); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "/browser_automation_studio.v1.workflows.WorkflowsService/ExecuteAdhocWorkflow", strings.NewReader(`{"parameters":{"headless":false`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	normalizeAdhocRequest(next).ServeHTTP(response, req)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("malformed request status = %d, want 400: %s", response.Code, response.Body.String())
	}
}
