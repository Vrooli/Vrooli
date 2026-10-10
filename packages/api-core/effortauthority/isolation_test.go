package effortauthority

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// A native subprocess with the issuer's UID can read mode0600 custody/state.
// This executable counterexample forbids claiming that chmod or localhost is
// a qualified boundary for an off-sandbox native profile. Contents are fixtures.
func TestSameUIDPrivateModesDoNotQualifyCustodyIsolation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "issuer-only")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "synthetic-ledger")
	if err := os.WriteFile(path, []byte("synthetic protected state"), 0600); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("cat", path).Output()
	if err != nil || string(out) != "synthetic protected state" {
		t.Fatalf("sameUID premise not exercised: %v", err)
	}
}

func TestCurrentSameUIDProcessCannotActivateFiniteIssuer(t *testing.T) {
	raw, e := os.ReadFile("/proc/" + strconv.Itoa(os.Getpid()) + "/exe")
	if e != nil {
		t.Fatal(e)
	}
	sum := sha256.Sum256(raw)
	process := ProtectedProcess{PID: os.Getpid(), ExecutableSHA256: hex.EncodeToString(sum[:])}
	dir := t.TempDir()
	path := filepath.Join(dir, "synthetic-private-state")
	if e = os.WriteFile(path, []byte("synthetic state"), 0600); e != nil {
		t.Fatal(e)
	}
	q := &NativeUIDIsolation{NativeUID: uint32(os.Getuid()), NativeProbePID: os.Getpid(), Services: map[string]ProtectedProcess{"issuer": process, "prompt-manager": process, "agent-manager": process}, ProtectedPaths: []string{dir, path}}
	if q.Qualify(context.Background()) == nil {
		t.Fatal("sameUID production topology incorrectly qualified")
	}
	if (*NativeUIDIsolation)(nil).Qualify(context.Background()) == nil {
		t.Fatal("absent physical evidence qualified")
	}
	process.ExecutableSHA256 = Digest("wrong build")
	if _, e = processUID(context.Background(), process); e == nil {
		t.Fatal("wrong executable identity qualified")
	}
}

func TestNativePrivilegePostureRefusesCapabilityAndPrivilegeEscalation(t *testing.T) {
	baseline := "CapInh:\t0000000000000000\nCapPrm:\t0000000000000000\nCapEff:\t0000000000000000\nCapBnd:\t0000000000000000\nCapAmb:\t0000000000000000\nNoNewPrivs:\t1\n"
	if nativePrivilegePosture([]byte(baseline)) != nil {
		t.Fatal("zero-capability no-new-privs parser refused")
	}
	for _, name := range []string{"CapInh", "CapPrm", "CapEff", "CapBnd", "CapAmb"} {
		t.Run(name, func(t *testing.T) {
			for _, capability := range []string{"0000000000000002", "0000000000000004", "0000000000000080", "0000000000080000"} {
				unsafe := strings.Replace(baseline, name+":\t0000000000000000", name+":\t"+capability, 1)
				if nativePrivilegePosture([]byte(unsafe)) == nil {
					t.Fatal("privileged probe accepted", name, capability)
				}
			}
		})
	}
	for _, unsafe := range []string{strings.Replace(baseline, "NoNewPrivs:\t1", "NoNewPrivs:\t0", 1), strings.Replace(baseline, "CapEff:\t0000000000000000\n", "", 1), baseline + "CapEff:\t0000000000000000\n", strings.Replace(baseline, "CapAmb:\t0000000000000000", "CapAmb:\tinvalid", 1)} {
		if nativePrivilegePosture([]byte(unsafe)) == nil {
			t.Fatal("missing/malformed/duplicate privilege posture accepted")
		}
	}
}
