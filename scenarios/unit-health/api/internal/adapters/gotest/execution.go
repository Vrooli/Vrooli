package gotest

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"unit-health/internal/adapters"
	"unit-health/internal/testquality"
)

func (Analyzer) PrepareExecutionEvidence(executable string, args []string) ([]string, bool) {
	if executable != "go" || len(args) == 0 || args[0] != "test" {
		return nil, false
	}
	for _, arg := range args[1:] {
		if arg == "-args" || arg == "--" || arg == "-count" || (strings.HasPrefix(arg, "-count=") && arg != "-count=1") || (strings.HasPrefix(arg, "-json=") && arg != "-json=true") {
			return nil, false
		}
	}
	out := []string{"test", "-json", "-count=1"}
	return append(out, args[1:]...), true
}

func (Analyzer) ObserveExecutionEvidence(input adapters.ExecutionEvidenceInput) ([]testquality.TestLinks, testquality.Reason) {
	// Source inventory is module-root-relative. Do not guess a package prefix
	// from ancestors or run go list (which could download missing dependencies).
	file, err := os.Open(filepath.Join(input.Root, "go.mod"))
	if err != nil {
		return nil, testquality.BuildContextUnavailable
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return nil, testquality.BuildContextUnavailable
	}
	module := ""
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			module = fields[1]
			if strings.HasPrefix(module, `"`) {
				module, err = strconv.Unquote(module)
				if err != nil {
					return nil, testquality.ParseFailure
				}
			}
			break
		}
	}
	return ObserveExecution(module, input.RunID, input.Stdout, input.Complete, input.Declarations)
}

// ObserveExecution consumes current, complete go test -json output captured by
// the executor, never a display tail or command exit code. Package-qualified
// native identities prevent same-named tests in sibling packages colliding.
func ObserveExecution(module, runID string, data []byte, complete bool, declarations []testquality.TestLinks) ([]testquality.TestLinks, testquality.Reason) {
	if !complete {
		return nil, testquality.ResolutionLimit
	}
	if module == "" || runID == "" || len(data) == 0 {
		return nil, testquality.MissingInput
	}
	if len(data) > 8<<20 {
		return nil, testquality.ResolutionLimit
	}
	type event struct{ Action, Package, Test, Time string }
	type state struct {
		started  bool
		terminal testquality.ExecutionState
	}
	states := map[string]state{}
	packages := map[string]bool{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	for count := 0; ; count++ {
		if count > 100000 {
			return nil, testquality.ResolutionLimit
		}
		var e event
		if err := decoder.Decode(&e); err == io.EOF {
			break
		} else if err != nil {
			return nil, testquality.ParseFailure
		}
		if e.Package == "" {
			return nil, testquality.ParseFailure
		}
		if e.Action == "start" && e.Test == "" {
			packages[e.Package] = true
			continue
		}
		if !packages[e.Package] {
			return nil, testquality.ParseFailure
		}
		if e.Test == "" {
			continue
		}
		key := e.Package + ":" + e.Test
		s := states[key]
		switch e.Action {
		case "run":
			if s.started || s.terminal != "" {
				return nil, testquality.ParseFailure
			}
			// Cached output omits event timestamps. It cannot prove fresh execution.
			s.started = e.Time != ""
		case "pass", "fail", "skip":
			if s.terminal != "" {
				return nil, testquality.ParseFailure
			}
			if s.started && e.Time != "" {
				switch e.Action {
				case "pass":
					s.terminal = testquality.ExecutionPassed
				case "fail":
					s.terminal = testquality.ExecutionFailed
				case "skip":
					s.terminal = testquality.ExecutionSkipped
				}
			}
		case "output", "pause", "cont":
		default:
			return nil, testquality.UnsupportedVersion
		}
		states[key] = s
	}
	if len(packages) == 0 {
		return nil, testquality.MissingInput
	}
	out := make([]testquality.TestLinks, 0, len(declarations))
	keyFor := func(declaration testquality.TestLinks) string {
		pkg := module
		if dir := path.Dir(declaration.Target.File); dir != "." {
			pkg += "/" + dir
		}
		name := strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return '_'
			}
			return r
		}, declaration.Target.TestID)
		return pkg + ":" + name
	}
	identities := map[string]int{}
	for _, declaration := range declarations {
		identities[keyFor(declaration)]++
	}
	for _, declaration := range declarations {
		observed := declaration
		observed.IDs = append([]string(nil), declaration.IDs...)
		observed.EvidenceKind = testquality.Runtime
		observed.RunID, observed.ExpectedRunID = runID, runID
		observed.Execution = testquality.ExecutionUnknown
		key := keyFor(declaration)
		// testing adds disambiguation suffixes for colliding rewritten names.
		// Without an exact native/source identity map, neither may borrow a pass.
		if s := states[key]; s.terminal != "" && identities[key] == 1 {
			observed.Execution = s.terminal
		}
		out = append(out, observed)
	}
	return out, testquality.ReasonNone
}
