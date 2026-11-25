package conversations

import (
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	"notification-hub/internal/hub"
)

// The Hub projects readiness, relatedness and follow-through onto a decision
// detail so the operator can judge it without opening anything else. The
// conversions stay here so the transport layer owns the wire shape exactly once.

func relatedToProto(items []hub.RelatedAsk) []*v1.RelatedAsk {
	if len(items) == 0 {
		return nil
	}
	out := make([]*v1.RelatedAsk, 0, len(items))
	for _, item := range items {
		out = append(out, &v1.RelatedAsk{
			AskId: item.AskID, Title: item.Title, Question: item.Question,
			Score: item.Score, Basis: item.Basis, State: item.State, Sender: item.Sender,
		})
	}
	return out
}

func followthroughToProto(report *hub.Followthrough) *v1.Followthrough {
	if report == nil {
		return nil
	}
	return &v1.Followthrough{
		State: report.State, Owner: report.Owner, RunRef: report.RunRef,
		StartedAt: report.StartedAt, FinishedAt: report.FinishedAt,
		BoundSeconds: int32(report.BoundSeconds), ResultUrl: report.ResultURL,
		Summary: report.Summary, ReportedAt: report.ReportedAt,
	}
}
