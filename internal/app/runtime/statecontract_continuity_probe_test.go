package runtimeapp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/statecontract"
)

// This opt-in drill builds the actual root CLI from this checkout three times
// with the governed stamps and compares the real images through the
// activation admission check:
//
//   - a behaviour-only variant (a changed persistence-owner function body plus
//     a new helper type, function and untyped constant in that owner) must
//     keep the contract and be admitted;
//   - variants with one added stored field, one changed stored status value
//     and one bumped format version must each be refused, naming the owner and
//     the stored type and field or the fact.
//
// Variants use `go build -overlay`, so the checkout is never modified. No
// serving image, unit, registry or activation store is touched.
func TestActualBuildsAdmitBehaviourOnlyChangesAndRefuseStoredShapeChanges(t *testing.T) {
	if os.Getenv("VROOLI_STATE_CONTRACT_BUILD_PROBE") != "1" {
		t.Skip("set VROOLI_STATE_CONTRACT_BUILD_PROBE=1 to build and compare actual root CLI variants")
	}
	root := withRepositoryCodecStamp(t)
	homeFlag, err := buildinfo.RuntimeHomeBuildFlag(root)
	if err != nil {
		t.Fatal(err)
	}
	ldflags := "-X github.com/vrooli/vrooli/internal/buildinfo.Fingerprint=" + strings.Repeat("a", 64) + " " + homeFlag
	work := t.TempDir()
	read := func(rel string) string {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	replace := func(rel, old, new string) string {
		source := read(rel)
		if strings.Count(source, old) != 1 {
			t.Fatalf("overlay anchor %q is not unique in %s", old, rel)
		}
		return strings.Replace(source, old, new, 1)
	}
	build := func(name string, overlay map[string]string) string {
		t.Helper()
		output := filepath.Join(work, name, "vrooli")
		// Governed builders stamp from the same tree they compile; stamp a
		// copy of the inventoried sources with this variant's edits applied.
		stateFlag, err := buildinfo.PersistentStateBuildFlag(stampedSourceCopy(t, root, filepath.Join(work, name+"-stamp-root"), overlay))
		if err != nil {
			t.Fatal(err)
		}
		args := []string{"build", "-trimpath", "-ldflags", ldflags + " " + stateFlag, "-o", output}
		if len(overlay) > 0 {
			spec := struct{ Replace map[string]string }{Replace: map[string]string{}}
			for rel, content := range overlay {
				path := filepath.Join(work, name+"-overlay", filepath.FromSlash(rel))
				if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
				spec.Replace[filepath.Join(root, filepath.FromSlash(rel))] = path
			}
			data, err := json.Marshal(spec)
			if err != nil {
				t.Fatal(err)
			}
			specPath := filepath.Join(work, name+"-overlay.json")
			if err := os.WriteFile(specPath, data, 0o600); err != nil {
				t.Fatal(err)
			}
			args = append(args, "-overlay", specPath)
		}
		cmd := exec.Command("go", append(args, "./cmd/vrooli")...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", name, err, out)
		}
		return output
	}
	base := build("base", nil)
	behaviour := build("behaviour", map[string]string{
		"internal/resources/managed_service_acquisition.go": replace("internal/resources/managed_service_acquisition.go",
			"\treturn path + \".pending\"\n", "\tstaged := path + \".pending\"\n\treturn staged\n"),
		"internal/resources/zz_state_contract_probe.go": "package resources\n\nconst stateContractProbeLimit = 3\n\ntype stateContractProbePlan struct{ Attempts int }\n\nfunc stateContractProbeAttempts(p stateContractProbePlan) int { return min(p.Attempts+1, stateContractProbeLimit) }\n",
	})
	shape := build("shape", map[string]string{
		"internal/process/process.go": replace("internal/process/process.go",
			"type Record struct {\n", "type Record struct {\n\tStateContractProbe string `json:\"state_contract_probe,omitempty\"`\n"),
	})
	servicesSlice := build("services-slice", map[string]string{
		"packages/platform-go/containment.go": replace("packages/platform-go/containment.go", "ServicesSlice = \"vrooli-services.slice\"", "ServicesSlice = \"vrooli-svc.slice\""),
	})
	unclassified := build("unclassified", map[string]string{
		"internal/resources/zz_state_contract_probe.go": "package resources\n\nconst stateContractProbeSuffix = \".pending\"\n",
	})
	vocabulary := build("vocabulary", map[string]string{
		"internal/scenarioruntime/types.go": replace("internal/scenarioruntime/types.go", "\tStatusStopping = \"stopping\"", "\tStatusStopping = \"draining\""),
	})
	version := build("version", map[string]string{
		"internal/cliinstall/uninstall_service.go": replace("internal/cliinstall/uninstall_service.go", "installRecordVersion = 2", "installRecordVersion = 3"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	identity := func(image string) string {
		t.Helper()
		if err := coreactivation.ValidateRootExecutableRole(image); err != nil {
			t.Fatal(err)
		}
		return image
	}
	probe := func(image string) (string, error) {
		id, err := coreactivation.ProbeRootIdentity(ctx, identity(image))
		if err != nil {
			return "", err
		}
		return id.GetPersistentStateContractSha256(), nil
	}
	baseID, err := coreactivation.ProbeRootIdentity(ctx, identity(base))
	if err != nil {
		t.Fatal(err)
	}
	behaviourID, err := coreactivation.ProbeRootIdentity(ctx, identity(behaviour))
	if err != nil {
		t.Fatal(err)
	}
	shapeID, err := coreactivation.ProbeRootIdentity(ctx, identity(shape))
	if err != nil {
		t.Fatal(err)
	}
	if err := coreactivation.QualifyRollbackIdentities(baseID, behaviourID); err != nil {
		t.Fatalf("behaviour-only persistence-owner change was refused: %v", coreactivation.ExplainRollbackRefusal(ctx, err, base, behaviour))
	}
	refusal := coreactivation.QualifyRollbackIdentities(baseID, shapeID)
	if !errors.Is(refusal, coreactivation.ErrStateContractChanged) {
		t.Fatalf("stored field change was not refused as a state-contract change: %v", refusal)
	}
	explained := coreactivation.ExplainRollbackRefusal(ctx, refusal, base, shape).Error()
	want := `owner process-records: stored type internal/process.Record field state_contract_probe added (json "state_contract_probe,omitempty", type string)`
	if !strings.Contains(explained, want) || strings.Count(explained, "\n  - ") != 1 {
		t.Fatalf("refusal is not precise:\n%s", explained)
	}
	if contract, err := probe(base); err != nil || contract != NativePersistentStateContract() {
		t.Fatalf("actual governed image does not report this checkout's contract: %s %v", contract, err)
	}
	for image, want := range map[string]string{
		vocabulary:    "owner runtime-registry: package constants changed",
		unclassified:  "owner resources: stored constant stateContractProbeSuffix added (if it is never persisted, classify it in statecontract.NotStoredConstants)",
		servicesSlice: "owner native-operation-identities: source changed",
		version:       "owner installation-inventory: format fact installRecordVersion changed",
	} {
		id, err := coreactivation.ProbeRootIdentity(ctx, identity(image))
		if err != nil {
			t.Fatal(err)
		}
		refusal := coreactivation.QualifyRollbackIdentities(baseID, id)
		if explained := coreactivation.ExplainRollbackRefusal(ctx, refusal, base, image); !errors.Is(refusal, coreactivation.ErrStateContractChanged) || !strings.Contains(explained.Error(), want) {
			t.Fatalf("stored value change was not refused precisely: %v", explained)
		}
	}
	t.Logf("base_contract=%s behaviour_contract=%s shape_contract=%s refusal=%q", baseID.GetPersistentStateContractSha256(), behaviourID.GetPersistentStateContractSha256(), shapeID.GetPersistentStateContractSha256(), explained)
}

// stampedSourceCopy copies every source the persistence stamp reads (codec
// packages, whole-source and constants owners, enclosing go.mod files) into
// dir and applies the overlay, so a variant is stamped from its own edits.
func stampedSourceCopy(t *testing.T, root, dir string, overlay map[string]string) string {
	t.Helper()
	dirs := append([]string(nil), statecontract.CodecPackages...)
	for _, patterns := range statecontract.WholeSources {
		for _, pattern := range patterns {
			if strings.HasSuffix(pattern, "*") {
				pattern = filepath.Dir(pattern)
			}
			dirs = append(dirs, pattern)
		}
	}
	for _, inventory := range []map[string][]string{statecontract.ConstantSources, statecontract.PreciseSources} {
		for _, sources := range inventory {
			dirs = append(dirs, sources...)
		}
	}
	write := func(rel string, data []byte) {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	copyFile := func(rel string) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err == nil {
			write(rel, data)
		}
	}
	copyFile("go.mod")
	for _, d := range dirs {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(d)))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
				copyFile(d + "/" + entry.Name())
			}
		}
		for parent := d; parent != "." && parent != "/"; parent = filepath.Dir(parent) {
			copyFile(parent + "/go.mod")
		}
	}
	for rel, content := range overlay {
		write(rel, []byte(content))
	}
	return dir
}
