// serial_episode_test.go uses the canonical disposable SQL store; no native unit effects.
package effortauthority

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

func serialAuthorityFixture(t *testing.T) (*Authority, Policy, Binding, Intent, *time.Time) {
	t.Helper()
	a, _, _, p, _, now := authorityFixture(t)
	p.ID = "serial-fixture"
	p.MaxConcurrent = 1
	p.Profiles = map[string]string{"leader-profile": Digest("leader complete profile"), "worker-profile": Digest("worker complete profile")}
	p.SerialEdges = []SerialEdge{{"leader", "leader-profile", "worker", "worker-profile"}, {"worker", "worker-profile", "leader", "leader-profile"}}
	if err := a.Approve(context.Background(), "fixture-human", p); err != nil {
		t.Fatal(err)
	}
	root := nativeIntent(p, "parent-episode")
	root.Profile = "leader-profile"
	root.ProfileDigest = p.Profiles[root.Profile]
	return a, p, binding(p), root, now
}
func serialNext(p Policy, source Intent, run, key, member, profile string) Intent {
	next := nativeIntent(p, key)
	next.Effect = "run.child"
	next.ParentRunID = run
	next.Member = member
	next.Profile = profile
	next.ProfileDigest = p.Profiles[profile]
	next.InputDigest = Digest(struct{ Previous, Key string }{source.IdempotencyKey, key})
	return next
}
func startSerialRoot(t *testing.T, a *Authority, b Binding, root Intent) {
	t.Helper()
	if _, _, err := a.Reserve(context.Background(), b, root, nil); err != nil {
		t.Fatal(err)
	}
	if err := a.Accept(context.Background(), b, root.IdempotencyKey, "native-parent"); err != nil {
		t.Fatal(err)
	}
}
func TestSerialHandoffDurableParentWorkerParentConservesCeilings(t *testing.T) {
	ctx := context.Background()
	a, p, b, root, _ := serialAuthorityFixture(t)
	startSerialRoot(t, a, b, root)
	next := serialNext(p, root, "native-parent", "worker-episode", "worker", "worker-profile")
	h, replay, err := a.PrepareSerialHandoff(ctx, b, root.IdempotencyKey, next)
	if err != nil || replay || h.Next.ParentRunID != "native-parent" {
		t.Fatal("prepare", err)
	}
	rec, _ := a.Store.Get(ctx, p.ID)
	if len(rec.Reservations) != 1 || rec.Reservations[root.IdempotencyKey].Terminal {
		t.Fatal("handoff released capacity")
	}
	before := Digest(rec)
	if _, _, err := a.ReserveSerialEpisode(ctx, b, next.IdempotencyKey); err == nil {
		t.Fatal("active parent allowed worker")
	}
	unchangedRecord(t, a, p.ID, before)
	if err := a.Settle(ctx, b, root.IdempotencyKey, "native-parent"); err != nil {
		t.Fatal(err)
	}
	restarted := *a
	slot, replay, err := restarted.ReserveSerialEpisode(ctx, b, next.IdempotencyKey)
	if err != nil || replay || slot.Terminal || slot.Intent.Profile != "worker-profile" {
		t.Fatal("worker", err)
	}
	if err := a.Accept(ctx, b, next.IdempotencyKey, "native-worker"); err != nil {
		t.Fatal(err)
	}
	parent := serialNext(p, next, "native-worker", "parent-second-episode", "leader", "leader-profile")
	if _, _, err := a.PrepareSerialHandoff(ctx, b, next.IdempotencyKey, parent); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.ReserveSerialEpisode(ctx, b, parent.IdempotencyKey); err == nil {
		t.Fatal("worker still live")
	}
	if err := a.Settle(ctx, b, next.IdempotencyKey, "native-worker"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.ReserveSerialEpisode(ctx, b, parent.IdempotencyKey); err != nil {
		t.Fatal(err)
	}
	rec, _ = a.Store.Get(ctx, p.ID)
	if len(rec.Reservations) != 3 || len(rec.SerialHandoffs) != 2 || !rec.Reservations[root.IdempotencyKey].Terminal || !rec.Reservations[next.IdempotencyKey].Terminal {
		t.Fatal("episode accounting")
	}
	var turns, tools, seconds int64
	active := 0
	for _, v := range rec.Reservations {
		turns += int64(v.Intent.Turns)
		tools += int64(v.Intent.ToolCalls)
		seconds += v.Intent.RunSeconds
		if !v.Terminal {
			active++
		}
	}
	if active != 1 || turns != 3*int64(p.MaxTurns) || tools != 3*int64(p.MaxToolCalls) || seconds != 3*p.MaxRunSeconds {
		t.Fatal("budget refunded or concurrency widened")
	}
}
func TestSerialHandoffExactReplayAndRefusalsHaveNoWrites(t *testing.T) {
	for _, name := range []string{"unaccepted", "terminal-before-prepare", "parent", "member", "profile", "digest", "budget", "same-key", "changed-input", "second-handoff", "direct-child", "missing-handoff", "expired", "revoked", "canceled"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			a, p, b, root, now := serialAuthorityFixture(t)
			startSerialRoot(t, a, b, root)
			next := serialNext(p, root, "native-parent", "worker-episode", "worker", "worker-profile")
			if name == "changed-input" || name == "second-handoff" {
				if _, _, err := a.PrepareSerialHandoff(ctx, b, root.IdempotencyKey, next); err != nil {
					t.Fatal(err)
				}
			}
			switch name {
			case "unaccepted":
				a.Store.Update(ctx, p.ID, func(r *Record) error {
					v := r.Reservations[root.IdempotencyKey]
					v.RunID = ""
					r.Reservations[root.IdempotencyKey] = v
					return nil
				})
			case "terminal-before-prepare":
				a.Settle(ctx, b, root.IdempotencyKey, "native-parent")
			case "parent":
				next.ParentRunID = "other"
			case "member":
				next.Member = "leader"
			case "profile":
				next.Profile = "leader-profile"
				next.ProfileDigest = p.Profiles[next.Profile]
			case "digest":
				next.ProfileDigest = Digest("other")
			case "budget":
				next.Turns--
			case "same-key":
				next.IdempotencyKey = root.IdempotencyKey
			case "changed-input":
				next.InputDigest = Digest("changed")
			case "second-handoff":
				next.IdempotencyKey = "second-worker"
			case "expired":
				*now = p.Deadline
			case "revoked":
				if err := a.Revoke(ctx, "fixture-human", p.ID); err != nil {
					t.Fatal(err)
				}
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			rec, _ := a.Store.Get(context.Background(), p.ID)
			before := Digest(rec)
			var err error
			if name == "direct-child" {
				_, _, err = a.Reserve(ctx, b, next, nil)
			} else if name == "missing-handoff" {
				_, _, err = a.ReserveSerialEpisode(ctx, b, next.IdempotencyKey)
			} else {
				_, _, err = a.PrepareSerialHandoff(ctx, b, root.IdempotencyKey, next)
			}
			if err == nil {
				t.Fatal("refusal missing")
			}
			unchangedRecord(t, a, p.ID, before)
		})
	}
	a, p, b, root, _ := serialAuthorityFixture(t)
	startSerialRoot(t, a, b, root)
	ctx := context.Background()
	next := serialNext(p, root, "native-parent", "worker", "worker", "worker-profile")
	original, _, err := a.PrepareSerialHandoff(ctx, b, root.IdempotencyKey, next)
	if err != nil {
		t.Fatal(err)
	}
	rec, _ := a.Store.Get(ctx, p.ID)
	before := Digest(rec)
	got, replay, err := a.PrepareSerialHandoff(ctx, b, root.IdempotencyKey, next)
	if err != nil || !replay || Digest(got) != Digest(original) {
		t.Fatal("handoff replay", err)
	}
	unchangedRecord(t, a, p.ID, before)
	a.Settle(ctx, b, root.IdempotencyKey, "native-parent")
	first, _, err := a.ReserveSerialEpisode(ctx, b, next.IdempotencyKey)
	if err != nil {
		t.Fatal(err)
	}
	rec, _ = a.Store.Get(ctx, p.ID)
	before = Digest(rec)
	gotSlot, replay, err := a.ReserveSerialEpisode(ctx, b, next.IdempotencyKey)
	if err != nil || !replay || Digest(first) != Digest(gotSlot) {
		t.Fatal("reservation replay", err)
	}
	unchangedRecord(t, a, p.ID, before)
}
func TestSerialPolicyEdgesAndLegacyDigestCompatibility(t *testing.T) {
	a, p, _, _, _ := serialAuthorityFixture(t)
	_ = a
	for _, name := range []string{"missing", "extra", "uncovered-member", "same-profile", "unknown", "not-reverse", "duplicate-digest", "concurrency", "window", "starts"} {
		t.Run(name, func(t *testing.T) {
			raw, _ := json.Marshal(p)
			var v Policy
			json.Unmarshal(raw, &v)
			switch name {
			case "missing":
				v.SerialEdges = v.SerialEdges[:1]
			case "extra":
				v.SerialEdges = append(v.SerialEdges, v.SerialEdges[0])
			case "uncovered-member":
				v.SerialEdges[0].ToMember = v.SerialEdges[0].FromMember
			case "same-profile":
				v.SerialEdges[0].ToProfile = v.SerialEdges[0].FromProfile
			case "unknown":
				v.SerialEdges[0].ToMember = "other"
			case "not-reverse":
				v.SerialEdges[1] = v.SerialEdges[0]
			case "duplicate-digest":
				v.Profiles["worker-profile"] = v.Profiles["leader-profile"]
			case "concurrency":
				v.MaxConcurrent = 2
			case "window":
				v.Deadline = v.NotBefore.Add(9 * time.Hour)
			case "starts":
				v.MaxStarts = 9
			}
			if v.Validate() == nil {
				t.Fatal("invalid serial policy accepted")
			}
		})
	}
	_, _, _, legacy, _, _ := authorityFixture(t)
	raw, _ := json.Marshal(legacy)
	if strings.Contains(string(raw), "serialEdges") {
		t.Fatal("nil serial field changed legacy serialization")
	}
	record := Record{Policy: legacy}
	raw, _ = json.Marshal(record)
	if strings.Contains(string(raw), "serialHandoffs") {
		t.Fatal("nil handoff changed legacy serialization")
	}
}
func TestSerialEpisodeStartAndAggregateExhaustionNeverRefunds(t *testing.T) {
	for _, limit := range []string{"starts", "turns", "tools", "seconds"} {
		t.Run(limit, func(t *testing.T) {
			ctx := context.Background()
			a, p, b, root, _ := serialAuthorityFixture(t)
			// Install the exact lower ceiling before the first reservation in this
			// disposable fixture; production approval remains immutable.
			switch limit {
			case "starts":
				p.MaxStarts = 1
			case "turns":
				p.TotalTurns = int64(p.MaxTurns)
			case "tools":
				p.TotalToolCalls = int64(p.MaxToolCalls)
			case "seconds":
				p.TotalRunSeconds = p.MaxRunSeconds
			}
			p.ID += "-lower"
			if err := a.Approve(ctx, "fixture-human", p); err != nil {
				t.Fatal(err)
			}
			b = binding(p)
			root = nativeIntent(p, "root")
			root.Profile = "leader-profile"
			root.ProfileDigest = p.Profiles[root.Profile]
			startSerialRoot(t, a, b, root)
			next := serialNext(p, root, "native-parent", "worker", "worker", "worker-profile")
			if _, _, err := a.PrepareSerialHandoff(ctx, b, root.IdempotencyKey, next); err != nil {
				t.Fatal(err)
			}
			if err := a.Settle(ctx, b, root.IdempotencyKey, "native-parent"); err != nil {
				t.Fatal(err)
			}
			rec, err := a.Store.Get(ctx, p.ID)
			if err != nil || !rec.Reservations[root.IdempotencyKey].Terminal {
				t.Fatal("source terminal prerequisite", err)
			}
			before := Digest(rec)
			if _, _, err := a.ReserveSerialEpisode(ctx, b, next.IdempotencyKey); err == nil {
				t.Fatal("exhausted budget reserved")
			}
			unchangedRecord(t, a, p.ID, before)
		})
	}
}

