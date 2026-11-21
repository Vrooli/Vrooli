package control

import (
	"context"
	"strings"
	"time"

	authdomain "device-control/internal/auth"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	authv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/auth"
	authconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/auth/auth_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func registerAuthConnectService(r *mux.Router, h *handler) {
	path, service := authconnect.NewAuthenticationServiceHandler(&authenticationConnect{h})
	connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: service})
}

type authenticationConnect struct{ h *handler }

func (c *authenticationConnect) ListProfiles(ctx context.Context, _ *connect.Request[authv1.ListProfilesRequest]) (*connect.Response[authv1.ListProfilesResponse], error) {
	profiles := c.h.service.AuthProfiles(ctx)
	out := make([]*authv1.AuthenticationProfile, 0, len(profiles))
	for _, profile := range profiles {
		out = append(out, authProfileProto(profile))
	}
	return connect.NewResponse(&authv1.ListProfilesResponse{Profiles: out}), nil
}

func (c *authenticationConnect) GetProfile(ctx context.Context, req *connect.Request[authv1.GetProfileRequest]) (*connect.Response[authv1.GetProfileResponse], error) {
	profile, provider, err := c.h.service.AuthProfileStatus(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&authv1.GetProfileResponse{Profile: authProfileProto(profile), Provider: providerProto(provider)}), nil
}

func (c *authenticationConnect) CreateProfile(ctx context.Context, req *connect.Request[authv1.CreateProfileRequest]) (*connect.Response[authv1.ProfileResponse], error) {
	if req.Msg.Profile == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidProfile)
	}
	profile, err := c.h.service.CreateAuthProfile(ctx, profileFromProto(req.Msg.Profile), req.Msg.Actor)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&authv1.ProfileResponse{Profile: authProfileProto(profile)}), nil
}

func (c *authenticationConnect) UpdateProfile(ctx context.Context, req *connect.Request[authv1.UpdateProfileRequest]) (*connect.Response[authv1.ProfileResponse], error) {
	if req.Msg.Profile == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidProfile)
	}
	profile, err := c.h.service.UpdateAuthProfile(ctx, req.Msg.Id, profileFromProto(req.Msg.Profile), req.Msg.Actor)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&authv1.ProfileResponse{Profile: authProfileProto(profile)}), nil
}

func (c *authenticationConnect) RevokeProfile(ctx context.Context, req *connect.Request[authv1.RevokeProfileRequest]) (*connect.Response[authv1.ProfileResponse], error) {
	profile, err := c.h.service.RevokeAuthProfile(ctx, req.Msg.Id, req.Msg.Actor)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&authv1.ProfileResponse{Profile: authProfileProto(profile)}), nil
}

func (c *authenticationConnect) TestProfile(ctx context.Context, req *connect.Request[authv1.TestProfileRequest]) (*connect.Response[authv1.ProfileResponse], error) {
	profile, provider, err := c.h.service.AuthProfileStatus(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&authv1.ProfileResponse{Profile: authProfileProto(profile), Provider: providerProto(provider)}), nil
}

func (c *authenticationConnect) UnlockDevice(ctx context.Context, req *connect.Request[authv1.UnlockDeviceRequest]) (*connect.Response[authv1.UnlockDeviceResponse], error) {
	result, err := c.h.service.UnlockDevice(ctx, req.Msg.ProfileId, req.Msg.DeviceId, req.Msg.Actor, req.Msg.LeaseToken)
	if err != nil {
		if strings.Contains(err.Error(), "lease") {
			return nil, connect.NewError(connect.CodePermissionDenied, internalError(safeUnlockError(err)))
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, internalError(safeUnlockError(err)))
	}
	return connect.NewResponse(&authv1.UnlockDeviceResponse{Result: unlockResultProto(result)}), nil
}

func authProfileProto(profile authdomain.Profile) *authv1.AuthenticationProfile {
	out := &authv1.AuthenticationProfile{Id: profile.ID, DeviceId: profile.DeviceID, Method: profile.Method, CredentialIdentity: profile.CredentialIdentity, CredentialField: profile.CredentialField, Verification: profile.Verification, Policy: &authv1.UnlockPolicy{MaxAttempts: safeInt32(profile.Policy.MaxAttempts), AttemptLimitMs: profile.Policy.AttemptLimit.Milliseconds(), SettleMs: profile.Policy.Settle.Milliseconds()}, Status: profile.Status, LastOutcome: profile.LastOutcome, CreatedAt: timestamppb.New(profile.CreatedAt), UpdatedAt: timestamppb.New(profile.UpdatedAt)}
	if !profile.RevokedAt.IsZero() {
		out.RevokedAt = timestamppb.New(profile.RevokedAt)
	}
	return out
}

func profileFromProto(profile *authv1.AuthenticationProfile) authdomain.Profile {
	out := authdomain.Profile{ID: profile.Id, DeviceID: profile.DeviceId, Method: profile.Method, CredentialIdentity: profile.CredentialIdentity, CredentialField: profile.CredentialField, Verification: profile.Verification, Status: profile.Status}
	if profile.Policy != nil {
		out.Policy = authdomain.Policy{MaxAttempts: int(profile.Policy.MaxAttempts), AttemptLimit: time.Duration(profile.Policy.AttemptLimitMs) * time.Millisecond, Settle: time.Duration(profile.Policy.SettleMs) * time.Millisecond}
	}
	return out
}

func providerProto(provider authdomain.ProviderStatus) *authv1.ProviderStatus {
	return &authv1.ProviderStatus{Provider: provider.Provider, ProviderState: provider.ProviderState, Configured: provider.Configured, ProviderDetail: provider.Detail}
}

func unlockResultProto(result authdomain.UnlockResponse) *authv1.UnlockResult {
	return &authv1.UnlockResult{ProfileId: result.ProfileID, DeviceId: result.DeviceID, Method: result.Method, Outcome: result.Outcome, NextAction: result.NextAction, Attempts: safeInt32(result.Attempts), ProviderState: result.ProviderState, BeforeLockState: result.BeforeLockState, AfterLockState: result.AfterLockState}
}

func safeInt32(value int) int32 {
	if value <= 0 {
		return 0
	}
	if int64(value) > int64(^uint32(0)>>1) {
		return int32(^uint32(0) >> 1)
	}
	return int32(value) // #nosec G115 -- value is range-checked above
}

var errInvalidProfile = internalError("authentication profile is required")

type internalError string

func (e internalError) Error() string { return string(e) }
