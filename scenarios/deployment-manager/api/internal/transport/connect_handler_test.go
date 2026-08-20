package transport

import (
	"context"
	"net/http"
	"testing"

	"deployment-manager/dependencies"
	"deployment-manager/deployments"
	"deployment-manager/fitness"
	"deployment-manager/swaps"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRoutesExposeEveryGeneratedOperatorService(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	routes := Routes(h)
	if got, want := len(routes), 9; got != want {
		t.Fatalf("generated service route count = %d, want %d", got, want)
	}
	for _, route := range routes {
		if route.Path == "" || route.Handler == nil {
			t.Fatalf("invalid generated route: %#v", route)
		}
	}
}

func TestTransportMethodsFailClosedWhenOptionalHandlerIsMissing(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	req := connect.NewRequest(structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
		"profile_id":      structpb.NewStringValue("profile/1"),
		"deployment_id":   structpb.NewStringValue("deployment/1"),
		"scenario":        structpb.NewStringValue("demo"),
		"from":            structpb.NewStringValue("postgres"),
		"to":              structpb.NewStringValue("sqlite"),
		"id":              structpb.NewStringValue("approval/1"),
		"release_id":      structpb.NewStringValue("release/1"),
		"git_commit_hash": structpb.NewStringValue("abc"),
		"name":            structpb.NewStringValue("task"),
	}}))
	tests := []struct {
		name string
		call func(context.Context, *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error)
	}{
		{"list telemetry", h.ListTelemetry},
		{"upload telemetry", h.UploadTelemetry},
		{"report migration", h.ReportMigration},
		{"status migration", h.StatusMigration},
		{"get lpbs config", h.GetLPBSConfig},
		{"save lpbs config", h.SaveLPBSConfig},
		{"list releases", h.ListReleases},
		{"get release", h.GetRelease},
		{"reverify release", h.ReverifyRelease},
		{"start release", h.StartRelease},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.call(context.Background(), req)
			if connect.CodeOf(err) != connect.CodeUnimplemented {
				t.Fatalf("error code = %s, want unimplemented: %v", connect.CodeOf(err), err)
			}
		})
	}
}

