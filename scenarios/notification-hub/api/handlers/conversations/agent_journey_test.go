package conversations_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"

	"notification-hub/internal/hub"
)

// The agent journey: one agent run raises, revises, follows and withdraws
// decisions with the real notification-hub CLI binary, while a second run of the
// same conversation reads and waits by subject key and the operator answers
// through the Hub's own API. Everything runs against an in-process Hub on a
// temporary database, so nothing reaches the live Hub or the operator's phone.
//
// The test asserts only what an agent relies on: the exit status, the one
// machine-readable line each command ends with, and the stable JSON.

// journey is the CLI built once, and the places it runs.
type journey struct {
	t       *testing.T
	fx      *readerFixture
	bin     string
	home    string
	workdir string
	pushes  *askPushCounter
}

// buildCLI compiles the CLI that agents run.
func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "notification-hub")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = filepath.Join("..", "..", "..", "cli")
	out, err := build.CombinedOutput()
	require.NoError(t, err, "building the CLI: %s", out)
	return bin
}

func newJourney(t *testing.T) *journey {
	t.Helper()
	fx := newReaderFixture(t)
	pushes := &askPushCounter{}
	fx.service.SetPushSender(pushes)
	require.NoError(t, fx.service.RegisterPushSubscription(context.Background(), "alice", hub.PushSubscription{
		ID: "phone", Endpoint: "https://push.example.test/phone", P256DH: "key", Auth: "secret", Origin: "https://app.example.test",
	}))
	return &journey{t: t, fx: fx, bin: buildCLI(t), home: t.TempDir(), workdir: t.TempDir(), pushes: pushes}
}

// file writes a file in the agent's working directory and returns its path.
func (j *journey) file(name, content string) string {
	j.t.Helper()
	path := filepath.Join(j.workdir, name)
	require.NoError(j.t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// run is one CLI process: a run of the conversation, named by its token.
type run struct {
	cmd            *exec.Cmd
	stdout, stderr *lockedOutput
	done           chan struct{}
	err            error
}

type lockedOutput struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (o *lockedOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.buf.Write(p)
}

func (o *lockedOutput) String() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.buf.String()
}

// result is what one finished command left an agent to read.
type result struct {
	stdout, stderr string
	code           int
}

// start launches the CLI as the run holding token, with only the environment an
// agent session has: the Hub's address and its identity token.
func (j *journey) start(token string, args ...string) *run {
	j.t.Helper()
	cmd := exec.Command(j.bin, append([]string{"--api-base", j.fx.url, "conversations"}, args...)...)
	cmd.Dir = j.workdir
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + j.home, "API_BASE_URL=" + j.fx.url, "VROOLI_AGENT_IDENTITY_TOKEN=" + token}
	r := &run{cmd: cmd, stdout: &lockedOutput{}, stderr: &lockedOutput{}, done: make(chan struct{})}
	cmd.Stdout, cmd.Stderr = r.stdout, r.stderr
	require.NoError(j.t, cmd.Start())
	go func() {
		r.err = cmd.Wait()
		close(r.done)
	}()
	j.t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
	return r
}

// finish waits for the command to end.
func (r *run) finish(t *testing.T) result {
	t.Helper()
	select {
	case <-r.done:
	case <-time.After(asyncWait):
		t.Fatalf("the command did not finish\nstdout:\n%s\nstderr:\n%s", r.stdout.String(), r.stderr.String())
	}
	code := 0
	var exit *exec.ExitError
	if errors.As(r.err, &exit) {
		code = exit.ExitCode()
	} else {
		require.NoError(t, r.err)
	}
	t.Logf("$ notification-hub conversations %s\nexit %d\nstdout:\n%s\nstderr:\n%s", strings.Join(r.cmd.Args[4:], " "), code, r.stdout.String(), r.stderr.String())
	return result{stdout: r.stdout.String(), stderr: r.stderr.String(), code: code}
}

// do runs a command to the end.
func (j *journey) do(token string, args ...string) result {
	j.t.Helper()
	return j.start(token, args...).finish(j.t)
}

var endLine = regexp.MustCompile(`(?m)^ask_id=(\S+) state=(\S+)(?: outcome=(\S+))?`)

