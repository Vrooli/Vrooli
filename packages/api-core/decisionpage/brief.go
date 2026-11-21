package decisionpage

import (
	"fmt"
	"path/filepath"
	"strings"
)

// BriefFile is the brief a writer run follows, one per attempt.
const BriefFile = "BRIEF.md"

// PageFile is the page a writer run writes into the job folder.
const PageFile = "page.html"

// ReturnFile is the writer run's short account of what it made and from what.
const ReturnFile = "RETURN.md"

// ShotsDir holds the run's phone screenshots.
const ShotsDir = "shots"

// Prompt is what the run is told; the brief holds the whole assignment.
func Prompt(dir string) string {
	return "Read " + filepath.Join(dir, BriefFile) + " and carry it out completely. It is your whole assignment. " +
		"Write only inside " + dir + " and send nothing."
}

// briefInput is what a brief is rendered from.
type briefInput struct {
	Dir      string
	Request  Request
	Attempt  int
	Previous *Attempt // the failed attempt a retry fixes; nil on the first
	SkillDir string   // the notification-hub skill folder (examples, references)
	Command  string   // the CLI that checks and photographs a page
}

// renderBrief writes the assignment. It is followable by a lower-powered
// model: numbered steps, one exact command per check, and the facts inline,
// so nothing has to be inferred from context the run does not have.
func renderBrief(in briefInput) string {
	r := in.Request
	page := filepath.Join(in.Dir, PageFile)
	var b strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }

	w("# Make one decision page\n\n")
	w("You make the page the operator sees in Notification Hub, on a phone, above the answer buttons of one decision. ")
	w("The operator reads it cold, between other things. A good page makes the choice obvious in ten seconds and shows ")
	w("what the question and options do not: the real numbers, the real list, the command, the cause. ")
	w("A page that repeats the question and the options in boxes is a failure, however polished it looks.\n\n")
	w("Your folder: `%s`. Write `%s`, `%s` and `%s/` there and nowhere else. Send nothing: the sender sends the decision.\n\n",
		in.Dir, PageFile, ReturnFile, ShotsDir)
	if in.Attempt > 1 && in.Previous != nil {
		w("## This is attempt %d: fix what failed\n\n", in.Attempt)
		w("Your first page is at `%s`. It failed the checks below. Start from it, fix every finding, and keep what was good.\n\n",
			filepath.Join(in.Dir, attemptDir(in.Previous.N), PageFile))
		if in.Previous.Problem != "" {
			w("- %s\n", in.Previous.Problem)
		}
		for _, f := range in.Previous.Findings {
			w("- `%s`: %s\n", f.Code, f.Message)
		}
		w("\n")
	}

	w("## 1. Read these, in order (nothing else is needed)\n\n")
	w("1. `prompt-manager skill read notification-hub`: the whole skill. Its \"Make the page\" section is your job.\n")
	w("2. `%s`: the page contract, the kit's colour tokens and blocks.\n", filepath.Join(in.SkillDir, "references", "html-explainers.md"))
	w("3. `ls %s`: the starters. Open the one closest to this decision, copy it to `%s`, then replace every word and number.\n",
		filepath.Join(in.SkillDir, "examples"), page)
	if len(r.Evidence) > 0 {
		w("4. The evidence, for the real numbers, names and lists:\n")
		for _, p := range r.Evidence {
			w("   - `%s`\n", p)
		}
		w("   A file that names another file with the details (a table, a list, a report) is worth opening too.\n")
	}
	w("\n")

	w("## 2. The decision\n\n")
	w("The Hub already shows the question, the big picture and the answer buttons as text around your page. ")
	w("Your page must add to them, not repeat them.\n\n")
	w("- Asked by: %s\n", r.Sender)
	w("- Question: %s\n", r.Question)
	if r.BigPicture != "" {
		w("- Big picture: %s\n", oneLine(r.BigPicture))
	}
	w("- Answers (key: label; what happens next):\n")
	for _, o := range r.Options {
		mark := ""
		if o.Key == r.Recommended {
			mark = " (recommended)"
		}
		w("  - `%s`%s: %s", o.Key, mark, o.Label)
		if o.Effect != "" {
			w("; %s", oneLine(o.Effect))
		}
		w("\n")
	}
	if r.Recommended != "" {
		reason := r.RecommendationReason
		if reason == "" {
			reason = "(the sender gave no reason: find it in the evidence and state it on the page)"
		}
		w("- Recommendation: `%s`, because %s\n", r.Recommended, oneLine(reason))
	}
	if r.IfNoAnswer != "" {
		w("- If there is no answer: %s\n", oneLine(r.IfNoAnswer))
	}
	if r.Cause != "" {
		w("- Why this is coming up (the cause): %s\n", oneLine(r.Cause))
	}
	if r.History != "" {
		w("- Asked before: %s\n", oneLine(r.History))
	}
	for _, c := range r.Commands {
		w("- The operator must run: `%s`", c.Run)
		if c.For != "" {
			w(" (for `%s`)", c.For)
		}
		if c.Where != "" {
			w("; where: %s", oneLine(c.Where))
		}
		if c.Shows != "" {
			w("; it shows before acting: %s", oneLine(c.Shows))
		}
		if c.Undo != "" {
			w("; undo: %s", oneLine(c.Undo))
		}
		w("\n")
	}
	if r.Facts != "" {
		w("\nFacts the sender holds:\n\n%s\n", strings.TrimSpace(r.Facts))
	}
	if r.TechnicalDetails != "" {
		w("\nTechnical details (for you; never show IDs or paths from here on the page):\n\n```text\n%s\n```\n", strings.TrimSpace(r.TechnicalDetails))
	}
	w("\n")

	w("Present what the sender and the evidence say, and invent nothing: no command, check, number, date or effect of an ")
	w("answer that is not written there. When the page needs something the evidence lacks, leave it out and name it in ")
	w("RETURN.md under \"Missing from the evidence\"; the sender fixes its brief, not you.\n\n")
	w("## 3. Plan the content before any design\n\n")
	w("Open `%s` and write, before you touch the page:\n\n", filepath.Join(in.Dir, ReturnFile))
	w("1. **New facts**: at least four facts from the evidence that the question, big picture and answers above do not say ")
	w("(a count, a size, a date, a name, what each item is, what it costs, what breaks). Write each with the file it came from.\n")
	w("2. **The shape of the data** and the one picture that shows it: pick the one row of this table that fits best ")
	w("and build that picture, not one from each row:\n\n")
	w("| The decision is about | Show it as (kit block) |\n|---|---|\n")
	w("| Many things of one kind (six or more files, runs, accounts, costs) | Groups with a count and what each group is (`.hub-groups`), plus a bar of their share or size (`.hub-proportion`). Never one card per item. |\n")
	w("| Several separate rules or changes in one ask | A numbered list, one row each: what it does with its real number, a `.hub-chip` for how reversible it is (Reversible / Deletes / No end date), and the default if declined. This list is the picture: put it on the first screen, right after the recommendation. |\n")
	w("| Sizes, money, time or counts | A `.hub-proportion` bar with the exact numbers in its legend, largest first, the long tail folded into one share. |\n")
	w("| Options against a few criteria | A `table.hub-matrix` (cells marked `data-tone=\"good|warn|bad\"`), recommended row marked. |\n")
	w("| A command the operator must run | A `.hub-command` block on the first screen: the exact command with a Copy button, where to run it, what it shows before it acts, how to undo it. |\n")
	w("| The same question asked before | A `.hub-recurrence` strip: how many times, since when, and which answer stops the repeats. |\n")
	w("| A plan over time | A `.hub-timeline` with today marked and the stop rule. |\n")
	w("| A change to an app's screen | Before and after (`data-hub-compare`) with real screenshots. |\n\n")
	w("The exact markup of each block is in the html-explainers reference you read in step 1. Two you will often need:\n\n")
	b.WriteString(kitSnippets)
	w("Every shape on the page must stand for a real number or name from your facts. A grid of identical squares, ")
	w("an icon row, a big lonely number or a bar that counts your own categories (\"4 of 8 are operating rules\") is ")
	w("decoration, not a picture of the data. A bar shows measured sizes: lines, bytes, money, hours, notifications.\n")
	w("3. **The cause**: if the decision exists because of a gate, a defect or a rule, or has been asked before, say so in one ")
	w("sentence on the page and say which answer fixes the cause rather than this one instance. If no answer fixes the cause, ")
	w("say what would.\n\n")

	w("## 4. Build the page\n\n")
	w("1. First screen, in this order: a one-line headline that names the real stake (not the question again); the ")
	w("recommendation and its reason in one sentence; then the picture from step 3.2, with any sentence that introduces it ")
	w("above it. If an answer needs a command, the recommended answer's command block is on the first screen too; when ")
	w("each answer has its own command, each answer's card holds its own command block.\n")
	w("2. One card per answer, recommended first, made with the kit's cards so they are padded and themed: ")
	w("`<div class=\"hub-tradeoffs\"><section class=\"is-recommended\">...</section><section>...</section></div>`. ")
	w("Each says what happens next, what it costs or risks, and ends with a button:\n")
	w("   `<button type=\"button\" data-answer=\"KEY\">Choose this</button>` for each key (%s), wired once:\n", quoteKeys(r.Keys()))
	w("   `document.querySelectorAll('[data-answer]').forEach(b => b.addEventListener('click', () => window.hub && window.hub.prefill && window.hub.prefill('answer', b.dataset.answer)));`\n")
	w("   Under the cards: \"Tap Choose this, then send.\"\n")
	w("3. What happens with no answer, as a real consequence.\n")
	w("4. Use the kit blocks exactly as the reference writes them. Never restyle a `.hub-*` class in your own `<style>`: ")
	w("the kit draws them in both themes and keeps a bar and its legend in the same colours. Your own `<style>` lays out ")
	w("only your own elements, and never sets a font: the kit's typeface and sizes are the Hub's.\n")
	w("5. Colours: only the kit tokens (`var(--hub-fg)`, `var(--hub-surface)`, `var(--hub-muted)`, `var(--hub-border)`, ")
	w("`var(--hub-accent)`, `var(--hub-success)`, `var(--hub-warning)`, `var(--hub-danger)`). Never write a literal white, ")
	w("black or hex background or text colour: the Hub has a dark theme and a fixed light card becomes unreadable there.\n")
	w("6. Plain words. No IDs, run names, file paths, test or build vocabulary on the page, with one exception: a command ")
	w("the operator must run is shown exactly as given above, character for character, IDs included, so Copy gives them ")
	w("something that works. Never shorten a command or put placeholders in it. Under 2 MiB, nothing loaded from ")
	w("the network, no `<form>`, `<input>`, `<select>` or `<textarea>`.\n\n")

	w("## 5. Check it, look at it, fix it\n\n")
	w("1. `%s pages check %s --cover-file %s`: Notification Hub's own packet check on your page with this decision. ", in.Command, page, filepath.Join(in.Dir, requestFile))
	w("Fix every blocking finding about the page and run it again until nothing about the page would refuse the ask. Findings about the question, the reason or the packet are the sender's; leave them. The sender runs the same check; a page that fails it is not sent.\n")
	w("2. `%s pages shoot %s`: phone screenshots at 390 px, light and dark, into `%s/`. ", in.Command, page, filepath.Join(in.Dir, ShotsDir))
	w("Open every screenshot and look at it as the operator would: is the stake clear in the first screen, is every number ")
	w("readable in both themes, does anything overflow or overlap? Fix and shoot again until all are right.\n\n")

	w("## 6. Finish\n\n")
	w("Complete `%s`: your new facts with their sources, the picture you chose and why it makes the choice clear, the ", filepath.Join(in.Dir, ReturnFile))
	w("starter you began from, anything missing from the evidence, the last check result and the screenshot paths. Then stop.\n")
	return b.String()
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

