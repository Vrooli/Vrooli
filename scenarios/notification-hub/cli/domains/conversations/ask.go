package conversations

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// buildPacket reads the decision packet named by --explainer, --attach,
// --packet (brief, its resolved brief file) and --form. The explainer goes
// first so it survives the Hub's attachment cap; the brief follows it.
//
// It never fails. Every part is optional, so a part that cannot be used (a
// missing or oversize file, a form mistake) is left out and named in
// packet.Warnings, and the ask is still sent: a broken packet part must never
// keep a decision from the operator.
func buildPacket(attach []string, explainer, brief, formPath string) (*packet, []*v1.AskQuestion) {
	pkt := newPacket()
	leaveOut := func(err error) { pkt.warnf("Left out %v. The ask is sent without it.", err) }
	if explainer != "" {
		if err := pkt.addAttachment(attachSpec{Path: explainer}, v1.AskAttachment_ROLE_EXPLAINER); err != nil {
			leaveOut(err)
		}
	}
	if brief != "" {
		if err := pkt.addAttachment(attachSpec{Path: brief}, v1.AskAttachment_ROLE_EVIDENCE); err != nil {
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
	featureCardPath := ctx.Flag("feature-card")
	var featureCard *featureDecisionCardInput
	if featureCardPath != "" {
		var err error
		featureCard, err = readFeatureDecisionCard(featureCardPath)
		if err != nil {
			return askRequest{}, err
		}
		for _, flag := range []string{"question", "body-file", "big-picture", "technical-details", "option", "yes-no", "recommend", "recommend-reason", "subject-key", "attach", "explainer", "form"} {
			if ctx.Flag(flag) != "" || len(ctx.FlagValues(flag)) > 0 || ctx.BoolFlag(flag) {
				return askRequest{}, fmt.Errorf("--feature-card cannot be combined with --%s; edit the card packet itself", flag)
			}
		}
	}
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
	if featureCard != nil {
		text.Question = "Keep or drop “" + featureCard.Title + "”?"
		text.BigPicture = featureCard.Summary
		text.TechnicalDetails = fmt.Sprintf("Finding %s. Before %s; after %s. Screen %s. Component %s. Source %s.", featureCard.Finding.Version, featureCard.Finding.BeforeBuild, featureCard.Finding.AfterBuild, featureCard.Location.Screen, featureCard.Finding.Component, featureCard.Finding.Source)
	}
	// A keyed ask may be a revision, which says only what changes: the Hub keeps
	// the question and answers of the open ask for whatever is left out, and
	// refuses a first send that has none. Without a key there is nothing to revise.
	subjectKey := ctx.Flag("subject-key")
	if featureCard != nil {
		subjectKey = featureCard.SubjectKey
	}
	keyed := subjectKey != ""
	if text.Question == "" && !keyed {
		return askRequest{}, fmt.Errorf("an ask needs a question: --question, or the first paragraph of --body-file")
	}
	for _, section := range text.Ignored {
		fmt.Fprintf(ctx.Stderr(), "Warning: the body file's section %q has no place in an ask and was left out; only the first paragraph, ## Background and ## Details are read.\n", section)
	}
	var askOptions []*v1.AskOption
	optionValues, yesNo := ctx.FlagValues("option"), ctx.BoolFlag("yes-no")
	if featureCard != nil {
		optionValues = []string{"keep=" + featureCard.Options[0].Label, "drop=" + featureCard.Options[1].Label}
		yesNo = false
	}
	if len(optionValues) > 0 || yesNo || !keyed {
		options, err := resolveOptions(optionValues, yesNo)
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
	reassertWithin, err := parseReassertWithin(ctx.Flag("reassert-within"), subjectKey)
	if err != nil {
		return askRequest{}, err
	}
	sender, err := parseFrom(ctx)
	if err != nil {
		return askRequest{}, err
	}
	packetFlag := ctx.Flag("packet")
	if featureCard != nil && packetFlag == "" {
		packetFlag = featureCard.PacketPath
	}
	packetPath, brief, err := resolvePacket(packetFlag)
	if err != nil {
		return askRequest{}, err
	}
	attach, explainer, form := ctx.FlagValues("attach"), ctx.Flag("explainer"), ctx.Flag("form")
	if featureCard != nil {
		attach = append(attach, featureCard.Path)
		explainer, form = featureCard.PagePath, ""
	}
	pkt, questions := buildPacket(attach, explainer, brief, form)
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
		SubjectKey:            subjectKey,
		ReassertWithinSeconds: givenSeconds(ctx, reassertWithin),
		Attachments:           attachments,
		Questions:             questions,
		ClearAttachments:      ctx.BoolFlag("clear-attachments"),
		ClearQuestions:        ctx.BoolFlag("clear-questions"),
		RemoveAttachments:     ctx.FlagValues("remove-attachment"),
		RemoveQuestions:       ctx.FlagValues("remove-question"),
		ContextResponse:       strings.TrimSpace(ctx.Flag("context-response")),
		Packet:                packetPath,
	}
	if featureCard != nil {
		recommended, reason := featureCard.Recommendation, featureCard.RecommendationReason
		req.Recommended, req.RecommendationReason = &recommended, &reason
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

type featureDecisionCardInput struct {
	Path                 string `json:"-"`
	PagePath             string `json:"-"`
	PacketPath           string `json:"-"`
	Schema               string `json:"schema"`
	ID                   string `json:"id"`
	FeatureID            string `json:"feature_id"`
	Title                string `json:"title"`
	Summary              string `json:"summary"`
	Recommendation       string `json:"recommendation"`
	RecommendationReason string `json:"recommendation_reason"`
	Status               string `json:"status"`
	SubjectKey           string `json:"subject_key"`
	Location             struct {
		Screen string `json:"screen"`
	} `json:"location"`
	Options []struct {
		Key   string `json:"key"`
		Label string `json:"label"`
	} `json:"options"`
	Finding struct {
		Version     string `json:"version"`
		BeforeBuild string `json:"before_build"`
		AfterBuild  string `json:"after_build"`
		Component   string `json:"component"`
		Source      string `json:"source"`
	} `json:"finding"`
}

func readFeatureDecisionCard(path string) (*featureDecisionCardInput, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("--feature-card %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("--feature-card %q must be a regular local JSON file", path)
	}
	if info.Size() > 4<<20 {
		return nil, fmt.Errorf("--feature-card %q exceeds 4 MiB", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("--feature-card %q: %w", path, err)
	}
	var card featureDecisionCardInput
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("--feature-card %q is invalid JSON: %w", path, err)
	}
	if card.Schema != "research-feature-decision/v2" || card.Status != "pending" || card.ID == "" || card.FeatureID == "" || card.Title == "" || card.Summary == "" || card.RecommendationReason == "" || card.SubjectKey == "" {
		return nil, fmt.Errorf("--feature-card %q must be a complete pending research-feature-decision/v2 card", path)
	}
	if len(card.Options) != 2 || card.Options[0].Key != "keep" || card.Options[1].Key != "drop" || card.Options[0].Label == "" || card.Options[1].Label == "" {
		return nil, fmt.Errorf("--feature-card %q must offer keep followed by drop", path)
	}
	if card.Recommendation != "keep" && card.Recommendation != "drop" {
		return nil, fmt.Errorf("--feature-card %q recommendation must be keep or drop", path)
	}
	subjectHash := sha256.Sum256([]byte(card.ID + "\x00" + card.FeatureID + "\x00" + card.Finding.Version))
	if card.SubjectKey != "feature-loss:"+hex.EncodeToString(subjectHash[:]) {
		return nil, fmt.Errorf("--feature-card %q subject_key does not match its finding", path)
	}
	if card.Finding.Version == "" || card.Finding.BeforeBuild == "" || card.Finding.AfterBuild == "" || card.Finding.BeforeBuild == card.Finding.AfterBuild {
		return nil, fmt.Errorf("--feature-card %q needs a distinct measured build pair and finding version", path)
	}
	if strings.EqualFold(filepath.Ext(path), ".json") == false {
		return nil, fmt.Errorf("--feature-card %q must end in .json", path)
	}
	card.Path = path
	card.PagePath = strings.TrimSuffix(path, filepath.Ext(path)) + ".html"
	card.PacketPath = strings.TrimSuffix(path, filepath.Ext(path)) + ".BRIEF.md"
	pageInfo, err := os.Lstat(card.PagePath)
	if err != nil {
		return nil, fmt.Errorf("--feature-card companion page %q: %w", card.PagePath, err)
	}
	if !pageInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("--feature-card companion page %q must be a regular local HTML file", card.PagePath)
	}
	briefInfo, err := os.Lstat(card.PacketPath)
	if err != nil {
		return nil, fmt.Errorf("--feature-card companion packet %q: %w", card.PacketPath, err)
	}
	if !briefInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("--feature-card companion packet %q must be a regular local Markdown file", card.PacketPath)
	}
	return &card, nil
}

// resolvePacket is --packet as the Hub keeps it: an absolute path, the folder's
// BRIEF.md when it names a folder that holds one. brief is the file to attach
// beside it ("" when there is no readable file); a packet that names nothing
// readable is still sent, and the attach step says why the brief is missing.
func resolvePacket(value string) (packet *string, brief string, err error) {
	if strings.TrimSpace(value) == "" {
		return nil, "", nil
	}
	abs, err := filepath.Abs(strings.TrimSpace(value))
	if err != nil {
		return nil, "", fmt.Errorf("--packet %q: %w", value, err)
	}
	if info, statErr := os.Stat(abs); statErr == nil && info.IsDir() {
		if _, err := os.Stat(filepath.Join(abs, "BRIEF.md")); err == nil {
			abs = filepath.Join(abs, "BRIEF.md")
		}
	}
	return &abs, abs, nil
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
		RetrievalHints: []string{dryRunNextStep(msg.GetWarnings())},
	}); err != nil {
		return err
	}
	writePacketWarnings(ctx.Stderr(), msg.GetWarnings(), false)
	return nil
}

// dryRunNextStep is what to do after a dry run: send it, or first fix what
// would refuse it.
func dryRunNextStep(warnings []*v1.PacketWarning) string {
	if hasBlocking(warnings) {
		return "Do not send it yet: the Hub would refuse it. Fix the findings listed under \"Would refuse the ask\" (on stderr), then run --dry-run again."
	}
	return "To send it, run the same command without --dry-run."
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
