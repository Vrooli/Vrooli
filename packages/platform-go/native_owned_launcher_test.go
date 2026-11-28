package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeLauncherOmissionPreservesLegacySpecBytesAndDigest(t *testing.T) {
	s := OwnedServiceSpec{Path: "/usr/bin/vrooli", Dir: "/", Args: []string{"serve"}, Env: []string{}, Limits: ServiceLimits{MemoryMaxBytes: 67108864, TasksMax: 16, CPUWeight: 100}}
	wantService := `{"Path":"/usr/bin/vrooli","Dir":"/","Args":["serve"],"Env":[],"Limits":{"memory_max_bytes":67108864,"tasks_max":16,"cpu_weight":100}}`
	wantOperation := `{"Path":"/usr/bin/vrooli","Args":["serve"],"Env":[],"Dir":"/","Limits":{"memory_max_bytes":67108864,"tasks_max":16,"cpu_weight":100,"wall_time_ns":0},"Gate":null}`
	wantCanonical := `{"path":"/usr/bin/vrooli","args":["serve"],"env":[],"dir":"/","limits":{"memory_max_bytes":67108864,"tasks_max":16,"cpu_weight":100,"wall_time_ns":0}}`
	for _, item := range []struct {
		value any
		want  string
	}{{s, wantService}, {s.processSpec(), wantOperation}} {
		got, err := json.Marshal(item.value)
		if err != nil || string(got) != item.want {
			t.Fatalf("legacy bytes changed: %s; error=%v", got, err)
		}
	}
	h := sha256.Sum256([]byte(wantCanonical))
	if got := operationSpecDigest(s.processSpec()); got != hex.EncodeToString(h[:]) {
		t.Fatalf("omitted proof changed legacy digest: %s", got)
	}
}

func TestNativeLauncherProofIsFrozenInServiceIntentAndDecodedSpec(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vrooli")
	s := OwnedServiceSpec{Path: path, Dir: filepath.Dir(path), Args: []string{"__vrooli-openrgb-native-capsule", filepath.Join(t.TempDir(), "capsule.json")}, Env: []string{}, NativeAccount: "uid:501", Limits: ServiceLimits{MemoryMaxBytes: 64 << 20, TasksMax: 16, CPUWeight: 100, Enforcement: "native-advisory"}, NativeLauncher: &NativeLauncherProof{SchemaVersion: NativeLauncherProofVersion, Profile: NativeLauncherProfileOpenRGB, SHA256: strings.Repeat("a", 64)}}
	i, err := NewServiceIntent("resource-initial", "attempt", s)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var decoded OwnedServiceSpec
	if err = json.Unmarshal(raw, &decoded); err != nil || decoded.NativeLauncher == s.NativeLauncher || decoded.NativeLauncher == nil || *decoded.NativeLauncher != *s.NativeLauncher {
		t.Fatal("independently decoded proof changed", err)
	}
	if err = i.ValidateSpec(decoded); err != nil {
		t.Fatal(err)
	}
	decoded.NativeLauncher.SHA256 = strings.Repeat("b", 64)
	if err = i.ValidateSpec(decoded); !errors.Is(err, ErrOperationIdentity) {
		t.Fatal("changed proof was not digest-bound", err)
	}
	decoded.NativeLauncher = nil
	if err = i.ValidateSpec(decoded); !errors.Is(err, ErrOperationIdentity) {
		t.Fatal("proof downgrade accepted", err)
	}
	s.NativeAccount = ""
	if _, err = NewServiceIntent("resource-initial", "attempt", s); !errors.Is(err, ErrOperationIdentity) {
		t.Fatal("proof lacks frozen runtime account", err)
	}
}

