package modeltest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const formalArtifactSchemaVersion = 6

var ignoredHashDirs = map[string]bool{
	".git":          true,
	"node_modules":  true,
	"dist":          true,
	"build":         true,
	"coverage":      true,
	"_apalache-out": true,
}

type FormalArtifact struct {
	SchemaVersion   int                        `json:"schemaVersion"`
	FlowID          string                     `json:"flowId"`
	Source          FormalArtifactSource       `json:"source"`
	Commands        map[string][]string        `json:"commands"`
	States          []string                   `json:"states"`
	Events          []string                   `json:"events"`
	Transitions     []FormalArtifactTransition `json:"transitions"`
	NamedTraces     []FormalArtifactTrace      `json:"namedTraces"`
	GeneratedTraces []FormalArtifactTrace      `json:"generatedTraces"`
	Invariants      []string                   `json:"invariants"`
	GeneratedChecks []string                   `json:"generatedChecks"`
	Coverage        FormalArtifactCoverage     `json:"coverage"`
	Checks          FormalArtifactChecks       `json:"checks"`
}

type FormalArtifactSource struct {
	ContractPath        string `json:"contractPath"`
	ContractSHA256      string `json:"contractSha256"`
	GeneratorPath       string `json:"generatorPath"`
	GeneratorSHA256     string `json:"generatorSha256"`
	GeneratorVersion    int    `json:"generatorVersion"`
	ModelPath           string `json:"modelPath"`
	ModelSHA256         string `json:"modelSha256"`
	QuintVersion        string `json:"quintVersion"`
	VerificationBackend string `json:"verificationBackend"`
}

type FormalArtifactChecks struct {
	Typechecked           bool `json:"typechecked"`
	Tested                bool `json:"tested"`
	Verified              bool `json:"verified"`
	GeneratedFromContract bool `json:"generatedFromContract"`
	GeneratedFromModel    bool `json:"generatedFromModel"`
}

type FormalArtifactCoverage struct {
	TransitionMatrixComplete   bool                        `json:"transitionMatrixComplete"`
	TerminalTransitionsChecked bool                        `json:"terminalTransitionsChecked"`
	NamedTraces                FormalArtifactTraceCoverage `json:"namedTraces"`
	GeneratedTraces            FormalArtifactTraceCoverage `json:"generatedTraces"`
}

type FormalArtifactTraceCoverage struct {
	AllStatesCovered bool     `json:"allStatesCovered"`
	AllEventsCovered bool     `json:"allEventsCovered"`
	CoveredStates    []string `json:"coveredStates"`
	CoveredEvents    []string `json:"coveredEvents"`
	CoveredPairs     []string `json:"coveredPairs,omitempty"`
	AllPairsCovered  bool     `json:"allPairsCovered,omitempty"`
}

type FormalArtifactTransition struct {
	From      string `json:"from"`
	Event     string `json:"event"`
	To        string `json:"to"`
	WantError bool   `json:"wantError"`
}

type FormalArtifactTrace struct {
	Name    string                    `json:"name"`
	Initial string                    `json:"initial"`
	Steps   []FormalArtifactTraceStep `json:"steps"`
}

type FormalArtifactTraceStep struct {
	Event     string `json:"event"`
	Want      string `json:"want"`
	WantError bool   `json:"wantError"`
}

func LoadFormalArtifact(t TestingT, path string) FormalArtifact {
	t.Helper()
	// Resolve path relative to the source file that called us. The
	// generated replay.go lives next to the artifact.json it loads, but
	// `go test` runs with CWD set to the *test package's* directory
	// (e.g. flow/), not the generated subpackage's directory. Using the
	// caller's source location keeps the generated code free of any
	// awareness of where `go test` was invoked from.
	if !filepath.IsAbs(path) {
		if _, callerFile, _, ok := runtime.Caller(1); ok {
			path = filepath.Join(filepath.Dir(callerFile), path)
		}
	}
	data, err := readFormalFile(path)
	if err != nil {
		t.Fatalf("read formal artifact %s: %v", path, err)
	}
	var artifact FormalArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		t.Fatalf("parse formal artifact %s: %v", path, err)
	}
	return artifact
}

