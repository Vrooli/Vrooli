package administration

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"golang.org/x/crypto/bcrypt"
)

// ProfileConnectHandler exposes the administrator identity settings through
// the generated contract. It deliberately shares the same credential policy
// and session store as the JSON compatibility edge.
type ProfileConnectHandler struct{ deps ProfileDependencies }

func NewProfileConnectHandler(deps ProfileDependencies) *ProfileConnectHandler {
	return &ProfileConnectHandler{deps: deps}
}

func (h *ProfileConnectHandler) GetAdminProfile(ctx context.Context, request *connect.Request[lpbsv1.GetAdminProfileRequest]) (*connect.Response[lpbsv1.GetAdminProfileResponse], error) {
	if request == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("profile request is required"))
	}
	r, _ := connectHTTP(ctx, request.Header())
	email, ok := profileSessionEmail(h.deps, r)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("admin session is not authenticated"))
	}
	profile, err := h.deps.Auth.Profile(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("admin session is not authenticated"))
	}
	if err != nil {
		h.deps.LogError("admin_profile_lookup_failed", map[string]any{"error": err.Error()})
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to load administrator profile"))
	}
	return connect.NewResponse(&lpbsv1.GetAdminProfileResponse{Profile: profileMessage(profileResponse(h.deps, profile.Email, profile.PasswordHash))}), nil
}

func (h *ProfileConnectHandler) UpdateAdminProfile(ctx context.Context, request *connect.Request[lpbsv1.UpdateAdminProfileRequest]) (*connect.Response[lpbsv1.UpdateAdminProfileResponse], error) {
	if request == nil || request.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("profile update request is required"))
	}
	r, w := connectHTTP(ctx, request.Header())
	currentEmail, ok := profileSessionEmail(h.deps, r)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("admin session is not authenticated"))
	}
	currentPassword := strings.TrimSpace(request.Msg.GetCurrentPassword())
	newEmail := strings.TrimSpace(request.Msg.GetNewEmail())
	newPassword := strings.TrimSpace(request.Msg.GetNewPassword())
	if currentPassword == "" || (newEmail == "" && newPassword == "") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("current password and at least one change are required"))
	}
	profile, err := h.deps.Auth.Profile(ctx, currentEmail)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("admin session is not authenticated"))
	}
	if err != nil {
		h.deps.LogError("admin_profile_load_failed", map[string]any{"error": err.Error()})
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to load administrator profile"))
	}
	if bcrypt.CompareHashAndPassword([]byte(profile.PasswordHash), []byte(currentPassword)) != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid credentials"))
	}
	targetEmail, targetHash := profile.Email, profile.PasswordHash
	if newEmail != "" && !strings.EqualFold(newEmail, profile.Email) {
		if err := h.deps.ValidateEmail(newEmail); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid email address"))
		}
		exists, err := h.deps.Auth.EmailInUse(ctx, newEmail, profile.ID)
		if err != nil {
			h.deps.LogError("admin_email_validation_failed", map[string]any{"error": err.Error()})
			return nil, connect.NewError(connect.CodeInternal, errors.New("failed to validate administrator email"))
		}
		if exists {
			return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("email already in use"))
		}
		targetEmail = newEmail
	}
	if newPassword != "" {
		if err := ValidateProfilePassword(newPassword, profile.PasswordHash, h.deps.DefaultPassword()); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			h.deps.LogError("admin_password_hash_failed", map[string]any{"error": err.Error()})
			return nil, connect.NewError(connect.CodeInternal, errors.New("failed to process administrator password"))
		}
		targetHash = string(hash)
	}
	if targetEmail == profile.Email && targetHash == profile.PasswordHash {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no profile changes detected"))
	}
	if err := h.deps.Auth.UpdateProfile(ctx, profile.ID, targetEmail, targetHash); err != nil {
		h.deps.LogError("admin_profile_update_failed", map[string]any{"error": err.Error()})
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to update administrator profile"))
	}
	session, _ := h.deps.Sessions.GetSession(r, sessionName)
	currentSessionID, _ := session.Values["session_id"].(string)
	if targetHash != profile.PasswordHash {
		if affected, err := h.deps.Auth.RevokeOtherSessions(ctx, currentEmail, currentSessionID); err != nil {
			h.deps.LogError("admin_sessions_invalidation_failed", map[string]any{"error": err.Error(), "email": currentEmail})
		} else if affected > 0 {
			h.deps.Log("admin_sessions_invalidated_on_password_change", map[string]any{"level": "info", "email": currentEmail, "sessions_revoked": affected, "security": true})
		}
	}
	session.Values["email"] = targetEmail
	if err := h.deps.Sessions.SaveSession(r, w, session); err != nil {
		h.deps.LogError("session_save_after_profile_update_failed", map[string]any{"error": err.Error()})
	}
	h.deps.Log("admin_profile_updated", map[string]any{"level": "info", "changed_email": targetEmail != profile.Email, "changed_secret": targetHash != profile.PasswordHash})
	response := connect.NewResponse(&lpbsv1.UpdateAdminProfileResponse{Profile: profileMessage(profileResponse(h.deps, targetEmail, targetHash))})
	copyHeaders(response.Header(), w.Header())
	return response, nil
}

func profileMessage(profile ProfileResponse) *lpbsv1.AdminProfile {
	return &lpbsv1.AdminProfile{Email: profile.Email, IsDefaultEmail: profile.IsDefaultEmail, IsDefaultPassword: profile.IsDefaultPassword}
}

func RegisterProfileConnectRoutes(router *mux.Router, deps ProfileDependencies, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	path, generated := lpbsconnect.NewAdminProfileServiceHandler(NewProfileConnectHandler(deps))
	router.PathPrefix(path).Handler(requireAdmin(generated.ServeHTTP))
}

var _ lpbsconnect.AdminProfileServiceHandler = (*ProfileConnectHandler)(nil)
