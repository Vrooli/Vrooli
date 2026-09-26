package devicegraph

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseSMARTNVMeHealthLog(t *testing.T) {
	reading := parseSMART([]byte(nvmeSMARTFixture))
	if reading.Blocked {
		t.Fatalf("readable NVMe drive reported blocked: %s", reading.Reason)
	}
	if reading.Protocol != "nvme" {
		t.Errorf("protocol = %q, want nvme", reading.Protocol)
	}
	for name, got := range map[string]*int64{
		"wear":         reading.WearPercentUsed,
		"power on":     reading.PowerOnHours,
		"media errors": reading.MediaErrors,
		"spare":        reading.AvailableSpare,
	} {
		if got == nil {
			t.Errorf("%s was not parsed", name)
		}
	}
	if reading.HealthPassed == nil || !*reading.HealthPassed {
		t.Error("smart_status.passed was not carried through")
	}
}

func TestParseSMARTATAAttributeTable(t *testing.T) {
	reading := parseSMART([]byte(ataSMARTFixture))
	if reading.Blocked {
		t.Fatalf("readable ATA drive reported blocked: %s", reading.Reason)
	}
	if reading.Protocol != "ata" {
		t.Errorf("protocol = %q, want ata", reading.Protocol)
	}
	if reading.ReallocatedSectors == nil || *reading.ReallocatedSectors != 7 {
		t.Errorf("reallocated sectors = %v, want 7", reading.ReallocatedSectors)
	}
	if reading.PendingSectors == nil || *reading.PendingSectors != 2 {
		t.Errorf("pending sectors = %v, want 2", reading.PendingSectors)
	}
	if reading.UncorrectableSectors == nil || *reading.UncorrectableSectors != 1 {
		t.Errorf("uncorrectable sectors = %v, want 1", reading.UncorrectableSectors)
	}
}

// ATA wear attributes normalize to life remaining; the graph reports consumed
// endurance so the NVMe and ATA numbers can be compared directly.
func TestParseSMARTATAWearIsReportedAsConsumedEndurance(t *testing.T) {
	const fixture = `{
      "smartctl": {"exit_status": 0},
      "device": {"protocol": "ATA"},
      "ata_smart_attributes": {"table": [
        {"id": 231, "name": "SSD_Life_Left", "value": 88, "raw": {"value": 88}}
      ]}
    }`
	reading := parseSMART([]byte(fixture))
	if reading.WearPercentUsed == nil || *reading.WearPercentUsed != 12 {
		t.Fatalf("wear consumed = %v, want 12 (100 - 88 remaining)", reading.WearPercentUsed)
	}
}

func TestParseSMARTPermissionDeniedIsBlockedNotEmpty(t *testing.T) {
	reading := parseSMART([]byte(permissionDeniedSMARTFixture))
	if !reading.Blocked || !reading.PermissionDenied {
		t.Fatalf("permission failure was not recognised: %+v", reading)
	}
	if !strings.Contains(reading.Reason, "Permission denied") {
		t.Errorf("reason = %q, want the tool's own message", reading.Reason)
	}
	if reading.hasAttributes() {
		t.Error("a refused read must carry no attributes")
	}
}

func TestParseSMARTOpenFailureWithoutAPermissionMessage(t *testing.T) {
	const fixture = `{"smartctl": {"exit_status": 2, "messages": [{"string": "Unable to detect device type"}]}}`
	reading := parseSMART([]byte(fixture))
	if !reading.Blocked {
		t.Fatal("a device that could not be opened must be blocked")
	}
	if reading.PermissionDenied {
		t.Error("an open failure that is not an access failure must not claim permission denied")
	}
	if !strings.Contains(reading.Reason, "Unable to detect device type") {
		t.Errorf("reason = %q, want the tool's own message", reading.Reason)
	}
}

func TestParseSMARTNonJSONOutputIsBlocked(t *testing.T) {
	reading := parseSMART([]byte("smartctl: command not found"))
	if !reading.Blocked || reading.Reason == "" {
		t.Fatalf("unparseable output must be blocked with a reason: %+v", reading)
	}
}

func TestReadSMARTWithoutAToolIsNotPresent(t *testing.T) {
	env := Env{
		LookPath: func(string) (string, error) { return "", errors.New("not found") },
		Run:      func(context.Context, time.Duration, string, ...string) ([]byte, error) { return nil, nil },
	}.normalized()
	reading := readSMART(context.Background(), env, "/dev/sda")
	if reading.ToolPresent {
		t.Fatal("a missing reader must not report as present")
	}
	if !strings.Contains(reading.Reason, smartToolName) {
		t.Errorf("reason = %q, want it to name the missing reader", reading.Reason)
	}
}

// A tool that runs but answers nothing is a blocked read, not a healthy drive
// with no attributes.
func TestReadSMARTEmptyOutputIsBlocked(t *testing.T) {
	env := Env{
		LookPath: func(name string) (string, error) { return "/usr/sbin/" + name, nil },
		Run: func(context.Context, time.Duration, string, ...string) ([]byte, error) {
			return []byte(""), errors.New("exit status 4")
		},
	}.normalized()
	reading := readSMART(context.Background(), env, "/dev/sda")
	if !reading.ToolPresent || !reading.Blocked {
		t.Fatalf("empty output from a present tool must be blocked: %+v", reading)
	}
}
