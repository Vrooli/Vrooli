package conversations

// `conversations wait` and the context-request status an agent can branch on.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

// Exit statuses of `conversations wait` (and of `ask --and-wait`). Each means
// one thing, so a caller branches on the status instead of parsing text:
//
//	0  an answer is available: the operator answered, or the default applied
//	3  the operator asked for more context; the ask is open, revise it and wait again
//	4  the caller's own wait (--timeout) ran out; the ask is still open
//	5  the ask closed without an answer: withdrawn (by id also when a newer ask
//	   replaced it), dismissed by the operator, or expired
//
// Status 1 is a failure to wait at all (unreachable Hub, refused read, bad flags).
const (
	ExitContextRequested = 3
	ExitWaitTimedOut     = 4
	ExitClosedUnanswered = 5
)

// statusError is an error that carries a documented process exit status. The
// scenario's main honors it.
type statusError struct {
	code int
	msg  string
}

func (e *statusError) Error() string { return e.msg }
func (e *statusError) ExitCode() int { return e.code }

// askOpen reports whether a state still waits for an answer.
func askOpen(state string) bool { return state == "pending" || state == "escalated" }

// waitTarget names the ask to wait on: by id, or by the exact subject key (which
// keeps working when the asker replaces the ask, and which a restarted agent can
// re-derive).
type waitTarget struct{ AskID, SubjectKey string }

func (t waitTarget) byKey() bool { return t.SubjectKey != "" }

func (t waitTarget) command() string { return waitCommand(t.AskID, t.SubjectKey) }

// waitTargetFrom reads the ask to wait on: a positional id, --ask-id, or
// --subject-key; exactly one.
// askIDFrom reads the ask id a command was given, as the positional id or
// --ask-id (the same thing). Empty when it named none.
func askIDFrom(ctx cliapp.RunContext) (string, error) {
	askID, positional := ctx.Flag("ask-id"), ctx.Positional("id")
	if askID != "" && positional != "" && askID != positional {
		return "", fmt.Errorf("the ask id is given twice and differs: %q and %q", positional, askID)
	}
	if askID == "" {
		askID = positional
	}
	return askID, nil
}

func waitTargetFrom(ctx cliapp.RunContext) (waitTarget, error) {
	askID, err := askIDFrom(ctx)
	if err != nil {
		return waitTarget{}, err
	}
	subjectKey := ctx.Flag("subject-key")
	if (askID == "") == (subjectKey == "") {
		return waitTarget{}, fmt.Errorf("give exactly one ask to wait on: an ask id (`wait <ask-id>`) or --subject-key")
	}
	return waitTarget{AskID: askID, SubjectKey: subjectKey}, nil
}

func (s *service) wait(ctx cliapp.RunContext) error {
	target, err := waitTargetFrom(ctx)
	if err != nil {
		return err
	}
	until, err := parseTime("--timeout", ctx.Flag("timeout"), s.now())
	if err != nil {
		return err
	}
	return s.awaitAndReport(ctx, target, until, ctx.BoolFlag("until-answer"))
}

