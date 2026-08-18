package androidtvremote

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"
	"time"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

func testServerCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	now := time.Now().UTC()
	der, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "fixture-tv"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}, &x509.Certificate{
		SerialNumber: big.NewInt(2), NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour),
	}, &key.PublicKey, key)
	require.NoError(t, err)
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	pair, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))
	require.NoError(t, err)
	return pair
}

func TestRemoteWireClientCompletesTLSConfigurationAndSendsKey(t *testing.T) {
	bundle, _ /* client cert */, err := newCertificateBundle()
	require.NoError(t, err)
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{testServerCertificate(t)}, ClientAuth: tls.RequireAnyClientCert})
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	serverDone := make(chan error, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverDone <- acceptErr
			return
		}
		defer conn.Close()
		tlsConn := conn.(*tls.Conn)
		if err := tlsConn.Handshake(); err != nil {
			serverDone <- err
			return
		}
		ctx := context.Background()
		if err := writeFrame(ctx, tlsConn, remoteConfigureMessage()); err != nil {
			serverDone <- err
			return
		}
		frame, readErr := readFrame(ctx, tlsConn)
		if readErr != nil || !hasBytesField(frame, 1) {
			serverDone <- fmt.Errorf("first client frame was not remote configuration: %v", readErr)
			return
		}
		if err := writeFrame(ctx, tlsConn, remoteSetActiveMessage(androidTVActiveFeatures)); err != nil {
			serverDone <- err
			return
		}
		frame, readErr = readFrame(ctx, tlsConn)
		if readErr != nil || !hasBytesField(frame, 2) {
			serverDone <- fmt.Errorf("second client frame was not remote activation: %v", readErr)
			return
		}
		if err := writeFrame(ctx, tlsConn, remoteStartMessage(true)); err != nil {
			serverDone <- err
			return
		}
		frame, readErr = readFrame(ctx, tlsConn)
		if readErr != nil || !hasBytesField(frame, 10) {
			serverDone <- fmt.Errorf("third client frame was not key injection: %v", readErr)
			return
		}
		serverDone <- nil
	}()

	client, err := newWireClient(listener.Addr().String(), bundle)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.Key(ctx, "DPAD_CENTER"))
	require.NoError(t, <-serverDone)
}

func TestMediaKeyCodeMapsRemoteActionsAndVolumeChanges(t *testing.T) {
	for _, test := range []struct {
		name    string
		command strategy.MediaCommand
		want    uint64
	}{
		{name: "play", command: strategy.MediaCommand{Action: "play"}, want: 126},
		{name: "pause", command: strategy.MediaCommand{Action: "pause"}, want: 127},
		{name: "volume up", command: strategy.MediaCommand{Action: "volume", Value: "up"}, want: 24},
		{name: "volume down", command: strategy.MediaCommand{Action: "volume-down"}, want: 25},
		{name: "mute", command: strategy.MediaCommand{Action: "volume", Value: "mute"}, want: 164},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := mediaKeyCode(test.command)
			require.True(t, ok)
			require.Equal(t, test.want, got)
		})
	}

	_, ok := mediaKeyCode(strategy.MediaCommand{Action: "volume", Value: 0.5})
	require.False(t, ok, "Android TV Remote exposes relative volume keys, not an absolute volume level")
}

