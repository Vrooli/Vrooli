package support

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type recordingHTTPClient struct {
	request *http.Request
}

func (c *recordingHTTPClient) Do(request *http.Request) (*http.Response, error) {
	c.request = request
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
}

func TestParseBodyValidatesJSON(t *testing.T) {
	body, err := ParseBody(`{"enabled":true}`)
	if err != nil {
		t.Fatalf("ParseBody returned an error for valid JSON: %v", err)
	}
	if string(body) != `{"enabled":true}` {
		t.Fatalf("ParseBody returned %q, want original JSON", body)
	}
	if _, err := ParseBody(`{`); err == nil {
		t.Fatal("ParseBody accepted malformed JSON")
	}
}

func TestAdminConnectHTTPClientAddsSessionWithoutDiscardingCookies(t *testing.T) {
	next := &recordingHTTPClient{}
	client := adminConnectHTTPClient{next: next, session: "signed-session"}
	req, err := http.NewRequest(http.MethodGet, "https://example.test/rpc", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: "existing", Value: "value"})

	response, err := client.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer response.Body.Close()
	if next.request == nil {
		t.Fatal("wrapped client did not receive request")
	}

	cookies := next.request.Cookies()
	got := make(map[string]string, len(cookies))
	for _, cookie := range cookies {
		got[cookie.Name] = cookie.Value
	}
	if got["admin_session"] != "signed-session" {
		t.Fatalf("admin session cookie = %q, want signed-session", got["admin_session"])
	}
	if got["existing"] != "value" {
		t.Fatalf("existing cookie = %q, want value", got["existing"])
	}
}
