// Package profiles exposes deployment profile management over Connect-RPC.
package profileshandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	profilesdomain "deployment-manager/profiles"
	"github.com/google/uuid"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	profilesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/profiles"
	profilesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/profiles/profilesv1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ConnectHandler is the typed transport adapter for the profiles repository.
// Domain persistence remains behind profilesdomain.Repository; no JSON request
// structs are used at the transport boundary.
type ConnectHandler struct {
	profilesconnect.UnimplementedProfilesServiceHandler
	repo profilesdomain.Repository
}

func NewConnectHandler(repo profilesdomain.Repository) *ConnectHandler {
	return &ConnectHandler{repo: repo}
}

func (h *ConnectHandler) ListProfiles(ctx context.Context, req *connect.Request[profilesv1.ListProfilesRequest]) (*connect.Response[profilesv1.ListProfilesResponse], error) {
	profiles, err := h.repo.List(ctx)
	if err != nil {
		return nil, internalError("list profiles", err)
	}
	offset, pageSize, err := pageWindow(req)
	if err != nil {
		return nil, err
	}
	if offset > len(profiles) {
		offset = len(profiles)
	}
	end := len(profiles)
	if pageSize > 0 && offset+pageSize < end {
		end = offset + pageSize
	}
	response := &profilesv1.ListProfilesResponse{Profiles: make([]*profilesv1.Profile, 0, end-offset)}
	for _, profile := range profiles[offset:end] {
		response.Profiles = append(response.Profiles, profileToProto(profile))
	}
	if end < len(profiles) {
		response.NextPageToken = strconv.Itoa(end)
	}
	return connect.NewResponse(response), nil
}

func (h *ConnectHandler) GetProfile(ctx context.Context, req *connect.Request[profilesv1.GetProfileRequest]) (*connect.Response[profilesv1.GetProfileResponse], error) {
	id, err := requiredID(req, func(r *profilesv1.GetProfileRequest) string { return r.GetProfileId() })
	if err != nil {
		return nil, err
	}
	profile, err := h.repo.Get(ctx, id)
	if err != nil {
		return nil, profileError("get profile", err)
	}
	if profile == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("profile %q not found", id))
	}
	return connect.NewResponse(&profilesv1.GetProfileResponse{Profile: profileToProto(*profile)}), nil
}

func (h *ConnectHandler) CreateProfile(ctx context.Context, req *connect.Request[profilesv1.CreateProfileRequest]) (*connect.Response[profilesv1.CreateProfileResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if strings.TrimSpace(req.Msg.GetName()) == "" || strings.TrimSpace(req.Msg.GetScenario()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name and scenario are required"))
	}
	profile := &profilesdomain.Profile{
		ID:       "profile-" + uuid.NewString(),
		Name:     strings.TrimSpace(req.Msg.GetName()),
		Scenario: strings.TrimSpace(req.Msg.GetScenario()),
		Tiers:    int32SliceToJSON(req.Msg.GetTiers()),
		Swaps:    jsonObjectToValue(req.Msg.GetSwaps()),
		Secrets:  jsonObjectToValue(req.Msg.GetSecrets()),
		Settings: jsonObjectToValue(req.Msg.GetSettings()),
	}
	profilesdomain.ApplyDefaults(profile)
	profile.Version = 1
	profileID, err := h.repo.Create(ctx, profile)
	if err != nil {
		return nil, internalError("create profile", err)
	}
	if strings.TrimSpace(profileID) != "" {
		profile.ID = profileID
	}
	created, err := h.repo.Get(ctx, profile.ID)
	if err != nil {
		return nil, internalError("read created profile", err)
	}
	if created == nil {
		created = profile
	}
	return connect.NewResponse(&profilesv1.CreateProfileResponse{Profile: profileToProto(*created)}), nil
}

func (h *ConnectHandler) UpdateProfile(ctx context.Context, req *connect.Request[profilesv1.UpdateProfileRequest]) (*connect.Response[profilesv1.UpdateProfileResponse], error) {
	id, err := requiredID(req, func(r *profilesv1.UpdateProfileRequest) string { return r.GetProfileId() })
	if err != nil {
		return nil, err
	}
	updates := make(map[string]interface{})
	if req.Msg.Name != nil {
		updates["name"] = req.Msg.GetName()
	}
	if req.Msg.Scenario != nil {
		updates["scenario"] = req.Msg.GetScenario()
	}
	if req.Msg.Tiers != nil {
		updates["tiers"] = int32SliceToJSON(req.Msg.GetTiers())
	}
	if req.Msg.Swaps != nil {
		updates["swaps"] = jsonObjectToValue(req.Msg.GetSwaps())
	}
	if req.Msg.Secrets != nil {
		updates["secrets"] = jsonObjectToValue(req.Msg.GetSecrets())
	}
	if req.Msg.Settings != nil {
		updates["settings"] = jsonObjectToValue(req.Msg.GetSettings())
	}
	profile, err := h.repo.Update(ctx, id, updates)
	if err != nil {
		return nil, profileError("update profile", err)
	}
	if profile == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("profile %q not found", id))
	}
	return connect.NewResponse(&profilesv1.UpdateProfileResponse{Profile: profileToProto(*profile)}), nil
}

func (h *ConnectHandler) DeleteProfile(ctx context.Context, req *connect.Request[profilesv1.DeleteProfileRequest]) (*connect.Response[profilesv1.DeleteProfileResponse], error) {
	id, err := requiredID(req, func(r *profilesv1.DeleteProfileRequest) string { return r.GetProfileId() })
	if err != nil {
		return nil, err
	}
	deleted, err := h.repo.Delete(ctx, id)
	if err != nil {
		return nil, internalError("delete profile", err)
	}
	if !deleted {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("profile %q not found", id))
	}
	return connect.NewResponse(&profilesv1.DeleteProfileResponse{ProfileId: id}), nil
}

