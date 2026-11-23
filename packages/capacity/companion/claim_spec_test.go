package companion_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	internalcapacity "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/packages/capacity/companion"
)

// Feature: a companion-created claim carries the resource's declared policy
//
//	As an operator who declared an idle-unload TTL in resource.json
//	I want every claim the companion creates to carry it
//	So that a resident model is unloaded after the declared dwell instead of
//	staying resident for days.

// repoRoot is the source root as seen from this package directory.
var repoRoot = filepath.Join("..", "..", "..")

func observed(bytes int64) companion.Observer {
	return companion.ObserverFunc(func(context.Context) (companion.Footprint, error) {
		return companion.Footprint{Bytes: bytes}, nil
	})
}

func claimCall(t *testing.T, rec *recorder) string {
	t.Helper()
	for _, call := range rec.calls {
		if strings.HasPrefix(call, "vrooli capacity claim ") {
			return call
		}
	}
	t.Fatalf("calls = %v, want a capacity claim", rec.calls)
	return ""
}

// Scenario: the manifest's idle policy reaches the claim arguments.
func TestDeclaredIdlePolicyReachesTheClaimArguments(t *testing.T) {
	// Given a resource that declares a 15 minute idle grace and unload TTL
	spec := internalcapacity.ResourceClaimSpec{
		ResourceKind: "vram", PreferredBytes: 12 << 30, Priority: "service",
		YieldWhenIdle: true, IdleGraceSeconds: 900, IdleUnloadTTLSeconds: 900,
		Profile: &internalcapacity.DegradeProfile{
			Steps: []internalcapacity.DegradeStep{{Label: "big", AmountBytes: 12 << 30}, {Label: "cpu"}},
			Apply: internalcapacity.DegradeApply{Verb: "capacity", Argv: []string{"degrade", "--to", "{label}"}},
		},
	}
	rec := &recorder{list: noClaims}
	runner, err := companion.New(companion.Config{Resource: "ollama", Observer: observed(7 << 30), Exec: rec.exec}.WithClaimSpec(spec))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// When the companion sees a model and creates the claim
	runner.SyncOnce(context.Background())

	// Then the claim carries the declared idle policy and the declared ladder
	call := claimCall(t, rec)
	for _, want := range []string{"--no-reclaim", "--idle-grace 15m0s", "--idle-unload-ttl 15m0s", "--yield-when-idle", "--priority service", "--resource-kind vram", "--preferred 12884901888", "--floor 0", `"label":"cpu"`} {
		if !strings.Contains(call, want) {
			t.Errorf("claim call missing %q:\n%s", want, call)
		}
	}
}

// Scenario: every resource that runs the companion carries its manifest.
func TestEveryCompanionResourceManifestReachesItsClaim(t *testing.T) {
	for _, resource := range []string{"ollama", "ollaya", "reranker", "pii-detector", "unstructured-io"} {
		t.Run(resource, func(t *testing.T) {
			// Given the resource's real manifest
			spec, ok, err := internalcapacity.LoadResourceClaimSpec(repoRoot, resource)
			if err != nil || !ok {
				t.Fatalf("LoadResourceClaimSpec(%s) = ok %v, err %v", resource, ok, err)
			}
			rec := &recorder{list: noClaims}
			cfg := companion.Config{Resource: resource, Observer: observed(1 << 30), Exec: rec.exec, GPUIndex: 0}.WithClaimSpec(spec)
			runner, err := companion.New(cfg)
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			// When the companion creates a claim
			runner.SyncOnce(context.Background())

			// Then the declared values are on it, and none that are not declared
			call := claimCall(t, rec)
			if got := strings.Contains(call, "--yield-when-idle"); got != spec.YieldWhenIdle {
				t.Errorf("--yield-when-idle present = %v, manifest says %v:\n%s", got, spec.YieldWhenIdle, call)
			}
			wantGrace := time.Duration(spec.IdleGraceSeconds) * time.Second
			if got := strings.Contains(call, "--idle-grace "+wantGrace.String()); got != (wantGrace > 0) {
				t.Errorf("--idle-grace %s present = %v, manifest declares %s:\n%s", wantGrace, got, wantGrace, call)
			}
			wantTTL := time.Duration(spec.IdleUnloadTTLSeconds) * time.Second
			if got := strings.Contains(call, "--idle-unload-ttl "+wantTTL.String()); got != (wantTTL > 0) {
				t.Errorf("--idle-unload-ttl %s present = %v, manifest declares %s:\n%s", wantTTL, got, wantTTL, call)
			}
			if kind := spec.ResourceKind; kind != "" && !strings.Contains(call, "--resource-kind "+kind) {
				t.Errorf("claim is not on the declared %s axis:\n%s", kind, call)
			}
			if spec.Priority != "" && !strings.Contains(call, "--priority "+spec.Priority) {
				t.Errorf("claim is not at the declared %s tier:\n%s", spec.Priority, call)
			}
		})
	}
}

