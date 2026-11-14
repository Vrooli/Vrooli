package executionevidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/vrooli/freshness-go/treedigest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func strictFixture(t *testing.T) (treedigest.InputManifest, StrictExecutionBinding) {
	t.Helper()
	root := t.TempDir()
	dep := t.TempDir()
	for p, v := range map[string]string{filepath.Join(root, "source.go"): "package fixture", filepath.Join(dep, "dependency.txt"): "nonsecret dependency"} {
		if err := os.WriteFile(p, []byte(v), 0600); err != nil {
			t.Fatal(err)
		}
	}
	m, err := treedigest.BuildInputManifestContext(context.Background(), treedigest.ManifestRequest{Primary: treedigest.RootSpec{Name: "product", Path: root, Files: []string{"source.go"}}, Dependencies: []treedigest.RootSpec{{Name: "dependency", Path: dep, Files: []string{"dependency.txt"}}}, Configuration: map[string]string{"mode": "bounded"}, Toolchain: map[string]string{"go": "fixture-toolchain"}})
	if err != nil {
		t.Fatal(err)
	}
	d := "sha256:" + strings.Repeat("a", 64)
	return m, StrictExecutionBinding{RunID: "fixture-run", Target: "fixture-scenario", OwnerBindingDigest: d, ExecutionEpoch: 7, ConfigurationDigest: d, DescriptorDigest: d, PhaseSetDigest: d}
}
func digestStrictBytes(b []byte) string {
	s := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(s[:])
}
func TestStrictArtifactRoundTripAndDefensiveCopies(t *testing.T) {
	m, b := strictFixture(t)
	a, e := NewStrictInputArtifact(m, b)
	if e != nil {
		t.Fatal(e)
	}
	m.Configuration["mode"] = "changed"
	m.Roots[0].Files[0].Path = "changed"
	out := a.Manifest()
	out.Toolchain["go"] = "changed"
	out.Roots[0].Files[0].Digest = "changed"
	raw, d, e := a.Encode(1 << 20)
	if e != nil {
		t.Fatal(e)
	}
	got, e := DecodeStrictInputArtifact(context.Background(), bytes.NewReader(raw), 1<<20, d, b)
	if e != nil {
		t.Fatal(e)
	}
	if got.Manifest().Configuration["mode"] != "bounded" || got.Manifest().Toolchain["go"] != "fixture-toolchain" {
		t.Fatal("aliased caller map")
	}
	if got.Binding() != b {
		t.Fatal("binding changed")
	}
	if bytes.Contains(raw, []byte(`"path":"/`)) {
		t.Fatal("unexpected capture path")
	}
	again, d2, e := got.Encode(1 << 20)
	if e != nil || d2 != d || !bytes.Equal(raw, again) {
		t.Fatal("unstable round trip")
	}
}
func TestStrictArtifactManifestTamperingRefuses(t *testing.T) {
	for _, name := range []string{"schema", "path", "digest", "size", "root", "configuration", "toolchain", "duplicate"} {
		t.Run(name, func(t *testing.T) {
			m, b := strictFixture(t)
			switch name {
			case "schema":
				m.SchemaVersion++
			case "path":
				m.Roots[0].Files[0].Path = "../escape"
			case "digest":
				m.Roots[0].Files[0].Digest = "sha256:" + strings.Repeat("b", 64)
			case "size":
				m.Roots[0].Files[0].Size++
			case "root":
				m.Roots[0].Name = "changed"
			case "configuration":
				m.Configuration["mode"] = "changed"
			case "toolchain":
				m.Toolchain["go"] = "changed"
			case "duplicate":
				m.Roots = append(m.Roots, m.Roots[0])
			}
			if a, e := NewStrictInputArtifact(m, b); e == nil || a != nil {
				t.Fatal("tampered manifest accepted")
			}
		})
	}
}
func TestStrictArtifactExpectedBindingRefuses(t *testing.T) {
	m, b := strictFixture(t)
	a, _ := NewStrictInputArtifact(m, b)
	raw, d, _ := a.Encode(1 << 20)
	for _, name := range []string{"run", "target", "owner", "epoch", "configuration", "descriptor", "phases"} {
		t.Run(name, func(t *testing.T) {
			x := b
			changed := "sha256:" + strings.Repeat("b", 64)
			switch name {
			case "run":
				x.RunID = "other"
			case "target":
				x.Target = "other"
			case "owner":
				x.OwnerBindingDigest = changed
			case "epoch":
				x.ExecutionEpoch++
			case "configuration":
				x.ConfigurationDigest = changed
			case "descriptor":
				x.DescriptorDigest = changed
			case "phases":
				x.PhaseSetDigest = changed
			}
			if got, e := DecodeStrictInputArtifact(context.Background(), bytes.NewReader(raw), 1<<20, d, x); e == nil || got != nil {
				t.Fatal("wrong execution accepted")
			}
		})
	}
}
func TestStrictArtifactReaderAndEncodingRefusals(t *testing.T) {
	m, b := strictFixture(t)
	a, _ := NewStrictInputArtifact(m, b)
	raw, d, _ := a.Encode(1 << 20)
	if _, _, e := a.Encode(int64(len(raw) - 1)); e == nil {
		t.Fatal("truncated encoding accepted")
	}
	variants := map[string][]byte{"truncated": raw[:len(raw)-1], "trailing": append(append([]byte{}, raw...), []byte(" {}")...), "duplicate": bytes.Replace(raw, []byte(`"schema_version":1`), []byte(`"schema_version":1,"schema_version":1`), 1), "unknown": bytes.Replace(raw, []byte(`"schema_version":1`), []byte(`"schema_version":1,"qualified":true`), 1)}
	for name, v := range variants {
		t.Run(name, func(t *testing.T) {
			if got, e := DecodeStrictInputArtifact(context.Background(), bytes.NewReader(v), 1<<20, digestStrictBytes(v), b); e == nil || got != nil {
				t.Fatal("invalid artifact accepted")
			}
		})
	}
	if got, e := DecodeStrictInputArtifact(context.Background(), bytes.NewReader(raw), int64(len(raw)-1), d, b); e == nil || got != nil {
		t.Fatal("oversize accepted")
	}
	if got, e := DecodeStrictInputArtifact(context.Background(), bytes.NewReader(raw), 1<<20, "sha256:"+strings.Repeat("b", 64), b); e == nil || got != nil {
		t.Fatal("wrong bytes accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, e := DecodeStrictInputArtifact(ctx, bytes.NewReader(raw), 1<<20, d, b); e == nil || got != nil {
		t.Fatal("cancellation accepted")
	}
	p := a.payload
	p.SchemaVersion++
	v, _ := json.Marshal(p)
	if got, e := DecodeStrictInputArtifact(context.Background(), bytes.NewReader(v), 1<<20, digestStrictBytes(v), b); e == nil || got != nil {
		t.Fatal("future artifact accepted")
	}
}
func TestStrictArtifactMissingInputAndIdentityRefuse(t *testing.T) {
	root := t.TempDir()
	if _, e := treedigest.BuildInputManifestContext(context.Background(), treedigest.ManifestRequest{Primary: treedigest.RootSpec{Name: "product", Path: root, Files: []string{"absent"}}}); e == nil {
		t.Fatal("missing complete input accepted")
	}
	m, b := strictFixture(t)
	for _, name := range []string{"epoch", "digest", "run"} {
		x := b
		switch name {
		case "epoch":
			x.ExecutionEpoch = 0
		case "digest":
			x.PhaseSetDigest = "td:legacy"
		case "run":
			x.RunID = " "
		}
		if a, e := NewStrictInputArtifact(m, x); e == nil || a != nil {
			t.Fatal("incomplete identity accepted")
		}
	}
}

func TestStrictArtifactExactSchemaSpellingAndEscapedAliases(t *testing.T) {
	m, b := strictFixture(t)
	a, _ := NewStrictInputArtifact(m, b)
	raw, _, _ := a.Encode(1 << 20)
	for _, tc := range []struct{ name, old, replacement string }{
		{"top-case", `"schema_version":1`, `"schema_version":1,"SCHEMA_VERSION":1`},
		{"binding-case", `"binding":`, `"Binding":`},
		{"nested-binding-case", `"run_id":`, `"RUN_ID":`},
		{"manifest-case", `"manifest":`, `"Manifest":`},
		{"root-case", `"name":`, `"Name":`},
		{"file-case", `"path":`, `"PATH":`},
		{"escaped-case", `"schema_version":1`, `"schema_version":1,"\u0053CHEMA_VERSION":1`},
		{"escaped-duplicate", `"run_id":"fixture-run"`, `"run_id":"fixture-run","\u0072un_id":"fixture-run"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !bytes.Contains(raw, []byte(tc.old)) {
				t.Fatal("fixture replacement not reached")
			}
			offered := bytes.Replace(raw, []byte(tc.old), []byte(tc.replacement), 1)
			if got, err := DecodeStrictInputArtifact(context.Background(), bytes.NewReader(offered), 1<<20, digestStrictBytes(offered), b); err == nil || got != nil {
				t.Fatal("schema alias accepted")
			}
		})
	}
}
func TestStrictArtifactArbitraryUniqueScalarMapKeysRemainCompatible(t *testing.T) {
	m, b := strictFixture(t)
	// Build with case-distinct map keys through the existing owner builder so its
	// identity includes both keys; typed field spelling restrictions must not
	// incorrectly impose a struct-field policy on the scalar configuration map.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "input"), []byte("nonsecret"), 0600); err != nil {
		t.Fatal(err)
	}
	var err error
	m, err = treedigest.BuildInputManifestContext(context.Background(), treedigest.ManifestRequest{Primary: treedigest.RootSpec{Name: "product", Path: root, Files: []string{"input"}}, Configuration: map[string]string{"Mode": "A", "mode": "B", "schema_version": "nonsecret selection"}, Toolchain: map[string]string{"Go": "owner selection", "go": "other selection"}})
	if err != nil {
		t.Fatal(err)
	}
	a, err := NewStrictInputArtifact(m, b)
	if err != nil {
		t.Fatal(err)
	}
	raw, d, err := a.Encode(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeStrictInputArtifact(context.Background(), bytes.NewReader(raw), 1<<20, d, b)
	if err != nil {
		t.Fatal(err)
	}
	if got.Manifest().Configuration["Mode"] != "A" || got.Manifest().Configuration["mode"] != "B" {
		t.Fatal("scalar map keys changed")
	}
}
