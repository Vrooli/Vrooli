package passivefidelityqualification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	ContractRow = "passive-fidelity"
	EvidenceDir = ".vrooli/runtime/rehabilitation-evidence"
)

var managedSources = []string{
	"docs/internal/REFRACTOR_CONTRACT.json",
	"playwright-driver/tests/integration/saved-workflow-fresh-context.test.ts",
	"playwright-driver/src/recording/capture/browser-scripts/recording-script.js",
	"playwright-driver/src/recording/orchestration/pipeline-manager.ts",
	"api/handlers/record_mode.go",
	"api/services/recording/service.go",
	"api/services/recording/persistence/sqlite.go",
}

var crashSources = []string{
	"docs/internal/REFRACTOR_CONTRACT.json",
	"api/services/recording/service.go",
	"api/services/recording/service_test.go",
	"api/services/recording/persistence/sqlite.go",
}

var semanticsSources = []string{
	"docs/internal/REFRACTOR_CONTRACT.json",
	"playwright-driver/tests/integration/pipeline-e2e.test.ts",
	"playwright-driver/src/recording/capture/browser-scripts/recording-script.js",
	"playwright-driver/src/recording/orchestration/pipeline-manager.ts",
	"playwright-driver/src/proto/recording.ts",
}

type managedEvidence struct {
	ContractRow   string            `json:"contractRow"`
	Result        string            `json:"result"`
	SourceSHA256  map[string]string `json:"source_sha256"`
	OwnerArtifact artifactRef       `json:"owner_artifact"`
	OwnerReceipt  struct {
		ContractRow                string `json:"contractRow"`
		ManagedBuildIdentityBefore string `json:"managedBuildIdentityBefore"`
		ManagedBuildIdentityAfter  string `json:"managedBuildIdentityAfter"`
		Actions                    int    `json:"actions"`
		FixtureEffects             int    `json:"fixtureEffects"`
		UniqueJournalIDs           int    `json:"uniqueJournalIds"`
		StrictlyIncreasingJournal  bool   `json:"strictlyIncreasingJournalSequence"`
		AppliedInputReceipts       int    `json:"appliedInputReceipts"`
		StrictlyIncreasingApplied  bool   `json:"strictlyIncreasingAppliedSequence"`
		StorageIsolation           struct {
			RoutedTestPool                bool `json:"routedTestPool"`
			TestPoolRequests              int  `json:"testPoolRequests"`
			PrimaryRequestsDuringTestMode int  `json:"primaryRequestsDuringTestMode"`
			TemporaryDatabase             bool `json:"temporaryDatabase"`
		} `json:"storageIsolation"`
	} `json:"owner_receipt"`
}

type crashEvidence struct {
	ContractRow   string            `json:"contractRow"`
	Result        string            `json:"result"`
	Tests         string            `json:"tests"`
	OwnerTest     string            `json:"ownerTest"`
	SourceSHA256  map[string]string `json:"source_sha256"`
	OwnerArtifact artifactRef       `json:"owner_artifact"`
	Owner         struct {
		Case                           string `json:"case"`
		ActionsBeforeCrash             int    `json:"actionsBeforeCrash"`
		CommittedBeforeAcknowledgment  bool   `json:"committedBeforeAcknowledgment"`
		ChildTerminatedAbruptly        bool   `json:"childTerminatedAbruptly"`
		ReopenedTotal                  int    `json:"reopenedTotal"`
		RetriedSameEventID             bool   `json:"retriedSameEventId"`
		TotalAfterRetry                int    `json:"totalAfterRetry"`
		ExpectedPrefixIntactAndOrdered bool   `json:"expectedPrefixIntactAndOrdered"`
	} `json:"owner"`
}

type semanticsEvidence struct {
	ContractRow   string            `json:"contractRow"`
	Result        string            `json:"result"`
	Tests         string            `json:"tests"`
	OwnerTests    []string          `json:"ownerTests"`
	SourceSHA256  map[string]string `json:"source_sha256"`
	OwnerArtifact artifactRef       `json:"owner_artifact"`
	Cases         []struct {
		Case        string   `json:"case"`
		ActionTypes []string `json:"actionTypes"`
		Assertions  []string `json:"assertions"`
	} `json:"cases"`
}

type artifactRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Validate joins independently produced managed, crash/reconnect, and browser
// semantics receipts. Each receipt is bound to the exact contract and owner
// source, and the managed cohort must match the currently running build.
func Validate(scenarioRoot, liveBuild string) error {
	if strings.TrimSpace(liveBuild) == "" {
		return fmt.Errorf("live managed build identity is empty")
	}
	contractSHA, err := fileSHA(filepath.Join(scenarioRoot, "docs/internal/REFRACTOR_CONTRACT.json"))
	if err != nil {
		return fmt.Errorf("read passive-fidelity contract: %w", err)
	}
	managed, err := currentManagedEvidence(scenarioRoot, liveBuild)
	if err != nil {
		return err
	}
	if err := validateSourceSet(scenarioRoot, managed.SourceSHA256, managedSources, contractSHA); err != nil {
		return fmt.Errorf("managed cohort: %w", err)
	}
	if err := validateArtifact(scenarioRoot, managed.OwnerArtifact); err != nil {
		return fmt.Errorf("managed cohort owner artifact: %w", err)
	}
	owner := managed.OwnerReceipt
	if managed.ContractRow != ContractRow || managed.Result != "passed" || owner.ContractRow != ContractRow ||
		owner.ManagedBuildIdentityBefore != liveBuild || owner.ManagedBuildIdentityAfter != liveBuild {
		return fmt.Errorf("managed cohort contract, result, or build identity is invalid")
	}
	if owner.Actions < 10000 || owner.FixtureEffects != owner.Actions || owner.UniqueJournalIDs != owner.Actions ||
		!owner.StrictlyIncreasingJournal || owner.AppliedInputReceipts != owner.Actions || !owner.StrictlyIncreasingApplied {
		return fmt.Errorf("managed cohort lost fixture effects, journal identity/order, or applied-input receipts")
	}
	storage := owner.StorageIsolation
	if !storage.RoutedTestPool || storage.TestPoolRequests < 1 || storage.PrimaryRequestsDuringTestMode != 0 || !storage.TemporaryDatabase {
		return fmt.Errorf("managed cohort did not prove isolated temporary storage")
	}

	crash, err := latestCrashEvidence(scenarioRoot)
	if err != nil {
		return err
	}
	if crash.ContractRow != ContractRow || crash.Result != "passed" || crash.Tests != "1/1" ||
		crash.OwnerTest != "TestJournalSameIDRetryRecoversAcrossServiceProcessDeath" {
		return fmt.Errorf("process crash owner did not pass its focused test")
	}
	if err := validateSourceSet(scenarioRoot, crash.SourceSHA256, crashSources, contractSHA); err != nil {
		return fmt.Errorf("process crash owner: %w", err)
	}
	if err := validateArtifact(scenarioRoot, crash.OwnerArtifact); err != nil {
		return fmt.Errorf("process crash owner artifact: %w", err)
	}
	if crash.Owner.Case != "recording-service-process-death" || crash.Owner.ActionsBeforeCrash < 10000 ||
		!crash.Owner.CommittedBeforeAcknowledgment || !crash.Owner.ChildTerminatedAbruptly ||
		crash.Owner.ReopenedTotal != crash.Owner.ActionsBeforeCrash+1 || !crash.Owner.RetriedSameEventID ||
		crash.Owner.TotalAfterRetry != crash.Owner.ReopenedTotal || !crash.Owner.ExpectedPrefixIntactAndOrdered {
		return fmt.Errorf("process crash owner did not prove durable same-ID reconnect without duplication")
	}

	semantics, err := latestSemanticsEvidence(scenarioRoot)
	if err != nil {
		return err
	}
	if semantics.ContractRow != ContractRow || semantics.Result != "passed" || semantics.Tests != "3/3" ||
		!contains(semantics.OwnerTests, "[CRITICAL] should capture all core event types in single session") ||
		!contains(semantics.OwnerTests, "[CRITICAL] should capture navigation events") ||
		!contains(semantics.OwnerTests, "[CRITICAL] should continue capturing events after navigation") {
		return fmt.Errorf("browser semantics owner did not pass all focused cases")
	}
	if err := validateSourceSet(scenarioRoot, semantics.SourceSHA256, semanticsSources, contractSHA); err != nil {
		return fmt.Errorf("browser semantics owner: %w", err)
	}
	if err := validateArtifact(scenarioRoot, semantics.OwnerArtifact); err != nil {
		return fmt.Errorf("browser semantics owner artifact: %w", err)
	}
	return validateSemanticCases(semantics.Cases)
}

