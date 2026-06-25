package audioports

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"swarm-manager/integrations/audiotools"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// RemoteStreamConfigAdmin implements StreamConfigAdmin against audio-tools'
// STTAdminService.
type RemoteStreamConfigAdmin struct {
	Client      *audiotools.Client
	Credentials func(ctx context.Context) audiotools.Credentials
}

var _ StreamConfigAdmin = (*RemoteStreamConfigAdmin)(nil)

func (r *RemoteStreamConfigAdmin) ensure() error {
	if r == nil || r.Client == nil {
		return audiotools.ErrUnavailable
	}
	if err := r.Client.Ensure(); err != nil {
		return audiotools.ErrUnavailable
	}
	return nil
}

func (r *RemoteStreamConfigAdmin) handleErr(err error) error {
	if err == nil {
		return nil
	}
	if isTransportFailure(err) {
		r.Client.HandleTransportFailure()
	}
	return audiotools.NormalizeError(err)
}

func (r *RemoteStreamConfigAdmin) attach(ctx context.Context, req connect.AnyRequest) {
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

func (r *RemoteStreamConfigAdmin) GetStreamConfig(ctx context.Context) (StreamConfig, error) {
	if err := r.ensure(); err != nil {
		return StreamConfig{}, err
	}
	req := connect.NewRequest(&sttv1.GetStreamConfigRequest{})
	r.attach(ctx, req)
	resp, err := r.Client.STTAdmin.GetStreamConfig(ctx, req)
	if err != nil {
		return StreamConfig{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return StreamConfig{}, errors.New("audiotools: empty get_stream_config response")
	}
	return streamConfigFromProto(resp.Msg.Config), nil
}

func (r *RemoteStreamConfigAdmin) UpdateStreamConfig(ctx context.Context, mask FieldMask, cfg StreamConfig) (StreamConfig, error) {
	if err := r.ensure(); err != nil {
		return StreamConfig{}, err
	}
	if len(mask.Paths) == 0 {
		return StreamConfig{}, audiotools.ErrInvalidArgument
	}
	req := connect.NewRequest(&sttv1.UpdateStreamConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: mask.Paths},
		Config:     streamConfigToProto(cfg),
	})
	r.attach(ctx, req)
	resp, err := r.Client.STTAdmin.UpdateStreamConfig(ctx, req)
	if err != nil {
		return StreamConfig{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return StreamConfig{}, errors.New("audiotools: empty update_stream_config response")
	}
	return streamConfigFromProto(resp.Msg.Config), nil
}
