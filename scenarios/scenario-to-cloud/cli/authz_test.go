package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/errors"

	"scenario-to-cloud/cli/internal/apierr"
	"scenario-to-cloud/cli/internal/testfakes"
)

// TestClientAttachesOperatorToken [REQ:STC-P0-012] proves protected commands
// send the operator token as a bearer credential on both transports: the
// generated Connect client (deployment list) and the REST client (manifest
// schema).
func TestClientAttachesOperatorToken(t *testing.T) {
	app := newTestApp(t)
	ps := newParityServer(t)
	var restSeen string
	ps.Mux.HandleFunc("/api/v1/manifest/schema", func(w http.ResponseWriter, r *http.Request) {
		restSeen = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"schema":{}}`)
	})
	captureStdout(t, func() {
		if err := app.Run([]string{"deployment", "list"}); err != nil {
			t.Fatalf("deployment list: %v", err)
		}
		if err := app.Run([]string{"manifest", "schema"}); err != nil {
			t.Fatalf("manifest schema: %v", err)
		}
	})
	connectSeen := false
	for _, h := range ps.headers {
		if h == "Bearer test-token" {
			connectSeen = true
		}
	}
	if !connectSeen || restSeen != "Bearer test-token" {
		t.Fatalf("Authorization connect=%v rest=%q, want the operator bearer token on both", ps.headers, restSeen)
	}
}

// TestUnauthenticatedRefusalExitsTwoWithNextAction [REQ:STC-P0-012] proves a
// typed 401 maps to exit 2 and surfaces the sign-in next action on both the
// REST envelope and the Connect error detail.
func TestUnauthenticatedRefusalExitsTwoWithNextAction(t *testing.T) {
	app := newTestApp(t)
	ps := newParityServer(t)
	ps.Mux.HandleFunc("/api/v1/manifest/schema", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"code":"unauthenticated","message":"A verified operator principal is required","retryable":false,"next_action":{"owner":"operator","kind":"sign_in","reference":"docs/reference/configuration.md#authentication","label":"Sign in and retry"},"details":{"reason":"missing"}}}`)
	})
	ps.Deployments.ListError = testfakes.TypedErrorWithNextAction(connect.CodeUnauthenticated, "unauthenticated", "A verified operator principal is required", map[string]any{"reason": "missing"},
		&errorsv1.NextAction{Owner: "operator", Kind: "sign_in", Reference: "docs/reference/configuration.md#authentication", Label: "Sign in and retry"})

	for _, args := range [][]string{{"manifest", "schema"}, {"deployment", "list"}} {
		var err error
		captureStdout(t, func() { err = app.Run(args) })
		if err == nil {
			t.Fatalf("%v: expected an error", args)
		}
		if code := apierr.ExitCode(err); code != apierr.ExitRefused {
			t.Fatalf("%v: exit code = %d, want %d", args, code, apierr.ExitRefused)
		}
		typed, ok := apierr.Decode(err)
		if !ok || typed.Code != "unauthenticated" || typed.NextAction == nil || typed.NextAction.Kind != "sign_in" {
			t.Fatalf("%v: typed = %+v ok=%v", args, typed, ok)
		}
		message := apierr.Format(err)
		for _, want := range []string{"unauthenticated", "Sign in", "SCENARIO_TO_CLOUD_API_TOKEN", "vrooli-bridge auth"} {
			if !strings.Contains(message, want) {
				t.Fatalf("%v: message missing %q: %s", args, want, message)
			}
		}
	}
	for code, want := range map[string]int{"forbidden_target": 2, "forbidden_revoked": 2, "forbidden_origin": 2, "needs_input": 3, "pending_operator_input": 3, "plan_stale": 2, "recovery_point_corrupt": 2, "internal": 1, "backup_failed": 1} {
		if got := (&apierr.Typed{Code: code}).ExitCode(); got != want {
			t.Fatalf("%s exit = %d, want %d", code, got, want)
		}
	}
}
