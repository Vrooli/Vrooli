package main

import (
	"net/http"

	adminhttp "landing-page-business-suite-api/handlers/administration"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/adminsecurity"
	"landing-page-business-suite-api/internal/logx"
)

// adminCredentialResetDependencies wires the service-principal-only recovery
// path. Authority writes go through the same credential authority the startup
// seeder reads, so a rotated password survives a restart on a bootstrap
// account.
func (s *Server) adminCredentialResetDependencies() adminhttp.AdminCredentialResetDependencies {
	var store administration.AdminAuthStore
	if s.routedDB != nil {
		store = s.routedDB
	} else {
		store = s.db
	}
	return adminhttp.AdminCredentialResetDependencies{
		Auth: s.adminAuth(),
		IsService: func(r *http.Request) bool {
			principal, _ := r.Context().Value(servicePrincipalContextKey).(string)
			return principal != ""
		},
		DefaultPassword:     func() string { return resolveSecret("ADMIN_DEFAULT_PASSWORD") },
		ValidateEmail:       func(email string) error { _, err := ValidateEmail(email); return err },
		PutAuthority:        administration.PutAuthorityCredential,
		RevokeOtherSessions: s.adminAuth().RevokeOtherSessions,
		SecurityEvents:      &adminsecurity.Repository{DB: store},
		Notifier:            securityNotifier{s.emailService},
		WriteError:          writeJSONError,
		Log:                 logx.Info,
		LogError:            logx.Error,
	}
}

func registerAdminCredentialResetRoutes(s *Server) {
	deps := s.adminCredentialResetDependencies()
	s.router.HandleFunc("/api/v1/admin/admin-credentials/reset", s.requireAdminOrService(adminhttp.ResetAdminCredential(deps))).Methods("POST")
}
