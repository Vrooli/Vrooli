package journal

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/provenance"
	heartbeat "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
)

const investigatorPublicationPrefix = "remember/scenario-qa/"
const legacyImportPrefix = "legacy-scenario-qa-bugs/v1/"

func authorizeReservedRequestKey(ctx context.Context, requestKey string) error {
	switch {
	case strings.HasPrefix(requestKey, investigatorPublicationPrefix):
		return authorizeAccountPublication(ctx)
	case strings.HasPrefix(requestKey, legacyImportPrefix):
		return connect.NewError(connect.CodePermissionDenied, errors.New("the Scenario QA history import is retired; read existing receipts with GetEntry"))
	}
	return nil
}

type heartbeatReader interface {
	GetHeartbeat(context.Context, *connect.Request[heartbeat.MemberRequest]) (*connect.Response[heartbeat.JsonResponse], error)
}

func authorizeAccountPublication(ctx context.Context) error {
	identity := provenance.FromContext(ctx)
	if !identity.IsVerifiedAgent() {
		return connect.NewError(connect.CodePermissionDenied, errors.New("publishing a defect account requires a verified investigator run"))
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	base, err := discovery.ResolveScenarioURLDefault(ctx, "prompt-manager")
	if err != nil {
		return connect.NewError(connect.CodeUnavailable, errors.New("the investigator owner is unavailable"))
	}
	reader := heartbeatconnect.NewHeartbeatServiceClient(http.DefaultClient, base)
	return verifyInvestigatorRun(ctx, reader, identity.RunID)
}

func verifyInvestigatorRun(ctx context.Context, reader heartbeatReader, runID string) error {
	req := connect.NewRequest(&heartbeat.MemberRequest{TeamId: "scenario-qa", AgentId: "bug-investigator"})
	res, err := reader.GetHeartbeat(ctx, req)
	if err != nil {
		return connect.NewError(connect.CodeUnavailable, errors.New("cannot check the investigator's heartbeat ownership"))
	}
	last := res.Msg.GetData().GetStructValue().GetFields()["lastExecution"].GetStructValue()
	publishedRun := last.GetFields()["runId"].GetStringValue()
	if runID == "" || strings.TrimSpace(publishedRun) != runID {
		return connect.NewError(connect.CodePermissionDenied, errors.New("this run does not own the investigator's current heartbeat"))
	}
	return nil
}
