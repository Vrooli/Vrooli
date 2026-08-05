package evidence

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence"
	evidenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence/evidencev1connect"
)

const GroupName = "evidence"

type handlers struct {
	client evidenceconnect.EvidenceServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: evidenceconnect.NewEvidenceServiceClient(httpClient, baseURL)}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*evidencev1.ListTargetVerdictsResponse, error) {
	response, err := h.client.ListTargetVerdicts(context.Background(), connect.NewRequest(&evidencev1.ListTargetVerdictsRequest{
		ProfileId:     ctx.Positional("profile_id"),
		GitCommitHash: ctx.Positional("git_commit_hash"),
		PageSize:      1000,
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list evidence verdicts", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no evidence response")
	}
	return response.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, response *evidencev1.ListTargetVerdictsResponse) cliapp.ListReport {
	results := make([]string, 0, len(response.GetVerdicts()))
	for _, verdict := range response.GetVerdicts() {
		target := verdict.GetTarget()
		if target == nil {
			results = append(results, fmt.Sprintf("%s — target unspecified", verdict.GetDisposition().String()))
			continue
		}
		results = append(results, fmt.Sprintf("%s — %s/%s (%s)", verdict.GetDisposition().String(), target.GetRamp(), target.GetPlatform(), target.GetOs()))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d evidence verdict(s).", response.GetCount())},
		ResultsHeading: "Target verdicts",
		Results:        results,
		RetrievalHints: []string{"The API response includes producer-owned evidence references and checksums."},
	}
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	return cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"EvidenceService.ListTargetVerdicts": cliapp.ProtoList(h.listCall, h.listReport),
	})
}
