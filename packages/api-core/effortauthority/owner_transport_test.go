package effortauthority

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func transportCertificate(t *testing.T) (tls.Certificate, *x509.CertPool, string) {
	t.Helper()
	key, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "disposable finite owner peer"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}}
	der, e := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	cert, e := x509.ParseCertificate(der)
	if e != nil {
		t.Fatal(e)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	sum := sha256.Sum256(der)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, pool, hex.EncodeToString(sum[:])
}
func TestFiniteOwnerPrivateTransportPinsPeerAndPolicy(t *testing.T) {
	a, owner, _, p, key, now := authorityFixture(t)
	cert, pool, pin := transportCertificate(t)
	h := &OwnerReadHandler{Store: a.Store, Owners: owner, Peers: map[string]map[string]bool{pin: {p.ID: true}}, Now: func() time.Time { return *now }}
	server := httptest.NewUnstartedServer(h)
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}, ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert}
	server.StartTLS()
	defer server.Close()
	client := &OwnerReadClient{URL: server.URL + OwnerReadEndpoint, ServerPin: pin, HTTP: &http.Client{Timeout: time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool, Certificates: []tls.Certificate{cert}}}}}
	a.Owners = client
	proof := signedFixture(t, a, p, key, nativeIntent(p, "private-transport"), "private-transport-nonce")
	b, e := a.CheckProof(context.Background(), proof)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = client.CurrentEffortOwner(context.Background(), p.Owner); e == nil {
		t.Fatal("remote arbitrary subject oracle available")
	}
	before, _ := a.Store.Get(context.Background(), p.ID)
	raw, _ := json.Marshal(binding(p))
	wrongChannel, _ := http.NewRequest(http.MethodPost, client.URL, bytes.NewReader(raw))
	wrongChannel.Header.Set("Authorization", "Bearer invalid offered channel")
	response, e := client.HTTP.Do(wrongChannel)
	if e != nil {
		t.Fatal(e)
	}
	response.Body.Close()
	if response.StatusCode != 403 {
		t.Fatal("private owner route accepted wrong credential channel")
	}
	unchangedRecord(t, a, p.ID, Digest(before))
	for _, name := range []string{"wrong-server-pin", "wrong-policy", "unregistered-peer", "revoked-grant", "expired", "owner-unavailable", "missing-client-cert", "insecure-client"} {
		t.Run(name, func(t *testing.T) {
			oldPin := client.ServerPin
			oldNow := *now
			oldScopes := owner.state.Scopes
			oldPeers := h.Peers
			oldHTTP := client.HTTP
			switch name {
			case "wrong-server-pin":
				client.ServerPin = "wrong"
			case "wrong-policy":
				h.Peers = map[string]map[string]bool{pin: {"other-policy": true}}
			case "unregistered-peer":
				h.Peers = map[string]map[string]bool{}
			case "revoked-grant":
				owner.state.Scopes = []string{"agent-manager:read"}
			case "expired":
				*now = p.Deadline
			case "owner-unavailable":
				owner.unavailable = true
			case "missing-client-cert", "insecure-client":
				cfg := client.HTTP.Transport.(*http.Transport).TLSClientConfig.Clone()
				if name == "missing-client-cert" {
					cfg.Certificates = nil
				} else {
					cfg.InsecureSkipVerify = true
				}
				client.HTTP = &http.Client{Timeout: time.Second, Transport: &http.Transport{TLSClientConfig: cfg}}
			}
			if _, e = a.CheckBinding(context.Background(), b); e == nil {
				t.Fatal("private owner refusal admitted")
			}
			unchangedRecord(t, a, p.ID, Digest(before))
			client.ServerPin = oldPin
			*now = oldNow
			owner.state.Scopes = oldScopes
			h.Peers = oldPeers
			client.HTTP = oldHTTP
			owner.unavailable = false
		})
	}
	*now = now.Add(5 * time.Hour)
	if _, e = a.CheckBinding(context.Background(), b); e != nil {
		t.Fatal("late private current-owner read failed", e)
	}
}
