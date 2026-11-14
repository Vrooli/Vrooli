package coreactivation

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/proto"
)

func TestAdmissionCannotReplaceFrozenExecutableEvidenceWithEqualRecordHashes(t *testing.T) {
	e, r, _, effects := fixture(t)
	r, _, _ = frozenFixture(t, e, r)
	e.qualifyFrozen = nil
	if _, err := e.Begin(context.Background(), r, false); err == nil {
		t.Fatal("marker images with equal declared contracts bypassed executable admission")
	}
	if len(effects.calls) != 0 {
		t.Fatal("unqualified images reached native admission effects")
	}
	if _, exists, err := e.Store.Current(); err != nil || exists {
		t.Fatalf("unqualified image produced an admitted record: %t %v", exists, err)
	}
}

func TestAdmissionRechecksCancellationAndDeadlineAfterQualification(t *testing.T) {
	for _, mode := range []string{"cancel", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			e, r, _, effects := fixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			effects.onCall = func(name string) {
				if name != "qualify" {
					return
				}
				if mode == "cancel" {
					cancel()
				} else {
					e.Now = func() time.Time { return time.Unix(0, r.DeadlineUnixNano) }
				}
			}
			if _, err := e.Begin(ctx, r, false); err == nil {
				t.Fatal("expired or cancelled qualification was admitted")
			}
			if _, exists, err := e.Store.Current(); err != nil || exists {
				t.Fatalf("expired qualification produced an admitted record: %t %v", exists, err)
			}
		})
	}
}

func TestRollbackCompatibilityRequiresBothCompiledContracts(t *testing.T) {
	prior := &cliv1.CliVersion{BuildIdentity: strings.Repeat("a", 64), RuntimeHomeContractSha256: strings.Repeat("b", 64), PersistentStateContractSha256: strings.Repeat("c", 64)}
	candidate := proto.Clone(prior).(*cliv1.CliVersion)
	candidate.BuildIdentity = strings.Repeat("d", 64)
	if err := QualifyRollbackIdentities(prior, candidate); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"missing_prior", "missing_candidate", "legacy_state", "changed_state", "changed_layout", "unstamped_build"} {
		t.Run(field, func(t *testing.T) {
			p, c := proto.Clone(prior).(*cliv1.CliVersion), proto.Clone(candidate).(*cliv1.CliVersion)
			switch field {
			case "missing_prior":
				p = nil
			case "missing_candidate":
				c = nil
			case "legacy_state":
				p.PersistentStateContractSha256 = ""
			case "changed_state":
				c.PersistentStateContractSha256 = strings.Repeat("e", 64)
			case "changed_layout":
				c.RuntimeHomeContractSha256 = strings.Repeat("e", 64)
			case "unstamped_build":
				c.BuildIdentity = "unknown"
			}
			err := QualifyRollbackIdentities(p, c)
			if !errors.Is(err, ErrConflict) {
				t.Fatal("unsupported rollback admitted")
			}
			// Only a stamped pair that differs in its state contract alone is
			// a state-contract change; the installer's remedy depends on it.
			if errors.Is(err, ErrStateContractChanged) != (field == "changed_state") {
				t.Fatalf("refusal kind is misclassified: %v", err)
			}
		})
	}
}

func TestStateContractRefusalExplanationNamesWhyItCannotCompare(t *testing.T) {
	refusal := QualifyRollbackIdentities(
		&cliv1.CliVersion{BuildIdentity: strings.Repeat("a", 64), RuntimeHomeContractSha256: strings.Repeat("b", 64), PersistentStateContractSha256: strings.Repeat("c", 64)},
		&cliv1.CliVersion{BuildIdentity: strings.Repeat("d", 64), RuntimeHomeContractSha256: strings.Repeat("b", 64), PersistentStateContractSha256: strings.Repeat("e", 64)})
	missing := filepath.Join(t.TempDir(), "vrooli")
	explained := ExplainRollbackRefusal(context.Background(), refusal, missing, missing)
	if !errors.Is(explained, ErrStateContractChanged) || !strings.Contains(explained.Error(), "serving cccccccccccc, candidate eeeeeeeeeeee") || !strings.Contains(explained.Error(), "\n  - the candidate does not report a per-owner state manifest") {
		t.Fatalf("explanation = %v", explained)
	}
	other := fmt.Errorf("%w: unrelated", ErrConflict)
	if ExplainRollbackRefusal(context.Background(), other, missing, missing) != other {
		t.Fatal("a non-state refusal was rewritten")
	}
}

