package localworkbridge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type brokerFence struct {
	calls   int
	failAt  int
	seen    SourceBrokerScope
	onCheck func()
}

func (f *brokerFence) Check(_ context.Context, scope SourceBrokerScope) error {
	f.calls++
	f.seen = scope
	if f.onCheck != nil {
		f.onCheck()
	}
	if f.failAt == f.calls {
		return errors.New("PRIVATE_FENCE")
	}
	return nil
}

type brokerCredentials struct {
	calls   int
	purpose string
	buf     []byte
	err     error
}

func (c *brokerCredentials) Credential(_ context.Context, p string) ([]byte, error) {
	c.calls++
	c.purpose = p
	c.buf = []byte("Bearer fixture-not-a-credential")
	return c.buf, c.err
}

type brokerOwners struct {
	calls     int
	operation string
	call      SourceCall
	event     json.RawMessage
	program   SourceBrokerProgram
	err       error
	run       func(context.Context, *SourcePermit)
}

func (o *brokerOwners) called(ctx context.Context, p *SourcePermit, op string) {
	o.calls++
	o.operation = op
	o.call = p.Call()
	if o.run != nil {
		o.run(ctx, p)
	}
}
func (o *brokerOwners) ReadEvent(ctx context.Context, p *SourcePermit) (json.RawMessage, error) {
	o.called(ctx, p, "read_event")
	return o.event, o.err
}
func (o *brokerOwners) AdmitCollector(ctx context.Context, p *SourcePermit) (SourceBrokerProgram, error) {
	o.called(ctx, p, "admit_collector")
	return o.program, o.err
}
func (o *brokerOwners) ObserveCollector(ctx context.Context, p *SourcePermit) (SourceBrokerProgram, error) {
	o.called(ctx, p, "observe_collector")
	return o.program, o.err
}
func (o *brokerOwners) WaitCollector(ctx context.Context, p *SourcePermit) (SourceBrokerProgram, error) {
	o.called(ctx, p, "wait_collector")
	return o.program, o.err
}