type FormalArtifactExpectation struct {
	ContractPath    string
	ModelPath       string
	GeneratorPath   string
	ContractSHA256  string
	ModelSHA256     string
	GeneratorSHA256 string
	Invariants      []string
	GeneratedChecks []string
}

// ValidateFormalArtifactFresh checks that a formal artifact is complete, fresh
// (its recorded source hashes still match the files on disk), and matches the
// caller's expectations. It returns every problem found rather than failing
// fast, so callers see the full picture in one shot. The checks are grouped into
// focused validators to keep each one independently readable and testable.
func ValidateFormalArtifactFresh(artifact FormalArtifact, expected FormalArtifactExpectation) []error {
	var errs []error
	errs = append(errs, validateArtifactIdentity(artifact, expected)...)
	errs = append(errs, validateArtifactSourceFreshness(artifact, expected)...)
	errs = append(errs, validateArtifactChecks(artifact.Checks)...)
	errs = append(errs, validateArtifactExpectations(artifact, expected)...)
	errs = append(errs, validateArtifactCoverage(artifact.Coverage)...)
	errs = append(errs, validateArtifactCollections(artifact)...)
	return errs
}

// validateArtifactIdentity checks the non-hash identity fields: schema, ids,
// declared source paths, and the required source metadata.
func validateArtifactIdentity(artifact FormalArtifact, expected FormalArtifactExpectation) []error {
	var errs []error
	if artifact.SchemaVersion != formalArtifactSchemaVersion {
		errs = append(errs, fmt.Errorf("formal artifact schemaVersion=%d, want %d", artifact.SchemaVersion, formalArtifactSchemaVersion))
	}
	if artifact.FlowID == "" {
		errs = append(errs, fmt.Errorf("formal artifact flowId is required"))
	}
	if artifact.Source.ContractPath != expected.ContractPath {
		errs = append(errs, fmt.Errorf("formal artifact contractPath=%s, want %s", artifact.Source.ContractPath, expected.ContractPath))
	}
	if artifact.Source.ModelPath != expected.ModelPath {
		errs = append(errs, fmt.Errorf("formal artifact modelPath=%s, want %s", artifact.Source.ModelPath, expected.ModelPath))
	}
	if expected.GeneratorPath != "" && artifact.Source.GeneratorPath != expected.GeneratorPath {
		errs = append(errs, fmt.Errorf("formal artifact generatorPath=%s, want %s", artifact.Source.GeneratorPath, expected.GeneratorPath))
	}
	if artifact.Source.GeneratorPath == "" {
		errs = append(errs, fmt.Errorf("formal artifact generatorPath is required"))
	}
	if artifact.Source.GeneratorVersion < 1 {
		errs = append(errs, fmt.Errorf("formal artifact generatorVersion is required"))
	}
	if artifact.Source.VerificationBackend == "" {
		errs = append(errs, fmt.Errorf("formal artifact verificationBackend is required"))
	}
	if artifact.Source.QuintVersion == "" {
		errs = append(errs, fmt.Errorf("formal artifact quintVersion is required"))
	}
	return errs
}

