package coreactivation

import (
	"context"
	"errors"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
)

func nativeFixture(t *testing.T) (NativeWorker, Record, platform.OperationRef) {
	t.Helper()
	e, r, ref, _ := fixture(t)
	admit(t, e, r)
	dispatch(t, e, r)
	r, _, _ = e.Store.Current()
	boot, _, _ := strings.Cut(r.PriorIncarnation, ":")
	o := &nativeWorkerOps{
		lookup: func(context.Context, platform.OperationIntent) (platform.OperationRef, error) { return ref, nil },
		inspect: func(context.Context, platform.OperationRef) (platform.OperationObservation, error) {
			return platform.OperationObservation{Ref: ref, State: platform.OperationRunning, RootAlive: true, Populated: true}, nil
		},
		stop: func(context.Context, platform.OperationRef) (platform.OperationStopReceipt, error) {
			return platform.OperationStopReceipt{Ref: ref, NativeQuiescent: true}, nil
		},
		release:     func(context.Context, platform.OperationRef) error { return nil },
		absent:      func(context.Context, platform.OperationIntent) (bool, error) { return false, nil },
		released:    func(context.Context, platform.OperationRef, string) (bool, error) { return false, nil },
		members:     func(context.Context, platform.ScopeRef) ([]int, error) { return []int{ref.RootPID}, nil },
		incarnation: func(int) (string, error) { return boot + ":" + ref.RootStartTime, nil },
		imageDigest: func(context.Context, int) (string, error) { return r.WorkerSHA, nil },
		pid:         func() int { return ref.RootPID },
	}
	return NativeWorker{native: o}, r, ref
}

