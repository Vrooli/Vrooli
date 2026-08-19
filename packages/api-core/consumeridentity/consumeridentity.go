// Package consumeridentity implements the portable trust boundary for
// subscription access tokens. A relying scenario receives public keys only;
// signing keys remain in the subscription authority.
package consumeridentity

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	Algorithm       = "RS256"
	DefaultLeeway   = 30 * time.Second
	DefaultCacheTTL = 5 * time.Minute
	maxTokenBytes   = 64 * 1024
)

var (
	ErrMalformed            = errors.New("consumer token is malformed")
	ErrUnsupportedAlgorithm = errors.New("consumer token uses an unsupported algorithm")
	ErrUnknownKey           = errors.New("consumer token uses an unpublished key")
	ErrSignatureInvalid     = errors.New("consumer token signature is invalid")
	ErrExpired              = errors.New("consumer token has expired")
	ErrNotYetValid          = errors.New("consumer token is not yet valid")
	ErrIssuerInvalid        = errors.New("consumer token issuer is invalid")
	ErrKeySetUnavailable    = errors.New("consumer token key set is unavailable")
)

type Claims struct {
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	NotBefore int64  `json:"nbf"`
	UserID    string `json:"uid"`
	Email     string `json:"email"`
	SessionID string `json:"sid"`
}

type PublicKey struct {
	ID  string
	Key *rsa.PublicKey
}

type Signer struct {
	KeyID     string
	Private   *rsa.PrivateKey
	Issuer    string
	AccessTTL time.Duration
	Now       func() time.Time
}

func NewSigner(keyID string, privateKey *rsa.PrivateKey, issuer string, ttl time.Duration) (*Signer, error) {
	if privateKey == nil || privateKey.N == nil || privateKey.N.BitLen() < 2048 {
		return nil, errors.New("consumer signing key must be an RSA key of at least 2048 bits")
	}
	if strings.TrimSpace(keyID) == "" || strings.TrimSpace(issuer) == "" {
		return nil, errors.New("consumer signing key id and issuer are required")
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return &Signer{KeyID: keyID, Private: privateKey, Issuer: issuer, AccessTTL: ttl, Now: time.Now}, nil
}

func GenerateSigner(keyID, issuer string, ttl time.Duration) (*Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate consumer signing key: %w", err)
	}
	return NewSigner(keyID, key, issuer, ttl)
}

func ParsePrivateKeyPEM(raw string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(raw)))
	if block == nil {
		return nil, errors.New("consumer signing key is not PEM encoded")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	return nil, errors.New("consumer signing key is not an RSA private key")
}

func MarshalPrivateKeyPEM(key *rsa.PrivateKey) ([]byte, error) {
	raw, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: raw}), nil
}

func (s *Signer) Sign(claims Claims) (string, time.Time, error) {
	if s == nil || s.Private == nil || !validPublicKey(&s.Private.PublicKey) {
		return "", time.Time{}, errors.New("consumer signing key is unavailable")
	}
	nowFn := s.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()
	expires := now.Add(s.AccessTTL)
	claims.Issuer, claims.IssuedAt, claims.NotBefore, claims.ExpiresAt = s.Issuer, now.Unix(), now.Unix(), expires.Unix()
	header := map[string]string{"alg": Algorithm, "typ": "JWT", "kid": s.KeyID}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", time.Time{}, err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}
	encoded := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(payload)
	hash := sha256.Sum256([]byte(encoded))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.Private, crypto.SHA256, hash[:])
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign consumer token: %w", err)
	}
	return encoded + "." + base64.RawURLEncoding.EncodeToString(signature), expires, nil
}

type KeySet struct {
	Keys map[string]*rsa.PublicKey
}

func NewKeySet(keys ...PublicKey) *KeySet {
	set := &KeySet{Keys: make(map[string]*rsa.PublicKey, len(keys))}
	for _, key := range keys {
		if validPublicKey(key.Key) && strings.TrimSpace(key.ID) != "" {
			set.Keys[key.ID] = key.Key
		}
	}
	return set
}

func (s *KeySet) Add(key PublicKey) {
	if s != nil && validPublicKey(key.Key) && key.ID != "" {
		if s.Keys == nil {
			s.Keys = map[string]*rsa.PublicKey{}
		}
		s.Keys[key.ID] = key.Key
	}
}

func validPublicKey(key *rsa.PublicKey) bool {
	if key == nil || key.N == nil || key.N.BitLen() < 2048 {
		return false
	}
	// The authority generates RSA keys with the standard public exponents.
	// Reject unusual values before crypto/rsa can be asked to process them.
	return key.E == 3 || key.E == 65537
}

type Verifier struct {
	Keys   *KeySet
	Issuer string
	Leeway time.Duration
	Now    func() time.Time
}

func NewVerifier(keys *KeySet, issuer string, leeway time.Duration) *Verifier {
	if leeway < 0 {
		leeway = DefaultLeeway
	}
	return &Verifier{Keys: keys, Issuer: issuer, Leeway: leeway, Now: time.Now}
}

