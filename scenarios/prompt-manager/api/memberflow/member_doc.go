package memberflow

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Member document contract.
//
// HEARTBEAT.md and RESPONSIBILITIES.md are the only prose that reaches a
// running agent verbatim; every other prompt section is generated from
// team.json or topics.json and is already validated. That asymmetry is why
// these two files carry a checked section vocabulary rather than a
// convention: a heading is the member's own index into its own instructions,
// and two members naming one concept differently is drift that no generated
// section can correct.
//
// Canon: docs/agent-system/TEAM_MEMBER_ARCHITECTURE.md §"Member document
// contract". Rule catalog: docs/agent-system/TOPICS_SCHEMA.md §"Validation
// rules". The tables below are the machine copy of that canon; changing one
// without the other is the drift these rules exist to prevent.

type memberDocStatus int

const (
	// memberDocRequired sections are present on every member today; their
	// absence is an error.
	memberDocRequired memberDocStatus = iota
	// memberDocRecommended sections carry a layer the file is supposed to
	// hold but that some members do not yet state. Warning, because the
	// content is team judgment — the fix is authored by the owning team
	// through its own decision context, not inferred by a validator.
	memberDocRecommended
	// memberDocOptional sections are named only so that teams choosing to
	// write them agree on the word.
	memberDocOptional
)

// memberDocContract is the checked vocabulary for one member document.
type memberDocContract struct {
	// File is the basename under teams/<team>/members/<member>/.
	File string
	// Sections maps canonical level-two heading text to its status.
	Sections map[string]memberDocStatus
	// Aliases maps a retired heading to the canonical heading that
	// replaced it. Retired names are errors: the concept has a home, and a
	// second word for it is the drift the vocabulary exists to stop.
	Aliases map[string]string
	// AbsentSeverity is the severity of member_doc_file_missing. A member
	// without HEARTBEAT.md falls back to a generic "review your
	// responsibilities" task and is a real defect; a member without
	// RESPONSIBILITIES.md still receives its full generated contract, so
	// that absence is a warning.
	AbsentSeverity Severity
}

var memberDocContracts = []memberDocContract{
	{
		File: "HEARTBEAT.md",
		Sections: map[string]memberDocStatus{
			"Reasoning Framework": memberDocOptional,
			"Task Loop":           memberDocRequired,
			"Handoff Shape":       memberDocRequired,
			"Stop Conditions":     memberDocRecommended,
		},
		Aliases: map[string]string{
			"Required Loop":            "Task Loop",
			"Required Output Sections": "Handoff Shape",
		},
		AbsentSeverity: SeverityError,
	},
	{
		File: "RESPONSIBILITIES.md",
		Sections: map[string]memberDocStatus{
			"Primary Duties":   memberDocRecommended,
			"Judgment":         memberDocOptional,
			"Failure Modes":    memberDocOptional,
			"Boundaries":       memberDocOptional,
			"Cross-references": memberDocOptional,
			"Available Skills": memberDocOptional,
		},
		Aliases: map[string]string{
			"Judgment Notes":            "Judgment",
			"Failure-Mode Rubric":       "Failure Modes",
			"Failure-Mode Framework":    "Failure Modes",
			"Forbidden":                 "Boundaries",
			"What I do NOT do":          "Boundaries",
			"Authority Boundaries":      "Boundaries",
			"Plan-of-Record References": "Cross-references",
			"Useful Skills":             "Available Skills",
		},
		AbsentSeverity: SeverityWarning,
	},
}

// ruleMemberDocSections checks both member documents for every member against
// the canonical section vocabulary.
//
// Silent when StoreDir is unset: unit tests that pass synthetic members with
// no backing tree get no findings rather than spurious file-missing errors.
func ruleMemberDocSections(members []MemberTopics, opts ValidationOptions) []Finding {
	storeDir := strings.TrimSpace(opts.StoreDir)
	if storeDir == "" {
		return nil
	}

	var out []Finding
	for _, m := range members {
		for _, contract := range memberDocContracts {
			out = append(out, checkMemberDoc(storeDir, m.Ref, contract)...)
		}
	}
	return out
}

