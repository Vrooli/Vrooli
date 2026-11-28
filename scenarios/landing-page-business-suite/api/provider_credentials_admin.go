package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite-api/internal/administration"
)

// adminProviderCredentialFields is deliberately closed. The browser may
// write only the provider credentials that this surface documents; it cannot
// use this endpoint to overwrite session, signing, payment, or generated
// credentials.
var adminProviderCredentialFields = map[string]struct{}{
	"mailgun-api-key":             {},
	"smtp-password":               {},
	"sendgrid-api-key":            {},
	"sendgrid-webhook-public-key": {},
}

type adminProviderCredentialRequest struct {
	Field           string `json:"field"`
	Value           string `json:"value"`
	ExpectedVersion string `json:"expected_version"`
}

// adminProviderCredentialWrite accepts a secret only long enough to put it in
// the shared credential authority. The response contains no secret material.
func (s *Server) adminProviderCredentialWrite(w http.ResponseWriter, r *http.Request) {
	var request adminProviderCredentialRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid credential request.", ApiErrorTypeValidation)
		return
	}
	request.Field = strings.ToLower(strings.TrimSpace(request.Field))
	if _, ok := adminProviderCredentialFields[request.Field]; !ok {
		writeJSONError(w, http.StatusBadRequest, "That credential cannot be managed from this page.", ApiErrorTypeValidation)
		return
	}
	if strings.TrimSpace(request.Value) == "" {
		writeJSONError(w, http.StatusBadRequest, "A credential value is required.", ApiErrorTypeValidation)
		return
	}

	receipt, err := administration.PutAuthorityCredentialIfVersion(request.Field, request.Value, request.ExpectedVersion)
	if err != nil {
		if errors.Is(err, credentialauthority.ErrVersionConflict) {
			writeJSONError(w, http.StatusConflict, "The provider credential changed. Refresh its status before retrying.", ApiErrorTypeValidation)
			return
		}
		writeJSONError(w, http.StatusServiceUnavailable, "The credential could not be stored in the credential authority.", ApiErrorTypeServerError)
		return
	}

	// Refresh the cached verdict immediately so the admin page does not make
	// operators wait for the hourly background verification cycle. This probes
	// providers without sending a message and returns only redacted verdicts.
	results := s.verifySignInProviders(r.Context())
	if s.providerVerification == nil {
		s.providerVerification = newProviderVerificationCache()
	}
	s.providerVerification.store(results...)
	writeJSON(w, map[string]any{
		"stored":          true,
		"identity":        receipt.Identity,
		"field":           receipt.Field,
		"version":         receipt.Version,
		"scope":           receipt.Scope,
		"cleanup_pending": receipt.CleanupPending,
		"results":         results,
	})
}
