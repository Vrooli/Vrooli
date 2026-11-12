package hosthandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	climanifest "github.com/vrooli/vrooli/cli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostwireless"
)

const (
	wirelessFixturePin         = "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	wirelessFixtureSigned      = "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	wirelessFixtureTransaction = "1234567890abcdef1234567890abcdef"
)

func dispatchWirelessFixture(t *testing.T, s wirelessServices, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	parent := &rootcli.CommandContext{Context: context.Background(), Stdout: &out, Stderr: &out}
	parent.Globals.JSON = true
	bindings := hostBindings(parent, hostCommandNames)
	bindings["wireless"] = func(run cliapp.RunContext) error { return runHostWirelessWithServices(run, parent, s) }
	group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "host", bindings)
	if err != nil {
		t.Fatal(err)
	}
	app := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli host", Commands: []cliapp.CommandGroup{{Commands: group.Subcommands}}})
	err = app.RunWithWriters(append([]string{"wireless"}, args...), &out, &out)
	return out.String(), err
}

func wirelessFixtureServices(t *testing.T) wirelessServices {
	t.Helper()
	return wirelessServices{
		operation: func(context.Context, string, hostwireless.WirelessOperationRequest, bool) (hostwireless.WirelessOperationResult, error) {
			t.Error("unexpected operation effect or preview")
			return hostwireless.WirelessOperationResult{}, nil
		},
		sign: func(context.Context, string, string, string, string) (hostwireless.SignedBundleReceipt, error) {
			t.Error("unexpected signer invocation")
			return hostwireless.SignedBundleReceipt{}, nil
		},
		publish: func(context.Context, string, string, string, string) (hostwireless.SignedBundleReceipt, error) {
			t.Error("unexpected publication")
			return hostwireless.SignedBundleReceipt{}, nil
		},
		verify: func(context.Context, string, string, string, string) (hostwireless.SignedCopyEvidence, error) {
			t.Error("unexpected signed-copy inspection")
			return hostwireless.SignedCopyEvidence{}, nil
		},
		plan: func(context.Context, string, string, string) (hostwireless.ActivationPlan, error) {
			t.Error("unexpected plan inspection")
			return hostwireless.ActivationPlan{}, nil
		},
		trust: func(context.Context, string) (hostwireless.SigningTrust, error) {
			t.Error("unexpected certificate inspection")
			return hostwireless.SigningTrust{}, nil
		},
		elevationRequired: func() bool { return true },
	}
}

func TestWirelessManifestRoutesExactTransactionsAndExplicitApply(t *testing.T) {
	for _, action := range []string{"stage", "activate", "recover", "rollback", "status", "record-recovery"} {
		for _, apply := range []bool{false, true} {
			if action == "status" && apply {
				continue
			}
			t.Run(action+"/apply="+map[bool]string{false: "false", true: "true"}[apply], func(t *testing.T) {
				s := wirelessFixtureServices(t)
				calls := 0
				s.operation = func(ctx context.Context, op string, req hostwireless.WirelessOperationRequest, effect bool) (hostwireless.WirelessOperationResult, error) {
					calls++
					if op != action || effect != apply || req.Profile != "noble-mt6639" || ctx == nil {
						t.Fatalf("incorrect dispatch %s %+v %t", op, req, effect)
					}
					if action == "stage" && (req.Bundle != "fixture-generation" || req.SignedBundle != wirelessFixtureSigned || req.ReceiptSHA256 != wirelessFixturePin || req.RecoveryKernel != "7.0.0-29-generic" || req.RecoveryEntry != "stock-entry") {
						t.Fatalf("stage dropped selectors %+v", req)
					}
					if action != "stage" && action != "record-recovery" && req.Transaction != wirelessFixtureTransaction {
						t.Fatalf("transaction lost %+v", req)
					}
					if action == "record-recovery" && (req.RecoveryKernel != "7.0.0-29-generic" || req.RecoveryEntry != "stock-entry") {
						t.Fatalf("recovery selectors lost %+v", req)
					}
					if action != "status" && action != "record-recovery" && !req.MaintenanceWindow {
						t.Fatalf("maintenance coordination lost %+v", req)
					}
					return hostwireless.WirelessOperationResult{Operation: op, Profile: req.Profile, State: "fixture", Transaction: req.Transaction, Checks: []hostwireless.WirelessCheck{{Name: "independent proof", Satisfied: false, Detail: "still required"}}}, nil
				}
				args := []string{action, "--profile", "noble-mt6639"}
				if action == "stage" {
					args = append(args, "--bundle", "fixture-generation", "--signed-bundle", wirelessFixtureSigned, "--receipt-sha256", wirelessFixturePin)
				}
				if action == "stage" || action == "record-recovery" {
					args = append(args, "--recovery-kernel", "7.0.0-29-generic", "--recovery-entry", "stock-entry")
				} else {
					args = append(args, "--transaction", wirelessFixtureTransaction)
				}
				if action != "status" && action != "record-recovery" {
					args = append(args, "--maintenance-window")
				}
				if apply {
					args = append(args, "--apply")
				}
				out, err := dispatchWirelessFixture(t, s, args...)
				if err != nil {
					t.Fatalf("%v output=%s", err, out)
				}
				var result hostwireless.WirelessOperationResult
				if err = json.Unmarshal([]byte(out), &result); err != nil || result.Operation != action || result.State != "fixture" || result.Ready || calls != 1 {
					t.Fatalf("result=%+v calls=%d err=%v output=%s", result, calls, err, out)
				}
			})
		}
	}
}

