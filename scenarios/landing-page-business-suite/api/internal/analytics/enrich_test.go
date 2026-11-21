package analytics

import (
	"net/http"
	"testing"
)

func TestEnrichNeverReturnsAddress(t *testing.T) {
	r := &http.Request{RemoteAddr: "203.0.113.10:1234", Header: http.Header{"User-Agent": []string{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)"}}}
	d := Enrich(r, "https://www.google.com/search?q=x")
	if d.ReferrerHost != "www.google.com" || d.ReferrerKind != "search" || d.DeviceClass != "mobile" {
		t.Fatalf("unexpected dimensions: %#v", d)
	}
}

func TestUntrustedGeoHeaderIgnored(t *testing.T) {
	r := &http.Request{RemoteAddr: "203.0.113.10:1234", Header: http.Header{"X-Geo-Country": []string{"US"}}}
	if got := Enrich(r, "").CountryCode; got != "" {
		t.Fatalf("untrusted country = %q", got)
	}
}

func TestReferrerKinds(t *testing.T) {
	for _, tc := range []struct{ host, want string }{{"", "direct"}, {"site.example", "direct"}, {"google.com", "search"}, {"facebook.com", "social"}, {"ads.example", "paid"}, {"partner.example", "referral"}} {
		if got := classifyReferrer(tc.host, "site.example"); got != tc.want {
			t.Errorf("%s: got %s want %s", tc.host, got, tc.want)
		}
	}
}
