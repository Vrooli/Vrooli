// Package push contains the dependency-free Web Push transport.
//
// The implementation deliberately keeps provider state out of the durable
// notification service: RFC 8291 encryption and VAPID authentication happen
// at the transport boundary, while the hub records only safe delivery facts.
package push

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/storage"
)

type GoneError struct{}

func (GoneError) Error() string { return "web push subscription is gone" }
func (GoneError) Gone() bool    { return true }

type Subscription struct {
	Endpoint string
	P256DH   string
	Auth     string
}

type Sender struct {
	Client     *http.Client
	PrivateKey *ecdsa.PrivateKey
	PublicKey  []byte
	Subject    string
	TTL        uint32
	Now        func() time.Time
}

func NewFromEnvironment(getenv func(string) string) (*Sender, error) {
	if getenv == nil {
		return nil, errors.New("environment reader is required")
	}
	privateValue := strings.TrimSpace(getenv("VROOLI_WEBPUSH_VAPID_PRIVATE_KEY"))
	subject := strings.TrimSpace(getenv("VROOLI_WEBPUSH_VAPID_SUBJECT"))
	if privateValue == "" || subject == "" {
		return nil, errors.New("VROOLI_WEBPUSH_VAPID_PRIVATE_KEY and VROOLI_WEBPUSH_VAPID_SUBJECT are required")
	}
	privateKey, err := parsePrivateKey(privateValue)
	if err != nil {
		return nil, fmt.Errorf("parse VAPID private key: %w", err)
	}
	publicKey := elliptic.Marshal(elliptic.P256(), privateKey.PublicKey.X, privateKey.PublicKey.Y)
	if value := strings.TrimSpace(getenv("VROOLI_WEBPUSH_VAPID_PUBLIC_KEY")); value != "" {
		publicKey, err = decodeBase64(value)
		if err != nil {
			return nil, fmt.Errorf("parse VAPID public key: %w", err)
		}
	}
	return newSender(privateKey, publicKey, subject), nil
}

// NewFromValues constructs a sender from already-resolved key material. The
// private value may be a raw base64url scalar or EC PEM; callers must keep it
// in the scenario's secret storage boundary.
func NewFromValues(privateValue, publicValue, subject string) (*Sender, error) {
	privateValue = strings.TrimSpace(privateValue)
	subject = strings.TrimSpace(subject)
	if privateValue == "" || subject == "" {
		return nil, errors.New("VAPID private key and subject are required")
	}
	privateKey, err := parsePrivateKey(privateValue)
	if err != nil {
		return nil, fmt.Errorf("parse VAPID private key: %w", err)
	}
	publicKey := elliptic.Marshal(elliptic.P256(), privateKey.PublicKey.X, privateKey.PublicKey.Y)
	if strings.TrimSpace(publicValue) != "" {
		publicKey, err = decodeBase64(publicValue)
		if err != nil {
			return nil, fmt.Errorf("parse VAPID public key: %w", err)
		}
	}
	return newSender(privateKey, publicKey, subject), nil
}

// GeneratePrivateKeyValue returns a base64url-encoded P-256 private scalar
// suitable for storage in the scenario's sensitive state entry.
func GeneratePrivateKeyValue() (string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("generate VAPID key: %w", err)
	}
	raw := make([]byte, 32)
	key.D.FillBytes(raw)
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// LoadOrCreatePrivateKey keeps the scenario-owned VAPID scalar stable across
// restarts. The caller supplies a path already resolved through api-core's
// storage seam; this helper never returns the key through a log or API.
func LoadOrCreatePrivateKey(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("VAPID private-key path is required")
	}
	contents, err := os.ReadFile(path) // #nosec G304 -- caller resolves the path through api-core storage.
	if err == nil {
		value := strings.TrimSpace(string(contents))
		if value == "" {
			return "", errors.New("VAPID private-key file is empty")
		}
		return value, nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("read VAPID private-key file: %w", err)
	}
	value, err := GeneratePrivateKeyValue()
	if err != nil {
		return "", err
	}
	if err := storage.WriteFileAtomic(path, []byte(value+"\n"), storage.SecretFilePerm); err != nil {
		return "", fmt.Errorf("persist VAPID private-key file: %w", err)
	}
	return value, nil
}

// PublicKeyValue returns the public VAPID key in the browser wire encoding.
func (s *Sender) PublicKeyValue() string {
	if s == nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(s.PublicKey)
}

func newSender(privateKey *ecdsa.PrivateKey, publicKey []byte, subject string) *Sender {
	return &Sender{Client: http.DefaultClient, PrivateKey: privateKey, PublicKey: publicKey, Subject: strings.TrimSpace(subject), TTL: 86400, Now: time.Now}
}

