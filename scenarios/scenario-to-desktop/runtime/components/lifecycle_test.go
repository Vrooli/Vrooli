package components

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
	component "github.com/vrooli/vrooli/packages/desktop-component"
	release "github.com/vrooli/vrooli/packages/resource-deployment"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

// These are owner operations, not invented persisted lifecycle labels. Every
// row/flag combination is exercised, including defensive combinations that the
// supervisor does not normally emit. The contract is in TEMPORAL-FLOWS.md.
var componentEvents = []string{
	"install", "retry", "bad-signature", "download-failure", "archive-failure",
	"merge", "merge-tamper", "merge-signature", "merge-range", "merge-collision",
	"ready", "stale-ready", "disable", "stale-disable", "rollback", "stale-rollback",
	"bad-fallback", "remove-keep", "remove-delete", "reopen",
}

type componentFixture struct {
	source   *component.Source
	entries  []component.SignedEntry
	trees    []string
	archives [][]byte
	server   *httptest.Server
	mu       sync.Mutex
	before   func(*http.Request)
}

func newComponentFixture(t *testing.T) *componentFixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	f := &componentFixture{source: &component.Source{PublicKeyPEM: string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pub}))}}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		before := f.before
		f.mu.Unlock()
		if before != nil {
			before(r)
		}
		for i := range f.entries {
			if r.URL.Path == fmt.Sprintf("/%d", i) {
				_, _ = w.Write(f.archives[i])
				return
			}
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(f.server.Close)
	for i := 0; i < 3; i++ {
		version := fmt.Sprintf("0.0.%d", i+1)
		bundle := manifest.Manifest{SchemaVersion: "desktop.v0.1", Target: "desktop", App: manifest.App{Name: "Audio", Scenario: "audio-tools", Version: version}, IPC: manifest.IPC{Host: "127.0.0.1"}, Services: []manifest.Service{{ID: "audio-tools--api", Type: "api-binary", Binaries: map[string]manifest.Binary{manifest.PlatformKeys(runtime.GOOS, runtime.GOARCH)[0]: {Path: "bin/api"}}, Ports: &manifest.ServicePorts{Requested: []manifest.PortRequest{{Name: "api"}}}, Health: manifest.HealthCheck{Type: "http", Path: "/health", PortName: "api"}, Readiness: manifest.ReadinessCheck{Type: "health_success"}}}}
		raw, err := json.Marshal(bundle)
		if err != nil {
			t.Fatal(err)
		}
		tree := t.TempDir()
		if err := os.MkdirAll(filepath.Join(tree, "bin"), 0700); err != nil {
			t.Fatal(err)
		}
		files := map[string][]byte{"bundle.json": raw, "bin/api": []byte("fixture " + version)}
		var archive bytes.Buffer
		gz := gzip.NewWriter(&archive)
		tw := tar.NewWriter(gz)
		for _, name := range []string{"bundle.json", "bin/api"} {
			if err := os.WriteFile(filepath.Join(tree, name), files[name], 0600); err != nil {
				t.Fatal(err)
			}
			if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(files[name]))}); err != nil {
				t.Fatal(err)
			}
			if _, err := tw.Write(files[name]); err != nil {
				t.Fatal(err)
			}
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
		digest, err := binaryfetch.TreeDigest(tree)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(archive.Bytes())
		entry := component.Entry{SchemaVersion: 1, ID: "audio-tools", Owner: "audio-tools", Version: version, OS: runtime.GOOS, Arch: runtime.GOARCH, HostRange: ">=0.0.1 <1.0.0", Provides: []component.Feature{{ID: "audio.processing", Service: "audio-tools--api", Port: "api"}}, StateClass: "stateless", License: "test", ArtifactURL: fmt.Sprintf("%s/%d", f.server.URL, i), SHA256: hex.EncodeToString(sum[:]), TreeSHA256: digest, Size: int64(archive.Len()), Manifest: raw}
		canonical, err := entry.CanonicalBytes()
		if err != nil {
			t.Fatal(err)
		}
		sum = sha256.Sum256(canonical)
		sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
		if err != nil {
			t.Fatal(err)
		}
		f.entries = append(f.entries, component.SignedEntry{Entry: entry, Signature: release.ReleaseSignature{SchemaVersion: "v1", KeyID: "test", Algorithm: "rsa-pkcs1v15-sha256", Signature: base64.StdEncoding.EncodeToString(sig)}})
		f.archives = append(f.archives, append([]byte(nil), archive.Bytes()...))
		f.trees = append(f.trees, tree)
	}
	return f
}

