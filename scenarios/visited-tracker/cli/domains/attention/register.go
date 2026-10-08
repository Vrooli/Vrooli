package attention

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	attentionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/visited-tracker/v1/attention"
	"github.com/vrooli/vrooli/packages/proto/gen/go/visited-tracker/v1/attention/attentionv1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func Register(core *cliapp.ScenarioApp, campaignID *string, manifest []byte) (cliapp.SubcommandGroup, error) {
	handlers := map[string]func(cliapp.RunContext) error{}
	for _, method := range []string{"EnsureCampaign", "Preview", "Claim", "Complete"} {
		handlers["AttentionService."+method] = func(ctx cliapp.RunContext) error { return run(core, campaignID, method, ctx) }
	}
	return cliapp.LoadFromManifest(manifest, "attention", handlers)
}

func run(core *cliapp.ScenarioApp, campaignID *string, operation string, runctx cliapp.RunContext) error {
	var location, tag, worker, requestID, claimID, outcome, evidence string
	var patterns []string
	var limit, maxFiles, ttl int32
	var id string
	parseNumber := func(name string, fallback int32) (int32, error) {
		raw := runctx.Flag(name)
		if raw == "" {
			return fallback, nil
		}
		value, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("--%s: %w", name, err)
		}
		return int32(value), nil
	}
	var err error
	if operation == "EnsureCampaign" {
		location, tag = runctx.Flag("location"), runctx.Flag("tag")
		patterns = runctx.FlagValues("patterns")
		maxFiles, err = parseNumber("max-files", 200)
	} else {
		id = runctx.Flag("campaign-id")
		if id == "" {
			id = *campaignID
		}
	}
	if operation == "Preview" {
		limit, err = parseNumber("limit", 10)
	}
	if operation == "Claim" || operation == "Complete" {
		worker = runctx.Flag("worker")
	}
	if operation == "Claim" {
		requestID = runctx.Flag("request-id")
		ttl, err = parseNumber("ttl-seconds", 600)
	}
	if operation == "Complete" {
		claimID, outcome, evidence = runctx.Flag("claim-id"), runctx.Flag("outcome"), runctx.Flag("evidence")
	}
	if err != nil {
		return err
	}

	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := attentionv1connect.NewAttentionServiceClient(httpClient, baseURL)
	ctx := context.Background()
	var msg proto.Message
	switch operation {
	case "EnsureCampaign":
		res, err := client.EnsureCampaign(ctx, connect.NewRequest(&attentionv1.EnsureCampaignRequest{Location: location, Tag: tag, Patterns: patterns, MaxFiles: int32(maxFiles)}))
		if err != nil {
			return err
		}
		msg = res.Msg
	case "Preview":
		res, err := client.Preview(ctx, connect.NewRequest(&attentionv1.PreviewRequest{CampaignId: id, Limit: int32(limit)}))
		if err != nil {
			return err
		}
		msg = res.Msg
	case "Claim":
		res, err := client.Claim(ctx, connect.NewRequest(&attentionv1.ClaimRequest{CampaignId: id, RequestId: requestID, Worker: worker, TtlSeconds: int32(ttl)}))
		if err != nil {
			return err
		}
		msg = res.Msg
	case "Complete":
		res, err := client.Complete(ctx, connect.NewRequest(&attentionv1.CompleteRequest{CampaignId: id, ClaimId: claimID, Worker: worker, Outcome: outcome, Evidence: evidence}))
		if err != nil {
			return err
		}
		msg = res.Msg
	}
	if runctx.JSON() {
		data, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(msg)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(runctx.Stdout(), string(data))
		return err
	}
	switch res := msg.(type) {
	case *attentionv1.CampaignResponse:
		fmt.Fprintf(runctx.Stdout(), "Campaign: %s (revision %d)\n", res.CampaignId, res.Revision)
	case *attentionv1.PreviewResponse:
		for _, row := range res.Candidates {
			fmt.Fprintf(runctx.Stdout(), "%s  score=%.2f  revision=%s\n", row.Path, row.Score, row.Revision)
		}
		fmt.Fprintf(runctx.Stdout(), "%d candidate(s); preview does not reserve work.\n", len(res.Candidates))
	case *attentionv1.ClaimResponse:
		if res.NoWork {
			fmt.Fprintln(runctx.Stdout(), "No unclaimed work is available.")
		} else if res.Claim != nil {
			fmt.Fprintf(runctx.Stdout(), "Claim: %s\nFile: %s\nRevision: %s\nExpires: %s\nOutcome: %s\n", res.Claim.Id, res.Claim.Path, res.Claim.Revision, res.Claim.ExpiresAt, res.Claim.Outcome)
		}
	}
	return nil
}