func TestClaimedWorkerRepeatsExactNativeProofAtEveryEffect(t *testing.T) {
	n, r, ref := nativeFixture(t)
	r.Phase, r.Fence, r.WorkerRef = Publishing, Claimed, &ref
	if err := n.VerifyClaimedSelf(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	r.Fence, r.Phase, r.FailureClass = Revoked, Restoring, "activation"
	if err := n.VerifyClaimedSelf(context.Background(), r); err == nil {
		t.Fatal("revoked worker retained destructive effect permission")
	}
	r.Fence, r.Phase, r.FailureClass = Claimed, Activating, ""
	n.native.imageDigest = func(context.Context, int) (string, error) { return strings.Repeat("9", 64), nil }
	if err := n.VerifyClaimedSelf(context.Background(), r); err == nil {
		t.Fatal("replaced native image retained claimed effect permission")
	}
}

func TestNativeWorkerRequiresExactLiveRootAndMembership(t *testing.T) {
	for _, failure := range []string{"none", "lookup_unknown", "wrong_pid", "wrong_invocation", "different_boot", "changed_start", "not_running", "unknown_membership", "outside_tree", "wrong_spec", "wrong_namespace", "wrong_image", "unknown_image"} {
		t.Run(failure, func(t *testing.T) {
			n, r, ref := nativeFixture(t)
			switch failure {
			case "lookup_unknown":
				n.native.lookup = func(context.Context, platform.OperationIntent) (platform.OperationRef, error) {
					return platform.OperationRef{}, errors.New("manager unavailable")
				}
			case "wrong_pid":
				n.native.pid = func() int { return ref.RootPID + 1 }
			case "wrong_invocation":
				ref.NativeInvocationID = strings.Repeat("9", 32)
			case "different_boot":
				n.native.incarnation = func(int) (string, error) { return "ffffffff-ffff-ffff-ffff-ffffffffffff:" + ref.RootStartTime, nil }
			case "changed_start":
				n.native.incarnation = func(int) (string, error) { return strings.Split(r.PriorIncarnation, ":")[0] + ":999", nil }
			case "not_running":
				n.native.inspect = func(context.Context, platform.OperationRef) (platform.OperationObservation, error) {
					return platform.OperationObservation{}, nil
				}
			case "unknown_membership":
				n.native.members = func(context.Context, platform.ScopeRef) ([]int, error) { return nil, errors.New("unreadable cgroup") }
			case "outside_tree":
				n.native.members = func(context.Context, platform.ScopeRef) ([]int, error) { return []int{ref.RootPID + 1}, nil }
			case "wrong_spec":
				ref.Intent.SpecDigest = strings.Repeat("8", 64)
			case "wrong_namespace":
				ref.Scope.Path = "/user.slice/foreign.service"
			case "wrong_image":
				n.native.imageDigest = func(context.Context, int) (string, error) { return strings.Repeat("f", 64), nil }
			case "unknown_image":
				n.native.imageDigest = func(context.Context, int) (string, error) { return "", errors.New("running image unavailable") }
			}
			err := n.AuthorizeWorker(context.Background(), r, ref)
			if (err == nil) != (failure == "none") {
				t.Fatalf("native authorization failure=%s err=%v", failure, err)
			}
		})
	}
}

func TestNativeExclusionRequiresRevokeAndProvenTreeQuiescence(t *testing.T) {
	for _, failure := range []string{"none", "unrevoked", "self", "unknown_stop", "unproven_quiescence", "changed_receipt", "lost_lookup", "absent_unclaimed", "released_claimed", "unknown_release"} {
		t.Run(failure, func(t *testing.T) {
			n, r, ref := nativeFixture(t)
			r.Phase = Restoring
			r.Fence = Revoked
			r.FailureClass = "worker_lost"
			r.WorkerRef = &ref
			n.native.pid = func() int { return ref.RootPID + 1 }
			calls := 0
			normalStop := n.native.stop
			n.native.stop = func(ctx context.Context, ref platform.OperationRef) (platform.OperationStopReceipt, error) {
				calls++
				return normalStop(ctx, ref)
			}
			switch failure {
			case "unrevoked":
				r.Phase = Prepared
				r.Fence = Ready
				r.FailureClass = ""
				r.WorkerRef = nil
			case "self":
				n.native.pid = func() int { return ref.RootPID }
			case "unknown_stop", "released_claimed", "unknown_release":
				n.native.stop = func(context.Context, platform.OperationRef) (platform.OperationStopReceipt, error) {
					return platform.OperationStopReceipt{}, errors.New("stop outcome unknown")
				}
			case "unproven_quiescence":
				n.native.stop = func(context.Context, platform.OperationRef) (platform.OperationStopReceipt, error) {
					return platform.OperationStopReceipt{Ref: ref}, nil
				}
			case "changed_receipt":
				n.native.stop = func(context.Context, platform.OperationRef) (platform.OperationStopReceipt, error) {
					other := ref
					other.NativeInvocationID = strings.Repeat("8", 32)
					return platform.OperationStopReceipt{Ref: other, NativeQuiescent: true}, nil
				}
			case "lost_lookup", "absent_unclaimed":
				r.WorkerRef = nil
				n.native.lookup = func(context.Context, platform.OperationIntent) (platform.OperationRef, error) {
					return platform.OperationRef{}, platform.ErrOperationOutcomeLost
				}
			}
			if failure == "absent_unclaimed" {
				n.native.absent = func(context.Context, platform.OperationIntent) (bool, error) { return true, nil }
			}
			if failure == "released_claimed" {
				n.native.released = func(_ context.Context, got platform.OperationRef, incarnation string) (bool, error) {
					boot, _, _ := strings.Cut(r.PriorIncarnation, ":")
					if incarnation != boot+":"+ref.RootStartTime {
						t.Fatal("worker release lost admission boot")
					}
					return true, nil
				}
			}
			if failure == "unknown_release" {
				n.native.released = func(context.Context, platform.OperationRef, string) (bool, error) {
					return false, errors.New("manager unknown")
				}
			}
			err := n.ExcludeWorker(context.Background(), r)
			wantSuccess := failure == "none" || failure == "absent_unclaimed" || failure == "released_claimed"
			if (err == nil) != wantSuccess {
				t.Fatalf("exclusion failure=%s err=%v", failure, err)
			}
			if (failure == "self" || failure == "unrevoked" || failure == "absent_unclaimed") && calls != 0 {
				t.Fatalf("unexpected native stop calls=%d", calls)
			}
		})
	}
}

func TestNativeReleaseRequiresTerminalOutcomeAndReconcilesLostAcknowledgement(t *testing.T) {
	for _, state := range []string{"nonterminal", "released", "release_ack_lost", "still_present", "manager_unknown"} {
		t.Run(state, func(t *testing.T) {
			n, r, ref := nativeFixture(t)
			r.WorkerRef = &ref
			r.Fence = Revoked
			r.Phase = FailedRestored
			r.FailureClass = "activation"
			r.RestorationRequired = true
			r.FailedReleases = []string{r.CandidateSHA}
			r.FailedBuilds = []string{r.CandidateBuild}
			n.native.pid = func() int { return ref.RootPID + 1 }
			calls := 0
			n.native.release = func(context.Context, platform.OperationRef) error {
				calls++
				if state == "released" {
					return nil
				}
				return errors.New("release acknowledgement lost")
			}
			if state == "nonterminal" {
				r.Phase = Restoring
			}
			if state == "release_ack_lost" {
				n.native.absent = func(context.Context, platform.OperationIntent) (bool, error) { return true, nil }
			}
			if state == "manager_unknown" {
				n.native.absent = func(context.Context, platform.OperationIntent) (bool, error) {
					return false, errors.New("manager unavailable")
				}
			}
			err := n.ReleaseWorker(context.Background(), r)
			wantSuccess := state == "released" || state == "release_ack_lost"
			if (err == nil) != wantSuccess {
				t.Fatalf("release state=%s err=%v", state, err)
			}
			if state == "nonterminal" && calls != 0 {
				t.Fatal("manager evidence released before terminal outcome")
			}
		})
	}
}
