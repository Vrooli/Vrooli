package main

import (
	"context"
	"net/http"
	"strings"

	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	adminhttp "landing-page-business-suite-api/handlers/administration"
	adminpasskeyhttp "landing-page-business-suite-api/handlers/adminpasskeys"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/adminsecurity"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/logx"
	passkeyinternal "landing-page-business-suite-api/internal/passkeys"
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
	var store administration.AdminAuthStore
	if s.routedDB != nil {
		store = s.routedDB
	} else {
		store = s.db
	}
	return adminhttp.AdminMFADependencies{
		MFA:        s.adminMFA,
		AdminEmail: s.sessionAdminEmail,
		IsService: func(r *http.Request) bool {
			principal, _ := r.Context().Value(servicePrincipalContextKey).(string)
			return principal != ""
		},
		WriteError:          writeJSONError,
		Log:                 logx.Info,
		LogError:            logx.Error,
		RevokeOtherSessions: s.adminAuth().RevokeOtherSessions,
		PromoteSession: func(ctx context.Context, oldID, email string) (string, error) {
			return s.adminAuth().PromoteSession(ctx, oldID, email)
		},
		Sessions: s.sessionManager,
		CurrentSessionID: func(r *http.Request) string {
			session, _ := s.sessionManager.GetSession(r, "admin_session")
			id, _ := session.Values["session_id"].(string)
			return id
		},
		SecurityEvents: &adminsecurity.Repository{DB: store},
		Notifier:       securityNotifier{s.emailService},
	}
}

func registerAdminMFARoutes(s *Server) {
	deps := s.adminMFADependencies()
	s.router.HandleFunc("/api/v1/admin/mfa", s.requireAdmin(adminhttp.AdminMFAStatus(deps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/mfa/enroll", s.requireAdmin(adminhttp.BeginAdminMFAEnrollment(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/mfa/confirm", s.requireAdmin(adminhttp.ConfirmAdminMFAEnrollment(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/mfa/disable", s.requireAdminStepUp(adminhttp.DisableAdminMFA(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/mfa/recovery-codes", s.requireAdminStepUp(adminhttp.RegenerateAdminRecoveryCodes(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/mfa/reset", s.requireAdminOrService(adminhttp.ResetAdminMFA(deps))).Methods("POST")
	registerAdminPasskeyRoutes(s)
}

func registerAdminPasskeyRoutes(s *Server) {
	store := s.primaryDB()
	_, handler := lpbsconnect.NewAdminPasskeyServiceHandler(adminpasskeyhttp.New(adminpasskeyhttp.Dependencies{Service: s.passkeyService, Challenges: s.passkeyChallenges, Credentials: &passkeyinternal.AdminCredentials{DB: store}, PublicOrigin: resolvePublicBaseURL(), MFARequired: adminhttp.AdminMFARequired, SecurityEvents: &adminsecurity.Repository{DB: store}, Notifier: securityNotifier{s.emailService}}))
	s.router.Handle(lpbsconnect.AdminPasskeyServiceBeginSecondFactorProcedure, http.HandlerFunc(handler.ServeHTTP)).Methods(http.MethodPost)
	withEmail := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			email, ok := s.sessionAdminEmail(r)
			if !ok {
				writeJSONError(w, http.StatusUnauthorized, "Session expired. Please log in again.", ApiErrorTypeUnauthorized)
				return
			}
			r.Header.Set("X-Lpbs-Admin-Email", email)
			next.ServeHTTP(w, r)
		})
	}
	admin := func(next http.Handler) http.Handler {
		return http.HandlerFunc(s.requireAdmin(func(w http.ResponseWriter, r *http.Request) { withEmail(next).ServeHTTP(w, r) }))
	}
	stepup := func(next http.Handler) http.Handler {
		return http.HandlerFunc(s.requireAdminStepUp(func(w http.ResponseWriter, r *http.Request) { withEmail(next).ServeHTTP(w, r) }))
	}
	for _, path := range []string{lpbsconnect.AdminPasskeyServiceBeginRegistrationProcedure, lpbsconnect.AdminPasskeyServiceFinishRegistrationProcedure, lpbsconnect.AdminPasskeyServiceListPasskeysProcedure, lpbsconnect.AdminPasskeyServiceRenamePasskeyProcedure} {
		s.router.Handle(path, admin(handler)).Methods(http.MethodPost)
	}
	s.router.Handle(lpbsconnect.AdminPasskeyServiceRevokePasskeyProcedure, stepup(handler)).Methods(http.MethodPost)
}