// transientWaitError reports a failure of the connection, not of the question:
// the Hub restarting, a proxy or tunnel closing a long request, a dropped stream.
// The ask is unaffected, so a wait carries on.
func transientWaitError(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		switch connectErr.Code() {
		case connect.CodeUnavailable, connect.CodeDeadlineExceeded, connect.CodeAborted:
			return true
		}
		return false
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

const (
	waitRetryFirst = time.Second
	waitRetryMax   = 15 * time.Second
)

// awaitAnswer waits until the ask is resolved, the operator asks for context
// (unless untilAnswer), or the caller's timeout passes (deadline is that time). A
// single Hub call is bounded (the Hub ends one after at most ten minutes, ask
// still open, and proxies or tunnels may cut a long request sooner), so it asks
// again, and carries on across a dropped connection with a short backoff, until
// the timeout.
func (s *service) awaitAnswer(target waitTarget, deadline string, untilAnswer bool) (*v1.WaitResponse, error) {
	until, err := time.Parse(time.RFC3339, deadline)
	if err != nil {
		return nil, err
	}
	waitCtx, cancel := waitContext(deadline)
	defer cancel()
	retry := waitRetryFirst
	for {
		resp, callErr := s.waitClient.Wait(waitCtx, connect.NewRequest(&v1.WaitRequest{
			AskId:                  target.AskID,
			SubjectKey:             target.SubjectKey,
			Deadline:               deadline,
			ReturnOnContextRequest: !untilAnswer,
		}))
		if callErr != nil {
			if !transientWaitError(callErr) || !s.now().Before(until) {
				return nil, callErr
			}
			s.pause(retry)
			retry = min(retry*2, waitRetryMax)
			continue
		}
		retry = waitRetryFirst
		msg := resp.Msg
		// Open, no question from the operator, deadline not reached: the Hub ended
		// this call at its longest length. Ask again.
		if askOpen(msg.GetState()) && msg.GetContextRequest() == nil && s.now().Before(until) {
			continue
		}
		return msg, nil
	}
}

// pause is the CLI's sleep, replaceable in tests.
func (s *service) pause(d time.Duration) {
	if s.sleep != nil {
		s.sleep(d)
		return
	}
	time.Sleep(d)
}

// awaitAndReport waits, prints the outcome, and returns the status error that
// carries the documented exit status (nil for an answer).
func (s *service) awaitAndReport(ctx cliapp.RunContext, target waitTarget, deadline string, untilAnswer bool) error {
	msg, err := s.awaitAnswer(target, deadline, untilAnswer)
	if err != nil {
		return readError("wait for notification answer", err)
	}
	if err := cliapp.RenderProtoList(ctx, msg, waitReport(msg)); err != nil {
		return err
	}
	if !ctx.JSON() {
		if _, err := fmt.Fprintln(ctx.Stdout(), waitLine(msg)); err != nil {
			return err
		}
	}
	switch {
	case askOpen(msg.GetState()) && msg.GetContextRequest() != nil:
		// Waiting by key already names the key; by id, ask the Hub for it (best
		// effort: a reader that waits by id is in the ask's lineage).
		subjectKey := target.SubjectKey
		if subjectKey == "" {
			subjectKey = s.subjectKey(msg.GetAskId())
		}
		writeContextGuidance(ctx.Stderr(), contextGuidance{
			AskID:       msg.GetAskId(),
			SubjectKey:  subjectKey,
			Question:    msg.GetContextRequest().GetQuestion(),
			WaitedByKey: target.byKey(),
		})
		return &statusError{
			code: ExitContextRequested,
			msg:  fmt.Sprintf("ask %s is still open: the operator asked for more context (exit status %d); answer it with `ask --subject-key K --context-response '...'`, then wait again", msg.GetAskId(), ExitContextRequested),
		}
	case askOpen(msg.GetState()):
		return &statusError{
			code: ExitWaitTimedOut,
			msg:  fmt.Sprintf("your wait timed out (exit status %d): the operator has not answered yet, and ask %s is still open; the ask has not expired. Wait again: %s", ExitWaitTimedOut, msg.GetAskId(), target.command()),
		}
	case msg.GetState() == "answered" || msg.GetState() == "defaulted":
		return nil
	default:
		return &statusError{
			code: ExitClosedUnanswered,
			msg:  closedMessage(msg, target),
		}
	}
}

// closedMessage says how an ask closed without an answer and what to do.
func closedMessage(msg *v1.WaitResponse, target waitTarget) string {
	text := fmt.Sprintf("ask %s was %s without an answer (exit status %d)", msg.GetAskId(), msg.GetState(), ExitClosedUnanswered)
	if reason := strings.TrimSpace(msg.GetReason()); reason != "" {
		text += ": " + reason
	}
	if msg.GetState() == "withdrawn" && !target.byKey() {
		text += ". A revision with new options replaces an ask; to follow replacements, wait by --subject-key"
	}
	return text
}

// waitLine is the one machine-readable line `wait` ends its output with.
func waitLine(msg *v1.WaitResponse) string {
	line := fmt.Sprintf("ask_id=%s state=%s", msg.GetAskId(), msg.GetState())
	if msg.GetAnswer() != "" {
		line += " answer=" + msg.GetAnswer()
	}
	return line
}

// subjectKey looks up the ask's subject key so the guidance can name the exact
// revise command. It is best effort: guidance without it is still correct.
func (s *service) subjectKey(askID string) string {
	resp, err := s.client.GetAsk(context.Background(), connect.NewRequest(&v1.GetAskRequest{AskId: askID}))
	if err != nil {
		return ""
	}
	return resp.Msg.GetAsk().GetSubjectKey()
}

func waitReport(msg *v1.WaitResponse) cliapp.ListReport {
	if req := msg.GetContextRequest(); req != nil && askOpen(msg.GetState()) {
		return cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("Ask %s: %s; the operator asked for more context", msg.GetAskId(), msg.GetState())},
			ResultsHeading: "Operator's question",
			Results:        []string{req.GetQuestion()},
			RetrievalHints: []string{"Read the whole ask: " + getCommand(msg.GetAskId())},
		}
	}
	// The summary already holds the note and every form answer, so the bare
	// fields only stand in for it when the Hub sent none.
	results := []string{msg.GetAnswerSummary()}
	if strings.TrimSpace(msg.GetAnswerSummary()) == "" {
		results = []string{msg.GetAnswer(), msg.GetNote(), msg.GetReason()}
		if notes := attachmentNoteLines(msg.GetAttachmentNotes()); len(notes) > 0 {
			results = append(results, "Notes on files:\n  "+strings.Join(notes, "\n  "))
		}
	}
	if packet := msg.GetPacket(); packet != "" && (msg.GetState() == "answered" || msg.GetState() == "defaulted") {
		results = append(results, "Packet: "+packet, "Start the executor with: "+executorMessage(packet, msg.GetAnswer()))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Ask %s: %s", msg.GetAskId(), msg.GetState())},
		ResultsHeading: "Answer",
		Results:        results,
		RetrievalHints: []string{"Read the whole ask: " + getCommand(msg.GetAskId())},
	}
}

