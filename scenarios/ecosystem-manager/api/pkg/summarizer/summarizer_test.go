package summarizer

import "testing"

func TestParseResultFullComplete(t *testing.T) {
	body := "**What was accomplished:** Finalized the release.\n**Current status:** Stable.\n**Remaining issues or limitations:** None reported.\n**Suggested next actions:** None\n**Validation evidence:** - make test"
	raw := "classification: full_complete\nnote:\n" + body

	result, err := parseResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Classification != classificationFull {
		t.Fatalf("expected classification %q, got %q", classificationFull, result.Classification)
	}

	expectedLead := "Likely complete, but may benefit from additional validation/tidying. Notes from last time:"
	expectedNote := expectedLead + "\n\n" + body
	if result.Note != expectedNote {
		t.Fatalf("unexpected note.\nexpected: %q\nactual:   %q", expectedNote, result.Note)
	}
}

func TestParseResultSignificantProgress(t *testing.T) {
	body := "**What was accomplished:** Core features deployed.\n**Current status:** Needs polish.\n**Remaining issues or limitations:** None reported.\n**Suggested next actions:** None reported.\n**Validation evidence:** - Tests pending"
	raw := "classification: significant_progress\nnote:\n" + body

	result, err := parseResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Classification != classificationSignificant {
		t.Fatalf("expected classification %q, got %q", classificationSignificant, result.Classification)
	}

	expectedLead := "Already pretty good, but could use some additional validation/tidying. Notes from last time:"
	expectedNote := expectedLead + "\n\n" + body
	if result.Note != expectedNote {
		t.Fatalf("unexpected note.\nexpected: %q\nactual:   %q", expectedNote, result.Note)
	}
}

func TestParseResultLegacyPartialMapsToSomeProgress(t *testing.T) {
	body := "**What was accomplished:** Drafted endpoints.\n**Current status:** Blocked on auth.\n**Remaining issues or limitations:** TODO list remains.\n**Suggested next actions:** None reported.\n**Validation evidence:** - Manual checks"
	raw := "classification: partial_progress\nnote:\n" + body

	result, err := parseResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Classification != classificationSome {
		t.Fatalf("expected classification %q, got %q", classificationSome, result.Classification)
	}

	expectedLead := "Notes from last time:"
	expectedNote := expectedLead + "\n\n" + body
	if result.Note != expectedNote {
		t.Fatalf("unexpected note decoration.\nexpected: %q\nactual:   %q", expectedNote, result.Note)
	}
}

func TestParseResultFallsBackOnEmptyNote(t *testing.T) {
	raw := "classification: some_progress\nnote:\n"

	result, err := parseResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Classification != classificationUncertain {
		t.Fatalf("expected fallback classification %q, got %q", classificationUncertain, result.Classification)
	}

	if result.Note != "Not sure current status" {
		t.Fatalf("expected fallback note, got %q", result.Note)
	}
}

func TestParseResultRequiresClassificationPrefix(t *testing.T) {
	if _, err := parseResult("invalid-format"); err == nil {
		t.Fatalf("expected error for missing classification prefix")
	}
}

func TestDefaultResult(t *testing.T) {
	def := DefaultResult()
	if def.Note == "" || def.Classification != classificationUncertain {
		t.Fatalf("default result should return uncertain classification")
	}
}
