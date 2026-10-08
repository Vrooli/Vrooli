package push

import (
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
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
