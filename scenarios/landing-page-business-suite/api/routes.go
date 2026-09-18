package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/go-webauthn/webauthn/webauthn"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	accounthttp "landing-page-business-suite-api/handlers/account"
	accountsecurityhttp "landing-page-business-suite-api/handlers/accountsecurity"
	adminhttp "landing-page-business-suite-api/handlers/administration"
	assethttp "landing-page-business-suite-api/handlers/assets"
	bundlehttp "landing-page-business-suite-api/handlers/bundles"
	businessaccounthttp "landing-page-business-suite-api/handlers/businessaccount"
	billinghttp "landing-page-business-suite-api/handlers/commerce"
	landinghttp "landing-page-business-suite-api/handlers/config"
	contenthttp "landing-page-business-suite-api/handlers/content"
	couponhttp "landing-page-business-suite-api/handlers/coupons"
	downloadhttp "landing-page-business-suite-api/handlers/delivery"
	deploymenthttp "landing-page-business-suite-api/handlers/deployment"
	desktoplinkhttp "landing-page-business-suite-api/handlers/desktoplink"
	digesthttp "landing-page-business-suite-api/handlers/digest"
	docshandler "landing-page-business-suite-api/handlers/docs"
	emaileventhttp "landing-page-business-suite-api/handlers/emailevents"
	varianthttp "landing-page-business-suite-api/handlers/experimentation"
	feedbackhttp "landing-page-business-suite-api/handlers/feedback"
	intelligencehandler "landing-page-business-suite-api/handlers/intelligence"
	measureshandler "landing-page-business-suite-api/handlers/measures"
	metricshttp "landing-page-business-suite-api/handlers/metrics"
	passkeyhttp "landing-page-business-suite-api/handlers/passkeys"
	pricinghandler "landing-page-business-suite-api/handlers/pricing"
	seohttp "landing-page-business-suite-api/handlers/seo"
	variantspacehttp "landing-page-business-suite-api/handlers/variant_space"
	authadmin "landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/emailevents"
	"landing-page-business-suite-api/internal/logx"
	"landing-page-business-suite-api/internal/monetization"
	passkeyinternal "landing-page-business-suite-api/internal/passkeys"
	"landing-page-business-suite-api/internal/providerprobe"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/identity"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
)

func (s *Server) setupRoutes() {
	s.router.Use(securityHeadersMiddleware)
	s.router.Use(loggingMiddleware)

	registerHealthRoutes(s)
	registerMonetizationJourneyRoute(s)
	registerLandingRoutes(s)
	registerBackdropRoutes(s)
	registerPresentationAssetRoutes(s)
	registerAuthRoutes(s)
	registerPasskeyRoutes(s)
	desktoplinkhttp.RegisterRoutes(s.router, desktoplinkhttp.Dependencies{
		Service:    s.desktopLinkService,
		LPBSUserID: getUserID,
		ResolveBusinessAccountID: func(ctx context.Context, userID, requestedID string) (string, error) {
			account, err := s.businessAccounts.ResolveForUser(ctx, userID, getUserEmail(ctx), requestedID)
			if err != nil {
				return "", err
			}
			return account.ID, nil
		},
		VerifyLocalIdentity: func(ctx context.Context, token string) (identity.Principal, error) {
			if s.desktopLinkVerifier == nil {
				return identity.Principal{}, authn.ErrNoProvider
			}
			return s.desktopLinkVerifier.Verify(ctx, token)
		},
		IssueLease:        s.issueDesktopEntitlementLease,
		WriteAudit:        logx.Info,
		RequireRecentAuth: s.requireRecentUserAuth,
	}, s.requireUserAuth)
	businessaccounthttp.RegisterRoutes(s.router, businessaccounthttp.Dependencies{
		Repository:        s.businessAccounts,
		UserID:            getUserID,
		UserEmail:         getUserEmail,
		RequireRecentAuth: s.requireRecentUserAuth,
	}, s.requireUserAuth)
	registerFixtureRoutes(s)
	registerAccountRoutes(s)
	registerReceiptRoutes(s)
	registerBillingRoutes(s)
	registerAdminCoreRoutes(s)
	registerReaderTokenRoutes(s)
	registerRemoteProfileRoutes(s)
	registerCommerceAdminRoutes(s)
	registerVariantRoutes(s)
	registerContentRoutes(s)
	registerMetricsRoutes(s)
	registerBusinessDigestRoutes(s)
	monetization.RegisterRoutes(s.router, s.primaryDB(), s.planService.BundleKey)
	registerFeedbackRoutes(s)
	registerWaitlistRoutes(s)
	registerCreditsRoutes(s)
	registerAIRoutes(s)
	registerDocsRoutes(s)
	registerAdminUserRoutes(s)
	registerUpdateRoutes(s)
	registerMeasuresRoutes(s)
	registerDeployReadinessRoute(s)
}

