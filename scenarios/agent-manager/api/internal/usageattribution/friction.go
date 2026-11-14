package usageattribution

import (
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/toolfriction"
)

// Finding statuses. A finding the operator has not acted on is open.
const (
	StatusOpen      = "open"
	StatusReported  = "reported"
	StatusDismissed = "dismissed"
)

// ToolFinding is an owner-level tool problem from the tool friction profiles
// with what the operator did with it.
type ToolFinding struct {
	toolfriction.Finding
	Status      string
	BugID       string
	StatusSetAt time.Time
}

// ProblemStatus is what the operator did with a problem.
type ProblemStatus struct {
	Fingerprint string
	Status      string
	BugID       string
	At          time.Time
}

// FrictionBug is what a scenario-qa report says about a friction problem.
type FrictionBug struct {
	Fingerprint, Title, Owner, OwnerKind string
	Actual, Description                  string
	Repro                                []string
	Context                              map[string]string
}

// bug is the scenario-qa report of a tool problem, filed against its owner.
func (f ToolFinding) bug() FrictionBug {
	context := map[string]string{"fingerprint": f.Fingerprint, "kind": f.Kind, "owner": f.Owner, "owner_kind": f.OwnerKind, "scenario": f.Owner}
	if f.CommandPath != "" {
		context["command"] = f.CommandPath
	}
	examples := "none recorded"
	if len(f.Examples) > 0 {
		examples = "`" + strings.Join(f.Examples, "`, `") + "`"
	}
	return FrictionBug{
		Fingerprint: f.Fingerprint, Title: fmt.Sprintf("tool friction (%s): %s", f.Kind, f.Owner), Owner: f.Owner, OwnerKind: f.OwnerKind,
		Actual:      f.Description + ".",
		Description: fmt.Sprintf("Agent Manager's tool friction profile found this %s problem owned by %s (%d calls in %d runs; cost %s). Example commands: %s. Evidence runs: %s.", f.Kind, f.Owner, f.Calls, f.Runs, shareSentences(f.Shares), examples, strings.Join(f.RunIDs, ", ")),
		Repro:       []string{"Run `agent-manager usage friction`", fmt.Sprintf("Find the tool finding with fingerprint %s", f.Fingerprint), "Run one of its example commands", "Open the evidence runs with `agent-manager run report <run-id> --facts`"},
		Context:     context,
	}
}

// shareSentences states each share of a finding with its base.
func shareSentences(shares []toolfriction.Share) string {
	parts := make([]string, len(shares))
	for i, s := range shares {
		parts[i] = s.Sentence()
	}
	return strings.Join(parts, "; ")
}
