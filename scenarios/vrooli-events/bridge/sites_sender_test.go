package bridge

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
)

type sitesAdmissionFixture struct {
	gate   *wakeGate
	calls  int
	value  string
	fail   bool
	cancel context.CancelFunc
}

func (p *sitesAdmissionFixture) SitesAuthorization(ctx context.Context, destination string) (string, error) {
	p.calls++
	if !p.gate.held || ctx.Err() != nil || destination != "https://flywheel.example.com"+IngressPath {
		return "", errors.New("invalid admission context")
	}
	if p.cancel != nil {
		p.cancel()
	}
	if p.fail {
		return "", errors.New("private credential provider detail")
	}
	if p.value != "" {
		return p.value, nil
	}
	return "Bearer PUBLIC-TEST-ONLY", nil
}

func TestTerminalSubscriptionIsExactSeparateAndDisabled(t *testing.T) {
	legacy, terminal := SubscriptionSpec(), TerminalSubscriptionSpec()
	if legacy.Enabled || terminal.Enabled || legacy.EventPattern != RunEventType || terminal.EventPattern != TerminalEventType || legacy.Name == terminal.Name {
		t.Fatal("subscription defaults or selections changed")
	}
	for _, spec := range []subscription.Subscription{legacy, terminal} {
		if ValidSubscription(spec) || !Reserved(spec) {
			t.Fatal("inert spec not quarantined")
		}
		spec.Enabled, spec.ID = true, 7
		if !ValidSubscription(spec) {
			t.Fatal("exact opt-in not valid")
		}
		for _, field := range []string{"name", "owner", "source", "event", "target", "type", "enabled", "id"} {
			bad := spec
			switch field {
			case "name":
				bad.Name = "other"
			case "owner":
				bad.OwnerScenario = "other"
			case "source":
				bad.SourceFilter = "*"
			case "event":
				bad.EventPattern = "agent_manager.run.*"
			case "target":
				bad.DeliveryTarget = "https://attacker.example"
			case "type":
				bad.DeliveryType = "other"
			case "enabled":
				bad.Enabled = false
			case "id":
				bad.ID = 0
			}
			if ValidSubscription(bad) {
				t.Fatalf("accepted mutation %s", field)
			}
		}
		spec.DeliveryTarget = "https://attacker.example"
		if !Reserved(spec) {
			t.Fatal("reserved name escaped generic quarantine")
		}
	}
	terminal.Enabled, terminal.ID, terminal.EventPattern = true, 7, RunEventType
	if ValidSubscription(terminal) {
		t.Fatal("terminal name inherited legacy selection")
	}
	legacy.Enabled, legacy.ID, legacy.EventPattern = true, 7, TerminalEventType
	if ValidSubscription(legacy) {
		t.Fatal("legacy name inherited terminal selection")
	}
}

func TestSitesAdmissionAndHMACRemainSeparateAcrossTerminalRetry(t *testing.T) {
	f, g, signer, _ := wakeFixture(t)
	admission := f.sitesAdmission.(*sitesAdmissionFixture)
	e := terminalFixture(t)
	body, selected, err := f.Project(context.Background(), e)
	if err != nil || !selected {
		t.Fatal("terminal projection failed", err)
	}
	sub := TerminalSubscriptionSpec()
	sub.Enabled, sub.ID = true, 7
	q := subscription.QueuedDelivery{SubscriptionID: sub.ID, EventID: e.EventID, PayloadJSON: body}
	var firstID string
	calls := 0
	f.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if !g.held || r.URL.String() != "https://flywheel.example.com"+IngressPath || r.Header.Get(SitesAdmissionHeader) != admission.value || r.Header.Get("Authorization") != "" || r.Header.Get("X-Flywheel-Bridge-Signature") == "" {
			t.Fatal("Sites and application boundaries not independently present")
		}
		raw, _ := io.ReadAll(r.Body)
		if string(raw) != body || strings.Contains(string(raw), "PUBLIC-TEST") || strings.Contains(string(signer.signed[len(signer.signed)-1]), "PUBLIC-TEST") {
			t.Fatal("credential entered projection or signing input")
		}
		if calls == 1 {
			firstID = r.Header.Get("Idempotency-Key")
			return nil, errors.New("unknown outcome with private data")
		}
		if r.Header.Get("Idempotency-Key") != firstID || firstID == "" {
			t.Fatal("retry identity changed")
		}
		return &http.Response{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
	})
	admission.value = "Bearer PUBLIC-TEST-FIRST"
	if err := f.Deliver(context.Background(), sub, q); err == nil || strings.Contains(err.Error(), "private") {
		t.Fatal("unknown outcome did not fail safely")
	}
	admission.value = "Bearer PUBLIC-TEST-ROTATED"
	f.now = func() time.Time { return fixtureNow.Add(time.Second) }
	if err := f.Deliver(context.Background(), sub, q); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || admission.calls != 2 || signer.calls != 2 || string(signer.signed[0]) == string(signer.signed[1]) {
		t.Fatal("credentials/signature not refreshed per retry")
	}
}

