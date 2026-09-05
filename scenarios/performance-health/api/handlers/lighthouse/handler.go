// Package lighthouse mounts performance-health's LighthouseService — wraps the
// Lighthouse CLI to score a scenario's UI, with a clean skip when unavailable.
package lighthouse

import (
	"context"
	"log"

	"connectrpc.com/connect"

	internallh "performance-health/internal/lighthouse"

	lighthousev1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/lighthouse"
	lighthouseconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/lighthouse/lighthouse_v1connect"
)

// Handler implements the generated LighthouseServiceHandler.
type Handler struct {
	lighthouseconnect.UnimplementedLighthouseServiceHandler
	svc    *internallh.Service
	logger *log.Logger
}

// NewHandler builds a lighthouse Handler.
func NewHandler(svc *internallh.Service, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{svc: svc, logger: logger}
}

var _ lighthouseconnect.LighthouseServiceHandler = (*Handler)(nil)

// RunLighthouse scores a scenario's pages (clean skip when unavailable).
func (h *Handler) RunLighthouse(ctx context.Context, req *connect.Request[lighthousev1.RunLighthouseRequest]) (*connect.Response[lighthousev1.RunLighthouseResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalid("scenario is required"))
	}
	res, err := h.svc.Score(ctx, scenario, req.Msg.GetPath())
	if err != nil {
		h.logger.Printf("lighthouse.RunLighthouse(%s): %v", scenario, err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &lighthousev1.RunLighthouseResponse{
		Scenario: res.Scenario,
		Outcome:  outcomeToProto(res.Outcome),
		Reason:   res.Reason,
	}
	for _, p := range res.Pages {
		out.Pages = append(out.Pages, &lighthousev1.PageScore{
			Url:           p.URL,
			Performance:   p.Performance,
			Accessibility: p.Accessibility,
			BestPractices: p.BestPractices,
			Seo:           p.SEO,
			Violations:    p.Violations,
		})
	}
	return connect.NewResponse(out), nil
}

func outcomeToProto(o internallh.Outcome) lighthousev1.LighthouseOutcome {
	switch o {
	case internallh.OutcomeScored:
		return lighthousev1.LighthouseOutcome_LIGHTHOUSE_OUTCOME_SCORED
	case internallh.OutcomeSkipped:
		return lighthousev1.LighthouseOutcome_LIGHTHOUSE_OUTCOME_SKIPPED
	case internallh.OutcomeFailed:
		return lighthousev1.LighthouseOutcome_LIGHTHOUSE_OUTCOME_FAILED
	default:
		return lighthousev1.LighthouseOutcome_LIGHTHOUSE_OUTCOME_UNSPECIFIED
	}
}

type invalidArg string

func (e invalidArg) Error() string { return string(e) }

func errInvalid(msg string) error { return invalidArg(msg) }
