package administration

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	admin "landing-page-business-suite-api/internal/administration"
)

var (
	errInvalidAuthorizationRequest = errors.New("invalid authorization request")
	errAuthorizationCodeExpired    = errors.New("authorization code expired")
	errAuthorizationCodeUsed       = errors.New("authorization code already used")
	errInvalidCodeVerifier         = errors.New("invalid code verifier")
)

// AuthorizationCodeStore holds one-use native-app grants in memory. The code
// is not a credential; it is bound to the PKCE verifier and expires quickly.
type AuthorizationCodeStore struct {
	mu    sync.Mutex
	codes map[string]authorizationGrant
	now   func() time.Time
}

type authorizationGrant struct {
	pair        *admin.TokenPair
	user        *admin.User
	challenge   string
	redirectURI string
	expiresAt   time.Time
	used        bool
}

// NewAuthorizationCodeStore creates the process-local native-app grant store.
func NewAuthorizationCodeStore() *AuthorizationCodeStore {
	return &AuthorizationCodeStore{codes: make(map[string]authorizationGrant), now: time.Now}
}

// Issue stores a single-use authorization code bound to a PKCE challenge.
func (s *AuthorizationCodeStore) Issue(code string, pair *admin.TokenPair, user *admin.User, challenge, redirectURI string, ttl time.Duration) error {
	if s == nil || strings.TrimSpace(code) == "" || pair == nil || !validLoopbackRedirect(redirectURI) || strings.TrimSpace(challenge) == "" {
		return errInvalidAuthorizationRequest
	}
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for existing, grant := range s.codes {
		if !s.now().Before(grant.expiresAt) {
			delete(s.codes, existing)
		}
	}
	s.codes[code] = authorizationGrant{pair: pair, user: user, challenge: challenge, redirectURI: redirectURI, expiresAt: s.now().Add(ttl)}
	return nil
}

// Exchange consumes a code only after its verifier and redirect match.
func (s *AuthorizationCodeStore) Exchange(code, verifier, redirectURI string) (*admin.TokenPair, *admin.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	grant, ok := s.codes[code]
	if !ok {
		return nil, nil, errAuthorizationCodeExpired
	}
	if grant.used {
		return nil, nil, errAuthorizationCodeUsed
	}
	if !s.now().Before(grant.expiresAt) {
		delete(s.codes, code)
		return nil, nil, errAuthorizationCodeExpired
	}
	if redirectURI != grant.redirectURI || !validLoopbackRedirect(redirectURI) || !pkceMatches(verifier, grant.challenge) {
		return nil, nil, errInvalidCodeVerifier
	}
	grant.used = true
	s.codes[code] = grant
	return grant.pair, grant.user, nil
}

// AuthorizeWithPKCE verifies a sign-in link or code and hands only a one-use
// authorization code to the loopback listener; tokens never enter a redirect
// URL. GET redirects directly (link flow); POST returns the redirect target as
// JSON so the page can also complete the code flow.
func AuthorizeWithPKCE(deps UserAuthDependencies, store *AuthorizationCodeStore) http.HandlerFunc {
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
		result, err := deps.Service.VerifySignIn(r.Context(), verification)
		if err != nil {
			writeSignInFailure(w, deps, err, "native_authorize_failed")
			return
		}
		code, err := randomAuthorizationCode()
		if err == nil {
			err = store.Issue(code, result.Tokens, result.User, challenge, redirectURI, time.Minute)
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
func ExchangeAuthorizationCode(deps UserAuthDependencies, store *AuthorizationCodeStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Code         string `json:"code"`
			CodeVerifier string `json:"code_verifier"`
			RedirectURI  string `json:"redirect_uri"`
		}
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid authorization request", "validation")
			return
		}
		pair, user, err := store.Exchange(request.Code, request.CodeVerifier, request.RedirectURI)
		if err != nil {
			deps.WriteError(w, http.StatusUnauthorized, "Authorization code rejected", "unauthorized")
			return
		}
		writeJSON(w, tokenResponse(pair, user), deps, "encode_response_failed")
	}
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
	if host != "127.0.0.1" && host != "[::1]" && host != "::1" {
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
