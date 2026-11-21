package httpserver

import (
	"bytes"
	"context"
	"github.com/gorilla/mux"
	"github.com/vrooli/freshness-go/treedigest"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	appruns "test-genie/internal/app/runs"
	"test-genie/internal/executionevidence"
	"test-genie/internal/orchestrator"
	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"
	"testing"
	"time"
)

func strictHTTPFixture(t *testing.T) (*appruns.Service, string, appruns.StrictInputReadBinding, string) {
	t.Helper()
	root := t.TempDir()
	manifestDir := filepath.Join(root, "demo", ".vrooli")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "service.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "demo")
	runID := "strict-reader"
	idx := sharedruns.NewIndex(dir)
	if err := idx.Append(sharedruns.RunRecord{RunID: runID, Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := sharedruns.NewDescriptorSnapshot([]sharedruns.PhaseDescriptorSnapshot{{Phase: "unit", EvidenceKinds: []string{"generic.file"}, Applicability: sharedruns.ApplicabilityDecisionSnapshot{Status: "applies", Planned: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := sharedruns.WriteDescriptorSnapshot(dir, runID, snapshot); err != nil {
		t.Fatal(err)
	}
	if err := idx.Update(runID, func(r *sharedruns.RunRecord) error {
		r.DescriptorSnapshotSchemaVersion = snapshot.SchemaVersion
		r.DescriptorSnapshotDigest = snapshot.Digest
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ds, err := sharedruns.ReadDescriptorSnapshot(dir, runID)
	if err != nil {
		t.Fatal(err)
	}
	h := strings.Repeat("a", 64)
	config := "execution-config:" + h
	phase := "phase-set:" + h
	if err := idx.Finalize(runID, &orchestrator.SuiteExecutionResult{ConfigurationFingerprint: config, PhaseSetDigest: phase}, func(r *sharedruns.RunRecord) error { r.PhaseSetDigest = phase; return nil }); err != nil {
		t.Fatal(err)
	}
	input := t.TempDir()
	if err := os.WriteFile(filepath.Join(input, "source"), []byte("nonsecret input"), 0600); err != nil {
		t.Fatal(err)
	}
	m, err := treedigest.BuildInputManifestContext(context.Background(), treedigest.ManifestRequest{Primary: treedigest.RootSpec{Name: "product", Path: input, Files: []string{"source"}}})
	if err != nil {
		t.Fatal(err)
	}
	mapped, err := appruns.StrictInputTerminalDigests(config, ds.Digest, phase)
	if err != nil {
		t.Fatal(err)
	}
	execution := executionevidence.StrictExecutionBinding{RunID: runID, Target: "demo", ExecutionEpoch: 3, OwnerBindingDigest: "sha256:" + h, ConfigurationDigest: mapped[0], DescriptorDigest: mapped[1], PhaseSetDigest: mapped[2]}
	a, err := executionevidence.NewStrictInputArtifact(m, execution)
	if err != nil {
		t.Fatal(err)
	}
	raw, hash, err := a.Encode(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sharedartifacts.RunDir(dir, runID), "strict-inputs.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	ref, err := sharedartifacts.RegisterArtifact(dir, runID, sharedartifacts.ArtifactRegistration{Kind: "strict.input-manifest", StorageRoot: "run", StoragePath: "strict-inputs.json", MediaType: "application/json"})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := sharedartifacts.ReadArtifactCatalog(dir, runID)
	if err != nil {
		t.Fatal(err)
	}
	b := appruns.StrictInputReadBinding{MappingVersion: 1, Target: "demo", RunID: runID, ArtifactID: ref.ID, ByteSHA256: hash, SizeBytes: int64(len(raw)), CatalogDigest: catalog.Digest, Execution: execution, ConfigurationFingerprint: config, DescriptorDigest: ds.Digest, PhaseSetDigest: phase}
	return appruns.NewService(root, nil, nil, nil).SetArtifactRootResolver(func(name string) (string, error) { return filepath.Join(root, name), nil }), root, b, path
}

func TestStrictArtifactHTTPActualValidatedBytesAndRefusals(t *testing.T) {
	for _, name := range []string{"valid", "missing", "corrupt"} {
		t.Run(name, func(t *testing.T) {
			base, _, b, path := strictHTTPFixture(t)
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			svc := base
			if name != "missing" {
				svc, err = appruns.NewServiceWithStrictInputBindings(base, []appruns.StrictInputReadBinding{b})
				if err != nil {
					t.Fatal(err)
				}
			}
			if name == "corrupt" {
				changed := append([]byte{}, want...)
				changed[len(changed)-1] = 'x'
				if err := os.WriteFile(path, changed, 0600); err != nil {
					t.Fatal(err)
				}
			}
			server := &Server{runsService: svc, logger: log.New(io.Discard, "", 0)}
			req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/scenarios/demo/runs/strict-reader/artifacts/"+b.ArtifactID, nil), map[string]string{"name": b.Target, "runId": b.RunID, "artifactId": b.ArtifactID})
			response := httptest.NewRecorder()
			server.handleGetRunArtifactByID(response, req)
			if name == "valid" {
				if response.Code != 200 || !bytes.Equal(response.Body.Bytes(), want) {
					t.Fatalf("exact bytes refused: %d %s", response.Code, response.Body.String())
				}
				if response.Header().Get("X-Content-Type-Options") != "nosniff" || response.Header().Get("Content-Security-Policy") == "" {
					t.Fatal("legacy safety headers lost")
				}
			} else {
				if response.Code != http.StatusConflict || bytes.Equal(response.Body.Bytes(), want) {
					t.Fatal("strict refusal failed")
				}
			}
		})
	}
}

func TestStrictArtifactHTTPKindDowngradeRefusesAllSupportedWhitespaceTargets(t *testing.T) {
	for _, target := range []string{"demo", " demo "} {
		t.Run(target, func(t *testing.T) {
			base, root, b, _ := strictHTTPFixture(t)
			svc, err := appruns.NewServiceWithStrictInputBindings(base, []appruns.StrictInputReadBinding{b})
			if err != nil {
				t.Fatal(err)
			}
			ref, err := sharedartifacts.RegisterArtifact(filepath.Join(root, "demo"), b.RunID, sharedartifacts.ArtifactRegistration{Kind: sharedartifacts.ArtifactKindGenericFile, StorageRoot: "run", StoragePath: "strict-inputs.json"})
			if err != nil {
				t.Fatal(err)
			}
			if ref.ID != b.ArtifactID {
				t.Fatal("different ID")
			}
			server := &Server{runsService: svc, logger: log.New(io.Discard, "", 0)}
			req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/artifact", nil), map[string]string{"name": target, "runId": b.RunID, "artifactId": b.ArtifactID})
			rec := httptest.NewRecorder()
			server.handleGetRunArtifactByID(rec, req)
			if rec.Code != http.StatusConflict {
				t.Fatalf("downgrade served: %d %s", rec.Code, rec.Body.String())
			}
		})
	}
}
func TestStrictArtifactHTTPUnboundGenericStillStreams(t *testing.T) {
	base, root, b, _ := strictHTTPFixture(t)
	svc, err := appruns.NewServiceWithStrictInputBindings(base, []appruns.StrictInputReadBinding{b})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "demo")
	path := filepath.Join(sharedartifacts.RunDir(dir, b.RunID), "ordinary.txt")
	want := []byte("ordinary generic bytes")
	if err := os.WriteFile(path, want, 0600); err != nil {
		t.Fatal(err)
	}
	ref, err := sharedartifacts.RegisterArtifact(dir, b.RunID, sharedartifacts.ArtifactRegistration{Kind: sharedartifacts.ArtifactKindGenericFile, StorageRoot: "run", StoragePath: "ordinary.txt"})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{runsService: svc, logger: log.New(io.Discard, "", 0)}
	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/artifact", nil), map[string]string{"name": " demo ", "runId": b.RunID, "artifactId": ref.ID})
	rec := httptest.NewRecorder()
	server.handleGetRunArtifactByID(rec, req)
	if rec.Code != 200 || !bytes.Equal(rec.Body.Bytes(), want) {
		t.Fatalf("ordinary stream changed: %d", rec.Code)
	}
}