func checkMemberDoc(storeDir string, ref MemberRef, contract memberDocContract) []Finding {
	path := filepath.Join(storeDir, "teams", ref.Team, "members", ref.Member, contract.File)

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return []Finding{{
				Rule:     "member_doc_unreadable",
				Severity: SeverityError,
				Team:     ref.Team, Member: ref.Member,
				Detail: fmt.Sprintf("%s cannot be read: %v",
					contract.File, err),
			}}
		}
		return []Finding{{
			Rule:     "member_doc_file_missing",
			Severity: contract.AbsentSeverity,
			Team:     ref.Team, Member: ref.Member,
			Detail: fmt.Sprintf("member %q on team %q has no %s; see docs/agent-system/TEAM_MEMBER_ARCHITECTURE.md §\"Member document contract\" for the required sections",
				ref.Member, ref.Team, contract.File),
		}}
	}

	headings := markdownH2Headings(string(data))

	var out []Finding
	seen := map[string]int{}
	for _, h := range headings {
		if canonical, retired := contract.Aliases[h]; retired {
			out = append(out, Finding{
				Rule:     "member_doc_section_alias",
				Severity: SeverityError,
				Team:     ref.Team, Member: ref.Member,
				Detail: fmt.Sprintf("%s uses retired heading %q; rename it to %q (docs/agent-system/TEAM_MEMBER_ARCHITECTURE.md §\"Retired aliases\")",
					contract.File, "## "+h, "## "+canonical),
			})
			// Count the alias against its canonical slot so a file
			// carrying both the old and new name reports the rename
			// and the resulting duplicate, not just the rename.
			seen[canonical]++
			continue
		}
		if _, known := contract.Sections[h]; known {
			seen[h]++
		}
	}

	duplicates := make([]string, 0, len(seen))
	for h, n := range seen {
		if n > 1 {
			duplicates = append(duplicates, h)
		}
	}
	sort.Strings(duplicates)
	for _, h := range duplicates {
		out = append(out, Finding{
			Rule:     "member_doc_section_duplicate",
			Severity: SeverityError,
			Team:     ref.Team, Member: ref.Member,
			Detail: fmt.Sprintf("%s declares section %q %d times; merge them into one — a split section makes the member's own index ambiguous",
				contract.File, "## "+h, seen[h]),
		})
	}

	missing := make([]string, 0, len(contract.Sections))
	for h := range contract.Sections {
		if seen[h] == 0 {
			missing = append(missing, h)
		}
	}
	sort.Strings(missing)
	for _, h := range missing {
		switch contract.Sections[h] {
		case memberDocRequired:
			out = append(out, Finding{
				Rule:     "member_doc_section_missing",
				Severity: SeverityError,
				Team:     ref.Team, Member: ref.Member,
				Detail: fmt.Sprintf("%s is missing required section %q (docs/agent-system/TEAM_MEMBER_ARCHITECTURE.md §\"Canonical section vocabulary\")",
					contract.File, "## "+h),
			})
		case memberDocRecommended:
			out = append(out, Finding{
				Rule:     "member_doc_section_recommended",
				Severity: SeverityWarning,
				Team:     ref.Team, Member: ref.Member,
				Detail: fmt.Sprintf("%s has no %q section; the owning team should add one through its own decision context — the content is team judgment, not something a validator can infer",
					contract.File, "## "+h),
			})
		}
	}

	return out
}

// markdownH2Headings returns the trimmed text of every level-two heading in
// src, skipping fenced code blocks.
//
// The fence skip is load-bearing, not defensive: every HEARTBEAT.md embeds its
// handoff template as a fenced block whose first line is "## HANDOFF". A
// naive scan reads that as a section and reports it on 27 of 27 members.
func markdownH2Headings(src string) []string {
	var out []string
	inFence := false
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		// Level-two only: "## x" but not "### x".
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
		if heading != "" {
			out = append(out, heading)
		}
	}
	return out
}
