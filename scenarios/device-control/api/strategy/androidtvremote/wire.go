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
	"io"
	"math/big"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"device-control/strategy"
	"google.golang.org/protobuf/encoding/protowire"
)

const (
	androidTVRemotePort     = 6466
	androidTVPairingPort    = 6467
	androidTVFrameLimit     = 8 << 20
	androidTVActiveFeatures = 622
)

// wirePairingClient is the production Android TV Remote v2 pairing exchange.
// The protocol uses mutual TLS and length-delimited protobuf messages; the
// certificate bundle returned to the credential authority is the durable
// identity of this controller on the television.
type wirePairingClient struct{}

func (wirePairingClient) Pair(ctx context.Context, device Device, pin string) ([]byte, error) {
	session, err := (wirePairingClient{}).Begin(ctx, device)
	if err != nil {
		return nil, err
	}
	defer session.Close()
	return session.Complete(ctx, pin)
}

func (wirePairingClient) Begin(ctx context.Context, device Device) (PairingSession, error) {
	bundle, clientCertificate, err := newCertificateBundle()
	if err != nil {
		return nil, fmt.Errorf("generate Android TV Remote certificate: %w", err)
	}
	host, targetPort, err := endpointHostPort(device.Endpoint, androidTVRemotePort)
	if err != nil {
		return nil, err
	}
	pairingPort := androidTVPairingPort
	// Production discovery reports the remote port (6466); a non-default port
	// is accepted for deterministic protocol fixtures and controlled gateways.
	if targetPort != androidTVRemotePort {
		pairingPort = targetPort
	}
	conn, err := dialTLS(ctx, net.JoinHostPort(host, strconv.Itoa(pairingPort)), clientCertificate)
	if err != nil {
		return nil, fmt.Errorf("connect Android TV Remote pairing endpoint: %w", err)
	}

	if err := writeFrame(ctx, conn, pairingRequest()); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("send Android TV Remote pairing request: %w", err)
	}
	if _, err := expectFrame(ctx, conn, 11); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("receive Android TV Remote pairing acknowledgement: %w", err)
	}
	if err := writeFrame(ctx, conn, pairingOption()); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("send Android TV Remote pairing option: %w", err)
	}
	if _, err := expectFrame(ctx, conn, 20); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("receive Android TV Remote pairing option: %w", err)
	}
	if err := writeFrame(ctx, conn, pairingConfiguration()); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("send Android TV Remote pairing configuration: %w", err)
	}
	if _, err := expectFrame(ctx, conn, 31); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("receive Android TV Remote pairing configuration acknowledgement: %w", err)
	}
	return &wirePairingSession{bundle: bundle, clientCertificate: clientCertificate, conn: conn}, nil
}

type wirePairingSession struct {
	bundle            certificateBundle
	clientCertificate tls.Certificate
	conn              *tls.Conn
	mu                sync.Mutex
	closed            bool
}

func (s *wirePairingSession) Complete(ctx context.Context, pin string) ([]byte, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, fmt.Errorf("Android TV Remote pairing session is closed")
	}
	conn := s.conn
	s.mu.Unlock()
	secret, err := pairingSecret(s.clientCertificate, conn.ConnectionState(), pin)
	if err != nil {
		return nil, err
	}
	if err := writeFrame(ctx, conn, pairingSecretMessage(secret)); err != nil {
		return nil, fmt.Errorf("send Android TV Remote pairing secret: %w", err)
	}
	if _, err := expectFrame(ctx, conn, 41); err != nil {
		return nil, fmt.Errorf("receive Android TV Remote pairing secret acknowledgement: %w", err)
	}
	return encodeCertificateBundle(s.bundle)
}

func (s *wirePairingSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	return s.conn.Close()
}

type wireClient struct {
	endpoint string
	bundle   certificateBundle
	mu       sync.Mutex
	conn     *tls.Conn
}

func newWireClient(endpoint string, bundle certificateBundle) (Client, error) {
	host, port, err := endpointHostPort(endpoint, androidTVRemotePort)
	if err != nil {
		return nil, err
	}
	return &wireClient{endpoint: net.JoinHostPort(host, strconv.Itoa(port)), bundle: bundle}, nil
}

