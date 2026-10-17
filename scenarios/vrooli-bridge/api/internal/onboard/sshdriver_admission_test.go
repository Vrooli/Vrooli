package onboard

import (
	"strings"
	"testing"
)

func TestAdmissionProbeUsesCandidateRouteSourceNotSSHConnection(t *testing.T) {
	command, err := admissionProbeCommand("http://192.168.1.173:18767", "VBADMISSION=")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(command, "ip route get") || !strings.Contains(command, "route -n get") || !strings.Contains(command, "ipconfig getifaddr") || !strings.Contains(command, "host='192.168.1.173'") {
		t.Fatalf("probe does not derive route source: %s", command)
	}
	if strings.Contains(command, "SSH_CONNECTION") {
		t.Fatalf("probe used SSH_CONNECTION: %s", command)
	}
}
