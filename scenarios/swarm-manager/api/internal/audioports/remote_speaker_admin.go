package audioports

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"swarm-manager/integrations/audiotools"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// RemoteSpeakerAdmin implements SpeakerAdmin by calling audio-tools'
// STTAdminService via the integrations/audiotools adapter.
type RemoteSpeakerAdmin struct {
	remoteBase
}

var _ SpeakerAdmin = (*RemoteSpeakerAdmin)(nil)

func (r *RemoteSpeakerAdmin) GetSpeakerConfig(ctx context.Context) (SpeakerConfig, error) {
	if err := r.ensure(); err != nil {
		return SpeakerConfig{}, err
	}
	req := connect.NewRequest(&sttv1.GetSpeakerConfigRequest{})
	r.attach(ctx, req)
	resp, err := r.Client.STTAdmin.GetSpeakerConfig(ctx, req)
	if err != nil {
		return SpeakerConfig{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return SpeakerConfig{}, errors.New("audiotools: empty get_speaker_config response")
	}
	return speakerConfigFromProto(resp.Msg.Config), nil
}

func (r *RemoteSpeakerAdmin) UpdateSpeakerConfig(ctx context.Context, mask FieldMask, cfg SpeakerConfig) (SpeakerConfig, error) {
	if err := r.ensure(); err != nil {
		return SpeakerConfig{}, err
	}
	if len(mask.Paths) == 0 {
		return SpeakerConfig{}, audiotools.ErrInvalidArgument
	}
	req := connect.NewRequest(&sttv1.UpdateSpeakerConfigRequest{
		UpdateMask: &fieldmaskpb.FieldMask{Paths: mask.Paths},
		Config:     speakerConfigToProto(cfg),
	})
	r.attach(ctx, req)
	resp, err := r.Client.STTAdmin.UpdateSpeakerConfig(ctx, req)
	if err != nil {
		return SpeakerConfig{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return SpeakerConfig{}, errors.New("audiotools: empty update_speaker_config response")
	}
	return speakerConfigFromProto(resp.Msg.Config), nil
}

func (r *RemoteSpeakerAdmin) GetSpeakerStatus(ctx context.Context) (SpeakerStatus, error) {
	if err := r.ensure(); err != nil {
		return SpeakerStatus{}, err
	}
	req := connect.NewRequest(&sttv1.GetSpeakerStatusRequest{})
	r.attach(ctx, req)
	resp, err := r.Client.STTAdmin.GetSpeakerStatus(ctx, req)
	if err != nil {
		return SpeakerStatus{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return SpeakerStatus{}, errors.New("audiotools: empty get_speaker_status response")
	}
	return speakerStatusFromProto(resp.Msg.Status), nil
}

func (r *RemoteSpeakerAdmin) ListSpeakerProfiles(ctx context.Context) ([]SpeakerProfile, error) {
	if err := r.ensure(); err != nil {
		return nil, err
	}
	req := connect.NewRequest(&sttv1.ListSpeakerProfilesRequest{})
	r.attach(ctx, req)
	resp, err := r.Client.STTAdmin.ListSpeakerProfiles(ctx, req)
	if err != nil {
		return nil, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return nil, nil
	}
	out := make([]SpeakerProfile, 0, len(resp.Msg.Profiles))
	for _, p := range resp.Msg.Profiles {
		out = append(out, speakerProfileFromProto(p))
	}
	return out, nil
}

func (r *RemoteSpeakerAdmin) EnrollSpeakerProfile(ctx context.Context, in EnrollSpeakerInput) (SpeakerEnrollResult, error) {
	if err := r.ensure(); err != nil {
		return SpeakerEnrollResult{}, err
	}
	pbReq := &sttv1.EnrollSpeakerProfileRequest{
		Audio:       in.Audio,
		Format:      in.Format.toProto(),
		ProfileId:   in.ProfileID,
		DisplayName: in.DisplayName,
		Notes:       in.Notes,
	}
	if in.AddToActive != nil {
		v := *in.AddToActive
		pbReq.AddToActive = &v
	}
	if in.Enable != nil {
		v := *in.Enable
		pbReq.Enable = &v
	}
	req := connect.NewRequest(pbReq)
	r.attach(ctx, req)
	resp, err := r.Client.STTAdmin.EnrollSpeakerProfile(ctx, req)
	if err != nil {
		return SpeakerEnrollResult{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return SpeakerEnrollResult{}, errors.New("audiotools: empty enroll_speaker_profile response")
	}
	return SpeakerEnrollResult{
		Enrollment: speakerEnrollmentFromProto(resp.Msg.Enrollment),
		Config:     speakerConfigFromProto(resp.Msg.Config),
	}, nil
}

func (r *RemoteSpeakerAdmin) ClearSpeakerProfileBinding(ctx context.Context) (SpeakerConfig, error) {
	if err := r.ensure(); err != nil {
		return SpeakerConfig{}, err
	}
	req := connect.NewRequest(&sttv1.ClearSpeakerProfileBindingRequest{})
	r.attach(ctx, req)
	resp, err := r.Client.STTAdmin.ClearSpeakerProfileBinding(ctx, req)
	if err != nil {
		return SpeakerConfig{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return SpeakerConfig{}, errors.New("audiotools: empty clear_speaker_profile_binding response")
	}
	return speakerConfigFromProto(resp.Msg.Config), nil
}

func (r *RemoteSpeakerAdmin) UnbindSpeakerProfile(ctx context.Context, profileID string) (SpeakerConfig, error) {
	if err := r.ensure(); err != nil {
		return SpeakerConfig{}, err
	}
	if profileID == "" {
		return SpeakerConfig{}, audiotools.ErrInvalidArgument
	}
	req := connect.NewRequest(&sttv1.UnbindSpeakerProfileRequest{ProfileId: profileID})
	r.attach(ctx, req)
	resp, err := r.Client.STTAdmin.UnbindSpeakerProfile(ctx, req)
	if err != nil {
		return SpeakerConfig{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return SpeakerConfig{}, errors.New("audiotools: empty unbind_speaker_profile response")
	}
	return speakerConfigFromProto(resp.Msg.Config), nil
}

func (r *RemoteSpeakerAdmin) DeleteSpeakerProfile(ctx context.Context, profileID string) (SpeakerConfig, error) {
	if err := r.ensure(); err != nil {
		return SpeakerConfig{}, err
	}
	if profileID == "" {
		return SpeakerConfig{}, audiotools.ErrInvalidArgument
	}
	req := connect.NewRequest(&sttv1.DeleteSpeakerProfileRequest{ProfileId: profileID})
	r.attach(ctx, req)
	resp, err := r.Client.STTAdmin.DeleteSpeakerProfile(ctx, req)
	if err != nil {
		return SpeakerConfig{}, r.handleErr(err)
	}
	if resp == nil || resp.Msg == nil {
		return SpeakerConfig{}, errors.New("audiotools: empty delete_speaker_profile response")
	}
	return speakerConfigFromProto(resp.Msg.Config), nil
}
