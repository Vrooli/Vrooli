package memberflow

import "testing"

// hasRule reports whether the validation result contains a finding for the
// named rule, and returns it for severity assertions.
func hasRule(r ValidationResult, rule string) (Finding, bool) {
	for _, f := range r.Findings {
		if f.Rule == rule {
			return f, true
		}
	}
	return Finding{}, false
}

func assertRule(t *testing.T, r ValidationResult, rule string, want Severity) {
	t.Helper()
	f, ok := hasRule(r, rule)
	if !ok {
		t.Fatalf("expected %s finding; findings=%v", rule, r.Findings)
	}
	if f.Severity != want {
		t.Errorf("%s severity = %q, want %q", rule, f.Severity, want)
	}
}

func assertNoRule(t *testing.T, r ValidationResult, rule string) {
	t.Helper()
	if f, ok := hasRule(r, rule); ok {
		t.Errorf("unexpected %s finding: %+v", rule, f)
	}
}

// sweepWithLedger is the canonical well-formed sweep: it writes a ledger
// prefix and reads the same prefix back, and names what it iterates.
func sweepWithLedger() Topics {
	return Topics{
		LoopKind:     LoopSweep,
		Population:   []string{"skill"},
		RequiredRead: []RequiredReadEntry{{Prefix: "skill-visited/<skill-id>"}},
		Output: []OutputEntry{
			{Prefix: "skill-visited/*", DestinationKind: DestinationKnowledge},
		},
	}
}

func TestLoopKind_MissingIsWarningNotError(t *testing.T) {
	// Adoption must not fail a team for a value it has not yet declared.
	members := []MemberTopics{
		mkMember("team-a", "alice", Topics{
			Output: []OutputEntry{{Prefix: "skill-audit/*", DestinationKind: DestinationKnowledge}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	assertRule(t, r, "loop_kind_missing", SeverityWarning)
}

func TestLoopKind_DeclaredKindSuppressesMissing(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "alice", Topics{
			LoopKind: LoopGenerative,
			Output:   []OutputEntry{{Prefix: "campaign-draft/*", DestinationKind: DestinationKnowledge}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	assertNoRule(t, r, "loop_kind_missing")
}

func TestLoopKind_UnknownValueIsError(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "alice", Topics{LoopKind: LoopKind("batch")}),
	}
	r := Validate(members, ValidationOptions{})
	assertRule(t, r, "loop_kind_invalid", SeverityError)
}

// The self-checking property: a declaration that contradicts the flow it
// describes must fail, so loop_kind cannot silently drift from topics.json.
func TestLoopKind_IntakeForcesQueue(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "alice", Topics{
			LoopKind: LoopSweep,
			Intake:   []IntakeEntry{{Prefix: "bug-inbox/*", Taxonomy: "tx"}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	assertRule(t, r, "loop_kind_intake_mismatch", SeverityError)
}

func TestLoopKind_IntakeWithQueueIsClean(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "alice", Topics{
			LoopKind: LoopQueue,
			Intake:   []IntakeEntry{{Prefix: "bug-inbox/*", Taxonomy: "tx"}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	assertNoRule(t, r, "loop_kind_intake_mismatch")
}

func TestSweep_WithoutLedgerIsError(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "alice", Topics{
			LoopKind:   LoopSweep,
			Population: []string{"skill"},
			Output: []OutputEntry{
				{Prefix: "skill-audit/*", DestinationKind: DestinationKnowledge},
			},
		}),
	}
	r := Validate(members, ValidationOptions{})
	assertRule(t, r, "sweep_without_ledger", SeverityError)
}

func TestSweep_WithLedgerIsClean(t *testing.T) {
	members := []MemberTopics{mkMember("team-a", "alice", sweepWithLedger())}
	r := Validate(members, ValidationOptions{})
	assertNoRule(t, r, "sweep_without_ledger")
	assertNoRule(t, r, "ledger_shape_invalid")
	assertNoRule(t, r, "sweep_population_missing")
}

// The enforce rules must stay silent on undeclared members: a team cannot
// fix a gap the model does not yet let it express.
func TestSweep_EnforceRulesDoNotFireWithoutDeclaration(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "alice", Topics{
			Output: []OutputEntry{
				{Prefix: "skill-audit/*", DestinationKind: DestinationKnowledge},
			},
		}),
	}
	r := Validate(members, ValidationOptions{})
	assertNoRule(t, r, "sweep_without_ledger")
	assertNoRule(t, r, "sweep_population_missing")
}

func TestLedger_WriteOnlyIsError(t *testing.T) {
	// A visited topic nobody reads back looks like coverage memory and
	// functions as none. Caught regardless of loop_kind.
	members := []MemberTopics{
		mkMember("team-a", "alice", Topics{
			Output: []OutputEntry{
				{Prefix: "skill-visited/*", DestinationKind: DestinationKnowledge},
			},
		}),
	}
	r := Validate(members, ValidationOptions{})
	assertRule(t, r, "ledger_shape_invalid", SeverityError)
}

func TestSweep_MissingPopulationIsWarning(t *testing.T) {
	tp := sweepWithLedger()
	tp.Population = nil
	members := []MemberTopics{mkMember("team-a", "alice", tp)}
	r := Validate(members, ValidationOptions{})
	assertRule(t, r, "sweep_population_missing", SeverityWarning)
	assertNoRule(t, r, "sweep_without_ledger")
}

// The retracted rule. `self_reference_not_ledger` was drafted to flag outputs
// that escape orphan_output only via their own writer's read. Against the real
// store it fired 18 times on legitimate continuity records, so it was dropped
// rather than shipped. These tests pin the behavior that replaced it: a
// member's own durable record is valid, and only a ledger-shaped output that
// nobody reads back is a defect.
func TestSelfRead_DurableRecordIsNotADefect(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "alice", Topics{
			LoopKind:     LoopGenerative,
			RequiredRead: []RequiredReadEntry{{Prefix: "outcome-target-record/*"}},
			Output: []OutputEntry{
				{Prefix: "outcome-target-record/*", DestinationKind: DestinationKnowledge},
			},
		}),
	}
	r := Validate(members, ValidationOptions{})
	assertNoRule(t, r, "self_reference_not_ledger")
	assertNoRule(t, r, "ledger_shape_invalid")
	if r.Errors != 0 {
		t.Errorf("continuity record must not error; findings=%v", r.Findings)
	}
}
