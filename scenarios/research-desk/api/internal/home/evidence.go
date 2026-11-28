// Package home is the one reader of the research home (~/.vrooli/research)
// and of the Codex session records Web Console keeps. Every derived value it
// returns names its source; a value that cannot be derived is unknown, never
// guessed. Nothing outside this package parses research files.
//
// Design: scenarios/research-desk/docs/concepts/SYSTEM.md §1, §3, §4, §9.
// File shapes: scenarios/research-desk/docs/reference/FILE-CONTRACT.md.
package home

import "time"

// Evidence names where one derived value came from.
type Evidence struct {
	Known  bool
	Source string
	AsOf   time.Time
	Reason string
}

// From is evidence read from a file with the given modification time.
func From(source string, asOf time.Time) Evidence {
	return Evidence{Known: true, Source: source, AsOf: asOf}
}

// Unknown is evidence for a value that could not be derived.
func Unknown(reason string) Evidence {
	return Evidence{Reason: reason}
}

// Text is a derived text value.
type Text struct {
	Value string
	Evidence
}

// Time is a derived time value.
type Time struct {
	Value time.Time
	// AtLeast marks a lower bound: the walk that found Value stopped early,
	// so a newer file may exist.
	AtLeast bool
	Evidence
}

// Flag is a derived yes/no value.
type Flag struct {
	Value bool
	Evidence
}

// later returns whichever known time is later; an unknown side loses.
func later(a, b Time) Time {
	if !b.Known || (a.Known && !b.Value.After(a.Value)) {
		return a
	}
	return b
}
