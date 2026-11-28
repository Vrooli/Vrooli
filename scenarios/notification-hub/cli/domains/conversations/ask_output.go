package conversations

// What `conversations ask` prints, and what it says when the Hub refuses.

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	"google.golang.org/protobuf/encoding/protojson"
)

// askJSON is the stable --json shape of `ask`: ask_id, state, outcome and
// warnings, always present. It is the wire AskResponse without the notification
// the Hub made for it.
var askJSON = protojson.MarshalOptions{UseProtoNames: true, EmitDefaultValues: true, Multiline: true, Indent: "  "}

// askState is the state of the ask after the call.
func askState(resp *v1.AskResponse) string {
	if resp.GetState() != "" {
		return resp.GetState()
	}
	return "pending"
}

// waitCommand is the command that waits for an ask's answer.
func waitCommand(askID, subjectKey string) string {
	if askID != "" {
		return "notification-hub conversations wait " + askID
	}
	return "notification-hub conversations wait --subject-key " + subjectKey
}

// renderAsk prints the result of an ask. Human output ends with one
// machine-readable line; --json prints the stable shape.
func renderAsk(ctx cliapp.RunContext, resp *v1.AskResponse, built askRequest) error {
	state := askState(resp)
	if ctx.JSON() {
		return cliapp.RenderProtoListWithJSONOptions(ctx, &v1.AskResponse{
			AskId: resp.GetAskId(), State: state, Outcome: resp.GetOutcome(), Warnings: resp.GetWarnings(), Kind: resp.GetKind(),
		}, cliapp.ListReport{}, askJSON)
	}
	waiting := ctx.BoolFlag("and-wait")
	line := fmt.Sprintf("ask_id=%s state=%s outcome=%s", resp.GetAskId(), state, outcomeName(resp))
	if built.derived {
		line += " idempotency_key=" + built.req.GetIdempotencyKey()
	}
	// With --and-wait the command goes on to wait, so it needs no next step.
	if !waiting && (state == "pending" || state == "escalated") {
		line += " next: " + waitCommand(resp.GetAskId(), "")
	}
	_, err := fmt.Fprintf(ctx.Stdout(), "%s\n%s\n", askOutcomeLine(resp), line)
	return err
}

// outcomeName is the outcome an ask without a subject key leaves empty.
func outcomeName(resp *v1.AskResponse) string {
	if resp.GetOutcome() == "" {
		return "created"
	}
	return resp.GetOutcome()
}

// askError says what went wrong with an ask and what to do next.
func askError(err error, req *v1.AskRequest) error {
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return cliapp.WrapAPIError("ask notification question", err, nil)
	}
	key := req.GetIdempotencyKey()
	switch connectErr.Code() {
	case connect.CodeAlreadyExists:
		return fmt.Errorf("ask notification question: an ask with --idempotency-key %q already exists with different terms (%s). "+
			"To send the same ask again, change nothing (or drop --idempotency-key: the CLI derives one from the terms). "+
			"To change the wording of an open ask, add --subject-key <situation> and drop --idempotency-key: the Hub then updates that ask in place. "+
			"To ask something new, drop --idempotency-key or pick another one", key, connectMessage(connectErr))
	case connect.CodeFailedPrecondition:
		if connectErr.Meta().Get(errorReasonHeader) == reasonAskRefused {
			return refusedError(connectMessage(connectErr))
		}
		if strings.Contains(connectErr.Message(), "withdrawn") {
			return fmt.Errorf("ask notification question: %s. The situation %q was withdrawn after this request was made; send the ask again (a request made now is newer than the withdrawal)", connectMessage(connectErr), req.GetSubjectKey())
		}
	case connect.CodeUnauthenticated:
		return fmt.Errorf("ask notification question: the Hub could not verify who is asking. Run this inside the agent run that needs the answer, or give the shell the owner's credential (set NOTIFICATION_HUB_API_TOKEN, or run it on this machine in personal_local mode)")
	}
	return cliapp.WrapAPIError("ask notification question", err, nil)
}

// The Hub names why it refused a request in this header; ask_refused is an
// agent's ask a blocking finding of the packet check refused.
const (
	errorReasonHeader = "Hub-Error-Reason"
	reasonAskRefused  = "ask_refused"
)

// refusedError lists, one per line, the findings that refused an ask and how
// to fix each. The Hub's message is "refused, nothing was sent: f1 | f2 | next".
func refusedError(message string) error {
	body := strings.TrimPrefix(message, "refused, nothing was sent: ")
	parts := strings.Split(body, " | ")
	var b strings.Builder
	b.WriteString("ask notification question: the Hub refused the ask and nothing was sent. Fix each finding:")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || strings.HasPrefix(part, "Fix each one") {
			continue
		}
		b.WriteString("\n  - " + part)
	}
	b.WriteString("\nThen check with the same command plus --dry-run until nothing would refuse it, and send again.")
	return errors.New(b.String())
}

// connectMessage is the Hub's own words, without the Connect error prefix.
func connectMessage(err *connect.Error) string { return err.Message() }

// writeKept tells a sender that revised an ask what it left to the open ask, so
// "I said nothing about it" is never a guess. A request that opened a new ask
// kept nothing.
func writeKept(w io.Writer, req *v1.AskRequest, outcome string) {
	switch outcome {
	case "updated", "reasserted", "replaced":
	default:
		return
	}
	kept := []string{}
	add := func(left bool, what string) {
		if left {
			kept = append(kept, what)
		}
	}
	add(req.GetQuestion() == "", "the question")
	add(len(req.GetOptions()) == 0, "the options")
	add(req.Recommended == nil && len(req.GetOptions()) == 0, "the recommendation")
	add(req.BigPicture == nil, "the big picture")
	add(req.TechnicalDetails == nil, "the technical details")
	add(req.ContextUrl == nil, "the context link")
	add(req.Urgency == nil, "the urgency")
	add(req.Sender == nil, "the sender")
	add(len(req.GetAttachments()) == 0 && !req.GetClearAttachments() && len(req.GetRemoveAttachments()) == 0, "the attachments")
	add(len(req.GetQuestions()) == 0 && !req.GetClearQuestions() && len(req.GetRemoveQuestions()) == 0, "the form")
	if len(kept) == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "Kept as the open ask has them: %s.\n", strings.Join(kept, ", "))
}
