package conversations

// Writing-guide reminder after `conversations ask`.
//
// Why: agents write decisions with their own context in mind — internal IDs,
// abbreviations, file paths — and the operator reads them cold on a phone. The
// notification-hub skill is the writing standard. After an ask is accepted the
// CLI asks Prompt Manager whether this agent run read that skill and, when it
// cannot confirm the read, prints how to read it and how to fix the ask in
// place. The check never blocks or fails the ask: a broken receipt path must
// not stop a decision from reaching the operator.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
)

// GuideSkillID is the skill that governs how decisions are written.
const GuideSkillID = "notification-hub"

// guideReadCommand is how an agent reads the guide so the read is recorded.
const guideReadCommand = "prompt-manager skill read " + GuideSkillID

// guideCheckTimeout bounds the whole receipt lookup, port discovery included.
const guideCheckTimeout = 2 * time.Second

// Receipt statuses from Prompt Manager's GetSkillReadReceipt.
const (
	guideRead        = "read"
	guideNotRead     = "not_read"
	guideUnverified  = "unverified"
	guideUnavailable = "unavailable"
)

type guideReceipt struct {
	Status string
	Reason string
}

// guideChecker answers whether the calling run read GuideSkillID.
type guideChecker func(ctx context.Context) guideReceipt

// checkGuide is replaced in tests.
var checkGuide guideChecker = promptManagerGuideChecker

// promptManagerBase resolves Prompt Manager's API origin; replaced in tests.
var promptManagerBase = func(ctx context.Context) (string, bool) {
	port := cliutil.LookupScenarioPort(ctx, "prompt-manager", "API_PORT", cliutil.DefaultPortCachePolicy)
	if !port.Resolved() {
		return "", false
	}
	return "http://127.0.0.1:" + port.Port, true
}

// promptManagerGuideChecker asks Prompt Manager with this process's own agent
// identity token. Without a token there is no run to ask about, so it makes no
// network call.
func promptManagerGuideChecker(ctx context.Context) guideReceipt {
	token := strings.TrimSpace(os.Getenv(cliutil.EnvIdentityToken))
	if token == "" {
		return guideReceipt{Status: guideUnverified, Reason: "this shell has no agent identity"}
	}
	ctx, cancel := context.WithTimeout(ctx, guideCheckTimeout)
	defer cancel()
	base, ok := promptManagerBase(ctx)
	if !ok {
		return guideReceipt{Status: guideUnavailable, Reason: "Prompt Manager is not reachable"}
	}
	client := skillsconnect.NewSkillsServiceClient(&http.Client{Timeout: guideCheckTimeout}, base)
	req := connect.NewRequest(&skillsv1.GetSkillReadReceiptRequest{SkillId: GuideSkillID})
	req.Header().Set(cliutil.HeaderAgentIdentityToken, token)
	resp, err := client.GetSkillReadReceipt(ctx, req)
	if err != nil {
		return guideReceipt{Status: guideUnavailable, Reason: "Prompt Manager could not answer the read check"}
	}
	return guideReceipt{Status: resp.Msg.GetStatus(), Reason: resp.Msg.GetReason()}
}

// askGuideContext is what the reminder needs to say about the ask just sent.
type askGuideContext struct {
	AskID      string
	SubjectKey string
	Outcome    string
}

// writeGuideReminder prints the reminder unless the run read the guide or the
// ask can no longer change. It writes to stderr so --json output stays clean.
func writeGuideReminder(w io.Writer, receipt guideReceipt, ask askGuideContext) {
	if receipt.Status == guideRead {
		return
	}
	// An answered or operator-dismissed ask is closed; editing advice is noise.
	if ask.Outcome == "returned" || ask.Outcome == "suppressed" {
		return
	}
	var b strings.Builder
	if receipt.Status == guideNotRead {
		b.WriteString("Writing guide: this agent run has not read the notification-hub skill.\n")
	} else {
		reason := strings.TrimSpace(receipt.Reason)
		if reason == "" {
			reason = "the read check did not answer"
		}
		fmt.Fprintf(&b, "Writing guide: could not confirm that you read the notification-hub skill (%s).\n", reason)
	}
	b.WriteString("  The operator reads this decision cold on a phone. The skill says how to write it so it can be answered without your context.\n")
	fmt.Fprintf(&b, "  1. Read it if you have not: %s\n", guideReadCommand)
	if strings.TrimSpace(ask.SubjectKey) != "" {
		fmt.Fprintf(&b, "  2. If the ask falls short, fix it in place (no new push): run the same ask again with --subject-key %s and the same options, with a revised --question, --big-picture or --technical-details.\n", ask.SubjectKey)
	} else {
		fmt.Fprintf(&b, "  2. If the ask falls short: this ask has no --subject-key, so it cannot be edited in place. Withdraw it with `notification-hub conversations withdraw --ask-id %s --reason 'rewritten'` and ask again with a --subject-key.\n", ask.AskID)
	}
	_, _ = io.WriteString(w, b.String())
}
