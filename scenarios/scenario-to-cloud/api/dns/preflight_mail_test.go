package dns

import (
	"testing"

	maildns "github.com/vrooli/vrooli/packages/maildns-go"
	"scenario-to-cloud/domain"
)

func TestMailDNSChecksRequiredBlocksIncompleteProvider(t *testing.T) {
	checks := MailDNSChecksFromReport(maildns.Report{Providers: []maildns.Verdict{{
		Provider: "sendgrid", Status: maildns.Fail,
		Detail: "DKIM selector is missing", Record: "CNAME s1._domainkey.example.com = sendgrid.net",
	}}}, domain.DNSPolicyRequired)

	if len(checks) != 1 || checks[0].Status != domain.PreflightFail {
		t.Fatalf("expected required mail DNS failure, got %+v", checks)
	}
	if checks[0].Data["record"] == "" {
		t.Fatal("expected failing check to name the record to publish")
	}
}

func TestMailDNSChecksWarnPolicyDoesNotBlock(t *testing.T) {
	checks := MailDNSChecksFromReport(maildns.Report{Providers: []maildns.Verdict{{
		Provider: "sendgrid", Status: maildns.Fail, Detail: "SPF missing", Record: "TXT example.com = v=spf1 include:sendgrid.net ...",
	}}}, domain.DNSPolicyWarn)

	if len(checks) != 1 || checks[0].Status != domain.PreflightWarn {
		t.Fatalf("expected warn-policy mail DNS warning, got %+v", checks)
	}
}
