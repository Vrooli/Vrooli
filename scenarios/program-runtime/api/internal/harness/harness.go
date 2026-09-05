// Package harness owns the standing brief given to the authoring model, and
// the contract that keeps it aligned with the other two surfaces that teach
// this runtime.
//
// The brief used to be a Go string constant. That made it invisible to every
// process that shapes it: it was not versioned as an artifact, it was not
// checked against `docs/guides/program-construction.md` or the prompt-manager
// skill, and nothing could tell you that a rule had been documented in one
// place and omitted from the others. The consequence was measurable — the
// constant never mentioned that `gather` takes zero-argument callables, and two
// of twelve authoring-eval cases failed on exactly that, so the eval was partly
// measuring prompt omission rather than surface quality.
//
// The contract inverts that. Each rule is declared once, carries the text the
// model sees, and names a probe that must appear in the construction guide and
// in the skill. The brief is generated from the rules, and TestNoRuleIsMissing
// fails when any surface falls behind. Changing what an agent is taught is now
// one edit with a mechanical consequence, which is the self-modifiable harness
// state this runtime was built to have.
package harness

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

//go:embed contract.json
var contractJSON []byte

// Rule is one thing an agent must know to write a correct program.
type Rule struct {
	ID    string `json:"id"`
	Brief string `json:"brief"`
	// DocProbe and SkillProbe are substrings that must appear in the
	// construction guide and the skill respectively. They are deliberately
	// substrings rather than exact copies of Brief: the three surfaces have
	// different audiences and lengths, and forcing identical prose would make
	// the check either trivially satisfied or permanently red.
	DocProbe   string             `json:"doc_probe"`
	SkillProbe string             `json:"skill_probe"`
	Signatures []FailureSignature `json:"failure_signatures"`
}

// FailureSignature is a deterministic fingerprint for a failure caused by a
// missing or violated harness rule. Cause is matched exactly after
// normalization; DetailPattern is a case-insensitive Go regular expression.
// Example is executable contract evidence used to prove every declared
// signature still matches the detail shape it documents.
type FailureSignature struct {
	Cause         string `json:"cause"`
	DetailPattern string `json:"detail_pattern"`
	Example       string `json:"example"`
}

// UnattributedRuleID is deliberately a first-class result. Guessing when no
// signature, or more than one signature, matches would corrupt the feedback
// loop that uses miss counts to judge brief changes.
const UnattributedRuleID = "unattributed"

// Contract is the versioned harness state.
type Contract struct {
	Version  int    `json:"version"`
	Preamble string `json:"preamble"`
	Closing  string `json:"closing"`
	Rules    []Rule `json:"rules"`
}

// Load returns the embedded contract. It panics on a malformed contract
// because a runtime that cannot state its own rules must not author programs
// against an empty brief and report the resulting score as a measurement.
func Load() Contract {
	var contract Contract
	if err := json.Unmarshal(contractJSON, &contract); err != nil {
		panic(fmt.Sprintf("harness contract is not valid JSON: %v", err))
	}
	if len(contract.Rules) == 0 {
		panic("harness contract declares no rules")
	}
	for _, rule := range contract.Rules {
		if len(rule.Signatures) == 0 {
			panic(fmt.Sprintf("harness rule %q declares no failure signatures", rule.ID))
		}
		for _, signature := range rule.Signatures {
			matcher, err := regexp.Compile("(?i)" + signature.DetailPattern)
			if err != nil {
				panic(fmt.Sprintf("harness rule %q has invalid failure signature: %v", rule.ID, err))
			}
			if strings.TrimSpace(signature.Cause) == "" || !matcher.MatchString(signature.Example) {
				panic(fmt.Sprintf("harness rule %q has an unproven failure signature", rule.ID))
			}
		}
	}
	return contract
}

// ResolveFailure returns the single rule whose signature matches cause and
// detail. An absent or ambiguous match is explicitly unattributed; contract
// order is never used as hidden precedence.
func (c Contract) ResolveFailure(cause, detail string) string {
	cause = strings.ToLower(strings.TrimSpace(cause))
	matches := map[string]struct{}{}
	for _, rule := range c.Rules {
		for _, signature := range rule.Signatures {
			if strings.ToLower(strings.TrimSpace(signature.Cause)) != cause {
				continue
			}
			matcher, err := regexp.Compile("(?i)" + signature.DetailPattern)
			if err == nil && matcher.MatchString(detail) {
				matches[rule.ID] = struct{}{}
			}
		}
	}
	if len(matches) != 1 {
		return UnattributedRuleID
	}
	for ruleID := range matches {
		return ruleID
	}
	return UnattributedRuleID
}

// Instruction renders the brief handed to the authoring model.
func (c Contract) Instruction() string {
	var builder strings.Builder
	builder.WriteString(c.Preamble)
	builder.WriteString("\n\nRules:\n")
	for _, rule := range c.Rules {
		builder.WriteString("- ")
		builder.WriteString(rule.Brief)
		builder.WriteString("\n")
	}
	builder.WriteString("\n")
	builder.WriteString(c.Closing)
	return builder.String()
}

// Stamp identifies the brief a measurement was taken against, so an authoring
// score is attributable to a specific harness version rather than to "the
// prompt at the time".
func (c Contract) Stamp() string {
	return fmt.Sprintf("authoring-brief@%d(%d rules)", c.Version, len(c.Rules))
}
