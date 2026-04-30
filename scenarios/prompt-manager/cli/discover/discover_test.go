package discover

import (
	"encoding/json"
	"net/url"
	"testing"
)

type fakeContext struct {
	method   string
	path     string
	payload  any
	response any
}

func (f *fakeContext) Get(path string, result interface{}) error {
	f.method = "GET"
	f.path = path
	return f.writeResult(result)
}

func (f *fakeContext) GetWithQuery(path string, _ url.Values, result interface{}) error {
	f.method = "GET"
	f.path = path
	return f.writeResult(result)
}

func (f *fakeContext) Post(path string, payload interface{}, result interface{}) error {
	f.method = "POST"
	f.path = path
	f.payload = payload
	return f.writeResult(result)
}

func (f *fakeContext) Put(path string, payload interface{}, result interface{}) error {
	f.method = "PUT"
	f.path = path
	f.payload = payload
	return f.writeResult(result)
}

func (f *fakeContext) Delete(path string) error {
	f.method = "DELETE"
	f.path = path
	return nil
}

func (f *fakeContext) writeResult(result interface{}) error {
	if result == nil || f.response == nil {
		return nil
	}
	raw, err := json.Marshal(f.response)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, result)
}

func TestFormatNumberAddsThousandsSeparators(t *testing.T) {
	cases := map[int]string{
		999:     "999",
		1200:    "1,200",
		1234567: "1,234,567",
	}
	for input, expected := range cases {
		if got := formatNumber(input); got != expected {
			t.Fatalf("formatNumber(%d) = %q, want %q", input, got, expected)
		}
	}
}

func TestCommandsRegistersDiscoverCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Discovery" || len(group.Commands) != 1 {
		t.Fatalf("unexpected command group: %+v", group)
	}
	if group.Commands[0].Name != "discover" || !group.Commands[0].NeedsAPI {
		t.Fatalf("unexpected command metadata: %+v", group.Commands[0])
	}
}

func TestCmdDiscoverDefaultsToSkillRequest(t *testing.T) {
	ctx := &fakeContext{response: DiscoverResponse{}}
	if err := cmdDiscover(ctx, []string{"debugging"}); err != nil {
		t.Fatalf("cmdDiscover: %v", err)
	}
	if ctx.method != "POST" || ctx.path != "/discover" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
	req, ok := ctx.payload.(DiscoverRequest)
	if !ok {
		t.Fatalf("payload type = %T, want DiscoverRequest", ctx.payload)
	}
	if req.Type != "" {
		t.Fatalf("default request type = %q, want empty for legacy API compatibility", req.Type)
	}
}

func TestCmdDiscoverSendsActionType(t *testing.T) {
	ctx := &fakeContext{response: DiscoverResponse{}}
	if err := cmdDiscover(ctx, []string{"list decisions", "--type=action"}); err != nil {
		t.Fatalf("cmdDiscover: %v", err)
	}
	req, ok := ctx.payload.(DiscoverRequest)
	if !ok {
		t.Fatalf("payload type = %T, want DiscoverRequest", ctx.payload)
	}
	if req.Type != "action" {
		t.Fatalf("request type = %q, want action", req.Type)
	}
}

func TestCmdDiscoverRejectsInvalidType(t *testing.T) {
	ctx := &fakeContext{}
	if err := cmdDiscover(ctx, []string{"debugging", "--type=capability"}); err == nil {
		t.Fatal("expected invalid type to fail")
	}
	if ctx.method != "" {
		t.Fatalf("expected no API request, got %s %s", ctx.method, ctx.path)
	}
}
