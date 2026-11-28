package validation

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	experimentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment/experiment_v1connect"
)

// requireLabQualificationEvidenceAPI detects the lab-only evidence service
// before a validation command starts inference, mutates settings, or assembles
// a large receipt. Product API deployments intentionally omit this surface.
func requireLabQualificationEvidenceAPI(ctx context.Context, client experimentconnect.ExperimentServiceClient, command string) error {
	if client == nil {
		return fmt.Errorf("qualification evidence client is unavailable; use `audio-tools --instance lab %s`", command)
	}
	_, err := client.ListQualificationEvidence(ctx, connect.NewRequest(&experimentv1.ListQualificationEvidenceRequest{
		EngineId: "__audio_tools_evidence_api_preflight__",
	}))
	if err == nil {
		return nil
	}
	if code := connect.CodeOf(err); code == connect.CodeNotFound || code == connect.CodeUnimplemented {
		return fmt.Errorf("qualification evidence is served by the Audio Tools lab API; retry with `audio-tools --instance lab %s`: %w", command, err)
	}
	return fmt.Errorf("preflight qualification evidence API: %w", err)
}
