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
//   - an embedded harness display-name edit must keep the contract, while a
//     home-state path edit must change it;
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
		"internal/peerrecord/record.go":                 replace("internal/peerrecord/record.go", "\tif record.OwnerPID <= 0 {\n\t\treturn errors.New(\"peer record owner PID is required\")", "\tif record.OwnerPID < 1 {\n\t\treturn errors.New(\"peer record owner PID is required\")"),
		"internal/setup/bootstrap.go":                   replace("internal/setup/bootstrap.go", `if name == "" || name == setupDockerTool {`, `if len(name) == 0 || name == setupDockerTool {`),
		"packages/api-core/scenariocli/resolver.go":     replace("packages/api-core/scenariocli/resolver.go", `if root == "" {`, `if len(root) == 0 {`),
		"internal/projectstate/setup_markers.go":        strings.Replace(replace("internal/projectstate/setup_markers.go", "if os.IsNotExist(err) {", "if errors.Is(err, os.ErrNotExist) {"), "import (\n", "import (\n\t\"errors\"\n", 1),
		"internal/resources/zz_state_contract_probe.go": "package resources\n\nconst stateContractProbeLimit = 3\n\ntype stateContractProbePlan struct{ Attempts int }\n\nfunc stateContractProbeAttempts(p stateContractProbePlan) int { return min(p.Attempts+1, stateContractProbeLimit) }\n",
	})
	harnessAsset := "packages/cli-core/agentcatalog/harness/builtin.json"
	displayOnly := build("display-only", map[string]string{
		harnessAsset: replace(harnessAsset, `"display_name": "Codex"`, `"display_name": "Codex CLI"`),
	})
	homeLayout := build("home-layout", map[string]string{
		harnessAsset: replace(harnessAsset, `"state_dir": "codex"`, `"state_dir": "codex-v2"`),
	})
	shape := build("shape", map[string]string{
		"internal/process/process.go": replace("internal/process/process.go",
			"type Record struct {\n", "type Record struct {\n\tStateContractProbe string `json:\"state_contract_probe,omitempty\"`\n"),
	})
	peerShape := build("peer-shape", map[string]string{
		"internal/peerrecord/record.go":             replace("internal/peerrecord/record.go", "type Record struct {\n", "type Record struct {\n\tContractProbe string `json:\"contract_probe,omitempty\"`\n"),
		"internal/setup/progress.go":                replace("internal/setup/progress.go", "type activeSetupState struct {\n", "type activeSetupState struct {\n\tContractProbe string `json:\"contract_probe,omitempty\"`\n"),
		"packages/api-core/scenariocli/resolver.go": replace("packages/api-core/scenariocli/resolver.go", "type installMetadata struct {\n", "type installMetadata struct {\n\tContractProbe string `json:\"contract_probe,omitempty\"`\n"),
		"internal/projectstate/setup_markers.go":    replace("internal/projectstate/setup_markers.go", "type MigrationLedger struct {\n", "type MigrationLedger struct {\n\tContractProbe string `json:\"contract_probe,omitempty\"`\n"),
	})
	peerIdentity := build("peer-identity", map[string]string{
		"internal/peerrecord/identity.go":           replace("internal/peerrecord/identity.go", `"peers"`, `"peers-v2"`),
		"internal/setup/onboarding.go":              replace("internal/setup/onboarding.go", `"phase":         "bootstrap_complete"`, `"phase":         "bootstrap_done"`),
		"packages/api-core/scenariocli/identity.go": replace("packages/api-core/scenariocli/identity.go", `.build.meta`, `.build-v2.meta`),
		"internal/projectstate/identity.go":         replace("internal/projectstate/identity.go", `"migrations.json"`, `"migrations-v2.json"`),
	})
	cliHelper := build("cli-helper", map[string]string{
		"packages/cli-core/buildinfo/buildinfo.go": replace("packages/cli-core/buildinfo/buildinfo.go", `filepath.Base(filepath.FromSlash(path)) != "coverage"`, `filepath.Base(filepath.FromSlash(path)) != "coverage-v2"`),
	})
	setupInputs := build("setup-inputs", map[string]string{
		"internal/config/repair.go":             replace("internal/config/repair.go", `req.ResumeAfter = filepath.Clean(req.ResumeAfter)`, `req.ResumeAfter = filepath.Clean(req.ResumeAfter + "/next")`),
		"internal/onboardinghandoff/handoff.go": replace("internal/onboardinghandoff/handoff.go", `Decision{Action: "url"`, `Decision{Action: "url-v2"`),
	})
	servicesSlice := build("services-slice", map[string]string{
		"packages/platform-go/containment.go": replace("packages/platform-go/containment.go", "ServicesSlice = \"vrooli-services.slice\"", "ServicesSlice = \"vrooli-svc.slice\""),
	})
	unclassified := build("unclassified", map[string]string{
		"internal/resources/zz_state_contract_probe.go": "package resources\n\nconst stateContractProbeSuffix = \".pending\"\n",
	})
	vocabulary := build("vocabulary", map[string]string{
		"internal/scenarioruntime/types.go": replace("internal/scenarioruntime/types.go", "\tStatusStopping = \"stopping\"", "\tStatusStopping = \"draining\""),
		"internal/setup/readiness.go":       strings.Replace(replace("internal/setup/readiness.go", `ReadinessStatusReady       = "ready"`, `ReadinessStatusReady       = "operational"`), `ReadinessSourceInProcess   = "in_process"`, `ReadinessSourceInProcess   = "in_process_v2"`, 1),
	})
	version := build("version", map[string]string{
		"internal/cliinstall/uninstall_service.go": replace("internal/cliinstall/uninstall_service.go", "installRecordVersion = 2", "installRecordVersion = 3"),
	})
	callerConstant := build("caller-constant", map[string]string{
		"internal/credentialescrow/provider.go": strings.Replace(read("internal/credentialescrow/provider.go"), "selected.Status != operatorcapability.CandidateStatusReady", "selected.Status != providerReady", 1),
	})
	foreignConstant := build("foreign-constant", map[string]string{
		"internal/credentialescrow/provider.go": strings.Replace(read("internal/credentialescrow/provider.go"), "selected.Status != operatorcapability.CandidateStatusReady", "selected.Status != config.TemplateBaseDirEnvVar", 1),
	})
	credentialOwnerEdit := build("credential-owner-edit", map[string]string{
		"internal/credentialescrow/provider.go": strings.Replace(read("internal/credentialescrow/provider.go"), "selected.Status != operatorcapability.CandidateStatusReady", "selected.Status != securestore.CopyScheduleReady", 1),
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
	if contract, err := probe(callerConstant); err != nil || contract != "" {
		t.Fatalf("native executable accepted a caller-owned stored vocabulary constant: contract %q, error %v", contract, err)
	}
	if contract, err := probe(foreignConstant); err != nil || contract != "" {
		t.Fatalf("native executable accepted an unstamped foreign vocabulary constant: contract %q, error %v", contract, err)
	}
	baseID, err := coreactivation.ProbeRootIdentity(ctx, identity(base))
	if err != nil {
		t.Fatal(err)
	}
	credentialOwnerID, err := coreactivation.ProbeRootIdentity(ctx, identity(credentialOwnerEdit))
	if err != nil {
		t.Fatal(err)
	}
	credentialOwnerRefusal := coreactivation.QualifyRollbackIdentities(baseID, credentialOwnerID)
	if !errors.Is(credentialOwnerRefusal, coreactivation.ErrStateContractChanged) || !strings.Contains(coreactivation.ExplainRollbackRefusal(ctx, credentialOwnerRefusal, base, credentialOwnerEdit).Error(), "owner credential-recovery: source changed") {
		t.Fatalf("a behavior edit within the whole-source credential owner was not attributed: %v", credentialOwnerRefusal)
	}
	behaviourID, err := coreactivation.ProbeRootIdentity(ctx, identity(behaviour))
	if err != nil {
		t.Fatal(err)
	}
	displayOnlyID, err := coreactivation.ProbeRootIdentity(ctx, identity(displayOnly))
	if err != nil {
		t.Fatal(err)
	}
	homeLayoutID, err := coreactivation.ProbeRootIdentity(ctx, identity(homeLayout))
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
	if err := coreactivation.QualifyRollbackIdentities(baseID, displayOnlyID); err != nil {
		t.Fatalf("display-only harness catalog change was refused: %v", coreactivation.ExplainRollbackRefusal(ctx, err, base, displayOnly))
	}
	assetRefusal := coreactivation.QualifyRollbackIdentities(baseID, homeLayoutID)
	if !errors.Is(assetRefusal, coreactivation.ErrStateContractChanged) || !strings.Contains(coreactivation.ExplainRollbackRefusal(ctx, assetRefusal, base, homeLayout).Error(), "owner agent-catalog-assets: source changed") {
		t.Fatalf("harness home-layout change was not attributed to its asset owner: %v", coreactivation.ExplainRollbackRefusal(ctx, assetRefusal, base, homeLayout))
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
	// Exercise the actual coordinator admission path, including its forensic
	// observation, with isolated images and no native supervisor mutation.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	layout, err := rootUpdateLayout(home)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(layout.Executable), 0700); err != nil {
		t.Fatal(err)
	}
	imageBytes, err := os.ReadFile(base)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.Executable, imageBytes, 0700); err != nil {
		t.Fatal(err)
	}
	record, err := SubmitRootUpdate(ctx, home, RootUpdateRequest{Source: root, Candidate: shape, RequireEnrollment: true})
	if !errors.Is(err, coreactivation.ErrStateContractChanged) || record.ID != "" {
		t.Fatalf("expected pre-admission contract refusal, got record=%+v err=%v", record, err)
	}
	state := filepath.Join(home, ".vrooli", "state", "root-rebaseline", "attempts")
	entries, err := os.ReadDir(state)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected one admission observation: %v %v", entries, err)
	}
	data, err := os.ReadFile(filepath.Join(state, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	var observation struct {
		Kind              string                           `json:"kind"`
		ServingContract   string                           `json:"serving_contract"`
		CandidateContract string                           `json:"candidate_contract"`
		Attribution       string                           `json:"attribution"`
		Owners            []string                         `json:"owners"`
		Events            []struct{ Stage, Reason string } `json:"events"`
	}
	if err := json.Unmarshal(data, &observation); err != nil {
		t.Fatal(err)
	}
	if observation.Kind != "install" || observation.ServingContract != baseID.GetPersistentStateContractSha256() || observation.CandidateContract != shapeID.GetPersistentStateContractSha256() || observation.Attribution != "compared" || !strings.Contains(strings.Join(observation.Owners, ","), "process-records") {
		t.Fatalf("incorrect admission attribution: %s", data)
	}
	if len(observation.Events) < 2 || observation.Events[len(observation.Events)-2].Stage != "admission_refused" || observation.Events[len(observation.Events)-2].Reason != "state_contract_changed" || observation.Events[len(observation.Events)-1].Reason != "failed" {
		t.Fatalf("incorrect admission outcome: %s", data)
	}
	if contract, err := probe(base); err != nil || contract != NativePersistentStateContract() {
		t.Fatalf("actual governed image does not report this checkout's contract: %s %v", contract, err)
	}
	for image, want := range map[string]string{
		peerShape:     `owner peer-records: stored type internal/peerrecord.Record field contract_probe added`,
		peerIdentity:  "owner peer-records: source changed",
		cliHelper:     "owner scenario-cli-install: source changed",
		setupInputs:   "owner setup-state: source changed",
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
	for image, want := range map[string]string{
		// Each image changes multiple owners, asserted separately below.
		peerShape:    `owner project-setup: stored type internal/projectstate.MigrationLedger field contract_probe added`,
		peerIdentity: "owner project-setup: source changed",
	} {
		id, err := coreactivation.ProbeRootIdentity(ctx, identity(image))
		if err != nil {
			t.Fatal(err)
		}
		refusal := coreactivation.QualifyRollbackIdentities(baseID, id)
		if explained := coreactivation.ExplainRollbackRefusal(ctx, refusal, base, image); !errors.Is(refusal, coreactivation.ErrStateContractChanged) || !strings.Contains(explained.Error(), want) {
			t.Fatalf("project format change was not refused precisely: %v", explained)
		}
	}
	for image, want := range map[string]string{
		peerShape:    `owner scenario-cli-install: stored type api-core/scenariocli.installMetadata field contract_probe added`,
		peerIdentity: "owner scenario-cli-install: source changed",
	} {
		id, err := coreactivation.ProbeRootIdentity(ctx, identity(image))
		if err != nil {
			t.Fatal(err)
		}
		refusal := coreactivation.QualifyRollbackIdentities(baseID, id)
		if explained := coreactivation.ExplainRollbackRefusal(ctx, refusal, base, image); !errors.Is(refusal, coreactivation.ErrStateContractChanged) || !strings.Contains(explained.Error(), want) {
			t.Fatalf("scenario CLI format change was not refused precisely: %v", explained)
		}
	}
	for image, want := range map[string]string{
		peerShape:    `owner setup-state: stored type internal/setup.activeSetupState field contract_probe added`,
		peerIdentity: "owner setup-state: source changed",
	} {
		id, err := coreactivation.ProbeRootIdentity(ctx, identity(image))
		if err != nil {
			t.Fatal(err)
		}
		refusal := coreactivation.QualifyRollbackIdentities(baseID, id)
		if explained := coreactivation.ExplainRollbackRefusal(ctx, refusal, base, image); !errors.Is(refusal, coreactivation.ErrStateContractChanged) || !strings.Contains(explained.Error(), want) {
			t.Fatalf("setup format change was not refused precisely: %v", explained)
		}
	}
	for _, want := range []string{"owner setup-state: stored constant ReadinessStatusReady changed", "owner setup-state: stored constant ReadinessSourceInProcess changed"} {
		id, err := coreactivation.ProbeRootIdentity(ctx, identity(vocabulary))
		if err != nil {
			t.Fatal(err)
		}
		refusal := coreactivation.QualifyRollbackIdentities(baseID, id)
		if explained := coreactivation.ExplainRollbackRefusal(ctx, refusal, base, vocabulary); !errors.Is(refusal, coreactivation.ErrStateContractChanged) || !strings.Contains(explained.Error(), want) {
			t.Fatalf("setup vocabulary change was not refused precisely: %v", explained)
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
			if strings.HasPrefix(pattern, "!") {
				continue
			}
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
	copyFile("go.sum")
	copyFile("packages/cli-core/agentcatalog/harness/builtin.json")
	// Stamp-time caller qualification and the acquisition generator inventory
	// inspect these trees recursively, including nested modules.
	for _, tree := range []string{"internal", "cmd", "packages/resource-deployment", "packages/binaryfetch", "packages/platform-go"} {
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(tree)), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if strings.HasSuffix(entry.Name(), ".go") || entry.Name() == "go.mod" {
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				copyFile(filepath.ToSlash(rel))
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
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