// validateArtifactSourceFreshness re-hashes the contract and model sources and
// compares the recorded hashes (both against disk and against any caller-pinned
// expectation). The generatorSha256 is recorded for traceability but not
// re-hashed: the flow-verifier scenario owns the generator and is external to
// the consumer scenario; `flow-verifier verify check` is authoritative for
// generator freshness.
func validateArtifactSourceFreshness(artifact FormalArtifact, expected FormalArtifactExpectation) []error {
	var errs []error
	if err := validateFreshHash(artifact.Source.ContractPath, artifact.Source.ContractSHA256, "contractSha256"); err != nil {
		errs = append(errs, err)
	}
	if expected.ContractSHA256 != "" && artifact.Source.ContractSHA256 != expected.ContractSHA256 {
		errs = append(errs, fmt.Errorf("formal artifact contractSha256=%s, want %s", artifact.Source.ContractSHA256, expected.ContractSHA256))
	}
	if expected.GeneratorSHA256 != "" && artifact.Source.GeneratorSHA256 != expected.GeneratorSHA256 {
		errs = append(errs, fmt.Errorf("formal artifact generatorSha256=%s, want %s", artifact.Source.GeneratorSHA256, expected.GeneratorSHA256))
	}
	if err := validateFreshHash(expected.ModelPath, artifact.Source.ModelSHA256, "modelSha256"); err != nil {
		errs = append(errs, err)
	}
	if expected.ModelSHA256 != "" && artifact.Source.ModelSHA256 != expected.ModelSHA256 {
		errs = append(errs, fmt.Errorf("formal artifact modelSha256=%s, want %s", artifact.Source.ModelSHA256, expected.ModelSHA256))
	}
	return errs
}

// validateArtifactChecks asserts every required pipeline stage ran.
func validateArtifactChecks(checks FormalArtifactChecks) []error {
	required := []struct {
		ok     bool
		reason string
	}{
		{checks.Typechecked, "was not typechecked"},
		{checks.Tested, "was not tested"},
		{checks.Verified, "was not verified"},
		{checks.GeneratedFromContract, "was not generated from contract"},
		{checks.GeneratedFromModel, "was not generated from model"},
	}
	var errs []error
	for _, c := range required {
		if !c.ok {
			errs = append(errs, fmt.Errorf("formal artifact %s", c.reason))
		}
	}
	return errs
}

// validateArtifactExpectations checks the caller-pinned invariants and generated
// checks are all present.
func validateArtifactExpectations(artifact FormalArtifact, expected FormalArtifactExpectation) []error {
	var errs []error
	for _, invariant := range expected.Invariants {
		if !containsString(artifact.Invariants, invariant) {
			errs = append(errs, fmt.Errorf("formal artifact missing invariant %s", invariant))
		}
	}
	for _, check := range expected.GeneratedChecks {
		if !containsString(artifact.GeneratedChecks, check) {
			errs = append(errs, fmt.Errorf("formal artifact missing generated check %s", check))
		}
	}
	return errs
}

// validateArtifactCoverage asserts the matrix/terminal/trace coverage flags.
func validateArtifactCoverage(coverage FormalArtifactCoverage) []error {
	var errs []error
	if !coverage.TransitionMatrixComplete {
		errs = append(errs, fmt.Errorf("formal artifact transition matrix is incomplete"))
	}
	if !coverage.TerminalTransitionsChecked {
		errs = append(errs, fmt.Errorf("formal artifact does not check terminal transitions"))
	}
	if !coverage.NamedTraces.AllStatesCovered {
		errs = append(errs, fmt.Errorf("formal artifact named traces do not cover all states"))
	}
	if !coverage.NamedTraces.AllEventsCovered {
		errs = append(errs, fmt.Errorf("formal artifact named traces do not cover all events"))
	}
	if len(coverage.GeneratedTraces.CoveredStates) == 0 {
		errs = append(errs, fmt.Errorf("formal artifact generated traces do not report covered states"))
	}
	if len(coverage.GeneratedTraces.CoveredEvents) == 0 {
		errs = append(errs, fmt.Errorf("formal artifact generated traces do not report covered events"))
	}
	if coverage.GeneratedTraces.CoveredPairs == nil {
		errs = append(errs, fmt.Errorf("formal artifact generated traces do not report covered pairs"))
	}
	return errs
}

// validateArtifactCollections asserts the artifact actually carries transitions
// and traces.
func validateArtifactCollections(artifact FormalArtifact) []error {
	var errs []error
	if len(artifact.Transitions) == 0 {
		errs = append(errs, fmt.Errorf("formal artifact transitions must not be empty"))
	}
	if len(artifact.NamedTraces) == 0 {
		errs = append(errs, fmt.Errorf("formal artifact namedTraces must not be empty"))
	}
	if len(artifact.GeneratedTraces) == 0 {
		errs = append(errs, fmt.Errorf("formal artifact generatedTraces must not be empty"))
	}
	return errs
}