func registerPasskeyRoutes(s *Server) {
	if s.passkeyService == nil {
		return
	}
	loadUser := func(ctx context.Context, id, email string) (passkeyinternal.User, error) {
		if id == "" {
			return passkeyinternal.User{}, errors.New("customer user is unavailable")
		}
		handle := sha256.Sum256([]byte("lpbs-webauthn-user:" + id))
		user := passkeyinternal.User{ID: id, Handle: handle[:], Email: email}
		if s.passkeyCredentials != nil {
			records, err := s.passkeyCredentials.List(ctx, id)
			if err != nil {
				return passkeyinternal.User{}, err
			}
			for _, record := range records {
				user.Credentials = append(user.Credentials, record.Credential)
			}
		}
		return user, nil
	}
	_, handler := lpbsconnect.NewPasskeyServiceHandler(passkeyhttp.New(passkeyhttp.Dependencies{
		Service:      s.passkeyService,
		Challenges:   s.passkeyChallenges,
		Credentials:  s.passkeyCredentials,
		PublicOrigin: resolvePublicBaseURL(),
		CurrentUser: func(ctx context.Context) (passkeyinternal.User, error) {
			id, email := getUserID(ctx), getUserEmail(ctx)
			if id == "" || email == "" {
				return passkeyinternal.User{}, errors.New("customer session is not authenticated")
			}
			return loadUser(ctx, id, email)
		},
		DiscoverableUser: func(ctx context.Context, credentialID, _ []byte) (passkeyinternal.User, error) {
			userID, err := s.passkeyCredentials.UserForCredential(ctx, credentialID)
			if err != nil {
				return passkeyinternal.User{}, err
			}
			user, err := s.userAuthService.GetUserByID(ctx, userID)
			if err != nil || user == nil {
				return passkeyinternal.User{}, errors.New("customer user is unavailable")
			}
			return loadUser(ctx, user.ID, user.Email)
		},
		EstablishSession: func(ctx context.Context, user passkeyinternal.User, userAgent string, response *connect.Response[lpbsv1.FinishPasskeyAuthenticationResponse]) error {
			account, err := s.userAuthService.GetUserByID(ctx, user.ID)
			if err != nil || account == nil {
				return errors.New("customer user is unavailable")
			}
			// Connect's request context does not expose the peer address. The
			// transport still records the authentication method and user-agent;
			// the neutral address keeps the session audit row valid without
			// trusting forwarding headers supplied by the browser.
			pair, err := s.userAuthService.CreateSessionWithMetadata(ctx, account, "0.0.0.0", userAgent, "passkey", time.Now().UTC())
			if err != nil {
				return err
			}
			secure := isSecureCookiesEnabled()
			access, refresh, hint := adminhttp.AuthCookieNames(secure)
			response.Header().Add("Set-Cookie", (&http.Cookie{Name: access, Value: pair.AccessToken, Path: "/", Expires: pair.ExpiresAt, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode}).String())
			response.Header().Add("Set-Cookie", (&http.Cookie{Name: refresh, Value: pair.RefreshToken, Path: "/api/v1/auth", Expires: pair.SessionExpiresAt, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode}).String())
			response.Header().Add("Set-Cookie", (&http.Cookie{Name: hint, Value: "1", Path: "/", Expires: pair.SessionExpiresAt, Secure: secure, SameSite: http.SameSiteLaxMode}).String())
			return nil
		},
		IssueNativeGrant: func(ctx context.Context, userID, rawContext, binding, userAgent string) (string, error) {
			var flow struct {
				RedirectURI         string `json:"redirect_uri"`
				CodeChallenge       string `json:"code_challenge"`
				CodeChallengeMethod string `json:"code_challenge_method"`
			}
			if err := json.Unmarshal([]byte(rawContext), &flow); err != nil || flow.RedirectURI == "" || flow.CodeChallenge == "" || flow.CodeChallengeMethod != "S256" {
				return "", errors.New("native sign-in context is invalid")
			}
			code := passkeyinternal.NewChallengeID()
			if err := s.nativeGrants.Issue(ctx, code, userID, flow.CodeChallenge, flow.RedirectURI, binding, "0.0.0.0", userAgent, time.Now().UTC(), time.Minute); err != nil {
				return "", err
			}
			return code, nil
		},
		Throttle: s.authThrottle,
		Notify: func(ctx context.Context, email, event, detail string) error {
			return s.emailService.SendPasskeyNotification(ctx, email, event, detail)
		},
	}))
	throttled := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, err := s.authThrottle.Allow(r.Context(), authadmin.ThrottleBucket("passkey-ip", getClientIP(r)), authadmin.ThrottleRule{Limit: 30, Window: 15 * time.Minute})
		if err != nil {
			writeJSONError(w, http.StatusServiceUnavailable, "passkey service unavailable", ApiErrorTypeServerError)
			return
		}
		if !allowed {
			writeJSONError(w, http.StatusTooManyRequests, "too many passkey requests", ApiErrorTypeRateLimited)
			return
		}
		handler.ServeHTTP(w, r)
	})
	public := throttled
	s.router.Handle(lpbsconnect.PasskeyServiceBeginAuthenticationProcedure, public).Methods(http.MethodPost)
	protected := s.requireUserAuth(public)
	recent := http.HandlerFunc(s.requireRecentUserAuth(throttled.ServeHTTP))
	for _, path := range []string{lpbsconnect.PasskeyServiceBeginRegistrationProcedure, lpbsconnect.PasskeyServiceFinishRegistrationProcedure, lpbsconnect.PasskeyServiceRevokePasskeyProcedure} {
		s.router.Handle(path, recent).Methods(http.MethodPost)
	}
	for _, path := range []string{lpbsconnect.PasskeyServiceListPasskeysProcedure, lpbsconnect.PasskeyServiceRenamePasskeyProcedure} {
		s.router.Handle(path, protected).Methods(http.MethodPost)
	}
	s.router.Handle(lpbsconnect.PasskeyServiceFinishAuthenticationProcedure, public).Methods(http.MethodPost)
}

func passkeyUserForAccount(ctx context.Context, credentials *passkeyinternal.Credentials, id, email string) (passkeyinternal.User, error) {
	handle := sha256.Sum256([]byte("lpbs-webauthn-user:" + id))
	user := passkeyinternal.User{ID: id, Handle: handle[:], Email: email}
	if credentials == nil {
		return user, nil
	}
	records, err := credentials.List(ctx, id)
	if err != nil {
		return passkeyinternal.User{}, err
	}
	for _, record := range records {
		user.Credentials = append(user.Credentials, record.Credential)
	}
	return user, nil
}

