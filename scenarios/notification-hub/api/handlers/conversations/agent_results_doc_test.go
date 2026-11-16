package conversations_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

// What an agent reads back from the CLI is documented by example in
// skills/notification-hub/references/cli-results.md. The examples are not typed:
// this test runs the real CLI against an in-process Hub, writes down what it
// printed, and fails when the file says anything else. Refresh it after a change
// to the output with
//
//	go test ./handlers/conversations -run TestAgentResultsDocument -update
var updateResultsDoc = flag.Bool("update", false, "rewrite skills/notification-hub/references/cli-results.md from what the CLI prints")

const resultsDocPath = "../../../skills/notification-hub/references/cli-results.md"

var (
	uuidPattern   = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	timePattern   = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z`)
	hashPattern   = regexp.MustCompile(`\b[0-9a-f]{64}\b`)
	keyPattern    = regexp.MustCompile(`ask-[0-9a-f]{20}`)
	sourcePattern = regexp.MustCompile(`"source_path": "[^"]*"`)
	hostPattern   = regexp.MustCompile(`"source_host": "[^"]*"`)
	// briefPattern is the absolute path of an example's packet brief.
	briefPattern   = regexp.MustCompile(`/[^\s"']*/work/`)
	versionPattern = regexp.MustCompile(`\(HTTP \d+\)`)
)

// canonicalJSON lays JSON documents out the same way on every run: the Go JSON
// encoder the CLI uses varies the spacing it puts between tokens from build to
// build, and the key order is the one the Hub's schema gives. stdout may hold
// several documents in a row (ask --and-wait --json prints two); each is laid
// out on its own. Text that is not JSON is returned as it is.
func canonicalJSON(text string) string {
	decoder := json.NewDecoder(strings.NewReader(text))
	var out strings.Builder
	for {
		var raw json.RawMessage
		err := decoder.Decode(&raw)
		if err == io.EOF {
			break
		}
		if err != nil {
			return text
		}
		var indented bytes.Buffer
		if json.Indent(&indented, raw, "", "  ") != nil {
			return text
		}
		out.WriteString(indented.String() + "\n")
	}
	return out.String()
}

