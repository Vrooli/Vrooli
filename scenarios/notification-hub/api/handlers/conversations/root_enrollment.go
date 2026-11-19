package conversations

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"notification-hub/internal/hub"
	"notification-hub/internal/identity"
)

type enrollmentCredentials struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	MFAChallenge string `json:"mfaChallenge"`
	TOTPCode     string `json:"totpCode"`
	RecoveryCode string `json:"recoveryCode"`
}

// rootEnrollment requires both a current, specifically authorized Access owner
// and freshly verified account credentials. The association is deliberately
// limited to root decision answers; it is not a host policy or Hub login grant.
func (h *handler) rootEnrollment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	origin, err := h.service.EnrollmentOrigin(r.Context())
	if err != nil || origin == "" || r.Header.Get("Origin") != origin || !strings.HasPrefix(origin, "https://") {
		http.Error(w, "trusted secure origin required", http.StatusForbidden)
		return
	}
	subject := identity.CloudflareSubject(r.Context())
	recipient := h.service.ResolveRecipient(r.Context())
	bindings, err := identity.OwnerBindings()
	if err != nil || !cloudflareRootOwnerAllowed(bindings, subject, recipient) {
		http.Error(w, "authorized owner required", http.StatusUnauthorized)
		return
	}
	if r.Method == http.MethodDelete {
		if err := h.service.RevokeRootEnrollments(r.Context(), recipient, subject); err != nil {
			http.Error(w, "enrollment unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	mediaType, _, mediaErr := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if h.stepUp == nil || mediaErr != nil || mediaType != "application/json" {
		http.Error(w, "enrollment unavailable", http.StatusBadRequest)
		return
	}
	var input enrollmentCredentials
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	generation, err := h.service.RootEnrollmentGeneration(r.Context(), recipient, subject)
	if err == nil && input.MFAChallenge != "" {
		var challenge string
		var original int64
		original, challenge, err = unwrapEnrollmentChallenge(input.MFAChallenge, subject, recipient)
		if err == nil && original != generation {
			err = errors.New("enrollment changed")
		}
		generation = original
		input.MFAChallenge = challenge
	}
	if err != nil {
		http.Error(w, "enrollment unavailable", http.StatusServiceUnavailable)
		return
	}
	digestInput, _ := json.Marshal([]string{"notification-hub/root-enrollment/v1", subject, recipient, strconv.FormatInt(generation, 10)})
	digest := sha256.Sum256(digestInput)
	proof, err := h.stepUp.VerifyDecision(r.Context(), input.Email, input.Password, input.MFAChallenge, input.TOTPCode, input.RecoveryCode, hex.EncodeToString(digest[:]))
	if err != nil {
		if errors.Is(err, identity.ErrRateLimited) {
			http.Error(w, "try again later", http.StatusTooManyRequests)
			return
		}
		if errors.Is(err, identity.ErrAuthUnavailable) {
			http.Error(w, "sign-in unavailable", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "sign-in failed", http.StatusUnauthorized)
		return
	}
	if proof.MFARequired {
		challenge, err := wrapEnrollmentChallenge(proof.MFAChallenge, subject, recipient, generation)
		if err != nil {
			http.Error(w, "enrollment unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"mfaRequired": true, "mfaChallenge": challenge})
		return
	}
	// Policy may have been revoked during the credential check.
	bindings, err = identity.OwnerBindings()
	if err != nil || !cloudflareRootOwnerAllowed(bindings, subject, recipient) || strings.TrimSpace(proof.Subject) == "" {
		http.Error(w, "sign-in failed", http.StatusUnauthorized)
		return
	}
	if err := h.service.EnrollRootAccount(r.Context(), recipient, subject, proof.Subject, generation); err != nil {
		http.Error(w, "enrollment unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"mfaRequired":false}`))
}

// Challenges are process-local, short-lived and bound to the exact owner and
// revocation generation. A restart requires starting enrollment again.
var enrollmentChallengeKey = func() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil
	}
	return key
}()

type enrollmentChallenge struct {
	Challenge, Owner, Recipient string
	Generation                  int64
	Expires                     int64
	Grants                      []hub.RootEnrollmentGrant
}

func wrapEnrollmentChallenge(challenge, owner, recipient string, generation int64) (string, error) {
	if len(enrollmentChallengeKey) == 0 || challenge == "" {
		return "", errors.New("challenge unavailable")
	}
	return wrapChallengeWithGrants(challenge, owner, recipient, generation, nil)
}
func challengeCipher() (cipher.AEAD, error) {
	if len(enrollmentChallengeKey) == 0 {
		return nil, errors.New("challenge unavailable")
	}
	block, err := aes.NewCipher(enrollmentChallengeKey)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
func wrapChallengeWithGrants(challenge, owner, recipient string, generation int64, grants []hub.RootEnrollmentGrant) (string, error) {
	if challenge == "" {
		return "", errors.New("challenge unavailable")
	}
	aead, err := challengeCipher()
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(enrollmentChallenge{Challenge: challenge, Owner: owner, Recipient: recipient, Generation: generation, Expires: time.Now().Add(5 * time.Minute).Unix(), Grants: grants})
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(aead.Seal(nonce, nonce, body, nil)), nil
}
func unwrapEnrollmentChallenge(token, owner, recipient string) (int64, string, error) {
	c, err := unwrapChallengeWithGrants(token, owner, recipient)
	return c.Generation, c.Challenge, err
}
func unwrapChallengeWithGrants(token, owner, recipient string) (enrollmentChallenge, error) {
	aead, err := challengeCipher()
	if err != nil {
		return enrollmentChallenge{}, err
	}
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(data) < aead.NonceSize() {
		return enrollmentChallenge{}, errors.New("invalid challenge")
	}
	body, err := aead.Open(nil, data[:aead.NonceSize()], data[aead.NonceSize():], nil)
	if err != nil {
		return enrollmentChallenge{}, errors.New("invalid challenge")
	}
	var c enrollmentChallenge
	if err := json.Unmarshal(body, &c); err != nil || c.Owner != owner || c.Recipient != recipient || c.Challenge == "" || time.Now().Unix() >= c.Expires {
		return enrollmentChallenge{}, errors.New("invalid challenge")
	}
	return c, nil
}
