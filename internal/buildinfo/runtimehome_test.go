package buildinfo

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func mustRuntimeHomeBuildFlag(t *testing.T, root string) string {
	t.Helper()
	flag, err := RuntimeHomeBuildFlag(root)
	if err != nil {
		t.Fatal(err)
	}
	return flag
}

func TestGovernedRuntimeHomeBuildMetadataUsesCanonicalAuthority(t *testing.T) {
	root := t.TempDir()
	repocontracttest.WriteRepoContract(t, root, "scenarios")
	old := RuntimeHomeSnapshotBase64
	t.Cleanup(func() { RuntimeHomeSnapshotBase64 = old })
	flag := mustRuntimeHomeBuildFlag(t, root)
	const prefix = "-X github.com/vrooli/vrooli/internal/buildinfo.RuntimeHomeSnapshotBase64="
	if !strings.HasPrefix(flag, prefix) {
		t.Fatal("runtime authority not carried through declared linker metadata")
	}
	RuntimeHomeSnapshotBase64 = strings.TrimPrefix(flag, prefix)
	digest, err := RuntimeHomeSnapshotDigest()
	if err != nil || len(digest) != 64 {
		t.Fatalf("compiled authority has no digest: %s %v", digest, err)
	}
	c, err := repocontract.LoadDefault(root)
	if err != nil {
		t.Fatal(err)
	}
	want, err := c.RuntimeHomeSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	got, err := base64.StdEncoding.DecodeString(RuntimeHomeSnapshotBase64)
	if err != nil || string(got) != string(want) {
		t.Fatal("build embedded a different runtime-home contract")
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if after, err := RuntimeHomeSnapshotDigest(); err != nil || after != digest {
		t.Fatalf("compiled identity depended on source: %s %v", after, err)
	}
}

func TestCompiledRuntimeHomeRejectsMissingInvalidAndOversizedMetadata(t *testing.T) {
	old := RuntimeHomeSnapshotBase64
	t.Cleanup(func() { RuntimeHomeSnapshotBase64 = old })
	for _, value := range []string{"", "not-base64", base64.StdEncoding.EncodeToString([]byte("{}")), strings.Repeat("a", 100000)} {
		RuntimeHomeSnapshotBase64 = value
		if _, err := RuntimeHomeSnapshotDigest(); err == nil {
			t.Fatal("invalid compiled metadata returned an identity")
		}
	}
	if _, err := RuntimeHomeBuildFlag(t.TempDir()); err == nil {
		t.Fatal("native build silently guessed a missing source contract")
	}
}

func TestLayoutOnlySourceChangeInvalidatesArtifactFreshness(t *testing.T) {
	root := t.TempDir()
	repocontracttest.WriteRepoContract(t, root, "scenarios")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	before, err := ComputeSourceFingerprintReport(root, FingerprintOptions{}, "cmd/vrooli")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".vrooli", "repo-contract.json")
	var document map[string]any
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	document["runtime_home"].(map[string]any)["entries"].(map[string]any)["runtime_db"].(map[string]any)["path"] = "state/registry-v2.db"
	data, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := ComputeSourceFingerprintReport(root, FingerprintOptions{}, "cmd/vrooli")
	if err != nil {
		t.Fatal(err)
	}
	if before.Fingerprint == after.Fingerprint || before.RuntimeHomeSnapshotDigest == after.RuntimeHomeSnapshotDigest {
		t.Fatal("layout-only change did not invalidate native artifact identity")
	}
	if before.MatchedFiles != after.MatchedFiles {
		t.Fatal("runtime-home metadata was counted as a Go source file")
	}
}
