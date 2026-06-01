package secrets

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	values map[string]string
	err    error
	calls  [][]string
}

func (f *fakeRunner) Run(_ context.Context, args []string, _ []byte) ([]byte, []byte, error) {
	f.calls = append(f.calls, append([]string(nil), args...))
	if f.err != nil {
		return nil, []byte("vault down"), f.err
	}
	if len(args) >= 4 && args[0] == "kv" && args[1] == "get" {
		key := args[3] + "::" + strings.TrimPrefix(args[2], "-field=")
		if v, ok := f.values[key]; ok {
			return []byte(v + "\n"), nil, nil
		}
		return nil, []byte("No value found"), errors.New("exit status 2")
	}
	if len(args) >= 4 && args[0] == "kv" && (args[1] == "patch" || args[1] == "put") {
		if f.values == nil {
			f.values = map[string]string{}
		}
		parts := strings.SplitN(args[3], "=", 2)
		if len(parts) == 2 {
			f.values[args[2]+"::"+parts[0]] = parts[1]
		}
		return nil, nil, nil
	}
	return nil, nil, nil
}

func TestScanCheckExportAndProvision(t *testing.T) {
	root := t.TempDir()
	writeSecrets(t, root, "demo", `
version: "1.0"
resource: "demo"
secrets:
  api_keys:
    - name: "api_key"
      path: "secret/resources/{resource}/api/key"
      required: true
      default_env: "DEMO_API_KEY"
    - name: "optional"
      path: "secret/resources/demo/optional"
      required: false
      default_env: "DEMO_OPTIONAL"
  dynamic:
    - name: "repo_secret"
      path: "secret/resources/demo/repo/{repo-name}/passphrase"
      required: true
initialization:
  auto_generate:
    - name: "optional"
      type: "random-32"
      path: "secret/resources/demo/optional"
`)
	r := &fakeRunner{values: map[string]string{
		"secret/resources/demo/api/key::value": "abc123",
	}}
	var out bytes.Buffer
	h := &Handlers{Runner: r, Stdout: &out, Stderr: &bytes.Buffer{}, Root: root}

	if err := h.Scan(nil); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !strings.Contains(out.String(), "demo: 3 secrets") {
		t.Fatalf("scan output = %q", out.String())
	}

	out.Reset()
	if err := h.Check([]string{"demo"}); err != nil {
		t.Fatalf("Check should ignore dynamic required declarations: %v", err)
	}
	if !strings.Contains(out.String(), "present api_key") || !strings.Contains(out.String(), "dynamic repo_secret") {
		t.Fatalf("check output = %q", out.String())
	}

	out.Reset()
	if err := h.Export([]string{"demo"}); err != nil {
		t.Fatalf("Export: %v", err)
	}
	if got := out.String(); got != "export DEMO_API_KEY='abc123'\n" {
		t.Fatalf("export output = %q", got)
	}

	out.Reset()
	if err := h.Provision([]string{"demo"}); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if _, ok := r.values["secret/resources/demo/optional::value"]; !ok {
		t.Fatal("provision did not store generated optional secret")
	}
}

func TestCheckSurfacesVaultErrors(t *testing.T) {
	root := t.TempDir()
	writeSecrets(t, root, "demo", `
resource: "demo"
secrets:
  api_keys:
    - name: "api_key"
      path: "secret/resources/demo/api/key"
      required: true
`)
	h := &Handlers{Runner: &fakeRunner{err: errors.New("exit status 1")}, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Root: root}
	if err := h.Check([]string{"demo"}); err == nil {
		t.Fatal("expected hard error for vault outage")
	}
}

func writeSecrets(t *testing.T, root, resource, body string) {
	t.Helper()
	dir := filepath.Join(root, "resources", resource, "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "secrets.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