func registerReaderTokenRoutes(s *Server) {
	_, handler := lpbsconnect.NewReaderTokenServiceHandler(adminhttp.NewReaderTokenHandler(adminhttp.ReaderTokenDependencies{Service: s.readerTokens}))
	wrapped := s.requireAdminStepUp(http.HandlerFunc(handler.ServeHTTP))
	for _, path := range []string{lpbsconnect.ReaderTokenServiceIssueReaderTokenProcedure, lpbsconnect.ReaderTokenServiceListReaderTokensProcedure, lpbsconnect.ReaderTokenServiceRevokeReaderTokenProcedure} {
		s.router.Handle(path, wrapped).Methods(http.MethodPost)
	}
}

func registerBusinessDigestRoutes(s *Server) {
	_, handler := lpbsconnect.NewBusinessDigestServiceHandler(digesthttp.New(s.metricsService))
	s.router.Handle(lpbsconnect.BusinessDigestServiceGetBusinessDigestProcedure, s.requireMetricsReader(http.HandlerFunc(handler.ServeHTTP))).Methods(http.MethodPost)
}

func registerPresentationAssetRoutes(s *Server) {
	if s.presentationAssetHandler == nil {
		return
	}
	// A literal byte route keeps the generated endpoint inventory complete;
	// the cache handler validates the full content-addressed PNG filename.
	s.router.HandleFunc("/api/v1/presentation-assets/{file}", s.presentationAssetHandler.ServeHTTP).Methods("GET", "HEAD")
}

// registerBackdropRoutes keeps the dynamic Backdrop Studio address on the
// server side. The UI calls this same-origin endpoint; api-core discovery
// resolves the current lifecycle-managed port for every request.
func registerBackdropRoutes(s *Server) {
	s.router.HandleFunc("/api/v1/backdrops/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(mux.Vars(r)["id"])
		if id == "" {
			http.Error(w, "backdrop id is required", http.StatusBadRequest)
			return
		}
		resolve := s.backdropResolver
		if resolve == nil {
			resolve = func(ctx context.Context) (string, error) {
				return discovery.ResolveScenarioURLDefault(ctx, "backdrop-studio")
			}
		}
		baseURL, err := resolve(r.Context())
		if err != nil || strings.TrimSpace(baseURL) == "" {
			http.Error(w, "backdrop studio is unavailable", http.StatusServiceUnavailable)
			return
		}
		body, err := json.Marshal(map[string]string{"id": id})
		if err != nil {
			http.Error(w, "failed to encode backdrop request", http.StatusInternalServerError)
			return
		}
		request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, strings.TrimRight(baseURL, "/")+"/vrooli.backdrop_studio.v1.release.ReleaseService/GetReference", bytes.NewReader(body))
		if err != nil {
			http.Error(w, "failed to create backdrop request", http.StatusInternalServerError)
			return
		}
		request.Header.Set("Content-Type", "application/json")
		client := s.backdropHTTPClient
		if client == nil {
			client = http.DefaultClient
		}
		response, err := client.Do(request)
		if err != nil {
			http.Error(w, "backdrop studio request failed", http.StatusBadGateway)
			return
		}
		defer response.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(response.StatusCode)
		if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
			payload, readErr := io.ReadAll(response.Body)
			if readErr == nil {
				var metadata map[string]any
				if json.Unmarshal(payload, &metadata) == nil {
					if uri, ok := metadata["uri"].(string); ok && uri != "" {
						if assetURL, ok := metadata["url"].(string); !ok || assetURL == "" || !strings.HasPrefix(assetURL, "http://") && !strings.HasPrefix(assetURL, "https://") {
							metadata["url"] = uri
							if !strings.HasPrefix(uri, "http://") && !strings.HasPrefix(uri, "https://") {
								metadata["url"] = strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(uri, "/")
							}
						}
					}
					if encoded, encodeErr := json.Marshal(metadata); encodeErr == nil {
						_, _ = w.Write(encoded)
						return
					}
				}
				_, _ = w.Write(payload)
				return
			}
		}
		_, _ = io.Copy(w, response.Body)
	}).Methods(http.MethodGet)
}

func registerReceiptRoutes(s *Server) {
	s.router.HandleFunc("/api/v1/subscriptions/receipts", s.requireUserAuth(billinghttp.RegisterReceipt(billinghttp.ReceiptDependencies{
		Validators: s.receiptValidators,
		Register:   s.accountService.RegisterReceipt,
		UserIdentity: func(ctx context.Context) string {
			return getUserEmail(ctx)
		},
		WriteError: writeJSONError,
	}))).Methods(http.MethodPost)
}