// endOf is the last `ask_id=... state=...` line of a command's output.
func endOf(t *testing.T, out string) (askID, state, outcome string) {
	t.Helper()
	all := endLine.FindAllStringSubmatch(out, -1)
	require.NotEmpty(t, all, "the output must end with an ask_id= line an agent can parse:\n%s", out)
	last := all[len(all)-1]
	return last[1], last[2], last[3]
}

// asJSON decodes a command's --json output.
func asJSON(t *testing.T, out string) map[string]any {
	t.Helper()
	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &decoded), "stdout must be one JSON document:\n%s", out)
	return decoded
}

// operator acts as the signed-in owner, over the Hub's API.
type operator struct {
	j *journey
}

func (o operator) requestContext(askID, question string) {
	o.j.t.Helper()
	_, err := o.j.fx.client.RequestContext(context.Background(), reading("owner", &v1.RequestContextRequest{AskId: askID, Question: question}))
	require.NoError(o.j.t, err)
}

func (o operator) answer(msg *v1.AnswerRequest) {
	o.j.t.Helper()
	_, err := o.j.fx.client.Answer(context.Background(), reading("owner", msg))
	require.NoError(o.j.t, err)
}

// asks is every ask the operator can see.
func (o operator) asks() []*v1.Ask {
	o.j.t.Helper()
	list, err := o.j.fx.client.ListAsks(context.Background(), reading("owner", &v1.ListAsksRequest{}))
	require.NoError(o.j.t, err)
	return list.Msg.GetAsks()
}

// waitForAsk blocks until the Hub holds an ask under the key and returns its id.
func (o operator) waitForAsk(subjectKey string) string {
	o.j.t.Helper()
	var id string
	require.Eventually(o.j.t, func() bool {
		for _, ask := range o.asks() {
			if ask.GetSubjectKey() == subjectKey {
				id = ask.GetId()
				return true
			}
		}
		return false
	}, asyncWait, 20*time.Millisecond)
	return id
}

const (
	planV1Text    = "# Release plan\n\n## Storage\nMove to the new volume.\n\n## Retention\nKeep 90 days.\n"
	explainerText = "<!doctype html><html><head><title>Release</title></head><body><h1>Release check</h1>" +
		"<table><tr><th>Check</th><th>Result</th></tr><tr><td>Unit and end-to-end suites</td><td>1,284 passed in 9 minutes</td></tr>" +
		"<tr><td>Storage migration dry run</td><td>3 volumes copied, checksums match</td></tr><tr><td>Canary on staging</td><td>No errors over 2 hours of traffic</td></tr></table>" +
		"<p>Shipping today puts the payments fix in front of customers before the weekend promotion; waiting moves it to Monday, after the traffic peak, with the rollback plan unchanged.</p></body></html>"
)

const (
	runOne = "valid-run-1+c=conversation-1"
	runTwo = "valid-run-2+c=conversation-1"
)

