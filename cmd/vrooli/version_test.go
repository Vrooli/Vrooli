package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/statecontract"
)

func stampPersistenceCodecs(t *testing.T) {
	t.Helper()
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	stamp, err := buildinfo.PersistentStateCodecStamp(root)
	if err != nil {
		t.Fatal(err)
	}
	old := statecontract.CodecStamp
	t.Cleanup(func() { statecontract.CodecStamp = old })
	statecontract.CodecStamp = stamp
}

func TestStandaloneVersionRetainsActualBuildWithoutAvailableSource(t *testing.T) {
	stampPersistenceCodecs(t)
	oldRoot, oldFingerprint := resolveSourceRootFn, buildinfo.Fingerprint
	t.Cleanup(func() {
		resolveSourceRootFn, buildinfo.Fingerprint = oldRoot, oldFingerprint
	})
	resolveSourceRootFn = func() (string, error) { return "", errors.New("source tree unavailable") }
	buildinfo.Fingerprint = "actual-executable-build"
	if len(runtimeapp.NativePersistentStateContract()) != 64 {
		t.Fatal("compiled native persistence contract was not qualified")
	}
	var out, stderr bytes.Buffer
	handled, code := writeStandaloneVersion([]string{"version", "--json"}, &out, &stderr)
	if !handled || code != 0 {
		t.Fatalf("identity query required host initialization: %t %d %s", handled, code, stderr.String())
	}
	var identity map[string]any
	if err := json.Unmarshal(out.Bytes(), &identity); err != nil {
		t.Fatal(err)
	}
	if identity["build_identity"] != "actual-executable-build" || identity["root"] != "" || identity["persistent_state_contract_sha256"] != runtimeapp.NativePersistentStateContract() {
		t.Fatalf("identity inferred from source: %v", identity)
	}
	out.Reset()
	handled, _ = writeStandaloneVersion([]string{"scenario", "start", "web-console"}, &out, &stderr)
	if handled || out.Len() != 0 {
		t.Fatal("artifact query path intercepted a lifecycle command")
	}
}

func TestStandaloneStateManifestIsTheVerifiedContract(t *testing.T) {
	stampPersistenceCodecs(t)
	var out, stderr bytes.Buffer
	handled, code := writeStandaloneVersion([]string{"version", "--state-manifest"}, &out, &stderr)
	if !handled || code != 0 {
		t.Fatalf("state manifest query failed: %t %d %s", handled, code, stderr.String())
	}
	var manifest statecontract.Manifest
	decoder := json.NewDecoder(&out)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatal(err)
	}
	if err := statecontract.Verify(manifest); err != nil || manifest.Contract != runtimeapp.NativePersistentStateContract() {
		t.Fatalf("state manifest is not the reported contract: %v", err)
	}
	statecontract.CodecStamp = ""
	out.Reset()
	stderr.Reset()
	if handled, code := writeStandaloneVersion([]string{"version", "--state-manifest"}, &out, &stderr); !handled || code == 0 || out.Len() != 0 || stderr.Len() == 0 {
		t.Fatal("an unqualified executable printed a state manifest")
	}
}