func TestPublishedNativeLauncherRejectsUnknownOrSubstitutedAuthorityBeforeOpen(t *testing.T) {
	root := filepath.Join(t.TempDir(), "launchers")
	p := NativeLauncherProof{SchemaVersion: NativeLauncherProofVersion, Profile: NativeLauncherProfileOpenRGB, SHA256: strings.Repeat("a", 64)}
	path := filepath.Join(root, p.SHA256, "vrooli")
	for _, item := range []struct {
		name string
		path string
		p    NativeLauncherProof
	}{
		{"unknown-version", path, NativeLauncherProof{SchemaVersion: 2, Profile: p.Profile, SHA256: p.SHA256}},
		{"unknown-publisher", path, NativeLauncherProof{SchemaVersion: p.SchemaVersion, Profile: "arbitrary-native-code", SHA256: p.SHA256}},
		{"uppercase-hash", path, NativeLauncherProof{SchemaVersion: p.SchemaVersion, Profile: p.Profile, SHA256: strings.Repeat("A", 64)}},
		{"malformed-hash", path, NativeLauncherProof{SchemaVersion: p.SchemaVersion, Profile: p.Profile, SHA256: strings.Repeat("g", 64)}},
		{"foreign-root", filepath.Join(t.TempDir(), p.SHA256, "vrooli"), p},
		{"different-version", filepath.Join(root, strings.Repeat("b", 64), "vrooli"), p},
		{"different-executable", filepath.Join(root, p.SHA256, "another"), p},
		{"relative-path", "vrooli", p},
	} {
		t.Run(item.name, func(t *testing.T) {
			opened := false
			_, err := openPublishedNativeLauncherWith(context.Background(), item.path, item.p, "vrooli", func() (string, error) { return root, nil }, func(context.Context, string, string) (*os.File, error) {
				opened = true
				return nil, errors.New("must not reach native image observer")
			})
			if err == nil || opened {
				t.Fatal("substituted publication reached native opener", err)
			}
		})
	}
}

func TestPublishedNativeLauncherReturnsVerifiedFDWithoutBootOrLedgerDependency(t *testing.T) {
	body := []byte("maintained launcher fixture")
	h := sha256.Sum256(body)
	p := NativeLauncherProof{SchemaVersion: NativeLauncherProofVersion, Profile: NativeLauncherProfileOpenRGB, SHA256: hex.EncodeToString(h[:])}
	root := filepath.Join(t.TempDir(), "launchers")
	path := filepath.Join(root, p.SHA256, "vrooli")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	opens := 0
	f, err := openPublishedNativeLauncherWith(context.Background(), path, p, "vrooli", func() (string, error) { return root, nil }, func(_ context.Context, actualPath, actualSHA string) (*os.File, error) {
		opens++
		if actualPath != path || actualSHA != p.SHA256 {
			t.Fatal("native verifier received different authority")
		}
		f, err := os.Open(actualPath)
		if err == nil {
			_, err = f.Seek(7, io.SeekStart)
		}
		return f, err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := io.ReadAll(f)
	if err != nil || string(got) != string(body) || opens != 1 {
		t.Fatal("verified descriptor was not rewound or exact", string(got), opens, err)
	}
}

func TestPublishedNativeLauncherCancellationClosesOpenedDescriptor(t *testing.T) {
	root := t.TempDir()
	p := NativeLauncherProof{SchemaVersion: NativeLauncherProofVersion, Profile: NativeLauncherProfileOpenRGB, SHA256: strings.Repeat("a", 64)}
	path := filepath.Join(root, p.SHA256, "vrooli")
	ctx, cancel := context.WithCancel(context.Background())
	var opened *os.File
	_, err := openPublishedNativeLauncherWith(ctx, path, p, "vrooli", func() (string, error) { return root, nil }, func(context.Context, string, string) (*os.File, error) {
		var err error
		opened, err = os.CreateTemp(t.TempDir(), "verified")
		cancel()
		return opened, err
	})
	if !errors.Is(err, context.Canceled) || opened == nil {
		t.Fatal("cancellation lost", err)
	}
	if _, err := opened.Stat(); err == nil {
		t.Fatal("canceled verified descriptor remained open")
	}
	called := false
	for _, ctx := range []context.Context{nil, ctx} {
		if _, err = openPublishedNativeLauncherWith(ctx, path, p, "vrooli", func() (string, error) { called = true; return root, nil }, nil); err == nil || called {
			t.Fatal("invalid context reached native authority lookup", err)
		}
	}
}
