package maildns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
)

type fakeResolver struct {
	txt   map[string][]string
	cname map[string]string
}

func (f fakeResolver) LookupTXT(_ context.Context, name string) ([]string, error) {
	v, ok := f.txt[name]
	if !ok {
		return nil, errors.New("not found")
	}
	return v, nil
}
func (f fakeResolver) LookupCNAME(_ context.Context, name string) (string, error) {
	v, ok := f.cname[name]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func TestVerifyProviderWithTXTAndCNAMEDKIM(t *testing.T) {
	f := fakeResolver{txt: map[string][]string{"vrooli.com": {"v=spf1 include:mailgun.org ~all"}, "mailgun.org": {"v=spf1 include:_spf.eu.mailgun.org ~all"}, "_spf.eu.mailgun.org": {"v=spf1 include:_spf.mailgun-edge.org ~all"}, "_spf.mailgun-edge.org": {"v=spf1 a ~all"}, "k1._domainkey.vrooli.com": {"v=DKIM1; k=rsa; p=mailgun-key"}, "_dmarc.vrooli.com": {"v=DMARC1; p=reject; aspf=s"}}, cname: map[string]string{"s1._domainkey.vrooli.com": "s1.provider.example."}}
	s := New(f)
	r := s.Verify(context.Background(), "vrooli.com", []ProviderRequirement{{Name: "Mailgun", SPFMechanism: "include:mailgun.org", DKIM: []DKIMRequirement{{Selector: "k1", Type: "TXT", Expected: "mailgun-key"}}}, {Name: "Other", SPFMechanism: "include:other.example", DKIM: []DKIMRequirement{{Selector: "s1", Type: "CNAME", Expected: "provider.example"}}}})
	if r.Providers[0].Status != Pass || r.Providers[0].Cost != 3 {
		t.Fatalf("mailgun = %#v", r.Providers[0])
	}
	if r.Providers[1].Status != Fail || r.Providers[1].Record == "" {
		t.Fatalf("other = %#v", r.Providers[1])
	}
	if r.DMARC.Status != Pass || r.DMARC.Detail == "" {
		t.Fatalf("dmarc = %#v", r.DMARC)
	}
}

func TestVerifyCacheIncludesDomainAndRequirements(t *testing.T) {
	f := fakeResolver{txt: map[string][]string{"a.example": {"v=spf1 include:a.example ~all"}, "_dmarc.a.example": {"v=DMARC1; p=reject"}, "b.example": {"v=spf1 include:b.example ~all"}, "_dmarc.b.example": {"v=DMARC1; p=reject"}}}
	s := New(f)
	if got := s.Verify(context.Background(), "a.example", nil).Domain; got != "a.example" {
		t.Fatal(got)
	}
	if got := s.Verify(context.Background(), "b.example", nil).Domain; got != "b.example" {
		t.Fatal(got)
	}
}

func TestVerifyRejectsDuplicateSPFAndMissingMechanism(t *testing.T) {
	f := fakeResolver{txt: map[string][]string{
		"duplicate.example":        {"v=spf1 include:mailgun.org ~all", "v=spf1 -all"},
		"missing.example":          {"v=spf1 include:other.example ~all"},
		"other.example":            {"v=spf1 a ~all"},
		"_dmarc.duplicate.example": {"v=DMARC1; p=reject"},
		"_dmarc.missing.example":   {"v=DMARC1; p=reject"},
	}}
	s := New(f)
	for _, tc := range []struct{ domain, mechanism, want string }{{"duplicate.example", "include:mailgun.org", "exactly one SPF"}, {"missing.example", "include:mailgun.org", "does not authorize"}} {
		r := s.Verify(context.Background(), tc.domain, []ProviderRequirement{{Name: "Mailgun", SPFMechanism: tc.mechanism}})
		if r.Providers[0].Status != Fail || !strings.Contains(r.Providers[0].Detail, tc.want) {
			t.Fatalf("%s verdict = %#v", tc.domain, r.Providers[0])
		}
	}
}

func TestVerifyRejectsSPFOverLookupBudget(t *testing.T) {
	txt := map[string][]string{"budget.example": {"v=spf1 include:spf-0.example ~all"}, "_dmarc.budget.example": {"v=DMARC1; p=reject"}}
	for i := 0; i < 11; i++ {
		host := fmt.Sprintf("spf-%d.example", i)
		if i == 10 {
			txt[host] = []string{"v=spf1 a ~all"}
		} else {
			txt[host] = []string{fmt.Sprintf("v=spf1 include:spf-%d.example ~all", i+1)}
		}
	}
	r := New(fakeResolver{txt: txt}).Verify(context.Background(), "budget.example", []ProviderRequirement{{Name: "Mailgun", SPFMechanism: "include:spf-0.example"}})
	if r.Providers[0].Status != Fail || !strings.Contains(r.Providers[0].Detail, "cost 11") {
		t.Fatalf("budget verdict = %#v", r.Providers[0])
	}
}

func TestVrooliLiveRecordsAuthorizeMailgun(t *testing.T) {
	resolver := NetResolver{Resolver: net.DefaultResolver}
	r := New(resolver).Verify(context.Background(), "vrooli.com", []ProviderRequirement{{Name: "Mailgun", SPFMechanism: "include:mailgun.org", DKIM: []DKIMRequirement{{Selector: "k1", Type: "TXT"}}}})
	if len(r.Providers) != 1 || r.Providers[0].Status != Pass {
		t.Fatalf("live Mailgun verdict = %#v", r)
	}
	if r.Providers[0].Cost != 5 {
		t.Fatalf("live SPF lookup cost = %d, want 5 from the currently published nested includes", r.Providers[0].Cost)
	}
	if r.DMARC.Status != Pass || !strings.Contains(strings.ToLower(r.DMARC.Detail), "p=reject") {
		t.Fatalf("live DMARC verdict = %#v", r.DMARC)
	}
}
