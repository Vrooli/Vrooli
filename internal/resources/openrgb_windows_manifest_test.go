package resources

import (
	"testing"
)

func TestOpenRGBWindowsStartupRequiresNoninteractiveAsInvoker(t *testing.T) {
	base := func(level, ui string) []byte {
		return []byte(`<assembly xmlns="urn:schemas-microsoft-com:asm.v1"><trustInfo xmlns="urn:schemas-microsoft-com:asm.v3"><security><requestedPrivileges><requestedExecutionLevel level="` + level + `" uiAccess="` + ui + `"/></requestedPrivileges></security></trustInfo></assembly>`)
	}
	if err := verifyOpenRGBAsInvokerXML(base("asInvoker", "false")); err != nil {
		t.Fatal(err)
	}
	for _, tc := range [][2]string{{"requireAdministrator", "false"}, {"highestAvailable", "false"}, {"asInvoker", "true"}} {
		if err := verifyOpenRGBAsInvokerXML(base(tc[0], tc[1])); err == nil {
			t.Fatalf("unsafe startup policy accepted: %v", tc)
		}
	}
	if err := verifyOpenRGBAsInvokerXML([]byte(`<assembly/>`)); err == nil {
		t.Fatal("missing process execution policy accepted")
	}
}