func TestSerialEpisodeConcurrentReplayAndCurrentGateRefusals(t *testing.T) {
	ctx := context.Background()
	a, p, b, root, now := serialAuthorityFixture(t)
	startSerialRoot(t, a, b, root)
	next := serialNext(p, root, "native-parent", "worker", "worker", "worker-profile")
	if _, _, err := a.PrepareSerialHandoff(ctx, b, root.IdempotencyKey, next); err != nil {
		t.Fatal(err)
	}
	if err := a.Settle(ctx, b, root.IdempotencyKey, "native-parent"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	type outcome struct {
		replay bool
		err    error
	}
	results := make(chan outcome, 2)
	for n := 0; n < 2; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, replay, err := a.ReserveSerialEpisode(ctx, b, next.IdempotencyKey)
			results <- outcome{replay, err}
		}()
	}
	wg.Wait()
	close(results)
	fresh := 0
	for result := range results {
		if result.err == nil {
			if !result.replay {
				fresh++
			}
		} else if result.err != ErrRefused {
			t.Fatal(result.err)
		}
	}
	rec, err := a.Store.Get(ctx, p.ID)
	if err != nil || fresh != 1 || len(rec.Reservations) != 2 || rec.Reservations[next.IdempotencyKey].Terminal {
		t.Fatal("concurrent duplicate reservation", err)
	}
	var turns, tools, seconds int64
	active := 0
	for _, reservation := range rec.Reservations {
		turns += int64(reservation.Intent.Turns)
		tools += int64(reservation.Intent.ToolCalls)
		seconds += reservation.Intent.RunSeconds
		if !reservation.Terminal {
			active++
		}
	}
	if active != 1 || turns != 2*int64(p.MaxTurns) || tools != 2*int64(p.MaxToolCalls) || seconds != 2*p.MaxRunSeconds {
		t.Fatal("racing conservation")
	}
	// The losing SQL CAS is a refusal, never an automatic retry. A later exact
	// owner request must observe the one already-retained reservation.
	beforeReplay := Digest(rec)
	if _, replay, err := a.ReserveSerialEpisode(ctx, b, next.IdempotencyKey); err != nil || !replay {
		t.Fatal("sequential exact replay", err)
	}
	unchangedRecord(t, a, p.ID, beforeReplay)
	// Expiry remains a gate even for exact old episode replay; no deadline reset.
	*now = p.Deadline
	before := Digest(rec)
	if _, _, err := a.ReserveSerialEpisode(ctx, b, next.IdempotencyKey); err == nil {
		t.Fatal("expired replay renewed")
	}
	unchangedRecord(t, a, p.ID, before)
	for _, name := range []string{"revoked", "scope", "canceled"} {
		t.Run(name, func(t *testing.T) {
			a, p, b, root, _ := serialAuthorityFixture(t)
			startSerialRoot(t, a, b, root)
			next := serialNext(p, root, "native-parent", "worker", "worker", "worker-profile")
			if _, _, err := a.PrepareSerialHandoff(ctx, b, root.IdempotencyKey, next); err != nil {
				t.Fatal(err)
			}
			a.Settle(ctx, b, root.IdempotencyKey, "native-parent")
			ctx := context.Background()
			if name == "revoked" {
				a.Revoke(ctx, "fixture-human", p.ID)
			} else if name == "scope" {
				a.Scope = &fixtureScope{invalid: true}
			} else {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			rec, _ := a.Store.Get(context.Background(), p.ID)
			before := Digest(rec)
			if _, _, err := a.ReserveSerialEpisode(ctx, b, next.IdempotencyKey); err == nil {
				t.Fatal("current gate bypass")
			}
			unchangedRecord(t, a, p.ID, before)
		})
	}
}

