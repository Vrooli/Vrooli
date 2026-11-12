//go:build linux

package hostwireless

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Sources: Linux v7.0 kernel/module/signing.c uses SECONDARY_KEYRING;
// certs/system_keyring.c links builtin into secondary; security/keys/keyctl_pkey.c
// specifies the read-only VERIFY operation. Platform/MOK presence alone does not
// qualify. RSA-SHA256 is deliberately bounded to the installed Ubuntu MOK kind.
type kernelTrustOps struct {
	keys   func() ([]byte, error)
	search func(int, string, string, int) (int, error)
	verify func(int, []byte, []byte) error
}

func evaluateKernelCertificate(ctx context.Context, cert *x509.Certificate) KernelTrustEvidence {
	return evaluateKernelCertificateWith(ctx, cert, kernelTrustOps{
		keys: func() ([]byte, error) {
			f, e := os.Open("/proc/keys")
			if e != nil {
				return nil, e
			}
			defer f.Close()
			return io.ReadAll(io.LimitReader(f, (4<<20)+1))
		},
		search: unix.KeyctlSearch, verify: kernelVerifyRSASHA256,
	})
}
func evaluateKernelCertificateWith(ctx context.Context, cert *x509.Certificate, ops kernelTrustOps) KernelTrustEvidence {
	r := KernelTrustEvidence{CertificateSHA256: digest(cert.Raw), State: "unverified"}
	if err := ctx.Err(); err != nil {
		r.Issue = err.Error()
		return r
	}
	if cert.SignatureAlgorithm != x509.SHA256WithRSA || len(cert.SubjectKeyId) == 0 ||
		cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature) != nil || !bytes.Equal(cert.RawIssuer, cert.RawSubject) {
		r.State = "unsupported_certificate"
		r.Issue = "requires a self-signed RSA-SHA256 certificate with subject key identifier"
		return r
	}
	data, err := ops.keys()
	if err != nil {
		r.State = "unavailable"
		r.Issue = err.Error()
		return r
	}
	if len(data) > 4<<20 {
		r.State = "unavailable"
		r.Issue = "kernel key listing exceeds bound"
		return r
	}
	rings, err := trustedRingSerials(string(data))
	if err != nil {
		r.Issue = err.Error()
		return r
	}
	identity, err := kernelCertificateID(cert)
	if err != nil {
		r.State = "unsupported_certificate"
		r.Issue = err.Error()
		return r
	}
	h := sha256.Sum256(cert.RawTBSCertificate)
	for _, name := range []string{".secondary_trusted_keys", ".builtin_trusted_keys"} {
		serial, ok := rings[name]
		if !ok {
			continue
		}
		if err := ctx.Err(); err != nil {
			r.Issue = err.Error()
			return r
		}
		key, err := ops.search(serial, "asymmetric", "ex:"+hex.EncodeToString(cert.SubjectKeyId), 0)
		if err != nil {
			r.Issue = fmt.Sprintf("%s search: %v", name, err)
			continue
		}
		same, err := ops.search(serial, "asymmetric", "ex:"+identity, 0)
		if err != nil || same != key {
			r.Issue = "kernel certificate issuer/serial identity does not match its SKID key"
			continue
		}
		// A matching SKID is not authentication. Verify the certificate's existing
		// self-signature with the kernel key to prove the same public key is usable.
		if err = ops.verify(key, h[:], cert.Signature); err != nil {
			r.Issue = fmt.Sprintf("%s cryptographic verification: %v", name, err)
			continue
		}
		r.Ring = name
		r.KeySerial = key
		r.Algorithm = "rsa-pkcs1-sha256"
		r.Revocation = "unverified"
		blacklist, ok := rings[".blacklist"]
		if !ok {
			r.State = "key_verified_revocation_unknown"
			r.Issue = "kernel key cryptography succeeded but system blacklist is inaccessible"
			return r
		}
		for _, q := range []struct{ kind, description string }{{"blacklist", "tbs:" + hex.EncodeToString(h[:])}, {"asymmetric", "ex:" + hex.EncodeToString(cert.SubjectKeyId)}} {
			_, err := ops.search(blacklist, q.kind, q.description, 0)
			if err == nil {
				r.State = "revoked"
				r.Revocation = "blocked"
				r.Issue = "certificate matches running kernel revocation material"
				return r
			}
			if !errors.Is(err, unix.ENOKEY) {
				r.State = "key_verified_revocation_unknown"
				r.Issue = "kernel revocation search unavailable: " + err.Error()
				return r
			}
		}
		r.Revocation = "clear"
		r.State = "verified"
		r.Ring = name
		r.KeySerial = key
		r.Algorithm = "rsa-pkcs1-sha256"
		r.Issue = ""
		return r
	}
	if r.Issue == "" {
		r.Issue = "no accessible module-verification trusted keyring contains the certificate"
	}
	return r
}
func trustedRingSerials(list string) (map[string]int, error) {
	out := map[string]int{}
	for _, line := range strings.Split(list, "\n") {
		f := strings.Fields(line)
		// serial flags usage timeout permissions uid gid type description
		if len(f) != 10 || f[7] != "keyring" || f[5] != "0" || f[6] != "0" {
			continue
		}
		name := strings.TrimSuffix(f[8], ":")
		if name != ".secondary_trusted_keys" && name != ".builtin_trusted_keys" && name != ".blacklist" {
			continue
		}
		if _, ok := out[name]; ok {
			return nil, errors.New("ambiguous trusted kernel keyring identity")
		}
		n, e := strconv.ParseUint(f[0], 16, 31)
		if e != nil || n == 0 {
			return nil, errors.New("invalid trusted kernel keyring serial")
		}
		out[name] = int(n)
	}
	return out, nil
}

