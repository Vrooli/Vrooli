package session_profiles

import (
	"context"
	"encoding/json"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	sessionprofilesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/session_profiles"
)

type service struct {
	deps Deps
}

// =============================================================================
// List
// =============================================================================

func (s *service) List(
	_ context.Context,
	_ *connect.Request[sessionprofilesv1.ListSessionProfilesRequest],
) (*connect.Response[sessionprofilesv1.ListSessionProfilesResponse], error) {
	profiles, err := s.deps.Repo.ListProfiles()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*sessionprofilesv1.SessionProfile, 0, len(profiles))
	for i := range profiles {
		out = append(out, toProto(&profiles[i]))
	}
	return connect.NewResponse(&sessionprofilesv1.ListSessionProfilesResponse{Profiles: out}), nil
}

// =============================================================================
// Create
// =============================================================================

func (s *service) Create(
	_ context.Context,
	req *connect.Request[sessionprofilesv1.CreateSessionProfileRequest],
) (*connect.Response[sessionprofilesv1.SessionProfileResponse], error) {
	profile, err := s.deps.Repo.CreateProfile(req.Msg.GetName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&sessionprofilesv1.SessionProfileResponse{Profile: toProto(profile)}), nil
}

// =============================================================================
// Update
// =============================================================================

func (s *service) Update(
	_ context.Context,
	req *connect.Request[sessionprofilesv1.UpdateSessionProfileRequest],
) (*connect.Response[sessionprofilesv1.SessionProfileResponse], error) {
	msg := req.Msg
	if _, err := uuid.Parse(msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidProfileID)
	}

	hasName := msg.Name != nil && strings.TrimSpace(msg.GetName()) != ""
	hasBrowserProfile := msg.GetBrowserProfile() != nil
	if !hasName && !hasBrowserProfile {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNothingToUpdate)
	}

	id := sessionprofilepersistence.ProfileID(msg.GetId())
	var (
		profile *sessionprofilepersistence.SessionProfile
		err     error
	)

	if hasName {
		profile, err = s.deps.Repo.RenameProfile(id, msg.GetName())
		if err != nil {
			return nil, mapStoreError(err)
		}
	}

	if hasBrowserProfile {
		bp, convErr := structToBrowserProfile(msg.GetBrowserProfile())
		if convErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidBrowser)
		}
		profile, err = s.deps.Repo.UpdateBrowserProfile(id, bp)
		if err != nil {
			return nil, mapStoreError(err)
		}
	}

	return connect.NewResponse(&sessionprofilesv1.SessionProfileResponse{Profile: toProto(profile)}), nil
}

// =============================================================================
// Delete
// =============================================================================

func (s *service) Delete(
	_ context.Context,
	req *connect.Request[sessionprofilesv1.DeleteSessionProfileRequest],
) (*connect.Response[sessionprofilesv1.DeleteSessionProfileResponse], error) {
	if _, err := uuid.Parse(req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidProfileID)
	}
	if err := s.deps.Repo.DeleteProfile(sessionprofilepersistence.ProfileID(req.Msg.GetId())); err != nil {
		return nil, mapStoreError(err)
	}
	return connect.NewResponse(&sessionprofilesv1.DeleteSessionProfileResponse{Id: req.Msg.GetId()}), nil
}

// =============================================================================
// Helpers
// =============================================================================

func toProto(p *sessionprofilepersistence.SessionProfile) *sessionprofilesv1.SessionProfile {
	if p == nil {
		return nil
	}
	out := &sessionprofilesv1.SessionProfile{
		Id:              string(p.ID),
		Name:            p.Name,
		CreatedAt:       timestamppb.New(p.CreatedAt),
		UpdatedAt:       timestamppb.New(p.UpdatedAt),
		LastUsedAt:      timestamppb.New(p.LastUsedAt),
		HasStorageState: hasActualStorage(p.StorageState),
	}
	if p.BrowserProfile != nil {
		if s, err := browserProfileToStruct(p.BrowserProfile); err == nil {
			out.BrowserProfile = s
		}
	}
	return out
}

// hasActualStorage returns true when storage_state holds cookies or origins.
// Cleared storage materializes as `{"cookies":[],"origins":[]}` — bytes but
// no content.
func hasActualStorage(storageState []byte) bool {
	if len(storageState) == 0 {
		return false
	}
	var state struct {
		Cookies []json.RawMessage `json:"cookies"`
		Origins []json.RawMessage `json:"origins"`
	}
	if err := json.Unmarshal(storageState, &state); err != nil {
		return true
	}
	return len(state.Cookies) > 0 || len(state.Origins) > 0
}

func browserProfileToStruct(bp *sessionprofilepersistence.BrowserProfile) (*structpb.Struct, error) {
	raw, err := json.Marshal(bp)
	if err != nil {
		return nil, err
	}
	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		return nil, err
	}
	return structpb.NewStruct(asMap)
}

func structToBrowserProfile(s *structpb.Struct) (*sessionprofilepersistence.BrowserProfile, error) {
	if s == nil {
		return nil, nil
	}
	raw, err := json.Marshal(s.AsMap())
	if err != nil {
		return nil, err
	}
	var bp sessionprofilepersistence.BrowserProfile
	if err := json.Unmarshal(raw, &bp); err != nil {
		return nil, err
	}
	return &bp, nil
}

func mapStoreError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not found"):
		return connect.NewError(connect.CodeNotFound, errProfileNotFound)
	case strings.Contains(msg, "invalid browser profile"):
		return connect.NewError(connect.CodeInvalidArgument, errInvalidBrowser)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
