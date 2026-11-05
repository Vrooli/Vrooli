package opportunities

import (
	"slices"
	"strings"
)

// Program is one library program as far as sequence matching needs it.
type Program struct {
	Name string
	// Scenario owns the program and Purpose says what it does.
	Scenario, Purpose string
	// Bindings are the governed callables it calls, as "scenario/group/command".
	Bindings []string
}

// Library is the program library read from program-runtime. Unavailable says
// why it could not be read; sequences are then unmatched, never "candidates".
type Library struct {
	Programs    []Program
	Unavailable string
}

// Match bases: how a program is linked to a sequence.
const (
	BasisBinding  = "binding"  // the program calls the commands of at least half the steps
	BasisAffinity = "affinity" // it belongs to a step's owner and names a rare word of the sequence
)

// ProgramMatch is the library program closest to a sequence.
type ProgramMatch struct {
	Name, Scenario string
	// Basis is BasisBinding or BasisAffinity, the weaker match: contracts do
	// not declare the CLI commands they replace.
	Basis string
	// Covered counts the sequence's steps whose commands the program calls,
	// of Steps; adopting it needs a majority.
	Covered, Steps int
	// Runs are calls that named the program in the window; the sequence's
	// manual occurrences are on the opportunity.
	Runs int
}

// bindingPath is the command path a governed binding is reached by:
// "test-genie/runs/wait" is `test-genie runs wait`.
func bindingPath(binding string) string { return strings.ReplaceAll(binding, "/", " ") }

// calls reports whether the program calls the binding behind a command path.
func (p Program) calls(path string) bool {
	for _, b := range p.Bindings {
		if bp := bindingPath(b); path == bp || strings.HasPrefix(path, bp+" ") {
			return true
		}
	}
	return false
}

// mentions reports whether the program's own name (without its scenario
// prefix) or purpose uses the word: the
// only link between a program and a CLI command that the contract carries
// when it does not call the command's binding.
func (p Program) mentions(word string) bool {
	_, local, _ := strings.Cut(p.Name, ".")
	text := strings.ToLower(local + " " + p.Purpose)
	for from := 0; ; {
		i := strings.Index(text[from:], strings.ToLower(word))
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(word)
		if (start == 0 || !isLetter(text[start-1])) && (end == len(text) || !isLetter(text[end])) {
			return true
		}
		from = start + 1
	}
}

func isLetter(b byte) bool { return b >= 'a' && b <= 'z' || b >= '0' && b <= '9' }

// rareShare bounds how common a word may be in the library's program texts
// and still tell programs apart: in a library of 108 contracts "test" is named
// by five and "skill" by seven.
const rareShare = 0.05

// rare reports whether few programs mention the word (at most rareShare of
// the library, and never fewer than two).
func (l Library) rare(word string) bool {
	n := 0
	for _, p := range l.Programs {
		if p.mentions(word) {
			n++
		}
	}
	return n <= max(2, int(rareShare*float64(len(l.Programs))))
}

// minAffinitySteps is how many of a sequence's steps a program must name to
// have affinity with it: one shared word ("list" in "a request list") links a
// program to any listing step, two name the work (test-genie.iterate's "test"
// and "wait" for a test loop).
const minAffinitySteps = 2

// affinity reports whether the program belongs to a scenario that owns a step
// and names, in its name or purpose, a rare word of the commands of at least
// minAffinitySteps steps (owner names excluded): the only link a contract
// carries to the commands it replaces when it does not call their bindings.
func (l Library) affinity(p Program, steps []step) bool {
	owns := false
	for _, s := range steps {
		owns = owns || s.owner == p.Scenario
	}
	if !owns {
		return false
	}
	named := 0
	for _, s := range steps {
		for _, word := range strings.Fields(s.path) {
			if word != s.owner && len(word) > 2 && p.mentions(word) && l.rare(word) {
				named++
				break
			}
		}
	}
	return named >= minAffinitySteps
}

// best is the program closest to the sequence: one calling the commands of
// at least half of its steps (basis binding), else one with affinity to it
// (basis affinity). Among them the most steps called, then the most owned,
// then name. Nil when none: a program that shares one step is related, not a
// replacement.
func (l Library) best(steps []step) *ProgramMatch {
	var best *ProgramMatch
	var bestScore [2]int
	for _, p := range l.Programs {
		called, owned := 0, 0
		for _, s := range steps {
			if p.calls(s.path) {
				called++
			}
			if s.owner == p.Scenario {
				owned++
			}
		}
		bound := called*2 >= len(steps)
		if !bound && !l.affinity(p, steps) {
			continue
		}
		score := [2]int{called, owned}
		if best == nil || slices.Compare(score[:], bestScore[:]) > 0 || slices.Compare(score[:], bestScore[:]) == 0 && p.Name < best.Name {
			basis := BasisAffinity
			if bound {
				basis = BasisBinding
			}
			best, bestScore = &ProgramMatch{Name: p.Name, Scenario: p.Scenario, Basis: basis, Covered: called, Steps: len(steps)}, score
		}
	}
	return best
}
