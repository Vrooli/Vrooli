package matrix

import (
	"context"
	"net/url"
	"testing"
)

type fakeHTTPClient struct {
	method string
	path   string
	body   interface{}
	result []byte
}

func (f *fakeHTTPClient) DoWithContext(_ context.Context, method, path string, _ url.Values, body interface{}) ([]byte, error) {
	f.method, f.path, f.body = method, path, body
	return f.result, nil
}

func TestRequestUsesMatrixHTTPContract(t *testing.T) {
	client := &fakeHTTPClient{result: []byte(`{"run_id":"matrix-1","state":"completed"}`)}
	value, err := request(nil, client, "/api/v1", "POST", "/validation/matrices/matrix-1/start", nil)
	if err != nil {
		t.Fatalf("request() error = %v", err)
	}
	if client.method != "POST" || client.path != "/api/v1/validation/matrices/matrix-1/start" {
		t.Fatalf("request() called %s %s, want POST /api/v1/validation/matrices/matrix-1/start", client.method, client.path)
	}
	if got := matrixID(value); got != "run_id=matrix-1" {
		t.Fatalf("matrixID() = %q, want run_id=matrix-1", got)
	}
	if got := matrixState(value); got != "completed" {
		t.Fatalf("matrixState() = %q, want completed", got)
	}
}

func TestTargetSelectorMatchesDescriptorAndNormalizesPlatforms(t *testing.T) {
	target := map[string]any{
		"os": "darwin",
		"descriptor": map[string]any{
			"target_id":    "bridge:node-1",
			"display_name": "minimouse",
		},
	}
	descriptor := target["descriptor"].(map[string]any)
	for _, selector := range []string{"bridge:node-1", "minimouse", "darwin", "mac", "macOS"} {
		if !targetSelectorMatches(selector, target, descriptor) {
			t.Errorf("targetSelectorMatches(%q) = false, want true", selector)
		}
	}
	if targetSelectorMatches("windows", target, descriptor) {
		t.Error("targetSelectorMatches(windows) = true, want false")
	}
}

func TestFindTargetCreatesHonestUnavailablePlatformWhenFleetHasNoMatch(t *testing.T) {
	client := &fakeHTTPClient{result: []byte(`{"targets":[]}`)}
	target, err := FindTarget(nil, client, "/api/v1", "windows")
	if err != nil {
		t.Fatal(err)
	}
	descriptor, ok := target["descriptor"].(map[string]any)
	if !ok {
		t.Fatalf("descriptor = %#v", target["descriptor"])
	}
	if descriptor["available"] != false || descriptor["target_id"] != "unavailable-windows" {
		t.Fatalf("unavailable descriptor = %#v", descriptor)
	}
	if descriptor["reason"] == "" {
		t.Fatal("unavailable descriptor has no reason")
	}
	if target["kind"] != "bridge" {
		t.Fatalf("target kind = %#v, want bridge", target["kind"])
	}
}