func brokerFixture(t *testing.T) (*SourceBroker, *brokerFence, *brokerCredentials, *brokerOwners, *sourceBindingStub, *sourceBindingStub) {
	t.Helper()
	events, _, eb := sourceFixture(t, SourceEvents)
	collector, _, cb := sourceFixture(t, SourceCollector)
	collector.policy.BindingID = "collector-binding"
	raw, _ := json.Marshal(collector.policy)
	collector.digest = fmt.Sprintf("%x", sha256.Sum256(raw))
	cb.b.ID = collector.policy.BindingID
	cb.b.ScopeSHA256 = collector.digest
	cb.b.Grant.Targets = []string{"program-runtime:local-work/" + collector.digest + "/*"}
	f, c, o := &brokerFence{}, &brokerCredentials{}, &brokerOwners{event: json.RawMessage(`{"eventId":"event-1","eventType":"agent_manager.run.notification.v1","source":{"scenario":"agent-manager"},"data":{"message":"selected event"}}`)}
	o.program = SourceBrokerProgram{Found: true, Status: "PROGRAM_STATUS_SUCCEEDED", Stdout: `{"snapshot":true}`, Identity: SourceProgramIdentity{ID: "prog_123", Name: SourceCollectorName, Digest: collector.policy.CollectorDigest, RequestKey: "local-work-" + strings.Repeat("b", 64), Subject: collector.policy.Subject, Owner: collector.policy.Owner, BindingID: collector.policy.BindingID, BindingVersion: collector.policy.BindingVersion, ScopeSHA256: collector.digest}}
	b, err := NewSourceBroker(SourceBrokerConfig{Enabled: true, LaunchGeneration: "launch-1", Selection: SourceBrokerSelection{RequestKey: "local-work-" + strings.Repeat("b", 64), AdmissionDeadline: "2026-11-07T12:05:00Z"}, Events: events, Collector: collector, Credentials: c, Fence: f, Owners: o})
	if err != nil {
		t.Fatal(err)
	}
	return b, f, c, o, eb, cb
}
func brokerRequest(op string) []byte {
	v := map[string]any{"schema": SourceBrokerSchema, "operation": op}
	switch op {
	case "read_event":
		v["event_id"] = "event-1"
	case "observe_collector":

	case "admit_collector":

	case "wait_collector":
		v["program_id"] = "prog_123"
		v["timeout_ms"] = 90000
	}
	raw, _ := json.Marshal(v)
	return raw
}
func TestSourceBrokerDisabledNoIO(t *testing.T) {
	_, f, c, o, _, _ := brokerFixture(t)
	b, err := NewSourceBroker(SourceBrokerConfig{Fence: f, Credentials: c, Owners: o})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range []*SourceBroker{nil, b} {
		raw, err := b.Execute(nil, []byte("invalid"))
		if !errors.Is(err, ErrDisabled) || raw != nil {
			t.Fatalf("%s %v", raw, err)
		}
	}
	if f.calls+c.calls+o.calls != 0 {
		t.Fatal("disabled core touched dependency")
	}
}
func TestSourceBrokerRequiresMatchingOwnerScope(t *testing.T) {
	for _, field := range []string{"subject", "owner", "epoch", "instance", "same binding", "generation", "events disabled", "collector missing", "fence missing", "credentials missing", "owners missing"} {
		t.Run(field, func(t *testing.T) {
			b, _, _, _, _, _ := brokerFixture(t)
			c := SourceBrokerConfig{Enabled: true, LaunchGeneration: "launch-1", Selection: SourceBrokerSelection{RequestKey: "local-work-" + strings.Repeat("b", 64), AdmissionDeadline: "2026-11-07T12:05:00Z"}, Events: b.events, Collector: b.collector, Fence: b.fence, Credentials: b.credentials, Owners: b.owners}
			switch field {
			case "subject":
				c.Collector.policy.Subject = "other"
			case "owner":
				c.Collector.policy.Owner = "other"
			case "epoch":
				c.Collector.policy.SourceEpoch = "other"
			case "instance":
				c.Collector.policy.SourceInstance = "other"
			case "same binding":
				c.Collector.policy.BindingID = c.Events.policy.BindingID
			case "generation":
				c.LaunchGeneration = ""
			case "events disabled":
				c.Events.enabled = false
			case "collector missing":
				c.Collector = nil
			case "fence missing":
				c.Fence = nil
			case "credentials missing":
				c.Credentials = nil
			case "owners missing":
				c.Owners = nil
			}
			if _, err := NewSourceBroker(c); err == nil {
				t.Fatal("invalid wiring accepted")
			}
		})
	}
}
func TestSourceBrokerFiniteOperationsAndCredentialIsolation(t *testing.T) {
	for _, op := range []string{"read_event", "observe_collector", "admit_collector", "wait_collector"} {
		t.Run(op, func(t *testing.T) {
			b, f, c, o, _, _ := brokerFixture(t)
			result, err := b.Execute(context.Background(), brokerRequest(op))
			if err != nil {
				t.Fatal(err)
			}
			if o.calls != 1 || o.operation != op || c.calls != 1 || f.calls != 4 {
				t.Fatalf("unexpected calls: %d %s %d %d", o.calls, o.operation, c.calls, f.calls)
			}
			want := "runtime_collector"
			if op == "read_event" {
				want = "events_reader"
			}
			if c.purpose != want {
				t.Fatal(c.purpose)
			}
			if !bytes.Equal(c.buf, make([]byte, len(c.buf))) || bytes.Contains(result, []byte("Bearer")) {
				t.Fatal("credential retained or returned")
			}
			if f.seen != b.scope || f.seen.LaunchGeneration != "launch-1" || f.seen.EventsScopeSHA256 != b.events.digest || f.seen.CollectorScopeSHA256 != b.collector.digest {
				t.Fatal("fence did not see exact scope")
			}
			if op == "admit_collector" && (string(o.call.Inputs) != string(b.collector.policy.CollectorInputs) || o.call.RequestKey != "local-work-"+strings.Repeat("b", 64)) {
				t.Fatal("admission not pinned")
			}
			var v any
			if json.Unmarshal(result, &v) != nil {
				t.Fatal("invalid response")
			}
		})
	}
}
func TestSourceBrokerRejectsRequestSmugglingBeforeIO(t *testing.T) {
	base := string(brokerRequest("read_event"))
	cases := []string{"", `null`, `[]`, base + `{}`, strings.Repeat(" ", 4097), strings.Replace(base, `"read_event"`, `"read_event","operation":"admit_collector"`, 1), strings.Replace(base, `"event_id"`, `"eventId"`, 1), strings.Replace(base, `"event-1"`, `"event-3"`, 1), strings.Replace(base, `"event-1"`, `"event-1&limit=100"`, 1), strings.Replace(base, `"read_event"`, `"http_request"`, 1), strings.Replace(base, SourceBrokerSchema, "v2", 1)}
	for _, extra := range []string{`"url":"http://127.0.0.1:1/"`, `"headers":{"Authorization":"Bearer unsafe"}`, `"token":"secret"`, `"scope":"admin"`, `"inputs":{}`, `"policy":{}`, `"source":"code"`, `"event_ids":["event-1"]`, `"schema":{}`} {
		cases = append(cases, strings.TrimSuffix(base, "}")+","+extra+"}")
	}
	for _, raw := range cases {
		t.Run(fmt.Sprintf("case-%d", len(raw))+raw[:min(len(raw), 40)], func(t *testing.T) {
			b, f, c, o, eb, cb := brokerFixture(t)
			result, err := b.Execute(context.Background(), []byte(raw))
			if err == nil || result != nil {
				t.Fatal("invalid request admitted")
			}
			if f.calls+c.calls+o.calls+eb.calls+cb.calls != 0 {
				t.Fatal("invalid request touched dependency")
			}
		})
	}
	for _, raw := range []string{strings.Replace(string(brokerRequest("wait_collector")), `90000`, `90001`, 1), strings.Replace(string(brokerRequest("wait_collector")), `90000`, `1.5`, 1), `{"schema":"flywheel.local-work-source-broker.v1","operation":"admit_collector","request_key":"local-work-` + strings.Repeat("c", 64) + `"}`} {
		b, _, c, o, _, _ := brokerFixture(t)
		if _, err := b.Execute(context.Background(), []byte(raw)); err == nil || c.calls+o.calls != 0 {
			t.Fatal("collector request reached credential or owner")
		}
	}
}
func TestSourceBrokerFenceRevocationStages(t *testing.T) {
	for _, stage := range []int{1, 2, 3, 4} {
		t.Run(fmt.Sprint(stage), func(t *testing.T) {
			b, f, c, o, _, _ := brokerFixture(t)
			f.failAt = stage
			result, err := b.Execute(context.Background(), brokerRequest("read_event"))
			if err == nil || result != nil || err.Error() != "local work source authority unavailable" {
				t.Fatal("fence bypass or error disclosure")
			}
			if stage == 1 && c.calls != 0 || stage < 3 && o.calls != 0 {
				t.Fatal("effect after fence failed")
			}
		})
	}
}
func TestSourceBrokerAuthorityFailuresDoNotRelease(t *testing.T) {
	for _, scenario := range []string{"revoked before", "revoked owner", "version owner", "expiry owner", "lookup error", "credential error", "owner error"} {
		t.Run(scenario, func(t *testing.T) {
			b, _, c, o, eb, _ := brokerFixture(t)
			switch scenario {
			case "revoked before":
				eb.b.Revoked = true
			case "revoked owner":
				o.run = func(context.Context, *SourcePermit) { eb.b.Revoked = true }
			case "version owner":
				o.run = func(context.Context, *SourcePermit) { eb.b.Version = "v2" }
			case "expiry owner":
				o.run = func(context.Context, *SourcePermit) { eb.b.ExpiresAt = b.events.clock() }
			case "lookup error":
				eb.err = errors.New("PRIVATE_BINDING")
			case "credential error":
				c.err = errors.New("PRIVATE_CREDENTIAL")
			case "owner error":
				o.err = errors.New("PRIVATE_OWNER")
			}
			result, err := b.Execute(context.Background(), brokerRequest("read_event"))
			if err != ErrSourceAuthority || result != nil {
				t.Fatalf("released or leaked error: %s %v", result, err)
			}
		})
	}
}
func TestSourceBrokerValidatesEventBytes(t *testing.T) {
	b, _, _, o, _, _ := brokerFixture(t)
	base := string(o.event)
	for _, raw := range []string{`null`, `[]`, base + `{}`, strings.Replace(base, `"event-1"`, `"event-2"`, 1), strings.Replace(base, `"agent-manager"`, `"other"`, 1), strings.Replace(base, `agent_manager.run.notification.v1`, `event.unselected`, 1), strings.Replace(base, `"eventId":"event-1"`, `"eventId":"other","eventId":"event-1"`, 1), strings.Replace(base, `"message":"selected event"`, `"message":"secret","message":"selected event"`, 1), strings.Repeat("x", sourceBrokerResponseLimit+1)} {
		b, _, _, o, _, _ = brokerFixture(t)
		o.event = []byte(raw)
		result, err := b.Execute(context.Background(), brokerRequest("read_event"))
		if err == nil || result != nil {
			t.Fatal("unvalidated event released")
		}
	}
	b, _, _, o, _, _ = brokerFixture(t)
	o.event = nil
	raw, err := b.Execute(context.Background(), brokerRequest("read_event"))
	if err != nil || string(raw) != "[]" {
		t.Fatal("missing event not explicit", err)
	}
}
func TestSourceBrokerRejectsProgramOwnershipAndOutput(t *testing.T) {
	for _, field := range []string{"id", "name", "digest", "key", "subject", "owner", "binding", "version", "scope", "status", "oversize", "partial output", "not found"} {
		t.Run(field, func(t *testing.T) {
			b, _, _, o, _, _ := brokerFixture(t)
			switch field {
			case "id":
				o.program.Identity.ID = "victim"
			case "name":
				o.program.Identity.Name = "other"
			case "digest":
				o.program.Identity.Digest = strings.Repeat("c", 64)
			case "key":
				o.program.Identity.RequestKey = "local-work-" + strings.Repeat("c", 64)
			case "subject":
				o.program.Identity.Subject = "victim"
			case "owner":
				o.program.Identity.Owner = "other"
			case "binding":
				o.program.Identity.BindingID = "other"
			case "version":
				o.program.Identity.BindingVersion = "v2"
			case "scope":
				o.program.Identity.ScopeSHA256 = strings.Repeat("c", 64)
			case "status":
				o.program.Status = "NEW"
			case "oversize":
				o.program.Stdout = strings.Repeat("x", 65537)
			case "partial output":
				o.program.Status = "PROGRAM_STATUS_RUNNING"
			case "not found":
				o.program.Found = false
			}
			result, err := b.Execute(context.Background(), brokerRequest("wait_collector"))
			if err == nil || result != nil {
				t.Fatal("unqualified program released")
			}
		})
	}
}
func TestSourceBrokerObservationAbsenceAndFiniteProjection(t *testing.T) {
	b, _, _, o, _, _ := brokerFixture(t)
	o.program = SourceBrokerProgram{}
	raw, err := b.Execute(context.Background(), brokerRequest("observe_collector"))
	if err != nil || string(raw) != `{"admissionContractVersion":2,"found":false}` {
		t.Fatalf("%s %v", raw, err)
	}
	o.program.Status = "PROGRAM_STATUS_RUNNING"
	if _, err = b.Execute(context.Background(), brokerRequest("observe_collector")); err == nil {
		t.Fatal("false absence with result data accepted")
	}
	b, _, _, o, _, _ = brokerFixture(t)
	o.program.Status = "PROGRAM_STATUS_RUNNING"
	o.program.Stdout = ""
	raw, err = b.Execute(context.Background(), brokerRequest("wait_collector"))
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	json.Unmarshal(raw, &v)
	if v["terminal"] != false {
		t.Fatal("running called terminal")
	}
	p := v["program"].(map[string]any)
	if len(p) != 4 {
		t.Fatal("extra fields released")
	}
}
func TestSourceBrokerCancellationJoinsAndSerializes(t *testing.T) {
	b, _, c, o, _, _ := brokerFixture(t)
	entered, done := make(chan struct{}), make(chan error, 1)
	o.run = func(ctx context.Context, _ *SourcePermit) { close(entered); <-ctx.Done() }
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _, err := b.Execute(ctx, brokerRequest("read_event")); done <- err }()
	<-entered
	if result, err := b.Execute(context.Background(), brokerRequest("read_event")); err == nil || result != nil {
		t.Fatal("concurrent operation admitted")
	}
	cancel()
	select {
	case err := <-done:
		if err != ErrSourceAuthority {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("did not join cooperative owner")
	}
	if c.calls != 1 || o.calls != 1 {
		t.Fatal("second call touched dependencies")
	}
	o.run = nil
	if _, err := b.Execute(context.Background(), brokerRequest("read_event")); err != nil {
		t.Fatal("joined execution retained busy lock", err)
	}
}
func TestSourceBrokerContextBudgetAndResultCopy(t *testing.T) {
	b, _, _, o, _, _ := brokerFixture(t)
	o.run = func(ctx context.Context, _ *SourcePermit) {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 100*time.Second {
			t.Error("unbounded owner")
		}
	}
	raw, err := b.Execute(context.Background(), brokerRequest("read_event"))
	if err != nil {
		t.Fatal(err)
	}
	old := bytes.Clone(raw)
	clear(o.event)
	if !bytes.Equal(old, raw) {
		t.Fatal("result aliases owner bytes")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	before := o.calls
	if _, err = b.Execute(ctx, brokerRequest("read_event")); err == nil || o.calls != before {
		t.Fatal("canceled request called owner")
	}
}
func FuzzSourceBrokerRequestParser(f *testing.F) {
	f.Add(brokerRequest("read_event"))
	f.Add(brokerRequest("admit_collector"))
	f.Add([]byte(`{"schema":null}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		b, _, _, _, _, _ := brokerFixture(t)
		if len(raw) > 4096 {
			return
		}
		_, _, _, _ = b.request(raw)
	})
}

func TestSourceBrokerOwnerFinalRecheckIncludesSessionFence(t *testing.T) {
	b, f, _, o, _, _ := brokerFixture(t)
	committed := false
	o.run = func(ctx context.Context, p *SourcePermit) {
		// The owner has not committed yet. Its serialized fence now sees the
		// launch as invalid even while the independent source grant stays valid.
		f.failAt = f.calls + 1
		if err := p.Recheck(ctx); err != nil {
			o.err = err
			return
		}
		committed = true
	}
	raw, err := b.Execute(context.Background(), brokerRequest("admit_collector"))
	if err != ErrSourceAuthority || raw != nil || committed {
		t.Fatal("owner effect escaped session revocation")
	}
}
func TestSourceBrokerWaitHonorsRequestedBudget(t *testing.T) {
	b, _, _, o, _, _ := brokerFixture(t)
	o.run = func(ctx context.Context, _ *SourcePermit) {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 10*time.Millisecond {
			t.Error("wait budget exceeded")
		}
		<-ctx.Done()
	}
	request := bytes.Replace(brokerRequest("wait_collector"), []byte("90000"), []byte("10"), 1)
	raw, err := b.Execute(context.Background(), request)
	if err != ErrSourceAuthority || raw != nil {
		t.Fatal("timed out wait released a result")
	}
}

func TestSourceBrokerFenceCannotOutliveAuthority(t *testing.T) {
	b, f, _, _, eb, _ := brokerFixture(t)
	now := b.events.clock()
	eb.b.Grant.ExpiresAt = now.Add(time.Second)
	f.onCheck = func() {
		if f.calls == 4 {
			b.events.clock = func() time.Time { return now.Add(2 * time.Second) }
		}
	}
	raw, err := b.Execute(context.Background(), brokerRequest("read_event"))
	if err != ErrSourceAuthority || raw != nil {
		t.Fatal("blocking final fence crossed grant expiry")
	}
}
func TestSourceBrokerWorkerCannotChangeDurableSelection(t *testing.T) {
	for _, op := range []string{"observe_collector", "admit_collector"} {
		for _, extra := range []string{`"request_key":"local-work-` + strings.Repeat("c", 64) + `"`, `"admission_deadline":"2026-11-07T12:09:00Z"`, `"inputs":{}`, `"event_ids":["event-2"]`} {
			b, _, c, o, _, _ := brokerFixture(t)
			raw := append(bytes.TrimSuffix(brokerRequest(op), []byte("}")), []byte(","+extra+"}")...)
			if _, err := b.Execute(context.Background(), raw); err != ErrSourceAuthority || c.calls+o.calls != 0 {
				t.Fatal("worker changed durable selection")
			}
		}
	}
}
