package finiteinstallation

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/vrooli/api-core/effortauthority"
)

// This fixture creates only disposable in-memory certificate records. It never
// acquires an existing owner credential, starts a listener, or makes a request.
func composedHandleRecord(t *testing.T, plan *Plan, usage string, serial int64) ([]byte, string) {
	t.Helper()
	now := time.Now().UTC()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTemplate := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: "disposable composition CA"}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	eku := x509.ExtKeyUsageClientAuth
	if usage == "server" {
		eku = x509.ExtKeyUsageServerAuth
	}
	leaf := &x509.Certificate{SerialNumber: big.NewInt(serial + 1), Subject: pkix.Name{CommonName: "disposable composition " + usage}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{eku}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}}
	der, err := x509.CreateCertificate(rand.Reader, leaf, caCert, &key.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	record := map[string]string{"usage": usage, "role": plan.bundle.Role, "policyDigest": effortauthority.Digest(plan.bundle.Installation.Policy), "certificate": string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), "privateKey": string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})), "ca": string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}))}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	pin := sha256.Sum256(der)
	return raw, hex.EncodeToString(pin[:])
}

func composedHandleFixture(t *testing.T) (*Plan, *tls.Config, *tls.Config, effortauthority.Installation) {
	t.Helper()
	cfg := bundleInstallationFixture()
	cfg.Builds["acceptance"] = effortauthority.Digest("acceptance-build")
	cfg.ServerPins["acceptance"] = effortauthority.Digest("acceptance-server")
	plan := &Plan{bundle: Bundle{Version: 1, Role: "pm", Installation: cfg, Endpoints: map[string]string{"am": "https://127.0.0.1:18443", "pm": "https://127.0.0.1:18444", "issuer": "https://127.0.0.1:18445", "acceptance": "https://127.0.0.1:18446"}, ProfileMappings: map[string]string{"bounded": "persisted-profile-uuid"}}}
	serverRecord, serverPin := composedHandleRecord(t, plan, "server", 10)
	clientRecord, clientPin := composedHandleRecord(t, plan, "client", 20)
	plan.bundle.Installation.ServerPins["pm"] = serverPin
	plan.bundle.ClientPin = clientPin
	server, err := plan.parseTLS(serverRecord, time.Now(), "server", serverPin)
	if err != nil {
		t.Fatal(err)
	}
	client, err := plan.parseTLS(clientRecord, time.Now(), "client", clientPin)
	if err != nil {
		t.Fatal(err)
	}
	// An independent copy makes mutations below incapable of changing expectations.
	raw, err := json.Marshal(plan.bundle.Installation)
	if err != nil {
		t.Fatal(err)
	}
	var original effortauthority.Installation
	if err := json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	return plan, server, client, original
}

func TestProtectedCompositionCapturesConcreteOwnerClientsWithoutRequests(t *testing.T) {
	plan, server, client, original := composedHandleFixture(t)
	handles, err := plan.composeHandles(server, client)
	if err != nil {
		t.Fatal(err)
	}
	if handles == nil || handles.Publication == nil || handles.Acceptance == nil || handles.Profiles == nil || handles.Scope == nil || handles.Commission == nil || handles.Broker == nil {
		t.Fatal("concrete protected handles missing")
	}
	check := func() {
		t.Helper()
		if handles.Publication.CheckInstallation(original) != nil || handles.Acceptance.CheckInstallation(original.Policy) != nil || handles.Profiles.CheckInstallation(original.Policy) != nil || handles.Scope.CheckInstallation(original.Policy) != nil || handles.Commission.CheckApprovalInstallation(original.Policy) != nil {
			t.Fatal("exact original installation was not captured")
		}
		if handles.Commission.CheckInstallation(original.Policy) == nil || handles.Scope.CheckApprovalInstallation(original.Policy) == nil {
			t.Fatal("commission and live scope roles conflated")
		}
	}
	check()
	plan.bundle.Installation.Policy.Members[0] = "foreign-member"
	plan.bundle.Installation.Policy.Profiles["bounded"] = effortauthority.Digest("foreign-profile")
	plan.bundle.ProfileMappings["foreign"] = "foreign-profile-uuid"
	plan.bundle.Endpoints["pm"] = "https://127.0.0.1:19999"
	plan.bundle.Installation.ServerPins["pm"] = effortauthority.Digest("foreign-pin")
	check()
	changed := original.Policy
	changed.Owner = "foreign-owner"
	if handles.Acceptance.CheckInstallation(changed) == nil || handles.Profiles.CheckInstallation(changed) == nil || handles.Scope.CheckInstallation(changed) == nil || handles.Commission.CheckApprovalInstallation(changed) == nil {
		t.Fatal("foreign policy accepted by a captured client")
	}
	// Client TLS state is deliberately private in the concrete clients. These
	// static checks prove policy/role capture, not a live TLS handshake or carrier.
}

func TestProtectedCompositionRefusesUntrustedOfferedTransportWithoutRequests(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Plan, *tls.Config, *tls.Config)
	}{
		{"wrong-publication-server-pin", func(p *Plan, s, c *tls.Config) {
			p.bundle.Installation.ServerPins["pm"] = effortauthority.Digest("wrong-server")
		}},
		{"anonymous-client", func(p *Plan, s, c *tls.Config) { c.Certificates = nil }},
		{"insecure-client", func(p *Plan, s, c *tls.Config) { c.InsecureSkipVerify = true }},
		{"missing-profile-mapping", func(p *Plan, s, c *tls.Config) { p.bundle.ProfileMappings = nil }},
		{"invalid-broker-pin", func(p *Plan, s, c *tls.Config) { p.bundle.Installation.ServerPins["issuer"] = "invalid" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, server, client, _ := composedHandleFixture(t)
			tc.change(plan, server, client)
			if h, err := plan.composeHandles(server, client); err == nil || h != nil {
				t.Fatal("untrusted offered composition accepted")
			}
		})
	}
}

func TestProtectedCompositionClientRecordRejectsWrongPinAndRole(t *testing.T) {
	plan, _, _, _ := composedHandleFixture(t)
	raw, pin := composedHandleRecord(t, plan, "client", 30)
	if cfg, err := plan.parseTLS(raw, time.Now(), "client", pin); err != nil || cfg == nil {
		t.Fatal("exact disposable client record refused", err)
	}
	for _, change := range []func(map[string]string){func(r map[string]string) { r["role"] = "issuer" }, func(r map[string]string) { r["usage"] = "server" }, func(r map[string]string) { r["policyDigest"] = effortauthority.Digest("foreign") }} {
		var record map[string]string
		if err := json.Unmarshal(raw, &record); err != nil {
			t.Fatal(err)
		}
		change(record)
		bad, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		if cfg, err := plan.parseTLS(bad, time.Now(), "client", pin); err == nil || cfg != nil {
			t.Fatal("foreign client record accepted")
		}
	}
	if cfg, err := plan.parseTLS(raw, time.Now(), "client", effortauthority.Digest("wrong-client-pin")); err == nil || cfg != nil {
		t.Fatal("wrong client pin accepted")
	}
}