func TestSitesAdmissionCannotBypassHMACOrPolicy(t *testing.T) {
	for _, name := range []string{"missing", "error", "empty-token", "raw-token", "wrong-scheme", "newline", "space", "oversize", "cancelled", "signer", "disabled", "wrong-subscription", "replaced-scope"} {
		t.Run(name, func(t *testing.T) {
			f, g, signer, e := wakeFixture(t)
			sub, q := projected(t, f, e)
			admission := f.sitesAdmission.(*sitesAdmissionFixture)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch name {
			case "missing":
				f.sitesAdmission = nil
			case "error":
				admission.fail = true
			case "empty-token":
				admission.value = "Bearer "
			case "raw-token":
				admission.value = "PUBLIC-TEST-ONLY"
			case "wrong-scheme":
				admission.value = "Basic PUBLIC-TEST-ONLY"
			case "newline":
				admission.value = "Bearer PUBLIC-TEST\r\nX-Evil: yes"
			case "space":
				admission.value = "Bearer PUBLIC TEST"
			case "oversize":
				admission.value = "Bearer " + strings.Repeat("x", 4096)
			case "cancelled":
				admission.cancel = cancel
			case "signer":
				signer.fail = true
			case "disabled":
				g.policy.Enabled = false
			case "wrong-subscription":
				sub = TerminalSubscriptionSpec()
				sub.Enabled, sub.ID = true, 7
			case "replaced-scope":
				g.policy.Scope.Epoch = "changed"
			}
			f.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("forbidden network operation"); return nil, nil })
			if err := f.Deliver(ctx, sub, q); err == nil || strings.Contains(err.Error(), "private") {
				t.Fatal("unsafe admission outcome", err)
			}
			if name != "signer" && signer.calls != 0 {
				t.Fatal("signed an inadmissible packet")
			}
			if (name == "disabled" || name == "wrong-subscription" || name == "replaced-scope") && admission.calls != 0 {
				t.Fatal("requested credential before policy/selector validation")
			}
		})
	}
}

func TestForwarderConstructionNeverAcquiresSitesAccess(t *testing.T) {
	_, g, signer, e := wakeFixture(t)
	admission := &sitesAdmissionFixture{gate: g}
	f, err := NewSitesForwarder("https://flywheel.example.com", signer, g, admission)
	if err != nil {
		t.Fatal(err)
	}
	f.now = func() time.Time { return fixtureNow }
	_, _ = projected(t, f, e)
	if admission.calls != 0 || signer.calls != 0 {
		t.Fatal("construction/projection acquired credentials")
	}
	if _, err := NewSitesForwarder("https://flywheel.example.com", signer, g, nil); err == nil {
		t.Fatal("missing provider accepted")
	}
	projectionOnly, err := NewForwarder("https://flywheel.example.com", signer, g)
	if err != nil {
		t.Fatal(err)
	}
	projectionOnly.now = f.now
	sub, q := projected(t, projectionOnly, e)
	projectionOnly.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("HMAC-only fallback"); return nil, nil })
	if err := projectionOnly.Deliver(context.Background(), sub, q); err == nil {
		t.Fatal("uncomposed sender activated")
	}
	if signer.calls != 0 {
		t.Fatal("uncomposed sender signed")
	}
}
