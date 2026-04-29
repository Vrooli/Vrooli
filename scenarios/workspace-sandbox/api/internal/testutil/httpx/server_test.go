package httpx_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/handlers"
	"workspace-sandbox/internal/testutil/httpx"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/testutil/mocks/sandboxiface"
	"workspace-sandbox/internal/types"
)

// TestNewLiveServer_APIInfoRoute confirms the harness wires the
// production middleware + handler routes correctly. Hits the real /
// (APIInfo) endpoint through a real HTTP client and expects a JSON
// body with the canonical service identity.
//
// /api/v1/health is registered in main.go (using api-core/health), not
// by handlers.RegisterRoutes, so it is intentionally absent from the
// harness; tests cover the api-core/health wiring at a higher level
// (the production binary itself).
func TestNewLiveServer_APIInfoRoute(t *testing.T) {
	h := &handlers.Handlers{
		Clock:      clock.System{},
		DB:         mocks.NewFakePinger(),
		DriverSlot: driver.NewSlot(mocks.NewFakeDriver()),
		Config:     config.Config{},
		Service:    &sandboxiface.FakeService{},
	}
	live := httpx.NewLiveServer(t, h)

	resp, body := live.Do(t, "GET", "/", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, string(body))
	}

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode body: %v (body=%s)", err, string(body))
	}
	if got["service"] != "Workspace Sandbox API" {
		t.Errorf("service = %v, want Workspace Sandbox API", got["service"])
	}
	if got["apiVersion"] != "v1" {
		t.Errorf("apiVersion = %v, want v1", got["apiVersion"])
	}
}

// TestNewLiveServer_MiddlewareLogs verifies the middleware actually
// runs in the harness (production parity gate). Every request must emit
// an api.request log line with the captured status code.
func TestNewLiveServer_MiddlewareLogs(t *testing.T) {
	h := &handlers.Handlers{
		Clock:      clock.System{},
		DB:         mocks.NewFakePinger(),
		DriverSlot: driver.NewSlot(mocks.NewFakeDriver()),
		Config:     config.Config{},
		Service:    &sandboxiface.FakeService{},
	}
	live := httpx.NewLiveServer(t, h)

	resp, _ := live.Do(t, "GET", "/", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	logs := live.LogBuffer.String()
	if !strings.Contains(logs, `"event":"api.request"`) {
		t.Errorf("middleware log missing api.request entry: %s", logs)
	}
	if !strings.Contains(logs, `"path":"/"`) {
		t.Errorf("middleware log missing / path: %s", logs)
	}
	if !strings.Contains(logs, `"statusCode":200`) {
		t.Errorf("middleware log missing 200 statusCode: %s", logs)
	}
}

// TestNewLiveServer_NotFoundRoutesReturn404 sanity-checks that a route
// not registered by handlers.RegisterRoutes returns 404 (i.e., the
// harness doesn't accidentally register a catch-all).
func TestNewLiveServer_NotFoundRoutesReturn404(t *testing.T) {
	h := &handlers.Handlers{
		Clock:      clock.System{},
		DB:         mocks.NewFakePinger(),
		DriverSlot: driver.NewSlot(mocks.NewFakeDriver()),
		Config:     config.Config{},
		Service:    &sandboxiface.FakeService{},
	}
	live := httpx.NewLiveServer(t, h)

	resp, _ := live.Do(t, "GET", "/api/v1/non-existent", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// TestNewLiveServer_DoJSONMarshalsBodyAndContentType is a smoke test
// for the convenience wrapper. POST /sandboxes round-trips a JSON
// payload through the production middleware + handler.
func TestNewLiveServer_DoJSONMarshalsBodyAndContentType(t *testing.T) {
	id := uuid.New()
	svc := &sandboxiface.FakeService{
		CreateFn: func(ctx context.Context, req *types.CreateRequest) (*types.Sandbox, error) {
			return &types.Sandbox{
				ID:          id,
				ScopePath:   req.ScopePath,
				ProjectRoot: req.ProjectRoot,
				Owner:       req.Owner,
				Status:      types.StatusActive,
				DriverID:    "overlayfs-userns",
			}, nil
		},
	}
	h := &handlers.Handlers{
		Clock:      clock.System{},
		DB:         mocks.NewFakePinger(),
		DriverSlot: driver.NewSlot(mocks.NewFakeDriver()),
		Service:    svc,
		Config:     config.Config{},
	}
	live := httpx.NewLiveServer(t, h)

	resp, body := live.DoJSON(t, "POST", "/api/v1/sandboxes",
		`{"scopePath":"/p/src","projectRoot":"/p","owner":"agent"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, body=%s", resp.StatusCode, string(body))
	}
	var got types.Sandbox
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID = %v, want %v", got.ID, id)
	}
}
