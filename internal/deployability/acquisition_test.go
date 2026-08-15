package deployability

import "testing"

func TestValidateMacOSAcquisition(t *testing.T) {
	if err := ValidateMacOSAcquisition(ToolAcquisitionDeclaration{Platforms: []HostOS{HostOSMacOS}}); err == nil {
		t.Fatal("expected a missing macOS acquisition path to fail")
	}
	if err := ValidateMacOSAcquisition(ToolAcquisitionDeclaration{}); err == nil {
		t.Fatal("an omitted platform declaration must still require an acquisition path")
	}
	for _, declaration := range []ToolAcquisitionDeclaration{
		{Platforms: []HostOS{HostOSMacOS}, Brew: "example"},
		{Platforms: []HostOS{HostOSMacOS}, Source: "release"},
		{Platforms: []HostOS{HostOSMacOS}, Handler: "native-handler"},
		{Platforms: []HostOS{HostOSMacOS}, Manual: true},
	} {
		if err := ValidateMacOSAcquisition(declaration); err != nil {
			t.Fatalf("declared acquisition path rejected: %v", err)
		}
	}
}
