//go:build linux

package hostwireless

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/pem"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestKernelSignatureVerifiesExactPayloadAndCertificate(t *testing.T) {
	if _, err := os.Stat("/usr/bin/openssl"); os.IsNotExist(err) {
		t.Skip("native OpenSSL unavailable")
	}
	dir := t.TempDir()
	payload := []byte("reviewed module bytes\x00\xff\r\n")
	original := FileIdentity{Path: "btmtk.ko", SHA256: digest(payload)}
	cert, key := testCertificate(t, dir, "signer")
	wrongCert, _ := testCertificate(t, dir, "wrong")
	certPEM, err := os.ReadFile(cert)
	if err != nil {
		t.Fatal(err)
	}
	wrongPEM, err := os.ReadFile(wrongCert)
	if err != nil {
		t.Fatal(err)
	}
	detached := signFixture(t, dir, cert, key, payload)
	attached := signFixture(t, dir, cert, key, payload, "-nodetach")
	sha384 := signFixture(t, dir, cert, key, payload, "-md", "sha384")
	good := appendModuleSignature(payload, detached)
	corruptSignature := bytes.Clone(good)
	corruptSignature[len(corruptSignature)-len(moduleSignatureMarker)-moduleSignatureHeader-1] ^= 1
	badMetadata := bytes.Clone(good)
	badMetadata[len(badMetadata)-len(moduleSignatureMarker)-moduleSignatureHeader+5] = 1
	badLength := bytes.Clone(good)
	badLength[len(badLength)-len(moduleSignatureMarker)-4] = 1
	for _, tc := range []struct {
		name, wantErr string
		signed, cert  []byte
	}{
		{"reviewed module", "", good, certPEM},
		{"wrong key with same issuer and serial", "could not verify", good, wrongPEM},
		{"changed payload", "digest differs", appendModuleSignature([]byte("changed"), detached), certPEM},
		{"changed signature", "could not verify", corruptSignature, certPEM},
		{"trailer metadata", "invalid PKCS#7", badMetadata, certPEM},
		{"trailer length", "length is out of bounds", badLength, certPEM},
		{"absent marker", "marker absent", good[:len(good)-1], certPEM},
		{"attached CMS", "must contain detached", appendModuleSignature(payload, attached), certPEM},
		{"wrong digest", "must use SHA-256", appendModuleSignature(payload, sha384), certPEM},
		{"trailing DER data", "not a single DER", appendModuleSignature(payload, append(bytes.Clone(detached), 0)), certPEM},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			proof, err := verifySignedModule(ctx, original, tc.signed, tc.cert)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) || proof != (SignedModuleEvidence{}) {
					t.Fatalf("got %+v, %v, wanted error %q", proof, err, tc.wantErr)
				}
			} else if err != nil || proof.Name != original.Path ||
				proof.UnsignedSHA256 != original.SHA256 || proof.SignedSHA256 != digest(good) {
				t.Fatalf("got %+v, %v", proof, err)
			}
		})
	}
	parsed, err := readPublicCertificate(cert)
	if err != nil {
		t.Fatal(err)
	}
	derPath := filepath.Join(dir, "public.der")
	writeFixture(t, derPath, string(parsed.Raw))
	if derCert, err := readPublicCertificate(derPath); err != nil || !bytes.Equal(derCert.Raw, parsed.Raw) {
		t.Fatalf("DER certificate verification: %v", err)
	}
	chainPath := filepath.Join(dir, "more-than-one.pem")
	writeFixture(t, chainPath, string(certPEM)+string(wrongPEM))
	if _, err := readPublicCertificate(chainPath); err == nil {
		t.Fatal("accepted more than one certificate")
	}
}

func testCertificate(t *testing.T, dir, name string) (string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(42), Subject: pkix.Name{CommonName: "Vrooli module test certificate"},
		NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Minute),
		KeyUsage: x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPath, keyPath := filepath.Join(dir, name+".pem"), filepath.Join(dir, name+".key")
	writeFixture(t, certPath, string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})))
	writeFixture(t, keyPath, string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})))
	return certPath, keyPath
}

func signFixture(t *testing.T, dir, cert, key string, payload []byte, extra ...string) []byte {
	t.Helper()
	input := filepath.Join(dir, "payload")
	writeFixture(t, input, string(payload))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	args := append([]string{"cms", "-sign", "-binary", "-md", "sha256", "-nocerts", "-noattr",
		"-outform", "DER", "-signer", cert, "-inkey", key, "-in", input}, extra...)
	cmd := exec.CommandContext(ctx, "/usr/bin/openssl", args...)
	cmd.Env = []string{"LC_ALL=C", "OPENSSL_CONF=/dev/null"}
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("sign test payload: %v", err)
	}
	return out
}

func appendModuleSignature(payload, signature []byte) []byte {
	var trailer [moduleSignatureHeader]byte
	trailer[2] = 2
	binary.BigEndian.PutUint32(trailer[8:], uint32(len(signature)))
	return bytes.Join([][]byte{payload, signature, trailer[:], []byte(moduleSignatureMarker)}, nil)
}