func (c *wireClient) Key(ctx context.Context, key string) error {
	code, ok := remoteKeyCode(key)
	if !ok {
		return &strategy.UnsupportedCapabilityError{Capability: strategy.CapInput, Operation: fmt.Sprintf("Android TV Remote key %q is unavailable", key)}
	}
	return c.send(ctx, remoteKeyMessage(code))
}

func (c *wireClient) Text(ctx context.Context, value string) error {
	return c.send(ctx, remoteTextMessage(value))
}

func (c *wireClient) Media(ctx context.Context, command strategy.MediaCommand) error {
	code, ok := mediaKeyCode(command)
	if !ok {
		return &strategy.UnsupportedCapabilityError{Capability: strategy.CapMedia, Operation: fmt.Sprintf("Android TV Remote media action %q is unavailable", command.Action)}
	}
	return c.send(ctx, remoteKeyMessage(code))
}

func (c *wireClient) send(ctx context.Context, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ensureConnected(ctx); err != nil {
		return err
	}
	if err := writeFrame(ctx, c.conn, payload); err != nil {
		_ = c.conn.Close()
		c.conn = nil
		return fmt.Errorf("send Android TV Remote command: %w", err)
	}
	return nil
}

func (c *wireClient) ensureConnected(ctx context.Context) error {
	if c.conn != nil {
		return nil
	}
	keyPair, err := tls.X509KeyPair([]byte(c.bundle.Certificate), []byte(c.bundle.PrivateKey))
	if err != nil {
		return fmt.Errorf("parse Android TV Remote certificate: %w", err)
	}
	conn, err := dialTLS(ctx, c.endpoint, keyPair)
	if err != nil {
		return fmt.Errorf("connect Android TV Remote endpoint: %w", err)
	}
	if _, err := expectFrame(ctx, conn, 1); err != nil {
		_ = conn.Close()
		return fmt.Errorf("receive Android TV Remote configuration: %w", err)
	}
	if err := writeFrame(ctx, conn, remoteConfigureMessage()); err != nil {
		_ = conn.Close()
		return fmt.Errorf("send Android TV Remote configuration: %w", err)
	}
	for {
		frame, err := readFrame(ctx, conn)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("receive Android TV Remote session state: %w", err)
		}
		switch {
		case hasBytesField(frame, 2):
			if err := writeFrame(ctx, conn, remoteSetActiveMessage(androidTVActiveFeatures)); err != nil {
				_ = conn.Close()
				return fmt.Errorf("activate Android TV Remote session: %w", err)
			}
		case hasBytesField(frame, 40):
			c.conn = conn
			return nil
		case hasBytesField(frame, 8):
			if response, ok := remotePingResponse(frame); ok {
				if err := writeFrame(ctx, conn, response); err != nil {
					_ = conn.Close()
					return fmt.Errorf("respond to Android TV Remote ping: %w", err)
				}
			}
		}
	}
}

func newCertificateBundle() (certificateBundle, tls.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return certificateBundle{}, tls.Certificate{}, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return certificateBundle{}, tls.Certificate{}, err
	}
	now := time.Now().UTC()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "androidtv-remote"},
		NotBefore:    now.Add(-time.Minute), NotAfter: now.AddDate(73, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return certificateBundle{}, tls.Certificate{}, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return certificateBundle{}, tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return certificateBundle{}, tls.Certificate{}, err
	}
	return certificateBundle{Certificate: string(certPEM), PrivateKey: string(keyPEM)}, pair, nil
}

func dialTLS(ctx context.Context, endpoint string, certificate tls.Certificate) (*tls.Conn, error) {
	dialer := &net.Dialer{}
	raw, err := dialer.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return nil, err
	}
	host, _, _ := net.SplitHostPort(endpoint)
	conn := tls.Client(raw, &tls.Config{ // #nosec G402 -- Android TV Remote uses a self-signed device certificate; pairing binds the peer key through the protocol secret.
		Certificates:       []tls.Certificate{certificate},
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		ServerName:         host,
	})
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if err := conn.HandshakeContext(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	return conn, nil
}

