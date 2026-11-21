package audioports

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"web-console/integrations/audiotools"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// RemoteWakeWordAdmin implements WakeWordAdmin against audio-tools'
// STTAdminService.
type RemoteWakeWordAdmin struct {
	Client      *audiotools.Client
	Credentials func(ctx context.Context) audiotools.Credentials
}

var _ WakeWordAdmin = (*RemoteWakeWordAdmin)(nil)

func (r *RemoteWakeWordAdmin) ensure() error {
	if r == nil || r.Client == nil {
		return audiotools.ErrUnavailable
	}
	if err := r.Client.Ensure(); err != nil {
		return audiotools.ErrUnavailable
	}
	return nil
}

func (r *RemoteWakeWordAdmin) handleErr(err error) error {
	if err == nil {
		return nil
	}
	if isTransportFailure(err) {
		r.Client.HandleTransportFailure()
	}
	return audiotools.NormalizeError(err)
}

func (r *RemoteWakeWordAdmin) attach(req connect.AnyRequest, ctx context.Context) {
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

func (r *RemoteWakeWordAdmin) GetWakeWordConfig(ctx context.Context) (WakeWordConfig, error) {
	if err := r.ensure(); err != nil {
		return WakeWordConfig{}, err
	}
	req := connect.NewRequest(&sttv1.GetWakeWordConfigRequest{})
	r.attach(req, ctx)
	resp, err := r.Client.STTAdmin.GetWakeWordConfig(ctx, req)
	if err != nil {
		return WakeWordConfig{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return WakeWordConfig{}, errors.New("audiotools: empty get_wake_word_config response")
	}
	return wakeWordConfigFromProto(resp.Msg.Config), nil
}

func (r *RemoteWakeWordAdmin) UpdateWakeWordTemplate(ctx context.Context, t WakeWordTemplate) (WakeWordConfig, error) {
	if err := r.ensure(); err != nil {
		return WakeWordConfig{}, err
	}
	req := connect.NewRequest(&sttv1.UpdateWakeWordTemplateRequest{
		Template: wakeWordTemplateToProto(&t),
	})
	r.attach(req, ctx)
	resp, err := r.Client.STTAdmin.UpdateWakeWordTemplate(ctx, req)
	if err != nil {
		return WakeWordConfig{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return WakeWordConfig{}, errors.New("audiotools: empty update_wake_word_template response")
	}
	return wakeWordConfigFromProto(resp.Msg.Config), nil
}

func (r *RemoteWakeWordAdmin) DeleteWakeWordTemplate(ctx context.Context) (WakeWordConfig, error) {
	if err := r.ensure(); err != nil {
		return WakeWordConfig{}, err
	}
	req := connect.NewRequest(&sttv1.DeleteWakeWordTemplateRequest{})
	r.attach(req, ctx)
	resp, err := r.Client.STTAdmin.DeleteWakeWordTemplate(ctx, req)
	if err != nil {
		return WakeWordConfig{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return WakeWordConfig{}, errors.New("audiotools: empty delete_wake_word_template response")
	}
	return wakeWordConfigFromProto(resp.Msg.Config), nil
}