func registerMeasuresRoutes(s *Server) {
	if err := measureshandler.RegisterRoutes(s.router, s.primaryDB(), nil, s.requireAdminOrService); err != nil {
		panic("register measures routes: " + err.Error())
	}
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("X-XSS-Protection", "0")
		header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

func registerDeployReadinessRoute(s *Server) {
	deps := deploymenthttp.Dependencies{
		Storage: s.downloadHosting, TestStorage: s.downloadHosting, Catalog: s.downloadService, RemoteProfiles: s.remoteProfileService,
		StripeReadiness: s.stripeReadiness,
		BundleKey:       s.planService.BundleKey,
		WriteError: func(w http.ResponseWriter, status int, message, kind string) {
			writeJSONError(w, status, message, kind)
		},
	}
	deploymenthttp.RegisterConnectRoutes(s.router, deps, s.requireAdminOrService)
	// Preserve the documented JSON endpoint until its existing callers migrate.
	s.router.HandleFunc(
		"/api/v1/deploy-readiness",
		s.requireAdminOrService(deploymenthttp.Readiness(deps)),
	).Methods("POST")
}

func registerHealthRoutes(s *Server) {
	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	// The watchdog asserts the policy the login path actually enforces, not a
	// second reading of the environment variable. The previous version only
	// complained when ADMIN_REQUIRE_MFA was explicitly "false", so an unset
	// variable that derived "not required" in production passed silently —
	// the one case where enforcement was off and nothing said so.
	adminMFAPolicyCheck := health.Func("admin_mfa_policy", func(context.Context) error {
		if isProductionEnvironment() && !adminhttp.AdminMFARequired() {
			return fmt.Errorf("administrator second factors are not required in this production deployment; set ADMIN_REQUIRE_MFA=true or declare the production environment")
		}
		return nil
	})
	healthHandler := health.New().Version("1.0.0").Check(health.DB(s.primaryDB()), health.Critical).Check(adminMFAPolicyCheck, health.Optional).Check(s.signInDeliveryCheck(), health.Optional).Check(s.signInEmailDNSCheck(), health.Optional).Check(s.paymentsCheck(), health.Optional).Check(s.signInProviderCredentialCheck(), health.Optional).Check(s.providerCredentialCheck("payments_credentials", providerprobe.ProviderStripe), health.Optional).Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
}

func registerLandingRoutes(s *Server) {
	landinghttp.RegisterLandingConfigConnectRoutes(s.router, s.landingConfigService)
	pricinghandler.RegisterRoutes(s.router, s.planService.GetPricingOverview)
	variantspacehttp.RegisterRoutes(s.router, s.variantSpace.JSONBytes)

	// Customization command for landing updates
	s.router.HandleFunc("/api/v1/customize", contenthttp.Customize(time.Now)).Methods("POST")
}

func registerAuthRoutes(s *Server) {
	// Consumer access-token verification is intentionally public: bundled
	// scenarios need the public keys, while the private signing key stays in
	// this authority and is never distributed to a relying party.
	s.router.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, _ *http.Request) {
		body, err := s.userAuthService.PublicKeySet()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "consumer key set unavailable", ApiErrorTypeServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=300")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}).Methods("GET")
	// User Authentication endpoints (magic link + JWT)
	// Public auth endpoints (no auth required)
	// Durable throttles replace the process-local limiter so limits survive
	// restarts and hold across replicas.
	// Keep the narrow recipient limiter wired in addition to the durable
	// per-email and per-IP throttle. This protects the public endpoint even if
	// the durable throttle is temporarily unavailable.
	deps := userAuthHandlerDependencies(s.userAuthService, s.magicLinkLimiter)
	deps.Throttle = s.authThrottle
	s.router.HandleFunc("/api/v1/auth/magic-link", adminhttp.RequestMagicLink(deps)).Methods("POST")
	s.router.HandleFunc("/api/v1/auth/magic-link/preview", adminhttp.PreviewSignIn(deps)).Methods("POST")
	// Verification consumes a credential, so it is POST-only: a GET that
	// signs in would let link scanners and prefetchers burn the link.
	s.router.HandleFunc("/api/v1/auth/verify", adminhttp.VerifyMagicLink(deps)).Methods("POST")
	s.router.HandleFunc("/api/v1/auth/verify-code", adminhttp.VerifySignInCode(deps)).Methods("POST")
	s.router.HandleFunc("/api/v1/auth/authorize", adminhttp.AuthorizeWithPKCE(deps, s.nativeGrants)).Methods("GET", "POST")
	s.router.HandleFunc("/api/v1/auth/token", adminhttp.ExchangeAuthorizationCode(deps, s.nativeGrants)).Methods("POST")
	s.router.HandleFunc("/api/v1/auth/refresh", adminhttp.RefreshTokens(deps)).Methods("POST")
	// Protected auth endpoints (require user auth)
	s.router.HandleFunc("/api/v1/auth/logout", s.requireUserAuth(adminhttp.LogoutUser(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/auth/me", s.requireUserAuth(adminhttp.Me(deps))).Methods("GET")
}

func registerAccountRoutes(s *Server) {
	accounthttp.RegisterRoutes(s.router, accounthttp.NewCommerceReader(s.accountService), getUserEmail, s.requireUserAuth)
	securityDeps := accountsecurityhttp.Dependencies{
		Service: s.accountSecurityService, UserID: getUserID, UserEmail: getUserEmail, SessionID: getSessionID,
		ClientIP: getClientIP, SecureCookies: isSecureCookiesEnabled, Now: time.Now,
		PublicMiddleware: func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				allowed, err := s.authThrottle.Allow(r.Context(), authadmin.ThrottleBucket("sign-in-delivery-status", getClientIP(r)), authadmin.ThrottleRule{Limit: 30, Window: 15 * time.Minute})
				if err != nil {
					http.Error(w, "delivery status unavailable", http.StatusServiceUnavailable)
					return
				}
				if !allowed {
					http.Error(w, "too many delivery status requests", http.StatusTooManyRequests)
					return
				}
				next(w, r)
			}
		},
	}
	securityDeps.StartPasskey = func(ctx context.Context, userID, binding string) (string, string, time.Time, error) {
		user, err := s.userAuthService.GetUserByID(ctx, userID)
		if err != nil || user == nil {
			return "", "", time.Time{}, errors.New("customer user is unavailable")
		}
		passkeyUser, err := passkeyUserForAccount(ctx, s.passkeyCredentials, user.ID, user.Email)
		if err != nil {
			return "", "", time.Time{}, err
		}
		if len(passkeyUser.Credentials) == 0 {
			return "", "", time.Time{}, nil
		}
		options, session, err := s.passkeyService.BeginUserAuthentication(passkeyUser)
		if err != nil {
			return "", "", time.Time{}, err
		}
		data, err := passkeyinternal.EncodeSessionData(session)
		if err != nil {
			return "", "", time.Time{}, err
		}
		id := passkeyinternal.NewChallengeID()
		expires := time.Now().UTC().Add(5 * time.Minute)
		if err := s.passkeyChallenges.Create(ctx, passkeyinternal.Challenge{ID: id, Value: session.Challenge, Purpose: "customer_reauth", Subject: userID, BindingHash: passkeyinternal.Hash(passkeyinternal.NormalizeBinding(binding)), SessionData: data, ExpiresAt: expires}); err != nil {
			return "", "", time.Time{}, err
		}
		return string(options), id, expires, nil
	}
	securityDeps.VerifyPasskey = func(ctx context.Context, userID, sessionID, binding, ceremony string, assertion []byte, _ string) (time.Time, error) {
		challenge, err := s.passkeyChallenges.Consume(ctx, ceremony, "customer_reauth", passkeyinternal.Hash(passkeyinternal.NormalizeBinding(binding)))
		if err != nil || challenge.Subject != userID {
			return time.Time{}, errors.New("passkey ceremony is not valid")
		}
		var session webauthn.SessionData
		if err := json.Unmarshal(challenge.SessionData, &session); err != nil {
			return time.Time{}, err
		}
		user, err := s.userAuthService.GetUserByID(ctx, userID)
		if err != nil || user == nil {
			return time.Time{}, errors.New("customer user is unavailable")
		}
		passkeyUser, err := passkeyUserForAccount(ctx, s.passkeyCredentials, user.ID, user.Email)
		if err != nil {
			return time.Time{}, err
		}
		credential, err := s.passkeyService.FinishLogin(passkeyUser, session, assertion, resolvePublicBaseURL())
		if err != nil {
			return time.Time{}, errors.New("passkey assertion rejected")
		}
		if err := s.passkeyCredentials.RecordUse(ctx, *credential); err != nil {
			return time.Time{}, err
		}
		now := time.Now().UTC()
		if err := s.accountSecurityService.RecordReauthentication(ctx, userID, sessionID, now); err != nil {
			return time.Time{}, err
		}
		return now, nil
	}
	accountsecurityhttp.RegisterRoutes(s.router, securityDeps, s.requireUserAuth)
	registerEntitlementRoute(s)
	// Downloads are open to anonymous visitors; the delivery authorizer fails
	// closed and demands an identity plus an active subscription only when the
	// selected asset is entitlement-gated.
	downloadhttp.RegisterConnectAuthorizationRoute(s.router, s.planService.BundleKey, s.downloadService, downloadConnectAuthorizationDependencies(s.downloadAuthorizer, s.downloadHosting, s.planService, s.db), s.optionalUserAuth)
	s.router.HandleFunc("/api/v1/downloads", s.optionalUserAuth(downloadhttp.Authorize(downloadAuthorizationDependencies(s.downloadAuthorizer, s.downloadHosting, s.planService, s.db)))).Methods("GET")
}

func registerEntitlementRoute(s *Server) {
	s.router.HandleFunc("/api/v1/entitlements", s.requireUserAuth(func(w http.ResponseWriter, r *http.Request) {
		identity := getUserEmail(r.Context())
		if requested := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("user"))); requested != "" && requested != strings.ToLower(identity) {
			writeJSONError(w, http.StatusForbidden, "entitlement identity does not match token", ApiErrorTypeForbidden)
			return
		}
		payload, err := s.accountService.GetEntitlementsContext(r.Context(), identity)
		if err != nil {
			writeJSONError(w, http.StatusServiceUnavailable, "entitlement service unavailable", ApiErrorTypeServerError)
			return
		}
		if payload.SharedDecision().Warning {
			w.Header().Set("X-Entitlement-Warning", "past_due")
		}
		lease, err := s.userAuthService.SignEntitlementLease(entitlementclient.Payload{
			UserIdentity:      identity,
			Status:            payload.Status,
			PlanTier:          payload.PlanTier,
			PlanRank:          payload.PlanRank,
			PriceID:           payload.PriceID,
			Features:          payload.Features,
			Limits:            payload.Limits,
			NotAfter:          payload.NotAfter,
			BillingCycleStart: payload.BillingCycleStart,
			Credits:           payload.Credits,
			Subscription:      payload.Subscription,
		})
		if err != nil {
			writeJSONError(w, http.StatusServiceUnavailable, "entitlement signing unavailable", ApiErrorTypeServerError)
			return
		}
		payload.Lease = lease
		writeJSON(w, payload)
	})).Methods(http.MethodGet)
}

