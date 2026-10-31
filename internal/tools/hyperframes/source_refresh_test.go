package hyperframes

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sourceRefreshFixture(t *testing.T) (string, map[string][]byte, map[string][]byte) {
	t.Helper()
	source := t.TempDir()
	before := map[string][]byte{}
	for _, name := range dependencyNames {
		before[name], _ = closure.ReadFile(name)
		if err := os.WriteFile(filepath.Join(source, name), before[name], 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var manifest map[string]any
	_ = json.Unmarshal(before["package.json"], &manifest)
	manifest["dependencies"] = map[string]string{"hyperframes": "0.8.112"}
	newManifest, _ := json.Marshal(manifest)
	newLock, _ := json.Marshal(map[string]any{
		"name": manifest["name"], "version": manifest["version"], "lockfileVersion": 3,
		"packages": map[string]any{
			"":                         map[string]any{"name": manifest["name"], "version": manifest["version"], "dependencies": map[string]string{"hyperframes": "0.8.112"}},
			"node_modules/hyperframes": map[string]any{"version": "0.8.112", "resolved": "https://registry.npmjs.org/hyperframes/-/hyperframes-0.8.112.tgz", "integrity": "sha512-" + base64.StdEncoding.EncodeToString(make([]byte, 64))},
		},
	})
	return source, before, map[string][]byte{"package.json": newManifest, "package-lock.json": newLock}
}

func sourceRefreshTestOps() refreshOperations {
	return refreshOperations{acquire: func(context.Context, string) (func(), error) { return func() {}, nil }, rename: os.Rename}
}

func writeRefreshCandidate(t *testing.T, dir string, after map[string][]byte) {
	t.Helper()
	for name, data := range after {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSourceRefreshPublishesOnlyValidatedSourcePair(t *testing.T) {
	source, _, after := sourceRefreshFixture(t)
	var candidate string
	out, err := refreshSource(context.Background(), source, "0.8.112", func(_ context.Context, dir string) (string, error) {
		candidate = dir
		writeRefreshCandidate(t, dir, after)
		return "resolver fixture", nil
	}, sourceRefreshTestOps())
	if err != nil || !strings.Contains(out, "Previous immutable runtimes were not modified") {
		t.Fatal(out, err)
	}
	for name, want := range after {
		got, _ := os.ReadFile(filepath.Join(source, name))
		if !bytes.Equal(got, want) {
			t.Fatal("source publication mismatch", name)
		}
	}
	if _, err := os.Stat(candidate); !os.IsNotExist(err) {
		t.Fatal("private candidate retained", err)
	}
	if err := ValidateSource(source); err == nil {
		t.Fatal("new source was accepted as the old compiled frozen closure")
	}
}

func TestSourceRefreshFailureCancelAndConcurrentEditNeverPublish(t *testing.T) {
	for _, failure := range []string{"resolver", "cancel", "concurrent", "node_modules", "invalid"} {
		t.Run(failure, func(t *testing.T) {
			source, before, after := sourceRefreshFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, err := refreshSource(ctx, source, "0.8.112", func(_ context.Context, dir string) (string, error) {
				writeRefreshCandidate(t, dir, after)
				switch failure {
				case "resolver":
					return "failed fixture", errors.New("resolver failed")
				case "cancel":
					cancel()
				case "concurrent":
					if err := os.WriteFile(filepath.Join(source, "package.json"), []byte("independent edit"), 0o600); err != nil {
						t.Fatal(err)
					}
				case "node_modules":
					if err := os.Mkdir(filepath.Join(dir, "node_modules"), 0o700); err != nil {
						t.Fatal(err)
					}
				case "invalid":
					if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(`{}`), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				return "fixture", nil
			}, sourceRefreshTestOps())
			if err == nil {
				t.Fatal("failure published a source pair")
			}
			for name, want := range before {
				if failure == "concurrent" && name == "package.json" {
					want = []byte("independent edit")
				}
				got, _ := os.ReadFile(filepath.Join(source, name))
				if !bytes.Equal(got, want) {
					t.Fatal("failure overwrote source", name)
				}
			}
		})
	}
}

func TestSourceRefreshSecondPublicationFailureRollsBackWithoutClobberingEdits(t *testing.T) {
	for _, concurrent := range []bool{false, true} {
		source, before, after := sourceRefreshFixture(t)
		ops := sourceRefreshTestOps()
		calls := 0
		ops.rename = func(from, to string) error {
			calls++
			if calls == 2 {
				if concurrent {
					if err := os.WriteFile(filepath.Join(source, "package-lock.json"), []byte("independent edit"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				return errors.New("second publication failed")
			}
			return os.Rename(from, to)
		}
		_, err := refreshSource(context.Background(), source, "0.8.112", func(_ context.Context, dir string) (string, error) {
			writeRefreshCandidate(t, dir, after)
			return "", nil
		}, ops)
		if err == nil {
			t.Fatal("partial publication succeeded")
		}
		for name, want := range before {
			if concurrent && name == "package-lock.json" {
				want = []byte("independent edit")
				if !strings.Contains(err.Error(), "recover") {
					t.Fatal("manual recovery not identified", err)
				}
			}
			got, _ := os.ReadFile(filepath.Join(source, name))
			if !bytes.Equal(got, want) {
				t.Fatal("rollback mismatch", name)
			}
		}
	}
}

func TestSourceRefreshRejectsManifestSubstitutionAndUnpinnedLock(t *testing.T) {
	_, before, after := sourceRefreshFixture(t)
	for _, failure := range []string{"manifest", "integrity", "registry", "link", "version", "root", "root-name", "root-version", "root-dev", "root-optional"} {
		t.Run(failure, func(t *testing.T) {
			mutated := map[string][]byte{"package.json": after["package.json"], "package-lock.json": after["package-lock.json"]}
			var lock map[string]any
			_ = json.Unmarshal(mutated["package-lock.json"], &lock)
			pkg := lock["packages"].(map[string]any)["node_modules/hyperframes"].(map[string]any)
			root := lock["packages"].(map[string]any)[""].(map[string]any)
			switch failure {
			case "manifest":
				mutated["package.json"] = []byte(`{"dependencies":{"hyperframes":"0.8.112"}}`)
			case "integrity":
				pkg["integrity"] = "sha512-invalid"
			case "registry":
				pkg["resolved"] = "file:/tmp/private.tgz"
			case "link":
				pkg["link"] = true
			case "version":
				pkg["version"] = "0.8.111"
			case "root":
				lock["name"] = "other"
			case "root-name":
				root["name"] = "substituted"
			case "root-version":
				root["version"] = "2.0.0"
			case "root-dev":
				root["devDependencies"] = map[string]string{"unexpected": "1.0.0"}
			case "root-optional":
				root["optionalDependencies"] = map[string]string{"unexpected": "1.0.0"}
			}
			mutated["package-lock.json"], _ = json.Marshal(lock)
			if err := validateRefreshedInputs(before["package.json"], mutated, "0.8.112"); err == nil {
				t.Fatal("invalid candidate accepted")
			}
		})
	}
}