func (s *Sender) Send(ctx context.Context, subscription Subscription, title, body string) (string, error) {
	if s == nil || s.PrivateKey == nil || len(s.PublicKey) == 0 {
		return "", errors.New("web push sender is not configured")
	}
	if s.Client == nil {
		s.Client = http.DefaultClient
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	u, err := url.Parse(subscription.Endpoint)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid push endpoint")
	}
	clientKey, err := decodeBase64(subscription.P256DH)
	if err != nil {
		return "", fmt.Errorf("decode subscription public key: %w", err)
	}
	auth, err := decodeBase64(subscription.Auth)
	if err != nil {
		return "", fmt.Errorf("decode subscription auth secret: %w", err)
	}
	encrypted, err := encrypt([]byte(structuredPayload(title, body)), clientKey, auth)
	if err != nil {
		return "", err
	}
	jwt, err := s.vapidJWT(u)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, subscription.Endpoint, strings.NewReader(string(encrypted)))
	if err != nil {
		return "", err
	}
	ttl := s.TTL
	if ttl == 0 {
		ttl = 86400
	}
	req.Header.Set("Authorization", "vapid t="+jwt+", k="+base64.RawURLEncoding.EncodeToString(s.PublicKey))
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("TTL", fmt.Sprintf("%d", ttl))
	resp, err := s.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
		return "", GoneError{}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("push provider returned status %d", resp.StatusCode)
	}
	provider := resp.Header.Get("Location")
	if provider == "" {
		provider = subscription.Endpoint
	}
	return provider, nil
}

func structuredPayload(title, body string) string {
	encoded, _ := json.Marshal(map[string]string{"title": title, "body": body})
	return string(encoded)
}

func (s *Sender) vapidJWT(endpoint *url.URL) (string, error) {
	now := s.Now().UTC()
	header := encodeJSON(map[string]string{"alg": "ES256", "typ": "JWT"})
	claims := encodeJSON(map[string]any{"aud": endpoint.Scheme + "://" + endpoint.Host, "exp": now.Add(12 * time.Hour).Unix(), "sub": s.Subject})
	unsigned := header + "." + claims
	digest := sha256.Sum256([]byte(unsigned))
	r, ss, err := ecdsa.Sign(rand.Reader, s.PrivateKey, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign VAPID token: %w", err)
	}
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	ss.FillBytes(signature[32:])
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func encodeJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func parsePrivateKey(value string) (*ecdsa.PrivateKey, error) {
	if block, _ := pem.Decode([]byte(value)); block != nil {
		if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
			return key, nil
		}
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		key, ok := parsed.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("VAPID key is not an ECDSA key")
		}
		return key, nil
	}
	raw, err := decodeBase64(value)
	if err != nil || len(raw) != 32 {
		return nil, errors.New("VAPID private key must be a 32-byte base64url value or EC PEM")
	}
	x, y := elliptic.P256().ScalarBaseMult(raw)
	return &ecdsa.PrivateKey{PublicKey: ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}, D: new(big.Int).SetBytes(raw)}, nil
}

func decodeBase64(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err == nil {
		return decoded, nil
	}
	return base64.URLEncoding.DecodeString(value)
}

func encrypt(plaintext, clientKey, auth []byte) ([]byte, error) {
	clientPublic, err := ecdh.P256().NewPublicKey(clientKey)
	if err != nil {
		return nil, fmt.Errorf("parse subscription public key: %w", err)
	}
	ephemeral, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ephemeral push key: %w", err)
	}
	shared, err := ephemeral.ECDH(clientPublic)
	if err != nil {
		return nil, fmt.Errorf("derive push shared secret: %w", err)
	}
	prk := hkdfExtract(auth, shared)
	keyInfo := append(append([]byte("WebPush: info\x00"), clientKey...), ephemeral.PublicKey().Bytes()...)
	ikm := hkdfExpand(prk, keyInfo, 32)
	// The salt is generated once and must be the same salt returned in the body.
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	prk = hkdfExtract(salt, ikm)
	cek := hkdfExpand(prk, []byte("Content-Encoding: aes128gcm\x00"), 16)
	nonce := hkdfExpand(prk, []byte("Content-Encoding: nonce\x00"), 12)
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(plaintext)+1+gcm.Overhead() > 4096 {
		return nil, errors.New("web push payload exceeds record size")
	}
	padded := append(append([]byte{}, plaintext...), 2)
	ciphertext := gcm.Seal(nil, nonce, padded, nil)
	out := append([]byte{}, salt...)
	recordSize := make([]byte, 4)
	binary.BigEndian.PutUint32(recordSize, 4096)
	out = append(out, recordSize...)
	out = append(out, byte(len(ephemeral.PublicKey().Bytes())))
	out = append(out, ephemeral.PublicKey().Bytes()...)
	return append(out, ciphertext...), nil
}

func hkdfExtract(salt, input []byte) []byte {
	mac := hmac.New(sha256.New, salt)
	_, _ = mac.Write(input)
	return mac.Sum(nil)
}

func hkdfExpand(prk, info []byte, length int) []byte {
	var output []byte
	var previous []byte
	for counter := byte(1); len(output) < length; counter++ {
		mac := hmac.New(sha256.New, prk)
		_, _ = mac.Write(previous)
		_, _ = mac.Write(info)
		_, _ = mac.Write([]byte{counter})
		previous = mac.Sum(nil)
		output = append(output, previous...)
	}
	return output[:length]
}
