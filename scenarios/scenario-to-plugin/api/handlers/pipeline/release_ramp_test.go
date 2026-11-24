package pipeline

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	att "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/attestation"
	comp "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/composition"
	conf "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/conformance"
	dist "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/distribution"
	reh "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/rehearsal"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	_ "modernc.org/sqlite"
)

func TestInstallChecksumSidecarMatchesDownloadedPayloadBytesInLocalFakeFlow(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh is unavailable")
	}
	sha256sum, err := exec.LookPath("sha256sum")
	if err != nil {
		t.Skip("sha256sum is unavailable")
	}
	_ = sh
	_ = sha256sum

	payload := []byte("fixture plugin executable bytes\n")
	digestBytes := sha256.Sum256(payload)
	digest := hex.EncodeToString(digestBytes[:])
	cases := []struct {
		name     string
		checksum string
		wantPass bool
	}{
		{name: "matching payload digest and filename", checksum: digest + "  tool\n", wantPass: true},
		{name: "wrong payload digest", checksum: strings.Repeat("0", 64) + "  tool\n"},
		{name: "sidecar names a different payload", checksum: digest + "  another-tool\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			fakeBin := filepath.Join(root, "bin")
			downloadDir := filepath.Join(root, "downloads")
			if err := os.MkdirAll(fakeBin, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(downloadDir, 0o755); err != nil {
				t.Fatal(err)
			}
			payloadPath := filepath.Join(root, "fixture-payload")
			checksumPath := filepath.Join(root, "fixture-checksum")
			if err := os.WriteFile(payloadPath, payload, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(checksumPath, []byte(tc.checksum), 0o600); err != nil {
				t.Fatal(err)
			}
			fakeCurl := `#!/bin/sh
set -eu
url=""
output=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o|--output) output="$2"; shift 2 ;;
    http://*|https://*) url="$1"; shift ;;
    *) shift ;;
  esac
done
case "$url" in
  */tool-v1.0.0) cp "$LOCAL_FIXTURE_PAYLOAD" "$output" ;;
  */tool-v1.0.0.sha256) cp "$LOCAL_FIXTURE_CHECKSUM" "$output" ;;
  *) echo "unexpected local fixture URL" >&2; exit 42 ;;
esac
`
			if err := os.WriteFile(filepath.Join(fakeBin, "curl"), []byte(fakeCurl), 0o755); err != nil {
				t.Fatal(err)
			}
			installer := `#!/bin/sh
set -eu
cd "$LOCAL_FIXTURE_DOWNLOAD_DIR"
curl -fsSL https://fixture.invalid/releases/tool-v1.0.0 -o tool
curl -fsSL https://fixture.invalid/releases/tool-v1.0.0.sha256 -o tool.sha256
sha256sum --check tool.sha256
`
			installerPath := filepath.Join(root, "install.sh")
			if err := os.WriteFile(installerPath, []byte(installer), 0o755); err != nil {
				t.Fatal(err)
			}
			downloads, verifications := parseInstallIntegrity(installer)
			if reason := checksumMappingFailure(downloads, verifications); reason != "" {
				t.Fatalf("static conformance rejected the structurally mapped fake install: %s", reason)
			}
			t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("LOCAL_FIXTURE_PAYLOAD", payloadPath)
			t.Setenv("LOCAL_FIXTURE_CHECKSUM", checksumPath)
			t.Setenv("LOCAL_FIXTURE_DOWNLOAD_DIR", downloadDir)
			cmd := exec.Command(sh, installerPath)
			output, runErr := cmd.CombinedOutput()
			if (runErr == nil) != tc.wantPass {
				t.Fatalf("local fake installer error=%v, wantPass=%t; output=%s", runErr, tc.wantPass, output)
			}
			gotPayload, err := os.ReadFile(filepath.Join(downloadDir, "tool"))
			if err != nil || string(gotPayload) != string(payload) {
				t.Fatalf("fake downloaded payload = %q, err=%v", gotPayload, err)
			}
			gotSidecar, err := os.ReadFile(filepath.Join(downloadDir, "tool.sha256"))
			if err != nil || string(gotSidecar) != tc.checksum {
				t.Fatalf("fake downloaded sidecar = %q, err=%v", gotSidecar, err)
			}
		})
	}
}

