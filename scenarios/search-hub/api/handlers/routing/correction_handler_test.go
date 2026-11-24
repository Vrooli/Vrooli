package routing_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"

	internalrouting "search-hub/internal/routing"
)

type correctionConnectDoer struct {
	body string
}

func (d *correctionConnectDoer) Do(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	d.body = string(body)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"results":[{"name":"command","description":"Run command","score":0.9}]}`)),
		Header:     make(http.Header),
	}, nil
}

func TestConnectQueryReturnsEvidenceBackedPossibleCorrectionWithoutRewritingQuery(t *testing.T) {
	provider := apiTestProvider("cli-health.commands", "cli-health", "CLI command", "cli-health")
	provider.Type = "command"
	strategy := internalrouting.RetrievalStrategy{
		Name:   "lexical-typo-fallback",
		Stages: []internalrouting.RetrievalStage{{Kind: internalrouting.StageLexical, Params: map[string]interface{}{"shortlist_width": float64(1), "typo_policy": "one_edit_unique"}}},
	}
	doer := &correctionConnectDoer{}
	router := internalrouting.NewRouter(internalrouting.Deps{
		Lister:   apiTestLister{providers: []*registryv1.ProviderDescriptor{provider}},
		Resolver: apiTestResolver{},
		Doer:     doer,
		Strategy: &strategy,
	})
	client := newClient(t, router)

	resp, err := client.Query(context.Background(), connect.NewRequest(&routingv1.QueryRequest{Query: "commnad"}))
	require.NoError(t, err)
	require.Contains(t, doer.body, `"query":"commnad"`, "correction suggestions must not rewrite the provider request")
	require.Len(t, resp.Msg.GetCorrectionCandidates(), 1)
	candidate := resp.Msg.GetCorrectionCandidates()[0]
	require.Equal(t, "commnad", candidate.GetSubmittedValue())
	require.Equal(t, "command", candidate.GetCandidateValue())
	require.Equal(t, "cli-health.commands", candidate.GetProviderId())
	require.Equal(t, routingv1.CorrectionCandidate_STRENGTH_BAND_POSSIBLE, candidate.GetStrength())
	require.NotNil(t, candidate.GetEvidence())
	require.Equal(t, "command", candidate.GetEvidence().GetHitId())
}