func TestInvokeTranslatesHTTPResponsesAndRequestShape(t *testing.T) {
	var gotMethod, gotPath, gotQuery, gotContentType string
	fn := func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotQuery, gotContentType = r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"count":2}`))
	}
	h := NewHandler(nil, nil, nil, nil, nil, fn, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	value := structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{"name": structpb.NewStringValue("demo")}})
	response, err := h.ListTelemetry(context.Background(), connect.NewRequest(value))
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet || gotPath != "/api/v1/telemetry" || gotContentType != "" {
		t.Fatalf("request = %s %s content-type=%q", gotMethod, gotPath, gotContentType)
	}
	if response.Msg.GetStructValue().Fields["ok"].GetBoolValue() != true || gotQuery != "" {
		t.Fatalf("response/query = %v/%q", response.Msg, gotQuery)
	}

	response, err = h.invoke(context.Background(), http.MethodPost, "/path", value, fn, nil, nil)
	if err != nil || response == nil {
		t.Fatalf("payload invoke error = %v", err)
	}
	if gotMethod != http.MethodPost || gotContentType != "application/json" {
		t.Fatalf("payload request = %s content-type=%q", gotMethod, gotContentType)
	}
}

func TestInvokeMapsHTTPAndDecodeFailures(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   connect.Code
	}{
		{"bad request", http.StatusBadRequest, `{"error":"bad"}`, connect.CodeInvalidArgument},
		{"not found", http.StatusNotFound, `{"error":"missing"}`, connect.CodeNotFound},
		{"conflict", http.StatusConflict, `{"error":"duplicate"}`, connect.CodeAlreadyExists},
		{"unauthorized", http.StatusUnauthorized, `{"error":"auth"}`, connect.CodeUnauthenticated},
		{"forbidden", http.StatusForbidden, `{"error":"forbidden"}`, connect.CodePermissionDenied},
		{"server error", http.StatusInternalServerError, `{"error":"broken"}`, connect.CodeInternal},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(nil, nil, nil, nil, nil, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
			_, err := h.ListTelemetry(context.Background(), connect.NewRequest(structpb.NewNullValue()))
			if connect.CodeOf(err) != tc.want {
				t.Fatalf("error code = %s, want %s: %v", connect.CodeOf(err), tc.want, err)
			}
		})
	}

	h := NewHandler(nil, nil, nil, nil, nil, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("not-json")) }, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := h.ListTelemetry(context.Background(), connect.NewRequest(structpb.NewNullValue()))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("decode error code = %s, want internal", connect.CodeOf(err))
	}
}

func TestFieldAndStatusHelpers(t *testing.T) {
	value := structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{"name": structpb.NewStringValue("value")}})
	if got := stringField(value, "name"); got != "value" {
		t.Fatalf("stringField = %q", got)
	}
	if got := stringFieldDefault(value, "missing", "fallback"); got != "fallback" {
		t.Fatalf("stringFieldDefault = %q", got)
	}
	if got := stringFieldDefault(nil, "missing", "fallback"); got != "fallback" {
		t.Fatalf("nil stringFieldDefault = %q", got)
	}
	for status, want := range map[int]connect.Code{
		http.StatusBadRequest:   connect.CodeInvalidArgument,
		http.StatusNotFound:     connect.CodeNotFound,
		http.StatusConflict:     connect.CodeAlreadyExists,
		http.StatusUnauthorized: connect.CodeUnauthenticated,
		http.StatusForbidden:    connect.CodePermissionDenied,
		http.StatusOK:           connect.CodeInternal,
	} {
		if got := codeForStatus(status); got != want {
			t.Errorf("codeForStatus(%d) = %s, want %s", status, got, want)
		}
	}
}

func TestDomainAdaptersTranslateRealHandlerResponses(t *testing.T) {
	logger := func(string, map[string]interface{}) {}
	h := NewHandler(
		dependencies.NewHandler(logger),
		fitness.NewHandler(logger),
		deployments.NewHandler(logger),
		nil,
		swaps.NewHandler(nil, logger),
		func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"status":"ok"}`)) },
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	call := func(fields map[string]string) *connect.Request[structpb.Value] {
		values := make(map[string]*structpb.Value, len(fields))
		for key, value := range fields {
			values[key] = structpb.NewStringValue(value)
		}
		return connect.NewRequest(structpb.NewStructValue(&structpb.Struct{Fields: values}))
	}
	if response, err := h.Score(context.Background(), call(map[string]string{"scenario": "demo"})); err != nil || response == nil {
		t.Fatalf("Score() = %v, %v", response, err)
	}
	if _, err := h.Analyze(context.Background(), call(map[string]string{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("Analyze() error code = %s, want invalid argument: %v", connect.CodeOf(err), err)
	}
	if _, err := h.Deploy(context.Background(), call(map[string]string{"profile_id": "bad"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("Deploy() error code = %s, want not found: %v", connect.CodeOf(err), err)
	}
	if _, err := h.Status(context.Background(), call(map[string]string{"deployment_id": "missing"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("Status() error code = %s, want not found: %v", connect.CodeOf(err), err)
	}
	if response, err := h.AnalyzeSwaps(context.Background(), call(map[string]string{"from": "postgres", "to": "sqlite"})); err != nil || response == nil {
		t.Fatalf("AnalyzeSwaps() = %v, %v", response, err)
	}
	if response, err := h.CascadeSwaps(context.Background(), call(map[string]string{"from": "postgres", "to": "sqlite"})); err != nil || response == nil {
		t.Fatalf("CascadeSwaps() = %v, %v", response, err)
	}
	if response, err := h.ListTelemetry(context.Background(), call(nil)); err != nil || response == nil {
		t.Fatalf("ListTelemetry() = %v, %v", response, err)
	}
	if _, err := h.ListSwaps(context.Background(), call(nil)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("ListSwaps() error code = %s, want invalid argument: %v", connect.CodeOf(err), err)
	}
	if _, err := h.AnalyzeSwaps(context.Background(), call(map[string]string{"from": "", "to": ""})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("AnalyzeSwaps(empty) error code = %s, want invalid argument: %v", connect.CodeOf(err), err)
	}
	if _, err := h.CascadeSwaps(context.Background(), call(map[string]string{"from": "", "to": ""})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("CascadeSwaps(empty) error code = %s, want invalid argument: %v", connect.CodeOf(err), err)
	}
	if _, err := h.ApplySwaps(context.Background(), call(nil)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("ApplySwaps() error code = %s, want invalid argument: %v", connect.CodeOf(err), err)
	}
	if _, err := h.ApplySwapsToProfile(context.Background(), call(nil)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("ApplySwapsToProfile() error code = %s, want invalid argument: %v", connect.CodeOf(err), err)
	}
}
