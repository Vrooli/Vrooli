//go:build !windows

package platform

import (
	"errors"
	"strings"
	"testing"
)

func TestDarwinRootlessSubmittedIntentCannotProveSameBootExtinction(t *testing.T) {
	for _, alive := range []bool{true, false} {
		t.Run(map[bool]string{true: "unassociated-live-wrapper", false: "unobserved-wrapper-exit"}[alive], func(t *testing.T) {
			b, f, i, s := newDarwinFixture(t)
			f.uncertainBootstrap = true
			if _, err := b.start(t.Context(), i, s); err == nil {
				t.Fatal("lost bootstrap response became successful submission")
			}
			f.present, f.retainRoot = false, true
			if !alive {
				f.process = darwinOwnedProcess{}
			}
			ref, err := b.lookup(t.Context(), i)
			if err != nil || ref.RootPID != 0 {
				t.Fatalf("rootless observation %+v %v", ref, err)
			}
			for retry := 0; retry < 2; retry++ {
				receipt, err := b.stop(t.Context(), ref)
				if !errors.Is(err, ErrOperationOutcomeLost) || receipt.NativeQuiescent {
					t.Fatalf("unknown root became quiescent: %+v %v", receipt, err)
				}
				if err := b.release(t.Context(), ref); !errors.Is(err, ErrOperationOutcomeLost) {
					t.Fatalf("unknown root lost native ledger: %v", err)
				}
				if released, err := b.released(t.Context(), ref, ref.Native.BootID); released || !errors.Is(err, ErrOperationOutcomeLost) {
					t.Fatalf("unknown root release confirmed: %t %v", released, err)
				}
				if absent, err := b.absent(t.Context(), i); absent || err != nil {
					t.Fatalf("same-boot unknown submit absent: %t %v", absent, err)
				}
				r, err := b.store.Read(i)
				if err != nil || !r.Revoked || !r.DispatchPrepared {
					t.Fatalf("revoked uncertainty not retained: %+v %v", r, err)
				}
			}
		})
	}
}

func TestDarwinRootlessUncertaintyCanRecoverExactRootWithoutRedispatch(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	f.uncertainBootstrap = true
	_, _ = b.start(t.Context(), i, s)
	f.present = false
	ref, err := b.lookup(t.Context(), i)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.stop(t.Context(), ref); !errors.Is(err, ErrOperationOutcomeLost) {
		t.Fatal(err)
	}
	f.present = true
	rooted, err := b.lookup(t.Context(), i)
	if err != nil || rooted.RootPID != 42 || rooted.Native.RootUniqueID != "999" {
		t.Fatalf("exact root recovery %+v %v", rooted, err)
	}
	if receipt, err := b.stop(t.Context(), rooted); err != nil || !receipt.NativeQuiescent {
		t.Fatalf("exact root did not retire: %+v %v", receipt, err)
	}
	if released, err := b.released(t.Context(), rooted, rooted.Native.BootID); released || err != nil {
		t.Fatalf("quiescence skipped exact metadata release %t %v", released, err)
	}
	if err := b.release(t.Context(), rooted); err != nil {
		t.Fatal(err)
	}
	if released, err := b.released(t.Context(), rooted, rooted.Native.BootID); !released || err != nil {
		t.Fatalf("exact-root release unresolved %t %v", released, err)
	}
	bootstrapCalls := 0
	for _, call := range f.calls {
		if strings.HasPrefix(call, "bootstrap user/501 ") {
			bootstrapCalls++
		}
	}
	if bootstrapCalls != 1 {
		t.Fatalf("recovery redispatched the native role %d times", bootstrapCalls)
	}
}

func TestDarwinRootlessCleanupSupportsNeverDispatchedAndOriginBootExtinction(t *testing.T) {
	for _, dispatched := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-private-bootstrap", true: "origin-boot-ended"}[dispatched], func(t *testing.T) {
			b, f, i, s := newDarwinFixture(t)
			if dispatched {
				f.uncertainBootstrap = true
			} else {
				f.domainMissing, f.domainCreationFails = true, true
			}
			if _, err := b.start(t.Context(), i, s); err == nil {
				t.Fatal("expected unresolved setup/submission")
			}
			f.domainMissing, f.present = false, false
			if dispatched {
				f.boot = "22222222-2222-3333-4444-555555555555"
			}
			ref, err := b.lookup(t.Context(), i)
			if err != nil || ref.RootPID != 0 {
				t.Fatalf("rootless cleanup ref %+v %v", ref, err)
			}
			if receipt, err := b.stop(t.Context(), ref); err != nil || !receipt.NativeQuiescent {
				t.Fatalf("proved extinction refused %+v %v", receipt, err)
			}
			if released, err := b.released(t.Context(), ref, ref.Native.BootID); released || err != nil {
				t.Fatalf("retained metadata was reported released %t %v", released, err)
			}
			if err := b.release(t.Context(), ref); err != nil {
				t.Fatal(err)
			}
			if released, err := b.released(t.Context(), ref, ref.Native.BootID); !released || err != nil {
				t.Fatalf("lost release acknowledgement not recoverable %t %v", released, err)
			}
		})
	}
}

func TestDarwinRootedReceiptCannotForgeBootExtinctionForLiveOrphan(t *testing.T) {
	b, f, i, s := newDarwinFixture(t)
	ref := startDarwinFixture(t, b, i, s)
	f.present = false
	forged := *ref.Native
	ref.Native = &forged
	forged.BootID = "22222222-2222-3333-4444-555555555555"
	if receipt, err := b.stop(t.Context(), ref); !errors.Is(err, ErrOperationIdentity) || receipt.NativeQuiescent {
		t.Fatalf("ref boot replaced persisted origin boot %+v %v", receipt, err)
	}
	if err := b.release(t.Context(), ref); !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("forged root receipt removed native uncertainty %v", err)
	}
	if released, err := b.released(t.Context(), ref, ref.Native.BootID); released || !errors.Is(err, ErrOperationIdentity) {
		t.Fatalf("forged receipt confirmed root extinction %t %v", released, err)
	}
	if _, err := b.store.Read(i); err != nil || f.process.PID != 42 {
		t.Fatalf("live orphan or native receipt changed %v", err)
	}
}
