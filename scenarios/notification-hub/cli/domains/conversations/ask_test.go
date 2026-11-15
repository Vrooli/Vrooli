package conversations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

func TestAskWithoutPacketPartsSendsNoAttachmentsOrQuestions(t *testing.T) {
	stub := &hubStub{}
	svc := newTestService(t, stub)
	stdout, _, err := runCommand(t, svc, "ask", askArgs()...)
	require.NoError(t, err)
	require.Contains(t, stdout, "Created ask ask-1.")
	require.Empty(t, stub.statCalls, "no attachments means no blob traffic")
	require.Len(t, stub.asks, 1)
	require.Empty(t, stub.asks[0].GetAttachments())
	require.Empty(t, stub.asks[0].GetQuestions())
}

func TestAskUploadsOnlyBlobsTheHubLacksAndReferencesThemByHash(t *testing.T) {
	dir := t.TempDir()
	plan := writeFile(t, dir, "plan.md", "# Plan\n\n## Storage\n\n## Retention\n")
	shot := writeFile(t, dir, "shot.png", "not really a png")
	known, err := describeFile(shot, maxAttachmentTotalBytes)
	require.NoError(t, err)
	stub := &hubStub{held: map[string]bool{known.SHA256: true}}
	svc := newTestService(t, stub)

	_, stderr, err := runCommand(t, svc, "ask", askArgs("--attach", plan+"=The plan", "--attach", shot)...)
	require.NoError(t, err)

	require.Len(t, stub.statCalls, 1)
	require.Len(t, stub.statCalls[0], 2)
	require.Len(t, stub.puts, 1, "the screenshot is already stored; only the plan uploads")
	require.Equal(t, "text/markdown", stub.puts[0].GetMediaType())
	require.Equal(t, "# Plan\n\n## Storage\n\n## Retention\n", string(stub.puts[0].GetContent()))
	require.Contains(t, stderr, "Uploading 1 attachment file(s)")

	host, _ := os.Hostname()
	atts := stub.asks[0].GetAttachments()
	require.Len(t, atts, 2)
	require.Equal(t, "plan.md", atts[0].GetName())
	require.Equal(t, "The plan", atts[0].GetLabel())
	require.Equal(t, v1.AskAttachment_KIND_FILE, atts[0].GetKind())
	require.Equal(t, v1.AskAttachment_ROLE_EVIDENCE, atts[0].GetRole())
	require.Equal(t, plan, atts[0].GetSourcePath())
	require.Equal(t, host, atts[0].GetSourceHost())
	require.Equal(t, stub.puts[0].GetSha256(), atts[0].GetSha256())
	require.Equal(t, known.SHA256, atts[1].GetSha256())
	require.Equal(t, "image/png", atts[1].GetMediaType())
}

func TestAskSendsAFolderAsOneAttachmentWithEntries(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "shots/a.png", "a")
	writeFile(t, dir, "shots/sub/b.png", "b")
	stub := &hubStub{}
	svc := newTestService(t, stub)

	_, _, err := runCommand(t, svc, "ask", askArgs("--attach", filepath.Join(dir, "shots"))...)
	require.NoError(t, err)
	atts := stub.asks[0].GetAttachments()
	require.Len(t, atts, 1)
	require.Equal(t, v1.AskAttachment_KIND_FOLDER, atts[0].GetKind())
	require.Equal(t, "shots", atts[0].GetName())
	var paths []string
	for _, entry := range atts[0].GetEntries() {
		paths = append(paths, entry.GetPath())
	}
	require.Equal(t, []string{"a.png", "sub/b.png"}, paths)
	require.Len(t, stub.puts, 2)
}

func TestAskPutsTheExplainerFirstAndFormQuestionsOnTheRequest(t *testing.T) {
	dir := t.TempDir()
	explainer := writeFile(t, dir, "why.html", "<html><meta name=viewport content=x></html>")
	plan := writeFile(t, dir, "plan.md", "## Storage\n### Retention\n")
	form := writeFile(t, dir, "form.yaml", `
questions:
  - id: layout
    type: single_choice
    prompt: Layout?
    choices: [{key: cards}, {key: table}]
    recommended: cards
  - id: plan
    type: section_review
    prompt: Review the plan.
    sections_from: plan.md
`)
	stub := &hubStub{}
	svc := newTestService(t, stub)

	_, _, err := runCommand(t, svc, "ask", askArgs("--attach", plan, "--explainer", explainer, "--form", form)...)
	require.NoError(t, err)
	atts := stub.asks[0].GetAttachments()
	require.Equal(t, []string{"why.html", "plan.md"}, []string{atts[0].GetName(), atts[1].GetName()})
	require.Equal(t, v1.AskAttachment_ROLE_EXPLAINER, atts[0].GetRole())
	require.Equal(t, "text/html", atts[0].GetMediaType())

	questions := stub.asks[0].GetQuestions()
	require.Len(t, questions, 2)
	require.Equal(t, []string{"cards"}, questions[0].GetRecommended().GetSelected())
	require.Equal(t, "layout", questions[0].GetRecommended().GetQuestionId())
	require.Equal(t, "plan.md", questions[1].GetSectionsFrom(), "the Hub reads the headings, and again when a revision replaces the plan")
	require.Empty(t, questions[1].GetSections())
}