func endpointHostPort(endpoint string, defaultPort int) (string, int, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", 0, fmt.Errorf("Android TV Remote endpoint is empty")
	}
	host, portText, err := net.SplitHostPort(endpoint)
	if err != nil {
		host = endpoint
		portText = strconv.Itoa(defaultPort)
	}
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "" {
		return "", 0, fmt.Errorf("Android TV Remote endpoint %q has no host", endpoint)
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("Android TV Remote endpoint %q has invalid port", endpoint)
	}
	return host, port, nil
}

func writeFrame(ctx context.Context, conn net.Conn, payload []byte) error {
	if len(payload) > androidTVFrameLimit {
		return fmt.Errorf("Android TV Remote frame exceeds %d bytes", androidTVFrameLimit)
	}
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetWriteDeadline(deadline); err != nil {
			return err
		}
	}
	frame := protowire.AppendVarint(nil, uint64(len(payload)))
	frame = append(frame, payload...)
	_, err := conn.Write(frame)
	_ = conn.SetWriteDeadline(time.Time{})
	return err
}

func readFrame(ctx context.Context, conn net.Conn) ([]byte, error) {
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetReadDeadline(deadline); err != nil {
			return nil, err
		}
	}
	var prefix [10]byte
	for i := range prefix {
		if _, err := io.ReadFull(conn, prefix[i:i+1]); err != nil {
			return nil, err
		}
		if prefix[i] < 0x80 {
			length, n := protowire.ConsumeVarint(prefix[:i+1])
			if n < 0 || length > androidTVFrameLimit {
				return nil, fmt.Errorf("invalid Android TV Remote frame length")
			}
			payload := make([]byte, int(length))
			if _, err := io.ReadFull(conn, payload); err != nil {
				return nil, err
			}
			_ = conn.SetReadDeadline(time.Time{})
			return payload, nil
		}
	}
	return nil, fmt.Errorf("Android TV Remote frame length varint is too long")
}

func expectFrame(ctx context.Context, conn net.Conn, field protowire.Number) ([]byte, error) {
	payload, err := readFrame(ctx, conn)
	if err != nil {
		return nil, err
	}
	if !hasBytesField(payload, field) {
		if status, ok := varintField(payload, 2); ok && status != 200 {
			return nil, fmt.Errorf("Android TV Remote protocol status %d", status)
		}
		return nil, fmt.Errorf("Android TV Remote response did not contain field %d", field)
	}
	return payload, nil
}

func pairingSecret(client tls.Certificate, state tls.ConnectionState, pin string) ([]byte, error) {
	code := strings.TrimSpace(pin)
	if len(code) != 6 {
		return nil, fmt.Errorf("Android TV Remote pairing code must contain six hexadecimal characters")
	}
	checkByte, err := hex.DecodeString(code[:2])
	if err != nil {
		return nil, fmt.Errorf("Android TV Remote pairing code must contain only hexadecimal characters")
	}
	codeBytes, err := hex.DecodeString(code[2:])
	if err != nil {
		return nil, fmt.Errorf("Android TV Remote pairing code must contain only hexadecimal characters")
	}
	if len(state.PeerCertificates) == 0 {
		return nil, fmt.Errorf("Android TV Remote pairing server certificate is missing")
	}
	clientCert, err := x509.ParseCertificate(client.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("parse Android TV Remote client certificate: %w", err)
	}
	clientRSA, ok := clientCert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Android TV Remote client certificate is not RSA")
	}
	serverRSA, ok := state.PeerCertificates[0].PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Android TV Remote server certificate is not RSA")
	}
	hash := sha256.New()
	_, _ = hash.Write(clientRSA.N.Bytes())
	_, _ = hash.Write(big.NewInt(int64(clientRSA.E)).Bytes())
	_, _ = hash.Write(serverRSA.N.Bytes())
	_, _ = hash.Write(big.NewInt(int64(serverRSA.E)).Bytes())
	_, _ = hash.Write(codeBytes)
	result := hash.Sum(nil)
	if result[0] != checkByte[0] {
		return nil, fmt.Errorf("Android TV Remote pairing code does not match this television")
	}
	return result, nil
}

func pairingRequest() []byte {
	request := appendString(nil, 1, "androidtv-remote")
	request = appendString(request, 2, "vrooli-device-control")
	return pairingEnvelope(10, request)
}

