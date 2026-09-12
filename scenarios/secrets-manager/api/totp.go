package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"hash"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

func generateTOTPCode(rawSecret string, now time.Time, period, digits int, algorithm string) (string, error) {
	secret, err := decodeTOTPSecret(rawSecret)
	if err != nil {
		return "", err
	}
	if period < 1 || period > 300 {
		return "", errors.New("totp period must be between 1 and 300 seconds")
	}
	if digits != 6 && digits != 8 {
		return "", errors.New("totp digits must be 6 or 8")
	}
	var newHash func() hash.Hash
	switch strings.ToUpper(strings.TrimSpace(algorithm)) {
	case "", "SHA1":
		newHash = sha1.New
	case "SHA256":
		newHash = sha256.New
	case "SHA512":
		newHash = sha512.New
	default:
		return "", errors.New("totp algorithm must be SHA1, SHA256, or SHA512")
	}
	counter := uint64(now.UTC().Unix() / int64(period))
	var counterBytes [8]byte
	binary.BigEndian.PutUint64(counterBytes[:], counter)
	mac := hmac.New(newHash, secret)
	_, _ = mac.Write(counterBytes[:])
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 0x0f
	value := (uint32(digest[offset])&0x7f)<<24 |
		uint32(digest[offset+1])<<16 |
		uint32(digest[offset+2])<<8 |
		uint32(digest[offset+3])
	modulus := uint32(1000000)
	if digits == 8 {
		modulus = 100000000
	}
	return leftPadNumber(value%modulus, digits), nil
}

func decodeTOTPSecret(raw string) ([]byte, error) {
	value := strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToLower(value), "otpauth://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return nil, errors.New("totp secret URI is invalid")
		}
		value = parsed.Query().Get("secret")
	}
	value = strings.ToUpper(strings.NewReplacer(" ", "", "-", "").Replace(value))
	if value == "" {
		return nil, errors.New("totp secret is required")
	}
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.TrimRight(value, "="))
	if err != nil || len(decoded) < 10 {
		return nil, errors.New("totp secret must be a valid base32 value")
	}
	return decoded, nil
}

func leftPadNumber(value uint32, digits int) string {
	return strings.Repeat("0", maxInt(digits-len(strconv.FormatUint(uint64(value), 10)), 0)) + strconv.FormatUint(uint64(value), 10)
}

func (h *passwordManagerHandlers) totpCode(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilitySecretReveal); err != nil {
		writeVaultError(w, err)
		return
	}
	var input struct {
		AssuranceToken string `json:"assurance_token"`
		RequestDigest  string `json:"request_digest"`
		Period         int    `json:"period"`
		Digits         int    `json:"digits"`
		Algorithm      string `json:"algorithm"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	vars := mux.Vars(r)
	fields, err := h.manager.reveal(r.Context(), workspace, vars["vault"], vars["id"], actor, input.AssuranceToken, input.RequestDigest, []string{"totp"})
	if err != nil {
		writeVaultError(w, err)
		return
	}
	if input.Period == 0 {
		input.Period = 30
	}
	if input.Digits == 0 {
		input.Digits = 6
	}
	code, err := generateTOTPCode(fields["totp"], time.Now().UTC(), input.Period, input.Digits, input.Algorithm)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	h.manager.recordAudit(r.Context(), auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "totp.generate", ItemID: vars["id"], Outcome: "success", Detail: "trusted-boundary code generation"})
	writeJSON(w, http.StatusOK, map[string]any{"item_id": vars["id"], "code": code, "digits": input.Digits, "period": input.Period, "expires_at": time.Now().UTC().Truncate(time.Duration(input.Period) * time.Second).Add(time.Duration(input.Period) * time.Second)})
}
