package operatordecision

import (
	"strings"
	"testing"
)

func assertContains(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("page lacks %q", want)
		}
	}
}

func TestProportionDrawsEachShareOfTheWholeWithExactValuesInTheLegend(t *testing.T) {
	got := NewPage("Back up your AI history?").Proportion("Where the 3.1 GiB is", "3.1 GiB",
		Share{Label: "Claude Code", Value: 2100, Display: "2.1 GiB", Note: "transcripts, memory, sign-in"},
		Share{Label: "Grok", Value: 365, Display: "365 MiB"},
		Share{Label: "40 <others>", Value: 4, Display: "≈ 4 MiB"},
	).HTML()
	assertContains(t, got,
		`<figure class="hub-proportion"><figcaption><span>Where the 3.1 GiB is</span><strong>3.1 GiB</strong></figcaption>`,
		`<span data-tone="1" style="--share:85.05%"></span>`,
		`<span data-tone="3" style="--share:0.16%"></span>`,
		`<span class="hub-proportion__label">Claude Code<small>transcripts, memory, sign-in</small></span><span class="hub-proportion__value">2.1 GiB<small>85%</small></span>`,
		"40 &lt;others&gt;", "<small>&lt;1%</small>",
		`role="img" aria-label="Claude Code 2.1 GiB (85%), Grok 365 MiB (15%), 40 &lt;others&gt; ≈ 4 MiB (&lt;1%)"`,
	)
	if empty := NewPage("x").Proportion("Nothing", "", Share{Label: "a"}).HTML(); strings.Contains(empty, "hub-proportion") {
		t.Error("a proportion with no positive share drew an empty bar")
	}
}

func TestGroupsTieTogglesToTheFormQuestionAndFoldTheItems(t *testing.T) {
	got := NewPage("Back up by group?").Groups("groups",
		Group{Key: "ai", Title: "AI tools' history", Meta: "12 items · 2.9 GiB", Summary: "Exists nowhere else.", Items: []string{"Claude Code", "Codex"}, Selected: true},
		Group{Key: "host", Title: "Host settings", Meta: "23 tiny files"},
	).HTML()
	assertContains(t, got,
		`<div class="hub-groups" data-hub-groups="groups">`,
		`<section class="hub-group" data-key="ai"><div class="hub-group__head"><h3>AI tools&#39; history</h3><button type="button" class="hub-group__toggle" aria-pressed="true"`,
		`<section class="hub-group" data-key="host"><div class="hub-group__head"><h3>Host settings</h3><button type="button" class="hub-group__toggle" aria-pressed="false"`,
		`<p class="hub-group__meta">12 items · 2.9 GiB</p>`,
		`<details><summary>Show the 2 items</summary><ul><li>Claude Code</li><li>Codex</li></ul></details>`,
	)
	shown := NewPage("x").Groups("", Group{Key: "ai", Title: "AI"}).HTML()
	if strings.Contains(shown, "data-hub-groups") || strings.Contains(shown, "hub-group__toggle") {
		t.Error("groups with no form question still offered switches that fill nothing")
	}
	for _, banned := range []string{"<input", "<form", "<select"} {
		if strings.Contains(got, banned) {
			t.Errorf("groups use %s, which the Hub flags as a form control", banned)
		}
	}
}

func TestCommandShowsWhereWhatFirstAndUndoBesideTheExactLine(t *testing.T) {
	got := NewPage("Run one command?").Command(Command{
		Line:     "cd ~/Vrooli && make rebaseline",
		Where:    "In a terminal on the Vrooli computer",
		Duration: "about 10 minutes",
		Steps:    []string{"Runs the full build check.", "Lists the stored-data changes and waits for you."},
		Shows:    "the five areas whose stored data changes, and asks you to confirm",
		Undo:     "no automatic rollback; a saved copy makes a manual one possible",
	}).HTML()
	assertContains(t, got,
		`<p class="hub-command__where">In a terminal on the Vrooli computer <small>about 10 minutes</small></p>`,
		`<div class="hub-command__line"><code>cd ~/Vrooli &amp;&amp; make rebaseline</code><button type="button" class="is-quiet" data-hub-copy>Copy</button></div>`,
		`<ol class="hub-command__steps"><li>Runs the full build check.</li>`,
		"<strong>Before it changes anything:</strong> the five areas",
		"<strong>Undo:</strong> no automatic rollback",
	)
	if NewPage("x").Command(Command{Where: "anywhere"}).HTML() != NewPage("x").HTML() {
		t.Error("a command block without a command line was drawn")
	}
}

func TestRecurrenceNamesTheRepeatAndTheAnswerThatEndsIt(t *testing.T) {
	got := NewPage("Keep the cheaper model on?").Recurrence(Recurrence{
		Count: 14, Since: "4 Nov", Marks: []string{"4 Nov · keep on", "6 Nov · keep on"},
		Stop: "“Keep it on and stop asking” ends the daily question.",
	}).HTML()
	assertContains(t, got,
		`<aside class="hub-recurrence"><strong class="hub-recurrence__count">14×</strong>`,
		"<p>You have been asked this 14 times since 4 Nov.</p>",
		`<p class="hub-recurrence__stop">“Keep it on and stop asking” ends the daily question.</p>`,
		`<li title="4 Nov · keep on"><span>4 Nov · keep on</span></li>`,
	)
	if strings.Contains(NewPage("x").Recurrence(Recurrence{Count: 1}).HTML(), "hub-recurrence") {
		t.Error("a first ask was shown as a repeat")
	}
}

func TestMatrixMarksEachCellAndHighlightsTheRecommendedRow(t *testing.T) {
	got := NewPage("Which bar should this work meet?").Matrix(
		[]string{"Work resumes", "Code shrinks"}, "shrink",
		MatrixRow{Key: "keep", Label: "Keep 95%", Cells: []Cell{{"Not until a set is found", ToneBad}, {"By 95% at once", ToneGood}}},
		MatrixRow{Key: "shrink", Label: "Every change shrinks", Cells: []Cell{{"Now", ToneGood}, {"Steadily", ToneWarn}}},
		MatrixRow{Key: "drop", Label: "Drop the bar", Cells: []Cell{{"Now", "loud"}}},
	).HTML()
	assertContains(t, got,
		`<table class="hub-matrix"><thead><tr><td></td><th scope="col">Work resumes</th><th scope="col">Code shrinks</th></tr></thead>`,
		`<tr data-key="keep"><th scope="row">Keep 95%</th><td data-label="Work resumes" data-tone="bad">Not until a set is found</td>`,
		`<tr class="is-recommended" data-key="shrink">`,
		`<td data-label="Work resumes">Now</td><td data-label="Code shrinks"></td></tr>`,
	)
}