func validateFreshHash(filePath string, wantHash string, field string) error {
	if wantHash == "" {
		return fmt.Errorf("formal artifact %s is required", field)
	}
	hash, err := repoPathSHA256(filePath)
	if err != nil {
		return fmt.Errorf("read formal artifact source %s: %v", filePath, err)
	}
	if hash != wantHash {
		return fmt.Errorf("formal artifact %s=%s, want %s", field, wantHash, hash)
	}
	return nil
}

func repoPathSHA256(repoPath string) (string, error) {
	abs, err := findRepoPath(repoPath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		data, err := readFormalFile(abs)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:]), nil
	}
	return repoTreeSHA256(abs)
}

func repoTreeSHA256(root string) (string, error) {
	var parts []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error { // #nosec G122 -- test helper hashes repo-local formal artifacts; path is filtered before read.
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if ignoredHashDirs[entry.Name()] {
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(rel)
		if strings.HasPrefix(slash, "testdata/") || strings.HasSuffix(slash, "_test.go") || strings.HasSuffix(slash, ".formal.generated.json") || strings.HasSuffix(slash, ".qnt") {
			return nil
		}
		if !(strings.HasSuffix(slash, ".go") || slash == "go.mod" || strings.HasSuffix(slash, ".schema.json")) {
			return nil
		}
		data, err := readFormalFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		parts = append(parts, slash+"\x00"+hex.EncodeToString(sum[:]))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:]), nil
}

func readFormalFile(path string) ([]byte, error) {
	clean := filepath.Clean(path)
	if clean == "." || clean == string(filepath.Separator) {
		return nil, fmt.Errorf("refusing to read invalid formal artifact path %q", path)
	}
	// #nosec G304 -- this test-only helper reads caller-provided artifact/source
	// paths after repo lookup or generated-test path resolution; production code
	// never calls it and route/provider inputs cannot reach it.
	return os.ReadFile(clean)
}

