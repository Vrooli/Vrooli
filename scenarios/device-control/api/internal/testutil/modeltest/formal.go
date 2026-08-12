package modeltest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

const formalArtifactSchemaVersion = 6

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
	data, err := os.ReadFile(path)
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