func pairingOption() []byte {
	encoding := appendVarint(nil, 1, 3)
	encoding = appendVarint(encoding, 2, 6)
	option := appendBytes(nil, 1, encoding)
	option = appendVarint(option, 3, 1)
	return pairingEnvelope(20, option)
}

func pairingConfiguration() []byte {
	encoding := appendVarint(nil, 1, 3)
	encoding = appendVarint(encoding, 2, 6)
	configuration := appendBytes(nil, 1, encoding)
	configuration = appendVarint(configuration, 2, 1)
	return pairingEnvelope(30, configuration)
}

func pairingSecretMessage(secret []byte) []byte {
	return pairingEnvelope(40, appendBytes(nil, 1, secret))
}

func pairingEnvelope(field protowire.Number, nested []byte) []byte {
	envelope := appendVarint(nil, 1, 2)
	envelope = appendVarint(envelope, 2, 200)
	return appendBytes(envelope, field, nested)
}

func remoteConfigureMessage() []byte {
	info := appendString(nil, 1, "vrooli-device-control")
	info = appendString(info, 2, "Vrooli")
	info = appendVarint(info, 3, 1)
	info = appendString(info, 4, "1")
	info = appendString(info, 5, "device-control")
	info = appendString(info, 6, "1.0.0")
	configure := appendVarint(nil, 1, 622)
	configure = appendBytes(configure, 2, info)
	return appendBytes(nil, 1, configure)
}

func remoteSetActiveMessage(active uint64) []byte {
	return appendBytes(nil, 2, appendVarint(nil, 1, active))
}

func remoteStartMessage(started bool) []byte {
	value := uint64(0)
	if started {
		value = 1
	}
	return appendBytes(nil, 40, appendVarint(nil, 1, value))
}

func remotePingResponse(payload []byte) ([]byte, bool) {
	request, ok := bytesField(payload, 8)
	if !ok {
		return nil, false
	}
	value, ok := varintField(request, 1)
	if !ok {
		return nil, false
	}
	return appendBytes(nil, 9, appendVarint(nil, 1, value)), true
}

func remoteKeyMessage(code uint64) []byte {
	key := appendVarint(nil, 1, code)
	key = appendVarint(key, 2, 3)
	return appendBytes(nil, 10, key)
}

func remoteTextMessage(value string) []byte {
	text := appendVarint(nil, 1, 0)
	text = appendVarint(text, 2, 0)
	field := appendVarint(nil, 1, 1)
	fieldStatus := appendVarint(nil, 1, 0)
	fieldStatus = appendVarint(fieldStatus, 2, 0)
	fieldStatus = appendString(fieldStatus, 3, value)
	field = appendBytes(field, 2, fieldStatus)
	text = appendBytes(text, 3, field)
	return appendBytes(nil, 21, text)
}

func remoteKeyCode(key string) (uint64, bool) {
	key = strings.ToUpper(strings.TrimSpace(key))
	key = strings.TrimPrefix(key, "KEYCODE_")
	codes := map[string]uint64{
		"HOME": 3, "BACK": 4, "DPAD_UP": 19, "DPAD_DOWN": 20, "DPAD_LEFT": 21, "DPAD_RIGHT": 22, "DPAD_CENTER": 23,
		"VOLUME_UP": 24, "VOLUME_DOWN": 25, "POWER": 26, "VOLUME_MUTE": 164, "MEDIA_PLAY_PAUSE": 85, "MEDIA_STOP": 86, "MEDIA_NEXT": 87, "MEDIA_PREVIOUS": 88,
		"MEDIA_PLAY": 126, "MEDIA_PAUSE": 127, "ENTER": 66, "DEL": 67, "SPACE": 62, "MENU": 82, "SEARCH": 84,
	}
	code, ok := codes[key]
	if ok {
		return code, true
	}
	if numeric, err := strconv.ParseUint(key, 10, 32); err == nil {
		return numeric, true
	}
	return 0, false
}

