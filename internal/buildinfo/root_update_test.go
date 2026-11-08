package buildinfo

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/coreactivation"
)

func TestEnrollmentAppliesToCanonicalRootAndAliasAndRefusesLostPolicy(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("native root updates are Linux-only")
	}
	home := useTestRebuildHome(t)
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := coreactivation.NewStore(home)
	if err != nil {
		t.Fatal(err)
	}
	qualify := func(context.Context) (func(), error) { return func() {}, nil }
	if _, err := store.ConfigureEnrollment(context.Background(), true, qualify); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "bin-alias")
	if err := os.Symlink(bin, alias); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{filepath.Join(bin, "vrooli"), filepath.Join(alias, "vrooli")} {
		if enrolled, err := EnrolledRootUpdate(target); err != nil || !enrolled {
			t.Fatalf("enrolled root %s was bypassed: %t %v", target, enrolled, err)
		}
	}
	for _, target := range []string{filepath.Join(bin, "scenario-cli"), filepath.Join(t.TempDir(), "vrooli")} {
		if enrolled, err := EnrolledRootUpdate(target); err != nil || enrolled {
			t.Fatalf("unrelated install inherited enrollment: %t %v", enrolled, err)
		}
	}
	if err := os.Remove(filepath.Join(store.Directory(), "enrollment.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := EnrolledRootUpdate(filepath.Join(bin, "vrooli")); err == nil {
		t.Fatal("lost policy permitted ordinary root publication")
	}
}

func TestIndependentRootUpdateHandoffRequiresOneVerifiedReceipt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	for _, tc := range []struct {
		name, output, exit, want string
	}{
		{"verified", `{"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","phase":"verified","failure_class":""}`, "0", ""},
		{"accepted is not verified", `{"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","phase":"prepared","failure_class":""}`, "0", "did not verify"},
		{"restored is failed", `{"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","phase":"failed_restored","failure_class":"worker_lost"}`, "1", "worker_lost"},
		{"no receipt is unresolved, not refused", "", "1", "outcome is unresolved: the installed coordinator returned no receipt; inspect `vrooli runtime update status --json`\ncoordinator: private-child-diagnostic"},
		{"refused before admission", `{"id":"","phase":"","failure_class":"admission_refused","reason":"another activation remains unresolved"}`, "1", "root update refused: another activation remains unresolved"},
		{"refused without a reason", `{"id":"","phase":"","failure_class":"admission_refused"}`, "1", "refused the request before admission"},
		{"unknown field", `{"id":"a","phase":"verified","extra":true}`, "0", "unresolved"},
		{"duplicate receipt", "{\"id\":\"a\",\"phase\":\"verified\"}\n{}", "0", "ambiguous"},
		{"oversized", strings.Repeat("x", 8193), "0", "exceeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := useTestRebuildHome(t)
			worker, err := coreactivation.InstalledWatchdogPath(home)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(worker), 0o700); err != nil {
				t.Fatal(err)
			}
			body := "#!/bin/sh\n[ \"$1\" = --submit-enrolled-core-update ] || exit 9\n[ \"$2\" = /candidate ] || exit 9\n[ \"$3\" = /source ] || exit 9\nprintf '%s\\n' '" + tc.output + "'\nprintf '%s\\n' private-child-diagnostic >&2\nexit " + tc.exit + "\n"
			if err := os.WriteFile(worker, []byte(body), 0o700); err != nil {
				t.Fatal(err)
			}
			err = RequestRootUpdate(context.Background(), "/candidate", "/source")
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("unexpected handoff outcome: %v", err)
			}
			// A structured refusal carries its own reason; the coordinator's
			// diagnostic tail is kept only where no structured reason exists.
			if err != nil && strings.Contains(err.Error(), "private-child-diagnostic") && strings.Contains(tc.output, `"reason"`) {
				t.Fatal("diagnostic tail duplicated a stated reason")
			}
		})
	}
}

func TestRootRefusalWithoutAReasonIsDiagnosedLocally(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("shell fixture")
	}
	home := useTestRebuildHome(t)
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	identity := func(state string) string {
		return "#!/bin/sh\n[ \"$1\" = version ] && [ \"$2\" = --json ] || exit 3\nprintf '%s\\n' '{\"cliVersion\":\"1\",\"platformVersion\":\"1\",\"buildIdentity\":\"" + strings.Repeat("a", 64) + "\",\"runtimeHomeContractSha256\":\"" + strings.Repeat("b", 64) + "\",\"persistentStateContractSha256\":\"" + state + "\"}'\n"
	}
	candidate := filepath.Join(t.TempDir(), "vrooli")
	for path, body := range map[string]string{filepath.Join(bin, "vrooli"): identity(strings.Repeat("c", 64)), candidate: identity(strings.Repeat("d", 64))} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	err = diagnoseRootRefusal(home, candidate, "/source", "", "")
	for _, want := range []string{"candidate changes the persistent-state contract", "serving cccccccccccc, candidate dddddddddddd", "per-owner state manifest", "Operator decision required", "make rebaseline", "/source", "Agents must not run it"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("state-contract refusal does not say %q: %v", want, err)
		}
	}
	if err := os.WriteFile(candidate, []byte(identity(strings.Repeat("c", 64))), 0o700); err != nil {
		t.Fatal(err)
	}
	err = diagnoseRootRefusal(home, candidate, "/source", "failed release requires explicit retry", "")
	if err == nil || err.Error() != "root update refused: failed release requires explicit retry" {
		t.Fatalf("compatible refusal lost the coordinator reason: %v", err)
	}
	err = diagnoseRootRefusal(home, candidate, "/source", "", "\ncoordinator: deadline")
	if err == nil || !strings.Contains(err.Error(), "outcome is unresolved") || strings.Contains(err.Error(), "refused") {
		t.Fatalf("a receipt-less compatible request was reported as refused: %v", err)
	}
}
