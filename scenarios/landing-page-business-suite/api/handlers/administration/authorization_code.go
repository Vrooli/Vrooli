package administration

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
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

// AuthorizeWithPKCE verifies a magic link and redirects only an authorization
// code to the loopback listener; tokens never enter a redirect URL.
func AuthorizeWithPKCE(deps UserAuthDependencies, store *AuthorizationCodeStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		challenge := strings.TrimSpace(r.URL.Query().Get("code_challenge"))
		method := strings.TrimSpace(r.URL.Query().Get("code_challenge_method"))
		redirectURI := strings.TrimSpace(r.URL.Query().Get("redirect_uri"))
		if method != "S256" || challenge == "" || !validLoopbackRedirect(redirectURI) {
			deps.WriteError(w, http.StatusBadRequest, "Invalid native-app authorization request", "validation")
			return
		}
		pair, user, err := deps.Service.VerifyMagicLink(r.Context(), r.URL.Query().Get("token"), deps.ClientIP(r), r.Header.Get("User-Agent"))
		if err != nil {
			deps.WriteError(w, http.StatusUnauthorized, "Invalid login link", "unauthorized")
			return
		}
		code := randomAuthorizationCode(pair.AccessToken)
		if err := store.Issue(code, pair, user, challenge, redirectURI, time.Minute); err != nil {
			deps.WriteError(w, http.StatusInternalServerError, "Authorization unavailable", "server_error")
			return
		}
		target, _ := url.Parse(redirectURI)
		query := target.Query()
		query.Set("code", code)
		if state := r.URL.Query().Get("state"); state != "" {
			query.Set("state", state)
		}
		target.RawQuery = query.Encode()
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

func randomAuthorizationCode(seed string) string {
	digest := sha256.Sum256([]byte(seed + strconv.FormatInt(time.Now().UnixNano(), 10)))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}