func TestPruneExpiredPackagesDeletesOnlyUnpublishedOwnedArtifacts(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	packagesRoot := filepath.Join(base, "packages")
	if err := os.MkdirAll(packagesRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)
	old := now.Add(-91 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	recent := now.Add(-3 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	for _, row := range []struct {
		id, when  string
		published bool
	}{
		{"expired", old, false}, {"published", old, true}, {"recent", recent, false},
	} {
		root := filepath.Join(packagesRoot, "plugin-"+row.id+"-fixture")
		if err := os.MkdirAll(root, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "payload"), []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO plugin_packages (id, scenario, source_revision, digest, artifact_root, state, created_at) VALUES (?, 'fixture', 'r1', 'sha256:test', ?, 'composed', ?)`, row.id, root, row.when); err != nil {
			t.Fatal(err)
		}
		if row.published {
			if _, err := db.Exec(`INSERT INTO plugin_publications (package_id, channel, digest, coordinate) VALUES (?, 'stable', 'sha256:test', 'fixture')`, row.id); err != nil {
				t.Fatal(err)
			}
		}
	}
	h := &handler{root: t.TempDir(), db: db, artifactRoot: func(context.Context) (string, error) { return base, nil }, packages: map[string]packageRecord{}}
	if err := h.pruneExpiredPackages(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id         string
		wantExists bool
	}{
		{"expired", false}, {"published", true}, {"recent", true},
	} {
		root := filepath.Join(packagesRoot, "plugin-"+tc.id+"-fixture")
		_, statErr := os.Stat(root)
		if got := statErr == nil; got != tc.wantExists {
			t.Errorf("artifact %s exists=%t want %t (err=%v)", tc.id, got, tc.wantExists, statErr)
		}
		var count int
		if err := db.QueryRow(`SELECT count(*) FROM plugin_packages WHERE id = ?`, tc.id).Scan(&count); err != nil {
			t.Fatal(err)
		}
		want := 1
		if !tc.wantExists {
			want = 0
		}
		if count != want {
			t.Errorf("package row %s count=%d want %d", tc.id, count, want)
		}
	}
}

func TestPruneExpiredPackagesRefusesArtifactPathOutsideOwnedRoot(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	base, outside := t.TempDir(), t.TempDir()
	unsafePath := filepath.Join(outside, "plugin-unsafe-fixture")
	if _, err := db.Exec(`INSERT INTO plugin_packages (id, scenario, source_revision, digest, artifact_root, state, created_at) VALUES ('unsafe', 'fixture', 'r1', 'sha256:test', ?, 'composed', datetime('now', '-91 days'))`, unsafePath); err != nil {
		t.Fatal(err)
	}
	h := &handler{root: t.TempDir(), db: db, artifactRoot: func(context.Context) (string, error) { return base, nil }, packages: map[string]packageRecord{}}
	if err := h.pruneExpiredPackages(context.Background(), time.Now().UTC()); err == nil || !strings.Contains(err.Error(), "outside its owned artifact directory") {
		t.Fatalf("prune error=%v, want owned-directory refusal", err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM plugin_packages WHERE id = 'unsafe'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("unsafe package row removed: count=%d", count)
	}
}

func TestPublishFullLocalPipelineSuccessAndRegistryRefusal(t *testing.T) {
	for _, tc := range []struct {
		name          string
		omitReferrer  string
		wantPublished bool
	}{
		{name: "local managed-evidence-shaped package publishes through fakes", wantPublished: true},
		{name: "missing registry SBOM confirmation refuses", omitReferrer: "application/vnd.cyclonedx+json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gateRequests atomic.Int32
			gate := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				gateRequests.Add(1)
				ready, _ := structpb.NewStruct(map[string]any{"ready": true})
				body, _ := proto.Marshal(structpb.NewStructValue(ready))
				w.Header().Set("Content-Type", "application/proto")
				_, _ = w.Write(body)
			}))
			defer gate.Close()
			registry, recorder := newLocalOCIRegistry(t, tc.omitReferrer)
			defer registry.Close()
			t.Setenv("DEPLOYMENT_MANAGER_API_BASE", gate.URL)
			t.Setenv("PLUGIN_REGISTRY", registry.URL+"/local/plugins")
			t.Setenv("PLUGIN_REGISTRY_USERNAME", "")
			t.Setenv("PLUGIN_REGISTRY_TOKEN", "")
			managedEvidence := t.TempDir()
			t.Setenv("SCENARIO_TO_PLUGIN_ATTESTATION_DIR", managedEvidence)

			repoRoot := scenarioRepoRoot(t)
			h := &handler{root: repoRoot, packages: map[string]packageRecord{}}
			composed, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "hello-plugin", SourceRevision: "local-release-ramp"}))
			if err != nil {
				t.Fatal(err)
			}
			checked, err := h.Check(context.Background(), connect.NewRequest(&conf.CheckRequest{PackageId: composed.Msg.Package.Id}))
			if err != nil || !checked.Msg.Passed {
				t.Fatalf("Check = %+v, err=%v", checked, err)
			}
			writeLocalManagedEvidence(t, managedEvidence, composed.Msg.Package.Digest)
			attested, err := h.Attest(context.Background(), connect.NewRequest(&att.AttestRequest{PackageId: composed.Msg.Package.Id, DryRun: false}))
			if err != nil || !attested.Msg.Passed || len(attested.Msg.Evidence) != 3 {
				t.Fatalf("local managed-shaped Attest = %+v, err=%v", attested, err)
			}
			published, err := h.Publish(context.Background(), connect.NewRequest(&dist.PublishRequest{
				PackageId: composed.Msg.Package.Id, SourceRevision: "local-release-ramp", Channel: "0.1.0",
			}))
			if err != nil {
				t.Fatal(err)
			}
			if published.Msg.Published != tc.wantPublished {
				t.Fatalf("Publish = %+v, want Published=%t", published.Msg, tc.wantPublished)
			}
			if tc.wantPublished {
				if published.Msg.Digest != composed.Msg.Package.Digest || !strings.Contains(published.Msg.ConfirmationReference, "/manifests/0.1.0") {
					t.Fatalf("published confirmation = %+v", published.Msg)
				}
			} else if !strings.Contains(published.Msg.Refusal, "PLG-DIST-CONFIRM") {
				t.Fatalf("registry refusal = %+v, want PLG-DIST-CONFIRM", published.Msg)
			}
			if got := gateRequests.Load(); got != 1 {
				t.Fatalf("local release gate requests = %d, want one", got)
			}
			if got := recorder.manifestPuts.Load(); got != 4 {
				t.Fatalf("local registry manifest puts = %d, want package and three evidence manifests", got)
			}
			if got := recorder.referrerIndexRequests.Load(); got != 1 {
				t.Fatalf("local registry referrer index requests = %d, want one", got)
			}
			wantManifestGets := int32(3)
			if tc.wantPublished {
				wantManifestGets = 4
			}
			if got := recorder.manifestGets.Load(); got != wantManifestGets {
				t.Fatalf("local registry manifest gets = %d, want %d", got, wantManifestGets)
			}
		})
	}
}

func TestReleaseChecklistKeepsSecretsOutOfArtifactsEvidenceAndRehearsalCapture(t *testing.T) {
	root := fixtureRoot(t, true)
	scenario := filepath.Join(root, "scenarios", "fixture")
	installScript := `#!/bin/sh
set -eu
prefix=${HELLO_PLUGIN_PREFIX:-"${XDG_BIN_HOME:-$HOME/.local/bin}"}
mkdir -p "$prefix"
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cp "$script_dir/fixture" "$prefix/fixture"
chmod 0755 "$prefix/fixture"
`
	runner := `#!/bin/sh
printf 'sk-%s\n' '0123456789abcdef0123456789abcdef'
printf 'ghp_%s\n' '0123456789abcdef0123456789abcdef'
printf 'gho_%s\n' '0123456789abcdef0123456789abcdef'
printf 'ghu_%s\n' '0123456789abcdef0123456789abcdef'
printf 'ghs_%s\n' '0123456789abcdef0123456789abcdef'
printf 'ghr_%s\n' '0123456789abcdef0123456789abcdef'
printf '%s %s %s\n' '-----BEGIN' 'PRIVATE' 'KEY-----'
printf '%s\n' 'synthetic-private-key-material'
printf '%s %s %s\n' '-----END' 'PRIVATE' 'KEY-----'
`
	for rel, body := range map[string]string{
		"cli/install.sh":        installScript,
		"cli/fixture":           runner,
		"skills/hello/SKILL.md": "---\nname: hello\ndescription: Fixture skill.\n---\n\nRun fixture hello.\n```bash\nfixture hello\n```\n",
	} {
		if err := os.WriteFile(filepath.Join(scenario, rel), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var verdict bytes.Buffer
	verdictServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(&verdict, r.Body)
		w.Header().Set("Content-Type", "application/proto")
	}))
	defer verdictServer.Close()
	t.Setenv("DEPLOYMENT_MANAGER_API_BASE", verdictServer.URL)
	h := &handler{root: root, packages: map[string]packageRecord{}}
	composed, err := h.Compose(context.Background(), connect.NewRequest(&comp.ComposeRequest{Scenario: "fixture", SourceRevision: "secret-output"}))
	if err != nil {
		t.Fatal(err)
	}
	checked, err := h.Check(context.Background(), connect.NewRequest(&conf.CheckRequest{PackageId: composed.Msg.Package.Id}))
	if err != nil || !checked.Msg.Passed {
		t.Fatalf("Check = %+v, err=%v", checked, err)
	}
	attested, err := h.Attest(context.Background(), connect.NewRequest(&att.AttestRequest{PackageId: composed.Msg.Package.Id, DryRun: true}))
	if err != nil || !attested.Msg.Passed {
		t.Fatalf("Attest = %+v, err=%v", attested, err)
	}
	for _, name := range []string{".agent-plugin.tar.gz", "cosign.signature.json", "provenance.intoto.json", "bom.json"} {
		body, readErr := os.ReadFile(filepath.Join(composed.Msg.Package.ArtifactRoot, name))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if secretInBytes(body) {
			t.Fatalf("secret literal leaked into emitted %s", name)
		}
	}
	rehearsal, err := h.Run(context.Background(), connect.NewRequest(&reh.RunRequest{PackageId: composed.Msg.Package.Id, Sandbox: "workspace-sandbox"}))
	if err != nil || !rehearsal.Msg.Passed || len(rehearsal.Msg.Commands) != 1 {
		t.Fatalf("Run = %+v, err=%v", rehearsal, err)
	}
	captured := rehearsal.Msg.Commands[0].RedactedOutput
	for _, token := range []string{
		"sk-0123456789abcdef0123456789abcdef",
		"ghp_0123456789abcdef0123456789abcdef",
		"gho_0123456789abcdef0123456789abcdef",
		"ghu_0123456789abcdef0123456789abcdef",
		"ghs_0123456789abcdef0123456789abcdef",
		"ghr_0123456789abcdef0123456789abcdef",
		"synthetic-private-key-material",
	} {
		if strings.Contains(captured, token) {
			t.Fatalf("credential material leaked into rehearsal capture: %q", token)
		}
	}
	if strings.Count(captured, "[REDACTED]") < 7 {
		t.Fatalf("expected every token and private-key block to be redacted, got %q", captured)
	}
	if verdict.Len() == 0 {
		t.Fatal("local fake deployment manager did not receive the rehearsal verdict")
	}
	for _, output := range []string{fmt.Sprint(composed.Msg), fmt.Sprint(checked.Msg), fmt.Sprint(attested.Msg), fmt.Sprint(rehearsal.Msg), verdict.String()} {
		for _, token := range []string{"0123456789abcdef0123456789abcdef", "synthetic-private-key-material"} {
			if strings.Contains(output, token) {
				t.Fatalf("credential material leaked into API result or verdict: %q", token)
			}
		}
	}
}

type localOCIRecorder struct {
	mu                    sync.Mutex
	manifests             map[string][]byte
	manifestPuts          atomic.Int32
	manifestGets          atomic.Int32
	referrerIndexRequests atomic.Int32
	omitReferrer          string
}

func newLocalOCIRegistry(t *testing.T, omitReferrer string) (*httptest.Server, *localOCIRecorder) {
	t.Helper()
	recorder := &localOCIRecorder{manifests: map[string][]byte{}, omitReferrer: omitReferrer}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/blobs/uploads") {
			w.Header().Set("Location", server.URL+"/uploads/local")
			w.WriteHeader(http.StatusAccepted)
			return
		}
		if r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/uploads/") {
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusCreated)
			return
		}
		if strings.Contains(r.URL.Path, "/manifests/") {
			key := strings.TrimPrefix(r.URL.Path, "/v2/local/plugins/manifests/")
			recorder.mu.Lock()
			defer recorder.mu.Unlock()
			if r.Method == http.MethodPut {
				body, _ := io.ReadAll(r.Body)
				recorder.manifests[key] = body
				recorder.manifestPuts.Add(1)
				w.WriteHeader(http.StatusCreated)
				return
			}
			body, ok := recorder.manifests[key]
			if !ok {
				http.NotFound(w, r)
				return
			}
			recorder.manifestGets.Add(1)
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			_, _ = w.Write(body)
			return
		}
		if strings.Contains(r.URL.Path, "/referrers/") && r.Method == http.MethodGet {
			recorder.mu.Lock()
			defer recorder.mu.Unlock()
			recorder.referrerIndexRequests.Add(1)
			var descriptors []map[string]string
			for reference, body := range recorder.manifests {
				var manifest struct {
					ArtifactType string         `json:"artifactType"`
					Subject      map[string]any `json:"subject"`
				}
				if json.Unmarshal(body, &manifest) != nil || manifest.Subject == nil || manifest.Subject["digest"] != strings.TrimPrefix(r.URL.Path, "/v2/local/plugins/referrers/") || manifest.ArtifactType == recorder.omitReferrer {
					continue
				}
				descriptors = append(descriptors, map[string]string{"digest": reference, "artifactType": manifest.ArtifactType})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"schemaVersion": 2, "manifests": descriptors})
			return
		}
		http.NotFound(w, r)
	}))
	return server, recorder
}

func writeLocalManagedEvidence(t *testing.T, destination, digest string) {
	t.Helper()
	hexDigest := strings.TrimPrefix(digest, "sha256:")
	files := map[string][]byte{
		"cosign.signature.json":  []byte(fmt.Sprintf(`{"verificationMaterial":{},"messageSignature":{"messageDigest":{"algorithm":"SHA2_256","digest":%q},"signature":"synthetic"}}`, hexDigest)),
		"provenance.intoto.json": []byte(fmt.Sprintf(`{"subject":[{"name":"agent-plugin.tar.gz","digest":{"sha256":%q}}]}`, hexDigest)),
		"bom.json":               []byte(`{"bomFormat":"CycloneDX","specVersion":"1.6","components":[]}`),
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(destination, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
