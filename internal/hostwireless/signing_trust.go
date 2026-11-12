package hostwireless

import (
	"context"
	"crypto/x509"
	"encoding/asn1"
	"strings"
)

// SigningTrust is a read-only observation of a public certificate. Enrollment,
// a mokutil keyring lookup and actual module acceptance are distinct evidence.
type SigningTrust struct {
	CertificateSHA256 string `json:"certificate_der_sha256"`
	ModuleOnly        bool   `json:"module_signing_only"`
	Enrollment        string `json:"enrollment"`
	KernelKeyring     string `json:"kernel_keyring"`
	Issue             string `json:"issue,omitempty"`
}

var moduleOnlyOID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 2312, 16, 1, 2}

// InspectSigningTrust never reads a private key or changes an enrollment request.
func InspectSigningTrust(ctx context.Context, certificatePath string) (SigningTrust, error) {
	cert, err := readPublicCertificate(certificatePath)
	if err != nil {
		return SigningTrust{}, err
	}
	return inspectSigningTrust(ctx, cert), nil
}

func inspectSigningTrust(ctx context.Context, cert *x509.Certificate) SigningTrust {
	r := SigningTrust{CertificateSHA256: digest(cert.Raw), Enrollment: "unknown", KernelKeyring: "unknown"}
	for _, oid := range cert.UnknownExtKeyUsage {
		if oid.Equal(moduleOnlyOID) {
			r.ModuleOnly = true
		}
	}
	output, err := queryMOKCertificate(ctx, cert.Raw)
	r.Enrollment, r.KernelKeyring = parseMOKCertificate(output)
	if err != nil {
		r.Issue = err.Error()
	} else if strings.Contains(output, "Failed to accesss kernel trusted keyring") {
		r.Issue = "mokutil could not access the kernel trusted keyring"
	}
	return r
}

// mokutil 0.6 test_key returns zero for NOT enrolled, and one for several
// positive, pending and blocked cases. Never turn its exit code into trust.
// Its built-in lookup compares SKID, so even the positive result is labelled
// as an observation rather than cryptographic proof of kernel acceptance.
func parseMOKCertificate(output string) (enrollment, keyring string) {
	enrollment, keyring = "unknown", "unknown"
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		switch strings.TrimSpace(line) {
		case "/proc/self/fd/3 is not enrolled":
			enrollment = "not_enrolled"
		case "/proc/self/fd/3 is already enrolled":
			enrollment = "enrolled"
		case "/proc/self/fd/3 is already in db":
			enrollment = "firmware_db"
		case "/proc/self/fd/3 is already in the enrollment request":
			enrollment = "pending"
		case "/proc/self/fd/3 is blocked in dbx", "/proc/self/fd/3 is blocked in MokListXRT":
			enrollment = "blocked"
		case "/proc/self/fd/3 is already in the built-in trusted keyring":
			keyring = "subject_key_identifier_found"
		}
	}
	// Conflicting positive and negative lines cannot authenticate anything.
	if len(lines) != 1 {
		keyring = "unknown"
		if enrollment != "not_enrolled" {
			enrollment = "unknown"
		}
	}
	return
}
