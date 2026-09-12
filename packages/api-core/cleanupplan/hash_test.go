package cleanupplan

import "testing"

func TestHashResolvedArtifactsIgnoresRequestMetadata(t *testing.T) {
	remove := []map[string]string{{"path": "/managed/a"}}
	keep := []map[string]string{{"path": "/shared/b", "reason": "pre-existing"}}
	cannot := []map[string]string{}
	first := HashResolvedArtifacts(remove, keep, cannot)
	second := HashResolvedArtifacts(remove, keep, cannot)
	if first == "" || first != second {
		t.Fatalf("hashes are not stable: %q %q", first, second)
	}
}
