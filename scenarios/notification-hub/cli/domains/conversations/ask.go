package conversations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// buildPacket reads the decision packet named by --explainer, --attach and
// --form. The explainer goes first so it survives the Hub's attachment cap.
//
// It never fails. Every part is optional, so a part that cannot be used (a
// missing or oversize file, a form mistake) is left out and named in
// packet.Warnings, and the ask is still sent: a broken packet part must never
// keep a decision from the operator.
func buildPacket(attach []string, explainer, formPath string) (*packet, []*v1.AskQuestion) {
	pkt := newPacket()
	leaveOut := func(err error) { pkt.warnf("Left out %v. The ask is sent without it.", err) }
	if explainer != "" {
		if err := pkt.addAttachment(attachSpec{Path: explainer}, v1.AskAttachment_ROLE_EXPLAINER); err != nil {
			leaveOut(err)
		}
	}
	for _, value := range attach {
		spec, err := parseAttachSpec(value)
		if err == nil {
			err = pkt.addAttachment(spec, v1.AskAttachment_ROLE_EVIDENCE)
		}
		if err != nil {
			leaveOut(err)
		}
	}
	var questions []*v1.AskQuestion
	if formPath != "" {
		questions = readForm(formPath, pkt)
	}
	return pkt, questions
}

// askRequest is an ask ready to send: the request, and whether the CLI derived
// its idempotency key.
type askRequest struct {
	req     *v1.AskRequest
	derived bool
	packet  *packet
}

// buildAsk turns the command's flags into a request. It reads the body file and
// the packet, and derives the idempotency key when none was given. The packet
// is returned beside the request, because a dry run has to send what the check
// reads of it while a real send uploads the blobs the Hub lacks.
func (s *service) buildAsk(ctx cliapp.RunContext) (askRequest, error) {
	var body *askText
	if path := ctx.Flag("body-file"); path != "" {
		parsed, err := s.readBody(path)
		if err != nil {
			return askRequest{}, err
		}
		body = &parsed
	}
	text, err := coverText(ctx.Flag("question"), ctx.Flag("big-picture"), ctx.Flag("technical-details"), body)
	if err != nil {
		return askRequest{}, err
	}
	// A keyed ask may be a revision, which says only what changes: the Hub keeps
	// the question and answers of the open ask for whatever is left out, and
	// refuses a first send that has none. Without a key there is nothing to revise.
	keyed := ctx.Flag("subject-key") != ""
	if text.Question == "" && !keyed {
		return askRequest{}, fmt.Errorf("an ask needs a question: --question, or the first paragraph of --body-file")
	}
	for _, section := range text.Ignored {
		fmt.Fprintf(ctx.Stderr(), "Warning: the body file's section %q has no place in an ask and was left out; only the first paragraph, ## Background and ## Details are read.\n", section)
	}
	var askOptions []*v1.AskOption
	if len(ctx.FlagValues("option")) > 0 || ctx.BoolFlag("yes-no") || !keyed {
		options, err := resolveOptions(ctx.FlagValues("option"), ctx.BoolFlag("yes-no"))
		if err != nil {
			return askRequest{}, err
		}
		for _, option := range options {
			askOptions = append(askOptions, &v1.AskOption{Key: option.Key, Label: option.Label})
		}
	}
	expires := ctx.Flag("expires")
	var deadline string
	if expires != "" {
		if deadline, err = parseTime("--expires", expires, s.now()); err != nil {
			return askRequest{}, err
		}
	}
	reassertWithin, err := parseReassertWithin(ctx.Flag("reassert-within"), ctx.Flag("subject-key"))
	if err != nil {
		return askRequest{}, err
	}
	sender, err := parseFrom(ctx)
	if err != nil {
		return askRequest{}, err
	}
	pkt, questions := buildPacket(ctx.FlagValues("attach"), ctx.Flag("explainer"), ctx.Flag("form"))
	for _, warning := range pkt.Warnings {
		fmt.Fprintf(ctx.Stderr(), "Warning: %s\n", warning)
	}
	host, _ := os.Hostname()
	attachments := make([]*v1.AskAttachment, 0, len(pkt.attachments))
	for _, att := range pkt.attachments {
		attachments = append(attachments, att.proto(host))
	}
	// Every optional field is sent only when the command gave it, so the Hub can
	// tell "keep what the open ask says" from "set it to this"; giving it an empty
	// value clears it.
	req := &v1.AskRequest{
		Question:              text.Question,
		Options:               askOptions,
		Recommended:           given(ctx, "recommend"),
		RecommendationReason:  given(ctx, "recommend-reason"),
		DefaultAnswer:         given(ctx, "default"),
		Urgency:               given(ctx, "urgency"),
		ContextUrl:            given(ctx, "context-url"),
		Sender:                sender,
		BigPicture:            givenText(ctx, "big-picture", text.BigPicture),
		TechnicalDetails:      givenText(ctx, "technical-details", text.TechnicalDetails),
		Deadline:              deadline,
		SensitivityLabel:      ctx.Flag("sensitivity-label"),
		IdempotencyKey:        ctx.Flag("idempotency-key"),
		SubjectKey:            ctx.Flag("subject-key"),
		ReassertWithinSeconds: givenSeconds(ctx, reassertWithin),
		Attachments:           attachments,
		Questions:             questions,
		ClearAttachments:      ctx.BoolFlag("clear-attachments"),
		ClearQuestions:        ctx.BoolFlag("clear-questions"),
		RemoveAttachments:     ctx.FlagValues("remove-attachment"),
		RemoveQuestions:       ctx.FlagValues("remove-question"),
		ContextResponse:       strings.TrimSpace(ctx.Flag("context-response")),
	}
	if ctx.BoolFlag("reversible") {
		req.Reversible = proto.Bool(true)
	}
	if keyed {
		// A keyed ask is answered by revising it, which is what the operator's
		// "I need more context" question needs; --no-context-requests opts out.
		req.AcceptsContextRequests = proto.Bool(!ctx.BoolFlag("no-context-requests"))
	}
	derived := req.IdempotencyKey == ""
	if derived {
		req.IdempotencyKey = deriveIdempotencyKey(req, expires)
	}
	// The session-label default goes in its own field, default_sender, which the
	// Hub uses only to name an ask it creates and never to rename an open one. It
	// is added after the key is derived and takes no part in it: whether the
	// lookup succeeds is not something the agent said, so a retry that resolved it
	// once and failed it once is still the same request.
	if req.Sender == nil {
		if label := s.defaultSender(); label != "" {
			req.DefaultSender = &label
		}
	}
	return askRequest{req: req, derived: derived, packet: pkt}, nil
}

