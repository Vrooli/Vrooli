package support

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type recordingHTTPClient struct {
	request *http.Request
}

func TestNormalizeAndValidateRemoteProfileAPIBase(t *testing.T) {
	if got := NormalizeAPIBase(" https://api.example.test/ "); got != "https://api.example.test" {
		t.Fatalf("NormalizeAPIBase = %q", got)
	}
	for _, raw := range []string{"https://api.example.test/api/v1", "http://localhost:17691/api/v1"} {
		if _, err := ValidateRemoteProfileAPIBase(raw); err != nil {
			t.Fatalf("ValidateRemoteProfileAPIBase(%q): %v", raw, err)
		}
	}
	for _, raw := range []string{"", "ftp://example.test", "/relative"} {
		if _, err := ValidateRemoteProfileAPIBase(raw); err == nil {
			t.Fatalf("ValidateRemoteProfileAPIBase(%q) accepted invalid base", raw)
		}
	}
}

func TestNormalizeRemoteProfileIDAcceptsStringAndNumber(t *testing.T) {
	for _, test := range []struct {
		raw  json.RawMessage
		want string
	}{
		{json.RawMessage(`" profile-1 "`), "profile-1"},
		{json.RawMessage(`42`), "42"},
	} {
		got, err := NormalizeRemoteProfileID(test.raw)
		if err != nil || got != test.want {
			t.Fatalf("NormalizeRemoteProfileID(%s) = %q, %v; want %q", test.raw, got, err, test.want)
		}
	}
	if _, err := NormalizeRemoteProfileID(json.RawMessage(`{}`)); err == nil {
		t.Fatal("NormalizeRemoteProfileID accepted object")
	}
}

func TestParseAndFlattenQueryValues(t *testing.T) {
	values, err := ParseQueries([]string{"page=2", "tag=alpha", "tag=beta"})
	if err != nil || values.Get("page") != "2" || strings.Join(values["tag"], ",") != "alpha,beta" {
		t.Fatalf("ParseQueries = %#v, %v", values, err)
	}
	if _, err := ParseQueries([]string{"%zz"}); err == nil {
		t.Fatal("ParseQueries accepted invalid escaped query")
	}
	flattened := FlattenQueryValues(url.Values{"tag": {"alpha", "beta"}, "page": {"2"}})
	if flattened["tag"] != "alpha" || flattened["page"] != "2" {
		t.Fatalf("FlattenQueryValues = %#v", flattened)
	}
}

func TestResolvePathAndKeyValuePairs(t *testing.T) {
	path, names, err := ResolvePath("/admin/apps/{app_key}/assets/{id}", []string{"desktop", "12"}, false)
	if err != nil || path != "/admin/apps/desktop/assets/12" || strings.Join(names, ",") != "app_key,id" {
		t.Fatalf("ResolvePath = %q, %#v, %v", path, names, err)
	}
	if _, _, err := ResolvePath("/items/{id}", nil, false); err == nil {
		t.Fatal("ResolvePath accepted missing route argument")
	}
	pairs, err := ParseKeyValuePairs([]string{"env=production", "tier=pro"})
	if err != nil || pairs["tier"] != "pro" {
		t.Fatalf("ParseKeyValuePairs = %#v, %v", pairs, err)
	}
	if _, err := ParseKeyValuePairs([]string{"bad"}); err == nil {
		t.Fatal("ParseKeyValuePairs accepted malformed pair")
	}
}

func TestDownloadAndCookieHelpers(t *testing.T) {
	if got, err := NormalizeDownloadPlatform(" MAC "); err != nil || got != "mac" {
		t.Fatalf("NormalizeDownloadPlatform = %q, %v", got, err)
	}
	if _, err := NormalizeDownloadPlatform("ios"); err == nil {
		t.Fatal("NormalizeDownloadPlatform accepted ios")
	}
	if got := ResolveContentType("release.zip", ""); got != "application/zip" {
		t.Fatalf("ResolveContentType = %q", got)
	}
	if got := ResolveContentType("release.bin", "application/custom"); got != "application/custom" {
		t.Fatalf("ResolveContentType override = %q", got)
	}
	expires := time.Now().Add(time.Hour).Round(time.Second)
	cookies := []*http.Cookie{{Name: "other", Value: "x"}, {Name: "session", Value: "token", Expires: expires}}
	if got := FindCookie(cookies, "session"); got == nil || got.Value != "token" {
		t.Fatalf("FindCookie = %#v", got)
	}
	if got := DeriveCookieExpiry(FindCookie(cookies, "session")); got == nil || !got.Equal(expires) {
		t.Fatalf("DeriveCookieExpiry = %v", got)
	}
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
