package hostwireless

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/asn1"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

const (
	moduleSignatureMarker       = "~Module signature appended~\n"
	moduleSignatureHeader       = 12
	maxSignatureBytes           = 64 << 10
	maxSignedModuleBytes  int64 = int64(maxArtifactBytes + maxSignatureBytes + moduleSignatureHeader + len(moduleSignatureMarker))
)

// SignedModuleEvidence ties the exact unsigned payload to its signed copy.
type SignedModuleEvidence struct {
	Name           string `json:"name"`
	UnsignedSHA256 string `json:"unsigned_sha256"`
	SignedSHA256   string `json:"signed_sha256"`
}

// SignedCopyEvidence records cryptographic verification against the supplied
// public certificate. Kernel trust and activation remain independent gates.
type SignedCopyEvidence struct {
	Profile             string                 `json:"profile"`
	KernelRelease       string                 `json:"kernel_release,omitempty"`
	Bundle              string                 `json:"bundle"`
	BundleReceiptSHA256 string                 `json:"bundle_receipt_sha256,omitempty"`
	CertificateSHA256   string                 `json:"certificate_der_sha256,omitempty"`
	Modules             []SignedModuleEvidence `json:"modules,omitempty"`
	SignaturesVerified  bool                   `json:"signatures_verified"`
	KernelTrustVerified bool                   `json:"kernel_trust_verified"`
	ActivationEligible  bool                   `json:"activation_eligible"`
}

// VerifySignedCopy verifies an explicitly selected set of detached kernel
// signatures against the canonical unsigned preparation and one public cert.
func VerifySignedCopy(ctx context.Context, profile, generation, signedDir, certPath string) (SignedCopyEvidence, error) {
	evidence := SignedCopyEvidence{Profile: profile, Bundle: generation}
	spec, ok := preparationProfiles[profile]
	if !ok || !bundleID.MatchString(generation) || signedDir == "" || certPath == "" {
		return evidence, errors.New("signed-copy verification requires a known profile, bundle generation, signed directory and public certificate")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	expected, err := Prepare(ctx, profile, false)
	if err != nil {
		return evidence, err
	}
	evidence.KernelRelease = expected.KernelRelease
	if expected.State != "eligible" {
		return evidence, fmt.Errorf("preparation is %s: %s", expected.State, expected.Reason)
	}
	files, receiptSHA, err := verifyPreparedBundle(ctx, expected, spec, generation)
	if err != nil {
		return evidence, err
	}
	evidence.BundleReceiptSHA256 = receiptSHA
	cert, err := readPublicCertificate(certPath)
	if err != nil {
		return evidence, err
	}
	evidence.CertificateSHA256 = digest(cert.Raw)
	info, err := os.Lstat(signedDir)
	if err != nil || !info.IsDir() {
		return evidence, errors.New("signed copies must be in a regular directory")
	}
	root, err := os.OpenRoot(signedDir)
	if err != nil {
		return evidence, err
	}
	defer root.Close()
	evidence.Modules, err = verifySignedFiles(ctx, bundleFiles{root}, files, cert)
	if err != nil {
		return evidence, err
	}
	evidence.SignaturesVerified = true
	return evidence, nil
}

func verifySignedFiles(ctx context.Context, signed fs.FS, originals []FileIdentity, cert *x509.Certificate) ([]SignedModuleEvidence, error) {
	entries, err := fs.ReadDir(signed, ".")
	if err != nil {
		return nil, err
	}
	if cert == nil || len(entries) != len(originals) {
		return nil, errors.New("signed verification requires one cert and exactly the reviewed module names")
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			return nil, errors.New("signed directory entries must be regular files")
		}
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	var modules []SignedModuleEvidence
	maxBytes := int64(maxSignedModuleBytes)
	for _, original := range originals {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		data, err := readSmallRegular(signed, original.Path, maxBytes)
		if err != nil {
			return nil, fmt.Errorf("read signed %s: %w", original.Path, err)
		}
		module, err := verifySignedModule(ctx, original, data, certPEM)
		if err != nil {
			return nil, fmt.Errorf("verify signed %s: %w", original.Path, err)
		}
		modules = append(modules, module)
	}
	return modules, nil
}

func readPublicCertificate(path string) (*x509.Certificate, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer root.Close()
	name := filepath.Base(path)
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("public certificate must be a regular file")
	}
	data, err := readSmallRegular(bundleFiles{root}, name, maxSignatureBytes)
	if err != nil {
		return nil, err
	}
	if trimmed := bytes.TrimSpace(data); bytes.HasPrefix(trimmed, []byte("-----BEGIN ")) {
		block, rest := pem.Decode(trimmed)
		if block == nil || block.Type != "CERTIFICATE" || len(block.Headers) != 0 ||
			bytes.Count(trimmed, []byte("-----BEGIN ")) != 1 || len(bytes.TrimSpace(rest)) != 0 {
			return nil, errors.New("supply exactly one public X.509 certificate")
		}
		data = block.Bytes
	}
	cert, err := x509.ParseCertificate(data)
	if err != nil {
		return nil, fmt.Errorf("parse public X.509 certificate: %w", err)
	}
	return cert, nil
}

