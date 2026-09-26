package audioports

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// RemoteWakeWordAdmin implements WakeWordAdmin against audio-tools'
// STTAdminService.
type RemoteWakeWordAdmin struct {
	remoteBase
}

var _ WakeWordAdmin = (*RemoteWakeWordAdmin)(nil)

func (r *RemoteWakeWordAdmin) GetWakeWordConfig(ctx context.Context) (WakeWordConfig, error) {
	if err := r.ensure(); err != nil {
		return WakeWordConfig{}, err
	}
	req := connect.NewRequest(&sttv1.GetWakeWordConfigRequest{})
	r.attach(ctx, req)
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
	r.attach(ctx, req)
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
	r.attach(ctx, req)
	resp, err := r.Client.STTAdmin.DeleteWakeWordTemplate(ctx, req)
	if err != nil {
		return WakeWordConfig{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return WakeWordConfig{}, errors.New("audiotools: empty delete_wake_word_template response")
	}
	return wakeWordConfigFromProto(resp.Msg.Config), nil
}
