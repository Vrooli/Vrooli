package opportunities

import (
	"fmt"
	"sort"

	"agent-manager/internal/toolfriction"
)

// outputShapes finds Vrooli commands whose JSON agents routinely cut fields
// out of with jq: per command path, the paths extracted. A command filtered
// in at least MinSessions runs wants a focused flag returning those fields.
func outputShapes(calls []toolfriction.Call) []Opportunity {
	type shape struct {
		evidence evidence
		paths    map[string]int
		owner    string
		calls    int
	}
	shapes := map[string]*shape{}
	for _, c := range calls {
		if c.Owner == "" || len(c.JSONPaths) == 0 {
			continue
		}
		s := shapes[c.CommandPath]
		if s == nil {
			s = &shape{paths: map[string]int{}, owner: c.Owner}
			shapes[c.CommandPath] = s
		}
		s.calls++
		s.evidence.run(c.RunID, 1)
		s.evidence.save(c.RunID, 1)
		s.evidence.example(c.Command)
		for _, p := range c.JSONPaths {
			s.paths[p]++
		}
	}
	commands := make([]string, 0, len(shapes))
	for command := range shapes {
		commands = append(commands, command)
	}
	sort.Strings(commands)
	var out []Opportunity
	for _, command := range commands {
		s := shapes[command]
		if s.evidence.sessions() < MinSessions {
			continue
		}
		o := Opportunity{
			Kind: KindOutputShape, Owner: s.owner, Subject: command, Occurrences: s.calls, Details: topCounts(s.paths, maxDetails),
			Action: fmt.Sprintf("agents piped `%s --json` through jq %d times in %d sessions to extract the same fields: add a `--fields` option or a focused flag returning them", command, s.calls, s.evidence.sessions()),
			Verify: verifyAfter("`"+command+"` jq extractions", s.evidence.sessions()),
		}
		s.evidence.fill(&o)
		out = append(out, o)
	}
	return out
}