func findRepoPath(repoPath string) (string, error) {
	if filepath.IsAbs(repoPath) {
		if _, err := os.Stat(repoPath); err != nil {
			return "", err
		}
		return repoPath, nil
	}
	var firstErr error
	candidates := repoFileCandidates(repoPath)
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		for _, candidate := range candidates {
			abs := filepath.Join(dir, candidate)
			if _, err := os.Stat(abs); err == nil {
				return abs, nil
			} else if firstErr == nil {
				firstErr = err
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", firstErr
}

func repoFileCandidates(repoPath string) []string {
	candidates := []string{repoPath}
	if trimmed, ok := strings.CutPrefix(repoPath, "api/"); ok {
		candidates = append(candidates, trimmed)
	}
	candidates = append(candidates, filepath.Base(repoPath))
	return candidates
}

func AssertFormalArtifactFresh(t TestingT, artifact FormalArtifact, expected FormalArtifactExpectation) {
	t.Helper()
	if errs := ValidateFormalArtifactFresh(artifact, expected); len(errs) > 0 {
		t.Fatalf("formal artifact is stale or incomplete:\n%s", formatErrors(errs))
	}
}

func ValidateFormalTransitionsReplay[S comparable, E comparable](
	artifact FormalArtifact,
	statuses []S,
	events []E,
	transition Transition[S, E],
) []error {
	rows, errs := formalRows(artifact, statuses, events)
	if len(errs) > 0 {
		return errs
	}
	return ValidateTransitionMatrix(statuses, events, rows, transition)
}

func AssertFormalTransitionsReplay[S comparable, E comparable](
	t TestingT,
	artifact FormalArtifact,
	statuses []S,
	events []E,
	transition Transition[S, E],
) {
	t.Helper()
	if errs := ValidateFormalTransitionsReplay(artifact, statuses, events, transition); len(errs) > 0 {
		t.Fatalf("formal transition replay mismatch:\n%s", formatErrors(errs))
	}
}

func ValidateFormalTracesReplay[S comparable, E comparable](
	artifact FormalArtifact,
	statuses []S,
	events []E,
	transition Transition[S, E],
) []error {
	traces, errs := formalTraces(append(artifact.NamedTraces, artifact.GeneratedTraces...), statuses, events)
	if len(errs) > 0 {
		return errs
	}
	return ValidateTraces(traces, transition)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func AssertFormalTracesReplay[S comparable, E comparable](
	t TestingT,
	artifact FormalArtifact,
	statuses []S,
	events []E,
	transition Transition[S, E],
) {
	t.Helper()
	if errs := ValidateFormalTracesReplay(artifact, statuses, events, transition); len(errs) > 0 {
		t.Fatalf("formal trace replay mismatch:\n%s", formatErrors(errs))
	}
}

func formalRows[S comparable, E comparable](
	artifact FormalArtifact,
	statuses []S,
	events []E,
) ([]MatrixRow[S, E], []error) {
	statusByName := valuesByString(statuses)
	eventByName := valuesByString(events)
	rows := make([]MatrixRow[S, E], 0, len(artifact.Transitions))
	var errs []error
	for i, transition := range artifact.Transitions {
		bad := false
		from, ok := statusByName[transition.From]
		if !ok {
			errs = append(errs, fmt.Errorf("formal transition %d unknown from state %s", i, transition.From))
			bad = true
		}
		to, ok := statusByName[transition.To]
		if !ok {
			errs = append(errs, fmt.Errorf("formal transition %d unknown to state %s", i, transition.To))
			bad = true
		}
		event, ok := eventByName[transition.Event]
		if !ok {
			errs = append(errs, fmt.Errorf("formal transition %d unknown event %s", i, transition.Event))
			bad = true
		}
		if bad {
			continue
		}
		rows = append(rows, MatrixRow[S, E]{
			Name:    fmt.Sprintf("formal transition %s/%s", transition.From, transition.Event),
			From:    from,
			Event:   event,
			To:      to,
			WantErr: transition.WantError,
		})
	}
	return rows, errs
}

func formalTraces[S comparable, E comparable](
	artifactTraces []FormalArtifactTrace,
	statuses []S,
	events []E,
) ([]Trace[S, E], []error) {
	statusByName := valuesByString(statuses)
	eventByName := valuesByString(events)
	traces := make([]Trace[S, E], 0, len(artifactTraces))
	var errs []error
	for traceIndex, formalTrace := range artifactTraces {
		initial, ok := statusByName[formalTrace.Initial]
		if !ok {
			errs = append(errs, fmt.Errorf("formal trace %s unknown initial state %s", formalTrace.Name, formalTrace.Initial))
			continue
		}
		steps := make([]TraceStep[S, E], 0, len(formalTrace.Steps))
		for stepIndex, step := range formalTrace.Steps {
			bad := false
			event, ok := eventByName[step.Event]
			if !ok {
				errs = append(errs, fmt.Errorf("formal trace %s step %d unknown event %s", formalTrace.Name, stepIndex, step.Event))
				bad = true
			}
			want, ok := statusByName[step.Want]
			if !ok {
				errs = append(errs, fmt.Errorf("formal trace %s step %d unknown want state %s", formalTrace.Name, stepIndex, step.Want))
				bad = true
			}
			if bad {
				continue
			}
			steps = append(steps, TraceStep[S, E]{
				Name:    fmt.Sprintf("step %d", stepIndex),
				Event:   event,
				Want:    want,
				WantErr: step.WantError,
			})
		}
		traces = append(traces, Trace[S, E]{
			Name:    traceName(formalTrace.Name, traceIndex),
			Initial: initial,
			Steps:   steps,
		})
	}
	return traces, errs
}

func valuesByString[T comparable](values []T) map[string]T {
	byName := make(map[string]T, len(values))
	for _, value := range values {
		byName[fmt.Sprint(value)] = value
	}
	return byName
}

func traceName(name string, index int) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("formal trace %d", index)
}
