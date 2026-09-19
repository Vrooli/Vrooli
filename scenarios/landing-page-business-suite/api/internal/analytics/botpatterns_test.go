package analytics

import "testing"

func TestClassifyTraffic(t *testing.T) {
	for _, ua := range []string{"Googlebot/2.1", "curl/8.0", "Mozilla/5.0 HeadlessChrome", "python-requests/2.31", "facebookexternalhit/1.1", "Mozilla/5.0 (compatible; bingbot/2.0)"} {
		if got := ClassifyTraffic(ua, "", ""); got != "bot" {
			t.Errorf("%q: got %s", ua, got)
		}
	}
	for _, ua := range []string{"Mozilla/5.0 Chrome/125.0", "Mozilla/5.0 Safari/17.0", "Mozilla/5.0 Firefox/126.0", "Mozilla/5.0 Edg/125.0", "Mozilla/5.0 iPhone"} {
		if got := ClassifyTraffic(ua, "", ""); got != "human" {
			t.Errorf("%q: got %s", ua, got)
		}
	}
	if got := ClassifyTraffic("Mozilla/5.0", "admin_session=abc", ""); got != "internal" {
		t.Errorf("admin cookie: got %s", got)
	}
	if got := ClassifyTraffic("Googlebot", "", "1"); got != "internal" {
		t.Errorf("internal header: got %s", got)
	}
}
