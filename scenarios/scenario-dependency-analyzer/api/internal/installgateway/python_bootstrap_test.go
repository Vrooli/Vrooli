package installgateway

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testWheelHash = "1acb95a0758941925d91e7a6d87ad826cb8eff83458810f746497d4086139c5d"
const testWheelURL = "https://github.com/Dao-AILab/flash-attention/releases/download/v2.8.3.post2/flash_attn-2.8.3.post2%2Bcu12torch2.7cxx11abiTRUE-cp312-cp312-linux_x86_64.whl#sha256=" + testWheelHash

func TestPythonCompileFlagsAcceptedByInstalledUVWithoutResolution(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("installed uv parser unavailable; fake-owner checks do not qualify external argv compatibility")
	}
	root := t.TempDir()
	missing, output := filepath.Join(root, "absent-requirements.in"), filepath.Join(root, "requirements.lock")
	t.Setenv("UV_OFFLINE", "1")
	t.Setenv("UV_PYTHON_DOWNLOADS", "never")
	t.Setenv("UV_CACHE_DIR", filepath.Join(root, "cache"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	diagnostic, err := compilePythonLock(ctx, root, missing, output, pythonIndexSet{Primary: "https://pypi.org/simple"})
	if err == nil || ctx.Err() != nil || !strings.Contains(diagnostic, "absent-requirements.in") || strings.Contains(diagnostic, "cannot be used with") {
		t.Fatalf("actual uv must accept owner argv and reach missing-input refusal without resolving dependencies: %v\n%s", err, diagnostic)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("parser-only refusal created an output: %v", err)
	}
	t.Logf("installed parser accepted wheel-only argv; no input or resolution: %s", strings.TrimSpace(diagnostic))
}

func bootstrapResource(t *testing.T, input string) (string, string) {
	t.Helper()
	repo := t.TempDir()
	root := filepath.Join(repo, "resources", "stable-audio-3")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{
		"requirements.in": input,
		"resource.json":   `{"managed_service":{"acquisition":{"targets":[{"compose":[{"kind":"python-wheels","lockfile":"requirements.lock","index_url":"https://download.pytorch.org/whl/cu126","extra_index_url":["https://pypi.org/simple"]}]}]}}}`,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return repo, root
}

func TestResolveBootstrapsDeclaredResourceLockFromAuthoredInput(t *testing.T) {
	repo, root := bootstrapResource(t, "torch==2.7.1+cu126\nflash-attn @ "+testWheelURL+"\n")
	r, err := Resolve(repo, "stable-audio-3", "resource", "pip", "torch", "2.7.1+cu126")
	if err != nil {
		t.Fatal("authored input must bootstrap without a fake existing lock:", err)
	}
	if r.PackageManager != "uv-lock" || len(r.PythonLockfiles) != 1 || r.PythonLockfiles[0] != filepath.Join(root, "requirements.lock") {
		t.Fatalf("wrong owner resolution: %+v", r)
	}
	if !strings.Contains(r.Command(), "requirements.in") || !strings.Contains(r.Command(), "--only-binary") {
		t.Fatal(r.Command())
	}
	if _, err := os.Stat(filepath.Join(root, "requirements.lock")); !os.IsNotExist(err) {
		t.Fatal("resolution mutated output lock")
	}
}

func TestPythonBootstrapRejectsUnsafeOrAmbiguousInputs(t *testing.T) {
	if _, err := parsePythonBootstrapInput([]byte("foo-bar==1.0\nfoobar==1.0")); err != nil {
		t.Fatal("distinct Python distribution names collapsed:", err)
	}
	for name, input := range map[string]string{
		"hash missing":         "flash-attn @ " + strings.Split(testWheelURL, "#")[0],
		"credential URL":       "flash-attn @ " + strings.Replace(testWheelURL, "https://", "https://secret@", 1),
		"query credential":     "flash-attn @ " + strings.Replace(testWheelURL, "#", "?token=secret#", 1),
		"source build":         "flash-attn @ " + strings.Replace(testWheelURL, ".whl#", ".tar.gz#", 1),
		"wrong wheel name":     "other @ " + testWheelURL,
		"duplicate":            "torch==2.7.1\ntorch==2.7.1",
		"normalized duplicate": "foo-bar==1.0\nfoo_bar==1.0",
		"range":                "torch>=2.7",
		"include":              "-r ungoverned.in",
		"source index option":  "--extra-index-url https://example.com/simple",
		"local source":         "flash-attn @ file:///tmp/flash.whl",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parsePythonBootstrapInput([]byte(input)); err == nil {
				t.Fatal("unsafe bootstrap accepted")
			}
		})
	}
}

func TestPythonBootstrapRefusesUndeclaredOrChangedInputs(t *testing.T) {
	repo, root := bootstrapResource(t, "torch==2.7.1+cu126\n")
	r, err := Resolve(repo, "stable-audio-3", "resource", "pip", "torch", "2.7.1+cu126")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "requirements.in"), []byte("torch==2.8.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := (ExecInstaller{}).Install(context.Background(), r); err == nil || !strings.Contains(err.Error(), "changed after approval") {
		t.Fatalf("changed source accepted: %v", err)
	}
	if _, err := Resolve(repo, "stable-audio-3", "resource", "pip", "torch", "2.7.1+cu126"); err == nil {
		t.Fatal("undeclared trigger accepted")
	}
	if err := os.WriteFile(filepath.Join(root, "resource.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(repo, "stable-audio-3", "resource", "pip", "torch", "2.8.0"); err == nil {
		t.Fatal("undeclared lock accepted")
	}
}