// given is a flag's value when the command gave the flag (even empty, which
// clears the field), and nil otherwise. A flag's manifest default is not a value
// the command gave.
func given(ctx cliapp.RunContext, flag string) *string {
	if !ctx.BoolFlag(flag) {
		return nil
	}
	value := ctx.Flag(flag)
	return &value
}

// givenText is a cover text part: given by its flag or by the body file.
func givenText(ctx cliapp.RunContext, flag, value string) *string {
	if !ctx.BoolFlag(flag) && value == "" {
		return nil
	}
	return &value
}

// maxFromRunes is the longest --from the Hub keeps.
const maxFromRunes = 60

// parseFrom is --from, the name the operator reads as the ask's sender, when the
// command gave it. The Hub keeps 1 to 60 characters of plain text; the CLI says
// so before the send rather than after, and an empty name is an error because
// leaving the flag out is how an ask keeps its sender or gets the Hub's own.
func parseFrom(ctx cliapp.RunContext) (*string, error) {
	if !ctx.BoolFlag("from") {
		return nil, nil
	}
	name := strings.Join(strings.Fields(ctx.Flag("from")), " ")
	if name == "" {
		return nil, fmt.Errorf("--from needs a name of 1 to %d characters, for example --from \"Research Desk\"; leave it out to keep the sender the ask has", maxFromRunes)
	}
	if utf8.RuneCountInString(name) > maxFromRunes {
		return nil, fmt.Errorf("--from is %d characters; the operator reads it as a short name, so use at most %d", utf8.RuneCountInString(name), maxFromRunes)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return nil, fmt.Errorf("--from must be plain text without control characters")
		}
	}
	return &name, nil
}

// givenSeconds is --reassert-within when the command gave it.
func givenSeconds(ctx cliapp.RunContext, seconds int32) *int32 {
	if !ctx.BoolFlag("reassert-within") {
		return nil
	}
	return &seconds
}