func (f *componentFixture) manager(t *testing.T, row *Installation) *Manager {
	t.Helper()
	m := New(t.TempDir(), "0.0.1", f.source)
	for i, e := range f.entries {
		if err := os.CopyFS(filepath.Join(m.Root, e.Entry.ID, "generations", e.Entry.SHA256), os.DirFS(f.trees[i])); err != nil {
			t.Fatal(err)
		}
	}
	rows := map[string]Installation{}
	if row != nil {
		rows["audio-tools"] = *row
	}
	if err := m.save(rows); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(m.Root, "audio-tools", "data")
	if err := os.MkdirAll(data, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "saved"), []byte("keep me"), 0600); err != nil {
		t.Fatal(err)
	}
	return m
}

func readComponent(t *testing.T, m *Manager) *Installation {
	t.Helper()
	rows, err := m.Installed(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		return nil
	}
	if len(rows) != 1 {
		t.Fatalf("unexpected rows: %+v", rows)
	}
	return &rows[0]
}

func TestComponentLifecycleMatrix(t *testing.T) {
	f := newComponentFixture(t)
	rows := []*Installation{nil}
	names := []string{"absent"}
	for _, state := range []string{Pending, Installed, NeedsUpdate, OutOfRange} {
		for flags := 0; flags < 8; flags++ {
			row := Installation{Signed: f.entries[1], Generation: f.entries[1].Entry.SHA256, State: state, ActivationPending: flags&1 != 0, ActivationDisabled: flags&2 != 0}
			if flags&4 != 0 {
				row.Previous = &Generation{Signed: f.entries[0], Generation: f.entries[0].Entry.SHA256}
			}
			rows = append(rows, &row)
			names = append(names, fmt.Sprintf("%s/flags-%d", state, flags))
		}
	}
	for i, initial := range rows {
		for _, event := range componentEvents {
			t.Run(names[i]+"/"+event, func(t *testing.T) {
				m := f.manager(t, initial)
				var want *Installation
				if initial != nil {
					copy := *initial
					want = &copy
				}
				generation := f.entries[1].Entry.SHA256
				var err error
				wantErr := false
				host := manifest.Manifest{}
				wantServices := 0
				switch event {
				case "install", "retry":
					entry := f.entries[2]
					if event == "retry" {
						entry = f.entries[1]
					}
					next := Installation{Signed: entry, Generation: entry.Entry.SHA256, State: Pending}
					if want != nil {
						// A repeated stage must retain its already selected fallback.
						if want.Previous != nil {
							next.Previous = want.Previous
						} else if want.Generation != next.Generation && want.State == Installed {
							next.Previous = &Generation{Signed: want.Signed, Generation: want.Generation}
						}
					}
					want = &next
					_, err = m.Install(t.Context(), entry)
				case "bad-signature":
					entry := f.entries[2]
					entry.Signature.Signature = "invalid"
					_, err = m.Install(t.Context(), entry)
					wantErr = true
				case "download-failure", "archive-failure":
					m.Client = &http.Client{Transport: componentTransport(func(req *http.Request) (*http.Response, error) {
						if event == "download-failure" {
							return nil, fmt.Errorf("offline")
						}
						return &http.Response{StatusCode: 200, Body: http.NoBody, Header: http.Header{}}, nil
					})}
					_, err = m.Install(t.Context(), f.entries[2])
					wantErr = true
				case "merge", "merge-tamper", "merge-signature", "merge-range", "merge-collision":
					if event == "merge-tamper" {
						if err := os.WriteFile(filepath.Join(m.Root, "audio-tools", "generations", generation, "tamper"), []byte("bad"), 0600); err != nil {
							t.Fatal(err)
						}
					}
					if event == "merge-signature" {
						m.Source = &component.Source{PublicKeyPEM: "invalid"}
					}
					if event == "merge-range" {
						m.Version = "9.0.0"
					}
					if event == "merge-collision" {
						host.Services = []manifest.Service{{ID: "audio-tools--api"}}
						wantServices = 1
					}
					if want != nil && !want.ActivationDisabled {
						if want.State == Pending {
							want.ActivationPending = true
						}
						switch event {
						case "merge":
							want.State = Installed
							want.Reason = ""
							wantServices = 1
						case "merge-tamper":
							want.State = NeedsUpdate
							want.Reason = "installed tree failed verification"
						case "merge-signature":
							want.State = NeedsUpdate
							want.Reason = "stored signature failed verification"
						case "merge-range":
							want.State = OutOfRange
							want.Reason = "component does not support this host version"
						case "merge-collision":
							want.State = NeedsUpdate
							want.Reason = "component service collides with a bundled service"
						}
					}
					err = m.Merge(t.Context(), &host)
				case "ready", "stale-ready":
					if event == "stale-ready" {
						generation = "superseded"
					}
					wantErr = want == nil || event == "stale-ready"
					if !wantErr {
						want.Previous = nil
						want.ActivationPending = false
					}
					err = m.CompleteActivation(t.Context(), "audio-tools", generation)
				case "disable", "stale-disable":
					if event == "stale-disable" {
						generation = "superseded"
					}
					wantErr = want == nil || event == "stale-disable"
					if !wantErr {
						want.State = NeedsUpdate
						want.Reason = "readiness failed"
						want.Previous = nil
						want.ActivationPending = false
						want.ActivationDisabled = true
					}
					err = m.Disable(t.Context(), "audio-tools", generation, "readiness failed")
				case "rollback", "stale-rollback", "bad-fallback":
					if event == "stale-rollback" {
						generation = "superseded"
					}
					if event == "bad-fallback" {
						if err := os.WriteFile(filepath.Join(m.Root, "audio-tools", "generations", f.entries[0].Entry.SHA256, "tamper"), []byte("bad"), 0600); err != nil {
							t.Fatal(err)
						}
					}
					wantErr = want == nil || want.Previous == nil || event != "rollback"
					if !wantErr {
						want = &Installation{Signed: f.entries[0], Generation: f.entries[0].Entry.SHA256, State: Installed, Reason: "The update could not start; the previous version was restored."}
					}
					_, err = m.Rollback(t.Context(), "audio-tools", generation)
				case "remove-keep", "remove-delete":
					err = m.Remove(t.Context(), "audio-tools", event == "remove-keep")
					want = nil
				case "reopen":
					m = New(filepath.Dir(m.Root), m.Version, m.Source)
				default:
					t.Fatalf("uncovered event %q", event)
				}
				if (err != nil) != wantErr {
					t.Fatalf("error=%v, want error=%v", err, wantErr)
				}
				if got := readComponent(t, m); !equalInstallation(got, want) {
					t.Fatalf("transition %s: got %+v; want %+v", event, got, want)
				}
				if strings.HasPrefix(event, "merge") && len(host.Services) != wantServices {
					t.Fatalf("projected %d services; want %d", len(host.Services), wantServices)
				}
				saved, err := os.ReadFile(filepath.Join(m.Root, "audio-tools", "data", "saved"))
				if event == "remove-delete" {
					if !os.IsNotExist(err) {
						t.Fatalf("data survives delete: %v", err)
					}
				} else if err != nil || string(saved) != "keep me" {
					t.Fatalf("data changed: %q %v", saved, err)
				}
				if event == "remove-keep" || event == "remove-delete" {
					if _, err := os.Stat(filepath.Join(m.Root, "audio-tools", "generations")); !os.IsNotExist(err) {
						t.Fatalf("code survives removal: %v", err)
					}
				}
			})
		}
	}
	t.Logf("replayed %d state/flag cells x %d owner events = %d transitions", len(rows), len(componentEvents), len(rows)*len(componentEvents))
}

