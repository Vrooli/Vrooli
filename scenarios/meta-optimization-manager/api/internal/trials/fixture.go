package trials

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Fixture is the committed substrate for one trial family: a self-contained
// minimal target codebase, a deterministic oracle that decides "solved", and a
// spec the Runner turns into the agent prompt. The Guide space defines WHICH
// families exist; fixtures define WHAT "solved" means, so the verdict is
// deterministic and the trend is comparable over time.
//
// On-disk layout (trials/fixtures/<family>/):
//
//	fixture.json   metadata (family, negative, prompt_file, oracle, target_dir)
//	spec.md        the agent prompt — the concrete task to perform on target/
//	check.sh       the oracle (run in a COPY of target/ with the agent's diff
//	               applied); exit 0 = solved. Absent for negative fixtures.
//	target/        the minimal codebase the agent edits (the agent's scope-path)
//
// The oracle lives OUTSIDE target/ so the agent (scoped to target/) cannot read
// it and game the check.
type Fixture struct {
	Family    string   // suite family this fixture represents
	Dir       string   // fixture dir (spec/check/target live here)
	TargetDir string   // abs path to target/ — the agent's scope-path
	Rev       string   // content revision of the fixture inputs (idempotency key)
	Prompt    string   // spec content — becomes the agent task description
	Oracle    []string // deterministic check command; run in a diff-applied copy of target/
	Negative  bool     // honesty/abstention fixture (pass = correct abstention)
}

// FixtureResolver maps a generated TrialTask to the committed fixture that
// defines its success condition. Declared at the consumer (seam-discovery);
// production reads the trials/fixtures/<family>/ corpus, tests fake it. Resolve
// returns ok=false when no fixture is registered for the task's family (the
// service then degrades that one run to VerdictError, never the suite).
type FixtureResolver interface {
	Resolve(ctx context.Context, task TrialTask) (Fixture, bool, error)
}

// fixtureManifest is the on-disk fixture.json shape.
type fixtureManifest struct {
	Family     string   `json:"family"`
	Negative   bool     `json:"negative"`
	PromptFile string   `json:"prompt_file"` // default "spec.md"
	Oracle     []string `json:"oracle"`      // e.g. ["bash","check.sh"]; empty for negative
	TargetDir  string   `json:"target_dir"`  // default "target"
}

// fileFixtureResolver reads the committed fixture corpus from the scenario tree.
type fileFixtureResolver struct {
	root string // trials/fixtures dir; empty → resolve lazily via repo-contract
}

// NewFixtureResolver returns the production resolver. It locates
// scenarios/meta-optimization-manager/trials/fixtures via the repo contract on
// first use (consistent with the coverage space reader).
func NewFixtureResolver() FixtureResolver { return &fileFixtureResolver{} }

// NewFixtureResolverWithRoot returns a resolver rooted at an explicit
// trials/fixtures directory (tests, or an overridden layout).
func NewFixtureResolverWithRoot(fixturesRoot string) FixtureResolver {
	return &fileFixtureResolver{root: fixturesRoot}
}

var _ FixtureResolver = (*fileFixtureResolver)(nil)

// fixtureScenario is the owning scenario whose tree holds the fixture corpus.
const fixtureScenario = "meta-optimization-manager"

func (r *fileFixtureResolver) fixturesRoot() (string, error) {
	if strings.TrimSpace(r.root) != "" {
		return r.root, nil
	}
	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return "", fmt.Errorf("trials: locate repo root: %w", err)
	}
	scenarioRoot, err := repocontract.ResolveScenarioPath(repoRoot, fixtureScenario)
	if err != nil {
		return "", fmt.Errorf("trials: resolve %s: %w", fixtureScenario, err)
	}
	return filepath.Join(scenarioRoot, "trials", "fixtures"), nil
}

// familyFor picks the fixture family for a task. Negative/honesty tasks all map
// to the single negative fixture; positive tasks map to their suite family.
func familyFor(task TrialTask) string {
	if task.Negative || task.Suite == SuiteNegative {
		return SuiteNegative
	}
	return task.Suite
}

func (r *fileFixtureResolver) Resolve(_ context.Context, task TrialTask) (Fixture, bool, error) {
	root, err := r.fixturesRoot()
	if err != nil {
		return Fixture{}, false, err
	}
	family := familyFor(task)
	dir := filepath.Join(root, family)
	manifestPath := filepath.Join(dir, "fixture.json")
	raw, err := os.ReadFile(manifestPath)
	if isNotExist(err) {
		return Fixture{}, false, nil // no fixture registered for this family
	}
	if err != nil {
		return Fixture{}, false, fmt.Errorf("trials: read %s: %w", manifestPath, err)
	}
	var m fixtureManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return Fixture{}, false, fmt.Errorf("trials: parse %s: %w", manifestPath, err)
	}
	return buildFixture(dir, family, m)
}

// buildFixture assembles a Fixture from a parsed manifest, reading the prompt
// and computing the content revision. Shared by the file resolver and tests.
func buildFixture(dir, family string, m fixtureManifest) (Fixture, bool, error) {
	promptFile := m.PromptFile
	if promptFile == "" {
		promptFile = "spec.md"
	}
	targetSub := m.TargetDir
	if targetSub == "" {
		targetSub = "target"
	}
	prompt, err := os.ReadFile(filepath.Join(dir, promptFile))
	if err != nil {
		return Fixture{}, false, fmt.Errorf("trials: read fixture prompt %s/%s: %w", family, promptFile, err)
	}
	targetDir := filepath.Join(dir, targetSub)
	rev, err := fixtureRev(dir, promptFile, targetSub, m.Oracle)
	if err != nil {
		return Fixture{}, false, fmt.Errorf("trials: revision %s: %w", family, err)
	}
	negative := m.Negative || family == SuiteNegative
	return Fixture{
		Family:    family,
		Dir:       dir,
		TargetDir: targetDir,
		Rev:       rev,
		Prompt:    strings.TrimSpace(string(prompt)),
		Oracle:    m.Oracle,
		Negative:  negative,
	}, true, nil
}

// fixtureRev is a stable content hash over the fixture inputs (prompt + oracle
// command + every file under target/). Editing any of them changes the rev,
// which invalidates the (task, model, fixture-rev) idempotency key so the trend
// never compares runs across incompatible fixture versions.
func fixtureRev(dir, promptFile, targetSub string, oracle []string) (string, error) {
	h := sha256.New()
	// Prompt + oracle command participate in the revision.
	promptBytes, err := os.ReadFile(filepath.Join(dir, promptFile))
	if err != nil {
		return "", err
	}
	fmt.Fprintf(h, "prompt:%x\n", sha256.Sum256(promptBytes))
	fmt.Fprintf(h, "oracle:%s\n", strings.Join(oracle, "\x00"))

	// Every file under target/, in a deterministic order.
	targetDir := filepath.Join(dir, targetSub)
	var files []string
	walkErr := filepath.WalkDir(targetDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, p)
		return nil
	})
	if walkErr != nil && !isNotExist(walkErr) {
		return "", walkErr
	}
	sort.Strings(files)
	for _, p := range files {
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		rel, _ := filepath.Rel(targetDir, p)
		fmt.Fprintf(h, "file:%s:%x\n", filepath.ToSlash(rel), sha256.Sum256(b))
	}
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:12]), nil // 24 hex chars — short but collision-safe here
}

// isNotExist is a tiny helper so callers read cleanly without importing
// errors purely for one os.IsNotExist check at several sites.
func isNotExist(err error) bool { return err != nil && os.IsNotExist(err) }
