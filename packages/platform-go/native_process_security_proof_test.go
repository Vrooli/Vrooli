package platform

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWindowsOperatorSecurityProofBindsActualUserAndRejectsElevation(t *testing.T) {
	first, e := windowsOperatorSecurityFactsProof("sid:S-1-5-21-100-200-300-1001", false)
	if e != nil || !validNativeBrokerDigest(first) {
		t.Fatal(first, e)
	}
	same, e := windowsOperatorSecurityFactsProof("sid:S-1-5-21-100-200-300-1001", false)
	if e != nil || first != same {
		t.Fatal("same actual user facts changed proof")
	}
	next, e := windowsOperatorSecurityFactsProof("sid:S-1-5-21-100-200-300-1002", false)
	if e != nil || first == next {
		t.Fatal("different user matched actual proof")
	}
	for _, principal := range []string{"", "LocalSystem", "sid:S-1-5-18", "sid:S-1-5-80-100", "sid:S-1-5-32-544"} {
		if _, e = windowsOperatorSecurityFactsProof(principal, false); e == nil {
			t.Fatal("broad principal accepted", principal)
		}
	}
	if _, e = windowsOperatorSecurityFactsProof("sid:S-1-5-21-100-200-300-1001", true); e == nil {
		t.Fatal("elevated token accepted")
	}
}
func TestWindowsChildClaimRetainsActualUserProof(t *testing.T) {
	child := &OwnedImageChild{PID: 42, Incarnation: "windows:42:1000", ImageID: "windows-file:test", ImageHandle: 1, NativeAccount: "sid:S-1-5-21-100-200-300-1001", ContainmentID: "Global\\VrooliOwned-example", SecurityProof: strings.Repeat("a", 64)}
	id := child.Identity()
	if id.SecurityProof != child.SecurityProof {
		t.Fatal("pre-resume identity lost observed actual security proof")
	}
	b, e := json.Marshal(id)
	if e != nil {
		t.Fatal(e)
	}
	var got OwnedImageChildIdentity
	if e = json.Unmarshal(b, &got); e != nil {
		t.Fatal(e)
	}
	if got != id {
		t.Fatal("persisted child facts changed")
	}
}
