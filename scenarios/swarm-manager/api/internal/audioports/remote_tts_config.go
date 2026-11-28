package audioports

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"swarm-manager/integrations/audiotools"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// RemoteTTSConfigAdmin implements TTSConfigAdmin against audio-tools'
// TTSService.GetConfig/UpdateConfig.
type RemoteTTSConfigAdmin struct {
	remoteBase
}

var _ TTSConfigAdmin = (*RemoteTTSConfigAdmin)(nil)

func (r *RemoteTTSConfigAdmin) GetTTSConfig(ctx context.Context) (TTSConfig, error) {
	if err := r.ensure(); err != nil {
		return TTSConfig{}, err
	}
	req := connect.NewRequest(&ttsv1.GetConfigRequest{})
	r.attach(ctx, req)
	resp, err := r.Client.TTS.GetConfig(ctx, req)
	if err != nil {
		return TTSConfig{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return TTSConfig{}, errors.New("audiotools: empty get_tts_config response")
	}
	return ttsConfigFromProto(resp.Msg.Config), nil
}

func (r *RemoteTTSConfigAdmin) UpdateTTSConfig(ctx context.Context, mask FieldMask, cfg TTSConfig) (TTSConfig, error) {
	if err := r.ensure(); err != nil {
		return TTSConfig{}, err
	}
	if len(mask.Paths) == 0 {
		return TTSConfig{}, audiotools.ErrInvalidArgument
	}
	req := connect.NewRequest(&ttsv1.UpdateConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: mask.Paths},
		Config:     ttsConfigToProto(cfg),
	})
	r.attach(ctx, req)
	resp, err := r.Client.TTS.UpdateConfig(ctx, req)
	if err != nil {
		return TTSConfig{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return TTSConfig{}, errors.New("audiotools: empty update_tts_config response")
	}
	return ttsConfigFromProto(resp.Msg.Config), nil
}