func TestPairingWireClientCompletesFixtureExchangeAndReturnsBundle(t *testing.T) {
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{testServerCertificate(t)}, ClientAuth: tls.RequireAnyClientCert})
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	serverDone := make(chan error, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverDone <- acceptErr
			return
		}
		defer conn.Close()
		tlsConn := conn.(*tls.Conn)
		if err := tlsConn.Handshake(); err != nil {
			serverDone <- err
			return
		}
		ctx := context.Background()
		steps := []struct {
			requestField protowire.Number
			response     []byte
		}{
			{requestField: 10, response: pairingEnvelope(11, nil)},
			{requestField: 20, response: pairingEnvelope(20, nil)},
			{requestField: 30, response: pairingEnvelope(31, nil)},
		}
		for _, step := range steps {
			frame, readErr := readFrame(ctx, tlsConn)
			if readErr != nil {
				serverDone <- readErr
				return
			}
			if !hasBytesField(frame, step.requestField) {
				serverDone <- fmt.Errorf("pairing frame missing field %d", step.requestField)
				return
			}
			if err := writeFrame(ctx, tlsConn, step.response); err != nil {
				serverDone <- err
				return
			}
		}
		secretFrame, readErr := readFrame(ctx, tlsConn)
		if readErr != nil {
			serverDone <- readErr
			return
		}
		secret, ok := bytesField(secretFrame, 40)
		if !ok || len(secret) == 0 {
			serverDone <- fmt.Errorf("pairing secret frame missing")
			return
		}
		if err := writeFrame(ctx, tlsConn, pairingEnvelope(41, nil)); err != nil {
			serverDone <- err
			return
		}
		serverDone <- nil
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := (wirePairingClient{}).Begin(ctx, Device{Serial: "fixture-tv", Endpoint: listener.Addr().String()})
	require.NoError(t, err)
	// Begin returns only after the configuration acknowledgement. This is the
	// protocol boundary at which a real television displays its PIN.
	wireSession, ok := session.(*wirePairingSession)
	require.True(t, ok)
	serverCertificate := wireSession.conn.ConnectionState().PeerCertificates[0]
	code := fixturePairingCode(t, wireSession.clientCertificate, serverCertificate)
	bundleBytes, err := session.Complete(ctx, code)
	require.NoError(t, err)
	require.NoError(t, session.Close())
	require.NoError(t, err)
	bundle, err := decodeCertificateBundle(bundleBytes)
	require.NoError(t, err)
	_, err = tls.X509KeyPair([]byte(bundle.Certificate), []byte(bundle.PrivateKey))
	require.NoError(t, err)
	require.NoError(t, <-serverDone)
}

func TestPairingSecretUsesChecksumAndLastFourHexCharacters(t *testing.T) {
	_, client, err := newCertificateBundle()
	require.NoError(t, err)
	server := testServerCertificate(t)
	serverCert, err := x509.ParseCertificate(server.Certificate[0])
	require.NoError(t, err)
	code := fixturePairingCode(t, client, serverCert)
	got, err := pairingSecret(client, tls.ConnectionState{PeerCertificates: []*x509.Certificate{serverCert}}, code)
	require.NoError(t, err)
	require.Equal(t, code[:2], fmt.Sprintf("%02X", got[0]))
	require.Len(t, got, sha256.Size)
}

func fixturePairingCode(t *testing.T, client tls.Certificate, server *x509.Certificate) string {
	t.Helper()
	clientCert, err := x509.ParseCertificate(client.Certificate[0])
	require.NoError(t, err)
	clientRSA := clientCert.PublicKey.(*rsa.PublicKey)
	serverRSA := server.PublicKey.(*rsa.PublicKey)
	for suffix := 0; suffix <= 0xffff; suffix++ {
		tail := fmt.Sprintf("%04X", suffix)
		hash := sha256.New()
		_, _ = hash.Write(clientRSA.N.Bytes())
		_, _ = hash.Write(big.NewInt(int64(clientRSA.E)).Bytes())
		_, _ = hash.Write(serverRSA.N.Bytes())
		_, _ = hash.Write(big.NewInt(int64(serverRSA.E)).Bytes())
		tailBytes, decodeErr := hex.DecodeString(tail)
		require.NoError(t, decodeErr)
		_, _ = hash.Write(tailBytes)
		digest := hash.Sum(nil)
		code := fmt.Sprintf("%02X%s", digest[0], tail)
		if _, pairingErr := pairingSecret(client, tls.ConnectionState{PeerCertificates: []*x509.Certificate{server}}, code); pairingErr == nil {
			return code
		}
	}
	t.Fatal("could not construct a valid fixture pairing code")
	return ""
}