func TestAskStillSendsWhenPacketPartsAreBrokenAndSaysWhatWasLeftOut(t *testing.T) {
	dir := t.TempDir()
	good := writeFile(t, dir, "good.txt", "good")
	huge := sparseFile(t, dir, "huge.bin", maxAttachmentFileBytes+1)
	form := writeFile(t, dir, "form.yaml", "- {id: ok, type: text, prompt: Fine}\n- {id: bad, type: single_choice, prompt: P}\n")
	stub := &hubStub{}
	svc := newTestService(t, stub)

	stdout, stderr, err := runCommand(t, svc, "ask", askArgs(
		"--attach", good, "--attach", filepath.Join(dir, "nope.txt"), "--attach", huge,
		"--explainer", dir, "--form", form)...)
	require.NoError(t, err)
	require.Contains(t, stdout, "Created ask ask-1.")
	require.Len(t, stub.asks, 1, "a broken packet part must not stop the decision")

	atts := stub.asks[0].GetAttachments()
	require.Len(t, atts, 1)
	require.Equal(t, "good.txt", atts[0].GetName())
	require.Len(t, stub.asks[0].GetQuestions(), 1)
	require.Equal(t, "ok", stub.asks[0].GetQuestions()[0].GetId())
	require.Len(t, stub.puts, 1, "only the good file travels; the oversize file is never uploaded")

	for _, want := range []string{
		`Left out --attach "` + filepath.Join(dir, "nope.txt") + `"`,
		`Left out --attach "` + huge + `"`, "per-file limit",
		`Left out --explainer "` + dir + `" is a folder`,
		`Left out form question 2 ("bad")`, "needs choices",
		"The ask is sent without it.",
	} {
		require.Contains(t, stderr, want)
	}
}

func TestAskStillRequiresAValidCover(t *testing.T) {
	stub := &hubStub{}
	svc := newTestService(t, stub)
	_, _, err := runCommand(t, svc, "ask", "--question", "Q?", "--option", "only", "--expires", "24h", "--sensitivity-label", "private", "--idempotency-key", "k")
	require.Error(t, err)
	require.Empty(t, stub.asks)
}

func TestAskStillReachesTheOperatorWhenAnUploadFails(t *testing.T) {
	dir := t.TempDir()
	file := writeFile(t, dir, "a.txt", "a")
	stub := &hubStub{putErr: connectError("boom")}
	svc := newTestService(t, stub)

	stdout, stderr, err := runCommand(t, svc, "ask", askArgs("--attach", file)...)
	require.NoError(t, err)
	require.Contains(t, stdout, "Created ask ask-1.")
	require.Contains(t, stderr, "The ask is sent anyway")
	require.Len(t, stub.asks, 1)
}

func TestAskPrintsPacketWarningsToStderrAndKeepsJSONStdoutClean(t *testing.T) {
	stub := &hubStub{askResp: &v1.AskResponse{AskId: "ask-9", Warnings: []*v1.PacketWarning{
		{Code: "long_question", Field: "question", Message: "Shorten it.", Severity: "advice"},
		{Code: "attachment_blob_missing", Field: "attachments[0]", Message: "Left out.", Severity: "dropped"},
	}}}
	svc := newTestService(t, stub)

	stdout, stderr, err := runCommand(t, svc, "ask", askArgs("--json")...)
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &parsed), "stdout must be pure JSON: %q", stdout)
	require.Equal(t, "ask-9", parsed["ask_id"])
	require.Contains(t, stderr, "Packet check: 1 left out, 1 advice")
	require.Less(t, strings.Index(stderr, "attachments[0]"), strings.Index(stderr, "question: Shorten it."), "dropped parts come before advice")
}

func TestAskWarnsLocallyWhenAFolderIsTruncated(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < maxFolderFiles+3; i++ {
		writeFile(t, dir, "big/"+strings.Repeat("0", 4-len(itoa(i)))+itoa(i)+".txt", "x")
	}
	stub := &hubStub{}
	svc := newTestService(t, stub)
	_, stderr, err := runCommand(t, svc, "ask", askArgs("--attach", filepath.Join(dir, "big"))...)
	require.NoError(t, err)
	require.Contains(t, stderr, "only the first 500")
	att := stub.asks[0].GetAttachments()[0]
	require.Len(t, att.GetEntries(), maxFolderFiles)
	require.True(t, att.GetTruncated())
}

func TestAnswerSendsFormResponsesFromAJSONFile(t *testing.T) {
	dir := t.TempDir()
	file := writeFile(t, dir, "responses.json", `[
	  {"question_id": "layout", "selected": ["cards"]},
	  {"question_id": "plan", "sections": [{"section_key": "storage", "verdict": "change", "note": "shorter"}]}
	]`)
	stub := &hubStub{}
	svc := newTestService(t, stub)
	_, _, err := runCommand(t, svc, "answer", "--ask-id", "a1", "--answer", "approve", "--responses-file", file)
	require.NoError(t, err)
	require.Len(t, stub.answers[0].GetResponses(), 2)
	require.Equal(t, "change", stub.answers[0].GetResponses()[1].GetSections()[0].GetVerdict())
}