type componentTransport func(*http.Request) (*http.Response, error)

func (f componentTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// The subprocess is a focused Go test helper, not a scenario runtime launched
// outside its lifecycle owner. It dies while Install holds the real file lock.
func TestComponentProcessRecovery(t *testing.T) {
	if root := os.Getenv("SM10_HELPER_ROOT"); root != "" {
		raw, err := os.ReadFile(filepath.Join(root, "helper.json"))
		if err != nil {
			t.Fatal(err)
		}
		var input struct {
			Source component.Source
			Entry  component.SignedEntry
		}
		if err := json.Unmarshal(raw, &input); err != nil {
			t.Fatal(err)
		}
		m := New(root, "0.0.1", &input.Source)
		if os.Getenv("SM10_HELPER_EVENT") == "merge" {
			var host manifest.Manifest
			if err := m.Merge(t.Context(), &host); err != nil {
				t.Fatal(err)
			}
			os.Exit(73)
		}
		if _, err := m.Install(t.Context(), input.Entry); err != nil {
			t.Fatal(err)
		}
		return
	}
	f := newComponentFixture(t)
	old := Installation{Signed: f.entries[0], Generation: f.entries[0].Entry.SHA256, State: Installed}
	m := f.manager(t, &old)
	// Redirect the signed URL at the transport/server boundary without changing
	// signature bytes. The helper uses the actual fixture HTTP server.
	entered := make(chan struct{})
	releaseDownload := make(chan struct{})
	f.mu.Lock()
	f.before = func(r *http.Request) {
		if r.URL.Path == "/1" {
			close(entered)
			<-releaseDownload
		}
	}
	f.mu.Unlock()
	raw, err := json.Marshal(struct {
		Source component.Source
		Entry  component.SignedEntry
	}{*f.source, f.entries[1]})
	if err != nil {
		t.Fatal(err)
	}
	appData := filepath.Dir(m.Root)
	if err := os.WriteFile(filepath.Join(appData, "helper.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	helper := func(event string) *exec.Cmd {
		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestComponentProcessRecovery$")
		cmd.Env = append(os.Environ(), "SM10_HELPER_ROOT="+appData, "SM10_HELPER_EVENT="+event)
		return cmd
	}
	cmd := helper("install")
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		close(releaseDownload)
		t.Fatalf("child did not reach download: %s", output.String())
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("child did not die")
	}
	close(releaseDownload)
	f.mu.Lock()
	f.before = nil
	f.mu.Unlock()
	reopened := New(appData, "0.0.1", f.source)
	if got := readComponent(t, reopened); !equalInstallation(got, &old) {
		t.Fatalf("interrupted stage changed committed pointer: %+v", got)
	}
	if _, err := reopened.Install(t.Context(), f.entries[1]); err != nil {
		t.Fatalf("retry after process death: %v", err)
	}
	cmd = helper("merge")
	out, err := cmd.CombinedOutput()
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 73 {
		t.Fatalf("merge child: %v %s", err, out)
	}
	reopened = New(appData, "0.0.1", f.source)
	got := readComponent(t, reopened)
	if got.State != Installed || !got.ActivationPending || got.Previous == nil {
		t.Fatalf("boot crash lost activation evidence: %+v", got)
	}
	var host manifest.Manifest
	if err := reopened.Merge(t.Context(), &host); err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.Rollback(t.Context(), "audio-tools", got.Generation); err != nil {
		t.Fatal(err)
	}
	if got := readComponent(t, reopened); got.Generation != old.Generation || got.ActivationPending || got.Previous != nil {
		t.Fatalf("recovery did not restore old version: %+v", got)
	}
	t.Log("killed staged download retained committed generation; retry reacquired lock; separate process merged update and exited before readiness; restart retained fallback and rolled back")
}

func equalInstallation(a, b *Installation) bool {
	// Persistence may indent RawMessage JSON without changing the signed value.
	x, e1 := json.Marshal(a)
	y, e2 := json.Marshal(b)
	return e1 == nil && e2 == nil && bytes.Equal(x, y)
}

func TestComponentLifecycleTraces(t *testing.T) {
	f := newComponentFixture(t)
	m := New(t.TempDir(), "0.0.1", f.source)
	install := func(index int) *Installation {
		t.Helper()
		if _, err := m.Install(t.Context(), f.entries[index]); err != nil {
			t.Fatal(err)
		}
		return readComponent(t, m)
	}
	boot := func() *Installation {
		t.Helper()
		m = New(filepath.Dir(m.Root), "0.0.1", f.source)
		var host manifest.Manifest
		if err := m.Merge(t.Context(), &host); err != nil {
			t.Fatal(err)
		}
		if len(host.Services) != 1 {
			t.Fatalf("expected verified graph: %+v", host.Services)
		}
		return readComponent(t, m)
	}
	ready := func(row *Installation) {
		t.Helper()
		if err := m.CompleteActivation(t.Context(), "audio-tools", row.Generation); err != nil {
			t.Fatal(err)
		}
	}
	row := install(0)
	if row.State != Pending || row.Previous != nil {
		t.Fatalf("initial stage: %+v", row)
	}
	row = boot()
	if row.State != Installed || !row.ActivationPending {
		t.Fatalf("activation not pending: %+v", row)
	}
	ready(row)
	row = install(1)
	if row.Previous == nil || row.Previous.Generation != f.entries[0].Entry.SHA256 {
		t.Fatal("update lost healthy fallback")
	}
	row = install(1)
	if row.Previous == nil {
		t.Fatal("same staged update retry lost healthy fallback")
	}
	row = boot()
	if _, err := m.Rollback(t.Context(), "audio-tools", row.Generation); err != nil {
		t.Fatalf("failed readiness could not restore fallback: %v", err)
	}
	row = install(1)
	row = boot()
	newer := install(2)
	if newer.Previous == nil || newer.Previous.Generation != f.entries[0].Entry.SHA256 {
		t.Fatal("superseding stage replaced healthy fallback with unready generation")
	}
	if err := m.CompleteActivation(t.Context(), "audio-tools", row.Generation); err != ErrActivationSuperseded {
		t.Fatalf("stale readiness: %v", err)
	}
	if got := readComponent(t, m); !equalInstallation(got, newer) {
		t.Fatal("stale readiness changed newest generation")
	}
	row = boot()
	ready(row)
	row = readComponent(t, m)
	if row.ActivationPending || row.Previous != nil {
		t.Fatal("successful readiness did not commit newest generation")
	}
	if err := m.Remove(t.Context(), "audio-tools", true); err != nil {
		t.Fatal(err)
	}
	row = install(0)
	row = boot()
	if err := m.Disable(t.Context(), "audio-tools", row.Generation, "no healthy fallback"); err != nil {
		t.Fatal(err)
	}
	m = New(filepath.Dir(m.Root), "0.0.1", f.source)
	var host manifest.Manifest
	if err := m.Merge(t.Context(), &host); err != nil {
		t.Fatal(err)
	}
	if len(host.Services) != 0 || !readComponent(t, m).ActivationDisabled {
		t.Fatal("disabled component returned on boot without explicit retry")
	}
	row = install(0)
	row = boot()
	ready(row)
	if readComponent(t, m).ActivationDisabled {
		t.Fatal("explicit reinstall did not recover disabled component")
	}
	t.Log("actual signed archive traces: clean install/readiness; update/retry/failure/rollback; superseded activation; failed initial activation/disabled restart/explicit retry")
}

func TestComponentRetainedJ11Replay(t *testing.T) {
	root := os.Getenv("J11_RELEASE_ROOT")
	if root == "" {
		t.Skip("J11_RELEASE_ROOT selects the retained real signed Audio Tools release")
	}
	raw, err := os.ReadFile(filepath.Join(root, "entry.json"))
	if err != nil {
		t.Fatal(err)
	}
	var signed component.SignedEntry
	if err := json.Unmarshal(raw, &signed); err != nil {
		t.Fatal(err)
	}
	pub, err := os.ReadFile(filepath.Join(root, "release.pub"))
	if err != nil {
		t.Fatal(err)
	}
	m := New(t.TempDir(), "0.0.1", &component.Source{PublicKeyPEM: string(pub)})
	m.Client = &http.Client{Transport: componentTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != signed.Entry.ArtifactURL {
			return nil, fmt.Errorf("unexpected artifact URL")
		}
		archive, err := os.Open(filepath.Join(root, "audio-tools.tar.gz"))
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Body: archive, Header: http.Header{}}, nil
	})}
	if _, err := m.Install(t.Context(), signed); err != nil {
		t.Fatal(err)
	}
	if readComponent(t, m).State != Pending {
		t.Fatal("real archive not staged")
	}
	var host manifest.Manifest
	if err := m.Merge(t.Context(), &host); err != nil {
		t.Fatal(err)
	}
	row := readComponent(t, m)
	if row.State != Installed || !row.ActivationPending || len(host.Services) == 0 {
		t.Fatal("real archive not admitted at boot")
	}
	if err := m.CompleteActivation(t.Context(), signed.Entry.ID, row.Generation); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(m.Root, signed.Entry.ID, "data")
	if err := os.MkdirAll(data, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "saved"), []byte("saved"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(m.Root, signed.Entry.ID, "generations", row.Generation, "tamper"), []byte("tamper"), 0600); err != nil {
		t.Fatal(err)
	}
	host = manifest.Manifest{}
	if err := m.Merge(t.Context(), &host); err != nil {
		t.Fatal(err)
	}
	if len(host.Services) != 0 || readComponent(t, m).State != NeedsUpdate {
		t.Fatal("tampered real archive admitted")
	}
	if err := m.Remove(t.Context(), signed.Entry.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(data, "saved")); err != nil {
		t.Fatal("keep data lost saved file")
	}
	if _, err := m.Install(t.Context(), signed); err != nil {
		t.Fatal(err)
	}
	if err := m.Remove(t.Context(), signed.Entry.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(data); !os.IsNotExist(err) {
		t.Fatal("delete data retained directory")
	}
	t.Logf("retained real J11 archive=%s version=%s: stage/boot/ready/tamper rejection/remove keep/reinstall/remove delete; readiness signal supplied by test, actual process readiness remains J11 evidence", signed.Entry.SHA256, signed.Entry.Version)
}