// keyctl_pkey_params is the fixed Linux UAPI (40 bytes on 32/64 bit).
func kernelVerifyRSASHA256(key int, hash, signature []byte) error {
	if len(hash) != 32 || len(signature) == 0 || len(signature) > 8192 {
		return errors.New("invalid kernel verification input")
	}
	params := struct {
		Key    int32
		InLen  uint32
		In2Len uint32
		Spare  [7]uint32
	}{Key: int32(key), InLen: uint32(len(hash)), In2Len: uint32(len(signature))}
	info := []byte("enc=pkcs1 hash=sha256\x00")
	_, _, errno := unix.Syscall6(unix.SYS_KEYCTL, unix.KEYCTL_PKEY_VERIFY, uintptr(unsafe.Pointer(&params)), uintptr(unsafe.Pointer(&info[0])), uintptr(unsafe.Pointer(&hash[0])), uintptr(unsafe.Pointer(&signature[0])), 0)
	runtime.KeepAlive(params)
	runtime.KeepAlive(info)
	runtime.KeepAlive(hash)
	runtime.KeepAlive(signature)
	if errno != 0 {
		return errno
	}
	return nil
}

// Linux x509_cert_parser.c joins the DER INTEGER serial contents with the
// issuer SEQUENCE contents (excluding ASN.1 headers), preserving zero padding.
func kernelCertificateID(cert *x509.Certificate) (string, error) {
	var tbs, field, issuer asn1.RawValue
	if rest, e := asn1.Unmarshal(cert.RawTBSCertificate, &tbs); e != nil || len(rest) != 0 {
		return "", errors.New("invalid certificate TBS")
	}
	rest, e := asn1.Unmarshal(tbs.Bytes, &field)
	if e != nil {
		return "", e
	}
	if field.Class == 2 && field.Tag == 0 {
		rest, e = asn1.Unmarshal(rest, &field)
		if e != nil {
			return "", e
		}
	}
	if field.Class != 0 || field.Tag != 2 || len(field.Bytes) == 0 {
		return "", errors.New("invalid certificate serial")
	}
	if rest, e = asn1.Unmarshal(cert.RawIssuer, &issuer); e != nil || len(rest) != 0 || issuer.Tag != 16 {
		return "", errors.New("invalid certificate issuer")
	}
	id := append(append([]byte(nil), field.Bytes...), issuer.Bytes...)
	return hex.EncodeToString(id), nil
}
