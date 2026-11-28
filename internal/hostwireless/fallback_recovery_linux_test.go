//go:build linux

package hostwireless

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestWirelessFallbackRequiresIndependentBootAndCurrentProviders(t *testing.T) {
	for _, mode := range []string{"verified", "unproven", "same-boot", "wrong-kernel", "wrong-image", "wired-down", "missing-gpu", "artifact-drift"} {
		t.Run(mode, func(t *testing.T) {
			o, req, write := recoveryFixture(t)
			if mode != "unproven" {
				if _, err := o.record(context.Background(), req); err != nil {
					t.Fatal(err)
				}
			}
			if mode != "same-boot" {
				write("proc/sys/kernel/random/boot_id", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
			}
			switch mode {
			case "wrong-kernel":
				o.release = func() (string, error) { return req.TargetABI, nil }
			case "wrong-image":
				write("proc/cmdline", "BOOT_IMAGE=/boot/vmlinuz-foreign")
			case "wired-down":
				write("sys/class/net/enp1s0/carrier", "0")
			case "missing-gpu":
				if err := o.root.Remove("sys/module/nvidia"); err != nil {
					t.Fatal(err)
				}
			case "artifact-drift":
				write("boot/initrd.img-"+req.RecoveryRelease, "foreign")
			}
			_, err := o.verifyFallback(context.Background(), req)
			if (err == nil) != (mode == "verified") {
				t.Fatalf("mode %s: %v", mode, err)
			}
		})
	}
}

func TestWirelessFallbackRestoresOnlyExactTargetFilesWithoutRadioCalls(t *testing.T) {
	for _, mode := range []string{"verified", "pending-radio", "wrong-kernel", "unproven", "wrong-machine", "foreign-file", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			f := radioTestFixture(t)
			ctx := context.Background()
			if mode == "pending-radio" {
				if err := activateWirelessRadio(ctx, f.owner); err != nil {
					t.Fatal(err)
				}
			}
			// The currently running fallback owns its own runtime. Even observing
			// candidate identity or invoking an unload would cross this boundary.
			forbidden := func() { t.Fatal("fallback file recovery touched radio runtime") }
			f.owner.radio = &wirelessRadioSystem{
				identity: func(context.Context, *durableWireless, string) (bool, bool, bool, error) {
					forbidden()
					return false, false, false, nil
				},
				unload:  func(context.Context, string) error { forbidden(); return nil },
				command: func(context.Context, string, ...string) ([]byte, error) { forbidden(); return nil, nil },
			}
			f.owner.independentRecovery = true
			f.write(t, "proc/sys/kernel/random/boot_id", "22222222-2222-4222-8222-222222222222")
			stored := f.owner.intent.Context
			release := stored.Recovery.RecoveryRelease
			if mode == "wrong-kernel" {
				release = nobleABI
			}
			proof := RecoveryProof{SchemaVersion: 1, Request: stored.Recovery, MachineID: stored.MachineID}
			if mode == "wrong-machine" {
				proof.MachineID = strings.Repeat("b", 32)
			}
			verify := func(context.Context, RecoveryProofRequest) (RecoveryProof, error) {
				if mode == "unproven" {
					return proof, errors.New("no independently measured recovery record")
				}
				return proof, nil
			}
			var foreignPath string
			if mode == "foreign-file" {
				for _, file := range f.owner.intent.Files {
					if !file.After.Absent {
						foreignPath = file.Path
						break
					}
				}
				if foreignPath == "" {
					t.Fatal("fixture has no published file")
				}
				f.write(t, foreignPath, "foreign owner bytes")
			}
			if mode == "canceled" {
				cancelCtx, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelCtx
			}
			_, err := runFallbackRecoveryWith(ctx, f.owner, release, true, verify)
			wantSuccess := mode == "verified" || mode == "pending-radio"
			if (err == nil) != wantSuccess {
				t.Fatalf("mode %s: %v", mode, err)
			}
			for _, file := range f.owner.intent.Files {
				got, e := f.owner.identity(file.Path)
				if e != nil {
					t.Fatal(e)
				}
				if file.Path == foreignPath {
					data, e := f.target.root.ReadFile(file.Path)
					if e != nil || string(data) != "foreign owner bytes" {
						t.Fatal("foreign file changed", e)
					}
					continue
				}
				want := file.After
				if wantSuccess {
					want = file.Before
				}
				if got != want {
					t.Fatalf("unexpected target change %s: %+v != %+v", file.Path, got, want)
				}
			}
			if mode == "pending-radio" {
				record, e := f.owner.readRadioRecord()
				if e != nil || record.RestoreKind != "independent_kernel_files_only" || record.RecoveryBootID != "22222222-2222-4222-8222-222222222222" {
					t.Fatalf("fallback completion not durable: %+v %v", record, e)
				}
			}
			if !wantSuccess {
				active, e := f.target.root.ReadFile(wirelessJournalDir + "/active.json")
				if e != nil && !errors.Is(e, os.ErrNotExist) {
					t.Fatal(e)
				}
				if !strings.Contains(string(active), f.owner.intent.ID) {
					t.Fatal("failed recovery lost active child")
				}
			}
		})
	}
}
