package edge

import (
	"errors"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/domain"
)

// [REQ:STC-P0-036] Step 4/5: A/AAAA bind independently to the target
// addresses; a missing AAAA is not a failure; an AAAA pointing elsewhere is;
// an IPv6-only target binds on AAAA alone.
func TestBindDNSHandlesFamiliesIndependently(t *testing.T) {
	v4 := domain.EdgeIPPolicy{IPv4: []string{"203.0.113.10"}, IPv6: []string{}}
	dual := domain.EdgeIPPolicy{IPv4: []string{"203.0.113.10"}, IPv6: []string{"2001:db8::10"}}
	v6only := domain.EdgeIPPolicy{IPv4: []string{}, IPv6: []string{"2001:db8::10"}}
	cases := []struct {
		name     string
		observed []string
		err      error
		policy   domain.EdgeIPPolicy
		match    bool
		reason   string
		v4, v6   string
	}{
		{"a only matches v4 target", []string{"203.0.113.10"}, nil, v4, true, ReasonDNSMatch, domain.EdgeBindingMatch, domain.EdgeBindingAbsent},
		{"a only matches dual target (aaaa absent ok)", []string{"203.0.113.10"}, nil, dual, true, ReasonDNSMatch, domain.EdgeBindingMatch, domain.EdgeBindingAbsent},
		{"wrong a record", []string{"198.51.100.7"}, nil, v4, false, ReasonDNSMismatch, domain.EdgeBindingMismatch, domain.EdgeBindingAbsent},
		{"aaaa points elsewhere", []string{"203.0.113.10", "2001:db8::99"}, nil, dual, false, ReasonDNSIPv6Mismatch, domain.EdgeBindingMatch, domain.EdgeBindingMismatch},
		{"aaaa present but target has no v6", []string{"203.0.113.10", "2001:db8::99"}, nil, v4, false, ReasonDNSIPv6Mismatch, domain.EdgeBindingMatch, domain.EdgeBindingMismatch},
		{"ipv6 only target", []string{"2001:db8::10"}, nil, v6only, true, ReasonDNSMatch, domain.EdgeBindingAbsent, domain.EdgeBindingMatch},
		{"ipv6 only target wrong aaaa", []string{"2001:db8::11"}, nil, v6only, false, ReasonDNSIPv6Mismatch, domain.EdgeBindingAbsent, domain.EdgeBindingMismatch},
		{"no records", []string{}, nil, dual, false, ReasonDNSNoRecord, domain.EdgeBindingAbsent, domain.EdgeBindingAbsent},
		{"lookup failed", nil, errors.New("nxdomain"), dual, false, ReasonDNSUnresolved, domain.EdgeBindingUnresolved, domain.EdgeBindingUnresolved},
		{"target unobserved", []string{"203.0.113.10"}, nil, domain.EdgeIPPolicy{}, false, ReasonTargetUnobserved, domain.EdgeBindingUnresolved, domain.EdgeBindingUnresolved},
		{"cloudflare proxied", []string{"104.16.1.1"}, nil, v4, true, ReasonDNSProxied, domain.EdgeBindingProxied, domain.EdgeBindingProxied},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := BindDNS("app.example.test", tc.observed, tc.err, tc.policy)
			if b.Match != tc.match || b.ReasonCode != tc.reason || b.IPv4.State != tc.v4 || b.IPv6.State != tc.v6 {
				t.Fatalf("binding = %+v", b)
			}
		})
	}
}

// [REQ:STC-P0-036] P14-A05: wrong DNS blocks the external claim even when
// the public probe would pass; local readiness is reported separately.
func TestReadinessSeparatesLocalFromExternalAndBlocksOnDNS(t *testing.T) {
	r := Readiness(nil, nil, false)
	if r.Local.Status != "passed" || r.External.Status != "blocked" || r.External.ReasonCode != ReasonExternalBlocked || ExternallyReady(r) {
		t.Fatalf("wrong dns readiness = %+v", r)
	}
	r = Readiness(errors.New("connection refused"), nil, true)
	if r.Local.Status != "failed" || r.Local.ReasonCode != ReasonLocalUnready || r.External.Status != "passed" || ExternallyReady(r) {
		t.Fatalf("local failure readiness = %+v", r)
	}
	r = Readiness(nil, errors.New("tls handshake"), true)
	if r.External.Status != "failed" || r.External.ReasonCode != ReasonExternalUnready || ExternallyReady(r) {
		t.Fatalf("external failure readiness = %+v", r)
	}
	if !ExternallyReady(Readiness(nil, nil, true)) {
		t.Fatal("bound dns with passing probes must be ready")
	}
}

