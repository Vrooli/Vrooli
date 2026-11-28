package privilegebroker

import (
	"strings"
	"testing"
)

func TestValidateSystemRebootAcceptsBoundedDelays(t *testing.T) {
	for _, delay := range []int{0, 1, 360, rebootMaxDelayMinutes} {
		req := Request{
			Version: ProtocolVersion, RequestID: "reboot-1", Action: ActionSystemReboot,
			Reboot: &RebootSubject{DelayMinutes: delay, DecisionID: "decision-abc:123"},
		}
		if err := Validate(req); err != nil {
			t.Fatalf("delay %d rejected: %v", delay, err)
		}
	}
}

func TestValidateSystemRebootRejectsOutOfRangeAndBadDecisionID(t *testing.T) {
	base := Request{Version: ProtocolVersion, RequestID: "reboot-2", Action: ActionSystemReboot}
	for _, delay := range []int{-1, rebootMaxDelayMinutes + 1} {
		req := base
		req.Reboot = &RebootSubject{DelayMinutes: delay}
		if Validate(req) == nil {
			t.Fatalf("delay %d accepted", delay)
		}
	}
	for _, id := range []string{"has space", "path/../x", "line\nbreak", strings.Repeat("a", rebootDecisionIDMax+1)} {
		req := base
		req.Reboot = &RebootSubject{DecisionID: id}
		if Validate(req) == nil {
			t.Fatalf("decision id %q accepted", id)
		}
	}
}

func TestValidateSystemRebootRequiresItsOwnSubjectOnly(t *testing.T) {
	noSubject := Request{Version: ProtocolVersion, RequestID: "reboot-3", Action: ActionSystemReboot}
	if Validate(noSubject) == nil {
		t.Fatal("reboot accepted without a subject")
	}
	withService := noSubject
	withService.Reboot = &RebootSubject{}
	withService.Service = &ServiceSubject{Unit: "rsyslog", Operation: "restart"}
	if Validate(withService) == nil {
		t.Fatal("reboot accepted a service subject")
	}
	straySubject := Request{
		Version: ProtocolVersion, RequestID: "reboot-4", Action: ActionCapabilitiesInspect,
		Reboot: &RebootSubject{DelayMinutes: 5},
	}
	if Validate(straySubject) == nil {
		t.Fatal("a reboot subject rode along on another action")
	}
}

func TestValidateSystemRebootCancelTakesNoSubject(t *testing.T) {
	req := Request{Version: ProtocolVersion, RequestID: "cancel-1", Action: ActionSystemRebootCancel}
	if err := Validate(req); err != nil {
		t.Fatalf("subjectless cancel rejected: %v", err)
	}
	req.Reboot = &RebootSubject{DelayMinutes: 5}
	if Validate(req) == nil {
		t.Fatal("cancel accepted a reboot subject")
	}
}

func TestValidateKernelMemoryStacksAcceptsOnlyClosedSorts(t *testing.T) {
	for _, sort := range []string{"", KernelMemorySortSize, KernelMemorySortCount} {
		req := Request{
			Version: ProtocolVersion, RequestID: "stacks-1", Action: ActionKernelMemoryStackSample,
			KernelMemory: &KernelMemorySubject{Sort: sort},
		}
		if err := Validate(req); err != nil {
			t.Fatalf("sort %q rejected: %v", sort, err)
		}
	}
	req := Request{
		Version: ProtocolVersion, RequestID: "stacks-2", Action: ActionKernelMemoryStackSample,
		KernelMemory: &KernelMemorySubject{Sort: "addresses"},
	}
	if Validate(req) == nil {
		t.Fatal("unknown sort accepted")
	}
	req = Request{Version: ProtocolVersion, RequestID: "stacks-3", Action: ActionKernelMemoryStackSample}
	req.Reboot = &RebootSubject{}
	if Validate(req) == nil {
		t.Fatal("a reboot subject rode along on a snapshot request")
	}
}
