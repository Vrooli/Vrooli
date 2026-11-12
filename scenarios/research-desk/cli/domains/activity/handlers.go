package activity

import (
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"

	"research-desk/cli/domains/render"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity"
	activityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity/activity_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type handlers struct {
	client activityconnect.ActivityServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: activityconnect.NewActivityServiceClient(httpClient, baseURL)}
}

func (h *handlers) nowCall(cliapp.OperationContext) (*activityv1.GetOverviewResponse, error) {
	return render.Call("read the overview", h.client.GetOverview, &activityv1.GetOverviewRequest{})
}

func (h *handlers) commissionsCall(ctx cliapp.OperationContext) (*activityv1.ListCommissionsResponse, error) {
	req := &activityv1.ListCommissionsRequest{}
	for _, word := range ctx.FlagValues("state") {
		state, ok := stateFromWord(word)
		if !ok {
			return nil, fmt.Errorf("unknown state %q; use one of %s", word, strings.Join(stateWords, ", "))
		}
		req.States = append(req.States, state)
	}
	return render.Call("list commissions", h.client.ListCommissions, req)
}

func (h *handlers) commissionCall(ctx cliapp.OperationContext) (*activityv1.GetCommissionResponse, error) {
	id := ctx.Positional("id")
	return render.Call(fmt.Sprintf("read commission %q", id), h.client.GetCommission, &activityv1.GetCommissionRequest{Id: id})
}

func (h *handlers) topicsCall(cliapp.OperationContext) (*activityv1.ListTopicsResponse, error) {
	return render.Call("list topics", h.client.ListTopics, &activityv1.ListTopicsRequest{})
}

// Subject maps the word "commission" or "topic" to the ChangeSubject enum.
func Subject(word string) (activityv1.ChangeSubject, error) {
	switch word {
	case "commission":
		return activityv1.ChangeSubject_CHANGE_SUBJECT_COMMISSION, nil
	case "topic":
		return activityv1.ChangeSubject_CHANGE_SUBJECT_TOPIC, nil
	}
	return 0, fmt.Errorf("the first argument is \"commission\" or \"topic\", not %q", word)
}

func (h *handlers) changesCall(ctx cliapp.OperationContext) (*activityv1.GetChangesResponse, error) {
	subject, err := Subject(ctx.Positional("subject"))
	if err != nil {
		return nil, err
	}
	req := &activityv1.GetChangesRequest{Subject: subject, Id: ctx.Positional("id")}
	if raw := ctx.Flag("since"); raw != "" {
		since, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, fmt.Errorf("--since takes a UTC time such as 2026-11-11T19:00:00Z, not %q", raw)
		}
		req.Since = timestamppb.New(since)
	}
	return render.Call(fmt.Sprintf("read changes for %s %q", ctx.Positional("subject"), req.Id), h.client.GetChanges, req)
}
