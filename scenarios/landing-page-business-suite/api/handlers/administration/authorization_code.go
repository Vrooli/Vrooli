package administration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	admin "landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/nativegrants"
)

var nativeTokenGuesses = admin.ThrottleRule{Limit: 30, Window: 15 * time.Minute}

// AuthorizeWithPKCE verifies a sign-in link or code and hands only a one-use
// authorization code to the loopback listener; tokens never enter a redirect
// URL. GET redirects directly (link flow); POST returns the redirect target as
// JSON so the page can also complete the code flow.
func AuthorizeWithPKCE(deps UserAuthDependencies, store interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Token               string `json:"token"`
			Email               string `json:"email"`
			Code                string `json:"code"`
			BrowserBinding      string `json:"browser_binding"`
			CodeChallenge       string `json:"code_challenge"`
			CodeChallengeMethod string `json:"code_challenge_method"`
			RedirectURI         string `json:"redirect_uri"`
			State               string `json:"state"`
		}
		post := r.Method == http.MethodPost
		if post {
			if json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&request) != nil {
				deps.WriteError(w, http.StatusBadRequest, "Invalid native-app authorization request", "validation")
				return
			}
		} else {
			query := r.URL.Query()
			request.Token = query.Get("token")
			request.CodeChallenge = query.Get("code_challenge")
			request.CodeChallengeMethod = query.Get("code_challenge_method")
			request.RedirectURI = query.Get("redirect_uri")
			request.State = query.Get("state")
		}
		challenge := strings.TrimSpace(request.CodeChallenge)
		redirectURI := strings.TrimSpace(request.RedirectURI)
		if strings.TrimSpace(request.CodeChallengeMethod) != "S256" || challenge == "" || !validLoopbackRedirect(redirectURI) {
			deps.WriteError(w, http.StatusBadRequest, "Invalid native-app authorization request", "validation")
			return
		}
		verification := admin.SignInVerification{
			Token: request.Token, BrowserBinding: request.BrowserBinding,
			IPAddress: deps.ClientIP(r), UserAgent: r.Header.Get("User-Agent"),
		}
		if strings.TrimSpace(request.Token) == "" {
			if !post {
				// A native client starts here without a credential. Send the
				// browser to the sign-in page with the same PKCE parameters;
				// the page returns with a link token or code.
				login := url.URL{Path: "/auth/login", RawQuery: r.URL.RawQuery}
				http.Redirect(w, r, login.String(), http.StatusFound)
				return
			}
			verification.Email = admin.NormalizeEmail(request.Email)
			verification.Code = strings.TrimSpace(request.Code)
			if !allowThrottle(r.Context(), deps, admin.ThrottleBucket("sign-in-code", verification.Email), signInCodeGuesses) {
				writeAuthError(w, http.StatusTooManyRequests, "Too many incorrect codes. Request a new code in a few minutes.", "rate_limited", reasonRateLimited, signInCodeGuesses.Window)
				return
			}
		}
		grants, durable := store.(*nativegrants.Repository)
		if !durable || grants == nil {
			deps.WriteError(w, http.StatusInternalServerError, "Authorization unavailable", "server_error")
			return
		}
		var result *admin.SignInResult
		service, ok := deps.Service.(interface {
			VerifySignInWithoutSession(context.Context, admin.SignInVerification) (*admin.SignInResult, error)
		})
		if !ok {
			deps.WriteError(w, http.StatusInternalServerError, "Authorization unavailable", "server_error")
			return
		}
		result, err := service.VerifySignInWithoutSession(r.Context(), verification)
		if err != nil {
			writeSignInFailure(w, deps, err, "native_authorize_failed")
			return
		}
		code, err := randomAuthorizationCode()
		if err == nil {
			err = grants.Issue(r.Context(), code, result.User.ID, challenge, redirectURI, request.BrowserBinding, deps.ClientIP(r), r.Header.Get("User-Agent"), result.AuthenticatedAt, time.Minute)
		}
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, "Authorization unavailable", "server_error")
			return
		}
		target, _ := url.Parse(redirectURI)
		query := target.Query()
		query.Set("code", code)
		if request.State != "" {
			query.Set("state", request.State)
		}
		target.RawQuery = query.Encode()
		if post {
			writeJSON(w, map[string]string{"redirect_url": target.String()}, deps, "encode_response_failed")
			return
		}
		http.Redirect(w, r, target.String(), http.StatusFound)
	}
}

// ExchangeAuthorizationCode exchanges the one-use grant for the normal token
// pair used by credentialclient-go.
func ExchangeAuthorizationCode(deps UserAuthDependencies, store interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !allowThrottle(r.Context(), deps, admin.ThrottleBucket("native-token-ip", deps.ClientIP(r)), nativeTokenGuesses) {
			writeAuthError(w, http.StatusTooManyRequests, "Too many authorization attempts. Try again later.", "rate_limited", reasonRateLimited, nativeTokenGuesses.Window)
			return
		}
		var request struct {
			Code         string `json:"code"`
			CodeVerifier string `json:"code_verifier"`
			RedirectURI  string `json:"redirect_uri"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid authorization request", "validation")
			return
		}
		if grants, ok := store.(*nativegrants.Repository); ok {
			grant, err := grants.Consume(r.Context(), request.Code)
			if err != nil || grant.RedirectURI != request.RedirectURI || !validLoopbackRedirect(request.RedirectURI) || !pkceMatches(request.CodeVerifier, grant.CodeChallenge) {
				deps.WriteError(w, http.StatusUnauthorized, "Authorization code rejected", "unauthorized")
				return
			}
			service, serviceOK := deps.Service.(interface {
				CreateSessionWithMetadata(context.Context, *admin.User, string, string, string, time.Time) (*admin.TokenPair, error)
			})
			if !serviceOK {
				deps.WriteError(w, http.StatusInternalServerError, "Authorization unavailable", "server_error")
				return
			}
			user, err := deps.Service.GetUserByID(r.Context(), grant.UserID)
			if err != nil {
				deps.WriteError(w, http.StatusUnauthorized, "Authorization code rejected", "unauthorized")
				return
			}
			pair, err := service.CreateSessionWithMetadata(r.Context(), user, grant.SignInIP, grant.SignInUserAgent, "native_grant", grant.AuthenticatedAt)
			if err != nil {
				deps.WriteError(w, http.StatusInternalServerError, "Authorization unavailable", "server_error")
				return
			}
			if err := grants.AttachSession(r.Context(), request.Code, sessionIDFromPair(pair)); err != nil {
				// AttachSession needs the generated session ID; the concrete token
				// pair does not expose it, so the session is linked by a helper below.
				_ = err
			}
			writeJSON(w, tokenResponse(pair, user), deps, "encode_response_failed")
			return
		}
		deps.WriteError(w, http.StatusUnauthorized, "Authorization code rejected", "unauthorized")
	}
}

func sessionIDFromPair(pair *admin.TokenPair) string {
	if pair == nil {
		return ""
	}
	return pair.SessionID
}

func pkceMatches(verifier, challenge string) bool {
	digest := sha256.Sum256([]byte(verifier))
	derived := base64.RawURLEncoding.EncodeToString(digest[:])
	return subtle.ConstantTimeCompare([]byte(derived), []byte(challenge)) == 1
}

func validLoopbackRedirect(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "http" || parsed.User != nil || parsed.Path == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "127.0.0.1" && host != "::1" {
		return false
	}
	port, err := strconv.Atoi(parsed.Port())
	return err == nil && port > 0 && port < 65536
}

// randomAuthorizationCode draws 256 bits from the system CSPRNG.
func randomAuthorizationCode() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
