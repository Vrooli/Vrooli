package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/adminsecurity"
)

const consumerSigningField = "consumer-auth-private-key"

func consumerSigningIssuer() string {
	return administration.ConsumerSigningIssuer(resolveConfig("JWT_ISSUER"))
}

type consumerSigningRotationRequest struct {
	ExpectedActiveVersion *string `json:"expected_active_version"`
}

// adminConsumerSigningRotation exposes a high-impact, no-input rotation
// workflow. It requires an authenticated administrator plus recent reauth,
// checks the reviewed version, writes an audit intent before mutation, and
// returns key identifiers/version metadata only.
func (s *Server) adminConsumerSigningRotation(w http.ResponseWriter, r *http.Request) {
	var request consumerSigningRotationRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || request.ExpectedActiveVersion == nil {
		writeJSONError(w, http.StatusBadRequest, "A reviewed active credential version is required.", ApiErrorTypeValidation)
		return
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		writeJSONError(w, http.StatusBadRequest, "The request must contain one JSON object.", ApiErrorTypeValidation)
		return
	}

	adminEmail, ok := s.sessionAdminEmail(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "Session expired. Please log in again.", ApiErrorTypeUnauthorized)
		return
	}
	status, authority, err := consumerSigningAuthorityStatus()
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "Consumer signing authority is unavailable.", ApiErrorTypeServerError)
		return
	}
	if !status.Configured || status.Version != *request.ExpectedActiveVersion {
		writeJSONError(w, http.StatusConflict, "Consumer signing credential changed. Refresh its status before retrying.", ApiErrorTypeValidation)
		return
	}

	securityStore := adminsecurity.Store(s.db)
	if s.routedDB != nil {
		securityStore = s.routedDB
	}
	sessionID := ""
	if session, sessionErr := s.sessionManager.GetSession(r, "admin_session"); sessionErr == nil && session != nil {
		sessionID, _ = session.Values["session_id"].(string)
	}
	repository := &adminsecurity.Repository{DB: securityStore}
	if err := repository.Record(r.Context(), "consumer_signing_rotation_requested", adminEmail, sessionID, r.RemoteAddr, r.UserAgent(), map[string]any{
		"identity": string(status.Identity), "field": consumerSigningField,
		"expected_active_version": status.Version,
	}); err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "Rotation audit could not be recorded; no change was made.", ApiErrorTypeServerError)
		return
	}

	sessionConfig, err := resolveUserSessionConfig()
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "Consumer signing policy is unavailable.", ApiErrorTypeServerError)
		return
	}
	result, rotateErr := administration.RotateConsumerSigningCredential(authority, administration.ConsumerSigningRotationOptions{
		Identity: status.Identity, Field: consumerSigningField, Scope: "signing:consumer-token",
		ExpectedVersion: status.Version, LegacyKeyID: resolveConfig("CONSUMER_AUTH_KEY_ID"),
		LegacyPreviousJWKS: resolveConfig("CONSUMER_AUTH_PREVIOUS_JWKS"), Issuer: consumerSigningIssuer(),
		AccessTTL: sessionConfig.AccessTTL, ClockSkew: 30 * time.Second,
		MaxTokenTTL:     entitlementLeaseTTL(),
		JWKSCacheMaxAge: administration.ConsumerSigningJWKSCacheMaxAge, Now: time.Now().UTC(),
	})
	if rotateErr != nil {
		failureClass := "rotation_failed"
		statusCode := http.StatusServiceUnavailable
		errorType := ApiErrorTypeServerError
		if errors.Is(rotateErr, credentialauthority.ErrVersionConflict) {
			failureClass = "version_conflict"
			statusCode = http.StatusConflict
			errorType = ApiErrorTypeValidation
		}
		_ = repository.Record(r.Context(), "consumer_signing_rotation_failed", adminEmail, sessionID, r.RemoteAddr, r.UserAgent(), map[string]any{
			"expected_active_version": status.Version, "failure_class": failureClass,
		})
		writeJSONError(w, statusCode, "Consumer signing rotation did not activate.", errorType)
		return
	}

	result.ApplyTo(s.userAuthService)
	receipt := result.Receipt
	auditRecorded := repository.Record(r.Context(), "consumer_signing_rotation_activated", adminEmail, sessionID, r.RemoteAddr, r.UserAgent(), map[string]any{
		"previous_key_id": receipt.PreviousKeyID, "active_key_id": receipt.ActiveKeyID,
		"authority_version": receipt.Version, "overlap_key_count": receipt.OverlapKeyCount,
		"previous_keys_expire_at":   receipt.PreviousKeysExpireAt.UTC().Format(time.RFC3339),
		"candidate_cleanup_pending": receipt.CleanupPending,
	}) == nil
	writeJSON(w, map[string]any{"rotated": true, "receipt": receipt, "audit_recorded": auditRecorded})
}

func (s *Server) adminConsumerSigningStatus(w http.ResponseWriter, _ *http.Request) {
	status, _, err := consumerSigningAuthorityStatus()
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "Consumer signing authority is unavailable.", ApiErrorTypeServerError)
		return
	}
	writeJSON(w, map[string]any{
		"identity": string(status.Identity), "field": status.Field, "configured": status.Configured,
		"active_version": status.Version, "provider_state": status.ProviderState, "checked_at": status.CheckedAt.UTC().Format(time.RFC3339),
	})
}

func consumerSigningAuthorityStatus() (credentialauthority.Status, *credentialauthority.Authority, error) {
	authority, err := administration.OpenCredentialAuthority()
	if err != nil {
		return credentialauthority.Status{}, nil, err
	}
	identity := credentialauthority.Identity("vrooli/landing-page-business-suite")
	status := authority.Status(identity, consumerSigningField)
	if string(status.ProviderState) != "available" {
		return status, nil, errors.New("credential provider unavailable")
	}
	return status, authority, nil
}
