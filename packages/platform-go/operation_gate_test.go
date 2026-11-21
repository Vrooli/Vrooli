package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func privateGateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}
func TestOperationGateRevocationIsDurableAndCannotBeReopened(t *testing.T) {
	dir := privateGateDir(t)
	gate, err := PrepareOperationGate(dir, "attempt-1")
	if err != nil {
		t.Fatal(err)
	}
	if closed, err := gate.Revoked(); err != nil || closed {
		t.Fatalf("new gate was not open: %t %v", closed, err)
	}
	if err := gate.Revoke(); err != nil {
		t.Fatal(err)
	}
	if closed, err := gate.Revoked(); err != nil || !closed {
		t.Fatalf("revocation not durable: %t %v", closed, err)
	}
	if err := gate.Revoke(); err != nil {
		t.Fatal("repeat revocation failed", err)
	}
	if _, err := PrepareOperationGate(dir, "attempt-1"); err == nil {
		t.Fatal("revoked gate recreated")
	}
}
func TestOperationGateRejectsPublicAliasedAndMalformedLocations(t *testing.T) {
	dir := privateGateDir(t)
	for _, path := range []string{"relative.ready", dir + "/../attempt.ready", filepath.Join(dir, "bad\nname.ready"), filepath.Join(dir, "wrong.ext")} {
		gate := OperationGate{ReadyPath: path}
		if err := gate.validate(); err == nil {
			t.Fatalf("malformed gate accepted: %q", path)
		}
	}
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := PrepareOperationGate(dir, "attempt"); err == nil {
		t.Fatal("public gate directory accepted")
	}
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "foreign")
	if err := os.WriteFile(target, []byte("foreign"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "attempt.ready")); err != nil {
		t.Fatal(err)
	}
	if err := (OperationGate{ReadyPath: filepath.Join(dir, "attempt.ready")}).Revoke(); err == nil {
		t.Fatal("aliased readiness marker accepted")
	}
	if raw, err := os.ReadFile(target); err != nil || string(raw) != "foreign" {
		t.Fatal("foreign gate target changed")
	}
}
func TestOperationGateBindsDigestAndPreservesUngatedLegacyDigest(t *testing.T) {
	spec := OperationSpec{Path: "/usr/bin/true", Args: []string{}, Env: []string{}, Dir: "/tmp", Limits: OperationLimits{MemoryMaxBytes: 256 << 20, TasksMax: 32, CPUWeight: 50, WallTime: time.Minute}}
	legacy := struct {
		Path   string          `json:"path"`
		Args   []string        `json:"args"`
		Env    []string        `json:"env"`
		Dir    string          `json:"dir"`
		Limits OperationLimits `json:"limits"`
	}{spec.Path, spec.Args, spec.Env, spec.Dir, spec.Limits}
	raw, _ := json.Marshal(legacy)
	digest := sha256.Sum256(raw)
	if operationSpecDigest(spec) != hex.EncodeToString(digest[:]) {
		t.Fatal("ungated legacy spec digest changed")
	}
	gate, err := PrepareOperationGate(privateGateDir(t), "attempt")
	if err != nil {
		t.Fatal(err)
	}
	before := operationSpecDigest(spec)
	spec.Gate = &gate
	if operationSpecDigest(spec) == before {
		t.Fatal("gate missing from immutable spec digest")
	}
	if err := validateOperationSpec(spec); err != nil {
		t.Fatal(err)
	}
	literal := systemdExecArgument("/private/%h/quote\"slash\\tab\t.ready")
	if !strings.Contains(literal, "%h") || strings.Contains(literal, "%%h") || !strings.Contains(literal, `\"`) || !strings.Contains(literal, `\\`) || !strings.Contains(literal, `\t`) {
		t.Fatalf("native escaping is incomplete: %s", literal)
	}
}

func TestOperationGateRevokeSerializesWithPreparationPastItsCheck(t *testing.T) {
	dir := privateGateDir(t)
	checked, release := make(chan struct{}), make(chan struct{})
	prepared := make(chan error, 1)
	go func() {
		_, err := prepareOperationGate(dir, "concurrent", func() { close(checked); <-release })
		prepared <- err
	}()
	<-checked
	gate := OperationGate{ReadyPath: filepath.Join(dir, "concurrent.ready")}
	revoked := make(chan error, 1)
	go func() { revoked <- gate.Revoke() }()
	early := false
	var revokeErr error
	select {
	case revokeErr = <-revoked:
		early = true
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	if err := <-prepared; err != nil {
		t.Fatal(err)
	}
	if !early {
		revokeErr = <-revoked
	}
	if early {
		t.Fatal("revocation returned while a checked preparation could still publish readiness")
	}
	if revokeErr != nil {
		t.Fatal(revokeErr)
	}
	if closed, err := gate.Revoked(); err != nil || !closed {
		t.Fatalf("late preparation revived revoked gate: %t %v", closed, err)
	}
}

func TestOperationGateReplacementFinishesInterruptedPrivateRevocation(t *testing.T) {
	for _, content := range []string{"", "rev", "revoked\n"} {
		t.Run(fmt.Sprint(len(content)), func(t *testing.T) {
			dir := privateGateDir(t)
			gate, err := PrepareOperationGate(dir, "interrupted")
			if err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(dir, "interrupted.revoked")
			if err := os.WriteFile(marker, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			if err := gate.Revoke(); err != nil {
				t.Fatalf("replacement could not finish interrupted denial: %v", err)
			}
			if closed, err := gate.Revoked(); err != nil || !closed {
				t.Fatalf("interrupted revocation did not converge: %t %v", closed, err)
			}
			data, err := os.ReadFile(marker)
			if err != nil || string(data) != "revoked\n" {
				t.Fatal("replacement did not publish canonical durable denial")
			}
			if _, err := PrepareOperationGate(dir, "interrupted"); err == nil {
				t.Fatal("interrupted marker repair revived execution authority")
			}
		})
	}
}
func TestOperationGateRevocationRefusesForeignMarkerAliasesAndModes(t *testing.T) {
	for _, alias := range []bool{false, true} {
		t.Run(fmt.Sprint(alias), func(t *testing.T) {
			dir := privateGateDir(t)
			gate, err := PrepareOperationGate(dir, "foreign-marker")
			if err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(dir, "foreign-marker.revoked")
			foreign := filepath.Join(privateGateDir(t), "foreign")
			if err := os.WriteFile(foreign, []byte("preserve foreign data"), 0600); err != nil {
				t.Fatal(err)
			}
			if alias {
				if err := os.Symlink(foreign, marker); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(marker, nil, 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := gate.Revoke(); err == nil {
				t.Fatal("unknown marker authority was silently replaced")
			}
			if data, err := os.ReadFile(foreign); err != nil || string(data) != "preserve foreign data" {
				t.Fatal("foreign marker target changed")
			}
			if _, err := os.Stat(gate.ReadyPath); err != nil {
				t.Fatal("refused revocation advertised readiness absence")
			}
		})
	}
}
