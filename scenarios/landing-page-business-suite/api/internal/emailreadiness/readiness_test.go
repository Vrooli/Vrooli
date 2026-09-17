package emailreadiness

import (
	"context"
	"errors"
	"testing"
	"time"
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

func TestEvaluateReadinessPassAndWarn(t *testing.T) {
	f := fakeResolver{
		txt:   map[string][]string{"mail.vrooli.com": {"v=spf1 include:sendgrid.net -all"}, "_dmarc.vrooli.com": {"v=DMARC1; p=none"}},
		cname: map[string]string{"s1._domainkey.mail.vrooli.com": "s1.domainkey.sendgrid.net.", "s2._domainkey.mail.vrooli.com": "s2.domainkey.sendgrid.net."},
	}
	s := New(f)
	now := time.Unix(1700000000, 0)
	s.UseClock(func() time.Time { return now })
	r := s.Evaluate(context.Background(), "sign-in@mail.vrooli.com", "https://vrooli.com", true, "", true, now)
	statuses := map[string]string{}
	for _, c := range r.Checks {
		statuses[c.Name] = c.Status
	}
	if statuses["from_domain_configured"] != "pass" || statuses["alignment"] != "pass" || statuses["spf"] != "pass" || statuses["dkim"] != "pass" || statuses["dmarc"] != "warn" || statuses["event_webhook"] != "pass" {
		t.Fatalf("checks = %#v", r.Checks)
	}
}

func TestEvaluateReadinessFailsMissingRecords(t *testing.T) {
	s := New(fakeResolver{})
	r := s.Evaluate(context.Background(), "noreply@example.com", "https://vrooli.com", true, "", false, time.Time{})
	statuses := map[string]string{}
	for _, c := range r.Checks {
		statuses[c.Name] = c.Status
	}
	for _, name := range []string{"from_domain_configured", "alignment", "spf", "dkim", "dmarc"} {
		if statuses[name] != "fail" {
			t.Errorf("%s = %s", name, statuses[name])
		}
	}
}
