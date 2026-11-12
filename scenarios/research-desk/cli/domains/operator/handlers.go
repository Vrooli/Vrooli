package operator

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"

	"research-desk/cli/domains/activity"
	"research-desk/cli/domains/render"

	operatorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/operator"
	operatorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/operator/operator_v1connect"
)

type handlers struct {
	client operatorconnect.OperatorServiceClient
}

func (h *handlers) needsYouCall(cliapp.OperationContext) (*operatorv1.ListNeedsYouResponse, error) {
	return render.Call("read the needs-you feed", h.client.ListNeedsYou, &operatorv1.ListNeedsYouRequest{})
}

func (h *handlers) answerCall(ctx cliapp.OperationContext) (*operatorv1.AnswerResponse, error) {
	req := &operatorv1.AnswerRequest{CommissionId: ctx.Positional("commission-id"), ItemId: ctx.Positional("item-id"), OptionKey: ctx.Positional("option-key"), Note: ctx.Flag("note"), DryRun: ctx.BoolFlag("dry-run")}
	return render.Call(fmt.Sprintf("answer %s in %s", req.ItemId, req.CommissionId), h.client.Answer, req)
}

func (h *handlers) sayCall(ctx cliapp.OperationContext) (*operatorv1.AddEntryResponse, error) {
	req := &operatorv1.AddEntryRequest{CommissionId: ctx.Positional("commission-id"), Text: ctx.Positional("text"), DryRun: ctx.BoolFlag("dry-run")}
	switch kind := ctx.Positional("kind"); kind {
	case "direction":
		req.Kind = operatorv1.EntryKind_ENTRY_KIND_DIRECTION
	case "wind-down":
		req.Kind = operatorv1.EntryKind_ENTRY_KIND_WIND_DOWN
	default:
		return nil, fmt.Errorf("the kind is \"direction\" or \"wind-down\", not %q (steering is `enable-steering <commission-id>`)", kind)
	}
	return render.Call("write to OPERATOR.md of "+req.CommissionId, h.client.AddEntry, req)
}

func (h *handlers) steeringCall(ctx cliapp.OperationContext) (*operatorv1.EnableSteeringResponse, error) {
	id := ctx.Positional("commission-id")
	return render.Call("enable steering for "+id, h.client.EnableSteering, &operatorv1.EnableSteeringRequest{CommissionId: id})
}

func (h *handlers) reviewsCall(cliapp.OperationContext) (*operatorv1.ListReviewsResponse, error) {
	return render.Call("list reviews", h.client.ListReviews, &operatorv1.ListReviewsRequest{})
}

func (h *handlers) verdictCall(ctx cliapp.OperationContext) (*operatorv1.AddVerdictResponse, error) {
	req := &operatorv1.AddVerdictRequest{ReviewId: ctx.Positional("review-id"), Topic: ctx.Positional("topic"), Text: ctx.Positional("text"), DryRun: ctx.BoolFlag("dry-run")}
	return render.Call("record a verdict in "+req.ReviewId, h.client.AddVerdict, req)
}

func (h *handlers) courierCall(ctx cliapp.OperationContext) (*operatorv1.SetCourierResponse, error) {
	req := &operatorv1.SetCourierRequest{CommissionId: ctx.Positional("commission-id"), Global: ctx.BoolFlag("global")}
	switch word := ctx.Positional("switch"); word {
	case "on":
		req.On = true
	case "off":
	default:
		return nil, fmt.Errorf("the switch is \"on\" or \"off\", not %q", word)
	}
	return render.Call("set the courier", h.client.SetCourier, req)
}

func (h *handlers) markSeenCall(ctx cliapp.OperationContext) (*operatorv1.MarkSeenResponse, error) {
	if ctx.BoolFlag("all") {
		if ctx.Positional("subject") != "" || ctx.Positional("id") != "" {
			return nil, fmt.Errorf("give --all, or a subject and id, not both")
		}
		return render.Call("record a look at everything", h.client.MarkSeen, &operatorv1.MarkSeenRequest{All: true})
	}
	if ctx.Positional("subject") == "" || ctx.Positional("id") == "" {
		return nil, fmt.Errorf("give a subject (commission or topic) and an id, or --all")
	}
	subject, err := activity.Subject(ctx.Positional("subject"))
	if err != nil {
		return nil, err
	}
	req := &operatorv1.MarkSeenRequest{Subject: subject, Id: ctx.Positional("id")}
	return render.Call(fmt.Sprintf("record a look at %s %q", ctx.Positional("subject"), req.Id), h.client.MarkSeen, req)
}

func (h *handlers) courierTestCall(cliapp.OperationContext) (*operatorv1.TestCourierResponse, error) {
	return render.Call("test delivery to Notification Hub", h.client.TestCourier, &operatorv1.TestCourierRequest{})
}