func quoteKeys(keys []string) string {
	q := make([]string, len(keys))
	for i, k := range keys {
		q[i] = "`" + k + "`"
	}
	return strings.Join(q, ", ")
}

// kitSnippets are the two kit blocks a writer needs most, as the kit draws
// them (operatordecision.Proportion and Command).
const kitSnippets = "```html\n" + `<figure class="hub-proportion"><figcaption><span>Where the 3.1 GiB is</span><strong>3.1 GiB</strong></figcaption>
  <div class="hub-proportion__bar" role="img" aria-label="Claude Code 68%, Grok 12%"><span data-tone="1" style="--share:68%"></span><span data-tone="2" style="--share:12%"></span></div>
  <ol class="hub-proportion__legend"><li data-tone="1"><span class="hub-proportion__label">Claude Code<small>transcripts, memory</small></span><span class="hub-proportion__value">2.1 GiB<small>68%</small></span></li></ol></figure>
<section class="hub-command"><p class="hub-command__where">In a terminal on the Vrooli computer <small>about 10 minutes</small></p>
  <div class="hub-command__line"><code>cd ~/Vrooli &amp;&amp; make rebaseline</code><button type="button" class="is-quiet" data-hub-copy>Copy</button></div>
  <ol class="hub-command__steps"><li>...</li></ol><p><strong>Before it changes anything:</strong> ...</p><p><strong>Undo:</strong> ...</p></section>
` + "```\n\n"