func (h *ConnectHandler) ListProfileVersions(ctx context.Context, req *connect.Request[profilesv1.ListProfileVersionsRequest]) (*connect.Response[profilesv1.ListProfileVersionsResponse], error) {
	id, err := requiredID(req, func(r *profilesv1.ListProfileVersionsRequest) string { return r.GetProfileId() })
	if err != nil {
		return nil, err
	}
	versions, err := h.repo.GetVersions(ctx, id)
	if err != nil {
		return nil, profileError("list profile versions", err)
	}
	offset, pageSize, err := pageWindowVersions(req)
	if err != nil {
		return nil, err
	}
	if offset > len(versions) {
		offset = len(versions)
	}
	end := len(versions)
	if pageSize > 0 && offset+pageSize < end {
		end = offset + pageSize
	}
	response := &profilesv1.ListProfileVersionsResponse{Versions: make([]*profilesv1.ProfileVersion, 0, end-offset)}
	for _, version := range versions[offset:end] {
		response.Versions = append(response.Versions, versionToProto(version))
	}
	if end < len(versions) {
		response.NextPageToken = strconv.Itoa(end)
	}
	return connect.NewResponse(response), nil
}

func profileToProto(profile profilesdomain.Profile) *profilesv1.Profile {
	return &profilesv1.Profile{
		Id:        profile.ID,
		Name:      profile.Name,
		Scenario:  profile.Scenario,
		Tiers:     jsonToInt32Slice(profile.Tiers),
		Swaps:     valueToJSONObj(profile.Swaps),
		Secrets:   valueToJSONObj(profile.Secrets),
		Settings:  valueToJSONObj(profile.Settings),
		Version:   int32(profile.Version),
		CreatedAt: timestamppb.New(profile.CreatedAt),
		UpdatedAt: timestamppb.New(profile.UpdatedAt),
		CreatedBy: profile.CreatedBy,
		UpdatedBy: profile.UpdatedBy,
	}
}

func versionToProto(version profilesdomain.Version) *profilesv1.ProfileVersion {
	return &profilesv1.ProfileVersion{
		ProfileId:         version.ProfileID,
		Version:           int32(version.Version),
		Name:              version.Name,
		Scenario:          version.Scenario,
		Tiers:             jsonToInt32Slice(version.Tiers),
		Swaps:             valueToJSONObj(version.Swaps),
		Secrets:           valueToJSONObj(version.Secrets),
		Settings:          valueToJSONObj(version.Settings),
		CreatedAt:         timestamppb.New(version.CreatedAt),
		CreatedBy:         version.CreatedBy,
		ChangeDescription: version.ChangeDescription,
	}
}

func valueToJSONObj(value interface{}) *commonv1.JsonObject {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	result := &commonv1.JsonObject{}
	if err := protojson.Unmarshal(raw, result); err != nil {
		return nil
	}
	return result
}

func jsonObjectToValue(value *commonv1.JsonObject) interface{} {
	if value == nil {
		return nil
	}
	raw, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(value)
	if err != nil {
		return nil
	}
	var result interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return result
}

func int32SliceToJSON(values []int32) []interface{} {
	result := make([]interface{}, len(values))
	for i, value := range values {
		result[i] = value
	}
	return result
}

func jsonToInt32Slice(value interface{}) []int32 {
	values, ok := value.([]interface{})
	if !ok {
		return nil
	}
	result := make([]int32, 0, len(values))
	for _, value := range values {
		switch number := value.(type) {
		case float64:
			result = append(result, int32(number))
		case int:
			result = append(result, int32(number))
		case int32:
			result = append(result, number)
		}
	}
	return result
}

func pageWindow(req *connect.Request[profilesv1.ListProfilesRequest]) (int, int, error) {
	if req == nil || req.Msg == nil {
		return 0, 100, nil
	}
	return parsePage(req.Msg.GetPageToken(), req.Msg.GetPageSize())
}

func pageWindowVersions(req *connect.Request[profilesv1.ListProfileVersionsRequest]) (int, int, error) {
	if req == nil || req.Msg == nil {
		return 0, 100, nil
	}
	return parsePage(req.Msg.GetPageToken(), req.Msg.GetPageSize())
}

func parsePage(token string, pageSize int32) (int, int, error) {
	offset := 0
	if strings.TrimSpace(token) != "" {
		parsed, err := strconv.Atoi(token)
		if err != nil || parsed < 0 {
			return 0, 0, connect.NewError(connect.CodeInvalidArgument, errors.New("page_token must be a non-negative offset"))
		}
		offset = parsed
	}
	if pageSize < 0 {
		return 0, 0, connect.NewError(connect.CodeInvalidArgument, errors.New("page_size must not be negative"))
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return offset, int(pageSize), nil
}

func requiredID[T any](req *connect.Request[T], get func(*T) string) (string, error) {
	if req == nil || req.Msg == nil || strings.TrimSpace(get(req.Msg)) == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("profile_id is required"))
	}
	return strings.TrimSpace(get(req.Msg)), nil
}

func profileError(operation string, err error) error {
	if errors.Is(err, profilesdomain.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("%s: profile not found", operation))
	}
	return internalError(operation, err)
}

func internalError(operation string, err error) error {
	return connect.NewError(connect.CodeInternal, fmt.Errorf("%s: %w", operation, err))
}
