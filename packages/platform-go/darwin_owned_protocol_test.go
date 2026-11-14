package platform

import (
	"strings"
	"testing"
)

func TestDarwinOwnedDefinitionPreservesFixedInvocation(t *testing.T) {
	s := OperationSpec{Path: "/sealed/OpenRGB", Dir: "/private/config", NativeAccount: "uid:501", Args: []string{"--config", "a&b<c"}, Env: []string{"XDG_CONFIG_HOME=/private/config", "APP_NAME=<rgb>"}, Limits: OperationLimits{MemoryMaxBytes: 100, TasksMax: 8, CPUWeight: 100}}
	i := OperationIntent{Owner: "openrgb", Kind: ownedServiceKind}
	raw := darwinOwnedPlist(i, s)
	for _, fact := range []string{"<string>a&amp;b&lt;c</string>", "<string>&lt;rgb&gt;</string>", "<key>KeepAlive</key><false/>", "<key>AbandonProcessGroup</key><false/>"} {
		if !strings.Contains(raw, fact) {
			t.Fatalf("missing %q in %s", fact, raw)
		}
	}
	if strings.Contains(raw, "/bin/sh") || strings.Contains(raw, "gui/") {
		t.Fatal("definition must use fixed program and exact user-domain admission")
	}
}
