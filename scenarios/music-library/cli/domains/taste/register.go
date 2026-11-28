package taste

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	tastev1 "github.com/vrooli/vrooli/packages/proto/gen/go/music-library/v1/taste"
	tastev1connect "github.com/vrooli/vrooli/packages/proto/gen/go/music-library/v1/taste/taste_v1connect"
)

const GroupName = "taste"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := tastev1connect.NewTasteServiceClient(httpClient, baseURL)
	profile := func(x cliapp.OperationContext) (*tastev1.GetProfileResponse, error) {
		r, err := client.GetProfile(context.Background(), connect.NewRequest(&tastev1.GetProfileRequest{Context: x.Flag("context")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("read taste profile", err, nil)
		}
		return r.Msg, nil
	}
	set := func(x cliapp.OperationContext) (*tastev1.SetControlResponse, error) {
		value, err := parseValue(x.Flag("value"))
		if err != nil {
			return nil, err
		}
		r, err := client.SetControl(context.Background(), connect.NewRequest(&tastev1.SetControlRequest{Context: x.Flag("context"), Control: &tastev1.TasteControl{Attribute: x.Flag("attribute"), Mode: x.Flag("mode"), Value: value}}))
		if err != nil {
			return nil, cliapp.WrapAPIError("set taste control", err, nil)
		}
		return r.Msg, nil
	}
	pair := func(x cliapp.OperationContext) (*tastev1.SelectPairResponse, error) {
		r, err := client.SelectPair(context.Background(), connect.NewRequest(&tastev1.SelectPairRequest{Context: x.Flag("context"), RandomBaseline: x.Flag("random") == "true", RandomSeed: parseSeed(x.Flag("seed"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("select taste pair", err, nil)
		}
		return r.Msg, nil
	}
	judge := func(x cliapp.OperationContext) (*tastev1.RecordJudgmentResponse, error) {
		r, err := client.RecordJudgment(context.Background(), connect.NewRequest(&tastev1.RecordJudgmentRequest{Context: x.Flag("context"), LeftTrackSha256: x.Flag("left"), RightTrackSha256: x.Flag("right"), Choice: x.Flag("choice"), Reason: x.Flag("reason"), AttributeScope: x.Flag("attribute-scope"), ReversesJudgmentId: x.Flag("reverses")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("record taste judgment", err, nil)
		}
		return r.Msg, nil
	}
	recommend := func(x cliapp.OperationContext) (*tastev1.RecommendResponse, error) {
		r, err := client.Recommend(context.Background(), connect.NewRequest(&tastev1.RecommendRequest{Context: x.Flag("context"), SeedTrackSha256: x.Flag("seed"), Limit: 10}))
		if err != nil {
			return nil, cliapp.WrapAPIError("recommend from taste", err, nil)
		}
		return r.Msg, nil
	}
	bindings := map[string]cliapp.PrimitiveHandler{
		"TasteService.GetProfile": cliapp.ProtoList(profile, func(_ cliapp.OperationContext, r *tastev1.GetProfileResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Taste profile %q revision %d", r.Profile.Context, r.Profile.Revision)}, ResultsHeading: "Weights", Results: formatWeights(r.Profile.Weights), RetrievalHints: []string{"taste pair — choose the next informative comparison", "taste recommend --seed <sha256> — build a recommendation brief"}}
		}),
		"TasteService.SetControl": cliapp.ProtoMutation(set, func(_ cliapp.OperationContext, r *tastev1.SetControlResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{fmt.Sprintf("Saved explicit taste control at revision %d.", r.Profile.Revision)}}
		}),
		"TasteService.SelectPair": cliapp.ProtoList(pair, func(_ cliapp.OperationContext, r *tastev1.SelectPairResponse) cliapp.ListReport {
			if r.Left == nil || r.Right == nil {
				return cliapp.ListReport{Summary: []string{"No fresh pair is available; the catalog or candidate evidence is sparse."}}
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Expected information %.3f; %d candidates.", r.ExpectedInformation, r.AvailableCandidateCount)}, ResultsHeading: "Compare these tracks", Results: []string{fmt.Sprintf("LEFT %s — %s [%s]", r.Left.TrackSha256, r.Left.Title, strings.Join(r.Left.Tags, ", ")), fmt.Sprintf("RIGHT %s — %s [%s]", r.Right.TrackSha256, r.Right.Title, strings.Join(r.Right.Tags, ", "))}}
		}),
		"TasteService.RecordJudgment": cliapp.ProtoMutation(judge, func(_ cliapp.OperationContext, r *tastev1.RecordJudgmentResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded operator judgment at profile revision %d.", r.Profile.Revision)}}
		}),
		"TasteService.Recommend": cliapp.ProtoList(recommend, func(_ cliapp.OperationContext, r *tastev1.RecommendResponse) cliapp.ListReport {
			lines := make([]string, 0, len(r.Recommendations))
			for _, x := range r.Recommendations {
				lines = append(lines, fmt.Sprintf("%d. %s — %.3f ± %.3f%s", x.Rank, x.Candidate.Title, x.PredictedPreference, x.Uncertainty, formatContributions(x.Contributions)))
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("%s · %d recommendations", r.EvidenceState, len(lines)), r.Mq2Brief}, ResultsHeading: "Score-derived recommendations", Results: lines}
		}),
	}
	return cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
}

func parseValue(raw string) (float64, error) {
	var value float64
	if _, err := fmt.Sscan(raw, &value); err != nil {
		return 0, fmt.Errorf("invalid --value: %w", err)
	}
	return value, nil
}
func parseSeed(raw string) int64 { var value int64; _, _ = fmt.Sscan(raw, &value); return value }
func formatWeights(weights []*tastev1.TasteWeight) []string {
	out := make([]string, 0, len(weights))
	for _, w := range weights {
		out = append(out, fmt.Sprintf("%s %.3f ± %.3f (%d evidence)", w.Attribute, w.Mean, w.Variance, w.EvidenceCount))
	}
	return out
}
func formatContributions(contributions []*tastev1.TasteContribution) string {
	parts := make([]string, 0, len(contributions))
	for _, c := range contributions {
		parts = append(parts, fmt.Sprintf("%s=%+.3f", c.Attribute, c.Contribution))
	}
	if len(parts) == 0 {
		return " (no supported feature contributions yet)"
	}
	return " (" + strings.Join(parts, ", ") + ")"
}