func registerBillingRoutes(s *Server) {
	// Generated Connect payment procedures preserve the public checkout,
	// authenticated portal, and admin cancellation boundaries.
	billingDeps := billingConnectDependencies(s.stripeService, s.businessAccounts)
	billingDeps.RequireRecentAuth = s.requireRecentUserAuth
	billinghttp.RegisterConnectRoutes(s.router, billingDeps, s.requireUserAuth, s.requireAdminStepUp)

	// Stripe webhook remains Stripe's signed HTTP callback, not a browser RPC.
	s.router.HandleFunc("/api/v1/webhooks/stripe", billinghttp.Webhook(billingWebhookDependencies(s.stripeService))).Methods("POST")
	// SendGrid owns this signed REST callback shape; it is intentionally not a Connect procedure.
	s.router.HandleFunc("/api/v1/webhooks/sendgrid", emaileventhttp.Handler(emaileventhttp.Dependencies{
		Store: &emailevents.Repository{DB: s.db}, PublicKey: func() string { return resolveSecret("SENDGRID_WEBHOOK_PUBLIC_KEY") },
		Throttle: s.authThrottle, ClientIP: getClientIP, Log: logx.Error,
	})).Methods("POST")
}

func registerAdminCoreRoutes(s *Server) {
	// Admin authentication/reset are generated Connect services. Session cookies
	// remain response headers and never enter protobuf payloads.
	landinghttp.RegisterPresentationAdminRoutes(s.router, s.configStore, s.requireAdminStepUp)
	adminhttp.RegisterSessionConnectRoutes(s.router, s.adminSessionDependencies(), adminhttp.ResetDependencies{Reset: s.resetDemoData, Now: time.Now, LogError: logx.Error}, s.requireAdmin)
	registerAdminMFARoutes(s)
	registerAdminCredentialResetRoutes(s)
	s.router.HandleFunc("/api/v1/admin/auth/delivery", s.requireMetricsReader(s.signInDeliveryReport)).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/auth/email-readiness", s.requireAdmin(s.emailReadinessReport)).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/provider-credentials", s.requireAdmin(s.providerVerificationReport)).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/security-events", s.requireAdmin(s.adminSecurityEvents)).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/auth/delivery-probe", s.requireAdmin(s.signInDeliveryProbe)).Methods("POST")
	profileDeps := s.adminProfileDependencies()
	adminhttp.RegisterProfileConnectRoutes(s.router, profileDeps, s.requireAdminProfile)
	billinghttp.RegisterStripeSettingsConnectRoutes(s.router, s.paymentSettings, s.stripeService, s.paymentAnomaly, s.requireAdminStepUp)
	s.router.HandleFunc("/api/v1/admin/stripe/verify-price", s.requireAdminStepUp(bundlehttp.VerifyStripePrice(bundleStripeHandlerDependencies(s.stripeService)))).Methods("GET")
	adminhttp.RegisterAPIKeyConnectRoutes(s.router, s.apiKeyService, s.requireAdminStepUp)
}