// quietLines are stderr lines of the test environment, not of the CLI's output:
// the in-process Hub has no health route, and no Prompt Manager runs to confirm
// that the skill was read.
func quiet(stderr string) string {
	var kept []string
	skipping := false
	for _, line := range strings.Split(stderr, "\n") {
		switch {
		case strings.HasPrefix(line, "Warning: notification-hub health reports"), strings.HasPrefix(line, "Uploading "):
			continue
		case strings.HasPrefix(line, "Writing guide:"):
			skipping = true
			continue
		case skipping && strings.HasPrefix(line, "  "):
			continue
		}
		skipping = false
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// stable replaces what differs on every run (ids, times, hashes, paths) with a
// placeholder that says what it is.
func stable(text string) string {
	text = uuidPattern.ReplaceAllString(text, "<ask-id>")
	text = timePattern.ReplaceAllString(text, "<time>")
	text = hashPattern.ReplaceAllString(text, "<sha256>")
	text = keyPattern.ReplaceAllString(text, "ask-<key>")
	text = sourcePattern.ReplaceAllString(text, `"source_path": "<path>"`)
	text = hostPattern.ReplaceAllString(text, `"source_host": "<host>"`)
	text = briefPattern.ReplaceAllString(text, "<home>/work/")
	return text
}

// resultsDoc collects the examples in the order they were run.
type resultsDoc struct {
	text strings.Builder
}

func (d *resultsDoc) heading(level int, title string) {
	d.text.WriteString("\n" + strings.Repeat("#", level) + " " + title + "\n\n")
}

func (d *resultsDoc) paragraph(text string) { d.text.WriteString(text + "\n\n") }

// example writes one command, how it ended, and everything an agent can read of it.
func (d *resultsDoc) example(command string, got result) {
	d.text.WriteString("```text\n$ " + command + "\n")
	d.text.WriteString("# exit status " + string(rune('0'+got.code)) + "\n")
	stdout := got.stdout
	if strings.HasPrefix(strings.TrimSpace(stdout), "{") {
		stdout = canonicalJSON(stdout)
	}
	if out := strings.TrimRight(stable(stdout), "\n"); out != "" {
		d.text.WriteString("# stdout\n" + out + "\n")
	}
	if errText := strings.TrimRight(stable(quiet(got.stderr)), "\n"); errText != "" {
		d.text.WriteString("# stderr\n" + errText + "\n")
	}
	d.text.WriteString("```\n\n")
}

const resultsDocIntro = `# What the CLI returns

Generated, not written: ` + "`TestAgentResultsDocumentIsWhatTheCLIPrints`" + ` runs the real CLI against an
in-process Hub and fails when this file differs from what it printed. Ids,
times, hashes and paths are shown as placeholders. Refresh it with
` + "`go test ./handlers/conversations -run TestAgentResultsDocument -update`" + ` in
` + "`scenarios/notification-hub/api`" + `.

Every ` + "`--json`" + ` command prints one JSON document on stdout (` + "`ask --and-wait`" + ` prints two, the
send's and the wait's), with the Hub's field names in snake_case, on every exit status
(0, 3, 4 and 5). Guidance and the packet check
go to stderr (the writing-guide reminder, which depends on the shell, is left out of
the examples). A field with no value is left out.
`

func TestAgentResultsDocumentIsWhatTheCLIPrints(t *testing.T) {
	j := newJourney(t)
	operator := operator{j}
	doc := &resultsDoc{}
	doc.text.WriteString(resultsDocIntro)

	plan := j.file("plan.md", "# Release plan\n\n## Storage\nMove to the new volume.\n\n## Retention\nKeep 90 days.\n")
	j.file("cards.png", "card stack mockup")
	j.file("table.png", "table mockup")
	require.NoError(t, os.MkdirAll(filepath.Join(j.workdir, "work", "RELEASE-CHECK"), 0o700))
	j.file("work/RELEASE-CHECK/BRIEF.md", "# Release check\n\nOn yes: tag and publish the release. On no: hold the window.\n")
	form := j.file("form.yaml", `questions:
  - id: rollout
    type: single_choice
    prompt: Which rollout do you prefer?
    choices:
      - {key: staged, label: Staged}
      - {key: all-at-once, label: All at once}
  - id: layout
    type: variant
    prompt: Which layout should the page use?
    choices:
      - {key: cards, label: Card stack, attachment: cards.png}
      - {key: table, label: Table, attachment: table.png}
  - id: plan
    type: section_review
    prompt: Review each section of the plan.
    sections_from: plan.md
  - id: why
    type: text
    prompt: Anything else we should know?
`)
	run := func(token string, args ...string) result { return j.do(token, args...) }
	show := func(got result, args ...string) (string, result) {
		return "notification-hub conversations " + shellWords(args), got
	}

	// ask --json: what an agent reads when it raises a decision.
	doc.heading(2, "ask --json")
	doc.paragraph("`ask_id`, `state`, `outcome` and `warnings` are always present. `outcome` is `created`, `updated`, `replaced`, `reasserted`, `suppressed` or `returned`; `state` is the ask's state after the call. Each warning has `code`, `field`, `message`, `severity` (`advice`: the ask is complete but reads poorly; `dropped`: the Hub left that part out) and `carried_over`.")
	args := []string{
		"ask", "--subject-key", "release-check", "--yes-no", "--question", "Ship the release today?",
		"--big-picture", "The release is green and the window closes at noon.", "--packet", "work/RELEASE-CHECK", "--attach", "plan.md", "--attach", "cards.png", "--attach", "table.png", "--form", "form.yaml", "--json",
	}
	created := run(runOne, replacePaths(args, j, plan, form)...)
	require.Zero(t, created.code, created.stderr)
	doc.example(show(created, args...))
	askID := operator.waitForAsk("release-check")

	// A revision repeats the packet check; what the open ask already had is marked.
	doc.paragraph("`carried_over` is true for a finding the open ask already had, about a part the revision did not touch; the human output folds those into one line (`2 earlier warnings still apply`). Here the first send has no big picture, and the revision adds technical details only:")
	first := []string{"ask", "--subject-key", "disk-low", "--yes-no", "--question", "Free space on the backup drive now?", "--json"}
	opened := run(runOne, first...)
	require.Zero(t, opened.code, opened.stderr)
	doc.example(show(opened, first...))
	again := []string{"ask", "--subject-key", "disk-low", "--technical-details", "Drive at 91 percent.", "--json"}
	repeated := run(runOne, again...)
	require.Zero(t, repeated.code, repeated.stderr)
	doc.example(show(repeated, again...))

	// ask without --json: what an agent reads in a terminal.
	doc.heading(2, "ask (text output)")
	doc.paragraph("The last line of stdout is `ask_id=... state=... outcome=...`, with `idempotency_key=...` when the CLI chose the key, and `next: ...` with the command to wait. The packet check and what a revision kept go to stderr.")
	text := []string{"ask", "--subject-key", "retry-window", "--yes-no", "--question", "Retry tonight's backup to the Elements drive now?"}
	created2 := run(runOne, text...)
	require.Zero(t, created2.code, created2.stderr)
	doc.example(show(created2, text...))
	textRevision := []string{"ask", "--subject-key", "retry-window", "--technical-details", "Run 42; the drive was remounted at 21:10."}
	revised2 := run(runOne, textRevision...)
	require.Zero(t, revised2.code, revised2.stderr)
	doc.example(show(revised2, textRevision...))

	// ask --dry-run: the preview of a revision, and of a new ask as JSON.
	doc.heading(2, "ask --dry-run")
	doc.paragraph("Exit status 0 when the request is valid. Nothing is stored and no ask is created; stdout is the notification the operator would get, stderr the packet check. A keyed request is checked as the revision merged over the open ask (`Would revise open ask ...`); otherwise it says it would create one. `--json` prints `{revises_ask_id, push_title, push_body, warnings[]}`, a field with no value left out.")
	preview := []string{"ask", "--subject-key", "release-check", "--technical-details", "Run 43.", "--dry-run"}
	dry := run(runOne, preview...)
	require.Zero(t, dry.code, dry.stderr)
	doc.example(show(dry, preview...))
	previewNew := []string{"ask", "--subject-key", "gate-check", "--yes-no", "--question", "Open the gate to the staging cluster?", "--dry-run", "--json"}
	dryNew := run(runOne, previewNew...)
	require.Zero(t, dryNew.code, dryNew.stderr)
	doc.example(show(dryNew, previewNew...))

	// wait --json, exit 3: the operator asked for more context.
	doc.heading(2, "wait --json")
	doc.paragraph("Exit status 0, 3, 4 and 5 all print the same shape: `ask_id` and `state`, plus what applies. The last line of the human output is `ask_id=... state=...`.")
	doc.heading(3, "Exit 3: the operator asked for more context")
	doc.paragraph("The ask is still open (`state` is `pending`). Their question is `context_request.question`. Answer it with `ask --subject-key K --context-response '...'`, then wait again.")
	waiting := j.start(runTwo, "wait", "--subject-key", "release-check", "--json")
	operator.requestContext(askID, "Is the storage migration finished?")
	asked := waiting.finish(t)
	require.Equal(t, 3, asked.code)
	doc.example(show(asked, "wait", "--subject-key", "release-check", "--json"))

	// The answer to the question: the wait that follows it does not return at once.
	answerArgs := []string{"ask", "--subject-key", "release-check", "--context-response", "Yes. The storage migration finished at 10:42 and the checksums match.", "--json"}
	revised := run(runOne, answerArgs...)
	require.Zero(t, revised.code, revised.stderr)
	doc.paragraph("The answer is sent with `ask --subject-key K --context-response '...'`; the command's own result is the `ask` document, with `outcome` `updated`:")
	doc.example(show(revised, answerArgs...))

	// exit 4
	doc.heading(3, "Exit 4: your wait timed out")
	doc.paragraph("`--timeout` ran out. Nothing happened to the ask: it is still open and expires on its own schedule (`--expires`). Wait again, or carry on and read it later with `get`.")
	late := run(runTwo, "wait", "--subject-key", "release-check", "--timeout", "1s", "--json")
	require.Equal(t, 4, late.code, "the answer addressed the operator's question, so the wait does not return it again")
	doc.example(show(late, "wait", "--subject-key", "release-check", "--timeout", "1s", "--json"))

	// exit 0 with every typed shape
	doc.heading(3, "Exit 0: answered")
	doc.paragraph("`answer` is the option key the operator chose (or the default that applied), `note` their free text. `answer_summary` is the whole answer as plain text. `responses[]` holds one entry per form question the operator answered, found by `question_id`: `selected` (a list of choice keys) for `single_choice`, `multi_choice` and `variant`; `text` for `text`; `sections` (a list of `{section_key, verdict, note}`, the verdict `approve`, `change` or `drop`) for `section_review`. `attachment_notes[]` are notes the operator pinned to files. Page selections are in `changes`, and attached answer images are in `images[]` with their SHA-256 and media type. A status update can be marked read without an answer. The main answer is `answer`, not a `responses[]` entry.")
	waiting = j.start(runTwo, "wait", "--subject-key", "release-check", "--json")
	operator.answer(&v1.AnswerRequest{
		AskId: askID, Answer: "yes", Note: "Go ahead.",
		Responses: []*v1.AskQuestionResponse{
			{QuestionId: "rollout", Selected: []string{"staged"}},
			{QuestionId: "layout", Selected: []string{"cards"}},
			{QuestionId: "plan", Sections: []*v1.AskSectionVerdict{{SectionKey: "storage", Verdict: "approve"}, {SectionKey: "retention", Verdict: "change", Note: "Keep 30 days."}}},
			{QuestionId: "why", Text: "Watch the disk graph for an hour."},
		},
		AttachmentNotes: []*v1.AskAttachmentNote{{Attachment: "plan.md", Note: "The storage section is the one that worries me."}},
	})
	answered := waiting.finish(t)
	require.Zero(t, answered.code, answered.stderr)
	doc.example(show(answered, "wait", "--subject-key", "release-check", "--json"))

	// ask --and-wait: the send and the wait in one command.
	doc.heading(2, "ask --and-wait")
	doc.paragraph("The command prints the send's result first and the wait's result after it, then exits with the status `wait` would (0, 3, 4 or 5). In text mode the output has two `ask_id=` lines and the last line is the wait's (`ask_id=... state=... answer=...`); read the id from the first. With `--json` stdout holds two documents, one after the other: the `ask` document, then the `wait` document. A send that fails exits 1 and prints only its error; a wait that fails after a good send exits 1 as well, after the `ask` document. The ask's id is in the first document (`ask_id`); wait again later by `--subject-key`.")
	andWait := []string{"ask", "--subject-key", "lunch-order", "--yes-no", "--question", "Order lunch for the team today?", "--big-picture", "The team is in the office all day and nobody has planned lunch.", "--and-wait", "--timeout", "1m"}
	pendingWait := j.start(runOne, andWait...)
	lunchID := operator.waitForAsk("lunch-order")
	operator.answer(&v1.AnswerRequest{AskId: lunchID, Answer: "yes", Note: "Pizza."})
	waited := pendingWait.finish(t)
	require.Zero(t, waited.code, waited.stderr)
	doc.example(show(waited, andWait...))
	andWaitJSON := []string{"ask", "--subject-key", "dessert-order", "--yes-no", "--question", "Order dessert too?", "--big-picture", "Lunch is ordered and there is budget left for dessert.", "--and-wait", "--timeout", "1m", "--json"}
	pendingWait = j.start(runOne, andWaitJSON...)
	dessertID := operator.waitForAsk("dessert-order")
	operator.answer(&v1.AnswerRequest{AskId: dessertID, Answer: "no"})
	waitedJSON := pendingWait.finish(t)
	require.Zero(t, waitedJSON.code, waitedJSON.stderr)
	doc.example(show(waitedJSON, andWaitJSON...))

	// exit 5 with a reason
	doc.heading(3, "Exit 5: closed without an answer")
	doc.paragraph("`state` is `withdrawn`, `dismissed` or `expired`; `reason` says why (the asker's reason, the operator's note, or that the ask expired). By ask id a revision with new options also ends here, as `withdrawn` with the reason `replaced`; by `--subject-key` the wait follows the replacement instead.")
	created = run(runOne, "ask", "--subject-key", "old-volume", "--yes-no", "--question", "Roll back to the old volume?", "--big-picture", "The new volume is healthy; the old one is still mounted.", "--json")
	require.Zero(t, created.code, created.stderr)
	closing := j.start(runTwo, "wait", "--subject-key", "old-volume", "--json")
	operator.waitForAsk("old-volume")
	withdrawn := run(runOne, "withdraw", "--subject-key", "old-volume", "--reason", "The release decision made this moot.")
	require.Zero(t, withdrawn.code, withdrawn.stderr)
	gone := closing.finish(t)
	require.Equal(t, 5, gone.code)
	doc.example(show(gone, "wait", "--subject-key", "old-volume", "--json"))

	// withdraw
	doc.heading(2, "withdraw")
	doc.paragraph("`--reason` is required. Exit status 0 when the ask was closed; the last line of stdout says `Withdrew ask ...`. A run waiting on the ask ends with status 5 and sees the reason. `--json` prints the ask as a document.")
	withdrawArgs := []string{"withdraw", "--subject-key", "gate-check-old", "--reason", "The gate was opened by hand."}
	created = run(runOne, "ask", "--subject-key", "gate-check-old", "--yes-no", "--question", "Open the gate to the staging cluster?", "--big-picture", "The staging cluster is closed to everyone.", "--json")
	require.Zero(t, created.code, created.stderr)
	plainWithdraw := run(runOne, withdrawArgs...)
	require.Zero(t, plainWithdraw.code, plainWithdraw.stderr)
	doc.example(show(plainWithdraw, withdrawArgs...))
	created = run(runOne, "ask", "--subject-key", "gate-check-json", "--yes-no", "--question", "Open the gate to the production cluster?", "--big-picture", "The production cluster is closed to everyone.", "--json")
	require.Zero(t, created.code, created.stderr)
	jsonWithdrawArgs := []string{"withdraw", "--subject-key", "gate-check-json", "--reason", "The gate was opened by hand.", "--json"}
	jsonWithdraw := run(runOne, jsonWithdrawArgs...)
	require.Zero(t, jsonWithdraw.code, jsonWithdraw.stderr)
	doc.example(show(jsonWithdraw, jsonWithdrawArgs...))
	noReason := run(runOne, "withdraw", "--subject-key", "release-check")
	require.Equal(t, 1, noReason.code)
	doc.paragraph("Without `--reason` the command fails before it reaches the Hub:")
	doc.example(show(noReason, "withdraw", "--subject-key", "release-check"))

	// get --json
	doc.heading(2, "get --json")
	doc.paragraph("`{ask: {...}}`: the whole ask, with `options`, `attachments`, `questions`, `responses`, `context_requests` (oldest first; each has the operator's `question` and, once you answered it, your `response` and `addressed_at`) and the rest. A count with no value is left out, so read a missing `open_context_request_count` as 0.")
	read := run(runTwo, "get", "--subject-key", "old-volume", "--json")
	require.Zero(t, read.code, read.stderr)
	doc.example(show(read, "get", "--subject-key", "old-volume", "--json"))
	doc.paragraph("An answered ask with a form, files, an operator's question and the answer to it. `responses[]` has one entry per form question the operator answered (`question_id` and the shape of its type); `attachment_notes[]` are their notes on files; `context_requests[]` has each question with your `response`:")
	answeredRead := run(runTwo, "get", "--subject-key", "release-check", "--json")
	require.Zero(t, answeredRead.code, answeredRead.stderr)
	doc.example(show(answeredRead, "get", "--subject-key", "release-check", "--json"))

	got := doc.text.String()
	if *updateResultsDoc {
		require.NoError(t, os.WriteFile(filepath.Clean(resultsDocPath), []byte(got), 0o644))
	}
	want, err := os.ReadFile(resultsDocPath)
	require.NoError(t, err, "the results document is missing: run the test with -update")
	require.Equal(t, string(want), got, "the CLI prints something other than references/cli-results.md says; run the test with -update and review the change")
}

// replacePaths turns the file names of an example command into the paths the CLI
// is run with, so the documented command reads as an agent would type it.
func replacePaths(args []string, j *journey, plan, form string) []string {
	out := make([]string, len(args))
	for i, arg := range args {
		switch arg {
		case "plan.md":
			out[i] = plan
		case "form.yaml":
			out[i] = form
		case "cards.png", "table.png", "work/RELEASE-CHECK":
			out[i] = filepath.Join(j.workdir, arg)
		default:
			out[i] = arg
		}
	}
	return out
}

// shellWords writes arguments the way an agent would type them: words with
// spaces in single quotes.
func shellWords(args []string) string {
	words := make([]string, len(args))
	for i, arg := range args {
		words[i] = arg
		if strings.ContainsAny(arg, " ?'\"") {
			words[i] = "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
		}
	}
	return strings.Join(words, " ")
}
