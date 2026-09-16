package main

import (
	"net/http"
	"strings"

	adminhttp "landing-page-business-suite-api/handlers/administration"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/logx"
	"landing-page-business-suite-api/internal/securevalue"
)

// resolveAdminMFARing loads the dedicated ring that seals administrator TOTP
// secrets. It is minted at startup with the other generated rings.
func resolveAdminMFARing() (securevalue.Ring, error) {
	encoded, err := resolveAuthorityCredential("LPBS_ADMIN_MFA_ENCRYPTION_KEY")
	if err != nil {
		return securevalue.Ring{}, err
	}
	return securevalue.ParseRing(encoded)
}

// adminMFAIssuer names the site in authenticator apps: the public brand when
// configured, otherwise the mail sender name.
func adminMFAIssuer(store *experimentation.ConfigStore) func() string {
	return func() string {
		if branding := store.GetBranding(); branding != nil && strings.TrimSpace(branding.SiteName) != "" {
			return branding.SiteName
		}
		return resolveConfig("EMAIL_FROM_NAME")
	}
}

// authThrottleOrNil keeps a nil *AuthThrottle from becoming a non-nil
// interface value in handler dependencies.
func (s *Server) authThrottleOrNil() adminhttp.AdminThrottle {
	if s.authThrottle == nil {
		return nil
	}
	return s.authThrottle
}

func (s *Server) adminSecondFactor() adminhttp.AdminSecondFactor {
	if s.adminMFA == nil {
		return nil
	}
	return s.adminMFA
}

var _ adminhttp.AdminSecondFactor = (*administration.AdminMFA)(nil)

func (s *Server) adminMFADependencies() adminhttp.AdminMFADependencies {
	return adminhttp.AdminMFADependencies{
		MFA:        s.adminMFA,
		AdminEmail: s.sessionAdminEmail,
		IsService: func(r *http.Request) bool {
			principal, _ := r.Context().Value(servicePrincipalContextKey).(string)
			return principal != ""
		},
		WriteError: writeJSONError,
		Log:        logx.Info,
		LogError:   logx.Error,
	}
}

func registerAdminMFARoutes(s *Server) {
	deps := s.adminMFADependencies()
	s.router.HandleFunc("/api/v1/admin/mfa", s.requireAdmin(adminhttp.AdminMFAStatus(deps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/mfa/enroll", s.requireAdmin(adminhttp.BeginAdminMFAEnrollment(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/mfa/confirm", s.requireAdmin(adminhttp.ConfirmAdminMFAEnrollment(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/mfa/disable", s.requireAdmin(adminhttp.DisableAdminMFA(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/mfa/recovery-codes", s.requireAdmin(adminhttp.RegenerateAdminRecoveryCodes(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/mfa/reset", s.requireAdminOrService(adminhttp.ResetAdminMFA(deps))).Methods("POST")
}