func currentManagedEvidence(scenarioRoot, liveBuild string) (managedEvidence, error) {
	paths, err := evidencePaths(scenarioRoot, "passive-fidelity-managed-*.json")
	if err != nil {
		return managedEvidence{}, err
	}
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var candidate managedEvidence
		if json.Unmarshal(data, &candidate) != nil || candidate.OwnerReceipt.ManagedBuildIdentityAfter != liveBuild {
			continue
		}
		return candidate, nil
	}
	return managedEvidence{}, fmt.Errorf("no managed passive-fidelity cohort matches live build %q", liveBuild)
}

func latestCrashEvidence(scenarioRoot string) (crashEvidence, error) {
	paths, err := evidencePaths(scenarioRoot, "passive-fidelity-process-crash-*.json")
	if err != nil {
		return crashEvidence{}, err
	}
	return readFirst[crashEvidence](paths, "current process crash/reconnect evidence")
}

func latestSemanticsEvidence(scenarioRoot string) (semanticsEvidence, error) {
	paths, err := evidencePaths(scenarioRoot, "passive-fidelity-semantics-*.json")
	if err != nil {
		return semanticsEvidence{}, err
	}
	return readFirst[semanticsEvidence](paths, "current browser semantics evidence")
}

func readFirst[T any](paths []string, label string) (T, error) {
	var zero T
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var result T
		if json.Unmarshal(data, &result) == nil {
			return result, nil
		}
	}
	return zero, fmt.Errorf("no %s receipt", label)
}

func evidencePaths(scenarioRoot, pattern string) ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(scenarioRoot, EvidenceDir, pattern))
	if err != nil {
		return nil, err
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	return paths, nil
}

func validateSourceSet(root string, hashes map[string]string, required []string, contractSHA string) error {
	for _, rel := range required {
		want, ok := hashes[rel]
		if !ok || len(want) != sha256.Size*2 {
			return fmt.Errorf("receipt lacks valid source digest: %s", rel)
		}
		got, err := fileSHA(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("read source %s: %w", rel, err)
		}
		if want != got {
			return fmt.Errorf("source digest mismatch: %s", rel)
		}
	}
	if hashes["docs/internal/REFRACTOR_CONTRACT.json"] != contractSHA {
		return fmt.Errorf("contract digest mismatch")
	}
	return nil
}

func validateArtifact(root string, artifact artifactRef) error {
	if strings.TrimSpace(artifact.Path) == "" || len(artifact.SHA256) != sha256.Size*2 || filepath.IsAbs(artifact.Path) {
		return fmt.Errorf("owner artifact path or digest is invalid")
	}
	clean := filepath.Clean(filepath.FromSlash(artifact.Path))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("owner artifact escapes scenario root")
	}
	got, err := fileSHA(filepath.Join(root, clean))
	if err != nil {
		return fmt.Errorf("read %s: %w", artifact.Path, err)
	}
	if got != artifact.SHA256 {
		return fmt.Errorf("owner artifact digest mismatch: %s", artifact.Path)
	}
	return nil
}

func validateSemanticCases(cases []struct {
	Case        string   `json:"case"`
	ActionTypes []string `json:"actionTypes"`
	Assertions  []string `json:"assertions"`
},
) error {
	required := map[string]struct {
		actions    []string
		assertions []string
	}{
		"core-events": {
			actions:    []string{"click", "type", "scroll"},
			assertions: []string{"one independent click effect", "typed value and input selector retained", "positive scroll delta retained", "unique increasing sequence numbers"},
		},
		"navigation": {
			actions:    []string{"click", "navigate"},
			assertions: []string{"click triggering navigation retained", "navigation entry identifies /page-2"},
		},
		"capture-after-navigation": {
			actions:    []string{"click", "navigate"},
			assertions: []string{"pre-navigation click retained", "navigation completed", "post-navigation click retained"},
		},
	}
	if len(cases) != len(required) {
		return fmt.Errorf("browser semantics receipt has %d cases, want %d", len(cases), len(required))
	}
	seen := make(map[string]bool, len(cases))
	for _, observation := range cases {
		want, ok := required[observation.Case]
		if !ok || seen[observation.Case] {
			return fmt.Errorf("unexpected or duplicate browser semantics case %q", observation.Case)
		}
		seen[observation.Case] = true
		for _, action := range want.actions {
			if !contains(observation.ActionTypes, action) {
				return fmt.Errorf("browser semantics case %s lacks %s", observation.Case, action)
			}
		}
		for _, assertion := range want.assertions {
			if !contains(observation.Assertions, assertion) {
				return fmt.Errorf("browser semantics case %s lacks fixture assertion %q", observation.Case, assertion)
			}
		}
	}
	return nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func fileSHA(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
