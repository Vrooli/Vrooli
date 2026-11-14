package audioports

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"web-console/integrations/audiotools"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// RemoteTTSConfigAdmin implements TTSConfigAdmin against audio-tools'
// TTSService.GetConfig/UpdateConfig.
type RemoteTTSConfigAdmin struct {
	Client      *audiotools.Client
	Credentials func(ctx context.Context) audiotools.Credentials
}

var _ TTSConfigAdmin = (*RemoteTTSConfigAdmin)(nil)

func (r *RemoteTTSConfigAdmin) ensure() error {
	if r == nil || r.Client == nil {
		return audiotools.ErrUnavailable
	}
	if err := r.Client.Ensure(); err != nil {
		return audiotools.ErrUnavailable
	}
	return nil
}

func (r *RemoteTTSConfigAdmin) handleErr(err error) error {
	if err == nil {
		return nil
	}
	if isTransportFailure(err) {
		r.Client.HandleTransportFailure()
	}
	return audiotools.NormalizeError(err)
}

func (r *RemoteTTSConfigAdmin) attach(req connect.AnyRequest, ctx context.Context) {
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

func (r *RemoteTTSConfigAdmin) GetTTSConfig(ctx context.Context) (TTSConfig, error) {
	if err := r.ensure(); err != nil {
		return TTSConfig{}, err
	}
	req := connect.NewRequest(&ttsv1.GetConfigRequest{})
	r.attach(req, ctx)
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
	r.attach(req, ctx)
	resp, err := r.Client.TTS.UpdateConfig(ctx, req)
	if err != nil {
		return TTSConfig{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return TTSConfig{}, errors.New("audiotools: empty update_tts_config response")
	}
	return ttsConfigFromProto(resp.Msg.Config), nil
}