// executorMessage is the standard spawn message for the executor of an
// answered decision (docs/agent-system/LEVELS.md), named by the packet's
// folder when the packet is a work/<ID>/BRIEF.md.
func executorMessage(packet, answer string) string {
	id := filepath.Base(packet)
	if id == "BRIEF.md" {
		id = filepath.Base(filepath.Dir(packet))
	}
	return fmt.Sprintf("Execute the approved decision %s. Read %s; the answer is %s.", id, packet, answer)
}

// getCommand is the command that reads one ask in full.
func getCommand(askID string) string { return "notification-hub conversations get " + askID }

type contextGuidance struct {
	AskID      string
	SubjectKey string
	Question   string
	// WaitedByKey says the caller waited by subject key, so it waits again the
	// same way: the key keeps working when the ask is replaced.
	WaitedByKey bool
}

// writeContextGuidance tells the waiting agent exactly how to answer the
// operator's question. It writes to stderr so --json stdout stays clean.
func writeContextGuidance(w io.Writer, g contextGuidance) {
	var b strings.Builder
	fmt.Fprintf(&b, "The operator asked for more context before answering ask %s:\n  %s\n", g.AskID, g.Question)
	if g.SubjectKey != "" {
		b.WriteString("Answer it with --context-response; the operator sees your answer under their question, and the ask is otherwise unchanged:\n")
		fmt.Fprintf(&b, "  notification-hub conversations ask --subject-key %s --context-response \"<your answer to their question>\"\n", g.SubjectKey)
		b.WriteString("  If the answer also changes the ask, send that in the same command (a revised --question, --big-picture, --attach, --remove-attachment, --form or --remove-question); the ask keeps everything else.\n")
	} else {
		fmt.Fprintf(&b, "  This ask has no --subject-key, so it cannot be revised in place. Withdraw it (`notification-hub conversations withdraw %s --reason 'needs more context'`) and ask again with a --subject-key, and the same --attach, --explainer and --form.\n", g.AskID)
	}
	if g.WaitedByKey || g.SubjectKey != "" {
		fmt.Fprintf(&b, "Then wait again: notification-hub conversations wait --subject-key %s\n", g.SubjectKey)
	} else {
		fmt.Fprintf(&b, "Then wait again: notification-hub conversations wait %s\n", g.AskID)
	}
	_, _ = io.WriteString(w, b.String())
}
