package destinationreadiness

import (
	"strings"
	"testing"
)

// Repairability is separate from day-to-day suitability. ext4 is both; NTFS is
// usable but not repairable here, and saying so is what stops a futile repair.
func TestFilesystemRepairVerdict(t *testing.T) {
	ext4 := checkFilesystemRepair("ext4")
	if ext4.Severity != SeverityPass {
		t.Fatalf("ext4 repair severity = %q, want pass: %+v", ext4.Severity, ext4)
	}

	ntfs := checkFilesystemRepair("ntfs")
	if ntfs.Severity != SeverityWarning {
		t.Fatalf("ntfs repair severity = %q, want warning: %+v", ntfs.Severity, ntfs)
	}
	if !strings.Contains(ntfs.Message, "cannot be repaired on this host") {
		t.Fatalf("ntfs repair message does not state the ceiling: %q", ntfs.Message)
	}
	if strings.TrimSpace(ntfs.NextAction) == "" {
		t.Fatal("an unrepairable filesystem must name a remedy")
	}

	unknown := checkFilesystemRepair("mysteryfs")
	if unknown.Severity != SeverityUnknown {
		t.Fatalf("unprofiled repair severity = %q, want unknown: %+v", unknown.Severity, unknown)
	}
}

// exFAT is usable and repairable on Linux, so it must not be conflated with
// NTFS just because both lack a kernel cleanliness signal.
func TestExFATIsRepairableEvenWithoutAStateSignal(t *testing.T) {
	repair := checkFilesystemRepair("exfat")
	if repair.Severity != SeverityPass {
		t.Fatalf("exfat repair severity = %q, want pass: %+v", repair.Severity, repair)
	}
}