func verifySignedModule(ctx context.Context, original FileIdentity, data, certPEM []byte) (SignedModuleEvidence, error) {
	payload, signature, err := splitModuleSignature(data)
	if err != nil {
		return SignedModuleEvidence{}, err
	}
	if digest(payload) != original.SHA256 {
		return SignedModuleEvidence{}, errors.New("unsigned module digest differs from reviewed bundle")
	}
	if err := validateKernelCMS(signature); err != nil {
		return SignedModuleEvidence{}, err
	}
	if err := verifyKernelCMS(ctx, payload, signature, certPEM); err != nil {
		return SignedModuleEvidence{}, err
	}
	return SignedModuleEvidence{Name: original.Path, UnsignedSHA256: original.SHA256, SignedSHA256: digest(data)}, nil
}

func splitModuleSignature(data []byte) ([]byte, []byte, error) {
	if len(data) <= moduleSignatureHeader+len(moduleSignatureMarker) || !bytes.HasSuffix(data, []byte(moduleSignatureMarker)) {
		return nil, nil, errors.New("kernel module signature marker absent")
	}
	headerOffset := len(data) - len(moduleSignatureMarker) - moduleSignatureHeader
	header := data[headerOffset : headerOffset+moduleSignatureHeader]
	if !bytes.Equal(header[:8], []byte{0, 0, 2, 0, 0, 0, 0, 0}) {
		return nil, nil, errors.New("invalid PKCS#7 kernel signature metadata")
	}
	size := uint64(binary.BigEndian.Uint32(header[8:]))
	if size == 0 || size > maxSignatureBytes || size >= uint64(headerOffset) || uint64(headerOffset)-size > maxArtifactBytes {
		return nil, nil, errors.New("kernel module signature length is out of bounds")
	}
	offset := headerOffset - int(size)
	return data[:offset], data[offset:headerOffset], nil
}

// The reviewed sign-file SHA-256 format is detached SignedData with one
// signer, one digest algorithm and no embedded certificate or CRL.
func validateKernelCMS(der []byte) error {
	outer, err := derSequence(der, 2)
	if err != nil || !derOID(outer[0], asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}) ||
		outer[1].Class != asn1.ClassContextSpecific || outer[1].Tag != 0 || !outer[1].IsCompound {
		return errors.New("kernel signature is not a single DER SignedData object")
	}
	signed, err := derSequence(outer[1].Bytes, 4)
	if err != nil || signed[0].Class != asn1.ClassUniversal || signed[0].Tag != asn1.TagInteger {
		return errors.New("kernel SignedData has unexpected fields")
	}
	content, err := derSequence(signed[2].FullBytes, 1)
	if err != nil || !derOID(content[0], asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}) {
		return errors.New("kernel SignedData must contain detached module data")
	}
	digests, err := derSingleSet(signed[1])
	if err != nil {
		return errors.New("kernel SignedData has an unexpected digest set")
	}
	if !derSHA256(digests) {
		return errors.New("kernel SignedData must use SHA-256")
	}
	signer, err := derSingleSet(signed[3])
	if err != nil {
		return errors.New("kernel SignedData must have one signer")
	}
	fields, err := derSequence(signer.FullBytes, 5)
	if err != nil || !derSHA256(fields[2]) || fields[4].Class != asn1.ClassUniversal ||
		fields[4].Tag != asn1.TagOctetString || fields[4].IsCompound || len(fields[4].Bytes) == 0 {
		return errors.New("kernel SignerInfo must contain an attribute-free SHA-256 signature")
	}
	return nil
}

func derSHA256(algorithm asn1.RawValue) bool {
	fields, err := derSequence(algorithm.FullBytes, 1)
	if err != nil {
		fields, err = derSequence(algorithm.FullBytes, 2)
		if err != nil || !bytes.Equal(fields[1].FullBytes, []byte{0x05, 0x00}) {
			return false
		}
	}
	return derOID(fields[0], asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1})
}

func derSequence(data []byte, count int) ([]asn1.RawValue, error) {
	var sequence asn1.RawValue
	rest, err := asn1.Unmarshal(data, &sequence)
	if err != nil || len(rest) != 0 || sequence.Class != asn1.ClassUniversal || sequence.Tag != asn1.TagSequence || !sequence.IsCompound {
		return nil, errors.New("invalid DER sequence")
	}
	fields := make([]asn1.RawValue, count)
	rest = sequence.Bytes
	for i := range fields {
		rest, err = asn1.Unmarshal(rest, &fields[i])
		if err != nil {
			return nil, err
		}
	}
	if len(rest) != 0 {
		return nil, errors.New("unexpected DER sequence fields")
	}
	return fields, nil
}

func derSingleSet(set asn1.RawValue) (asn1.RawValue, error) {
	var field asn1.RawValue
	if set.Class != asn1.ClassUniversal || set.Tag != asn1.TagSet || !set.IsCompound {
		return field, errors.New("invalid DER set")
	}
	rest, err := asn1.Unmarshal(set.Bytes, &field)
	if err != nil || len(rest) != 0 || field.Class != asn1.ClassUniversal || field.Tag != asn1.TagSequence || !field.IsCompound {
		return field, errors.New("DER set must contain one sequence")
	}
	return field, nil
}

func derOID(value asn1.RawValue, expected asn1.ObjectIdentifier) bool {
	var oid asn1.ObjectIdentifier
	rest, err := asn1.Unmarshal(value.FullBytes, &oid)
	return err == nil && len(rest) == 0 && oid.Equal(expected)
}