func mediaKeyCode(command strategy.MediaCommand) (uint64, bool) {
	action := strings.ToLower(strings.TrimSpace(command.Action))
	switch action {
	case "play":
		return remoteKeyCode("MEDIA_PLAY")
	case "pause":
		return remoteKeyCode("MEDIA_PAUSE")
	case "stop":
		return remoteKeyCode("MEDIA_STOP")
	case "next":
		return remoteKeyCode("MEDIA_NEXT")
	case "previous":
		return remoteKeyCode("MEDIA_PREVIOUS")
	case "volume-up", "volume_down", "volume-down", "volume_up":
		if strings.Contains(action, "down") {
			return remoteKeyCode("VOLUME_DOWN")
		}
		return remoteKeyCode("VOLUME_UP")
	case "volume-mute", "volume_mute", "mute":
		return remoteKeyCode("VOLUME_MUTE")
	case "volume":
		value, ok := command.Value.(string)
		if !ok {
			return 0, false
		}
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "up", "volume-up", "volume_up":
			return remoteKeyCode("VOLUME_UP")
		case "down", "volume-down", "volume_down":
			return remoteKeyCode("VOLUME_DOWN")
		case "mute", "volume-mute", "volume_mute":
			return remoteKeyCode("VOLUME_MUTE")
		default:
			return 0, false
		}
	default:
		return remoteKeyCode(action)
	}
}

func appendVarint(dst []byte, field protowire.Number, value uint64) []byte {
	dst = protowire.AppendTag(dst, field, protowire.VarintType)
	return protowire.AppendVarint(dst, value)
}

func appendBytes(dst []byte, field protowire.Number, value []byte) []byte {
	dst = protowire.AppendTag(dst, field, protowire.BytesType)
	return protowire.AppendBytes(dst, value)
}

func appendString(dst []byte, field protowire.Number, value string) []byte {
	return appendBytes(dst, field, []byte(value))
}

func hasBytesField(payload []byte, wanted protowire.Number) bool {
	for len(payload) > 0 {
		number, typ, tagLen := protowire.ConsumeTag(payload)
		if tagLen < 0 {
			return false
		}
		payload = payload[tagLen:]
		switch typ {
		case protowire.BytesType:
			_, valueLen := protowire.ConsumeBytes(payload)
			if valueLen < 0 {
				return false
			}
			if number == wanted {
				return true
			}
			payload = payload[valueLen:]
		case protowire.VarintType:
			_, valueLen := protowire.ConsumeVarint(payload)
			if valueLen < 0 {
				return false
			}
			payload = payload[valueLen:]
		default:
			valueLen := protowire.ConsumeFieldValue(number, typ, payload)
			if valueLen < 0 {
				return false
			}
			payload = payload[valueLen:]
		}
	}
	return false
}

func bytesField(payload []byte, wanted protowire.Number) ([]byte, bool) {
	for len(payload) > 0 {
		number, typ, tagLen := protowire.ConsumeTag(payload)
		if tagLen < 0 {
			return nil, false
		}
		payload = payload[tagLen:]
		switch typ {
		case protowire.BytesType:
			value, valueLen := protowire.ConsumeBytes(payload)
			if valueLen < 0 {
				return nil, false
			}
			if number == wanted {
				return value, true
			}
			payload = payload[valueLen:]
		case protowire.VarintType:
			_, valueLen := protowire.ConsumeVarint(payload)
			if valueLen < 0 {
				return nil, false
			}
			payload = payload[valueLen:]
		default:
			valueLen := protowire.ConsumeFieldValue(number, typ, payload)
			if valueLen < 0 {
				return nil, false
			}
			payload = payload[valueLen:]
		}
	}
	return nil, false
}

func varintField(payload []byte, wanted protowire.Number) (uint64, bool) {
	for len(payload) > 0 {
		number, typ, tagLen := protowire.ConsumeTag(payload)
		if tagLen < 0 {
			return 0, false
		}
		payload = payload[tagLen:]
		if typ == protowire.VarintType {
			value, valueLen := protowire.ConsumeVarint(payload)
			if valueLen < 0 {
				return 0, false
			}
			if number == wanted {
				return value, true
			}
			payload = payload[valueLen:]
			continue
		}
		valueLen := protowire.ConsumeFieldValue(number, typ, payload)
		if valueLen < 0 {
			return 0, false
		}
		payload = payload[valueLen:]
	}
	return 0, false
}