func (s *service) ask(ctx cliapp.RunContext) error {
	if ctx.BoolFlag("dry-run") && ctx.BoolFlag("and-wait") {
		return fmt.Errorf("--dry-run creates no ask, so there is nothing to wait for; drop --and-wait (or drop --dry-run to send the ask and wait)")
	}
	built, err := s.buildAsk(ctx)
	if err != nil {
		return err
	}
	req := built.req
	if ctx.BoolFlag("dry-run") {
		return s.dryRun(ctx, built)
	}
	// A blob that fails to upload must not stop the decision from reaching the
	// operator: the Hub leaves out an attachment it does not hold and says so in
	// the packet check.
	if err := s.uploadPacket(built.packet, ctx); err != nil {
		fmt.Fprintf(ctx.Stderr(), "Warning: %v\n  The ask is sent anyway; the Hub leaves out attachments it does not hold.\n", err)
	}
	resp, callErr := s.client.Ask(context.Background(), connect.NewRequest(req))
	if callErr != nil {
		return askError(callErr, req)
	}
	if err := renderAsk(ctx, resp.Msg, built); err != nil {
		return err
	}
	writeKept(ctx.Stderr(), req, resp.Msg.GetOutcome())
	writePacketWarnings(ctx.Stderr(), resp.Msg.GetWarnings(), true)
	writeGuideReminder(ctx.Stderr(), checkGuide(context.Background()), askGuideContext{AskID: resp.Msg.GetAskId(), SubjectKey: req.GetSubjectKey(), Outcome: resp.Msg.GetOutcome()})
	if !ctx.BoolFlag("and-wait") {
		return nil
	}
	wait, err := parseTime("--timeout", ctx.Flag("timeout"), s.now())
	if err != nil {
		return err
	}
	// A keyed ask is waited on by its key, which keeps working if the ask is replaced.
	target := waitTarget{AskID: resp.Msg.GetAskId()}
	if req.GetSubjectKey() != "" {
		target = waitTarget{SubjectKey: req.GetSubjectKey()}
	}
	return s.awaitAndReport(ctx, target, wait, false)
}

// uploadPacket stores every file of the packet the Hub does not hold yet.
func (s *service) uploadPacket(pkt *packet, ctx cliapp.RunContext) error {
	var files []localFile
	for _, att := range pkt.attachments {
		files = append(files, att.files()...)
	}
	return uploader{client: s.uploadClient, progress: ctx.Stderr()}.upload(context.Background(), files)
}

func (s *service) requestContext(ctx cliapp.RunContext) error {
	resp, callErr := s.client.RequestContext(context.Background(), connect.NewRequest(&v1.RequestContextRequest{
		AskId:    ctx.Flag("ask-id"),
		Question: ctx.Flag("question"),
	}))
	if callErr != nil {
		return cliapp.WrapAPIError("request context for notification question", callErr, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Recorded your question on ask %s; the ask stays open until the requester revises it.", resp.Msg.GetAskId())},
	})
}