func TestAgentJourney_RaiseReviseFollowAndWithdrawDecisionsWithTheCLI(t *testing.T) {
	j := newJourney(t)
	operator := operator{j}
	plan := j.file("plan.md", planV1Text)
	notes := j.file("notes.md", "Benchmark notes.\n")
	explainer := j.file("explainer.html", explainerText)
	form := j.file("form.yaml", `questions:
  - id: layout
    type: single_choice
    prompt: Which rollout do you prefer?
    required: true
    choices:
      - {key: staged, label: Staged}
      - {key: all-at-once, label: All at once}
    recommended: staged
  - id: plan
    type: section_review
    prompt: Review each section of the plan.
    sections_from: plan.md
  - id: why
    type: text
    prompt: Anything else we should know?
`)
	_ = notes

	// 1. Preview: nothing is sent, nothing is stored, nobody is pushed.
	preview := j.do(runOne, "ask", "--subject-key", "release-check", "--yes-no",
		"--question", "Ship the release today?", "--attach", plan, "--explainer", explainer, "--form", form, "--dry-run")
	require.Zero(t, preview.code, preview.stderr)
	require.Contains(t, preview.stdout, "Dry run: nothing was stored and no ask was created.")
	require.Contains(t, preview.stderr, "explainer_no_viewport", "the check read the explainer page it was given, though nothing was uploaded")
	require.Contains(t, preview.stdout, "Ship the release today?")
	require.Empty(t, operator.asks(), "a dry run stores no ask")
	require.Zero(t, j.pushes.n, "a dry run pushes nothing")
	missing, err := j.fx.service.StatAttachmentBlobs(context.Background(), []string{sha256Hex(planV1Text), sha256Hex(explainerText)})
	require.NoError(t, err)
	require.Len(t, missing, 2, "a dry run uploads nothing: the Hub holds neither the plan nor the explainer")
	require.NotContains(t, preview.stdout+preview.stderr, "Uploading", "and does not say it did")

	// A dry run and --and-wait contradict each other; the command says so at once.
	both := j.do(runOne, "ask", "--subject-key", "release-check", "--yes-no", "--question", "Ship the release today?", "--dry-run", "--and-wait")
	require.NotZero(t, both.code)
	require.Contains(t, both.stderr, "--dry-run creates no ask")

	// 2. Raise it for real and wait in the same command. The operator asks for
	// more context, so the command ends with status 3 and the question printed.
	raising := j.start(runOne, "ask", "--subject-key", "release-check", "--yes-no", "--recommend", "yes", "--recommend-reason", "Every check is green and the window closes at noon.",
		"--question", "Ship the release today?", "--big-picture", "The release is green and the window closes at noon.",
		"--attach", plan, "--explainer", explainer, "--form", form, "--and-wait")
	askID := operator.waitForAsk("release-check")
	operator.requestContext(askID, "Is the storage migration finished?")
	asked := raising.finish(t)
	require.Equal(t, 3, asked.code, "the operator asked for more context")
	require.Contains(t, asked.stdout, "Is the storage migration finished?")
	require.Contains(t, asked.stderr, "--subject-key release-check", "the next step names the key to revise")
	endID, endState, _ := endOf(t, asked.stdout)
	require.Equal(t, askID, endID)
	require.Equal(t, "pending", endState)
	require.Equal(t, 1, j.pushes.n, "the operator was pushed once")

	// 3. Answer the question with --context-response. The answer is not part of
	// the ask: nothing else about it changes, and nothing is resent.
	revised := j.do(runOne, "ask", "--subject-key", "release-check",
		"--context-response", "The storage migration finished at 10:42 and the checksums match.")
	require.Zero(t, revised.code, revised.stderr)
	_, _, outcome := endOf(t, revised.stdout)
	require.Equal(t, "updated", outcome, "the same key answers the question on the open ask, with no new push")
	require.Equal(t, 1, j.pushes.n)

	// A second run of the conversation reads it by key and sees nothing was lost,
	// and the question with its answer.
	kept := j.get(runTwo, "release-check")
	require.Equal(t, "Ship the release today?", kept["question"])
	require.Equal(t, "The release is green and the window closes at noon.", kept["big_picture"], "the answer is not written into the big picture")
	require.Nil(t, kept["technical_details"])
	require.Equal(t, []string{"yes", "no"}, optionKeys(kept), "the answers are the ones it had")
	require.ElementsMatch(t, []string{"plan.md", "explainer.html"}, attachmentNames(kept))
	require.Equal(t, []string{"layout", "plan", "why"}, questionIDs(kept), "the typed form is kept")
	require.Nil(t, kept["open_context_request_count"], "answering addressed the operator's question (a zero count is left out of the JSON)")
	require.Equal(t, [][2]string{{"Is the storage migration finished?", "The storage migration finished at 10:42 and the checksums match."}}, contextHistory(kept))

	// An agent that returns to wording it used before still revises the ask.
	for _, text := range []string{"Checksums are being re-verified.", "The storage migration finished at 10:42 and the checksums match."} {
		again := j.do(runOne, "ask", "--subject-key", "release-check", "--technical-details", text)
		require.Zero(t, again.code, again.stderr)
		_, _, outcome = endOf(t, again.stdout)
		require.Equal(t, "updated", outcome, text)
	}
	require.Equal(t, "The storage migration finished at 10:42 and the checksums match.", j.get(runTwo, "release-check")["technical_details"], "going back to earlier wording is a revision, not a replay")

	// 4. Replace one attachment and add another; the explainer stays. The plan
	// gained a section, and the review that follows it gains the section too.
	j.file("plan.md", "# Release plan\n\n## Storage\nMove to the new volume.\n\n## Rollout\nStaged, one region at a time.\n\n## Retention\nKeep 30 days.\n")
	j.file("notes.md", "Benchmark notes.\n")
	more := j.do(runOne, "ask", "--subject-key", "release-check", "--attach", plan, "--attach", notes)
	require.Zero(t, more.code, more.stderr)
	_, _, outcome = endOf(t, more.stdout)
	require.Equal(t, "updated", outcome)
	packet := j.get(runTwo, "release-check")
	require.ElementsMatch(t, []string{"plan.md", "explainer.html", "notes.md"}, attachmentNames(packet))
	require.Equal(t, []string{"layout", "plan", "why"}, questionIDs(packet))
	require.Equal(t, []string{"storage", "rollout", "retention"}, sectionKeysOf(packet, "plan"), "the review follows the plan that replaced the old one, though the revision never mentioned the form")
	require.Equal(t, 1, j.pushes.n, "revisions never push again")
	require.Equal(t, sha256Hex("# Release plan\n\n## Storage\nMove to the new volume.\n\n## Rollout\nStaged, one region at a time.\n\n## Retention\nKeep 30 days.\n"), attachmentSHA(packet, "plan.md"), "the sent file replaced the one of the same name")
	open := asJSON(t, j.do(runTwo, "list", "--open", "--json").stdout)
	require.Len(t, open["asks"], 1, "the second run lists the conversation's open asks")

	// One attachment goes by name; the explainer and the plan stay. A dry run of
	// the same revision previews the merged ask and changes nothing.
	preview = j.do(runOne, "ask", "--subject-key", "release-check", "--remove-attachment", "notes.md", "--dry-run")
	require.Zero(t, preview.code, preview.stderr)
	require.Contains(t, preview.stdout, "Would revise open ask "+askID, "the preview is of the revision merged over the open ask")
	require.Contains(t, preview.stdout, "Ship the release today?", "and carries the question it kept")
	require.ElementsMatch(t, []string{"plan.md", "explainer.html", "notes.md"}, attachmentNames(j.get(runTwo, "release-check")), "a dry run removes nothing")
	removed := j.do(runOne, "ask", "--subject-key", "release-check", "--remove-attachment", "notes.md")
	require.Zero(t, removed.code, removed.stderr)
	require.ElementsMatch(t, []string{"plan.md", "explainer.html"}, attachmentNames(j.get(runTwo, "release-check")))
	typo := j.do(runOne, "ask", "--subject-key", "release-check", "--remove-attachment", "plan.mdd")
	require.Zero(t, typo.code)
	require.Contains(t, typo.stderr, `no attachment named "plan.mdd"`, "a name the ask does not have is reported")

	// A form question added for one revision goes by its id; the others stay. The
	// operator asks a second question, the agent answers it and drops the extra
	// question in one command, and both rounds stay readable.
	extra := j.file("extra.yaml", "questions:\n  - id: owner\n    type: text\n    prompt: Who owns the rollout?\n")
	added := j.do(runOne, "ask", "--subject-key", "release-check", "--form", extra)
	require.Zero(t, added.code, added.stderr)
	require.Equal(t, []string{"layout", "plan", "why", "owner"}, questionIDs(j.get(runTwo, "release-check")))
	operator.requestContext(askID, "Which regions go first?")
	again := j.do(runTwo, "wait", "--subject-key", "release-check", "--timeout", "5s")
	require.Equal(t, 3, again.code, "a new question ends the wait again")
	require.Contains(t, again.stdout, "Which regions go first?")
	require.Contains(t, again.stderr, "--context-response", "the guidance names the flag that answers it")
	// The answer and the next wait are one command: it sends the answer, then waits
	// again, and (nobody has decided yet) ends at its own timeout, status 4.
	answeredAgain := j.do(runOne, "ask", "--subject-key", "release-check", "--context-response", "Europe first, then the US.", "--remove-question", "owner", "--and-wait", "--timeout", "1s")
	require.Equal(t, 4, answeredAgain.code, answeredAgain.stderr)
	require.Contains(t, answeredAgain.stdout, "outcome=updated", "the answer was sent and printed first")
	require.Contains(t, answeredAgain.stderr, "your wait timed out", "and the wait did not return the question it answered")
	rounds := j.get(runTwo, "release-check")
	require.Equal(t, []string{"layout", "plan", "why"}, questionIDs(rounds))
	require.Equal(t, [][2]string{
		{"Is the storage migration finished?", "The storage migration finished at 10:42 and the checksums match."},
		{"Which regions go first?", "Europe first, then the US."},
	}, contextHistory(rounds), "every round of question and answer is kept, oldest first")
	typoQuestion := j.do(runOne, "ask", "--subject-key", "release-check", "--remove-question", "ownr")
	require.Zero(t, typoQuestion.code)
	require.Contains(t, typoQuestion.stderr, `no form question with the id "ownr"`)
	nobody := j.do(runOne, "ask", "--subject-key", "release-check", "--context-response", "Nobody asked.")
	require.Zero(t, nobody.code)
	require.Contains(t, nobody.stderr, "context_response_unused", "an answer with no open question is not stored, and the command says so")
	require.Len(t, contextHistory(j.get(runTwo, "release-check")), 2)

	// The wait that follows an answer does not return the question it answered:
	// it ends at its own timeout, status 4, saying the wait ran out.
	idle := j.do(runTwo, "wait", "--subject-key", "release-check", "--timeout", "1s")
	require.Equal(t, 4, idle.code)
	require.Contains(t, idle.stderr, "your wait timed out")

	// The old flag name is gone; the error names the one that replaced it.
	legacy := j.do(runTwo, "wait", "--subject-key", "release-check", "--deadline", "1s")
	require.Equal(t, 1, legacy.code)
	require.Contains(t, legacy.stderr, "--deadline was renamed to --timeout")

	// 5. The second run waits by key; the operator answers the typed form.
	waiting := j.start(runTwo, "wait", "--subject-key", "release-check", "--json")
	operator.answer(&v1.AnswerRequest{
		AskId: askID, Answer: "yes", Note: "Go ahead.",
		Responses: []*v1.AskQuestionResponse{
			{QuestionId: "layout", Selected: []string{"staged"}},
			{QuestionId: "why", Text: "Watch the disk graph for an hour."},
		},
	})
	answered := waiting.finish(t)
	require.Zero(t, answered.code, answered.stderr)
	reply := asJSON(t, answered.stdout)
	require.Equal(t, "answered", reply["state"])
	require.Equal(t, "yes", reply["answer"])
	require.Equal(t, "Go ahead.", reply["note"])
	require.Equal(t, map[string]any{"layout": []any{"staged"}, "why": "Watch the disk graph for an hour."}, typedAnswers(reply))

	// The operator answered while the agent was still revising. The revision is
	// refused with what became of the ask, so the agent reads the answer instead of guessing.
	tooLate := j.do(runOne, "ask", "--subject-key", "release-check", "--technical-details", "One more detail.")
	require.NotZero(t, tooLate.code)
	require.Contains(t, tooLate.stderr, `was answered "yes"`)
	require.Contains(t, tooLate.stderr, "conversations get --subject-key release-check")
	require.Contains(t, tooLate.stderr, "your revision was not applied")

	// 6. A second decision, raised for JSON: the fields an agent parses.
	second := j.do(runOne, "ask", "--subject-key", "old-volume", "--yes-no", "--question", "Roll back to the old volume?",
		"--big-picture", "The new volume is healthy; the old one is still mounted.", "--json")
	require.Zero(t, second.code, second.stderr)
	created := asJSON(t, second.stdout)
	require.Equal(t, "pending", created["state"])
	require.Equal(t, "created", created["outcome"])
	require.NotEmpty(t, created["ask_id"])
	secondID, _ := created["ask_id"].(string)

	// Still open at the deadline: status 4, and the command says how to go on.
	late := j.do(runTwo, "wait", secondID, "--timeout", "1s")
	require.Equal(t, 4, late.code)
	require.Contains(t, late.stderr, "your wait timed out")
	require.Contains(t, late.stdout, "ask_id="+secondID+" state=pending")

	// The asker withdraws it; the other run's wait ends with status 5.
	closing := j.start(runTwo, "wait", "--subject-key", "old-volume")
	withdrawn := j.do(runOne, "withdraw", "--subject-key", "old-volume", "--reason", "The release decision made this moot.")
	require.Zero(t, withdrawn.code, withdrawn.stderr)
	gone := closing.finish(t)
	require.Equal(t, 5, gone.code, "closed without an answer")
	require.Contains(t, gone.stdout, "ask_id="+secondID+" state=withdrawn")
	require.Contains(t, gone.stderr, "The release decision made this moot.", "the reason reaches the waiting run")

	// 7. An ask with no key is withdrawn by its id, given as the first argument.
	plain := asJSON(t, j.do(runOne, "ask", "--question", "Use the blue theme?", "--option", "blue", "--option", "green", "--json").stdout)
	plainID, _ := plain["ask_id"].(string)
	require.NotEmpty(t, plainID)
	require.Zero(t, j.do(runOne, "withdraw", plainID, "--reason", "Decided without you.").code)
	require.Equal(t, "withdrawn", j.getByID(runTwo, plainID)["state"])
}

