package audioports

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"web-console/integrations/audiotools"

	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// RemoteSummarizeConfigAdmin implements SummarizeConfigAdmin against
// audio-tools' SummarizeService.
type RemoteSummarizeConfigAdmin struct {
	Client      *audiotools.Client
	Credentials func(ctx context.Context) audiotools.Credentials
}

var _ SummarizeConfigAdmin = (*RemoteSummarizeConfigAdmin)(nil)

func (r *RemoteSummarizeConfigAdmin) ensure() error {
	if r == nil || r.Client == nil {
		return audiotools.ErrUnavailable
	}
	if err := r.Client.Ensure(); err != nil {
		return audiotools.ErrUnavailable
	}
	return nil
}

func (r *RemoteSummarizeConfigAdmin) handleErr(err error) error {
	if err == nil {
		return nil
	}
	if isTransportFailure(err) {
		r.Client.HandleTransportFailure()
	}
	return audiotools.NormalizeError(err)
}

func (r *RemoteSummarizeConfigAdmin) attach(req connect.AnyRequest, ctx context.Context) {
	if r.Credentials == nil || req == nil {
		return
	}
	creds := r.Credentials(ctx)
	if creds.BYOKKey != "" {
		req.Header().Set("X-Audio-BYOK-Key", creds.BYOKKey)
		req.Header().Set("X-Audio-BYOK-Provider", creds.BYOKProvider)
	}
	if creds.LPBSToken != "" {
		req.Header().Set("X-Audio-LPBS-Token", creds.LPBSToken)
	}
	if creds.UserIdentity != "" {
		req.Header().Set("X-Audio-User-Identity", creds.UserIdentity)
	}
}

func (r *RemoteSummarizeConfigAdmin) GetSummarizeConfig(ctx context.Context) (SummarizeConfig, error) {
	if err := r.ensure(); err != nil {
		return SummarizeConfig{}, err
	}
	req := connect.NewRequest(&summv1.GetSummarizeConfigRequest{})
	r.attach(req, ctx)
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
	r.attach(req, ctx)
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
	r.attach(req, ctx)
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
