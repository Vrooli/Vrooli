package content

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeRunner records the vault argv it is asked to run and returns canned
// results keyed by the verb (first arg after "kv").
type fakeRunner struct {
	calls    [][]string
	stdout   []byte
	stderr   []byte
	err      error
	patchErr error // when set, the first `kv patch` call returns this error
}

func (f *fakeRunner) Run(_ context.Context, vaultArgs []string, _ []byte) ([]byte, []byte, error) {
	f.calls = append(f.calls, vaultArgs)
	if len(vaultArgs) >= 2 && vaultArgs[0] == "kv" && vaultArgs[1] == "patch" && f.patchErr != nil {
		return nil, []byte("not found"), f.patchErr
	}
	return f.stdout, f.stderr, f.err
}

func (f *fakeRunner) lastCall() []string {
	if len(f.calls) == 0 {
		return nil
	}
	return f.calls[len(f.calls)-1]
}

func newHandlers(r Runner) (*Handlers, *bytes.Buffer) {
	out := &bytes.Buffer{}
	return &Handlers{Runner: r, Stdout: out, Stderr: &bytes.Buffer{}}, out
}

func equalArgs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestGetRawTranslatesToFieldFlag(t *testing.T) {
	r := &fakeRunner{stdout: []byte("the-passphrase\n")}
	h, out := newHandlers(r)
	if err := h.Get([]string{"--path", "secret/resources/kopia/repo/x/passphrase", "--key", "passphrase", "--format", "raw"}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	want := []string{"kv", "get", "-field=passphrase", "secret/resources/kopia/repo/x/passphrase"}
	if !equalArgs(r.lastCall(), want) {
		t.Fatalf("argv = %v, want %v", r.lastCall(), want)
	}
	if out.String() != "the-passphrase\n" {
		t.Fatalf("stdout = %q, want raw value passthrough", out.String())
	}
}

func TestGetRawDefaultsKeyWhenOmitted(t *testing.T) {
	r := &fakeRunner{stdout: []byte("v")}
	h, _ := newHandlers(r)
	if err := h.Get([]string{"--path", "secret/x", "--format", "raw"}); err != nil {
		t.Fatalf("Get: %v", err)
	}
	want := []string{"kv", "get", "-field=value", "secret/x"}
	if !equalArgs(r.lastCall(), want) {
		t.Fatalf("argv = %v, want default field %v", r.lastCall(), want)
	}
}

func TestSetDefaultsKeyWhenOmitted(t *testing.T) {
	r := &fakeRunner{}
	h, _ := newHandlers(r)
	if err := h.Set([]string{"--path", "secret/x", "--value", "v"}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	want := []string{"kv", "patch", "secret/x", "value=v"}
	if !equalArgs(r.lastCall(), want) {
		t.Fatalf("argv = %v, want default field %v", r.lastCall(), want)
	}
}

func TestGetJSONOmitsField(t *testing.T) {
	r := &fakeRunner{stdout: []byte(`{"data":{}}`)}
	h, _ := newHandlers(r)
	if err := h.Get([]string{"--path", "secret/x", "--format", "json"}); err != nil {
		t.Fatalf("Get json: %v", err)
	}
	want := []string{"kv", "get", "-format=json", "secret/x"}
	if !equalArgs(r.lastCall(), want) {
		t.Fatalf("argv = %v, want %v", r.lastCall(), want)
	}
}

func TestGetMissingTreatedAsErrorExit(t *testing.T) {
	// A missing field makes `vault kv get -field` exit non-zero; Get must
	// propagate the error so kopia's CLIVault can treat it as "not found".
	r := &fakeRunner{err: errors.New("exit status 2"), stderr: []byte("No value found")}
	h, _ := newHandlers(r)
	if err := h.Get([]string{"--path", "secret/x", "--key", "missing"}); err == nil {
		t.Fatal("expected non-nil error for missing field")
	}
}

func TestSetUsesPatchWhenSecretExists(t *testing.T) {
	r := &fakeRunner{} // patch succeeds
	h, _ := newHandlers(r)
	if err := h.Set([]string{"--path", "secret/x", "--key", "passphrase", "--value", "v123"}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	want := []string{"kv", "patch", "secret/x", "passphrase=v123"}
	if !equalArgs(r.lastCall(), want) {
		t.Fatalf("argv = %v, want %v", r.lastCall(), want)
	}
	if len(r.calls) != 1 {
		t.Fatalf("expected single patch call, got %v", r.calls)
	}
}

func TestSetFallsBackToPutWhenSecretAbsent(t *testing.T) {
	r := &fakeRunner{patchErr: errors.New("exit status 2")} // patch fails => put
	h, _ := newHandlers(r)
	if err := h.Set([]string{"--path", "secret/new", "--key", "k", "--value", "v"}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if len(r.calls) != 2 {
		t.Fatalf("expected patch then put, got %v", r.calls)
	}
	want := []string{"kv", "put", "secret/new", "k=v"}
	if !equalArgs(r.lastCall(), want) {
		t.Fatalf("put argv = %v, want %v", r.lastCall(), want)
	}
}

func TestSetRejectsAtPrefixedValue(t *testing.T) {
	r := &fakeRunner{}
	h, _ := newHandlers(r)
	if err := h.Set([]string{"--path", "secret/x", "--key", "k", "--value", "@/etc/passwd"}); err == nil {
		t.Fatal("expected rejection of '@'-prefixed value (Vault file reference)")
	}
	if len(r.calls) != 0 {
		t.Fatalf("runner should not run for rejected value, got %v", r.calls)
	}
}

func TestSetValueWithEqualsPreservesRemainder(t *testing.T) {
	r := &fakeRunner{}
	h, _ := newHandlers(r)
	// base64 padding contains '='; the whole value must survive as key=value.
	if err := h.Set([]string{"--path", "secret/x", "--key", "passphrase", "--value", "YWJj=="}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	want := []string{"kv", "patch", "secret/x", "passphrase=YWJj=="}
	if !equalArgs(r.lastCall(), want) {
		t.Fatalf("argv = %v, want %v", r.lastCall(), want)
	}
}

func TestDeleteAndList(t *testing.T) {
	r := &fakeRunner{stdout: []byte("keys\n")}
	h, _ := newHandlers(r)
	if err := h.Delete([]string{"--path", "secret/x"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if want := []string{"kv", "delete", "secret/x"}; !equalArgs(r.lastCall(), want) {
		t.Fatalf("delete argv = %v, want %v", r.lastCall(), want)
	}
	if err := h.List([]string{"--path", "secret/resources/kopia"}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if want := []string{"kv", "list", "secret/resources/kopia"}; !equalArgs(r.lastCall(), want) {
		t.Fatalf("list argv = %v, want %v", r.lastCall(), want)
	}
}

func TestPathRequired(t *testing.T) {
	r := &fakeRunner{}
	h, _ := newHandlers(r)
	for _, run := range []func([]string) error{h.Get, h.Set, h.Delete, h.List} {
		if err := run(nil); err == nil {
			t.Fatal("expected --path required error")
		}
	}
}

func TestCommandsSurface(t *testing.T) {
	grp := Commands(nil)
	if grp.Name != "content" {
		t.Fatalf("group name = %q", grp.Name)
	}
	got := map[string]bool{}
	for _, c := range grp.Subcommands {
		got[c.Name] = true
		if c.Run == nil {
			t.Fatalf("subcommand %q has nil Run", c.Name)
		}
	}
	for _, want := range []string{"get", "set", "delete", "list"} {
		if !got[want] {
			t.Fatalf("missing subcommand %q", want)
		}
	}
}

func TestNativeRunnerRequiresScopedToken(t *testing.T) {
	runner := &nativeRunner{binary: "vault", addr: "http://127.0.0.1:8200"}
	if _, _, err := runner.Run(context.Background(), []string{"kv", "get", "secret/example"}, nil); err == nil || !strings.Contains(err.Error(), "VAULT_TOKEN") {
		t.Fatalf("Run() error = %v, want scoped token requirement", err)
	}
}

func TestNativeRunnerRejectsDirectRemoteVrooliAccess(t *testing.T) {
	runner := &nativeRunner{binary: "vault", addr: "https://vault.example", token: "scoped", provider: "remote-vrooli"}
	if _, _, err := runner.Run(context.Background(), []string{"kv", "get", "secret/example"}, nil); err == nil || !strings.Contains(err.Error(), "scenario API") {
		t.Fatalf("Run() error = %v, want scenario API boundary", err)
	}
}

func TestDefaultRunnerUsesNativeManagedService(t *testing.T) {
	if _, ok := NewDefaultRunner().(*nativeRunner); !ok {
		t.Fatal("default runner must use the native managed service")
	}
}
