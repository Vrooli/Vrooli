//go:build linux

package hostwireless

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"golang.org/x/sys/unix"
	"math/big"
	"testing"
	"time"
)

func testKernelCertificate(t *testing.T) *x509.Certificate {
	t.Helper()
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "fixture"}, NotBefore: time.Unix(1, 0), NotAfter: time.Unix(2000000000, 0), SubjectKeyId: []byte{1, 2, 3}, SignatureAlgorithm: x509.SHA256WithRSA, KeyUsage: x509.KeyUsageDigitalSignature}
	der, e := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	cert, e := x509.ParseCertificate(der)
	if e != nil {
		t.Fatal(e)
	}
	return cert
}
func TestKernelTrustRequiresCryptoAndRevocation(t *testing.T) {
	cert := testKernelCertificate(t)
	listing := []byte("00000010 I------ 1 perm 1f030000 0 0 keyring .secondary_trusted_keys: 1\n00000011 I------ 1 perm 1f030000 0 0 keyring .blacklist: empty\n")
	ops := kernelTrustOps{keys: func() ([]byte, error) { return listing, nil }, search: func(r int, kind, desc string, dest int) (int, error) {
		if dest != 0 {
			t.Fatal("attempted keyring mutation")
		}
		if r == 16 && kind == "asymmetric" {
			return 25, nil
		}
		return 0, unix.ENOKEY
	}, verify: func(key int, h, sig []byte) error {
		expected := sha256.Sum256(cert.RawTBSCertificate)
		if key != 25 || string(h) != string(expected[:]) || string(sig) != string(cert.Signature) {
			t.Fatal("wrong cryptographic input")
		}
		return nil
	}}
	r := evaluateKernelCertificateWith(context.Background(), cert, ops)
	if r.State != "verified" || r.Revocation != "clear" {
		t.Fatalf("%+v", r)
	}
	originalSearch := ops.search
	ops.search = func(r int, k, d string, dest int) (int, error) {
		if r == 16 && d != "ex:010203" {
			return 26, nil
		}
		return originalSearch(r, k, d, dest)
	}
	if r := evaluateKernelCertificateWith(context.Background(), cert, ops); r.State == "verified" {
		t.Fatal("different kernel issuer/serial key accepted", r)
	}
	ops.search = originalSearch
	ops.verify = func(int, []byte, []byte) error { return unix.EKEYREJECTED }
	if r := evaluateKernelCertificateWith(context.Background(), cert, ops); r.State == "verified" {
		t.Fatalf("SKID collision accepted: %+v", r)
	}
	ops.verify = func(int, []byte, []byte) error { return nil }
	saved := ops.search
	ops.search = func(r int, k, d string, x int) (int, error) {
		if r == 17 {
			return 99, nil
		}
		return saved(r, k, d, x)
	}
	if r := evaluateKernelCertificateWith(context.Background(), cert, ops); r.State != "revoked" {
		t.Fatalf("%+v", r)
	}
	ops.search = func(r int, k, d string, x int) (int, error) {
		if r == 17 {
			return 0, unix.EACCES
		}
		return saved(r, k, d, x)
	}
	if r := evaluateKernelCertificateWith(context.Background(), cert, ops); r.State != "key_verified_revocation_unknown" {
		t.Fatalf("%+v", r)
	}
	ops.keys = func() ([]byte, error) { return nil, unix.EPERM }
	if r := evaluateKernelCertificateWith(context.Background(), cert, ops); r.State != "unavailable" {
		t.Fatalf("%+v", r)
	}
}
func TestTrustedRingIdentityAmbiguity(t *testing.T) {
	line := "10 I------ 1 perm 1f030000 0 0 keyring .secondary_trusted_keys: 1\n"
	if _, e := trustedRingSerials(line + line); e == nil {
		t.Fatal("accepted duplicate ring identity")
	}
	got, e := trustedRingSerials("10 I------ 1 perm 1f030000 1000 0 keyring .secondary_trusted_keys: 1\n")
	if e != nil || len(got) != 0 {
		t.Fatal(got, e)
	}
}
func TestKernelVerificationRejectsMalformedInputs(t *testing.T) {
	if e := kernelVerifyRSASHA256(1, nil, nil); e == nil {
		t.Fatal("empty verification input accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := evaluateKernelCertificateWith(ctx, testKernelCertificate(t), kernelTrustOps{})
	if !errors.Is(ctx.Err(), context.Canceled) || r.State == "verified" {
		t.Fatal(r)
	}
}