func (v *Verifier) Verify(raw string) (Claims, error) {
	if len(raw) == 0 || len(raw) > maxTokenBytes {
		return Claims{}, ErrMalformed
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return Claims{}, ErrMalformed
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrMalformed
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrMalformed
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, ErrMalformed
	}
	var header struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}
	if json.Unmarshal(headerBytes, &header) != nil || header.KeyID == "" {
		return Claims{}, ErrMalformed
	}
	if header.Algorithm != Algorithm {
		return Claims{}, ErrUnsupportedAlgorithm
	}
	key := (*rsa.PublicKey)(nil)
	if v.Keys != nil {
		key = v.Keys.Keys[header.KeyID]
	}
	if !validPublicKey(key) {
		return Claims{}, ErrUnknownKey
	}
	hash := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], signature); err != nil {
		return Claims{}, ErrSignatureInvalid
	}
	var claims Claims
	if json.Unmarshal(payloadBytes, &claims) != nil || claims.Subject == "" || claims.ExpiresAt == 0 {
		return Claims{}, ErrMalformed
	}
	if v.Issuer != "" && claims.Issuer != v.Issuer {
		return Claims{}, ErrIssuerInvalid
	}
	now := v.Now().UTC()
	if now.After(time.Unix(claims.ExpiresAt, 0).Add(v.Leeway)) {
		return Claims{}, ErrExpired
	}
	if now.Before(time.Unix(claims.NotBefore, 0).Add(-v.Leeway)) {
		return Claims{}, ErrNotYetValid
	}
	return claims, nil
}

type JWKSKey struct {
	KeyType   string `json:"kty"`
	Use       string `json:"use"`
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	Modulus   string `json:"n"`
	Exponent  string `json:"e"`
}
type JWKS struct {
	Keys []JWKSKey `json:"keys"`
}

func (s *KeySet) JWKS() ([]byte, error) {
	result := JWKS{}
	for id, key := range s.Keys {
		if !validPublicKey(key) {
			continue
		}
		result.Keys = append(result.Keys, JWKSKey{KeyType: "RSA", Use: "sig", Algorithm: Algorithm, KeyID: id, Modulus: base64.RawURLEncoding.EncodeToString(key.N.Bytes()), Exponent: base64.RawURLEncoding.EncodeToString(uintBytes(uint64(key.E)))})
	}
	if len(result.Keys) == 0 {
		return nil, errors.New("consumer key set contains no usable keys")
	}
	return json.Marshal(result)
}

func uintBytes(value uint64) []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], value)
	first := 0
	for first < len(buf)-1 && buf[first] == 0 {
		first++
	}
	return buf[first:]
}

func ParseJWKS(raw []byte) (*KeySet, error) {
	var body JWKS
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, fmt.Errorf("parse consumer key set: %w", err)
	}
	set := NewKeySet()
	for _, item := range body.Keys {
		if item.KeyType != "RSA" || item.Algorithm != Algorithm || item.KeyID == "" {
			continue
		}
		n, err := base64.RawURLEncoding.DecodeString(item.Modulus)
		if err != nil {
			return nil, ErrMalformed
		}
		e, err := base64.RawURLEncoding.DecodeString(item.Exponent)
		if err != nil || len(e) == 0 || len(e) > 8 {
			return nil, ErrMalformed
		}
		var exponent uint64
		for _, b := range e {
			exponent = exponent<<8 | uint64(b)
		}
		set.Add(PublicKey{ID: item.KeyID, Key: &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(exponent)}})
	}
	if len(set.Keys) == 0 {
		return nil, errors.New("consumer key set contains no usable keys")
	}
	return set, nil
}

type (
	Fetcher func(*http.Request) ([]byte, error)
	Cache   struct {
		mu              sync.RWMutex
		keys            *KeySet
		issuer          string
		leeway          time.Duration
		fetch           Fetcher
		refreshInterval time.Duration
		refreshedAt     time.Time
	}
)

func NewCache(issuer string, leeway, refreshInterval time.Duration, fetch Fetcher) *Cache {
	if refreshInterval <= 0 {
		refreshInterval = DefaultCacheTTL
	}
	return &Cache{keys: NewKeySet(), issuer: issuer, leeway: leeway, fetch: fetch, refreshInterval: refreshInterval}
}

func (c *Cache) Verify(request *http.Request, token string) (Claims, error) {
	c.mu.RLock()
	keys := c.keys
	refreshed := c.refreshedAt
	c.mu.RUnlock()
	verifier := NewVerifier(keys, c.issuer, c.leeway)
	claims, err := verifier.Verify(token)
	if err == nil {
		return claims, nil
	}
	if c.fetch == nil || (err != ErrUnknownKey && time.Since(refreshed) < c.refreshInterval) {
		return Claims{}, err
	}
	body, fetchErr := c.fetch(request)
	if fetchErr != nil {
		if errors.Is(err, ErrExpired) {
			return Claims{}, fmt.Errorf("%w: %v", ErrKeySetUnavailable, fetchErr)
		}
		return Claims{}, fetchErr
	}
	keys, parseErr := ParseJWKS(body)
	if parseErr != nil {
		return Claims{}, parseErr
	}
	c.mu.Lock()
	c.keys, c.refreshedAt = keys, time.Now()
	c.mu.Unlock()
	claims, err = NewVerifier(keys, c.issuer, c.leeway).Verify(token)
	return claims, err
}