func TestCandidateEntrypointFenceKeepsUnverifiedCommandsOutOfStateWriters(t *testing.T) {
	e, r, ref, _ := fixture(t)
	r, err := e.Begin(context.Background(), r, false)
	if err != nil {
		t.Fatal(err)
	}
	check := func(native bool, sha, build, state string, want bool) {
		t.Helper()
		err := e.Store.GuardEntrypoint(context.Background(), sha, build, state, native)
		if (err == nil) != want {
			t.Fatalf("native=%t phase=%s fence=%s allowed=%t: %v", native, r.Phase, r.Fence, want, err)
		}
	}
	check(false, r.PriorSHA, r.PriorBuild, r.PriorStateContract, true)
	check(false, r.CandidateSHA, r.CandidateBuild, r.CandidateStateContract, false)
	check(true, r.CandidateSHA, r.CandidateBuild, r.CandidateStateContract, false)
	check(false, strings.Repeat("9", 64), r.CandidateBuild, r.CandidateStateContract, false)
	r.Phase, r.Fence, r.DispatchCommitted, r.WorkerRef = Activating, Claimed, true, &ref
	if err := e.Store.put(&r); err != nil {
		t.Fatal(err)
	}
	check(true, r.CandidateSHA, r.CandidateBuild, r.CandidateStateContract, true)
	check(false, r.CandidateSHA, r.CandidateBuild, r.CandidateStateContract, false)
	check(true, r.CandidateSHA, "wrong-build", r.CandidateStateContract, false)
	check(true, r.CandidateSHA, r.CandidateBuild, strings.Repeat("f", 64), false)
	r.Phase, r.Fence, r.FailureClass = Restoring, Revoked, "activation"
	if err := e.Store.put(&r); err != nil {
		t.Fatal(err)
	}
	check(true, r.CandidateSHA, r.CandidateBuild, r.CandidateStateContract, false)
	check(true, r.PriorSHA, r.PriorBuild, r.PriorStateContract, true)
	r.Phase, r.RestorationRequired, r.FailedReleases, r.FailedBuilds = FailedRestored, true, []string{r.CandidateSHA}, []string{r.CandidateBuild}
	if err := e.Store.put(&r); err != nil {
		t.Fatal(err)
	}
	check(false, r.CandidateSHA, r.CandidateBuild, r.CandidateStateContract, true)
}

func TestExpiredCandidateCannotRestartFromItsOldClaimAfterBootOrDeadline(t *testing.T) {
	e, r, ref, _ := fixture(t)
	r = admit(t, e, r)
	r.Phase, r.Fence, r.WorkerRef, r.DispatchCommitted = Activating, Claimed, &ref, true
	r.CreatedUnixNano = time.Now().Add(-time.Minute).UnixNano()
	r.DeadlineUnixNano = time.Now().Add(-time.Second).UnixNano()
	if err := e.Store.put(&r); err != nil {
		t.Fatal(err)
	}
	if err := e.Store.GuardEntrypoint(context.Background(), r.CandidateSHA, r.CandidateBuild, r.CandidateStateContract, true); err == nil {
		t.Fatal("expired candidate started from an old durable claim")
	}
	if err := e.Store.GuardEntrypoint(context.Background(), r.PriorSHA, r.PriorBuild, r.PriorStateContract, true); err != nil {
		t.Fatal("deadline blocked retained prior recovery")
	}
}