func (s *service) answer(ctx cliapp.RunContext) error {
	var answers answerFile
	if path := ctx.Flag("responses-file"); path != "" {
		var err error
		if answers, err = readResponsesFile(path); err != nil {
			return err
		}
	}
	resp, callErr := s.client.Answer(context.Background(), connect.NewRequest(&v1.AnswerRequest{
		AskId:           ctx.Flag("ask-id"),
		Answer:          ctx.Flag("answer"),
		Note:            ctx.Flag("note"),
		Responses:       answers.Responses,
		AttachmentNotes: answers.AttachmentNotes,
	}))
	if callErr != nil {
		return cliapp.WrapAPIError("answer notification question", callErr, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Answered ask %s.", resp.Msg.GetAskId())}})
}

// answerFile is what --responses-file holds: form answers and notes pinned to
// attachments.
type answerFile struct {
	Responses       []*v1.AskQuestionResponse
	AttachmentNotes []*v1.AskAttachmentNote
}

// readResponsesFile parses --responses-file. It is a JSON list of
// AskQuestionResponse objects, or an object {"responses": [...],
// "attachment_notes": [{"attachment", "path", "note"}]}; both use the proto's
// JSON shape (question_id, selected, text, sections).
func readResponsesFile(path string) (answerFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return answerFile{}, fmt.Errorf("--responses-file %q: %w", path, unwrapPathError(err))
	}
	var rawResponses, rawNotes []json.RawMessage
	var object struct {
		Responses       []json.RawMessage `json:"responses"`
		AttachmentNotes []json.RawMessage `json:"attachment_notes"`
	}
	if err := json.Unmarshal(data, &rawResponses); err != nil {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if objErr := dec.Decode(&object); objErr != nil {
			return answerFile{}, fmt.Errorf("--responses-file %s must be a JSON list of responses or an object with responses and attachment_notes: %w", path, objErr)
		}
		rawResponses, rawNotes = object.Responses, object.AttachmentNotes
	}
	var out answerFile
	for i, item := range rawResponses {
		resp := &v1.AskQuestionResponse{}
		if err := protojson.Unmarshal(item, resp); err != nil {
			return answerFile{}, fmt.Errorf("--responses-file %s: response %d: %w", path, i+1, err)
		}
		if resp.GetQuestionId() == "" {
			return answerFile{}, fmt.Errorf("--responses-file %s: response %d needs a question_id", path, i+1)
		}
		out.Responses = append(out.Responses, resp)
	}
	for i, item := range rawNotes {
		note := &v1.AskAttachmentNote{}
		if err := protojson.Unmarshal(item, note); err != nil {
			return answerFile{}, fmt.Errorf("--responses-file %s: attachment note %d: %w", path, i+1, err)
		}
		if note.GetAttachment() == "" || strings.TrimSpace(note.GetNote()) == "" {
			return answerFile{}, fmt.Errorf("--responses-file %s: attachment note %d needs an attachment and a note", path, i+1)
		}
		out.AttachmentNotes = append(out.AttachmentNotes, note)
	}
	return out, nil
}

// dryRun is `ask --dry-run`: the packet check and the notification the operator
// would get, and nothing sent or stored. Nothing is uploaded either: the check
// is given each attachment's SHA-256 and size, and the content of the two kinds
// of file it reads (the explainer page and the Markdown a review follows), which
// the Hub reads from the request and keeps nowhere.
func (s *service) dryRun(ctx cliapp.RunContext, built askRequest) error {
	resp, callErr := s.client.CheckAsk(context.Background(), connect.NewRequest(&v1.CheckAskRequest{
		Ask: built.req, FileContents: checkedContents(built),
	}))
	if callErr != nil {
		return askError(callErr, built.req)
	}
	msg := resp.Msg
	if ctx.JSON() {
		return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{})
	}
	var action string
	if id := msg.GetRevisesAskId(); id != "" {
		action = fmt.Sprintf("Would revise open ask %s in place; no new notification would be sent.", id)
	} else {
		action = "Would create a new ask and notify the operator."
	}
	body := strings.ReplaceAll(strings.TrimSpace(msg.GetPushBody()), "\n", "\n    ")
	summary := []string{"Dry run: nothing was stored and no ask was created.", action}
	if line := senderLine(msg); line != "" {
		summary = append(summary, line)
	}
	if err := ctx.RenderList(cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Notification the operator would get",
		Results:        []string{"Title: " + msg.GetPushTitle(), "Body:\n    " + body},
		RetrievalHints: []string{"To send it, run the same command without --dry-run."},
	}); err != nil {
		return err
	}
	writePacketWarnings(ctx.Stderr(), msg.GetWarnings(), false)
	return nil
}

// senderSources say, for each sender_source the Hub reports, where the name the
// operator will read came from.
var senderSources = map[string]string{
	"declared":  "your --from",
	"default":   "your Web Console session label",
	"kept":      "kept from the open ask",
	"service":   "named by the Hub from the service that is asking",
	"agent_run": "named by the Hub: no --from, and no session label was found",
	"unnamed":   "named by the Hub: it could not tell who is asking",
}

// senderLine is the dry run's answer to "who will the operator see?", from the
// Hub's own reckoning so it matches what a real send stores. It is empty when
// the Hub did not say.
func senderLine(msg *v1.CheckAskResponse) string {
	if msg.GetSender() == "" {
		return ""
	}
	line := "Sender the operator would see: " + msg.GetSender()
	if source, ok := senderSources[msg.GetSenderSource()]; ok {
		line += " (" + source + ")"
	}
	return line
}

// checkedContents is what a dry run lets the Hub read: the explainer page and
// every file a review question follows, by SHA-256. Both are small text files;
// the rest of the packet is checked from its hashes and sizes alone.
func checkedContents(built askRequest) map[string][]byte {
	contents := map[string][]byte{}
	read := func(file *localFile) {
		if file == nil || file.Size > maxCheckedFileBytes {
			return
		}
		if data, err := os.ReadFile(file.Path); err == nil {
			contents[file.SHA256] = data
		}
	}
	for _, att := range built.packet.attachments {
		if att.Role == v1.AskAttachment_ROLE_EXPLAINER {
			read(att.File)
		}
	}
	for _, question := range built.req.GetQuestions() {
		if ref := question.GetSectionsFrom(); ref != "" {
			if file, ok := built.packet.sourceFile(ref); ok {
				read(file)
			}
		}
	}
	return contents
}

// maxCheckedFileBytes is the most a dry run sends of one file: the Hub reads an
// explainer up to 2 MiB and a followed Markdown file up to 1 MiB.
const maxCheckedFileBytes = 2 << 20
