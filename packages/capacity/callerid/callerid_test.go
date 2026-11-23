package callerid

import "testing"

func TestLabelNamesTheStableScopeAndExecutable(t *testing.T) {
	cases := []struct{ cgroup, exe, want string }{
		{
			"/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-services.slice/vrooli-service-ai-gateway-start-api-840a86ab4d214161d1f45c69cb87e44a.scope",
			"/home/u/Vrooli/scenarios/ai-gateway/api/ai-gateway-api",
			"scenario:ai-gateway-start-api/ai-gateway-api",
		},
		{
			"/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-services.slice/vrooli-service-resource-ollama-companion-capacity-sync-0123456789abcdef.scope",
			"/home/u/.vrooli/bin/resource-ollama (deleted)",
			"resource:ollama-companion-capacity-sync/resource-ollama",
		},
		{
			"/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/vrooli-agent-claude-code-c55986207bbb9e85282173873338c52f.scope",
			"/usr/bin/opencode",
			"agent:claude-code/opencode",
		},
		{
			"/user.slice/user-1000.slice/user@1000.service/app.slice/vte-spawn-e9609de2-d2d0-4004-981d-1e0479f99b23.scope",
			"/usr/bin/curl",
			"host:vte-spawn/curl",
		},
		// Agent Manager's replacement API scope carries a second generation digest.
		{
			"/x/vrooli-service-agent-manager-start-api-00112233445566778899aabbccddeeff-ffeeddccbbaa99887766554433221100.scope",
			"agent-manager-api",
			"scenario:agent-manager-start-api/agent-manager-api",
		},
		{"", "", "host:unknown/unknown"},
	}
	for _, tc := range cases {
		if got := Label(tc.cgroup, tc.exe); got != tc.want {
			t.Errorf("Label(%q, %q) = %q, want %q", tc.cgroup, tc.exe, got, tc.want)
		}
	}
	// Hex-letter words are names, not instance digests.
	if got := trimInstanceSuffix("vrooli-face-bead"); got != "vrooli-face-bead" {
		t.Errorf("trimInstanceSuffix kept %q", got)
	}
}

func TestParseSplitsALabelAndRefusesEverythingElse(t *testing.T) {
	kind, unit, exe, ok := Parse("scenario:agent-manager-start-api/agent-manager-api")
	if !ok || kind != KindScenario || unit != "agent-manager-start-api" || exe != "agent-manager-api" {
		t.Fatalf("Parse = %q %q %q %v", kind, unit, exe, ok)
	}
	for _, bad := range []string{Unknown, "", "scenario:", "scenario:unit", ":unit/exe", "scenario:/exe"} {
		if _, _, _, ok := Parse(bad); ok {
			t.Errorf("Parse(%q) accepted a non-label", bad)
		}
	}
}