func hashedBootstrapFixture() string {
	return "torch==2.7.1+cu126 \\\n    --hash=sha256:" + strings.Repeat("a", 64) + "\nflash-attn @ " + testWheelURL + " \\\n    --hash=sha256:" + testWheelHash + "\ntransitive==1.0 \\\n    --hash=sha256:" + strings.Repeat("b", 64) + "\n"
}

func TestPythonBootstrapRequiresCompleteHashedOutput(t *testing.T) {
	authored, err := parsePythonBootstrapInput([]byte("torch==2.7.1+cu126\nflash-attn @ " + testWheelURL))
	if err != nil {
		t.Fatal(err)
	}
	valid := hashedBootstrapFixture()
	if err := validatePythonGeneratedLock([]byte(valid), authored); err != nil {
		t.Fatal(err)
	}
	normalized := strings.Replace(valid, "#sha256="+testWheelHash, "", 1)
	if err := validatePythonGeneratedLock([]byte(normalized), authored); err != nil {
		t.Fatal("valid normalized URL refused:", err)
	}
	for name, invalid := range map[string]string{
		"transitive unhashed":     strings.Replace(valid, "    --hash=sha256:"+strings.Repeat("b", 64), "", 1),
		"publisher hash changed":  strings.Replace(valid, "--hash=sha256:"+testWheelHash, "--hash=sha256:"+strings.Repeat("c", 64), 1),
		"URL hash changed":        strings.Replace(valid, "#sha256="+testWheelHash, "#sha256="+strings.Repeat("c", 64), 1),
		"direct hash set widened": strings.Replace(normalized, "--hash=sha256:"+testWheelHash, "--hash=sha256:"+testWheelHash+" \\\n    --hash=sha256:"+strings.Repeat("c", 64), 1),
		"direct omitted":          strings.Split(valid, "flash-attn")[0],
		"version changed":         strings.Replace(valid, "torch==2.7.1+cu126", "torch==2.8.0", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if err := validatePythonGeneratedLock([]byte(invalid), authored); err == nil {
				t.Fatal("invalid resolver output published")
			}
		})
	}
}

func TestPythonBootstrapRefusesChangedIndexManifestBeforeResolver(t *testing.T) {
	repo, root := bootstrapResource(t, "torch==2.7.1+cu126\n")
	r, err := Resolve(repo, "stable-audio-3", "resource", "pip", "torch", "2.7.1+cu126")
	if err != nil {
		t.Fatal(err)
	}
	if r.PythonBootstrap.Indexes.Primary != "https://download.pytorch.org/whl/cu126" {
		t.Fatal("index snapshot not bound")
	}
	if err := os.WriteFile(filepath.Join(root, "resource.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir()) // No resolver exists; the input fence must run first.
	if _, err := (ExecInstaller{}).Install(context.Background(), r); err == nil || !strings.Contains(err.Error(), "changed after approval") {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "requirements.lock")); !os.IsNotExist(err) {
		t.Fatal("changed manifest produced a lock")
	}
}

// This test binary is the resolver fixture. It never runs uv/pip or downloads.
func TestPythonBootstrapResolverFixture(t *testing.T) {
	if os.Getenv("SDA_BOOTSTRAP_FIXTURE") != "1" {
		return
	}
	args := os.Args
	for i, arg := range args {
		if arg == "--output-file" && i+1 < len(args) {
			if err := os.WriteFile(args[i+1], []byte(hashedBootstrapFixture()), 0o600); err != nil {
				os.Exit(3)
			}
		}
	}
	if path := os.Getenv("SDA_BOOTSTRAP_ARGV"); path != "" {
		b, _ := json.Marshal(args)
		_ = os.WriteFile(path, b, 0o600)
	}
	os.Exit(0)
}

func TestPythonBootstrapPublishesOnlyValidatedNewLock(t *testing.T) {
	repo, root := bootstrapResource(t, "torch==2.7.1+cu126\nflash-attn @ "+testWheelURL)
	r, err := Resolve(repo, "stable-audio-3", "resource", "pip", "torch", "2.7.1+cu126")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "uv"), []byte("#!/bin/sh\nexec \""+self+"\" -test.run=^TestPythonBootstrapResolverFixture$ -- \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SDA_BOOTSTRAP_FIXTURE", "1")
	argv := filepath.Join(bin, "argv.json")
	t.Setenv("SDA_BOOTSTRAP_ARGV", argv)
	if _, err := (ExecInstaller{}).Install(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(filepath.Join(root, "requirements.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), testWheelHash) || strings.Contains(string(generated), ".sda-python-lock-") {
		t.Fatal(string(generated))
	}
	actual, err := os.ReadFile(argv)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"--generate-hashes", "--only-binary", "--no-config", "3.12", "x86_64-unknown-linux-gnu", "https://download.pytorch.org/whl/cu126", "https://pypi.org/simple"} {
		if !strings.Contains(string(actual), expected) {
			t.Fatal(string(actual))
		}
	}
	if strings.Contains(string(actual), "--no-build") {
		t.Fatal("uv rejects redundant --no-build combined with --only-binary", string(actual))
	}
	if _, err := (ExecInstaller{}).Install(context.Background(), r); err == nil {
		t.Fatal("bootstrap replaced an existing output")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := compilePythonLock(ctx, root, "unused", "unused", pythonIndexSet{Primary: "https://pypi.org/simple"}); err == nil {
		t.Fatal("canceled resolver started")
	}
}