func TestAnswerRejectsAMalformedResponsesFile(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"not a list":      `{"question_id": "a"}`,
		"unknown field":   `[{"question_id": "a", "pick": "x"}]`,
		"no question id":  `[{"selected": ["x"]}]`,
		"not json at all": `nope`,
	} {
		stub := &hubStub{}
		svc := newTestService(t, stub)
		_, _, err := runCommand(t, svc, "answer", "--ask-id", "a1", "--answer", "approve", "--responses-file", writeFile(t, dir, name+".json", content))
		require.Error(t, err, name)
		require.Empty(t, stub.answers, name)
	}
}

func TestRequestContextRecordsTheOperatorsQuestion(t *testing.T) {
	stub := &hubStub{}
	svc := newTestService(t, stub)
	stdout, _, err := runCommand(t, svc, "request-context", "--ask-id", "a1", "--question", "Which region?")
	require.NoError(t, err)
	require.Equal(t, "a1", stub.contexts[0].GetAskId())
	require.Equal(t, "Which region?", stub.contexts[0].GetQuestion())
	require.Contains(t, stdout, "Recorded your question on ask a1")
}

func TestAnswerSendsAttachmentNotesFromAnObjectFile(t *testing.T) {
	file := writeFile(t, t.TempDir(), "answers.json", `{
	  "responses": [{"question_id": "layout", "selected": ["cards"]}],
	  "attachment_notes": [
	    {"attachment": "character.glb", "note": "Hair is too glossy."},
	    {"attachment": "shots", "path": "a.png", "note": "Crop tighter."}
	  ]
	}`)
	stub := &hubStub{}
	svc := newTestService(t, stub)
	_, _, err := runCommand(t, svc, "answer", "--ask-id", "a1", "--answer", "approve", "--responses-file", file)
	require.NoError(t, err)
	require.Len(t, stub.answers[0].GetResponses(), 1)
	notes := stub.answers[0].GetAttachmentNotes()
	require.Len(t, notes, 2)
	require.Equal(t, "Hair is too glossy.", notes[0].GetNote())
	require.Equal(t, "a.png", notes[1].GetPath())
}

func TestAnswerRejectsAMalformedAttachmentNote(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"no attachment": `{"attachment_notes": [{"note": "x"}]}`,
		"no note":       `{"attachment_notes": [{"attachment": "a"}]}`,
		"unknown field": `{"attachment_notes": [{"attachment": "a", "note": "x", "file": "b"}]}`,
		"unknown key":   `{"notes": []}`,
	} {
		stub := &hubStub{}
		svc := newTestService(t, stub)
		_, _, err := runCommand(t, svc, "answer", "--ask-id", "a1", "--answer", "approve", "--responses-file", writeFile(t, dir, name+".json", content))
		require.Error(t, err, name)
		require.Empty(t, stub.answers, name)
	}
}

func TestAskWithoutAnIdentityUploadsAndAttachesItsPacket(t *testing.T) {
	file := writeFile(t, t.TempDir(), "a.txt", "a")
	stub := &hubStub{
		statErr: connect.NewError(connect.CodeUnauthenticated, nil),
		askResp: &v1.AskResponse{AskId: "ask-3"},
	}
	svc := newTestService(t, stub)
	stdout, stderr, err := runCommand(t, svc, "ask", askArgs("--attach", file)...)
	require.NoError(t, err, "the decision still reaches the operator")
	require.Contains(t, stdout, "Created ask ask-3.")
	require.Contains(t, stderr, "Uploading 1 attachment file(s), 1 B...")
	require.Len(t, stub.puts, 1, "the anonymous CLI blindly uploads attachment bytes")
	require.Len(t, stub.asks, 1)
	require.Len(t, stub.asks[0].GetAttachments(), 1)
}

func TestAskDeclaresItAnswersContextRequestsOnlyWhenKeyedAndNotOptedOut(t *testing.T) {
	for name, tc := range map[string]struct {
		args []string
		want bool
	}{
		"keyed":               {[]string{"--subject-key", "disk-pressure"}, true},
		"keyed and opted out": {[]string{"--subject-key", "disk-pressure", "--no-context-requests"}, false},
		"unkeyed":             {nil, false},
		"unkeyed, opted out":  {[]string{"--no-context-requests"}, false},
	} {
		t.Run(name, func(t *testing.T) {
			stub := &hubStub{}
			_, _, err := runCommand(t, newTestService(t, stub), "ask", askArgs(tc.args...)...)
			require.NoError(t, err)
			require.Len(t, stub.asks, 1)
			require.Equal(t, tc.want, stub.asks[0].GetAcceptsContextRequests())
		})
	}
}
