package domains

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
	walkv1 "github.com/vrooli/vrooli/packages/proto/gen/go/command-center/v1/walk"
	walkconnect "github.com/vrooli/vrooli/packages/proto/gen/go/command-center/v1/walk/walk_v1connect"
)

func walkHandlers(client walkconnect.WalkServiceClient, read func(cliapp.OperationContext) (*walkv1.ReadResponse, error), report func(cliapp.OperationContext, *walkv1.ReadResponse) cliapp.ListReport) map[string]cliapp.PrimitiveHandler {
	state := func(c cliapp.OperationContext) (*walkv1.StateResponse, error) {
		r, e := client.State(context.Background(), connect.NewRequest(&walkv1.StateRequest{Channel: c.Flag("channel")}))
		if e != nil {
			return nil, e
		}
		return r.Msg, nil
	}
	publish := func(c cliapp.OperationContext) (*walkv1.Receipt, error) {
		r, e := client.Publish(context.Background(), connect.NewRequest(&walkv1.PublishRequest{Channel: c.Flag("channel"), RequestKey: c.Flag("request-key"), ExpectedPreviousId: c.Flag("expected-previous-id"), ProgramId: c.Flag("program-id"), EnvelopeJson: c.Flag("envelope-json"), Briefing: c.Flag("briefing"), FleetHealthJson: c.Flag("fleet-health-json")}))
		if e != nil {
			return nil, e
		}
		return r.Msg, nil
	}
	checkpoint := func(c cliapp.OperationContext) (*walkv1.Receipt, error) {
		r, e := client.Checkpoint(context.Background(), connect.NewRequest(&walkv1.CheckpointRequest{Channel: c.Flag("channel"), RequestKey: c.Flag("request-key"), ExpectedPreviousId: c.Flag("expected-previous-id"), WalkId: c.Flag("walk-id"), State: c.Flag("state"), ResumePhase: c.Flag("resume-phase"), Content: c.Flag("content")}))
		if e != nil {
			return nil, e
		}
		return r.Msg, nil
	}
	receipt := func(_ cliapp.OperationContext, r *walkv1.Receipt) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Verified ledger receipt %s; existing=%t; channel=%s", r.EntryId, r.Existing, r.Channel)}}
	}
	stateReport := func(_ cliapp.OperationContext, r *walkv1.StateResponse) cliapp.ListReport {
		lines := []string{}
		if r.Briefing != nil {
			lines = append(lines, "Briefing "+r.Briefing.EntryId+": "+r.Briefing.Body)
		}
		if r.Checkpoint != nil {
			lines = append(lines, "Checkpoint "+r.Checkpoint.EntryId+": "+r.Checkpoint.Body)
		}
		return cliapp.ListReport{Summary: []string{"Latest owned walk records"}, Results: lines}
	}
	return map[string]cliapp.PrimitiveHandler{"WalkService.Read": cliapp.ProtoList(read, report), "WalkService.State": cliapp.ProtoList(state, stateReport), "WalkService.Publish": cliapp.ProtoMutation(publish, receipt), "WalkService.Checkpoint": cliapp.ProtoMutation(checkpoint, receipt)}
}
