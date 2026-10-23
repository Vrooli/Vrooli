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
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// [REQ:NOTIFICA-P1-008]
func TestSender_EncryptsPayloadAndUsesWebPushHeaders(t *testing.T) {
	clientKey, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	auth := make([]byte, 16)
	_, err = rand.Read(auth)
	require.NoError(t, err)
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "aes128gcm", r.Header.Get("Content-Encoding"))
		require.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))
		require.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "vapid t="))
		ciphertext, readErr := io.ReadAll(r.Body)
		require.NoError(t, readErr)
		require.NotContains(t, string(ciphertext), "private body")
		w.Header().Set("Location", "provider-receipt")
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	sender := &Sender{Client: server.Client(), PrivateKey: privateKey, PublicKey: elliptic.Marshal(elliptic.P256(), privateKey.PublicKey.X, privateKey.PublicKey.Y), Subject: "mailto:owner@example.test", Now: func() time.Time { return time.Unix(100, 0).UTC() }}
	provider, err := sender.Send(context.Background(), Subscription{Endpoint: server.URL, P256DH: base64.RawURLEncoding.EncodeToString(clientKey.PublicKey().Bytes()), Auth: base64.RawURLEncoding.EncodeToString(auth)}, "title", "private body")
	require.NoError(t, err)
	require.Equal(t, "provider-receipt", provider)
}

func TestSenderErrorsDoNotRetainProviderEndpointOrResponseDetails(t *testing.T) {
	clientKey, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	auth := make([]byte, 16)
	_, err = rand.Read(auth)
	require.NoError(t, err)
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	publicKey := elliptic.Marshal(elliptic.P256(), privateKey.PublicKey.X, privateKey.PublicKey.Y)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("https://provider.example/push?token=secret credential=private"))
	}))
	endpoint := server.URL + "/subscription/private-token"
	subscription := Subscription{
		Endpoint: endpoint,
		P256DH:   base64.RawURLEncoding.EncodeToString(clientKey.PublicKey().Bytes()),
		Auth:     base64.RawURLEncoding.EncodeToString(auth),
		Origin:   "https://hub.example.test",
	}
	sender := &Sender{Client: server.Client(), PrivateKey: privateKey, PublicKey: publicKey}

	_, err = sender.Send(context.Background(), subscription, "title", "body")
	require.EqualError(t, err, "push provider returned status 502")
	require.NotContains(t, err.Error(), "provider.example")
	require.NotContains(t, err.Error(), "secret")

	server.Close()
	_, err = sender.Send(context.Background(), subscription, "title", "body")
	require.EqualError(t, err, "push provider request failed")
	require.NotContains(t, err.Error(), endpoint)
}

func TestEncryptRespectsWebPushBodyLimitAndDecryptsAtMaximum(t *testing.T) {
	clientKey, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	auth := make([]byte, 16)
	_, err = rand.Read(auth)
	require.NoError(t, err)

	maximum := make([]byte, 3993)
	for i := range maximum {
		maximum[i] = byte(i % 251)
	}
	ciphertext, err := encrypt(maximum, clientKey.PublicKey().Bytes(), auth)
	require.NoError(t, err)
	require.Len(t, ciphertext, 4096)
	decrypted, err := decryptWebPushForTest(ciphertext, clientKey, auth)
	require.NoError(t, err)
	require.Equal(t, append(append([]byte(nil), maximum...), 0x02), decrypted)

	tooLarge := make([]byte, 3994)
	require.Len(t, tooLarge, 3994)
	ciphertext, err = encrypt(tooLarge, clientKey.PublicKey().Bytes(), auth)
	require.Nil(t, ciphertext)
	require.EqualError(t, err, "web push payload exceeds record size")
}

func TestRFC8291PublishedEncryptionVectorDecrypts(t *testing.T) {
	// RFC 8291 section 5's complete aes128gcm body, with the Appendix A
	// receiver private key and auth secret. This independently exercises the
	// key derivation and record parsing against the published wire vector.
	body := decodeRawURLTest(t, "DGv6ra1nlYgDCS1FRnbzlwAAEABBBP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27ml"+
		"mlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A_yl95bQpu6cVPT"+
		"pK4Mqgkf1CXztLVBSt2Ks3oZwbuwXPXLWyouBWLVWGNWQexSgSxsj_Qulcy4a-fN")
	userAgentPrivate := decodeRawURLTest(t, "q1dXpw3UpT5VOmu_cf_v6ih07Aems3njxI-JWgLcM94")
	receiverKey, err := ecdh.P256().NewPrivateKey(userAgentPrivate)
	require.NoError(t, err)
	auth := decodeRawURLTest(t, "BTBZMqHH6r4Tts7J_aSIgg")

	plaintext, err := decryptWebPushForTest(body, receiverKey, auth)
	require.NoError(t, err)
	require.Equal(t, []byte("When I grow up, I want to be a watermelon\x02"), plaintext)
}

