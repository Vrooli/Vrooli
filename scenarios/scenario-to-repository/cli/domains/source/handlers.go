package source

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-repository/v1/source"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-repository/v1/source/source_v1connect"
)

type handlers struct {
	client sourceconnect.SourceRepositoryServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 5*time.Minute)
	return &handlers{client: sourceconnect.NewSourceRepositoryServiceClient(httpClient, baseURL)}
}

func (h *handlers) analyzeCall(ctx cliapp.OperationContext) (*sourcev1.ClosureResponse, error) {
	resp, err := h.client.AnalyzeClosure(context.Background(), connect.NewRequest(&sourcev1.AnalyzeClosureRequest{Scenario: ctx.Positional("scenario"), SourceRoot: ctx.Flag("source-root"), SourceDigest: ctx.Flag("source-digest")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("analyze source closure", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) analyzeReport(_ cliapp.OperationContext, msg *sourcev1.ClosureResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Closure %s: %d file(s), %d unresolved obligation(s).", msg.Closure.ClosureDigest, len(msg.Closure.Files), len(msg.Closure.Unresolved))}}
}

func (h *handlers) assembleCall(ctx cliapp.OperationContext) (*sourcev1.ArtifactResponse, error) {
	resp, err := h.client.AssembleExport(context.Background(), connect.NewRequest(&sourcev1.AssembleExportRequest{Scenario: ctx.Positional("scenario"), SourceRoot: ctx.Flag("source-root"), SourceDigest: ctx.Flag("source-digest"), RecipeYaml: ctx.Flag("recipe-yaml"), OutputPath: ctx.Flag("output-path")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("assemble source export", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) assembleReport(_ cliapp.OperationContext, msg *sourcev1.ArtifactResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Prepared %s (%s).", msg.Artifact.ArtifactId, msg.Artifact.ArchiveDigest)}, NextCommand: []string{"`source verify <artifact-id>` — verify this exact archive"}}
}

func (h *handlers) verifyCall(ctx cliapp.OperationContext) (*sourcev1.VerificationResponse, error) {
	resp, err := h.client.VerifyExport(context.Background(), connect.NewRequest(&sourcev1.VerifyExportRequest{ArtifactId: ctx.Positional("artifact-id"), ArchivePath: ctx.Flag("archive-path")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("verify source export", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) verifyReport(_ cliapp.OperationContext, msg *sourcev1.VerificationResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Artifact %s verification: %s.", msg.ArtifactId, msg.Status)}, Results: msg.GatesPassed, ResultsHeading: "Passed gates"}
}

func (h *handlers) preparePublicationCall(ctx cliapp.OperationContext) (*sourcev1.PublicationPreviewResponse, error) {
	resp, err := h.client.PreparePublication(context.Background(), connect.NewRequest(&sourcev1.PreparePublicationRequest{DistributionId: ctx.Positional("distribution-id"), ArtifactId: ctx.Flag("artifact-id"), DestinationKind: ctx.Flag("destination-kind"), DestinationReference: ctx.Flag("destination-reference")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("prepare human publication", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) preparePublicationReport(_ cliapp.OperationContext, msg *sourcev1.PublicationPreviewResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{msg.HumanAction}, NextCommand: []string{msg.ReadbackOracle}}
}

func (h *handlers) getDistributionCall(ctx cliapp.OperationContext) (*sourcev1.DistributionResponse, error) {
	resp, err := h.client.GetDistribution(context.Background(), connect.NewRequest(&sourcev1.GetDistributionRequest{DistributionId: ctx.Positional("distribution-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get distribution", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) getDistributionReport(_ cliapp.OperationContext, msg *sourcev1.DistributionResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Distribution %s: verification=%s publication=%s.", msg.Distribution.DistributionId, msg.Distribution.VerificationStatus, msg.Distribution.PublicationStatus)}}
}