// [REQ:STC-P0-036] Step 11: blocked challenge ports and conflicting edge
// services carry typed reason codes; the edge proxy holding its own ports is
// not a conflict.
func TestClassifyEdgePortsTypesConflictsAndBlocks(t *testing.T) {
	issues := ClassifyEdgePorts([]PortObservation{{Port: 80, Process: "nginx", Service: "nginx"}, {Port: 443, Process: "caddy", Service: "caddy"}, {Port: 22, Process: "sshd"}}, []int{80})
	if len(issues) != 3 {
		t.Fatalf("issues = %+v", issues)
	}
	if issues[0] != (PortIssue{Port: 80, ReasonCode: ReasonEdgePortBlocked}) || issues[1] != (PortIssue{Port: 80, ReasonCode: ReasonEdgePortConflict, Process: "nginx"}) || issues[2] != (PortIssue{Port: 443, ReasonCode: ReasonEdgePortOwned, Process: "caddy"}) {
		t.Fatalf("issues = %+v", issues)
	}
	if !HasConflict(issues) || HasConflict(issues[2:]) {
		t.Fatal("conflict detection wrong")
	}
	if a := NextActionFreePort(issues[1]); !strings.Contains(a.Reference, "process.stop.scoped") || strings.Contains(a.Reference, "kill") || strings.Contains(a.Reference, "sudo") {
		t.Fatalf("free-port action = %+v", a)
	}
	if a := NextActionFirewall(443); !strings.Contains(a.Reference, "edge.ufw.allow") || !strings.Contains(a.Reference, `"port":443`) {
		t.Fatalf("firewall action = %+v", a)
	}
}

// [REQ:STC-P0-036] P14-A04: expiry and renewal failure are observed with an
// actionable state before the certificate lapses.
func TestObserveTLSReportsRenewalStates(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	cert := func(days int, valid bool) CertificateFacts {
		return CertificateFacts{Present: true, Valid: valid, Issuer: "R11", NotAfter: now.Add(time.Duration(days) * 24 * time.Hour), SANs: []string{"app.example.test"}}
	}
	cases := []struct {
		name    string
		cert    CertificateFacts
		journal RenewalEvidence
		state   string
		reason  string
		warn    bool
		fail    bool
	}{
		{"healthy", cert(60, true), RenewalEvidence{}, domain.EdgeRenewalValid, "", false, false},
		{"renewal window", cert(25, true), RenewalEvidence{}, domain.EdgeRenewalWindow, ReasonCertRenewalWindow, false, false},
		{"expiring soon", cert(14, true), RenewalEvidence{}, domain.EdgeRenewalExpiringSoon, ReasonCertExpiringSoon, true, false},
		{"renewal failed with valid cert", cert(20, true), RenewalEvidence{Observed: true, Failed: true, LastError: "could not get certificate from issuer"}, domain.EdgeRenewalFailed, ReasonRenewalFailed, true, false},
		{"expired", cert(-1, false), RenewalEvidence{}, domain.EdgeRenewalExpired, ReasonCertExpired, false, true},
		{"hostname mismatch", CertificateFacts{Present: true, Valid: false, ValidationError: "x509: certificate is valid for other.example.test", Issuer: "R11", NotAfter: now.Add(50 * 24 * time.Hour), SANs: []string{"other.example.test"}}, RenewalEvidence{}, domain.EdgeRenewalHostnameWrong, ReasonCertInvalid, false, true},
		{"no certificate, journal failing", CertificateFacts{}, RenewalEvidence{Observed: true, Failed: true, LastError: "challenge failed"}, domain.EdgeRenewalFailed, ReasonRenewalFailed, true, false},
		{"no certificate", CertificateFacts{}, RenewalEvidence{}, domain.EdgeRenewalNotObserved, ReasonTLSNotObserved, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := ObserveTLS("app.example.test", tc.cert, tc.journal, now)
			if state.RenewalState != tc.state || state.ReasonCode != tc.reason {
				t.Fatalf("state = %+v", state)
			}
			warn, fail := TLSNeedsAttention(state)
			if warn != tc.warn || fail != tc.fail {
				t.Fatalf("attention warn=%v fail=%v state=%+v", warn, fail, state)
			}
		})
	}
	staging := ObserveTLS("app.example.test", CertificateFacts{Present: true, Valid: true, Issuer: "(STAGING) Wannabe Watercress R11", NotAfter: now.Add(80 * 24 * time.Hour), SANs: []string{"app.example.test"}}, RenewalEvidence{}, now)
	if staging.ACMEEnvironment != domain.ACMEEnvironmentStaging {
		t.Fatalf("staging issuer not recognised: %+v", staging)
	}
}

func TestParseCaddyJournalDetectsFailureAndLaterSuccess(t *testing.T) {
	failing := `{"level":"info","msg":"obtaining certificate","identifier":"app.example.test"}
{"level":"error","logger":"tls.obtain","msg":"could not get certificate from issuer","identifier":"app.example.test","error":"HTTP 403 urn:ietf:params:acme:error:unauthorized","token":"canary-secret-9f3a"}
`
	ev := ParseCaddyJournal(failing)
	if !ev.Observed || !ev.Failed || !strings.Contains(ev.LastError, "could not get certificate") || !strings.Contains(ev.LastError, "unauthorized") {
		t.Fatalf("evidence = %+v", ev)
	}
	if strings.Contains(ev.LastError, "canary-secret") {
		t.Fatalf("journal evidence copied a token-like field: %+v", ev)
	}
	recovered := failing + `{"level":"info","logger":"tls.obtain","msg":"certificate obtained successfully","identifier":"app.example.test"}` + "\n"
	if ev := ParseCaddyJournal(recovered); ev.Failed || ev.LastSuccess == "" {
		t.Fatalf("recovered evidence = %+v", ev)
	}
	if ev := ParseCaddyJournal(""); ev.Observed || ev.Failed {
		t.Fatalf("empty journal = %+v", ev)
	}
	argv := JournalArgv(0)
	if strings.Join(argv, " ") != "journalctl -u caddy --no-pager -n 200 -o cat" {
		t.Fatalf("argv = %q", argv)
	}
}