// Scenario: the Ollama manifest's 15 minute unload dwell is on its claim. This
// is the defect: the ledger row had the shared 1 minute grace and no TTL.
func TestOllamaManifestDeclaresFifteenMinuteIdleUnloadOnItsClaim(t *testing.T) {
	spec, _, err := internalcapacity.LoadResourceClaimSpec(repoRoot, "ollama")
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	rec := &recorder{list: noClaims}
	runner, err := companion.New(companion.Config{Resource: "ollama", Observer: observed(7 << 30), Exec: rec.exec}.WithClaimSpec(spec))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	runner.SyncOnce(context.Background())
	call := claimCall(t, rec)
	for _, want := range []string{"--idle-grace 15m0s", "--idle-unload-ttl 15m0s"} {
		if !strings.Contains(call, want) {
			t.Errorf("ollama claim missing %q:\n%s", want, call)
		}
	}
}

// Scenario: a resource that derives its own ladder keeps it as one unit.
func TestResourceDerivedLadderIsNotMixedWithTheManifestLadder(t *testing.T) {
	spec := internalcapacity.ResourceClaimSpec{
		PreferredBytes: 12 << 30, FloorBytes: 0, Priority: "service",
		Profile: &internalcapacity.DegradeProfile{Steps: []internalcapacity.DegradeStep{{Label: "manifest", AmountBytes: 12 << 30}}},
	}
	cfg := companion.Config{PreferredBytes: 6 << 30, FloorBytes: 3 << 30, Profile: `{"steps":[{"label":"own"}]}`}.WithClaimSpec(spec)
	if cfg.PreferredBytes != 6<<30 || cfg.FloorBytes != 3<<30 || !strings.Contains(cfg.Profile, `"own"`) {
		t.Fatalf("config = %+v, want the resource's own ladder preserved", cfg)
	}
	if cfg.Priority != "service" {
		t.Fatalf("priority = %q, want the manifest tier", cfg.Priority)
	}
}

