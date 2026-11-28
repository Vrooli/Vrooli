package execution

import "testing"

func TestHostEvidenceFromEnvironmentUsesStableRelevantFacts(t *testing.T) {
	env := map[string]string{
		"VROOLI_BRIDGE_NODE_ID":   "  ",
		"VROOLI_NODE_ID":          " node-secondary ",
		"VROOLI_HOST_NODE":        "node-tertiary",
		"VROOLI_INIT_SYSTEM":      "systemd",
		"XDG_SESSION_TYPE":        "wayland",
		"VROOLI_DISPLAY_ATTACHED": "1",
		"VROOLI_HEADLESS":         "0",
		"VROOLI_WSL":              "true",
	}
	lookup := func(name string) string { return env[name] }

	first := hostEvidenceFromEnvironment("linux", "amd64", lookup)
	second := hostEvidenceFromEnvironment("linux", "amd64", lookup)
	if first.Node != "node-secondary" {
		t.Fatalf("node = %q, want the first non-empty trimmed node ID", first.Node)
	}
	if first.FactDigest == "" || first.FactDigest != second.FactDigest {
		t.Fatalf("repeated identical host facts have unstable digest: %q vs %q", first.FactDigest, second.FactDigest)
	}

	changed := make(map[string]string, len(env))
	for name, value := range env {
		changed[name] = value
	}
	changed["XDG_SESSION_TYPE"] = "x11"
	changedEvidence := hostEvidenceFromEnvironment("linux", "amd64", func(name string) string { return changed[name] })
	if changedEvidence.FactDigest == first.FactDigest {
		t.Fatal("changing an outcome-relevant host fact did not change the digest")
	}

	changed["VROOLI_HOST_NODE"] = "unused-lower-priority-node"
	unchangedEvidence := hostEvidenceFromEnvironment("linux", "amd64", func(name string) string { return changed[name] })
	if unchangedEvidence.Node != first.Node {
		t.Fatalf("lower-priority node changed selection: %q", unchangedEvidence.Node)
	}
	if unchangedEvidence.FactDigest != changedEvidence.FactDigest {
		t.Fatalf("unused lower-priority node changed digest: %q vs %q", changedEvidence.FactDigest, unchangedEvidence.FactDigest)
	}
	if first.OS != "linux" || first.Arch != "amd64" {
		t.Fatalf("runtime identity = %s/%s, want linux/amd64", first.OS, first.Arch)
	}
}
