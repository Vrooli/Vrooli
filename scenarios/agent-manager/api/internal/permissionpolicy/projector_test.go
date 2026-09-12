package permissionpolicy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
)

type fakeCommandExecutor struct {
	command string
	args    []string
	run     func(context.Context, string, ...string) ([]byte, error)
}

func (f *fakeCommandExecutor) Run(ctx context.Context, command string, args ...string) ([]byte, error) {
	f.command = command
	f.args = append([]string(nil), args...)
	return f.run(ctx, command, args...)
}

func TestResourcePermissionProjectorPlansStrictCorrelatedEvidence(t *testing.T) {
	request := projectionRequest(t, domain.RunnerTypeCodex)
	executor := &fakeCommandExecutor{run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
		data, err := os.ReadFile(args[3])
		if err != nil {
			t.Fatal(err)
		}
		return validPlanResponse("codex", "user", digest(data)), nil
	}}

	result, err := NewResourcePermissionProjector(executor).Plan(context.Background(), request)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if executor.command != "resource-codex" {
		t.Fatalf("command = %q", executor.command)
	}
	if got, want := append([]string(nil), executor.args[:3]...), []string{"permissions", "plan", "--document"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("args prefix = %#v, want %#v", got, want)
	}
	if !containsPair(executor.args, "--scope", "user") || contains(executor.args, "--i-was-explicitly-authorized") {
		t.Fatalf("args = %#v", executor.args)
	}
	if result.Runner != domain.RunnerTypeCodex || result.DesiredDigest == "" || result.Enforcement.Permissions != "intent_only" {
		t.Fatalf("result = %#v", result)
	}
	if !reflect.DeepEqual(result.NativePaths, []string{"/native/a", "/native/z"}) {
		t.Fatalf("native paths = %#v", result.NativePaths)
	}
	if _, err := os.Stat(executor.args[3]); !os.IsNotExist(err) {
		t.Fatalf("temporary document was not removed: %v", err)
	}
}

func TestResourcePermissionProjectorRejectsUncorrelatedResponse(t *testing.T) {
	request := projectionRequest(t, domain.RunnerTypeCodex)
	executor := &fakeCommandExecutor{run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return validPlanResponse("claude-code", "user", strings.Repeat("a", 64)), nil
	}}
	_, err := NewResourcePermissionProjector(executor).Plan(context.Background(), request)
	if !errors.Is(err, ErrInvalidResourceResponse) || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("Plan error = %v", err)
	}
}

func TestResourcePermissionProjectorRequiresAuthorizationForReconcile(t *testing.T) {
	called := false
	executor := &fakeCommandExecutor{run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		called = true
		return nil, nil
	}}
	_, err := NewResourcePermissionProjector(executor).Reconcile(context.Background(), projectionRequest(t, domain.RunnerTypeOpenCode), false)
	if !errors.Is(err, ErrAuthorizationRequired) || called {
		t.Fatalf("Reconcile error = %v, called = %t", err, called)
	}
}

func TestResourcePermissionProjectorForwardsAuthorizationAndBoundsTimeout(t *testing.T) {
	request := projectionRequest(t, domain.RunnerTypeOpenCode)
	executor := &fakeCommandExecutor{run: func(ctx context.Context, _ string, _ ...string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	_, err := newResourcePermissionProjector(executor, time.Millisecond).Reconcile(context.Background(), request, true)
	if !errors.Is(err, ErrResourceUnavailable) || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("Reconcile error = %v", err)
	}
	if !contains(executor.args, "--i-was-explicitly-authorized") {
		t.Fatalf("authorization flag not forwarded: %#v", executor.args)
	}
}

func TestResourcePermissionProjectorRejectsUnknownResponseFields(t *testing.T) {
	request := projectionRequest(t, domain.RunnerTypeOpenCode)
	executor := &fakeCommandExecutor{run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
		data, err := os.ReadFile(args[3])
		if err != nil {
			t.Fatal(err)
		}
		return []byte(strings.TrimSuffix(string(validPlanResponse("opencode", "user", digest(data))), "}") + `,"unexpected":true}`), nil
	}}
	_, err := NewResourcePermissionProjector(executor).Plan(context.Background(), request)
	if !errors.Is(err, ErrInvalidResourceResponse) || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Plan error = %v", err)
	}
}

func projectionRequest(t *testing.T, runner domain.RunnerType) ProjectionRequest {
	t.Helper()
	revision, err := Parse([]byte(validCatalogJSON))
	if err != nil {
		t.Fatalf("Parse catalog: %v", err)
	}
	document, err := revision.Catalog().ResourceDocument("user")
	if err != nil {
		t.Fatalf("ResourceDocument: %v", err)
	}
	return ProjectionRequest{Runner: runner, Document: document}
}

func validPlanResponse(runner, scope, desiredDigest string) []byte {
	response := map[string]any{
		"schema_version":      "v1",
		"runner":              runner,
		"scope":               scope,
		"desired_digest":      desiredDigest,
		"desired_fingerprint": "desired-fingerprint",
		"live_fingerprint":    "live-fingerprint",
		"drift":               true,
		"changes":             []string{"replace deny rules"},
		"native_paths":        []string{"/native/z", "/native/a"},
		"enforcement":         map[string]any{"permissions": "intent_only"},
	}
	data, err := json.Marshal(response)
	if err != nil {
		panic(fmt.Sprintf("marshal response: %v", err))
	}
	return data
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsPair(values []string, first, second string) bool {
	for index := 0; index+1 < len(values); index++ {
		if values[index] == first && values[index+1] == second {
			return true
		}
	}
	return false
}
