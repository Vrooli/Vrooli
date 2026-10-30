package opportunities

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"agent-manager/internal/toolfriction"
)

var (
	sqlTable  = regexp.MustCompile(`(?i)\b(?:from|join|into|update|table|schema)\s+"?([a-z_][a-z0-9_]*)`)
	httpPath  = regexp.MustCompile(`(?:localhost|127\.0\.0\.1|0\.0\.0\.0|\[::1\]):\d+(/[^\s'"?#]*)`)
	statePath = regexp.MustCompile(`(?:scenarios/[^/\s'"]+/data|\.vrooli/(?:state|data)/vrooli/[^/\s'"]+)(/[^\s'"]*)?`)
	sqlWords  = map[string]bool{"select": true, "where": true, "values": true, "set": true, "the": true}
	idSegment = regexp.MustCompile(`^(?:\d+|[0-9a-f]{8,}(?:-[0-9a-f]+)*|[A-Za-z]+_[0-9a-f-]{8,})$`)
)

// bypassTargets are what one bypassing command reached, from its stored
// command line: tables for sqlite3, URL paths for an HTTP call, directory
// paths for a state read.
func bypassTargets(c toolfriction.Call) []string {
	switch c.BypassMethod {
	case toolfriction.BypassDatabase:
		var tables []string
		for _, m := range sqlTable.FindAllStringSubmatch(c.Command, -1) {
			if name := strings.ToLower(m[1]); len(name) > 2 && !sqlWords[name] {
				tables = append(tables, "table "+name)
			}
		}
		return tables
	case toolfriction.BypassHTTP:
		if m := httpPath.FindStringSubmatch(c.Command); m != nil {
			return []string{shortPath(m[1])}
		}
	case toolfriction.BypassState:
		if m := statePath.FindStringSubmatch(c.Command); m != nil {
			return []string{"state" + shortPath(m[1])}
		}
	}
	return nil
}

// shortPath keeps a path's first three segments with identifiers replaced, so
// requests that differ only by id are one target.
func shortPath(path string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i, s := range segments {
		if idSegment.MatchString(s) {
			segments[i] = "{id}"
		}
	}
	segments = segments[:min(len(segments), 3)]
	if len(segments) == 1 && segments[0] == "" {
		return ""
	}
	return "/" + strings.Join(segments, "/")
}

// bypasses clusters the calls that reached a scenario's internals around its
// CLI by owner and method and names what they touched. A cluster seen in at
// least MinSessions runs is a capability the owner should serve. Reads by runs
// working on that scenario are not bypasses (toolfriction.Call.OwnWork); the
// action says how many it set aside.
func bypasses(calls []toolfriction.Call) []Opportunity {
	type cluster struct {
		evidence evidence
		targets  map[string]int
		calls    int
	}
	clusters := map[[2]string]*cluster{}
	own := toolfriction.OwnWorkByTarget(calls)
	for _, c := range calls {
		if !c.Bypasses() {
			continue
		}
		key := [2]string{c.BypassOwner, c.BypassMethod}
		cl := clusters[key]
		if cl == nil {
			cl = &cluster{targets: map[string]int{}}
			clusters[key] = cl
		}
		cl.calls++
		cl.evidence.run(c.RunID, 1)
		cl.evidence.save(c.RunID, 1)
		cl.evidence.example(c.Command)
		for _, t := range bypassTargets(c) {
			cl.targets[t]++
		}
	}
	keys := make([][2]string, 0, len(clusters))
	for k := range clusters {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i][0]+keys[i][1] < keys[j][0]+keys[j][1] })
	var out []Opportunity
	for _, key := range keys {
		cl := clusters[key]
		if cl.evidence.sessions() < MinSessions {
			continue
		}
		owner, method := key[0], key[1]
		subject := owner + " via " + method
		o := Opportunity{
			Kind: KindBypass, Owner: owner, Subject: subject, Occurrences: cl.calls, Details: topCounts(cl.targets, maxDetails),
			Verify: verifyAfter("`"+subject+"`", cl.evidence.sessions()),
		}
		reached := "what they read (not recorded for these calls)"
		if len(o.Details) > 0 {
			reached = "the targets listed"
		}
		o.Action = fmt.Sprintf("agents reached %s's internals by %s %d times in %d sessions%s: add a CLI command or RPC for %s", owner, method, cl.calls, cl.evidence.sessions(), own[key].Note(owner), reached)
		cl.evidence.fill(&o)
		out = append(out, o)
	}
	return out
}