// The installed Marketing team retains one orchestrator member; profiles are
// distinct exact roles, not permission to invent a second team enrollment.
func TestSerialEpisodeOneRetainedMemberTwoProfiles(t *testing.T) {
	ctx := context.Background()
	a, p, _, _, _ := serialAuthorityFixture(t)
	p.ID += "-one-member"
	p.Members = []string{"leader"}
	p.SerialEdges = []SerialEdge{{"leader", "leader-profile", "leader", "worker-profile"}, {"leader", "worker-profile", "leader", "leader-profile"}}
	if err := a.Approve(ctx, "fixture-human", p); err != nil {
		t.Fatal(err)
	}
	b := binding(p)
	root := nativeIntent(p, "one-member-parent")
	root.Profile = "leader-profile"
	root.ProfileDigest = p.Profiles[root.Profile]
	startSerialRoot(t, a, b, root)
	next := serialNext(p, root, "native-parent", "one-member-worker", "leader", "worker-profile")
	if _, _, err := a.PrepareSerialHandoff(ctx, b, root.IdempotencyKey, next); err != nil {
		t.Fatal(err)
	}
	if err := a.Settle(ctx, b, root.IdempotencyKey, "native-parent"); err != nil {
		t.Fatal(err)
	}
	if _, replay, err := a.ReserveSerialEpisode(ctx, b, next.IdempotencyKey); err != nil || replay {
		t.Fatal(err)
	}
	rec, err := a.Store.Get(ctx, p.ID)
	if err != nil || len(rec.Policy.Members) != 1 || len(rec.Reservations) != 2 {
		t.Fatal("invented member or lost episode", err)
	}
}