// get reads an ask by key as a run, and returns the ask object of the JSON.
func (j *journey) get(token, subjectKey string) map[string]any {
	j.t.Helper()
	read := j.do(token, "get", "--subject-key", subjectKey, "--json")
	require.Zero(j.t, read.code, read.stderr)
	ask, ok := asJSON(j.t, read.stdout)["ask"].(map[string]any)
	require.True(j.t, ok, "get --json holds the ask under \"ask\":\n%s", read.stdout)
	return ask
}

// getByID reads an ask by id as a run.
func (j *journey) getByID(token, askID string) map[string]any {
	j.t.Helper()
	read := j.do(token, "get", askID, "--json")
	require.Zero(j.t, read.code, read.stderr)
	return asJSON(j.t, read.stdout)["ask"].(map[string]any)
}

func sha256Hex(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func attachmentSHA(ask map[string]any, name string) string {
	for _, attachment := range ask["attachments"].([]any) {
		if entry := attachment.(map[string]any); entry["name"] == name {
			return entry["sha256"].(string)
		}
	}
	return ""
}

// contextHistory is the ask's context questions with their answers, oldest first.
func contextHistory(ask map[string]any) [][2]string {
	var out [][2]string
	requests, _ := ask["context_requests"].([]any)
	for _, raw := range requests {
		request, _ := raw.(map[string]any)
		question, _ := request["question"].(string)
		response, _ := request["response"].(string)
		out = append(out, [2]string{question, response})
	}
	return out
}

func optionKeys(ask map[string]any) []string {
	var keys []string
	for _, option := range ask["options"].([]any) {
		if key := option.(map[string]any)["key"].(string); key != "other" {
			keys = append(keys, key)
		}
	}
	return keys
}

func attachmentNames(ask map[string]any) []string {
	var names []string
	for _, attachment := range ask["attachments"].([]any) {
		names = append(names, attachment.(map[string]any)["name"].(string))
	}
	return names
}

// sectionKeysOf is the keys of the sections of one review question.
func sectionKeysOf(ask map[string]any, questionID string) []string {
	var keys []string
	for _, question := range ask["questions"].([]any) {
		entry := question.(map[string]any)
		if entry["id"] != questionID {
			continue
		}
		for _, section := range entry["sections"].([]any) {
			keys = append(keys, section.(map[string]any)["key"].(string))
		}
	}
	return keys
}

func questionIDs(ask map[string]any) []string {
	var ids []string
	for _, question := range ask["questions"].([]any) {
		ids = append(ids, question.(map[string]any)["id"].(string))
	}
	return ids
}

// typedAnswers is the operator's answers to the form, by question id: the chosen
// keys, or the text.
func typedAnswers(reply map[string]any) map[string]any {
	answers := map[string]any{}
	for _, response := range reply["responses"].([]any) {
		entry := response.(map[string]any)
		if text, ok := entry["text"].(string); ok && text != "" {
			answers[entry["question_id"].(string)] = text
			continue
		}
		answers[entry["question_id"].(string)] = entry["selected"]
	}
	return answers
}