func TestWirelessManifestRejectsIncompleteAndIgnoredFlagsBeforeOwnerCalls(t *testing.T) {
	cases := [][]string{
		{"status", "--transaction", wirelessFixtureTransaction, "--apply"},
		{"status", "--transaction", wirelessFixtureTransaction, "--maintenance-window"},
		{"stage", "--bundle", "generation", "--recovery-kernel", "7.0.0-29-generic", "--recovery-entry", "entry", "--apply"},
		{"stage", "--bundle", "generation", "--signed-bundle", wirelessFixtureSigned, "--recovery-kernel", "7.0.0-29-generic", "--recovery-entry", "entry", "--apply"},
		{"activate"},
		{"recover"},
		{"rollback"},
		{"status"},
		{"record-recovery", "--recovery-kernel", "7.0.0-29-generic"},
		{"sign", "--bundle", "generation", "--cert", "public.der", "--apply"},
		{"sign", "--bundle", "generation", "--cert", "public.der", "--receipt-sha256", "bad", "--apply"},
		{"sign", "--bundle", "generation", "--cert", "public.der", "--signed-dir", "foreign"},
		{"publish-signed", "--bundle", "generation", "--cert", "public.der"},
		{"publish-signed", "--bundle", "generation", "--signed-dir", "signed", "--cert", "public.der", "--receipt-sha256", wirelessFixturePin},
		{"prepare", "--transaction", wirelessFixtureTransaction},
		{"activation-plan", "--signed-bundle", wirelessFixtureSigned},
		{"verify-signed-copy", "--apply"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			args = append(args, "--profile", "noble-mt6639")
			if out, err := dispatchWirelessFixture(t, wirelessFixtureServices(t), args...); err == nil {
				t.Fatalf("accepted incomplete or misleading request %v: %s", args, out)
			}
		})
	}
}

