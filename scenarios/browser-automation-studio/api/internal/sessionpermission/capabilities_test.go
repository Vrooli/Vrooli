package sessionpermission

import (
	"reflect"
	"testing"
)

func TestNegotiateCapabilitiesRequiresLiveQualificationAndReportsLimits(t *testing.T) {
	required := []Capability{CapabilityObserve, CapabilityControl, CapabilityPreserveUserState, CapabilityAccessibilityTree}
	reports := []BackendReport{
		{Backend: BackendManaged, Qualified: true, EvidenceID: "managed-receipt-1", Capabilities: []Capability{CapabilityObserve, CapabilityControl, CapabilityAccessibilityTree}},
		{Backend: BackendUserAttached, Qualified: false, Capabilities: []Capability{CapabilityObserve, CapabilityControl}, Unavailable: map[Capability]string{CapabilityPreserveUserState: "bridge not installed"}},
	}
	rows := NegotiateCapabilities(reports, required)
	if rows[0].Accepted || !reflect.DeepEqual(rows[0].Missing, []Capability{CapabilityPreserveUserState}) {
		t.Fatalf("managed mode should report missing preservation: %#v", rows[0])
	}
	if rows[0].Reasons[CapabilityPreserveUserState] != "backend does not provide this capability" {
		t.Fatalf("unexpected managed limitation: %#v", rows[0].Reasons)
	}
	if rows[1].Accepted || rows[1].Qualified || rows[1].Reasons[CapabilityPreserveUserState] != "bridge not installed" {
		t.Fatalf("unqualified user attachment must remain unavailable: %#v", rows[1])
	}
}

func TestAccessibilityTreeRequiresLiveEvidenceAndCapability(t *testing.T) {
	required := []Capability{CapabilityObserve, CapabilityAccessibilityTree}
	for name, report := range map[string]BackendReport{
		"no report":         {},
		"no evidence id":    {Backend: BackendManaged, Qualified: true, Capabilities: []Capability{CapabilityObserve, CapabilityAccessibilityTree}},
		"tree not reported": {Backend: BackendManaged, Qualified: true, EvidenceID: "observe-only", Capabilities: []Capability{CapabilityObserve}},
	} {
		t.Run(name, func(t *testing.T) {
			rows := NegotiateCapabilities([]BackendReport{report}, required)
			if len(rows) != 1 || rows[0].Accepted {
				t.Fatalf("accessibility tree accepted without live qualified evidence: %#v", rows)
			}
		})
	}
	rows := NegotiateCapabilities([]BackendReport{{Backend: BackendManaged, Qualified: true, EvidenceID: "a11y-tree-live-1", Capabilities: []Capability{CapabilityObserve, CapabilityAccessibilityTree}}}, required)
	if len(rows) != 1 || !rows[0].Accepted || !rows[0].Qualified {
		t.Fatalf("live-qualified accessibility tree rejected: %#v", rows)
	}
}

func TestNegotiateCapabilitiesRejectsQualifiedClaimWithoutEvidenceID(t *testing.T) {
	rows := NegotiateCapabilities([]BackendReport{{Backend: BackendInstalled, Qualified: true, Capabilities: []Capability{CapabilityObserve}}}, []Capability{CapabilityObserve})
	if len(rows) != 1 || rows[0].Accepted || rows[0].Qualified || len(rows[0].Missing) != 1 || len(rows[0].Available) != 0 {
		t.Fatalf("source-only qualification accepted: %#v", rows)
	}
}

func TestUnqualifiedCapabilityClaimsAreNotShownAsAvailable(t *testing.T) {
	rows := NegotiateCapabilities([]BackendReport{{
		Backend: BackendManaged, Qualified: false, EvidenceID: "stale-evidence",
		Capabilities: []Capability{CapabilityObserve, CapabilityControl},
	}}, []Capability{CapabilityObserve})
	if len(rows) != 1 || rows[0].Qualified || rows[0].Accepted || len(rows[0].Available) != 0 || len(rows[0].Missing) != 1 {
		t.Fatalf("unqualified capability claims leaked as available: %#v", rows)
	}
}

func TestUnqualifiedCapabilityReportNeverAcceptedWithoutRequirements(t *testing.T) {
	rows := NegotiateCapabilities([]BackendReport{{Backend: BackendManaged, Qualified: true}}, nil)
	if len(rows) != 1 || rows[0].Qualified || rows[0].Accepted {
		t.Fatalf("report without evidence accepted: %#v", rows)
	}
}
