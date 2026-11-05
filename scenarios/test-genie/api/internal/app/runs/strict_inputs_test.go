package runs

import (
	"connectrpc.com/connect"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/vrooli/freshness-go/treedigest"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	"os"
	"path/filepath"
	"strings"
	"test-genie/internal/executionevidence"
	"test-genie/internal/orchestrator"
	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"
	"testing"
	"time"
)

func strictRunFixture(t *testing.T) (*Service, string, StrictInputReadBinding, string) {
	t.Helper()
	root := newFleetRoot(t)
	dir := filepath.Join(root, "demo")
	runID := "strict-reader"
	idx := sharedruns.NewIndex(dir)
	if err := idx.Append(sharedruns.RunRecord{RunID: runID, Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed}); err != nil {
		t.Fatal(err)
	}
	seedDescriptorSnapshot(t, root, runID, capturedPhase("unit", "Unit"))
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
	mapped, err := StrictInputTerminalDigests(config, ds.Digest, phase)
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
	ref, err := sharedartifacts.RegisterArtifact(dir, runID, sharedartifacts.ArtifactRegistration{Kind: strictInputArtifactKind, StorageRoot: "run", StoragePath: "strict-inputs.json", MediaType: "application/json"})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := sharedartifacts.ReadArtifactCatalog(dir, runID)
	if err != nil {
		t.Fatal(err)
	}
	b := StrictInputReadBinding{MappingVersion: 1, Target: "demo", RunID: runID, ArtifactID: ref.ID, ByteSHA256: hash, SizeBytes: int64(len(raw)), CatalogDigest: catalog.Digest, Execution: execution, ConfigurationFingerprint: config, DescriptorDigest: ds.Digest, PhaseSetDigest: phase}
	return NewService(root, nil, nil, nil).SetArtifactRootResolver(testArtifactRoot(root)), root, b, path
}
func strictReadTree(t *testing.T, root string) string {
	t.Helper()
	h := sha256.New()
	if err := filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() {
			raw, e := os.ReadFile(p)
			if e != nil {
				return e
			}
			h.Write([]byte(p))
			h.Write(raw)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}
func TestStrictInputGetRunArtifactActualReadAndFrozenTable(t *testing.T) {
	base, root, b, path := strictRunFixture(t)
	bindings := []StrictInputReadBinding{b}
	svc, err := NewServiceWithStrictInputBindings(base, bindings)
	if err != nil {
		t.Fatal(err)
	}
	bindings[0].Execution.ExecutionEpoch = 99
	before := strictReadTree(t, root)
	req := connect.NewRequest(&runspb.GetRunArtifactRequest{Target: b.Target, RunId: b.RunID, ArtifactId: b.ArtifactID})
	got, err := svc.GetRunArtifact(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetArtifact().GetId() != b.ArtifactID || got.Msg.GetLegacyDiscovered() {
		t.Fatal("wrong metadata")
	}
	_, raw, err := svc.ReadStrictInputArtifact(context.Background(), b.Target, b.RunID, b.ArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := os.ReadFile(path)
	if string(raw) != string(want) {
		t.Fatal("bytes changed")
	}
	if strictReadTree(t, root) != before {
		t.Fatal("read changed tree")
	}
	if _, err := base.GetRunArtifact(context.Background(), req); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatal("base service mutated")
	}
}
func TestStrictInputReadRefusalsHaveNoTreeEffects(t *testing.T) {
	for _, name := range []string{"missing", "hash", "size", "config", "descriptor", "phases", "catalog", "epoch", "corrupt", "canceled", "unknown", "legacy"} {
		t.Run(name, func(t *testing.T) {
			base, root, b, path := strictRunFixture(t)
			bindings := []StrictInputReadBinding{b}
			ctx := context.Background()
			switch name {
			case "missing":
				bindings = nil
			case "hash":
				bindings[0].ByteSHA256 = "sha256:" + strings.Repeat("b", 64)
			case "size":
				bindings[0].SizeBytes++
			case "config":
				bindings[0].ConfigurationFingerprint = "execution-config:" + strings.Repeat("b", 64)
			case "descriptor":
				bindings[0].DescriptorDigest = "ds:sha256:" + strings.Repeat("b", 64)
			case "phases":
				bindings[0].PhaseSetDigest = "phase-set:" + strings.Repeat("b", 64)
			case "catalog":
				bindings[0].CatalogDigest = "ac:sha256:" + strings.Repeat("b", 64)
			case "epoch":
				bindings[0].Execution.ExecutionEpoch++
			case "corrupt":
				raw, _ := os.ReadFile(path)
				raw[len(raw)-1] = 'x'
				if e := os.WriteFile(path, raw, 0600); e != nil {
					t.Fatal(e)
				}
			case "legacy":
				dir := filepath.Join(root, "demo")
				catalog, err := sharedartifacts.ReadArtifactCatalog(dir, b.RunID)
				if err != nil {
					t.Fatal(err)
				}
				catalog.LegacyDiscovered = true
				catalog.Digest = "" // Ask the canonical writer to recompute after the fixture change.
				if err := sharedartifacts.WriteArtifactCatalog(dir, catalog); err != nil {
					t.Fatal(err)
				}
				catalog, err = sharedartifacts.ReadArtifactCatalog(dir, b.RunID)
				if err != nil {
					t.Fatal(err)
				}
				bindings[0].CatalogDigest = catalog.Digest
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "unknown":
				bindings[0].ArtifactID = "artifact_" + strings.Repeat("b", 32)
			}
			if len(bindings) > 0 && (name == "config" || name == "descriptor" || name == "phases") {
				mapped, err := StrictInputTerminalDigests(bindings[0].ConfigurationFingerprint, bindings[0].DescriptorDigest, bindings[0].PhaseSetDigest)
				if err != nil {
					t.Fatal(err)
				}
				bindings[0].Execution.ConfigurationDigest = mapped[0]
				bindings[0].Execution.DescriptorDigest = mapped[1]
				bindings[0].Execution.PhaseSetDigest = mapped[2]
			}
			svc, e := NewServiceWithStrictInputBindings(base, bindings)
			if e != nil {
				t.Fatal(e)
			}
			before := strictReadTree(t, root)
			_, e = svc.GetRunArtifact(ctx, connect.NewRequest(&runspb.GetRunArtifactRequest{Target: b.Target, RunId: b.RunID, ArtifactId: b.ArtifactID}))
			if connect.CodeOf(e) != connect.CodeFailedPrecondition {
				t.Fatalf("want refusal: %v", e)
			}
			if strictReadTree(t, root) != before {
				t.Fatal("refusal changed tree")
			}
		})
	}
}
func TestStrictInputConstructorRefusesDuplicateZeroAndDomainAliases(t *testing.T) {
	base, _, b, _ := strictRunFixture(t)
	for _, name := range []string{"duplicate", "zero", "domain"} {
		rows := []StrictInputReadBinding{b}
		switch name {
		case "duplicate":
			rows = append(rows, b)
		case "zero":
			rows[0].Execution.ExecutionEpoch = 0
		case "domain":
			rows[0].ConfigurationFingerprint = rows[0].Execution.ConfigurationDigest
		}
		if s, e := NewServiceWithStrictInputBindings(base, rows); e == nil || s != nil {
			t.Fatal("invalid installation accepted")
		}
	}
}

func TestStrictInputMappingDomainsRefuseAliases(t *testing.T) {
	base, _, b, _ := strictRunFixture(t)
	for _, name := range []string{"strip", "swapped", "upper", "nul", "version"} {
		x := b
		switch name {
		case "strip":
			x.Execution.ConfigurationDigest = "sha256:" + strings.TrimPrefix(x.ConfigurationFingerprint, "execution-config:")
		case "swapped":
			x.Execution.ConfigurationDigest, x.Execution.PhaseSetDigest = x.Execution.PhaseSetDigest, x.Execution.ConfigurationDigest
		case "upper":
			x.ConfigurationFingerprint = strings.ToUpper(x.ConfigurationFingerprint)
		case "nul":
			x.ConfigurationFingerprint += "\x00"
		case "version":
			x.MappingVersion = 2
		}
		if s, e := NewServiceWithStrictInputBindings(base, []StrictInputReadBinding{x}); e == nil || s != nil {
			t.Fatal("unsupported reference mapping accepted")
		}
	}
}

func TestStrictBoundIDCannotDowngradeKindThroughTargetAlias(t *testing.T) {
	for _, target := range []string{"demo", " demo "} {
		t.Run(target, func(t *testing.T) {
			base, root, b, path := strictRunFixture(t)
			svc, err := NewServiceWithStrictInputBindings(base, []StrictInputReadBinding{b})
			if err != nil {
				t.Fatal(err)
			}
			ref, err := sharedartifacts.RegisterArtifact(filepath.Join(root, "demo"), b.RunID, sharedartifacts.ArtifactRegistration{Kind: sharedartifacts.ArtifactKindGenericFile, StorageRoot: "run", StoragePath: "strict-inputs.json", MediaType: "application/json"})
			if err != nil {
				t.Fatal(err)
			}
			if ref.ID != b.ArtifactID {
				t.Fatal("fixture did not retain opaque ID")
			}
			before := strictReadTree(t, root)
			_, err = svc.GetRunArtifact(context.Background(), connect.NewRequest(&runspb.GetRunArtifactRequest{Target: target, RunId: b.RunID, ArtifactId: b.ArtifactID}))
			if connect.CodeOf(err) != connect.CodeFailedPrecondition {
				t.Fatalf("downgrade escaped: %v", err)
			}
			if strictReadTree(t, root) != before {
				t.Fatal("downgrade refusal wrote")
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestStrictBindingLeavesUnboundGenericArtifactReadable(t *testing.T) {
	base, root, b, _ := strictRunFixture(t)
	svc, err := NewServiceWithStrictInputBindings(base, []StrictInputReadBinding{b})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "demo")
	path := filepath.Join(sharedartifacts.RunDir(dir, b.RunID), "ordinary.txt")
	if err := os.WriteFile(path, []byte("legacy ordinary bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	ref, err := sharedartifacts.RegisterArtifact(dir, b.RunID, sharedartifacts.ArtifactRegistration{Kind: sharedartifacts.ArtifactKindGenericFile, StorageRoot: "run", StoragePath: "ordinary.txt"})
	if err != nil {
		t.Fatal(err)
	}
	before := strictReadTree(t, root)
	got, err := svc.GetRunArtifact(context.Background(), connect.NewRequest(&runspb.GetRunArtifactRequest{Target: " demo ", RunId: b.RunID, ArtifactId: ref.ID}))
	if err != nil || got.Msg.GetArtifact().GetId() != ref.ID {
		t.Fatalf("ordinary read changed: %v", err)
	}
	if strictReadTree(t, root) != before {
		t.Fatal("ordinary read wrote")
	}
}
func TestStrictInputMappingGoldenV1(t *testing.T) {
	h := strings.Repeat("a", 64)
	got, err := StrictInputTerminalDigests("execution-config:"+h, "ds:sha256:"+h, "phase-set:"+h)
	if err != nil {
		t.Fatal(err)
	}
	want := [3]string{"sha256:323c4bcba3aa4a3bbd6301f0c267e10863feb7cc4c0436506d56c37c1048680c", "sha256:d2ba41697c6830f02b041c08a2b1f82919ecd76c221955f17bd1ae2866ae993e", "sha256:e9ad04c6176c9ae165d4ab62d4ffc8345d1f0abfc2e0709e973bc7bebe7af2db"}
	if got != want {
		t.Fatalf("reference encoding changed: %v", got)
	}
}
