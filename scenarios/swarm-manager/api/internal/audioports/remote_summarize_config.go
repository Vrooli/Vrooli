package audioports

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"swarm-manager/integrations/audiotools"

	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// RemoteSummarizeConfigAdmin implements SummarizeConfigAdmin against
// audio-tools' SummarizeService.
type RemoteSummarizeConfigAdmin struct {
	remoteBase
}

var _ SummarizeConfigAdmin = (*RemoteSummarizeConfigAdmin)(nil)

func (r *RemoteSummarizeConfigAdmin) GetSummarizeConfig(ctx context.Context) (SummarizeConfig, error) {
	if err := r.ensure(); err != nil {
		return SummarizeConfig{}, err
	}
	req := connect.NewRequest(&summv1.GetSummarizeConfigRequest{})
	r.attach(ctx, req)
	resp, err := r.Client.Summarize.GetSummarizeConfig(ctx, req)
	if err != nil {
		return SummarizeConfig{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return SummarizeConfig{}, errors.New("audiotools: empty get_summarize_config response")
	}
	return summarizeConfigFromProto(resp.Msg.Config), nil
}

func (r *RemoteSummarizeConfigAdmin) UpdateSummarizeConfig(ctx context.Context, mask FieldMask, cfg SummarizeConfig) (SummarizeConfig, error) {
	if err := r.ensure(); err != nil {
		return SummarizeConfig{}, err
	}
	if len(mask.Paths) == 0 {
		return SummarizeConfig{}, audiotools.ErrInvalidArgument
	}
	req := connect.NewRequest(&summv1.UpdateSummarizeConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: mask.Paths},
		Config:     summarizeConfigToProto(cfg),
	})
	r.attach(ctx, req)
	resp, err := r.Client.Summarize.UpdateSummarizeConfig(ctx, req)
	if err != nil {
		return SummarizeConfig{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return SummarizeConfig{}, errors.New("audiotools: empty update_summarize_config response")
	}
	return summarizeConfigFromProto(resp.Msg.Config), nil
}

func (r *RemoteSummarizeConfigAdmin) ListSummarizeModels(ctx context.Context) ([]SummarizeModel, error) {
	if err := r.ensure(); err != nil {
		return nil, err
	}
	req := connect.NewRequest(&summv1.ListSummarizeModelsRequest{})
	r.attach(ctx, req)
	resp, err := r.Client.Summarize.ListSummarizeModels(ctx, req)
	if err != nil {
		return nil, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return nil, errors.New("audiotools: empty list_summarize_models response")
	}
	out := make([]SummarizeModel, 0, len(resp.Msg.Models))
	for _, model := range resp.Msg.Models {
		out = append(out, summarizeModelFromProto(model))
	}
	return out, nil
}
