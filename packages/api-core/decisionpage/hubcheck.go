package decisionpage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/discovery"
	convv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	convconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
)

// PageAttachment is the name the page travels under in the check, the same
// name the Hub gives a scenario's inline page.
const PageAttachment = "decision.html"

// HubCheck runs Notification Hub's CheckAsk: the packet check a real send
// gets (`conversations ask --dry-run`), with nothing stored. Every check the
// Hub adds applies to the writer without a change here.
type HubCheck struct {
	// BaseURL resolves Notification Hub's API; nil uses lifecycle discovery.
	BaseURL func(context.Context) (string, error)
	HTTP    *http.Client
}

// Check sends the decision with the page as its explainer.
func (h *HubCheck) Check(ctx context.Context, r Request, html []byte) ([]Finding, error) {
	base, err := h.base(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: notification-hub: %v", ErrUnavailable, err)
	}
	client := h.HTTP
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := convconnect.NewConversationsServiceClient(client, strings.TrimRight(base, "/")).CheckAsk(ctx, connect.NewRequest(CheckRequest(r, html)))
	if err != nil {
		var ce *connect.Error
		if errors.As(err, &ce) && ce.Code() == connect.CodeInvalidArgument {
			return nil, fmt.Errorf("the Hub refused the decision: %s", ce.Message())
		}
		return nil, fmt.Errorf("%w: notification-hub: %v", ErrUnavailable, err)
	}
	out := make([]Finding, 0, len(resp.Msg.GetWarnings()))
	for _, w := range resp.Msg.GetWarnings() {
		out = append(out, Finding{Code: w.GetCode(), Field: w.GetField(), Message: w.GetMessage(), Severity: w.GetSeverity()})
	}
	return out, nil
}

func (h *HubCheck) base(ctx context.Context) (string, error) {
	if h.BaseURL != nil {
		return h.BaseURL(ctx)
	}
	return discovery.ResolveScenarioURLDefault(ctx, "notification-hub")
}

// CheckRequest is the dry-run request for a page: the decision as the sender
// will raise it, the page attached as its explainer, the page's bytes inline.
// It carries no subject key, so the check judges this decision alone rather
// than a revision of whatever the checking caller has open.
func CheckRequest(r Request, html []byte) *convv1.CheckAskRequest {
	ask := &convv1.AskRequest{Question: r.Question, Sender: &r.Sender}
	for _, o := range r.Options {
		ask.Options = append(ask.Options, &convv1.AskOption{Key: o.Key, Label: o.Label})
	}
	if r.Recommended != "" {
		ask.Recommended = &r.Recommended
	}
	if r.RecommendationReason != "" {
		ask.RecommendationReason = &r.RecommendationReason
	}
	if r.BigPicture != "" {
		ask.BigPicture = &r.BigPicture
	}
	if r.TechnicalDetails != "" {
		ask.TechnicalDetails = &r.TechnicalDetails
	}
	sum := sha(html)
	ask.Attachments = []*convv1.AskAttachment{{
		Name: PageAttachment, Label: "Decision page", Kind: convv1.AskAttachment_KIND_FILE, Role: convv1.AskAttachment_ROLE_EXPLAINER,
		Sha256: sum, SizeBytes: int64(len(html)), MediaType: "text/html",
	}}
	return &convv1.CheckAskRequest{Ask: ask, FileContents: map[string][]byte{sum: html}}
}