func registerRemoteProfileRoutes(s *Server) {
	// List and test/proxy use requireAdminOrService so inter-scenario clients (e.g. s2d) can call them with a service bearer token.
	deps := remoteProfileHandlerDependencies(s.remoteProfileService, s.sessionAdminEmail)
	s.router.HandleFunc("/api/v1/admin/remote-profiles", s.requireAdminOrService(adminhttp.ListRemoteProfiles(deps))).Methods("GET")
	// The local LPBS service identity is used by scenario-to-cloud during a
	// governed deployment to reconcile a sealed target credential. It is not a
	// browser/admin credential and cannot access the remaining profile controls.
	s.router.HandleFunc("/api/v1/admin/remote-profiles", s.requireAdminOrService(adminhttp.CreateRemoteProfile(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}", s.requireAdminOrService(adminhttp.UpdateRemoteProfile(deps))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}", s.requireAdminStepUp(adminhttp.DeleteRemoteProfile(deps))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/login", s.requireAdminStepUp(adminhttp.LoginRemoteProfile(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/logout", s.requireAdminStepUp(adminhttp.LogoutRemoteProfile(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/test", s.requireAdminOrService(adminhttp.TestRemoteProfile(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/session-links", s.requireAdminStepUp(adminhttp.RemoteProfileSessionLinks(deps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/remote-revoke", s.requireAdminStepUp(adminhttp.RevokeRemoteProfileSessions(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/proxy", s.requireAdminOrService(adminhttp.ProxyRemoteProfile(deps))).Methods("POST")
	sessionDeps := remoteProfileSessionDependencies(s.routedDB)
	s.router.HandleFunc("/api/v1/admin/remote-profile-sessions", s.requireAdminStepUp(adminhttp.ListIncomingRemoteProfileSessions(sessionDeps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/remote-profile-sessions/{session_id}", s.requireAdminStepUp(adminhttp.RevokeIncomingRemoteProfileSession(sessionDeps))).Methods("DELETE")
}

func registerCommerceAdminRoutes(s *Server) {
	// Download hosting + assets
	downloadhttp.RegisterConnectAppRoutes(s.router, s.planService.BundleKey, s.downloadService, s.requireAdminStepUp)
	downloadAppDependencies := deliveryAppDependencies(s.planService)
	s.router.HandleFunc("/api/v1/admin/download-apps", s.requireAdminOrService(downloadhttp.ListApps(downloadAppDependencies, s.downloadService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-apps", s.requireAdminStepUp(downloadhttp.CreateApp(downloadAppDependencies, s.downloadService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-apps/{app_key}", s.requireAdminStepUp(downloadhttp.SaveApp(downloadAppDependencies, s.downloadService))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/download-apps/{app_key}", s.requireAdminStepUp(downloadhttp.DeleteApp(downloadAppDependencies, s.downloadService))).Methods("DELETE")
	downloadAdminDependencies := downloadAdminDependencies(s.downloadHosting, s.planService)
	downloadAdminAssetDependencies := downloadAdminAssetDependencies(s.downloadService, s.downloadHosting, s.planService)
	s.router.HandleFunc("/api/v1/admin/download-storage", s.requireAdmin(downloadhttp.GetStorage(downloadAdminDependencies))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-storage", s.requireAdminStepUp(downloadhttp.UpdateStorage(downloadAdminDependencies))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/download-storage/test", s.requireAdminOrService(downloadhttp.TestStorage(downloadAdminDependencies))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-artifacts", s.requireAdmin(downloadhttp.ListArtifacts(downloadAdminDependencies))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-artifacts/by-app", s.requireAdmin(downloadhttp.ListArtifactsByApp(downloadAdminDependencies))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-artifacts/presign-upload", s.requireAdminOrService(downloadhttp.PresignUpload(downloadAdminDependencies))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-artifacts/commit", s.requireAdminOrService(downloadhttp.CommitArtifact(downloadAdminDependencies))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-artifacts/{artifact_id}/presign-get", s.requireAdmin(downloadhttp.PresignGet(downloadAdminDependencies))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-assets/apply", s.requireAdminOrService(downloadhttp.ApplyArtifact(downloadAdminAssetDependencies))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-assets/set-current", s.requireAdminStepUp(downloadhttp.SetArtifactCurrent(downloadAdminAssetDependencies))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-channels/promote", s.requireAdminOrService(downloadhttp.PromoteChannel(downloadAdminAssetDependencies))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-channels/head", s.requireAdminOrService(downloadhttp.GetChannelHead(downloadAdminAssetDependencies))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-channels/halt", s.requireAdminOrService(downloadhttp.SetChannelHalt(downloadAdminAssetDependencies))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-channels/recover", s.requireAdminOrService(downloadhttp.RecoverChannel(downloadAdminAssetDependencies))).Methods("POST")

	// Bundles + pricing
	registerBundleAdminConnectRoutes(s.router, s.planService, s.stripeService, s.requireAdminStepUp)
	s.router.HandleFunc("/api/v1/admin/bundles/{bundle_key}/prices", s.requireAdminStepUp(bundlehttp.CreatePrice(bundleCreateHandlerDependencies(s.planService, s.stripeService)))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/bundles/{bundle_key}/prices/{price_id}", s.requireAdminStepUp(bundlehttp.DeletePrice(bundleDeleteHandlerDependencies(s.planService)))).Methods("DELETE")

	// Stripe import tools
	s.router.HandleFunc("/api/v1/admin/stripe/import-preview", s.requireAdmin(bundlehttp.PreviewStripeImport(bundleImportHandlerDependencies(s.stripeService, s.planService)))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/stripe/import", s.requireAdminStepUp(bundlehttp.ImportStripePrices(bundleStripeImportDependencies(s.stripeService, s.planService)))).Methods("POST")

	couponhttp.RegisterConnectRoutes(s.router, s.stripeService, s.planService, s.routedDB, s.requireAdminStepUp, couponProviderError, logx.Info)
}

func registerVariantRoutes(s *Server) {
	varianthttp.RegisterConnectRoutes(s.router, s.configStore, s.requireAdminStepUp)
	writeDependencies := varianthttp.WriteDependencies{Store: s.configStore, WriteJSON: writeJSON, WriteError: writeJSONError, Log: logx.Info, LogError: logx.Error}

	// A/B Testing variant endpoints (OT-P0-014 through OT-P0-018)
	// Admin endpoints (require auth)
	s.router.HandleFunc("/api/v1/variants", s.requireAdminStepUp(varianthttp.List(variantReadDependencies(s.configStore, "")))).Methods("GET")
	s.router.HandleFunc("/api/v1/variants/{slug}", s.requireAdminStepUp(varianthttp.AdminGet(variantReadDependencies(s.configStore, "/api/v1/variants/")))).Methods("GET")
	s.router.HandleFunc("/api/v1/variants/{slug}", s.requireAdminStepUp(varianthttp.Update(writeDependencies))).Methods("PATCH")
	s.router.HandleFunc("/api/v1/variants/{slug}", s.requireAdminStepUp(varianthttp.Delete(writeDependencies))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/variants/sync", s.requireAdminStepUp(varianthttp.Sync(writeDependencies))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/variants/{slug}/export", s.requireAdminStepUp(varianthttp.Export(writeDependencies))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/variants/{slug}/import", s.requireAdminStepUp(varianthttp.Import(writeDependencies))).Methods("PUT")

	// Content Customization endpoints - sections are part of variant snapshots
	s.router.HandleFunc("/api/v1/variants/{variant_slug}/sections", s.requireAdminStepUp(contenthttp.Admin(contentHTTPDependencies(s.configStore)))).Methods("GET")
}

func registerContentRoutes(s *Server) {
	varianthttp.RegisterBrandingConnectRoutes(s.router, s.configStore, s.requireAdminStepUp)
	contenthttp.RegisterSEOConnectRoutes(s.router, s.seoService, s.requireAdminStepUp)
	contenthttp.RegisterAssetsConnectRoutes(s.router, s.assetsService, s.requireAdminStepUp)

	// Asset upload endpoints (admin-only for file uploads)
	s.router.HandleFunc("/api/v1/admin/assets/upload", s.requireAdminStepUp(assethttp.Upload(assetsHTTPDependencies(s.assetsService)))).Methods("POST")

	// Serve uploaded files publicly through the request-aware asset root.
	s.router.HandleFunc("/api/v1/uploads/{path:.*}", assethttp.Serve(assetsHTTPDependencies(s.assetsService))).Methods("GET", "HEAD")

	// Sitemap and robots.txt
	s.router.HandleFunc("/sitemap.xml", seohttp.Sitemap(seoHTTPDependencies(s.seoService))).Methods("GET")
	s.router.HandleFunc("/robots.txt", seohttp.Robots(seoHTTPDependencies(s.seoService))).Methods("GET")
}

func registerMetricsRoutes(s *Server) {
	metricshttp.RegisterConnectRoutes(s.router, metricsConnectDependencies(s.metricsService), s.requireMetricsReader)
}

func registerFeedbackRoutes(s *Server) {
	feedbackhttp.RegisterConnectRoutes(s.router, s.feedbackService, feedbackEmailNotifier{configStore: s.configStore, emailService: s.emailService}, s.requireAdminStepUp)
}

func registerWaitlistRoutes(s *Server) {
	metricshttp.RegisterWaitlistConnectRoutes(s.router, metricshttp.WaitlistConnectDependencies{
		Service: s.waitlistService, ValidateEmail: ValidateEmail,
	}, s.requireAdmin)
}

func registerCreditsRoutes(s *Server) {
	// Credit System: Tier Limits (Admin)
	limitsDeps := billingLimitsDependencies()
	s.router.HandleFunc("/api/v1/admin/tiers/limits", s.requireAdmin(billinghttp.GetTierLimits(s.limitsService, limitsDeps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/tiers/{tier}/limits", s.requireAdmin(billinghttp.GetTierLimits(s.limitsService, limitsDeps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/tiers/{tier}/limits", s.requireAdminStepUp(billinghttp.UpdateTierLimits(s.limitsService, limitsDeps))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/limits", s.requireAdminStepUp(billinghttp.CreateTierLimit(s.limitsService, limitsDeps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/limits", s.requireAdminStepUp(billinghttp.DeleteTierLimit(s.limitsService, limitsDeps))).Methods("DELETE")

	// Credit System: App Limits (Admin)
	s.router.HandleFunc("/api/v1/admin/apps/{app}/limits", s.requireAdmin(billinghttp.GetAppLimits(s.limitsService, limitsDeps))).Methods("GET")

	// Credit System: Usage (User Auth + Admin). The write path derives identity
	// from the verified access token; no shared service credential is accepted.
	usageDeps := usageHTTPDependencies()
	s.router.HandleFunc("/api/v1/usage/report", s.requireUserAuth(billinghttp.ReportUsage(s.usageService, usageDeps))).Methods("POST")
	s.router.HandleFunc("/api/v1/usage/reservations", s.requireUserAuth(billinghttp.ReserveCredits(s.usageService, s.accountService, usageDeps))).Methods("POST")
	s.router.HandleFunc("/api/v1/usage/reservations/{reservationID}/finalize", s.requireUserAuth(billinghttp.FinalizeReservation(s.usageService, usageDeps))).Methods("POST")
	s.router.HandleFunc("/api/v1/usage/reservations/{reservationID}/release", s.requireUserAuth(billinghttp.ReleaseReservation(s.usageService, usageDeps))).Methods("POST")
	s.router.HandleFunc("/api/v1/usage/summary", s.requireUserAuth(billinghttp.GetUsageSummary(s.usageService, s.accountService, usageDeps))).Methods("GET")
	s.router.HandleFunc("/api/v1/usage/check", s.requireUserAuth(billinghttp.CheckLimit(s.usageService, usageDeps))).Methods("GET")
	s.router.HandleFunc("/api/v1/usage/health", billinghttp.UsageHealth(s.usageService, usageDeps)).Methods("GET") // Unauthenticated for monitoring
	s.router.HandleFunc("/api/v1/admin/usage", s.requireAdmin(billinghttp.AdminUsageSummary(s.usageService, usageDeps))).Methods("GET")
}

func usageHTTPDependencies() billinghttp.UsageDependencies {
	return billinghttp.UsageDependencies{
		UserEmail:  getUserEmail,
		WriteError: writeJSONError,
		LogError:   logx.Error,
	}
}

func registerAIRoutes(s *Server) {
	intelligencehandler.RegisterConnectRoutes(s.router, s.meteredInferenceDeps, s.requireUserAuth)

	// AI MeteredInferenceProvider endpoints
	// Public endpoint for listing available models
	s.router.HandleFunc("/api/v1/ai/models", s.meteredInferenceHandler.Models()).Methods("GET")
	// Health check (public for monitoring)
	s.router.HandleFunc("/api/v1/ai/health", s.meteredInferenceHandler.Health()).Methods("GET")
	// User auth required for AI operations
	s.router.HandleFunc("/api/v1/ai/chat", s.requireUserAuth(s.meteredInferenceHandler.Chat())).Methods("POST")
	s.router.HandleFunc("/api/v1/ai/inference", s.requireUserAuth(s.meteredInferenceHandler.Inference())).Methods("POST")
	s.router.HandleFunc("/api/v1/ai/stream", s.requireUserAuth(s.meteredInferenceHandler.Stream())).Methods("POST")
	s.router.HandleFunc("/api/v1/ai/usage", s.requireUserAuth(s.meteredInferenceHandler.Usage())).Methods("GET")
}

func registerDocsRoutes(s *Server) {
	// Documentation is a generated Connect service; the UI uses its typed client.
	docshandler.RegisterConnectRoutes(s.router, docsConnectDependencies(), s.requireAdminStepUp)
}

func registerAdminUserRoutes(s *Server) {
	// User Management endpoints (Admin)
	userDeps := userManagementDependencies(s.userManagementService)
	s.router.HandleFunc("/api/v1/admin/users", s.requireAdminStepUp(adminhttp.ListUsers(userDeps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/users/{id}", s.requireAdminStepUp(adminhttp.GetUser(userDeps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/users/{id}/sessions", s.requireAdminStepUp(adminhttp.ListUserSessions(userDeps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/users/{id}/sessions/{sid}", s.requireAdminStepUp(adminhttp.RevokeUserSession(userDeps))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/users/{id}/sessions/revoke-all", s.requireAdminStepUp(adminhttp.RevokeAllUserSessions(userDeps))).Methods("POST")
}