func TestWirelessManifestSignPreviewNeverSignsAndRequiresReviewedPinForApply(t *testing.T) {
	s := wirelessFixtureServices(t)
	plans, trusts := 0, 0
	s.plan = func(_ context.Context, profile, generation, recovery string) (hostwireless.ActivationPlan, error) {
		plans++
		return hostwireless.ActivationPlan{Profile: profile, Bundle: generation, BundleReceiptSHA256: wirelessFixturePin}, nil
	}
	s.trust = func(_ context.Context, cert string) (hostwireless.SigningTrust, error) {
		trusts++
		if cert != "public.der" {
			t.Fatal(cert)
		}
		return hostwireless.SigningTrust{ModuleOnly: true, Enrollment: "not_enrolled"}, nil
	}
	args := []string{"sign", "--profile", "noble-mt6639", "--bundle", "generation", "--cert", "public.der"}
	out, err := dispatchWirelessFixture(t, s, args...)
	if err != nil || plans != 1 || trusts != 1 || !strings.Contains(out, "not_enrolled") {
		t.Fatalf("preview err=%v output=%s", err, out)
	}
	out, err = dispatchWirelessFixture(t, s, append(args, "--receipt-sha256", wirelessFixtureSigned)...)
	if err == nil || !strings.Contains(err.Error(), "differs") {
		t.Fatalf("wrong reviewed preview pin accepted %v %s", err, out)
	}
	_, err = dispatchWirelessFixture(t, s, append(args, "--receipt-sha256", wirelessFixturePin, "--apply")...)
	if !errors.Is(err, hostreqkit.ErrElevationRequired) {
		t.Fatalf("expected truthful elevation failure, got %v", err)
	}
	s.elevationRequired = func() bool { return false }
	s.sign = func(_ context.Context, profile, generation, cert, pin string) (hostwireless.SignedBundleReceipt, error) {
		if profile != "noble-mt6639" || generation != "generation" || cert != "public.der" || pin != wirelessFixturePin {
			t.Fatal("signer selectors lost")
		}
		return hostwireless.SignedBundleReceipt{ReceiptSHA256: wirelessFixtureSigned, PublicCertificate: "retained/certificate.der"}, nil
	}
	out, err = dispatchWirelessFixture(t, s, append(args, "--receipt-sha256", wirelessFixturePin, "--apply")...)
	if err != nil || !strings.Contains(out, "retained/certificate.der") {
		t.Fatalf("sign dispatch %v %s", err, out)
	}
}

func TestWirelessManifestPublicationPreviewAndApplyHaveDistinctOwners(t *testing.T) {
	for _, apply := range []bool{false, true} {
		t.Run(map[bool]string{false: "preview", true: "apply"}[apply], func(t *testing.T) {
			s := wirelessFixtureServices(t)
			calls := 0
			check := func(profile, generation, dir, cert string) {
				calls++
				if profile != "noble-mt6639" || generation != "generation" || dir != "signed" || cert != "public.der" {
					t.Fatal("publication selectors lost")
				}
			}
			if apply {
				s.publish = func(_ context.Context, p, g, d, c string) (hostwireless.SignedBundleReceipt, error) {
					check(p, g, d, c)
					return hostwireless.SignedBundleReceipt{ReceiptSHA256: wirelessFixtureSigned, ModulesDirectory: "retained/modules"}, nil
				}
			} else {
				s.verify = func(_ context.Context, p, g, d, c string) (hostwireless.SignedCopyEvidence, error) {
					check(p, g, d, c)
					return hostwireless.SignedCopyEvidence{Profile: p, Bundle: g, SignaturesVerified: true}, nil
				}
			}
			args := []string{"publish-signed", "--profile", "noble-mt6639", "--bundle", "generation", "--signed-dir", "signed", "--cert", "public.der"}
			if apply {
				args = append(args, "--apply")
			}
			out, err := dispatchWirelessFixture(t, s, args...)
			if err != nil || calls != 1 {
				t.Fatalf("calls=%d err=%v output=%s", calls, err, out)
			}
			if apply && !strings.Contains(out, "retained/modules") {
				t.Fatal("lost retained directory")
			}
		})
	}
}

func TestWirelessManifestPreservesFailedOperationEvidenceAndElevation(t *testing.T) {
	s := wirelessFixtureServices(t)
	s.operation = func(_ context.Context, op string, req hostwireless.WirelessOperationRequest, _ bool) (hostwireless.WirelessOperationResult, error) {
		return hostwireless.WirelessOperationResult{Operation: op, State: "elevation_required", Transaction: req.Transaction}, hostreqkit.ErrElevationRequired
	}
	out, err := dispatchWirelessFixture(t, s, "recover", "--profile", "noble-mt6639", "--transaction", wirelessFixtureTransaction, "--apply")
	if !errors.Is(err, hostreqkit.ErrElevationRequired) || !strings.Contains(err.Error(), "sudo") || !strings.Contains(out, "elevation_required") {
		t.Fatalf("lost failed evidence %v %s", err, out)
	}
}