// Scenario: Run applies the declared claim before it claims anything.
func TestRunAppliesTheDeclaredClaim(t *testing.T) {
	rec := &recorder{list: noClaims}
	spec := internalcapacity.ResourceClaimSpec{ResourceKind: "vram", PreferredBytes: 1 << 30, Priority: "service", IdleUnloadTTLSeconds: 900}
	err := companion.Run(companion.CommandOptions{
		Config: companion.Config{Resource: "demo", Observer: observed(1 << 30), Exec: rec.exec},
		Claim: func(resource string) (internalcapacity.ResourceClaimSpec, bool, error) {
			if resource != "demo" {
				t.Errorf("claim source asked for %q, want the companion's own resource", resource)
			}
			return spec, true, nil
		},
		Stderr: &bytes.Buffer{},
	}, []string{"--once", "--parent-pid", "2"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if call := claimCall(t, rec); !strings.Contains(call, "--idle-unload-ttl 15m0s") {
		t.Fatalf("claim = %s, want the declared idle-unload TTL", call)
	}
}

// Scenario: a companion that cannot read its manifest refuses to start rather
// than create claims without the declared policy.
func TestRunRefusesWithoutADeclaredClaim(t *testing.T) {
	for name, source := range map[string]companion.ClaimSource{
		"unreadable": func(string) (internalcapacity.ResourceClaimSpec, bool, error) {
			return internalcapacity.ResourceClaimSpec{}, false, errors.New("disk on fire")
		},
		"undeclared": func(string) (internalcapacity.ResourceClaimSpec, bool, error) {
			return internalcapacity.ResourceClaimSpec{}, false, nil
		},
	} {
		t.Run(name, func(t *testing.T) {
			rec := &recorder{list: noClaims}
			err := companion.Run(companion.CommandOptions{
				Config: companion.Config{Resource: "demo", Observer: observed(1 << 30), Exec: rec.exec},
				Claim:  source,
				Stderr: &bytes.Buffer{},
			}, []string{"--once", "--parent-pid", "2"})
			if err == nil {
				t.Fatal("Run succeeded without a declared claim")
			}
			if len(rec.calls) != 0 {
				t.Fatalf("calls = %v, want no ledger access", rec.calls)
			}
		})
	}
}

// Scenario: the default claim source reads the resource's own resource.json.
func TestDeclaredClaimReadsTheResourceManifest(t *testing.T) {
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	getenv := func(key string) string {
		if key == "RESOURCE_ROOT" {
			return filepath.Join(root, "resources", "ollama")
		}
		return ""
	}
	spec, ok, err := companion.DeclaredClaim(getenv)("ollama")
	if err != nil || !ok {
		t.Fatalf("DeclaredClaim = ok %v, err %v", ok, err)
	}
	if spec.IdleUnloadTTLSeconds != 900 || spec.IdleGraceSeconds != 900 {
		t.Fatalf("spec idle policy = grace %ds/ttl %ds, want 900/900", spec.IdleGraceSeconds, spec.IdleUnloadTTLSeconds)
	}
}

// Feature: a claim whose idle policy predates the manifest converges to it.
//
// No verb edits a claim's idle policy in place, and the live Ollama claim had
// been created before the TTL was carried, so recreating the companion alone
// would never have changed the row.

func driftConfig(t *testing.T, rec *recorder) *companion.Runner {
	t.Helper()
	runner, err := companion.New(companion.Config{
		Resource: "ollama", Observer: observed(7 << 30), Exec: rec.exec,
		PreferredBytes: 12 << 30, Priority: "service", YieldWhenIdle: true,
		IdleGrace: 15 * time.Minute, IdleUnloadTTL: 15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return runner
}

func ledgerClaim(status, activity, grace, ttl string, amount int64) string {
	claim := map[string]any{
		"claim_id": "clm-o", "owner_id": "ollama", "owner_kind": "resource", "resource_kind": "vram",
		"gpu_index": 0, "amount_bytes": amount, "generation": 3, "status": status,
		"activity_state": activity, "yield_when_idle": true, "idle_grace": grace,
	}
	if ttl != "" {
		claim["idle_unload_ttl"] = ttl
	}
	data, _ := json.Marshal(map[string]any{"claims": []any{claim}})
	return string(data)
}

func TestClaimWithoutTheDeclaredUnloadTTLIsReplacedOnce(t *testing.T) {
	// Given a live granted claim with the shared 1m grace and no TTL, in use
	rec := &recorder{list: ledgerClaim("granted", "active", "1m0s", "", 7<<30)}

	// When the companion syncs
	driftConfig(t, rec).SyncOnce(context.Background())

	// Then the old row is released and a new one carries the declared policy
	var release, claim = -1, -1
	for i, call := range rec.calls {
		switch {
		case strings.Contains(call, "capacity release --claim-id clm-o"):
			release = i
		case strings.HasPrefix(call, "vrooli capacity claim "):
			claim = i
		}
	}
	if release < 0 || claim < release {
		t.Fatalf("calls = %v, want release of the stale row then a new claim", rec.calls)
	}
	if call := rec.calls[claim]; !strings.Contains(call, "--idle-unload-ttl 15m0s") || !strings.Contains(call, "--idle-grace 15m0s") {
		t.Fatalf("replacement claim = %s, want the declared idle policy", call)
	}
}

func TestReplacementClaimInheritsTheWorkOwnersActivity(t *testing.T) {
	// Given a stale claim that the work owner reported active
	rec := &recorder{
		list:      ledgerClaim("granted", "active", "1m0s", "", 7<<30),
		claimResp: `{"claim":` + strings.TrimSuffix(strings.TrimPrefix(ledgerClaim("granted", "idle", "15m0s", "15m0s", 7<<30), `{"claims":[`), `]}`) + `}`,
	}

	driftConfig(t, rec).SyncOnce(context.Background())

	// Then the new row is marked active, so it is not unloaded mid-request
	if !rec.sawCall("capacity activity --claim-id clm-o --generation 3 --state active") {
		t.Fatalf("calls = %v, want the replacement to carry the active state", rec.calls)
	}
}

// Scenario: a grant that snaps to a smaller rung is corrected at once. The
// free memory the host reports already excludes what the resource holds, so a
// resident 7 GiB model can be granted a 3 GiB rung; leaving it would invite an
// upshift restart of a resource that is already running.
func TestFreshClaimBelowTheFootprintIsResizedImmediately(t *testing.T) {
	rec := &recorder{
		list:      noClaims,
		claimResp: `{"claim":{"claim_id":"clm-o","owner_id":"ollama","owner_kind":"resource","resource_kind":"vram","gpu_index":0,"amount_bytes":3221225472,"generation":1,"status":"degraded"}}`,
	}
	runner, err := companion.New(companion.Config{Resource: "ollama", Observer: observed(7 << 30), Exec: rec.exec})
	if err != nil {
		t.Fatal(err)
	}

	runner.SyncOnce(context.Background())

	if !rec.sawCall("capacity resize --claim-id clm-o --generation 1 --amount 7516192768") {
		t.Fatalf("calls = %v, want the new claim resized to the 7 GiB footprint in the same pass", rec.calls)
	}
}

func TestClaimMatchingTheDeclarationIsNotReplaced(t *testing.T) {
	rec := &recorder{list: ledgerClaim("granted", "idle", "15m0s", "15m0s", 7<<30)}
	driftConfig(t, rec).SyncOnce(context.Background())
	if rec.sawCall("capacity release") || rec.sawCall("capacity claim") {
		t.Fatalf("calls = %v, a converged claim must only be heartbeated", rec.calls)
	}
	if !rec.sawCall("capacity heartbeat --claim-id clm-o") {
		t.Fatalf("calls = %v, want a heartbeat", rec.calls)
	}
}

func TestDegradedClaimKeepsItsModeInsteadOfBeingReplaced(t *testing.T) {
	rec := &recorder{list: ledgerClaim("degraded", "idle", "1m0s", "", 4<<30)}
	driftConfig(t, rec).SyncOnce(context.Background())
	if rec.sawCall("capacity release") || rec.sawCall("capacity claim") {
		t.Fatalf("calls = %v, a degraded claim records the operating mode and must not be replaced", rec.calls)
	}
}

func TestUndeclaredGraceIsNotTreatedAsDrift(t *testing.T) {
	rec := &recorder{list: ledgerClaim("granted", "idle", "1m0s", "", 7<<30)}
	runner, err := companion.New(companion.Config{Resource: "ollama", Observer: observed(7 << 30), Exec: rec.exec, YieldWhenIdle: true})
	if err != nil {
		t.Fatal(err)
	}
	runner.SyncOnce(context.Background())
	if rec.sawCall("capacity release") {
		t.Fatalf("calls = %v, the list view reports the effective grace, so an undeclared grace cannot drift", rec.calls)
	}
}

// Feature: a lost generation race is re-read, not logged as a failure
//
//	The work owner's edge reports activity, which bumps the claim's
//	generation, between this loop's read and its write. Heartbeat and resize
//	stay guarded by the generation, so the loop reads again and retries once.

// racyLedger is a ledger whose generation another writer bumps after every
// list, the way the activity edge does while Ollama is in use.
type racyLedger struct {
	generation int64
	bumps      int // list calls after which the generation moves
	calls      []string
	// replaceAfterList swaps the claim id after the first list.
	replaceAfterList bool
	id               string
}

func (l *racyLedger) claimJSON() map[string]any {
	return map[string]any{
		"claim_id": l.id, "owner_id": "ollama", "owner_kind": "resource", "resource_kind": "vram",
		"gpu_index": 0, "amount_bytes": int64(4 << 30), "generation": l.generation, "status": "granted",
	}
}

func (l *racyLedger) exec(_ context.Context, _ string, args ...string) ([]byte, error) {
	l.calls = append(l.calls, strings.Join(args, " "))
	flag := func(name string) string {
		for i, a := range args {
			if a == name && i+1 < len(args) {
				return args[i+1]
			}
		}
		return ""
	}
	switch args[1] {
	case "list":
		out, _ := json.Marshal(map[string]any{"claims": []any{l.claimJSON()}})
		if l.bumps > 0 {
			l.bumps--
			l.generation++
			if l.replaceAfterList {
				l.id = "clm-new"
			}
		}
		return out, nil
	case "heartbeat", "resize":
		if flag("--generation") != strconv.FormatInt(l.generation, 10) {
			return nil, errors.New("exit status 1: " + `{"error":"capacity claim generation is stale: claim ` + l.id + ` is granted at generation ` + strconv.FormatInt(l.generation, 10) + `"}`)
		}
		if args[1] == "resize" {
			l.generation++
		}
		out, _ := json.Marshal(l.claimJSON())
		return out, nil
	}
	return []byte(`{}`), nil
}

func (l *racyLedger) count(verb string) int {
	n := 0
	for _, c := range l.calls {
		if strings.HasPrefix(c, "capacity "+verb+" ") {
			n++
		}
	}
	return n
}

func syncAgainst(t *testing.T, l *racyLedger, footprint int64) string {
	t.Helper()
	var log strings.Builder
	runner, err := companion.New(companion.Config{Resource: "ollama", Observer: observed(footprint), Exec: l.exec, Log: &log})
	if err != nil {
		t.Fatal(err)
	}
	runner.SyncOnce(context.Background())
	return log.String()
}

func TestHeartbeatThatLosesAGenerationRaceRetriesOnceAgainstTheCurrentGeneration(t *testing.T) {
	// Given activity moves the generation between the list and the heartbeat
	l := &racyLedger{generation: 3, bumps: 1, id: "clm-o"}

	log := syncAgainst(t, l, 4<<30)

	// Then the heartbeat is retried at the new generation and lands
	if l.count("heartbeat") != 2 || !strings.Contains(l.calls[len(l.calls)-1], "--generation 4") {
		t.Fatalf("calls = %v, want a failed heartbeat at 3 then one at 4", l.calls)
	}
	if !strings.Contains(log, "retrying once") {
		t.Fatalf("log = %q, want the retry reported", log)
	}
}

func TestResizeThatLosesAGenerationRaceRetriesOnceAgainstTheCurrentGeneration(t *testing.T) {
	l := &racyLedger{generation: 3, bumps: 1, id: "clm-o"}

	syncAgainst(t, l, 9<<30)

	if l.count("resize") != 2 || !strings.Contains(l.calls[len(l.calls)-1], "--generation 4") {
		t.Fatalf("calls = %v, want a failed resize at 3 then one at 4", l.calls)
	}
}

func TestRetryIsBoundedToOneAttempt(t *testing.T) {
	// Given a writer that moves the generation after every read
	l := &racyLedger{generation: 3, bumps: 100, id: "clm-o"}

	syncAgainst(t, l, 4<<30)

	if got := l.count("heartbeat"); got != 2 {
		t.Fatalf("heartbeats = %d, want exactly one retry (%v)", got, l.calls)
	}
}

func TestRetryNeverFollowsAReplacedClaim(t *testing.T) {
	// Given the claim was replaced between the read and the write
	l := &racyLedger{generation: 3, bumps: 1, id: "clm-o", replaceAfterList: true}

	syncAgainst(t, l, 4<<30)

	if got := l.count("heartbeat"); got != 1 {
		t.Fatalf("heartbeats = %d, want no retry against a different claim (%v)", got, l.calls)
	}
}

func TestReporterHeartbeatRetriesAStaleGenerationOnce(t *testing.T) {
	l := &racyLedger{generation: 3, bumps: 1, id: "clm-o"}
	reporter := companion.Reporter{Resource: "ollama", Exec: l.exec}

	if err := reporter.Heartbeat(context.Background()); err != nil {
		t.Fatalf("Heartbeat: %v (calls %v)", err, l.calls)
	}
	if l.count("heartbeat") != 2 {
		t.Fatalf("calls = %v, want one retry", l.calls)
	}
}
