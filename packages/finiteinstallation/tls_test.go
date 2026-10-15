package finiteinstallation

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"github.com/vrooli/api-core/effortauthority"
	"math/big"
	"net"
	"testing"
	"time"
)

func TestExistingPurposeTLSParserExactUsageRolePolicyAndPins(t *testing.T) {
	now := time.Now().UTC()
	key, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "disposable"}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour), IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}}
	der, e := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	encodedKey, e := x509.MarshalPKCS8PrivateKey(key)
	if e != nil {
		t.Fatal(e)
	}
	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	keyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encodedKey}))
	hash := sha256.Sum256(der)
	pin := hex.EncodeToString(hash[:])
	cfg := bundleInstallationFixture()
	plan := &Plan{bundle: Bundle{Role: "pm", Installation: cfg}}
	record := map[string]string{"usage": "server", "role": "pm", "policyDigest": effortauthority.Digest(cfg.Policy), "certificate": certPEM, "privateKey": keyPEM, "ca": certPEM}
	raw, _ := json.Marshal(record)
	if config, e := plan.parseTLS(raw, now, "server", pin); e != nil || config == nil {
		t.Fatalf("existing owner TLS positive: %v", e)
	}
	for _, change := range []func(map[string]string){func(r map[string]string) { r["usage"] = "client" }, func(r map[string]string) { r["role"] = "am" }, func(r map[string]string) { r["policyDigest"] = effortauthority.Digest("other") }} {
		copy := map[string]string{}
		for k, v := range record {
			copy[k] = v
		}
		change(copy)
		bad, _ := json.Marshal(copy)
		if config, e := plan.parseTLS(bad, now, "server", pin); e == nil || config != nil {
			t.Fatal("wrong purpose TLS accepted")
		}
	}
	if config, e := plan.parseTLS(raw, now, "server", effortauthority.Digest("wrong-pin")); e == nil || config != nil {
		t.Fatal("wrong pinned identity accepted")
	}
	if config, e := plan.parseTLS(raw, template.NotAfter, "server", pin); e == nil || config != nil {
		t.Fatal("expired existing identity accepted")
	}
}
