package hosthardening

import (
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func sysctlValue(t *testing.T, pol policy, name string) int {
	t.Helper()
	for _, setting := range managedSysctls(pol) {
		if setting.Name == name {
			return setting.Value
		}
	}
	t.Fatalf("%s is not a managed sysctl", name)
	return -1
}

// The handler's fallback defaults and the manifest's declared defaults must
// agree. They are written in two places — Go for the no-config path, JSON for
// the operator-facing schema — and a drift between them would mean setup and
// the handler disagree about what "unconfigured" means.

// A soft lockup is a stall, not proof of corruption, and on a saturated host it
// is a routine event. The default must not reboot the machine for one.
func TestSoftlockupDefaultsToWarning(t *testing.T) {
	if got := sysctlValue(t, resolvePolicy(nil), "kernel.softlockup_panic"); got != 0 {
		t.Fatalf("kernel.softlockup_panic = %d by default, want 0", got)
	}
}

func TestPolicyDrivesSysctlValues(t *testing.T) {
	for _, tc := range []struct {
		name             string
		config           map[string]any
		wantOopsPanic    int
		wantSoftlockup   int
		wantHungTaskSecs int
	}{
		{
			name:             "defaults",
			config:           nil,
			wantOopsPanic:    1,
			wantSoftlockup:   0,
			wantHungTaskSecs: 120,
		},
		{
			name:             "workstation keeps running through a kernel fault",
			config:           map[string]any{"oops_policy": oopsPolicyLogAndContinue},
			wantOopsPanic:    0,
			wantSoftlockup:   0,
			wantHungTaskSecs: 120,
		},
		{
			name:             "fleet node fails fast on both",
			config:           map[string]any{"oops_policy": oopsPolicyPanicAndDump, "softlockup_policy": softlockupPolicyPanic},
			wantOopsPanic:    1,
			wantSoftlockup:   1,
			wantHungTaskSecs: 120,
		},
		{
			name:             "hung-task timeout from JSON number",
			config:           map[string]any{"hung_task_timeout_secs": float64(240)},
			wantOopsPanic:    1,
			wantSoftlockup:   0,
			wantHungTaskSecs: 240,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pol := resolvePolicy(tc.config)
			if got := sysctlValue(t, pol, "kernel.panic_on_oops"); got != tc.wantOopsPanic {
				t.Errorf("panic_on_oops = %d, want %d", got, tc.wantOopsPanic)
			}
			if got := sysctlValue(t, pol, "kernel.softlockup_panic"); got != tc.wantSoftlockup {
				t.Errorf("softlockup_panic = %d, want %d", got, tc.wantSoftlockup)
			}
			if got := sysctlValue(t, pol, "kernel.hung_task_timeout_secs"); got != tc.wantHungTaskSecs {
				t.Errorf("hung_task_timeout_secs = %d, want %d", got, tc.wantHungTaskSecs)
			}
		})
	}
}

// Settings that are not operator-configurable must stay put whatever the
// policy, so a workstation profile cannot accidentally disable sysrq or the
// post-panic reboot.
func TestNonConfigurableSysctlsAreStable(t *testing.T) {
	for _, pol := range []policy{
		resolvePolicy(nil),
		resolvePolicy(map[string]any{"oops_policy": oopsPolicyLogAndContinue, "softlockup_policy": softlockupPolicyWarn}),
	} {
		if got := sysctlValue(t, pol, "kernel.sysrq"); got != 1 {
			t.Errorf("kernel.sysrq = %d, want 1", got)
		}
		if got := sysctlValue(t, pol, "kernel.unknown_nmi_panic"); got != 1 {
			t.Errorf("kernel.unknown_nmi_panic = %d, want 1", got)
		}
		if got := sysctlValue(t, pol, "kernel.panic"); got != 10 {
			t.Errorf("kernel.panic = %d, want 10", got)
		}
	}
}

// panic-on-oops without a loaded crash kernel produces reboots with no vmcore —
// strictly worse than leaving the kernel to limp on. The dependency was
// documented for months and enforced nowhere; Inspect must now refuse.
func TestInspectRefusesPanicPolicyWithoutArmedKdump(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	inner := readSysctlsAtTarget(defaultPolicy())
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == kexecCrashLoadedPath {
			return []byte("0\n"), nil
		}
		return inner(path)
	}

	status := newTestHandler().Inspect(linuxHost(), linuxReq())

	if status.Applied {
		t.Fatal("must not report applied when the crash kernel is not loaded")
	}
	if status.BlockingReason != hostreqkit.BlockingPrerequisiteMissing {
		t.Fatalf("BlockingReason = %q, want %q", status.BlockingReason, hostreqkit.BlockingPrerequisiteMissing)
	}
	if !strings.Contains(strings.Join(status.Notes, " | "), "log-and-continue") {
		t.Errorf("note should offer the alternative policy; got %v", status.Notes)
	}
}

// An absent probe file is not an armed crash kernel. A host that cannot answer
// must be treated as unarmed rather than assumed ready.
func TestInspectTreatsMissingProbeAsUnarmed(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	hostreqkit.ReadFileFn = func(string) ([]byte, error) { return nil, os.ErrNotExist }

	status := newTestHandler().Inspect(linuxHost(), linuxReq())

	if status.BlockingReason != hostreqkit.BlockingPrerequisiteMissing {
		t.Fatalf("BlockingReason = %q, want %q", status.BlockingReason, hostreqkit.BlockingPrerequisiteMissing)
	}
}

// log-and-continue has no crash-kernel dependency, so it must apply on a host
// with no kdump at all.
func TestLogAndContinueNeedsNoCrashKernel(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	pol := resolvePolicy(map[string]any{"oops_policy": oopsPolicyLogAndContinue})
	inner := readSysctlsAtTarget(pol)
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == kexecCrashLoadedPath {
			return nil, os.ErrNotExist
		}
		return inner(path)
	}

	req := linuxReq()
	req.Config = map[string]any{"oops_policy": oopsPolicyLogAndContinue}
	status := newTestHandler().Inspect(linuxHost(), req)

	if status.BlockingReason == hostreqkit.BlockingPrerequisiteMissing {
		t.Fatalf("log-and-continue must not require a crash kernel; notes: %v", status.Notes)
	}
	if !status.Applied {
		t.Fatalf("expected already-applied under log-and-continue; notes: %v", status.Notes)
	}
}

// The written drop-in must record which policy produced it, so an operator
// reading /etc/sysctl.d can tell a deliberate setting from a stale one.
func TestSysctlContentRecordsThePolicy(t *testing.T) {
	content := buildSysctlContent(resolvePolicy(map[string]any{
		"oops_policy":       oopsPolicyLogAndContinue,
		"softlockup_policy": softlockupPolicyWarn,
	}))
	if !strings.Contains(content, "oops_policy=log-and-continue") {
		t.Errorf("content should record the policy; got:\n%s", content)
	}
	if !strings.Contains(content, "kernel.panic_on_oops = 0") {
		t.Errorf("content should reflect the policy; got:\n%s", content)
	}
}
