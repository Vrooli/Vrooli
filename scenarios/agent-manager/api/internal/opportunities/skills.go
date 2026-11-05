package opportunities

import (
	"fmt"
	"sort"

	"agent-manager/internal/toolfriction"
)

// skillOwner owns every skill-size action: it is the scenario that serves and
// publishes skills.
const skillOwner = "prompt-manager"

// skillKey is one skill loaded in one session within one compaction segment.
type skillKey struct {
	run, skill string
	segment    int64
}

// skillReloads finds skills read again while the first read was still in
// context: a load after the first within the same compaction segment of the
// same run. A load in a later segment follows a compaction that dropped the
// text, so it is not extra. A skill is reported when extras occur in at least
// MinSessions runs. A repeat in a run whose segments are not known
// (Call.SegmentsKnown: every load sits in segment 0) says nothing of
// residency: it saves nothing and is counted apart, as residency unknown. A
// read whose output went through a filter (`skill read x | rg y`) never put
// the skill in context: it is neither a load nor a reload
// (toolfriction.Call.LoadsSkill). A reload's tokens are its own read's.
func skillReloads(calls []toolfriction.Call) []Opportunity {
	loads := map[skillKey]int{}
	type total struct {
		evidence               evidence
		extra, tokensUnknown   int
		tokens                 int64
		unknown                int
		knownRuns, unknownRuns map[string]bool
	}
	skills := map[string]*total{}
	for _, c := range calls {
		if !c.LoadsSkill() {
			continue
		}
		key := skillKey{c.RunID, c.SkillID, c.ResidencySegment}
		loads[key]++
		if loads[key] == 1 {
			continue
		}
		t := skills[c.SkillID]
		if t == nil {
			t = &total{knownRuns: map[string]bool{}, unknownRuns: map[string]bool{}}
			skills[c.SkillID] = t
		}
		t.evidence.run(c.RunID, 1)
		t.evidence.example(c.Command)
		if !c.SegmentsKnown() {
			t.unknown++
			t.unknownRuns[c.RunID] = true
			continue
		}
		t.knownRuns[c.RunID] = true
		t.evidence.save(c.RunID, 1)
		t.extra++
		t.tokens += c.ResultTokens
		if !c.TokensKnown() {
			t.tokensUnknown++ // its output sits on another segment of the command, or was never read
		}
	}
	names := make([]string, 0, len(skills))
	for name := range skills {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []Opportunity
	for _, name := range names {
		t := skills[name]
		if t.evidence.sessions() < MinSessions {
			continue
		}
		o := Opportunity{
			Kind: KindSkillReload, Owner: skillOwner, Subject: name, Occurrences: t.extra + t.unknown, TokensSaved: t.tokens, TokensUnknown: t.tokensUnknown,
			Verify: verifyAfter("skill `"+name+"` reloads", t.evidence.sessions()),
		}
		t.evidence.fill(&o)
		tokens := fmt.Sprintf("%d result tokens", t.tokens)
		if t.tokensUnknown > 0 {
			tokens += fmt.Sprintf("; tokens unknown for %d of them", t.tokensUnknown)
		}
		o.Action = fmt.Sprintf("skill `%s` was read again %d times while still in context in %d sessions (%s): split it, shrink it, or add a digest an agent can re-read cheaply", name, t.extra, len(t.knownRuns), tokens)
		unknown := fmt.Sprintf("%d repeats in %d sessions whose facts record no compactions (residency unknown: a repeat may follow one)", t.unknown, len(t.unknownRuns))
		switch {
		case t.extra == 0:
			o.Action = fmt.Sprintf("skill `%s` was read again: %s", name, unknown)
		case t.unknown > 0:
			o.Action += "; " + unknown
		}
		out = append(out, o)
	}
	return out
}