func decryptWebPushForTest(body []byte, receiver *ecdh.PrivateKey, auth []byte) ([]byte, error) {
	if len(body) < 86+17 {
		return nil, errors.New("encrypted body is shorter than the Web Push header and record minimum")
	}
	if len(body) > 4096 {
		return nil, errors.New("encrypted body exceeds the RFC 8291 maximum")
	}
	if binary.BigEndian.Uint32(body[16:20]) != 4096 {
		return nil, errors.New("unexpected Web Push record size")
	}
	idLen := int(body[20])
	if idLen != 65 || 21+idLen != 86 || 21+idLen > len(body) {
		return nil, errors.New("invalid Web Push key identifier")
	}
	salt := body[:16]
	serverKey, err := ecdh.P256().NewPublicKey(body[21 : 21+idLen])
	if err != nil {
		return nil, err
	}
	shared, err := receiver.ECDH(serverKey)
	if err != nil {
		return nil, err
	}
	userAgentPublic := receiver.PublicKey().Bytes()
	serverPublic := serverKey.Bytes()
	prkKey := hkdfExtractTest(auth, shared)
	keyInfo := append(append([]byte("WebPush: info\x00"), userAgentPublic...), serverPublic...)
	ikm := hkdfExpandTest(prkKey, keyInfo, 32)
	prk := hkdfExtractTest(salt, ikm)
	cek := hkdfExpandTest(prk, []byte("Content-Encoding: aes128gcm\x00"), 16)
	nonce := hkdfExpandTest(prk, []byte("Content-Encoding: nonce\x00"), 12)
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, body[21+idLen:], nil)
}

func hkdfExtractTest(salt, input []byte) []byte {
	mac := hmac.New(sha256.New, salt)
	_, _ = mac.Write(input)
	return mac.Sum(nil)
}

func hkdfExpandTest(prk, info []byte, length int) []byte {
	var result, previous []byte
	for counter := byte(1); len(result) < length; counter++ {
		mac := hmac.New(sha256.New, prk)
		_, _ = mac.Write(previous)
		_, _ = mac.Write(info)
		_, _ = mac.Write([]byte{counter})
		previous = mac.Sum(nil)
		result = append(result, previous...)
	}
	return result[:length]
}

func decodeRawURLTest(t *testing.T, encoded string) []byte {
	t.Helper()
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	return decoded
}

func TestLoadOrCreatePrivateKeyIsStableAndPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vapid-private-key")
	first, err := LoadOrCreatePrivateKey(path)
	require.NoError(t, err)
	second, err := LoadOrCreatePrivateKey(path)
	require.NoError(t, err)
	require.Equal(t, first, second)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	sender, err := NewFromValues(first, "", "mailto:owner@example.test")
	require.NoError(t, err)
	require.NotEmpty(t, sender.PublicKeyValue())
}

func TestStructuredPayloadCarriesTheDeepLinkForTheServiceWorker(t *testing.T) {
	payload := structuredPayload(Message{Title: "Decision needed", Body: "Close or re-aim?", ID: "notification-1", URL: "asks/ask-1"})
	require.JSONEq(t, `{"title":"Decision needed","body":"Close or re-aim?","id":"notification-1","url":"asks/ask-1"}`, payload)
}

func TestVAPIDSubjectReplacesLocalhostContactWithHTTPSOrigin(t *testing.T) {
	// Apple's push service answers 403 BadJwtToken for a localhost contact.
	require.Equal(t, "https://hub.example", vapidSubject("mailto:notification-hub@localhost", "https://hub.example"))
	require.Equal(t, "https://hub.example", vapidSubject("https://localhost:8080", "https://hub.example/"))
	require.Equal(t, "mailto:ops@example.com", vapidSubject("mailto:ops@example.com", "https://hub.example"))
	require.Equal(t, "mailto:notification-hub@localhost", vapidSubject("mailto:notification-hub@localhost", "http://127.0.0.1:20001"))
}
