package vroolierr

import (
	"errors"
	"testing"
)

type exitCodeFixture struct{ code int }

func (f exitCodeFixture) Error() string { return "fixture" }
func (f exitCodeFixture) ExitCode() int { return f.code }

func TestErrorPrefersMessageFormatting(t *testing.T) {
	err := (&Error{Message: "start failed", Err: errors.New("boom")}).Error()
	if err != "start failed: boom" {
		t.Fatalf("Error() = %q", err)
	}
}

func TestErrorFallsBackToOperationResourceFormatting(t *testing.T) {
	err := (&Error{Operation: "start", Resource: "redis", Err: errors.New("boom")}).Error()
	if err != "start redis: boom" {
		t.Fatalf("Error() = %q", err)
	}
}

func TestExitCodePrefersExplicitThenWrapped(t *testing.T) {
	if got := (&Error{Exit: 17}).ExitCode(); got != 17 {
		t.Fatalf("ExitCode() = %d, want 17", got)
	}
	if got := (&Error{Err: exitCodeFixture{code: 23}}).ExitCode(); got != 23 {
		t.Fatalf("wrapped ExitCode() = %d, want 23", got)
	}
}

func TestHelpersReadWrappedMetadata(t *testing.T) {
	err := &Error{
		Code:        "scenario_not_found",
		Category:    "Usage",
		Hint:        "Use vrooli scenario list",
		Suggestions: []string{"scenario list"},
		HTTPStatus:  404,
	}
	if got := Code(err, "fallback"); got != "scenario_not_found" {
		t.Fatalf("Code() = %q", got)
	}
	if got := Category(err); got != "Usage" {
		t.Fatalf("Category() = %q", got)
	}
	if got := Hint(err); got != "Use vrooli scenario list" {
		t.Fatalf("Hint() = %q", got)
	}
	if got := HTTPStatus(err, 500); got != 404 {
		t.Fatalf("HTTPStatus() = %d", got)
	}
	if suggestions := Suggestions(err); len(suggestions) != 1 || suggestions[0] != "scenario list" {
		t.Fatalf("Suggestions() = %#v", suggestions)
	}
}
